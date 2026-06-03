// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const localRepo = "engalar/mxcli"

// runLocal delegates all `mxcli local *` subcommands to mxcli-local.
// It ensures the binary is installed, then execs it with inherited stdio.
func (e *Env) runLocal(args []string) int {
	// Intercept lifecycle commands — these are managed by the launcher, not mxcli-local.
	if len(args) > 0 {
		switch args[0] {
		case "upgrade":
			if err := e.acquireUpgradeLock(); err != nil {
				fmt.Fprintf(os.Stderr, "mxcli local upgrade: %v\n", err)
				return 1
			}
			defer e.releaseUpgradeLock()
			if err := e.upgradeComponent(e.localComponentConfig()); err != nil {
				fmt.Fprintf(os.Stderr, "mxcli local upgrade: %v\n", err)
				return 1
			}
			return 0

		case "rollback":
			if err := e.rollbackComponent(e.localComponentConfig()); err != nil {
				fmt.Fprintf(os.Stderr, "mxcli local rollback: %v\n", err)
				return 1
			}
			return 0
		}
	}

	if err := e.ensureLocalBinary(); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli local: %v\n", err)
		return 1
	}
	cmd := exec.Command(e.localBinaryPath(), args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "mxcli local: %v\n", err)
		return 1
	}
	return 0
}

// ensureLocalBinary ensures ~/.mxcli/local/mxcli-local is present.
// Downloads the latest local-v* release if missing.
func (e *Env) ensureLocalBinary() error {
	if err := os.MkdirAll(e.localDir(), 0700); err != nil {
		return fmt.Errorf("create local dir: %w", err)
	}
	if localBinaryExists(e.localBinaryPath()) {
		return nil
	}
	fmt.Fprintln(os.Stderr, "mxcli: mxcli-local not found, downloading latest version...")
	return e.downloadLocal(e.localBinaryPath())
}

func localBinaryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// downloadLocal fetches the latest local-v* release and installs it.
func (e *Env) downloadLocal(destPath string) error {
	tag, err := e.fetchLatestLocalTag()
	if err != nil {
		return err
	}
	return e.downloadLocalVersion(tag, destPath)
}

func (e *Env) fetchLatestLocalTag() (string, error) {
	return e.fetchLatestTagWithPrefix("local-v")
}

// fetchLatestTagWithPrefix finds the latest release whose tag starts with prefix.
// Uses the Atom feed (/releases.atom) — no auth required, no rate limits,
// includes pre-releases, ordered newest-first.
func (e *Env) fetchLatestTagWithPrefix(prefix string) (string, error) {
	url := fmt.Sprintf("https://github.com/%s/releases.atom", localRepo)
	resp, err := e.HTTPClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch releases: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub releases feed: HTTP %d", resp.StatusCode)
	}
	return parseLatestTagFromAtom(resp.Body, prefix)
}

// downloadLocalVersion downloads and extracts mxcli-local for the current platform.
func (e *Env) downloadLocalVersion(tag, destPath string) error {
	return e.downloadLocalVersionForPlatform(tag, destPath, runtime.GOOS, runtime.GOARCH, "mxcli-local")
}

func (e *Env) downloadLocalVersionForPlatform(tag, destPath, goos, goarch, assetName string) error {
	var archiveExt string
	if goos == "windows" {
		archiveExt = ".exe.zip"
	} else {
		archiveExt = ".tar.zst"
	}
	fullAsset := fmt.Sprintf("%s-%s-%s%s", assetName, goos, goarch, archiveExt)

	expectedHash, err := e.fetchAssetChecksumFromTag(tag, fullAsset)
	if err != nil {
		return fmt.Errorf("fetch checksum: %w", err)
	}

	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", localRepo, tag, fullAsset)
	fmt.Fprintf(os.Stderr, "  Downloading %s...\n", url)

	return e.downloadAndExtractComponent(url, expectedHash, destPath, goos, assetName)
}

// parseLatestTagFromAtom scans the Atom feed body for the first tag href
// matching the given prefix. Atom entries are ordered newest-first.
func parseLatestTagFromAtom(body io.Reader, prefix string) (string, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return "", fmt.Errorf("read atom feed: %w", err)
	}
	needle := "/releases/tag/" + prefix
	content := string(data)
	for {
		idx := strings.Index(content, needle)
		if idx < 0 {
			break
		}
		rest := content[idx+len(needle):]
		end := strings.IndexAny(rest, "\"'<")
		if end < 0 {
			break
		}
		tag := prefix + rest[:end]
		if tag != "" {
			return tag, nil
		}
		content = content[idx+1:]
	}
	return "", fmt.Errorf("no release found with tag prefix %q", prefix)
}

// fetchAssetChecksumFromTag fetches SHA256SUMS from a specific release tag.
func (e *Env) fetchAssetChecksumFromTag(tag, assetName string) (string, error) {
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/SHA256SUMS", localRepo, tag)
	resp, err := e.HTTPClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch SHA256SUMS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("SHA256SUMS: HTTP %d", resp.StatusCode)
	}
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return parseChecksumFile(string(content), assetName)
}

// downloadAndExtractComponent downloads an archive, verifies its checksum,
// and extracts the named binary to destPath.
func (e *Env) downloadAndExtractComponent(url, expectedHash, destPath, goos, binaryName string) error {
	resp, err := e.HTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	archiveTmp := destPath + ".archive-dl"
	defer os.Remove(archiveTmp)

	h := sha256.New()
	tee := io.TeeReader(resp.Body, h)
	af, err := os.OpenFile(archiveTmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	if _, err := io.Copy(af, tee); err != nil {
		af.Close()
		return err
	}
	af.Close()

	actualHash := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actualHash, expectedHash) {
		return fmt.Errorf("checksum mismatch: expected %s got %s", expectedHash, actualHash)
	}

	if goos == "windows" {
		return extractZip(archiveTmp, destPath, binaryName+".exe")
	}
	ar, err := os.Open(archiveTmp)
	if err != nil {
		return err
	}
	defer ar.Close()
	return extractTarZst(ar, destPath, binaryName)
}

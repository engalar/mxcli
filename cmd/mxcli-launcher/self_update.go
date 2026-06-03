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
	"strconv"
	"strings"
	"time"
)

// PIDWaiter waits for a process to exit.
type PIDWaiter interface {
	WaitForExit(pid int, timeout time.Duration) error
}

// runInternalUpdate is called in the forked child process (--internal-update mode).
// It waits for the parent process (pid) to exit, then atomically replaces
// targetPath with newBinPath.
func runInternalUpdate(pid int, newBinPath, targetPath string, waiter PIDWaiter, timeout time.Duration) error {
	if err := waiter.WaitForExit(pid, timeout); err != nil {
		return fmt.Errorf("waiting for parent PID %d: %w", pid, err)
	}

	// POSIX: atomic rename (newBin replaces target in one syscall).
	// Windows: rename target → target.old, then rename new → target.
	// (Can't overwrite a file on Windows even after process exit if handles remain,
	// but rename is allowed.)
	if runtime.GOOS == "windows" {
		oldPath := targetPath + ".old"
		os.Remove(oldPath) // clean up from prior run
		if err := os.Rename(targetPath, oldPath); err != nil {
			return fmt.Errorf("backup current binary: %w", err)
		}
	}

	if err := os.Rename(newBinPath, targetPath); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}
	return nil
}

// cleanupOldBinary removes targetPath+".old" if it exists (Windows leftover from prior self-upgrade).
func cleanupOldBinary(targetPath string) {
	if runtime.GOOS == "windows" {
		os.Remove(targetPath + ".old")
	}
}

const launcherRepo = "engalar/mxcli"

// runSelfUpgrade downloads the latest launcher release, spawns itself as an
// updater child process (--internal-update mode), then exits.
// The child waits for this process to exit before replacing the binary.
func (e *Env) runSelfUpgrade(args []string) int {
	tag, err := e.fetchLatestTagWithPrefixFor(launcherRepo, "v")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mxcli upgrade: fetch latest version: %v\n", err)
		return 1
	}

	if tag == Version {
		fmt.Printf("mxcli is already at %s — nothing to do.\n", tag)
		return 0
	}
	fmt.Printf("Upgrading mxcli %s → %s\n", Version, tag)

	selfPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mxcli upgrade: resolve self path: %v\n", err)
		return 1
	}

	tmpDest := selfPath + ".new"
	if err := e.downloadLauncherVersion(tag, tmpDest); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli upgrade: download: %v\n", err)
		os.Remove(tmpDest)
		return 1
	}

	pid := os.Getpid()
	cmd := exec.Command(selfPath,
		"--internal-update",
		fmt.Sprintf("--pid=%d", pid),
		fmt.Sprintf("--new=%s", tmpDest),
		fmt.Sprintf("--target=%s", selfPath),
	)
	hideDaemonWindow(cmd) // suppress console window on Windows
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli upgrade: start updater: %v\n", err)
		os.Remove(tmpDest)
		return 1
	}
	go func() { _ = cmd.Wait() }() // prevent zombie

	fmt.Printf("Updater started (PID %d). mxcli will restart automatically.\n", cmd.Process.Pid)
	os.Exit(0) // parent exits → updater takes over
	return 0
}

// downloadLauncherVersion downloads the launcher binary for the current platform.
func (e *Env) downloadLauncherVersion(tag, destPath string) error {
	var ext string
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	assetName := fmt.Sprintf("mxcli-%s-%s%s", runtime.GOOS, runtime.GOARCH, ext)

	// Launcher is not compressed (plain binary), so fetch checksum and download directly.
	expectedHash, err := e.fetchAssetChecksumFromTagRepo(launcherRepo, tag, assetName)
	if err != nil {
		return fmt.Errorf("fetch checksum: %w", err)
	}

	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", launcherRepo, tag, assetName)
	fmt.Fprintf(os.Stderr, "  Downloading %s...\n", url)
	return e.downloadBinaryDirect(url, expectedHash, destPath)
}

// fetchAssetChecksumFromTagRepo fetches SHA256SUMS from a specific repo+tag.
func (e *Env) fetchAssetChecksumFromTagRepo(repo, tag, assetName string) (string, error) {
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/SHA256SUMS", repo, tag)
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

// fetchLatestTagWithPrefixFor fetches the latest tag from an arbitrary repo.
func (e *Env) fetchLatestTagWithPrefixFor(repo, prefix string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=20", repo)
	resp, err := e.HTTPClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch releases: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub releases: HTTP %d", resp.StatusCode)
	}
	return parseLatestTagWithPrefix(resp.Body, prefix)
}

// downloadBinaryDirect downloads a plain binary (no archive) to destPath.
func (e *Env) downloadBinaryDirect(url, expectedHash, destPath string) error {
	resp, err := e.HTTPClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	h := sha256.New()
	tee := io.TeeReader(resp.Body, h)
	tmp := destPath + ".dl-tmp"
	defer os.Remove(tmp)

	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, tee); err != nil {
		f.Close()
		return err
	}
	f.Close()

	actualHash := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actualHash, expectedHash) {
		return fmt.Errorf("checksum mismatch: expected %s got %s", expectedHash, actualHash)
	}
	return os.Rename(tmp, destPath)
}

// parseInternalUpdateArgs parses the --internal-update flag set.
// Returns pid, newBinPath, targetPath.
func parseInternalUpdateArgs(args []string) (pid int, newBin, target string, ok bool) {
	for _, a := range args {
		switch {
		case strings.HasPrefix(a, "--pid="):
			n, err := strconv.Atoi(strings.TrimPrefix(a, "--pid="))
			if err == nil {
				pid = n
			}
		case strings.HasPrefix(a, "--new="):
			newBin = strings.TrimPrefix(a, "--new=")
		case strings.HasPrefix(a, "--target="):
			target = strings.TrimPrefix(a, "--target=")
		}
	}
	ok = pid > 0 && newBin != "" && target != ""
	return
}

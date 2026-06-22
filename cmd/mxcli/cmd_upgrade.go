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

	"github.com/spf13/cobra"
)

const mxcliRepo = "engalar/mxcli"

var upgradeCmd = &cobra.Command{
	Use:   "upgrade [tag]",
	Short: "Upgrade mxcli to the latest version",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSelfUpgrade(args)
	},
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback mxcli to the previous version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRollback()
	},
}

// PIDWaiter waits for a process to exit.
type PIDWaiter interface {
	WaitForExit(pid int, timeout time.Duration) error
}

// runSelfUpgrade downloads the latest mxcli and replaces the running binary.
func runSelfUpgrade(args []string) error {
	selfPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve self path: %w", err)
	}

	tag := ""
	if len(args) > 0 {
		tag = args[0]
	} else {
		tag, err = fetchLatestTagForRepo(mxcliRepo)
		if err != nil {
			return fmt.Errorf("fetch latest version: %w", err)
		}
	}

	fmt.Printf("Downloading mxcli %s...\n", tag)

	tmpDest := selfPath + ".new"
	if err := downloadBinary(tag, tmpDest); err != nil {
		os.Remove(tmpDest)
		return fmt.Errorf("download: %w", err)
	}

	pid := os.Getpid()
	cmd := exec.Command(selfPath,
		"--internal-update",
		fmt.Sprintf("--pid=%d", pid),
		fmt.Sprintf("--new=%s", tmpDest),
		fmt.Sprintf("--target=%s", selfPath),
	)
	hideDaemonWindow(cmd)
	if err := cmd.Start(); err != nil {
		os.Remove(tmpDest)
		return fmt.Errorf("start updater: %w", err)
	}
	go func() { _ = cmd.Wait() }()

	fmt.Printf("Updater started (PID %d). mxcli will restart automatically.\n", cmd.Process.Pid)
	os.Exit(0)
	return nil
}

// runRollback restores the previous binary backup.
func runRollback() error {
	selfPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve self path: %w", err)
	}
	backupPath := selfPath + ".old"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("no backup binary found at %s", backupPath)
	}
	pid := os.Getpid()
	cmd := exec.Command(selfPath,
		"--internal-update",
		fmt.Sprintf("--pid=%d", pid),
		fmt.Sprintf("--new=%s", backupPath),
		fmt.Sprintf("--target=%s", selfPath),
	)
	hideDaemonWindow(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start updater: %w", err)
	}
	go func() { _ = cmd.Wait() }()

	fmt.Println("Rollback started. mxcli will restart automatically.")
	os.Exit(0)
	return nil
}

// downloadBinary downloads the mxcli binary for the current platform.
func downloadBinary(tag, destPath string) error {
	var ext string
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	assetName := fmt.Sprintf("mxcli-%s-%s%s", runtime.GOOS, runtime.GOARCH, ext)

	expectedHash, err := fetchAssetChecksum(tag, assetName)
	if err != nil {
		return fmt.Errorf("fetch checksum: %w", err)
	}

	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", mxcliRepo, tag, assetName)
	resp, err := http.Get(url)
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

// fetchAssetChecksum downloads SHA256SUMS and extracts the hash for assetName.
func fetchAssetChecksum(tag, assetName string) (string, error) {
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/SHA256SUMS", mxcliRepo, tag)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
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

// parseChecksumFile extracts the SHA256 hash for a given filename from a SHA256SUMS file.
func parseChecksumFile(content, filename string) (string, error) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == filename {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("checksum for %s not found", filename)
}

// fetchLatestTagForRepo fetches the latest tag from a GitHub release feed.
func fetchLatestTagForRepo(repo string) (string, error) {
	url := fmt.Sprintf("https://github.com/%s/releases.atom", repo)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch releases: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub releases feed: HTTP %d", resp.StatusCode)
	}
	return parseLatestTagFromAtom(resp.Body)
}

// parseLatestTagFromAtom extracts the first tag from an Atom release feed.
func parseLatestTagFromAtom(body io.Reader) (string, error) {
	content, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	text := string(content)
	// Look for <id>tag:vX.Y.Z</id> or similar patterns in the Atom feed.
	// GitHub's Atom feed uses: <id>tag:v0.28.0</id> or similar.
	marker := "tag:"
	if idx := strings.Index(text, marker); idx >= 0 {
		rest := text[idx+len(marker):]
		if nl := strings.IndexAny(rest, "<\n\r"); nl >= 0 {
			return strings.TrimSpace(rest[:nl]), nil
		}
	}
	return "", fmt.Errorf("no release tag found in Atom feed")
}

// runInternalUpdate is called in the forked child process (--internal-update mode).
func runInternalUpdate(pid int, newBinPath, targetPath string, waiter PIDWaiter, timeout time.Duration) error {
	if err := waiter.WaitForExit(pid, timeout); err != nil {
		return fmt.Errorf("waiting for parent PID %d: %w", pid, err)
	}

	if runtime.GOOS == "windows" {
		oldPath := targetPath + ".old"
		os.Remove(oldPath)
		if err := os.Rename(targetPath, oldPath); err != nil {
			return fmt.Errorf("backup current binary: %w", err)
		}
	}

	if err := os.Rename(newBinPath, targetPath); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}
	return nil
}

// parseInternalUpdateArgs parses --internal-update flag set.
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

// cleanupOldBinary removes the .old backup from a prior self-upgrade.
func cleanupOldBinary(selfPath string) {
	if runtime.GOOS == "windows" {
		os.Remove(selfPath + ".old")
	}
}

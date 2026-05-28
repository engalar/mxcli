// SPDX-License-Identifier: Apache-2.0

package main

import (
	"archive/tar"
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/mendixlabs/mxcli/internal/launcherproto"
)

const (
	daemonRepo    = "engalar/mxcli"
	daemonTimeout = 10 * time.Second
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// isDaemonRunning returns true if the unix socket exists and accepts connections.
func isDaemonRunning(sockPath string) bool {
	conn, err := net.DialTimeout("unix", sockPath, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// daemonBinaryExists reports whether the daemon binary file exists.
func daemonBinaryExists(binPath string) bool {
	info, err := os.Stat(binPath)
	return err == nil && !info.IsDir()
}

// readVersionFile reads a one-line version string from path, trimming whitespace.
// Returns "" if the file cannot be read.
func readVersionFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// ensureDaemon checks that the daemon binary exists (downloading if needed)
// and that the daemon process is running (starting it if not).
func ensureDaemon() error {
	if err := os.MkdirAll(daemonDir(), 0755); err != nil {
		return fmt.Errorf("create daemon dir: %w", err)
	}
	if !daemonBinaryExists(daemonBinaryPath()) {
		fmt.Fprintln(os.Stderr, "mxcli: daemon not found, downloading latest version...")
		if err := downloadDaemon(daemonBinaryPath()); err != nil {
			return fmt.Errorf("download daemon: %w", err)
		}
	}
	if !isDaemonRunning(daemonSocketPath()) {
		if err := startDaemon(); err != nil {
			return fmt.Errorf("start daemon: %w", err)
		}
	}
	return nil
}

// startDaemon launches mxcli-daemon in the background and waits until its
// socket is ready (up to daemonTimeout). Kills the process if it times out.
func startDaemon() error {
	cmd := exec.Command(daemonBinaryPath(), "--serve", daemonSocketPath())
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("exec daemon: %w", err)
	}
	os.WriteFile(daemonPIDPath(), []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644)
	deadline := time.Now().Add(daemonTimeout)
	for time.Now().Before(deadline) {
		if isDaemonRunning(daemonSocketPath()) {
			return nil
		}
		time.Sleep(50 * time.Millisecond) // intentional poll — waiting for socket bind
	}
	cmd.Process.Kill()
	return fmt.Errorf("daemon did not start within %v", daemonTimeout)
}

// healthCheck sends a health-check request to the daemon and returns its version.
func healthCheck(sockPath string) (string, error) {
	conn, err := net.DialTimeout("unix", sockPath, 3*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	req := launcherproto.Request{Argv: []string{"__healthcheck__"}, Cwd: "/", Env: map[string]string{}}
	if err := launcherproto.WriteMsg(conn, req); err != nil {
		return "", err
	}
	var frame launcherproto.Frame
	if err := launcherproto.ReadMsg(conn, &frame); err != nil {
		return "", err
	}
	if !frame.OK {
		return "", fmt.Errorf("health check returned ok=false")
	}
	return frame.Version, nil
}

// downloadDaemon fetches the compressed daemon for the current platform and
// decompresses it to destPath.
func downloadDaemon(destPath string) error {
	tag, err := fetchLatestTag()
	if err != nil {
		return err
	}
	return downloadDaemonVersion(tag, destPath)
}

// downloadDaemonVersion downloads a specific tagged version of the daemon,
// verifies its SHA256 checksum, and extracts the binary to destPath.
func downloadDaemonVersion(tag, destPath string) error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var archiveExt string
	if goos == "windows" {
		archiveExt = ".zip"
	} else {
		archiveExt = ".tar.zst"
	}
	assetName := fmt.Sprintf("mxcli-daemon-%s-%s%s", goos, goarch, archiveExt)

	expectedHash, err := fetchAssetChecksum(tag, assetName)
	if err != nil {
		return fmt.Errorf("fetch checksum: %w", err)
	}

	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", daemonRepo, tag, assetName)
	fmt.Fprintf(os.Stderr, "  Downloading %s...\n", url)
	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	// Download archive to temp file while computing SHA256.
	archiveTmp := destPath + ".archive-dl"
	defer os.Remove(archiveTmp)

	h := sha256.New()
	tee := io.TeeReader(resp.Body, h)
	af, err := os.OpenFile(archiveTmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("create archive temp: %w", err)
	}
	if _, err := io.Copy(af, tee); err != nil {
		af.Close()
		return fmt.Errorf("download archive: %w", err)
	}
	af.Close()

	actualHash := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actualHash, expectedHash) {
		return fmt.Errorf("checksum mismatch for %s: expected %s got %s", assetName, expectedHash, actualHash)
	}

	binaryName := "mxcli-daemon"
	if goos == "windows" {
		binaryName = "mxcli-daemon.exe"
		return extractZip(archiveTmp, destPath, binaryName)
	}
	ar, err := os.Open(archiveTmp)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer ar.Close()
	return extractTarZst(ar, destPath, binaryName)
}

// fetchAssetChecksum downloads SHA256SUMS for the release and returns the
// expected hex hash for assetName.
func fetchAssetChecksum(tag, assetName string) (string, error) {
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/SHA256SUMS", daemonRepo, tag)
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch SHA256SUMS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("SHA256SUMS: HTTP %d", resp.StatusCode)
	}
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read SHA256SUMS: %w", err)
	}
	return parseChecksumFile(string(content), assetName)
}

// parseChecksumFile parses a SHA256SUMS file (sha256sum format) and returns
// the hex hash for the given filename. Handles both plain and starred names
// (e.g. "abc123 *file.zip" produced by some tools on Windows).
func parseChecksumFile(content, filename string) (string, error) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		if name == filename {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no checksum for %q in SHA256SUMS", filename)
}

// extractTarZst decompresses a .tar.zst stream and writes the entry named
// expectedName to destPath with executable permissions.
func extractTarZst(r io.Reader, destPath, expectedName string) error {
	zr, err := zstd.NewReader(r)
	if err != nil {
		return err
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(hdr.Name) != expectedName {
			continue
		}
		tmp := destPath + ".tmp"
		f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, tr); err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
		f.Close()
		return os.Rename(tmp, destPath)
	}
	return fmt.Errorf("no file named %q found in archive", expectedName)
}

// extractZip extracts the entry named expectedName from the zip archive at
// srcPath and writes it to destPath with executable permissions.
func extractZip(srcPath, destPath, expectedName string) error {
	zr, err := zip.OpenReader(srcPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if filepath.Base(f.Name) != expectedName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		tmp := destPath + ".tmp"
		out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		rc.Close()
		out.Close()
		if copyErr != nil {
			os.Remove(tmp)
			return copyErr
		}
		return os.Rename(tmp, destPath)
	}
	return fmt.Errorf("no file named %q found in zip archive", expectedName)
}

// fetchLatestTag queries the GitHub releases API for the latest tag.
func fetchLatestTag() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", daemonRepo)
	return fetchTagFromURL(url)
}

// fetchTagFromURL fetches a GitHub releases JSON from url and extracts tag_name.
// Separated from fetchLatestTag for testability.
func fetchTagFromURL(url string) (string, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parse GitHub response: %w", err)
	}
	if result.TagName == "" {
		return "", fmt.Errorf("tag_name not found in GitHub response")
	}
	return result.TagName, nil
}

// killPIDFile reads pidPath, kills the process with that PID if valid, then
// removes both the PID file and the socket file. Safe to call when files are absent.
func killPIDFile(pidPath, sockPath string) {
	b, err := os.ReadFile(pidPath)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err == nil && pid > 0 {
		if p, err := os.FindProcess(pid); err == nil {
			_ = p.Kill()
		}
	}
	os.Remove(pidPath)
	os.Remove(sockPath)
}

// killRunningDaemon kills the daemon process recorded in the PID file and
// removes the socket. Called before rollback to avoid leaving a stale daemon.
func killRunningDaemon() {
	killPIDFile(daemonPIDPath(), daemonSocketPath())
}

// rollback restores the daemon from .bak. Called on upgrade failure.
func rollback() {
	if !daemonBinaryExists(daemonBakPath()) {
		fmt.Fprintln(os.Stderr, "mxcli: no backup to restore")
		return
	}
	killRunningDaemon()
	os.Remove(daemonBinaryPath())
	os.Rename(daemonBakPath(), daemonBinaryPath())
	os.Rename(daemonVersionBakPath(), daemonVersionPath())
	ver := readVersionFile(daemonVersionPath())
	fmt.Printf("Rolled back to %s\n", ver)
}

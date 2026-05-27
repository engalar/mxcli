// SPDX-License-Identifier: Apache-2.0

package main

import (
	"archive/tar"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/mendixlabs/mxcli/internal/launcherproto"
)

const (
	daemonRepo    = "engalar/mxcli"
	daemonTimeout = 10 * time.Second
)

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
// socket is ready (up to daemonTimeout).
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
		time.Sleep(50 * time.Millisecond)
	}
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

// downloadDaemonVersion downloads a specific tagged version of the daemon.
func downloadDaemonVersion(tag, destPath string) error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	ext := ".tar.zst"
	if goos == "windows" {
		ext = ".zip"
	}
	assetName := fmt.Sprintf("mxcli-daemon-%s-%s%s", goos, goarch, ext)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", daemonRepo, tag, assetName)
	fmt.Fprintf(os.Stderr, "  Downloading %s...\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	if goos == "windows" {
		return fmt.Errorf("zip extraction not yet implemented; download %s manually", destPath)
	}
	return extractTarZst(resp.Body, destPath)
}

// extractTarZst decompresses a .tar.zst stream and writes the first regular
// file entry to destPath with executable permissions.
func extractTarZst(r io.Reader, destPath string) error {
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
	return fmt.Errorf("no regular file found in archive")
}

// fetchLatestTag queries the GitHub releases API for the latest tag.
func fetchLatestTag() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", daemonRepo)
	return fetchTagFromURL(url)
}

// fetchTagFromURL fetches a GitHub releases JSON from url and extracts tag_name.
// Separated from fetchLatestTag for testability.
func fetchTagFromURL(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	s := string(body)
	key := `"tag_name":"`
	idx := strings.Index(s, key)
	if idx < 0 {
		return "", fmt.Errorf("tag_name not found in GitHub response")
	}
	s = s[idx+len(key):]
	end := strings.Index(s, `"`)
	if end < 0 {
		return "", fmt.Errorf("malformed tag_name in GitHub response")
	}
	return s[:end], nil
}

// rollback restores the daemon from .bak. Called on upgrade failure.
func rollback() {
	if !daemonBinaryExists(daemonBakPath()) {
		fmt.Fprintln(os.Stderr, "mxcli: no backup to restore")
		return
	}
	os.Remove(daemonBinaryPath())
	os.Rename(daemonBakPath(), daemonBinaryPath())
	os.Rename(daemonVersionBakPath(), daemonVersionPath())
	ver := readVersionFile(daemonVersionPath())
	fmt.Printf("Rolled back to %s\n", ver)
}

// writeVersionFile writes a version string to path.
func writeVersionFile(path, version string) error {
	return os.WriteFile(path, []byte(version), 0644)
}

// daemonBinaryDir returns the directory containing the daemon binary.
func daemonBinaryDir() string {
	return filepath.Dir(daemonBinaryPath())
}

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

func isDaemonRunning(sockPath string) bool {
	conn, err := net.DialTimeout("unix", sockPath, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func daemonBinaryExists(binPath string) bool {
	info, err := os.Stat(binPath)
	return err == nil && !info.IsDir()
}

func readVersionFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func (e *Env) ensureDaemon() error {
	if err := os.MkdirAll(e.daemonDir(), 0755); err != nil {
		return fmt.Errorf("create daemon dir: %w", err)
	}
	if !daemonBinaryExists(e.daemonBinaryPath()) {
		fmt.Fprintln(os.Stderr, "mxcli: daemon not found, downloading latest version...")
		if err := e.downloadDaemon(e.daemonBinaryPath()); err != nil {
			return fmt.Errorf("download daemon: %w", err)
		}
	}
	if !isDaemonRunning(e.daemonSocketPath()) {
		if err := e.startDaemon(); err != nil {
			return fmt.Errorf("start daemon: %w", err)
		}
	}
	return nil
}

func (e *Env) startDaemon() error {
	cmd := exec.Command(e.daemonBinaryPath(), "--serve", e.daemonSocketPath())
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("exec daemon: %w", err)
	}
	os.WriteFile(e.daemonPIDPath(), []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644)
	deadline := time.Now().Add(daemonTimeout)
	for time.Now().Before(deadline) {
		if isDaemonRunning(e.daemonSocketPath()) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	cmd.Process.Kill()
	return fmt.Errorf("daemon did not start within %v", daemonTimeout)
}

func (e *Env) healthCheck(sockPath string) (string, error) {
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

func (e *Env) downloadDaemon(destPath string) error {
	tag, err := e.fetchLatestTag()
	if err != nil {
		return err
	}
	return e.downloadDaemonVersion(tag, destPath)
}

func (e *Env) downloadDaemonVersion(tag, destPath string) error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var archiveExt string
	if goos == "windows" {
		archiveExt = ".exe.zip"
	} else {
		archiveExt = ".tar.zst"
	}
	assetName := fmt.Sprintf("mxcli-daemon-%s-%s%s", goos, goarch, archiveExt)

	expectedHash, err := e.fetchAssetChecksum(tag, assetName)
	if err != nil {
		return fmt.Errorf("fetch checksum: %w", err)
	}

	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", daemonRepo, tag, assetName)
	fmt.Fprintf(os.Stderr, "  Downloading %s...\n", url)
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

func (e *Env) fetchAssetChecksum(tag, assetName string) (string, error) {
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/SHA256SUMS", daemonRepo, tag)
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
		return "", fmt.Errorf("read SHA256SUMS: %w", err)
	}
	return parseChecksumFile(string(content), assetName)
}

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

func (e *Env) fetchLatestTag() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", daemonRepo)
	return e.fetchTagFromURL(url)
}

func (e *Env) fetchTagFromURL(url string) (string, error) {
	resp, err := e.HTTPClient.Get(url)
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

func (e *Env) killRunningDaemon() {
	killPIDFile(e.daemonPIDPath(), e.daemonSocketPath())
}

func (e *Env) rollback() {
	if !daemonBinaryExists(e.daemonBakPath()) {
		fmt.Fprintln(os.Stderr, "mxcli: no backup to restore")
		return
	}
	e.killRunningDaemon()
	os.Remove(e.daemonBinaryPath())
	os.Rename(e.daemonBakPath(), e.daemonBinaryPath())
	os.Rename(e.daemonVersionBakPath(), e.daemonVersionPath())
	ver := readVersionFile(e.daemonVersionPath())
	fmt.Printf("Rolled back to %s\n", ver)
}

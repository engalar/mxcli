// SPDX-License-Identifier: Apache-2.0

package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func TestIsDaemonRunning_NoSocket(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "nosuch.sock")
	if isDaemonRunning(sockPath) {
		t.Error("expected false when socket does not exist")
	}
}

func TestDaemonBinaryExists_Missing(t *testing.T) {
	if daemonBinaryExists(filepath.Join(t.TempDir(), "no-daemon")) {
		t.Error("expected false for missing binary")
	}
}

func TestDaemonBinaryExists_Present(t *testing.T) {
	p := filepath.Join(t.TempDir(), "mxcli-daemon")
	os.WriteFile(p, []byte("fake"), 0755)
	if !daemonBinaryExists(p) {
		t.Error("expected true for existing binary")
	}
}

func TestReadVersionFile_Missing(t *testing.T) {
	v := readVersionFile(filepath.Join(t.TempDir(), "version"))
	if v != "" {
		t.Errorf("expected empty, got %q", v)
	}
}

func TestReadVersionFile_Present(t *testing.T) {
	p := filepath.Join(t.TempDir(), "version")
	os.WriteFile(p, []byte("v0.14.0\n"), 0644)
	v := readVersionFile(p)
	if v != "v0.14.0" {
		t.Errorf("expected v0.14.0, got %q", v)
	}
}

func TestFetchTagFromURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v1.2.3","name":"Release v1.2.3"}`))
	}))
	defer srv.Close()
	e := &Env{HTTPClient: srv.Client()}
	tag, err := e.fetchTagFromURL(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "v1.2.3" {
		t.Errorf("expected v1.2.3, got %q", tag)
	}
}

// makeTarZst creates an in-memory .tar.zst archive containing the given files.
func makeTarZst(t *testing.T, files map[string][]byte) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	zw, err := zstd.NewWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(zw)
	for name, content := range files {
		hdr := &tar.Header{
			Name:     name,
			Typeflag: tar.TypeReg,
			Size:     int64(len(content)),
			Mode:     0755,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	tw.Close()
	zw.Close()
	return &buf
}

func TestExtractTarZst_CorrectFilename(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tar.zst extraction not compiled on Windows (daemon uses .exe.zip)")
	}
	archive := makeTarZst(t, map[string][]byte{"mxcli-daemon": []byte("binary-data")})
	dest := filepath.Join(t.TempDir(), "mxcli-daemon")
	if err := extractTarZst(archive, dest, "mxcli-daemon"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != "binary-data" {
		t.Errorf("wrong content: %q", got)
	}
}

func TestExtractTarZst_RejectsWrongFilename(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tar.zst extraction not compiled on Windows (daemon uses .exe.zip)")
	}
	archive := makeTarZst(t, map[string][]byte{"readme.txt": []byte("docs")})
	dest := filepath.Join(t.TempDir(), "mxcli-daemon")
	if err := extractTarZst(archive, dest, "mxcli-daemon"); err == nil {
		t.Error("expected error when archive has no matching file")
	}
}

func TestExtractZip_CorrectFilename(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("mxcli-daemon.exe")
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte("win-binary"))
	w.Close()

	zipPath := filepath.Join(t.TempDir(), "test.zip")
	if err := os.WriteFile(zipPath, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "mxcli-daemon.exe")
	if err := extractZip(zipPath, dest, "mxcli-daemon.exe"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != "win-binary" {
		t.Errorf("wrong content: %q", got)
	}
}

func TestExtractZip_RejectsWrongFilename(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("readme.txt")
	f.Write([]byte("docs"))
	w.Close()

	zipPath := filepath.Join(t.TempDir(), "test.zip")
	os.WriteFile(zipPath, buf.Bytes(), 0644)

	dest := filepath.Join(t.TempDir(), "mxcli-daemon.exe")
	if err := extractZip(zipPath, dest, "mxcli-daemon.exe"); err == nil {
		t.Error("expected error when zip has no matching file")
	}
}

func TestParseChecksumFile_Found(t *testing.T) {
	content := "abc123  mxcli-daemon-linux-amd64.tar.zst\ndef456  mxcli-daemon-darwin-amd64.tar.zst\n"
	hash, err := parseChecksumFile(content, "mxcli-daemon-linux-amd64.tar.zst")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash != "abc123" {
		t.Errorf("expected abc123, got %q", hash)
	}
}

func TestParseChecksumFile_NotFound(t *testing.T) {
	content := "abc123  mxcli-daemon-linux-amd64.tar.zst\n"
	_, err := parseChecksumFile(content, "mxcli-daemon-darwin-amd64.tar.zst")
	if err == nil {
		t.Error("expected error for missing filename")
	}
}

func TestParseChecksumFile_StarredName(t *testing.T) {
	content := "abc123 *mxcli-daemon-windows-amd64.exe.zip\n"
	hash, err := parseChecksumFile(content, "mxcli-daemon-windows-amd64.exe.zip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash != "abc123" {
		t.Errorf("expected abc123, got %q", hash)
	}
}

func TestKillPIDFile_NoPIDFile(t *testing.T) {
	dir := t.TempDir()
	killPIDFile(filepath.Join(dir, "pid"), filepath.Join(dir, "sock"))
}

func TestKillPIDFile_InvalidPID(t *testing.T) {
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "pid")
	sockPath := filepath.Join(dir, "sock")
	os.WriteFile(pidPath, []byte("garbage"), 0644)
	os.WriteFile(sockPath, []byte{}, 0644)

	killPIDFile(pidPath, sockPath)

	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Error("pid file should be removed even with invalid PID")
	}
	if _, err := os.Stat(sockPath); !os.IsNotExist(err) {
		t.Error("sock file should be removed even with invalid PID")
	}
}

func TestKillPIDFile_KillsProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires sleep command")
	}
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "pid")
	sockPath := filepath.Join(dir, "sock")
	os.WriteFile(sockPath, []byte{}, 0644)

	cmd := exec.Command("sleep", "100")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0644)

	killPIDFile(pidPath, sockPath)

	if err := cmd.Wait(); err == nil {
		t.Error("expected process to have been killed")
	}
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Error("pid file should be removed after kill")
	}
	if _, err := os.Stat(sockPath); !os.IsNotExist(err) {
		t.Error("sock file should be removed after kill")
	}
}

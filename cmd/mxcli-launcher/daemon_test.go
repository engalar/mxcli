// SPDX-License-Identifier: Apache-2.0

package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
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
	tag, err := fetchTagFromURL(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "v1.2.3" {
		t.Errorf("expected v1.2.3, got %q", tag)
	}
}

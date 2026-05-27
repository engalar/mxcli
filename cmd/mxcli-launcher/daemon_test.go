// SPDX-License-Identifier: Apache-2.0

package main

import (
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

func TestReadDaemonVersion_Missing(t *testing.T) {
	vPath := filepath.Join(t.TempDir(), "version")
	v := readVersionFile(vPath)
	if v != "" {
		t.Errorf("expected empty version, got %q", v)
	}
}

func TestReadDaemonVersion_Present(t *testing.T) {
	vPath := filepath.Join(t.TempDir(), "version")
	os.WriteFile(vPath, []byte("v0.14.0\n"), 0644)
	v := readVersionFile(vPath)
	if v != "v0.14.0" {
		t.Errorf("expected v0.14.0, got %q", v)
	}
}

func TestDaemonBinaryExists_Missing(t *testing.T) {
	if daemonBinaryExists(filepath.Join(t.TempDir(), "no-daemon")) {
		t.Error("expected false for missing binary")
	}
}

func TestDaemonBinaryExists_Present(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "mxcli-daemon")
	os.WriteFile(p, []byte("fake"), 0755)
	if !daemonBinaryExists(p) {
		t.Error("expected true for existing binary")
	}
}

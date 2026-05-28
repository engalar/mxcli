// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestEnv(t *testing.T) *Env {
	t.Helper()
	return &Env{HomeDir: t.TempDir(), HTTPClient: nil}
}

func TestEnvDaemonDir_IsUnderHome(t *testing.T) {
	e := newTestEnv(t)
	dir := e.daemonDir()
	if !filepath.IsAbs(dir) {
		t.Errorf("daemonDir must be absolute, got %q", dir)
	}
	if dir == e.HomeDir {
		t.Error("daemonDir must not equal HomeDir")
	}
}

func TestEnvPaths_Consistent(t *testing.T) {
	e := newTestEnv(t)
	dir := e.daemonDir()
	if e.daemonBinaryPath() != filepath.Join(dir, "mxcli-daemon") {
		t.Error("daemonBinaryPath mismatch")
	}
	if e.daemonBakPath() != filepath.Join(dir, "mxcli-daemon.bak") {
		t.Error("daemonBakPath mismatch")
	}
	if e.daemonSocketPath() != filepath.Join(dir, "mxcli.sock") {
		t.Error("daemonSocketPath mismatch")
	}
	if e.daemonVersionPath() != filepath.Join(dir, "version") {
		t.Error("daemonVersionPath mismatch")
	}
	if e.daemonVersionBakPath() != filepath.Join(dir, "version.bak") {
		t.Error("daemonVersionBakPath mismatch")
	}
	if e.daemonUpdateAvailablePath() != filepath.Join(dir, "update-available") {
		t.Error("daemonUpdateAvailablePath mismatch")
	}
	if e.daemonLastCheckPath() != filepath.Join(dir, "last-check") {
		t.Error("daemonLastCheckPath mismatch")
	}
	if e.daemonPIDPath() != filepath.Join(dir, "mxcli-daemon.pid") {
		t.Error("daemonPIDPath mismatch")
	}
}

func TestDefaultEnv_HomeDir(t *testing.T) {
	e := DefaultEnv()
	home, _ := os.UserHomeDir()
	if e.HomeDir != home {
		t.Errorf("DefaultEnv HomeDir = %q, want %q", e.HomeDir, home)
	}
	if e.HTTPClient == nil {
		t.Error("DefaultEnv HTTPClient must not be nil")
	}
}

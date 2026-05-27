// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDaemonDir_ContainsHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	dir := daemonDir()
	if !filepath.IsAbs(dir) {
		t.Errorf("daemonDir must be absolute, got %q", dir)
	}
	if dir == home {
		t.Error("daemonDir must not equal home dir")
	}
}

func TestDaemonPaths_Consistent(t *testing.T) {
	dir := daemonDir()
	if daemonBinaryPath() != filepath.Join(dir, "mxcli-daemon") {
		t.Error("daemonBinaryPath mismatch")
	}
	if daemonBakPath() != filepath.Join(dir, "mxcli-daemon.bak") {
		t.Error("daemonBakPath mismatch")
	}
	if daemonSocketPath() != filepath.Join(dir, "mxcli.sock") {
		t.Error("daemonSocketPath mismatch")
	}
	if daemonVersionPath() != filepath.Join(dir, "version") {
		t.Error("daemonVersionPath mismatch")
	}
	if daemonVersionBakPath() != filepath.Join(dir, "version.bak") {
		t.Error("daemonVersionBakPath mismatch")
	}
	if daemonUpdateAvailablePath() != filepath.Join(dir, "update-available") {
		t.Error("daemonUpdateAvailablePath mismatch")
	}
	if daemonLastCheckPath() != filepath.Join(dir, "last-check") {
		t.Error("daemonLastCheckPath mismatch")
	}
	if daemonPIDPath() != filepath.Join(dir, "mxcli-daemon.pid") {
		t.Error("daemonPIDPath mismatch")
	}
}

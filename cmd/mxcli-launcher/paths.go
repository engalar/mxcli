// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
)

func daemonDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return filepath.Join(home, ".mxcli", "daemon")
}

func daemonBinaryPath() string          { return filepath.Join(daemonDir(), "mxcli-daemon") }
func daemonBakPath() string             { return filepath.Join(daemonDir(), "mxcli-daemon.bak") }
func daemonSocketPath() string          { return filepath.Join(daemonDir(), "mxcli.sock") }
func daemonVersionPath() string         { return filepath.Join(daemonDir(), "version") }
func daemonVersionBakPath() string      { return filepath.Join(daemonDir(), "version.bak") }
func daemonUpdateAvailablePath() string { return filepath.Join(daemonDir(), "update-available") }
func daemonLastCheckPath() string       { return filepath.Join(daemonDir(), "last-check") }
func daemonPIDPath() string             { return filepath.Join(daemonDir(), "mxcli-daemon.pid") }

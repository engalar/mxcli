// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func (e *Env) daemonDir() string { return filepath.Join(e.HomeDir, ".mxcli", "daemon") }

func (e *Env) daemonBinaryPath() string {
	name := "mxcli-daemon"
	if runtime.GOOS == "windows" {
		name = "mxcli-daemon.exe"
	}
	return filepath.Join(e.daemonDir(), name)
}

func (e *Env) daemonBakPath() string         { return filepath.Join(e.daemonDir(), "mxcli-daemon.bak") }
func (e *Env) daemonSocketPath() string      { return filepath.Join(e.daemonDir(), "mxcli.sock") }
func (e *Env) daemonVersionPath() string     { return filepath.Join(e.daemonDir(), "version") }
func (e *Env) daemonVersionBakPath() string  { return filepath.Join(e.daemonDir(), "version.bak") }
func (e *Env) daemonUpdateAvailablePath() string { return filepath.Join(e.daemonDir(), "update-available") }
func (e *Env) daemonLastCheckPath() string   { return filepath.Join(e.daemonDir(), "last-check") }
func (e *Env) daemonPIDPath() string { return filepath.Join(e.daemonDir(), "mxcli-daemon.pid") }

// mprDaemonSocketPath returns the Unix socket path for a per-MPR daemon bound to mprAbsPath.
// Format: mpr-{mprHash[0:6]}-{binHash[0:2]}.sock
//   - mprHash identifies the project; same prefix = same project
//   - binHash changes when the binary is recompiled, invalidating stale daemons
func (e *Env) mprDaemonSocketPath(mprAbsPath string) string {
	mprHash := sha256.Sum256([]byte(mprAbsPath))
	binHash := sha256.Sum256([]byte(e.daemonBinaryMtime()))
	name := fmt.Sprintf("mpr-%x-%x.sock", mprHash[:6], binHash[:2])
	return filepath.Join(e.daemonDir(), name)
}

// mprSocketPrefix returns the fixed prefix of all socket files for mprAbsPath,
// used to identify old-binary sockets for the same project.
func (e *Env) mprSocketPrefix(mprAbsPath string) string {
	hash := sha256.Sum256([]byte(mprAbsPath))
	return fmt.Sprintf("mpr-%x-", hash[:6])
}

// daemonBinaryMtime returns the daemon binary's modification time as a string.
// Used as part of the socket path hash to invalidate stale daemons when binary updates.
func (e *Env) daemonBinaryMtime() string {
	info, err := os.Stat(e.daemonBinaryPath())
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d", info.ModTime().UnixNano())
}

func (e *Env) localDir() string { return filepath.Join(e.HomeDir, ".mxcli", "local") }

func (e *Env) localBinaryPath() string {
	name := "mxcli-local"
	if runtime.GOOS == "windows" {
		name = "mxcli-local.exe"
	}
	return filepath.Join(e.localDir(), name)
}

func (e *Env) localBinaryBakPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(e.localDir(), "mxcli-local.bak.exe")
	}
	return filepath.Join(e.localDir(), "mxcli-local.bak")
}

func (e *Env) localVersionPath() string   { return filepath.Join(e.localDir(), "version") }
func (e *Env) localLastCheckPath() string { return filepath.Join(e.localDir(), "last-check") }
func (e *Env) localUpdateAvailablePath() string {
	return filepath.Join(e.localDir(), "update-available")
}

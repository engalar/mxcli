// SPDX-License-Identifier: Apache-2.0

package main

import "path/filepath"

func (e *Env) daemonDir() string             { return filepath.Join(e.HomeDir, ".mxcli", "daemon") }
func (e *Env) daemonBinaryPath() string      { return filepath.Join(e.daemonDir(), "mxcli-daemon") }
func (e *Env) daemonBakPath() string         { return filepath.Join(e.daemonDir(), "mxcli-daemon.bak") }
func (e *Env) daemonSocketPath() string      { return filepath.Join(e.daemonDir(), "mxcli.sock") }
func (e *Env) daemonVersionPath() string     { return filepath.Join(e.daemonDir(), "version") }
func (e *Env) daemonVersionBakPath() string  { return filepath.Join(e.daemonDir(), "version.bak") }
func (e *Env) daemonUpdateAvailablePath() string { return filepath.Join(e.daemonDir(), "update-available") }
func (e *Env) daemonLastCheckPath() string   { return filepath.Join(e.daemonDir(), "last-check") }
func (e *Env) daemonPIDPath() string         { return filepath.Join(e.daemonDir(), "mxcli-daemon.pid") }

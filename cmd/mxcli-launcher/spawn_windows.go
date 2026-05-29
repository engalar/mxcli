// SPDX-License-Identifier: Apache-2.0

//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// hideDaemonWindow prevents a console window from flashing when the daemon
// process is started in the background on Windows.
func hideDaemonWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

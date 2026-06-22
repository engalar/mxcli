// SPDX-License-Identifier: Apache-2.0

//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

func hideDaemonWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

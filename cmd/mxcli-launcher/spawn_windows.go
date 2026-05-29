// SPDX-License-Identifier: Apache-2.0

//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

const (
	// createBreakawayFromJob detaches the child process from the parent's
	// Job Object. Without this flag, PowerShell pipelines (which use Job
	// Objects) kill all child processes when the launcher exits — causing
	// the daemon to be killed on every invocation and forcing cold restarts.
	createBreakawayFromJob = 0x01000000
)

// hideDaemonWindow hides the console window and detaches the daemon from the
// launcher's Job Object so it survives across PowerShell pipeline boundaries.
func hideDaemonWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createBreakawayFromJob,
	}
}

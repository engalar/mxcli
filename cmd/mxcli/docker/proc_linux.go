// SPDX-License-Identifier: Apache-2.0

//go:build linux

package docker

import (
	"os/exec"
	"syscall"
)

// CmdWithPdeathsig configures cmd so the child process receives SIGTERM
// when the parent (mxcli) exits, even on crash/panic.  On Linux this uses
// Pdeathsig + Setpgid; on other platforms it is a no-op.
func CmdWithPdeathsig(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.SysProcAttr.Pdeathsig = syscall.SIGTERM
}

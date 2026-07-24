// SPDX-License-Identifier: Apache-2.0

//go:build !linux

package docker

import "os/exec"

// CmdWithPdeathsig is a no-op on non-Linux platforms where Pdeathsig is
// not supported by the OS.
func CmdWithPdeathsig(cmd *exec.Cmd) {
}

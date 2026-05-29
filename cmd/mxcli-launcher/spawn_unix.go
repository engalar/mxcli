// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package main

import "os/exec"

func hideDaemonWindow(cmd *exec.Cmd) {}

// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package main

import (
	"fmt"
	"syscall"
	"time"
)

// RealPIDWaiter polls kill(pid, 0) until the process is gone.
type RealPIDWaiter struct{}

func (r *RealPIDWaiter) WaitForExit(pid int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		err := syscall.Kill(pid, 0)
		if err == syscall.ESRCH {
			return nil // process no longer exists
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("PID %d did not exit within %v", pid, timeout)
}

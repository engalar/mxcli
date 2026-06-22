// SPDX-License-Identifier: Apache-2.0

//go:build windows

package main

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows"
)

// RealPIDWaiter uses OpenProcess + WaitForSingleObject on Windows.
type RealPIDWaiter struct{}

func (r *RealPIDWaiter) WaitForExit(pid int, timeout time.Duration) error {
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		// Process already gone (access denied or not found) → treat as exited.
		return nil
	}
	defer windows.CloseHandle(handle)

	ms := uint32(timeout.Milliseconds())
	result, _ := windows.WaitForSingleObject(handle, ms)
	switch result {
	case windows.WAIT_OBJECT_0:
		return nil
	case uint32(windows.WAIT_TIMEOUT):
		return fmt.Errorf("PID %d did not exit within %v", pid, timeout)
	default:
		return fmt.Errorf("WaitForSingleObject returned %v", result)
	}
}

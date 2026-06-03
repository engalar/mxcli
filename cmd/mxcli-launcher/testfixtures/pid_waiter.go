// SPDX-License-Identifier: Apache-2.0

package testfixtures

import (
	"fmt"
	"time"
)

// FakePIDWaiter is a test double for PIDWaiter.
// Close ExitC to simulate the target process exiting.
type FakePIDWaiter struct {
	ExitC chan struct{}
}

// NewFakePIDWaiter returns a FakePIDWaiter ready for use.
func NewFakePIDWaiter() *FakePIDWaiter {
	return &FakePIDWaiter{ExitC: make(chan struct{})}
}

// WaitForExit blocks until ExitC is closed or timeout elapses.
func (f *FakePIDWaiter) WaitForExit(pid int, timeout time.Duration) error {
	select {
	case <-f.ExitC:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for PID %d to exit", pid)
	}
}

// SimulateExit signals that the target process has exited.
func (f *FakePIDWaiter) SimulateExit() {
	select {
	case <-f.ExitC:
		// already closed
	default:
		close(f.ExitC)
	}
}

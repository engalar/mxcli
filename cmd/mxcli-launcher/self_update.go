// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

// PIDWaiter waits for a process to exit.
type PIDWaiter interface {
	WaitForExit(pid int, timeout time.Duration) error
}

// runInternalUpdate is called in the forked child process (--internal-update mode).
// It waits for the parent process (pid) to exit, then atomically replaces
// targetPath with newBinPath.
func runInternalUpdate(pid int, newBinPath, targetPath string, waiter PIDWaiter, timeout time.Duration) error {
	if err := waiter.WaitForExit(pid, timeout); err != nil {
		return fmt.Errorf("waiting for parent PID %d: %w", pid, err)
	}

	// POSIX: atomic rename (newBin replaces target in one syscall).
	// Windows: rename target → target.old, then rename new → target.
	// (Can't overwrite a file on Windows even after process exit if handles remain,
	// but rename is allowed.)
	if runtime.GOOS == "windows" {
		oldPath := targetPath + ".old"
		os.Remove(oldPath) // clean up from prior run
		if err := os.Rename(targetPath, oldPath); err != nil {
			return fmt.Errorf("backup current binary: %w", err)
		}
	}

	if err := os.Rename(newBinPath, targetPath); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}
	return nil
}

// cleanupOldBinary removes targetPath+".old" if it exists (Windows leftover from prior self-upgrade).
func cleanupOldBinary(targetPath string) {
	if runtime.GOOS == "windows" {
		os.Remove(targetPath + ".old")
	}
}

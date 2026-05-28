// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// lockFile holds the open lock file descriptor.
type lockFile struct{ f *os.File }

func (e *Env) acquireUpgradeLock() error {
	lockPath := filepath.Join(e.daemonDir(), "upgrade.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("open lock: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return fmt.Errorf("upgrade in progress")
	}
	e.upgradeLock = &lockFile{f: f}
	return nil
}

func (e *Env) releaseUpgradeLock() {
	if e.upgradeLock == nil {
		return
	}
	syscall.Flock(int(e.upgradeLock.f.Fd()), syscall.LOCK_UN)
	e.upgradeLock.f.Close()
	e.upgradeLock = nil
}

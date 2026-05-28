// SPDX-License-Identifier: Apache-2.0

//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var modkernel32 = syscall.NewLazyDLL("kernel32.dll")
var procLockFileEx = modkernel32.NewProc("LockFileEx")
var procUnlockFile = modkernel32.NewProc("UnlockFile")

const lockfileFailImmediately = 0x00000001
const lockfileExclusiveLock = 0x00000002

type lockFile struct{ f *os.File }

func (e *Env) acquireUpgradeLock() error {
	lockPath := filepath.Join(e.daemonDir(), "upgrade.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("open lock: %w", err)
	}
	ol := new(syscall.Overlapped)
	r, _, _ := procLockFileEx.Call(
		uintptr(f.Fd()),
		uintptr(lockfileExclusiveLock|lockfileFailImmediately),
		0, 1, 0,
		uintptr(unsafe.Pointer(ol)),
	)
	if r == 0 {
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
	procUnlockFile.Call(uintptr(e.upgradeLock.f.Fd()), 0, 1, 0, 0)
	e.upgradeLock.f.Close()
	e.upgradeLock = nil
}

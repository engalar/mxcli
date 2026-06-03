// SPDX-License-Identifier: Apache-2.0

package main

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestLocalDir(t *testing.T) {
	e := &Env{HomeDir: t.TempDir()}
	want := filepath.Join(e.HomeDir, ".mxcli", "local")
	if got := e.localDir(); got != want {
		t.Errorf("localDir: got %s, want %s", got, want)
	}
}

func TestLocalBinaryPath(t *testing.T) {
	e := &Env{HomeDir: t.TempDir()}
	p := e.localBinaryPath()
	base := filepath.Base(p)
	if runtime.GOOS == "windows" {
		if base != "mxcli-local.exe" {
			t.Errorf("Windows: got %s, want mxcli-local.exe", base)
		}
	} else {
		if base != "mxcli-local" {
			t.Errorf("non-Windows: got %s, want mxcli-local", base)
		}
	}
}

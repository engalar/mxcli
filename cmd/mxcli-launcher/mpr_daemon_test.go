// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- extractMPRFromArgs ---

func TestExtractMPRFromArgs_ShortFlag(t *testing.T) {
	got := extractMPRFromArgs([]string{"-p", "app.mpr", "-c", "show entities"})
	if got != "app.mpr" {
		t.Errorf("got %q, want app.mpr", got)
	}
}

func TestExtractMPRFromArgs_LongFlag(t *testing.T) {
	got := extractMPRFromArgs([]string{"--project", "app.mpr"})
	if got != "app.mpr" {
		t.Errorf("got %q, want app.mpr", got)
	}
}

func TestExtractMPRFromArgs_EqualForm(t *testing.T) {
	got := extractMPRFromArgs([]string{"--project=app.mpr", "-c", "show"})
	if got != "app.mpr" {
		t.Errorf("got %q, want app.mpr", got)
	}
}

func TestExtractMPRFromArgs_None(t *testing.T) {
	got := extractMPRFromArgs([]string{"--help"})
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestExtractMPRFromArgs_DanglingFlag(t *testing.T) {
	// -p at end of slice (no value follows)
	got := extractMPRFromArgs([]string{"-p"})
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

// --- mprDaemonSocketPath ---

func TestMPRDaemonSocketPath_Deterministic(t *testing.T) {
	e := newTestEnv(t)
	a := e.mprDaemonSocketPath("/home/user/project.mpr")
	b := e.mprDaemonSocketPath("/home/user/project.mpr")
	if a != b {
		t.Errorf("same input produced different paths: %q vs %q", a, b)
	}
}

func TestMPRDaemonSocketPath_DifferentMPR(t *testing.T) {
	e := newTestEnv(t)
	a := e.mprDaemonSocketPath("/home/user/project-a.mpr")
	b := e.mprDaemonSocketPath("/home/user/project-b.mpr")
	if a == b {
		t.Errorf("different MPR paths produced same socket: %q", a)
	}
}

func TestMPRDaemonSocketPath_HasMprPrefix(t *testing.T) {
	e := newTestEnv(t)
	p := e.mprDaemonSocketPath("/some/project.mpr")
	base := filepath.Base(p)
	if !strings.HasPrefix(base, "mpr-") {
		t.Errorf("socket name %q does not start with mpr-", base)
	}
	if !strings.HasSuffix(base, ".sock") {
		t.Errorf("socket name %q does not end with .sock", base)
	}
}

func TestMPRDaemonSocketPath_InDaemonDir(t *testing.T) {
	e := newTestEnv(t)
	p := e.mprDaemonSocketPath("/some/project.mpr")
	if !strings.HasPrefix(p, e.daemonDir()) {
		t.Errorf("socket path %q is not under daemonDir %q", p, e.daemonDir())
	}
}

// --- mprSocketPrefix ---

func TestMPRSocketPrefix_SameForSameMPR(t *testing.T) {
	e := newTestEnv(t)
	a := e.mprSocketPrefix("/home/user/project.mpr")
	b := e.mprSocketPrefix("/home/user/project.mpr")
	if a != b {
		t.Errorf("same MPR produced different prefix: %q vs %q", a, b)
	}
}

func TestMPRSocketPrefix_DifferentForDifferentMPR(t *testing.T) {
	e := newTestEnv(t)
	a := e.mprSocketPrefix("/home/user/a.mpr")
	b := e.mprSocketPrefix("/home/user/b.mpr")
	if a == b {
		t.Errorf("different MPR produced same prefix: %q", a)
	}
}

// --- cleanupStaleMPRSockets ---

func TestCleanupStaleMPRSockets_RemovesSameMPROldSocket(t *testing.T) {
	e := newTestEnv(t)
	if err := os.MkdirAll(e.daemonDir(), 0755); err != nil {
		t.Fatal(err)
	}
	mprPath := "/home/user/project.mpr"

	// Create a fake stale socket with the same MPR prefix but different binary hash suffix.
	prefix := e.mprSocketPrefix(mprPath)
	stale := filepath.Join(e.daemonDir(), prefix+"aabbcc.sock")
	if err := os.WriteFile(stale, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	// current socket should NOT be deleted.
	current := e.mprDaemonSocketPath(mprPath)
	if err := os.WriteFile(current, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	e.cleanupStaleMPRSockets(mprPath, current)

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("stale same-MPR socket should have been removed")
	}
	if _, err := os.Stat(current); os.IsNotExist(err) {
		t.Error("current socket should NOT have been removed")
	}
}

func TestCleanupStaleMPRSockets_RemovesDeadOrphanSocket(t *testing.T) {
	e := newTestEnv(t)
	if err := os.MkdirAll(e.daemonDir(), 0755); err != nil {
		t.Fatal(err)
	}

	// Orphan socket from a different MPR (different prefix) that is NOT alive.
	orphan := filepath.Join(e.daemonDir(), "mpr-deadbeef-0a0b.sock")
	if err := os.WriteFile(orphan, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	current := e.mprDaemonSocketPath("/home/user/project.mpr")
	e.cleanupStaleMPRSockets("/home/user/project.mpr", current)

	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Error("dead orphan socket should have been removed")
	}
}

func TestCleanupStaleMPRSockets_KeepsNonSocketFiles(t *testing.T) {
	e := newTestEnv(t)
	if err := os.MkdirAll(e.daemonDir(), 0755); err != nil {
		t.Fatal(err)
	}

	// Non-.sock file in daemon dir should be untouched.
	other := filepath.Join(e.daemonDir(), "version")
	if err := os.WriteFile(other, []byte("v1"), 0644); err != nil {
		t.Fatal(err)
	}

	current := e.mprDaemonSocketPath("/some/project.mpr")
	e.cleanupStaleMPRSockets("/some/project.mpr", current)

	if _, err := os.Stat(other); os.IsNotExist(err) {
		t.Error("non-socket file should not have been removed")
	}
}

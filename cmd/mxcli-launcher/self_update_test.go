// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"testing"
	"time"

	"github.com/mendixlabs/mxcli/cmd/mxcli-launcher/testfixtures"
)

func writeTestFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "bin-*")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(content)
	f.Close()
	os.Chmod(f.Name(), 0755)
	return f.Name()
}

func TestRunInternalUpdate_WaitsForParentExit(t *testing.T) {
	waiter := testfixtures.NewFakePIDWaiter()
	oldBin := writeTestFile(t, "old-content")
	newBin := writeTestFile(t, "new-content")

	done := make(chan error, 1)
	go func() {
		done <- runInternalUpdate(99999, newBin, oldBin, waiter, 5*time.Second)
	}()

	// Before exit: old content must still be in place.
	time.Sleep(20 * time.Millisecond)
	got, _ := os.ReadFile(oldBin)
	if string(got) != "old-content" {
		t.Errorf("premature replacement: got %q", got)
	}

	// Simulate parent exit → updater should complete.
	waiter.SimulateExit()
	if err := <-done; err != nil {
		t.Fatalf("runInternalUpdate: %v", err)
	}
	got, _ = os.ReadFile(oldBin)
	if string(got) != "new-content" {
		t.Errorf("after exit: got %q, want new-content", got)
	}
}

func TestRunInternalUpdate_Timeout(t *testing.T) {
	waiter := testfixtures.NewFakePIDWaiter()
	oldBin := writeTestFile(t, "old-content")
	newBin := writeTestFile(t, "new-content")

	// Never signal exit → should time out.
	err := runInternalUpdate(99999, newBin, oldBin, waiter, 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	// Original file must be untouched.
	got, _ := os.ReadFile(oldBin)
	if string(got) != "old-content" {
		t.Errorf("after timeout: content changed to %q", got)
	}
}

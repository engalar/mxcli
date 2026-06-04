//go:build linux || darwin

// SPDX-License-Identifier: Apache-2.0

package testutil_test

import (
	"os"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/executor/testutil"
)

func TestMountMPR_RoundTrip(t *testing.T) {
	mprBytes, err := os.ReadFile("../../../testdata/expr-checker/minimal.mpr")
	if err != nil {
		t.Skipf("fixture not found: %v", err)
	}

	mount := testutil.MountMPR(t, mprBytes)
	if mount.Path() == "" {
		t.Fatal("MountMPR returned empty path")
	}

	info, err := os.Stat(mount.Path())
	if err != nil {
		t.Fatalf("Stat(%s): %v", mount.Path(), err)
	}
	if info.IsDir() {
		t.Fatal("expected file, got directory")
	}

	got := mount.Bytes()
	if len(got) != len(mprBytes) {
		t.Errorf("Bytes() len = %d, want %d", len(got), len(mprBytes))
	}
}

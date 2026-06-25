//go:build poc
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"testing"
)

// TestDriver_MinesFixtureMPR runs the full-corpus mining benchmark against a
// large MPR fixture. Because the fixture is developer-specific and the full
// run is slow (1500+ microflows), the test is OPT-IN only: set
// MXCLI_LARGE_FIXTURE_MPR to the fixture path. Default behavior is to skip,
// so `go test ./...` stays fast for CI and routine dev iteration.
//
// The fast TestDriver_MinesMinimalFixture in fixture_test.go always runs
// against the in-tree minimal fixture and is the regression-coverage path.
func TestDriver_MinesFixtureMPR(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow MPR mining test in -short mode (see fixture_test.go for the fast minimal-fixture variant)")
	}
	fixtureMPRPath := os.Getenv("MXCLI_LARGE_FIXTURE_MPR")
	if fixtureMPRPath == "" {
		t.Skip("MXCLI_LARGE_FIXTURE_MPR not set; skipping slow full-corpus mining benchmark (export MXCLI_LARGE_FIXTURE_MPR=/path/to/large.mpr to run)")
	}
	if _, err := os.Stat(fixtureMPRPath); err != nil {
		t.Skipf("MXCLI_LARGE_FIXTURE_MPR=%s does not exist: %v", fixtureMPRPath, err)
	}
	m := NewMiner()
	if err := MineMPR(m, fixtureMPRPath); err != nil {
		t.Fatalf("MineMPR: %v", err)
	}
	if len(m.Records) == 0 {
		t.Fatal("expected non-zero records mined from MPR")
	}
	var ifCount int
	for _, r := range m.Records {
		if r.SlotPath == "IfStmt.Condition" {
			ifCount++
		}
	}
	if ifCount < 50 {
		t.Errorf("expected >= 50 IfStmt.Condition records, got %d", ifCount)
	}
}

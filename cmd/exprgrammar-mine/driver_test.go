// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"testing"
)

const fixtureMPRPath = "/mnt/data_sdd/gh/Mx2026AIDay/Factory Management.mpr"

func TestDriver_MinesFixtureMPR(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow MPR mining test in -short mode (see fixture_test.go for the fast minimal-fixture variant)")
	}
	if _, err := os.Stat(fixtureMPRPath); err != nil {
		t.Skipf("full corpus fixture not present at %s; the fast variant in fixture_test.go uses testdata/expr-checker/minimal.mpr", fixtureMPRPath)
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

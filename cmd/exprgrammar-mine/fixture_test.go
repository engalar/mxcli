//go:build poc
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"path/filepath"
	"testing"
)

func TestDriver_MinesMinimalFixture(t *testing.T) {
	mprPath, _ := filepath.Abs("../../testdata/expr-checker/minimal.mpr")
	m := NewMiner()
	if err := MineMPR(m, mprPath); err != nil {
		t.Fatalf("MineMPR: %v", err)
	}
	t.Logf("mined %d records from minimal fixture", len(m.Records))
}

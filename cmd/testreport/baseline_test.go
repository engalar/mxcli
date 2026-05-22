// SPDX-License-Identifier: Apache-2.0
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBaselineRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bench-baseline.json")

	// empty baseline
	bl, err := loadBaseline(path)
	if err != nil {
		t.Fatalf("loadBaseline on missing file: %v", err)
	}
	if len(bl.Runs) != 0 {
		t.Fatalf("expected empty runs, got %d", len(bl.Runs))
	}

	// append a run
	bl.append("abc123", "BenchmarkFoo 100 10 ns/op")
	if err := saveBaseline(path, bl); err != nil {
		t.Fatalf("saveBaseline: %v", err)
	}

	// reload
	bl2, err := loadBaseline(path)
	if err != nil {
		t.Fatalf("loadBaseline after save: %v", err)
	}
	if len(bl2.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(bl2.Runs))
	}
	if bl2.Runs[0].GitHash != "abc123" {
		t.Errorf("expected hash abc123, got %q", bl2.Runs[0].GitHash)
	}
}

func TestBaselineSlidingWindow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bench-baseline.json")

	bl, _ := loadBaseline(path)
	for i := 0; i < 12; i++ {
		bl.append("hash", "data")
	}
	if err := saveBaseline(path, bl); err != nil {
		t.Fatalf("saveBaseline: %v", err)
	}
	bl2, _ := loadBaseline(path)
	if len(bl2.Runs) != 10 {
		t.Errorf("expected sliding window of 10, got %d", len(bl2.Runs))
	}
}

func TestLoadBaseline_MissingFile(t *testing.T) {
	bl, err := loadBaseline(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if bl == nil || len(bl.Runs) != 0 {
		t.Error("expected empty baseline for missing file")
	}
}

func TestBaselinePath(t *testing.T) {
	_ = os.MkdirAll("../../coverage", 0o755)
	// just ensure defaultBaselinePath() returns a non-empty string
	p := defaultBaselinePath()
	if p == "" {
		t.Error("defaultBaselinePath returned empty string")
	}
}

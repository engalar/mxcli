// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const maxBaselineRuns = 10

// BenchRun は1回のbenchmark実行結果。
type BenchRun struct {
	Timestamp time.Time `json:"timestamp"`
	GitHash   string    `json:"git_hash"`
	Results   string    `json:"results"` // raw benchstat-compatible text
}

// Baseline holds the sliding window of recent benchmark runs.
type Baseline struct {
	Runs []BenchRun `json:"runs"`
}

// append adds a new run and trims to maxBaselineRuns.
func (b *Baseline) append(gitHash, results string) {
	b.Runs = append(b.Runs, BenchRun{
		Timestamp: time.Now().UTC(),
		GitHash:   gitHash,
		Results:   results,
	})
	if len(b.Runs) > maxBaselineRuns {
		b.Runs = b.Runs[len(b.Runs)-maxBaselineRuns:]
	}
}

// loadBaseline reads the baseline file, returning an empty Baseline if the
// file does not exist.
func loadBaseline(path string) (*Baseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Baseline{}, nil
		}
		return nil, err
	}
	var bl Baseline
	if err := json.Unmarshal(data, &bl); err != nil {
		return &Baseline{}, nil // treat corrupt file as empty
	}
	return &bl, nil
}

// saveBaseline writes the baseline to path (creating parent dirs as needed).
func saveBaseline(path string, bl *Baseline) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(bl, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// defaultBaselinePath returns the path to coverage/bench-baseline.json
// relative to the repo root (two levels up from cmd/testreport/).
func defaultBaselinePath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "coverage/bench-baseline.json"
	}
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	return filepath.Join(repoRoot, "coverage", "bench-baseline.json")
}

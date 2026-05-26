// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// TestIndex maps "package/TestName" → layerID.
// Built by scanning source files so classification works even though
// go test -json does not include file names in its output.
type TestIndex map[string]string

// scanTestIndex walks repoRoot and classifies every test function by the
// source file it lives in, using the same suffix rules as classifyTest.
// repoRoot should be the repository root (where go.mod lives).
func scanTestIndex(repoRoot string) TestIndex {
	idx := TestIndex{}

	// map of glob-like suffix rules for executor/ and backend/mpr/:
	//   suffix (lowercase) → layerID
	type rule struct {
		suffix  string
		prefix  string // for roundtrip_ prefix check
		layerID string
	}
	executorRules := []rule{
		{suffix: "describe_sanity_test.go", layerID: "L6b"},
		{prefix: "roundtrip_", layerID: "L6a"},
		{suffix: "_gen_test.go", layerID: "L4"},
		{suffix: "_mock_test.go", layerID: "L3"},
		{suffix: "_bench_test.go", layerID: "bench"},
	}
	backendRules := []rule{
		{suffix: "_bench_test.go", layerID: "bench"},
		// any file whose name contains "roundtrip" → L5
	}
	_ = backendRules

	dirs := map[string]string{
		"mdl/visitor":     "L1L2",
		"mdl/executor":    "", // classified per-file
		"mdl/backend/mpr": "", // classified per-file
	}

	for rel, defaultLayer := range dirs {
		absDir := filepath.Join(repoRoot, rel)
		entries, err := os.ReadDir(absDir)
		if err != nil {
			continue
		}

		// derive the Go import path prefix from go.mod module name
		modulePkg := moduleName(repoRoot)

		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			fname := strings.ToLower(e.Name())

			// determine layer for this file
			layerID := defaultLayer
			if defaultLayer == "" {
				// executor or backend/mpr: apply per-file rules
				if strings.Contains(rel, "executor") {
					layerID = "other"
					for _, r := range executorRules {
						if r.prefix != "" && strings.HasPrefix(fname, r.prefix) {
							layerID = r.layerID
							break
						}
						if r.suffix != "" && (strings.HasSuffix(fname, r.suffix) || fname == r.suffix) {
							layerID = r.layerID
							break
						}
					}
				} else if strings.Contains(rel, "backend/mpr") {
					layerID = "L5"
					if strings.HasSuffix(fname, "_bench_test.go") {
						layerID = "bench"
					}
				}
			}

			if layerID == "" || layerID == "other" {
				continue // no need to index "other" — it's the fallback
			}

			// scan the file for test function names
			pkgImport := modulePkg + "/" + rel
			tests := extractTestNames(filepath.Join(absDir, e.Name()))
			for _, t := range tests {
				idx[pkgImport+"/"+t] = layerID
			}
		}
	}
	return idx
}

// extractTestNames returns all top-level Test*/Benchmark* function names
// declared in the given Go source file.
func extractTestNames(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var names []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "func Test") && !strings.HasPrefix(line, "func Benchmark") {
			continue
		}
		// extract name: "func TestFoo(t *testing.T)" → "TestFoo"
		after := strings.TrimPrefix(line, "func ")
		paren := strings.Index(after, "(")
		if paren < 0 {
			continue
		}
		names = append(names, after[:paren])
	}
	return names
}

// moduleName reads the module name from go.mod.
func moduleName(repoRoot string) string {
	f, err := os.Open(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		return "github.com/mendixlabs/mxcli"
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return "github.com/mendixlabs/mxcli"
}

// SPDX-License-Identifier: Apache-2.0
package main

import "strings"

// layerDef は報告の中で使われる層の識別子。
type layerDef struct {
	ID       string
	Name     string
	Priority int // 表示順
}

var layers = []layerDef{
	{"L1L2", "L1/L2 Parser+Visitor", 1},
	{"L3", "L3 Executor Mock", 2},
	{"L4", "L4 Executor Gen", 3},
	{"L5", "L5 Decode", 4},
	{"L6a", "L6a Roundtrip", 5},
	{"L6b", "L6b Describe Sanity", 6},
	{"bench", "Benchmark", 7},
	{"other", "Other", 8},
}

// classifyTest maps a package path + test file name to a layer ID and display name.
// testFile may be empty (package-level result without file info).
func classifyTest(pkg, testFile string) (id, name string) {
	f := strings.ToLower(testFile)

	// bench files first (can appear in any package)
	if strings.HasSuffix(f, "_bench_test.go") {
		return "bench", "Benchmark"
	}

	// visitor package: always L1/L2
	if strings.Contains(pkg, "/mdl/visitor") {
		return "L1L2", "L1/L2 Parser+Visitor"
	}

	// executor package: classify by suffix
	if strings.Contains(pkg, "/mdl/executor") {
		switch {
		case f == "describe_sanity_test.go":
			return "L6b", "L6b Describe Sanity"
		case strings.HasPrefix(f, "roundtrip_"):
			return "L6a", "L6a Roundtrip"
		case strings.HasSuffix(f, "_gen_test.go"):
			return "L4", "L4 Executor Gen"
		case strings.HasSuffix(f, "_mock_test.go"):
			return "L3", "L3 Executor Mock"
		}
	}

	// backend/mpr: classify by suffix
	if strings.Contains(pkg, "/mdl/backend/mpr") {
		if strings.Contains(f, "roundtrip") {
			return "L5", "L5 Decode"
		}
	}

	return "other", "Other"
}

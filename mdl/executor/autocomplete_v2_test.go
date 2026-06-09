// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5a tests: autocomplete_gen.go gen-typed LSP completion.

package executor

import (
	"sort"
	"testing"
)

// TestGetMicroflowNamesACGen verifies the gen-typed autocomplete
// provider returns the same set as the legacy sdk-typed version.
// Connected() requires Backend != nil, which the describe context
// satisfies via the parallel sdk/mpr.Writer wired in newGenDescribeContext.
func TestGetMicroflowNamesACGen(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)

	got := getMicroflowNamesACGen(ctx, "")
	want := getMicroflowNamesAC(ctx, "")

	if len(got) == 0 {
		t.Fatalf("getMicroflowNamesACGen returned 0 entries; expected fixture to contain microflows")
	}
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("entry count mismatch: gen=%d legacy=%d\n  gen: %v\n  legacy: %v", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("entry %d mismatch: gen=%q legacy=%q", i, got[i], want[i])
		}
	}
}

// TestGetMicroflowNamesACGen_ModuleFilter verifies the module filter
// path returns only that module's microflows.
func TestGetMicroflowNamesACGen_ModuleFilter(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)

	all := getMicroflowNamesACGen(ctx, "")
	if len(all) == 0 {
		t.Fatalf("no microflows in fixture")
	}

	// Pick a module from the first qualified name.
	firstQN := all[0]
	dot := -1
	for i, c := range firstQN {
		if c == '.' {
			dot = i
			break
		}
	}
	if dot < 0 {
		t.Fatalf("first QN %q has no dot", firstQN)
	}
	module := firstQN[:dot]

	filtered := getMicroflowNamesACGen(ctx, module)
	if len(filtered) == 0 {
		t.Fatalf("module filter %q returned 0 entries", module)
	}
	for _, qn := range filtered {
		if len(qn) <= len(module) || qn[:len(module)] != module || qn[len(module)] != '.' {
			t.Errorf("filter %q matched out-of-module entry %q", module, qn)
		}
	}
}

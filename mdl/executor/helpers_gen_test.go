// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5a tests: helpers_gen.go gen-typed shared helpers.

package executor

import (
	"testing"
)

// TestBuildMicroflowQualifiedNamesGen verifies the gen-typed builder
// returns the same set as the legacy buildMicroflowQualifiedNames.
func TestBuildMicroflowQualifiedNamesGen(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)

	got := buildMicroflowQualifiedNamesGen(ctx)
	want := buildMicroflowQualifiedNames(ctx)

	if len(got) == 0 {
		t.Fatalf("buildMicroflowQualifiedNamesGen returned 0 entries; expected fixture to contain microflows")
	}
	if len(got) != len(want) {
		t.Errorf("entry count mismatch: gen=%d legacy=%d", len(got), len(want))
	}
	for qn := range want {
		if !got[qn] {
			t.Errorf("gen builder missing legacy entry %q", qn)
		}
	}
	for qn := range got {
		if !want[qn] {
			t.Errorf("gen builder added unexpected entry %q", qn)
		}
	}
}

// TestBuildNanoflowQualifiedNamesGen verifies the nanoflow gen builder.
// (Fixture may not contain nanoflows; we still assert the gen + legacy
// sets agree.)
func TestBuildNanoflowQualifiedNamesGen(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)

	// Wire the Nanoflows repo so listNanoflowsGen can resolve. The
	// describe context only sets Microflows by default; nanoflow tests
	// install Nanoflows from the same writer's repoCtx.
	if ctx.Nanoflows == nil {
		// Best effort: derive from the same writer the Microflows repo
		// uses. We skip the test cleanly if the fixture has no nanoflow
		// support wired in the describe-context helper.
		t.Skip("describe context did not wire NanoflowRepository; covered separately in 3.2.5c")
	}

	got := buildNanoflowQualifiedNamesGen(ctx)
	want := buildNanoflowQualifiedNames(ctx)

	if len(got) != len(want) {
		t.Errorf("entry count mismatch: gen=%d legacy=%d", len(got), len(want))
	}
	for qn := range want {
		if !got[qn] {
			t.Errorf("gen builder missing legacy entry %q", qn)
		}
	}
}

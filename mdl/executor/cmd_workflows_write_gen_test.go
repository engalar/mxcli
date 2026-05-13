// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5b tests for the gen-typed workflow write helpers.

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/sdk/workflows"
)

// TestAutoBindCallMicroflowGen_DefaultOutcomeInjected asserts the
// gen-typed auto-binder still injects a default VoidConditionOutcome
// when the task has no outcomes (CE6686 parity with the legacy
// autoBindCallMicroflow).
func TestAutoBindCallMicroflowGen_DefaultOutcomeInjected(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)

	task := &workflows.CallMicroflowTask{
		Microflow: "MyFirstModule.UnknownMicroflow",
	}
	task.Name = "  CallMF  " // sanitize will trim/clean

	autoBindCallMicroflowGen(ctx, task)

	if len(task.Outcomes) != 1 {
		t.Fatalf("expected 1 default VoidConditionOutcome, got %d", len(task.Outcomes))
	}
	if _, ok := task.Outcomes[0].(*workflows.VoidConditionOutcome); !ok {
		t.Errorf("expected VoidConditionOutcome, got %T", task.Outcomes[0])
	}
	if task.Outcomes[0].(*workflows.VoidConditionOutcome).Flow == nil {
		t.Errorf("VoidConditionOutcome must have non-nil Flow (BSON ID required)")
	}
}

// TestAutoBindCallMicroflowGen_SkipsExistingMappings asserts the binder
// is a no-op for ParameterMappings when explicit mappings already exist.
func TestAutoBindCallMicroflowGen_SkipsExistingMappings(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)

	existingMapping := &workflows.ParameterMapping{
		Parameter:  "MyFirstModule.SomeMF.Foo",
		Expression: "$Custom",
	}
	task := &workflows.CallMicroflowTask{
		Microflow:         "MyFirstModule.MyFirstLogic",
		ParameterMappings: []*workflows.ParameterMapping{existingMapping},
	}

	autoBindCallMicroflowGen(ctx, task)

	if len(task.ParameterMappings) != 1 {
		t.Fatalf("expected 1 ParameterMapping (explicit), got %d", len(task.ParameterMappings))
	}
	if task.ParameterMappings[0] != existingMapping {
		t.Errorf("explicit mapping was replaced; auto-bind should be skipped")
	}
}

// TestAutoBindCallMicroflowGen_MissingMicroflow asserts the binder
// silently no-ops when the named microflow does not exist (matches
// legacy fallback behaviour — best-effort, no error returned).
func TestAutoBindCallMicroflowGen_MissingMicroflow(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)

	task := &workflows.CallMicroflowTask{
		Microflow: "MyFirstModule.DoesNotExist_Stage3_2_5b",
	}

	autoBindCallMicroflowGen(ctx, task)

	if len(task.ParameterMappings) != 0 {
		t.Errorf("expected no ParameterMappings for unknown microflow, got %d", len(task.ParameterMappings))
	}
	// Default outcome should still have been injected.
	if len(task.Outcomes) != 1 {
		t.Errorf("expected default outcome injection regardless of lookup result, got %d", len(task.Outcomes))
	}
}

// TestGenMicroflowParameterNames_KnownMicroflow asserts the parameter-
// name extractor returns a stable result for an existing microflow in
// the fixture. Whether it returns 0 or N parameters depends on the
// fixture; we only assert the path completes and returns a slice
// (possibly nil) without panicking.
func TestGenMicroflowParameterNames_KnownMicroflow(t *testing.T) {
	w := openMprWriterForTest(t)
	mf := findMicroflowByQN(t, w, "MyFirstModule.MyFirstLogic")

	names := genMicroflowParameterNames(mf)
	for _, n := range names {
		if n == "" {
			t.Errorf("parameter name should not be empty; got %v", names)
		}
	}
}

// TestGenMicroflowParameterNames_NilSafe asserts the helper handles
// a nil receiver without panicking.
func TestGenMicroflowParameterNames_NilSafe(t *testing.T) {
	if got := genMicroflowParameterNames(nil); got != nil {
		t.Errorf("expected nil for nil microflow, got %v", got)
	}
}

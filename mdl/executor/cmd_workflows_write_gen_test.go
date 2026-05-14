// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.3.E0 — gen-typed parity tests for the autoBind helper.
// Replaces the legacy autoBindCallMicroflowGen tests (which still
// operated on sdk *workflows.CallMicroflowTask) with the gen-native
// autoBindCallMicroflowGenActivity sibling shipped in Stage 3.3.3.D2.

package executor

import (
	"testing"

	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// TestAutoBindCallMicroflowGen_DefaultOutcomeInjected asserts the
// gen-typed auto-binder still injects a default VoidConditionOutcome
// when the activity has no outcomes (CE6686 parity with the legacy
// autoBindCallMicroflow).
func TestAutoBindCallMicroflowGen_DefaultOutcomeInjected(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)

	act := genWf.NewCallMicroflowActivity()
	act.SetMicroflowQualifiedName("MyFirstModule.UnknownMicroflow")
	act.SetName("  CallMF  ") // sanitize will trim/clean

	autoBindCallMicroflowGenActivity(ctx, act)

	outcomes := act.OutcomesItems()
	if len(outcomes) != 1 {
		t.Fatalf("expected 1 default VoidConditionOutcome, got %d", len(outcomes))
	}
	oc, ok := outcomes[0].(*genWf.VoidConditionOutcome)
	if !ok {
		t.Errorf("expected VoidConditionOutcome, got %T", outcomes[0])
	}
	if oc != nil && oc.Flow() == nil {
		t.Errorf("VoidConditionOutcome must have non-nil Flow (BSON ID required)")
	}
}

// TestAutoBindCallMicroflowGen_SkipsExistingMappings asserts the binder
// is a no-op for ParameterMappings when explicit mappings already exist.
func TestAutoBindCallMicroflowGen_SkipsExistingMappings(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)

	existingMapping := genWf.NewMicroflowCallParameterMapping()
	existingMapping.SetParameterQualifiedName("MyFirstModule.SomeMF.Foo")
	existingMapping.SetExpression("$Custom")

	act := genWf.NewCallMicroflowActivity()
	act.SetMicroflowQualifiedName("MyFirstModule.MyFirstLogic")
	act.AddParameterMappings(existingMapping)

	autoBindCallMicroflowGenActivity(ctx, act)

	mappings := act.ParameterMappingsItems()
	if len(mappings) != 1 {
		t.Fatalf("expected 1 ParameterMapping (explicit), got %d", len(mappings))
	}
	if mappings[0] != existingMapping {
		t.Errorf("explicit mapping was replaced; auto-bind should be skipped")
	}
}

// TestAutoBindCallMicroflowGen_MissingMicroflow asserts the binder
// silently no-ops when the named microflow does not exist (matches
// legacy fallback behaviour — best-effort, no error returned).
func TestAutoBindCallMicroflowGen_MissingMicroflow(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)

	act := genWf.NewCallMicroflowActivity()
	act.SetMicroflowQualifiedName("MyFirstModule.DoesNotExist_Stage3_2_5b")

	autoBindCallMicroflowGenActivity(ctx, act)

	if mappings := act.ParameterMappingsItems(); len(mappings) != 0 {
		t.Errorf("expected no ParameterMappings for unknown microflow, got %d", len(mappings))
	}
	// Default outcome should still have been injected.
	if outcomes := act.OutcomesItems(); len(outcomes) != 1 {
		t.Errorf("expected default outcome injection regardless of lookup result, got %d", len(outcomes))
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

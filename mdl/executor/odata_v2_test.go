// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5b tests for the gen-typed OData consumer scan.

package executor

import (
	"bytes"
	"strings"
	"testing"
)

// TestListExternalActionsGen_NoActions asserts the gen-typed scan
// runs to completion against a fixture with no CallExternalAction
// references and emits the empty-state message in non-JSON mode.
//
// The expr-checker fixture does not contain any CallExternalAction
// activities, so this exercises the empty branch end-to-end.
func TestListExternalActionsGen_NoActions(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenVizContext(t, &bytes.Buffer{})

	if err := listExternalActionsGen(ctx, ""); err != nil {
		t.Fatalf("listExternalActionsGen: %v", err)
	}
	out := ctx.Output.(*bytes.Buffer).String()
	if !strings.Contains(out, "No external actions found") {
		t.Errorf("expected empty-state message; got %q", out)
	}
	_ = w
}

// TestListExternalActionsGen_ModuleFilter asserts the IN module
// filter is honoured and that an unknown module name still produces
// the empty-state message rather than an error.
func TestListExternalActionsGen_ModuleFilter(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenVizContext(t, &bytes.Buffer{})

	if err := listExternalActionsGen(ctx, "DoesNotExistModule_Stage3_2_5b"); err != nil {
		t.Fatalf("listExternalActionsGen with unknown module: %v", err)
	}
	out := ctx.Output.(*bytes.Buffer).String()
	if !strings.Contains(out, "No external actions found") {
		t.Errorf("expected empty-state message for unknown module; got %q", out)
	}
	_ = w
}

// TestListExternalActionsGen_JSONFormat asserts that JSON mode
// returns a TableResult even when there are no actions (no
// "No external actions found" text in JSON mode — the empty table
// is a legitimate response).
func TestListExternalActionsGen_JSONFormat(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenVizContext(t, &bytes.Buffer{})
	ctx.Format = FormatJSON

	if err := listExternalActionsGen(ctx, ""); err != nil {
		t.Fatalf("listExternalActionsGen JSON: %v", err)
	}
	out := ctx.Output.(*bytes.Buffer).String()
	if strings.Contains(out, "No external actions found") {
		t.Errorf("JSON mode must not include the human-readable empty message; got %q", out)
	}
	_ = w
}

// TestExternalActionParameterNames_NilSafe covers the nil-receiver
// branch of the parameter-name extractor.
func TestExternalActionParameterNames_NilSafe(t *testing.T) {
	if got := externalActionParameterNames(nil); got != nil {
		t.Errorf("expected nil for nil receiver, got %v", got)
	}
}

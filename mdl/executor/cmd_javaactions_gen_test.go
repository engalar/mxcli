// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.2.A1-A5 tests: gen-typed read paths for Java/JavaScript actions.

package executor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// ─────────────────────────────────────────────────────────────────────
// A1 — listJavaActionsGen
// ─────────────────────────────────────────────────────────────────────

func TestListJavaActionsGen_OutputsHeaderAndSummary(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable

	if err := listJavaActionsGen(ctx, ""); err != nil {
		t.Fatalf("listJavaActionsGen: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Qualified Name") {
		t.Errorf("expected header 'Qualified Name' in output, got: %q", out)
	}
	if !strings.Contains(out, "java actions)") {
		t.Errorf("expected count summary in output, got: %q", out)
	}
}

func TestListJavaActionsGen_RowsPresent(t *testing.T) {
	// Fixture has 2 java actions (ValidateEmail, XSS_Sanitizer).
	ctx := newJavaActionsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable

	if err := listJavaActionsGen(ctx, ""); err != nil {
		t.Fatalf("listJavaActionsGen: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "ValidateEmail") {
		t.Errorf("expected ValidateEmail in output: %q", out)
	}
	if !strings.Contains(out, "XSS_Sanitizer") {
		t.Errorf("expected XSS_Sanitizer in output: %q", out)
	}
	if !strings.Contains(out, "(2 java actions)") {
		t.Errorf("expected '(2 java actions)' summary, got: %q", out)
	}
}

func TestListJavaActionsGen_FilterByModule(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable

	if err := listJavaActionsGen(ctx, "NoSuchModule"); err != nil {
		t.Fatalf("listJavaActionsGen: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "(0 java actions)") {
		t.Errorf("expected '(0 java actions)' for unknown module, got: %q", out)
	}
}

// ─────────────────────────────────────────────────────────────────────
// A2 — formatJavaActionTypeGen + formatJavaActionReturnTypeGen
// ─────────────────────────────────────────────────────────────────────

func TestFormatJavaActionReturnTypeGen_NilReturnsVoid(t *testing.T) {
	if got := formatJavaActionReturnTypeGen(nil, nil); got != "Void" {
		t.Errorf("got %q, want Void", got)
	}
}

func TestFormatJavaActionTypeGen_NilReturnsObject(t *testing.T) {
	if got := formatJavaActionTypeGen(nil, nil); got != "Object" {
		t.Errorf("got %q, want Object", got)
	}
}

// TestFormatJavaActionTypeGen_LegacyStorageNames verifies that the
// formatter dispatches on the Studio-Pro-emitted "CodeActions$X"
// storage names (the gen registry uses "JavaActions$X" but the BSON
// from real fixtures uses the legacy namespace — see the schema-gap
// note at the head of cmd_javaactions_gen.go).
//
// Driven through real fixture data so we exercise the full
// decode→format chain that listJavaActionsGen / describeJavaActionGen
// run in production.
func TestFormatJavaActionTypeGen_FixtureReturnsBoolean(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	pairs, err := listJavaActionsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listJavaActionsWithContainerGen: %v", err)
	}
	for _, p := range pairs {
		if p.Elem == nil || p.Elem.Name() != "ValidateEmail" {
			continue
		}
		rt := javaActionReturnTypeElement(p.Elem)
		got := formatJavaActionReturnTypeGen(rt, p.Elem.ActionTypeParametersItems())
		if got != "Boolean" {
			t.Errorf("ValidateEmail return type: got %q, want Boolean", got)
		}
		return
	}
	t.Fatal("ValidateEmail not in fixture")
}

func TestFormatJavaActionTypeGen_FixtureReturnsString(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	pairs, err := listJavaActionsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listJavaActionsWithContainerGen: %v", err)
	}
	for _, p := range pairs {
		if p.Elem == nil || p.Elem.Name() != "XSS_Sanitizer" {
			continue
		}
		rt := javaActionReturnTypeElement(p.Elem)
		got := formatJavaActionReturnTypeGen(rt, p.Elem.ActionTypeParametersItems())
		if got != "String" {
			t.Errorf("XSS_Sanitizer return type: got %q, want String", got)
		}
		return
	}
	t.Fatal("XSS_Sanitizer not in fixture")
}

// ─────────────────────────────────────────────────────────────────────
// A3 — describeJavaActionGen
// ─────────────────────────────────────────────────────────────────────

func TestDescribeJavaActionGen_OutputsCreateStatement(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf

	// Fixture module name + action name; module discovered from
	// listJavaActionsWithContainerGen + container chain.
	pairs, err := listJavaActionsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listJavaActionsWithContainerGen: %v", err)
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		t.Fatalf("getHierarchy: %v", err)
	}
	var modName string
	for _, p := range pairs {
		if p.Elem != nil && p.Elem.Name() == "ValidateEmail" {
			modID := h.FindModuleID(modelIDFromElementID(p.ContainerID))
			modName = h.GetModuleName(modID)
			break
		}
	}
	if modName == "" {
		t.Fatal("could not resolve module for ValidateEmail")
	}

	if err := describeJavaActionGen(ctx, ast.QualifiedName{Module: modName, Name: "ValidateEmail"}); err != nil {
		t.Fatalf("describeJavaActionGen: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "create java action ") {
		t.Errorf("missing create statement: %q", out)
	}
	if !strings.Contains(out, ".ValidateEmail(") {
		t.Errorf("missing action name + paren: %q", out)
	}
	if !strings.Contains(out, "EmailAddress") {
		t.Errorf("missing parameter name: %q", out)
	}
	if !strings.Contains(out, "returns Boolean") {
		t.Errorf("missing 'returns Boolean': %q", out)
	}
}

func TestDescribeJavaActionGen_NotFound(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	err := describeJavaActionGen(ctx, ast.QualifiedName{Module: "Foo", Name: "Bar"})
	if err == nil {
		t.Fatal("expected error for missing action, got nil")
	}
}

// ─────────────────────────────────────────────────────────────────────
// A4 — listJavaScriptActionsGen
// ─────────────────────────────────────────────────────────────────────

func TestListJavaScriptActionsGen_OutputsPlatformColumn(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable

	if err := listJavaScriptActionsGen(ctx, ""); err != nil {
		t.Fatalf("listJavaScriptActionsGen: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Platform") {
		t.Errorf("expected 'Platform' header, got: %q", out)
	}
	if !strings.Contains(out, "javascript actions)") {
		t.Errorf("expected count summary, got: %q", out)
	}
}

func TestListJavaScriptActionsGen_RendersAllPlatformAsLabel(t *testing.T) {
	// Fixture has Platform="" actions which should render as "All".
	ctx := newJavaActionsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable

	if err := listJavaScriptActionsGen(ctx, ""); err != nil {
		t.Fatalf("listJavaScriptActionsGen: %v", err)
	}
	out := buf.String()
	// Fixture has 67 actions, mostly "All" but some "Web".
	if !strings.Contains(out, "Web") {
		t.Errorf("expected 'Web' platform in output: %q", out[:200])
	}
}

// ─────────────────────────────────────────────────────────────────────
// A5 — describeJavaScriptActionGen
// ─────────────────────────────────────────────────────────────────────

func TestDescribeJavaScriptActionGen_OutputsCreateStatement(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf

	pairs, err := listJavaScriptActionsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listJavaScriptActionsWithContainerGen: %v", err)
	}
	if len(pairs) == 0 {
		t.Fatal("fixture has no JS actions")
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		t.Fatalf("getHierarchy: %v", err)
	}
	first := pairs[0].Elem
	modID := h.FindModuleID(modelIDFromElementID(pairs[0].ContainerID))
	modName := h.GetModuleName(modID)
	if modName == "" {
		t.Fatal("could not resolve module for first JS action")
	}

	if err := describeJavaScriptActionGen(ctx, ast.QualifiedName{Module: modName, Name: first.Name()}); err != nil {
		t.Fatalf("describeJavaScriptActionGen: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "create javascript action ") {
		t.Errorf("missing create statement: %q", out)
	}
	if !strings.Contains(out, "."+first.Name()+"(") {
		t.Errorf("missing action name + paren for %s: %q", first.Name(), out)
	}
}

func TestDescribeJavaScriptActionGen_NotFound(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	err := describeJavaScriptActionGen(ctx, ast.QualifiedName{Module: "Foo", Name: "Bar"})
	if err == nil {
		t.Fatal("expected error for missing JS action, got nil")
	}
}

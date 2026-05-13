// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// newLoopFB builds a minimal flowBuilder seeded with a List parameter so
// that the loop variable type can be derived from the list element type.
func newLoopFB() *flowBuilder {
	return &flowBuilder{
		spacing:      HorizontalSpacing,
		varTypes:     map[string]string{"Items": "List of WF_Engine.WFApplication"},
		declaredVars: map[string]string{},
		measurer:     &layoutMeasurer{varTypes: map[string]string{"Items": "List of WF_Engine.WFApplication"}},
	}
}

// loopStmt builds a minimal LoopStmt AST node.
func loopStmt(loopVar, listVar string) *ast.LoopStmt {
	return &ast.LoopStmt{
		LoopVariable: loopVar,
		ListVariable: listVar,
	}
}

// --- Rejection tests ---

// TestCE0111_TwoFlatLoops_SameVar: two consecutive loops with $E → error.
func TestCE0111_TwoFlatLoops_SameVar(t *testing.T) {
	fb := newLoopFB()

	// First loop: registers $E in varTypes — must succeed.
	id1 := fb.addLoopStatement(loopStmt("E", "Items"))
	if id1 == "" {
		t.Fatal("first loop should succeed")
	}
	if errs := fb.GetErrors(); len(errs) != 0 {
		t.Fatalf("first loop should not produce errors, got: %v", errs)
	}

	// Second loop: $E already in varTypes — must be rejected.
	id2 := fb.addLoopStatement(loopStmt("E", "Items"))
	if id2 != "" {
		t.Fatal("second loop with duplicate var should return empty ID")
	}
	errs := fb.GetErrors()
	if len(errs) == 0 {
		t.Fatal("expected a CE0111 error, got none")
	}
	if !strings.Contains(errs[0], "CE0111") {
		t.Errorf("error should mention CE0111, got: %q", errs[0])
	}
}

// TestCE0111_TwoFlatLoops_DifferentVars: two loops with different names → OK.
func TestCE0111_TwoFlatLoops_DifferentVars(t *testing.T) {
	fb := newLoopFB()
	fb.varTypes["Others"] = "List of WF_Engine.WFApplication"

	fb.addLoopStatement(loopStmt("E", "Items"))
	fb.addLoopStatement(loopStmt("V", "Others"))

	if errs := fb.GetErrors(); len(errs) != 0 {
		t.Errorf("different var names should not produce errors, got: %v", errs)
	}
}

// TestCE0111_DeclareThenLoop_SameName: declare $X then loop $X → no error
// (loop shadows declared variable; Mendix allows this).
func TestCE0111_DeclareThenLoop_SameName(t *testing.T) {
	fb := newLoopFB()
	// Seed declaredVars as if a prior DECLARE $E String happened.
	fb.declaredVars["E"] = "String"

	fb.addLoopStatement(loopStmt("E", "Items"))

	if errs := fb.GetErrors(); len(errs) != 0 {
		t.Errorf("loop shadowing a declare should not produce CE0111, got: %v", errs)
	}
}

// --- Hint tests ---

// TestCE0111_HintDerivesEntityName: hint contains the entity short name + "Item".
func TestCE0111_HintDerivesEntityName(t *testing.T) {
	fb := newLoopFB()
	// Register $E once so the second call conflicts.
	fb.addLoopStatement(loopStmt("E", "Items"))
	fb.errors = nil // reset to isolate hint test

	// Manually seed $E in varTypes to force conflict on next call.
	// (addLoopStatement already added it; clear errors then call again.)
	fb.addLoopStatement(loopStmt("E", "Items"))

	errs := fb.GetErrors()
	if len(errs) == 0 {
		t.Fatal("expected CE0111 error")
	}
	if !strings.Contains(errs[0], "WFApplicationItem") {
		t.Errorf("hint should suggest $WFApplicationItem, got: %q", errs[0])
	}
}

// TestCE0111_HintFallback_UnknownListType: when list type is unknown, hint uses var+"2".
func TestCE0111_HintFallback_UnknownListType(t *testing.T) {
	fb := &flowBuilder{
		spacing:      HorizontalSpacing,
		varTypes:     map[string]string{"E": "WF_Engine.WFApplication"}, // E already registered, no list type
		declaredVars: map[string]string{},
		measurer:     &layoutMeasurer{varTypes: map[string]string{}},
	}

	fb.addLoopStatement(loopStmt("E", "UnknownList"))

	errs := fb.GetErrors()
	if len(errs) == 0 {
		t.Fatal("expected CE0111 error")
	}
	if !strings.Contains(errs[0], "E2") {
		t.Errorf("fallback hint should suggest $E2, got: %q", errs[0])
	}
}

// --- Scope isolation tests ---

// TestCE0111_LoopBuilderDoesNotLeakVarToOuter: after addLoopStatement,
// a variable declared INSIDE the loop body is NOT visible in the outer scope.
// This tests that loopBuilder uses cloned maps.
func TestCE0111_LoopBuilderScopeIsolation(t *testing.T) {
	fb := newLoopFB()

	stmt := &ast.LoopStmt{
		LoopVariable: "Item",
		ListVariable: "Items",
		Body: []ast.MicroflowStatement{
			&ast.DeclareStmt{
				Variable: "InnerVar",
				Type:     ast.DataType{Kind: ast.TypeString},
			},
		},
	}
	fb.addLoopStatement(stmt)

	// InnerVar declared inside loop body must NOT appear in outer scope.
	if _, found := fb.declaredVars["InnerVar"]; found {
		t.Error("InnerVar declared inside loop body should not leak to outer scope")
	}
	// Loop iterator $Item SHOULD appear in outer scope (needed for Mendix type inference).
	if _, found := fb.varTypes["Item"]; !found {
		t.Error("loop iterator Item should be registered in outer varTypes")
	}
}

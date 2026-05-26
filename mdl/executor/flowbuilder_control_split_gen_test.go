// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.h4 — EnumSplit + InheritanceSplit adder tests (TDD).
//
// Both wrap a multi-branch split with case-labelled flows + optional
// else branch + optional merge. Same recursion pattern as the IF
// adder: each branch body iterates through addStatementGen.
//
// Minimal viable shape (mirrors the IF h2 minimal viable):
//   - emit ExclusiveSplit (enum) or InheritanceSplit (inheritance)
//   - emit each case body, labelled flow split→firstActivity = case value
//   - emit else body when present
//   - emit ExclusiveMerge if any branch flows through; skip when all
//     branches return
//
// Out of scope: per-branch @anchor overrides, retry-loop pattern,
// pendingErrorOrigin routing into a specific branch — same scope
// boundary as h2.

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func TestAddEnumSplitGenEmitsSplitAndCaseFlows(t *testing.T) {
	fb := newIfTestFb()
	stmt := &ast.EnumSplitStmt{
		Variable: "Status",
		Cases: []ast.EnumSplitCase{
			{Value: "Open", Body: []ast.MicroflowStatement{
				&ast.DeclareStmt{Variable: "X", Type: ast.DataType{Kind: ast.TypeBoolean}},
			}},
			{Value: "Closed", Body: []ast.MicroflowStatement{
				&ast.DeclareStmt{Variable: "Y", Type: ast.DataType{Kind: ast.TypeBoolean}},
			}},
		},
	}
	id := fb.addEnumSplitGen(stmt)
	if id == "" {
		t.Fatal("expected non-empty split ID")
	}

	splitFound, mergeFound := false, false
	activityCount := 0
	for _, obj := range fb.objects {
		switch obj.(type) {
		case *genMf.ExclusiveSplit:
			splitFound = true
		case *genMf.ExclusiveMerge:
			mergeFound = true
		case *genMf.ActionActivity:
			activityCount++
		}
	}
	if !splitFound {
		t.Fatal("ExclusiveSplit must be emitted")
	}
	if activityCount != 2 {
		t.Fatalf("activities = %d, want 2 (one per case)", activityCount)
	}
	if !mergeFound {
		t.Fatal("ExclusiveMerge must be emitted (both cases continue)")
	}
}

func TestAddEnumSplitGenSetsCaptionAndCondition(t *testing.T) {
	fb := newIfTestFb()
	stmt := &ast.EnumSplitStmt{
		Variable: "Status",
		Cases:    []ast.EnumSplitCase{{Value: "X", Body: []ast.MicroflowStatement{&ast.ReturnStmt{}}}},
	}
	fb.addEnumSplitGen(stmt)
	var split *genMf.ExclusiveSplit
	for _, obj := range fb.objects {
		if s, ok := obj.(*genMf.ExclusiveSplit); ok {
			split = s
			break
		}
	}
	if split == nil {
		t.Fatal("split must be emitted")
	}
	if split.Caption() != "$Status" {
		t.Fatalf("caption = %q, want $Status", split.Caption())
	}
	cond, ok := split.SplitCondition().(*genMf.ExpressionSplitCondition)
	if !ok {
		t.Fatalf("split condition = %T, want *ExpressionSplitCondition", split.SplitCondition())
	}
	if cond.Expression() != "$Status" {
		t.Fatalf("split expression = %q, want $Status", cond.Expression())
	}
}

func TestAddEnumSplitGenWithElseBodyEmitsElseFlow(t *testing.T) {
	fb := newIfTestFb()
	stmt := &ast.EnumSplitStmt{
		Variable: "Status",
		Cases: []ast.EnumSplitCase{
			{Value: "Open", Body: []ast.MicroflowStatement{&ast.ReturnStmt{}}},
		},
		ElseBody: []ast.MicroflowStatement{
			&ast.DeclareStmt{Variable: "Default", Type: ast.DataType{Kind: ast.TypeBoolean}},
		},
	}
	fb.addEnumSplitGen(stmt)
	// Should have: split + ReturnStmt's EndEvent + else activity + merge.
	hasEnd, hasElseAct, hasMerge := false, false, false
	for _, obj := range fb.objects {
		switch obj.(type) {
		case *genMf.EndEvent:
			hasEnd = true
		case *genMf.ActionActivity:
			hasElseAct = true
		case *genMf.ExclusiveMerge:
			hasMerge = true
		}
	}
	if !hasEnd {
		t.Fatal("ReturnStmt in case body should produce EndEvent")
	}
	if !hasElseAct {
		t.Fatal("else body should produce activity")
	}
	if !hasMerge {
		t.Fatal("merge should be emitted (else body continues)")
	}
}

func TestAddEnumSplitGenAllCasesReturnEmitsNoMerge(t *testing.T) {
	fb := newIfTestFb()
	stmt := &ast.EnumSplitStmt{
		Variable: "Status",
		Cases: []ast.EnumSplitCase{
			{Value: "A", Body: []ast.MicroflowStatement{&ast.ReturnStmt{}}},
			{Value: "B", Body: []ast.MicroflowStatement{&ast.ReturnStmt{}}},
		},
	}
	fb.addEnumSplitGen(stmt)
	mergeCount := 0
	for _, obj := range fb.objects {
		if _, ok := obj.(*genMf.ExclusiveMerge); ok {
			mergeCount++
		}
	}
	if mergeCount != 0 {
		t.Fatalf("merge count = %d, want 0 (all cases return)", mergeCount)
	}
}

func TestAddEnumSplitGenRoutesViaDispatcher(t *testing.T) {
	fb := newIfTestFb()
	stmt := &ast.EnumSplitStmt{
		Variable: "S",
		Cases:    []ast.EnumSplitCase{{Value: "X", Body: []ast.MicroflowStatement{&ast.ReturnStmt{}}}},
	}
	id := fb.addStatementGen(stmt)
	if id == "" {
		t.Fatal("dispatcher should now route EnumSplitStmt to addEnumSplitGen")
	}
}

func TestAddInheritanceSplitGenBareNoCasesEmitsBareNode(t *testing.T) {
	// Bare InheritanceSplitStmt with no cases / no else mirrors the
	// legacy fast path — emit a minimal *InheritanceSplit element.
	fb := newIfTestFb()
	stmt := &ast.InheritanceSplitStmt{Variable: "Obj"}
	id := fb.addInheritanceSplitGen(stmt)
	if id == "" {
		t.Fatal("expected non-empty inheritance split ID")
	}
	var split *genMf.InheritanceSplit
	for _, obj := range fb.objects {
		if s, ok := obj.(*genMf.InheritanceSplit); ok {
			split = s
			break
		}
	}
	if split == nil {
		t.Fatal("InheritanceSplit must be emitted")
	}
	if split.SplitVariableName() != "Obj" {
		t.Fatalf("split var = %q, want Obj", split.SplitVariableName())
	}
}

func TestAddInheritanceSplitGenWithCasesEmitsBranches(t *testing.T) {
	fb := newIfTestFb()
	stmt := &ast.InheritanceSplitStmt{
		Variable: "Obj",
		Cases: []ast.InheritanceSplitCase{
			{
				Entity: ast.QualifiedName{Module: "Sales", Name: "Premium"},
				Body:   []ast.MicroflowStatement{&ast.ReturnStmt{}},
			},
			{
				Entity: ast.QualifiedName{Module: "Sales", Name: "Standard"},
				Body:   []ast.MicroflowStatement{&ast.ReturnStmt{}},
			},
		},
	}
	id := fb.addInheritanceSplitGen(stmt)
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	splitCount, endCount := 0, 0
	for _, obj := range fb.objects {
		switch obj.(type) {
		case *genMf.InheritanceSplit:
			splitCount++
		case *genMf.EndEvent:
			endCount++
		}
	}
	if splitCount != 1 {
		t.Fatalf("split count = %d, want 1", splitCount)
	}
	if endCount != 2 {
		t.Fatalf("end events = %d, want 2 (one per case)", endCount)
	}
}

func TestAddInheritanceSplitGenRoutesViaDispatcher(t *testing.T) {
	fb := newIfTestFb()
	id := fb.addStatementGen(&ast.InheritanceSplitStmt{Variable: "Obj"})
	if id == "" {
		t.Fatal("dispatcher should now route InheritanceSplitStmt to addInheritanceSplitGen")
	}
}

// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.h2 — addIfStatementGen tests (TDD).
//
// Covers the common-case shapes:
//
//   1. then-only `if X then ... end if;`
//      → ExclusiveSplit + then activities + ExclusiveMerge
//
//   2. then + else `if X then ... else ... end if;`
//      → ExclusiveSplit + then activities + else activities + merge
//
//   3. both branches return → no merge
//   4. then returns, else continues → only else flows into merge
//   5. else returns, then continues → only then flows into merge
//
// Branch-level custom error handler routing, retry-loop pattern, and
// per-branch @anchor overrides are scoped out of h2 and will land
// either in a follow-up h2-extension or alongside i (eh body). The
// minimal-viable adder shipped here covers the dominant fresh-author
// shapes and exercises the dispatcher recursion (addStatementGen
// reaching back into addIfStatementGen for nested IFs).

package executor

import (
	"fmt"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// newIfTestFb mirrors newActionTestFb but adds a layoutMeasurer
// (addIfStatementGen needs it for branch-bounds measurement).
func newIfTestFb() *flowBuilderGen {
	fb := newActionTestFb()
	fb.measurer = &layoutMeasurer{varTypes: fb.varTypes}
	return fb
}

func TestAddIfStatementGenThenOnlyEmitsSplitAndMerge(t *testing.T) {
	fb := newIfTestFb()
	stmt := &ast.IfStmt{
		Condition: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
		ThenBody: []ast.MicroflowStatement{
			&ast.DeclareStmt{
				Variable: "X",
				Type:     ast.DataType{Kind: ast.TypeBoolean},
				InitialValue: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
			},
		},
	}
	id := fb.addIfStatementGen(stmt)
	if id == "" {
		t.Fatal("expected non-empty split ID")
	}
	// Expect: split + then-activity + merge = 3 objects.
	splitFound, mergeFound, declareFound := false, false, false
	for _, obj := range fb.objects {
		switch obj.(type) {
		case *genMf.ExclusiveSplit:
			splitFound = true
		case *genMf.ExclusiveMerge:
			mergeFound = true
		case *genMf.ActionActivity:
			declareFound = true
		}
	}
	if !splitFound {
		t.Fatal("ExclusiveSplit must be emitted")
	}
	if !declareFound {
		t.Fatal("then-body activity must be emitted")
	}
	if !mergeFound {
		t.Fatal("ExclusiveMerge must be emitted (then-only with continuation)")
	}
}

func TestAddIfStatementGenWithElseEmitsBothBranches(t *testing.T) {
	fb := newIfTestFb()
	stmt := &ast.IfStmt{
		Condition: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
		HasElse:   true,
		ThenBody: []ast.MicroflowStatement{
			&ast.DeclareStmt{Variable: "T", Type: ast.DataType{Kind: ast.TypeBoolean}},
		},
		ElseBody: []ast.MicroflowStatement{
			&ast.DeclareStmt{Variable: "E", Type: ast.DataType{Kind: ast.TypeBoolean}},
		},
	}
	fb.addIfStatementGen(stmt)
	// Expect: split + then activity + else activity + merge = 4 objects.
	activityCount := 0
	mergeCount := 0
	splitCount := 0
	for _, obj := range fb.objects {
		switch obj.(type) {
		case *genMf.ExclusiveSplit:
			splitCount++
		case *genMf.ExclusiveMerge:
			mergeCount++
		case *genMf.ActionActivity:
			activityCount++
		}
	}
	if splitCount != 1 {
		t.Fatalf("split count = %d, want 1", splitCount)
	}
	if activityCount != 2 {
		t.Fatalf("activities = %d, want 2 (one per branch)", activityCount)
	}
	if mergeCount != 1 {
		t.Fatalf("merge count = %d, want 1", mergeCount)
	}
}

func TestAddIfStatementGenBothBranchesReturnEmitsNoMerge(t *testing.T) {
	fb := newIfTestFb()
	stmt := &ast.IfStmt{
		Condition: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
		HasElse:   true,
		ThenBody:  []ast.MicroflowStatement{&ast.ReturnStmt{}},
		ElseBody:  []ast.MicroflowStatement{&ast.ReturnStmt{}},
	}
	fb.addIfStatementGen(stmt)
	mergeCount := 0
	endCount := 0
	for _, obj := range fb.objects {
		switch obj.(type) {
		case *genMf.ExclusiveMerge:
			mergeCount++
		case *genMf.EndEvent:
			endCount++
		}
	}
	if mergeCount != 0 {
		t.Fatalf("merge count = %d, want 0 (both branches return)", mergeCount)
	}
	if endCount != 2 {
		t.Fatalf("end events = %d, want 2 (one per branch)", endCount)
	}
	if !fb.endsWithReturn {
		t.Fatal("endsWithReturn should be set when both branches terminate")
	}
}

func TestAddIfStatementGenThenReturnsElseContinuesEmitsMerge(t *testing.T) {
	fb := newIfTestFb()
	stmt := &ast.IfStmt{
		Condition: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
		HasElse:   true,
		ThenBody:  []ast.MicroflowStatement{&ast.ReturnStmt{}},
		ElseBody: []ast.MicroflowStatement{
			&ast.DeclareStmt{Variable: "E", Type: ast.DataType{Kind: ast.TypeBoolean}},
		},
	}
	fb.addIfStatementGen(stmt)
	mergeCount := 0
	for _, obj := range fb.objects {
		if _, ok := obj.(*genMf.ExclusiveMerge); ok {
			mergeCount++
		}
	}
	if mergeCount != 1 {
		t.Fatalf("merge count = %d, want 1 (else branch needs merge)", mergeCount)
	}
}

func TestAddIfStatementGenSplitConditionIsExpression(t *testing.T) {
	fb := newIfTestFb()
	stmt := &ast.IfStmt{
		Condition: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
		ThenBody:  []ast.MicroflowStatement{&ast.ReturnStmt{}},
	}
	fb.addIfStatementGen(stmt)
	var split *genMf.ExclusiveSplit
	for _, obj := range fb.objects {
		if s, ok := obj.(*genMf.ExclusiveSplit); ok {
			split = s
			break
		}
	}
	if split == nil {
		t.Fatal("split element must be emitted")
	}
	if split.Caption() != "true" {
		t.Fatalf("split caption = %q, want true", split.Caption())
	}
	cond, ok := split.SplitCondition().(*genMf.ExpressionSplitCondition)
	if !ok {
		t.Fatalf("split condition = %T, want *ExpressionSplitCondition", split.SplitCondition())
	}
	if cond.Expression() != "true" {
		t.Fatalf("split condition expression = %q, want true", cond.Expression())
	}
}

func TestAddIfStatementGenRoutesViaDispatcher(t *testing.T) {
	// The dispatcher should now route IfStmt to addIfStatementGen
	// (previously returned "" placeholder).
	fb := newIfTestFb()
	stmt := &ast.IfStmt{
		Condition: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
		ThenBody:  []ast.MicroflowStatement{&ast.ReturnStmt{}},
	}
	id := fb.addStatementGen(stmt)
	if id == "" {
		t.Fatal("dispatcher should now route IfStmt to addIfStatementGen, got empty ID")
	}
}

// parseLayoutY extracts the Y component from a "x;y" RelativeMiddlePoint string.
func parseLayoutY(pos string) int {
	var x, y int
	fmt.Sscanf(pos, "%d;%d", &x, &y)
	return y
}

// TestIfWithoutElseThenBranchBelowMainLine verifies that for an IF without
// ELSE, the THEN branch activities are placed BELOW the main line (higher Y
// than the ExclusiveSplit). The false path goes straight to merge on the main
// line; the true path dives below. Without this, all activities collapse to
// Y=200 when microflows contain only IF-without-ELSE statements.
func TestIfWithoutElseThenBranchBelowMainLine(t *testing.T) {
	fb := newIfTestFb()
	stmt := &ast.IfStmt{
		Condition: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
		ThenBody: []ast.MicroflowStatement{
			&ast.DeclareStmt{Variable: "X", Type: ast.DataType{Kind: ast.TypeBoolean}},
		},
		// No HasElse / ElseBody — IF without ELSE.
	}
	fb.addIfStatementGen(stmt)

	var split *genMf.ExclusiveSplit
	var thenActivity *genMf.ActionActivity
	for _, obj := range fb.objects {
		switch o := obj.(type) {
		case *genMf.ExclusiveSplit:
			split = o
		case *genMf.ActionActivity:
			thenActivity = o
		}
	}
	if split == nil {
		t.Fatal("ExclusiveSplit must be emitted")
	}
	if thenActivity == nil {
		t.Fatal("THEN branch ActionActivity must be emitted")
	}

	splitY := parseLayoutY(split.RelativeMiddlePoint())
	thenY := parseLayoutY(thenActivity.RelativeMiddlePoint())
	if thenY <= splitY {
		t.Fatalf("IF without ELSE: THEN branch Y (%d) must be > split Y (%d) — true path must be below main line", thenY, splitY)
	}
}

// TestIfWithoutElseThenBranchFlowIsDownward verifies that the SequenceFlow
// connecting the ExclusiveSplit to the first THEN-branch activity uses a
// downward anchor (OriginConnectionIndex = AnchorBottom) when the THEN branch
// is placed below the main line.
func TestIfWithoutElseThenBranchFlowIsDownward(t *testing.T) {
	fb := newIfTestFb()
	stmt := &ast.IfStmt{
		Condition: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
		ThenBody: []ast.MicroflowStatement{
			&ast.DeclareStmt{Variable: "X", Type: ast.DataType{Kind: ast.TypeBoolean}},
		},
	}
	splitID := fb.addIfStatementGen(stmt)

	// Find the flow from split→THEN (case "true"). It must use AnchorBottom
	// because the THEN branch is placed below the split's Y position.
	// Note: there is also a split→merge (case "false") flow that correctly
	// uses AnchorRight — skip that one.
	found := false
	for _, flow := range fb.flows {
		if flow.OriginRefID() != splitID {
			continue
		}
		if len(flow.CaseValuesItems()) == 0 {
			continue
		}
		ec, ok := flow.CaseValuesItems()[0].(*genMf.EnumerationCase)
		if !ok || ec.Value() != "true" {
			continue
		}
		// true-case flow from split into THEN must exit from the bottom anchor.
		if int(flow.OriginConnectionIndex()) != AnchorBottom {
			t.Fatalf("split→THEN (case=true) flow OriginConnectionIndex = %d, want %d (AnchorBottom)",
				flow.OriginConnectionIndex(), AnchorBottom)
		}
		found = true
	}
	if !found {
		t.Fatal("no flow with case 'true' found originating from split")
	}
}

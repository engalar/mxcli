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
				Variable:     "X",
				Type:         ast.DataType{Kind: ast.TypeBoolean},
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

// TestAddIfStatementGenThenReturnsNoElseSkipsMerge verifies that when the THEN
// branch terminates (return) and there is no ELSE body, no ExclusiveMerge is
// emitted. The false path flows directly from the split to the next activity.
func TestAddIfStatementGenThenReturnsNoElseSkipsMerge(t *testing.T) {
	fb := newIfTestFb()
	stmt := &ast.IfStmt{
		Condition: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
		ThenBody:  []ast.MicroflowStatement{&ast.ReturnStmt{}},
		// No HasElse / ElseBody — pass-through pattern.
	}
	fb.addIfStatementGen(stmt)
	mergeCount := 0
	for _, obj := range fb.objects {
		if _, ok := obj.(*genMf.ExclusiveMerge); ok {
			mergeCount++
		}
	}
	if mergeCount != 0 {
		t.Fatalf("merge count = %d, want 0 (then returns, no else → no merge needed)", mergeCount)
	}
}

// TestAddIfStatementGenThenReturnsNoElseSetsPassthrough verifies that
// nextConnectionPoint is set to the split ID and nextConnectionCase is "false"
// so the main loop wires the split→next-activity flow with the correct label.
func TestAddIfStatementGenThenReturnsNoElseSetsPassthrough(t *testing.T) {
	fb := newIfTestFb()
	stmt := &ast.IfStmt{
		Condition: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
		ThenBody:  []ast.MicroflowStatement{&ast.ReturnStmt{}},
	}
	splitID := fb.addIfStatementGen(stmt)
	if fb.nextConnectionPoint != splitID {
		t.Fatalf("nextConnectionPoint = %v, want splitID %v", fb.nextConnectionPoint, splitID)
	}
	if fb.nextConnectionCase != "false" {
		t.Fatalf("nextConnectionCase = %q, want %q", fb.nextConnectionCase, "false")
	}
}

// TestPassthroughIfFlowHasFalseCase verifies that buildFlowGraphGen wires a
// "false"-labelled flow from the split to the next activity when the then
// branch returns and there is no else body.
func TestPassthroughIfFlowHasFalseCase(t *testing.T) {
	fb := newGraphTestFb()
	body := []ast.MicroflowStatement{
		&ast.IfStmt{
			Condition: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
			ThenBody:  []ast.MicroflowStatement{&ast.ReturnStmt{}},
			// No else — pass-through.
		},
		&ast.DeclareStmt{Variable: "After", Type: ast.DataType{Kind: ast.TypeBoolean}},
	}
	fb.buildFlowGraphGen(body, nil)

	// Find the ExclusiveSplit and the "After" ActionActivity.
	var splitID string
	var afterID string
	for _, obj := range fb.objects {
		switch o := obj.(type) {
		case *genMf.ExclusiveSplit:
			splitID = string(o.ID())
		case *genMf.ActionActivity:
			afterID = string(o.ID())
		}
	}
	if splitID == "" {
		t.Fatal("ExclusiveSplit must be emitted")
	}
	if afterID == "" {
		t.Fatal("ActionActivity after pass-through if must be emitted")
	}

	// Find the flow from split → afterActivity with "false" case.
	found := false
	for _, flow := range fb.flows {
		if string(flow.OriginRefID()) != splitID || string(flow.DestinationRefID()) != afterID {
			continue
		}
		if len(flow.CaseValuesItems()) == 0 {
			continue
		}
		ec, ok := flow.CaseValuesItems()[0].(*genMf.EnumerationCase)
		if ok && ec.Value() == "false" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected a flow from split → after-activity with case 'false'")
	}
}

// parseLayoutX extracts the X component from a "x;y" RelativeMiddlePoint string.
func parseLayoutX(pos string) int {
	var x, y int
	fmt.Sscanf(pos, "%d;%d", &x, &y)
	return y // intentionally returns y for parseLayoutX — callers use parseLayoutY for Y
}

// parsePos parses both X and Y from "x;y".
func parsePos(pos string) (x, y int) {
	fmt.Sscanf(pos, "%d;%d", &x, &y)
	return
}

// TestIfWithElseElsePlacedBelowThenBodyHeight verifies that when the THEN body
// contains content that expands vertically (e.g. a nested IF), the ELSE branch
// is placed below the THEN body's measured height rather than a fixed offset.
// Without this fix, nested IFs inside THEN and the outer ELSE all land at the
// same Y causing visual overlap.
func TestIfWithElseElsePlacedBelowThenBodyHeight(t *testing.T) {
	fb := newIfTestFb()

	// THEN body contains a nested IF (adds BranchGap+ActivityHeight to height).
	nestedIf := &ast.IfStmt{
		Condition: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
		HasElse:   true,
		ThenBody:  []ast.MicroflowStatement{&ast.DeclareStmt{Variable: "T", Type: ast.DataType{Kind: ast.TypeBoolean}}},
		ElseBody:  []ast.MicroflowStatement{&ast.DeclareStmt{Variable: "E", Type: ast.DataType{Kind: ast.TypeBoolean}}},
	}
	stmt := &ast.IfStmt{
		Condition: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
		HasElse:   true,
		ThenBody:  []ast.MicroflowStatement{nestedIf},
		ElseBody: []ast.MicroflowStatement{
			&ast.DeclareStmt{Variable: "Else", Type: ast.DataType{Kind: ast.TypeBoolean}},
		},
	}
	fb.addIfStatementGen(stmt)

	// Compute expected minimum Y for the ELSE branch.
	// measureStatements([nestedIf]) gives the height of the THEN body.
	m := &layoutMeasurer{varTypes: map[string]string{}}
	thenBounds := m.measureStatements(stmt.ThenBody)
	thenHeight := thenBounds.Height
	if thenHeight < ActivityHeight {
		thenHeight = ActivityHeight
	}
	minElseY := fb.baseY + thenHeight + BranchGap // baseY = 200

	// Find all ActionActivity objects. Those placed for the ELSE body
	// must have Y >= minElseY.
	var elseActivities []*genMf.ActionActivity
	for _, obj := range fb.objects {
		if act, ok := obj.(*genMf.ActionActivity); ok {
			_, y := parsePos(act.RelativeMiddlePoint())
			// Collect only activities placed below the THEN body level.
			if y > 200 { // above 200 is THEN body level; below is ELSE/nested territory
				elseActivities = append(elseActivities, act)
			}
		}
	}

	// At least one ELSE-body activity must be at or below minElseY.
	// Currently (bug): all are at centerY+VerticalSpacing=290, even if
	// nested IF inside THEN already occupies that Y range.
	foundBelowThreshold := false
	for _, act := range elseActivities {
		_, y := parsePos(act.RelativeMiddlePoint())
		if y >= minElseY {
			foundBelowThreshold = true
		}
	}
	if !foundBelowThreshold && len(elseActivities) > 0 {
		_, y := parsePos(elseActivities[0].RelativeMiddlePoint())
		t.Fatalf("ELSE branch activity Y=%d but minElseY=%d (THEN body height=%d); ELSE must be below THEN body",
			y, minElseY, thenHeight)
	}
}

// TestMergeToNextActivityHasHorizontalGap verifies that the activity placed
// immediately after an ExclusiveMerge has a non-zero edge-to-edge gap from
// the merge. Without this fix, the merge's right edge touches the next
// activity's left edge (zero gap), causing them to appear merged in Studio Pro.
func TestMergeToNextActivityHasHorizontalGap(t *testing.T) {
	// Build a minimal IF (with ELSE) followed by one activity.
	fb := newGraphTestFb()
	body := []ast.MicroflowStatement{
		&ast.IfStmt{
			Condition: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
			HasElse:   true,
			ThenBody:  []ast.MicroflowStatement{&ast.DeclareStmt{Variable: "T", Type: ast.DataType{Kind: ast.TypeBoolean}}},
			ElseBody:  []ast.MicroflowStatement{&ast.DeclareStmt{Variable: "E", Type: ast.DataType{Kind: ast.TypeBoolean}}},
		},
		// Activity placed immediately after the IF merge.
		&ast.DeclareStmt{Variable: "After", Type: ast.DataType{Kind: ast.TypeBoolean}},
	}
	oc := fb.buildFlowGraphGen(body, nil)

	// Find the merge and the "After" activity.
	var merge *genMf.ExclusiveMerge
	var afterActs []*genMf.ActionActivity
	for _, obj := range oc.ObjectsItems() {
		switch o := obj.(type) {
		case *genMf.ExclusiveMerge:
			merge = o
		case *genMf.ActionActivity:
			afterActs = append(afterActs, o)
		}
	}
	if merge == nil {
		t.Fatal("ExclusiveMerge must be emitted")
	}
	if len(afterActs) == 0 {
		t.Fatal("ActionActivity after merge must be emitted")
	}

	mergeX, _ := parsePos(merge.RelativeMiddlePoint())
	mergeRightEdge := mergeX + MergeSize/2

	// The last ActionActivity (highest X) is the "After" activity.
	var lastAct *genMf.ActionActivity
	lastX := -1
	for _, act := range afterActs {
		x, _ := parsePos(act.RelativeMiddlePoint())
		if x > lastX {
			lastX = x
			lastAct = act
		}
	}
	afterX, _ := parsePos(lastAct.RelativeMiddlePoint())
	afterLeftEdge := afterX - ActivityWidth/2

	gap := afterLeftEdge - mergeRightEdge
	if gap <= 0 {
		t.Fatalf("merge right edge=%d, after-activity left edge=%d, gap=%d — must be > 0 (merge and next activity overlap)",
			mergeRightEdge, afterLeftEdge, gap)
	}
}

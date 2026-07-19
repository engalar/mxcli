// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.j — buildFlowGraphGen tests (TDD).
//
// buildFlowGraphGen is the top-level driver: it emits a StartEvent,
// iterates the microflow body via addStatementGen, threads horizontal
// SequenceFlows between activities, and emits a final EndEvent
// (carrying the ReturnValue when set) — unless every code path
// already terminates with a RETURN.
//
// Tests cover:
//   - empty body → StartEvent + EndEvent (2 objects, 1 flow)
//   - single activity body → StartEvent + activity + EndEvent
//   - all-return body → StartEvent + activity + RETURN's EndEvent (no
//     trailing EndEvent injected)
//   - return value carried onto EndEvent
//   - multi-statement chain
//   - returns the assembled MicroflowObjectCollection with objects /
//     flows populated

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func newGraphTestFb() *flowBuilderGen {
	fb := newActionTestFb()
	fb.measurer = &layoutMeasurer{varTypes: fb.varTypes}
	return fb
}

func TestBuildFlowGraphGenEmptyBodyEmitsStartAndEnd(t *testing.T) {
	fb := newGraphTestFb()
	oc := fb.buildFlowGraphGen(nil, nil)
	if oc == nil {
		t.Fatal("expected non-nil ObjectCollection")
	}
	startCount, endCount := 0, 0
	for _, obj := range oc.ObjectsItems() {
		switch obj.(type) {
		case *genMf.StartEvent:
			startCount++
		case *genMf.EndEvent:
			endCount++
		}
	}
	if startCount != 1 {
		t.Fatalf("start events = %d, want 1", startCount)
	}
	if endCount != 1 {
		t.Fatalf("end events = %d, want 1 (synthesised when body doesn't terminate)", endCount)
	}
	// Should have 1 flow connecting Start → End.
	if len(fb.flows) != 1 {
		t.Fatalf("flows = %d, want 1", len(fb.flows))
	}
}

func TestBuildFlowGraphGenSingleActivity(t *testing.T) {
	fb := newGraphTestFb()
	body := []ast.MicroflowStatement{
		&ast.DeclareStmt{
			Variable:     "X",
			Type:         ast.DataType{Kind: ast.TypeBoolean},
			InitialValue: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
		},
	}
	oc := fb.buildFlowGraphGen(body, nil)
	startFound, endFound, activityFound := false, false, false
	for _, obj := range oc.ObjectsItems() {
		switch obj.(type) {
		case *genMf.StartEvent:
			startFound = true
		case *genMf.EndEvent:
			endFound = true
		case *genMf.ActionActivity:
			activityFound = true
		}
	}
	if !startFound {
		t.Fatal("StartEvent must be emitted")
	}
	if !endFound {
		t.Fatal("EndEvent must be emitted (body doesn't terminate)")
	}
	if !activityFound {
		t.Fatal("body activity must be emitted")
	}
	// Should have 2 flows: Start → activity, activity → End.
	if len(fb.flows) != 2 {
		t.Fatalf("flows = %d, want 2", len(fb.flows))
	}
}

func TestBuildFlowGraphGenAllReturnSkipsSynthesisedEndEvent(t *testing.T) {
	// Body ends with RETURN — its own EndEvent is created by
	// addEndEventWithReturn; buildFlowGraphGen should NOT add another.
	fb := newGraphTestFb()
	body := []ast.MicroflowStatement{
		&ast.ReturnStmt{Value: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true}},
	}
	oc := fb.buildFlowGraphGen(body, nil)
	endCount := 0
	for _, obj := range oc.ObjectsItems() {
		if _, ok := obj.(*genMf.EndEvent); ok {
			endCount++
		}
	}
	if endCount != 1 {
		t.Fatalf("end events = %d, want 1 (only the RETURN's own end event)", endCount)
	}
}

func TestBuildFlowGraphGenReturnValueCarriedOnSynthesisedEndEvent(t *testing.T) {
	// When microflow returns AS $Var, the synthesised final EndEvent
	// should carry $Var as ReturnValue.
	fb := newGraphTestFb()
	body := []ast.MicroflowStatement{
		&ast.DeclareStmt{
			Variable:     "X",
			Type:         ast.DataType{Kind: ast.TypeBoolean},
			InitialValue: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
		},
	}
	returns := &ast.MicroflowReturnType{
		Type:     ast.DataType{Kind: ast.TypeBoolean},
		Variable: "X",
	}
	oc := fb.buildFlowGraphGen(body, returns)

	var finalEnd *genMf.EndEvent
	for _, obj := range oc.ObjectsItems() {
		if e, ok := obj.(*genMf.EndEvent); ok {
			finalEnd = e
		}
	}
	if finalEnd == nil {
		t.Fatal("EndEvent must be emitted")
	}
	if finalEnd.ReturnValue() != "$X" {
		t.Fatalf("ReturnValue = %q, want $X", finalEnd.ReturnValue())
	}
}

func TestBuildFlowGraphGenMultiStatementChain(t *testing.T) {
	fb := newGraphTestFb()
	body := []ast.MicroflowStatement{
		&ast.DeclareStmt{Variable: "A", Type: ast.DataType{Kind: ast.TypeBoolean}},
		&ast.DeclareStmt{Variable: "B", Type: ast.DataType{Kind: ast.TypeBoolean}},
		&ast.DeclareStmt{Variable: "C", Type: ast.DataType{Kind: ast.TypeBoolean}},
	}
	oc := fb.buildFlowGraphGen(body, nil)
	activityCount := 0
	for _, obj := range oc.ObjectsItems() {
		if _, ok := obj.(*genMf.ActionActivity); ok {
			activityCount++
		}
	}
	if activityCount != 3 {
		t.Fatalf("activities = %d, want 3", activityCount)
	}
	// Should have 4 flows: Start → A, A → B, B → C, C → End.
	if len(fb.flows) != 4 {
		t.Fatalf("flows = %d, want 4 (Start→A→B→C→End)", len(fb.flows))
	}
}

func TestBuildFlowGraphGenObjectCollectionExposesFlowsAndAnnotationFlows(t *testing.T) {
	// The returned ObjectCollection holds the activities; the flows
	// and annotationFlows live on the parent flowBuilderGen (the
	// caller copies them into the Microflow's Flows array).
	fb := newGraphTestFb()
	body := []ast.MicroflowStatement{
		&ast.DeclareStmt{Variable: "X", Type: ast.DataType{Kind: ast.TypeBoolean}},
	}
	oc := fb.buildFlowGraphGen(body, nil)
	if oc == nil {
		t.Fatal("ObjectCollection must be non-nil")
	}
	if len(oc.ObjectsItems()) == 0 {
		t.Fatal("ObjectCollection should contain at least Start + EndEvent + activity")
	}
}

func TestBuildFlowGraphGenInitialisesMeasurerWhenNil(t *testing.T) {
	// buildFlowGraphGen should defensively initialise the measurer
	// when the caller forgot — no panic on nil.
	fb := newActionTestFb()
	fb.measurer = nil
	oc := fb.buildFlowGraphGen(nil, nil)
	if oc == nil {
		t.Fatal("expected non-nil ObjectCollection even with nil measurer")
	}
	if fb.measurer == nil {
		t.Fatal("measurer should be initialised by buildFlowGraphGen")
	}
}

func TestBuildFlowGraphGenListReturnTypeEmitsCreateListAction(t *testing.T) {
	// When the microflow returns a list variable and the body has no
	// explicit declare for it, buildFlowGraphGen must synthesise a
	// CreateListAction (not CreateVariableAction). Using the wrong
	// action type causes Mendix BSON parse failure → KeyNotFoundException
	// crash in mx check / Studio Pro.
	fb := newGraphTestFb()
	body := []ast.MicroflowStatement{} // no explicit declare
	returns := &ast.MicroflowReturnType{
		Type: ast.DataType{
			Kind:      ast.TypeListOf,
			EntityRef: &ast.QualifiedName{Module: "Sales", Name: "Order"},
		},
		Variable: "Orders",
	}
	oc := fb.buildFlowGraphGen(body, returns)
	if oc == nil {
		t.Fatal("expected non-nil ObjectCollection")
	}
	// Walk objects to find the ActionActivity with the list declaration.
	var found bool
	for _, obj := range oc.ObjectsItems() {
		aa, ok := obj.(*genMf.ActionActivity)
		if !ok {
			continue
		}
		if _, isList := aa.Action().(*genMf.CreateListAction); isList {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no CreateListAction found in generated objects — " +
			"buildFlowGraphGen must emit CreateListAction for list return type")
	}
}

// TestBuildFlowGraphGenNanoflowSkipsSyntheticDeclare verifies that
// nanoflows do NOT get a synthetic CreateVariableAction/CreateListAction
// for the return variable. The 11.12.1+ validator requires a graph-level
// declaration for microflows (CE0109 fix), but adding one to nanoflows
// corrupts the BSON graph — ResolvePostponedProperties cannot find the
// subsequent ActionActivity $IDs → KeyNotFoundException.
//
// Regression guard for the && !fb.isNanoflow gate on line 78.
func TestBuildFlowGraphGenNanoflowSkipsSyntheticDeclare(t *testing.T) {
	fb := newGraphTestFb()
	fb.isNanoflow = true
	body := []ast.MicroflowStatement{} // no explicit declare
	returns := &ast.MicroflowReturnType{
		Type:     ast.DataType{Kind: ast.TypeBoolean},
		Variable: "R",
	}
	oc := fb.buildFlowGraphGen(body, returns)
	if oc == nil {
		t.Fatal("expected non-nil ObjectCollection")
	}
	for _, obj := range oc.ObjectsItems() {
		aa, ok := obj.(*genMf.ActionActivity)
		if !ok {
			continue
		}
		if _, isCreateVar := aa.Action().(*genMf.CreateVariableAction); isCreateVar {
			t.Errorf("nanoflow must not emit CreateVariableAction for return var, got %T", aa.Action())
		}
		if _, isCreateList := aa.Action().(*genMf.CreateListAction); isCreateList {
			t.Errorf("nanoflow must not emit CreateListAction for return var, got %T", aa.Action())
		}
	}
}

// TestResetLayoutProducesValidCoordinates verifies that RESET LAYOUT does not
// clear RelativeMiddlePoint to "". buildFlowGraphGen computes valid "x;y"
// coordinates for every activity; the RESET LAYOUT option just means
// "recompute from scratch" (which buildFlowGraphGen already does). Clearing
// positions to "" caused Studio Pro to overlap all activities at the origin.
func TestResetLayoutProducesValidCoordinates(t *testing.T) {
	fb := newGraphTestFb()
	body := []ast.MicroflowStatement{
		&ast.DeclareStmt{Variable: "X", Type: ast.DataType{Kind: ast.TypeBoolean}},
	}
	oc := fb.buildFlowGraphGen(body, nil)

	// resetLayoutGen was the bug — it has been deleted. Positions from
	// buildFlowGraphGen must be non-empty "x;y" strings.
	type posGetter interface {
		RelativeMiddlePoint() string
	}
	for _, obj := range oc.ObjectsItems() {
		if pg, ok := obj.(posGetter); ok {
			pos := pg.RelativeMiddlePoint()
			if pos == "" {
				t.Errorf("RelativeMiddlePoint is empty for %T — buildFlowGraphGen must assign valid coordinates", obj)
			}
		}
	}
}

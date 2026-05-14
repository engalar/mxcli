// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.i — error-handler body emission tests (TDD).
//
// addErrorHandlerFlowGen builds the error-handler body activities,
// connects them with a labelled error edge from the source activity,
// and returns the tail (last activity ID) for the caller to merge
// back to the main flow.
//
// Recursion: each statement in the body goes through addStatementGen
// (h1), so all action types are reachable from inside an EH body.
//
// Out of scope: buildRetryLoopErrorHandlerGen (h-extension when
// retry pattern is detected) is left as a TODO — it requires a merge
// inserted before the source, and the legacy implementation has its
// own dedicated path.

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func TestAddErrorHandlerFlowGenEmptyBodyReturnsEmpty(t *testing.T) {
	fb := newIfTestFb()
	tail := fb.addErrorHandlerFlowGen("src-id", 100, nil)
	if tail.id != "" {
		t.Fatalf("empty body should return empty tail, got id=%s", tail.id)
	}
	if len(fb.objects) != 0 {
		t.Fatalf("empty body should not emit any objects, got %d", len(fb.objects))
	}
}

func TestAddErrorHandlerFlowGenSingleActivityEmitsErrorEdgeAndTail(t *testing.T) {
	fb := newIfTestFb()
	body := []ast.MicroflowStatement{
		&ast.DeclareStmt{
			Variable: "Errored",
			Type:     ast.DataType{Kind: ast.TypeBoolean},
			InitialValue: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
		},
	}
	tail := fb.addErrorHandlerFlowGen("src-id", 100, body)
	if tail.id == "" {
		t.Fatal("non-empty body should return non-empty tail ID")
	}

	// Should have 1 activity emitted.
	activityCount := 0
	for _, obj := range fb.objects {
		if _, ok := obj.(*genMf.ActionActivity); ok {
			activityCount++
		}
	}
	if activityCount != 1 {
		t.Fatalf("activity count = %d, want 1", activityCount)
	}

	// Should have 1 error-handler flow connecting source → first activity.
	errorFlowCount := 0
	for _, f := range fb.flows {
		if f.IsErrorHandler() && f.OriginRefID() == "src-id" {
			errorFlowCount++
		}
	}
	if errorFlowCount != 1 {
		t.Fatalf("error handler flow count = %d, want 1", errorFlowCount)
	}
}

func TestAddErrorHandlerFlowGenTerminatingBodyReturnsEmptyTail(t *testing.T) {
	// When the error body ends with RaiseErrorStmt (or Return), the
	// handler terminates there — caller should not create a merge.
	fb := newIfTestFb()
	body := []ast.MicroflowStatement{&ast.RaiseErrorStmt{}}
	tail := fb.addErrorHandlerFlowGen("src-id", 100, body)
	if tail.id != "" {
		t.Fatalf("terminating body should return empty tail, got %s", tail.id)
	}
}

func TestAddErrorHandlerFlowGenMultipleActivitiesChained(t *testing.T) {
	fb := newIfTestFb()
	body := []ast.MicroflowStatement{
		&ast.DeclareStmt{Variable: "A", Type: ast.DataType{Kind: ast.TypeBoolean}},
		&ast.DeclareStmt{Variable: "B", Type: ast.DataType{Kind: ast.TypeBoolean}},
	}
	tail := fb.addErrorHandlerFlowGen("src-id", 100, body)
	if tail.id == "" {
		t.Fatal("expected non-empty tail")
	}

	activityCount := 0
	for _, obj := range fb.objects {
		if _, ok := obj.(*genMf.ActionActivity); ok {
			activityCount++
		}
	}
	if activityCount != 2 {
		t.Fatalf("activity count = %d, want 2", activityCount)
	}

	// Should have 1 error edge (src → first) + 1 horizontal chain (first → second).
	errorFlowCount, normalFlowCount := 0, 0
	for _, f := range fb.flows {
		if f.IsErrorHandler() {
			errorFlowCount++
		} else {
			normalFlowCount++
		}
	}
	if errorFlowCount != 1 {
		t.Fatalf("error handler flows = %d, want 1 (only the src→first edge)", errorFlowCount)
	}
	if normalFlowCount != 1 {
		t.Fatalf("normal flows = %d, want 1 (chain link)", normalFlowCount)
	}
}

func TestFinishCustomErrorHandlerGenNilClauseIsNoOp(t *testing.T) {
	fb := newIfTestFb()
	// Empty/nil clause shouldn't crash or change state.
	fb.finishCustomErrorHandlerGen("act-id", 100, nil, "")
	if len(fb.objects) != 0 {
		t.Fatalf("nil clause should be no-op, got %d objects", len(fb.objects))
	}
}

func TestFinishCustomErrorHandlerGenEmptyClauseRegistersHandler(t *testing.T) {
	// Empty `on error custom` clause routes to the existing
	// registerEmptyCustomErrorHandlerWithSkipGen path (commit d).
	fb := newIfTestFb()
	fb.finishCustomErrorHandlerGen("act-id", 100, &ast.ErrorHandlingClause{Type: ast.ErrorHandlingCustom}, "skipVar")
	if fb.errorHandlerSource != "act-id" {
		t.Fatalf("errorHandlerSource = %s, want act-id", fb.errorHandlerSource)
	}
	if fb.errorHandlerSkipVar != "skipVar" {
		t.Fatalf("errorHandlerSkipVar = %s, want skipVar", fb.errorHandlerSkipVar)
	}
}

func TestFinishCustomErrorHandlerGenWithBodyEmitsBodyAndQueuesMerge(t *testing.T) {
	fb := newIfTestFb()
	clause := &ast.ErrorHandlingClause{
		Type: ast.ErrorHandlingCustom,
		Body: []ast.MicroflowStatement{
			&ast.DeclareStmt{Variable: "Err", Type: ast.DataType{Kind: ast.TypeBoolean}},
		},
	}
	fb.finishCustomErrorHandlerGen("act-id", 100, clause, "")

	// Should emit the body activity + an error-handler flow from
	// "act-id" into it.
	activityCount := 0
	for _, obj := range fb.objects {
		if _, ok := obj.(*genMf.ActionActivity); ok {
			activityCount++
		}
	}
	if activityCount != 1 {
		t.Fatalf("activity count = %d, want 1 (the EH body activity)", activityCount)
	}

	// Should have queued the merge for the next normal-flow rejoin.
	if fb.errorHandlerSource != "act-id" {
		t.Fatalf("errorHandlerSource = %s, want act-id", fb.errorHandlerSource)
	}
	if fb.errorHandlerTailFrom == "" {
		t.Fatal("errorHandlerTailFrom should be set to the tail of the EH body")
	}
}

// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.h1 — addStatementGen dispatcher tests (TDD).
//
// Dispatcher is the type-switch entry point that the control-flow
// adders (h2/h3/h4) recursively call to process branch / loop /
// case bodies. It also handles three small leaf events that don't
// have their own sub-commit:
//
//   - addBreakEventGen      — `break;` (terminates loop)
//   - addContinueEventGen   — `continue;` (loops back)
//   - return / raise error  — already covered by the Stage 3.2.3.b
//                             EndEvent / ErrorEvent adders
//
// Sub-stage routing: the dispatcher maps each AST statement type to
// its add*Gen counterpart. All 41 action adders shipped in a-g are
// wired here. The 5 control-flow types (IfStmt / LoopStmt / WhileStmt
// / EnumSplitStmt / InheritanceSplitStmt) return an empty ID until
// h2/h3/h4 land — we keep the dispatcher case present so the type
// is recognised; the empty return mirrors the legacy "unknown
// statement" fallback (no panic) so partial wiring stays buildable.

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func TestAddStatementGenRoutesDeclareToCreateVariable(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.DeclareStmt{
		Variable: "X", Type: ast.DataType{Kind: ast.TypeBoolean},
		InitialValue: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
	}
	id := fb.addStatementGen(stmt)
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	if _, ok := fb.objects[0].(*genMf.ActionActivity); !ok {
		t.Fatalf("want *ActionActivity wrapping CreateVariableAction, got %T", fb.objects[0])
	}
	act := fb.objects[0].(*genMf.ActionActivity).Action()
	if _, ok := act.(*genMf.CreateVariableAction); !ok {
		t.Fatalf("inner action = %T, want *CreateVariableAction", act)
	}
}

func TestAddStatementGenRoutesReturnStmtToEndEvent(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.ReturnStmt{Value: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true}}
	id := fb.addStatementGen(stmt)
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	if _, ok := fb.objects[0].(*genMf.EndEvent); !ok {
		t.Fatalf("want *EndEvent, got %T", fb.objects[0])
	}
	if !fb.endsWithReturn {
		t.Fatal("endsWithReturn flag should be set after ReturnStmt")
	}
}

func TestAddStatementGenRoutesRaiseErrorStmtToErrorEvent(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.RaiseErrorStmt{}
	id := fb.addStatementGen(stmt)
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	if _, ok := fb.objects[0].(*genMf.ErrorEvent); !ok {
		t.Fatalf("want *ErrorEvent, got %T", fb.objects[0])
	}
}

func TestAddStatementGenRoutesBreakStmtToBreakEvent(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.BreakStmt{}
	id := fb.addStatementGen(stmt)
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	if _, ok := fb.objects[0].(*genMf.BreakEvent); !ok {
		t.Fatalf("want *BreakEvent, got %T", fb.objects[0])
	}
}

func TestAddStatementGenRoutesContinueStmtToContinueEvent(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.ContinueStmt{}
	id := fb.addStatementGen(stmt)
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	if _, ok := fb.objects[0].(*genMf.ContinueEvent); !ok {
		t.Fatalf("want *ContinueEvent, got %T", fb.objects[0])
	}
}

func TestAddStatementGenContinueStmtRespectsManualLoopBackTarget(t *testing.T) {
	// When manualLoopBackTarget is set, addContinueEventGen should
	// return the existing target ID without emitting a new event
	// (matches legacy addContinueEvent).
	fb := newActionTestFb()
	fb.manualLoopBackTarget = "back-to-here"
	id := fb.addStatementGen(&ast.ContinueStmt{})
	if id != "back-to-here" {
		t.Fatalf("got %s, want manualLoopBackTarget=back-to-here", id)
	}
	if len(fb.objects) != 0 {
		t.Fatalf("no event should be emitted when manualLoopBackTarget set, got %d objects", len(fb.objects))
	}
}

func TestAddStatementGenRoutesMfSetWithPathToChangeObject(t *testing.T) {
	// SET with a path target ($Var/Module.Assoc) routes to ChangeObject
	// instead of ChangeVariable (legacy parity).
	fb := newActionTestFb()
	fb.varTypes["Order"] = "Sales.Order"
	stmt := &ast.MfSetStmt{
		Target: "$Order/Sales.Order_Customer",
		Value:  &ast.VariableExpr{Name: "Cust"},
	}
	id := fb.addStatementGen(stmt)
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	act := fb.objects[0].(*genMf.ActionActivity).Action()
	if _, ok := act.(*genMf.ChangeObjectAction); !ok {
		t.Fatalf("path-target SET should route to ChangeObjectAction, got %T", act)
	}
}

func TestAddStatementGenRoutesMfSetBareToChangeVariable(t *testing.T) {
	fb := newActionTestFb()
	fb.declaredVars["X"] = "Boolean"
	stmt := &ast.MfSetStmt{
		Target: "X",
		Value:  &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
	}
	fb.addStatementGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity).Action()
	if _, ok := act.(*genMf.ChangeVariableAction); !ok {
		t.Fatalf("bare SET should route to ChangeVariableAction, got %T", act)
	}
}

func TestAddStatementGenRoutesShowMessageToShowMessage(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.ShowMessageStmt{
		Message: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "Hi"},
	}
	fb.addStatementGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity).Action()
	if _, ok := act.(*genMf.ShowMessageAction); !ok {
		t.Fatalf("ShowMessage should route to *ShowMessageAction, got %T", act)
	}
}

func TestAddStatementGenRoutesCallMicroflowToMicroflowCallAction(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CallMicroflowStmt{
		MicroflowName: ast.QualifiedName{Module: "M", Name: "WF"},
	}
	fb.addStatementGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity).Action()
	if _, ok := act.(*genMf.MicroflowCallAction); !ok {
		t.Fatalf("CallMicroflow should route to *MicroflowCallAction, got %T", act)
	}
}

func TestAddStatementGenAppliesPendingPositionAnnotation(t *testing.T) {
	// @position annotation must apply BEFORE the activity is created
	// so the activity lands at the requested coordinates.
	fb := newActionTestFb()
	stmt := &ast.DeclareStmt{
		Variable:    "X",
		Type:        ast.DataType{Kind: ast.TypeBoolean},
		Annotations: &ast.ActivityAnnotations{Position: &ast.Position{X: 999, Y: 888}},
	}
	fb.addStatementGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity)
	if act.RelativeMiddlePoint() != "999 888" {
		t.Fatalf("position = %q, want 999 888", act.RelativeMiddlePoint())
	}
}

func TestAddStatementGenAttachesPendingFreeAnnotations(t *testing.T) {
	// @annotation entries on the statement should produce free
	// Annotation objects (no flow attachment).
	fb := newActionTestFb()
	stmt := &ast.DeclareStmt{
		Variable: "X",
		Type:     ast.DataType{Kind: ast.TypeBoolean},
		Annotations: &ast.ActivityAnnotations{
			FreeAnnotations: []string{"free comment"},
		},
	}
	fb.addStatementGen(stmt)
	// First object is the free annotation, second is the activity.
	if len(fb.objects) != 2 {
		t.Fatalf("want 2 objects (free annotation + activity), got %d", len(fb.objects))
	}
	if _, ok := fb.objects[0].(*genMf.Annotation); !ok {
		t.Fatalf("first object should be *Annotation, got %T", fb.objects[0])
	}
}

// Note: ast.MicroflowStatement uses an unexported marker method
// (isMicroflowStatement), so we can't construct an "unknown" statement
// type from a test-external package. The empty-fallback path is
// covered by the IfStmt/LoopStmt/etc placeholder cases (h2-h4 not
// yet landed) — those return "" until their adders ship.

// TestAddStatementGenIfStmtPlaceholderReturnsEmpty was removed when
// h2 wired IfStmt → addIfStatementGen. See
// flowbuilder_control_if_gen_test.go for IfStmt routing coverage.

// TestAddStatementGenLoopStmtPlaceholderReturnsEmpty was removed
// when h3 wired LoopStmt → addLoopStatementGen. See
// flowbuilder_control_loop_gen_test.go for routing coverage.

// TestAddStatementGenEnumSplitStmtPlaceholderReturnsEmpty was removed
// when h4 wired EnumSplitStmt → addEnumSplitGen. See
// flowbuilder_control_split_gen_test.go for routing coverage.

func TestAddStatementGenRoutesCallWorkflowToWorkflowCallAction(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CallWorkflowStmt{
		Workflow: ast.QualifiedName{Module: "M", Name: "WF"},
	}
	fb.addStatementGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity).Action()
	if _, ok := act.(*genMf.WorkflowCallAction); !ok {
		t.Fatalf("CallWorkflow should route to *WorkflowCallAction, got %T", act)
	}
}

func TestAddBreakEventGenAdvancesPosByHalfSpacing(t *testing.T) {
	fb := newActionTestFb()
	startX := fb.posX
	id := fb.addBreakEventGen()
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	if fb.posX != startX+HorizontalSpacing/2 {
		t.Fatalf("posX = %d, want %d (advance by spacing/2 like other event emitters)",
			fb.posX, startX+HorizontalSpacing/2)
	}
}

func TestAddContinueEventGenAdvancesPosByHalfSpacing(t *testing.T) {
	fb := newActionTestFb()
	startX := fb.posX
	id := fb.addContinueEventGen()
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	if fb.posX != startX+HorizontalSpacing/2 {
		t.Fatalf("posX = %d, want %d", fb.posX, startX+HorizontalSpacing/2)
	}
}

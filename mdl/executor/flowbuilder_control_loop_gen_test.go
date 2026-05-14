// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.h3 — addLoopStatementGen + addWhileStatementGen tests (TDD).
//
// Both wrap a `*genMf.LoopedActivity` with different LoopSource:
//
//   Loop:  IterableList{ListVariableName, VariableName}
//   While: WhileLoopCondition{WhileExpression}
//
// Body statements live in a nested ObjectCollection. Internal flows
// go to the parent's flows list (top-level), not inside the loop's
// ObjectCollection — matches Mendix BSON layout.

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func newLoopTestFb() *flowBuilderGen {
	fb := newActionTestFb()
	fb.measurer = &layoutMeasurer{varTypes: fb.varTypes}
	return fb
}

func TestAddLoopStatementGenSetsIterableList(t *testing.T) {
	fb := newLoopTestFb()
	fb.varTypes["Items"] = "List of Sales.Order"
	stmt := &ast.LoopStmt{
		ListVariable: "Items",
		LoopVariable: "Item",
	}
	id := fb.addLoopStatementGen(stmt)
	if id == "" {
		t.Fatal("expected non-empty loop ID")
	}

	var loop *genMf.LoopedActivity
	for _, obj := range fb.objects {
		if l, ok := obj.(*genMf.LoopedActivity); ok {
			loop = l
			break
		}
	}
	if loop == nil {
		t.Fatal("LoopedActivity must be emitted")
	}
	src, ok := loop.LoopSource().(*genMf.IterableList)
	if !ok {
		t.Fatalf("LoopSource = %T, want *IterableList", loop.LoopSource())
	}
	if src.ListVariableName() != "Items" {
		t.Fatalf("list var = %q, want Items", src.ListVariableName())
	}
	if src.VariableName() != "Item" {
		t.Fatalf("loop var = %q, want Item", src.VariableName())
	}
	// Loop var should be registered with the element type.
	if fb.varTypes["Item"] != "Sales.Order" {
		t.Fatalf("loop var type = %q, want Sales.Order", fb.varTypes["Item"])
	}
}

func TestAddLoopStatementGenEmptyBodyEmitsObjectCollection(t *testing.T) {
	fb := newLoopTestFb()
	fb.varTypes["L"] = "List of M.E"
	stmt := &ast.LoopStmt{ListVariable: "L", LoopVariable: "I"}
	fb.addLoopStatementGen(stmt)
	loop := fb.objects[len(fb.objects)-1].(*genMf.LoopedActivity)
	oc, ok := loop.ObjectCollection().(*genMf.MicroflowObjectCollection)
	if !ok {
		t.Fatalf("ObjectCollection = %T, want *MicroflowObjectCollection", loop.ObjectCollection())
	}
	if len(oc.ObjectsItems()) != 0 {
		t.Fatalf("empty body → ObjectCollection should have 0 objects, got %d", len(oc.ObjectsItems()))
	}
}

func TestAddLoopStatementGenWithBodyEmitsActivities(t *testing.T) {
	fb := newLoopTestFb()
	fb.varTypes["L"] = "List of M.E"
	stmt := &ast.LoopStmt{
		ListVariable: "L",
		LoopVariable: "I",
		Body: []ast.MicroflowStatement{
			&ast.DeclareStmt{Variable: "X", Type: ast.DataType{Kind: ast.TypeBoolean}},
		},
	}
	fb.addLoopStatementGen(stmt)
	loop := fb.objects[len(fb.objects)-1].(*genMf.LoopedActivity)
	oc := loop.ObjectCollection().(*genMf.MicroflowObjectCollection)
	if len(oc.ObjectsItems()) != 1 {
		t.Fatalf("body activities = %d, want 1", len(oc.ObjectsItems()))
	}
	if _, ok := oc.ObjectsItems()[0].(*genMf.ActionActivity); !ok {
		t.Fatalf("body[0] = %T, want *ActionActivity", oc.ObjectsItems()[0])
	}
}

func TestAddLoopStatementGenDuplicateLoopVarReportsError(t *testing.T) {
	fb := newLoopTestFb()
	fb.varTypes["I"] = "Sales.Other"
	stmt := &ast.LoopStmt{ListVariable: "L", LoopVariable: "I"}
	id := fb.addLoopStatementGen(stmt)
	if id != "" {
		t.Fatalf("duplicate loop var should return empty ID, got %s", id)
	}
	if len(fb.GetErrors()) == 0 {
		t.Fatal("duplicate loop var should produce a CE0111 validation error")
	}
}

func TestAddWhileStatementGenSetsWhileLoopCondition(t *testing.T) {
	fb := newLoopTestFb()
	stmt := &ast.WhileStmt{
		Condition: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
	}
	id := fb.addWhileStatementGen(stmt)
	if id == "" {
		t.Fatal("expected non-empty while ID")
	}
	var loop *genMf.LoopedActivity
	for _, obj := range fb.objects {
		if l, ok := obj.(*genMf.LoopedActivity); ok {
			loop = l
			break
		}
	}
	if loop == nil {
		t.Fatal("LoopedActivity must be emitted")
	}
	src, ok := loop.LoopSource().(*genMf.WhileLoopCondition)
	if !ok {
		t.Fatalf("LoopSource = %T, want *WhileLoopCondition", loop.LoopSource())
	}
	if src.WhileExpression() != "true" {
		t.Fatalf("while expression = %q, want true", src.WhileExpression())
	}
}

func TestAddLoopStatementGenRoutesViaDispatcher(t *testing.T) {
	fb := newLoopTestFb()
	fb.varTypes["L"] = "List of M.E"
	id := fb.addStatementGen(&ast.LoopStmt{
		ListVariable: "L",
		LoopVariable: "I",
	})
	if id == "" {
		t.Fatal("dispatcher should now route LoopStmt to addLoopStatementGen")
	}
}

func TestAddWhileStatementGenRoutesViaDispatcher(t *testing.T) {
	fb := newLoopTestFb()
	id := fb.addStatementGen(&ast.WhileStmt{
		Condition: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
	})
	if id == "" {
		t.Fatal("dispatcher should now route WhileStmt to addWhileStatementGen")
	}
}

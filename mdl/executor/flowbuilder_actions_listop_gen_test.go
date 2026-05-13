// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.f4a — list-operation adder tests (TDD).
//
// Each test below describes the behaviour the corresponding
// addListOperationActionGen branch must exhibit. Tests are written
// FIRST (RED), then the production file is added; tests must fail
// for the right reason (function undefined / not implemented) before
// the production code lands.
//
// 13 list-operation primitives are covered:
//
//   ast.ListOpHead       → *genMf.Head
//   ast.ListOpTail       → *genMf.Tail
//   ast.ListOpFind       → *genMf.Find (binary= attr) | *genMf.FindByExpression
//   ast.ListOpFilter     → *genMf.Filter (binary= attr) | *genMf.FilterByExpression
//   ast.ListOpSort       → *genMf.Sort with SortItemList of SortItem
//   ast.ListOpUnion      → *genMf.Union
//   ast.ListOpIntersect  → *genMf.Intersect
//   ast.ListOpSubtract   → *genMf.Subtract
//   ast.ListOpContains   → *genMf.Contains
//   ast.ListOpEquals     → *genMf.ListEquals
//   ast.ListOpRange      → *genMf.ListRange wrapping *genMf.CustomRange
//
// + output-variable type tracking: list-preserving ops carry the input
// type forward; Head/Find unwrap "List of E" to "E"; Contains/Equals
// don't track (Boolean not in varTypes scheme).

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func TestAddListOperationActionGenHead(t *testing.T) {
	fb := newActionTestFb()
	fb.varTypes["L"] = "List of M.E"
	stmt := &ast.ListOperationStmt{
		Operation:      ast.ListOpHead,
		InputVariable:  "L",
		OutputVariable: "First",
	}
	fb.addListOperationActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ListOperationAction)
	op, ok := act.Operation().(*genMf.Head)
	if !ok {
		t.Fatalf("operation = %T, want *Head", act.Operation())
	}
	if op.ListVariableName() != "L" {
		t.Fatalf("list var = %q", op.ListVariableName())
	}
	if act.OutputVariableName() != "First" {
		t.Fatalf("output var = %q", act.OutputVariableName())
	}
	// Head returns single element — output type strips "List of ".
	if fb.varTypes["First"] != "M.E" {
		t.Fatalf("output type = %q, want M.E", fb.varTypes["First"])
	}
}

func TestAddListOperationActionGenTail(t *testing.T) {
	fb := newActionTestFb()
	fb.varTypes["L"] = "List of M.E"
	stmt := &ast.ListOperationStmt{
		Operation:      ast.ListOpTail,
		InputVariable:  "L",
		OutputVariable: "Rest",
	}
	fb.addListOperationActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ListOperationAction)
	op, ok := act.Operation().(*genMf.Tail)
	if !ok {
		t.Fatalf("operation = %T, want *Tail", act.Operation())
	}
	if op.ListVariableName() != "L" {
		t.Fatalf("list var = %q", op.ListVariableName())
	}
	// Tail preserves list type.
	if fb.varTypes["Rest"] != "List of M.E" {
		t.Fatalf("output type = %q, want List of M.E", fb.varTypes["Rest"])
	}
}

func TestAddListOperationActionGenFindByExpression(t *testing.T) {
	// A non-binary condition (or non-attribute) → FindByExpression.
	fb := newActionTestFb()
	fb.varTypes["L"] = "List of M.E"
	stmt := &ast.ListOperationStmt{
		Operation:      ast.ListOpFind,
		InputVariable:  "L",
		OutputVariable: "Found",
		Condition:      &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
	}
	fb.addListOperationActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ListOperationAction)
	op, ok := act.Operation().(*genMf.FindByExpression)
	if !ok {
		t.Fatalf("operation = %T, want *FindByExpression", act.Operation())
	}
	if op.ListVariableName() != "L" {
		t.Fatalf("list var = %q", op.ListVariableName())
	}
	if op.Expression() != "true" {
		t.Fatalf("expression = %q, want true", op.Expression())
	}
	// Find returns single element.
	if fb.varTypes["Found"] != "M.E" {
		t.Fatalf("output type = %q, want M.E", fb.varTypes["Found"])
	}
}

func TestAddListOperationActionGenFilterByExpression(t *testing.T) {
	fb := newActionTestFb()
	fb.varTypes["L"] = "List of M.E"
	stmt := &ast.ListOperationStmt{
		Operation:      ast.ListOpFilter,
		InputVariable:  "L",
		OutputVariable: "Filtered",
		Condition:      &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
	}
	fb.addListOperationActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ListOperationAction)
	op, ok := act.Operation().(*genMf.FilterByExpression)
	if !ok {
		t.Fatalf("operation = %T, want *FilterByExpression", act.Operation())
	}
	if op.Expression() != "true" {
		t.Fatalf("expression = %q", op.Expression())
	}
	// Filter preserves list type.
	if fb.varTypes["Filtered"] != "List of M.E" {
		t.Fatalf("output type = %q, want List of M.E", fb.varTypes["Filtered"])
	}
}

func TestAddListOperationActionGenSortBuildsItemList(t *testing.T) {
	fb := newActionTestFb()
	fb.varTypes["L"] = "List of Sales.Order"
	stmt := &ast.ListOperationStmt{
		Operation:      ast.ListOpSort,
		InputVariable:  "L",
		OutputVariable: "Sorted",
		SortSpecs: []ast.SortSpec{
			{Attribute: "Amount", Ascending: true},
			{Attribute: "Status", Ascending: false},
			{Attribute: "Mod.Other.Field", Ascending: true}, // pre-qualified
		},
	}
	fb.addListOperationActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ListOperationAction)
	op, ok := act.Operation().(*genMf.Sort)
	if !ok {
		t.Fatalf("operation = %T, want *Sort", act.Operation())
	}
	if op.ListVariableName() != "L" {
		t.Fatalf("list var = %q", op.ListVariableName())
	}
	itemList, ok := op.SortItemList().(*genMf.SortItemList)
	if !ok {
		t.Fatalf("SortItemList = %T, want *SortItemList", op.SortItemList())
	}
	items := itemList.ItemsItems()
	if len(items) != 3 {
		t.Fatalf("items = %d, want 3", len(items))
	}
	// Item 1: bare attribute → qualified with input entity type.
	si1 := items[0].(*genMf.SortItem)
	if si1.AttributePath() != "Sales.Order.Amount" {
		t.Fatalf("item 1 path = %q, want Sales.Order.Amount", si1.AttributePath())
	}
	if si1.SortOrder() != "Ascending" {
		t.Fatalf("item 1 order = %q, want Ascending", si1.SortOrder())
	}
	// Item 2: descending.
	si2 := items[1].(*genMf.SortItem)
	if si2.SortOrder() != "Descending" {
		t.Fatalf("item 2 order = %q, want Descending", si2.SortOrder())
	}
	// Item 3: pre-qualified — left intact.
	si3 := items[2].(*genMf.SortItem)
	if si3.AttributePath() != "Mod.Other.Field" {
		t.Fatalf("item 3 path = %q, want Mod.Other.Field (pre-qualified)", si3.AttributePath())
	}
	// Sort preserves list type.
	if fb.varTypes["Sorted"] != "List of Sales.Order" {
		t.Fatalf("output type = %q", fb.varTypes["Sorted"])
	}
}

func TestAddListOperationActionGenUnion(t *testing.T) {
	fb := newActionTestFb()
	fb.varTypes["A"] = "List of M.E"
	stmt := &ast.ListOperationStmt{
		Operation:      ast.ListOpUnion,
		InputVariable:  "A",
		SecondVariable: "B",
		OutputVariable: "Out",
	}
	fb.addListOperationActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ListOperationAction)
	op, ok := act.Operation().(*genMf.Union)
	if !ok {
		t.Fatalf("operation = %T, want *Union", act.Operation())
	}
	if op.ListVariableName() != "A" || op.SecondListOrObjectVariableName() != "B" {
		t.Fatalf("vars = %q / %q, want A / B", op.ListVariableName(), op.SecondListOrObjectVariableName())
	}
	if fb.varTypes["Out"] != "List of M.E" {
		t.Fatalf("output type = %q", fb.varTypes["Out"])
	}
}

func TestAddListOperationActionGenIntersectAndSubtract(t *testing.T) {
	cases := []struct {
		op   ast.ListOperationType
		want string
	}{
		{ast.ListOpIntersect, "*microflows.Intersect"},
		{ast.ListOpSubtract, "*microflows.Subtract"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			fb := newActionTestFb()
			fb.varTypes["A"] = "List of M.E"
			stmt := &ast.ListOperationStmt{
				Operation:      tc.op,
				InputVariable:  "A",
				SecondVariable: "B",
				OutputVariable: "Out",
			}
			fb.addListOperationActionGen(stmt)
			act := actionFromObjects(t, fb).(*genMf.ListOperationAction)
			switch tc.op {
			case ast.ListOpIntersect:
				if _, ok := act.Operation().(*genMf.Intersect); !ok {
					t.Fatalf("op = %T, want *Intersect", act.Operation())
				}
			case ast.ListOpSubtract:
				if _, ok := act.Operation().(*genMf.Subtract); !ok {
					t.Fatalf("op = %T, want *Subtract", act.Operation())
				}
			}
		})
	}
}

func TestAddListOperationActionGenContains(t *testing.T) {
	fb := newActionTestFb()
	fb.varTypes["L"] = "List of M.E"
	stmt := &ast.ListOperationStmt{
		Operation:      ast.ListOpContains,
		InputVariable:  "L",
		SecondVariable: "Item",
		OutputVariable: "Has",
	}
	fb.addListOperationActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ListOperationAction)
	op, ok := act.Operation().(*genMf.Contains)
	if !ok {
		t.Fatalf("op = %T, want *Contains", act.Operation())
	}
	if op.ListVariableName() != "L" || op.SecondListOrObjectVariableName() != "Item" {
		t.Fatalf("vars = %q / %q", op.ListVariableName(), op.SecondListOrObjectVariableName())
	}
	// Contains returns Boolean — output not tracked in varTypes.
	if _, tracked := fb.varTypes["Has"]; tracked {
		t.Fatalf("Contains output should not be tracked in varTypes (Boolean)")
	}
}

func TestAddListOperationActionGenEquals(t *testing.T) {
	fb := newActionTestFb()
	fb.varTypes["A"] = "List of M.E"
	stmt := &ast.ListOperationStmt{
		Operation:      ast.ListOpEquals,
		InputVariable:  "A",
		SecondVariable: "B",
		OutputVariable: "Same",
	}
	fb.addListOperationActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ListOperationAction)
	if _, ok := act.Operation().(*genMf.ListEquals); !ok {
		t.Fatalf("op = %T, want *ListEquals", act.Operation())
	}
}

func TestAddListOperationActionGenRangeWithBothExprs(t *testing.T) {
	fb := newActionTestFb()
	fb.varTypes["L"] = "List of M.E"
	stmt := &ast.ListOperationStmt{
		Operation:      ast.ListOpRange,
		InputVariable:  "L",
		OutputVariable: "Slice",
		OffsetExpr:     &ast.LiteralExpr{Kind: ast.LiteralInteger, Value: int64(5)},
		LimitExpr:      &ast.LiteralExpr{Kind: ast.LiteralInteger, Value: int64(10)},
	}
	fb.addListOperationActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ListOperationAction)
	op, ok := act.Operation().(*genMf.ListRange)
	if !ok {
		t.Fatalf("op = %T, want *ListRange", act.Operation())
	}
	if op.ListVariableName() != "L" {
		t.Fatalf("list var = %q", op.ListVariableName())
	}
	cr, ok := op.CustomRange().(*genMf.CustomRange)
	if !ok {
		t.Fatalf("CustomRange = %T, want *CustomRange", op.CustomRange())
	}
	if cr.OffsetExpression() != "5" {
		t.Fatalf("offset = %q", cr.OffsetExpression())
	}
	if cr.LimitExpression() != "10" {
		t.Fatalf("limit = %q", cr.LimitExpression())
	}
	// Range preserves list type.
	if fb.varTypes["Slice"] != "List of M.E" {
		t.Fatalf("output type = %q", fb.varTypes["Slice"])
	}
}

func TestAddListOperationActionGenRangeNoExprsLeavesCustomRangeBareButPresent(t *testing.T) {
	// Range without exprs still wraps a CustomRange element so the
	// gen describer can still pattern-match it as a Range op.
	fb := newActionTestFb()
	fb.varTypes["L"] = "List of M.E"
	stmt := &ast.ListOperationStmt{
		Operation:      ast.ListOpRange,
		InputVariable:  "L",
		OutputVariable: "Slice",
	}
	fb.addListOperationActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ListOperationAction)
	op := act.Operation().(*genMf.ListRange)
	if op.CustomRange() == nil {
		t.Fatal("CustomRange must be wrapped even when no expressions supplied")
	}
}

func TestAddListOperationActionGenUnknownOpReturnsEmpty(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.ListOperationStmt{Operation: ast.ListOperationType(99)}
	if id := fb.addListOperationActionGen(stmt); id != "" {
		t.Fatalf("unknown op should return empty ID, got %s", id)
	}
	if len(fb.objects) != 0 {
		t.Fatal("unknown op should not emit any object")
	}
}

// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.f4b — addRetrieveActionGen tests (TDD).
//
// Scope of the tests below:
//
//   - Database retrieve (no StartVariable):
//       - bare `retrieve $V from M.E` → DatabaseRetrieveSource w/ entity QN
//       - `where <expr>` → XPathConstraint wrapped in [...]
//       - `limit 1` no offset → ConstantRange{SingleObject: true}
//       - `limit N` or with offset → CustomRange
//       - sort by attr → SortItemList of SortItem
//       - bare attr in sort qualifies with entity QN; pre-qualified passes
//   - Association retrieve ($Start/Module.Assoc):
//       - emits AssociationRetrieveSource w/ start var + assoc QN
//   - Output type registration:
//       - LIMIT 1 → single entity
//       - else → "List of E"
//       - Association retrieve → "List of E" (offline default; reverse
//         detection requires backend, deferred to integration tests)

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func TestAddRetrieveActionGenDatabaseBare(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.RetrieveStmt{
		Variable: "Orders",
		Source:   ast.QualifiedName{Module: "Sales", Name: "Order"},
	}
	fb.addRetrieveActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RetrieveAction)
	if act.OutputVariableName() != "Orders" {
		t.Fatalf("output var = %q", act.OutputVariableName())
	}
	src, ok := act.RetrieveSource().(*genMf.DatabaseRetrieveSource)
	if !ok {
		t.Fatalf("source = %T, want *DatabaseRetrieveSource", act.RetrieveSource())
	}
	if src.EntityQualifiedName() != "Sales.Order" {
		t.Fatalf("entity = %q", src.EntityQualifiedName())
	}
	if src.XPathConstraint() != "" {
		t.Fatalf("xpath should be empty, got %q", src.XPathConstraint())
	}
	if src.Range() != nil {
		t.Fatalf("range should be nil for bare retrieve, got %T", src.Range())
	}
	// No LIMIT → registered as "List of E".
	if fb.varTypes["Orders"] != "List of Sales.Order" {
		t.Fatalf("output type = %q", fb.varTypes["Orders"])
	}
}

func TestAddRetrieveActionGenLimit1ProducesConstantRangeSingleObject(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.RetrieveStmt{
		Variable: "Order",
		Source:   ast.QualifiedName{Module: "Sales", Name: "Order"},
		Limit:    "1",
	}
	fb.addRetrieveActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RetrieveAction)
	src := act.RetrieveSource().(*genMf.DatabaseRetrieveSource)
	cr, ok := src.Range().(*genMf.ConstantRange)
	if !ok {
		t.Fatalf("range = %T, want *ConstantRange", src.Range())
	}
	if !cr.SingleObject() {
		t.Fatal("ConstantRange.SingleObject must be true for LIMIT 1")
	}
	// LIMIT 1 → single entity output type (not list).
	if fb.varTypes["Order"] != "Sales.Order" {
		t.Fatalf("output type = %q, want Sales.Order", fb.varTypes["Order"])
	}
}

func TestAddRetrieveActionGenLimitWithOffsetProducesCustomRange(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.RetrieveStmt{
		Variable: "Page",
		Source:   ast.QualifiedName{Module: "Sales", Name: "Order"},
		Limit:    "10",
		Offset:   "20",
	}
	fb.addRetrieveActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RetrieveAction)
	src := act.RetrieveSource().(*genMf.DatabaseRetrieveSource)
	cr, ok := src.Range().(*genMf.CustomRange)
	if !ok {
		t.Fatalf("range = %T, want *CustomRange", src.Range())
	}
	if cr.LimitExpression() != "10" {
		t.Fatalf("limit = %q", cr.LimitExpression())
	}
	if cr.OffsetExpression() != "20" {
		t.Fatalf("offset = %q", cr.OffsetExpression())
	}
	// LIMIT > 1 still registers as list.
	if fb.varTypes["Page"] != "List of Sales.Order" {
		t.Fatalf("output type = %q", fb.varTypes["Page"])
	}
}

func TestAddRetrieveActionGenWhereProducesXPathBracketed(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.RetrieveStmt{
		Variable: "Open",
		Source:   ast.QualifiedName{Module: "Sales", Name: "Order"},
		Where: &ast.BinaryExpr{
			Left:     &ast.AttributePathExpr{Variable: "Order", Path: []string{"Status"}},
			Operator: "=",
			Right:    &ast.LiteralExpr{Kind: ast.LiteralString, Value: "Open"},
		},
	}
	fb.addRetrieveActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RetrieveAction)
	src := act.RetrieveSource().(*genMf.DatabaseRetrieveSource)
	if src.XPathConstraint() == "" {
		t.Fatal("XPathConstraint must be set when WHERE is present")
	}
	// Per legacy retrieveXPathConstraint, output is bracket-wrapped.
	if src.XPathConstraint()[0] != '[' || src.XPathConstraint()[len(src.XPathConstraint())-1] != ']' {
		t.Fatalf("xpath should be bracket-wrapped, got %q", src.XPathConstraint())
	}
}

func TestAddRetrieveActionGenSortBuildsItemList(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.RetrieveStmt{
		Variable: "Sorted",
		Source:   ast.QualifiedName{Module: "Sales", Name: "Order"},
		SortColumns: []ast.SortColumnDef{
			{Attribute: "Amount", Order: "asc"},
			{Attribute: "Status", Order: "DESC"},
		},
	}
	fb.addRetrieveActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RetrieveAction)
	src := act.RetrieveSource().(*genMf.DatabaseRetrieveSource)
	itemList, ok := src.SortItemList().(*genMf.SortItemList)
	if !ok {
		t.Fatalf("SortItemList = %T, want *SortItemList", src.SortItemList())
	}
	items := itemList.ItemsItems()
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	si1 := items[0].(*genMf.SortItem)
	if si1.AttributeRef() == nil {
		t.Fatalf("item 1 has nil AttributeRef")
	}
	ar1 := si1.AttributeRef().(*genDm.AttributeRef)
	if ar1.AttributeQualifiedName() != "Sales.Order.Amount" {
		t.Fatalf("item 1 attribute = %q, want %q", ar1.AttributeQualifiedName(), "Sales.Order.Amount")
	}
	if si1.SortOrder() != "Ascending" {
		t.Fatalf("item 1 order = %q", si1.SortOrder())
	}
	si2 := items[1].(*genMf.SortItem)
	if si2.SortOrder() != "Descending" {
		t.Fatalf("item 2 order = %q (case-insensitive `desc` should map to Descending)", si2.SortOrder())
	}
}

func TestAddRetrieveActionGenAssociationRetrieve(t *testing.T) {
	fb := newActionTestFb()
	fb.varTypes["Order"] = "Sales.Order"
	stmt := &ast.RetrieveStmt{
		Variable:      "Customer",
		Source:        ast.QualifiedName{Module: "Sales", Name: "Order_Customer"},
		StartVariable: "Order",
	}
	fb.addRetrieveActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RetrieveAction)
	src, ok := act.RetrieveSource().(*genMf.AssociationRetrieveSource)
	if !ok {
		t.Fatalf("source = %T, want *AssociationRetrieveSource", act.RetrieveSource())
	}
	if src.StartVariableName() != "Order" {
		t.Fatalf("start var = %q", src.StartVariableName())
	}
	if src.AssociationQualifiedName() != "Sales.Order_Customer" {
		t.Fatalf("assoc QN = %q", src.AssociationQualifiedName())
	}
	// Offline (no backend → no association lookup), default to list type.
	if fb.varTypes["Customer"] != "List of Sales.Order_Customer" {
		t.Fatalf("output type = %q", fb.varTypes["Customer"])
	}
}

func TestAddRetrieveActionGenWrapsActivityAndAdvancesPos(t *testing.T) {
	fb := newActionTestFb()
	startX := fb.posX
	stmt := &ast.RetrieveStmt{
		Variable: "Orders",
		Source:   ast.QualifiedName{Module: "M", Name: "E"},
	}
	id := fb.addRetrieveActionGen(stmt)
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	if fb.posX != startX+HorizontalSpacing {
		t.Fatalf("posX = %d, want %d", fb.posX, startX+HorizontalSpacing)
	}
}

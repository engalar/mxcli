// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.f2 — CreateObject / ChangeObject adder tests.
//
// Verifies the MemberChange wiring shape: each `Set` clause becomes
// a *genMf.MemberChange with the right Value and either
// AttributeQualifiedName or AssociationQualifiedName populated by
// resolveMemberChangeFallback (used in the absence of backend-driven
// resolution — f2's tests run offline so the fallback path is the
// path under test).

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func TestAddCreateObjectActionGenSetsEntityAndCommitDefault(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CreateObjectStmt{
		Variable:   "Order",
		EntityType: ast.QualifiedName{Module: "Sales", Name: "Order"},
	}
	fb.addCreateObjectActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.CreateObjectAction)
	if act.OutputVariableName() != "Order" {
		t.Fatalf("output var = %q", act.OutputVariableName())
	}
	if act.EntityQualifiedName() != "Sales.Order" {
		t.Fatalf("entity QN = %q", act.EntityQualifiedName())
	}
	if act.Commit() != "No" {
		t.Fatalf("commit default = %q, want No", act.Commit())
	}
	if fb.varTypes["Order"] != "Sales.Order" {
		t.Fatalf("varTypes[Order] = %q, want Sales.Order", fb.varTypes["Order"])
	}
}

func TestAddCreateObjectActionGenEmitsMemberChangeForEachAttr(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CreateObjectStmt{
		Variable:   "Order",
		EntityType: ast.QualifiedName{Module: "Sales", Name: "Order"},
		Changes: []ast.ChangeItem{
			{Attribute: "Status", Value: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "Open"}},
			{Attribute: "Amount", Value: &ast.LiteralExpr{Kind: ast.LiteralInteger, Value: int64(100)}},
		},
	}
	fb.addCreateObjectActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.CreateObjectAction)
	items := act.ItemsItems()
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	mc1, ok := items[0].(*genMf.MemberChange)
	if !ok {
		t.Fatalf("item 0 = %T, want *MemberChange", items[0])
	}
	if mc1.Type() != "Set" {
		t.Fatalf("type = %q, want Set", mc1.Type())
	}
	if mc1.AttributeQualifiedName() != "Sales.Order.Status" {
		t.Fatalf("attribute QN = %q, want Sales.Order.Status (offline fallback)", mc1.AttributeQualifiedName())
	}
	if mc1.Value() != "'Open'\n" {
		t.Fatalf("value = %q, want 'Open'\n", mc1.Value())
	}
	mc2 := items[1].(*genMf.MemberChange)
	if mc2.AttributeQualifiedName() != "Sales.Order.Amount" {
		t.Fatalf("attribute 2 QN = %q", mc2.AttributeQualifiedName())
	}
	if mc2.Value() != "100\n" {
		t.Fatalf("value 2 = %q, want 100\n", mc2.Value())
	}
}

func TestAddCreateObjectActionGenAssociationMemberFallback(t *testing.T) {
	// Offline (no backend) — a member name with one dot is treated as
	// `Module.Association` per resolveMemberChangeFallback.
	fb := newActionTestFb()
	stmt := &ast.CreateObjectStmt{
		Variable:   "Order",
		EntityType: ast.QualifiedName{Module: "Sales", Name: "Order"},
		Changes: []ast.ChangeItem{
			{Attribute: "Sales.Order_Customer", Value: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "X"}},
		},
	}
	fb.addCreateObjectActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.CreateObjectAction)
	mc := act.ItemsItems()[0].(*genMf.MemberChange)
	if mc.AssociationQualifiedName() != "Sales.Order_Customer" {
		t.Fatalf("association QN = %q, want Sales.Order_Customer", mc.AssociationQualifiedName())
	}
	if mc.AttributeQualifiedName() != "" {
		t.Fatalf("attribute QN should be empty, got %q", mc.AttributeQualifiedName())
	}
}

func TestAddChangeObjectActionGenAutoPromotesEmptyToRefresh(t *testing.T) {
	// CE0032: empty change with no commit must auto-set RefreshInClient
	// so Studio Pro doesn't reject the empty action.
	fb := newActionTestFb()
	fb.varTypes["Order"] = "Sales.Order"
	stmt := &ast.ChangeObjectStmt{Variable: "Order", Changes: nil}
	fb.addChangeObjectActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ChangeObjectAction)
	if !act.RefreshInClient() {
		t.Fatal("empty change must auto-promote RefreshInClient=true")
	}
	if act.Commit() != "No" {
		t.Fatalf("commit = %q, want No", act.Commit())
	}
}

func TestAddChangeObjectActionGenWithItems(t *testing.T) {
	fb := newActionTestFb()
	fb.varTypes["Order"] = "Sales.Order"
	stmt := &ast.ChangeObjectStmt{
		Variable: "Order",
		Changes: []ast.ChangeItem{
			{Attribute: "Amount", Value: &ast.LiteralExpr{Kind: ast.LiteralInteger, Value: int64(200)}},
		},
	}
	fb.addChangeObjectActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ChangeObjectAction)
	if act.RefreshInClient() {
		t.Fatal("with non-empty items, RefreshInClient must NOT auto-promote")
	}
	items := act.ItemsItems()
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	mc := items[0].(*genMf.MemberChange)
	if mc.AttributeQualifiedName() != "Sales.Order.Amount" {
		t.Fatalf("attribute QN = %q", mc.AttributeQualifiedName())
	}
}

func TestResolveMemberChangeGenOfflineFallback(t *testing.T) {
	fb := newActionTestFb()
	cases := []struct {
		name              string
		memberName        string
		entityQN          string
		wantAttribute     string
		wantAssociation   string
	}{
		{"bare attr with entity", "Status", "Sales.Order", "Sales.Order.Status", ""},
		{"bare attr no entity", "Status", "", "Status", ""},
		{"single-dot becomes association", "Sales.Order_Customer", "Sales.Order", "", "Sales.Order_Customer"},
		{"two-dot stays attribute", "Sales.Order.Status", "Sales.Order", "Sales.Order.Status", ""},
		{"empty member is no-op", "", "Sales.Order", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := fb.resolveMemberChangeGen(tc.memberName, tc.entityQN)
			if r.attributeQN != tc.wantAttribute {
				t.Fatalf("attribute = %q, want %q", r.attributeQN, tc.wantAttribute)
			}
			if r.associationQN != tc.wantAssociation {
				t.Fatalf("association = %q, want %q", r.associationQN, tc.wantAssociation)
			}
		})
	}
}

func TestApplyResolvedMemberChangeGenSelectsNonEmpty(t *testing.T) {
	mc := genMf.NewMemberChange()
	assignFreshID(mc)
	applyResolvedMemberChangeGen(mc, resolvedMemberChange{attributeQN: "Sales.Order.Status"})
	if mc.AttributeQualifiedName() != "Sales.Order.Status" {
		t.Fatalf("attribute = %q", mc.AttributeQualifiedName())
	}
	if mc.AssociationQualifiedName() != "" {
		t.Fatalf("association should remain empty when only attribute is supplied, got %q", mc.AssociationQualifiedName())
	}

	mc2 := genMf.NewMemberChange()
	assignFreshID(mc2)
	applyResolvedMemberChangeGen(mc2, resolvedMemberChange{associationQN: "Sales.Order_Customer"})
	if mc2.AssociationQualifiedName() != "Sales.Order_Customer" {
		t.Fatalf("association = %q", mc2.AssociationQualifiedName())
	}
}

func TestMemberExpressionToStringGenOfflineLeavesLiteralsUnchanged(t *testing.T) {
	// Offline (no backend) — the enum-rewrite branch is unreachable;
	// expressions render as plain quoted literals.
	fb := newActionTestFb()
	got := fb.memberExpressionToStringGen(
		&ast.LiteralExpr{Kind: ast.LiteralString, Value: "Open"},
		"Sales.Order", "Status",
	)
	if got != "'Open'" {
		t.Fatalf("got %q, want %q", got, "'Open'")
	}
}

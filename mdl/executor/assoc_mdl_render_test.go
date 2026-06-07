// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestRenderAssocMDL_Basic(t *testing.T) {
	spec := assocMDLSpec{
		module:         "Sales",
		name:           "Order_Customer",
		fromQN:         "Sales.Order",
		toQN:           "Sales.Customer",
		assocType:      "Reference",
		owner:          "Default",
		deleteBehavior: "DeleteMeButKeepReferences",
	}
	out := renderAssocMDL(spec)

	for _, want := range []string{
		"create association Sales.Order_Customer",
		"from Sales.Order to Sales.Customer",
		"type Reference",
		"owner Default",
		"delete behavior DeleteMeButKeepReferences",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered MDL missing %q\n--- got ---\n%s", want, out)
		}
	}
	if strings.Contains(out, "/**") {
		t.Errorf("expected no doc block, got:\n%s", out)
	}
}

func TestRenderAssocMDL_WithDocumentation(t *testing.T) {
	spec := assocMDLSpec{
		module:         "Sales",
		name:           "Order_Customer",
		fromQN:         "Sales.Order",
		toQN:           "Sales.Customer",
		documentation:  "Links an order to its customer.",
		assocType:      "ReferenceSet",
		owner:          "Both",
		deleteBehavior: "DeleteMeAndReferences",
	}
	out := renderAssocMDL(spec)

	if !strings.Contains(out, "/**\n * Links an order to its customer.\n */") {
		t.Errorf("doc block missing or malformed:\n%s", out)
	}
	if !strings.HasPrefix(out, "/**") {
		t.Errorf("doc block should be first:\n%s", out)
	}
	if !strings.Contains(out, "type ReferenceSet") {
		t.Errorf("expected ReferenceSet:\n%s", out)
	}
	if !strings.Contains(out, "owner Both") {
		t.Errorf("expected owner Both:\n%s", out)
	}
}

func TestAssocSpecFromAST(t *testing.T) {
	s := &ast.CreateAssociationStmt{
		Name:           ast.QualifiedName{Module: "Sales", Name: "Order_Customer"},
		Parent:         ast.QualifiedName{Module: "Sales", Name: "Order"},
		Child:          ast.QualifiedName{Module: "Sales", Name: "Customer"},
		Type:           ast.AssocReferenceSet,
		Owner:          ast.OwnerBoth,
		DeleteBehavior: ast.DeleteCascade,
		Documentation:  "doc text",
	}
	spec := assocSpecFromAST(s)

	if spec.module != "Sales" || spec.name != "Order_Customer" {
		t.Errorf("name mismatch: module=%q name=%q", spec.module, spec.name)
	}
	if spec.fromQN != "Sales.Order" {
		t.Errorf("fromQN = %q, want Sales.Order", spec.fromQN)
	}
	if spec.toQN != "Sales.Customer" {
		t.Errorf("toQN = %q, want Sales.Customer", spec.toQN)
	}
	if spec.assocType != "ReferenceSet" {
		t.Errorf("assocType = %q, want ReferenceSet", spec.assocType)
	}
	if spec.owner != "Both" {
		t.Errorf("owner = %q, want Both", spec.owner)
	}
	if spec.deleteBehavior != "DeleteMeAndReferences" {
		t.Errorf("deleteBehavior = %q, want DeleteMeAndReferences", spec.deleteBehavior)
	}
	if spec.documentation != "doc text" {
		t.Errorf("documentation = %q", spec.documentation)
	}
}

func TestAstDeleteBehaviorStr(t *testing.T) {
	cases := []struct {
		in   ast.DeleteBehavior
		want string
	}{
		{ast.DeleteCascade, "DeleteMeAndReferences"},
		{ast.DeleteBoth, "DeleteBoth"},
		{ast.DeleteKeepParentDeleteChild, "KeepParentDeleteChild"},
		{ast.DeleteKeepChildDeleteParent, "KeepChildDeleteParent"},
		{ast.DeleteIfNoReferences, "DeleteIfNoReferences"},
		{ast.DeleteBehavior(-1), "DeleteMeButKeepReferences"},
	}
	for _, c := range cases {
		if got := astDeleteBehaviorStr(c.in); got != c.want {
			t.Errorf("astDeleteBehaviorStr(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestAstAssocTypeAndOwnerStr(t *testing.T) {
	if got := astAssocTypeStr(ast.AssocReferenceSet); got != "ReferenceSet" {
		t.Errorf("astAssocTypeStr(ReferenceSet) = %q", got)
	}
	if got := astAssocTypeStr(ast.AssocReference); got != "Reference" {
		t.Errorf("astAssocTypeStr(Reference) = %q", got)
	}
	if got := astOwnerStr(ast.OwnerBoth); got != "Both" {
		t.Errorf("astOwnerStr(OwnerBoth) = %q", got)
	}
	if got := astOwnerStr(ast.OwnerDefault); got != "Default" {
		t.Errorf("astOwnerStr(OwnerDefault) = %q", got)
	}
}

func TestGenAssocDeleteBehaviorToMDL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"DeleteMeAndReferences", "DeleteMeAndReferences"},
		{"DeleteBoth", "DeleteBoth"},
		{"KeepParentDeleteChild", "KeepParentDeleteChild"},
		{"KeepChildDeleteParent", "KeepChildDeleteParent"},
		{"DeleteMeIfNoReferences", "DeleteIfNoReferences"},
		{"DeleteMeButKeepReferences", "DeleteMeButKeepReferences"},
		{"", "DeleteMeButKeepReferences"},
	}
	for _, c := range cases {
		if got := genAssocDeleteBehaviorToMDL(c.in); got != c.want {
			t.Errorf("genAssocDeleteBehaviorToMDL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

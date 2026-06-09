// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
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
		"create or modify association Sales.Order_Customer",
		"from Sales.Order to Sales.Customer",
		"type Reference",
		"owner Default",
		"delete_behavior DeleteMeButKeepReferences",
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
	if spec.deleteBehavior != "DELETE_AND_REFERENCES" {
		t.Errorf("deleteBehavior = %q, want DELETE_AND_REFERENCES", spec.deleteBehavior)
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
		// MDL grammar tokens: DELETE_AND_REFERENCES, DELETE_BUT_KEEP_REFERENCES,
		// DELETE_IF_NO_REFERENCES, CASCADE, PREVENT.
		// BSON values with no direct MDL equivalent fall back to DELETE_BUT_KEEP_REFERENCES.
		{ast.DeleteCascade, "DELETE_AND_REFERENCES"},
		{ast.DeleteBoth, "DELETE_BUT_KEEP_REFERENCES"},             // no MDL equivalent
		{ast.DeleteKeepParentDeleteChild, "DELETE_BUT_KEEP_REFERENCES"}, // no MDL equivalent
		{ast.DeleteKeepChildDeleteParent, "DELETE_BUT_KEEP_REFERENCES"}, // no MDL equivalent
		{ast.DeleteIfNoReferences, "DELETE_IF_NO_REFERENCES"},
		{ast.DeleteBehavior(-1), "DELETE_BUT_KEEP_REFERENCES"},
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
		// MDL grammar tokens: DELETE_AND_REFERENCES, DELETE_BUT_KEEP_REFERENCES,
		// DELETE_IF_NO_REFERENCES. BSON values without a direct MDL equivalent
		// fall back to DELETE_BUT_KEEP_REFERENCES (lossy but grammar-valid).
		{"DeleteMeAndReferences", "DELETE_AND_REFERENCES"},
		{"DeleteBoth", "DELETE_BUT_KEEP_REFERENCES"},
		{"KeepParentDeleteChild", "DELETE_BUT_KEEP_REFERENCES"},
		{"KeepChildDeleteParent", "DELETE_BUT_KEEP_REFERENCES"},
		{"DeleteMeIfNoReferences", "DELETE_IF_NO_REFERENCES"},
		{"DeleteMeButKeepReferences", "DELETE_BUT_KEEP_REFERENCES"},
		{"", "DELETE_BUT_KEEP_REFERENCES"},
	}
	for _, c := range cases {
		if got := genAssocDeleteBehaviorToMDL(c.in); got != c.want {
			t.Errorf("genAssocDeleteBehaviorToMDL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestAssocSpecFromGen(t *testing.T) {
	const parentID = element.ID("parent-uuid-001")
	const childID = element.ID("child-uuid-002")

	a := genDm.NewAssociation()
	a.SetName("Order_Customer")
	a.SetType("ReferenceSet")
	a.SetOwner("Both")
	a.SetDocumentation("Links an order to its customer.")
	a.SetParentID(parentID)
	a.SetChildID(childID)

	dbe := genDm.NewAssociationDeleteBehavior()
	dbe.SetChildDeleteBehavior("DeleteMeAndReferences")
	a.SetDeleteBehavior(dbe)

	entityNames := map[string]string{
		string(parentID): "Sales.Order",
		string(childID):  "Sales.Customer",
	}

	spec := assocSpecFromGen("Sales", a, entityNames)

	if spec.module != "Sales" {
		t.Errorf("module = %q, want Sales", spec.module)
	}
	if spec.name != "Order_Customer" {
		t.Errorf("name = %q, want Order_Customer", spec.name)
	}
	// ParentRefID maps to fromQN (FROM entity)
	if spec.fromQN != "Sales.Order" {
		t.Errorf("fromQN = %q, want Sales.Order", spec.fromQN)
	}
	// ChildRefID maps to toQN (TO entity)
	if spec.toQN != "Sales.Customer" {
		t.Errorf("toQN = %q, want Sales.Customer", spec.toQN)
	}
	if spec.assocType != "ReferenceSet" {
		t.Errorf("assocType = %q, want ReferenceSet", spec.assocType)
	}
	if spec.owner != "Both" {
		t.Errorf("owner = %q, want Both", spec.owner)
	}
	if spec.deleteBehavior != "DELETE_AND_REFERENCES" {
		t.Errorf("deleteBehavior = %q, want DELETE_AND_REFERENCES", spec.deleteBehavior)
	}
	if spec.documentation != "Links an order to its customer." {
		t.Errorf("documentation = %q", spec.documentation)
	}

	// Full render round-trip: output must be parseable MDL containing key tokens.
	out := renderAssocMDL(spec)
	for _, want := range []string{
		"create or modify association Sales.Order_Customer",
		"from Sales.Order to Sales.Customer",
		"type ReferenceSet",
		"owner Both",
		"delete_behavior DELETE_AND_REFERENCES",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("renderAssocMDL output missing %q\n--- got ---\n%s", want, out)
		}
	}
}

func TestAssocSpecFromGen_FallbackToRawID(t *testing.T) {
	// When entityNames does not contain the ID, the raw ID string is used as-is.
	const parentID = element.ID("unknown-parent-id")
	const childID = element.ID("unknown-child-id")

	a := genDm.NewAssociation()
	a.SetName("Foo_Bar")
	a.SetParentID(parentID)
	a.SetChildID(childID)

	spec := assocSpecFromGen("Mod", a, map[string]string{})

	if spec.fromQN != "unknown-parent-id" {
		t.Errorf("fromQN fallback = %q, want unknown-parent-id", spec.fromQN)
	}
	if spec.toQN != "unknown-child-id" {
		t.Errorf("toQN fallback = %q, want unknown-child-id", spec.toQN)
	}
}

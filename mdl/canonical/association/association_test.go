// SPDX-License-Identifier: Apache-2.0

package association_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/canonical/association"
	"github.com/stretchr/testify/assert"
)

func TestLift_Reference(t *testing.T) {
	stmt := &ast.CreateAssociationStmt{
		Name:           ast.QualifiedName{Module: "M", Name: "Order_Customer"},
		Parent:         ast.QualifiedName{Module: "M", Name: "Order"},
		Child:          ast.QualifiedName{Module: "M", Name: "Customer"},
		Type:           ast.AssocReference,
		Owner:          ast.OwnerDefault,
		Storage:        ast.StorageTable,
		DeleteBehavior: ast.DeleteKeepReferences,
		Documentation:  "Links orders to customers",
	}
	m := association.Lift(stmt)
	assert.Equal(t, "M.Order_Customer", m.Name.String())
	assert.Equal(t, "M.Order", m.From.String())
	assert.Equal(t, "M.Customer", m.To.String())
	assert.Equal(t, association.AssocReference, m.Type)
	assert.Equal(t, association.OwnerDefault, m.Owner)
	assert.Equal(t, association.StorageTable, m.Storage)
	assert.Equal(t, association.DeleteKeepReferences, m.DeleteBehavior)
	assert.Equal(t, "Links orders to customers", m.Documentation)
}

func TestLift_ReferenceSet(t *testing.T) {
	stmt := &ast.CreateAssociationStmt{
		Name:   ast.QualifiedName{Module: "M", Name: "Product_Tag"},
		Parent: ast.QualifiedName{Module: "M", Name: "Product"},
		Child:  ast.QualifiedName{Module: "M", Name: "Tag"},
		Type:   ast.AssocReferenceSet,
		Owner:  ast.OwnerBoth,
	}
	m := association.Lift(stmt)
	assert.Equal(t, association.AssocReferenceSet, m.Type)
	assert.Equal(t, association.OwnerBoth, m.Owner)
}

func TestLift_CascadeDelete(t *testing.T) {
	stmt := &ast.CreateAssociationStmt{
		Name:           ast.QualifiedName{Module: "M", Name: "Order_Line"},
		Parent:         ast.QualifiedName{Module: "M", Name: "Order"},
		Child:          ast.QualifiedName{Module: "M", Name: "OrderLine"},
		Type:           ast.AssocReference,
		DeleteBehavior: ast.DeleteCascade,
	}
	m := association.Lift(stmt)
	assert.Equal(t, association.DeleteCascade, m.DeleteBehavior)
}

func TestToMDL_Reference(t *testing.T) {
	m := &association.AssociationModel{
		Name:           association.QualifiedName{Module: "M", Name: "Order_Customer"},
		From:           association.QualifiedName{Module: "M", Name: "Order"},
		To:             association.QualifiedName{Module: "M", Name: "Customer"},
		Type:           association.AssocReference,
		Owner:          association.OwnerDefault,
		Storage:        association.StorageTable,
		DeleteBehavior: association.DeleteKeepReferences,
	}
	want := "create association M.Order_Customer\nfrom M.Order to M.Customer\ntype Reference\nowner Default\ndelete behavior DeleteMeButKeepReferences"
	assert.Equal(t, want, m.ToMDL())
}

func TestToMDL_ReferenceSetWithDoc(t *testing.T) {
	m := &association.AssociationModel{
		Name:          association.QualifiedName{Module: "M", Name: "Product_Tag"},
		From:          association.QualifiedName{Module: "M", Name: "Product"},
		To:            association.QualifiedName{Module: "M", Name: "Tag"},
		Type:          association.AssocReferenceSet,
		Owner:         association.OwnerBoth,
		Storage:       association.StorageTable,
		Documentation: "Product tags",
	}
	got := m.ToMDL()
	assert.Contains(t, got, "/**\n * Product tags\n */\n")
	assert.Contains(t, got, "type ReferenceSet")
	assert.Contains(t, got, "owner Both")
}

func TestToMDL_CascadeDelete(t *testing.T) {
	m := &association.AssociationModel{
		Name:           association.QualifiedName{Module: "M", Name: "Order_Line"},
		From:           association.QualifiedName{Module: "M", Name: "Order"},
		To:             association.QualifiedName{Module: "M", Name: "OrderLine"},
		Type:           association.AssocReference,
		DeleteBehavior: association.DeleteCascade,
	}
	assert.Contains(t, m.ToMDL(), "delete behavior DeleteMeAndReferences")
}

// TestToMDL_LiftRoundTrip verifies Lift→ToMDL is deterministic.
func TestToMDL_LiftRoundTrip(t *testing.T) {
	stmt := &ast.CreateAssociationStmt{
		Name:   ast.QualifiedName{Module: "M", Name: "A_B"},
		Parent: ast.QualifiedName{Module: "M", Name: "A"},
		Child:  ast.QualifiedName{Module: "M", Name: "B"},
		Type:   ast.AssocReference,
	}
	m := association.Lift(stmt)
	got := m.ToMDL()
	assert.Contains(t, got, "create association M.A_B")
	assert.Contains(t, got, "from M.A to M.B")
}

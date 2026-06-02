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

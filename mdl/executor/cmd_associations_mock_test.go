// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

func TestShowAssociations_Mock(t *testing.T) {
	mod := mkModule("MyModule")
	ent1 := mkEntityGen("Order")
	ent2 := mkEntityGen("Customer")
	assoc := mkAssociationGen("Order_Customer", model.ID(ent1.ID()), model.ID(ent2.ID()))
	dm := mkDomainModelGen(mod.ID, ent1, ent2)
	dm.AddAssociations(assoc)
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{mod.ID: {dm}})

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withDomainModelsRepo(dmRepo))
	assertNoError(t, listAssociationsGen(ctx, ""))

	out := buf.String()
	assertContainsStr(t, out, "MyModule.Order_Customer")
	assertContainsStr(t, out, "MyModule.Order")
	assertContainsStr(t, out, "MyModule.Customer")
	assertContainsStr(t, out, "Reference")
	assertContainsStr(t, out, "(1 associations)")
}

func TestShowAssociations_Mock_FilterByModule(t *testing.T) {
	mod1 := mkModule("Sales")
	mod2 := mkModule("HR")
	ent1 := mkEntityGen("Order")
	ent2 := mkEntityGen("Product")
	ent3 := mkEntityGen("Employee")
	ent4 := mkEntityGen("Department")

	dm1 := mkDomainModelGen(mod1.ID, ent1, ent2)
	dm1.AddAssociations(mkAssociationGen("Order_Product", model.ID(ent1.ID()), model.ID(ent2.ID())))
	dm2 := mkDomainModelGen(mod2.ID, ent3, ent4)
	dm2.AddAssociations(mkAssociationGen("Employee_Dept", model.ID(ent3.ID()), model.ID(ent4.ID())))
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{
		mod1.ID: {dm1},
		mod2.ID: {dm2},
	})

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod1, mod2}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withDomainModelsRepo(dmRepo))
	assertNoError(t, listAssociationsGen(ctx, "HR"))

	out := buf.String()
	assertNotContainsStr(t, out, "Sales.Order_Product")
	assertContainsStr(t, out, "HR.Employee_Dept")
	assertContainsStr(t, out, "(1 associations)")
}

// NOTE: listAssociations and describeAssociation have no Connected() guard.
// They call backend directly — error propagation is the only failure mode.

func TestShowAssociations_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return nil, fmt.Errorf("connection lost") },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listAssociations(ctx, ""))
}

func TestShowAssociations_JSON(t *testing.T) {
	mod := mkModule("App")
	ent1 := mkEntityGen("A")
	ent2 := mkEntityGen("B")
	assoc := mkAssociationGen("A_B", model.ID(ent1.ID()), model.ID(ent2.ID()))
	dm := mkDomainModelGen(mod.ID, ent1, ent2)
	dm.AddAssociations(assoc)
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{mod.ID: {dm}})

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withDomainModelsRepo(dmRepo))
	assertNoError(t, listAssociationsGen(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "A_B")
}

func TestCreateAssociation_OrModify_UpdatesInPlace(t *testing.T) {
	mod := mkModule("MyModule")
	ent1 := mkEntityGen("Order")
	ent2 := mkEntityGen("Customer")
	assocID := nextID("assoc")
	existingAssoc := mkAssociationGen("Order_Customer", model.ID(ent1.ID()), model.ID(ent2.ID()))
	existingAssoc.SetID(element.ID(assocID))

	dm := mkDomainModelGen(mod.ID, ent1, ent2)
	dm.SetID(element.ID(nextID("dm")))
	dm.AddAssociations(existingAssoc)
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{mod.ID: {dm}})

	updateCalled := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelGenFunc: func(id model.ID) (*genDm.DomainModel, error) {
			return dm, nil
		},
		UpdateDomainModelGenFunc: func(d *genDm.DomainModel) error {
			updateCalled = true
			return nil
		},
		ReconcileMemberAccessesFunc: func(dmID model.ID, moduleName string) (int, error) { return 0, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withDomainModelsRepo(dmRepo))
	err := execCreateAssociation(ctx, &ast.CreateAssociationStmt{
		Name:           ast.QualifiedName{Module: "MyModule", Name: "Order_Customer"},
		Parent:         ast.QualifiedName{Module: "MyModule", Name: "Order"},
		Child:          ast.QualifiedName{Module: "MyModule", Name: "Customer"},
		Type:           ast.AssocReference,
		CreateOrModify: true,
	})
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Modified association")
	if !updateCalled {
		t.Fatal("UpdateDomainModelGen was not called")
	}
	// Verify the existing association UUID is preserved
	if model.ID(existingAssoc.ID()) != assocID {
		t.Errorf("Association ID changed from %q to %q", assocID, existingAssoc.ID())
	}
}

// Issue #389 — cross-module CREATE ASSOCIATION must also reject duplicates.
func TestCreateAssociation_CrossModule_AlreadyExists_Issue389(t *testing.T) {
	mod1 := mkModule("ModA")
	mod2 := mkModule("ModB")
	ent1 := mkEntityGen("Order")
	ent2 := mkEntityGen("Product")

	existingCA := genDm.NewCrossAssociation()
	existingCA.SetID(element.ID(nextID("ca")))
	existingCA.SetName("Order_Product")
	existingCA.SetParentID(ent1.ID())
	existingCA.SetChildQualifiedName("ModB.Product")
	dm1 := mkDomainModelGen(mod1.ID, ent1)
	dm1.SetID(element.ID(nextID("dm1")))
	dm1.AddCrossAssociations(existingCA)
	dm2 := mkDomainModelGen(mod2.ID, ent2)
	dm2.SetID(element.ID(nextID("dm2")))
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{
		mod1.ID: {dm1},
		mod2.ID: {dm2},
	})

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{mod1, mod2}, nil
		},
		GetDomainModelGenFunc: func(id model.ID) (*genDm.DomainModel, error) {
			if id == mod1.ID {
				return dm1, nil
			}
			return dm2, nil
		},
	}

	ctx, _ := newMockCtx(t, withBackend(mb), withDomainModelsRepo(dmRepo))
	err := execCreateAssociation(ctx, &ast.CreateAssociationStmt{
		Name:   ast.QualifiedName{Module: "ModA", Name: "Order_Product"},
		Parent: ast.QualifiedName{Module: "ModA", Name: "Order"},
		Child:  ast.QualifiedName{Module: "ModB", Name: "Product"},
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "already exists")
}

func TestCreateAssociation_AlreadyExists_NoOrModify(t *testing.T) {
	mod := mkModule("MyModule")
	ent1 := mkEntityGen("Order")
	ent2 := mkEntityGen("Customer")
	existingAssoc := mkAssociationGen("Order_Customer", model.ID(ent1.ID()), model.ID(ent2.ID()))

	dm := mkDomainModelGen(mod.ID, ent1, ent2)
	dm.SetID(element.ID(nextID("dm")))
	dm.AddAssociations(existingAssoc)
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{mod.ID: {dm}})

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelGenFunc: func(id model.ID) (*genDm.DomainModel, error) {
			return dm, nil
		},
	}

	ctx, _ := newMockCtx(t, withBackend(mb), withDomainModelsRepo(dmRepo))
	err := execCreateAssociation(ctx, &ast.CreateAssociationStmt{
		Name:   ast.QualifiedName{Module: "MyModule", Name: "Order_Customer"},
		Parent: ast.QualifiedName{Module: "MyModule", Name: "Order"},
		Child:  ast.QualifiedName{Module: "MyModule", Name: "Customer"},
	})
	assertError(t, err)
}

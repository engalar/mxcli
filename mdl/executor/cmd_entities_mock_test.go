// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
)

func TestShowEntities_Mock(t *testing.T) {
	mod := mkModule("MyModule")
	ent1 := mkEntityGen("Customer")
	ent2 := mkEntityGen("Order")
	dm := mkDomainModelGen(mod.ID, ent1, ent2)
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{mod.ID: {dm}})

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withDomainModelsRepo(dmRepo))
	assertNoError(t, listEntitiesGen(ctx, ""))

	out := buf.String()
	assertContainsStr(t, out, "MyModule.Customer")
	assertContainsStr(t, out, "MyModule.Order")
	assertContainsStr(t, out, "Persistent")
	assertContainsStr(t, out, "(2 entities)")
}

func TestShowEntities_Mock_FilterByModule(t *testing.T) {
	mod1 := mkModule("Sales")
	mod2 := mkModule("HR")
	ent1 := mkEntityGen("Product")
	ent2 := mkEntityGen("Employee")

	dm1 := mkDomainModelGen(mod1.ID, ent1)
	dm2 := mkDomainModelGen(mod2.ID, ent2)
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{
		mod1.ID: {dm1},
		mod2.ID: {dm2},
	})

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod1, mod2}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withDomainModelsRepo(dmRepo))
	assertNoError(t, listEntitiesGen(ctx, "HR"))

	out := buf.String()
	assertNotContainsStr(t, out, "Sales.Product")
	assertContainsStr(t, out, "HR.Employee")
	assertContainsStr(t, out, "(1 entities)")
}

// NOTE: listEntities has no Connected() guard — it calls backend directly.

func TestShowEntities_BackendError_Modules(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return nil, fmt.Errorf("not connected") },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listEntities(ctx, ""))
}

func TestShowEntities_DomainModelRepoErrorIgnored(t *testing.T) {
	mod := mkModule("Sales")
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
	}
	dmRepo := makeDomainModelsRepo(nil)
	dmRepo.ListFunc = func(moduleID model.ID) ([]*genDm.DomainModel, error) {
		return nil, fmt.Errorf("backend down")
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withDomainModelsRepo(dmRepo))
	assertNoError(t, listEntitiesGen(ctx, ""))
	assertContainsStr(t, buf.String(), "(0 entities)")
}

// Issue #392 — CREATE ENTITY must reject attribute types that don't resolve
// to a known primitive, enumeration, or entity.
func TestCreateEntity_UnknownAttributeType_Issue392(t *testing.T) {
	mod := mkModule("M")
	dm := mkDomainModel(mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:      func() bool { return true },
		ListModulesFunc:      func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListDomainModelsFunc: func() ([]*domainmodel.DomainModel, error) { return []*domainmodel.DomainModel{dm}, nil },
		GetDomainModelFunc:   func(id model.ID) (*domainmodel.DomainModel, error) { return dm, nil },
		ListEnumerationsFunc: func() ([]*model.Enumeration, error) { return nil, nil },
	}
	h := mkHierarchy(mod)
	withContainer(h, dm.ID, mod.ID)

	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execCreateEntity(ctx, &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "M", Name: "E"},
		Kind: ast.EntityPersistent,
		Attributes: []ast.Attribute{
			{
				Name: "Field1",
				Type: ast.DataType{
					Kind:    ast.TypeEnumeration,
					EnumRef: &ast.QualifiedName{Name: "invalidtype"},
				},
			},
		},
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "invalidtype")
}

// TestAlterEntity_AllowCreateChangeLocally_Issue534 verifies that
// ALTER ENTITY SET ALLOW_CREATE_CHANGE_LOCALLY = true sets CreateChangeLocally on the entity.
func TestAlterEntity_AllowCreateChangeLocally_Issue534(t *testing.T) {
	mod := mkModule("TripPin")
	entity := &domainmodel.Entity{
		BaseElement: model.BaseElement{ID: nextID("ent")},
		Name:        "People",
	}
	dm := mkDomainModel(mod.ID, entity)

	var updated *domainmodel.Entity
	mb := &mock.MockBackend{
		IsConnectedFunc:    func() bool { return true },
		ListModulesFunc:    func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelFunc: func(id model.ID) (*domainmodel.DomainModel, error) { return dm, nil },
		UpdateEntityFunc:   func(dmID model.ID, e *domainmodel.Entity) error { updated = e; return nil },
	}

	ctx, _ := newMockCtx(t, withBackend(mb))
	err := execAlterEntity(ctx, &ast.AlterEntityStmt{
		Name:      ast.QualifiedName{Module: "TripPin", Name: "People"},
		Operation: ast.AlterEntitySetAllowCreateChangeLocally,
		BoolValue: true,
	})
	assertNoError(t, err)
	if updated == nil {
		t.Fatal("expected UpdateEntity to be called")
	}
	if !updated.CreateChangeLocally {
		t.Errorf("expected CreateChangeLocally = true, got false")
	}
}

func TestShowEntities_JSON(t *testing.T) {
	mod := mkModule("App")
	ent := mkEntityGen("Item")
	dm := mkDomainModelGen(mod.ID, ent)
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{mod.ID: {dm}})

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withDomainModelsRepo(dmRepo))
	assertNoError(t, listEntitiesGen(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "Item")
}

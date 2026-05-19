// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strings"
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
	assertNoError(t, listAssociations(ctx, ""))

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
	assertNoError(t, listAssociations(ctx, "HR"))

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
	assertNoError(t, listAssociations(ctx, ""))
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
		ReconcileMemberAccessesFunc: func(dmID model.ID, moduleName string) ([]string, error) { return nil, nil },
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

// --- Improvement 1: Multiplicity column in listAssociations ---

func TestShowAssociations_Multiplicity_OneToMany(t *testing.T) {
	mod := mkModule("Sales")
	ent1 := mkEntityGen("Order")
	ent2 := mkEntityGen("Customer")
	assoc := mkAssociationGen("Order_Customer", model.ID(ent1.ID()), model.ID(ent2.ID()))
	// Reference + Default = one-to-many (defaults from mkAssociationGen)
	dm := mkDomainModelGen(mod.ID, ent1, ent2)
	dm.AddAssociations(assoc)
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{mod.ID: {dm}})
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withDomainModelsRepo(dmRepo))
	assertNoError(t, listAssociations(ctx, ""))
	out := buf.String()
	assertContainsStr(t, out, "Multiplicity")
	assertContainsStr(t, out, "*-->1")
	assertContainsStr(t, out, "FROM (owner)")
	assertContainsStr(t, out, "TO (referenced)")
}

func TestShowAssociations_Multiplicity_OneToOne(t *testing.T) {
	mod := mkModule("Sales")
	ent1 := mkEntityGen("Customer")
	ent2 := mkEntityGen("Profile")
	assoc := mkAssociationGen("Customer_Profile", model.ID(ent1.ID()), model.ID(ent2.ID()))
	assoc.SetOwner(genDm.AssociationOwnerBoth) // Reference + Both = 1-1
	dm := mkDomainModelGen(mod.ID, ent1, ent2)
	dm.AddAssociations(assoc)
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{mod.ID: {dm}})
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withDomainModelsRepo(dmRepo))
	assertNoError(t, listAssociations(ctx, ""))
	assertContainsStr(t, buf.String(), "1--1")
}

func TestShowAssociations_Multiplicity_ManyToMany_Default(t *testing.T) {
	mod := mkModule("Sales")
	ent1 := mkEntityGen("Customer")
	ent2 := mkEntityGen("Group")
	assoc := mkAssociationGen("Customer_Group", model.ID(ent1.ID()), model.ID(ent2.ID()))
	assoc.SetType(genDm.AssociationTypeReferenceSet)
	assoc.SetOwner(genDm.AssociationOwnerDefault)
	dm := mkDomainModelGen(mod.ID, ent1, ent2)
	dm.AddAssociations(assoc)
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{mod.ID: {dm}})
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withDomainModelsRepo(dmRepo))
	assertNoError(t, listAssociations(ctx, ""))
	assertContainsStr(t, buf.String(), "*-->*")
}

func TestShowAssociations_Multiplicity_ManyToMany_Both(t *testing.T) {
	mod := mkModule("Sales")
	ent1 := mkEntityGen("Accountant")
	ent2 := mkEntityGen("Group")
	assoc := mkAssociationGen("Accountant_Group", model.ID(ent1.ID()), model.ID(ent2.ID()))
	assoc.SetType(genDm.AssociationTypeReferenceSet)
	assoc.SetOwner(genDm.AssociationOwnerBoth)
	dm := mkDomainModelGen(mod.ID, ent1, ent2)
	dm.AddAssociations(assoc)
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{mod.ID: {dm}})
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withDomainModelsRepo(dmRepo))
	assertNoError(t, listAssociations(ctx, ""))
	assertContainsStr(t, buf.String(), "*--*")
}

// --- Improvement 2: SHOW ASSOCIATION (single) shows Parent/Child entity names ---

func TestShowAssociation_Single_ShowsParentAndChild(t *testing.T) {
	mod := mkModule("Sales")
	ent1 := mkEntityGen("Order")
	ent2 := mkEntityGen("Customer")
	assoc := mkAssociationGen("Order_Customer", model.ID(ent1.ID()), model.ID(ent2.ID()))
	dm := mkDomainModelGen(mod.ID, ent1, ent2)
	dm.SetID(element.ID(nextID("dm")))
	dm.AddAssociations(assoc)
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{mod.ID: {dm}})
	mb := &mock.MockBackend{
		IsConnectedFunc:       func() bool { return true },
		ListModulesFunc:       func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelGenFunc: func(id model.ID) (*genDm.DomainModel, error) { return dm, nil },
	}
	name := ast.QualifiedName{Module: "Sales", Name: "Order_Customer"}
	ctx, buf := newMockCtx(t, withBackend(mb), withDomainModelsRepo(dmRepo))
	assertNoError(t, listAssociation(ctx, &name))
	out := buf.String()
	assertContainsStr(t, out, "FROM (owner)")
	assertContainsStr(t, out, "TO (referenced)")
	assertContainsStr(t, out, "Sales.Order")
	assertContainsStr(t, out, "Sales.Customer")
	assertContainsStr(t, out, "*-->1")
}

// --- Improvement 3: DESCRIBE ASSOCIATION emits a human-readable comment ---

func TestDescribeAssociation_Comment_OneToMany(t *testing.T) {
	mod := mkModule("Sales")
	ent1 := mkEntityGen("Order")
	ent2 := mkEntityGen("Customer")
	assoc := mkAssociationGen("Order_Customer", model.ID(ent1.ID()), model.ID(ent2.ID()))
	// Reference + Default = one-to-many
	dm := mkDomainModelGen(mod.ID, ent1, ent2)
	dm.AddAssociations(assoc)
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{mod.ID: {dm}})
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withDomainModelsRepo(dmRepo))
	assertNoError(t, describeAssociation(ctx, ast.QualifiedName{Module: "Sales", Name: "Order_Customer"}))
	out := buf.String()
	if !strings.Contains(out, "-- one-to-many") {
		t.Errorf("expected '-- one-to-many' comment in describe output, got:\n%s", out)
	}
}

func TestDescribeAssociation_Comment_OneToOne(t *testing.T) {
	mod := mkModule("Sales")
	ent1 := mkEntityGen("Customer")
	ent2 := mkEntityGen("Profile")
	assoc := mkAssociationGen("Customer_Profile", model.ID(ent1.ID()), model.ID(ent2.ID()))
	assoc.SetOwner(genDm.AssociationOwnerBoth)
	dm := mkDomainModelGen(mod.ID, ent1, ent2)
	dm.AddAssociations(assoc)
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{mod.ID: {dm}})
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withDomainModelsRepo(dmRepo))
	assertNoError(t, describeAssociation(ctx, ast.QualifiedName{Module: "Sales", Name: "Customer_Profile"}))
	if !strings.Contains(buf.String(), "-- one-to-one") {
		t.Errorf("expected '-- one-to-one' comment, got:\n%s", buf.String())
	}
}

func TestDescribeAssociation_Comment_ManyToMany(t *testing.T) {
	mod := mkModule("Sales")
	ent1 := mkEntityGen("Customer")
	ent2 := mkEntityGen("Group")
	assoc := mkAssociationGen("Customer_Group", model.ID(ent1.ID()), model.ID(ent2.ID()))
	assoc.SetType(genDm.AssociationTypeReferenceSet)
	dm := mkDomainModelGen(mod.ID, ent1, ent2)
	dm.AddAssociations(assoc)
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{mod.ID: {dm}})
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withDomainModelsRepo(dmRepo))
	assertNoError(t, describeAssociation(ctx, ast.QualifiedName{Module: "Sales", Name: "Customer_Group"}))
	if !strings.Contains(buf.String(), "-- many-to-many") {
		t.Errorf("expected '-- many-to-many' comment, got:\n%s", buf.String())
	}
}

// --- Improvement 4: CREATE ASSOCIATION rejects persistable entity as owner when paired with non-persistable ---

func mkNonPersistableEntityGen(name string) *genDm.Entity {
	ent := genDm.NewEntity()
	ent.SetID(element.ID(nextID("ent")))
	ent.SetName(name)
	ent.SetGeneralization(mkNoGeneralizationGen(false)) // false = non-persistable
	return ent
}

func TestCreateAssociation_PersistableOwner_NonPersistableChild_Rejected(t *testing.T) {
	mod := mkModule("MyModule")
	persistEnt := mkEntityGen("Order")           // persistable (FROM side = owner)
	nonPersistEnt := mkNonPersistableEntityGen("Filter") // non-persistable (TO side)
	dm := mkDomainModelGen(mod.ID, persistEnt, nonPersistEnt)
	dm.SetID(element.ID(nextID("dm")))
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{mod.ID: {dm}})
	mb := &mock.MockBackend{
		IsConnectedFunc:       func() bool { return true },
		ListModulesFunc:       func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelGenFunc: func(id model.ID) (*genDm.DomainModel, error) { return dm, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withDomainModelsRepo(dmRepo))
	err := execCreateAssociation(ctx, &ast.CreateAssociationStmt{
		Name:   ast.QualifiedName{Module: "MyModule", Name: "Order_Filter"},
		Parent: ast.QualifiedName{Module: "MyModule", Name: "Order"},   // persistable = FROM
		Child:  ast.QualifiedName{Module: "MyModule", Name: "Filter"},  // non-persistable = TO
	})
	if err == nil {
		t.Fatal("expected error: persistable entity cannot be owner when paired with non-persistable")
	}
	if !strings.Contains(err.Error(), "non-persistent") && !strings.Contains(err.Error(), "owner") {
		t.Errorf("expected error to mention non-persistent/owner, got: %v", err)
	}
}

func TestCreateAssociation_NonPersistableOwner_PersistableChild_Accepted(t *testing.T) {
	mod := mkModule("MyModule")
	nonPersistEnt := mkNonPersistableEntityGen("Filter") // non-persistable (FROM side = owner ✓)
	persistEnt := mkEntityGen("Result")                  // persistable (TO side)
	dm := mkDomainModelGen(mod.ID, nonPersistEnt, persistEnt)
	dm.SetID(element.ID(nextID("dm")))
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{mod.ID: {dm}})
	created := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelGenFunc: func(id model.ID) (*genDm.DomainModel, error) { return dm, nil },
		CreateAssociationGenFunc: func(dmID model.ID, a *genDm.Association) error {
			created = true
			return nil
		},
		ReconcileMemberAccessesFunc: func(dmID model.ID, moduleName string) ([]string, error) { return nil, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withDomainModelsRepo(dmRepo))
	err := execCreateAssociation(ctx, &ast.CreateAssociationStmt{
		Name:   ast.QualifiedName{Module: "MyModule", Name: "Filter_Result"},
		Parent: ast.QualifiedName{Module: "MyModule", Name: "Filter"}, // non-persistable = FROM ✓
		Child:  ast.QualifiedName{Module: "MyModule", Name: "Result"}, // persistable = TO ✓
	})
	if err != nil {
		t.Fatalf("expected no error for valid non-persistable→persistable association, got: %v", err)
	}
	if !created {
		t.Fatal("CreateAssociationGen was not called")
	}
}

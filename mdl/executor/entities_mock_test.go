// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genRest "github.com/mendixlabs/mxcli/modelsdk/gen/rest"
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
	dm := mkDomainModelGen(mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:         func() bool { return true },
		ListModulesFunc:         func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListDomainModelsGenFunc: func() ([]*genDm.DomainModel, error) { return []*genDm.DomainModel{dm}, nil },
		GetDomainModelGenFunc:   func(id model.ID) (*genDm.DomainModel, error) { return dm, nil },
		ListEnumerationsFunc:    func() ([]*model.Enumeration, error) { return nil, nil },
	}
	h := mkHierarchy(mod)
	withContainer(h, model.ID(dm.ID()), mod.ID)

	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := ExecCreateEntity(ctx, &ast.CreateEntityStmt{
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
	// Use gen entity with an OData remote source (CreateChangeLocally is on the source)
	entity := mkEntityGen("People")
	src := genRest.NewODataRemoteEntitySource()
	src.SetSourceDocumentQualifiedName("TripPin.Service")
	src.SetRemoteName("People")
	src.SetCreateChangeLocally(false)
	entity.SetSource(src)
	dm := mkDomainModelGen(mod.ID, entity)

	var updated *genDm.Entity
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelGenFunc: func(id model.ID) (*genDm.DomainModel, error) {
			return dm, nil
		},
		UpdateEntityGenFunc: func(dmID model.ID, e *genDm.Entity) error {
			updated = e
			return nil
		},
	}

	ctx, _ := newMockCtx(t, withBackend(mb))
	err := execAlterEntity(ctx, &ast.AlterEntityStmt{
		Name:      ast.QualifiedName{Module: "TripPin", Name: "People"},
		Operation: ast.AlterEntitySetAllowCreateChangeLocally,
		BoolValue: true,
	})
	assertNoError(t, err)
	if updated == nil {
		t.Fatal("expected UpdateEntityGen to be called")
	}
	updatedSrc, ok := updated.Source().(*genRest.ODataRemoteEntitySource)
	if !ok {
		t.Fatalf("expected ODataRemoteEntitySource, got %T", updated.Source())
	}
	if !updatedSrc.CreateChangeLocally() {
		t.Errorf("expected CreateChangeLocally = true, got false")
	}
}

func TestAlterEntity_AllowCreateChangeLocally_GenRemoteEntity(t *testing.T) {
	mod := mkModule("TripPin")
	entity := mkEntityGen("People")
	src := genRest.NewODataRemoteEntitySource()
	src.SetSourceDocumentQualifiedName("TripPin.Service")
	src.SetRemoteName("People")
	src.SetEntitySet("People")
	src.SetCreateChangeLocally(false)
	entity.SetSource(src)
	dm := mkDomainModelGen(mod.ID, entity)

	var updated *genDm.Entity
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelGenFunc: func(id model.ID) (*genDm.DomainModel, error) {
			return dm, nil
		},
		UpdateEntityGenFunc: func(dmID model.ID, e *genDm.Entity) error {
			updated = e
			return nil
		},
	}

	ctx, _ := newMockCtx(t, withBackend(mb))
	err := execAlterEntity(ctx, &ast.AlterEntityStmt{
		Name:      ast.QualifiedName{Module: "TripPin", Name: "People"},
		Operation: ast.AlterEntitySetAllowCreateChangeLocally,
		BoolValue: true,
	})
	assertNoError(t, err)
	if updated == nil {
		t.Fatal("expected UpdateEntityGen to be called")
	}
	gotSrc, _ := updated.Source().(*genRest.ODataRemoteEntitySource)
	if gotSrc == nil || !gotSrc.CreateChangeLocally() {
		t.Errorf("expected CreateChangeLocally = true on gen remote entity, got %#v", gotSrc)
	}
}

func TestAlterEntity_RenameAttribute_Gen(t *testing.T) {
	mod := mkModule("TripPin")
	entity := mkEntityGen("People")
	attr := genDm.NewAttribute()
	attr.SetName("DisplayName")
	entity.AddAttributes(attr)
	dm := mkDomainModelGen(mod.ID, entity)

	var updated *genDm.Entity
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelGenFunc: func(id model.ID) (*genDm.DomainModel, error) {
			return dm, nil
		},
		UpdateEntityGenFunc: func(dmID model.ID, e *genDm.Entity) error {
			updated = e
			return nil
		},
	}

	ctx, _ := newMockCtx(t, withBackend(mb))
	err := execAlterEntity(ctx, &ast.AlterEntityStmt{
		Name:          ast.QualifiedName{Module: "TripPin", Name: "People"},
		Operation:     ast.AlterEntityRenameAttribute,
		AttributeName: "DisplayName",
		NewName:       "FullName",
	})
	assertNoError(t, err)
	if updated == nil {
		t.Fatal("expected UpdateEntityGen to be called")
	}
	got := findAttributeGenByName(updated, "FullName")
	if got == nil {
		t.Fatal("expected renamed attribute to exist")
	}
}

func TestAlterEntity_ModifyAttribute_GenCalculated(t *testing.T) {
	mod := mkModule("TripPin")
	entity := mkEntityGen("People")
	attr := genDm.NewAttribute()
	attr.SetName("DisplayName")
	attr.SetType(genDm.NewStringAttributeType())
	entity.AddAttributes(attr)
	dm := mkDomainModelGen(mod.ID, entity)

	var updated *genDm.Entity
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelGenFunc: func(id model.ID) (*genDm.DomainModel, error) {
			return dm, nil
		},
		UpdateEntityGenFunc: func(dmID model.ID, e *genDm.Entity) error {
			updated = e
			return nil
		},
	}

	ctx, _ := newMockCtx(t, withBackend(mb))
	err := execAlterEntity(ctx, &ast.AlterEntityStmt{
		Name:          ast.QualifiedName{Module: "TripPin", Name: "People"},
		Operation:     ast.AlterEntityModifyAttribute,
		AttributeName: "DisplayName",
		DataType:      ast.DataType{Kind: ast.TypeString, Length: 500},
		Calculated:    true,
		CalculatedMicroflow: &ast.QualifiedName{
			Module: "TripPin",
			Name:   "ComputeDisplayName",
		},
	})
	assertNoError(t, err)
	if updated == nil {
		t.Fatal("expected UpdateEntityGen to be called")
	}
	got := findAttributeGenByName(updated, "DisplayName")
	if got == nil {
		t.Fatal("expected modified attribute to exist")
	}
	typ, _ := got.Type().(*genDm.StringAttributeType)
	if typ == nil || typ.Length() != 500 {
		t.Fatalf("expected string length 500, got %#v", got.Type())
	}
	val, _ := got.Value().(*genDm.CalculatedValue)
	if val == nil || val.MicroflowQualifiedName() != "TripPin.ComputeDisplayName" {
		t.Fatalf("expected calculated value with microflow, got %#v", got.Value())
	}
}

func TestAlterEntity_AddAttribute_Gen(t *testing.T) {
	mod := mkModule("TripPin")
	entity := mkEntityGen("People")
	dm := mkDomainModelGen(mod.ID, entity)

	var updated *genDm.Entity
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelGenFunc: func(id model.ID) (*genDm.DomainModel, error) {
			return dm, nil
		},
		UpdateEntityGenFunc: func(dmID model.ID, e *genDm.Entity) error {
			updated = e
			return nil
		},
	}

	ctx, _ := newMockCtx(t, withBackend(mb))
	err := execAlterEntity(ctx, &ast.AlterEntityStmt{
		Name:      ast.QualifiedName{Module: "TripPin", Name: "People"},
		Operation: ast.AlterEntityAddAttribute,
		Attribute: &ast.Attribute{
			Name:    "IsActive",
			Type:    ast.DataType{Kind: ast.TypeBoolean},
			NotNull: true,
		},
	})
	assertNoError(t, err)
	if updated == nil {
		t.Fatal("expected UpdateEntityGen to be called")
	}
	got := findAttributeGenByName(updated, "IsActive")
	if got == nil {
		t.Fatal("expected added attribute to exist")
	}
	val, _ := got.Value().(*genDm.StoredValue)
	if val == nil || val.DefaultValue() != "false" {
		t.Fatalf("expected boolean default false, got %#v", got.Value())
	}
	if len(updated.ValidationRulesItems()) != 1 {
		t.Fatalf("expected 1 validation rule, got %d", len(updated.ValidationRulesItems()))
	}
}

func TestAlterEntity_DropAttribute_GenCleanup(t *testing.T) {
	mod := mkModule("TripPin")
	entity := mkEntityGen("People")
	attr := genDm.NewAttribute()
	attr.SetName("DisplayName")
	entity.AddAttributes(attr)

	vr := genDm.NewValidationRule()
	vr.SetAttributeQualifiedName("TripPin.People.DisplayName")
	vr.SetRuleInfo(genDm.NewRequiredRuleInfo())
	entity.AddValidationRules(vr)

	idx := genDm.NewIndex()
	ia := genDm.NewIndexedAttribute()
	ia.SetAttributeID(attr.ID())
	idx.AddAttributes(ia)
	entity.AddIndexes(idx)

	ar := genDm.NewAccessRule()
	ma := genDm.NewMemberAccess()
	ma.SetAttributeQualifiedName("TripPin.People.DisplayName")
	ar.AddMemberAccesses(ma)
	entity.AddAccessRules(ar)

	dm := mkDomainModelGen(mod.ID, entity)
	var updated *genDm.Entity
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelGenFunc: func(id model.ID) (*genDm.DomainModel, error) {
			return dm, nil
		},
		UpdateEntityGenFunc: func(dmID model.ID, e *genDm.Entity) error {
			updated = e
			return nil
		},
	}

	ctx, _ := newMockCtx(t, withBackend(mb))
	err := execAlterEntity(ctx, &ast.AlterEntityStmt{
		Name:          ast.QualifiedName{Module: "TripPin", Name: "People"},
		Operation:     ast.AlterEntityDropAttribute,
		AttributeName: "DisplayName",
	})
	assertNoError(t, err)
	if updated == nil {
		t.Fatal("expected UpdateEntityGen to be called")
	}
	if findAttributeGenByName(updated, "DisplayName") != nil {
		t.Fatal("expected attribute to be removed")
	}
	if len(updated.ValidationRulesItems()) != 0 {
		t.Fatalf("expected validation rules to be cleaned up, got %d", len(updated.ValidationRulesItems()))
	}
	if len(updated.IndexesItems()) != 0 {
		t.Fatalf("expected indexes to be cleaned up, got %d", len(updated.IndexesItems()))
	}
	rule, _ := updated.AccessRulesItems()[0].(*genDm.AccessRule)
	if rule == nil || len(rule.MemberAccessesItems()) != 0 {
		t.Fatalf("expected member access cleanup, got %#v", updated.AccessRulesItems())
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

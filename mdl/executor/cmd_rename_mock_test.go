// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	repostesting "github.com/mendixlabs/mxcli/mdl/repos/testing"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
	"github.com/mendixlabs/mxcli/sdk/pages"
)

// ---------------------------------------------------------------------------
// Not connected
// ---------------------------------------------------------------------------

func TestRename_NotConnected(t *testing.T) {
	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return false }}
	ctx, _ := newMockCtx(t, withBackend(mb))
	err := execRename(ctx, &ast.RenameStmt{
		ObjectType: "entity",
		Name:       ast.QualifiedName{Module: "MyModule", Name: "OldName"},
		NewName:    "NewName",
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "not connected")
}

// ---------------------------------------------------------------------------
// Unsupported type
// ---------------------------------------------------------------------------

func TestRename_UnsupportedType(t *testing.T) {
	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}
	ctx, _ := newMockCtx(t, withBackend(mb))
	err := execRename(ctx, &ast.RenameStmt{
		ObjectType: "snippet",
		Name:       ast.QualifiedName{Module: "M", Name: "N"},
		NewName:    "X",
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "not supported")
}

// ---------------------------------------------------------------------------
// Rename entity — happy path
// ---------------------------------------------------------------------------

func TestRename_Entity_Success(t *testing.T) {
	mod := mkModule("MyModule")
	ent := mkEntity(mod.ID, "OldEntity")
	dm := mkDomainModel(mod.ID, ent)
	dmUpdated := false
	mb := &mock.MockBackend{
		IsConnectedFunc:    func() bool { return true },
		ListModulesFunc:    func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelFunc: func(id model.ID) (*domainmodel.DomainModel, error) { return dm, nil },
		RenameReferencesFunc: func(old, new string, dryRun bool) ([]types.RenameHit, error) {
			return []types.RenameHit{{UnitID: "u1", Name: "SomeDoc", Count: 2}}, nil
		},
		UpdateDomainModelFunc: func(d *domainmodel.DomainModel) error { dmUpdated = true; return nil },
	}
	h := mkHierarchy(mod)
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, execRename(ctx, &ast.RenameStmt{
		ObjectType: "entity",
		Name:       ast.QualifiedName{Module: "MyModule", Name: "OldEntity"},
		NewName:    "NewEntity",
	}))
	if !dmUpdated {
		t.Error("Expected UpdateDomainModel to be called")
	}
	assertContainsStr(t, buf.String(), "Renamed entity")
	assertContainsStr(t, buf.String(), "MyModule.OldEntity")
	assertContainsStr(t, buf.String(), "MyModule.NewEntity")
	assertContainsStr(t, buf.String(), "Updated 2 reference(s)")
}

// ---------------------------------------------------------------------------
// Rename entity — not found
// ---------------------------------------------------------------------------

func TestRename_Entity_NotFound(t *testing.T) {
	mod := mkModule("MyModule")
	dm := mkDomainModel(mod.ID) // no entities
	mb := &mock.MockBackend{
		IsConnectedFunc:    func() bool { return true },
		ListModulesFunc:    func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelFunc: func(id model.ID) (*domainmodel.DomainModel, error) { return dm, nil },
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execRename(ctx, &ast.RenameStmt{
		ObjectType: "entity",
		Name:       ast.QualifiedName{Module: "MyModule", Name: "Missing"},
		NewName:    "New",
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// Rename entity — dry run
// ---------------------------------------------------------------------------

func TestRename_Entity_DryRun(t *testing.T) {
	mod := mkModule("MyModule")
	ent := mkEntity(mod.ID, "OldEntity")
	dm := mkDomainModel(mod.ID, ent)
	mb := &mock.MockBackend{
		IsConnectedFunc:    func() bool { return true },
		ListModulesFunc:    func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelFunc: func(id model.ID) (*domainmodel.DomainModel, error) { return dm, nil },
		RenameReferencesFunc: func(old, new string, dryRun bool) ([]types.RenameHit, error) {
			if !dryRun {
				t.Error("Expected dryRun=true")
			}
			return []types.RenameHit{{UnitID: "u1", Name: "Page1", UnitType: "Pages$Page", Count: 1}}, nil
		},
	}
	h := mkHierarchy(mod)
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, execRename(ctx, &ast.RenameStmt{
		ObjectType: "entity",
		Name:       ast.QualifiedName{Module: "MyModule", Name: "OldEntity"},
		NewName:    "NewEntity",
		DryRun:     true,
	}))
	assertContainsStr(t, buf.String(), "Would rename")
	assertContainsStr(t, buf.String(), "Page1")
}

// ---------------------------------------------------------------------------
// Rename microflow (document type) — happy path
// ---------------------------------------------------------------------------

func TestRename_Microflow_Success(t *testing.T) {
	mod := mkModule("MyModule")
	mf := mkMicroflowGen("OldMF")
	mfID := model.ID(mf.ID())
	renameCalled := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
		RenameReferencesFunc: func(old, new string, dryRun bool) ([]types.RenameHit, error) {
			return nil, nil
		},
		RenameDocumentByNameFunc: func(mod, old, new string) error {
			renameCalled = true
			return nil
		},
	}
	mfRepo := &repostesting.RecordingMicroflowRepository{
		ListAllFunc: func() ([]*genMf.Microflow, error) {
			return []*genMf.Microflow{mf}, nil
		},
		GetContainerUUIDFunc: func(id model.ID) (model.ID, error) {
			if id == mfID {
				return mod.ID, nil
			}
			return "", nil
		},
	}
	h := mkHierarchy(mod)
	ctx, buf := newMockCtx(t,
		withBackend(mb),
		withHierarchy(h),
		withMicroflowsRepo(mfRepo),
	)
	assertNoError(t, execRename(ctx, &ast.RenameStmt{
		ObjectType: "microflow",
		Name:       ast.QualifiedName{Module: "MyModule", Name: "OldMF"},
		NewName:    "NewMF",
	}))
	if !renameCalled {
		t.Error("Expected RenameDocumentByName to be called")
	}
	assertContainsStr(t, buf.String(), "Renamed microflow")
}

// ---------------------------------------------------------------------------
// Rename page — not found
// ---------------------------------------------------------------------------

func TestRename_Page_NotFound(t *testing.T) {
	mod := mkModule("MyModule")
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
		ListPagesFunc:   func() ([]*pages.Page, error) { return nil, nil },
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execRename(ctx, &ast.RenameStmt{
		ObjectType: "page",
		Name:       ast.QualifiedName{Module: "MyModule", Name: "Missing"},
		NewName:    "New",
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// Rename module — happy path
// ---------------------------------------------------------------------------

func TestRename_Module_Success(t *testing.T) {
	mod := mkModule("OldModule")
	moduleUpdated := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		RenameReferencesFunc: func(old, new string, dryRun bool) ([]types.RenameHit, error) {
			return nil, nil
		},
		UpdateModuleFunc: func(m *model.Module) error {
			moduleUpdated = true
			if m.Name != "NewModule" {
				t.Errorf("Expected new name NewModule, got %s", m.Name)
			}
			return nil
		},
	}
	h := mkHierarchy(mod)
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, execRename(ctx, &ast.RenameStmt{
		ObjectType: "module",
		Name:       ast.QualifiedName{Module: "OldModule"},
		NewName:    "NewModule",
	}))
	if !moduleUpdated {
		t.Error("Expected UpdateModule to be called")
	}
	assertContainsStr(t, buf.String(), "Renamed module")
}

// ---------------------------------------------------------------------------
// Rename association — happy path
// ---------------------------------------------------------------------------

func TestRename_Association_Success(t *testing.T) {
	mod := mkModule("MyModule")
	ent1 := mkEntity(mod.ID, "Parent")
	ent2 := mkEntity(mod.ID, "Child")
	assoc := mkAssociation(mod.ID, "OldAssoc", ent1.ID, ent2.ID)
	dm := mkDomainModel(mod.ID, ent1, ent2)
	dm.Associations = []*domainmodel.Association{assoc}
	dmUpdated := false
	mb := &mock.MockBackend{
		IsConnectedFunc:    func() bool { return true },
		ListModulesFunc:    func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelFunc: func(id model.ID) (*domainmodel.DomainModel, error) { return dm, nil },
		RenameReferencesFunc: func(old, new string, dryRun bool) ([]types.RenameHit, error) {
			return nil, nil
		},
		UpdateDomainModelFunc: func(d *domainmodel.DomainModel) error { dmUpdated = true; return nil },
	}
	h := mkHierarchy(mod)
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, execRename(ctx, &ast.RenameStmt{
		ObjectType: "association",
		Name:       ast.QualifiedName{Module: "MyModule", Name: "OldAssoc"},
		NewName:    "NewAssoc",
	}))
	if !dmUpdated {
		t.Error("Expected UpdateDomainModel to be called")
	}
	assertContainsStr(t, buf.String(), "Renamed association")
}

// ---------------------------------------------------------------------------
// Rename association — not found
// ---------------------------------------------------------------------------

func TestRename_Association_NotFound(t *testing.T) {
	mod := mkModule("MyModule")
	dm := mkDomainModel(mod.ID) // no associations
	mb := &mock.MockBackend{
		IsConnectedFunc:    func() bool { return true },
		ListModulesFunc:    func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelFunc: func(id model.ID) (*domainmodel.DomainModel, error) { return dm, nil },
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execRename(ctx, &ast.RenameStmt{
		ObjectType: "association",
		Name:       ast.QualifiedName{Module: "MyModule", Name: "Missing"},
		NewName:    "New",
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// Rename backend error
// ---------------------------------------------------------------------------

func TestRename_Entity_BackendError(t *testing.T) {
	mod := mkModule("MyModule")
	ent := mkEntity(mod.ID, "Ent")
	dm := mkDomainModel(mod.ID, ent)
	mb := &mock.MockBackend{
		IsConnectedFunc:    func() bool { return true },
		ListModulesFunc:    func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelFunc: func(id model.ID) (*domainmodel.DomainModel, error) { return dm, nil },
		RenameReferencesFunc: func(old, new string, dryRun bool) ([]types.RenameHit, error) {
			return nil, fmt.Errorf("scan error")
		},
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execRename(ctx, &ast.RenameStmt{
		ObjectType: "entity",
		Name:       ast.QualifiedName{Module: "MyModule", Name: "Ent"},
		NewName:    "New",
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "scan references")
}

// ---------------------------------------------------------------------------
// Rename java action — happy path
// ---------------------------------------------------------------------------

func TestRename_JavaAction_Success(t *testing.T) {
	mod := mkModule("MyModule")
	jaGen := genJA.NewJavaAction()
	jaGen.SetName("OldHelper")
	documentRenamed := false
	sourceRenamed := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
		ListJavaActionsGenFunc: func() ([]*genJA.JavaAction, error) {
			return []*genJA.JavaAction{jaGen}, nil
		},
		RenameReferencesFunc: func(old, new string, dryRun bool) ([]types.RenameHit, error) {
			return []types.RenameHit{{UnitID: "u1", Name: "SomeMF", Count: 1}}, nil
		},
		RenameDocumentByNameFunc: func(module, old, newName string) error {
			documentRenamed = true
			return nil
		},
		RenameJavaSourceFileFunc: func(module, old, newName string) error {
			sourceRenamed = true
			return nil
		},
	}
	h := mkHierarchy(mod)
	withContainer(h, model.ID(jaGen.ID()), mod.ID)
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, execRename(ctx, &ast.RenameStmt{
		ObjectType: "javaaction",
		Name:       ast.QualifiedName{Module: "MyModule", Name: "OldHelper"},
		NewName:    "NewHelper",
	}))
	if !documentRenamed {
		t.Error("Expected RenameDocumentByName to be called")
	}
	if !sourceRenamed {
		t.Error("Expected RenameJavaSourceFile to be called")
	}
	assertContainsStr(t, buf.String(), "Renamed java action")
	assertContainsStr(t, buf.String(), "MyModule.OldHelper")
	assertContainsStr(t, buf.String(), "MyModule.NewHelper")
}

// ---------------------------------------------------------------------------
// Rename java action — not found
// ---------------------------------------------------------------------------

func TestRename_JavaAction_NotFound(t *testing.T) {
	mod := mkModule("MyModule")
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
		ListJavaActionsGenFunc: func() ([]*genJA.JavaAction, error) {
			return nil, nil
		},
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execRename(ctx, &ast.RenameStmt{
		ObjectType: "javaaction",
		Name:       ast.QualifiedName{Module: "MyModule", Name: "Missing"},
		NewName:    "New",
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// Rename workflow — happy path
// ---------------------------------------------------------------------------

func TestRename_Workflow_Success(t *testing.T) {
	mod := mkModule("BPModule")
	wfGen := mkWorkflowGen(string(nextID("wf")), "OldProcess")
	renameCalled := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
		RenameReferencesFunc: func(old, new string, dryRun bool) ([]types.RenameHit, error) {
			return nil, nil
		},
		RenameDocumentByNameFunc: func(module, old, newName string) error {
			renameCalled = true
			return nil
		},
	}
	h := mkHierarchy(mod)
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Workflows = makeWorkflowsRepo([]*genWf.Workflow{wfGen}, mod.ID)
	assertNoError(t, execRename(ctx, &ast.RenameStmt{
		ObjectType: "workflow",
		Name:       ast.QualifiedName{Module: "BPModule", Name: "OldProcess"},
		NewName:    "NewProcess",
	}))
	if !renameCalled {
		t.Error("Expected RenameDocumentByName to be called")
	}
	assertContainsStr(t, buf.String(), "Renamed workflow")
	assertContainsStr(t, buf.String(), "BPModule.OldProcess")
	assertContainsStr(t, buf.String(), "BPModule.NewProcess")
}

// ---------------------------------------------------------------------------
// Rename workflow — not found
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Collision detection — renaming to an existing name must fail
// ---------------------------------------------------------------------------

func TestRename_Nanoflow_CollisionError(t *testing.T) {
	mod := mkModule("MyModule")
	nf1 := mkNanoflowGen("NF1")
	nf2 := mkNanoflowGen("NF2")
	nf1ID := model.ID(nf1.ID())
	nf2ID := model.ID(nf2.ID())
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
	}
	// genFlowContainerModule resolves nanoflow container via
	// ctx.Microflows.GetContainerUUID (nanoflows live alongside
	// microflows in the unit table); seed both repos.
	containerLookup := func(id model.ID) (model.ID, error) {
		if id == nf1ID || id == nf2ID {
			return mod.ID, nil
		}
		return "", nil
	}
	mfRepo := &repostesting.RecordingMicroflowRepository{
		GetContainerUUIDFunc: containerLookup,
	}
	nfRepo := &repostesting.RecordingNanoflowRepository{
		ListFunc: func(model.ID) ([]*genMf.Nanoflow, error) {
			return []*genMf.Nanoflow{nf1, nf2}, nil
		},
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t,
		withBackend(mb),
		withHierarchy(h),
		withMicroflowsRepo(mfRepo),
		withNanoflowsRepo(nfRepo),
	)
	err := execRename(ctx, &ast.RenameStmt{
		ObjectType: "nanoflow",
		Name:       ast.QualifiedName{Module: "MyModule", Name: "NF1"},
		NewName:    "NF2",
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "already exists")
}

func TestRename_Microflow_CollisionError(t *testing.T) {
	mod := mkModule("MyModule")
	mf1 := mkMicroflowGen("MF1")
	mf2 := mkMicroflowGen("MF2")
	mf1ID := model.ID(mf1.ID())
	mf2ID := model.ID(mf2.ID())
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
	}
	mfRepo := &repostesting.RecordingMicroflowRepository{
		ListAllFunc: func() ([]*genMf.Microflow, error) {
			return []*genMf.Microflow{mf1, mf2}, nil
		},
		GetContainerUUIDFunc: func(id model.ID) (model.ID, error) {
			if id == mf1ID || id == mf2ID {
				return mod.ID, nil
			}
			return "", nil
		},
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t,
		withBackend(mb),
		withHierarchy(h),
		withMicroflowsRepo(mfRepo),
	)
	err := execRename(ctx, &ast.RenameStmt{
		ObjectType: "microflow",
		Name:       ast.QualifiedName{Module: "MyModule", Name: "MF1"},
		NewName:    "MF2",
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "already exists")
}

func TestRename_Entity_CollisionError(t *testing.T) {
	mod := mkModule("MyModule")
	ent1 := mkEntity(mod.ID, "EntityA")
	ent2 := mkEntity(mod.ID, "EntityB")
	dm := mkDomainModel(mod.ID, ent1, ent2)
	mb := &mock.MockBackend{
		IsConnectedFunc:    func() bool { return true },
		ListModulesFunc:    func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelFunc: func(id model.ID) (*domainmodel.DomainModel, error) { return dm, nil },
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execRename(ctx, &ast.RenameStmt{
		ObjectType: "entity",
		Name:       ast.QualifiedName{Module: "MyModule", Name: "EntityA"},
		NewName:    "EntityB",
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "already exists")
}

func TestRename_Workflow_NotFound(t *testing.T) {
	mod := mkModule("BPModule")
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Workflows = makeWorkflowsRepo(nil, "")
	err := execRename(ctx, &ast.RenameStmt{
		ObjectType: "workflow",
		Name:       ast.QualifiedName{Module: "BPModule", Name: "Missing"},
		NewName:    "New",
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "not found")
}

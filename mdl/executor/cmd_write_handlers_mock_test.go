// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	repostesting "github.com/mendixlabs/mxcli/mdl/repos/testing"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
	"github.com/mendixlabs/mxcli/sdk/pages"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

func TestExecCreateModule_Mock(t *testing.T) {
	called := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) {
			return nil, nil // no existing modules
		},
		CreateModuleFunc: func(m *model.Module) error {
			called = true
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb))
	err := execCreateModule(ctx, &ast.CreateModuleStmt{Name: "NewModule"})
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Created module: NewModule")
	if !called {
		t.Fatal("CreateModuleFunc was not called")
	}
}

func TestExecDropEnumeration_Mock(t *testing.T) {
	mod := mkModule("MyModule")
	enum := mkEnumeration(mod.ID, "Status", "Active", "Inactive")

	called := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListEnumerationsFunc: func() ([]*model.Enumeration, error) {
			return []*model.Enumeration{enum}, nil
		},
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{mod}, nil
		},
		DeleteEnumerationFunc: func(id model.ID) error {
			called = true
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb))
	err := execDropEnumeration(ctx, &ast.DropEnumerationStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "Status"},
	})
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Dropped enumeration:")
	if !called {
		t.Fatal("DeleteEnumerationFunc was not called")
	}
}

func TestExecCreateEnumeration_Mock(t *testing.T) {
	mod := mkModule("MyModule")

	h := mkHierarchy(mod)

	called := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{mod}, nil
		},
		ListEnumerationsFunc: func() ([]*model.Enumeration, error) {
			return nil, nil // no duplicates
		},
		CreateEnumerationFunc: func(e *model.Enumeration) error {
			called = true
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execCreateEnumeration(ctx, &ast.CreateEnumerationStmt{
		Name:   ast.QualifiedName{Module: "MyModule", Name: "Color"},
		Values: []ast.EnumValue{{Name: "Red", Caption: "Red"}, {Name: "Blue", Caption: "Blue"}},
	})
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Created enumeration:")
	if !called {
		t.Fatal("CreateEnumerationFunc was not called")
	}
}

func TestExecDropEntity_Mock(t *testing.T) {
	mod := mkModule("MyModule")
	ent := mkEntity(mod.ID, "Customer")
	dm := mkDomainModel(mod.ID, ent)

	called := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{mod}, nil
		},
		GetDomainModelFunc: func(moduleID model.ID) (*domainmodel.DomainModel, error) {
			return dm, nil
		},
		DeleteEntityFunc: func(domainModelID model.ID, entityID model.ID) error {
			called = true
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb))
	err := execDropEntity(ctx, &ast.DropEntityStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "Customer"},
	})
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Dropped entity:")
	if !called {
		t.Fatal("DeleteEntityFunc was not called")
	}
}

// Stage 3.2.6.5 → Followup D rewrite: cmd_microflows_drop.go now reads
// from ctx.Microflows (modelsdk repo) and deletes via
// deleteMicroflowViaRepoOrBackend, which itself routes to
// ctx.Microflows.Delete when the repo is wired. Test injects a
// RecordingMicroflowRepository to seed the existing microflow + assert
// the Delete call.
func TestExecDropMicroflow_Mock(t *testing.T) {
	mod := mkModule("MyModule")
	mf := mkMicroflowGen("DoSomething")

	h := mkHierarchy(mod)
	// Hierarchy must contain the (containerID → moduleID) link the
	// drop handler walks: `h.GetModuleName(h.FindModuleID(containerID))`.
	// We resolve the container via GetContainerUUIDFunc below, returning
	// mod.ID directly so FindModuleID falls through to the moduleIDs map.

	mfRepo := &repostesting.RecordingMicroflowRepository{
		ListAllFunc: func() ([]*genMf.Microflow, error) {
			return []*genMf.Microflow{mf}, nil
		},
		GetContainerUUIDFunc: func(id model.ID) (model.ID, error) {
			if id == model.ID(mf.ID()) {
				return mod.ID, nil
			}
			return "", nil
		},
	}

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}

	ctx, buf := newMockCtx(t,
		withBackend(mb),
		withHierarchy(h),
		withMicroflowsRepo(mfRepo),
	)
	err := execDropMicroflow(ctx, &ast.DropMicroflowStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "DoSomething"},
	})
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Dropped microflow:")
	if len(mfRepo.Deleted) != 1 || mfRepo.Deleted[0] != model.ID(mf.ID()) {
		t.Fatalf("Repo.Delete must have been called once with the microflow ID; got %v", mfRepo.Deleted)
	}
}

func TestExecDropPage_Mock(t *testing.T) {
	mod := mkModule("MyModule")
	pg := mkPage(mod.ID, "HomePage")

	h := mkHierarchy(mod)
	withContainer(h, pg.ContainerID, mod.ID)

	called := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListPagesFunc: func() ([]*pages.Page, error) {
			return []*pages.Page{pg}, nil
		},
		DeletePageFunc: func(id model.ID) error {
			called = true
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execDropPage(ctx, &ast.DropPageStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "HomePage"},
	})
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Dropped page")
	if !called {
		t.Fatal("DeletePageFunc was not called")
	}
}

func TestExecDropSnippet_Mock(t *testing.T) {
	mod := mkModule("MyModule")
	snp := mkSnippet(mod.ID, "HeaderSnippet")

	h := mkHierarchy(mod)
	withContainer(h, snp.ContainerID, mod.ID)

	called := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListSnippetsFunc: func() ([]*pages.Snippet, error) {
			return []*pages.Snippet{snp}, nil
		},
		DeleteSnippetFunc: func(id model.ID) error {
			called = true
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execDropSnippet(ctx, &ast.DropSnippetStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "HeaderSnippet"},
	})
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Dropped snippet")
	if !called {
		t.Fatal("DeleteSnippetFunc was not called")
	}
}

func TestExecDropAssociation_Mock(t *testing.T) {
	mod := mkModule("MyModule")
	ent1 := mkEntity(mod.ID, "Order")
	ent2 := mkEntity(mod.ID, "Customer")
	assoc := mkAssociation(mod.ID, "Order_Customer", ent1.ID, ent2.ID)

	dm := mkDomainModel(mod.ID, ent1, ent2)
	dm.Associations = []*domainmodel.Association{assoc}

	called := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{mod}, nil
		},
		GetDomainModelFunc: func(moduleID model.ID) (*domainmodel.DomainModel, error) {
			return dm, nil
		},
		DeleteAssociationFunc: func(domainModelID model.ID, assocID model.ID) error {
			called = true
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb))
	err := execDropAssociation(ctx, &ast.DropAssociationStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "Order_Customer"},
	})
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Dropped association:")
	if !called {
		t.Fatal("DeleteAssociationFunc was not called")
	}
}

func TestExecDropJavaAction_Mock(t *testing.T) {
	mod := mkModule("MyModule")

	h := mkHierarchy(mod)

	// Stage 3.3.2.E1: execDropJavaActionGen reads via the gen path
	// (ctx.JavaActions or ctx.Backend.ListJavaActionsGen). Legacy
	// types.JavaAction fixture removed — only gen mock now.
	jaGen := genJA.NewJavaAction()
	jaGen.SetName("MyAction")

	called := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListJavaActionsGenFunc: func() ([]*genJA.JavaAction, error) {
			return []*genJA.JavaAction{jaGen}, nil
		},
		DeleteJavaActionFunc: func(id model.ID) error {
			called = true
			return nil
		},
		// MockBackend.DeleteJavaSourceFile no longer returns nil by default
		// (Stage 3.3.2.C5 mock audit) — must opt in explicitly.
		DeleteJavaSourceFileFunc: func(moduleName, actionName string) error { return nil },
	}
	withContainer(h, model.ID(jaGen.ID()), mod.ID)

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execDropJavaActionGen(ctx, &ast.DropJavaActionStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "MyAction"},
	})
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Dropped java action:")
	if !called {
		t.Fatal("DeleteJavaActionFunc was not called")
	}
}

func TestExecDropFolder_Mock(t *testing.T) {
	mod := mkModule("MyModule")
	folderID := nextID("folder")
	folder := &types.FolderInfo{
		ID:          folderID,
		ContainerID: mod.ID,
		Name:        "Resources",
	}

	h := mkHierarchy(mod)
	withContainer(h, folderID, mod.ID)

	called := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{mod}, nil
		},
		ListFoldersFunc: func() ([]*types.FolderInfo, error) {
			return []*types.FolderInfo{folder}, nil
		},
		DeleteFolderFunc: func(id model.ID) error {
			called = true
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execDropFolder(ctx, &ast.DropFolderStmt{
		FolderPath: "Resources",
		Module:     "MyModule",
	})
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Dropped folder:")
	if !called {
		t.Fatal("DeleteFolderFunc was not called")
	}
}

func TestExecCreateModule_Mock_AlreadyExists(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{{Name: "Existing"}}, nil
		},
	}
	ctx, buf := newMockCtx(t, withBackend(mb))
	assertNoError(t, execCreateModule(ctx, &ast.CreateModuleStmt{Name: "Existing"}))
	assertContainsStr(t, buf.String(), "already exists")
}

func TestExecDropEnumeration_Mock_NotFound(t *testing.T) {
	mod := mkModule("MyModule")
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListEnumerationsFunc: func() ([]*model.Enumeration, error) {
			return nil, nil
		},
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{mod}, nil
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, execDropEnumeration(ctx, &ast.DropEnumerationStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "NonExistent"},
	}))
}

func TestExecDropEntity_Mock_NotFound(t *testing.T) {
	mod := mkModule("MyModule")
	dm := mkDomainModel(mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{mod}, nil
		},
		GetDomainModelFunc: func(moduleID model.ID) (*domainmodel.DomainModel, error) {
			return dm, nil
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, execDropEntity(ctx, &ast.DropEntityStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "NonExistent"},
	}))
}

func TestExecDropMicroflow_Mock_NotFound(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	// Stage 3.2.6.5: execDropMicroflow now reads from ctx.Microflows
	// (modelsdk repo); no repo seeded → empty list → NotFound, which
	// is the expected error path for this test.
	assertError(t, execDropMicroflow(ctx, &ast.DropMicroflowStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "NonExistent"},
	}))
}

func TestExecDropPage_Mock_NotFound(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListPagesFunc:   func() ([]*pages.Page, error) { return nil, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertError(t, execDropPage(ctx, &ast.DropPageStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "NonExistent"},
	}))
}

func TestExecDropSnippet_Mock_NotFound(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc:  func() bool { return true },
		ListSnippetsFunc: func() ([]*pages.Snippet, error) { return nil, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertError(t, execDropSnippet(ctx, &ast.DropSnippetStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "NonExistent"},
	}))
}

func TestExecDropAssociation_Mock_NotFound(t *testing.T) {
	mod := mkModule("MyModule")
	dm := mkDomainModel(mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{mod}, nil
		},
		GetDomainModelFunc: func(moduleID model.ID) (*domainmodel.DomainModel, error) {
			return dm, nil
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, execDropAssociation(ctx, &ast.DropAssociationStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "NonExistent"},
	}))
}

// TestDropThenCreatePreservesMicroflowUnitID is a regression test for the
// MPR corruption bug documented in docs/MXCLI_MPR_CORRUPTION_PROMPT_0015.md.
//
// When a script runs `DROP MICROFLOW X; CREATE OR MODIFY MICROFLOW X ...` in
// the same session, the executor used to delete the Unit row and then insert
// a new one with a freshly generated UUID. Studio Pro treats the rewritten
// ContainerID/UnitID pair as an unrelated document and refuses to open the
// resulting .mpr ("file does not look like a Mendix Studio Pro project").
//
// The fix records the UnitID of dropped microflows on the executor cache and
// reuses it when a subsequent CREATE OR REPLACE/MODIFY targets the same
// qualified name, so the delete+insert behaves like an in-place update.
// Stage 3.2.6.3a → Followup D rewrite: DROP MICROFLOW followed by
// CREATE OR MODIFY MICROFLOW for the same qualified name must reuse the
// dropped UnitID, otherwise Studio Pro reports MPR corruption (see
// docs/MXCLI_MPR_CORRUPTION_PROMPT_0015.md). The new implementation
// keeps the dropped UnitID in ctx.Cache.droppedMicroflows and feeds it
// back via consumeDroppedMicroflow inside execCreateMicroflowGen.
func TestDropThenCreatePreservesMicroflowUnitID(t *testing.T) {
	mod := mkModule("MyModule")
	mf := mkMicroflowGen("DoSomething")
	originalID := model.ID(mf.ID())
	mf.SetAllowedModuleRolesQualifiedNames([]string{"MyModule.Admin", "MyModule.User"})

	h := mkHierarchy(mod)

	listedMicroflows := []*genMf.Microflow{mf}

	mfRepo := &repostesting.RecordingMicroflowRepository{
		ListAllFunc: func() ([]*genMf.Microflow, error) {
			return listedMicroflows, nil
		},
		GetContainerUUIDFunc: func(id model.ID) (model.ID, error) {
			if id == originalID {
				return mod.ID, nil
			}
			return "", nil
		},
		DeleteFunc: func(id model.ID) error {
			// Simulate real deletion: hide the microflow from subsequent
			// ListAll calls so CREATE OR MODIFY sees no existing unit
			// (matching the bug reproduction exactly).
			listedMicroflows = nil
			return nil
		},
	}

	ms := genSec.NewModuleSecurity()

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{mod}, nil
		},
		GetModuleSecurityGenFunc: func(moduleID model.ID) (*genSec.ModuleSecurity, error) {
			return ms, nil
		},
		AddModuleRoleFunc: func(moduleSecurityID model.ID, name, description string) error {
			return nil
		},
		ListDomainModelsFunc: func() ([]*domainmodel.DomainModel, error) {
			return nil, nil
		},
		ListConsumedRestServicesFunc: func() ([]*model.ConsumedRestService, error) {
			return nil, nil
		},
	}

	ctx, _ := newMockCtx(t,
		withBackend(mb),
		withHierarchy(h),
		withMicroflowsRepo(mfRepo),
	)

	if err := execDropMicroflow(ctx, &ast.DropMicroflowStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "DoSomething"},
	}); err != nil {
		t.Fatalf("DROP MICROFLOW failed: %v", err)
	}

	// The UnitID and ContainerID must have been stashed on the cache before deletion.
	if ctx.Cache == nil || ctx.Cache.droppedMicroflows == nil ||
		ctx.Cache.droppedMicroflows["MyModule.DoSomething"] == nil ||
		ctx.Cache.droppedMicroflows["MyModule.DoSomething"].ID != originalID {
		t.Fatalf("expected droppedMicroflows[MyModule.DoSomething].ID = %q, got cache=%+v",
			originalID, ctx.Cache)
	}

	// CREATE OR MODIFY with the same qualified name must reuse the dropped ID.
	createStmt := &ast.CreateMicroflowStmt{
		Name:           ast.QualifiedName{Module: "MyModule", Name: "DoSomething"},
		CreateOrModify: true,
		Body:           nil, // empty body is fine for this test
	}
	if err := execCreateMicroflowGen(ctx, createStmt); err != nil {
		t.Fatalf("CREATE OR MODIFY MICROFLOW failed: %v", err)
	}

	if len(mfRepo.Created) != 1 {
		t.Fatalf("CREATE OR MODIFY must call repo.Create exactly once; got %d", len(mfRepo.Created))
	}
	createdID := element.ID(originalID)
	if mfRepo.Created[0].Microflow.ID() != createdID {
		t.Fatalf("CREATE OR MODIFY must reuse dropped UnitID: got %q, want %q",
			mfRepo.Created[0].Microflow.ID(), createdID)
	}

	// The cache entry must be consumed so repeated CREATEs don't collide.
	if ctx.Cache != nil && ctx.Cache.droppedMicroflows != nil {
		if _, stillThere := ctx.Cache.droppedMicroflows["MyModule.DoSomething"]; stillThere {
			t.Errorf("droppedMicroflows entry should be cleared after reuse")
		}
	}
}

// Stage 3.2.6.3a → Followup D rewrite: CREATE OR MODIFY MICROFLOW for an
// existing microflow must preserve the existing AllowedModuleRoles
// rather than wipe them to the module defaults.
func TestCreateOrModifyMicroflowPreservesAllowedRoles(t *testing.T) {
	mod := mkModule("MyModule")
	mf := mkMicroflowGen("DoSomething")
	mf.SetAllowedModuleRolesQualifiedNames([]string{"MyModule.Admin"})
	originalID := model.ID(mf.ID())

	h := mkHierarchy(mod)

	mfRepo := &repostesting.RecordingMicroflowRepository{
		ListAllFunc: func() ([]*genMf.Microflow, error) {
			return []*genMf.Microflow{mf}, nil
		},
		GetContainerUUIDFunc: func(id model.ID) (model.ID, error) {
			if id == originalID {
				return mod.ID, nil
			}
			return "", nil
		},
	}

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{mod}, nil
		},
		ListDomainModelsFunc: func() ([]*domainmodel.DomainModel, error) {
			return nil, nil
		},
		ListConsumedRestServicesFunc: func() ([]*model.ConsumedRestService, error) {
			return nil, nil
		},
	}

	ctx, _ := newMockCtx(t,
		withBackend(mb),
		withHierarchy(h),
		withMicroflowsRepo(mfRepo),
	)

	if err := execCreateMicroflowGen(ctx, &ast.CreateMicroflowStmt{
		Name:           ast.QualifiedName{Module: "MyModule", Name: "DoSomething"},
		CreateOrModify: true,
	}); err != nil {
		t.Fatalf("CREATE OR MODIFY MICROFLOW failed: %v", err)
	}

	if len(mfRepo.Updated) != 1 {
		t.Fatalf("CREATE OR MODIFY must call repo.Update exactly once; got %d", len(mfRepo.Updated))
	}
	got := mfRepo.Updated[0].AllowedModuleRolesQualifiedNames()
	if len(got) != 1 || got[0] != "MyModule.Admin" {
		t.Fatalf("expected existing allowed roles to be preserved, got %v", got)
	}
}


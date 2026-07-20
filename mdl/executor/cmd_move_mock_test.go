// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// ---------------------------------------------------------------------------
// execMove — not connected
// ---------------------------------------------------------------------------

func TestMove_NotConnected(t *testing.T) {
	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return false }}
	ctx, _ := newMockCtx(t, withBackend(mb))
	err := execMoveFn(ctx, &ast.MoveStmt{
		DocumentType: ast.DocumentTypePage,
		Name:         ast.QualifiedName{Module: "MyModule", Name: "MyPage"},
	}, ctx.Deps)
	assertError(t, err)
	assertContainsStr(t, err.Error(), "not connected")
}

// ---------------------------------------------------------------------------
// execMove — page happy path
// ---------------------------------------------------------------------------

func TestMove_Page_ToFolder(t *testing.T) {
	mod := mkModule("MyModule")
	pg := mkPageGen(string(nextID("pg")), "MyPage")
	folderID := nextID("folder")
	folders := []*types.FolderInfo{
		{ID: folderID, ContainerID: mod.ID, Name: "Admin"},
	}
	var movedID model.ID
	var movedContainerID model.ID
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return folders, nil },
		MoveDocumentGenFunc: func(id, containerID model.ID) error {
			movedID = id
			movedContainerID = containerID
			return nil
		},
	}
	h := mkHierarchy(mod)
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Pages = makePagesRepo([]*genPg.Page{pg}, mod.ID)
	rebuildDeps(ctx)
	assertNoError(t, execMoveFn(ctx, &ast.MoveStmt{
		DocumentType: ast.DocumentTypePage,
		Name:         ast.QualifiedName{Module: "MyModule", Name: "MyPage"},
		Folder:       "Admin",
	}, ctx.Deps))
	if movedID == "" {
		t.Fatal("Expected MovePageGen to be called")
	}
	if movedContainerID != folderID {
		t.Errorf("Expected container %s, got %s", folderID, movedContainerID)
	}
	assertContainsStr(t, buf.String(), "Moved page")
}

// ---------------------------------------------------------------------------
// execMove — page not found
// ---------------------------------------------------------------------------

func TestMove_Page_NotFound(t *testing.T) {
	mod := mkModule("MyModule")
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Pages = makePagesRepo(nil, mod.ID)
	rebuildDeps(ctx)
	err := execMoveFn(ctx, &ast.MoveStmt{
		DocumentType: ast.DocumentTypePage,
		Name:         ast.QualifiedName{Module: "MyModule", Name: "NonExistent"},
		Folder:       "SomeFolder",
	}, ctx.Deps)
	assertError(t, err)
	assertContainsStr(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// execMove — cross-module move updates references
// ---------------------------------------------------------------------------

func TestMove_Page_CrossModule(t *testing.T) {
	srcMod := mkModule("SrcModule")
	dstMod := mkModule("DstModule")
	pg := mkPageGen(string(nextID("pg")), "MyPage")
	moved := false
	refUpdated := false
	mb := &mock.MockBackend{
		IsConnectedFunc:     func() bool { return true },
		ListModulesFunc:     func() ([]*model.Module, error) { return []*model.Module{srcMod, dstMod}, nil },
		ListFoldersFunc:     func() ([]*types.FolderInfo, error) { return nil, nil },
		MoveDocumentGenFunc: func(id, containerID model.ID) error { moved = true; return nil },
		UpdateQualifiedNameInAllUnitsFunc: func(old, new string) (int, error) {
			refUpdated = true
			return 3, nil
		},
	}
	h := mkHierarchy(srcMod, dstMod)
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Pages = makePagesRepo([]*genPg.Page{pg}, srcMod.ID)
	rebuildDeps(ctx)
	assertNoError(t, execMoveFn(ctx, &ast.MoveStmt{
		DocumentType: ast.DocumentTypePage,
		Name:         ast.QualifiedName{Module: "SrcModule", Name: "MyPage"},
		TargetModule: "DstModule",
	}, ctx.Deps))
	if !moved {
		t.Fatal("Expected MovePageGen to be called")
	}
	if !refUpdated {
		t.Error("Expected reference update for cross-module move")
	}
	assertContainsStr(t, buf.String(), "Moved page")
	assertContainsStr(t, buf.String(), "Updated references")
}

// ---------------------------------------------------------------------------
// execMove — unsupported type
// ---------------------------------------------------------------------------

func TestMove_UnsupportedType(t *testing.T) {
	mod := mkModule("MyModule")
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execMoveFn(ctx, &ast.MoveStmt{
		DocumentType: "UNKNOWN",
		Name:         ast.QualifiedName{Module: "MyModule", Name: "Thing"},
	}, ctx.Deps)
	assertError(t, err)
	assertContainsStr(t, err.Error(), "unsupported")
}

// ---------------------------------------------------------------------------
// execMove — backend error on move
// ---------------------------------------------------------------------------

func TestMove_Page_BackendError(t *testing.T) {
	mod := mkModule("MyModule")
	pg := mkPageGen(string(nextID("pg")), "MyPage")
	mb := &mock.MockBackend{
		IsConnectedFunc:     func() bool { return true },
		ListModulesFunc:     func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc:     func() ([]*types.FolderInfo, error) { return nil, nil },
		MoveDocumentGenFunc: func(id, containerID model.ID) error { return fmt.Errorf("disk full") },
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Pages = makePagesRepo([]*genPg.Page{pg}, mod.ID)
	rebuildDeps(ctx)
	err := execMoveFn(ctx, &ast.MoveStmt{
		DocumentType: ast.DocumentTypePage,
		Name:         ast.QualifiedName{Module: "MyModule", Name: "MyPage"},
	}, ctx.Deps)
	assertError(t, err)
	assertContainsStr(t, err.Error(), "move page")
}

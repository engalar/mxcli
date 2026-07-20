// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

func TestShowImageCollections_Mock(t *testing.T) {
	mod := mkModule("Icons")
	ic := &types.ImageCollection{
		BaseElement: model.BaseElement{ID: nextID("ic")},
		ContainerID: mod.ID,
		Name:        "AppIcons",
		ExportLevel: "Hidden",
	}

	h := mkHierarchy(mod)
	withContainer(h, ic.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{ic}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, listImageCollections(ctx, ""))

	out := buf.String()
	assertContainsStr(t, out, "Image Collection")
	assertContainsStr(t, out, "Icons.AppIcons")
}

func TestShowImageCollections_FilterByModule(t *testing.T) {
	mod1 := mkModule("Icons")
	mod2 := mkModule("Other")
	ic1 := &types.ImageCollection{
		BaseElement: model.BaseElement{ID: nextID("ic")},
		ContainerID: mod1.ID,
		Name:        "AppIcons",
		ExportLevel: "Hidden",
	}
	ic2 := &types.ImageCollection{
		BaseElement: model.BaseElement{ID: nextID("ic")},
		ContainerID: mod2.ID,
		Name:        "OtherIcons",
		ExportLevel: "Hidden",
	}

	h := mkHierarchy(mod1, mod2)
	withContainer(h, ic1.ContainerID, mod1.ID)
	withContainer(h, ic2.ContainerID, mod2.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{ic1, ic2}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, listImageCollections(ctx, "Icons"))

	out := buf.String()
	assertContainsStr(t, out, "Icons.AppIcons")
	assertNotContainsStr(t, out, "Other.OtherIcons")
}

func TestDescribeImageCollection_NotFound(t *testing.T) {
	mod := mkModule("Icons")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return nil, nil },
	}

	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertError(t, describeImageCollection(ctx, ast.QualifiedName{Module: "Icons", Name: "NoSuch"}))
}

func TestDescribeImageCollection_Mock(t *testing.T) {
	mod := mkModule("Icons")
	ic := &types.ImageCollection{
		BaseElement: model.BaseElement{ID: nextID("ic")},
		ContainerID: mod.ID,
		Name:        "AppIcons",
		ExportLevel: "Hidden",
	}

	h := mkHierarchy(mod)
	withContainer(h, ic.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{ic}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, describeImageCollection(ctx, ast.QualifiedName{Module: "Icons", Name: "AppIcons"}))

	out := buf.String()
	assertContainsStr(t, out, "create or modify image collection")
}

func TestCreateImageCollection_AlreadyExists_Error(t *testing.T) {
	mod := mkModule("MyModule")
	existing := &types.ImageCollection{
		BaseElement: model.BaseElement{ID: nextID("ic")},
		ContainerID: mod.ID,
		Name:        "Icons",
	}
	h := mkHierarchy(mod)
	withContainer(h, existing.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		ListModulesFunc:          func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{existing}, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))

	err := execCreateImageCollectionFn(ctx, &ast.CreateImageCollectionStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "Icons"},
	}, ctx.Deps)
	assertError(t, err)
}

func TestCreateImageCollection_OrModify_PreservesIDAndUpdates(t *testing.T) {
	mod := mkModule("MyModule")
	existing := &types.ImageCollection{
		BaseElement: model.BaseElement{ID: nextID("ic")},
		ContainerID: mod.ID,
		Name:        "Icons",
	}
	h := mkHierarchy(mod)
	withContainer(h, existing.ContainerID, mod.ID)

	updated := false
	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		ListModulesFunc:          func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{existing}, nil },
		UpdateImageCollectionFunc: func(ic *types.ImageCollection) error {
			if ic.ID != existing.ID {
				t.Errorf("expected existing ID %s, got %s", existing.ID, ic.ID)
			}
			updated = true
			return nil
		},
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))

	err := execCreateImageCollectionFn(ctx, &ast.CreateImageCollectionStmt{
		Name:           ast.QualifiedName{Module: "MyModule", Name: "Icons"},
		CreateOrModify: true,
	}, ctx.Deps)
	assertNoError(t, err)
	if !updated {
		t.Error("expected existing collection to be updated")
	}
	assertContainsStr(t, buf.String(), "Modified image collection")
}

func mkImageCollectionWithImages(mod *model.Module, name string, images ...types.Image) *types.ImageCollection {
	return &types.ImageCollection{
		BaseElement: model.BaseElement{ID: nextID("ic")},
		ContainerID: mod.ID,
		Name:        name,
		ExportLevel: "Hidden",
		Images:      images,
	}
}

func TestAlterImageCollection_NotConnected(t *testing.T) {
	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return false }}
	ctx, _ := newMockCtx(t, withBackend(mb))
	err := execAlterImageCollectionFn(ctx, &ast.AlterImageCollectionStmt{
		Name:    ast.QualifiedName{Module: "Mod", Name: "Icons"},
		Actions: []ast.ImageCollectionAction{&ast.DropImageAction{ImageName: "logo"}},
	}, ctx.Deps)
	assertError(t, err)
}

func TestAlterImageCollection_NotFound(t *testing.T) {
	mod := mkModule("Mod")
	h := mkHierarchy(mod)
	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return nil, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execAlterImageCollectionFn(ctx, &ast.AlterImageCollectionStmt{
		Name:    ast.QualifiedName{Module: "Mod", Name: "NoSuch"},
		Actions: []ast.ImageCollectionAction{&ast.DropImageAction{ImageName: "x"}},
	}, ctx.Deps)
	assertError(t, err)
}

func TestAlterImageCollection_Drop(t *testing.T) {
	mod := mkModule("Mod")
	ic := mkImageCollectionWithImages(mod, "Icons",
		types.Image{Name: "logo", Data: []byte{1}, Format: "Png"},
		types.Image{Name: "banner", Data: []byte{2}, Format: "Png"},
	)
	h := mkHierarchy(mod)
	withContainer(h, ic.ContainerID, mod.ID)

	var saved *types.ImageCollection
	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListImageCollectionsFunc:  func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{ic}, nil },
		UpdateImageCollectionFunc: func(c *types.ImageCollection) error { saved = c; return nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))

	err := execAlterImageCollectionFn(ctx, &ast.AlterImageCollectionStmt{
		Name:    ast.QualifiedName{Module: "Mod", Name: "Icons"},
		Actions: []ast.ImageCollectionAction{&ast.DropImageAction{ImageName: "logo"}},
	}, ctx.Deps)
	assertNoError(t, err)
	if saved == nil {
		t.Fatal("expected UpdateImageCollection to be called")
	}
	if len(saved.Images) != 1 || saved.Images[0].Name != "banner" {
		t.Errorf("expected only banner remaining, got %+v", saved.Images)
	}
}

func TestAlterImageCollection_DropMissingImage(t *testing.T) {
	mod := mkModule("Mod")
	ic := mkImageCollectionWithImages(mod, "Icons", types.Image{Name: "logo"})
	h := mkHierarchy(mod)
	withContainer(h, ic.ContainerID, mod.ID)
	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{ic}, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execAlterImageCollectionFn(ctx, &ast.AlterImageCollectionStmt{
		Name:    ast.QualifiedName{Module: "Mod", Name: "Icons"},
		Actions: []ast.ImageCollectionAction{&ast.DropImageAction{ImageName: "ghost"}},
	}, ctx.Deps)
	assertError(t, err)
}

func TestAlterImageCollection_Rename(t *testing.T) {
	mod := mkModule("Mod")
	ic := mkImageCollectionWithImages(mod, "Icons", types.Image{Name: "old", Data: []byte{1}})
	h := mkHierarchy(mod)
	withContainer(h, ic.ContainerID, mod.ID)

	var saved *types.ImageCollection
	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListImageCollectionsFunc:  func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{ic}, nil },
		UpdateImageCollectionFunc: func(c *types.ImageCollection) error { saved = c; return nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execAlterImageCollectionFn(ctx, &ast.AlterImageCollectionStmt{
		Name:    ast.QualifiedName{Module: "Mod", Name: "Icons"},
		Actions: []ast.ImageCollectionAction{&ast.RenameImageAction{From: "old", To: "new"}},
	}, ctx.Deps)
	assertNoError(t, err)
	if saved == nil || len(saved.Images) != 1 || saved.Images[0].Name != "new" {
		t.Errorf("expected image renamed to new, got %+v", saved)
	}
}

func TestAlterImageCollection_RenameToExisting(t *testing.T) {
	mod := mkModule("Mod")
	ic := mkImageCollectionWithImages(mod, "Icons",
		types.Image{Name: "a"}, types.Image{Name: "b"})
	h := mkHierarchy(mod)
	withContainer(h, ic.ContainerID, mod.ID)
	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{ic}, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execAlterImageCollectionFn(ctx, &ast.AlterImageCollectionStmt{
		Name:    ast.QualifiedName{Module: "Mod", Name: "Icons"},
		Actions: []ast.ImageCollectionAction{&ast.RenameImageAction{From: "a", To: "b"}},
	}, ctx.Deps)
	assertError(t, err)
}

func TestAlterImageCollection_AddAndSetFromFile(t *testing.T) {
	dir := t.TempDir()
	logoPath := filepath.Join(dir, "logo.png")
	if err := os.WriteFile(logoPath, []byte("PNGDATA"), 0o644); err != nil {
		t.Fatal(err)
	}

	mod := mkModule("Mod")
	ic := mkImageCollectionWithImages(mod, "Icons", types.Image{Name: "existing", Data: []byte("old")})
	h := mkHierarchy(mod)
	withContainer(h, ic.ContainerID, mod.ID)

	var saved *types.ImageCollection
	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListImageCollectionsFunc:  func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{ic}, nil },
		UpdateImageCollectionFunc: func(c *types.ImageCollection) error { saved = c; return nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))

	err := execAlterImageCollectionFn(ctx, &ast.AlterImageCollectionStmt{
		Name: ast.QualifiedName{Module: "Mod", Name: "Icons"},
		Actions: []ast.ImageCollectionAction{
			&ast.AddImageAction{ImageName: "logo", FilePath: logoPath},
			&ast.SetImageAction{ImageName: "existing", FilePath: logoPath},
		},
	}, ctx.Deps)
	assertNoError(t, err)
	if saved == nil {
		t.Fatal("expected update")
	}
	idx := -1
	for i, img := range saved.Images {
		if img.Name == "logo" {
			idx = i
		}
	}
	if idx < 0 || string(saved.Images[idx].Data) != "PNGDATA" || saved.Images[idx].Format != "Png" {
		t.Errorf("logo not added correctly: %+v", saved.Images)
	}
}

func TestAlterImageCollection_Move(t *testing.T) {
	src := mkModule("Mod")
	dst := mkModule("Other")
	ic := mkImageCollectionWithImages(src, "Icons", types.Image{Name: "logo"})
	h := mkHierarchy(src, dst)
	withContainer(h, ic.ContainerID, src.ID)

	var moved *types.ImageCollection
	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		ListModulesFunc:          func() ([]*model.Module, error) { return []*model.Module{src, dst}, nil },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{ic}, nil },
		MoveImageCollectionFunc:  func(c *types.ImageCollection) error { moved = c; return nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execAlterImageCollectionFn(ctx, &ast.AlterImageCollectionStmt{
		Name:    ast.QualifiedName{Module: "Mod", Name: "Icons"},
		Actions: []ast.ImageCollectionAction{&ast.MoveImageCollectionAction{Target: ast.QualifiedName{Module: "Other", Name: "Icons"}}},
	}, ctx.Deps)
	assertNoError(t, err)
	if moved == nil || moved.ContainerID != dst.ID {
		t.Errorf("expected move to Other module (%s), got %+v", dst.ID, moved)
	}
}

func TestAlterImageCollection_Export(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "sub", "out.png")

	mod := mkModule("Mod")
	ic := mkImageCollectionWithImages(mod, "Icons", types.Image{Name: "logo", Data: []byte("EXPORTED"), Format: "Png"})
	h := mkHierarchy(mod)
	withContainer(h, ic.ContainerID, mod.ID)
	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{ic}, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execAlterImageCollectionFn(ctx, &ast.AlterImageCollectionStmt{
		Name:    ast.QualifiedName{Module: "Mod", Name: "Icons"},
		Actions: []ast.ImageCollectionAction{&ast.ExportImageAction{ImageName: "logo", FilePath: outPath}},
	}, ctx.Deps)
	assertNoError(t, err)
	data, readErr := os.ReadFile(outPath)
	if readErr != nil {
		t.Fatalf("expected exported file: %v", readErr)
	}
	if string(data) != "EXPORTED" {
		t.Errorf("exported data = %q, want EXPORTED", data)
	}
}

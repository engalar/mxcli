// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

func layoutStmt(module, name string) *ast.CreateLayoutStmt {
	return &ast.CreateLayoutStmt{
		Name:       ast.QualifiedName{Module: module, Name: name},
		LayoutType: "Responsive",
		Widgets: []*ast.LayoutWidgetV3{
			{
				Kind: ast.LayoutWidgetScrollContainer,
				Name: "sc1",
				Regions: []*ast.LayoutRegionV3{
					{Name: "center", Placeholders: []*ast.LayoutPlaceholderV3{{Name: "Main"}}},
					{Name: "top", Placeholders: []*ast.LayoutPlaceholderV3{{Name: "Header"}}},
				},
			},
		},
	}
}

func TestExecCreateLayout_CallsCreateLayoutGen(t *testing.T) {
	mod := mkModule("Mod")

	var created *genPg.Layout
	var createdParent string
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetContainerIDFunc: func(moduleID model.ID, folder string) (model.ID, error) {
			return "container-uuid", nil
		},
		CreateLayoutGenFunc: func(parentUUID, containmentName string, layout *genPg.Layout) error {
			createdParent = parentUUID
			created = layout
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(mkHierarchy(mod)))
	ctx.Layouts = makeLayoutsRepo(nil, mod.ID)
	ctx.Deps = ctx.buildDeps()

	if err := ExecCreateOrModifyLayoutFn(ctx, layoutStmt("Mod", "MyLayout"), ctx.Deps); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created == nil {
		t.Fatal("CreateLayoutGen was not called")
	}
	if created.Name() != "MyLayout" {
		t.Errorf("layout Name = %q, want MyLayout", created.Name())
	}
	if createdParent != "container-uuid" {
		t.Errorf("parent UUID = %q, want container-uuid", createdParent)
	}
	if got := buf.String(); !strings.Contains(got, "Created layout Mod.MyLayout") {
		t.Errorf("output = %q, want it to contain 'Created layout Mod.MyLayout'", got)
	}
}

func TestExecCreateLayout_NotConnected(t *testing.T) {
	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return false }}
	ctx, _ := newMockCtx(t, withBackend(mb))
	err := ExecCreateOrModifyLayoutFn(ctx, layoutStmt("Mod", "MyLayout"), ctx.Deps)
	assertError(t, err)
	assertContainsStr(t, err.Error(), "not connected")
}

func TestExecCreateLayout_AlreadyExists(t *testing.T) {
	mod := mkModule("Mod")
	existing := genPg.NewLayout()
	existing.SetName("MyLayout")

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		CreateLayoutGenFunc: func(string, string, *genPg.Layout) error {
			t.Fatal("CreateLayoutGen must not be called when layout already exists")
			return nil
		},
	}

	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(mkHierarchy(mod)))
	ctx.Layouts = makeLayoutsRepo([]*genPg.Layout{existing}, mod.ID)
	ctx.Deps = ctx.buildDeps()

	s := layoutStmt("Mod", "MyLayout") // IsModify=false → must error
	err := ExecCreateOrModifyLayoutFn(ctx, s, ctx.Deps)
	assertError(t, err)
	assertContainsStr(t, err.Error(), "already exists")
}

func TestExecCreateLayout_ModifyReplacesExisting(t *testing.T) {
	mod := mkModule("Mod")
	existing := genPg.NewLayout()
	existing.SetID(element.ID(nextID("lay")))
	existing.SetName("MyLayout")

	deleted := false
	created := false
	mb := &mock.MockBackend{
		IsConnectedFunc:    func() bool { return true },
		ListModulesFunc:    func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetContainerIDFunc: func(model.ID, string) (model.ID, error) { return "container-uuid", nil },
		DeleteLayoutGenFunc: func(id model.ID) error {
			deleted = true
			if model.ID(existing.ID()) != id {
				t.Errorf("DeleteLayoutGen id = %q, want %q", id, existing.ID())
			}
			return nil
		},
		CreateLayoutGenFunc: func(string, string, *genPg.Layout) error {
			created = true
			if !deleted {
				t.Error("CreateLayoutGen called before DeleteLayoutGen")
			}
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(mkHierarchy(mod)))
	ctx.Layouts = makeLayoutsRepo([]*genPg.Layout{existing}, mod.ID)
	ctx.Deps = ctx.buildDeps()

	s := layoutStmt("Mod", "MyLayout")
	s.IsModify = true
	if err := ExecCreateOrModifyLayoutFn(ctx, s, ctx.Deps); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted || !created {
		t.Fatalf("expected delete+create; deleted=%v created=%v", deleted, created)
	}
	if got := buf.String(); !strings.Contains(got, "Modified layout Mod.MyLayout") {
		t.Errorf("output = %q, want it to contain 'Modified layout Mod.MyLayout'", got)
	}
}

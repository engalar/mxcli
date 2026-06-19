// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// ---------------------------------------------------------------------------
// Not connected
// ---------------------------------------------------------------------------

func TestAlterPage_NotConnected(t *testing.T) {
	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return false }}
	ctx, _ := newMockCtx(t, withBackend(mb))
	err := execAlterPage(ctx, &ast.AlterPageStmt{
		PageName: ast.QualifiedName{Module: "M", Name: "P"},
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "not connected")
}

// ---------------------------------------------------------------------------
// Page not found
// ---------------------------------------------------------------------------

func TestAlterPage_PageNotFound(t *testing.T) {
	mod := mkModule("MyModule")
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Pages = makePagesRepo(nil, mod.ID)
	err := execAlterPage(ctx, &ast.AlterPageStmt{
		PageName: ast.QualifiedName{Module: "MyModule", Name: "Missing"},
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// Page happy path — SET property + Save
// ---------------------------------------------------------------------------

func TestAlterPage_SetProperty_Success(t *testing.T) {
	mod := mkModule("MyModule")
	pg := mkPageGen(string(nextID("pg")), "TestPage")
	saved := false
	setPropCalled := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
		OpenPageForMutationFunc: func(unitID model.ID) (backend.PageMutator, error) {
			return &mock.MockPageMutator{
				SetWidgetPropertyFunc: func(widgetRef string, prop string, value any) error {
					setPropCalled = true
					if widgetRef != "myWidget" {
						t.Errorf("expected widgetRef myWidget, got %s", widgetRef)
					}
					if prop != "Caption" {
						t.Errorf("expected prop Caption, got %s", prop)
					}
					return nil
				},
				SaveFunc: func() error { saved = true; return nil },
			}, nil
		},
	}
	h := mkHierarchy(mod)
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Pages = makePagesRepo([]*genPg.Page{pg}, mod.ID)
	assertNoError(t, execAlterPage(ctx, &ast.AlterPageStmt{
		PageName: ast.QualifiedName{Module: "MyModule", Name: "TestPage"},
		Operations: []ast.AlterPageOperation{
			&ast.SetPropertyOp{
				Target:     ast.WidgetRef{Widget: "myWidget"},
				Properties: map[string]any{"Caption": "Hello"},
			},
		},
	}))
	if !setPropCalled {
		t.Error("expected SetWidgetProperty to be called")
	}
	if !saved {
		t.Error("expected Save to be called")
	}
	assertContainsStr(t, buf.String(), "Altered page")
	assertContainsStr(t, buf.String(), "MyModule.TestPage")
}

// ---------------------------------------------------------------------------
// Snippet happy path
// ---------------------------------------------------------------------------

func TestAlterPage_Snippet_Success(t *testing.T) {
	mod := mkModule("MyModule")
	snp := mkSnippetGen(string(nextID("snp")), "TestSnippet")
	saved := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
		OpenPageForMutationFunc: func(unitID model.ID) (backend.PageMutator, error) {
			return &mock.MockPageMutator{
				SaveFunc: func() error { saved = true; return nil },
			}, nil
		},
	}
	h := mkHierarchy(mod)
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Snippets = makeSnippetsRepo([]*genPg.Snippet{snp}, mod.ID)
	assertNoError(t, execAlterPage(ctx, &ast.AlterPageStmt{
		ContainerType: "snippet",
		PageName:      ast.QualifiedName{Module: "MyModule", Name: "TestSnippet"},
	}))
	if !saved {
		t.Error("expected Save to be called")
	}
	assertContainsStr(t, buf.String(), "Altered snippet")
}

// Issue #402 — visitor sets ContainerType to uppercase "SNIPPET"; executor
// must normalise before comparing so the snippet branch is taken.
func TestAlterPage_Snippet_UppercaseContainerType_Issue402(t *testing.T) {
	mod := mkModule("MyModule")
	snp := mkSnippetGen(string(nextID("snp")), "TestSnippet")
	saved := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
		OpenPageForMutationFunc: func(unitID model.ID) (backend.PageMutator, error) {
			return &mock.MockPageMutator{
				SaveFunc: func() error { saved = true; return nil },
			}, nil
		},
	}
	h := mkHierarchy(mod)
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Snippets = makeSnippetsRepo([]*genPg.Snippet{snp}, mod.ID)
	assertNoError(t, execAlterPage(ctx, &ast.AlterPageStmt{
		ContainerType: "SNIPPET", // uppercase as produced by the AST visitor
		PageName:      ast.QualifiedName{Module: "MyModule", Name: "TestSnippet"},
	}))
	if !saved {
		t.Error("expected Save to be called")
	}
	assertContainsStr(t, buf.String(), "Altered snippet")
}

// ---------------------------------------------------------------------------
// Open mutator error
// ---------------------------------------------------------------------------

func TestAlterPage_OpenMutatorError(t *testing.T) {
	mod := mkModule("MyModule")
	pg := mkPageGen(string(nextID("pg")), "TestPage")
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
		OpenPageForMutationFunc: func(unitID model.ID) (backend.PageMutator, error) {
			return nil, fmt.Errorf("lock error")
		},
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Pages = makePagesRepo([]*genPg.Page{pg}, mod.ID)
	err := execAlterPage(ctx, &ast.AlterPageStmt{
		PageName: ast.QualifiedName{Module: "MyModule", Name: "TestPage"},
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "open page")
}

// ---------------------------------------------------------------------------
// Save error
// ---------------------------------------------------------------------------

func TestAlterPage_SaveError(t *testing.T) {
	mod := mkModule("MyModule")
	pg := mkPageGen(string(nextID("pg")), "TestPage")
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
		OpenPageForMutationFunc: func(unitID model.ID) (backend.PageMutator, error) {
			return &mock.MockPageMutator{
				SaveFunc: func() error { return fmt.Errorf("disk full") },
			}, nil
		},
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Pages = makePagesRepo([]*genPg.Page{pg}, mod.ID)
	err := execAlterPage(ctx, &ast.AlterPageStmt{
		PageName: ast.QualifiedName{Module: "MyModule", Name: "TestPage"},
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "save")
}

// ---------------------------------------------------------------------------
// DROP widget via mutator
// ---------------------------------------------------------------------------

func TestAlterPage_DropWidget_Success(t *testing.T) {
	mod := mkModule("MyModule")
	pg := mkPageGen(string(nextID("pg")), "TestPage")
	dropCalled := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
		OpenPageForMutationFunc: func(unitID model.ID) (backend.PageMutator, error) {
			return &mock.MockPageMutator{
				DropWidgetFunc: func(refs []backend.WidgetRef) error {
					dropCalled = true
					if len(refs) != 1 || refs[0].Widget != "oldWidget" {
						t.Errorf("unexpected refs: %v", refs)
					}
					return nil
				},
				SaveFunc: func() error { return nil },
			}, nil
		},
	}
	h := mkHierarchy(mod)
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Pages = makePagesRepo([]*genPg.Page{pg}, mod.ID)
	assertNoError(t, execAlterPage(ctx, &ast.AlterPageStmt{
		PageName: ast.QualifiedName{Module: "MyModule", Name: "TestPage"},
		Operations: []ast.AlterPageOperation{
			&ast.DropWidgetOp{
				Targets: []ast.WidgetRef{{Widget: "oldWidget"}},
			},
		},
	}))
	if !dropCalled {
		t.Error("expected DropWidget to be called")
	}
	assertContainsStr(t, buf.String(), "Altered page")
}

// ---------------------------------------------------------------------------
// ADD VARIABLE
// ---------------------------------------------------------------------------

func TestAlterPage_AddVariable_Success(t *testing.T) {
	mod := mkModule("MyModule")
	pg := mkPageGen(string(nextID("pg")), "TestPage")
	addVarCalled := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
		OpenPageForMutationFunc: func(unitID model.ID) (backend.PageMutator, error) {
			return &mock.MockPageMutator{
				AddVariableFunc: func(name, dataType, defaultValue string) error {
					addVarCalled = true
					if name != "MyVar" || dataType != "String" || defaultValue != "hello" {
						t.Errorf("unexpected variable: %s %s %s", name, dataType, defaultValue)
					}
					return nil
				},
				SaveFunc: func() error { return nil },
			}, nil
		},
	}
	h := mkHierarchy(mod)
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Pages = makePagesRepo([]*genPg.Page{pg}, mod.ID)
	assertNoError(t, execAlterPage(ctx, &ast.AlterPageStmt{
		PageName: ast.QualifiedName{Module: "MyModule", Name: "TestPage"},
		Operations: []ast.AlterPageOperation{
			&ast.AddVariableOp{
				Variable: ast.PageVariable{Name: "MyVar", DataType: "String", DefaultValue: "hello"},
			},
		},
	}))
	if !addVarCalled {
		t.Error("expected AddVariable to be called")
	}
	assertContainsStr(t, buf.String(), "Altered page")
}

// ---------------------------------------------------------------------------
// SET Layout on snippet — unsupported
// ---------------------------------------------------------------------------

func TestAlterPage_SetLayout_Snippet_Unsupported(t *testing.T) {
	mod := mkModule("MyModule")
	snp := mkSnippetGen(string(nextID("snp")), "TestSnippet")
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
		OpenPageForMutationFunc: func(unitID model.ID) (backend.PageMutator, error) {
			return &mock.MockPageMutator{}, nil
		},
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Snippets = makeSnippetsRepo([]*genPg.Snippet{snp}, mod.ID)
	err := execAlterPage(ctx, &ast.AlterPageStmt{
		ContainerType: "snippet",
		PageName:      ast.QualifiedName{Module: "MyModule", Name: "TestSnippet"},
		Operations: []ast.AlterPageOperation{
			&ast.SetLayoutOp{
				NewLayout: ast.QualifiedName{Module: "M", Name: "L"},
			},
		},
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "not supported")
}

// ---------------------------------------------------------------------------
// REPLACE widget — BUG-08: replacement may reuse the target widget's name
// ---------------------------------------------------------------------------

func TestAlterPage_ReplaceWidget_SameName_Allowed(t *testing.T) {
	mod := mkModule("MyModule")
	pg := mkPageGen(string(nextID("pg")), "TestPage")
	replaceCalled := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
		OpenPageForMutationFunc: func(unitID model.ID) (backend.PageMutator, error) {
			return &mock.MockPageMutator{
				// Inherited scope contains the widget being replaced. Without
				// the excludeFromScope fix, the duplicate-name check in
				// pageBuilder.registerWidgetName would reject the same-name
				// replacement.
				WidgetScopeFunc: func() map[string]model.ID {
					return map[string]model.ID{"lblHeading": model.ID(nextID("lbl"))}
				},
				ParamScopeFunc: func() (map[string]model.ID, map[string]string) {
					return map[string]model.ID{}, map[string]string{}
				},
				EnclosingEntityFunc: func(widgetRef string) string { return "" },
				FindWidgetFunc:      func(name string) bool { return name == "lblHeading" },
				ReplaceWidgetGenFunc: func(widgetRef string, columnRef string, widgets []element.Element) error {
					replaceCalled = true
					if widgetRef != "lblHeading" {
						t.Errorf("expected widgetRef lblHeading, got %s", widgetRef)
					}
					if len(widgets) != 1 {
						t.Fatalf("expected 1 replacement widget, got %d", len(widgets))
					}
					return nil
				},
				SaveFunc: func() error { return nil },
			}, nil
		},
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Pages = makePagesRepo([]*genPg.Page{pg}, mod.ID)
	assertNoError(t, execAlterPage(ctx, &ast.AlterPageStmt{
		PageName: ast.QualifiedName{Module: "MyModule", Name: "TestPage"},
		Operations: []ast.AlterPageOperation{
			&ast.ReplaceWidgetOp{
				Target: ast.WidgetRef{Widget: "lblHeading"},
				NewWidgets: []*ast.WidgetV3{
					{Type: "text", Name: "lblHeading", Properties: map[string]any{"Content": "New label"}},
				},
			},
		},
	}))
	if !replaceCalled {
		t.Error("expected ReplaceWidgetGen to be called")
	}
}

// ---------------------------------------------------------------------------
// collectWidgetNamesV3 unit tests
// ---------------------------------------------------------------------------

func TestCollectWidgetNamesV3(t *testing.T) {
	tests := []struct {
		name     string
		widgets  []*ast.WidgetV3
		expected []string
	}{
		{
			name:     "empty",
			widgets:  nil,
			expected: nil,
		},
		{
			name: "single named widget",
			widgets: []*ast.WidgetV3{
				{Name: "btnSave", Type: "actionbutton"},
			},
			expected: []string{"btnSave"},
		},
		{
			name: "unnamed widget skipped",
			widgets: []*ast.WidgetV3{
				{Type: "dynamictext", Properties: map[string]any{"content": "hello"}},
			},
			expected: nil,
		},
		{
			name: "widget with children",
			widgets: []*ast.WidgetV3{
				{
					Name: "parent",
					Children: []*ast.WidgetV3{
						{Name: "child1"},
						{Name: "child2"},
					},
				},
			},
			expected: []string{"parent", "child1", "child2"},
		},
		{
			name: "deep nesting",
			widgets: []*ast.WidgetV3{
				{
					Name: "a",
					Children: []*ast.WidgetV3{
						{Name: "b", Children: []*ast.WidgetV3{
							{Name: "c"},
						}},
					},
				},
			},
			expected: []string{"a", "b", "c"},
		},
		{
			name: "mixed named and unnamed",
			widgets: []*ast.WidgetV3{
				{Name: "w1"},
				{Type: "text", Name: ""},
				{Name: "w2", Children: []*ast.WidgetV3{
					{Type: "inner", Name: ""},
					{Name: "w3"},
				}},
			},
			expected: []string{"w1", "w2", "w3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collectWidgetNamesV3(tt.widgets)
			if len(got) != len(tt.expected) {
				t.Fatalf("collectWidgetNamesV3() = %v (len=%d), want %v (len=%d)", got, len(got), tt.expected, len(tt.expected))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("collectWidgetNamesV3()[%d] = %q, want %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

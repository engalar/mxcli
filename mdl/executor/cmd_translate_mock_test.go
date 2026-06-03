// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

func TestTranslatePage_SetWidgetTranslation(t *testing.T) {
	mod := mkModule("MyModule")
	pg := mkPageGen(string(nextID("pg")), "Home")
	saved := false
	var gotWidget, gotProp, gotLang, gotText string
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
		OpenPageForMutationFunc: func(unitID model.ID) (backend.PageMutator, error) {
			return &mock.MockPageMutator{
				SetWidgetTranslationFunc: func(widgetRef, prop, langCode, text string) error {
					gotWidget, gotProp, gotLang, gotText = widgetRef, prop, langCode, text
					return nil
				},
				SaveFunc: func() error { saved = true; return nil },
			}, nil
		},
	}
	h := mkHierarchy(mod)
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Pages = makePagesRepo([]*genPg.Page{pg}, mod.ID)

	stmt := &ast.TranslateStmt{
		DocType: "PAGE",
		QName:   ast.QualifiedName{Module: "MyModule", Name: "Home"},
		Lang:    "zh_CN",
		Ops: []ast.TranslateSetOp{
			{Path: "Button_Submit.caption", Text: "提交"},
		},
	}
	if err := translateDocument(ctx, stmt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotWidget != "Button_Submit" {
		t.Errorf("expected widget Button_Submit, got %q", gotWidget)
	}
	if gotProp != "caption" {
		t.Errorf("expected prop caption, got %q", gotProp)
	}
	if gotLang != "zh_CN" {
		t.Errorf("expected lang zh_CN, got %q", gotLang)
	}
	if gotText != "提交" {
		t.Errorf("expected text 提交, got %q", gotText)
	}
	if !saved {
		t.Error("expected Save to be called")
	}
	if !strings.Contains(buf.String(), "Translated page") {
		t.Errorf("expected 'Translated page' in output: %s", buf.String())
	}
}

func TestTranslatePage_SetPageTitle(t *testing.T) {
	mod := mkModule("MyModule")
	pg := mkPageGen(string(nextID("pg")), "Home")
	var gotLang, gotText string
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
		OpenPageForMutationFunc: func(unitID model.ID) (backend.PageMutator, error) {
			return &mock.MockPageMutator{
				SetPageTitleTranslationFunc: func(langCode, text string) error {
					gotLang, gotText = langCode, text
					return nil
				},
				SaveFunc: func() error { return nil },
			}, nil
		},
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Pages = makePagesRepo([]*genPg.Page{pg}, mod.ID)

	stmt := &ast.TranslateStmt{
		DocType: "PAGE",
		QName:   ast.QualifiedName{Module: "MyModule", Name: "Home"},
		Lang:    "zh_CN",
		Ops: []ast.TranslateSetOp{
			{Path: "title", Text: "首页"},
		},
	}
	if err := translateDocument(ctx, stmt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotLang != "zh_CN" || gotText != "首页" {
		t.Errorf("expected title translation zh_CN/首页, got %q/%q", gotLang, gotText)
	}
}

func TestTranslate_EnumerationStub(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	stmt := &ast.TranslateStmt{
		DocType: "ENUMERATION",
		QName:   ast.QualifiedName{Module: "MyModule", Name: "Status"},
		Lang:    "zh_CN",
		Ops:     []ast.TranslateSetOp{{Path: "Open", Text: "打开"}},
	}
	err := translateDocument(ctx, stmt)
	if err == nil {
		t.Fatal("expected not-implemented error for ENUMERATION stub")
	}
}

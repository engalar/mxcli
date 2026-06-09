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
		IsConnectedFunc:        func() bool { return true },
		GetProjectSettingsFunc: translateTestSettings,
		ListModulesFunc:        func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc:        func() ([]*types.FolderInfo, error) { return nil, nil },
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
		IsConnectedFunc:        func() bool { return true },
		GetProjectSettingsFunc: translateTestSettings,
		ListModulesFunc:        func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc:        func() ([]*types.FolderInfo, error) { return nil, nil },
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

func TestTranslateEnumeration_InvalidPath(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:        func() bool { return true },
		GetProjectSettingsFunc: translateTestSettings,
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	stmt := &ast.TranslateStmt{
		DocType: "ENUMERATION",
		QName:   ast.QualifiedName{Module: "MyModule", Name: "Status"},
		Lang:    "zh_CN",
		Ops:     []ast.TranslateSetOp{{Path: "Open", Text: "打开"}}, // missing .caption
	}
	err := translateDocument(ctx, stmt)
	if err == nil || !strings.Contains(err.Error(), "invalid enumeration path") {
		t.Fatalf("expected 'invalid enumeration path' error, got: %v", err)
	}
}

func TestTranslate_LangNotRegistered(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) {
			return &model.ProjectSettings{
				Language: &model.LanguageSettings{
					DefaultLanguageCode: "en_US",
					Languages:           []model.Language{{Code: "en_US"}},
				},
			}, nil
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	stmt := &ast.TranslateStmt{
		DocType: "PAGE",
		QName:   ast.QualifiedName{Module: "MyModule", Name: "Home"},
		Lang:    "zh_CN",
		Ops:     []ast.TranslateSetOp{{Path: "Button1.caption", Text: "提交"}},
	}
	err := translateDocument(ctx, stmt)
	if err == nil {
		t.Fatal("expected error for unregistered language")
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Errorf("expected 'not registered' error, got: %v", err)
	}
}

func TestTranslateEnumeration_SetsValueCaption(t *testing.T) {
	var gotEnum, gotValue, gotLang, gotText string
	calls := 0
	mb := &mock.MockBackend{
		IsConnectedFunc:        func() bool { return true },
		GetProjectSettingsFunc: translateTestSettings,
		SetEnumerationTranslationFunc: func(enumQN, valueName, langCode, text string) error {
			gotEnum, gotValue, gotLang, gotText = enumQN, valueName, langCode, text
			calls++
			return nil
		},
	}
	ctx, buf := newMockCtx(t, withBackend(mb))
	stmt := &ast.TranslateStmt{
		DocType: "ENUMERATION",
		QName:   ast.QualifiedName{Module: "MyModule", Name: "Status"},
		Lang:    "zh_CN",
		Ops:     []ast.TranslateSetOp{{Path: "ACTIVE.caption", Text: "活跃"}},
	}
	if err := translateDocument(ctx, stmt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 SetEnumerationTranslation call, got %d", calls)
	}
	if gotEnum != "MyModule.Status" || gotValue != "ACTIVE" || gotLang != "zh_CN" || gotText != "活跃" {
		t.Errorf("unexpected args: enum=%q value=%q lang=%q text=%q", gotEnum, gotValue, gotLang, gotText)
	}
	if !strings.Contains(buf.String(), "Translated enumeration MyModule.Status") {
		t.Errorf("unexpected output: %s", buf.String())
	}
}

func TestTranslateEnumeration_UnregisteredLang(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:        func() bool { return true },
		GetProjectSettingsFunc: translateTestSettings, // only en_US + zh_CN
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	stmt := &ast.TranslateStmt{
		DocType: "ENUMERATION",
		QName:   ast.QualifiedName{Module: "MyModule", Name: "Status"},
		Lang:    "fr_FR",
		Ops:     []ast.TranslateSetOp{{Path: "ACTIVE", Text: "actif"}},
	}
	err := translateDocument(ctx, stmt)
	if err == nil || !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("expected 'not registered' error, got: %v", err)
	}
}

func TestTranslateWorkflow_NotSupported(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:        func() bool { return true },
		GetProjectSettingsFunc: translateTestSettings,
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	stmt := &ast.TranslateStmt{
		DocType: "WORKFLOW",
		QName:   ast.QualifiedName{Module: "MyModule", Name: "Approval"},
		Lang:    "zh_CN",
		Ops:     []ast.TranslateSetOp{{Path: "userTask1.taskName", Text: "审核"}},
	}
	err := translateDocument(ctx, stmt)
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("expected 'not supported' error for WORKFLOW, got: %v", err)
	}
}

// translateTestSettings returns project settings with en_US + zh_CN registered.
func translateTestSettings() (*model.ProjectSettings, error) {
	return &model.ProjectSettings{
		Language: &model.LanguageSettings{
			DefaultLanguageCode: "en_US",
			Languages:           []model.Language{{Code: "en_US"}, {Code: "zh_CN"}},
		},
	}, nil
}

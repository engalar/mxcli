// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/mdl/catalog"
	"github.com/mendixlabs/mxcli/model"
)

func TestListLanguages_NilCatalog(t *testing.T) {
	ctx, _ := newMockCtx(t)
	// Catalog is nil by default in newMockCtx
	err := listLanguages(ctx)
	assertError(t, err)
}

func TestListLanguages_EmptyStringsTable(t *testing.T) {
	cat, err := catalog.New()
	if err != nil {
		t.Fatalf("failed to create catalog: %v", err)
	}
	defer cat.Close()

	ctx, buf := newMockCtx(t)
	ctx.Catalog = cat

	assertNoError(t, listLanguages(ctx))
	assertContainsStr(t, buf.String(), "No translatable strings found")
}

func TestListLanguages_WithRows(t *testing.T) {
	cat, err := catalog.New()
	if err != nil {
		t.Fatalf("failed to create catalog: %v", err)
	}
	defer cat.Close()

	db := cat.CatalogDB()
	_, err = db.Exec(`INSERT INTO strings (QualifiedName, ObjectType, StringValue, StringContext, Language, ElementId, ModuleName) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"MyModule.HomePage", "Page", "Hello", "Caption", "en_US", "id1", "MyModule")
	if err != nil {
		t.Fatalf("failed to seed strings table: %v", err)
	}
	_, err = db.Exec(`INSERT INTO strings (QualifiedName, ObjectType, StringValue, StringContext, Language, ElementId, ModuleName) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"MyModule.HomePage", "Page", "Bonjour", "Caption", "fr_FR", "id2", "MyModule")
	if err != nil {
		t.Fatalf("failed to seed strings table: %v", err)
	}

	ctx, buf := newMockCtx(t)
	ctx.Catalog = cat

	assertNoError(t, listLanguages(ctx))
	out := buf.String()
	assertContainsStr(t, out, "en_US")
	assertContainsStr(t, out, "fr_FR")
	assertContainsStr(t, out, "Language")
}

func TestShowSupportedLanguages(t *testing.T) {
	ctx, buf := newMockCtx(t)
	err := listSupportedLanguages(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "en_US") {
		t.Errorf("expected en_US in output, got: %s", out)
	}
	if !strings.Contains(out, "zh_CN") {
		t.Errorf("expected zh_CN in output, got: %s", out)
	}
	if !strings.Contains(out, "nl_NL") {
		t.Errorf("expected nl_NL in output, got: %s", out)
	}
}

func TestIsValidLanguageCode(t *testing.T) {
	if !isValidLanguageCode("en_US") {
		t.Error("en_US should be valid")
	}
	if !isValidLanguageCode("zh_CN") {
		t.Error("zh_CN should be valid")
	}
	if isValidLanguageCode("chinese") {
		t.Error("chinese should not be valid")
	}
	if isValidLanguageCode("EN_US") {
		t.Error("EN_US should not be valid (case-sensitive)")
	}
}

func TestAlterLanguageAdd_InvalidCode(t *testing.T) {
	ctx, _ := newMockCtx(t)
	stmt := &ast.AlterLanguageStmt{Op: ast.AlterLanguageAdd, Code: "chinese"}
	err := alterLanguage(ctx, stmt)
	if err == nil {
		t.Fatal("expected error for invalid language code")
	}
	if !strings.Contains(err.Error(), "not a valid Mendix language code") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAlterLanguageAdd_Success(t *testing.T) {
	var saved *model.ProjectSettings
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
		UpdateProjectSettingsFunc: func(ps *model.ProjectSettings) error {
			saved = ps
			return nil
		},
	}
	ctx, buf := newMockCtx(t, withBackend(mb))
	stmt := &ast.AlterLanguageStmt{Op: ast.AlterLanguageAdd, Code: "nl_NL"}
	if err := alterLanguage(ctx, stmt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved == nil {
		t.Fatal("expected UpdateProjectSettings to be called")
	}
	if len(saved.Language.Languages) != 2 {
		t.Fatalf("expected 2 languages, got %d", len(saved.Language.Languages))
	}
	if !strings.Contains(buf.String(), "nl_NL") {
		t.Errorf("expected output to mention nl_NL, got: %s", buf.String())
	}
}

func TestListLanguages_FromSettings(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) {
			return &model.ProjectSettings{
				Language: &model.LanguageSettings{
					DefaultLanguageCode: "en_US",
					Languages: []model.Language{
						{Code: "en_US"},
						{Code: "nl_NL", CheckCompleteness: true},
					},
				},
			}, nil
		},
	}
	ctx, buf := newMockCtx(t, withBackend(mb))
	if err := listLanguages(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	assertContainsStr(t, out, "en_US")
	assertContainsStr(t, out, "nl_NL")
	// Settings-backed listing should not require the catalog.
	if strings.Contains(out, "refresh catalog full") {
		t.Errorf("settings-backed listing should not mention catalog, got: %s", out)
	}
}

func TestListLanguages_SettingsEmptyFallsBackToCatalog(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) {
			return &model.ProjectSettings{}, nil // no Language part
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	// No catalog set → should fall back and report the catalog requirement.
	err := listLanguages(ctx)
	assertError(t, err)
}

func TestListLanguagesFromSettings_NoLanguages(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) {
			return &model.ProjectSettings{
				Language: &model.LanguageSettings{DefaultLanguageCode: "en_US"},
			}, nil
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	ok, err := listLanguagesFromSettings(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected handled=false when no languages registered")
	}
}

func TestAlterLanguageDrop_DefaultLang(t *testing.T) {
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
	stmt := &ast.AlterLanguageStmt{Op: ast.AlterLanguageDrop, Code: "en_US"}
	err := alterLanguage(ctx, stmt)
	if err == nil {
		t.Fatal("expected error when dropping default language")
	}
}

// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/catalog"
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

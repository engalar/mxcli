// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.g3 — page-action adder tests (TDD).
//
// Covers ShowPage / ShowHomePage / ShowMessage / ClosePage. ShowPage
// is the heaviest: nested PageSettings element carrying the page
// qualified name, location enum, and a parameter mappings list whose
// entries live in modelsdk/gen/pages.

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genTexts "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
)

func TestAddShowPageActionGenSetsPageSettings(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.ShowPageStmt{
		PageName: ast.QualifiedName{Module: "Sales", Name: "OrderEditor"},
		Arguments: []ast.ShowPageArg{
			{ParamName: "Order", Value: &ast.VariableExpr{Name: "Ord"}},
		},
	}
	fb.addShowPageActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ShowPageAction)
	ps, ok := act.PageSettings().(*genPg.PageSettings)
	if !ok {
		t.Fatalf("PageSettings = %T, want *PageSettings", act.PageSettings())
	}
	if ps.PageQualifiedName() != "Sales.OrderEditor" {
		t.Fatalf("page QN = %q", ps.PageQualifiedName())
	}
	mappings := ps.ParameterMappingsItems()
	if len(mappings) != 1 {
		t.Fatalf("mappings = %d, want 1", len(mappings))
	}
	pm := mappings[0].(*genPg.PageParameterMapping)
	if pm.ParameterQualifiedName() != "Sales.OrderEditor.Order" {
		t.Fatalf("param QN = %q", pm.ParameterQualifiedName())
	}
	if pm.Argument() != "$Ord" {
		t.Fatalf("param arg = %q", pm.Argument())
	}
}

func TestAddShowPageActionGenLocationDefault(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.ShowPageStmt{
		PageName: ast.QualifiedName{Module: "M", Name: "P"},
	}
	fb.addShowPageActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ShowPageAction)
	ps := act.PageSettings().(*genPg.PageSettings)
	if ps.Location() != "Content" {
		t.Fatalf("location default = %q, want Content", ps.Location())
	}
}

func TestAddShowPageActionGenLocationPopupAndModal(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Popup", "Popup"},
		{"Modal", "Modal"},
		{"Content", "Content"},
		{"", "Content"},
		{"unknown", "Content"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			fb := newActionTestFb()
			stmt := &ast.ShowPageStmt{
				PageName: ast.QualifiedName{Module: "M", Name: "P"},
				Location: tc.in,
			}
			fb.addShowPageActionGen(stmt)
			act := actionFromObjects(t, fb).(*genMf.ShowPageAction)
			ps := act.PageSettings().(*genPg.PageSettings)
			if ps.Location() != tc.want {
				t.Fatalf("location = %q, want %q", ps.Location(), tc.want)
			}
		})
	}
}

func TestAddShowPageActionGenForObjectSetsPassedVariable(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.ShowPageStmt{
		PageName:  ast.QualifiedName{Module: "M", Name: "P"},
		ForObject: "Order",
	}
	fb.addShowPageActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ShowPageAction)
	if act.PassedObjectVariableName() != "Order" {
		t.Fatalf("passed object = %q, want Order (no $ prefix)", act.PassedObjectVariableName())
	}
}

func TestAddShowPageActionGenTitleOverrideSetsTextTemplate(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.ShowPageStmt{
		PageName: ast.QualifiedName{Module: "M", Name: "P"},
		Title:    "Custom Title",
	}
	fb.addShowPageActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ShowPageAction)
	ps := act.PageSettings().(*genPg.PageSettings)
	override, ok := ps.TitleOverride().(*genTexts.Text)
	if !ok {
		t.Fatalf("TitleOverride = %T, want *Text", ps.TitleOverride())
	}
	tr := override.TranslationsItems()
	if len(tr) != 1 {
		t.Fatalf("translations = %d, want 1", len(tr))
	}
	t1 := tr[0].(*genTexts.Translation)
	if t1.Text() != "Custom Title" || t1.LanguageCode() != "en_US" {
		t.Fatalf("translation = (%q, %q), want (Custom Title, en_US)", t1.Text(), t1.LanguageCode())
	}
}

func TestAddShowHomePageActionGenEmpty(t *testing.T) {
	fb := newActionTestFb()
	id := fb.addShowHomePageActionGen(&ast.ShowHomePageStmt{})
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	act := actionFromObjects(t, fb).(*genMf.ShowHomePageAction)
	// Default ErrorHandlingType for microflow = "Rollback"
	if act.ErrorHandlingType() != "Rollback" {
		t.Fatalf("eh = %q, want Rollback", act.ErrorHandlingType())
	}
}

func TestAddShowMessageActionGenSimpleStringTemplate(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.ShowMessageStmt{
		Message: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "Saved"},
	}
	fb.addShowMessageActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ShowMessageAction)
	tmpl, ok := act.Template().(*genMf.TextTemplate)
	if !ok {
		t.Fatalf("template = %T, want *TextTemplate", act.Template())
	}
	textElem := tmpl.Text().(*genTexts.Text)
	tr := textElem.TranslationsItems()
	if len(tr) != 1 {
		t.Fatalf("translations = %d", len(tr))
	}
	t1 := tr[0].(*genTexts.Translation)
	if t1.Text() != "Saved" {
		t.Fatalf("text = %q, want Saved", t1.Text())
	}
}

func TestAddShowMessageActionGenComplexExprUsesPlaceholder(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.ShowMessageStmt{
		Message: &ast.VariableExpr{Name: "Var"},
	}
	fb.addShowMessageActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ShowMessageAction)
	tmpl := act.Template().(*genMf.TextTemplate)
	textElem := tmpl.Text().(*genTexts.Text)
	t1 := textElem.TranslationsItems()[0].(*genTexts.Translation)
	if t1.Text() != "{1}" {
		t.Fatalf("text = %q, want {1}", t1.Text())
	}
	args := tmpl.ArgumentsItems()
	if len(args) != 1 {
		t.Fatalf("args = %d, want 1", len(args))
	}
	a := args[0].(*genMf.TemplateArgument)
	if a.Expression() != "$Var" {
		t.Fatalf("arg = %q", a.Expression())
	}
}

func TestAddShowMessageActionGenTypeDefaultsInformation(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.ShowMessageStmt{
		Message: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "x"},
	}
	fb.addShowMessageActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ShowMessageAction)
	if act.Type() != "Information" {
		t.Fatalf("type default = %q, want Information", act.Type())
	}
}

func TestAddShowMessageActionGenTypeOverride(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.ShowMessageStmt{
		Message: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "x"},
		Type:    "Warning",
	}
	fb.addShowMessageActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ShowMessageAction)
	if act.Type() != "Warning" {
		t.Fatalf("type = %q, want Warning", act.Type())
	}
}

func TestAddClosePageActionGenDefaultsToOnePage(t *testing.T) {
	fb := newActionTestFb()
	fb.addClosePageActionGen(&ast.ClosePageStmt{})
	act := actionFromObjects(t, fb).(*genMf.CloseFormAction)
	if act.NumberOfPages() != 1 {
		t.Fatalf("number of pages default = %d, want 1", act.NumberOfPages())
	}
}

func TestAddClosePageActionGenExplicitN(t *testing.T) {
	fb := newActionTestFb()
	fb.addClosePageActionGen(&ast.ClosePageStmt{NumberOfPages: 3})
	act := actionFromObjects(t, fb).(*genMf.CloseFormAction)
	if act.NumberOfPages() != 3 {
		t.Fatalf("number of pages = %d, want 3", act.NumberOfPages())
	}
}

// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestQuotedIdentifiersInPageWidgets(t *testing.T) {
	input := `CREATE PAGE MaisonElegance."Collection_Overview" (
		Layout: Atlas_Core."Atlas_Default"
	) {
		DATAVIEW dv (DataSource: DATABASE FROM MaisonElegance."Collection") {
			TEXTBOX txtName (Attribute: Name, Label: 'Name')
			ACTIONBUTTON btnEdit (
				Caption: 'Edit',
				Action: SHOW_PAGE MaisonElegance."Collection_NewEdit"
			)
			ACTIONBUTTON btnRun (
				Caption: 'Run',
				Action: MICROFLOW MaisonElegance."ACT_Collection_Run"
			)
		}
		SNIPPETCALL sc (Snippet: MaisonElegance."Footer_Snippet")
	};`

	prog, errs := Build(input)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("Parse error: %v", err)
		}
		t.FailNow()
	}

	if len(prog.Statements) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(prog.Statements))
	}

	stmt, ok := prog.Statements[0].(*ast.CreatePageStmtV3)
	if !ok {
		t.Fatalf("Expected CreatePageStmtV3, got %T", prog.Statements[0])
	}

	// Layout should be unquoted
	if stmt.Layout != "Atlas_Core.Atlas_Default" {
		t.Errorf("Layout: expected 'Atlas_Core.Atlas_Default', got %q", stmt.Layout)
	}

	// Page name should be unquoted
	if stmt.Name.Module != "MaisonElegance" || stmt.Name.Name != "Collection_Overview" {
		t.Errorf("Page name: expected MaisonElegance.Collection_Overview, got %s.%s", stmt.Name.Module, stmt.Name.Name)
	}

	// Find the DataView and check DataSource entity reference is unquoted
	if len(stmt.Widgets) < 1 {
		t.Fatal("Expected at least 1 child widget")
	}
	dv := stmt.Widgets[0]
	if dv.GetDataSource() == nil {
		t.Fatal("DataView DataSource is nil")
	}
	if dv.GetDataSource().Reference != "MaisonElegance.Collection" {
		t.Errorf("DataSource.Reference: expected 'MaisonElegance.Collection', got %q", dv.GetDataSource().Reference)
	}

	// Find SHOW_PAGE action button and check target is unquoted
	btnEdit := findChildByName(dv, "btnEdit")
	if btnEdit == nil {
		t.Fatal("btnEdit widget not found")
	}
	action := btnEdit.GetAction()
	if action == nil {
		t.Fatal("btnEdit Action is nil")
	}
	if action.Target != "MaisonElegance.Collection_NewEdit" {
		t.Errorf("SHOW_PAGE target: expected 'MaisonElegance.Collection_NewEdit', got %q", action.Target)
	}

	// Find MICROFLOW action button and check target is unquoted
	btnRun := findChildByName(dv, "btnRun")
	if btnRun == nil {
		t.Fatal("btnRun widget not found")
	}
	runAction := btnRun.GetAction()
	if runAction == nil {
		t.Fatal("btnRun Action is nil")
	}
	if runAction.Target != "MaisonElegance.ACT_Collection_Run" {
		t.Errorf("MICROFLOW target: expected 'MaisonElegance.ACT_Collection_Run', got %q", runAction.Target)
	}

	// Find SNIPPETCALL and check snippet reference is unquoted
	sc := findChildByName2(stmt.Widgets, "sc")
	if sc == nil {
		t.Fatal("sc (SnippetCall) widget not found")
	}
	snippetRef, ok := sc.Properties["Snippet"].(string)
	if !ok {
		t.Fatal("Snippet property not a string")
	}
	if snippetRef != "MaisonElegance.Footer_Snippet" {
		t.Errorf("Snippet ref: expected 'MaisonElegance.Footer_Snippet', got %q", snippetRef)
	}
}

func findChildByName(parent *ast.WidgetV3, name string) *ast.WidgetV3 {
	for _, c := range parent.Children {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func findChildByName2(widgets []*ast.WidgetV3, name string) *ast.WidgetV3 {
	for _, w := range widgets {
		if w.Name == name {
			return w
		}
		if found := findChildByName(w, name); found != nil {
			return found
		}
	}
	return nil
}

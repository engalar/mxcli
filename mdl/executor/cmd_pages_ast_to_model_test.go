// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/types"
)

// TestPageASTToModel_DataGridColumns verifies that DataGrid's `column` child
// AST widgets are lifted into DataGrid.Columns (not left in the generic
// Children slice), and that Attribute / Caption / Name survive.
func TestPageASTToModel_DataGridColumns(t *testing.T) {
	stmt := &ast.CreatePageStmtV3{
		Name:  ast.QualifiedName{Module: "M", Name: "P"},
		Title: "T",
		Widgets: []*ast.WidgetV3{
			{
				Type: "datagrid",
				Name: "dg",
				Properties: map[string]any{
					"datasource": &ast.DataSourceV3{Type: "database", Reference: "M.E"},
				},
				Children: []*ast.WidgetV3{
					{Type: "column", Name: "c1", Properties: map[string]any{"attribute": "Name", "caption": "N"}},
					{Type: "column", Name: "c2", Properties: map[string]any{"attribute": "Score", "caption": "S"}},
				},
			},
		},
	}
	pm, err := pageASTToModel(nil, stmt)
	if err != nil {
		t.Fatalf("pageASTToModel: %v", err)
	}
	if len(pm.Widgets) != 1 {
		t.Fatalf("expected 1 top widget, got %d", len(pm.Widgets))
	}
	dg := pm.Widgets[0]
	if dg.Kind != types.WidgetDataGrid {
		t.Fatalf("kind = %q, want datagrid", dg.Kind)
	}
	if dg.Children != nil {
		t.Errorf("DataGrid.Children should be nil after lifting columns, got %d", len(dg.Children))
	}
	if dg.DataGrid == nil {
		t.Fatal("expected DataGrid props")
	}
	if got := len(dg.DataGrid.Columns); got != 2 {
		t.Fatalf("columns count = %d, want 2", got)
	}
	if dg.DataGrid.Columns[0].Name != "c1" || dg.DataGrid.Columns[0].Attribute != "Name" || dg.DataGrid.Columns[0].Caption != "N" {
		t.Errorf("column[0] = %+v", dg.DataGrid.Columns[0])
	}
	if dg.DataSource == nil || dg.DataSource.Kind != types.DataSourceDatabase || dg.DataSource.Entity != "M.E" {
		t.Errorf("datasource = %+v", dg.DataSource)
	}
}

// TestPageASTToModel_BasicContainer verifies that a CreatePageStmtV3 with a
// single container holding one button converts to a PageModel with the
// expected widget tree shape (kind + name).
func TestPageASTToModel_BasicContainer(t *testing.T) {
	stmt := &ast.CreatePageStmtV3{
		Name:   ast.QualifiedName{Module: "Mod", Name: "P"},
		Title:  "T",
		Layout: "Atlas_Core.Atlas_Default",
		Widgets: []*ast.WidgetV3{
			{
				Type: "container",
				Name: "mainBox",
				Children: []*ast.WidgetV3{
					{
						Type:       "actionbutton",
						Name:       "btn",
						Properties: map[string]any{"caption": "Click Me"},
					},
				},
			},
		},
	}

	pm, err := pageASTToModel(nil, stmt)
	if err != nil {
		t.Fatalf("pageASTToModel: %v", err)
	}
	if pm == nil {
		t.Fatal("expected non-nil PageModel")
	}
	if pm.ModuleName != "Mod" || pm.Name != "P" {
		t.Errorf("name = %q.%q, want Mod.P", pm.ModuleName, pm.Name)
	}
	if pm.Title != "T" {
		t.Errorf("Title = %q, want T", pm.Title)
	}
	if pm.Layout != "Atlas_Core.Atlas_Default" {
		t.Errorf("Layout = %q, want Atlas_Core.Atlas_Default", pm.Layout)
	}
	if len(pm.Widgets) != 1 {
		t.Fatalf("expected 1 widget, got %d", len(pm.Widgets))
	}
	w := pm.Widgets[0]
	if w.Kind != types.WidgetContainer {
		t.Errorf("widget Kind = %q, want container", w.Kind)
	}
	if w.Name != "mainBox" {
		t.Errorf("widget Name = %q, want mainBox", w.Name)
	}
	if len(w.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(w.Children))
	}
	btn := w.Children[0]
	if btn.Kind != types.WidgetButton {
		t.Errorf("child Kind = %q, want actionbutton/button", btn.Kind)
	}
	if btn.Caption != "Click Me" {
		t.Errorf("child Caption = %q, want Click Me", btn.Caption)
	}
}

func TestPropInt_AutoFill(t *testing.T) {
	w := &ast.WidgetV3{
		Properties: map[string]any{
			"TabletWidth": "AutoFill",
			"PhoneWidth":  "autofill", // lowercase variant
		},
	}
	if got := propInt(w, "tabletwidth"); got != -1 {
		t.Errorf("propInt AutoFill: want -1, got %d", got)
	}
	if got := propInt(w, "phonewidth"); got != -1 {
		t.Errorf("propInt autofill (lowercase): want -1, got %d", got)
	}
}

func TestPropInt_NormalInt(t *testing.T) {
	w := &ast.WidgetV3{
		Properties: map[string]any{"DesktopWidth": 12},
	}
	if got := propInt(w, "desktopwidth"); got != 12 {
		t.Errorf("propInt int: want 12, got %d", got)
	}
}

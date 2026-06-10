// SPDX-License-Identifier: Apache-2.0
package ast_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestCreateLayoutStmt_IsStatement(t *testing.T) {
	s := &ast.CreateLayoutStmt{
		Name:       ast.QualifiedName{Module: "Mod", Name: "Layout1"},
		LayoutType: "Responsive",
		Folder:     "Web/Layouts",
		IsModify:   true,
		Widgets: []*ast.LayoutWidgetV3{
			{
				Kind: ast.LayoutWidgetScrollContainer,
				Name: "sc1",
				Regions: []*ast.LayoutRegionV3{
					{
						Name: "center",
						Placeholders: []*ast.LayoutPlaceholderV3{
							{Name: "Main"},
						},
					},
				},
			},
			{
				Kind:            ast.LayoutWidgetPlaceholder,
				PlaceholderName: "Header",
			},
		},
	}
	if s.Name.Module != "Mod" {
		t.Errorf("Module = %q, want Mod", s.Name.Module)
	}
	if len(s.Widgets) != 2 {
		t.Fatalf("Widgets count = %d, want 2", len(s.Widgets))
	}
	if s.Widgets[0].Kind != ast.LayoutWidgetScrollContainer {
		t.Error("first widget should be ScrollContainer")
	}
	if s.Widgets[1].Kind != ast.LayoutWidgetPlaceholder {
		t.Error("second widget should be Placeholder")
	}
	if s.Widgets[0].Regions[0].Placeholders[0].Name != "Main" {
		t.Errorf("placeholder name = %q, want Main", s.Widgets[0].Regions[0].Placeholders[0].Name)
	}
}

func TestCreateLayoutStmt_SatisfiesStatementInterface(t *testing.T) {
	var _ ast.Statement = (*ast.CreateLayoutStmt)(nil)
}

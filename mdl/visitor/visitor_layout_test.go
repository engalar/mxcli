// SPDX-License-Identifier: Apache-2.0
package visitor_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

func TestVisitor_CreateLayout_BasicParse(t *testing.T) {
	src := `create or modify layout Atlas_Core.MyLayout (
  type: Responsive,
  folder: 'Web/Layouts'
) {
  scrollcontainer sc1 {
    center { placeholder Main }
    top    { placeholder Header }
  }
}`
	prog, errs := visitor.Build(src)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}
	s, ok := prog.Statements[0].(*ast.CreateLayoutStmt)
	if !ok {
		t.Fatalf("expected *ast.CreateLayoutStmt, got %T", prog.Statements[0])
	}
	if s.Name.Module != "Atlas_Core" || s.Name.Name != "MyLayout" {
		t.Errorf("Name = %v, want Atlas_Core.MyLayout", s.Name)
	}
	if s.LayoutType != "Responsive" {
		t.Errorf("LayoutType = %q, want Responsive", s.LayoutType)
	}
	if s.Folder != "Web/Layouts" {
		t.Errorf("Folder = %q, want Web/Layouts", s.Folder)
	}
	if !s.IsModify {
		t.Error("IsModify should be true for 'create or modify'")
	}
	if len(s.Widgets) != 1 {
		t.Fatalf("expected 1 widget, got %d", len(s.Widgets))
	}
	w := s.Widgets[0]
	if w.Kind != ast.LayoutWidgetScrollContainer || w.Name != "sc1" {
		t.Errorf("widget = {%v, %q}, want {ScrollContainer, sc1}", w.Kind, w.Name)
	}
	if len(w.Regions) != 2 {
		t.Fatalf("expected 2 regions, got %d", len(w.Regions))
	}
	if w.Regions[0].Name != "center" || len(w.Regions[0].Placeholders) != 1 {
		t.Errorf("center region unexpected: %+v", w.Regions[0])
	}
	if w.Regions[0].Placeholders[0].Name != "Main" {
		t.Errorf("placeholder name = %q, want Main", w.Regions[0].Placeholders[0].Name)
	}
}

func TestVisitor_CreateLayout_NativePlaceholder(t *testing.T) {
	src := `create layout Mod.NativeLayout (type: Default) {
  placeholder Content
}`
	prog, errs := visitor.Build(src)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	s, ok := prog.Statements[0].(*ast.CreateLayoutStmt)
	if !ok {
		t.Fatalf("expected *ast.CreateLayoutStmt, got %T", prog.Statements[0])
	}
	if len(s.Widgets) != 1 {
		t.Fatalf("expected 1 widget, got %d", len(s.Widgets))
	}
	if s.Widgets[0].Kind != ast.LayoutWidgetPlaceholder {
		t.Errorf("expected LayoutWidgetPlaceholder, got %v", s.Widgets[0].Kind)
	}
	if s.Widgets[0].PlaceholderName != "Content" {
		t.Errorf("PlaceholderName = %q, want Content", s.Widgets[0].PlaceholderName)
	}
}

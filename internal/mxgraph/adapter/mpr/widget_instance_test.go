package mpr

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func TestWidgetInstanceAdapter_Name(t *testing.T) {
	t.Parallel()
	a := &WidgetInstanceAdapter{}
	if a.Name() != "widgetinstance" {
		t.Errorf("Name() = %q, want widgetinstance", a.Name())
	}
}

func TestWidgetInstanceAdapter_Schema(t *testing.T) {
	t.Parallel()
	a := &WidgetInstanceAdapter{}
	s := a.Schema()
	if s == nil {
		t.Fatal("Schema() returned nil")
	}
	labels := map[mxgraph.Label]bool{}
	for _, l := range s.NodeLabels {
		labels[l] = true
	}
	if !labels["WidgetInstance"] {
		t.Errorf("Schema missing label WidgetInstance")
	}
	rels := map[mxgraph.RelType]bool{}
	for _, et := range s.EdgeTypes {
		rels[et.Type] = true
	}
	if !rels["HAS_WIDGET_INSTANCE"] {
		t.Errorf("Schema missing edge type HAS_WIDGET_INSTANCE")
	}
}

func TestIsWidgetType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		typeName string
		want     bool
	}{
		{"Forms$DivContainer", true},
		{"Forms$ActionButton", true},
		{"Forms$TextBox", true},
		{"Forms$DataGrid", true},
		{"Forms$LayoutCall", false},
		{"Forms$FormCall", false},
		{"Forms$Appearance", false},
		{"Forms$DesignPropertyValue", false},
		{"DomainModels$Entity", false},
	}
	for _, tt := range tests {
		got := isWidgetType(tt.typeName)
		if got != tt.want {
			t.Errorf("isWidgetType(%q) = %v, want %v", tt.typeName, got, tt.want)
		}
	}
}

func TestShortTypeName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"Forms$DivContainer", "DivContainer"},
		{"Forms$ActionButton", "ActionButton"},
		{"Pages$TextBox", "TextBox"},
		{"Unknown", "Unknown"},
	}
	for _, tt := range tests {
		got := shortTypeName(tt.input)
		if got != tt.want {
			t.Errorf("shortTypeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

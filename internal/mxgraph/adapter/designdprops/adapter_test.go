package designdprops

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

type recordingSink struct {
	events []mxgraph.Event
}

func (s *recordingSink) Emit(events []mxgraph.Event) error {
	s.events = append(s.events, events...)
	return nil
}

func TestDesignPropertyAdapter_Name(t *testing.T) {
	a := &DesignPropertyAdapter{}
	if a.Name() != "designdprops" {
		t.Errorf("Name() = %q, want designdprops", a.Name())
	}
}

func TestDesignPropertyAdapter_Schema(t *testing.T) {
	a := &DesignPropertyAdapter{}
	s := a.Schema()
	if s == nil {
		t.Fatal("Schema() returned nil")
	}
	found := false
	for _, l := range s.NodeLabels {
		if l == "DesignProperty" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Schema missing label DesignProperty")
	}
}

func TestDesignPropertyAdapter_Build(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "themesource", "atlas_core", "web"), 0700)
	os.MkdirAll(filepath.Join(dir, "themesource", "datawidgets", "web"), 0700)

	os.WriteFile(filepath.Join(dir, "themesource", "atlas_core", "web", "design-properties.json"), []byte(`{
	"DivContainer": [
		{
			"name": "Card style",
			"type": "Toggle",
			"description": "Render container as card",
			"class": "card"
		},
		{
			"name": "Background color",
			"type": "ColorPicker",
			"category": "Appearance",
			"options": [
				{ "name": "Brand Primary", "preview": "--brand-primary", "class": "background-primary" },
				{ "name": "Brand Success", "preview": "--brand-success", "class": "background-success" }
			]
		}
	],
	"Widget": [
		{
			"name": "Spacing",
			"type": "Spacing",
			"margin": [ { "name": "None", "top": { "class": "spacing-outer-top-none" } } ],
			"padding": [ { "name": "None", "top": { "class": "spacing-inner-top-none" } } ]
		}
	]
}`), 0600)

	os.WriteFile(filepath.Join(dir, "themesource", "datawidgets", "web", "design-properties.json"), []byte(`{
	"DataGrid": [
		{
			"name": "Striped rows",
			"type": "Toggle",
			"class": "datagrid-striped"
		}
	]
}`), 0600)

	a := &DesignPropertyAdapter{ProjectDir: dir}
	sink := &recordingSink{}
	err := a.Build(context.Background(), sink)
	if err != nil {
		t.Fatal(err)
	}

	var nodeCreated int
	for _, e := range sink.events {
		if e.Type == mxgraph.NodeCreated {
			nodeCreated++
		}
	}
	// Card style, Background color, Spacing, Striped rows = 4
	if nodeCreated != 4 {
		t.Fatalf("NodeCreated events = %d, want 4", nodeCreated)
	}

	// Verify Background color has referenced vars
	found := false
	for _, e := range sink.events {
		if e.Node != nil && e.Node.Props["Name"] == "Background color" {
			vars, ok := e.Node.Props["ReferencedVars"].([]string)
			if !ok || len(vars) == 0 {
				t.Errorf("Background color should have ReferencedVars, got %#v", e.Node.Props["ReferencedVars"])
			} else {
				found = true
			}
			break
		}
	}
	if !found {
		t.Error("Background color DesignProperty node not found")
	}
}

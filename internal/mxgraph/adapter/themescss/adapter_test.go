package themescss

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

func TestThemeScssAdapter_Name(t *testing.T) {
	a := &ThemeScssAdapter{}
	if a.Name() != "themescss" {
		t.Errorf("Name() = %q, want themescss", a.Name())
	}
}

func TestThemeScssAdapter_Schema(t *testing.T) {
	a := &ThemeScssAdapter{}
	s := a.Schema()
	if s == nil {
		t.Fatal("Schema() returned nil")
	}
	found := false
	for _, l := range s.NodeLabels {
		if l == "ThemeVariable" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Schema missing label ThemeVariable")
	}
}

func TestThemeScssAdapter_Build(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "theme", "web"), 0700)
	os.MkdirAll(filepath.Join(dir, "themesource", "demo", "web"), 0700)

	// project-level custom-variables.scss
	os.WriteFile(filepath.Join(dir, "theme", "web", "custom-variables.scss"), []byte(`
$brand-logo: false;
$use-css-variables: true;
:root {
  --brand-primary: #264ae5;
  --brand-success: #16aa16;
  // --brand-warning: #cd8501;
}
@import "../../themesource/atlas_core/web/variables";
`), 0600)

	// project-level main.scss
	os.WriteFile(filepath.Join(dir, "theme", "web", "main.scss"), []byte(`
$brand-primary: #1565C0;
$brand-primary-dark: #0D47A1;
`), 0600)

	// module-level variables.scss
	os.WriteFile(filepath.Join(dir, "themesource", "demo", "web", "variables.scss"), []byte(`
$brand-primary: #264ae5 !default;
$bg-color: #fff !default;
`), 0600)

	a := &ThemeScssAdapter{ProjectDir: dir}
	sink := &recordingSink{}
	err := a.Build(context.Background(), sink)
	if err != nil {
		t.Fatal(err)
	}

	if len(sink.events) == 0 {
		t.Fatal("expected at least 1 event")
	}

	var nodeCreated int
	for _, e := range sink.events {
		if e.Type == mxgraph.NodeCreated && e.Node != nil {
			nodeCreated++
		}
	}
	// custom-variables: $brand-logo, $use-css-variables, --brand-primary, --brand-success, --brand-warning(commented) = 5
	// main.scss: $brand-primary, $brand-primary-dark = 2
	// demo/variables.scss: $brand-primary(default), $bg-color(default) = 2
	// Total: 9
	if nodeCreated != 9 {
		t.Fatalf("NodeCreated events = %d, want 9", nodeCreated)
	}

	// Verify main.scss $brand-primary has value #1565C0
	found := false
	for _, e := range sink.events {
		if e.Node != nil && e.Node.Props["Name"] == "$brand-primary" {
			source, _ := e.Node.Props["Source"].(string)
			val, _ := e.Node.Props["Value"].(string)
			if source == "project-main" && val == "#1565C0" {
				found = true
			}
		}
	}
	if !found {
		t.Error("$brand-primary from project-main with value #1565C0 not found")
	}

	// Verify custom-variables.scss --brand-primary has value #264ae5
	found = false
	for _, e := range sink.events {
		if e.Node != nil && e.Node.Props["Name"] == "--brand-primary" {
			source, _ := e.Node.Props["Source"].(string)
			if source == "project-custom-variables" {
				found = true
			}
		}
	}
	if !found {
		t.Error("--brand-primary from project-custom-variables not found")
	}

	// Verify commented var is inactive
	for _, e := range sink.events {
		if e.Node != nil && e.Node.Props["Name"] == "--brand-warning" {
			active, _ := e.Node.Props["IsActive"].(bool)
			if active {
				t.Error("--brand-warning should be inactive (commented)")
			}
		}
	}
}

func TestThemeScssAdapter_Build_NoFiles(t *testing.T) {
	dir := t.TempDir() // empty
	a := &ThemeScssAdapter{ProjectDir: dir}
	sink := &recordingSink{}
	err := a.Build(context.Background(), sink)
	if err != nil {
		t.Fatal(err)
	}
	if len(sink.events) != 0 {
		t.Errorf("expected 0 events for empty project, got %d", len(sink.events))
	}
}

// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/sdk/widgets/mpk"
)

// writeMiniMPK creates a minimal .mpk ZIP in widgetsDir with the given widget ID.
func writeMiniMPK(t *testing.T, widgetsDir, widgetID string) {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// last segment of ID becomes the XML file name
	parts := strings.Split(widgetID, ".")
	xmlName := parts[len(parts)-1] + ".xml"

	pkg, _ := w.Create("package.xml")
	fmt.Fprintf(pkg,
		`<package><clientModule name="Test" version="1.0.0"><widgetFiles><widgetFile path=%q/></widgetFiles></clientModule></package>`,
		xmlName,
	)

	wxml, _ := w.Create(xmlName)
	fmt.Fprintf(wxml,
		`<widget id=%q pluginWidget="true"><name>Test</name><properties><propertyGroup caption="General"><property key="caption" type="string" defaultValue="Hi"/></propertyGroup></properties></widget>`,
		widgetID,
	)

	w.Close()

	mpkPath := filepath.Join(widgetsDir, parts[len(parts)-1]+".mpk")
	if err := os.WriteFile(mpkPath, buf.Bytes(), 0644); err != nil {
		t.Fatalf("write MPK: %v", err)
	}
}

func TestRegistryMPKFallbackGet(t *testing.T) {
	mpk.ClearCache()
	dir := t.TempDir()
	widgetsDir := filepath.Join(dir, "widgets")
	os.MkdirAll(widgetsDir, 0755)
	writeMiniMPK(t, widgetsDir, "com.test.widget.Testwidget")

	reg, err := NewWidgetRegistry()
	if err != nil {
		t.Fatalf("NewWidgetRegistry: %v", err)
	}
	if err := reg.SetProjectDir(dir); err != nil {
		t.Fatalf("SetProjectDir: %v", err)
	}

	def, ok := reg.Get("TESTWIDGET")
	if !ok {
		t.Fatal("expected TESTWIDGET via MPK fallback, got not-found")
	}
	if def.WidgetID != "com.test.widget.Testwidget" {
		t.Errorf("WidgetID = %q, want com.test.widget.Testwidget", def.WidgetID)
	}
	if def.WidgetKind != "pluggable" {
		t.Errorf("WidgetKind = %q, want pluggable", def.WidgetKind)
	}
}

func TestRegistryMPKFallbackGetByWidgetID(t *testing.T) {
	mpk.ClearCache()
	dir := t.TempDir()
	widgetsDir := filepath.Join(dir, "widgets")
	os.MkdirAll(widgetsDir, 0755)
	writeMiniMPK(t, widgetsDir, "com.test.widget.Testwidget")

	reg, _ := NewWidgetRegistry()
	reg.SetProjectDir(dir)

	def, ok := reg.GetByWidgetID("com.test.widget.Testwidget")
	if !ok {
		t.Fatal("expected GetByWidgetID fallback to find widget")
	}
	if def.MDLName != "testwidget" {
		t.Errorf("MDLName = %q, want testwidget", def.MDLName)
	}
}

func TestRegistryMPKFallbackCached(t *testing.T) {
	mpk.ClearCache()
	dir := t.TempDir()
	widgetsDir := filepath.Join(dir, "widgets")
	os.MkdirAll(widgetsDir, 0755)
	writeMiniMPK(t, widgetsDir, "com.test.widget.Testwidget")

	reg, _ := NewWidgetRegistry()
	reg.SetProjectDir(dir)

	def1, ok1 := reg.Get("TESTWIDGET")
	if !ok1 {
		t.Fatal("first lookup failed")
	}
	def2, ok2 := reg.Get("TESTWIDGET")
	if !ok2 {
		t.Fatal("second lookup failed")
	}
	if def1 != def2 {
		t.Error("expected same pointer on second lookup (in-registry cache)")
	}
}

func TestRegistryMPKFallbackSkipsBuiltins(t *testing.T) {
	mpk.ClearCache()
	dir := t.TempDir()
	widgetsDir := filepath.Join(dir, "widgets")
	os.MkdirAll(widgetsDir, 0755)
	// Write a fake MPK that would derive "GALLERY" — should be ignored
	writeMiniMPK(t, widgetsDir, "com.mendix.widget.web.gallery.Gallery")

	reg, _ := NewWidgetRegistry()
	builtinDef, builtinOK := reg.Get("GALLERY")

	reg.SetProjectDir(dir)

	afterDef, afterOK := reg.Get("GALLERY")
	if builtinOK {
		// Gallery is a builtin: SetProjectDir must not replace it
		if afterDef != builtinDef {
			t.Error("SetProjectDir must not override a built-in definition")
		}
	} else {
		// Gallery not a builtin in this build: MPK fallback is fine
		if !afterOK {
			t.Error("expected GALLERY to be found after SetProjectDir")
		}
	}
}

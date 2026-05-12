// SPDX-License-Identifier: Apache-2.0
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePropertySpec(t *testing.T) {
	tests := []struct {
		input   string
		wantKey string
		wantXML string
		wantSub string
		wantErr bool
	}{
		{"value:attribute:Decimal", "value", "attribute", "Decimal", false},
		{"label:string", "label", "string", "", false},
		{"onChange:action", "onChange", "action", "", false},
		{"data:datasource", "data", "datasource", "", false},
		{"slot:widgets", "slot", "widgets", "", false},
		{"count:integer", "count", "integer", "", false},
		{"visible:boolean", "visible", "boolean", "", false},
		{"expr:expression", "expr", "expression", "", false},
		{"value:attribute:String", "value", "attribute", "String", false},
		{"bad", "", "", "", true},
		{"key:unknown", "", "", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			spec, err := parsePropertySpec(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parsePropertySpec(%q) expected error, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePropertySpec(%q) unexpected error: %v", tc.input, err)
			}
			if spec.Key != tc.wantKey || spec.XMLType != tc.wantXML || spec.Subtype != tc.wantSub {
				t.Errorf("got {%q %q %q}, want {%q %q %q}",
					spec.Key, spec.XMLType, spec.Subtype,
					tc.wantKey, tc.wantXML, tc.wantSub)
			}
		})
	}
}

func TestDeriveWidgetID(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"MySlider", "com.mendix.widget.custom.MySlider.MySlider"},
		{"CrusherSlider", "com.mendix.widget.custom.CrusherSlider.CrusherSlider"},
		{"Foo", "com.mendix.widget.custom.Foo.Foo"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveWidgetID(tc.name)
			if got != tc.want {
				t.Errorf("deriveWidgetID(%q) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestHumanizeWidgetName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"MySlider", "My Slider"},
		{"CrusherSimCanvas", "Crusher Sim Canvas"},
		{"Foo", "Foo"},
		{"HeatmapViz", "Heatmap Viz"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := humanizeWidgetName(tc.input)
			if got != tc.want {
				t.Errorf("humanizeWidgetName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestGenerateWidgetXML(t *testing.T) {
	props := []PropertySpec{
		{Key: "value", XMLType: "attribute", Subtype: "Decimal"},
		{Key: "label", XMLType: "string"},
		{Key: "onChange", XMLType: "action"},
	}
	xml := generateWidgetXML("MySlider", "com.acme.widget.MySlider.MySlider", false, props)

	checks := []string{
		`id="com.acme.widget.MySlider.MySlider"`,
		`pluginWidget="true"`,
		`offlineCapable="false"`,
		`<name>My Slider</name>`,
		`key="value" type="attribute"`,
		`<attributeType name="Decimal"/>`,
		`key="label" type="string"`,
		`key="onChange" type="action"`,
	}
	for _, want := range checks {
		if !strings.Contains(xml, want) {
			t.Errorf("generateWidgetXML: missing %q\ngot:\n%s", want, xml)
		}
	}
}

func TestGenerateWidgetXML_Offline(t *testing.T) {
	xml := generateWidgetXML("Foo", "com.a.b.c.Foo.Foo", true, nil)
	if !strings.Contains(xml, `offlineCapable="true"`) {
		t.Errorf("expected offlineCapable=true, got:\n%s", xml)
	}
}

func TestGeneratePackageXML(t *testing.T) {
	xml := generatePackageXML("MyPkg", []string{"WidgetA", "WidgetB"})
	checks := []string{
		`name="MyPkg"`,
		`version="1.0.0"`,
		`<widgetFile path="WidgetA.xml"/>`,
		`<widgetFile path="WidgetB.xml"/>`,
	}
	for _, want := range checks {
		if !strings.Contains(xml, want) {
			t.Errorf("generatePackageXML: missing %q\ngot:\n%s", want, xml)
		}
	}
}

func TestGenerateJSX(t *testing.T) {
	props := []PropertySpec{
		{Key: "value", XMLType: "attribute"},
		{Key: "label", XMLType: "string"},
		{Key: "onChange", XMLType: "action"},
	}
	jsx := generateJSX("MySlider", props)
	checks := []string{
		`export function MySlider`,
		`{ value, label, onChange }`,
		`export default MySlider`,
		`createElement`,
	}
	for _, want := range checks {
		if !strings.Contains(jsx, want) {
			t.Errorf("generateJSX: missing %q\ngot:\n%s", want, jsx)
		}
	}
}

func TestGenerateJSX_NoProps(t *testing.T) {
	jsx := generateJSX("Foo", nil)
	if !strings.Contains(jsx, "export function Foo(") {
		t.Errorf("generateJSX (no props): missing function signature\ngot:\n%s", jsx)
	}
}

func TestGenerateEditorConfig(t *testing.T) {
	props := []PropertySpec{{Key: "label", XMLType: "string"}}
	js := generateEditorConfig("MySlider", props)
	if !strings.Contains(js, "getCustomCaption") || !strings.Contains(js, "getPreview") {
		t.Errorf("generateEditorConfig: missing functions\ngot:\n%s", js)
	}
	if !strings.Contains(js, "MySlider") {
		t.Errorf("generateEditorConfig: missing widget name\ngot:\n%s", js)
	}
}

func TestGeneratePackageJSON(t *testing.T) {
	json := generatePackageJSON("my-slider")
	if !strings.Contains(json, `"esbuild"`) {
		t.Errorf("generatePackageJSON: missing esbuild\ngot:\n%s", json)
	}
	if !strings.Contains(json, `"name": "my-slider"`) {
		t.Errorf("generatePackageJSON: missing name\ngot:\n%s", json)
	}
}

func TestScaffoldWidget_CreatesExpectedFiles(t *testing.T) {
	dir := t.TempDir()
	props := []PropertySpec{
		{Key: "value", XMLType: "attribute", Subtype: "Decimal"},
		{Key: "label", XMLType: "string"},
	}
	err := scaffoldWidget(dir, "MySlider", "com.acme.widget.MySlider.MySlider", false, props)
	if err != nil {
		t.Fatalf("scaffoldWidget: %v", err)
	}
	wantFiles := []string{
		"src/MySlider.xml",
		"src/MySlider.jsx",
		"src/MySlider.editorConfig.js",
		"src/MySlider.editorPreview.js",
		"src/MySlider.icon.png",
		"src/MySlider.icon.dark.png",
		"src/MySlider.tile.png",
		"src/MySlider.tile.dark.png",
	}
	for _, rel := range wantFiles {
		full := filepath.Join(dir, rel)
		if _, err := os.Stat(full); err != nil {
			t.Errorf("expected file %s to exist: %v", rel, err)
		}
	}
	xmlBytes, _ := os.ReadFile(filepath.Join(dir, "src/MySlider.xml"))
	if !strings.Contains(string(xmlBytes), "com.acme.widget.MySlider.MySlider") {
		t.Errorf("MySlider.xml missing widget ID")
	}
}

func TestScaffoldSingleWidget_TopLevelFiles(t *testing.T) {
	dir := t.TempDir()
	props := []PropertySpec{{Key: "label", XMLType: "string"}}
	if err := scaffoldWidget(dir, "MySlider", deriveWidgetID("MySlider"), false, props); err != nil {
		t.Fatalf("scaffoldWidget: %v", err)
	}
	pkgJSON := filepath.Join(dir, "package.json")
	pkgXML := filepath.Join(dir, "package.xml")
	if err := os.WriteFile(pkgJSON, []byte(generatePackageJSON("my-slider")), 0644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := os.WriteFile(pkgXML, []byte(generatePackageXML("MySlider", []string{"MySlider"})), 0644); err != nil {
		t.Fatalf("write package.xml: %v", err)
	}
	for _, rel := range []string{"package.json", "package.xml", "src/MySlider.xml", "src/MySlider.jsx"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("expected %s to exist: %v", rel, err)
		}
	}
}

func TestScaffoldPackage_EmptySrc(t *testing.T) {
	dir := t.TempDir()
	if err := scaffoldPackage(dir, "CrusherWidgets"); err != nil {
		t.Fatalf("scaffoldPackage: %v", err)
	}
	for _, rel := range []string{"package.json", "package.xml", "src"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("expected %s to exist: %v", rel, err)
		}
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "src"))
	if len(entries) != 0 {
		t.Errorf("src/ must be empty for package scaffold, got %d entries", len(entries))
	}
	xmlBytes, _ := os.ReadFile(filepath.Join(dir, "package.xml"))
	if strings.Contains(string(xmlBytes), "<widgetFile") {
		t.Errorf("package.xml must not list widget files yet\ngot:\n%s", string(xmlBytes))
	}
}

func TestAppendWidgetFileToPackageXML(t *testing.T) {
	dir := t.TempDir()
	initial := generatePackageXML("MyPkg", []string{"WidgetA"})
	pkgXML := filepath.Join(dir, "package.xml")
	os.WriteFile(pkgXML, []byte(initial), 0644)

	if err := appendWidgetFileToPackageXML(pkgXML, "WidgetB"); err != nil {
		t.Fatalf("appendWidgetFileToPackageXML: %v", err)
	}

	data, _ := os.ReadFile(pkgXML)
	content := string(data)
	if !strings.Contains(content, `path="WidgetA.xml"`) {
		t.Errorf("WidgetA entry must still exist\ngot:\n%s", content)
	}
	if !strings.Contains(content, `path="WidgetB.xml"`) {
		t.Errorf("WidgetB entry must be added\ngot:\n%s", content)
	}
}

func TestAppendWidgetFileToPackageXML_NoDuplicate(t *testing.T) {
	dir := t.TempDir()
	initial := generatePackageXML("MyPkg", []string{"WidgetA"})
	pkgXML := filepath.Join(dir, "package.xml")
	os.WriteFile(pkgXML, []byte(initial), 0644)

	_ = appendWidgetFileToPackageXML(pkgXML, "WidgetA")
	data, _ := os.ReadFile(pkgXML)
	count := strings.Count(string(data), `path="WidgetA.xml"`)
	if count != 1 {
		t.Errorf("expected 1 WidgetA entry, got %d\ncontent:\n%s", count, string(data))
	}
}

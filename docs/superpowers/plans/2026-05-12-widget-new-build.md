# Widget Scaffold & Build Pipeline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `mxcli widget new`, `mxcli widget add-widget`, and `mxcli widget build` commands so developers can scaffold and build Mendix pluggable widget projects without hand-writing `build.sh`.

**Architecture:** Two new files in `cmd/mxcli/` — `widget_scaffold.go` (pure template functions + `new`/`add-widget` handlers) and `widget_build.go` (build pipeline: discover/validate/compile/package/verify). Template content lives as Go `const` strings rendered with `fmt.Sprintf`; placeholder PNG images are generated in-memory with `image/png`. `cmd_widget.go` gains three new subcommand registrations.

**Tech Stack:** Go stdlib (`encoding/xml`, `archive/zip`, `image/png`, `os/exec`), `github.com/spf13/cobra`, reuses `sdk/widgets/mpk` for package.xml parsing during build.

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `cmd/mxcli/widget_scaffold.go` | Create | `PropertySpec` type, template generators, `scaffoldWidget`, `scaffoldPackage`, `runWidgetNew`, `runWidgetAddWidget` |
| `cmd/mxcli/widget_scaffold_test.go` | Create | Unit tests for all pure functions in widget_scaffold.go |
| `cmd/mxcli/widget_build.go` | Create | `widgetInfo` type, `discoverWidgets`, `validateWidgetInfo`, `detectToolchain`, `installDeps`, `compileWidget`, `copyAssets`, `packageMPK`, `verifyMPK`, `runWidgetBuild` |
| `cmd/mxcli/widget_build_test.go` | Create | Unit tests for discover and validate |
| `cmd/mxcli/cmd_widget.go` | Modify | Register `widgetNewCmd`, `widgetAddWidgetCmd`, `widgetBuildCmd` in `init()` |

---

## Task 1: Property Spec Parsing + Widget ID Helpers

**Files:**
- Create: `cmd/mxcli/widget_scaffold.go`
- Create: `cmd/mxcli/widget_scaffold_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// cmd/mxcli/widget_scaffold_test.go
// SPDX-License-Identifier: Apache-2.0
package main

import (
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./cmd/mxcli/ -run "TestParsePropertySpec|TestDeriveWidgetID|TestHumanizeWidgetName" -v 2>&1 | head -20
```
Expected: FAIL — `parsePropertySpec`, `deriveWidgetID`, `humanizeWidgetName` undefined.

- [ ] **Step 3: Implement the pure functions**

Create `cmd/mxcli/widget_scaffold.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"strings"
	"unicode"
)

// PropertySpec represents one parsed --property flag value (key:type[:subtype]).
type PropertySpec struct {
	Key     string // e.g. "value"
	XMLType string // e.g. "attribute", "string", "action"
	Subtype string // e.g. "Decimal" — only for attribute types
}

// validXMLTypes is the set of XML property types the scaffold understands.
var validXMLTypes = map[string]bool{
	"attribute":  true,
	"string":     true,
	"integer":    true,
	"boolean":    true,
	"action":     true,
	"datasource": true,
	"expression": true,
	"widgets":    true,
}

// parsePropertySpec parses a --property flag value of the form key:type[:subtype].
func parsePropertySpec(s string) (PropertySpec, error) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) < 2 {
		return PropertySpec{}, fmt.Errorf("invalid property spec %q: must be key:type or key:type:subtype", s)
	}
	key, xmlType := parts[0], parts[1]
	if !validXMLTypes[xmlType] {
		return PropertySpec{}, fmt.Errorf("invalid property type %q in %q: must be one of attribute, string, integer, boolean, action, datasource, expression, widgets", xmlType, s)
	}
	subtype := ""
	if len(parts) == 3 {
		subtype = parts[2]
	}
	return PropertySpec{Key: key, XMLType: xmlType, Subtype: subtype}, nil
}

// deriveWidgetID returns the default widget ID for a widget named name.
// e.g. "MySlider" → "com.mendix.widget.custom.MySlider.MySlider"
func deriveWidgetID(name string) string {
	return fmt.Sprintf("com.mendix.widget.custom.%s.%s", name, name)
}

// humanizeWidgetName inserts a space before each uppercase letter (after the first).
// e.g. "MySlider" → "My Slider", "CrusherSimCanvas" → "Crusher Sim Canvas"
func humanizeWidgetName(name string) string {
	var b strings.Builder
	for i, r := range name {
		if i > 0 && unicode.IsUpper(r) {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./cmd/mxcli/ -run "TestParsePropertySpec|TestDeriveWidgetID|TestHumanizeWidgetName" -v
```
Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli/widget_scaffold.go cmd/mxcli/widget_scaffold_test.go
git commit -m "feat(widget): PropertySpec parser + widget ID / name helpers"
```

---

## Task 2: XML Template Generation

**Files:**
- Modify: `cmd/mxcli/widget_scaffold.go` (add generateWidgetXML, generatePackageXML)
- Modify: `cmd/mxcli/widget_scaffold_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `widget_scaffold_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./cmd/mxcli/ -run "TestGenerateWidgetXML|TestGeneratePackageXML" -v 2>&1 | head -10
```
Expected: FAIL — functions undefined.

- [ ] **Step 3: Implement the XML generators**

Add to `cmd/mxcli/widget_scaffold.go`:

```go
// generateWidgetXML renders the widget property-definition XML for src/<Name>.xml.
func generateWidgetXML(name, widgetID string, offline bool, props []PropertySpec) string {
	human := humanizeWidgetName(name)
	offlineStr := "false"
	if offline {
		offlineStr = "true"
	}

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	b.WriteString(fmt.Sprintf(
		`<widget id=%q pluginWidget="true" offlineCapable=%q`+"\n"+
			`        xmlns="http://www.mendix.com/widget/1.0/"`+"\n"+
			`        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`+"\n"+
			`        xsi:schemaLocation="http://www.mendix.com/widget/1.0/ ../../../../node_modules/mendix/custom_widget.xsd">`+"\n",
		widgetID, offlineStr))
	b.WriteString(fmt.Sprintf("  <name>%s</name>\n", human))
	b.WriteString("  <description></description>\n")
	b.WriteString("  <properties>\n")
	b.WriteString("    <propertyGroup caption=\"General\">\n")
	for _, p := range props {
		b.WriteString(renderPropertyXML(p))
	}
	b.WriteString("    </propertyGroup>\n")
	b.WriteString("  </properties>\n")
	b.WriteString("</widget>\n")
	return b.String()
}

// renderPropertyXML renders a single <property> element for the widget XML.
func renderPropertyXML(p PropertySpec) string {
	var b strings.Builder
	human := humanizeWidgetName(p.Key)
	switch p.XMLType {
	case "attribute":
		attrType := p.Subtype
		if attrType == "" {
			attrType = "String"
		}
		b.WriteString(fmt.Sprintf(
			"      <property key=%q type=\"attribute\" required=\"true\">\n"+
				"        <caption>%s</caption>\n"+
				"        <description/>\n"+
				"        <attributeTypes><attributeType name=%q/></attributeTypes>\n"+
				"      </property>\n",
			p.Key, human, attrType))
	case "string":
		b.WriteString(fmt.Sprintf(
			"      <property key=%q type=\"string\" required=\"true\" defaultValue=\"\">\n"+
				"        <caption>%s</caption>\n"+
				"        <description/>\n"+
				"      </property>\n",
			p.Key, human))
	case "integer":
		b.WriteString(fmt.Sprintf(
			"      <property key=%q type=\"integer\" required=\"true\" defaultValue=\"0\">\n"+
				"        <caption>%s</caption>\n"+
				"        <description/>\n"+
				"      </property>\n",
			p.Key, human))
	case "boolean":
		b.WriteString(fmt.Sprintf(
			"      <property key=%q type=\"boolean\" required=\"true\" defaultValue=\"false\">\n"+
				"        <caption>%s</caption>\n"+
				"        <description/>\n"+
				"      </property>\n",
			p.Key, human))
	case "action":
		b.WriteString(fmt.Sprintf(
			"      <property key=%q type=\"action\" required=\"true\">\n"+
				"        <caption>%s</caption>\n"+
				"        <description/>\n"+
				"      </property>\n",
			p.Key, human))
	case "datasource":
		b.WriteString(fmt.Sprintf(
			"      <property key=%q type=\"datasource\" required=\"true\" isList=\"true\">\n"+
				"        <caption>%s</caption>\n"+
				"        <description/>\n"+
				"      </property>\n",
			p.Key, human))
	case "expression":
		b.WriteString(fmt.Sprintf(
			"      <property key=%q type=\"expression\" required=\"true\" defaultValue=\"true\">\n"+
				"        <caption>%s</caption>\n"+
				"        <description/>\n"+
				"        <returnType type=\"Boolean\"/>\n"+
				"      </property>\n",
			p.Key, human))
	case "widgets":
		b.WriteString(fmt.Sprintf(
			"      <property key=%q type=\"widgets\" required=\"false\">\n"+
				"        <caption>%s</caption>\n"+
				"        <description/>\n"+
				"      </property>\n",
			p.Key, human))
	}
	return b.String()
}

// generatePackageXML renders package.xml — the MPK manifest listing all widget XML files.
func generatePackageXML(packageName string, widgetNames []string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	b.WriteString(`<package xmlns="http://www.mendix.com/package/1.0/">` + "\n")
	b.WriteString(fmt.Sprintf(
		`  <clientModule name=%q version="1.0.0" xmlns="http://www.mendix.com/clientmodule/1.0/">`+"\n",
		packageName))
	b.WriteString("    <widgetFiles>\n")
	for _, name := range widgetNames {
		b.WriteString(fmt.Sprintf("      <widgetFile path=%q/>\n", name+".xml"))
	}
	b.WriteString("    </widgetFiles>\n")
	b.WriteString("  </clientModule>\n")
	b.WriteString("</package>\n")
	return b.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./cmd/mxcli/ -run "TestGenerateWidgetXML|TestGeneratePackageXML" -v
```
Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli/widget_scaffold.go cmd/mxcli/widget_scaffold_test.go
git commit -m "feat(widget): XML template generators (widget XML + package manifest)"
```

---

## Task 3: JS and JSON Template Generation

**Files:**
- Modify: `cmd/mxcli/widget_scaffold.go`
- Modify: `cmd/mxcli/widget_scaffold_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `widget_scaffold_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./cmd/mxcli/ -run "TestGenerateJSX|TestGenerateEditorConfig|TestGeneratePackageJSON" -v 2>&1 | head -10
```
Expected: FAIL — functions undefined.

- [ ] **Step 3: Implement the JS/JSON generators**

Add to `cmd/mxcli/widget_scaffold.go`:

```go
// generateJSX renders the React stub for src/<Name>.jsx.
func generateJSX(name string, props []PropertySpec) string {
	var params []string
	for _, p := range props {
		params = append(params, p.Key)
	}
	propsStr := ""
	if len(params) > 0 {
		propsStr = "{ " + strings.Join(params, ", ") + " }"
	} else {
		propsStr = "_props"
	}
	labelExpr := fmt.Sprintf("'%s'", name)
	for _, p := range props {
		if p.Key == "label" {
			labelExpr = "label ?? '" + name + "'"
			break
		}
	}
	return fmt.Sprintf(
		`import { createElement } from 'react';

export function %s(%s) {
    return createElement('div', { className: '%s' },
        createElement('span', null, %s),
        // TODO: implement
    );
}

export default %s;
`, name, propsStr, strings.ToLower(name), labelExpr, name)
}

// generateEditorConfig renders the Studio Pro design-time preview script.
func generateEditorConfig(name string, props []PropertySpec) string {
	// Check if there's a label property to use as caption
	hasLabel := false
	for _, p := range props {
		if p.Key == "label" {
			hasLabel = true
			break
		}
	}
	captionBody := fmt.Sprintf(`return %q;`, name)
	if hasLabel {
		captionBody = fmt.Sprintf(`return props && props.label ? props.label : %q;`, name)
	}
	return fmt.Sprintf(
		`"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getCustomCaption = function (props) {
    %s
};
exports.getPreview = function (props, isDarkMode) {
    return {
        type: "RowLayout",
        columnSize: "grow",
        children: [{
            type: "Text",
            content: %q,
            fontColor: isDarkMode ? "#cba6f7" : "#89b4fa",
        }]
    };
};
`, captionBody, name)
}

// generateEditorPreview renders the minimal browser-preview stub.
func generateEditorPreview() string {
	return `"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.preview = function () { return null; };
`
}

// generatePackageJSON renders the package.json with esbuild as the only dev dependency.
func generatePackageJSON(packageName string) string {
	return fmt.Sprintf(`{
  "name": %q,
  "version": "1.0.0",
  "private": true,
  "type": "module",
  "devDependencies": {
    "esbuild": "^0.20.0"
  }
}
`, packageName)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./cmd/mxcli/ -run "TestGenerateJSX|TestGenerateEditorConfig|TestGeneratePackageJSON" -v
```
Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli/widget_scaffold.go cmd/mxcli/widget_scaffold_test.go
git commit -m "feat(widget): JSX, editorConfig, editorPreview, package.json generators"
```

---

## Task 4: scaffoldWidget Helper

**Files:**
- Modify: `cmd/mxcli/widget_scaffold.go`
- Modify: `cmd/mxcli/widget_scaffold_test.go`

- [ ] **Step 1: Write the failing test**

Add to `widget_scaffold_test.go`:

```go
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
	// XML must contain the widget ID
	xmlBytes, _ := os.ReadFile(filepath.Join(dir, "src/MySlider.xml"))
	if !strings.Contains(string(xmlBytes), "com.acme.widget.MySlider.MySlider") {
		t.Errorf("MySlider.xml missing widget ID")
	}
}
```

Add `"path/filepath"` and `"os"` to the imports in the test file.

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./cmd/mxcli/ -run "TestScaffoldWidget_CreatesExpectedFiles" -v 2>&1 | head -10
```
Expected: FAIL — `scaffoldWidget` undefined.

- [ ] **Step 3: Implement scaffoldWidget**

Add to `cmd/mxcli/widget_scaffold.go` (add `"bytes"`, `"image"`, `"image/color"`, `"image/png"`, `"os"`, `"path/filepath"` to imports):

```go
// scaffoldWidget writes all source files for one widget into dir/src/.
// Call this for both single-widget projects and packages (add-widget).
func scaffoldWidget(dir, name, widgetID string, offline bool, props []PropertySpec) error {
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return err
	}
	files := map[string][]byte{
		name + ".xml":             []byte(generateWidgetXML(name, widgetID, offline, props)),
		name + ".jsx":             []byte(generateJSX(name, props)),
		name + ".editorConfig.js": []byte(generateEditorConfig(name, props)),
		name + ".editorPreview.js": []byte(generateEditorPreview()),
		name + ".icon.png":        minimalPNG(),
		name + ".icon.dark.png":   minimalPNG(),
		name + ".tile.png":        minimalPNG(),
		name + ".tile.dark.png":   minimalPNG(),
	}
	for filename, content := range files {
		dest := filepath.Join(srcDir, filename)
		if err := os.WriteFile(dest, content, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", filename, err)
		}
	}
	return nil
}

// minimalPNG returns a minimal valid 1×1 transparent PNG image as bytes.
func minimalPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.Transparent)
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./cmd/mxcli/ -run "TestScaffoldWidget_CreatesExpectedFiles" -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli/widget_scaffold.go cmd/mxcli/widget_scaffold_test.go
git commit -m "feat(widget): scaffoldWidget helper writes all source files"
```

---

## Task 5: `widget new` Command (Single Widget + Package)

**Files:**
- Modify: `cmd/mxcli/widget_scaffold.go` (add `runWidgetNew`, `scaffoldPackage`)
- Modify: `cmd/mxcli/widget_scaffold_test.go`
- Modify: `cmd/mxcli/cmd_widget.go` (register `widgetNewCmd`)

- [ ] **Step 1: Write the failing tests**

Add to `widget_scaffold_test.go`:

```go
func TestScaffoldSingleWidget_TopLevelFiles(t *testing.T) {
	dir := t.TempDir()
	// Simulate runWidgetNew logic: scaffold + write package.json + package.xml
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
	// src/ must exist but be empty
	entries, _ := os.ReadDir(filepath.Join(dir, "src"))
	if len(entries) != 0 {
		t.Errorf("src/ must be empty for package scaffold, got %d entries", len(entries))
	}
	// package.xml must have no widgetFiles
	xmlBytes, _ := os.ReadFile(filepath.Join(dir, "package.xml"))
	if strings.Contains(string(xmlBytes), "<widgetFile") {
		t.Errorf("package.xml must not list widget files yet\ngot:\n%s", string(xmlBytes))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./cmd/mxcli/ -run "TestScaffoldSingleWidget_TopLevelFiles|TestScaffoldPackage_EmptySrc" -v 2>&1 | head -10
```
Expected: FAIL — `scaffoldPackage` undefined.

- [ ] **Step 3: Implement scaffoldPackage and the widgetNewCmd**

Add `scaffoldPackage` to `widget_scaffold.go`:

```go
// scaffoldPackage creates an empty multi-widget package project skeleton.
// No widgets are added; use add-widget to append them.
func scaffoldPackage(dir, name string) error {
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0755); err != nil {
		return err
	}
	pkgName := strings.ToLower(name)
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(generatePackageJSON(pkgName)), 0644); err != nil {
		return err
	}
	// Empty package.xml — no widget files yet
	return os.WriteFile(filepath.Join(dir, "package.xml"), []byte(generatePackageXML(name, nil)), 0644)
}
```

Add `runWidgetNew` to `widget_scaffold.go`:

```go
// runWidgetNew implements `mxcli widget new <name>`.
func runWidgetNew(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("widget name required: mxcli widget new <name>")
	}
	name := args[0]
	isPackage, _ := cmd.Flags().GetBool("package")
	outDir := name

	if _, err := os.Stat(outDir); err == nil {
		return fmt.Errorf("directory %q already exists", outDir)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	if isPackage {
		if err := scaffoldPackage(outDir, name); err != nil {
			return fmt.Errorf("scaffolding package: %w", err)
		}
		fmt.Printf("Created widget package project: %s/\n", outDir)
		fmt.Printf("  Add widgets: cd %s && mxcli widget add-widget <WidgetName>\n", outDir)
		fmt.Printf("  Build:       mxcli widget build\n")
		return nil
	}

	// Single widget
	widgetID, _ := cmd.Flags().GetString("id")
	if widgetID == "" {
		widgetID = deriveWidgetID(name)
	}
	offline, _ := cmd.Flags().GetBool("offline")
	propStrs, _ := cmd.Flags().GetStringArray("property")
	var props []PropertySpec
	for _, s := range propStrs {
		p, err := parsePropertySpec(s)
		if err != nil {
			return err
		}
		props = append(props, p)
	}

	if err := scaffoldWidget(outDir, name, widgetID, offline, props); err != nil {
		return fmt.Errorf("scaffolding widget: %w", err)
	}
	pkgName := strings.ToLower(name)
	if err := os.WriteFile(filepath.Join(outDir, "package.json"), []byte(generatePackageJSON(pkgName)), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "package.xml"), []byte(generatePackageXML(name, []string{name})), 0644); err != nil {
		return err
	}

	fmt.Printf("Created widget project: %s/\n", outDir)
	fmt.Printf("  Widget ID: %s\n", widgetID)
	fmt.Printf("  Edit:      %s/src/%s.jsx\n", outDir, name)
	fmt.Printf("  Build:     cd %s && mxcli widget build\n", outDir)
	return nil
}
```

Add the import for cobra at the top of `widget_scaffold.go`:
```go
import (
    "bytes"
    "fmt"
    "image"
    "image/color"
    "image/png"
    "os"
    "path/filepath"
    "strings"
    "unicode"

    "github.com/spf13/cobra"
)
```

Add command declaration and registration. In `cmd_widget.go`, add inside `init()` before `rootCmd.AddCommand(widgetCmd)`:

```go
// Declare at package level (add near the top of cmd_widget.go with other var blocks):
var widgetNewCmd = &cobra.Command{
    Use:   "new <name>",
    Short: "Scaffold a new pluggable widget project",
    Long: `Create a new Mendix pluggable widget project with a parameterized XML definition,
JSX stub, editorConfig, editorPreview, and placeholder icons.

Examples:
  mxcli widget new MySlider
  mxcli widget new MySlider --property "value:attribute:Decimal" --property "label:string" --property "onChange:action"
  mxcli widget new MySlider --id com.acme.widget.MySlider --offline
  mxcli widget new CrusherWidgets --package`,
    RunE: runWidgetNew,
}
```

In `cmd_widget.go` `init()`, add before `rootCmd.AddCommand(widgetCmd)`:

```go
widgetNewCmd.Flags().String("id", "", "Widget ID (default: com.mendix.widget.custom.<Name>.<Name>)")
widgetNewCmd.Flags().StringArray("property", nil, "Property spec: key:type or key:type:subtype (repeatable)")
widgetNewCmd.Flags().Bool("offline", false, "Set offlineCapable=true in the widget XML")
widgetNewCmd.Flags().Bool("package", false, "Create a multi-widget package project (empty src/)")
widgetCmd.AddCommand(widgetNewCmd)
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./cmd/mxcli/ -run "TestScaffoldSingleWidget_TopLevelFiles|TestScaffoldPackage_EmptySrc" -v
```
Expected: All PASS.

- [ ] **Step 5: Build and smoke-test manually**

```bash
go build -o /tmp/mxcli ./cmd/mxcli && \
  cd /tmp && /tmp/mxcli widget new TestWidget --property "value:attribute:Decimal" --property "label:string" && \
  ls -la TestWidget/ TestWidget/src/ && \
  cat TestWidget/src/TestWidget.xml && \
  cat TestWidget/src/TestWidget.jsx && \
  rm -rf TestWidget
```
Expected: directory structure with all 10 files.

- [ ] **Step 6: Commit**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add cmd/mxcli/widget_scaffold.go cmd/mxcli/widget_scaffold_test.go cmd/mxcli/cmd_widget.go
git commit -m "feat(widget): mxcli widget new (single + --package)"
```

---

## Task 6: `widget add-widget` Command

**Files:**
- Modify: `cmd/mxcli/widget_scaffold.go` (add `runWidgetAddWidget`, `appendWidgetFileToPackageXML`)
- Modify: `cmd/mxcli/widget_scaffold_test.go`
- Modify: `cmd/mxcli/cmd_widget.go` (register `widgetAddWidgetCmd`)

- [ ] **Step 1: Write the failing tests**

Add to `widget_scaffold_test.go`:

```go
func TestAppendWidgetFileToPackageXML(t *testing.T) {
	dir := t.TempDir()
	// Start with a package.xml that has one widget
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

	// Adding same widget twice must not duplicate
	_ = appendWidgetFileToPackageXML(pkgXML, "WidgetA")
	data, _ := os.ReadFile(pkgXML)
	count := strings.Count(string(data), `path="WidgetA.xml"`)
	if count != 1 {
		t.Errorf("expected 1 WidgetA entry, got %d\ncontent:\n%s", count, string(data))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./cmd/mxcli/ -run "TestAppendWidgetFile" -v 2>&1 | head -10
```
Expected: FAIL — `appendWidgetFileToPackageXML` undefined.

- [ ] **Step 3: Implement appendWidgetFileToPackageXML and runWidgetAddWidget**

Add to `widget_scaffold.go`:

```go
// appendWidgetFileToPackageXML reads package.xml, adds a <widgetFile> entry for widgetName
// (if not already present), and writes it back. Uses string manipulation to preserve
// the existing XML structure without a full re-parse/re-serialize round-trip.
func appendWidgetFileToPackageXML(pkgXMLPath, widgetName string) error {
	data, err := os.ReadFile(pkgXMLPath)
	if err != nil {
		return fmt.Errorf("reading package.xml: %w", err)
	}
	entry := fmt.Sprintf(`path="%s.xml"`, widgetName)
	if strings.Contains(string(data), entry) {
		return nil // already present
	}
	// Insert before </widgetFiles>
	newEntry := fmt.Sprintf("      <widgetFile path=%q/>\n", widgetName+".xml")
	updated := strings.Replace(string(data), "</widgetFiles>", newEntry+"    </widgetFiles>", 1)
	if updated == string(data) {
		return fmt.Errorf("could not find </widgetFiles> in package.xml")
	}
	return os.WriteFile(pkgXMLPath, []byte(updated), 0644)
}

// runWidgetAddWidget implements `mxcli widget add-widget <name>` (run inside package dir).
func runWidgetAddWidget(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("widget name required: mxcli widget add-widget <name>")
	}
	name := args[0]
	dir := "."

	pkgXMLPath := filepath.Join(dir, "package.xml")
	if _, err := os.Stat(pkgXMLPath); err != nil {
		return fmt.Errorf("package.xml not found in current directory — run this command inside a widget package project")
	}

	widgetID, _ := cmd.Flags().GetString("id")
	if widgetID == "" {
		widgetID = deriveWidgetID(name)
	}
	offline, _ := cmd.Flags().GetBool("offline")
	propStrs, _ := cmd.Flags().GetStringArray("property")
	var props []PropertySpec
	for _, s := range propStrs {
		p, err := parsePropertySpec(s)
		if err != nil {
			return err
		}
		props = append(props, p)
	}

	// Check widget files don't already exist
	if _, err := os.Stat(filepath.Join(dir, "src", name+".xml")); err == nil {
		return fmt.Errorf("widget %q already exists in src/", name)
	}

	if err := scaffoldWidget(dir, name, widgetID, offline, props); err != nil {
		return fmt.Errorf("scaffolding widget: %w", err)
	}
	if err := appendWidgetFileToPackageXML(pkgXMLPath, name); err != nil {
		return fmt.Errorf("updating package.xml: %w", err)
	}

	fmt.Printf("Added widget: %s\n", name)
	fmt.Printf("  Edit: src/%s.jsx\n", name)
	fmt.Printf("  Build: mxcli widget build\n")
	return nil
}
```

In `cmd_widget.go`, add at package level:

```go
var widgetAddWidgetCmd = &cobra.Command{
    Use:   "add-widget <name>",
    Short: "Add a widget to an existing package project",
    Long: `Add a new widget to a multi-widget package project.
Run inside the package directory (where package.xml lives).

Examples:
  mxcli widget add-widget CrusherSlider
  mxcli widget add-widget CrusherSlider --property "value:attribute:Decimal" --property "label:string"`,
    RunE: runWidgetAddWidget,
}
```

In `cmd_widget.go` `init()`, add:

```go
widgetAddWidgetCmd.Flags().String("id", "", "Widget ID (default: com.mendix.widget.custom.<Name>.<Name>)")
widgetAddWidgetCmd.Flags().StringArray("property", nil, "Property spec: key:type or key:type:subtype (repeatable)")
widgetAddWidgetCmd.Flags().Bool("offline", false, "Set offlineCapable=true")
widgetCmd.AddCommand(widgetAddWidgetCmd)
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./cmd/mxcli/ -run "TestAppendWidgetFile" -v
```
Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli/widget_scaffold.go cmd/mxcli/widget_scaffold_test.go cmd/mxcli/cmd_widget.go
git commit -m "feat(widget): mxcli widget add-widget (append widget to package)"
```

---

## Task 7: Build Pipeline — Discover + Validate

**Files:**
- Create: `cmd/mxcli/widget_build.go`
- Create: `cmd/mxcli/widget_build_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// cmd/mxcli/widget_build_test.go
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeXML writes content to dir/src/<filename>.
func writeXML(t *testing.T, dir, filename, content string) {
	t.Helper()
	srcDir := filepath.Join(dir, "src")
	os.MkdirAll(srcDir, 0755)
	if err := os.WriteFile(filepath.Join(srcDir, filename), []byte(content), 0644); err != nil {
		t.Fatalf("writeXML %s: %v", filename, err)
	}
}

const validWidgetXML = `<?xml version="1.0" encoding="utf-8"?>
<widget id="com.acme.widget.MySlider.MySlider" pluginWidget="true" offlineCapable="false"
        xmlns="http://www.mendix.com/widget/1.0/">
  <name>My Slider</name>
  <properties>
    <propertyGroup caption="General">
      <property key="value" type="attribute" required="true">
        <caption>Value</caption><description/>
        <attributeTypes><attributeType name="Decimal"/></attributeTypes>
      </property>
    </propertyGroup>
  </properties>
</widget>`

func TestDiscoverWidgets_FindsXML(t *testing.T) {
	dir := t.TempDir()
	writeXML(t, dir, "MySlider.xml", validWidgetXML)

	infos, err := discoverWidgets(dir)
	if err != nil {
		t.Fatalf("discoverWidgets: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("expected 1 widget, got %d", len(infos))
	}
	if infos[0].Name != "MySlider" || infos[0].WidgetID != "com.acme.widget.MySlider.MySlider" {
		t.Errorf("unexpected widget info: %+v", infos[0])
	}
}

func TestDiscoverWidgets_EmptySrc(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "src"), 0755)
	infos, err := discoverWidgets(dir)
	if err != nil {
		t.Fatalf("discoverWidgets: %v", err)
	}
	if len(infos) != 0 {
		t.Errorf("expected 0 widgets, got %d", len(infos))
	}
}

func TestValidateWidgetInfo_ValidID(t *testing.T) {
	info := widgetInfo{Name: "MySlider", WidgetID: "com.acme.widget.MySlider.MySlider", DisplayName: "My Slider"}
	if err := validateWidgetInfo(info); err != nil {
		t.Errorf("validateWidgetInfo: unexpected error: %v", err)
	}
}

func TestValidateWidgetInfo_BadID(t *testing.T) {
	cases := []struct {
		id      string
		wantMsg string
	}{
		{"com.short", "4 dot-separated segments"},
		{"com.a.b.c.d.e", "4 dot-separated segments"},
		{"", "4 dot-separated segments"},
	}
	for _, tc := range cases {
		info := widgetInfo{Name: "Foo", WidgetID: tc.id, DisplayName: "Foo"}
		err := validateWidgetInfo(info)
		if err == nil {
			t.Errorf("validateWidgetInfo(%q): expected error", tc.id)
		} else if !strings.Contains(err.Error(), tc.wantMsg) {
			t.Errorf("validateWidgetInfo(%q): error %q does not contain %q", tc.id, err.Error(), tc.wantMsg)
		}
	}
}

func TestValidateWidgetInfo_EmptyName(t *testing.T) {
	info := widgetInfo{Name: "MySlider", WidgetID: "com.a.b.c.MySlider", DisplayName: ""}
	err := validateWidgetInfo(info)
	if err == nil {
		t.Errorf("validateWidgetInfo: expected error for empty display name")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./cmd/mxcli/ -run "TestDiscoverWidgets|TestValidateWidgetInfo" -v 2>&1 | head -10
```
Expected: FAIL — types undefined.

- [ ] **Step 3: Implement discoverWidgets + validateWidgetInfo**

Create `cmd/mxcli/widget_build.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// widgetInfo holds the discovered metadata for one widget (from src/<Name>.xml).
type widgetInfo struct {
	Name        string // e.g. "MySlider"
	WidgetID    string // e.g. "com.acme.widget.MySlider.MySlider"
	DisplayName string // e.g. "My Slider" (from <name> element)
	XMLPath     string // absolute path to the XML file
}

// xmlWidgetRoot is a minimal struct for parsing just id and name from a widget XML.
type xmlWidgetRoot struct {
	XMLName     xml.Name `xml:"widget"`
	ID          string   `xml:"id,attr"`
	DisplayName string   `xml:"name"`
}

// discoverWidgets globs src/*.xml in projectDir and parses each to extract widgetID and name.
func discoverWidgets(projectDir string) ([]widgetInfo, error) {
	pattern := filepath.Join(projectDir, "src", "*.xml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	var infos []widgetInfo
	for _, xmlPath := range matches {
		data, err := os.ReadFile(xmlPath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", xmlPath, err)
		}
		var root xmlWidgetRoot
		if err := xml.Unmarshal(data, &root); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", xmlPath, err)
		}
		name := strings.TrimSuffix(filepath.Base(xmlPath), ".xml")
		infos = append(infos, widgetInfo{
			Name:        name,
			WidgetID:    root.ID,
			DisplayName: root.DisplayName,
			XMLPath:     xmlPath,
		})
	}
	return infos, nil
}

// validateWidgetInfo checks that a discovered widget has a valid ID format and non-empty name.
func validateWidgetInfo(info widgetInfo) error {
	parts := strings.Split(info.WidgetID, ".")
	if len(parts) != 4 {
		return fmt.Errorf("widget %q: id %q must have exactly 4 dot-separated segments (e.g. com.acme.widget.Name)", info.Name, info.WidgetID)
	}
	if info.DisplayName == "" {
		return fmt.Errorf("widget %q: <name> element is empty in XML", info.Name)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./cmd/mxcli/ -run "TestDiscoverWidgets|TestValidateWidgetInfo" -v
```
Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli/widget_build.go cmd/mxcli/widget_build_test.go
git commit -m "feat(widget): build pipeline — discoverWidgets + validateWidgetInfo"
```

---

## Task 8: Build Pipeline — Toolchain Detection + Dependency Install

**Files:**
- Modify: `cmd/mxcli/widget_build.go`
- Modify: `cmd/mxcli/widget_build_test.go`

- [ ] **Step 1: Write the failing test**

Add to `widget_build_test.go`:

```go
func TestDetectToolchain_FindsSomething(t *testing.T) {
	// This test just verifies detectToolchain doesn't panic.
	// It may return an error on machines without bun/npm, which is fine.
	tool, err := detectToolchain()
	if err == nil && tool != "bun" && tool != "npm" {
		t.Errorf("detectToolchain returned unexpected tool %q", tool)
	}
	// If err != nil, the error message must be instructive
	if err != nil && !strings.Contains(err.Error(), "bun") {
		t.Errorf("error must mention bun: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./cmd/mxcli/ -run "TestDetectToolchain" -v 2>&1 | head -10
```
Expected: FAIL — `detectToolchain` undefined.

- [ ] **Step 3: Implement detectToolchain + installDeps**

Add to `cmd/mxcli/widget_build.go`:

```go
import (
    // add to existing imports:
    "os/exec"
)

// detectToolchain returns "bun" or "npm" depending on what is available in PATH.
func detectToolchain() (string, error) {
	if _, err := exec.LookPath("bun"); err == nil {
		return "bun", nil
	}
	if _, err := exec.LookPath("npm"); err == nil {
		return "npm", nil
	}
	return "", fmt.Errorf("bun not found, npm not found\n" +
		"  install bun: curl -fsSL https://bun.sh/install | bash\n" +
		"  install npm: https://nodejs.org/")
}

// installDeps runs bun install or npm install if node_modules/ is absent.
func installDeps(projectDir, tool string) error {
	if _, err := os.Stat(filepath.Join(projectDir, "node_modules")); err == nil {
		return nil // already installed
	}
	fmt.Printf("Installing dependencies (%s install)...\n", tool)
	cmd := exec.Command(tool, "install")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./cmd/mxcli/ -run "TestDetectToolchain" -v
```
Expected: PASS (either detects a tool or returns expected error).

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli/widget_build.go cmd/mxcli/widget_build_test.go
git commit -m "feat(widget): build pipeline — detectToolchain + installDeps"
```

---

## Task 9: Build Pipeline — Compile, Package, Verify + runWidgetBuild

**Files:**
- Modify: `cmd/mxcli/widget_build.go`
- Modify: `cmd/mxcli/cmd_widget.go` (register `widgetBuildCmd`)

- [ ] **Step 1: Write the failing test for packageMPK**

Add to `widget_build_test.go`:

```go
func TestPackageMPK_CreatesZip(t *testing.T) {
	dir := t.TempDir()
	// Create a minimal dist/ with a widget JS file
	distDir := filepath.Join(dir, "dist")
	jsDir := filepath.Join(distDir, "com", "mendix", "widget", "custom", "MySlider")
	os.MkdirAll(jsDir, 0755)
	os.WriteFile(filepath.Join(jsDir, "MySlider.js"), []byte("// bundle"), 0644)
	os.WriteFile(filepath.Join(distDir, "MySlider.xml"), []byte("<widget/>"), 0644)
	os.WriteFile(filepath.Join(distDir, "package.xml"), []byte("<package/>"), 0644)

	mpkPath, err := packageMPK(dir, "MySlider")
	if err != nil {
		t.Fatalf("packageMPK: %v", err)
	}
	if _, err := os.Stat(mpkPath); err != nil {
		t.Errorf("MPK file not created at %s: %v", mpkPath, err)
	}
	if !strings.HasSuffix(mpkPath, "MySlider.mpk") {
		t.Errorf("expected MySlider.mpk, got %s", mpkPath)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./cmd/mxcli/ -run "TestPackageMPK" -v 2>&1 | head -10
```
Expected: FAIL — `packageMPK` undefined.

- [ ] **Step 3: Implement compile, assets, package, verify, and runWidgetBuild**

Add to `cmd/mxcli/widget_build.go`:

```go
import (
    // add to existing imports:
    "archive/zip"
    "io"
)

// compileWidget invokes esbuild (via `bun x esbuild` or `npx esbuild`) to bundle one widget as CJS and ESM.
func compileWidget(projectDir, tool string, info widgetInfo) error {
	distJS := filepath.Join(projectDir, "dist", "com", "mendix", "widget", "custom", info.Name, info.Name+".js")
	distMJS := filepath.Join(projectDir, "dist", "com", "mendix", "widget", "custom", info.Name, info.Name+".mjs")
	if err := os.MkdirAll(filepath.Dir(distJS), 0755); err != nil {
		return err
	}

	src := filepath.Join(projectDir, "src", info.Name+".jsx")
	externals := []string{"--external:react", "--external:react-dom", "--external:big.js"}

	for _, out := range []struct{ format, outfile string }{
		{"cjs", distJS},
		{"esm", distMJS},
	} {
		esbuildArgs := append([]string{src, "--bundle",
			"--format=" + out.format, "--outfile=" + out.outfile},
			externals...)

		// bun x esbuild <args>   vs   npx esbuild <args>
		var cmd *exec.Cmd
		if tool == "bun" {
			cmd = exec.Command("bun", append([]string{"x", "esbuild"}, esbuildArgs...)...)
		} else {
			cmd = exec.Command("npx", append([]string{"esbuild"}, esbuildArgs...)...)
		}
		cmd.Dir = projectDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("esbuild %s (%s): %w", info.Name, out.format, err)
		}
	}
	return nil
}

// copyAssets copies XML descriptors, editor scripts, and icon/tile PNGs to dist/.
func copyAssets(projectDir string, infos []widgetInfo) error {
	srcDir := filepath.Join(projectDir, "src")
	distDir := filepath.Join(projectDir, "dist")
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return err
	}

	// package.xml
	if err := copyFile(filepath.Join(projectDir, "package.xml"), filepath.Join(distDir, "package.xml")); err != nil {
		return err
	}

	for _, info := range infos {
		suffixes := []string{
			".xml",
			".editorConfig.js",
			".editorPreview.js",
			".icon.png",
			".icon.dark.png",
			".tile.png",
			".tile.dark.png",
		}
		for _, suf := range suffixes {
			src := filepath.Join(srcDir, info.Name+suf)
			dst := filepath.Join(distDir, info.Name+suf)
			if _, err := os.Stat(src); err == nil {
				if err := copyFile(src, dst); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// copyFile copies src to dst, creating dst's parent directory if needed.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// packageMPK zips dist/ into <packageName>.mpk in projectDir and returns the MPK path.
func packageMPK(projectDir, packageName string) (string, error) {
	distDir := filepath.Join(projectDir, "dist")
	mpkPath := filepath.Join(projectDir, packageName+".mpk")

	f, err := os.Create(mpkPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	w := zip.NewWriter(f)
	defer w.Close()

	err = filepath.Walk(distDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(distDir, path)
		rel = filepath.ToSlash(rel) // ZIP always uses forward slashes
		entry, err := w.Create(rel)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = entry.Write(data)
		return err
	})
	return mpkPath, err
}

// verifyMPK opens the MPK ZIP and confirms each widget has a .js bundle inside.
func verifyMPK(mpkPath string, infos []widgetInfo) error {
	r, err := zip.OpenReader(mpkPath)
	if err != nil {
		return fmt.Errorf("opening MPK: %w", err)
	}
	defer r.Close()

	found := make(map[string]bool)
	for _, f := range r.File {
		found[f.Name] = true
	}
	for _, info := range infos {
		jsPath := fmt.Sprintf("com/mendix/widget/custom/%s/%s.js", info.Name, info.Name)
		if !found[jsPath] {
			return fmt.Errorf("MPK missing expected JS bundle: %s", jsPath)
		}
	}
	return nil
}

// readPackageName extracts the <clientModule name="..."> attribute from package.xml.
func readPackageName(projectDir string) (string, error) {
	type xmlClientModule struct {
		Name string `xml:"name,attr"`
	}
	type xmlPackage struct {
		ClientModule xmlClientModule `xml:"clientModule"`
	}
	data, err := os.ReadFile(filepath.Join(projectDir, "package.xml"))
	if err != nil {
		return "", err
	}
	var pkg xmlPackage
	if err := xml.Unmarshal(data, &pkg); err != nil {
		return "", err
	}
	if pkg.ClientModule.Name == "" {
		return "", fmt.Errorf("package.xml: missing clientModule name attribute")
	}
	return pkg.ClientModule.Name, nil
}

// runWidgetBuild implements `mxcli widget build [--dir <path>]`.
func runWidgetBuild(cmd *cobra.Command, args []string) error {
	dir, _ := cmd.Flags().GetString("dir")
	if dir == "" {
		dir = "."
	}

	// 1. Discover
	infos, err := discoverWidgets(dir)
	if err != nil {
		return fmt.Errorf("discovering widgets: %w", err)
	}
	if len(infos) == 0 {
		return fmt.Errorf("no widget XML files found in %s/src/", dir)
	}

	// 2. Validate
	for _, info := range infos {
		if err := validateWidgetInfo(info); err != nil {
			return err
		}
	}
	fmt.Printf("Found %d widget(s): ", len(infos))
	for i, info := range infos {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(info.Name)
	}
	fmt.Println()

	// 3. Detect toolchain
	tool, err := detectToolchain()
	if err != nil {
		return err
	}

	// 4. Install deps
	if err := installDeps(dir, tool); err != nil {
		return fmt.Errorf("installing dependencies: %w", err)
	}

	// 5. Clean dist
	distDir := filepath.Join(dir, "dist")
	_ = os.RemoveAll(distDir)

	// 6. Compile each widget
	for _, info := range infos {
		fmt.Printf("  Compiling %s...\n", info.Name)
		if err := compileWidget(dir, tool, info); err != nil {
			return err
		}
	}

	// 7. Copy assets
	if err := copyAssets(dir, infos); err != nil {
		return fmt.Errorf("copying assets: %w", err)
	}

	// 8. Package
	packageName, err := readPackageName(dir)
	if err != nil {
		return err
	}
	mpkPath, err := packageMPK(dir, packageName)
	if err != nil {
		return fmt.Errorf("packaging MPK: %w", err)
	}

	// 9. Verify
	if err := verifyMPK(mpkPath, infos); err != nil {
		return err
	}

	fi, _ := os.Stat(mpkPath)
	size := int64(0)
	if fi != nil {
		size = fi.Size() / 1024
	}
	fmt.Printf("Built %s (%d widget(s), %d KB)\n", filepath.Base(mpkPath), len(infos), size)
	return nil
}
```

Add the cobra import to `widget_build.go`:

```go
import (
    "archive/zip"
    "encoding/xml"
    "fmt"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "strings"

    "github.com/spf13/cobra"
)
```

In `cmd_widget.go`, add at package level:

```go
var widgetBuildCmd = &cobra.Command{
    Use:   "build",
    Short: "Build widget project into an .mpk package",
    Long: `Build a Mendix pluggable widget project into an .mpk file.

Discovers all widgets in src/*.xml, validates their definitions, invokes esbuild
(via bun or npm) to compile each JSX bundle, copies assets, packages everything
into <PackageName>.mpk, and verifies the output.

Examples:
  mxcli widget build
  mxcli widget build --dir ./CrusherWidgets`,
    RunE: runWidgetBuild,
}
```

In `cmd_widget.go` `init()`, add:

```go
widgetBuildCmd.Flags().String("dir", ".", "Widget project root directory")
widgetCmd.AddCommand(widgetBuildCmd)
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./cmd/mxcli/ -run "TestPackageMPK" -v
```
Expected: PASS.

- [ ] **Step 5: Full test suite for cmd/mxcli/**

```bash
go test ./cmd/mxcli/ -v 2>&1 | tail -30
```
Expected: All existing + new tests PASS. No regressions.

- [ ] **Step 6: Build and full smoke-test**

```bash
go build -o /tmp/mxcli ./cmd/mxcli && \

# Single widget end-to-end (needs bun or npm in PATH)
cd /tmp && \
/tmp/mxcli widget new TestSlider \
  --property "value:attribute:Decimal" \
  --property "label:string" \
  --property "onChange:action" && \
cd TestSlider && \
/tmp/mxcli widget build && \
ls -lh TestSlider.mpk && \

# Package end-to-end
cd /tmp && \
/tmp/mxcli widget new MyPackage --package && \
cd MyPackage && \
/tmp/mxcli widget add-widget SliderA --property "value:attribute:Decimal" && \
/tmp/mxcli widget add-widget SliderB --property "count:integer" && \
/tmp/mxcli widget build && \
ls -lh MyPackage.mpk
```
Expected: Both MPK files created, correct sizes reported.

- [ ] **Step 7: Cleanup and commit**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
rm -rf /tmp/TestSlider /tmp/MyPackage
git add cmd/mxcli/widget_build.go cmd/mxcli/widget_build_test.go cmd/mxcli/cmd_widget.go
git commit -m "feat(widget): mxcli widget build — compile + package + verify pipeline"
```

---

## Task 10: Final Integration Tests + Cleanup

**Files:**
- Modify: `cmd/mxcli/widget_scaffold_test.go` (add round-trip test)
- Modify: `cmd/mxcli/widget_build_test.go` (add integration-style test)

- [ ] **Step 1: Write round-trip integration tests**

Add to `widget_scaffold_test.go`:

```go
func TestRoundTrip_ScaffoldThenDiscover(t *testing.T) {
	dir := t.TempDir()
	props := []PropertySpec{
		{Key: "value", XMLType: "attribute", Subtype: "Decimal"},
		{Key: "label", XMLType: "string"},
	}
	// Scaffold a widget
	err := scaffoldWidget(dir, "RoundTrip", "com.test.widget.RoundTrip.RoundTrip", false, props)
	if err != nil {
		t.Fatalf("scaffoldWidget: %v", err)
	}
	// Discover it with the build pipeline
	infos, err := discoverWidgets(dir)
	if err != nil {
		t.Fatalf("discoverWidgets: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("expected 1 widget, got %d", len(infos))
	}
	if err := validateWidgetInfo(infos[0]); err != nil {
		t.Errorf("validateWidgetInfo failed on scaffolded widget: %v", err)
	}
	if infos[0].WidgetID != "com.test.widget.RoundTrip.RoundTrip" {
		t.Errorf("wrong widgetID: %s", infos[0].WidgetID)
	}
}
```

- [ ] **Step 2: Run the round-trip test**

```bash
go test ./cmd/mxcli/ -run "TestRoundTrip" -v
```
Expected: PASS.

- [ ] **Step 3: Run the full test suite**

```bash
go test ./cmd/mxcli/ -count=1 2>&1 | tail -5
```
Expected: `ok  	github.com/mendixlabs/mxcli/cmd/mxcli`.

- [ ] **Step 4: Run make build and make test**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02 && make build 2>&1 | tail -5
make test 2>&1 | tail -10
```
Expected: Both pass with no errors.

- [ ] **Step 5: Final commit**

```bash
git add cmd/mxcli/widget_scaffold_test.go cmd/mxcli/widget_build_test.go
git commit -m "test(widget): round-trip scaffold→discover integration test"
```

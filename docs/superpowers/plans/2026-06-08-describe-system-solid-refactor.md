# Describe System SOLID Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the four-file, two-giant-switch raw-BSON page describe pipeline with a SOLID-compliant formatter registry where each widget type has its own formatter, unknown pluggable widgets are handled via schema self-introspection, and adding a new widget type requires no existing code changes.

**Architecture:** A `FormatterDispatcher` maps BSON `$Type` strings (and widget IDs for pluggable widgets) to `FormatterFactory` functions via data-driven `FactoryEntry` entries with optional `SubKeyExtractor`. Each `WidgetFormatter` receives `raw map[string]any` and writes MDL to a `FormatContext`. During Phase 2 the legacy path is the dispatcher's fallback; in Phase 3 the old files are deleted.

**Tech Stack:** Go, `go.mongodb.org/mongo-driver/v2/bson`, `github.com/mendixlabs/mxcli/modelsdk/codec`, existing gen pages types in `modelsdk/gen/pages/`.

**Spec:** `docs/superpowers/specs/2026-06-08-describe-system-solid-refactor-design.md`

---

## File Map

**New files (Phase 1–2):**
- `mdl/executor/widget_formatter.go` — core interfaces + `FormatterDispatcher`
- `mdl/executor/widget_schema.go` — schema introspection (`buildSchemaMap`, `filterDefaults`)
- `mdl/executor/widget_fmt_pluggable.go` — `GenericPluggableFormatter` (unknown widget fallback)
- `mdl/executor/widget_fmt_basic.go` — leaf widgets: ActionButton, DynamicText, TextBox, TextArea, DatePicker, RadioButtons, CheckBox, Label, Title, SnippetCallWidget
- `mdl/executor/widget_fmt_container.go` — DivContainer, GroupBox, ScrollContainer, TabControl/TabPage, Footer (synthetic)
- `mdl/executor/widget_fmt_layout.go` — LayoutGrid with rows and columns
- `mdl/executor/widget_fmt_data.go` — DataView, ListView, NavigationList
- `mdl/executor/widget_fmt_datagrid.go` — built-in `Forms$DataGrid`
- `mdl/executor/widget_fmt_datagrid2.go` — pluggable DataGrid2 (registered by widget ID)
- `mdl/executor/widget_fmt_gallery.go` — pluggable Gallery
- `mdl/executor/widget_fmt_combobox.go` — pluggable ComboBox
- `mdl/executor/widget_fmt_image.go` — pluggable Image
- `mdl/executor/widget_fmt_test_helpers_test.go` — shared test utilities (newTestFormatContext, etc.)

**Modified (Phase 1 bridge, Phase 3 deletion):**
- `mdl/executor/cmd_pages_describe.go` — add bridge to dispatcher (Phase 1), delete rawWidget usage (Phase 3)
- `mdl/executor/cmd_pages_describe_pluggable.go` — fix `extractExplicitProperties()` (Phase 1)

**Deleted (Phase 3):**
- `mdl/executor/cmd_pages_describe_parse.go`
- `mdl/executor/cmd_pages_describe_output.go`
- `mdl/executor/cmd_pages_describe_pluggable.go`

---

## Phase 1 — Infrastructure

### Task 1: Core interfaces in `widget_formatter.go`

**Files:**
- Create: `mdl/executor/widget_formatter.go`
- Create: `mdl/executor/widget_formatter_test.go`

- [ ] **Step 1.1: Write the failing test**

```go
// mdl/executor/widget_formatter_test.go
package executor

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormatterDispatcher_DispatchesByBSONType(t *testing.T) {
	d := newDefaultDispatcher()
	var buf bytes.Buffer
	called := false
	d.registerBSONType("Test$Widget", FactoryEntry{
		Factory: func(raw map[string]any) WidgetFormatter {
			return &funcFormatter{fn: func(ctx *FormatContext) {
				called = true
				ctx.Write("test-widget")
			}}
		},
	})
	ctx := &FormatContext{Output: &buf, Indent: 0, Dispatcher: d}
	d.Format(ctx, map[string]any{"$Type": "Test$Widget", "Name": "w1"})
	if !called {
		t.Error("formatter was not called")
	}
	if !strings.Contains(buf.String(), "test-widget") {
		t.Errorf("unexpected output: %q", buf.String())
	}
}

func TestFormatterDispatcher_SubKeyDispatch(t *testing.T) {
	d := newDefaultDispatcher()
	var buf bytes.Buffer
	d.registerBSONType("Meta$Widget", FactoryEntry{
		Factory:         d.fallback, // default for unknown sub-key
		SubKeyExtractor: func(raw map[string]any) string { return safeStr(raw, "widgetID") },
	})
	d.registerBSONType("com.co.specific", FactoryEntry{
		Factory: func(raw map[string]any) WidgetFormatter {
			return &funcFormatter{fn: func(ctx *FormatContext) { ctx.Write("specific") }}
		},
	})
	ctx := &FormatContext{Output: &buf, Indent: 0, Dispatcher: d}
	d.Format(ctx, map[string]any{"$Type": "Meta$Widget", "widgetID": "com.co.specific"})
	if !strings.Contains(buf.String(), "specific") {
		t.Errorf("sub-key dispatch failed: %q", buf.String())
	}
}

func TestFormatterDispatcher_FallbackForUnknown(t *testing.T) {
	d := newDefaultDispatcher()
	var buf bytes.Buffer
	ctx := &FormatContext{Output: &buf, Indent: 0, Dispatcher: d}
	d.Format(ctx, map[string]any{"$Type": "Unknown$Widget", "Name": "w1"})
	if !strings.Contains(buf.String(), "-- widget") {
		t.Errorf("fallback should emit comment: %q", buf.String())
	}
}

func TestFormatContext_Child_IncrementsIndent(t *testing.T) {
	var buf bytes.Buffer
	d := newDefaultDispatcher()
	ctx := &FormatContext{Output: &buf, Indent: 2, Dispatcher: d}
	child := ctx.Child()
	if child.Indent != 3 {
		t.Errorf("Child().Indent = %d, want 3", child.Indent)
	}
}

// funcFormatter is a test helper implementing WidgetFormatter via a closure.
type funcFormatter struct{ fn func(*FormatContext) }
func (f *funcFormatter) FormatMDL(ctx *FormatContext) { f.fn(ctx) }
```

- [ ] **Step 1.2: Run test to verify it fails**

```bash
go test ./mdl/executor/ -run "TestFormatterDispatcher|TestFormatContext" -v 2>&1 | head -20
```
Expected: compile errors (types not yet defined).

- [ ] **Step 1.3: Implement `widget_formatter.go`**

```go
// mdl/executor/widget_formatter.go
package executor

import (
	"fmt"
	"io"
	"strings"
)

// ─── Narrow interfaces (I principle) ─────────────────────────────────────────

// NameResolver resolves entity and module names from container IDs.
// ExecContext implements this interface.
type NameResolver interface {
	resolveEntityContext(containerID string) string
	resolveModuleName(containerID string) string
}

// WidgetDispatcher dispatches widget formatting by raw BSON type.
type WidgetDispatcher interface {
	Format(ctx *FormatContext, raw map[string]any)
}

// ─── WidgetFormatter ─────────────────────────────────────────────────────────

// WidgetFormatter formats a single widget node as MDL.
type WidgetFormatter interface {
	FormatMDL(ctx *FormatContext)
}

// FormatterFactory creates a WidgetFormatter from raw widget BSON.
type FormatterFactory func(raw map[string]any) WidgetFormatter

// ─── FormatContext ────────────────────────────────────────────────────────────

// FormatContext carries shared state for one describe pass.
// It depends only on interfaces, never on concrete types (D principle).
type FormatContext struct {
	Output     io.Writer
	Indent     int
	Dispatcher WidgetDispatcher
}

// Child returns a FormatContext with Indent incremented by one.
func (ctx *FormatContext) Child() *FormatContext {
	return &FormatContext{
		Output:     ctx.Output,
		Indent:     ctx.Indent + 1,
		Dispatcher: ctx.Dispatcher,
	}
}

// Write writes a single line at the current indent level.
func (ctx *FormatContext) Write(s string) {
	fmt.Fprintf(ctx.Output, "%s%s\n", strings.Repeat("  ", ctx.Indent), s)
}

// WriteRaw writes s without any indent prefix (for multi-line props blocks).
func (ctx *FormatContext) WriteRaw(s string) {
	fmt.Fprint(ctx.Output, s)
}

// ─── FactoryEntry ────────────────────────────────────────────────────────────

// FactoryEntry is a registration record. SubKeyExtractor, when non-nil,
// extracts a secondary dispatch key (e.g. pluggable widget ID) from the raw map.
// This keeps Mendix widget-category knowledge out of the dispatcher (O principle).
type FactoryEntry struct {
	Factory         FormatterFactory
	SubKeyExtractor func(raw map[string]any) string
}

// ─── FormatterDispatcher ─────────────────────────────────────────────────────

// FormatterDispatcher maps BSON $Type strings (and sub-keys for pluggable widgets)
// to FormatterFactory functions. Dispatch is purely data-driven: no hard-coded
// special cases for any widget category (O principle).
type FormatterDispatcher struct {
	entries  map[string]FactoryEntry
	fallback FormatterFactory
}

func newDefaultDispatcher() *FormatterDispatcher {
	d := &FormatterDispatcher{
		entries: make(map[string]FactoryEntry),
	}
	d.fallback = func(raw map[string]any) WidgetFormatter {
		return &unknownWidgetFormatter{raw: raw}
	}
	return d
}

func (d *FormatterDispatcher) registerBSONType(bsonType string, entry FactoryEntry) {
	d.entries[bsonType] = entry
}

// Format dispatches to the appropriate formatter. If no entry exists, the
// fallback is used. For entries with a SubKeyExtractor, a second lookup is
// performed on the extracted key; if that also misses, fallback is used.
func (d *FormatterDispatcher) Format(ctx *FormatContext, raw map[string]any) {
	bsonType, _ := raw["$Type"].(string)
	entry, ok := d.entries[bsonType]
	if !ok {
		d.fallback(raw).FormatMDL(ctx)
		return
	}
	if entry.SubKeyExtractor != nil {
		subKey := entry.SubKeyExtractor(raw)
		if sub, ok := d.entries[subKey]; ok {
			sub.Factory(raw).FormatMDL(ctx)
			return
		}
		// Unknown sub-key → fallback (GenericPluggableFormatter once installed)
		d.fallback(raw).FormatMDL(ctx)
		return
	}
	entry.Factory(raw).FormatMDL(ctx)
}

// unknownWidgetFormatter is the default fallback for unregistered widget types.
type unknownWidgetFormatter struct{ raw map[string]any }

func (f *unknownWidgetFormatter) FormatMDL(ctx *FormatContext) {
	bsonType, _ := f.raw["$Type"].(string)
	name := safeStr(f.raw, "Name")
	ctx.Write(fmt.Sprintf("-- widget %s %s", bsonType, name))
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

// safeStr safely extracts a string from a raw BSON map. Returns "" on miss.
func safeStr(raw map[string]any, key string) string {
	v, _ := raw[key].(string)
	return v
}

// indentStr returns n repetitions of two spaces.
func indentStr(n int) string { return strings.Repeat("  ", n) }
```

- [ ] **Step 1.4: Run tests**

```bash
go test ./mdl/executor/ -run "TestFormatterDispatcher|TestFormatContext" -v
```
Expected: all 4 tests PASS.

- [ ] **Step 1.5: Verify build**

```bash
go build ./mdl/executor/
```
Expected: no errors.

- [ ] **Step 1.6: Commit**

```bash
git add mdl/executor/widget_formatter.go mdl/executor/widget_formatter_test.go
git commit -m "feat(describe): add WidgetFormatter interface + FormatterDispatcher (Phase 1)"
```

---

### Task 2: Schema introspection utilities in `widget_schema.go`

**Files:**
- Create: `mdl/executor/widget_schema.go`
- Create: `mdl/executor/widget_schema_test.go`

The schema is embedded in every `CustomWidgets$CustomWidget` BSON:
- `raw["Type"]["ObjectType"]["PropertyTypes"]` → array of `{$ID, PropertyKey, ValueType: {Type, DefaultValue}}`
- `raw["Object"]["Properties"]` → array of `{TypePointer, Value: {PrimitiveValue, ...}}`

- [ ] **Step 2.1: Write failing tests**

```go
// mdl/executor/widget_schema_test.go
package executor

import (
	"testing"
)

func makeTestRaw(propTypes []map[string]any, props []map[string]any) map[string]any {
	return map[string]any{
		"$Type": "CustomWidgets$CustomWidget",
		"Name":  "testWidget",
		"Type": map[string]any{
			"ObjectType": map[string]any{
				"PropertyTypes": propTypes,
			},
		},
		"Object": map[string]any{
			"Properties": props,
		},
	}
}

func TestBuildSchemaMap_ExtractsKeyAndDefault(t *testing.T) {
	raw := makeTestRaw(
		[]map[string]any{
			{"$ID": "ptr1", "PropertyKey": "source", "ValueType": map[string]any{"Type": "Enumeration", "DefaultValue": "context"}},
		},
		nil,
	)
	schema := buildSchemaMap(raw)
	entry, ok := schema["ptr1"]
	if !ok {
		t.Fatal("ptr1 not found in schema")
	}
	if entry.Key != "source" { t.Errorf("Key = %q, want %q", entry.Key, "source") }
	if entry.DefaultValue != "context" { t.Errorf("DefaultValue = %q, want %q", entry.DefaultValue, "context") }
	if entry.ValueType != "Enumeration" { t.Errorf("ValueType = %q, want %q", entry.ValueType, "Enumeration") }
}

func TestFilterDefaults_SkipsMatchingPrimitive(t *testing.T) {
	schema := SchemaMap{"ptr1": SchemaEntry{Key: "source", DefaultValue: "context", ValueType: "Enumeration"}}
	props := []PropertyValue{{TypePointerID: "ptr1", PrimitiveValue: "context"}}
	result := filterDefaults(props, schema)
	if len(result) != 0 {
		t.Errorf("expected 0 non-default props, got %d", len(result))
	}
}

func TestFilterDefaults_OutputsNonDefaultPrimitive(t *testing.T) {
	schema := SchemaMap{"ptr1": SchemaEntry{Key: "source", DefaultValue: "context", ValueType: "Enumeration"}}
	props := []PropertyValue{{TypePointerID: "ptr1", PrimitiveValue: "datasource"}}
	result := filterDefaults(props, schema)
	if len(result) != 1 {
		t.Fatalf("expected 1 non-default prop, got %d", len(result))
	}
	if result[0].Key != "source" { t.Errorf("Key = %q, want %q", result[0].Key, "source") }
	if result[0].PrimitiveValue != "datasource" { t.Errorf("Value = %q", result[0].PrimitiveValue) }
}

func TestFilterDefaults_SkipsBoolDefaultFalse(t *testing.T) {
	schema := SchemaMap{"ptr1": SchemaEntry{Key: "enabled", DefaultValue: "false", ValueType: "Boolean"}}
	props := []PropertyValue{{TypePointerID: "ptr1", PrimitiveValue: "false"}}
	if len(filterDefaults(props, schema)) != 0 {
		t.Error("false should be skipped when default is false")
	}
}

func TestFilterDefaults_OutputsNonDefaultBool(t *testing.T) {
	schema := SchemaMap{"ptr1": SchemaEntry{Key: "enabled", DefaultValue: "false", ValueType: "Boolean"}}
	props := []PropertyValue{{TypePointerID: "ptr1", PrimitiveValue: "true"}}
	result := filterDefaults(props, schema)
	if len(result) != 1 { t.Fatalf("expected 1, got %d", len(result)) }
}

func TestFilterDefaults_SkipsEmptyStringDefault(t *testing.T) {
	schema := SchemaMap{"ptr1": SchemaEntry{Key: "label", DefaultValue: "", ValueType: "String"}}
	props := []PropertyValue{{TypePointerID: "ptr1", PrimitiveValue: ""}}
	if len(filterDefaults(props, schema)) != 0 {
		t.Error("empty string matching default should be skipped")
	}
}

func TestExtractWidgetTypeID_ReadsFromTypeField(t *testing.T) {
	raw := map[string]any{
		"Type": map[string]any{"WidgetId": "com.mendix.widget.web.datagrid2.DataGrid"},
	}
	got := extractWidgetTypeID(raw)
	want := "com.mendix.widget.web.datagrid2.DataGrid"
	if got != want { t.Errorf("got %q, want %q", got, want) }
}

func TestReadProperties_ExtractsPrimitiveValues(t *testing.T) {
	raw := makeTestRaw(nil, []map[string]any{
		{"TypePointer": "ptr1", "Value": map[string]any{"PrimitiveValue": "hello"}},
	})
	props := readProperties(raw)
	if len(props) != 1 { t.Fatalf("expected 1, got %d", len(props)) }
	if props[0].TypePointerID != "ptr1" { t.Errorf("TypePointerID = %q", props[0].TypePointerID) }
	if props[0].PrimitiveValue != "hello" { t.Errorf("PrimitiveValue = %q", props[0].PrimitiveValue) }
}
```

- [ ] **Step 2.2: Run tests to verify they fail**

```bash
go test ./mdl/executor/ -run "TestBuildSchemaMap|TestFilterDefaults|TestExtractWidgetTypeID|TestReadProperties" -v 2>&1 | head -10
```
Expected: compile errors.

- [ ] **Step 2.3: Implement `widget_schema.go`**

```go
// mdl/executor/widget_schema.go
package executor

import "fmt"

// ─── Types ───────────────────────────────────────────────────────────────────

// SchemaEntry describes one property type from a widget's Type BSON.
type SchemaEntry struct {
	Key          string // PropertyKey (e.g. "source", "attribute")
	DefaultValue string // from ValueType.DefaultValue
	ValueType    string // "Boolean" | "Integer" | "Enumeration" | "String" | "Expression" | "Object" | ...
}

// SchemaMap maps TypePointer ID strings to their schema entry.
type SchemaMap map[string]SchemaEntry

// PropertyValue is one entry from Object.Properties[].
type PropertyValue struct {
	TypePointerID  string
	Key            string // resolved from schema
	PrimitiveValue string
	ValueType      string // mirrors SchemaEntry.ValueType
}

// ─── buildSchemaMap ───────────────────────────────────────────────────────────

// buildSchemaMap reads Type.ObjectType.PropertyTypes from a CustomWidget raw map
// and returns a map from TypePointer $ID to SchemaEntry.
func buildSchemaMap(raw map[string]any) SchemaMap {
	result := make(SchemaMap)
	typeDoc, _ := raw["Type"].(map[string]any)
	if typeDoc == nil {
		return result
	}
	objType, _ := typeDoc["ObjectType"].(map[string]any)
	if objType == nil {
		return result
	}
	propTypes := getBsonArrayElements(objType["PropertyTypes"])
	for _, pt := range propTypes {
		ptMap, ok := pt.(map[string]any)
		if !ok {
			continue
		}
		id := extractBinaryID(ptMap["$ID"])
		key, _ := ptMap["PropertyKey"].(string)
		vt, _ := ptMap["ValueType"].(map[string]any)
		var defaultVal, valueType string
		if vt != nil {
			defaultVal, _ = vt["DefaultValue"].(string)
			valueType, _ = vt["Type"].(string)
		}
		if id != "" && key != "" {
			result[id] = SchemaEntry{Key: key, DefaultValue: defaultVal, ValueType: valueType}
		}
	}
	return result
}

// ─── readProperties ───────────────────────────────────────────────────────────

// readProperties reads Object.Properties[] from a CustomWidget raw map and
// returns all entries as PropertyValues. Schema is used to resolve Key names.
func readProperties(raw map[string]any) []PropertyValue {
	objDoc, _ := raw["Object"].(map[string]any)
	if objDoc == nil {
		return nil
	}
	propsRaw := getBsonArrayElements(objDoc["Properties"])
	result := make([]PropertyValue, 0, len(propsRaw))
	for _, pr := range propsRaw {
		prMap, ok := pr.(map[string]any)
		if !ok {
			continue
		}
		ptrID := extractBinaryID(prMap["TypePointer"])
		if ptrID == "" {
			continue
		}
		val, _ := prMap["Value"].(map[string]any)
		primVal := ""
		if val != nil {
			primVal, _ = val["PrimitiveValue"].(string)
		}
		result = append(result, PropertyValue{
			TypePointerID:  ptrID,
			PrimitiveValue: primVal,
		})
	}
	return result
}

// ─── filterDefaults ───────────────────────────────────────────────────────────

// filterDefaults returns only those PropertyValues whose PrimitiveValue differs
// from the default declared in the schema. It also populates Key and ValueType
// on each returned entry.
func filterDefaults(props []PropertyValue, schema SchemaMap) []PropertyValue {
	var result []PropertyValue
	for _, p := range props {
		entry, ok := schema[p.TypePointerID]
		if !ok {
			continue // skip properties not in schema
		}
		p.Key = entry.Key
		p.ValueType = entry.ValueType

		// Determine effective default
		def := entry.DefaultValue
		if def == "" {
			switch entry.ValueType {
			case "Boolean":
				def = "false"
			case "Integer":
				def = "0"
			}
		}
		if p.PrimitiveValue == def {
			continue // matches default, skip
		}
		result = append(result, p)
	}
	return result
}

// ─── formatPropertyValue ─────────────────────────────────────────────────────

// formatPropertyValue formats a PropertyValue's primitive as an MDL value string.
func formatPropertyValue(p PropertyValue) string {
	switch p.ValueType {
	case "Boolean", "Integer":
		return p.PrimitiveValue
	default:
		return fmt.Sprintf("'%s'", p.PrimitiveValue)
	}
}

// ─── extractWidgetTypeID ─────────────────────────────────────────────────────

// extractWidgetTypeID reads the widget ID from Type.WidgetId in a
// CustomWidgets$CustomWidget raw map. Returns "" if not present.
func extractWidgetTypeID(raw map[string]any) string {
	typeDoc, _ := raw["Type"].(map[string]any)
	if typeDoc == nil {
		return ""
	}
	id, _ := typeDoc["WidgetId"].(string)
	return id
}
```

- [ ] **Step 2.4: Run tests**

```bash
go test ./mdl/executor/ -run "TestBuildSchemaMap|TestFilterDefaults|TestExtractWidgetTypeID|TestReadProperties" -v
```
Expected: 8 tests PASS.

- [ ] **Step 2.5: Commit**

```bash
git add mdl/executor/widget_schema.go mdl/executor/widget_schema_test.go
git commit -m "feat(describe): add schema introspection utilities (buildSchemaMap, filterDefaults)"
```

---

### Task 3: `GenericPluggableFormatter` for unknown widgets

**Files:**
- Create: `mdl/executor/widget_fmt_pluggable.go`
- Create: `mdl/executor/widget_fmt_pluggable_test.go`

- [ ] **Step 3.1: Write failing tests**

```go
// mdl/executor/widget_fmt_pluggable_test.go
package executor

import (
	"bytes"
	"strings"
	"testing"
)

func TestGenericPluggableFormatter_NoNonDefaultProps(t *testing.T) {
	raw := map[string]any{
		"$Type": "CustomWidgets$CustomWidget",
		"Name":  "myWidget",
		"Type": map[string]any{
			"WidgetId": "com.co.widget.Foo",
			"ObjectType": map[string]any{
				"PropertyTypes": []map[string]any{
					{"$ID": "ptr1", "PropertyKey": "enabled", "ValueType": map[string]any{"Type": "Boolean", "DefaultValue": "false"}},
				},
			},
		},
		"Object": map[string]any{
			"Properties": []map[string]any{
				{"TypePointer": "ptr1", "Value": map[string]any{"PrimitiveValue": "false"}},
			},
		},
	}
	var buf bytes.Buffer
	d := newDefaultDispatcher()
	ctx := &FormatContext{Output: &buf, Indent: 1, Dispatcher: d}
	GenericPluggableFactory(raw).FormatMDL(ctx)

	got := buf.String()
	if !strings.Contains(got, "pluggablewidget 'com.co.widget.Foo' myWidget") {
		t.Errorf("expected pluggablewidget line, got: %q", got)
	}
	if strings.Contains(got, "enabled") {
		t.Error("default-value property should not appear in output")
	}
}

func TestGenericPluggableFormatter_OutputsNonDefaultProp(t *testing.T) {
	raw := map[string]any{
		"$Type": "CustomWidgets$CustomWidget",
		"Name":  "myWidget",
		"Type": map[string]any{
			"WidgetId": "com.co.widget.Foo",
			"ObjectType": map[string]any{
				"PropertyTypes": []map[string]any{
					{"$ID": "ptr1", "PropertyKey": "source", "ValueType": map[string]any{"Type": "Enumeration", "DefaultValue": "context"}},
				},
			},
		},
		"Object": map[string]any{
			"Properties": []map[string]any{
				{"TypePointer": "ptr1", "Value": map[string]any{"PrimitiveValue": "xpath"}},
			},
		},
	}
	var buf bytes.Buffer
	d := newDefaultDispatcher()
	ctx := &FormatContext{Output: &buf, Indent: 0, Dispatcher: d}
	GenericPluggableFactory(raw).FormatMDL(ctx)

	got := buf.String()
	if !strings.Contains(got, "source: 'xpath'") {
		t.Errorf("expected non-default prop in output, got: %q", got)
	}
}
```

- [ ] **Step 3.2: Implement `widget_fmt_pluggable.go`**

```go
// mdl/executor/widget_fmt_pluggable.go
package executor

import "fmt"

// GenericPluggableFormatter handles any CustomWidgets$CustomWidget whose widget
// ID is not registered in the dispatcher. It reads the embedded schema from the
// Type BSON, compares each property value against its declared default, and
// outputs only the non-default values. This ensures the output is the minimal
// sufficient MDL needed to recreate the widget.
type GenericPluggableFormatter struct{ raw map[string]any }

// GenericPluggableFactory is the FormatterFactory for GenericPluggableFormatter.
func GenericPluggableFactory(raw map[string]any) WidgetFormatter {
	return &GenericPluggableFormatter{raw: raw}
}

func (f *GenericPluggableFormatter) FormatMDL(ctx *FormatContext) {
	name := safeStr(f.raw, "Name")
	widgetID := extractWidgetTypeID(f.raw)
	if widgetID == "" {
		widgetID = "unknown"
	}

	schema := buildSchemaMap(f.raw)
	props := readProperties(f.raw)
	nonDef := filterDefaults(props, schema)

	if len(nonDef) == 0 {
		ctx.Write(fmt.Sprintf("pluggablewidget '%s' %s", widgetID, name))
		return
	}

	ctx.Write(fmt.Sprintf("pluggablewidget '%s' %s (", widgetID, name))
	for _, p := range nonDef {
		ctx.Child().Write(fmt.Sprintf("%s: %s,", p.Key, formatPropertyValue(p)))
	}
	ctx.Write(")")
}
```

- [ ] **Step 3.3: Run tests**

```bash
go test ./mdl/executor/ -run "TestGenericPluggableFormatter" -v
```
Expected: 2 tests PASS.

- [ ] **Step 3.4: Commit**

```bash
git add mdl/executor/widget_fmt_pluggable.go mdl/executor/widget_fmt_pluggable_test.go
git commit -m "feat(describe): add GenericPluggableFormatter with schema introspection"
```

---

### Task 4: Bridge dispatcher into `describePage()` with legacy fallback

During Phase 2 the dispatcher handles only registered types; the legacy path (`parseRawWidget` + `outputWidgetMDLV3`) is its fallback. This means describe output is unchanged while formatters are added incrementally.

**Files:**
- Create: `mdl/executor/widget_fmt_init.go` — default dispatcher singleton + registration
- Modify: `mdl/executor/cmd_pages_describe.go` — wire `getPageWidgetMaps` + dispatcher

- [ ] **Step 4.1: Add `widget_fmt_init.go`**

```go
// mdl/executor/widget_fmt_init.go
package executor

import "sync"

var (
	defaultDispatcherOnce sync.Once
	defaultDispatcherInst *FormatterDispatcher
)

// DefaultDispatcher returns the process-level formatter dispatcher.
// All widget formatters register themselves here via init() in their respective files.
func DefaultDispatcher() *FormatterDispatcher {
	defaultDispatcherOnce.Do(func() {
		d := newDefaultDispatcher()
		// Phase 1: only GenericPluggable as fallback; specific registrations
		// are added by init() calls in widget_fmt_*.go files.
		// The legacy bridge fallback (legacyWidgetFallback) is set by
		// cmd_pages_describe.go before first use and removed in Phase 3.
		defaultDispatcherInst = d
	})
	return defaultDispatcherInst
}
```

- [ ] **Step 4.2: Add `getPageWidgetMaps` to `cmd_pages_describe.go`**

Add this function just below `getPageWidgetsFromRaw` in `cmd_pages_describe.go`:

```go
// getPageWidgetMaps returns the top-level widget BSON maps for a page,
// unwrapping the conditionalVisibilityWidget container when present.
// Unlike getPageWidgetsFromRaw, this returns raw maps for the dispatcher.
func getPageWidgetMaps(ctx *ExecContext, pageID model.ID) []map[string]any {
	rawData, err := ctx.Backend.GetRawUnit(pageID)
	if err != nil {
		return nil
	}
	formCall, _ := rawData["FormCall"].(map[string]any)
	if formCall == nil {
		formCall, _ = rawData["LayoutCall"].(map[string]any)
	}
	if formCall == nil {
		return nil
	}
	args := getBsonArrayElements(formCall["Arguments"])
	var result []map[string]any
	for _, arg := range args {
		argMap, ok := arg.(map[string]any)
		if !ok {
			continue
		}
		var widgetMaps []map[string]any
		argWidgets := getBsonArrayElements(argMap["Widgets"])
		if argWidgets == nil {
			if w, ok := argMap["Widget"].(map[string]any); ok {
				argWidgets = []any{w}
			}
		}
		for _, w := range argWidgets {
			if wMap, ok := w.(map[string]any); ok {
				widgetMaps = append(widgetMaps, wMap)
			}
		}
		// Unwrap conditionalVisibilityWidget containers
		for _, wm := range widgetMaps {
			name := safeStr(wm, "Name")
			if len(name) > 27 && name[:27] == "conditionalVisibilityWidget" {
				for _, child := range getBsonArrayElements(wm["Widgets"]) {
					if cm, ok := child.(map[string]any); ok {
						result = append(result, cm)
					}
				}
			} else {
				result = append(result, wm)
			}
		}
	}
	return result
}
```

- [ ] **Step 4.3: Wire legacy fallback in `describePage()`**

In `cmd_pages_describe.go`, in the `describePage` function, find the block that calls `getPageWidgetsFromRaw` and `outputWidgetMDLV3`, and replace it with:

```go
	// Phase 1 bridge: use dispatcher for registered types, legacy for the rest.
	// In Phase 3 this block is replaced with pure dispatcher calls.
	rawWidgetMaps := getPageWidgetMaps(ctx, pageID)
	d := DefaultDispatcher()
	// Set legacy fallback so unregistered types still produce correct output.
	d.fallback = func(raw map[string]any) WidgetFormatter {
		return &legacyWidgetFallback{raw: raw, ctx: ctx}
	}
	if len(rawWidgetMaps) > 0 {
		fmtCtx := &FormatContext{Output: ctx.Output, Indent: 1, Dispatcher: d}
		formatWidgetProps(ctx.Output, "", header, props, " {\n")
		for _, wm := range rawWidgetMaps {
			d.Format(fmtCtx, wm)
		}
		fmt.Fprint(ctx.Output, "}")
	} else {
		formatWidgetProps(ctx.Output, "", header, props, " {\n}")
	}
```

Add `legacyWidgetFallback` just below:

```go
// legacyWidgetFallback wraps the legacy parse+output pipeline as a WidgetFormatter.
// Used during Phase 2 migration. Removed in Phase 3.
type legacyWidgetFallback struct {
	raw map[string]any
	ctx *ExecContext
}

func (f *legacyWidgetFallback) FormatMDL(ctx *FormatContext) {
	widgets := parseRawWidget(f.ctx, f.raw)
	for _, w := range widgets {
		outputWidgetMDLV3(f.ctx, w, ctx.Indent)
	}
}
```

- [ ] **Step 4.4: Build and verify existing tests still pass**

```bash
go build ./mdl/executor/ && \
go test ./mdl/executor/ -run "TestHelpdeskGolden_DescribeSnapshot" -tags linux,integration -timeout 2m -v 2>&1 | tail -5
```
Expected: test PASS (output unchanged — all widgets still go through legacy fallback).

- [ ] **Step 4.5: Commit**

```bash
git add mdl/executor/widget_fmt_init.go mdl/executor/cmd_pages_describe.go
git commit -m "feat(describe): wire dispatcher bridge with legacy fallback (Phase 1 complete)"
```

---

### Task 5: Fix `extractExplicitProperties()` to use schema DefaultValue

Currently this function hardcodes `"true"/"false"` checks. Replace with schema-based comparison.

**Files:**
- Modify: `mdl/executor/cmd_pages_describe_pluggable.go` (function at line ~1071)

- [ ] **Step 5.1: Write a test that fails with current code**

Add to `mdl/executor/cmd_pages_describe_pluggable_test.go` (or create it):

```go
// mdl/executor/widget_schema_extractexplicit_test.go
package executor

import (
	"testing"
)

// TestExtractExplicitProperties_UsesSchemaDefault verifies that a non-boolean
// property matching its schema default is suppressed.
func TestExtractExplicitProperties_UsesSchemaDefault(t *testing.T) {
	raw := makeTestRaw(
		[]map[string]any{
			{"$ID": "ptr1", "PropertyKey": "size", "ValueType": map[string]any{"Type": "Integer", "DefaultValue": "10"}},
		},
		[]map[string]any{
			{"TypePointer": "ptr1", "Value": map[string]any{"PrimitiveValue": "10"}}, // equals default
		},
	)
	// extractExplicitProperties should return nothing (default value)
	result := extractExplicitPropertiesFromRaw(raw)
	if len(result) != 0 {
		t.Errorf("expected 0 explicit props, got %d: %v", len(result), result)
	}
}
```

- [ ] **Step 5.2: Add `extractExplicitPropertiesFromRaw` that uses schema**

In `cmd_pages_describe_pluggable.go`, add this function (replaces the existing hardcoded logic):

```go
// extractExplicitPropertiesFromRaw returns the non-default ExplicitProperties
// for a CustomWidget, using the embedded Type schema to determine defaults.
// This replaces the old hardcoded true/false comparison.
func extractExplicitPropertiesFromRaw(raw map[string]any) []rawExplicitProp {
	schema := buildSchemaMap(raw)
	props := readProperties(raw)
	nonDef := filterDefaults(props, schema)

	result := make([]rawExplicitProp, 0, len(nonDef))
	for _, p := range nonDef {
		result = append(result, rawExplicitProp{
			Key:   p.Key,
			Value: formatPropertyValue(p),
		})
	}
	return result
}
```

Update `extractExplicitProperties` (the existing function) to delegate:

```go
func extractExplicitProperties(ctx *ExecContext, w map[string]any) []rawExplicitProp {
	return extractExplicitPropertiesFromRaw(w)
}
```

- [ ] **Step 5.3: Run tests**

```bash
go test ./mdl/executor/ -run "TestExtractExplicitProperties" -v
```
Expected: new test PASS; existing tests unaffected.

- [ ] **Step 5.4: Commit**

```bash
git add mdl/executor/cmd_pages_describe_pluggable.go \
        mdl/executor/widget_schema_extractexplicit_test.go
git commit -m "fix(describe): use schema DefaultValue in extractExplicitProperties (Phase 1)"
```

---

## Phase 2 — Per-Type Formatters

**Pattern for each formatter task:**
1. Add test in `widget_fmt_*_test.go` with a minimal raw BSON map and expected MDL substring
2. Implement formatter struct + factory + `FormatMDL`
3. Register in file's `init()` function
4. Run test + run golden test to verify output unchanged
5. Commit

The test helper used by all formatter tests:

- [ ] **Step P2.0: Create shared test helper**

```go
// mdl/executor/widget_fmt_test_helpers_test.go
package executor

import (
	"bytes"
	"testing"
)

func newTestFormatCtx(t *testing.T) (*FormatContext, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	d := newDefaultDispatcher()
	return &FormatContext{Output: &buf, Indent: 0, Dispatcher: d}, &buf
}

func assertOutput(t *testing.T, buf *bytes.Buffer, want string) {
	t.Helper()
	got := buf.String()
	if !containsStr(got, want) {
		t.Errorf("output missing %q\nfull output:\n%s", want, got)
	}
}

func assertNotOutput(t *testing.T, buf *bytes.Buffer, notWant string) {
	t.Helper()
	if containsStr(buf.String(), notWant) {
		t.Errorf("output should not contain %q\nfull output:\n%s", notWant, buf.String())
	}
}

func containsStr(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && strings.Contains(s, sub))
}
```

```bash
git add mdl/executor/widget_fmt_test_helpers_test.go
git commit -m "test(describe): add shared formatter test helpers"
```

---

### Task 6: Basic leaf widget formatters (`widget_fmt_basic.go`)

Covers: ActionButton, DynamicText, TextBox, TextArea, DatePicker, RadioButtons, CheckBox, Label/StaticText, Title, SnippetCallWidget.

**Files:**
- Create: `mdl/executor/widget_fmt_basic.go`
- Create: `mdl/executor/widget_fmt_basic_test.go`

- [ ] **Step 6.1: Write tests**

```go
// mdl/executor/widget_fmt_basic_test.go
package executor

import (
	"testing"
)

func TestActionButtonFormatter_PrimaryStyle(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := map[string]any{
		"$Type":       "Forms$ActionButton",
		"Name":        "btnSubmit",
		"ButtonStyle": "primary",
		// CaptionTemplate omitted → empty caption
	}
	basicFormatters[0].factory(raw).FormatMDL(ctx) // ActionButton factory
	assertOutput(t, buf, "actionbutton btnSubmit")
	assertOutput(t, buf, "buttonstyle: primary")
	assertNotOutput(t, buf, "buttonstyle: Default")
}

func TestActionButtonFormatter_DefaultStyleSuppressed(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := map[string]any{
		"$Type":       "Forms$ActionButton",
		"Name":        "btnCancel",
		"ButtonStyle": "Default",
	}
	basicFormatters[0].factory(raw).FormatMDL(ctx)
	assertNotOutput(t, buf, "buttonstyle")
}

func TestTextBoxFormatter_AttributeAndLabel(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := map[string]any{
		"$Type": "Forms$TextBox",
		"Name":  "tbName",
		// Label and Content omitted for minimal test
	}
	textBoxFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "textbox tbName")
}

func TestCheckBoxFormatter_NonDefaultEditable(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := map[string]any{
		"$Type":    "Forms$CheckBox",
		"Name":     "cbActive",
		"Editable": "Never",
	}
	checkBoxFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "checkbox cbActive")
	assertOutput(t, buf, "editable: Never")
}
```

- [ ] **Step 6.2: Implement `widget_fmt_basic.go`**

```go
// mdl/executor/widget_fmt_basic.go
package executor

import (
	"fmt"
	"strings"
)

func init() {
	d := DefaultDispatcher()
	for _, bsonType := range []string{"Forms$ActionButton", "Pages$ActionButton"} {
		d.registerBSONType(bsonType, FactoryEntry{Factory: actionButtonFactory})
	}
	for _, bsonType := range []string{"Forms$DynamicText", "Pages$DynamicText"} {
		d.registerBSONType(bsonType, FactoryEntry{Factory: dynamicTextFactory})
	}
	for _, bsonType := range []string{"Forms$TextBox", "Pages$TextBox"} {
		d.registerBSONType(bsonType, FactoryEntry{Factory: textBoxFactory})
	}
	for _, bsonType := range []string{"Forms$TextArea", "Pages$TextArea"} {
		d.registerBSONType(bsonType, FactoryEntry{Factory: textAreaFactory})
	}
	for _, bsonType := range []string{"Forms$DatePicker", "Pages$DatePicker"} {
		d.registerBSONType(bsonType, FactoryEntry{Factory: datePickerFactory})
	}
	for _, bsonType := range []string{"Forms$RadioButtons", "Pages$RadioButtons", "Forms$RadioButtonGroup", "Pages$RadioButtonGroup"} {
		d.registerBSONType(bsonType, FactoryEntry{Factory: radioButtonsFactory})
	}
	for _, bsonType := range []string{"Forms$CheckBox", "Pages$CheckBox"} {
		d.registerBSONType(bsonType, FactoryEntry{Factory: checkBoxFactory})
	}
	for _, bsonType := range []string{"Forms$Label", "Pages$Label"} {
		d.registerBSONType(bsonType, FactoryEntry{Factory: labelFactory})
	}
	for _, bsonType := range []string{"Forms$Text", "Pages$Text"} {
		d.registerBSONType(bsonType, FactoryEntry{Factory: staticTextFactory})
	}
	for _, bsonType := range []string{"Forms$Title", "Pages$Title"} {
		d.registerBSONType(bsonType, FactoryEntry{Factory: titleFactory})
	}
	for _, bsonType := range []string{"Forms$SnippetCallWidget", "Pages$SnippetCallWidget"} {
		d.registerBSONType(bsonType, FactoryEntry{Factory: snippetCallFactory})
	}
}

// basicFormatters is exposed only for testing the factory slice ordering.
// Production code uses DefaultDispatcher() registrations.
var basicFormatters = []struct{ factory FormatterFactory }{
	{factory: actionButtonFactory},
}

// ─── ActionButton ─────────────────────────────────────────────────────────────

type actionButtonFormatter struct{ raw map[string]any }

func actionButtonFactory(raw map[string]any) WidgetFormatter {
	return &actionButtonFormatter{raw: raw}
}

func (f *actionButtonFormatter) FormatMDL(ctx *FormatContext) {
	name := safeStr(f.raw, "Name")
	caption := extractButtonCaption(nil, f.raw)   // reuse existing helper
	params := extractButtonCaptionParameters(nil, f.raw)
	action := extractButtonAction(nil, f.raw)
	style := safeStr(f.raw, "ButtonStyle")

	var props []string
	if caption != "" {
		props = append(props, fmt.Sprintf("caption: %s", mdlQuote(caption)))
	}
	if len(params) > 0 {
		props = append(props, fmt.Sprintf("contentparams: [%s]", strings.Join(formatParametersV3(params), ", ")))
	}
	if action != "" {
		props = append(props, fmt.Sprintf("action: %s", action))
	}
	if style != "" && style != "Default" {
		props = append(props, fmt.Sprintf("buttonstyle: %s", strings.ToLower(style)))
	}
	props = appendAppearanceProps(props, rawWidgetFromMap(f.raw))
	writeWidgetLine(ctx, "actionbutton "+name, props)
}

// ─── DynamicText ──────────────────────────────────────────────────────────────

type dynamicTextFormatter struct{ raw map[string]any }

func dynamicTextFactory(raw map[string]any) WidgetFormatter { return &dynamicTextFormatter{raw: raw} }

func (f *dynamicTextFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	var props []string
	if w.Content != "" {
		props = append(props, fmt.Sprintf("content: %s", mdlQuote(w.Content)))
	}
	if w.RenderMode != "" && w.RenderMode != "Text" {
		props = append(props, fmt.Sprintf("rendermode: %s", w.RenderMode))
	}
	if len(w.Parameters) > 0 {
		props = append(props, fmt.Sprintf("contentparams: [%s]", strings.Join(formatParametersV3(w.Parameters), ", ")))
	}
	props = appendAppearanceProps(props, w)
	writeWidgetLine(ctx, "dynamictext "+w.Name, props)
}

// ─── TextBox ──────────────────────────────────────────────────────────────────

type textBoxFormatter struct{ raw map[string]any }

func textBoxFactory(raw map[string]any) WidgetFormatter { return &textBoxFormatter{raw: raw} }

func (f *textBoxFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	var props []string
	if w.Caption != "" {
		props = append(props, fmt.Sprintf("label: %s", mdlQuote(w.Caption)))
	}
	if w.Content != "" {
		props = append(props, fmt.Sprintf("attribute: %s", w.Content))
	}
	props = appendAppearanceProps(props, w)
	writeWidgetLine(ctx, "textbox "+w.Name, props)
}

// ─── TextArea ─────────────────────────────────────────────────────────────────

type textAreaFormatter struct{ raw map[string]any }

func textAreaFactory(raw map[string]any) WidgetFormatter { return &textAreaFormatter{raw: raw} }

func (f *textAreaFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	var props []string
	if w.Caption != "" {
		props = append(props, fmt.Sprintf("label: %s", mdlQuote(w.Caption)))
	}
	if w.Content != "" {
		props = append(props, fmt.Sprintf("attribute: %s", w.Content))
	}
	props = appendAppearanceProps(props, w)
	writeWidgetLine(ctx, "textarea "+w.Name, props)
}

// ─── DatePicker ───────────────────────────────────────────────────────────────

type datePickerFormatter struct{ raw map[string]any }

func datePickerFactory(raw map[string]any) WidgetFormatter { return &datePickerFormatter{raw: raw} }

func (f *datePickerFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	var props []string
	if w.Caption != "" {
		props = append(props, fmt.Sprintf("label: %s", mdlQuote(w.Caption)))
	}
	if w.Content != "" {
		props = append(props, fmt.Sprintf("attribute: %s", w.Content))
	}
	props = appendAppearanceProps(props, w)
	writeWidgetLine(ctx, "datepicker "+w.Name, props)
}

// ─── RadioButtons ─────────────────────────────────────────────────────────────

type radioButtonsFormatter struct{ raw map[string]any }

func radioButtonsFactory(raw map[string]any) WidgetFormatter { return &radioButtonsFormatter{raw: raw} }

func (f *radioButtonsFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	var props []string
	if w.Caption != "" {
		props = append(props, fmt.Sprintf("label: %s", mdlQuote(w.Caption)))
	}
	if w.Content != "" {
		props = append(props, fmt.Sprintf("attribute: %s", w.Content))
	}
	props = appendAppearanceProps(props, w)
	writeWidgetLine(ctx, "radiobuttons "+w.Name, props)
}

// ─── CheckBox ─────────────────────────────────────────────────────────────────

type checkBoxFormatter struct{ raw map[string]any }

func checkBoxFactory(raw map[string]any) WidgetFormatter { return &checkBoxFormatter{raw: raw} }

func (f *checkBoxFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	var props []string
	if w.Caption != "" {
		props = append(props, fmt.Sprintf("label: %s", mdlQuote(w.Caption)))
	}
	if w.Content != "" {
		props = append(props, fmt.Sprintf("attribute: %s", w.Content))
	}
	if w.Editable != "" {
		props = append(props, fmt.Sprintf("editable: %s", w.Editable))
	}
	if w.ReadOnlyStyle != "" {
		props = append(props, fmt.Sprintf("readonlystyle: %s", w.ReadOnlyStyle))
	}
	if w.ShowLabel {
		props = append(props, "showlabel: true")
	}
	props = appendAppearanceProps(props, w)
	writeWidgetLine(ctx, "checkbox "+w.Name, props)
}

// ─── Label / StaticText / Title ───────────────────────────────────────────────

type labelFormatter struct{ raw map[string]any }

func labelFactory(raw map[string]any) WidgetFormatter { return &labelFormatter{raw: raw} }

func (f *labelFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	var props []string
	if w.Content != "" {
		props = append(props, fmt.Sprintf("content: %s", mdlQuote(w.Content)))
	}
	props = appendAppearanceProps(props, w)
	writeWidgetLine(ctx, "statictext", props)
}

type staticTextFormatter struct{ raw map[string]any }

func staticTextFactory(raw map[string]any) WidgetFormatter { return &staticTextFormatter{raw: raw} }

func (f *staticTextFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	var props []string
	if w.Content != "" {
		props = append(props, fmt.Sprintf("content: %s", mdlQuote(w.Content)))
	}
	props = appendAppearanceProps(props, w)
	writeWidgetLine(ctx, "statictext", props)
}

type titleFormatter struct{ raw map[string]any }

func titleFactory(raw map[string]any) WidgetFormatter { return &titleFormatter{raw: raw} }

func (f *titleFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	var props []string
	if w.Caption != "" {
		props = append(props, fmt.Sprintf("content: %s", mdlQuote(w.Caption)))
	}
	props = appendAppearanceProps(props, w)
	writeWidgetLine(ctx, "title "+w.Name, props)
}

// ─── SnippetCallWidget ────────────────────────────────────────────────────────

type snippetCallFormatter struct{ raw map[string]any }

func snippetCallFactory(raw map[string]any) WidgetFormatter { return &snippetCallFormatter{raw: raw} }

func (f *snippetCallFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	var props []string
	if w.Content != "" {
		props = append(props, fmt.Sprintf("snippet: %s", w.Content))
	}
	props = appendAppearanceProps(props, w)
	writeWidgetLine(ctx, "snippetcall "+w.Name, props)
}

// ─── Helpers used by basic formatters ────────────────────────────────────────

// rawWidgetFromMap converts a raw BSON map to a rawWidget by delegating to the
// existing parseRawWidget. Used during Phase 2 to reuse legacy field extraction.
// Removed in Phase 3 when rawWidget is deleted.
func rawWidgetFromMap(raw map[string]any) rawWidget {
	widgets := parseRawWidget(nil, raw)
	if len(widgets) == 0 {
		return rawWidget{Type: safeStr(raw, "$Type"), Name: safeStr(raw, "Name")}
	}
	return widgets[0]
}

// writeWidgetLine writes a widget props line to ctx using the existing
// formatWidgetProps helper (reused from old output code during Phase 2).
func writeWidgetLine(ctx *FormatContext, header string, props []string) {
	formatWidgetProps(ctx.Output, indentStr(ctx.Indent), header, props, "\n")
}
```

- [ ] **Step 6.3: Run unit tests**

```bash
go test ./mdl/executor/ -run "TestActionButtonFormatter|TestTextBoxFormatter|TestCheckBoxFormatter" -v
```
Expected: tests PASS.

- [ ] **Step 6.4: Run golden describe snapshot test**

```bash
HELPDESK_VERSION=11.6.6 go test ./internal/goldenfs/ \
  -tags linux,integration -run TestHelpdeskGolden_DescribeSnapshot -timeout 2m -v 2>&1 | tail -5
```
Expected: PASS (output unchanged — registered formatters produce same output as legacy fallback).

- [ ] **Step 6.5: Commit**

```bash
git add mdl/executor/widget_fmt_basic.go mdl/executor/widget_fmt_basic_test.go
git commit -m "feat(describe): add basic leaf widget formatters (ActionButton, TextBox, etc.)"
```

---

### Task 7: Container formatters (`widget_fmt_container.go`)

Covers: DivContainer, GroupBox, ScrollContainer, TabControl/TabPage, Footer (synthetic).

**Files:**
- Create: `mdl/executor/widget_fmt_container.go`
- Create: `mdl/executor/widget_fmt_container_test.go`

- [ ] **Step 7.1: Write tests**

```go
// mdl/executor/widget_fmt_container_test.go
package executor

import "testing"

func TestDivContainerFormatter_RendersChildrenIndented(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := map[string]any{
		"$Type":   "Forms$DivContainer",
		"Name":    "ctnMain",
		"Widgets": []any{
			map[string]any{"$Type": "Forms$Label", "Name": "lbl1"},
		},
	}
	divContainerFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "container ctnMain")
	assertOutput(t, buf, "statictext") // child rendered inside block
}

func TestGroupBoxFormatter_WithCaption(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := map[string]any{
		"$Type":   "Forms$GroupBox",
		"Name":    "gbSection",
		"Caption": "My Section",
	}
	groupBoxFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "groupbox gbSection")
	assertOutput(t, buf, "caption: 'My Section'")
}
```

- [ ] **Step 7.2: Implement `widget_fmt_container.go`**

```go
// mdl/executor/widget_fmt_container.go
package executor

import "fmt"

func init() {
	d := DefaultDispatcher()
	for _, t := range []string{"Forms$DivContainer", "Pages$DivContainer"} {
		d.registerBSONType(t, FactoryEntry{Factory: divContainerFactory})
	}
	for _, t := range []string{"Forms$GroupBox", "Pages$GroupBox"} {
		d.registerBSONType(t, FactoryEntry{Factory: groupBoxFactory})
	}
	for _, t := range []string{"Forms$ScrollContainer", "Pages$ScrollContainer"} {
		d.registerBSONType(t, FactoryEntry{Factory: scrollContainerFactory})
	}
	for _, t := range []string{"Forms$TabControl", "Pages$TabControl"} {
		d.registerBSONType(t, FactoryEntry{Factory: tabControlFactory})
	}
	d.registerBSONType("Pages$TabPage", FactoryEntry{Factory: tabPageFactory})
}

type divContainerFormatter struct{ raw map[string]any }

func divContainerFactory(raw map[string]any) WidgetFormatter { return &divContainerFormatter{raw: raw} }

func (f *divContainerFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	props := appendAppearanceProps(nil, w)
	formatWidgetProps(ctx.Output, indentStr(ctx.Indent), "container "+w.Name, props, " {\n")
	for _, child := range w.Children {
		childRaw := rawWidgetToMap(child)
		ctx.Dispatcher.Format(ctx.Child(), childRaw)
	}
	fmt.Fprintf(ctx.Output, "%s}\n", indentStr(ctx.Indent))
}

type groupBoxFormatter struct{ raw map[string]any }

func groupBoxFactory(raw map[string]any) WidgetFormatter { return &groupBoxFormatter{raw: raw} }

func (f *groupBoxFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	var props []string
	if w.Caption != "" {
		props = append(props, fmt.Sprintf("caption: %s", mdlQuote(w.Caption)))
	}
	if w.Collapsible != "" {
		props = append(props, fmt.Sprintf("collapsible: %s", w.Collapsible))
	}
	if w.HeaderMode != "" {
		props = append(props, fmt.Sprintf("headermode: %s", w.HeaderMode))
	}
	props = appendAppearanceProps(props, w)
	formatWidgetProps(ctx.Output, indentStr(ctx.Indent), "groupbox "+w.Name, props, " {\n")
	for _, child := range w.Children {
		ctx.Dispatcher.Format(ctx.Child(), rawWidgetToMap(child))
	}
	fmt.Fprintf(ctx.Output, "%s}\n", indentStr(ctx.Indent))
}

type scrollContainerFormatter struct{ raw map[string]any }

func scrollContainerFactory(raw map[string]any) WidgetFormatter {
	return &scrollContainerFormatter{raw: raw}
}

func (f *scrollContainerFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	props := appendAppearanceProps(nil, w)
	formatWidgetProps(ctx.Output, indentStr(ctx.Indent), "scrollcontainer "+w.Name, props, " {\n")
	for _, child := range w.Children {
		ctx.Dispatcher.Format(ctx.Child(), rawWidgetToMap(child))
	}
	fmt.Fprintf(ctx.Output, "%s}\n", indentStr(ctx.Indent))
}

type tabControlFormatter struct{ raw map[string]any }

func tabControlFactory(raw map[string]any) WidgetFormatter { return &tabControlFormatter{raw: raw} }

func (f *tabControlFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	props := appendAppearanceProps(nil, w)
	formatWidgetProps(ctx.Output, indentStr(ctx.Indent), "tabcontainer "+w.Name, props, " {\n")
	for _, child := range w.Children {
		ctx.Dispatcher.Format(ctx.Child(), rawWidgetToMap(child))
	}
	fmt.Fprintf(ctx.Output, "%s}\n", indentStr(ctx.Indent))
}

type tabPageFormatter struct{ raw map[string]any }

func tabPageFactory(raw map[string]any) WidgetFormatter { return &tabPageFormatter{raw: raw} }

func (f *tabPageFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	var props []string
	if w.TabCaption != "" {
		props = append(props, fmt.Sprintf("caption: %s", mdlQuote(w.TabCaption)))
	}
	props = appendAppearanceProps(props, w)
	formatWidgetProps(ctx.Output, indentStr(ctx.Indent), "tabpage "+w.Name, props, " {\n")
	for _, child := range w.Children {
		ctx.Dispatcher.Format(ctx.Child(), rawWidgetToMap(child))
	}
	fmt.Fprintf(ctx.Output, "%s}\n", indentStr(ctx.Indent))
}

// rawWidgetToMap is the reverse of rawWidgetFromMap: converts rawWidget back to
// a minimal raw map for dispatcher dispatch. Used during Phase 2 transition.
// Removed in Phase 3 when rawWidget is no longer used.
func rawWidgetToMap(w rawWidget) map[string]any {
	return map[string]any{"$Type": w.Type, "Name": w.Name, "_rawWidget": w}
}
```

Note: `rawWidgetToMap` stores the `rawWidget` in `_rawWidget` so that `rawWidgetFromMap` can retrieve it without re-parsing. Add this check to `rawWidgetFromMap`:

```go
func rawWidgetFromMap(raw map[string]any) rawWidget {
	if w, ok := raw["_rawWidget"].(rawWidget); ok {
		return w
	}
	widgets := parseRawWidget(nil, raw)
	if len(widgets) == 0 {
		return rawWidget{Type: safeStr(raw, "$Type"), Name: safeStr(raw, "Name")}
	}
	return widgets[0]
}
```

- [ ] **Step 7.3: Run tests**

```bash
go test ./mdl/executor/ -run "TestDivContainerFormatter|TestGroupBoxFormatter" -v
```
Expected: PASS.

- [ ] **Step 7.4: Run golden test**

```bash
HELPDESK_VERSION=11.6.6 go test ./internal/goldenfs/ \
  -tags linux,integration -run TestHelpdeskGolden_DescribeSnapshot -timeout 2m 2>&1 | tail -3
```
Expected: PASS.

- [ ] **Step 7.5: Commit**

```bash
git add mdl/executor/widget_fmt_container.go mdl/executor/widget_fmt_container_test.go \
        mdl/executor/widget_fmt_basic.go
git commit -m "feat(describe): add container widget formatters (DivContainer, GroupBox, TabControl)"
```

---

### Task 8: LayoutGrid formatter (`widget_fmt_layout.go`)

**Files:**
- Create: `mdl/executor/widget_fmt_layout.go`
- Create: `mdl/executor/widget_fmt_layout_test.go`

- [ ] **Step 8.1: Write test**

```go
// mdl/executor/widget_fmt_layout_test.go
package executor

import "testing"

func TestLayoutGridFormatter_EmitsRowsAndColumns(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := map[string]any{
		"$Type": "Forms$LayoutGrid",
		"Name":  "lgMain",
	}
	layoutGridFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "layoutgrid lgMain")
}
```

- [ ] **Step 8.2: Implement**

```go
// mdl/executor/widget_fmt_layout.go
package executor

import "fmt"

func init() {
	d := DefaultDispatcher()
	for _, t := range []string{"Forms$LayoutGrid", "Pages$LayoutGrid"} {
		d.registerBSONType(t, FactoryEntry{Factory: layoutGridFactory})
	}
}

type layoutGridFormatter struct{ raw map[string]any }

func layoutGridFactory(raw map[string]any) WidgetFormatter { return &layoutGridFormatter{raw: raw} }

func (f *layoutGridFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	props := appendAppearanceProps(nil, w)
	formatWidgetProps(ctx.Output, indentStr(ctx.Indent), "layoutgrid "+w.Name, props, " {\n")
	for _, row := range w.Rows {
		fmt.Fprintf(ctx.Output, "%srow %s {\n", indentStr(ctx.Indent+1), row.Name)
		for _, col := range row.Columns {
			var colProps []string
			if col.DesktopWidth > 0 {
				colProps = append(colProps, fmt.Sprintf("DesktopWidth: %d", col.DesktopWidth))
			}
			if col.TabletWidth != "" {
				colProps = append(colProps, fmt.Sprintf("TabletWidth: %s", col.TabletWidth))
			}
			if col.PhoneWidth != "" {
				colProps = append(colProps, fmt.Sprintf("PhoneWidth: %s", col.PhoneWidth))
			}
			formatWidgetProps(ctx.Output, indentStr(ctx.Indent+2), "column "+col.Name, colProps, " {\n")
			for _, child := range col.Children {
				ctx.Dispatcher.Format(ctx.withIndent(ctx.Indent+3), rawWidgetToMap(child))
			}
			fmt.Fprintf(ctx.Output, "%s}\n", indentStr(ctx.Indent+2))
		}
		fmt.Fprintf(ctx.Output, "%s}\n", indentStr(ctx.Indent+1))
	}
	fmt.Fprintf(ctx.Output, "%s}\n", indentStr(ctx.Indent))
}
```

Add `withIndent` helper to `FormatContext`:

```go
func (ctx *FormatContext) withIndent(n int) *FormatContext {
	return &FormatContext{Output: ctx.Output, Indent: n, Dispatcher: ctx.Dispatcher}
}
```

- [ ] **Step 8.3: Run tests + golden**

```bash
go test ./mdl/executor/ -run "TestLayoutGridFormatter" -v
HELPDESK_VERSION=11.6.6 go test ./internal/goldenfs/ \
  -tags linux,integration -run TestHelpdeskGolden_DescribeSnapshot -timeout 2m 2>&1 | tail -3
```
Expected: both PASS.

- [ ] **Step 8.4: Commit**

```bash
git add mdl/executor/widget_fmt_layout.go mdl/executor/widget_fmt_layout_test.go \
        mdl/executor/widget_formatter.go
git commit -m "feat(describe): add LayoutGrid formatter"
```

---

### Task 9: Data container formatters (`widget_fmt_data.go`)

Covers: DataView, ListView, NavigationList.

**Files:**
- Create: `mdl/executor/widget_fmt_data.go`
- Create: `mdl/executor/widget_fmt_data_test.go`

- [ ] **Step 9.1: Write tests**

```go
// mdl/executor/widget_fmt_data_test.go
package executor

import "testing"

func TestDataViewFormatter_EmitsDataviewKeyword(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := map[string]any{
		"$Type": "Forms$DataView",
		"Name":  "dvTicket",
	}
	dataViewFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "dataview dvTicket")
}

func TestListViewFormatter_EmitsListviewKeyword(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := map[string]any{"$Type": "Forms$ListView", "Name": "lvItems"}
	listViewFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "listview lvItems")
}
```

- [ ] **Step 9.2: Implement `widget_fmt_data.go`**

```go
// mdl/executor/widget_fmt_data.go
package executor

import "fmt"

func init() {
	d := DefaultDispatcher()
	for _, t := range []string{"Forms$DataView", "Pages$DataView"} {
		d.registerBSONType(t, FactoryEntry{Factory: dataViewFactory})
	}
	for _, t := range []string{"Forms$ListView", "Pages$ListView"} {
		d.registerBSONType(t, FactoryEntry{Factory: listViewFactory})
	}
	for _, t := range []string{"Forms$NavigationList", "Pages$NavigationList"} {
		d.registerBSONType(t, FactoryEntry{Factory: navigationListFactory})
	}
}

type dataViewFormatter struct{ raw map[string]any }

func dataViewFactory(raw map[string]any) WidgetFormatter { return &dataViewFormatter{raw: raw} }

func (f *dataViewFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	var props []string
	if w.DataSource != nil {
		props = append(props, fmt.Sprintf("datasource: %s", formatDataSourceV3(w.DataSource)))
	}
	props = appendAppearanceProps(props, w)
	formatWidgetProps(ctx.Output, indentStr(ctx.Indent), "dataview "+w.Name, props, " {\n")
	if w.EntityContext != "" {
		outputDataContainerContext(ctx.Output, indentStr(ctx.Indent+1)+"  ", w.Name, w.EntityContext, false)
	}
	for _, child := range w.Children {
		ctx.Dispatcher.Format(ctx.Child(), rawWidgetToMap(child))
	}
	fmt.Fprintf(ctx.Output, "%s}\n", indentStr(ctx.Indent))
}

type listViewFormatter struct{ raw map[string]any }

func listViewFactory(raw map[string]any) WidgetFormatter { return &listViewFormatter{raw: raw} }

func (f *listViewFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	var props []string
	if w.DataSource != nil {
		props = append(props, fmt.Sprintf("datasource: %s", formatDataSourceV3(w.DataSource)))
	}
	props = appendAppearanceProps(props, w)
	formatWidgetProps(ctx.Output, indentStr(ctx.Indent), "listview "+w.Name, props, " {\n")
	if w.EntityContext != "" {
		outputDataContainerContext(ctx.Output, indentStr(ctx.Indent+1)+"  ", w.Name, w.EntityContext, false)
	}
	for _, child := range w.Children {
		ctx.Dispatcher.Format(ctx.Child(), rawWidgetToMap(child))
	}
	fmt.Fprintf(ctx.Output, "%s}\n", indentStr(ctx.Indent))
}

type navigationListFormatter struct{ raw map[string]any }

func navigationListFactory(raw map[string]any) WidgetFormatter {
	return &navigationListFormatter{raw: raw}
}

func (f *navigationListFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	props := appendAppearanceProps(nil, w)
	formatWidgetProps(ctx.Output, indentStr(ctx.Indent), "navigationlist "+w.Name, props, " {\n")
	for _, child := range w.Children {
		ctx.Dispatcher.Format(ctx.Child(), rawWidgetToMap(child))
	}
	fmt.Fprintf(ctx.Output, "%s}\n", indentStr(ctx.Indent))
}
```

- [ ] **Step 9.3: Run tests + golden**

```bash
go test ./mdl/executor/ -run "TestDataViewFormatter|TestListViewFormatter" -v
HELPDESK_VERSION=11.6.6 go test ./internal/goldenfs/ \
  -tags linux,integration -run TestHelpdeskGolden_DescribeSnapshot -timeout 2m 2>&1 | tail -3
```

- [ ] **Step 9.4: Commit**

```bash
git add mdl/executor/widget_fmt_data.go mdl/executor/widget_fmt_data_test.go
git commit -m "feat(describe): add DataView, ListView, NavigationList formatters"
```

---

### Task 10: Built-in DataGrid formatter (`widget_fmt_datagrid.go`)

Covers: `Forms$DataGrid` (legacy built-in DataGrid, not DataGrid2).

**Files:**
- Create: `mdl/executor/widget_fmt_datagrid.go`
- Create: `mdl/executor/widget_fmt_datagrid_test.go`

- [ ] **Step 10.1: Write test**

```go
// mdl/executor/widget_fmt_datagrid_test.go
package executor

import "testing"

func TestBuiltinDataGridFormatter_EmitsDatagridKeyword(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := map[string]any{"$Type": "Forms$DataGrid", "Name": "dgItems"}
	builtinDataGridFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "datagrid dgItems")
}
```

- [ ] **Step 10.2: Implement**

```go
// mdl/executor/widget_fmt_datagrid.go
package executor

import "fmt"

func init() {
	d := DefaultDispatcher()
	for _, t := range []string{"Forms$DataGrid", "Pages$DataGrid"} {
		d.registerBSONType(t, FactoryEntry{Factory: builtinDataGridFactory})
	}
}

type builtinDataGridFormatter struct{ raw map[string]any }

func builtinDataGridFactory(raw map[string]any) WidgetFormatter {
	return &builtinDataGridFormatter{raw: raw}
}

func (f *builtinDataGridFormatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	var props []string
	if w.DataSource != nil {
		props = append(props, fmt.Sprintf("datasource: %s", formatDataSourceV3(w.DataSource)))
	}
	if w.Selection != "" {
		props = append(props, fmt.Sprintf("selection: %s", w.Selection))
	}
	if w.PagingPosition != "" {
		props = append(props, fmt.Sprintf("pagingposition: %s", w.PagingPosition))
	}
	props = appendAppearanceProps(props, w)
	// DataGrid may have children (columns + controlbar)
	hasChildren := len(w.Children) > 0 || len(w.FilterWidgets) > 0
	if !hasChildren {
		writeWidgetLine(ctx, "datagrid "+w.Name, props)
		return
	}
	formatWidgetProps(ctx.Output, indentStr(ctx.Indent), "datagrid "+w.Name, props, " {\n")
	if w.EntityContext != "" {
		outputDataContainerContext(ctx.Output, indentStr(ctx.Indent+1)+"  ", w.Name, w.EntityContext, false)
	}
	for _, child := range w.Children {
		ctx.Dispatcher.Format(ctx.Child(), rawWidgetToMap(child))
	}
	fmt.Fprintf(ctx.Output, "%s}\n", indentStr(ctx.Indent))
}
```

- [ ] **Step 10.3: Run tests + golden**

```bash
go test ./mdl/executor/ -run "TestBuiltinDataGridFormatter" -v
HELPDESK_VERSION=11.6.6 go test ./internal/goldenfs/ \
  -tags linux,integration -run TestHelpdeskGolden_DescribeSnapshot -timeout 2m 2>&1 | tail -3
```

- [ ] **Step 10.4: Commit**

```bash
git add mdl/executor/widget_fmt_datagrid.go mdl/executor/widget_fmt_datagrid_test.go
git commit -m "feat(describe): add built-in DataGrid formatter"
```

---

### Task 11: DataGrid2 pluggable formatter (`widget_fmt_datagrid2.go`)

Registered by widget ID `com.mendix.widget.web.datagrid2.DataGrid`. The complex column extraction logic is moved from `cmd_pages_describe_pluggable.go` into this formatter.

**Files:**
- Create: `mdl/executor/widget_fmt_datagrid2.go`
- Create: `mdl/executor/widget_fmt_datagrid2_test.go`

- [ ] **Step 11.1: Write test**

```go
// mdl/executor/widget_fmt_datagrid2_test.go
package executor

import "testing"

func TestDataGrid2Formatter_EmitsDatagridKeyword(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := map[string]any{
		"$Type": "CustomWidgets$CustomWidget",
		"Name":  "dgTickets",
		"Type":  map[string]any{"WidgetId": "com.mendix.widget.web.datagrid2.DataGrid"},
		"Object": map[string]any{"Properties": []any{}},
	}
	dataGrid2Factory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "datagrid dgTickets")
}
```

- [ ] **Step 11.2: Implement `widget_fmt_datagrid2.go`**

```go
// mdl/executor/widget_fmt_datagrid2.go
package executor

import "fmt"

const widgetIDDataGrid2 = "com.mendix.widget.web.datagrid2.DataGrid"

func init() {
	d := DefaultDispatcher()
	// Register CustomWidget meta-type with sub-key extractor if not already registered
	d.registerBSONType("CustomWidgets$CustomWidget", FactoryEntry{
		Factory:         GenericPluggableFactory,
		SubKeyExtractor: extractWidgetTypeID,
	})
	d.registerBSONType(widgetIDDataGrid2, FactoryEntry{Factory: dataGrid2Factory})
}

type dataGrid2Formatter struct{ raw map[string]any }

func dataGrid2Factory(raw map[string]any) WidgetFormatter { return &dataGrid2Formatter{raw: raw} }

func (f *dataGrid2Formatter) FormatMDL(ctx *FormatContext) {
	w := rawWidgetFromMap(f.raw)
	var props []string
	if w.DataSource != nil {
		props = append(props, fmt.Sprintf("datasource: %s", formatDataSourceV3(w.DataSource)))
	}
	if w.Selection != "" {
		props = append(props, fmt.Sprintf("selection: %s", w.Selection))
	}
	if w.PagingPosition != "" {
		props = append(props, fmt.Sprintf("pagingposition: %s", w.PagingPosition))
	}
	if w.PageSize != "" && w.PageSize != "20" {
		props = append(props, fmt.Sprintf("pagesize: %s", w.PageSize))
	}
	props = appendAppearanceProps(props, w)

	hasContent := len(w.ControlBar) > 0 || len(w.DataGridColumns) > 0
	if !hasContent {
		writeWidgetLine(ctx, "datagrid "+w.Name, props)
		return
	}

	formatWidgetProps(ctx.Output, indentStr(ctx.Indent), "datagrid "+w.Name, props, " {\n")
	if w.EntityContext != "" {
		outputDataContainerContext(ctx.Output, indentStr(ctx.Indent+1)+"  ", w.Name, w.EntityContext, false)
	}
	if len(w.ControlBar) > 0 {
		fmt.Fprintf(ctx.Output, "%scontrolbar controlBar1 {\n", indentStr(ctx.Indent+1))
		for _, cb := range w.ControlBar {
			ctx.Dispatcher.Format(ctx.withIndent(ctx.Indent+2), rawWidgetToMap(cb))
		}
		fmt.Fprintf(ctx.Output, "%s}\n", indentStr(ctx.Indent+1))
	}
	for _, col := range w.DataGridColumns {
		outputDataGrid2ColumnV3(ctx.Output, col, indentStr(ctx.Indent+1), ctx)
	}
	fmt.Fprintf(ctx.Output, "%s}\n", indentStr(ctx.Indent))
}
```

- [ ] **Step 11.3: Run tests + golden**

```bash
go test ./mdl/executor/ -run "TestDataGrid2Formatter" -v
HELPDESK_VERSION=11.6.6 go test ./internal/goldenfs/ \
  -tags linux,integration -run TestHelpdeskGolden_DescribeSnapshot -timeout 2m 2>&1 | tail -3
```

- [ ] **Step 11.4: Commit**

```bash
git add mdl/executor/widget_fmt_datagrid2.go mdl/executor/widget_fmt_datagrid2_test.go
git commit -m "feat(describe): add DataGrid2 pluggable formatter"
```

---

### Tasks 12–14: Gallery, ComboBox, Image formatters

Each follows the same structure as Task 11. The widget IDs are:
- Gallery: `com.mendix.widget.web.gallery.Gallery`
- ComboBox: `com.mendix.widget.web.combobox.ComboBox`
- Image: `com.mendix.widget.web.image.Image`

For each:

- [ ] **Step N.1:** Write test asserting the MDL keyword appears in output (gallery/combobox/image)
- [ ] **Step N.2:** Implement formatter, register by widget ID in `init()`
- [ ] **Step N.3:** Run unit test + golden
- [ ] **Step N.4:** Commit with `feat(describe): add <Widget> formatter`

**Gallery formatter** (`widget_fmt_gallery.go`) outputs `gallery NAME (datasource, columns, selection) { filter { } template { } }`, reusing `extractGalleryColumns` and `extractGalleryContentWidgets` from existing pluggable code.

**ComboBox formatter** (`widget_fmt_combobox.go`) outputs `combobox NAME (datasource, attribute)`, reading CaptionAttribute from the raw Object.

**Image formatter** (`widget_fmt_image.go`) outputs `image NAME (imagetype, imageurl, displayas, onclick)`.

---

## Phase 3 — Delete Old Layer

### Task 15: Switch describe fallback to GenericPluggableFormatter

Now all widget types are registered. Remove the `legacyWidgetFallback` and set the fallback to `GenericPluggableFactory`.

**Files:**
- Modify: `mdl/executor/cmd_pages_describe.go`
- Modify: `mdl/executor/widget_fmt_init.go`

- [ ] **Step 15.1: Update default dispatcher fallback**

In `widget_fmt_init.go`, change dispatcher initialization:

```go
func DefaultDispatcher() *FormatterDispatcher {
	defaultDispatcherOnce.Do(func() {
		d := newDefaultDispatcher()
		d.fallback = GenericPluggableFactory  // production fallback: schema introspection
		defaultDispatcherInst = d
	})
	return defaultDispatcherInst
}
```

- [ ] **Step 15.2: Remove legacy fallback override from `describePage()`**

In `cmd_pages_describe.go`, remove the line:
```go
d.fallback = func(raw map[string]any) WidgetFormatter { ... }  // remove this
```

- [ ] **Step 15.3: Delete `legacyWidgetFallback` struct from `cmd_pages_describe.go`**

Remove the entire `legacyWidgetFallback` struct and its `FormatMDL` method.

- [ ] **Step 15.4: Run golden tests**

```bash
HELPDESK_VERSION=11.6.6 go test ./internal/goldenfs/ \
  -tags linux,integration \
  -run "TestHelpdeskGolden_DescribeSnapshot|TestHelpdeskGolden_Regression_DescribeMDL" \
  -timeout 5m -v 2>&1 | tail -10
```
Expected: both PASS.

- [ ] **Step 15.5: Commit**

```bash
git add mdl/executor/cmd_pages_describe.go mdl/executor/widget_fmt_init.go
git commit -m "refactor(describe): replace legacy fallback with GenericPluggableFormatter"
```

---

### Task 16: Remove `rawWidgetFromMap` / `rawWidgetToMap` bridge and `parseRawWidget` dependency

Now that all formatters are registered, the bridge helpers that called `parseRawWidget` can be replaced with direct raw BSON access.

**Files:**
- Modify: `mdl/executor/widget_fmt_basic.go`, `widget_fmt_container.go`, etc.
- Modify: `mdl/executor/cmd_pages_describe.go`

- [ ] **Step 16.1: Replace `rawWidgetFromMap` calls in each formatter**

For each formatter file, remove the `rawWidgetFromMap(f.raw)` call and instead read fields directly from `f.raw`. Example for `ActionButtonFormatter`:

```go
func (f *actionButtonFormatter) FormatMDL(ctx *FormatContext) {
	name := safeStr(f.raw, "Name")
	caption := extractButtonCaption(nil, f.raw)
	params := extractButtonCaptionParameters(nil, f.raw)
	action := extractButtonAction(nil, f.raw)
	style := safeStr(f.raw, "ButtonStyle")
	// ... (same as before but no rawWidgetFromMap)
}
```

Container formatters read children from `getBsonArrayElements(f.raw["Widgets"])` directly.

- [ ] **Step 16.2: Run all tests**

```bash
go test ./mdl/executor/ -v 2>&1 | grep -E "^(ok|FAIL|---)" | head -30
```
Expected: no FAIL.

- [ ] **Step 16.3: Commit**

```bash
git add mdl/executor/
git commit -m "refactor(describe): remove rawWidgetFromMap bridge, read raw BSON directly in formatters"
```

---

### Task 17: Delete old parse/output/pluggable files

**Files to delete:**
- `mdl/executor/cmd_pages_describe_parse.go`
- `mdl/executor/cmd_pages_describe_output.go`
- `mdl/executor/cmd_pages_describe_pluggable.go`

- [ ] **Step 17.1: Move shared helpers to formatters or `widget_schema.go`**

Before deleting, identify helpers still used:
```bash
grep -rn "extractButtonCaption\|extractButtonAction\|formatDataSourceV3\|appendAppearanceProps\|formatWidgetProps\|mdlQuote\|outputDataContainerContext\|outputDataGrid2ColumnV3\|formatParametersV3" mdl/executor/ | grep -v "_test\|describe_output\|describe_parse\|describe_pluggable" | head -30
```

Move the referenced functions to appropriate formatter files or a new `widget_fmt_helpers.go`.

- [ ] **Step 17.2: Delete the three files**

```bash
git rm mdl/executor/cmd_pages_describe_parse.go \
       mdl/executor/cmd_pages_describe_output.go \
       mdl/executor/cmd_pages_describe_pluggable.go
```

- [ ] **Step 17.3: Build to find any remaining references**

```bash
go build ./mdl/executor/ 2>&1 | head -30
```
Fix any compile errors by moving remaining functions.

- [ ] **Step 17.4: Delete `rawWidget` struct from `cmd_pages_describe.go`**

Remove the `rawWidget`, `rawDataGridColumn`, `rawDataSource`, `rawWidgetRow`, `rawExplicitProp`, `rawDesignProp` struct definitions (they were in the deleted files or in `cmd_pages_describe.go`).

- [ ] **Step 17.5: Run full test suite**

```bash
go test ./mdl/executor/ ./internal/goldenfs/ \
  -tags linux,integration \
  -run "TestHelpdeskGolden_DescribeSnapshot|TestHelpdeskGolden_Regression_DescribeMDL" \
  -timeout 5m 2>&1 | tail -10
```
Expected: PASS.

- [ ] **Step 17.6: Commit**

```bash
git add mdl/executor/
git commit -m "refactor(describe): delete old parse/output/pluggable files + rawWidget struct (Phase 3)"
```

---

### Task 18: Thin `cmd_pages_describe.go` to < 200 lines

**Files:**
- Modify: `mdl/executor/cmd_pages_describe.go`

- [ ] **Step 18.1: Remove `getPageWidgetsFromRaw` (replaced by `getPageWidgetMaps`)**

```bash
grep -n "getPageWidgetsFromRaw" mdl/executor/cmd_pages_describe.go
```
Delete the function and all references.

- [ ] **Step 18.2: Verify line count**

```bash
wc -l mdl/executor/cmd_pages_describe.go
```
Expected: < 200 lines.

- [ ] **Step 18.3: Run idempotency test**

```bash
HELPDESK_VERSION=11.6.6 go test ./internal/goldenfs/ \
  -tags linux,integration -run TestHelpdeskGolden_DescribeSnapshot_Idempotent \
  -timeout 5m -v 2>&1 | tail -5
```
Expected: PASS.

- [ ] **Step 18.4: Commit**

```bash
git add mdl/executor/cmd_pages_describe.go
git commit -m "refactor(describe): thin cmd_pages_describe.go to metadata-only entry point (Phase 3 complete)"
```

---

## Phase 4 — Final Verification

### Task 19: Update golden snapshots and run full suite

- [ ] **Step 19.1: Run `make update-snapshots`**

```bash
make update-snapshots 2>&1 | tail -20
```
Expected: both versions output `✓ Check passed!`.

- [ ] **Step 19.2: Run complete test suite**

```bash
make test 2>&1 | grep -E "^(ok|FAIL)" | head -30
```
Expected: no FAIL in executor, goldenfs, or backend packages (excluding the pre-existing fixture-count failures in `mdl/backend/mpr` and `mdl/backend/mpr/repos` which are unrelated to this refactor).

- [ ] **Step 19.3: Run all three golden integration tests**

```bash
for VERSION in 11.6.6 11.10.0; do
  HELPDESK_VERSION=$VERSION go test ./internal/goldenfs/ \
    -tags linux,integration \
    -run "TestHelpdeskGolden_DescribeSnapshot$|TestHelpdeskGolden_Regression_DescribeMDL$|TestHelpdeskGolden_DescribeSnapshot_Idempotent$" \
    -timeout 5m -v 2>&1 | grep -E "^(--- |PASS|FAIL|ok )"
done
```
Expected: all 6 tests PASS (3 × 2 versions).

- [ ] **Step 19.4: Final commit**

```bash
git add testdata/
git commit -m "chore(describe): update golden snapshots after SOLID refactor (Phase 4 complete)"
```

---

## Summary

| Phase | Tasks | Key deliverable |
|-------|-------|----------------|
| 1 | 1–5 | Infrastructure: interfaces, schema utils, bridge. Zero behavior change. |
| 2 | 6–14 | All formatter files. Each type migrated individually with regression test. |
| 3 | 15–18 | Old files deleted. `rawWidget` gone. `cmd_pages_describe.go` < 200 lines. |
| 4 | 19 | Golden snapshots verified. Make test clean. |

**Exit criteria for the whole refactor:**
- `make test` clean (excluding pre-existing fixture-count failures)
- `TestHelpdeskGolden_DescribeSnapshot_Idempotent` passes for both versions
- `mxcli check describe-snapshot.mdl --references` passes for both versions
- `cmd_pages_describe_parse.go`, `cmd_pages_describe_output.go`, `cmd_pages_describe_pluggable.go` deleted
- `cmd_pages_describe.go` < 200 lines
- Adding a new widget formatter requires only: create one file + add one `init()` registration

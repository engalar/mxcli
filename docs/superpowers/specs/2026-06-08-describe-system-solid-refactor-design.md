# Describe System SOLID Refactor Design

**Date:** 2026-06-08  
**Scope:** `mdl/executor/` — page describe pipeline  
**Goal:** Migrate page describe from raw BSON + giant switch to a formatter registry that satisfies SOLID principles and handles arbitrary (unknown) pluggable widgets via schema self-introspection.

---

## Problem Statement

The current page describe system violates four SOLID principles:

| Principle | Violation |
|-----------|-----------|
| S | `rawWidget` (39 fields) carries data for all widget types; `FormatterRegistry.Format()` mixes registration with Mendix-specific dispatch logic |
| O | Adding a new widget type requires modifying two giant switch statements (`parseRawWidget` 21 cases, `outputWidgetMDLV3` 28 cases) |
| I | `DescribeResolver` bundles `GetRawUnit` with name-resolution methods; simple formatters are forced to depend on raw-unit access they never use |
| D | `FormatContext` holds a concrete `*FormatterRegistry`; high-level describe code directly manipulates `map[string]any` BSON details |

Additionally, `extractExplicitProperties()` hardcodes `"true"/"false"` instead of reading `DefaultValue` from the widget's embedded schema — unknown pluggable widgets silently lose all user-configured property values.

---

## Core Design Principle

> **Any widget, known or unknown, outputs the minimal sufficient MDL by comparing stored property values against their defaults from the widget's embedded Type BSON schema.**

This mirrors the NBL creation logic: creation reads `Type.ObjectType.PropertyTypes[*].ValueType.DefaultValue` to initialize properties; describe reads the same schema to suppress defaults in output. The output can always serve as its own input to recreate the original widget.

---

## Architecture

### Data Flow

```
Page BSON (mxunit file)
  │
  ▼ FormatterDispatcher.Format(ctx, raw)          ← raw map always, no mandatory gen decode
    ├─ lookup by bsonType → FactoryEntry
    │   └─ SubKeyExtractor? → lookup by widgetID
    └─ factory(raw) → WidgetFormatter
         │  factory decides: use raw directly OR decode to gen type (per formatter)
         ▼ formatter.FormatMDL(ctx)
           ├─ for known widgets: idiomatic MDL keywords
           ├─ for unknown pluggable: schema-introspected minimal MDL
           └─ child widgets: ctx.Dispatcher.Format(ctx.Child(), childRaw)
```

### SOLID-Compliant Core Types

```go
// ── Narrow interfaces (I principle) ──────────────────────────────────────

type NameResolver interface {
    ResolveEntityName(containerID string) string
    ResolveModuleName(id model.ID) string
}

type RawUnitProvider interface {
    GetRawUnit(id model.ID) (map[string]any, error)
}

type WidgetDispatcher interface {
    Format(ctx *FormatContext, raw map[string]any)
}

// ── FormatContext depends only on interfaces (D principle) ────────────────

type FormatContext struct {
    Output     io.Writer
    Indent     int
    Dispatcher WidgetDispatcher  // interface, not *FormatterDispatcher
    Names      NameResolver      // interface, not *ExecContext
}

func (ctx *FormatContext) Child() *FormatContext  // returns Indent+1 copy
func (ctx *FormatContext) Write(s string)         // writes indented line

// ── WidgetFormatter: single-method interface (I principle) ────────────────

type WidgetFormatter interface {
    FormatMDL(ctx *FormatContext)
}

type FormatterFactory func(raw map[string]any) WidgetFormatter

// ── Data-driven dispatch: no Mendix knowledge in dispatcher (O principle) ─

type FactoryEntry struct {
    Factory         FormatterFactory
    SubKeyExtractor func(raw map[string]any) string  // nil = no sub-dispatch
}

type FormatterDispatcher struct {
    entries  map[string]FactoryEntry
    fallback FormatterFactory
}

// Format is purely data-driven — no isCustomWidget() special-casing
func (d *FormatterDispatcher) Format(ctx *FormatContext, raw map[string]any) {
    bsonType, _ := raw["$Type"].(string)
    entry, ok := d.entries[bsonType]
    if !ok {
        d.fallback(raw).FormatMDL(ctx)
        return
    }
    key := bsonType
    if entry.SubKeyExtractor != nil {
        key = entry.SubKeyExtractor(raw)
        if sub, ok := d.entries[key]; ok {
            sub.Factory(raw).FormatMDL(ctx)
            return
        }
        // unknown pluggable widget → fallback (GenericPluggableFormatter)
        d.fallback(raw).FormatMDL(ctx)
        return
    }
    entry.Factory(raw).FormatMDL(ctx)
}

// Registration examples
dispatcher.Register("Forms$ActionButton", FactoryEntry{Factory: actionButtonFactory})
dispatcher.Register("CustomWidgets$CustomWidget", FactoryEntry{
    Factory:         GenericPluggableFactory,
    SubKeyExtractor: extractWidgetTypeID,  // reads Type.WidgetId
})
dispatcher.Register("com.mendix.widget.web.datagrid2.DataGrid", FactoryEntry{
    Factory: dataGrid2Factory,
})
```

---

## Unknown Pluggable Widget: Schema Self-Introspection

### Source of Truth

Every `CustomWidgets$CustomWidget` BSON document carries its own schema:

```
Type.ObjectType.PropertyTypes[]
  └─ PropertyType
      ├─ $ID                          ← TypePointer (join key)
      ├─ PropertyKey                  ← MDL property name
      └─ ValueType
          ├─ Type                     ← "Boolean" | "Integer" | "Enumeration" | ...
          └─ DefaultValue             ← default value string
Object.Properties[]
  └─ Property
      ├─ TypePointer                  ← references PropertyType.$ID
      └─ Value.PrimitiveValue         ← actual stored value
```

### GenericPluggableFormatter

```go
type GenericPluggableFormatter struct{ raw map[string]any }

func (f *GenericPluggableFormatter) FormatMDL(ctx *FormatContext) {
    name     := safeStr(f.raw, "Name")
    widgetID := extractWidgetTypeID(f.raw)
    schema   := buildSchemaMap(f.raw)        // TypePointerID → {key, defaultValue, type}
    props    := readProperties(f.raw)        // []PropertyValue
    nonDef   := filterDefaults(props, schema) // compare value vs defaultValue

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

### filterDefaults: Six Value Types

| ValueType | Default rule | Output format |
|-----------|-------------|---------------|
| Boolean | value == schema.DefaultValue (or `"false"`) | `true` / `false` |
| Integer | value == schema.DefaultValue (or `"0"`) | bare integer |
| Enumeration | value == schema.DefaultValue | `'value'` |
| String | value == schema.DefaultValue (or `""`) | `'text'` |
| Expression | value == schema.DefaultValue (or `""`) | expression string |
| Object/Widgets | non-empty = always output | recursive |

This replaces the current `extractExplicitProperties()` which only hardcodes `"true"`/`"false"`.

---

## Formatter Conventions

**1. Factory owns the decode strategy; formatter never does type assertions.**

```go
// Simple: factory passes raw map directly
func actionButtonFactory(raw map[string]any) WidgetFormatter {
    return &ActionButtonFormatter{raw: raw}
}

// Complex: factory decodes to gen type; formatter uses typed getters
func dataViewFactory(raw map[string]any) WidgetFormatter {
    dv, _ := codec.DefaultRegistry.DecodeFromMap(raw).(*genPg.DataView)
    return &DataViewFormatter{raw: raw, dv: dv}
}
```

**2. Child widgets dispatched through `ctx.Dispatcher`, never direct formatter calls.**

```go
for _, childRaw := range getWidgetsArray(f.raw) {
    ctx.Dispatcher.Format(ctx.Child(), childRaw)
}
```

**3. Never panic. Safe field access everywhere.**

```go
func safeStr(raw map[string]any, key string) string {
    v, _ := raw[key].(string)
    return v
}
```

**4. `ctx.Child()` for indentation; `ctx.Names` for entity/module name resolution.**

---

## File Structure

```
mdl/executor/
  # New formatter system
  widget_formatter.go          # WidgetFormatter, WidgetDispatcher, FormatterDispatcher,
                               # FormatContext, FactoryEntry, NameResolver, RawUnitProvider
  widget_schema.go             # buildSchemaMap, filterDefaults, readProperties,
                               # formatPropertyValue, extractWidgetTypeID
  widget_fmt_basic.go          # ActionButton, TextBox, TextArea, DynamicText,
                               # CheckBox, DatePicker, RadioButtons, Label, Title
  widget_fmt_container.go      # DivContainer, GroupBox, ScrollContainer,
                               # TabControl+TabPage, LayoutGrid
  widget_fmt_data.go           # DataView, ListView
  widget_fmt_datagrid.go       # DataGrid (built-in, Forms$DataGrid)
  widget_fmt_datagrid2.go      # DataGrid2 (pluggable, registered by widget ID)
  widget_fmt_gallery.go        # Gallery (pluggable)
  widget_fmt_combobox.go       # ComboBox (pluggable)
  widget_fmt_nav.go            # NavigationList, SnippetCallWidget
  widget_fmt_image.go          # Image widget (pluggable)
  widget_fmt_pluggable.go      # GenericPluggableFormatter + schema utilities

  # Retained, progressively thinned
  cmd_pages_describe.go        # Page metadata only (params, grant, layout name)
                               # Widget tree delegated to FormatterDispatcher

  # Removed in Phase 3
  cmd_pages_describe_parse.go      # Replaced by codec.DefaultRegistry + Dispatcher
  cmd_pages_describe_output.go     # Replaced by widget_fmt_*.go files
  cmd_pages_describe_pluggable.go  # Merged into widget_fmt_datagrid2.go + widget_fmt_gallery.go
```

---

## Migration Phases

### Phase 1 — Infrastructure (zero behavior change)
- Add `widget_formatter.go`, `widget_schema.go`, `widget_fmt_pluggable.go`
- Fix `extractExplicitProperties()` to use schema DefaultValue instead of hardcoded strings
- Dispatcher registered but empty; legacy code path unchanged
- **Exit criteria:** all existing tests pass unchanged

### Phase 2 — Per-type formatter migration
- Implement and register formatters one file at a time
- Each registration removes the corresponding case from `outputWidgetMDLV3`
- Order: basic → container → data → datagrid/gallery
- Per-formatter migration regression test: new output == old output (byte-for-byte)
- **Exit criteria:** `outputWidgetMDLV3` is empty

### Phase 3 — Remove parse layer
- Replace `parseRawWidget` with `codec.DefaultRegistry.Decode` + Dispatcher
- Delete `rawWidget` struct (all fields migrated into formatter-local structs)
- Delete `cmd_pages_describe_parse.go`, `cmd_pages_describe_output.go`, `cmd_pages_describe_pluggable.go`
- **Exit criteria:** three old files deleted; `cmd_pages_describe.go` < 200 lines

### Phase 4 — Cleanup
- Remove any remaining `rawDataGridColumn` and similar intermediate structs
- Remove migration regression test helpers
- **Exit criteria:** `make test` clean; golden snapshot idempotency test passes

---

## Testing Strategy

### Three Layers

**L1 — Formatter unit tests** (no MPR required)

```go
func TestActionButtonFormatter_NonDefaultStyle(t *testing.T) {
    raw := map[string]any{
        "$Type": "Forms$ActionButton",
        "Name":  "btnSubmit",
        "ButtonStyle": "primary",
    }
    ctx := newTestFormatContext(t)
    actionButtonFactory(raw).FormatMDL(ctx)
    assertContains(t, ctx.String(), "buttonstyle: primary")
}
```

**L2 — Schema utility unit tests** (six value type cases)

```go
func TestFilterDefaults_SkipsMatchingEnum(t *testing.T) {
    schema := SchemaMap{"ptr1": {Key: "source", DefaultValue: "context"}}
    props  := []PropertyValue{{TypePointerID: "ptr1", PrimitiveValue: "context"}}
    assert.Empty(t, filterDefaults(props, schema))
}
```

**L3 — Migration regression tests** (Phase 2/3 safety net)

```go
func TestMigration_ActionButton_OutputUnchanged(t *testing.T) {
    mpr    := openGoldenMPR(t)
    newOut := describePageViaDispatcher(t, mpr, "HD.Ticket_Detail")
    oldOut := describePageViaLegacy(t, mpr, "HD.Ticket_Detail")
    assert.Equal(t, oldOut, newOut)
}
```

**Existing golden tests retained throughout all phases:**

```
TestHelpdeskGolden_DescribeSnapshot_Idempotent   # roundtrip idempotency
TestHelpdeskGolden_Regression_DescribeMDL        # semantic consistency
mxcli check describe-snapshot.mdl --references   # MDL validity
```

### Coverage Matrix

| Target | Layer | MPR needed |
|--------|-------|-----------|
| Each formatter | L1 unit | No |
| filterDefaults × 6 types | L2 unit | No |
| Dispatcher sub-key dispatch | L2 unit | No |
| Per-migration output parity | L3 regression | Yes (golden) |
| Full roundtrip | Existing golden | Yes (golden) |

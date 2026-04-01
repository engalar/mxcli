---
name: mendix-custom-widgets
description: Use when writing MDL for GALLERY, COMBOBOX, or third-party pluggable widgets in CREATE PAGE / ALTER PAGE statements. Covers built-in widget syntax, child slots (TEMPLATE/FILTER), adding new custom widgets via .def.json, and engine internals.
---

# Custom & Pluggable Widgets in MDL

## Built-in Pluggable Widgets

### GALLERY

Card-layout list with optional template content and filters.

```sql
GALLERY galleryName (
  DataSource: DATABASE FROM Module.Entity SORT BY Name ASC,
  Selection: Single | Multiple | None,
  DesktopColumns: 3,
  TabletColumns: 2,
  PhoneColumns: 1
) {
  TEMPLATE template1 {
    DYNAMICTEXT title (Content: '{1}', ContentParams: [{1} = Name], RenderMode: H4)
    DYNAMICTEXT info  (Content: '{1}', ContentParams: [{1} = Email])
  }
  FILTER filter1 {
    TEXTFILTER   searchName  (Attribute: Name)
    NUMBERFILTER searchScore (Attribute: Score)
    DROPDOWNFILTER searchStatus (Attribute: Status)
    DATEFILTER   searchDate  (Attribute: CreatedAt)
  }
}
```

- `TEMPLATE` block -> mapped to `content` property (child widgets rendered per row)
- `FILTER` block -> mapped to `filtersPlaceholder` property (shown above list)
- `Selection: None` omits the selection property (default if omitted)
- `DesktopColumns`, `TabletColumns`, `PhoneColumns` control responsive grid columns (default: 1 each, omit if default)
- Children written directly under GALLERY (no container) go to the first slot with `mdlContainer: "TEMPLATE"`

### COMBOBOX

Two modes depending on the attribute type:

```sql
-- Enumeration mode (Attribute is an enum)
COMBOBOX cbStatus (Label: 'Status', Attribute: Status)

-- Association mode (Attribute is an association)
COMBOBOX cmbCustomer (
  Label: 'Customer',
  Attribute: Order_Customer,
  DataSource: DATABASE Module.Customer,
  CaptionAttribute: Name
)
```

- Engine detects association mode when `DataSource` is present (`hasDataSource` condition)
- `CaptionAttribute` is the display attribute on the **target** entity
- In association mode, mapping order matters: DataSource must resolve before Association (sets entityContext)

## Adding a Third-Party Widget

### Step 1 -- Extract .def.json from .mpk

```bash
mxcli widget extract --mpk widgets/MyWidget.mpk
# Output: .mxcli/widgets/mywidget.def.json

# Override MDL keyword
mxcli widget extract --mpk widgets/MyWidget.mpk --mdl-name MYWIDGET
```

The `extract` command parses the .mpk (ZIP archive containing `package.xml` + widget XML) and auto-infers operations from XML property types:

| XML Type | Operation | MDL Source Key |
|----------|-----------|----------------|
| attribute | attribute | `Attribute` |
| association | association | `Association` |
| datasource | datasource | `DataSource` |
| selection | selection | `Selection` |
| widgets | widgets (child slot) | container name (key uppercased) |
| boolean/string/enumeration/integer/decimal | primitive | hardcoded `Value` from defaultValue |
| action/expression/textTemplate/object/icon/image/file | *skipped* | too complex for auto-mapping |

Skipped types require manual configuration in the .def.json.

### Step 2 -- Extract BSON template from Studio Pro

The .def.json only describes mapping rules. The engine also needs a **template JSON** with the complete Type + Object BSON structure.

```bash
# 1. In Studio Pro: drag the widget onto a test page, save the project
# 2. Extract the widget's BSON:
mxcli bson dump -p App.mpr --type page --object "Module.TestPage" --format json
# 3. Extract the Type and Object fields from the CustomWidget, save as:
```

Place at: `project/.mxcli/widgets/mywidget.json`

Template JSON format:

```json
{
  "widgetId": "com.vendor.widget.MyWidget",
  "name": "My Widget",
  "version": "1.0.0",
  "extractedFrom": "TestModule.TestPage",
  "type": {
    "$ID": "aa000000000000000000000000000001",
    "$Type": "CustomWidgets$CustomWidgetType",
    "WidgetId": "com.vendor.widget.MyWidget",
    "PropertyTypes": [
      {
        "$ID": "aa000000000000000000000000000010",
        "$Type": "CustomWidgets$WidgetPropertyType",
        "PropertyKey": "datasource",
        "ValueType": { "$ID": "...", "Type": "DataSource" }
      }
    ]
  },
  "object": {
    "$ID": "aa000000000000000000000000000100",
    "$Type": "CustomWidgets$WidgetObject",
    "TypePointer": "aa000000000000000000000000000001",
    "Properties": [
      2,
      {
        "$ID": "...",
        "$Type": "CustomWidgets$WidgetProperty",
        "TypePointer": "aa000000000000000000000000000010",
        "Value": {
          "$Type": "CustomWidgets$WidgetValue",
          "DataSource": null,
          "AttributeRef": null,
          "PrimitiveValue": "",
          "Widgets": [2],
          "Selection": "None"
        }
      }
    ]
  }
}
```

**CRITICAL**: Template must include both `type` (PropertyTypes schema) and `object` (default WidgetObject with all property values). Extract from a real Studio Pro MPR -- do NOT generate programmatically. Mismatched structure causes CE0463.

### Step 3 -- Place files

```
project/.mxcli/widgets/mywidget.def.json   <- project scope (highest priority)
project/.mxcli/widgets/mywidget.json       <- template JSON (same directory)
~/.mxcli/widgets/mywidget.def.json         <- global scope
```

Set `"templateFile": "mywidget.json"` in the .def.json. Project definitions override global ones; global overrides embedded.

### Step 4 -- Use in MDL

```sql
MYWIDGET myWidget1 (DataSource: DATABASE Module.Entity, Attribute: Name) {
  TEMPLATE content1 {
    DYNAMICTEXT label1 (Content: '{1}', ContentParams: [{1}=Name])
  }
}
```

## .def.json Reference

```json
{
  "widgetId":        "com.vendor.widget.web.mywidget.MyWidget",
  "mdlName":         "MYWIDGET",
  "templateFile":    "mywidget.json",
  "defaultEditable": "Always",
  "propertyMappings": [
    {"propertyKey": "datasource",  "source": "DataSource", "operation": "datasource"},
    {"propertyKey": "attribute",   "source": "Attribute",  "operation": "attribute"},
    {"propertyKey": "someFlag",    "value":  "true",       "operation": "primitive"}
  ],
  "childSlots": [
    {"propertyKey": "content", "mdlContainer": "TEMPLATE", "operation": "widgets"}
  ],
  "modes": [
    {
      "name": "association",
      "condition": "hasDataSource",
      "propertyMappings": [
        {"propertyKey": "optionsSource", "value": "association", "operation": "primitive"},
        {"propertyKey": "assocDS",       "source": "DataSource",  "operation": "datasource"},
        {"propertyKey": "assoc",         "source": "Association", "operation": "association"}
      ]
    },
    {
      "name": "default",
      "propertyMappings": [
        {"propertyKey": "attr", "source": "Attribute", "operation": "attribute"}
      ]
    }
  ]
}
```

### Mode Conditions

| Condition | Checks |
|-----------|--------|
| `hasDataSource` | AST widget has a `DataSource` property |
| `hasAttribute` | AST widget has an `Attribute` property |
| `hasProp:XYZ` | AST widget has a property named `XYZ` |

Modes are evaluated in definition order -- first match wins. A mode with no `condition` is the default fallback.

### 6 Built-in Operations

| Operation | What it does | Typical Source |
|-----------|-------------|----------------|
| `attribute` | Sets `Value.AttributeRef` on a WidgetProperty | `Attribute` |
| `association` | Sets `Value.AttributeRef` + `Value.EntityRef` | `Association` |
| `primitive` | Sets `Value.PrimitiveValue` | static `value` or property name |
| `datasource` | Sets `Value.DataSource` (serialized BSON) | `DataSource` |
| `selection` | Sets `Value.Selection` (mode string) | `Selection` |
| `widgets` | Replaces `Value.Widgets` array with child widget BSON | child slot |
| `texttemplate` | Sets text in `Value.TextTemplate` (Forms$ClientTemplate) | property name (resolved as string) |
| `action` | Sets `Value.Action` with serialized client action BSON | `OnClick` (resolved from AST Action) |

### Mapping Order Constraints

- **`Association` source must come AFTER `DataSource` source** in the mappings array. The association operation depends on `entityContext` set by a prior DataSource mapping. The registry validates this at load time.
- **`value` takes priority over `source`**: if both are set, the static `value` is used.

### Source Resolution

| Source | Resolution logic |
|--------|-----------------|
| `Attribute` | `w.GetAttribute()` -> `pageBuilder.resolveAttributePath()` |
| `DataSource` | `w.GetDataSource()` -> `pageBuilder.buildDataSourceV3()` -> also updates `entityContext` |
| `Association` | `w.GetAttribute()` -> `pageBuilder.resolveAssociationPath()` + uses current `entityContext` |
| `Selection` | `w.GetSelection()` or `mapping.Default` fallback |
| `CaptionAttribute` | `w.GetStringProp("CaptionAttribute")` -> auto-prefixed with `entityContext` if relative |
| *(other)* | Treated as generic property name: `w.GetStringProp(source)` |

## Engine Internals

### Build Pipeline

When `buildWidgetV3()` encounters an unrecognized widget type:

```
1. Registry lookup: widgetRegistry.Get("MYWIDGET") -> WidgetDefinition
2. Template loading: GetTemplateFullBSON(widgetID, idGenerator, projectPath)
   a. Load JSON from embed.FS (or .mxcli/widgets/)
   b. Augment from project's .mpk (if newer version available)
   c. Phase 1: Collect all $ID values -> generate new UUID mapping
   d. Phase 2: Convert Type JSON -> BSON, extract PropertyTypeIDMap
   e. Phase 3: Convert Object JSON -> BSON (TypePointer remapped via same mapping)
   f. Placeholder leak check (aa000000-prefix IDs must all be remapped)
3. Mode selection: evaluateCondition() on each mode in order -> first match wins
4. Property mappings: for each mapping, resolveMapping() -> OperationFunc()
   Each operation locates the WidgetProperty by matching TypePointer against PropertyTypeIDMap
5. Child slots: group AST children by container name, build to BSON, embed via opWidgets
6. Assemble CustomWidget{RawType, RawObject, PropertyTypeIDMap, ObjectTypeID}
```

### PropertyTypeIDMap

The map links PropertyKey names (from .def.json) to their BSON IDs:

```
PropertyTypeIDMap["datasource"] = {
  PropertyTypeID: "a1b2c3d4...",   // $ID of WidgetPropertyType in Type
  ValueTypeID:    "e5f6a7b8...",   // $ID of ValueType within PropertyType
  DefaultValue:   "",
  ValueType:      "DataSource",    // Type string
  ObjectTypeID:   "...",           // For nested object list properties
}
```

Operations use this map to locate the correct WidgetProperty in the Object's Properties array by comparing `TypePointer` (binary GUID) against `PropertyTypeID`.

### MPK Augmentation

At template load time, `augmentFromMPK()` checks if the project has a newer `.mpk` for the widget:

```
project/widgets/*.mpk -> FindMPK(projectDir, widgetID) -> ParseMPK()
-> AugmentTemplate(clone, mpkDef)
   -> Add missing properties from newer .mpk version
   -> Remove stale properties no longer in .mpk
```

This reduces CE0463 errors from widget version drift without requiring manual template re-extraction.

### 3-Tier Registry

| Priority | Location | Scope |
|----------|----------|-------|
| 1 (highest) | `<project>/.mxcli/widgets/*.def.json` | Project |
| 2 | `~/.mxcli/widgets/*.def.json` | Global (user) |
| 3 (lowest) | `sdk/widgets/definitions/*.def.json` (embedded) | Built-in |

Higher priority definitions override lower ones with the same MDL name (case-insensitive).

## Verify & Debug

```bash
# List registered widgets
mxcli widget list -p App.mpr

# Check after creating a page
mxcli check script.mdl -p App.mpr --references

# Full mx check (catches CE0463)
~/.mxcli/mxbuild/*/modeler/mx check App.mpr

# Debug CE0463 -- compare NDSL dumps
mxcli bson dump -p App.mpr --type page --object "Module.PageName" --format ndsl
```

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| CE0463 after page creation | Template version mismatch -- extract fresh template from Studio Pro MPR, or ensure .mpk augmentation picks up new properties |
| Widget not recognized | Check `mxcli widget list`; .def.json must be in `.mxcli/widgets/` with `.def.json` extension |
| TEMPLATE content missing | Widget needs `childSlots` entry with `"mdlContainer": "TEMPLATE"` |
| Association COMBOBOX shows enum behavior | Add `DataSource` to trigger association mode (`hasDataSource` condition) |
| Association mapping fails | Ensure DataSource mapping appears **before** Association mapping in the array |
| Custom widget not found | Place .def.json in `.mxcli/widgets/` inside the project directory |
| Placeholder ID leak error | Template JSON has unreferenced `$ID` values starting with `aa000000` -- ensure all IDs are in the `collectIDs` traversal path |

## Key Source Files

| File | Purpose |
|------|---------|
| `mdl/executor/widget_engine.go` | PluggableWidgetEngine, 6 operations, Build() pipeline |
| `mdl/executor/widget_registry.go` | 3-tier WidgetRegistry, definition validation |
| `sdk/widgets/loader.go` | Template loading, ID remapping, MPK augmentation |
| `sdk/widgets/mpk/mpk.go` | .mpk ZIP parsing, XML property extraction |
| `cmd/mxcli/cmd_widget.go` | `mxcli widget extract/list` CLI commands |
| `sdk/widgets/definitions/*.def.json` | Built-in widget definitions (ComboBox, Gallery) |
| `sdk/widgets/templates/mendix-11.6/*.json` | Embedded BSON templates |
| `mdl/executor/cmd_pages_builder_input.go` | `updateWidgetPropertyValue()` -- TypePointer matching |

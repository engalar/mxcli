# Widget Templates

This directory contains JSON templates for Mendix pluggable widgets. These templates are extracted from a reference Mendix project and embedded into the mxcli binary via `go:embed`.

## Structure

```
templates/
├── mendix-11.6/                    # Templates for Mendix 11.6.x
│   ├── combobox.json               # com.mendix.widget.web.combobox.Combobox
│   ├── datagrid.json               # com.mendix.widget.web.datagrid.Datagrid
│   ├── gallery.json                # com.mendix.widget.web.gallery.Gallery
│   ├── datagrid-text-filter.json   # DatagridTextFilter
│   ├── datagrid-date-filter.json   # DatagridDateFilter
│   ├── datagrid-dropdown-filter.json
│   └── datagrid-number-filter.json
└── README.md
```

## Template Format

Each template is a JSON file containing **both** the `CustomWidgetType` and `WidgetObject` structures:

```json
{
  "widgetId": "com.mendix.widget.web.combobox.Combobox",
  "name": "Combo box",
  "version": "11.6.0",
  "extractedFrom": "PageTemplates.Customer_NewEdit",
  "type": {
    "$ID": "aa000000000000000000000000000001",
    "$Type": "CustomWidgets$CustomWidgetType",
    "WidgetId": "com.mendix.widget.web.combobox.Combobox",
    "PropertyTypes": [
      {
        "$ID": "aa000000000000000000000000000010",
        "$Type": "CustomWidgets$WidgetPropertyType",
        "PropertyKey": "attributeEnumeration",
        "ValueType": {
          "$ID": "aa000000000000000000000000000011",
          "Type": "Attribute",
          "DefaultValue": ""
        }
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
        "$ID": "aa000000000000000000000000000110",
        "$Type": "CustomWidgets$WidgetProperty",
        "TypePointer": "aa000000000000000000000000000010",
        "Value": {
          "$ID": "aa000000000000000000000000000111",
          "$Type": "CustomWidgets$WidgetValue",
          "AttributeRef": null,
          "DataSource": null,
          "PrimitiveValue": "",
          "Widgets": [2],
          "Selection": "None"
        }
      }
    ]
  }
}
```

### Why Both `type` AND `object` Are Required

The `type` field defines the widget's PropertyTypes (schema), while the `object` field contains the actual property values with correct defaults. Studio Pro expects:

1. **Consistent cross-references**: `object.Properties[].TypePointer` must reference valid `type.PropertyTypes[].$ID` values; `object.TypePointer` must reference `type.$ID`
2. **All properties present**: Every PropertyType in the Type must have a corresponding WidgetProperty in the Object
3. **Correct default values**: Properties like `TextTemplate` need proper `Forms$ClientTemplate` structures, not null

Without the `object` field, mxcli must build the WidgetObject from scratch, which is error-prone and often triggers CE0463 "widget definition has changed" in Studio Pro.

### ID Cross-Reference Structure

```
Type                                    Object
├─ $ID ◄──────────────────────────── TypePointer (WidgetObject → CustomWidgetType)
└─ PropertyTypes[]                   └─ Properties[]
   ├─ $ID ◄──────────────────────────── TypePointer (WidgetProperty → WidgetPropertyType)
   └─ ValueType                         └─ Value
      └─ $ID ◄──────────────────────────── TypePointer (WidgetValue → ValueType)
```

At load time, all `$ID` values are remapped to fresh UUIDs. The same mapping is applied to both Type and Object, preserving these cross-references.

## Runtime Loading Pipeline

`GetTemplateFullBSON()` in `loader.go` executes a 3-phase pipeline:

### Phase 1: Collect IDs

`collectIDs()` recursively walks both `type` and `object` JSON, creates `oldID → newUUID` mapping for every `$ID` field.

### Phase 2: Convert Type JSON → BSON

`jsonToBSONWithMappingAndObjectType()` converts the Type, replacing IDs and simultaneously extracting `PropertyTypeIDMap`:

```
PropertyTypeIDMap["attributeEnumeration"] = {
  PropertyTypeID: "newUUID-010",  // remapped $ID of WidgetPropertyType
  ValueTypeID:    "newUUID-011",  // remapped $ID of ValueType
  DefaultValue:   "",
  ValueType:      "Attribute",
}
```

This map is the bridge between `.def.json` property keys and the BSON structure — the engine uses it to locate which WidgetProperty to modify for each mapping.

### Phase 3: Convert Object JSON → BSON

`jsonToBSONObjectWithMapping()` converts the Object using the same ID mapping. `TypePointer` fields are specially handled to ensure they point to the new IDs from Phase 2.

### Placeholder Leak Detection

After both phases, `containsPlaceholderID()` checks for any remaining `aa000000`-prefix IDs. If found, the load fails immediately rather than producing a corrupt MPR.

### MPK Augmentation

Before the 3-phase pipeline, `augmentFromMPK()` checks if the project has a newer `.mpk` for the widget (in `project/widgets/`). If found, it deep-clones the template and merges property changes from the `.mpk` XML definition, adding missing properties and removing stale ones. This reduces CE0463 from widget version drift.

## Extracting New Templates

### Important: Use Studio Pro-Created Widgets

When extracting templates, **always use widgets that have been created or "fixed" by Studio Pro**. This ensures the WidgetObject contains correct default values. If you programmatically create a widget and extract it, you'll just get the same incorrect structure back.

### Extraction Process

1. **Create the widget in Studio Pro** — Add the widget to a page and configure it with default settings

2. **If updating an existing template** — If Studio Pro shows "widget definition has changed", right-click and select "Update widget" to let Studio Pro fix it

3. **Extract the BSON**:
```bash
# Dump the page containing the widget
mxcli bson dump -p App.mpr --type page --object "Module.TestPage" --format json

# Extract the CustomWidget's Type and Object fields from the JSON output
# Save as templates/mendix-11.6/widgetname.json
```

4. **Extract skeleton .def.json** (for new widgets):
```bash
mxcli widget extract --mpk widgets/MyWidget.mpk
# Generates .mxcli/widgets/mywidget.def.json with auto-inferred mappings
```

### Verifying Templates

After updating a template, verify it works:

```bash
# Create a test page with the widget
mxcli -p test.mpr -c "CREATE PAGE Test.TestPage ... COMBOBOX ..."

# Check for errors (should have no CE0463 errors)
~/.mxcli/mxbuild/*/modeler/mx check test.mpr

# Compare BSON if issues persist
mxcli bson dump -p test.mpr --type page --object "Test.TestPage" --format ndsl
```

## Usage

Templates are automatically used when creating pluggable widgets via MDL:

```sql
COMBOBOX myCombo (Label: 'Country', Attribute: Country)
```

### 3-Tier Widget Registry

When creating a pluggable widget, mxcli resolves definitions and templates:

| Priority | Location | Scope |
|----------|----------|-------|
| 1 (highest) | `<project>/.mxcli/widgets/*.def.json` | Project-specific overrides |
| 2 | `~/.mxcli/widgets/*.def.json` | Global user definitions |
| 3 (lowest) | `sdk/widgets/definitions/*.def.json` (embedded) | Built-in definitions |

Each `.def.json` declares property mappings and child slots; the `PluggableWidgetEngine` applies them to the BSON template at build time. See `docs/plans/2026-03-25-pluggable-widget-engine-design.md` for the full architecture.

### Widget Version Drift

Static templates are tied to the widget version they were extracted from. If the target project has a **newer** `.mpk`, the MPK augmentation mechanism (described above) handles this at runtime by merging property changes from the `.mpk` XML.

For cases where augmentation is insufficient, extract a fresh template from a Studio Pro project using the newer widget version.

## TextTemplate Property Requirements

Properties with `"Type": "TextTemplate"` in the Type definition require special handling. They **cannot** be `null` in the Object section.

### Problem: CE0463 "widget definition has changed"

If a TextTemplate property is `null` in the Object section, Studio Pro shows:
```
CE0463: The definition of this widget has changed. Update this widget...
```

### Required Structure

TextTemplate properties must have a proper `Forms$ClientTemplate` structure:

```json
"TextTemplate": {
  "$ID": "<32-char-guid>",
  "$Type": "Forms$ClientTemplate",
  "Fallback": {
    "$ID": "<32-char-guid>",
    "$Type": "Texts$Text",
    "Items": []
  },
  "Parameters": [],
  "Template": {
    "$ID": "<32-char-guid>",
    "$Type": "Texts$Text",
    "Items": []
  }
}
```

### Important: Empty Arrays

Empty arrays must be `[]`, NOT `[2]`:
```json
// WRONG - serializes as array containing integer 2
"Items": [2]

// CORRECT - truly empty array
"Items": []
```

### How to Identify TextTemplate Properties

1. Search the Type section for `"Type": "TextTemplate"`
2. Note the `$ID` from the parent `ValueType` object
3. Find Object properties where `Value.TypePointer` matches that ID
4. Update those properties' `TextTemplate` from `null` to proper structure

### Affected Widgets

Filter widgets commonly have TextTemplate properties:
- **TextFilter**: `placeholder`, `screenReaderButtonCaption`, `screenReaderInputCaption`
- **DateFilter**: `placeholder`, `screenReaderButtonCaption`, `screenReaderCalendarCaption`, `screenReaderInputCaption`
- **DropdownFilter**: `emptyOptionCaption`, `ariaLabel`, `emptySelectionCaption`, `filterInputPlaceholderCaption`
- **NumberFilter**: `placeholder`, `screenReaderButtonCaption`, `screenReaderInputCaption`

## Key Source Files

| File | Purpose |
|------|---------|
| `sdk/widgets/loader.go` | Template loading, 3-phase ID remapping, MPK augmentation, placeholder detection |
| `sdk/widgets/mpk/mpk.go` | .mpk ZIP parsing, XML property extraction, FindMPK |
| `sdk/widgets/definitions/*.def.json` | Built-in widget definition files |
| `mdl/executor/widget_engine.go` | PluggableWidgetEngine, 6 operations, Build() pipeline |
| `mdl/executor/widget_registry.go` | 3-tier WidgetRegistry, load-time validation |
| `mdl/executor/cmd_pages_builder_input.go` | `updateWidgetPropertyValue()`, TypePointer matching |
| `cmd/mxcli/cmd_widget.go` | `mxcli widget extract/list` CLI commands |

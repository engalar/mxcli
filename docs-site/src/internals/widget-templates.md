# Widget Template System

Pluggable widgets (DataGrid2, ComboBox, Gallery, etc.) require embedded template definitions for correct BSON serialization. This page explains how the template system works.

## Why Templates Are Needed

Pluggable widgets in Mendix are defined by two components:

1. **Type** (`PropertyTypes` schema) -- defines what properties the widget accepts
2. **Object** (`WidgetObject` defaults) -- provides default values for all properties

Both must be present in the BSON output. The type defines the schema; the object provides a valid instance. If the object's property structure does not match the type's schema, Studio Pro reports **CE0463 "widget definition changed"**.

## Template Location

Templates are embedded in the binary via Go's `go:embed` directive:

```
sdk/widgets/
├── loader.go                  # Template loading with go:embed
└── templates/
    ├── README.md              # Template extraction requirements
    ├── 10.6.0/               # Templates by Mendix version
    │   ├── DataGrid2.json
    │   ├── ComboBox.json
    │   ├── Gallery.json
    │   └── ...
    └── 11.6.0/
        ├── DataGrid2.json
        └── ...
```

Each JSON file contains both the `type` and `object` fields for a specific widget at a specific Mendix version.

## Template Structure

A template JSON file looks like:

```json
{
  "type": {
    "$Type": "CustomWidgets$WidgetPropertyTypes",
    "Properties": [
      { "Name": "columns", "Type": "Object", ... },
      { "Name": "showLabel", "Type": "Boolean", ... }
    ]
  },
  "object": {
    "$Type": "CustomWidgets$WidgetObject",
    "Properties": [
      { "Name": "columns", "Value": [] },
      { "Name": "showLabel", "Value": true }
    ]
  }
}
```

## Extracting Templates

Templates must be extracted from **Studio Pro-created widgets**, not generated programmatically. The extraction process:

1. Create a widget instance in Studio Pro
2. Save the project
3. Extract the BSON from the `.mpr` file
4. Convert the `type` and `object` sections to JSON
5. Save as a template file

This ensures the property structure exactly matches what Studio Pro expects.

> **Important:** Programmatically generated templates often have subtle differences in property ordering, default values, or nested structures that cause CE0463 errors. Always extract from Studio Pro.

## Loading Templates at Runtime

The `loader.go` file provides functions to load templates:

```go
// Load a widget template for the project's Mendix version
template, err := widgets.LoadTemplate("DataGrid2", mendixVersion)
```

The loader searches for the most specific version match, falling back to earlier versions if an exact match is not available.

## JSON Templates vs BSON Serialization

When editing template JSON files, use standard JSON conventions:

- Empty arrays are `[]` (not `[3]`)
- Booleans are `true`/`false`
- Strings are `"quoted"`

The array count prefixes (`[3, ...]`) required by Mendix BSON are added automatically during serialization. Writing `[2]` in a JSON template creates an array containing the integer 2, not an empty BSON array.

## Debugging CE0463 Errors

If a page fails with CE0463 after creation:

1. Create the same widget manually in Studio Pro
2. Extract its BSON from the saved project
3. Compare the template's `object` properties against the Studio Pro version
4. Look for missing properties, wrong default values, or incorrect nesting
5. Update the template JSON to match

See the debug workflow in `.claude/skills/debug-bson.md` for detailed steps.

# Pluggable Widget Engine

The Pluggable Widget Engine replaces hardcoded Go builder functions with a data-driven system. Widget behavior is described in declarative `.def.json` files, and a generic engine applies them against BSON templates at build time.

## Architecture Overview

```
MDL Script: COMBOBOX cmbStatus (Label: 'Status', Attribute: Priority)
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│                    WidgetRegistry                           │
│  3-tier lookup: embedded → ~/.mxcli/widgets/ → .mxcli/     │
│                                                             │
│  "COMBOBOX" → combobox.def.json                             │
│  "GALLERY"  → gallery.def.json                              │
│  "MYWIDGET" → mywidget.def.json  (user-defined)             │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                PluggableWidgetEngine.Build()                 │
│                                                             │
│  1. Load template   (combobox.json → augment from .mpk)     │
│  2. Select mode     (enum mode vs association mode)          │
│  3. Apply mappings  (set AttributeRef, PrimitiveValue, etc.) │
│  4. Apply child slots  (embed child widgets into BSON)       │
│  5. Assemble widget (CustomWidgets$CustomWidget)             │
│                                                             │
│            OperationRegistry                                │
│  ┌──────────┬───────────┬───────────┬──────────┬─────────┐  │
│  │attribute │association│ primitive │datasource│ widgets │  │
│  │          │           │           │selection │         │  │
│  └──────────┴───────────┴───────────┴──────────┴─────────┘  │
└─────────────────────────────────────────────────────────────┘
                      │
                      ▼
               CustomWidget BSON
         (written to .mpr / .mxunit)
```

## Widget Definitions (`.def.json`)

Each pluggable widget has a definition file that maps MDL syntax to template properties.

### ComboBox Example (two modes)

```json
{
  "widgetId": "com.mendix.widget.web.combobox.Combobox",
  "mdlName": "COMBOBOX",
  "templateFile": "combobox.json",
  "defaultEditable": "Always",
  "modes": [
    {
      "name": "association",
      "condition": "hasDataSource",
      "propertyMappings": [
        {"propertyKey": "optionsSourceType", "value": "association", "operation": "primitive"},
        {"propertyKey": "optionsSourceAssociationDataSource", "source": "DataSource", "operation": "datasource"},
        {"propertyKey": "attributeAssociation", "source": "Association", "operation": "association"},
        {"propertyKey": "optionsSourceAssociationCaptionAttribute", "source": "CaptionAttribute", "operation": "attribute"}
      ]
    },
    {
      "name": "default",
      "propertyMappings": [
        {"propertyKey": "attributeEnumeration", "source": "Attribute", "operation": "attribute"}
      ]
    }
  ]
}
```

### Gallery Example (child slots)

```json
{
  "widgetId": "com.mendix.widget.web.gallery.Gallery",
  "mdlName": "GALLERY",
  "templateFile": "gallery.json",
  "propertyMappings": [
    {"propertyKey": "datasource", "source": "DataSource", "operation": "datasource"},
    {"propertyKey": "itemSelection", "source": "Selection", "operation": "selection", "default": "Single"},
    {"propertyKey": "pageSize", "value": "20", "operation": "primitive"}
  ],
  "childSlots": [
    {"propertyKey": "content", "mdlContainer": "TEMPLATE", "operation": "widgets"},
    {"propertyKey": "filtersPlaceholder", "mdlContainer": "FILTER", "operation": "widgets"}
  ]
}
```

## Mode Selection

Modes are evaluated in definition order. The first mode whose condition matches the MDL AST is selected. A mode without a condition is the fallback (default).

| Condition | Checks |
|-----------|--------|
| `hasDataSource` | AST widget has a `DataSource` property |
| `hasAttribute` | AST widget has an `Attribute` property |
| `hasProp:XYZ` | AST widget has a property named `XYZ` |
| *(none)* | Fallback -- selected if no other mode matches |

## Six Built-in Operations

| Operation | What It Sets | Input |
|-----------|-------------|-------|
| `attribute` | `Value.AttributeRef` | Qualified path (`Module.Entity.Attr`) |
| `association` | `Value.EntityRef` (IndirectEntityRef + EntityRefStep) | Association path + target entity |
| `primitive` | `Value.PrimitiveValue` | Static string value |
| `datasource` | `Value.DataSource` | BSON data source object |
| `selection` | `Value.Selection` | `"Single"`, `"Multi"`, or `"None"` |
| `widgets` | `Value.Widgets` array | Serialized child widget BSON |

All operations are registered in an `OperationRegistry`. Custom operations can be added without modifying the engine.

## Mapping Order Dependency

**The engine processes property mappings in array order.** Some operations depend on side effects:

- `datasource` sets `pageBuilder.entityContext` as a side effect
- `association` reads `pageBuilder.entityContext` to resolve the target entity

Therefore, in any mode using both, **datasource must come before association** in the mappings array. This is enforced at definition load time -- a validation error is raised if an `association` mapping appears before any `datasource` mapping.

## Source/Operation Compatibility

Not all source/operation combinations are valid. These are rejected at load time:

| Source | Incompatible Operations |
|--------|------------------------|
| `Attribute` | `association`, `datasource` |
| `Association` | `attribute`, `datasource` |
| `DataSource` | `attribute`, `association` |

## Version Handling

### The Problem

Widget property schemas change between Mendix versions. The Gallery widget has 23 properties in Mendix 10.24 (widget v3.0.1) but 33 in 11.6.3. Writing BSON with wrong properties causes CE0463 ("widget definition changed").

### The Solution: Baseline + Augmentation

```
Embedded template (11.6.0 baseline)
         │
         ▼
    Deep clone (never mutate cache)
         │
         ▼
    augmentFromMPK()
    ├── FindMPK(projectDir, widgetID)
    │     Scan project/widgets/*.mpk
    │     Match by widget ID from package.xml
    │
    ├── ParseMPK(mpkPath)
    │     Extract property definitions from widget XML
    │     Extract widget version from package.xml
    │
    └── AugmentTemplate(clone, mpkDef)
          Compare template keys vs .mpk keys:
          ├── Missing (in .mpk, not in template) → clone from exemplar or create
          └── Stale (in template, not in .mpk) → remove
         │
         ▼
    Augmented template (matches project's widget version)
         │
         ▼
    3-phase BSON conversion (ID remap → Type → Object)
```

### Why the Baseline Must Be Conservative

The embedded template should be from the **oldest supported Mendix version**, not the newest:

- **Augmentation adds properties:** New properties in a newer `.mpk` are cloned from type-matching exemplars or created with sensible defaults. This is reliable.
- **Augmentation removes properties:** Extra properties from the baseline that don't exist in an older `.mpk` are stripped cleanly.
- **Augmentation cannot restructure:** If a newer template has fundamentally different property types, nesting, or internal structure, augmentation fails silently. Starting from an older baseline avoids this.

### Example: Gallery Across Versions

| Version | Properties | Augmentation Action |
|---------|-----------|-------------------|
| 10.24 (v3.0.1) | 23 | Remove 10 stale properties from 11.6 baseline |
| 11.6.0 | 23 | No augmentation needed (matches baseline) |
| 11.6.3 | 33 | Add 10 missing properties from .mpk |
| 11.8.0 | 33+ | Add any new properties from .mpk |

## Widget Registry (3-Tier)

Definitions are loaded in priority order. Higher-priority definitions override lower ones:

| Priority | Location | Use Case |
|----------|----------|----------|
| 1 (highest) | `<project>/.mxcli/widgets/*.def.json` | Project-specific widget overrides |
| 2 | `~/.mxcli/widgets/*.def.json` | Global user-defined widgets |
| 3 (lowest) | `sdk/widgets/definitions/` (embedded) | Built-in ComboBox, Gallery |

Definitions are looked up by MDL name (case-insensitive): `COMBOBOX`, `GALLERY`, or any custom name.

### Validation at Load Time

When a definition is loaded, it is validated:

- All `operation` fields must reference a registered operation
- `Source` and `operation` must be compatible (see table above)
- `association` mappings must appear after `datasource` mappings
- `widgetId` and `mdlName` must be non-empty

Invalid definitions produce an immediate error rather than failing silently at build time.

## Adding a Custom Widget

### Step 1: Extract from `.mpk`

```bash
mxcli widget extract --mpk widgets/MyRatingWidget.mpk
# Creates: .mxcli/widgets/myratingwidget.def.json (skeleton)
```

The `extract` command parses the `.mpk` ZIP, reads the widget XML, and auto-infers operations from property types:

| XML Type | Inferred Operation |
|----------|-------------------|
| `attribute` | `attribute` |
| `association` | `association` |
| `datasource` | `datasource` |
| `expression` | *(skipped -- manual)* |
| `widgets` | `widgets` (child slot) |
| `enumeration` | `primitive` |
| `boolean` | `primitive` |
| `integer` | `primitive` |

### Step 2: Extract Template

Create the widget in Studio Pro, then extract its BSON:

```bash
mxcli bson dump -p App.mpr --type page --object "Module.TestPage" --format json
```

Save the `type` and `object` sections as `.mxcli/widgets/myratingwidget.json`.

### Step 3: Edit the Definition

Adjust the generated `.def.json` to map MDL properties correctly. Set the `mdlName` to your desired keyword:

```json
{
  "widgetId": "com.example.widget.RatingWidget",
  "mdlName": "RATING",
  "templateFile": "myratingwidget.json",
  "propertyMappings": [
    {"propertyKey": "ratingAttribute", "source": "Attribute", "operation": "attribute"},
    {"propertyKey": "maxRating", "value": "5", "operation": "primitive"}
  ]
}
```

### Step 4: Use in MDL

```sql
CREATE PAGE MyModule.ReviewPage (...) {
  CONTAINER main () {
    RATING starRating (Label: 'Rating', Attribute: Review_Rating)
  }
}
```

## What Stays Hardcoded

**Native Mendix widgets** (TextBox, DataView, ListView, LayoutGrid, Container, etc.) use `Forms$TextBox`, `Forms$DataView`, etc. -- NOT `CustomWidgets$CustomWidget`. These stay as hardcoded builders because they have fundamentally different BSON structures and don't use the template system.

## Key Source Files

| File | Purpose |
|------|---------|
| `mdl/executor/widget_engine.go` | PluggableWidgetEngine, OperationRegistry, Build() pipeline |
| `mdl/executor/widget_registry.go` | 3-tier definition loading, validation |
| `sdk/widgets/definitions/*.def.json` | Built-in widget definitions |
| `sdk/widgets/loader.go` | Template loading, 3-phase ID remapping |
| `sdk/widgets/augment.go` | MPK augmentation (add/remove properties) |
| `sdk/widgets/mpk/mpk.go` | .mpk ZIP parsing, property extraction |
| `sdk/widgets/templates/mendix-11.6/*.json` | Embedded baseline templates |
| `cmd/mxcli/cmd_widget.go` | `mxcli widget extract/list` CLI |

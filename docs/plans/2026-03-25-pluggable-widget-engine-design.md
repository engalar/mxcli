# Pluggable Widget Engine: Declarative Widget Build System

**Date**: 2026-03-25
**Status**: Implemented

## Problem

Each pluggable widget currently requires a hardcoded Go builder function (`buildComboBoxV3`, `buildGalleryV3`, etc.) plus a switch-case registration. Adding a new widget requires changes in 4 places:

1. `sdk/pages/pages_widgets_advanced.go` — Add WidgetID constant
2. `sdk/widgets/templates/` — Add JSON template
3. `mdl/executor/cmd_pages_builder_v3_pluggable.go` — Write a dedicated build function (50-200 lines)
4. `mdl/executor/cmd_pages_builder_v3.go` — Add case in `buildWidgetV3()` switch

This causes:
- Users cannot add pluggable widget support on their own
- Significant code duplication (30+ builder functions sharing ~80% boilerplate)
- Maintenance cost grows linearly with widget count

## Solution: Declarative Widget Definitions + Generic Build Engine

### Architecture

```
┌─────────────────────────────────────────────────────┐
│                  WidgetRegistry                      │
│  ┌────────────┐ ┌────────────┐ ┌──────────────────┐ │
│  │ combobox   │ │ gallery    │ │ user-custom      │ │
│  │ .def.json  │ │ .def.json  │ │ .def.json        │ │
│  └─────┬──────┘ └─────┬──────┘ └────────┬─────────┘ │
│        └──────────────┼─────────────────┘            │
│                       ▼                               │
│           PluggableWidgetEngine                       │
│     ┌──────────────────────────────┐                  │
│     │ 1. loadTemplate()            │                  │
│     │ 2. selectMode(conditions)    │                  │
│     │ 3. applyPropertyMappings()   │                  │
│     │ 4. applyChildSlots()         │                  │
│     │ 5. buildCustomWidget()       │                  │
│     └──────────────────────────────┘                  │
│                       │                               │
│            OperationRegistry                          │
│     ┌─────────┬───────────┬───────────┬──────┐       │
│     │attribute│primitive  │datasource │ ...  │       │
│     │         │           │           │extend│       │
│     └─────────┴───────────┴───────────┴──────┘       │
└─────────────────────────────────────────────────────┘
```

### Widget Definition Format (`.def.json`)

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
      "description": "Association mode with DataSource",
      "propertyMappings": [
        {
          "propertyKey": "optionsSourceType",
          "value": "association",
          "operation": "primitive"
        },
        {
          "propertyKey": "optionsSourceAssociationDataSource",
          "source": "DataSource",
          "operation": "datasource"
        },
        {
          "propertyKey": "attributeAssociation",
          "source": "Attribute",
          "operation": "association"
        },
        {
          "propertyKey": "optionsSourceAssociationCaptionAttribute",
          "source": "CaptionAttribute",
          "operation": "attribute"
        }
      ]
    },
    {
      "name": "default",
      "description": "Enumeration mode",
      "propertyMappings": [
        {
          "propertyKey": "attributeEnumeration",
          "source": "Attribute",
          "operation": "attribute"
        }
      ]
    }
  ]
}
```

Gallery (with child slots):

```json
{
  "widgetId": "com.mendix.widget.web.gallery.Gallery",
  "mdlName": "GALLERY",
  "templateFile": "gallery.json",
  "defaultEditable": "Always",

  "propertyMappings": [
    {"propertyKey": "advanced", "value": "false", "operation": "primitive"},
    {"propertyKey": "datasource", "source": "DataSource", "operation": "datasource"},
    {"propertyKey": "itemSelection", "source": "Selection", "operation": "selection"},
    {"propertyKey": "itemSelectionMode", "value": "clear", "operation": "primitive"},
    {"propertyKey": "desktopItems", "value": "1", "operation": "primitive"},
    {"propertyKey": "tabletItems", "value": "1", "operation": "primitive"},
    {"propertyKey": "phoneItems", "value": "1", "operation": "primitive"},
    {"propertyKey": "pageSize", "value": "20", "operation": "primitive"},
    {"propertyKey": "pagination", "value": "buttons", "operation": "primitive"},
    {"propertyKey": "pagingPosition", "value": "below", "operation": "primitive"},
    {"propertyKey": "showEmptyPlaceholder", "value": "none", "operation": "primitive"},
    {"propertyKey": "onClickTrigger", "value": "single", "operation": "primitive"}
  ],

  "childSlots": [
    {"propertyKey": "content", "mdlContainer": "TEMPLATE", "operation": "widgets"},
    {"propertyKey": "emptyPlaceholder", "mdlContainer": "EMPTYPLACEHOLDER", "operation": "widgets"},
    {"propertyKey": "filtersPlaceholder", "mdlContainer": "FILTER", "operation": "widgets"}
  ]
}
```

### 6 Operation Types

All existing pluggable widget builders use combinations of these 6 operations:

| Operation | Function | Input | Description |
|-----------|----------|-------|-------------|
| `attribute` | `setAttributeRef()` | `source` → MDL Attribute prop | Sets `AttributeRef` with qualified path (`Module.Entity.Attr`) |
| `association` | `setAssociationRef()` | `source` → MDL Attribute prop | Sets association path + entity ref |
| `primitive` | `setPrimitiveValue()` | `value` or `source` | Sets `PrimitiveValue` string (enum selection, boolean, etc.) |
| `datasource` | `setDataSource()` | `source` → MDL DataSource prop | Builds and sets `DataSource` object |
| `selection` | `setSelectionMode()` | `value` or `source` | Set widget selection mode (Single/Multi) |
| `widgets` | inline child BSON | `childSlots` config | Embeds serialized child widgets into `Widgets` array |

Operations are registered in an `OperationRegistry` and new types can be added without modifying the engine.

### Definition Schema (`WidgetDefinition`)

```go
type WidgetDefinition struct {
    WidgetID         string                      `json:"widgetId"`
    MDLName          string                      `json:"mdlName"`
    TemplateFile     string                      `json:"templateFile"`
    DefaultEditable  string                      `json:"defaultEditable"`
    DefaultSelection string                      `json:"defaultSelection,omitempty"`

    // Simple case: single mode
    PropertyMappings []PropertyMapping           `json:"propertyMappings,omitempty"`
    ChildSlots       []ChildSlotMapping          `json:"childSlots,omitempty"`

    // Multi-mode case (e.g., ComboBox enum vs association)
    // Uses slice instead of map to preserve evaluation order (first-match-wins semantics)
    Modes            []WidgetMode                `json:"modes,omitempty"`
}

type WidgetMode struct {
    Condition        string                      `json:"condition,omitempty"`
    Description      string                      `json:"description,omitempty"`
    PropertyMappings []PropertyMapping           `json:"propertyMappings"`
    ChildSlots       []ChildSlotMapping          `json:"childSlots,omitempty"`
}

type PropertyMapping struct {
    PropertyKey string `json:"propertyKey"`       // Template property key
    Source      string `json:"source,omitempty"`   // MDL AST property name
    Value       string `json:"value,omitempty"`    // Static value (mutually exclusive with Source)
    Operation   string `json:"operation"`          // attribute|association|primitive|datasource
    Default     string `json:"default,omitempty"`  // Default value if source is empty
}

type ChildSlotMapping struct {
    PropertyKey  string `json:"propertyKey"`       // Template property key for widget list
    MDLContainer string `json:"mdlContainer"`      // MDL child container name (TEMPLATE, FILTER)
    Operation    string `json:"operation"`          // Always "widgets"
}
```

### Critical: Property Mapping Order Dependency

**The engine processes `propertyMappings` in array order.** Some operations depend on side effects of earlier ones:

- `datasource` sets `pageBuilder.entityContext` as a side effect
- `association` reads `pageBuilder.entityContext` to resolve the target entity

Therefore, in any mode that uses both, **`datasource` must come before `association`** in the mappings array. Getting this wrong produces silently incorrect BSON (wrong entity reference).

### Operation Validation

Operation names in `.def.json` files are validated at load time against the 6 known operations: `attribute`, `association`, `primitive`, `selection`, `datasource`, `widgets`. Invalid operation names produce an error when `NewWidgetRegistry()` or `LoadUserDefinitions()` runs, rather than failing silently at build time.

### Mode Selection Conditions

Built-in conditions (extensible):

| Condition | Logic |
|-----------|-------|
| `hasDataSource` | `w.GetDataSource() != nil` |
| `hasAttribute` | `w.GetAttribute() != ""` |
| `hasProp:X` | `w.GetStringProp("X") != ""` |
| (none) | Fallback — first no-condition mode wins if multiple exist |

### Engine Flow

```go
func (e *PluggableWidgetEngine) Build(def *WidgetDefinition, w *ast.WidgetV3) (*pages.CustomWidget, error) {
    // 1. Load template
    tmplType, tmplObj, tmplIDs, objTypeID, err := widgets.GetTemplateFullBSON(
        def.WidgetID, mpr.GenerateID, e.projectPath)

    // 2. Select mode
    mode := e.selectMode(def, w)

    // 3. Apply property mappings
    propTypeIDs := convertPropertyTypeIDs(tmplIDs)
    updatedObj := tmplObj
    for _, mapping := range mode.PropertyMappings {
        op := e.operations.Get(mapping.Operation)
        value := e.resolveSource(mapping, w)
        updatedObj = op.Apply(updatedObj, propTypeIDs, mapping.PropertyKey, value, e.buildCtx)
    }

    // 4. Apply child slots
    for _, slot := range mode.ChildSlots {
        childBSONs := e.buildChildWidgets(w, slot.MDLContainer)
        updatedObj = e.applyWidgetSlot(updatedObj, propTypeIDs, slot.PropertyKey, childBSONs)
    }

    // 5. Build CustomWidget
    return &pages.CustomWidget{
        BaseWidget:        pages.BaseWidget{...},
        Editable:          def.DefaultEditable,
        RawType:           tmplType,
        RawObject:         updatedObj,
        PropertyTypeIDMap: propTypeIDs,
        ObjectTypeID:      objTypeID,
    }, nil
}
```

### Integration with buildWidgetV3()

```go
func (pb *pageBuilder) buildWidgetV3(w *ast.WidgetV3) (pages.Widget, error) {
    switch strings.ToUpper(w.Type) {
    // Native Mendix widgets — keep hardcoded (different BSON structure)
    case "DATAVIEW":
        return pb.buildDataViewV3(w)
    case "LISTVIEW":
        return pb.buildListViewV3(w)
    case "TEXTBOX":
        return pb.buildTextBoxV3(w)
    // ... other native widgets

    default:
        // All pluggable widgets go through declarative engine
        if def, ok := pb.widgetRegistry.Get(strings.ToUpper(w.Type)); ok {
            return pb.pluggableEngine.Build(def, w)
        }
        return nil, fmt.Errorf("unsupported widget type: %s", w.Type)
    }
}
```

### User Extension Points

**File locations (project-level takes priority):**

```
project/
└── .mxcli/
    └── widgets/
        ├── my-rating-widget.def.json      # Widget definition
        └── my-rating-widget.json          # BSON template

~/.mxcli/
└── widgets/
    ├── shared-chart.def.json              # Global widget definition
    └── shared-chart.json                  # Global BSON template
```

**Template extraction tool:**

```bash
# Extract template from .mpk widget package
mxcli widget extract --mpk path/to/widget.mpk

# Generates:
#   .mxcli/widgets/<widget-name>.json      (template with type + object)
#   .mxcli/widgets/<widget-name>.def.json  (skeleton definition)
```

### Migration Plan

| Phase | Scope | Deliverable |
|-------|-------|-------------|
| 1 | Engine skeleton + ComboBox migration | Validate the approach with simplest widget |
| 2 | Migrate Gallery, DataGrid, 4 Filters | Delete 6 hardcoded builder functions (~600 lines) |
| 3 | User extension: registry scan + extract tool | Users can add custom widget support |
| 4 | LSP integration | Completion, hover, diagnostics for custom widgets |

### What Stays Hardcoded

**Native Mendix widgets** (TextBox, DataView, ListView, LayoutGrid, Container, etc.) use a fundamentally different BSON structure (`Forms$TextBox`, `Forms$DataView`) — NOT `CustomWidgets$CustomWidget`. These stay as hardcoded builders because:
- They don't use the template system
- Their BSON structure varies significantly per widget type
- There are ~20 of them and they're stable (rarely new ones added)

### Risk Analysis

| Risk | Mitigation |
|------|------------|
| Complex widgets may not fit declarative model | `modes` + extensible operations provide escape hatches |
| Template version drift | Existing `augment.go` handles .mpk sync, works unchanged |
| Performance regression | Template loading is already cached; engine adds minimal overhead |
| User-provided templates may be invalid | Validate on load: check type+object sections exist, PropertyKey coverage |
| MPK zip-bomb attack | `ParseMPK` enforces per-file (50MB) and total (200MB) extraction limits |
| Invalid operation names in .def.json | Validated at load time, not build time — immediate feedback |
| Engine init failure retried on every widget | Init error cached; subsequent widgets skip immediately |

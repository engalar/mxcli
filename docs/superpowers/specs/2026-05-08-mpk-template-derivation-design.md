# MPK-Derived Widget Templates

**Date:** 2026-05-08  
**Status:** Approved  
**Scope:** `sdk/widgets/`

## Problem

mxcli can only use pluggable widgets that have pre-built templates embedded in the binary (`sdk/widgets/templates/mendix-11.6/*.json`). Non-embedded widgets (e.g. third-party marketplace widgets, custom widgets like CrusherWidgets) require a manual Studio Pro extraction workflow before they can be used in `create page` MDL commands.

## Goal

Any pluggable widget whose `.mpk` file is present in `project/widgets/` can be used in MDL commands without pre-built templates. The derivation happens transparently at runtime when the widget is first referenced.

## Approach

**Approach B — new `generate.go`, parallel to `augment.go`.**

All logic for building individual PropertyType/Property pairs from scratch already exists in `augment.go` (`createPropertyPair`, `createDefaultValueType`, `createDefaultWidgetValue`, `buildNestedObjectType`, `xmlTypeToBSONType`). `generate.go` only needs to build the outer `CustomWidgetType` + `WidgetObject` shells and call those existing functions.

Rejected alternatives:
- **A (extend augment.go)**: mixed "patch" and "generate" semantics in one file → harder to maintain
- **C (pipeline refactor)**: highest risk, touches stable loader.go core for marginal architectural gain

## Architecture

### File changes

```
sdk/widgets/
├── mpk/mpk.go     — unchanged (ParseMPK / FindMPK already complete)
├── augment.go     — unchanged (all helpers reused as-is)
├── loader.go      — +25 lines: fallback path after embedded cache miss
└── generate.go    — new, ~60 lines: outer shell construction
```

### Template resolution flow

```
GetTemplateBSON / GetTemplateFullBSON(widgetID, idGenerator, projectPath)
  │
  └─ getOrGenerateTemplate(widgetID, projectPath)   ← new internal helper
       │
       ├─ 1. GetTemplate(widgetID)                  ← embedded cache (existing)
       │       └─ found → augmentFromMPK() → return
       │
       ├─ 2. generatedCache.Load(widgetID)          ← session cache (new)
       │       └─ found → return
       │
       ├─ 3. mpk.FindMPK(projectPath, widgetID)
       │       └─ not found → return nil, nil
       │
       ├─ 4. mpk.ParseMPK(mpkPath) → WidgetDefinition
       │
       ├─ 5. GenerateFromMPK(def) → WidgetTemplate  ← new
       │
       └─ 6. generatedCache.Store(widgetID, tmpl) → return
```

**Performance fallback (deferred):** If step 5 proves too slow for large widgets, write the generated template to `.mxcli/widgets/<widgetID>.json` and check that path between steps 2 and 3. This can be added without restructuring.

## generate.go

### API

```go
// GenerateFromMPK builds a complete WidgetTemplate from an MPK WidgetDefinition.
// All $IDs are placeholder IDs; loader.go's collectIDs remaps them to real UUIDs
// before BSON serialization.
func GenerateFromMPK(def *mpk.WidgetDefinition) *WidgetTemplate
```

### Algorithm

```
typeID    = placeholderID()
objTypeID = placeholderID()

for each p in def.Properties:
    bsonType = xmlTypeToBSONType(p.Type)
    skip if bsonType == ""
    pt, prop = createPropertyPair(p, bsonType)
    append pt  → propTypes
    append prop → objProps

type = map{
    "$ID":      typeID,
    "$Type":    "CustomWidgets$CustomWidgetType",
    "WidgetId": def.ID,
    "ObjectType": map{
        "$ID":           objTypeID,
        "$Type":         "CustomWidgets$WidgetObjectType",
        "PropertyTypes": [2, ...propTypes],
    },
}

object = map{
    "$ID":         placeholderID(),
    "$Type":       "CustomWidgets$WidgetObject",
    "TypePointer": typeID,
    "Properties":  [2, ...objProps],
}

return &WidgetTemplate{
    WidgetID: def.ID,
    Name:     def.Name,
    Version:  def.Version,
    Type:     type,
    Object:   object,
}
```

### System properties

System properties (Label, Visibility, Editability) are **not** added by `GenerateFromMPK`. Studio Pro injects them automatically when opening the project. This matches the current behaviour of `AugmentTemplate`.

## loader.go changes

`GetTemplate(widgetID string)` has no `projectPath` parameter; it returns `nil, nil` on cache miss.
`projectPath` is only available in `GetTemplateBSON` / `GetTemplateFullBSON`.

**Strategy:** introduce a package-internal `getOrGenerateTemplate(widgetID, projectPath string)` called by both BSON functions instead of `GetTemplate` directly.

```go
// getOrGenerateTemplate returns the template for widgetID, falling back to
// MPK-based generation when no embedded template exists.
func getOrGenerateTemplate(widgetID, projectPath string) (*WidgetTemplate, error) {
    // 1. embedded templates (existing path)
    if tmpl, err := GetTemplate(widgetID); err != nil || tmpl != nil {
        return tmpl, err
    }

    // 2. session cache of previously generated templates
    if cached, ok := generatedCache.Load(widgetID); ok {
        return cached.(*WidgetTemplate), nil
    }

    // 3. derive from MPK in project/widgets/
    if projectPath == "" {
        return nil, nil // no project path, can't locate MPK
    }
    mpkPath := mpk.FindMPK(projectPath, widgetID)
    if mpkPath == "" {
        return nil, nil // not found — callers treat nil as "widget unknown"
    }
    def, err := mpk.ParseMPK(mpkPath)
    if err != nil {
        return nil, fmt.Errorf("widget %q: parse MPK: %w", widgetID, err)
    }
    tmpl := GenerateFromMPK(def)
    generatedCache.Store(widgetID, tmpl)
    return tmpl, nil
}
```

`GetTemplateBSON` and `GetTemplateFullBSON` replace their `GetTemplate(widgetID)` call with `getOrGenerateTemplate(widgetID, projectPath)`. No other changes to those functions.

`generatedCache` is a package-level `sync.Map` (key: widgetID string, value: `*WidgetTemplate`). The cached value is the pre-ID-remapping template, identical in lifecycle to embedded templates.

## PropertyDef → BSON mapping

Already fully implemented in `augment.go:xmlTypeToBSONType()`. Covers all 17 known XML property types:

`attribute` `expression` `textTemplate` `widgets` `enumeration` `boolean` `integer` `datasource` `action` `selection` `association` `object` `string` `decimal` `icon` `image` `file`

Unknown types are silently skipped (property omitted from generated template).

## Tests

| Test | File | What it checks |
|------|------|----------------|
| `TestGenerateFromMPK_BasicTypes` | `generate_test.go` | string/boolean/integer/expression/attribute → PropertyTypes and Properties counts match, TypePointers cross-reference correctly |
| `TestGenerateFromMPK_NestedObject` | `generate_test.go` | object-type property with children → nested ObjectType built correctly |
| `TestGenerateFromMPK_UnknownTypeSkipped` | `generate_test.go` | unknown XML type → no panic, property count reduced by 1 |
| `TestGenerateFromMPK_PlaceholderIDsRemapped` | `generate_test.go` | after `GetTemplateFullBSON`, no `aa000000`-prefix IDs remain |
| `TestGetTemplate_FallsBackToMPK` | `loader_test.go` | CavitySelector widget ID + CrusherWidgets.mpk fixture → valid template returned without embedded template |

**Acceptance criterion:** `create page` MDL command referencing CavitySelector in CrusherCopilot project → Studio Pro opens without CE0463 error.

## Out of scope

- ALTER PAGE operations on generated widget instances (follow-on work)
- Caching generated templates to disk (deferred until performance evidence)
- `cmd/crusher-templates` CLI (superseded by this runtime approach; can be removed separately)

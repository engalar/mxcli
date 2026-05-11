# Proposal: Version-Aware BSON Schema Registry

> **Status: Superseded by [UNIFIED_SCHEMA_REGISTRY.md](UNIFIED_SCHEMA_REGISTRY.md)**
>
> The unified proposal keeps the registry concept and overall shape from this document
> but replaces the data sources: `mx dump-mpr` and `.mpk` XML parsing instead of
> TypeScript-derived reflection data + `supplements.json`. The unified proposal also
> adds workflow integration (init/refresh/check/lsp/skills), inspection commands,
> migration tooling, and native-vs-pluggable dispatch via data — none of which were
> covered here.
>
> Preserved for context.

## Problem Statement

Mendix documents are stored as BSON in `.mpr` files. The expected BSON structure varies by:

1. **Mendix platform version** — Properties are added, removed, or changed across versions (9.x → 10.x → 11.x). The reflection data shows ~42% type growth from 9.0 to 11.6, with property-level changes in nearly every release.
2. **Widget version** — Pluggable widgets (ComboBox, DataGrid2, etc.) define their own BSON schemas via `CustomWidgetType` PropertyTypes. A mismatch between the widget version in the project and the template we use causes CE0463 errors.
3. **Studio Pro extensions (Mendix 11+)** — Extensions can define entirely new document types (`CustomBlobDocument`) with custom storage formats.

The current implementation hardcodes BSON structure in hand-written parser/writer files (`parser_microflow.go`, `writer_microflow_actions.go`, etc.). This works but cannot adapt to version differences — it targets one version and silently produces incorrect BSON for others.

If a document is stored with missing properties, wrong field order, or a schema mismatch, Studio Pro will show errors or refuse to open the project. There is no external specification of these rules; they are hardcoded in Studio Pro itself.

## Current Architecture (What We Have)

```
Reflection data (json)              Hand-coded parsers/writers
  ~111 versions available             parser_microflow.go (1200+ lines)
  structures.json per version         parser_page.go (800+ lines)
  storageNames.json per version       writer_microflow_actions.go (400+ lines)
                                      writer_widgets.go (300+ lines)
         │                                      │
         ▼                                      ▼
  Code Generator                    runtime serialization
  (generates Go structs             (type-switches on Go types,
   with BSON tags — but             builds bson.D manually,
   these are not used               must know storage names,
   for serialization)               array prefixes, ref kinds)
```

Key problems:
- The **generated metamodel** (757 structs) is not connected to the **serialization layer**
- **Storage names** are hardcoded in switch statements, not derived from reflection data
- **Default values** for new properties are not applied when writing for newer versions
- **Widget templates** only cover Mendix 11.6 — no fallback for 10.x projects
- **No validation** — we don't know if our BSON output is correct until Studio Pro rejects it

## Proposed Architecture: Schema Registry

The core idea is a **runtime schema registry** that loads type definitions from reflection data and drives both serialization and validation. The hand-coded parsers/writers remain for complex logic but delegate field-level concerns to the registry.

```
┌─────────────────────────────────────────────────────────────┐
│                      schema Registry                        │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │ type Schemas  │  │ widget       │  │ Extension        │  │
│  │ (from refl.  │  │ Schemas      │  │ Schemas          │  │
│  │  data, per   │  │ (from widget │  │ (from extension  │  │
│  │  version)    │  │  packages,   │  │  manifests,      │  │
│  │              │  │  per version)│  │  per project)    │  │
│  └──────┬───────┘  └──────┬───────┘  └───────┬──────────┘  │
│         └─────────────┬────┘                  │             │
│                       ▼                       │             │
│            ┌──────────────────┐               │             │
│            │  Unified schema  │◄──────────────┘             │
│            │  Lookup          │                              │
│            └────────┬─────────┘                              │
│                     │                                        │
│         ┌───────────┼───────────┐                            │
│         ▼           ▼           ▼                            │
│  ┌────────────┐ ┌────────┐ ┌───────────┐                    │
│  │ Serializer │ │ Parser │ │ Validator │                     │
│  │ (write)    │ │ (read) │ │ (check)   │                    │
│  └────────────┘ └────────┘ └───────────┘                    │
└─────────────────────────────────────────────────────────────┘
```

## Detailed Design

### 1. Schema Registry (`schema/registry.go`)

The registry is the central lookup for type metadata at runtime.

```go
package schema

// Registry holds all type schemas for a specific Mendix version.
type Registry struct {
    version       string
    types         map[string]*TypeSchema    // keyed by qualified name
    storageIndex  map[string]*TypeSchema    // keyed by storage name (for parsing)
    widgetSchemas map[string]*WidgetSchema  // keyed by widget ID
    extSchemas    map[string]*ExtSchema     // keyed by extension document type
}

// TypeSchema describes one BSON document/element type.
type TypeSchema struct {
    QualifiedName string              // e.g. "DomainModels$entity"
    StorageName   string              // e.g. "DomainModels$EntityImpl"
    IsAbstract    bool
    properties    []*PropertySchema   // ordered — field order matters for BSON
    Defaults      map[string]any      // default values from reflection data
    Subtypes      []string            // qualified names of concrete subtypes
    Parent        string              // parent type for inheritance
}

// PropertySchema describes one field within a type.
type PropertySchema struct {
    Name            string
    StorageName     string            // BSON field name (may differ from Name)
    type            PropertyType      // Primitive, enum, Element, Unit
    ElementType     string            // qualified name if type is Element/Unit
    ReferenceKind   ReferenceKind     // PART, BY_NAME, BY_ID, LOCAL_BY_NAME
    IsList          bool
    ListEncoding    ListEncoding      // Compact(1), KeyValue(2), Array(3)
    required        bool
    DefaultValue    any
    IntroducedIn    string            // version where this property first appeared
    RemovedIn       string            // version where this property was removed (empty = still present)
}
```

**Loading**: The registry loads from the embedded reflection data JSON files. At project open time, we detect the Mendix version and load the matching schema (see "Reflection Data Version Count" for which versions to embed).

```go
// Load registry for a specific version.
func LoadRegistry(version string) (*Registry, error) {
    structures := loadStructures(version)    // {version}-structures.json
    storageNames := loadStorageNames(version) // {version}-storageNames.json
    return buildRegistry(structures, storageNames)
}
```

### 2. Schema-Aware Serialization

The key insight is that we don't need to replace the hand-coded writers — we need to **augment** them. The registry handles:

- **Field completeness**: Ensuring all required fields are present with defaults
- **Storage name resolution**: Looking up the correct `$type` value
- **Array encoding**: Applying the correct prefix (type 1, 2, or 3) per property
- **Reference encoding**: Choosing BY_NAME (string) vs BY_ID (binary UUID) per property

```go
// SchemaWriter wraps existing serialization with schema awareness.
type SchemaWriter struct {
    registry *Registry
}

// Complete fills in missing fields with defaults for the target version.
// Called after the hand-coded writer produces its bson.D.
func (sw *SchemaWriter) Complete(typeName string, doc bson.D) (bson.D, error) {
    schema := sw.registry.LookupByStorage(typeName)
    if schema == nil {
        return doc, nil // unknown type, pass through unchanged
    }

    existing := make(map[string]bool)
    for _, elem := range doc {
        existing[elem.Key] = true
    }

    // add missing properties with their default values
    for _, prop := range schema.Properties {
        if !existing[prop.StorageName] && prop.DefaultValue != nil {
            doc = append(doc, bson.E{key: prop.StorageName, value: prop.DefaultValue})
        }
    }

    return doc, nil
}

// Validate checks a bson.D document against its schema.
func (sw *SchemaWriter) Validate(typeName string, doc bson.D) []ValidationError {
    // check required fields are present
    // check field types match schema
    // check reference kinds are correct
    // check list encodings are correct
}
```

The existing hand-coded writers continue to handle complex logic (microflow action parameters, widget datasource binding, expression serialization). The schema layer sits on top as a safety net and completeness check.

### 3. Schema-Aware Parsing (Tolerant Reader)

For reading, the registry enables a **tolerant reader** pattern — we can parse documents even if we don't have hand-coded parsers for every type:

```go
// GenericParse reads any BSON document into a semi-structured representation
// using the schema to interpret field types and references.
func (r *Registry) GenericParse(raw map[string]interface{}) (*GenericDocument, error) {
    typeName := raw["$type"].(string)
    schema := r.LookupByStorage(typeName)

    doc := &GenericDocument{
        type:       typeName,
        schema:     schema,
        properties: make(map[string]any),
    }

    for _, prop := range schema.Properties {
        val, ok := raw[prop.StorageName]
        if !ok {
            continue
        }

        switch prop.ReferenceKind {
        case PART:
            // Recursively parse nested elements
            doc.Properties[prop.Name] = r.GenericParse(extractBsonMap(val))
        case BY_NAME_REFERENCE:
            doc.Properties[prop.Name] = val.(string) // qualified name
        case BY_ID_REFERENCE:
            doc.Properties[prop.Name] = extractBsonID(val) // UUID
        default:
            doc.Properties[prop.Name] = val
        }
    }

    return doc, nil
}
```

This generic parser provides a fallback for the ~48 metamodel domains we haven't implemented yet, and serves as the foundation for extension document support.

### 4. Widget Schema Registry

Widget schemas are different from platform schemas — they come from widget packages in the project, not from reflection data. Each project bundles its own widget versions in a `widgets/` folder.

```go
// WidgetSchema describes a pluggable widget's BSON structure.
type WidgetSchema struct {
    WidgetID      string                      // e.g. "com.mendix.widget.web.combobox.Combobox"
    version       string                      // e.g. "3.5.0"
    PropertyTypes []*WidgetPropertyTypeSchema // from CustomWidgetType
    DefaultObject bson.D                      // default WidgetObject
}

// WidgetPropertyTypeSchema describes one property of a widget.
type WidgetPropertyTypeSchema struct {
    key          string
    Category     string
    ValueType    string   // "enumeration", "TextTemplate", "expression", etc.
    DefaultValue any
    required     bool
}
```

**Loading strategy** (three tiers):

```
Tier 1: project widgets/  →  Extract schema from .mpk widget packages at project open time
Tier 2: Embedded templates →  Bundled templates (current sdk/widgets/templates/) as fallback
Tier 3: Generic passthrough → Unknown widgets preserved as raw BSON (round-trip safe)
```

```go
// WidgetRegistry manages widget schemas with tiered resolution.
type WidgetRegistry struct {
    projectWidgets  map[string]*WidgetSchema  // from project's widgets/ folder
    embeddedWidgets map[string]*WidgetSchema  // from go:embed templates
}

func (wr *WidgetRegistry) Lookup(widgetID string) *WidgetSchema {
    if s, ok := wr.projectWidgets[widgetID]; ok {
        return s  // prefer project-specific version
    }
    if s, ok := wr.embeddedWidgets[widgetID]; ok {
        return s  // fall back to embedded
    }
    return nil  // unknown widget — use raw passthrough
}
```

**Widget package extraction**: Widget `.mpk` files are ZIP archives containing a `package.xml` manifest. The `CustomWidgetType` BSON can be extracted from these packages. This is similar to what Studio Pro does when importing widgets.

### 5. Extension Document Support (Mendix 11+)

Studio Pro extensions define custom document types stored as `CustomBlobDocument` in the MPR. The reflection data already includes:

- `CustomBlobDocuments$CustomBlobDocument` with `Contents`, `CustomDocumentType`, `Metadata`
- `CustomBlobDocuments$CustomBlobDocumentMetadata` with `CreatedByExtension`, `ReadableTypeName`

Extensions can't define arbitrary BSON — they store their data as a blob within the `Contents` field. Our approach:

```go
// ExtSchema describes an extension-defined document type.
type ExtSchema struct {
    ExtensionID    string    // which extension defines this
    DocumentType   string    // the CustomDocumentType identifier
    ContentFormat  string    // "json", "xml", "binary", etc.
    // We don't interpret Contents — we round-trip it as-is
}

// ExtensionDocument represents a custom document from an extension.
type ExtensionDocument struct {
    ID              model.ID
    Name            string
    CustomType      string
    CreatedBy       string    // extension name
    ReadableType    string    // human-readable type name
    RawContents     []byte    // opaque blob — we preserve but don't interpret
}
```

For extensions, our strategy is **safe round-tripping**: read the document, preserve its contents byte-for-byte, write it back. We don't need to understand extension document internals — we just need to not corrupt them.

If we later need to modify extension documents (e.g., for an MDL command), we'd need the extension's own schema, which would be loaded from the extension manifest in the project.

### 6. Version Migration

The reflection data across ~111 versions gives us a diff-able history of every type. We can compute migrations:

```go
// Migration describes changes between two versions for one type.
type Migration struct {
    FromVersion string
    ToVersion   string
    TypeName    string
    added       []*PropertySchema    // new properties (need defaults)
    Removed     []*PropertySchema    // dropped properties (strip on write)
    changed     []*PropertyChange    // type/default changes
}

// PropertyChange describes a property that changed between versions.
type PropertyChange struct {
    Property       string
    OldDefault     any
    NewDefault     any
    OldStorageName string
    NewStorageName string    // rare but happens
}

// ComputeMigration diffs two version registries for a given type.
func ComputeMigration(from, to *Registry, typeName string) *Migration {
    fromSchema := from.Lookup(typeName)
    toSchema := to.Lookup(typeName)
    // diff properties...
}
```

**Migration application**: When writing a document for version Y that was read from version X:

```go
func (sw *SchemaWriter) Migrate(doc bson.D, fromVersion, toVersion string) (bson.D, error) {
    migration := ComputeMigration(
        LoadRegistry(fromVersion),
        LoadRegistry(toVersion),
        getType(doc),
    )

    // remove properties that don't exist in target version
    for _, removed := range migration.Removed {
        doc = removeField(doc, removed.StorageName)
    }

    // add new properties with defaults
    for _, added := range migration.Added {
        doc = addFieldWithDefault(doc, added.StorageName, added.DefaultValue)
    }

    return doc, nil
}
```

### 7. Validation Framework

The validator checks BSON documents against their schema before writing:

```go
type ValidationError struct {
    path     string          // e.g. "microflows$Microflow.ObjectCollection.Objects[3].Action"
    Code     string          // e.g. "MISSING_REQUIRED", "TYPE_MISMATCH", "UNKNOWN_FIELD"
    message  string
    Severity Severity        // error, warning, info
}

type Validator struct {
    registry *Registry
    widgets  *WidgetRegistry
}

func (v *Validator) ValidateDocument(doc bson.D) []ValidationError {
    typeName := getType(doc)
    schema := v.registry.LookupByStorage(typeName)
    if schema == nil {
        return []ValidationError{{Code: "UNKNOWN_TYPE", message: typeName}}
    }

    var errors []ValidationError

    // check required fields
    for _, prop := range schema.Properties {
        if prop.Required && !hasField(doc, prop.StorageName) {
            errors = append(errors, ValidationError{
                path: prop.StorageName,
                Code: "MISSING_REQUIRED",
            })
        }
    }

    // check widget schemas for pages
    if isPageType(typeName) {
        errors = append(errors, v.validateWidgets(doc)...)
    }

    // check for fields not in schema (potential corruption)
    for _, elem := range doc {
        if elem.Key[0] == '$' { continue } // skip $ID, $type
        if !schema.HasProperty(elem.Key) {
            errors = append(errors, ValidationError{
                path: elem.Key,
                Code: "UNKNOWN_FIELD",
                Severity: warning,
            })
        }
    }

    return errors
}
```

## Implementation Plan

### Phase 1: Registry Core (foundation)

**Goal**: Load reflection data at runtime, resolve storage names, provide type lookup.

1. Create `schema/` package with `Registry`, `TypeSchema`, `PropertySchema` types
2. Implement `LoadRegistry(version)` from existing embedded reflection JSON
3. Implement storage name ↔ qualified name resolution
4. Add `registry` field to `mpr.Reader` and `mpr.Writer`, loaded from project version
5. Write tests comparing registry lookups against current hardcoded values

**Deliverable**: `registry.Lookup("DomainModels$entity")` returns full property metadata including storage names, defaults, reference kinds, and list encodings.

### Phase 2: Write-Side Completion

**Goal**: Ensure BSON output includes all required fields with correct defaults.

1. Implement `SchemaWriter.Complete()` — fills in missing defaults
2. Wire into existing writer pipeline: hand-coded writer → Complete() → bson.Marshal
3. Implement `SchemaWriter.Validate()` — pre-write validation
4. Add validation as opt-in check in write path (log warnings, don't block)
5. Test against known-good BSON from Studio Pro projects

**Deliverable**: Writing a microflow for Mendix 11.x automatically includes properties that were added since 10.x.

### Phase 3: Widget Schema Resolution

**Goal**: Load widget schemas from the project's widget packages.

1. Implement `.mpk` (ZIP) extraction of `CustomWidgetType` definitions
2. Build `WidgetRegistry` with three-tier resolution (project → embedded → passthrough)
3. Wire into page writer: widget serialization uses project-specific schema
4. Validate widget properties against schema before writing

**Deliverable**: Writing a ComboBox widget uses the exact property schema from the project's widget version, not a hardcoded template.

### Phase 4: Generic Parser

**Goal**: Read any BSON document type using schema metadata.

1. Implement `GenericDocument` type for semi-structured representation
2. Implement `Registry.GenericParse()` using schema-driven field interpretation
3. Add `GenericDocument` support to reader for unimplemented document types
4. Enable MDL `describe` for any document type via generic parsing

**Deliverable**: `describe workflow Module.MyWorkflow` works even though workflows don't have hand-coded parsers for every field.

### Phase 5: Version Migration

**Goal**: Compute and apply property diffs between versions.

1. Implement `ComputeMigration()` from two registries
2. Implement `SchemaWriter.Migrate()` for document-level migration
3. Generate migration report: what changes when upgrading project from X to Y
4. Integrate with write path for cross-version scenarios

**Deliverable**: A document read from a Mendix 10.0 project can be correctly written to a Mendix 11.6 project with all new properties defaulted.

### Phase 6: Extension Documents

**Goal**: Round-trip custom extension documents safely.

1. Implement `CustomBlobDocument` reader (parse outer structure, preserve Contents)
2. Implement `CustomBlobDocument` writer (reconstruct outer structure, write Contents)
3. Add MDL commands: `show EXTENSIONS`, `describe EXTENSION MODULE.Name`
4. Add extension documents to catalog tables

**Deliverable**: Projects with Studio Pro extensions can be read and written without corrupting extension documents.

## What This Does NOT Replace

The hand-coded parsers and writers remain essential for:

- **Complex serialization logic**: Expression strings, microflow flow connections, widget datasource binding
- **Type-specific business rules**: Microflow parameter mapping, page layout constraints
- **MDL execution**: The executor still needs to understand types semantically, not just structurally

The schema registry is a **complement**, not a replacement. It handles the mechanical aspects (field completeness, storage names, defaults, encoding) while hand-coded logic handles the semantic aspects (what does this microflow action actually do).

## Reflection Data Version Count

The `reference/mendixmodellib/reflection-data/` directory contains two JSON files per Mendix minor release — `{version}-structures.json` and `{version}-storageNames.json`. The metamodel schema changes at the **minor release** level (e.g. 10.5 → 10.6), not at patch level (10.5.0 → 10.5.1), so there is one schema per minor. The approximate breakdown:

| Major | Minor versions | Range |
|-------|---------------|-------|
| 6.x   | ~14           | 6.0 – 6.10+ |
| 7.x   | ~24           | 7.0 – 7.23 |
| 8.x   | ~19           | 8.0 – 8.18 |
| 9.x   | ~25           | 9.0 – 9.24 |
| 10.x  | ~22           | 10.0 – 10.21+ |
| 11.x  | ~7            | 11.0 – 11.6 |
| **Total** | **~111** | |

**Not all of these matter equally.** In practice we can dramatically reduce the set we embed:

- **Mendix 6, 7, 8**: End-of-life. Customers still on these versions would be on the **last minor** of each major (6.10, 7.23, 8.18) since that's the only one with security fixes. We could embed just **3 versions** instead of ~57.
- **Mendix 9**: Also end-of-support. The last minor (9.24) is the only practically relevant one. **1 version**.
- **Mendix 10**: Customers typically choose **MTS** (medium-term support: 10.6, 10.12, 10.18) or **LTS** (long-term support: 10.24) releases. We could embed just these **4 versions** plus 10.0 as a baseline, instead of all ~22.
- **Mendix 11**: Active development. All minors are relevant since customers are actively adopting. **~7 versions** (growing).

This reduces the embedded set from ~111 to roughly **~15 versions** while covering the vast majority of real-world projects. A `mxcli update-schemas` command or on-demand download could provide the remaining versions for edge cases.

## Embedding Strategy

At runtime, the registry loads schema data for the target version detected from the project's MPR metadata. Only one version's schema is active at a time. Memory footprint for one version's registry: ~2-5 MB (comparable to current embedded widget templates).

With the reduced version set (~15 versions), the embedded reflection data would add roughly **~7-10 MB** to the binary instead of ~50 MB for all 111.

For widget schemas, we'd extract from `.mpk` files on project open and cache in the `.mxcli/` directory alongside the catalog database.

```
.mxcli/
├── catalog.db           # existing catalog cache
├── widget-schemas/      # extracted widget schemas (new)
│   ├── com.mendix.widget.web.combobox.Combobox@3.5.0.json
│   └── com.mendix.widget.web.datagrid.DataGrid@2.22.0.json
└── schema-cache/        # parsed registry (optional, for startup speed)
    └── 11.6.0.gob
```

## Risk Assessment

| Risk | Mitigation |
|------|-----------|
| Reflection data doesn't capture all BSON rules | Use as safety net, not sole source of truth. Hand-coded writers remain for known-tricky types. Validate against `mx check`. |
| Performance overhead of schema lookup per field | Cache schema lookups. Only validate in write path, not read path. Lazy-load registries. |
| Widget .mpk format changes across versions | Fall back to embedded templates, then to raw passthrough. Never fail — degrade gracefully. |
| Extension document formats are completely opaque | Round-trip as raw bytes. Don't try to interpret Contents field. |
| Field ordering matters in BSON | Use `bson.D` (ordered) throughout. Schema stores properties in declaration order from reflection data. |

## Open Questions

1. **Which versions to embed?** — As discussed in the Reflection Data Version Count section, we can reduce from ~111 to ~15 by targeting LTS/MTS releases and last-minor-of-EOL-majors. The remaining versions could be downloaded on demand via `mxcli update-schemas`.

2. **Should the generic parser replace hand-coded parsers over time?** — The generic parser could handle 80% of types. The question is whether we want to maintain two code paths or gradually migrate.

3. **How do we handle Mendix versions newer than our embedded data?** — Use the newest available schema and log warnings for unknown properties. Or provide a `mxcli update-schemas` command to download newer reflection data.

4. **Should widget schema extraction happen at project open or on demand?** — On demand is simpler but slower on first use. At open time is more predictable but adds startup cost.

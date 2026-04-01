# Multi-Version Support: Consolidated Architecture & Status

## Problem

Mendix projects vary along three versioning axes, and mxcli must handle all of them correctly:

1. **Platform version** (9.x / 10.x / 11.x) — BSON field names, default values, available properties, and reference kinds change across Mendix releases. The reflection data shows ~42% type growth from 9.0 to 11.6.
2. **Widget version** — Pluggable widgets (ComboBox, DataGrid2, Gallery) define their own BSON schemas via `CustomWidgetType` PropertyTypes. Each project bundles specific widget `.mpk` versions in `widgets/`. A mismatch between the embedded template and the installed widget causes CE0463 or `KeyNotFoundException` crashes.
3. **Extension documents** (Mendix 11+) — Studio Pro extensions define custom `CustomBlobDocument` types. These must be round-tripped safely.

Today's codebase hardcodes BSON structure in hand-written parsers/writers targeting ~Mendix 11.6. This works for a single version but silently produces incorrect BSON for others. Widget templates are static and break when the project has different widget versions.

## End-State Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                        Schema Registry                           │
│                        (sdk/schema/)                              │
│                                                                   │
│  ┌─────────────────┐  ┌──────────────────┐  ┌────────────────┐   │
│  │ Platform Schemas │  │ Widget Schemas    │  │ Extension      │   │
│  │ (reflection data │  │ (from .mpk files  │  │ Schemas        │   │
│  │  per version,    │  │  in project, with │  │ (from ext      │   │
│  │  ~15 embedded)   │  │  template         │  │  manifests,    │   │
│  │                  │  │  fallback)        │  │  round-trip)   │   │
│  └────────┬─────────┘  └────────┬─────────┘  └───────┬────────┘   │
│           └──────────────┬──────┘                     │            │
│                          ▼                            │            │
│               ┌──────────────────┐                    │            │
│               │  Unified Schema  │◄───────────────────┘            │
│               │  Lookup          │                                  │
│               └────────┬─────────┘                                  │
│                        │                                            │
│          ┌─────────────┼─────────────┐                              │
│          ▼             ▼             ▼                              │
│   ┌────────────┐ ┌──────────┐ ┌───────────┐                        │
│   │ Completer  │ │ Generic  │ │ Validator │                         │
│   │ (fill      │ │ Parser   │ │ (pre-write│                         │
│   │  defaults) │ │ (read    │ │  checks)  │                         │
│   │            │ │  any     │ │           │                         │
│   └────────────┘ │  type)   │ └───────────┘                         │
│                  └──────────┘                                       │
└──────────────────────────────────────────────────────────────────┘
                            │
                            ▼
              Existing hand-coded parsers/writers
              (complex logic: expressions, flow
               connections, widget datasources)
```

The schema registry **complements** hand-coded parsers/writers — it handles the mechanical aspects (field completeness, storage names, defaults, encoding) while hand-coded logic handles semantic aspects (microflow actions, expression serialization, page layout).

## Implementation Phases

### Phase W: Widget Template Augmentation from .mpk

**Status: DONE**

Reads the `.mpk` widget package from the project's `widgets/` folder at runtime and augments the static JSON template — adding missing properties and removing stale ones — before BSON conversion.

| Component | File | Status |
|-----------|------|--------|
| MPK parser | `sdk/widgets/mpk/mpk.go` | Done |
| MPK parser tests | `sdk/widgets/mpk/mpk_test.go` | Done |
| Template augmentation | `sdk/widgets/augment.go` | Done |
| Augmentation tests | `sdk/widgets/augment_test.go` | Done |
| Placeholder leak detection | `sdk/widgets/placeholder_test.go` | Done |
| Loader integration | `sdk/widgets/loader.go` (`augmentFromMPK`) | Done |
| Call site updates | `cmd_pages_builder_v3_pluggable.go` etc. | Done |

**Pipeline**: Static template -> deep-clone -> augment from .mpk -> collectIDs (remap all placeholders to real UUIDs) -> BSON conversion -> leak check.

**Graceful degradation**: If `.mpk` not found or XML parsing fails, falls back to static template (current behavior). Never makes things worse.

**Bugs addressed**: Bug 1 (DataGrid2 crashes), Bug 2 (ComboBox crashes) from the Bike Returns project.

### Phase 1: Schema Registry Core

**Status: NOT STARTED**

Load reflection data at runtime, resolve storage names, provide type lookup.

| Component | File | Status |
|-----------|------|--------|
| Registry types | `sdk/schema/registry.go` | Not started |
| Registry tests | `sdk/schema/registry_test.go` | Not started |
| Embedded reflection data | `sdk/schema/reflection-data/` | Not started |
| Type/Property definitions | `sdk/schema/types.go` (move from `internal/codegen/schema/`) | Not started |
| Storage name index | Part of registry | Not started |
| Reader integration | `sdk/mpr/reader_documents.go` (`SchemaRegistry()`) | Not started |

**Deliverable**: `registry.LookupByStorage("Forms$LayoutGrid")` returns full property metadata.

**Embedding strategy**: ~15 key versions (LTS/MTS + last-minor-of-EOL), not all ~111. ~18 MB total.

| Major | Versions to embed |
|-------|-------------------|
| 6.x | 6.10 (last minor) |
| 7.x | 7.23 (last minor) |
| 8.x | 8.18 (last minor) |
| 9.x | 9.24 (last minor) |
| 10.x | 10.0, 10.6 (MTS), 10.12 (MTS), 10.18 (MTS), 10.24 (LTS) |
| 11.x | 11.0 through 11.6 (all, active development) |

### Phase 2: Write-Side Completion

**Status: NOT STARTED**

Ensure BSON output includes all required fields with correct defaults for the target version.

| Component | File | Status |
|-----------|------|--------|
| `SchemaWriter.Complete()` | `sdk/schema/writer.go` | Not started |
| `SchemaWriter.Validate()` | `sdk/schema/writer.go` | Not started |
| Writer pipeline integration | `sdk/mpr/writer.go` | Not started |

**Deliverable**: Writing a microflow for Mendix 11.x automatically includes properties added since 10.x.

### Phase 3: Built-in Widget Properties

**Status: NOT STARTED**

Expose and modify scalar properties on built-in widgets (LayoutGrid, DataView, TextBox) via the schema registry.

| Component | File | Status |
|-----------|------|--------|
| `ScalarProperties()` query | Part of Phase 1 registry | Not started |
| DESCRIBE PAGE scalar props | `cmd_pages_describe_parse.go` | Not started |
| Raw BSON widget walker | `widget_property_raw.go` | Not started |
| UPDATE WIDGETS raw path | `cmd_widgets.go` | Not started |
| `UpdateRawUnit` writer | `sdk/mpr/writer_page.go` | Not started |

**Deliverable**: `UPDATE WIDGETS SET 'Width' = 'FullWidth' WHERE WidgetType LIKE '%LayoutGrid%'` works for built-in widgets.

### Phase 4: Generic Parser

**Status: NOT STARTED**

Read any BSON document type using schema metadata, enabling DESCRIBE/SHOW for unimplemented types (workflows, REST services, etc.).

| Component | File | Status |
|-----------|------|--------|
| `GenericDocument` type | `sdk/schema/generic.go` | Not started |
| `Registry.GenericParse()` | `sdk/schema/generic.go` | Not started |
| Reader integration | `sdk/mpr/reader.go` | Not started |
| MDL DESCRIBE integration | `mdl/executor/` | Not started |

**Deliverable**: `DESCRIBE WORKFLOW Module.MyWorkflow` works without hand-coded workflow parsers.

### Phase 5: Version Migration

**Status: NOT STARTED**

Compute property diffs between Mendix versions and apply migrations when writing cross-version.

| Component | File | Status |
|-----------|------|--------|
| `ComputeMigration()` | `sdk/schema/migration.go` | Not started |
| `SchemaWriter.Migrate()` | `sdk/schema/migration.go` | Not started |
| Migration report | CLI command | Not started |

**Deliverable**: Documents from Mendix 10.0 projects are correctly written to Mendix 11.6 with new properties defaulted.

### Phase 6: Extension Documents

**Status: NOT STARTED**

Round-trip custom extension documents safely.

| Component | File | Status |
|-----------|------|--------|
| `CustomBlobDocument` reader | `sdk/mpr/parser_extensions.go` | Not started |
| `CustomBlobDocument` writer | `sdk/mpr/writer_extensions.go` | Not started |
| MDL commands | `mdl/executor/cmd_extensions.go` | Not started |

**Deliverable**: Projects with Studio Pro extensions can be read/written without corruption.

## Phase Dependencies

```
Phase W (Widget Augmentation) ─── DONE
    │
    │   Reuses mpk.ParseMPK() as Tier 1 widget loader
    │
Phase 1 (Registry Core) ─── foundation for everything below
    │
    ├── Phase 2 (Write-Side Completion)
    │       │
    │       └── Phase 5 (Version Migration) ─── requires two registries
    │
    ├── Phase 3 (Built-in Widget Properties) ─── uses ScalarProperties()
    │
    ├── Phase 4 (Generic Parser) ─── uses full type metadata
    │
    └── Phase 6 (Extension Documents) ─── uses round-trip infrastructure
```

Phase W is independent and complete. Phases 2-6 all require Phase 1. Phases 2-4 are independent of each other. Phase 5 requires Phase 2. Phase 6 is independent of 2-5.

## Recommended Implementation Order

1. **Phase 1** (Registry Core) — unlocks all subsequent phases
2. **Phase 3** (Built-in Widget Properties) — high user value, validates registry design
3. **Phase 2** (Write-Side Completion) — correctness safety net
4. **Phase 4** (Generic Parser) — expands coverage to unimplemented domains
5. **Phase 5** (Version Migration) — cross-version support
6. **Phase 6** (Extension Documents) — Mendix 11+ feature

## Source Proposals

This document consolidates the following proposals (preserved for detailed design reference):

| Proposal | Covers |
|----------|--------|
| [`version-aware-mdl.md`](version-aware-mdl.md) | High-level version strategy, architecture options, MDL syntax |
| [`BSON_SCHEMA_REGISTRY_PROPOSAL.md`](BSON_SCHEMA_REGISTRY_PROPOSAL.md) | Detailed registry design, 6-phase plan, embedding strategy |
| [`PROPOSAL_mpk_widget_augmentation.md`](PROPOSAL_mpk_widget_augmentation.md) | Phase W design — .mpk parsing and template augmentation |
| [`PROPOSAL_update_builtin_widget_properties.md`](PROPOSAL_update_builtin_widget_properties.md) | Phase 3 design — built-in widget scalar properties |

## What This Does NOT Replace

Hand-coded parsers/writers remain essential for:
- **Complex serialization**: Expression strings, microflow flow connections, widget datasource binding
- **Type-specific business rules**: Microflow parameter mapping, page layout constraints
- **MDL execution**: The executor needs semantic understanding, not just structural

The schema registry is a **complement** that handles mechanical aspects (field completeness, storage names, defaults, encoding) while hand-coded logic handles semantic aspects.

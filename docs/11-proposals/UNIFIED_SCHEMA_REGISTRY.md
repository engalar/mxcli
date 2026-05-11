# Proposal: Unified Schema Registry

**Status**: Proposed
**Supersedes**: [`BSON_SCHEMA_REGISTRY_PROPOSAL.md`](BSON_SCHEMA_REGISTRY_PROPOSAL.md), [`PROPOSAL_schema_extract.md`](PROPOSAL_schema_extract.md)

## Summary

Mendix project metadata (BSON document schemas, widget property structures, MDL keyword
dispatch) is currently spread across three uncoordinated systems:

- TypeScript-derived reflection data + a hand-maintained `supplements.json`
- Embedded widget JSON templates (one frozen Mendix version) + MPK augmentation at runtime
- A separate `mxcli widget init`-driven `.def.json` file system for pluggable widgets

This proposal replaces all three with a single **Schema Registry** sourced from authoritative
artifacts that ship with each Mendix release: `mx dump-mpr` for platform schemas, `.mpk` XML
for widget schemas. The registry drives serialization, validation, inspection, dispatch,
LSP support, skill generation, and migration tooling — one source of truth for everything that
asks "what does Mendix expect here?"

## Problem

Three separate systems, none self-updating, overlapping responsibilities, divergent staleness:

| System | Source | Update mechanism | Failure mode |
|---|---|---|---|
| `reference/mendixmodellib/reflection-data/` | TS SDK extraction | Manual per release | Stale at 11.6, missing storage names / list encodings / ref kinds |
| `supplements.json` | Hand-curated overrides | Per-release human review | Gaps discovered at runtime when Studio Pro rejects output |
| `sdk/widgets/templates/mendix-11.6/` + augmentation | Manual extraction | One-off per Mendix version | Frozen at 11.6; structural BSON shifts not handled |
| `.mxcli/widgets/*.def.json` (per-project) | `mxcli widget init` parses MPK XML | User-triggered | Lossy: ignores object-list properties (Accordion `groups`, etc.); no integration with init/refresh |
| `sdk/mpr/system_module.go` | Hand-maintained from 11.6.4 | Manual | Only one version; drifts as System module evolves |

For users and AI generating MDL, this surfaces as confusion:

- **Widget choice**: When does `DATAGRID` mean native vs pluggable? Today it depends on
  hardcoded `case "datagrid"` in the executor, regardless of project version.
- **Property bindings**: Custom widgets have complex rules (which properties go together,
  what data types are allowed, what reference kinds work). Nothing today validates these
  against the actually-installed widget version.
- **Version drift**: A script that works on 10.24 silently produces wrong BSON on 11.9 (or
  vice versa). Users discover this when Studio Pro rejects the project.
- **Object lists**: Pluggable widgets with `IsList: true` properties (Accordion `groups`,
  PopupMenu `basicItems`, AreaChart `series`, Maps `markers`) are non-functional via the
  pluggable path because the current `.def.json` format doesn't represent them.

The two earlier proposals each addressed part of this:

- [`BSON_SCHEMA_REGISTRY_PROPOSAL.md`](BSON_SCHEMA_REGISTRY_PROPOSAL.md) — proposed the
  registry concept, but kept TS reflection data + supplements.json as the source. The
  source is the brittle part; the registry shape is sound.
- [`PROPOSAL_schema_extract.md`](PROPOSAL_schema_extract.md) — proposed empirical
  extraction via `mx dump-mpr`, MCP-driven mxunit decoding, and `.mpk` XML parsing. This
  is the right source. What's missing is how the extracted data is consumed end-to-end.

This proposal merges them: registry shape from the first, source of truth from the second,
plus end-to-end workflow integration that neither addressed.

## Architecture

### Two halves with distinct lifecycles

The registry shares a unified API for lookups but the two halves have different
implementations: platform schemas are **build-time codegen** from `mx dump-mpr` output;
widget schemas are **runtime extraction** from per-project `.mpk` files. The dispatch
table is a small static policy file we own.

```
   Compile-time                          Runtime (per project)
   ─────────────                         ──────────────────────

   ┌──────────────────┐                  ┌────────────────────┐
   │ mx dump-mpr      │                  │ Widget schemas     │
   │ (per Mendix      │                  │ (per .mpk in       │
   │  version)        │                  │  project widgets/) │
   │                  │                  │                    │
   │ Run via          │                  │ ZIP + XML parse    │
   │ mxcli schema     │                  │ via mxcli widget   │
   │   extract        │                  │   init             │
   │   platform       │                  │                    │
   │                  │                  │ Cached at:         │
   │ → JSON file      │                  │ .mxcli/widgets/    │
   └────────┬─────────┘                  │   *.def.json       │
            │                            └─────────┬──────────┘
            ▼                                      │
   ┌──────────────────┐                            │
   │ cmd/codegen/     │                            │
   │ Generates Go     │                            │
   │ structs with     │                            │
   │ BSON metadata    │                            │
   └────────┬─────────┘                            │
            │                                      │
            ▼                                      │
   ┌──────────────────┐    ┌────────────────────┐  │
   │ Embedded Go      │    │ Keyword dispatch   │  │
   │ types            │    │ (static policy     │  │
   │ (generated/      │    │  data file)        │  │
   │  metamodel/)     │    │                    │  │
   └────────┬─────────┘    └─────────┬──────────┘  │
            │                        │             │
            └────────────┬───────────┴─────────────┘
                         ▼
              ┌──────────────────────┐
              │ Schema Registry API  │
              │ (Lookup, Describe,   │
              │  Diff, Validate)     │
              └──────────┬───────────┘
                         │
        ┌────────────────┼─────────────────────┐
        ▼                ▼                     ▼
  ┌────────────┐  ┌────────────┐    ┌──────────────────┐
  │ Serializer │  │ Inspector  │    │ Migration        │
  │ Validator  │  │ (CLI /     │    │ planner          │
  │ Dispatcher │  │  LSP /     │    │ (check --post-   │
  │            │  │  catalog / │    │  migration,      │
  │            │  │  skills)   │    │  widget upgrade) │
  └────────────┘  └────────────┘    └──────────────────┘
```

The unified API hides the implementation split — callers don't need to know whether a
type came from generated code or runtime JSON. But the implementations are independent
enough to ship separately, which is what the phasing reflects.

### Why these sources

| Source | Why authoritative |
|---|---|
| `mx dump-mpr` (platform schemas) | Output of the Mendix `mx` binary itself; the same data Studio Pro reads. Ships with every Mendix version via `mxcli setup mxbuild`. Works offline, no Studio Pro needed. **Effectively a Mendix-blessed schema export delivered via CLI** — the only difference from a published schema package is the invocation step. |
| `.mpk` XML (widget schemas) | The canonical source widget authors edit. Studio Pro itself reads this XML at install time. Richer than the BSON `CustomWidgetType` blob (preserves `<description>` text, category structure, conditional visibility rules). |
| Studio Pro MCP / `.mxunit` decoding | Used for **verification during development** of the registry itself (confirm field encodings) — not at runtime. Captured in `PROPOSAL_schema_extract.md` Path 1–4. |

What we drop:

- **TypeScript reflection data** (`reference/mendixmodellib/reflection-data/`) — lossy
  intermediate, manually updated. `mx dump-mpr` provides the same data with no gaps.
- **`supplements.json`** — exists only because TS reflection lacks storage names / list
  encodings / ref kinds. `mx dump-mpr` has all of these directly.
- **Embedded widget templates** (`sdk/widgets/templates/mendix-11.6/`) — replaced by
  per-project widget extraction from `.mpk` files. Augmentation logic in `augment.go`
  becomes unnecessary.
- **Hand-maintained `system_module.go`** — replaced by the System section of
  `{version}-platform.json`, generated from `mx dump-mpr --module-names='System'`.

## Native vs pluggable dispatch — data, not code

Today, `cmd_pages_builder_v3.go` has hardcoded `case "datagrid"` that always invokes the
native builder, regardless of project version. The registry replaces this with data-driven
dispatch.

### Dual-stack widgets

Several Mendix widgets have both a legacy native (dojo-based) implementation *and* a newer
pluggable (React-based) replacement. Studio Pro's project upgrade does **not** auto-convert
native widgets to their pluggable replacements — the underlying technology is different
(dojo runtime vs React runtime, distinct BSON storage types, distinct property schemas).
Users converting a project from Mendix 10 to 11 routinely end up with both stacks
coexisting on the same project, sometimes on the same page.

Affected widgets include:

| Legacy native (dojo) | Modern pluggable (React) |
|---|---|
| `Forms$DataGrid` | `com.mendix.widget.web.datagrid.Datagrid` |
| `Forms$ListView` | `com.mendix.widget.custom.gallery.Gallery` |
| `Forms$DropDown`, `Forms$ReferenceSelector` | `com.mendix.widget.web.combobox.Combobox` |

The MDL surface needs to express *both* unambiguously, with a sensible default per Mendix
version:

```json
{
  "keywordMappings": [
    {
      "keyword": "DATAGRID",
      "versions": [
        { "min": "9.0.0", "max": "10.99.99",
          "kind": "native",   "type": "Forms$DataGrid" },
        { "min": "11.0.0",
          "kind": "pluggable", "widgetId": "com.mendix.widget.web.datagrid.Datagrid" }
      ]
    },
    {
      "keyword": "LEGACYDATAGRID",
      "versions": [
        { "min": "9.0.0",
          "kind": "native",   "type": "Forms$DataGrid",
          "deprecatedFrom": "11.0.0" }
      ]
    },
    {
      "keyword": "GALLERY",
      "versions": [
        { "min": "9.18.0",
          "kind": "pluggable", "widgetId": "com.mendix.widget.custom.gallery.Gallery" }
      ]
    },
    {
      "keyword": "LEGACYLISTVIEW",
      "versions": [
        { "min": "9.0.0",
          "kind": "native",   "type": "Forms$ListView",
          "deprecatedFrom": "9.18.0" }
      ]
    }
  ]
}
```

How this resolves the practical cases:

| Scenario | MDL | Result |
|---|---|---|
| Fresh project on 10.24, `create page` | `DATAGRID` | native (only option on 10.x) |
| Fresh project on 11.9, `create page` | `DATAGRID` | pluggable (recommended) |
| Migrated 11.9, creating a new grid | `DATAGRID` | pluggable |
| Migrated 11.9, `ALTER PAGE` editing existing legacy grid | `LEGACYDATAGRID` (DESCRIBE round-trips it) | native, preserved as-is |
| Migrated 11.9, intentionally creating a native grid | `LEGACYDATAGRID` | native, with deprecation warning |

**Round-trip rule for the parser**: a `Forms$DataGrid` in BSON always emits as
`LEGACYDATAGRID` from `DESCRIBE PAGE` on Mendix 11+, and as `DATAGRID` on Mendix ≤ 10.x
(where it's the only option). A pluggable `com.mendix.widget.web.datagrid.Datagrid`
always emits as `DATAGRID`. This means editing a mixed project via MDL is unambiguous:
the keyword tells you which stack each widget is on.

The executor looks up the binding at write time using the project's Mendix version
plus the explicit keyword. Adding a new pluggable replacement is a schema-data change,
not an executor code change.

### Auto-conversion is out of scope

Converting a native widget instance to its pluggable replacement (e.g. `Forms$DataGrid` →
`com.mendix.widget.web.datagrid.Datagrid`) requires widget-author-specific knowledge
of how to map properties between the two stacks. mxcli does not attempt this. Users
convert one widget at a time in Studio Pro by replacing the widget. mxcli's role is
to help them *find* the legacy widgets via `mxcli check --post-migration` (see Migration
model) and to keep the rest of the page editable while they migrate piecemeal.

## Object lists — first-class child blocks

Widgets like Accordion (`groups`), PopupMenu (`basicItems` / `customItems`), AreaChart
(`series`), and Maps (`markers`) have `Type: "Object"` + `IsList: true` properties: lists
of structured objects with their own sub-property trees. These are equivalent in
expressiveness to DataGrid's `columns` but are entirely missing from the current
pluggable engine.

The widget schema captures the full sub-property tree from the `.mpk` XML. The registry
exposes each list as a child block with a singular keyword:

```mdl
pluggablewidget 'com.mendix.widget.web.accordion.Accordion' acc1 {
  group panel1 (HeaderText: 'Section One') {
    dynamictext c1 (content: 'First section content')
  }
  group panel2 (HeaderText: 'Section Two') {
    dynamictext c2 (content: 'Second section content')
  }
}
```

Slot keyword (`group`), property mapping (`groups`), and sub-properties (`headerText`,
`headerContent`, `content`) all come from the widget schema. No hand-coded support per
widget. The same mechanism handles `popupmenu` items, `timeline` series, `maps` markers,
etc.

## Inspection surface

The registry is queryable via three coordinated surfaces. None of them duplicates content
— each consumes the registry's `Describe(typeName) → []PropertyDoc` API.

### CLI: `mxcli schema`

```bash
mxcli schema list                                 # all document types + widgets + keywords
mxcli schema list widgets                         # only widgets installed in project's widgets/
mxcli schema list keywords                        # MDL keyword → version-gated dispatch table

mxcli schema show DATAGRID                        # MDL keyword (resolves per project version)
mxcli schema show entity                          # document type
mxcli schema show 'com.mendix.widget.web.accordion.Accordion'
mxcli schema show DATAGRID --version 10.24        # explicit version
mxcli schema show --since 11.0                    # what's new (registry-computed)

mxcli schema diff --from 10.24.0 --to 11.9.0      # cross-version delta
mxcli schema extract platform --version 10.18.0   # extract via mx dump-mpr (cache)
mxcli schema extract widgets -p app.mpr           # re-extract from project's .mpk files
```

REPL equivalents: `show schema DATAGRID`, etc.

### Catalog tables

```sql
select * from CATALOG.schema_types where domain = 'Microflows';
select * from CATALOG.schema_properties where keyword = 'DATAGRID';
select * from CATALOG.schema_keywords where min_version <= '10.24.0';
```

This makes the registry queryable from MDL itself, consistent with `show structure`,
`select from CATALOG.entities`, etc.

### LSP / hover / completion / skills

The LSP and skills generation read from the same `Describe()` API. Adding a property to a
widget's `.mpk` automatically flows into:

- LSP completion (suggests the new property name)
- LSP hover (shows the `<description>` from the XML)
- `mxcli check` (validates property bindings)
- Generated skills (per-widget `.md` documents the new property)

### Relationship to `mxcli syntax`

Curated `mxcli syntax <topic>` content stays — it's the place for patterns, examples,
gotchas, anti-patterns. The registry is for property/value reference. Each `syntax` topic
links into `schema show` for the full property table; the two are complementary:

| Surface | Content | Update mechanism |
|---|---|---|
| `mxcli syntax DATAGRID` | "Use sortable columns for tables that need user-driven ordering. Avoid these patterns..." | Hand-written, version-stable |
| `mxcli schema show DATAGRID` | "Properties: pageSize (integer, default 20), pagination (enum: buttons/virtualScrolling), ..." | Registry-derived, project-accurate |

## Migration model

Two distinct cases. mxcli's role in both is **detect, plan, validate, selectively
re-save** — never replicate Studio Pro's transformation logic.

### Project Mendix version upgrade (10.24 → 11.9)

Studio Pro and `mx upgrade-project` execute the actual transformation. mxcli observes:

```bash
# Detect — refuse incompatible writes with actionable hint
$ mxcli exec script.mdl -p app.mpr
✗ DATAGRID property `Pagination: virtualScrolling` requires Mendix 11.0+
  Project is on 10.24.0. Either downgrade syntax, or upgrade project:
    mx upgrade-project --target 11.9.0 app.mpr

# Plan — registry-computed diff
$ mxcli schema diff --from 10.24.0 --to 11.9.0
DocumentTypes added (3): Workflows$ParallelSplit, ...
Properties added (47): Microflows$Microflow.AllowedAsAction, ...
Defaults changed (12): Pages$DataView.RefreshTime 0 → -1, ...
KeywordMappings changed (2): DATAGRID native → pluggable, ...

# Validate post-migration — list every widget still on a legacy stack
$ mxcli check --post-migration -p app.mpr
Project on Mendix 11.9 — 14 widgets still on legacy (dojo) stack:
  Pages/Customers     LEGACYDATAGRID dgCustomers   (consider DATAGRID)
  Pages/Orders        LEGACYDROPDOWN ddStatus      (consider COMBOBOX)
  Pages/OrderDetail   LEGACYLISTVIEW lvLines       (consider GALLERY)
  ...
Auto-rewrite is out of scope — convert each widget in Studio Pro by replacing it.
```

Per-property `introducedIn`/`removedIn` and per-keyword version ranges already encode the
data needed; these commands are thin wrappers over `Registry.Diff()`. The post-migration
check uses the keyword dispatch table's `deprecatedFrom` field to identify legacy-stack
widgets remaining in the project — it gives users a visible to-do list rather than
discovering legacy grids page by page.

### Widget version upgrade (DataGrid 2.22 → 2.30)

Two workflows, two roles:

**Workflow A — Studio Pro upgrades the widget.** User installs new widget version via
Marketplace. Studio Pro's "Upgrade Widgets" prompt handles each instance with
widget-author-specific knowledge mxcli doesn't have. mxcli's role: passive observer. After
`refresh catalog`, just re-extract the new schema. No fixing — Studio Pro already did it.

**Workflow B — mxcli upgrades the widget.** User runs `mxcli widget upgrade` (or drops a
new `.mpk` into `widgets/`). Studio Pro hasn't seen it. mxcli now owns the gap.
Classify drift per widget instance, treat tiers differently:

| Drift kind | Example | mxcli action |
|---|---|---|
| **Additive-safe** | new property added with default value | auto-fix: re-save with default filled in |
| **Removed-tolerated** | property removed, BSON has stale field | leave it (Studio Pro typically tolerates extra fields too) |
| **Required-no-default** | new required property without a meaningful default | flag, don't auto-fix — point to Studio Pro |
| **Type/structure change** | enum value removed, property renamed, sub-tree restructured | flag, don't auto-fix — point to Studio Pro |

Re-saving handles the common case (the ~80% additive scenario most widget upgrades fall
into). For the rest, mxcli doesn't replicate widget-author-specific upgrade logic — it
tells the user to open in Studio Pro.

```bash
$ mxcli widget upgrade datagrid --version 2.30 -p app.mpr
Downloaded com.mendix.widget.web.datagrid.Datagrid@2.30.1
Schema drift across 7 pages:
  ✓ 5 pages: additive-safe (auto-fixed)
  ⚠ 2 pages: required property without default — open in Studio Pro:
      MyApp/Orders/dgOrders → property `customLoadingState` (added v2.28, required)
      MyApp/Customers/dgCust → property `customLoadingState`
```

Cache key includes widget version: `~/.mxcli/widget-schemas/{widgetId}@{version}.json`.
Old extractions persist; if the user reverts the `.mpk`, registry falls back to the
previously-cached extraction with no re-extraction needed.

### What we explicitly don't do

- Execute Studio Pro's project upgrade transformations. BSON shifts between versions are
  too version-specific to replicate safely.
- Convert legacy native (dojo) widgets to their pluggable (React) replacements. The two
  stacks have different runtime, BSON storage types, and property schemas; mapping
  properties between them is widget-author-specific and outside our scope. Users
  convert one widget at a time in Studio Pro.
- Pin a project to multiple Mendix versions simultaneously (one project = one version at
  a time).
- Predict changes for Mendix versions newer than the latest schema we've extracted.
  Refuse with `mxcli schema extract platform --version <new>` as the path forward.

## Workflow integration

Every workflow surface consumes the registry. Adding new behavior is wiring, not new
parallel data sources.

| Surface | Today | Unified |
|---|---|---|
| `mxcli init` | static skill copy + vsix install | + extract platform schema for project's version + extract widget schemas from project's `widgets/` |
| `refresh catalog` | re-scan project artifacts | + detect `.mpk` mtime changes, re-extract widget schemas; invalidate keyword mapping cache |
| `mxcli check` | grammar + reference validation | + validate widget property bindings against schema; reject version-incompatible keywords with actionable hint; classify widget BSON drift |
| `mxcli check --post-migration` | n/a | new — flags pages still using pre-migration patterns |
| LSP completion | static keyword list | + property suggestions from widget schema; attribute type filtering from `valueType` |
| LSP hover | hardcoded help | + property `<description>` from widget XML |
| Skills generation | one-time copy at `init` | + regenerate per-widget skills from extracted schema; auto-documents object list child blocks |
| `mxcli syntax` | static topics | unchanged content; topics link to `schema show` for raw reference |
| Executor BSON write | hardcoded builders + per-project def.json | platform schema fills defaults; widget schema validates output; keyword mapping picks native vs pluggable |
| Catalog SQL | existing tables | + `schema_types`, `schema_properties`, `schema_keywords`, `widget_schemas` |

## Phased implementation

The original phasing bundled platform schema work and widget work together. They share
the *concept* of a registry but not the implementation: widget schemas are per-project
runtime data extracted from `.mpk` files; platform schemas are per-Mendix-version data
that can be code-generated at build time. Phasing reflects this — widget improvements
ship first (building on the existing extraction system), platform schema work follows.

Each phase is designed to deliver visible value at its boundary, not at Phase 5.

### Phase 1 — Object list support (extends existing widget format)

**Goal**: Make Accordion `groups`, PopupMenu items, Maps `markers`, AreaChart `series`
expressible in MDL.

- Extend `.mxcli/widgets/*.def.json` format with an additive `objectLists` field —
  existing files stay valid (no field = behaves as today)
- Extend `mxcli widget init` MPK parser to recognize `<property type="object" isList="true">`
  and emit the new `objectLists` block, including sub-property trees
- Extend grammar/visitor/executor for object-list child block syntax (`group panel1 (...) { ... }`)
  via the existing `WidgetEngine`
- Add doctype tests for object-list-bearing widgets

**Deliverable**: Accordion `group`, PopupMenu `item`, Maps `marker`, AreaChart `series`
all work via MDL. Existing widget setups continue to work unchanged.

### Phase 2 — Dual-stack dispatch table

**Goal**: `DATAGRID` does the right thing per Mendix version, with explicit `LEGACYDATAGRID`
keyword for migrated projects with mixed stacks.

- Small policy data file (`mdl/backend/widget/dispatch.json` or similar) with the keyword →
  version-range → binding mapping
- Replace hardcoded `case "datagrid"` in `cmd_pages_builder_v3.go` with dispatch lookup
- Add `LEGACYDATAGRID`, `LEGACYLISTVIEW`, `LEGACYDROPDOWN` keywords
- DESCRIBE round-trips each widget to the keyword matching its actual BSON stack
- This is editorial policy data we own; not blocked on platform schema work

**Deliverable**: Same MDL script produces native `Forms$DataGrid` on Mendix 10.24 and
pluggable `Datagrid` on 11.9. Mixed projects round-trip cleanly through DESCRIBE.

### Phase 3 — Workflow integration (widget side)

**Goal**: Threaded init/refresh/check/lsp/skills around the existing widget engine.

- `mxcli init` runs `mxcli widget init` automatically for the project's `widgets/` folder
- `refresh catalog` detects `.mpk` mtime changes and re-runs widget extraction
- `mxcli check` validates widget property bindings against extracted schemas; flags
  legacy-stack widgets with `--post-migration`
- LSP wires completion/hover through the widget engine (property suggestions, descriptions)
- Skills regeneration reads from the same widget definitions
- `mxcli syntax` topics gain "see also" links for per-widget property reference

**Deliverable**: Adding a `.mpk` to a project automatically flows into completion, hover,
validation, and skills. Migrated projects get a visible to-do list of legacy widgets.

### Phase 4 — Platform schema codegen

**Goal**: Replace TS reflection data with `mx dump-mpr` output, via build-time codegen.

The existing `cmd/codegen/main.go` already generates Go types from a JSON-shaped reflection
input. This phase swaps its data source from `reference/mendixmodellib/reflection-data/`
to `mx dump-mpr` output, and extends the generator to emit storage names, list encodings,
and ref kinds (the fields TS reflection drops).

- Add `mxcli schema extract platform [--version X.Y.Z]` wrapping `mx dump-mpr`
- Output `{version}-platform.json` per Mendix version (intermediate format)
- Extend `cmd/codegen/main.go` to consume this format and emit Go structs with full BSON
  metadata (storage names, list encodings, ref kinds, defaults)
- Build pipeline regenerates Go code per embedded version (~15 LTS/MTS releases)
- Generated code replaces hand-maintained `system_module.go`

**Deliverable**: Compile-time-typed platform schema lookups. No runtime registry to load
or invalidate. `mxcli schema show entity` reads from generated Go types.

### Phase 5 — Drop `supplements.json` and finish migration tooling

**Goal**: Retire legacy data sources; ship cross-version inspection and migration commands.

- Validate Phase 4 codegen has full coverage of properties currently in `supplements.json`
- Drop `supplements.json` and `reference/mendixmodellib/reflection-data/`
- Implement `mxcli schema list/show/diff` reading from generated platform schemas + runtime
  widget schemas
- Implement `mxcli check --post-migration`, `mxcli widget upgrade` with tier-based drift
  classification
- Add catalog tables: `schema_types`, `schema_properties`, `schema_keywords`

**Deliverable**: Zero hand-maintained schema data files in the codebase. Cross-version
diff/migration commands work end-to-end. A user upgrading a project from 10.24 to 11.9
or upgrading a widget gets clear diagnostics and actionable hints.

## Trade-offs and decisions

| Question | Decision | Rationale |
|---|---|---|
| Source of platform schemas | `mx dump-mpr` against blank project | Mendix-blessed CLI export; same data Studio Pro consumes; works offline; no TS dependency. Equivalent in authority to a published schema package. |
| Platform schema delivery | Build-time codegen via `cmd/codegen/main.go` | Reuses existing generator pipeline; compile-time-typed lookups; no runtime registry to load. Format changes become build failures, not runtime errors. |
| Source of widget schemas | `.mpk` XML directly | Canonical source widget authors edit; richer than BSON `CustomWidgetType` |
| Studio Pro MCP role | Dev-time verification only | Required for confirming field encodings during registry development; not a runtime dependency |
| Embed vs download platform schemas | Embed ~15 LTS/MTS versions; download remainder | Diff/migration commands need local schemas; ~7-10 MB acceptable; downloadable for edge cases |
| `supplements.json` fate | Drop entirely | `mx dump-mpr` removes the gap that motivated it |
| Existing `.mxcli/widgets/*.def.json` | Auto-migrate on first `refresh catalog` after upgrade | Zero user action needed; old files removed once new schemas extracted |
| Project version migration execution | Studio Pro / `mx upgrade-project` only | Too brittle to replicate widget-author-specific transformations |
| Widget version upgrade execution | Tier-based: auto-fix additive, flag everything else | Captures the ~80% case (additive defaults) without replicating Studio Pro's full upgrade logic |
| Object list keyword convention | Singular form of property name (`group` for `groups`, `item` for `basicItems`) | Reads naturally; mechanical to derive |

## Open questions

1. **Object list keyword collisions**: Two widgets could have object lists with the same
   suggested singular keyword (e.g. both PopupMenu and a hypothetical other widget have
   `items`). The keyword is scoped to the parent widget, so technically no collision —
   but it could confuse users. Document the per-widget keyword in `schema show` output.

2. **Widget XML schema version drift**: The `.mpk` `<widget>` XML schema has evolved
   slightly across Mendix versions (added attributes, new property types). Extraction
   needs to handle older XML schemas gracefully. Mitigation: parse permissively, fall
   back to "raw passthrough" for unknown XML elements.

3. **`mx dump-mpr` output format stability**: The format is subject to the same
   backward-compatibility considerations as any Mendix-released artifact — no different
   from depending on a hypothetical Mendix-published schema package. Codegen runs at
   build time, so format changes surface as build failures (loud, fixable) rather than
   runtime errors. Mitigation: validate output structure during codegen; pin a
   minimum-supported `mx` version per registry data file.

4. **Cross-project widget schema cache**: `~/.mxcli/widget-schemas/` is shared across
   projects. If two projects use different versions of the same widget, both extractions
   coexist (versioned cache key). What if two projects use the same version but the user
   has different `.mpk` files (e.g. local dev modifications)? Resolution: cache key
   includes content hash for non-published widgets. Published widgets keyed by version
   alone.

5. **Performance**: Platform schemas are codegen output, so lookup is struct field access
   (free). Widget schemas are per-project on-disk JSON, loaded once at command start and
   cached for the session. Mitigation: lazy-load widget schemas (only when first needed);
   measure once Phase 3 runs end-to-end before optimizing further.

## Non-goals

- **Replacing Studio Pro's UI for any operation**. The registry powers programmatic
  workflows; visual modeling stays in Studio Pro.
- **Auto-converting legacy native widgets to their pluggable replacements**. Different
  runtime stacks (dojo vs React), different BSON storage, widget-author-specific
  property mappings. We surface legacy widgets via `check --post-migration`; conversion
  is a manual Studio Pro operation.
- **Inferring widget behavior at runtime**. We capture *structural* schema (properties,
  types, defaults). Widget rendering logic, validation rules beyond schema, and runtime
  behavior remain widget-author concerns.
- **Lockstep with every Mendix release**. Embedded versions cover LTS/MTS; users on
  bleeding-edge versions run `mxcli schema extract platform --version <new>` to populate
  their cache.

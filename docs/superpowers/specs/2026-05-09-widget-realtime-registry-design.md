# Widget Real-Time Registry Design

**Date:** 2026-05-09  
**Branch:** feature/mpk-template-derivation  
**Status:** Approved

## Problem

`mxcli widget init` pre-extracts `.def.json` files from `.mpk` widget packages into
`.mxcli/widgets/`. The command skips files that already exist on disk (no mtime or
version check), so upgrading a widget in Mendix Studio Pro silently leaves stale
definitions in place. Users must remember to delete `.def.json` files and re-run
`widget init` after every widget upgrade — a step that is easy to forget and produces
no visible error until BSON serialization produces a wrong page.

## Goal

Eliminate the staleness problem by making widget definition lookup real-time: when
`CREATE PAGE` references a widget that is not in the built-in registry, mxcli derives
the `WidgetDefinition` on-the-fly from the project's `widgets/*.mpk` files. No
pre-extraction step is required for normal operation.

## Non-Goals

- Removing support for hand-crafted `.def.json` overrides (kept as escape hatch).
- Removing the `widget init` / `widget extract` commands (repurposed, not deleted).
- Changing the BSON template derivation path (already handled on this branch).

## Chosen Approach: Lazy MPK Derivation with Optional `.def.json` Override

Lookup order (unchanged for built-ins, new fallback for unknowns):

```
MDL keyword
  └─ Registry built-in (.def.json embedded at compile time)  → hit: use it
  └─ Registry user override (.mxcli/widgets/*.def.json)      → hit: use it
  └─ MPK pre-scan map (mdlName → widgetID)                   → hit: full-parse MPK
       └─ derive WidgetDefinition, cache in-memory            → use it
  └─ not found                                               → error
```

### Phase 1 — Session startup (project path known)

Call `registry.SetProjectDir(projectDir)`, which triggers a **lightweight pre-scan**:

- Glob `<projectDir>/widgets/*.mpk`
- For each MPK: open ZIP, read only `package.xml` (already implemented in
  `mpk.getWidgetIDsFromMPK`)
- For each widgetID found: compute `deriveMDLName(widgetID)` → store in
  `mpkNameMap[mdlName] = widgetID`
- Skip names already in the built-in or user-override registry (built-ins win)
- Cost: ~1–2 ms per widget; a project with 30 widgets adds ~50 ms at most

This map is used by LSP completion to offer all available widget keywords without
requiring `widget init`.

### Phase 2 — First use of a widget (CREATE PAGE execution)

On `registry.Get(mdlName)` or `registry.GetByWidgetID(widgetID)` miss:

1. Look up `widgetID` from `mpkNameMap`
2. Call `mpk.FindMPK(projectDir, widgetID)` → mpkPath
3. Call `mpk.ParseMPKForWidget(mpkPath, widgetID)` → `*mpk.WidgetDefinition`  
   (already cached in `mpk.defCache` after first parse)
4. Convert to `executor.WidgetDefinition` via `deriveFromMPK()` (logic moved from
   `cmd_widget.go:generateDefJSON`)
5. Register in `byMDLName` and `byWidgetID` — subsequent lookups are O(1)

### Phase 3 — Hand-crafted override (optional, for complex widgets)

Users who need custom `Modes`, non-standard `Operation` types, or MDL name overrides
write a `.def.json` by hand (or use `widget extract` as a starting point). These
files in `.mxcli/widgets/` are loaded at startup before the pre-scan, so they win
over MPK derivation.

## Dependency Analysis

No circular dependency is introduced:

```
sdk/widgets/mpk   →  (stdlib only, zero mdl/ imports)
mdl/executor      →  sdk/widgets/definitions  (existing)
                  →  sdk/widgets/mpk          (new, safe)
cmd/mxcli         →  mdl/executor + sdk/widgets/mpk  (unchanged)
```

## Code Changes

### `mdl/executor/widget_registry.go`

Add fields:
```go
projectDir  string
mpkNameMap  map[string]string // mdlName (upper) → widgetID
```

Add methods:
- `SetProjectDir(dir string) error` — stores dir, calls `preScanWidgets`
- `preScanWidgets(dir string) error` — builds `mpkNameMap`
- `deriveFromMPK(widgetID string) (*WidgetDefinition, error)` — converts
  `mpk.WidgetDefinition` to `executor.WidgetDefinition`; this is the
  `generateDefJSON` logic moved here from `cmd_widget.go`

Modify:
- `Get(name string)` — add MPK fallback after registry miss
- `GetByWidgetID(id string)` — add MPK fallback after registry miss

New import:
- `github.com/mendixlabs/mxcli/sdk/widgets/mpk`

### `cmd/mxcli/cmd_widget.go`

- Remove `generateDefJSON` — replaced by `registry.deriveFromMPK`
- `widget init`: remove skip-if-exists guard; change help text to describe it as a
  debugging/customization tool, not a required setup step; add `--force` flag to
  overwrite existing files
- `widget extract`: unchanged

### Executor / REPL initialization

Wherever `executor.NewWidgetRegistry()` is called and a project path is available,
follow up with `registry.SetProjectDir(projectDir)`. Concrete locations:

- `mdl/executor/executor.go` (or its `Context` initializer)
- `cmd/mxcli/repl.go` (REPL session start)
- Any backend init that receives a project path

### `cmd/mxcli/cmd_widget.go` — `widget init` new help text

```
Extract and dump widget definitions for inspection or customization.

mxcli widget init is no longer required for CREATE PAGE to work — definitions
are derived automatically from widgets/*.mpk at runtime. Run this command only
when you need to inspect or hand-edit a widget's property mappings.

Flags:
  --force   overwrite existing .def.json files
```

## Testing

- Unit test: `TestRegistryMPKFallback` — create a temp dir with a minimal MPK,
  call `SetProjectDir`, verify `Get(derivedName)` returns a non-nil definition
- Unit test: `TestPreScanSkipsBuiltins` — verify that a built-in widget name (e.g.
  `GALLERY`) in `mpkNameMap` is ignored in favour of the built-in registry entry
- Unit test: `TestDeriveFromMPKCached` — call `Get` twice for the same widget,
  verify MPK is parsed only once (inspect `mpk.defCache`)
- Integration: existing `mdl-examples/doctype-tests/` page tests that use third-party
  widgets should pass without `widget init` having been run

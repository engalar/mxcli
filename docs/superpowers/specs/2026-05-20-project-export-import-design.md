# Project Export / Import Design

**Date:** 2026-05-20  
**Status:** Approved  
**Feature:** `mxcli export` / `mxcli import` — batch export a Mendix project to structured MDL files and reimport into an empty template.

---

## Goals

- **Migration / cloning** — copy business logic from one project to another
- **Version control / archiving** — MDL files are human-readable and git-diffable
- **AI / LLM context** — structured MDL directory feeds into LLM analysis
- **Bidirectional sync** — export → edit MDL → import as a migration workflow

---

## Scope

| Layer | What is exported |
|-------|-----------------|
| **Project-level** | Settings, user roles, demo users, navigation |
| **Marketplace modules** | Module name + version only (comment file, not executable MDL) |
| **Regular modules** | All documents: entities, associations, enumerations, constants, microflows, nanoflows, pages, layouts, snippets, workflows, security roles, Java/JavaScript actions, image collections |

**Import precondition:** the target project already has marketplace modules pre-installed. The import command handles only regular modules and project-level config.

---

## CLI Syntax

```bash
# Export entire project
mxcli export -p app.mpr --output ./export-dir

# Export one module only
mxcli export -p app.mpr --output ./export-dir --module MyFirstModule

# Import into target project (marketplace modules pre-installed)
mxcli import -p target.mpr --input ./export-dir

# Import one module only
mxcli import -p target.mpr --input ./export-dir --module MyFirstModule

# Preview without writing / modifying
mxcli export -p app.mpr --output ./export-dir --dry-run
mxcli import -p target.mpr --input ./export-dir --dry-run

# Continue past errors (import)
mxcli import -p target.mpr --input ./export-dir --skip-errors
```

---

## Output Directory Structure

```
export-dir/
├── _project/
│   ├── settings.mdl          # ALTER PROJECT SETTINGS
│   ├── security.mdl          # User roles + demo users
│   └── navigation.mdl        # Navigation profiles + home pages
│
├── _marketplace.mdl          # Comment-only: lists marketplace modules + versions
│
└── MyFirstModule/            # One directory per regular module
    ├── _module.mdl           # CREATE MODULE (module-level config)
    ├── _associations.mdl     # All associations in this module
    ├── Domain/               # Mirrors Studio Pro folder hierarchy
    │   ├── MyFirstModule.Customer.mdl
    │   └── MyFirstModule.Order.mdl
    ├── Microflows/
    │   ├── ACT/
    │   │   └── MyFirstModule.ACT_CreateOrder.mdl
    │   └── SUB/
    │       └── MyFirstModule.SUB_Validate.mdl
    └── Pages/
        └── MyFirstModule.Order_Overview.mdl
```

**File naming:** QName of the document (e.g. `MyFirstModule.ACT_CreateOrder.mdl`).  
**Folder structure:** mirrors the Studio Pro project explorer folder hierarchy via `ListFolders()`.  
**Each file** contains one complete, re-executable MDL statement (`CREATE ENTITY`, `create microflow`, etc.).

### `_marketplace.mdl` format (not executable)

```mdl
-- Marketplace modules detected in this project.
-- Reinstall these before running mxcli import.
--
-- Module: Administration          (version: 3.4.0)
-- Module: AtlasCore               (version: 4.0.0)
-- Module: NanoflowCommons         (version: 4.1.0)
```

---

## Internal Architecture

### New files

| File | Role |
|------|------|
| `cmd/mxcli/cmd_export.go` | Cobra command: parse flags, call `executor.ExportProject()` |
| `cmd/mxcli/cmd_import.go` | Cobra command: parse flags, call `executor.ImportProject()` |
| `mdl/executor/cmd_export_project.go` | `ExportProject()` orchestration logic |
| `mdl/executor/cmd_import_project.go` | `ImportProject()` orchestration logic |
| `mdl/executor/cmd_export_project_test.go` | Unit + integration + round-trip tests |

### Export flow

```
ExportProject(outputDir, opts)
  ├── 1. ListModules() → classify: marketplace (fromAppStore=true) vs regular
  ├── 2. Write _marketplace.mdl (comments only)
  ├── 3. Write _project/{settings,security,navigation}.mdl
  │       via existing describeProjectSecurity(), describeNavigation(), etc.
  └── 4. For each regular module:
          ├── ListFolders(moduleID) → build folder tree for path mapping
          ├── Write _module.mdl
          ├── Enumerate each document type:
          │     Entity / Microflow / Nanoflow / Page / Layout / Snippet /
          │     Workflow / Enumeration / Constant / JavaAction /
          │     JavaScriptAction / ImageCollection
          ├── For each document → call describe*Gen() → write to path
          └── Collect associations → write _associations.mdl
```

**Key principle:** `describe*Gen()` functions in the executor package are reused directly — no serialization logic is duplicated. Functions that are currently unexported are exported (capitalised) or wrapped in a thin internal API.

**Marketplace detection:** via the `fromAppStore` field already present in the module BSON — no network request needed.

### Import flow

```
ImportProject(inputDir, opts)
  ├── 1. Scan directory tree → collect all .mdl files
  ├── 2. Sort by dependency topology:
  │     Enumerations → Entities → Associations → Constants →
  │     ModuleRoles → JavaActions → Microflows/Nanoflows →
  │     Layouts → Pages/Snippets → Workflows →
  │     Navigation → ProjectSecurity → Settings
  ├── 3. Execute each file sequentially in a single executor session
  │     (reuses existing execFile() / exec() path)
  └── 4. Collect errors → print summary report
```

---

## Error Handling

| Situation | Default behaviour | Override |
|-----------|------------------|---------|
| Single document describe fails during export | Skip + warn, continue | — |
| Single .mdl file fails during import | **Stop** | `--skip-errors` continues |
| `--dry-run` export | List files that would be written, no disk writes | — |
| `--dry-run` import | Parse all files, report errors, no project changes | — |

### Progress output (stderr)

```
Exporting MyApp → ./export-dir
  [project]  settings.mdl
  [project]  security.mdl
  [project]  navigation.mdl
  [market]   3 marketplace modules → _marketplace.mdl
  [module]   MyFirstModule (42 documents)
    ✓ 12 entities
    ✓ 18 microflows
    ✓  8 pages
    ✓  4 enumerations
Done: 59 documents exported in 1.2s
```

---

## Testing Strategy

| Level | What is tested | Location |
|-------|---------------|---------|
| Unit | Folder tree construction, module classification, topological sort | `cmd_export_project_test.go` |
| Integration | Export `testdata/expr-checker/minimal.mpr` → verify output files exist and parse | `cmd_export_project_test.go` |
| Round-trip | Export → fresh project → import → verify entity/microflow count matches | `cmd_export_project_test.go` |

The round-trip test is the core correctness gate: exported MDL must execute without errors and produce a structurally equivalent project.

---

## Out of Scope

- Automatic marketplace module installation during import (user installs manually)
- Conflict resolution when a module already exists in the target (import fails with a clear error)
- Incremental / delta export (export only what changed since last export)
- Binary assets (custom widgets, JAR files) — not included in MDL export

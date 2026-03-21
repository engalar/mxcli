# BSON Discover Tool Design

## Problem

MDL's DESCRIBE output is a lossy projection of BSON data. When implementing new features (workflows, pages, microflows), there's no systematic way to:

1. **Discover** which BSON fields are missing from DESCRIBE/CREATE coverage
2. **Compare** reference BSON (Studio Pro) vs generated BSON (mxcli) field-by-field
3. **Validate** round-trip fidelity: CREATE → mx check → DESCRIBE → compare original BSON

Today this requires ad-hoc Go scripts, manual JSON diffing, and Python one-liners. The workflow baseline comparison (2026-03-22) revealed that `DESCRIBE WORKFLOW` drops ~50% of semantic fields (UserTargeting, DueDate, CompletionCriteria, TaskName, TaskDescription, OnCreatedEvent, etc.).

## Solution

A unified `mxcli bson` subcommand suite for BSON discovery, comparison, and round-trip validation. Debug-only — not included in release builds.

## Commands

### `mxcli bson discover`

Analyzes field coverage: what BSON fields exist vs what MDL DESCRIBE outputs.

```bash
# Scan all workflows, report field coverage per $Type
mxcli bson discover -p app.mpr --type workflow

# Scan a specific object
mxcli bson discover -p app.mpr --type workflow --object "Module.WfName"

# Other types
mxcli bson discover -p app.mpr --type microflow
mxcli bson discover -p app.mpr --type page
```

**Output:**

```
Workflows$Workflow (2 objects scanned)
  ✓ Name                          covered
  ✓ Parameter                     covered (PARAMETER)
  ✓ Flow                          covered (BEGIN...END)
  ✗ Title                         UNCOVERED (string, ex: "Workflow")
  ✗ AdminPage                     UNCOVERED (null)
  ✗ DueDate                       UNCOVERED (string, ex: "")
  ✗ WorkflowName                  UNCOVERED (*MicroflowsStringTemplate)
  ✗ WorkflowDescription           UNCOVERED (*MicroflowsStringTemplate)
  ✗ OnWorkflowEvent               UNCOVERED (slice, empty)
  ✗ WorkflowV2                    UNCOVERED (bool, ex: false)
  - $ID, PersistentId, Size...    structural (5 fields)

  Coverage: 8/15 semantic fields (53%)

Workflows$SingleUserTaskActivity (5 instances)
  ✓ Name, Caption, TaskPage       covered
  ✓ Outcomes                      covered (OUTCOMES)
  ✗ UserTargeting                 UNCOVERED (*WorkflowsUserTargeting)
  ✗ AutoAssignSingleTargetUser    UNCOVERED (bool, ex: false)
  ✗ DueDate                       UNCOVERED (string, ex: "")
  ✗ TaskName                      UNCOVERED (*MicroflowsStringTemplate)
  ✗ TaskDescription               UNCOVERED (*MicroflowsStringTemplate)
  ✗ OnCreatedEvent                UNCOVERED (*WorkflowsUserTaskEvent)
  ✗ BoundaryEvents                UNCOVERED (slice, empty)

  Coverage: 4/13 semantic fields (31%)
```

**Algorithm:**

1. Read all BSON objects of the given type from MPR
2. For each unique `$Type` encountered, collect all field names and sample values
3. Look up the `$Type` in the reflect-based TypeRegistry → get full field list from generated metamodel structs
4. Run DESCRIBE on the object → capture MDL text output
5. For each semantic field, use heuristic matching to check if its value appears in MDL output:
   - String values: substring search in MDL text
   - Bool/int values: search stringified form
   - Object/null: mark as `unknown` (needs manual confirmation, or defaults to uncovered if null/default value)
6. Fields in BSON but not in reflect metadata → flagged as `unknown-to-schema` (version mismatch)
7. Report coverage per `$Type`

### `mxcli bson compare`

Diff two BSON objects, skipping structural/layout noise.

```bash
# Same MPR, two objects
mxcli bson compare -p app.mpr --type workflow "Module.WfA" "Module.WfB"

# Cross-MPR comparison (reference vs generated)
mxcli bson compare --type workflow -p ref.mpr "Module.Wf" -p2 test.mpr "Module.Wf"

# Show all differences including structural
mxcli bson compare --type workflow -p ref.mpr "Module.Wf" -p2 test.mpr "Module.Wf" --all
```

**Output:**

```
Workflows$Workflow
  = Name: "Workflow"
  ≠ Title: "Workflow" vs ""                          ← value mismatch
  ≠ WorkflowV2: false vs (missing)                   ← field absent

Workflows$SingleUserTaskActivity [userTask1]
  = Name, Caption, TaskPage
  + AutoAssignSingleTargetUser: false                 ← only in left
  + UserTargeting: XPathUserTargeting{...}            ← only in left
  + OnCreatedEvent: NoEvent                           ← only in left

Summary: 12 differences, 8 only-in-left, 0 only-in-right, 4 value-mismatches
```

**Design:**

- Extends existing `dump-bson --compare` logic (currently in `cmd_dump_bson.go`)
- Recursive diff on BSON `map[string]any` trees
- Default skip set: `$ID`, `PersistentId`, `RelativeMiddlePoint`, `Size` (overridable with `--all`)
- Activity matching: by `Name` field within the same `$Type` (not by array index)
- Array diffing: match elements by `$Type` + identifying field (`Name`, `Value`, `Caption`)

### `mxcli bson roundtrip`

Automated CREATE → mx check → DESCRIBE → compare cycle.

```bash
# Full round-trip validation
mxcli bson roundtrip -p app.mpr --type workflow --object "Module.WfName"

# All objects of a type
mxcli bson roundtrip -p app.mpr --type workflow --all

# Skip mx check (fast mode)
mxcli bson roundtrip -p app.mpr --type workflow --object "Module.WfName" --skip-check
```

**Output:**

```
Step 1: DESCRIBE → MDL .......................... OK (45 lines)
Step 2: Copy project ............................ OK (/tmp/mxcli-rt-xxxxx/)
Step 3: DROP + CREATE ........................... OK
Step 4: mx check ................................ OK (0 errors)
Step 5: BSON compare (ref vs generated) ......... 8 differences
  + UserTargeting (5 instances)              lost in round-trip
  + AutoAssignSingleTargetUser (5 instances) lost in round-trip
  + OnCreatedEvent (5 instances)             lost in round-trip
  = Flow structure                           identical
  = Parameter mappings                       identical
Step 6: MDL compare (DESCRIBE before vs after) .. 0 differences

Result: BSON round-trip has 8 field losses, MDL round-trip is lossless
```

**Workflow:**

1. Read original BSON from source MPR (reference)
2. DESCRIBE → capture MDL text
3. Copy project to temp directory
4. Execute DROP + CREATE with generated MDL
5. Run `mx check` on temp project (skip if `--skip-check` or mxbuild not installed)
6. Read new BSON from temp project
7. Run Compare (reference vs new) → report field-level differences
8. DESCRIBE again on temp project → compare MDL text (MDL round-trip check)
9. Clean up temp directory (unless `--keep-temp`)

### `mxcli bson dump`

Existing functionality from `dump-bson`, relocated under the `bson` parent command.

```bash
mxcli bson dump -p app.mpr --type workflow --list
mxcli bson dump -p app.mpr --type workflow --object "Module.WfName"
```

## Architecture

### Build Tag Isolation

All BSON tool code compiles only with `//go:build debug`:

```
cmd/mxcli/
  cmd_bson.go              //go:build debug  — register "bson" parent command
  cmd_dump_bson.go         //go:build debug  — existing dump-bson, moved under "bson dump"

bson/                      //go:build debug  — core logic package
  registry.go              — reflect-based TypeRegistry
  discover.go              — field coverage analysis
  compare.go               — recursive BSON diff
  roundtrip.go             — full round-trip validation
  classify.go              — field category rules
```

**Build commands:**

```makefile
# Release (default) — no bson commands, no size impact
make build

# Debug — includes bson discover/compare/roundtrip/dump
make build-debug
# go build -tags debug -o bin/mxcli-debug ./cmd/mxcli
```

### TypeRegistry (registry.go)

Runtime reflect scanning of `generated/metamodel` structs. No external JSON files needed.

```go
//go:build debug

package bson

import (
    "reflect"
    "strings"
    "github.com/mendixlabs/mxcli/generated/metamodel"
)

// TypeRegistry maps BSON $Type → Go reflect.Type
// Populated at init time.
var TypeRegistry = map[string]reflect.Type{
    "Workflows$Workflow":                reflect.TypeOf(metamodel.WorkflowsWorkflow{}),
    "Workflows$SingleUserTaskActivity":  reflect.TypeOf(metamodel.WorkflowsSingleUserTaskActivity{}),
    "Workflows$MultiUserTaskActivity":   reflect.TypeOf(metamodel.WorkflowsMultiUserTaskActivity{}),
    "Workflows$CallMicroflowTask":       reflect.TypeOf(metamodel.WorkflowsCallMicroflowTask{}),
    "Workflows$CallWorkflowActivity":    reflect.TypeOf(metamodel.WorkflowsCallWorkflowActivity{}),
    "Workflows$ExclusiveSplitActivity":  reflect.TypeOf(metamodel.WorkflowsExclusiveSplitActivity{}),
    "Workflows$ParallelSplitActivity":   reflect.TypeOf(metamodel.WorkflowsParallelSplitActivity{}),
    "Workflows$JumpToActivity":          reflect.TypeOf(metamodel.WorkflowsJumpToActivity{}),
    "Workflows$WaitForTimerActivity":    reflect.TypeOf(metamodel.WorkflowsWaitForTimerActivity{}),
    "Workflows$WaitForNotificationActivity": reflect.TypeOf(metamodel.WorkflowsWaitForNotificationActivity{}),
    // ... extensible to other namespaces (Microflows$, Forms$, DomainModels$)
}

// PropertyMeta describes a single field's metadata derived from reflect.
type PropertyMeta struct {
    GoFieldName string        // "UserTargeting"
    StorageName string        // "userTargeting" (from json tag, maps to BSON field name)
    GoType      string        // "*WorkflowsUserTargeting"
    IsList      bool          // reflect.Kind == Slice
    IsPointer   bool          // reflect.Kind == Ptr
    IsRequired  bool          // json tag lacks "omitempty"
    Category    FieldCategory // structural / layout / semantic
}

type FieldCategory int

const (
    Semantic   FieldCategory = iota // Business fields (Name, Entity, Expression)
    Structural                      // Internal ($ID, $Type, PersistentId)
    Layout                          // Visual only (RelativeMiddlePoint, Size)
)

// GetFieldMeta returns all field metadata for a $Type.
// Returns nil if the type is not in the registry.
func GetFieldMeta(bsonType string) []PropertyMeta {
    rt, ok := TypeRegistry[bsonType]
    if !ok {
        return nil
    }

    var result []PropertyMeta
    for i := 0; i < rt.NumField(); i++ {
        f := rt.Field(i)
        if f.Anonymous { continue } // skip BaseElement

        jsonTag := f.Tag.Get("json")
        storageName, _ := parseJSONTag(jsonTag)
        isRequired := !strings.Contains(jsonTag, "omitempty")

        result = append(result, PropertyMeta{
            GoFieldName: f.Name,
            StorageName: storageName,
            GoType:      f.Type.String(),
            IsList:      f.Type.Kind() == reflect.Slice,
            IsPointer:   f.Type.Kind() == reflect.Ptr,
            IsRequired:  isRequired,
            Category:    classifyField(storageName),
        })
    }
    return result
}
```

### Field Classification (classify.go)

Minimal hardcoded rules. Everything else defaults to Semantic.

```go
var structuralFields = map[string]bool{
    "$ID": true, "$Type": true, "PersistentId": true,
}

var layoutFields = map[string]bool{
    "RelativeMiddlePoint": true, "Size": true,
}

func classifyField(storageName string) FieldCategory {
    if structuralFields[storageName] { return Structural }
    if layoutFields[storageName]     { return Layout }
    return Semantic
}
```

### MDL Coverage Heuristic (discover.go)

No hardcoded field-to-MDL mapping table. Instead, value-based matching:

```go
func checkFieldCoverage(storageName string, bsonValue any, mdlText string) CoverageStatus {
    switch v := bsonValue.(type) {
    case string:
        if v == "" { return DefaultValue }
        if strings.Contains(mdlText, v) { return Covered }
        return Uncovered

    case bool:
        // booleans are tricky — "false" might not appear in MDL
        if !v { return DefaultValue }
        if strings.Contains(mdlText, "true") { return Covered }
        return Uncovered

    case nil:
        return DefaultValue  // null PartProperty

    case map[string]any:
        // Nested object — check if any leaf value appears in MDL
        return checkNestedCoverage(v, mdlText)

    default:
        return Unknown
    }
}
```

Coverage statuses:
- `Covered` — field value found in MDL output
- `Uncovered` — field has non-default value but not in MDL
- `DefaultValue` — field is empty/null/false (may be intentionally omitted from MDL)
- `Unknown` — cannot determine automatically

### Supported Types

The `--type` flag maps to BSON unit types and `ListRawUnits()`:

| --type | BSON prefix | TypeRegistry entries needed |
|--------|-------------|---------------------------|
| workflow | Workflows$Workflow | Workflows$* (all activity types) |
| microflow | Microflows$Microflow | Microflows$* (60+ action types) |
| page | Forms$Page | Forms$* (50+ widget types) |
| entity | DomainModels$Entity | DomainModels$* |
| enumeration | Enumerations$Enumeration | Enumerations$* |

TypeRegistry is extensible — add entries as needed. Types not in the registry still work for compare/dump (raw BSON diff), but discover reports them as `unknown-to-schema`.

## Scope & Priorities

### Phase 1 (Initial)

- `bson dump` — migrate existing `dump-bson` under new parent command
- `bson discover` — field coverage for workflow types
- `bson compare` — single-MPR and cross-MPR diff
- Build tag isolation (`//go:build debug`)
- `make build-debug` target

### Phase 2

- `bson roundtrip` — full automated round-trip validation
- Extend TypeRegistry to microflow types

### Phase 3

- Extend TypeRegistry to page and domain model types
- `--format json` output for CI integration

## Non-Goals

- No changes to release binary
- No changes to generated `types.go` (no `mx` struct tags)
- No embedded reflection JSON files
- No automated fix suggestions (just discovery and reporting)

## Cleanup

- Remove `/mnt/data_sdd/gh/mxcli/cmd/dump-wf/` (temporary dump script created during this design session)

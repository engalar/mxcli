# Version-Aware modelsdk Write Path — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the `CallMicroflowTask`/`CallMicroflowActivity` BSON type-name bug and add systematic version-aware type selection + property gating so future Mendix renames never silently corrupt MPR files.

**Architecture:** The codegen reads `type_renames` from `supplements.json`, validates against the TS SDK `versionInfo`, and generates a version-aware factory `NewXxxForVersion(v version.Version)` per renamed type. The executor passes the project version through a `wfBuildCtx` and calls this factory instead of the plain constructor. Separately, `codec.Encoder` gains a `Version` field that skips properties not yet introduced in the target Mendix version, enforced via a global `VersionRegistry` populated by each domain's `init()`.

**Tech Stack:** Go, `go/template` (codegen), `go.mongodb.org/mongo-driver/v2/bson`, `modelsdk/version.Version`

**Spec:** `docs/superpowers/specs/2026-05-31-version-aware-modelsdk-design.md`

---

## File Map

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/codegen/supplements.json` | Modify | Remove stale alias; add `type_renames` section |
| `cmd/modelsdk-codegen/main.go` | Modify | Load `type_renames`; validate against SDK; pass to emitter |
| `internal/codegen/emitter/emitter.go` | Modify | Add `ClassIntroduced`, `ClassDeleted`, `IsVersionRename` to `TypeData`; add `TypeRenameData`; populate both |
| `internal/codegen/emitter/templates.go` | Modify | Fix `init*()` template; add `NewXxxForVersion` template block; extend `version.go` template with class-level fields and `init()` registration |
| `modelsdk/gen/workflows/types.go` | Regenerate | `initCallMicroflowActivity()` uses new TypeName; `NewCallMicroflowForVersion` generated |
| `modelsdk/gen/workflows/version.go` | Regenerate | class-level `Introduced`/`Deleted` in `TypeVersionInfo`; `init()` registers into `DefaultVersionRegistry` |
| `modelsdk/version/version.go` | Modify | Add `Introduced`/`Deleted` to `TypeVersionInfo`; add `DefaultVersionRegistry` + `VersionRegistry` type |
| `modelsdk/codec/encoder.go` | Modify | Add `Version version.Version` field; `shouldEmitProperty` check in `buildDoc` new-element branch |
| `mdl/backend/mpr/backend.go` | Modify | Add `newEncoder()` helper; replace all `codec.Encoder{}` calls with `b.newEncoder()` |
| `mdl/executor/cmd_workflows_write_gen2.go` | Modify | Add `wfBuildCtx`; thread through all builder functions; use `NewCallMicroflowForVersion` |
| `mdl/executor/cmd_alter_workflow.go` | Modify | `buildAndBindActivitiesGen` receives `wbc` |
| `mdl/executor/cmd_workflows_write_gen2_test.go` | Modify | Update all call sites to pass `wbc`; add TypeName + gating tests |

---

## Phase 1 — Type rename fix (Tasks 1–6)

---

### Task 1: Write failing tests

**Files:**
- Modify: `mdl/executor/cmd_workflows_write_gen2_test.go`

- [ ] **Step 1.1 — Add TypeName unit tests**

Append to `cmd_workflows_write_gen2_test.go`:

```go
func TestNewCallMicroflowActivity_TypeName(t *testing.T) {
	// initCallMicroflowActivity currently sets "Workflows$CallMicroflowTask" — this test proves the bug.
	act := genWf.NewCallMicroflowActivity()
	if act.TypeName() != "Workflows$CallMicroflowActivity" {
		t.Errorf("TypeName = %q, want Workflows$CallMicroflowActivity", act.TypeName())
	}
}

func TestNewCallMicroflowForVersion_Modern(t *testing.T) {
	v := version.Parse("11.9.0")
	elem := genWf.NewCallMicroflowForVersion(v)
	if elem.TypeName() != "Workflows$CallMicroflowActivity" {
		t.Errorf("TypeName for 11.9.0 = %q, want Workflows$CallMicroflowActivity", elem.TypeName())
	}
}

func TestNewCallMicroflowForVersion_Legacy(t *testing.T) {
	v := version.Parse("11.8.0")
	elem := genWf.NewCallMicroflowForVersion(v)
	if elem.TypeName() != "Workflows$CallMicroflowTask" {
		t.Errorf("TypeName for 11.8.0 = %q, want Workflows$CallMicroflowTask", elem.TypeName())
	}
}

func TestNewCallMicroflowForVersion_ZeroVersion(t *testing.T) {
	elem := genWf.NewCallMicroflowForVersion(version.Version{})
	if elem.TypeName() != "Workflows$CallMicroflowTask" {
		t.Errorf("TypeName for zero version = %q, want Workflows$CallMicroflowTask (legacy fallback)", elem.TypeName())
	}
}
```

Add imports at top of file (within existing import block):

```go
import (
    // existing imports...
    "github.com/mendixlabs/mxcli/modelsdk/version"
)
```

- [ ] **Step 1.2 — Run tests to confirm they fail**

```bash
go test ./mdl/executor/ -run "TestNewCallMicroflowActivity_TypeName|TestNewCallMicroflowForVersion" -v
```

Expected: `FAIL — NewCallMicroflowForVersion undefined` and `TypeName got "Workflows$CallMicroflowTask"`.

- [ ] **Step 1.3 — Commit failing tests**

```bash
git add mdl/executor/cmd_workflows_write_gen2_test.go
git commit -m "test(workflow): add failing tests for version-aware CallMicroflow factory"
```

---

### Task 2: Fix supplements.json

**Files:**
- Modify: `internal/codegen/supplements.json`

- [ ] **Step 2.1 — Remove stale alias**

In `internal/codegen/supplements.json`, locate the `storage_aliases` block and **delete** this line:

```json
"Workflows$CallMicroflowActivity":         "Workflows$CallMicroflowTask",
```

- [ ] **Step 2.2 — Add `type_renames` section**

Add the following block immediately after the `storage_aliases` closing brace (before `property_key_overrides`):

```json
  "type_renames": {
    "_doc": "BSON type renames versioned by Mendix release. Key = old BSON name (deleted), value = new BSON name (introduced). The `since` version is auto-derived from old.versionInfo.deleted in the SDK — do not add it here. Codegen validates new.versionInfo.introduced == old.versionInfo.deleted.",
    "Workflows$CallMicroflowTask": "Workflows$CallMicroflowActivity"
  },
```

- [ ] **Step 2.3 — Verify JSON is valid**

```bash
python3 -c "import json; json.load(open('internal/codegen/supplements.json')); print('OK')"
```

Expected: `OK`

- [ ] **Step 2.4 — Commit**

```bash
git add internal/codegen/supplements.json
git commit -m "feat(codegen): add type_renames for CallMicroflowTask→Activity; remove stale alias"
```

---

### Task 3: Extend codegen to load and validate `type_renames`

**Files:**
- Modify: `cmd/modelsdk-codegen/main.go`

- [ ] **Step 3.1 — Add `TypeRenames` field to supplements struct**

In `cmd/modelsdk-codegen/main.go`, update the `supplements` struct (around line 146):

```go
type supplements struct {
	StorageAliases         map[string]string          `json:"storage_aliases"`
	PropertyKeyOverrides   map[string]string          `json:"property_key_overrides"`
	PropertyOrderOverrides map[string][]string        `json:"property_order_overrides"`
	RefListVersion3List    []string                   `json:"ref_list_version3_fields"`
	ForceConcreteTypes     []string                   `json:"force_concrete_types"`
	EdgeKindOverrides      map[string]string          `json:"edge_kind_overrides"`
	IdRefScope             map[string]string          `json:"id_ref_scope"`
	ExtraProperties        map[string]json.RawMessage `json:"extra_properties"`
	ExtraTypes             map[string]json.RawMessage `json:"extra_types"`
	TypeRenames            map[string]string          `json:"type_renames"` // old_bson → new_bson

	// Derived after loading.
	forceConcreteSet      map[string]bool
	refListVersion3Fields map[string]bool
	parsedExtraProps      map[string][]supplementProp
	parsedExtraTypes      map[string][]supplementTypeDef
}
```

- [ ] **Step 3.2 — Clean `_doc` from TypeRenames in loadSupplements**

In `loadSupplements()` (around line 189), add after the other `delete` calls:

```go
delete(s.TypeRenames, "_doc")
```

- [ ] **Step 3.3 — Add conflict guard and validation in main()**

In `main()`, after the aliases loop (after the `meta.StorageAliases = aliases` block, around line 93), add:

```go
// Validate type_renames: derive `since` from SDK versionInfo, check new.introduced == since.
// Also guard against a name appearing in both storage_aliases and type_renames.
type renameEntry struct{ oldName, newName, since string }
var renames []renameEntry
for oldName, newName := range suppl.TypeRenames {
    if _, inAliases := suppl.StorageAliases[newName]; inAliases {
        log.Fatalf("type_renames conflict: %q also appears as a storage_alias new name", newName)
    }
    // Find `since` from the old name's versionInfo.deleted in the current domain's JS.
    oldSince := ""
    newIntroduced := ""
    for _, cls := range meta.Classes {
        if cls.StructureTypeName == oldName && cls.VersionInfo != nil {
            oldSince = cls.VersionInfo.Deleted
        }
        if cls.StructureTypeName == newName && cls.VersionInfo != nil {
            newIntroduced = cls.VersionInfo.Introduced
        }
    }
    if oldSince == "" {
        // old name not in this domain — skip (it will be found in the domain that owns it)
        continue
    }
    if newIntroduced != oldSince {
        log.Fatalf("type_renames validation failed: %q deleted=%q but %q introduced=%q (must match)",
            oldName, oldSince, newName, newIntroduced)
    }
    renames = append(renames, renameEntry{oldName, newName, oldSince})
    fmt.Printf("  type_rename: %s → %s (since %s)\n", oldName, newName, oldSince)
}
meta.TypeRenames = renames // add this field in the next task
```

- [ ] **Step 3.4 — Commit**

```bash
git add cmd/modelsdk-codegen/main.go
git commit -m "feat(codegen): load and validate type_renames from supplements.json"
```

---

### Task 4: Extend emitter TypeData and DomainMeta

**Files:**
- Modify: `internal/codegen/emitter/emitter.go`

- [ ] **Step 4.1 — Add TypeRenameData type**

In `emitter.go`, after the `TypeData` struct definition (around line 50), add:

```go
// TypeRenameData describes a versioned BSON type rename for NewXxxForVersion generation.
type TypeRenameData struct {
	OldTypeName string // "Workflows$CallMicroflowTask"
	NewTypeName string // "Workflows$CallMicroflowActivity"
	Since       string // "11.9.0"  — use new name when project >= Since
	OldGoName   string // "CallMicroflowTask"
	NewGoName   string // "CallMicroflowActivity"
}
```

- [ ] **Step 4.2 — Add fields to TypeData**

Extend the `TypeData` struct:

```go
type TypeData struct {
	Name              string
	StructureTypeName string
	StorageAlias      string
	IsAbstract        bool
	IsVersionRename   bool   // true when this type is the NEW name in a type_renames entry
	ClassIntroduced   string // from cls.VersionInfo.Introduced
	ClassDeleted      string // from cls.VersionInfo.Deleted
	Fields            []FieldData
	Refs              []RefData
}
```

- [ ] **Step 4.3 — Add TypeRenames to DomainMeta (the meta struct passed to Generate)**

Locate the `DomainMeta` (or equivalent) struct passed into `Generate`. It lives in `internal/codegen/dtsparser/jsparser.go`. Add the field there:

```go
// In dtsparser/jsparser.go, DomainMeta struct:
TypeRenames []emitter.TypeRenameData // set by codegen main, not by parser
```

Or, if TypeRenames is passed separately into Generate, add it as a parameter. Check the existing `Generate` signature:

```bash
grep -n "func Generate\|DomainMeta\|type.*Meta" internal/codegen/emitter/emitter.go | head -10
```

Add `TypeRenames []TypeRenameData` to whatever struct or parameter carries domain-level data into `Generate`.

- [ ] **Step 4.4 — Populate IsVersionRename + ClassIntroduced/ClassDeleted in Generate()**

In `emitter.Generate()`, in the loop where `TypeData` is built (around line 152), add:

```go
// Populate class-level version fields.
if cls.VersionInfo != nil {
    td.ClassIntroduced = cls.VersionInfo.Introduced
    td.ClassDeleted    = cls.VersionInfo.Deleted
}
// Mark as version rename target if this type is the NEW name in any rename pair.
for _, r := range meta.TypeRenames {
    if cls.StructureTypeName == r.NewTypeName {
        td.IsVersionRename = true
        break
    }
}
```

- [ ] **Step 4.5 — Derive OldGoName / NewGoName and attach renames to DomainMeta**

In `cmd/modelsdk-codegen/main.go`, after the rename validation loop in Step 3.3, build `emitter.TypeRenameData` values:

```go
var emitRenames []emitter.TypeRenameData
for _, r := range renames {
    oldGo := goNameFromSTN(r.oldName) // helper: "Workflows$CallMicroflowTask" → "CallMicroflowTask"
    newGo := goNameFromSTN(r.newName)
    emitRenames = append(emitRenames, emitter.TypeRenameData{
        OldTypeName: r.oldName,
        NewTypeName: r.newName,
        Since:       r.since,
        OldGoName:   oldGo,
        NewGoName:   newGo,
    })
}
meta.TypeRenames = emitRenames
```

Add helper function:

```go
// goNameFromSTN extracts the Go type name from a structureTypeName.
// "Workflows$CallMicroflowTask" → "CallMicroflowTask"
func goNameFromSTN(stn string) string {
    if idx := strings.Index(stn, "$"); idx >= 0 {
        return stn[idx+1:]
    }
    return stn
}
```

- [ ] **Step 4.6 — Commit**

```bash
git add internal/codegen/emitter/emitter.go cmd/modelsdk-codegen/main.go
git commit -m "feat(codegen): extend TypeData with IsVersionRename/ClassIntroduced/ClassDeleted"
```

---

### Task 5: Update codegen templates

**Files:**
- Modify: `internal/codegen/emitter/templates.go`

- [ ] **Step 5.1 — Fix `init*()` template to use StructureTypeName for rename targets**

In `typesTemplate`, locate the init factory block:

```go
// BEFORE:
o.SetTypeName("{{if .StorageAlias}}{{.StorageAlias}}{{else}}{{.StructureTypeName}}{{end}}")

// AFTER:
o.SetTypeName("{{if and .StorageAlias (not .IsVersionRename)}}{{.StorageAlias}}{{else}}{{.StructureTypeName}}{{end}}")
```

- [ ] **Step 5.2 — Add `NewXxxForVersion` template block**

After the `New{{.Name}}()` function in the factory section, add a new block that only emits when the type is part of a rename. Add this to the `typesFileData` struct passed to the template:

```go
// In emitter.go, typesFileData struct:
type typesFileData struct {
    Package  string
    Types    []TypeData
    Renames  []TypeRenameData // NEW
}
```

Add to the `typesTemplate` (after the `{{end}}{{end}}` of the New/init block):

```go
{{range .Renames}}
// New{{.NewGoName}}ForVersion returns the version-correct concrete type for the
// {{.NewGoName}} / {{.OldGoName}} workflow activity.
//   < {{.Since}} → *{{.OldGoName}} ($Type "{{.OldTypeName}}")
//   ≥ {{.Since}} → *{{.NewGoName}} ($Type "{{.NewTypeName}}")
func New{{.NewGoName}}ForVersion(v version.Version) element.Element {
	if v.Compare(version.Parse("{{.Since}}")) >= 0 {
		return New{{.NewGoName}}()
	}
	return New{{.OldGoName}}()
}
{{end}}
```

Add the version import to the types template header:

```go
import (
    // existing imports
    "github.com/mendixlabs/mxcli/modelsdk/version"
)
```

Only emit this import when `len(.Renames) > 0`. Use a template conditional:

```
{{if .Renames}}
	"github.com/mendixlabs/mxcli/modelsdk/version"
{{end}}
```

- [ ] **Step 5.3 — Pass Renames into typesFileData in Generate()**

In `emitter.Generate()`, where `typesFileData` is constructed, add `Renames: meta.TypeRenames`.

- [ ] **Step 5.4 — Commit templates**

```bash
git add internal/codegen/emitter/templates.go internal/codegen/emitter/emitter.go
git commit -m "feat(codegen): generate NewXxxForVersion factory; fix init* TypeName for rename targets"
```

---

### Task 6: Regenerate workflows and verify

**Files:**
- Regenerate: `modelsdk/gen/workflows/types.go`

- [ ] **Step 6.1 — Run codegen for workflows domain**

```bash
go run ./cmd/modelsdk-codegen -domains workflows
```

Expected output includes:
```
  alias removed (now type_rename): Workflows$CallMicroflowActivity → Workflows$CallMicroflowTask
  type_rename: Workflows$CallMicroflowTask → Workflows$CallMicroflowActivity (since 11.9.0)
Generated workflows: N classes, M enums
```

- [ ] **Step 6.2 — Verify initCallMicroflowActivity TypeName**

```bash
grep -A3 "func initCallMicroflowActivity" modelsdk/gen/workflows/types.go
```

Expected:
```go
func initCallMicroflowActivity() *CallMicroflowActivity {
	o := &CallMicroflowActivity{}
	o.SetTypeName("Workflows$CallMicroflowActivity")
```

- [ ] **Step 6.3 — Verify NewCallMicroflowForVersion generated**

```bash
grep -A8 "func NewCallMicroflowForVersion" modelsdk/gen/workflows/types.go
```

Expected:
```go
func NewCallMicroflowForVersion(v version.Version) element.Element {
	if v.Compare(version.Parse("11.9.0")) >= 0 {
		return NewCallMicroflowActivity()
	}
	return NewCallMicroflowTask()
}
```

- [ ] **Step 6.4 — Run unit tests that were failing**

```bash
go test ./mdl/executor/ -run "TestNewCallMicroflowActivity_TypeName|TestNewCallMicroflowForVersion" -v
```

Expected: `TestNewCallMicroflowActivity_TypeName PASS`, `TestNewCallMicroflowForVersion_Modern PASS`, `TestNewCallMicroflowForVersion_Legacy PASS`, `TestNewCallMicroflowForVersion_ZeroVersion PASS`.

- [ ] **Step 6.5 — Run full test suite**

```bash
go test ./modelsdk/... ./mdl/...
```

Expected: no new failures.

- [ ] **Step 6.6 — Commit generated files**

```bash
git add modelsdk/gen/workflows/types.go
git commit -m "feat(gen): regenerate workflows — NewCallMicroflowForVersion; fix initCallMicroflowActivity TypeName"
```

---

### Task 7: Add `wfBuildCtx` and thread through executor

**Files:**
- Modify: `mdl/executor/cmd_workflows_write_gen2.go`
- Modify: `mdl/executor/cmd_alter_workflow.go`
- Modify: `mdl/executor/cmd_workflows_write_gen2_test.go`

- [ ] **Step 7.1 — Add wfBuildCtx struct**

At the top of `cmd_workflows_write_gen2.go` (after package and imports), add:

```go
import (
    // existing imports ...
    "github.com/mendixlabs/mxcli/modelsdk/version"
    genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// wfBuildCtx carries the project version through the stateless workflow
// builder chain so version-aware factory functions can select the correct
// BSON type name.
type wfBuildCtx struct {
    version version.Version // zero = treat as oldest (legacy fallback)
}

// newWfBuildCtx creates a wfBuildCtx from the execution context.
func newWfBuildCtx(ctx *ExecContext) *wfBuildCtx {
    wbc := &wfBuildCtx{}
    if ctx != nil && ctx.Connected() {
        rpv := ctx.Backend.ProjectVersion()
        wbc.version = version.Parse(rpv.ProductVersion)
    }
    return wbc
}
```

- [ ] **Step 7.2 — Update function signatures (add wbc parameter)**

Apply the following signature changes throughout `cmd_workflows_write_gen2.go`. For each function, add `wbc *wfBuildCtx` as the first parameter and pass it to any recursive call that previously had none:

| Function | Change |
|----------|--------|
| `buildWorkflowActivitiesGen` | add `wbc *wfBuildCtx`; pass to `buildWorkflowActivityGen` |
| `buildWorkflowActivityGen` | add `wbc *wfBuildCtx`; pass to composite builders |
| `buildBoundaryEventsGen` | add `wbc *wfBuildCtx`; pass to `buildBoundaryEventGen` |
| `buildBoundaryEventGen` | add `wbc *wfBuildCtx`; pass to `buildWorkflowActivitiesGen` |
| `buildUserTaskGenActivity` | add `wbc *wfBuildCtx`; pass to sub-builders |
| `buildSingleUserTaskGenActivity` | add `wbc *wfBuildCtx`; pass to `buildUserTaskOutcomesGen`, `buildBoundaryEventsGen` |
| `buildMultiUserTaskGenActivity` | add `wbc *wfBuildCtx`; pass to `buildUserTaskOutcomesGen`, `buildBoundaryEventsGen` |
| `buildUserTaskOutcomesGen` | add `wbc *wfBuildCtx`; pass to `buildWorkflowActivitiesGen` |
| `buildCallMicroflowGenActivity` | add `wbc *wfBuildCtx`; use `NewCallMicroflowForVersion` |
| `buildConditionOutcomesGen` | add `wbc *wfBuildCtx`; pass to `buildConditionOutcomeGen` |
| `buildConditionOutcomeGen` | add `wbc *wfBuildCtx`; pass to `buildWorkflowActivitiesGen` |
| `buildExclusiveSplitGenActivity` | add `wbc *wfBuildCtx`; pass to `buildConditionOutcomeGen` |
| `buildParallelSplitGenActivity` | add `wbc *wfBuildCtx`; pass to `buildWorkflowActivitiesGen` |
| `buildWaitForNotificationGenActivity` | add `wbc *wfBuildCtx`; pass to `buildBoundaryEventsGen` |

Functions that do **not** need wbc (no recursive dependency on CallMicroflow): `buildJumpToGenActivity`, `buildWaitForTimerGenActivity`, `buildEndWorkflowGenActivity`, `buildAnnotationActivityGen`, `buildCallWorkflowGenActivity`.

- [ ] **Step 7.3 — Update buildCallMicroflowGenActivity to use factory**

```go
func buildCallMicroflowGenActivity(wbc *wfBuildCtx, n *ast.WorkflowCallMicroflowNode) element.Element {
    act := genWf.NewCallMicroflowForVersion(wbc.version) // version-correct type
    act.(interface{ SetID(element.ID) }).SetID(element.ID(types.GenerateID()))
    // ... rest of function unchanged, but cast act to the common interface
```

Since `NewCallMicroflowForVersion` returns `element.Element`, use type-assertion to access setters shared by both types. Both `*CallMicroflowActivity` and `*CallMicroflowTask` implement the same setters. Use the existing setter methods via interface or cast:

```go
func buildCallMicroflowGenActivity(wbc *wfBuildCtx, n *ast.WorkflowCallMicroflowNode) element.Element {
    act := genWf.NewCallMicroflowForVersion(wbc.version)
    act.SetID(element.ID(types.GenerateID()))
    
    // Both *CallMicroflowActivity and *CallMicroflowTask expose these via embedded Base:
    name := n.Microflow.Name
    caption := n.Caption
    if caption == "" {
        caption = name
    }
    mfQN := n.Microflow.Module + "." + name
    
    // Use type switch to access type-specific setters:
    switch v := act.(type) {
    case *genWf.CallMicroflowActivity:
        v.SetID(element.ID(types.GenerateID()))
        v.SetName(name)
        v.SetCaption(caption)
        v.SetMicroflowQualifiedName(mfQN)
        for _, oc := range buildConditionOutcomesGen(wbc, n.Outcomes) {
            v.AddOutcomes(oc)
        }
        for _, pm := range n.ParameterMappings {
            mapping := genWf.NewMicroflowCallParameterMapping()
            mapping.SetID(element.ID(types.GenerateID()))
            mapping.SetParameterQualifiedName(mfQN + "." + pm.Parameter)
            mapping.SetExpression(pm.Expression)
            v.AddParameterMappings(mapping)
        }
        for _, ev := range buildBoundaryEventsGen(wbc, n.BoundaryEvents) {
            v.AddBoundaryEvents(ev)
        }
    case *genWf.CallMicroflowTask:
        v.SetID(element.ID(types.GenerateID()))
        v.SetName(name)
        v.SetCaption(caption)
        v.SetMicroflowQualifiedName(mfQN)
        for _, oc := range buildConditionOutcomesGen(wbc, n.Outcomes) {
            v.AddOutcomes(oc)
        }
        for _, pm := range n.ParameterMappings {
            mapping := genWf.NewMicroflowCallParameterMapping()
            mapping.SetID(element.ID(types.GenerateID()))
            mapping.SetParameterQualifiedName(mfQN + "." + pm.Parameter)
            mapping.SetExpression(pm.Expression)
            v.AddParameterMappings(mapping)
        }
        for _, ev := range buildBoundaryEventsGen(wbc, n.BoundaryEvents) {
            v.AddBoundaryEvents(ev)
        }
    }
    return act
}
```

- [ ] **Step 7.4 — Update execCreateWorkflowGen to use wbc**

In `execCreateWorkflowGen` (around line 752 of the original), replace:

```go
// BEFORE:
userActivities := buildWorkflowActivitiesGen(s.Activities)

// AFTER:
wbc := newWfBuildCtx(ctx)
userActivities := buildWorkflowActivitiesGen(wbc, s.Activities)
```

- [ ] **Step 7.5 — Update cmd_alter_workflow.go**

In `buildAndBindActivitiesGen`:

```go
// BEFORE:
func buildAndBindActivitiesGen(ctx *ExecContext, nodes []ast.WorkflowActivityNode) []element.Element {
    acts := buildWorkflowActivitiesGen(nodes)

// AFTER:
func buildAndBindActivitiesGen(ctx *ExecContext, nodes []ast.WorkflowActivityNode) []element.Element {
    wbc := newWfBuildCtx(ctx)
    acts := buildWorkflowActivitiesGen(wbc, nodes)
```

- [ ] **Step 7.6 — Update existing tests to pass wbc**

In `cmd_workflows_write_gen2_test.go`, all calls to functions that now require `wbc` must be updated. Use `&wfBuildCtx{}` (zero version = legacy behavior) for tests that don't care about version:

```go
// Example — find every occurrence and replace:
// BEFORE: buildBoundaryEventGen(ast.WorkflowBoundaryEventNode{...})
// AFTER:  buildBoundaryEventGen(&wfBuildCtx{}, ast.WorkflowBoundaryEventNode{...})

// BEFORE: buildUserTaskGenActivity(n)
// AFTER:  buildUserTaskGenActivity(&wfBuildCtx{}, n)

// BEFORE: buildCallMicroflowGenActivity(n)
// AFTER:  buildCallMicroflowGenActivity(&wfBuildCtx{version: version.Parse("11.10.0")}, n)
// (or &wfBuildCtx{} for tests that test non-TypeName properties)

// BEFORE: buildWorkflowActivitiesGen(nodes)
// AFTER:  buildWorkflowActivitiesGen(&wfBuildCtx{}, nodes)
```

Run this sed to find all locations needing update:

```bash
grep -n "buildWorkflowActivitiesGen\|buildBoundaryEventGen\|buildBoundaryEventsGen\|buildUserTaskGenActivity\|buildSingleUserTaskGenActivity\|buildMultiUserTaskGenActivity\|buildUserTaskOutcomesGen\|buildCallMicroflowGenActivity\|buildConditionOutcomeGen\|buildExclusiveSplitGenActivity\|buildParallelSplitGenActivity\|buildWaitForNotificationGenActivity" \
  mdl/executor/cmd_workflows_write_gen2_test.go
```

- [ ] **Step 7.7 — Run all tests**

```bash
go test ./mdl/executor/ -v -count=1 2>&1 | tail -20
```

Expected: all pass including the new TypeName tests from Task 1.

- [ ] **Step 7.8 — Commit**

```bash
git add mdl/executor/cmd_workflows_write_gen2.go \
        mdl/executor/cmd_alter_workflow.go \
        mdl/executor/cmd_workflows_write_gen2_test.go
git commit -m "feat(executor): wfBuildCtx threads project version; NewCallMicroflowForVersion replaces plain constructor"
```

---

## Phase 2 — Property gating (Tasks 8–11)

---

### Task 8: Extend TypeVersionInfo with class-level fields and VersionRegistry

**Files:**
- Modify: `modelsdk/version/version.go`

- [ ] **Step 8.1 — Add Introduced/Deleted to TypeVersionInfo**

In `modelsdk/version/version.go`, update `TypeVersionInfo`:

```go
type TypeVersionInfo struct {
	Introduced string // Mendix version when this type was introduced (empty = baseline)
	Deleted    string // Mendix version when this type was deleted (empty = still present)
	Properties map[string]PropertyVersionInfo
}
```

- [ ] **Step 8.2 — Add VersionRegistry**

Append to `modelsdk/version/version.go`:

```go
import "sync"

// DefaultVersionRegistry is the global registry of TypeVersionInfo, populated
// by each domain package's init() function via generated code.
var DefaultVersionRegistry = &VersionRegistry{}

// VersionRegistry stores TypeVersionInfo by BSON type name.
type VersionRegistry struct {
	mu      sync.RWMutex
	entries map[string]TypeVersionInfo
}

// Register adds or replaces a TypeVersionInfo entry.
func (r *VersionRegistry) Register(typeName string, info TypeVersionInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.entries == nil {
		r.entries = make(map[string]TypeVersionInfo)
	}
	r.entries[typeName] = info
}

// Lookup returns the TypeVersionInfo for a BSON type name, if registered.
func (r *VersionRegistry) Lookup(typeName string) (TypeVersionInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, ok := r.entries[typeName]
	return info, ok
}
```

- [ ] **Step 8.3 — Verify package compiles**

```bash
go build ./modelsdk/version/...
```

Expected: no errors.

- [ ] **Step 8.4 — Commit**

```bash
git add modelsdk/version/version.go
git commit -m "feat(version): extend TypeVersionInfo with Introduced/Deleted; add DefaultVersionRegistry"
```

---

### Task 9: Update version.go template to emit class-level data and register

**Files:**
- Modify: `internal/codegen/emitter/emitter.go` (VersionData struct + Generate logic)
- Modify: `internal/codegen/emitter/templates.go` (versionTemplate)

- [ ] **Step 9.1 — Add ClassIntroduced/ClassDeleted to VersionData**

In `emitter.go`, locate the `VersionData` and `VersionPropData` structs. Add:

```go
type VersionData struct {
	StructureTypeName string
	ClassIntroduced   string // NEW
	ClassDeleted      string // NEW
	Props             []VersionPropData
}
```

- [ ] **Step 9.2 — Populate ClassIntroduced/ClassDeleted in Generate()**

In the versions-building loop (around line 241), after `vd := VersionData{StructureTypeName: mappedName}`:

```go
vd.ClassIntroduced = cls.VersionInfo.Introduced
vd.ClassDeleted    = cls.VersionInfo.Deleted
```

Also emit a version entry even when `len(cls.VersionInfo.PropertyInfos) == 0` but `cls.VersionInfo.Introduced != ""` or `cls.VersionInfo.Deleted != ""` — remove the `if len(cls.VersionInfo.PropertyInfos) == 0 { continue }` guard or weaken it to:

```go
hasProps    := len(cls.VersionInfo.PropertyInfos) > 0
hasClassVer := cls.VersionInfo.Introduced != "" || cls.VersionInfo.Deleted != ""
if !hasProps && !hasClassVer {
    continue
}
```

- [ ] **Step 9.3 — Update versionTemplate to emit class-level fields and init() registration**

Replace the `versionTemplate` string in `templates.go`:

```go
const versionTemplate = `// Code generated by mxcli codegen; DO NOT EDIT.
// To modify: edit internal/codegen/supplements.json or internal/codegen/emitter/templates.go,
// then run: go run ./cmd/modelsdk-codegen

package {{.Package}}

import "github.com/mendixlabs/mxcli/modelsdk/version"

// VersionInfos maps structure-type names to their TypeVersionInfo.
var VersionInfos = map[string]version.TypeVersionInfo{
{{- range .Versions}}
	"{{.StructureTypeName}}": {
		{{- if .ClassIntroduced}}Introduced: "{{.ClassIntroduced}}",{{end}}
		{{- if .ClassDeleted}}Deleted: "{{.ClassDeleted}}",{{end}}
		Properties: map[string]version.PropertyVersionInfo{
		{{- range .Props}}
			"{{.Name}}": {
				{{- if .Introduced}}Introduced: "{{.Introduced}}", {{end -}}
				{{- if .Deleted}}Deleted: "{{.Deleted}}", {{end -}}
				{{- if .Required}}Required: true, {{end -}}
				{{- if .Public}}Public: true, {{end -}}
			},
		{{- end}}
		},
	},
{{- end}}
}

func init() {
	for name, info := range VersionInfos {
		version.DefaultVersionRegistry.Register(name, info)
	}
}
`
```

- [ ] **Step 9.4 — Regenerate all domains**

```bash
go run ./cmd/modelsdk-codegen
```

Expected: all domain version.go files regenerated. Each will have a `func init()` that registers into `DefaultVersionRegistry`.

- [ ] **Step 9.5 — Verify workflows/version.go**

```bash
grep -A4 "CallMicroflowTask\|func init" modelsdk/gen/workflows/version.go | head -20
```

Expected: `CallMicroflowTask` entry has `Introduced: "9.0.2", Deleted: "11.9.0"`. `func init()` block present.

- [ ] **Step 9.6 — Run tests**

```bash
go test ./modelsdk/... ./internal/...
```

Expected: no failures.

- [ ] **Step 9.7 — Commit**

```bash
git add internal/codegen/emitter/emitter.go \
        internal/codegen/emitter/templates.go \
        modelsdk/gen/
git commit -m "feat(gen): emit class-level Introduced/Deleted in TypeVersionInfo; add init() VersionRegistry registration"
```

---

### Task 10: Version-aware codec.Encoder

**Files:**
- Modify: `modelsdk/codec/encoder.go`
- Modify: `modelsdk/codec/encoder_test.go` (if exists, otherwise create)

- [ ] **Step 10.1 — Write failing property gating test**

In `modelsdk/codec/encoder_test.go` (create if missing):

```go
package codec_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/version"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestEncoder_SkipsPropertyNotYetIntroduced(t *testing.T) {
	// boundaryEvents introduced in 10.14.0; project is 10.13.0 → must be absent
	act := genWf.NewCallMicroflowTask()
	act.SetID("test-id")
	act.SetName("MyMF")
	be := genWf.NewTimerBoundaryEvent()
	be.SetID("be-id")
	act.AddBoundaryEvents(be) // mark dirty

	enc := &codec.Encoder{Version: version.Parse("10.13.0")}
	data, err := enc.Encode(act)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var doc bson.M
	if err := bson.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := doc["BoundaryEvents"]; ok {
		t.Error("BoundaryEvents should be absent for project version 10.13.0")
	}
}

func TestEncoder_EmitsPropertyWhenVersionSufficient(t *testing.T) {
	act := genWf.NewCallMicroflowTask()
	act.SetID("test-id")
	act.SetName("MyMF")
	be := genWf.NewTimerBoundaryEvent()
	be.SetID("be-id")
	act.AddBoundaryEvents(be)

	enc := &codec.Encoder{Version: version.Parse("10.14.0")}
	data, err := enc.Encode(act)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var doc bson.M
	if err := bson.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := doc["BoundaryEvents"]; !ok {
		t.Error("BoundaryEvents should be present for project version 10.14.0")
	}
}

func TestEncoder_NoVersionGating_EmitsAll(t *testing.T) {
	// Zero version = no gating; all dirty properties emitted
	act := genWf.NewCallMicroflowTask()
	act.SetID("test-id")
	act.SetName("MyMF")
	be := genWf.NewTimerBoundaryEvent()
	be.SetID("be-id")
	act.AddBoundaryEvents(be)

	enc := &codec.Encoder{} // zero version
	data, err := enc.Encode(act)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var doc bson.M
	bson.Unmarshal(data, &doc)
	if _, ok := doc["BoundaryEvents"]; !ok {
		t.Error("BoundaryEvents should be present with zero-version encoder")
	}
}
```

Run to confirm failure:

```bash
go test ./modelsdk/codec/ -run "TestEncoder_Skips\|TestEncoder_Emits\|TestEncoder_No" -v
```

Expected: `FAIL` (Version field undefined).

- [ ] **Step 10.2 — Add Version field and shouldEmitProperty**

In `modelsdk/codec/encoder.go`:

Add import:

```go
import (
    // existing...
    "github.com/mendixlabs/mxcli/modelsdk/version"
)
```

Update struct:

```go
// Encoder serializes Element trees back to BSON bytes.
type Encoder struct {
    // Version gates property emission: properties introduced after this
    // Mendix version are skipped for new elements. Zero value = no gating.
    Version version.Version
}
```

Add helper (before `Encode`):

```go
// shouldEmitProperty returns false when e.Version is set and the named property
// is not yet available in that version according to DefaultVersionRegistry.
// Only applied on the new-element (raw == nil) path.
func (e *Encoder) shouldEmitProperty(typeName, propName string) bool {
    if e.Version.IsZero() {
        return true
    }
    info, ok := version.DefaultVersionRegistry.Lookup(typeName)
    if !ok {
        return true // type not in registry — no restriction
    }
    pvi, ok := info.Properties[propName]
    if !ok {
        return true // property not in registry — no restriction
    }
    return pvi.IsAvailableIn(e.Version)
}
```

- [ ] **Step 10.3 — Apply gating in buildDoc new-element branch**

In `buildDoc()`, inside the `if raw == nil` block, the property loop currently reads:

```go
for _, prop := range elem.Properties() {
    idx := findRebuild(bytesOf(prop.Name()))
    if idx < 0 {
        continue
    }
    val, err := e.encodeEntry(rebuild[idx])
```

Change to:

```go
for _, prop := range elem.Properties() {
    if !e.shouldEmitProperty(elem.TypeName(), prop.Name()) {
        continue // skip: not available in project version
    }
    idx := findRebuild(bytesOf(prop.Name()))
    if idx < 0 {
        continue
    }
    val, err := e.encodeEntry(rebuild[idx])
```

- [ ] **Step 10.4 — Run gating tests**

```bash
go test ./modelsdk/codec/ -run "TestEncoder_Skips\|TestEncoder_Emits\|TestEncoder_No" -v
```

Expected: all three pass.

- [ ] **Step 10.5 — Run full codec tests**

```bash
go test ./modelsdk/codec/ -v
```

Expected: no regressions.

- [ ] **Step 10.6 — Commit**

```bash
git add modelsdk/codec/encoder.go modelsdk/codec/encoder_test.go
git commit -m "feat(codec): Encoder.Version gates property emission for new elements"
```

---

### Task 11: Version-aware encoder in MprBackend

**Files:**
- Modify: `mdl/backend/mpr/backend.go`

- [ ] **Step 11.1 — Add newEncoder() helper**

In `mdl/backend/mpr/backend.go`, add:

```go
import (
    // existing...
    mdlversion "github.com/mendixlabs/mxcli/modelsdk/version"
)

// newEncoder returns a codec.Encoder configured with the project's Mendix version
// for property-level gating. Called for all write operations.
func (b *MprBackend) newEncoder() *codec.Encoder {
    pv := b.msdkReader.ProjectVersion()
    return &codec.Encoder{
        Version: mdlversion.Parse(pv.ProductVersion),
    }
}
```

- [ ] **Step 11.2 — Replace codec.Encoder{} with b.newEncoder()**

Find all occurrences of `codec.Encoder{}` and `&codec.Encoder{}` in `backend.go`:

```bash
grep -n "codec\.Encoder{}" mdl/backend/mpr/backend.go
```

Replace each with `b.newEncoder()`. Also update `newEncoderForGenSerialize()` if it exists — replace with `b.newEncoder()` or remove if it's only called from the backend.

- [ ] **Step 11.3 — Run backend tests**

```bash
go test ./mdl/backend/... -v 2>&1 | tail -20
```

Expected: no failures.

- [ ] **Step 11.4 — Run full test suite**

```bash
go test ./... 2>&1 | grep -E "FAIL|ok" | tail -20
```

Expected: all packages `ok`.

- [ ] **Step 11.5 — Commit**

```bash
git add mdl/backend/mpr/backend.go
git commit -m "feat(backend): use version-aware Encoder for all workflow write operations"
```

---

## Phase 3 — Integration verification (Task 12)

---

### Task 12: End-to-end integration test and roundtrip

**Files:**
- Modify: `mdl/executor/roundtrip_workflow_test.go` (integration, build tag `integration`)

- [ ] **Step 12.1 — Add version-specific roundtrip test**

In `mdl/executor/roundtrip_workflow_test.go`, append a test that explicitly checks the BSON `$type` written for a call microflow activity:

```go
func TestCallMicroflowTypeName_LegacyProject(t *testing.T) {
	// Uses testdata/helpdesk-golden/minimal.mpr which is Mendix 11.6.6.
	// Expects "Workflows$CallMicroflowTask" written.
	env := openExistingTestEnv(t, "../../testdata/helpdesk-golden/minimal.mpr")
	defer env.teardown()

	acts, err := env.backend.SerializeWorkflowActivityGen(func() element.Element {
		wbc := &wfBuildCtx{version: version.Parse("11.6.6")}
		n := &ast.WorkflowCallMicroflowNode{
			Microflow: ast.QualifiedName{Module: "HD", Name: "WFS_Init"},
		}
		return buildCallMicroflowGenActivity(wbc, n)
	}())
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	doc, _ := acts.(bson.D)
	var typeName string
	for _, e := range doc {
		if e.Key == "$Type" {
			typeName, _ = e.Value.(string)
		}
	}
	if typeName != "Workflows$CallMicroflowTask" {
		t.Errorf("$Type = %q, want Workflows$CallMicroflowTask for 11.6.6 project", typeName)
	}
}
```

- [ ] **Step 12.2 — Run integration tests**

```bash
go test ./mdl/executor/ -tags integration -run "TestCallMicroflowTypeName|TestRoundtripWorkflow" -v -timeout 120s
```

Expected: pass.

- [ ] **Step 12.3 — Final full test sweep**

```bash
go test ./... 2>&1 | grep -v "^ok" | grep -v "^?" | head -30
```

Expected: no FAIL lines.

- [ ] **Step 12.4 — Final commit**

```bash
git add mdl/executor/roundtrip_workflow_test.go
git commit -m "test(executor): integration roundtrip for version-aware CallMicroflow type selection"
```

---

## Self-Review Checklist

- [x] **Spec § Type renames → `supplements.json`**: covered by Task 2
- [x] **Spec § Codegen pipeline (TypeData, type_renames loading, validation)**: Tasks 3–4
- [x] **Spec § Templates (IsVersionRename, NewXxxForVersion)**: Task 5
- [x] **Spec § Generated artifacts (fixed initCallMicroflowActivity, factory)**: Task 6
- [x] **Spec § TypeVersionInfo class-level + VersionRegistry**: Tasks 8–9
- [x] **Spec § codec.Encoder{Version}**: Task 10
- [x] **Spec § MprBackend.newEncoder()**: Task 11
- [x] **Spec § wfBuildCtx + executor threading**: Task 7
- [x] **Decision: zero version = no gating**: `Version.IsZero()` check in shouldEmitProperty (Task 10)
- [x] **Decision: wfBuildCtx workflow-only**: threaded only in cmd_workflows_write_gen2.go + cmd_alter_workflow.go
- [x] **Decision: version boundary 11.9.0**: hardcoded in generated `NewCallMicroflowForVersion`
- [x] **No placeholders**: all code blocks contain actual compilable code
- [x] **TDD**: failing tests written first in Tasks 1 and 10
- [x] **Type consistency**: `version.Version` (not `version.SemVer`) used throughout; `version.Parse(rpv.ProductVersion)` at entry points

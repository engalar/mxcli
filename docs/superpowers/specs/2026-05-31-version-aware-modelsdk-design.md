# Version-Aware modelsdk Write Path

**Date:** 2026-05-31  
**Status:** Draft  
**Scope:** BSON type-name selection and property gating based on Mendix project version

---

## Problem Statement

Two classes of version-dependent behavior are currently missing from the write path:

**Type renames.** Mendix renames BSON `$Type` strings between major versions. `CallMicroflowTask` was deleted in 11.9.0 and replaced by `CallMicroflowActivity`. mxcli always writes the old name, causing the 11.10.0 runtime to crash with `Class 'Workflows$CallMicroflowTask' could not be found`. This is not a one-off bug — the TS SDK records 171 class-level deletions. Future renames will recur.

**Property gating.** Properties have `introduced`/`deleted` version metadata in the generated `version.go`, but nothing in the write path enforces it. Writing a property introduced in 10.14.0 to a 10.13.0 project silently produces an MPR that Studio Pro rejects on load.

---

## Evidence Basis

Key findings from the research phase:

| Finding | Evidence |
|---------|---------|
| dtsparser already parses class-level `introduced`/`deleted` | `jsparser.go:97-99` |
| emitter silently drops class-level version | `TypeData` struct has no `Introduced`/`Deleted`; `emitter.go:152` |
| `TypeVersionInfo` is property-only | `version.go:70` — only `Properties map[string]PropertyVersionInfo` |
| 48/49 `storage_aliases` are permanent BSON names (not versioned) | All 48 appear in MPR corpus |
| 1/49 alias is a true version rename: `CallMicroflowTask` → `CallMicroflowActivity` since 11.9.0 | `workflows.js` versionInfo |
| auto-derivation of old→new mapping is unreliable | 4 "unambiguous" auto-candidates are all semantically wrong |
| `$Type` is written at `codec.Encoder.buildDoc()` from `elem.TypeName()` | `encoder.go:131` |
| `TypeName` is mutable via `SetTypeName()` before codec is called | `element.go:71` |
| `codec.Encoder` is a stateless struct `type Encoder struct{}` | `encoder.go:15` |
| `MprBackend` holds `b.msdkReader.ProjectVersion()` | `backend.go:135` |

---

## Non-Goals

- Changing `supplements.json` `storage_aliases` — these are permanent BSON names, not versioned.
- Enforcing read-time property filtering (existing BSON is trusted as-is).
- Auto-detecting new type renames without human confirmation.
- Retroactively gating existing MPRs (only applies to new writes).

---

## Architecture: Five Layers

```
supplements.json          ← human declares (old → new) pair; nothing else
      ↓
cmd/modelsdk-codegen      ← reads SDK versionInfo, derives `since`, validates,
                             generates type-safe factory + extends TypeVersionInfo
      ↓
modelsdk/gen/*/           ← NewXxxForVersion(v) factory + class-level TypeVersionInfo
      ↓
modelsdk/codec/           ← Encoder{Version} skips unavailable properties
      ↓
mdl/executor/ (wfBuildCtx)← carries SemVer, calls version-aware factory
```

**Invariant:** No BSON type-name string ever appears in executor or backend business logic. All type-name knowledge lives in generated `init*()` functions. Callers only see Go types and `version.SemVer`.

---

## Component 1 — supplements.json: `type_renames`

New section, separate from `storage_aliases`:

```json
"type_renames": {
  "_doc": "BSON type renames that took effect at a specific Mendix version. Key = old BSON name (deleted), value = new BSON name (introduced). The `since` version is derived automatically from the SDK versionInfo — do not add it here. Codegen validates that new.introduced == old.deleted.",
  "Workflows$CallMicroflowTask": "Workflows$CallMicroflowActivity"
}
```

**What humans provide:** only the `old → new` pair. One line per rename.  
**What codegen derives automatically:** the `since` version from `old.versionInfo.deleted`.  
**Codegen validation (fatal if fails):** `new.versionInfo.introduced == old.versionInfo.deleted`.

### Why not storage_aliases?

`storage_aliases` means "the BSON name is permanently different from the SDK name; write the old name always." `type_renames` means "the BSON name changed at version X; write old name before X, new name from X onward." The two concepts must stay separate.

---

## Component 2 — Codegen Pipeline Extensions

### 2a. `emitter.TypeData` gains class-level version fields

```go
type TypeData struct {
    Name              string
    StructureTypeName string
    StorageAlias      string
    IsAbstract        bool
    Fields            []FieldData
    Refs              []RefData
    // NEW:
    ClassIntroduced   string // from cls.VersionInfo.Introduced
    ClassDeleted      string // from cls.VersionInfo.Deleted
    IsVersionRename   bool   // true when this type is the NEW name in a type_renames entry
}
```

Populated in `emitter.Generate()` from the already-parsed `cls.VersionInfo` (data has been available since the dtsparser was written — just unused).

### 2b. `cmd/modelsdk-codegen/main.go` loads `type_renames`

```go
type supplements struct {
    StorageAliases  map[string]string `json:"storage_aliases"`
    // ... existing fields ...
    // NEW:
    TypeRenames map[string]string `json:"type_renames"` // old_bson_name → new_bson_name
}
```

Processing loop (per domain):
1. For each entry `(old, new)` in `TypeRenames`:
   - Find `old` class in domain JS → read `versionInfo.deleted` → `since`
   - Find `new` class in domain JS → read `versionInfo.introduced` → validate == `since`
   - If not found or mismatch → `log.Fatalf`
2. Pass `(old, new, since)` triples to the emitter as `TypeRenameData`.

### 2c. New emitter input type

```go
type TypeRenameData struct {
    OldTypeName string // "Workflows$CallMicroflowTask"
    NewTypeName string // "Workflows$CallMicroflowActivity"
    Since       string // "11.9.0"
    OldGoName   string // "CallMicroflowTask"
    NewGoName   string // "CallMicroflowActivity"
}
```

---

## Component 3 — Generated Artifacts

### 3a. Fix `initCallMicroflowActivity()` TypeName (the immediate bug)

```go
// BEFORE (bug):
func initCallMicroflowActivity() *CallMicroflowActivity {
    o := &CallMicroflowActivity{}
    o.SetTypeName("Workflows$CallMicroflowTask")  // ← wrong: storage_alias applied unconditionally

// AFTER:
func initCallMicroflowActivity() *CallMicroflowActivity {
    o := &CallMicroflowActivity{}
    o.SetTypeName("Workflows$CallMicroflowActivity")  // ← correct: StructureTypeName
```

**Template change is surgical, not global.** The existing template rule:

```
SetTypeName("{{if .StorageAlias}}{{.StorageAlias}}{{else}}{{.StructureTypeName}}{{end}}")
```

must become:

```
SetTypeName("{{if and .StorageAlias (not .IsVersionRename)}}{{.StorageAlias}}{{else}}{{.StructureTypeName}}{{end}}")
```

`IsVersionRename` is set on `TypeData` when the type appears as the *new* name in a `type_renames` entry. For the 48 permanent `storage_aliases`, `IsVersionRename = false` so the template is unchanged — those `init*()` still write the old BSON name. Only `CallMicroflowActivity` (and any future rename targets) get `IsVersionRename = true` and use `StructureTypeName`.

**supplements.json change:** Remove the existing `"Workflows$CallMicroflowActivity": "Workflows$CallMicroflowTask"` entry from `storage_aliases` — it is superseded by the `type_renames` entry. Codegen will fatal if both are present for the same type (conflict guard).

### 3b. Generated version-aware factory — `NewXxxForVersion`

Generated into `modelsdk/gen/<domain>/types.go` for each rename pair:

```go
// NewCallMicroflowForVersion returns the version-correct concrete type.
//   < 11.9.0 → *CallMicroflowTask   ($Type "Workflows$CallMicroflowTask")
//   ≥ 11.9.0 → *CallMicroflowActivity ($Type "Workflows$CallMicroflowActivity")
func NewCallMicroflowForVersion(v version.SemVer) element.Element {
    if v.Compare(version.Parse("11.9.0")) >= 0 {
        return NewCallMicroflowActivity()
    }
    return NewCallMicroflowTask()
}
```

No string appears in the caller. The Go type returned determines the `$Type` written.

### 3c. Class-level version in `TypeVersionInfo`

```go
// modelsdk/version/version.go — extended
type TypeVersionInfo struct {
    Introduced string // class introduced in this version (empty = baseline)
    Deleted    string // class deleted in this version (empty = still present)
    Properties map[string]PropertyVersionInfo
}
```

`version.go` template gains a class-level header per type:

```go
var VersionInfos = map[string]version.TypeVersionInfo{
    "Workflows$CallMicroflowTask": {
        Introduced: "9.0.2",
        Deleted:    "11.9.0",
        Properties: map[string]version.PropertyVersionInfo{
            "boundaryEvents": {Introduced: "10.14.0", Public: true},
        },
    },
    "Workflows$CallMicroflowActivity": {
        Introduced: "11.9.0",
        Properties: map[string]version.PropertyVersionInfo{ ... },
    },
}
```

---

## Component 4 — `codec.Encoder{Version}` for Property Gating

```go
// modelsdk/codec/encoder.go
type Encoder struct {
    Version version.SemVer // zero value = skip gating (backward compat)
}
```

In `buildDoc()`, before writing each property:

```go
func (e *Encoder) shouldEmitProperty(typeName string, prop element.Property) bool {
    if e.Version == (version.SemVer{}) {
        return true // no version set → write everything (read path, tests)
    }
    vi, ok := lookupPropertyVersionInfo(typeName, prop.Name())
    if !ok {
        return true // unknown property → write it (safe default)
    }
    return vi.IsAvailableIn(e.Version)
}
```

`lookupPropertyVersionInfo` is a package-level function over the generated `VersionInfos` maps registered at init time (same pattern as the codec registry).

### Backend creates version-aware encoder

```go
// mdl/backend/mpr/backend.go
func (b *MprBackend) newEncoder() *codec.Encoder {
    pv := b.msdkReader.ProjectVersion()
    return &codec.Encoder{
        Version: version.SemVer{
            Major: pv.MajorVersion,
            Minor: pv.MinorVersion,
            Patch: pv.PatchVersion,
        },
    }
}
```

All callers of `codec.Encoder{}` inside `MprBackend` switch to `b.newEncoder()`. Zero changes to executor layer for property gating.

---

## Component 5 — Executor `wfBuildCtx` for Type Selection

### Structure

```go
// mdl/executor/cmd_workflows_write_gen2.go
type wfBuildCtx struct {
    version version.SemVer // project version; zero = treat as latest
}

func newWfBuildCtx(ctx *ExecContext) *wfBuildCtx {
    wbc := &wfBuildCtx{}
    if ctx != nil && ctx.Connected() {
        rpv := ctx.Backend.ProjectVersion()
        wbc.version = version.SemVer{
            Major: rpv.MajorVersion,
            Minor: rpv.MinorVersion,
            Patch: rpv.PatchVersion,
        }
    }
    return wbc
}
```

### Usage in builder

```go
func buildCallMicroflowGenActivity(wbc *wfBuildCtx, n *ast.WorkflowCallMicroflowNode) element.Element {
    act := genWf.NewCallMicroflowForVersion(wbc.version) // ← type-safe, no strings
    act.SetID(element.ID(types.GenerateID()))
    // ... rest unchanged
}
```

### Threading

`wbc *wfBuildCtx` is threaded through the pure builder chain:

```
execCreateWorkflowGen(ctx)
  wbc := newWfBuildCtx(ctx)
  buildWorkflowActivitiesGen(wbc, nodes)
    buildWorkflowActivityGen(wbc, node)
      buildCallMicroflowGenActivity(wbc, n)     ← uses wbc.version
      buildUserTaskGenActivity(wbc, n)          ← passes wbc to sub-builders
      buildConditionOutcomeGen(wbc, n)          ← passes wbc down
      buildBoundaryEventGen(wbc, be)            ← passes wbc down
      buildParallelSplitGenActivity(wbc, n)     ← passes wbc down
```

`buildAndBindActivitiesGen` in `cmd_alter_workflow.go` receives the same treatment.

Functions that build only leaf nodes with no type-rename relevance (annotation, wait-for-timer, end, jump-to) do not need `wbc` — they have no version-dependent type selection.

---

## Data Flow: Three Scenarios

### Scenario A — Write call microflow, project 11.10.0

```
execCreateWorkflowGen(ctx)
  wbc.version = {11,10,0}
  act := genWf.NewCallMicroflowForVersion({11,10,0})
    → 11.10.0 ≥ 11.9.0 → return NewCallMicroflowActivity()
    → act.TypeName() == "Workflows$CallMicroflowActivity"
  ctx.Backend.CreateWorkflowGen(wf)
    enc := b.newEncoder()  // Version={11,10,0}
    enc.Encode(act)
    → "$Type": "Workflows$CallMicroflowActivity"  ✓
```

### Scenario B — Write call microflow, project 11.6.6

```
  wbc.version = {11,6,6}
  act := genWf.NewCallMicroflowForVersion({11,6,6})
    → 11.6.6 < 11.9.0 → return NewCallMicroflowTask()
    → act.TypeName() == "Workflows$CallMicroflowTask"
    → "$Type": "Workflows$CallMicroflowTask"  ✓
```

### Scenario C — Write boundaryEvents property, project 10.13.0

```
  enc := b.newEncoder()  // Version={10,13,0}
  enc.Encode(callMicroflowAct)
    shouldEmitProperty("Workflows$CallMicroflowTask", "BoundaryEvents")
      → VersionInfos[...].Properties["boundaryEvents"].Introduced = "10.14.0"
      → IsAvailableIn({10,13,0}) = false
      → skip field  ✓
```

### Scenario D — Read any MPR (unchanged)

```
  codec registry:
    "Workflows$CallMicroflowTask"     → initCallMicroflowActivity()
    "Workflows$CallMicroflowActivity" → initCallMicroflowActivity() + override TypeName
  Both decode into *CallMicroflowActivity  ✓  (no change)
```

---

## Testing Strategy

### Unit tests (no MPR)

| Test | Asserts |
|------|---------|
| `TestNewCallMicroflowForVersion_Modern` | `NewCallMicroflowForVersion({11,9,0}).TypeName() == "Workflows$CallMicroflowActivity"` |
| `TestNewCallMicroflowForVersion_Legacy` | `NewCallMicroflowForVersion({11,8,0}).TypeName() == "Workflows$CallMicroflowTask"` |
| `TestEncoder_SkipsIntroducedAfterVersion` | property with `Introduced:"10.14.0"` not emitted when `Encoder.Version={10,13,0}` |
| `TestEncoder_EmitsWhenVersionSufficient` | same property emitted when `Encoder.Version={10,14,0}` |
| `TestEncoder_NoVersionGating` | `Encoder{}` (zero version) emits all properties |
| `TestCodegen_TypeRenamesValidation` | codegen fatals when `new.introduced != old.deleted` |

### Integration tests (MPR corpus)

| Test | Asserts |
|------|---------|
| `TestBuildCallMicroflowGenActivity_TypeName_11_10` | executor writes `CallMicroflowActivity` into a 11.10.0 MPR |
| `TestBuildCallMicroflowGenActivity_TypeName_11_6` | executor writes `CallMicroflowTask` into a 11.6.6 MPR |
| `TestRoundtrip_CallMicroflow_LegacyMPR` | read `CallMicroflowTask` from 11.6.6 MPR → describe → re-write → no corruption |

---

## Migration Path

1. **Add `type_renames` to `supplements.json`** — one entry.
2. **Extend `emitter.TypeData`** with `ClassIntroduced`/`ClassDeleted`; update template to fix `initCallMicroflowActivity()` TypeName.
3. **Generate `NewCallMicroflowForVersion`** from template; run `make grammar` to regenerate.
4. **Add `Version` to `codec.Encoder`**; update `newEncoderForGenSerialize()` and `MprBackend` callers.
5. **Add `wfBuildCtx`** and thread through workflow builder chain.
6. **Write failing tests** before each step; verify green after.

Steps 2–3 fix the immediate bug. Steps 4–5 add complete property gating. All steps are independently reviewable.

---

## Decisions

1. **Property gating default on unknown type:** write it silently. Missing version info = no restriction. Detection belongs in `make audit` (codegen audit mode), not runtime.
2. **`wfBuildCtx` scope:** workflow-only for now (YAGNI). Promote to shared `buildCtx` when a second domain gains a rename — concrete need drives the refactor.
3. **Version boundary:** use **11.9.0** (SDK-authoritative). `CallMicroflowTask.versionInfo.deleted = "11.9.0"` is the Mendix-maintained source of truth. Conservative fallback to 11.10.0 is a one-constant change if 11.9.x evidence contradicts the SDK.

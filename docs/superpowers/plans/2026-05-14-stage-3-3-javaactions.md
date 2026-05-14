# Stage 3.3 JavaActions Domain — Detailed Sub-Plan

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans`. Steps use checkbox `- [ ]` syntax for trackability.
>
> Generated from `2026-05-14-stage-3-3-domain-marathon-master.md` §6 priority #2 + §4 phase template + §8.2 stub. Builds on the security plan's playbook (`2026-05-14-stage-3-3-security.md`) and inherits the Stage 3.2 + Stage 3.3.1 infrastructure (`mprread.ListUnitsByType`, `helpers_gen_container.go`, `ByNameRefList` versioned-array fix).

**Goal:** Migrate the `javaactions` domain off the legacy hand-written `sdk/javaactions` package onto the auto-generated `modelsdk/gen/javaactions` types. Final state: zero `sdk/javaactions` imports outside `sdk/mpr/` (Stage 4 territory) and zero in `modelsdk/`, `api/`, `mdl/`.

---

## §1 Background — Status Snapshot

The `javaactions` domain is on **medium-difficulty** — smaller than security in raw call density (6 executor sites vs. security's 16) but with a **richer polymorphic type tree**: `CodeActionReturnType` and `CodeActionParameterType` interfaces (sdk side) map to **multiple gen-side types** (`StringType`, `BooleanType`, `EntityType`/`ConcreteEntityType`, `EntityTypeParameterType`, `TypeParameter`/`ParameterizedEntityType`, `ListType`, `EnumerationType`, etc.).

Crucially, the `JavaScriptActions$JavaScriptAction` document **shares** the same parameter/return type tree (`CodeActions$*` storage types) as JavaAction. `mdl/types/java.go::JavaScriptAction` currently inlines `[]*javaactions.JavaActionParameter` and `javaactions.CodeActionReturnType` from the legacy sdk. Migrating javaactions therefore unavoidably touches the JS side as well — but Studio Pro models the runtime in `modelsdk/gen/javascriptactions/` (sibling gen package), so the migration is symmetric, not a Stage-4 dependency.

### Already migrated — DO NOT redo

- `mdl/backend/mpr/create_services_modelsdk.go::createJavaActionViaModelsdk` — wraps `mpr.SerializeJavaAction(ja)` and writes via `b.msdkWriter.InsertUnit`. **Note**: this still serializes from `*sdk/javaactions.JavaAction`; it is "modelsdk write path" only in the sense that the *insert* uses `msdkWriter`. The serialize source is still legacy. Truly retiring `sdk/javaactions` requires replacing this with a gen-typed serializer (covered in this plan, Phase D).
- `mdl/backend/mpr/update_services_modelsdk.go::updateJavaActionViaModelsdk` — gen-native scalar update (Name, Documentation, Excluded, ExportLevel, ActionDefaultReturnName). PartList children (Parameters, TypeParameters, ActionReturnType, JavaReturnType, MicroflowActionInfo) preserved by LazyDoc but **NOT mutated** — those are the gap this plan must close (Phase D).
- `mdl/backend/mpr/services_modelsdk_test.go` — has a `&javaactions.JavaAction{...}` fixture for the existing scalar-update test. Will be migrated to gen builder in Phase C.
- `modelsdk/gen/javaactions/{types,enums,refs,version}.go` — auto-generated; complete. **Verified accessor parity for JavaAction and JavaActionParameter** (see §2.4 below).
- `modelsdk/gen/javascriptactions/{types,enums,refs,version}.go` — auto-generated; complete. JS side has parallel API.
- `mprread.ListUnitsByType[T]` — the generic lister landed before Stage 3.3.1; usable directly for `JavaActions$JavaAction` and `JavaScriptActions$JavaScriptAction`.

### Still to migrate (this plan's scope)

| File | LoC | What stays | What leaves |
|---|---|---|---|
| `sdk/javaactions/javaactions.go` | 269 | nothing | full package (after E3) |
| `mdl/types/java.go` | 54 | `JavaAction` (lite descriptor) — already pure | `JavaScriptAction` re-types params/return to gen |
| `mdl/backend/java.go` | 25 | interface methods | `*javaactions.*` from method signatures |
| `mdl/backend/mpr/backend.go` | 881 | shim wrappers | `*javaactions.*` types in 8 method signatures |
| `mdl/backend/mpr/create_services_modelsdk.go` | 279 | InsertUnit wrapper | `*javaactions.JavaAction` parameter (replace with gen) |
| `mdl/backend/mpr/update_services_modelsdk.go` | 278 | scalar setters | `*javaactions.JavaAction` parameter |
| `mdl/backend/mpr/java_files.go` | 60 | path-based file I/O | `*javaactions.JavaActionParameter`, `javaactions.CodeActionReturnType` in func sig |
| `mdl/backend/mpr/services_modelsdk_test.go` | 686 | infrastructure | one `&javaactions.JavaAction{}` literal |
| `mdl/backend/mock/backend.go` | 309 | Func-field decls | `*javaactions.*` in 5 Func types |
| `mdl/backend/mock/mock_java.go` | 93 | shim layer | `*javaactions.*` in 5 method signatures |
| `mdl/catalog/builder.go` | 585 | reader interface | one method `ListJavaActionsFull() ([]*javaactions.JavaAction, error)` |
| `mdl/executor/cmd_javaactions.go` | 667 | nothing | full file |
| `mdl/executor/cmd_javascript_actions.go` | 278 | nothing | full file |
| `mdl/executor/cmd_javaactions_mock_test.go` | 143 | needs migration | 3 `*javaactions.JavaAction` fixtures |
| `mdl/executor/validate_system_javaaction_test.go` | 91 | needs migration | 1 `*javaactions.JavaAction` fixture |
| `mdl/executor/cmd_structure.go` | 751 | structure helpers | `outputJavaActions(ctx, mod, []*javaactions.JavaAction, …)` signature + body |
| `mdl/executor/cmd_structure_gen.go` | 509 | wf/mf helpers | `structureJaMapGen` + ListJavaActionsFull call |
| `sdk/mpr/parser_javaactions.go` | (Stage 4 territory) | — | DO NOT touch |
| `sdk/mpr/parser_javaactions_test.go` | (Stage 4 territory) | — | DO NOT touch |
| `sdk/mpr/parser_misc.go` | (Stage 4 territory) | — | DO NOT touch (parses JavaScriptAction) |
| `sdk/mpr/serialize_exports.go` | (Stage 4 territory) | — | DO NOT touch (re-exports `SerializeJavaAction`) |
| `sdk/mpr/system_java_actions.go` | (Stage 4 territory) | — | DO NOT touch |
| `sdk/mpr/writer_javaactions.go` | (Stage 4 territory) | — | DO NOT touch |

### Why this domain is priority #2

- **Small surface in mdl/executor (6 callers)** — half of security's caller count
- **Self-contained** — no domainmodel cross-dep at write time (entity/list types reference qualified names, not `*domainmodel.Entity`)
- **Gen accessor parity is high** — see §2.4: `Name`/`Documentation`/`Excluded`/`ExportLevel`/`ActionDefaultReturnName` map 1:1; the polymorphic types map cleanly with `element.Element` discrimination
- **Master plan explicitly carved out Java-FS calls** (`docs/.../master.md` §8.2): Java source-file I/O is path-based and is **not** in `sdk/mpr` write path; we keep `mdl/backend/mpr/java_files.go` but drop the `sdk/javaactions` types from its signature
- **Cache helpers and `mprread` already in place** from Stage 3.2 + 3.3.1; no new infra task needed

---

## §2 Pre-Flight Survey Results

### S2.1 sdk/javaactions importers (full list, per `grep -rln`)

```
mdl/backend/java.go
mdl/backend/mock/backend.go
mdl/backend/mock/mock_java.go
mdl/backend/mpr/backend.go
mdl/backend/mpr/create_services_modelsdk.go
mdl/backend/mpr/java_files.go
mdl/backend/mpr/services_modelsdk_test.go
mdl/backend/mpr/update_services_modelsdk.go
mdl/catalog/builder.go
mdl/executor/cmd_javaactions.go                 ← 32 references
mdl/executor/cmd_javaactions_mock_test.go        ← 3
mdl/executor/cmd_javascript_actions.go           ← 2
mdl/executor/cmd_structure.go                    ← 3
mdl/executor/cmd_structure_gen.go                ← 1 (import) + 2 type refs
mdl/executor/validate_system_javaaction_test.go  ← 1
mdl/types/java.go                                ← 4 refs
sdk/mpr/parser_javaactions.go                   ← Stage 4 territory
sdk/mpr/parser_javaactions_test.go              ← Stage 4 territory
sdk/mpr/parser_misc.go                          ← Stage 4 (JavaScriptAction parser)
sdk/mpr/serialize_exports.go                    ← Stage 4 territory
sdk/mpr/system_java_actions.go                  ← Stage 4 territory
sdk/mpr/writer_javaactions.go                   ← Stage 4 territory
```

**In-scope total: 16 files (excluding 6 sdk/mpr files = Stage 4).**

### S2.2 Read funcs in `cmd_javaactions.go` + `cmd_javascript_actions.go`

`cmd_javaactions.go` (legacy):
1. `listJavaActions(ctx, moduleName)` → calls `ctx.Backend.ListJavaActions()` returning `[]*types.JavaAction` (lite). Renders qualifiedName/module/name/folder. Pure read.
2. `describeJavaAction(ctx, name)` → calls `ctx.Backend.ReadJavaActionByName(qn)` returning `*javaactions.JavaAction` (full). Walks Parameters, ReturnType, MicroflowActionInfo; reads Java source via `readJavaActionUserCode`. Pure read.
3. `readJavaActionUserCode(mprPath, mod, name)` (helper) — pure file I/O, no sdk/javaactions dep.
4. `formatJavaActionType(t javaactions.CodeActionParameterType)` — sdk-typed switch.
5. `formatJavaActionReturnType(t javaactions.CodeActionReturnType)` — sdk-typed switch.

`cmd_javascript_actions.go` (legacy):
1. `listJavaScriptActions(ctx, moduleName)` → `ctx.Backend.ListJavaScriptActions()` returning `[]*types.JavaScriptAction`. Renders qualifiedName/module/name/platform/folder.
2. `describeJavaScriptAction(ctx, name)` → `ctx.Backend.ReadJavaScriptActionByName(qn)` returning `*types.JavaScriptAction` (which re-uses `*javaactions.JavaActionParameter` and `javaactions.CodeActionReturnType` for its fields).
3. `formatJavaScriptActionType(t javaactions.CodeActionParameterType)` — same shape as javaactions equivalent.

### S2.3 Write funcs in `cmd_javaactions.go`

1. `execCreateJavaAction(ctx, s)` (lines 285–452) — builds `*javaactions.JavaAction` from AST, including Parameters, TypeParameters, MicroflowActionInfo, ReturnType. Then calls `ctx.Backend.CreateJavaAction(ja)` OR `UpdateJavaAction(ja)` for `OR MODIFY`. Then `WriteJavaSourceFile(...)` with the same sdk-typed params slice + return type.
2. `execDropJavaAction(ctx, s)` — list + filter + `DeleteJavaAction(id)` + `DeleteJavaSourceFile(mod, name)`.

Helpers:
- `astDataTypeToJavaActionParamType(dt) javaactions.CodeActionParameterType` (lines 455–537)
- `astDataTypeToJavaActionReturnType(dt) javaactions.CodeActionReturnType` (lines 540–637)
- `isTypeParamRef(dt, names)` / `getTypeParamRefName(dt)` (lines 642–667) — pure ast helpers, no sdk dep.

JavaScript side has no write commands (read-only domain in MDL).

### S2.4 Gen accessor name map (CRITICAL — verified against `modelsdk/gen/javaactions/types.go`)

**JavaAction (`type JavaAction struct`, line 294):**

| sdk field/method | gen accessor | Notes |
|---|---|---|
| `ja.Name` (string) | `Name() / SetName(string)` | exact |
| `ja.Documentation` | `Documentation() / SetDocumentation` | exact |
| `ja.Excluded` | `Excluded() / SetExcluded` | exact |
| `ja.ExportLevel` | `ExportLevel() / SetExportLevel` | exact |
| `ja.ActionDefaultReturnName` | `ActionDefaultReturnName() / SetActionDefaultReturnName` | exact |
| `ja.Parameters` (`[]*JavaActionParameter`) | **`ActionParametersItems() []element.Element`** + `AddActionParameters` / `RemoveActionParameters` | ALSO `ParametersItems()`, `AddParameters`, `RemoveParameters` for legacy property — see §3 R1 |
| `ja.TypeParameters` (`[]*TypeParameterDef`) | **`ActionTypeParametersItems() []element.Element`** + `AddActionTypeParameters` / `RemoveActionTypeParameters` | ALSO `TypeParametersItems()` legacy property |
| `ja.ReturnType` (`CodeActionReturnType`) | **`ActionReturnType() element.Element / SetActionReturnType`** | ALSO `JavaReturnType()` (legacy) AND `ReturnType() string` (raw type-string scalar — NOT the rich type) |
| `ja.MicroflowActionInfo` (`*MicroflowActionInfo`) | **`ModelerActionInfo() element.Element / SetModelerActionInfo`** | ALSO `MicroflowActionInfo()` (legacy alias) |

**JavaActionParameter (`type JavaActionParameter struct`, line 534):**

| sdk field/method | gen accessor | Notes |
|---|---|---|
| `param.Name` | `Name() / SetName` | exact |
| `param.Description` | `Description() / SetDescription` | exact |
| `param.Category` | `Category() / SetCategory` | exact |
| `param.IsRequired` | `IsRequired() / SetIsRequired` | exact |
| `param.ParameterType` (`CodeActionParameterType`) | **`ActionParameterType() element.Element / SetActionParameterType`** | ALSO `JavaType()`, `ParameterType()` (legacy alias) |

**Polymorphic type tree — sdk vs gen:**

| sdk type | gen type (qualified storage) | Notes |
|---|---|---|
| `VoidType` | (no gen sibling) | Void is "no return type" — handled by absent `ActionReturnType` in gen, OR gen does have `VoidType` as one of the `CodeActions$*` schemas. **Verify with `grep -n VoidType modelsdk/gen/javaactions/types.go`** before Phase A1 |
| `BooleanType` | `BooleanType` | exact |
| `IntegerType` | `IntegerType` | exact |
| `LongType` | (TBD — gen has `IntegerType` but no `LongType` in the listing above) | **Schema gap risk** — verify; may need `setRawBSONField` workaround for `CodeActions$LongType` |
| `DecimalType` | `DecimalType` | exact |
| `StringType` | `StringType` | exact |
| `DateTimeType` | `DateTimeType` | exact |
| `EntityType{Entity string}` | `ConcreteEntityType{EntityQualifiedName()}` | gen splits abstract `EntityType` (interface-ish) from `ConcreteEntityType` (concrete shape with name) |
| `ListType{Entity string}` | `ListType{Parameter() element.Element / SetParameter}` | gen uses sub-`Parameter` element rather than direct field; the entity ref lives inside the Parameter element. **Verify shape in §A1.** |
| `StringTemplateParameterType{Grammar}` | (TBD — not in listing above) | Verify |
| `FileDocumentType` | (TBD) | Verify |
| `EnumerationType{Enumeration string}` | `EnumerationType{EnumerationQualifiedName()}` | exact pattern |
| `MicroflowType` | `MicroflowParameterType` (line 738) and/or `MicroflowJavaActionParameterType` (726) | gen has TWO; pick the one used for JavaAction (likely `MicroflowJavaActionParameterType` per name, but storage-type verification required) |
| `NanoflowType` | (TBD — only used by JavaScriptAction; check `gen/javascriptactions/types.go`) | |
| `TypeParameter{TypeParameter string, TypeParameterID model.ID}` | `TypeParameter{Name() / SetName}` (line 792) **AND** `ParameterizedEntityType{TypeParameterRefID() / SetTypeParameterID}` (line 750) | sdk's "TypeParameter" is overloaded (return type AND param-type). gen splits: `TypeParameter` is the named definition (with `Name()`); `ParameterizedEntityType` is the *use site* in a parameter slot referencing a TypeParameter by ID |
| `TypeParameterDef{Name string}` | `TypeParameter{Name() / SetName}` (line 792) | **Naming flip**: sdk's `TypeParameterDef` (the def) maps to gen's `TypeParameter` (also the def). gen's "TypeParameter" is **not** the use-site; it IS the def. The ParameterizedEntityType holds the BY_ID reference |
| `EntityTypeParameterType{TypeParameterID, TypeParameterName}` | `EntityTypeParameterType{TypeParameterRefID() / SetTypeParameterID}` | gen does NOT carry a `Name`/display field — must look up via the ParametersItems list. Display-name resolution is now lookup-by-ID, not stored |
| `MicroflowActionInfo{Caption, Category, Icon, ImageData}` | `MicroflowActionInfo{Caption(), Category(), IconQualifiedName()}` | **Schema gap risk**: no `ImageData` accessor — verify. Icon is now `IconQualifiedName()` (qname ref, not free string) |

**TBD entries above MUST be verified by running `grep -n "type.*Type struct\|type Microflow.*type\|type Nanoflow.*type\|VoidType\|LongType\|StringTemplate\|FileDocument" modelsdk/gen/javaactions/types.go modelsdk/gen/javascriptactions/types.go` at the start of Phase A. If any gen type is missing, file a `gen_schema_gaps` note and use `codec.ReadBSONFieldString(elem.Raw(), "$Type")` discrimination as the fallback.**

### S2.5 Backend interface methods (`mdl/backend/java.go`)

```go
JavaBackend:
  ListJavaActions() ([]*types.JavaAction, error)               // lite — no sdk/javaactions dep
  ListJavaActionsFull() ([]*javaactions.JavaAction, error)     // sdk-typed — REPLACE
  ListJavaScriptActions() ([]*types.JavaScriptAction, error)   // types still re-uses sdk via JavaScriptAction.ReturnType etc.
  ReadJavaActionByName(qn string) (*javaactions.JavaAction, error)        // sdk-typed — REPLACE
  ReadJavaScriptActionByName(qn) (*types.JavaScriptAction, error)         // types still re-uses sdk
  CreateJavaAction(ja *javaactions.JavaAction) error                       // sdk-typed — REPLACE
  UpdateJavaAction(ja *javaactions.JavaAction) error                       // sdk-typed — REPLACE
  DeleteJavaAction(id model.ID) error                                      // ok (just ID)
  WriteJavaSourceFile(mod, action, code string, params []*javaactions.JavaActionParameter, returnType javaactions.CodeActionReturnType, …) error  // sdk-typed — REPLACE
  DeleteJavaSourceFile(mod, action) error                                  // ok
  RenameJavaSourceFile(mod, oldName, newName) error                        // ok
  ReadJavaSourceFile(mod, action) (string, error)                          // ok
```

The **8 methods** that mention `*javaactions.*` need parallel `*Gen` siblings (additive in Phase C, retire in Phase E).

### S2.6 ExecContext field

`ExecContext` (mdl/executor/exec_context.go:27–107) currently exposes `Microflows`, `Nanoflows`, `Security` repos. **There is no `ctx.JavaActions` field.** Pattern from Stage 3.3.1 §A0 is to add it during Phase A.

### S2.7 MockBackend conformance

`mdl/backend/mock/mock_java.go` returns `nil, nil` (or `nil`) on missing Func fields, NOT `"MockBackend.X not configured"` errors. Same pattern violation as security (master plan §5 P3). This sub-plan task C5 brings the mock funcs into compliance.

### S2.8 Existing repos infrastructure

No `mdl/repos/javaactions.go` or `mdl/backend/mpr/repos/javaactions.go` exist. Pattern from `mdl/repos/microflows.go` + `mdl/backend/mpr/repos/microflows.go` MUST be cloned (Phase A0).

### S2.9 Storage type names verified

- `JavaActions$JavaAction` — confirmed at `sdk/mpr/parser_javaactions.go:17`
- `JavaScriptActions$JavaScriptAction` — confirmed at `sdk/mpr/parser_misc.go:202`
- `CodeActions$VoidType`, `CodeActions$BooleanType`, `CodeActions$IntegerType`, `CodeActions$LongType`, `CodeActions$DecimalType`, `CodeActions$StringType`, `CodeActions$DateTimeType`, `CodeActions$EntityType`, `CodeActions$ListType` — confirmed from `cmd_javaactions.go:455–637` (sdk write path uses these as `TypeName`)

### S2.10 Container linkage

JavaAction units live under module containment. The container UUID is the module ID — verified by `cmd_javaactions.go:44 (h.FindModuleID(ja.ContainerID))`. Codec-decoded gen `*JavaAction` does NOT carry `ContainerID` (per §3.7 of master plan). Resolution path: same as microflows — `ctx.JavaActions.GetContainerUUID(id)` after building cache helper.

---

## §3 Risks Specific to JavaActions Domain

| # | Risk | Impact | Mitigation |
|---|---|---|---|
| R1 | **Dual accessor families on JavaAction**: gen has both `ParametersItems()` AND `ActionParametersItems()`, both `JavaReturnType()` AND `ActionReturnType()`. Picking the wrong one silently returns the wrong (or empty) collection because LazyDoc preserves both BSON keys but only ONE is truly canonical | High — silent data loss | A1 verification step: write a smoke test that opens an existing project with a Java action that has ≥1 parameter. Assert which accessor returns non-empty. Pin in helper comments. **Hypothesis**: `ActionParametersItems` / `ActionReturnType` / `ActionParameterType` (the longer names) are the canonical Mendix property names; the shorter ones are auto-generated aliases for legacy storage keys. **Verify before any A1 commit.** |
| R2 | **Polymorphic type discrimination gap**: gen returns `element.Element` for `ActionReturnType()` and `ActionParameterType()`. Distinguishing `*StringType` vs `*EntityType` vs `*ConcreteEntityType` requires either a Go type switch (works only if the gen package registered the concrete struct) OR `elem.TypeName()` string discrimination (works always but less compile-safe) | Medium | Use `elem.TypeName()` switch keyed on `"CodeActions$BooleanType"` etc. as the primary path, fall back to type-assertion when gen has the concrete type registered. Document in helper file a table mapping storage names → gen types |
| R3 | **`MicroflowActionInfo.ImageData` schema gap**: gen `MicroflowActionInfo` has only `Caption`, `Category`, `IconQualifiedName` — sdk has `Icon (string)` + `ImageData`. Studio Pro stores embedded images here; if we drop the ImageData field on roundtrip, custom toolbox icons are lost | Medium | Phase D Verify by `mx check` round-trip on a fixture with a custom icon. If gap confirmed, use raw-BSON read fallback `codec.ReadBSONFieldString(info.Raw(), "ImageData")` when reading; for write, use `setRawBSONField`-style workaround until gen narrows |
| R4 | **`VoidType` / `LongType` / `StringTemplateParameterType` / `FileDocumentType` / `NanoflowType` may not exist in gen**: only the types listed in §S2.4 are confirmed. The unverified set could trigger silent fallback in Phase A formatters | Medium | A1 first step: run the verification grep (§S2.4 last paragraph). Update the type table accordingly. If a type IS missing, use `elem.TypeName()` raw-string switch to render the right syntax in `formatJavaActionTypeGen` |
| R5 | **TypeParameter naming flip** between sdk `TypeParameterDef` ↔ gen `TypeParameter` (§S2.4). Confusing both in code can lead to misnaming type parameters | Low — caught at compile time | Pin in a comment block in helpers and execCreateJavaActionGen; add a smoke test that round-trips a Java action with `<pEntity>` parameter |
| R6 | **`ListType` shape inversion**: sdk `ListType{Entity string}` is flat; gen `ListType{Parameter() element.Element}` wraps a sub-element (`*ConcreteEntityType` or similar) | Medium | A1 formatter walks `lt.Parameter()` and dispatches via `TypeName()` to render `"List of <qname>"`. Write helper added in A1 |
| R7 | **`EntityTypeParameterType` lost `TypeParameterName` field** (sdk had it for display). Gen requires lookup-by-ID via `TypeParameterRefID()` | Low | Pass the action's `ActionTypeParametersItems()` (the def list) into the formatter so it can resolve ID → Name |
| R8 | **JavaScriptAction parser still in `sdk/mpr/parser_misc.go` (Stage 4)**: the gen `javascriptactions` package exists but the read path goes through `sdk/mpr.parseJavaScriptAction` which builds a `*sdk/mpr.JavaScriptAction` (sdk-typed). Until Stage 4 lands, the JS read path can't go fully gen-native | High coordination cost | Phase A2 (JavaScript): use `mprread.ListUnitsByType[*genJSA.JavaScriptAction](r, "JavaScriptActions$JavaScriptAction")` directly, bypassing the sdk reader. Documented in master plan §3.7 (container linkage) — same pattern. The legacy `b.reader.ListJavaScriptActions()` stays in `sdk/mpr` for backward compat but is no longer called from `mdl/` after Phase A |
| R9 | **`mdl/types/java.go::JavaScriptAction` is currently a re-typed wrapper of sdk types**. Cannot just delete the sdk import without restructuring `JavaScriptAction` itself | High blast radius (catalog, executor consumers) | C1 dedicated step: redefine `types.JavaScriptAction` with a self-contained shape (`Parameters []*types.CodeActionParameter`, etc.) OR retire it entirely in favor of gen `*genJSA.JavaScriptAction`. The "retire" path is preferred to avoid maintaining a third type tree. Decision deferred to a §6 task; default = retire and use gen directly |
| R10 | **Stage 4 boundary at sdk/mpr**: 6 files in `sdk/mpr/` import `sdk/javaactions`. Stage 4 owns these and is actively rewriting them | Medium | Acceptance grep in §8 explicitly excludes `^./sdk/mpr/` matches. E3 deletion of `sdk/javaactions/` is conditional on Stage 4 having removed the sdk/mpr importers; otherwise E3 holds with deprecation header. Same escape hatch as security plan §11 |

---

## §4 Phase A — Read Path Migration

Goal: every read function in `cmd_javaactions.go` and `cmd_javascript_actions.go` has a `*Gen` twin reading from `ctx.JavaActions` / `ctx.JavaScriptActions` (via `mprread.ListUnitsByType`), and the dispatcher routes to the gen variants.

### Task A0: `mdl/repos/javaactions.go` + `mdl/backend/mpr/repos/javaactions.go` + cache helpers

Set up the repository pattern (no MicroflowRepository-like writer for Phase A; reader-only initially, then extend in Phase D).

**Files:**
- Create: `mdl/repos/javaactions.go` (interface)
- Create: `mdl/backend/mpr/repos/javaactions.go` (direct-mode implementation)
- Create: `mdl/backend/mpr/repos/javaactions_test.go`
- Create: `mdl/executor/helpers_javaactions_gen.go`
- Create: `mdl/executor/helpers_javaactions_gen_test.go`
- Modify: `mdl/executor/exec_context.go` — add `JavaActions repos.JavaActionRepository`, `JavaScriptActions repos.JavaScriptActionRepository` fields
- Modify: `mdl/executor/executor.go` — add `javaActionsWithContainerGen []ContainerWithGen[*genJA.JavaAction]`, `javaScriptActionsWithContainerGen []ContainerWithGen[*genJSA.JavaScriptAction]` to `executorCache`
- Modify: BackendFactory wiring (locate in `executor.go` or wherever `Microflows`/`Nanoflows`/`Security` are wired)

#### A0.S1: Verify gen accessor surface (one-off pre-flight, NO commit)

```bash
grep -n "type.*Type struct\|VoidType\|LongType\|StringTemplate\|FileDocument\|NanoflowType\|MicroflowType" modelsdk/gen/javaactions/types.go modelsdk/gen/javascriptactions/types.go > /tmp/javaactions-gen-types.txt
cat /tmp/javaactions-gen-types.txt
```

Update the §2.4 table in this plan with the actual accessor names if any TBD turns out different.

#### A0.S2: Write failing test for repo + cache helper

```go
// mdl/backend/mpr/repos/javaactions_test.go
func TestJavaActionRepository_ListAll_DecodesGenTypes(t *testing.T) {
    w, cleanup := openFixtureWriter(t, "testdata/javaactions-fixture.mpr") // create fixture as part of A0
    defer cleanup()
    repo := NewJavaActionRepository(w)
    actions, err := repo.ListAll()
    if err != nil { t.Fatalf("ListAll: %v", err) }
    if len(actions) == 0 { t.Fatal("ListAll returned empty") }
    if actions[0].Name() == "" { t.Errorf("Name() empty") }
}

func TestJavaActionRepository_GetContainerUUID_Resolves(t *testing.T) {
    w, cleanup := openFixtureWriter(t, "testdata/javaactions-fixture.mpr")
    defer cleanup()
    repo := NewJavaActionRepository(w)
    actions, _ := repo.ListAll()
    cid, err := repo.GetContainerUUID(model.ID(actions[0].ID()))
    if err != nil { t.Fatalf("GetContainerUUID: %v", err) }
    if cid == "" { t.Errorf("ContainerUUID empty") }
}
```

Pattern is identical to `mdl/backend/mpr/repos/microflows_test.go`. **Use an EXISTING fixture from `mdl/backend/mpr/repos/testdata/`** if one with Java actions exists; otherwise create a minimal one via the fixture-builder helper used by other repos.

```go
// mdl/executor/helpers_javaactions_gen_test.go
func TestListJavaActionsWithContainerGen_CachesAcrossCalls(t *testing.T) {
    ctx := newJavaActionsTestContext(t)
    list1, err := listJavaActionsWithContainerGen(ctx)
    if err != nil { t.Fatalf("listJavaActionsWithContainerGen: %v", err) }
    list2, _ := listJavaActionsWithContainerGen(ctx)
    if len(list1) != len(list2) {
        t.Fatalf("cache produced different lengths: %d vs %d", len(list1), len(list2))
    }
}
```

#### A0.S3: Confirm RED

```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/backend/mpr/repos/ -run TestJavaActionRepository_ -v
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/executor/ -run TestListJavaActionsWithContainerGen_CachesAcrossCalls -v
```
Expected: FAIL (`undefined: NewJavaActionRepository / listJavaActionsWithContainerGen`).

#### A0.S4: Implement repo + helpers

```go
// mdl/repos/javaactions.go
package repos

import (
    "github.com/mendixlabs/mxcli/model"
    genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
    genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
)

type JavaActionReader interface {
    Get(id model.ID) (*genJA.JavaAction, error)
    List(moduleID model.ID) ([]*genJA.JavaAction, error)
    ListAll() ([]*genJA.JavaAction, error)
    FindByQualifiedName(qn string) (*genJA.JavaAction, error)
    GetContainerUUID(id model.ID) (model.ID, error)
}

type JavaActionWriter interface {
    Create(parentUUID, containmentName string, ja *genJA.JavaAction) error
    Update(ja *genJA.JavaAction) error
    Delete(id model.ID) error
}

type JavaActionRepository interface {
    JavaActionReader
    JavaActionWriter
}

type JavaScriptActionReader interface {
    Get(id model.ID) (*genJSA.JavaScriptAction, error)
    List(moduleID model.ID) ([]*genJSA.JavaScriptAction, error)
    ListAll() ([]*genJSA.JavaScriptAction, error)
    FindByQualifiedName(qn string) (*genJSA.JavaScriptAction, error)
    GetContainerUUID(id model.ID) (model.ID, error)
}

type JavaScriptActionRepository interface {
    JavaScriptActionReader
    // No writer for JS — MDL has no `create javascript action` syntax today.
}
```

```go
// mdl/backend/mpr/repos/javaactions.go (skeleton; mirrors microflows.go)
package mprrepos

import (
    "fmt"
    "github.com/mendixlabs/mxcli/mdl/repos"
    "github.com/mendixlabs/mxcli/model"
    genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
    mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const javaActionTypeName = "JavaActions$JavaAction"

type javaActionRepo struct {
    w   *mmpr.Writer
    r   *mmpr.Reader
    dec *decoder
}

func NewJavaActionRepository(w *mmpr.Writer) repos.JavaActionRepository {
    return &javaActionRepo{
        w:   w,
        r:   w.ConcreteReader(),
        dec: newDecoder(),
    }
}

func (r *javaActionRepo) Get(id model.ID) (*genJA.JavaAction, error) {
    bytes, err := r.r.GetRawUnitBytes(string(id))
    if err != nil { return nil, err }
    if len(bytes) == 0 { return nil, fmt.Errorf("java action not found: %s", id) }
    elem, err := r.dec.Decode(bytes)
    if err != nil { return nil, fmt.Errorf("decode java action %s: %w", id, err) }
    ja, ok := elem.(*genJA.JavaAction)
    if !ok { return nil, fmt.Errorf("unit %s is not a JavaAction (got %T)", id, elem) }
    return ja, nil
}

func (r *javaActionRepo) ListAll() ([]*genJA.JavaAction, error) {
    return mprread.ListUnitsByType[*genJA.JavaAction](r.r, javaActionTypeName)
}

func (r *javaActionRepo) List(moduleID model.ID) ([]*genJA.JavaAction, error) {
    all, err := r.ListAll()
    if err != nil { return nil, err }
    if moduleID == "" { return all, nil }
    // Same module-by-name filter pattern as microflows.List(moduleID)
    // ... (copy from mdl/backend/mpr/repos/microflows.go::List)
}

func (r *javaActionRepo) FindByQualifiedName(qn string) (*genJA.JavaAction, error) {
    // Pattern: split qn into "Module.Name", iterate ListAll, match name + container module name
    // Mirror microflows.FindByQualifiedName
}

func (r *javaActionRepo) GetContainerUUID(id model.ID) (model.ID, error) {
    cid, err := r.r.GetUnitContainerID(string(id))
    if err != nil { return "", err }
    return model.ID(cid), nil
}

// Writer methods stub to fmt.Errorf("not implemented in Phase A"); Phase D fills them.
func (r *javaActionRepo) Create(...) error { return fmt.Errorf("Create not implemented (Phase D)") }
func (r *javaActionRepo) Update(...) error { return fmt.Errorf("Update not implemented (Phase D)") }
func (r *javaActionRepo) Delete(id model.ID) error { return fmt.Errorf("Delete not implemented (Phase D)") }

var _ repos.JavaActionRepository = (*javaActionRepo)(nil)
```

Same shape for `javaScriptActionRepo` (read-only, no writer). Storage type: `JavaScriptActions$JavaScriptAction`.

```go
// mdl/executor/helpers_javaactions_gen.go
package executor

import (
    "github.com/mendixlabs/mxcli/model"
    genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
    genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
)

func listJavaActionsWithContainerGen(ctx *ExecContext) ([]ContainerWithGen[*genJA.JavaAction], error) {
    if ctx == nil || ctx.JavaActions == nil { return nil, nil }
    return listUnitsWithContainerGen(
        func() ([]*genJA.JavaAction, error) { return ctx.JavaActions.ListAll() },
        func(id element.ID) (element.ID, error) {
            c, err := ctx.JavaActions.GetContainerUUID(model.ID(id))
            return element.ID(c), err
        },
        func() ([]ContainerWithGen[*genJA.JavaAction], bool) {
            if ctx.Cache != nil && ctx.Cache.javaActionsWithContainerGen != nil {
                return ctx.Cache.javaActionsWithContainerGen, true
            }
            return nil, false
        },
        func(s []ContainerWithGen[*genJA.JavaAction]) {
            if ctx.Cache != nil { ctx.Cache.javaActionsWithContainerGen = s }
        },
    )
}

// listJavaScriptActionsWithContainerGen (mirror)

func invalidateJavaActionsCache(ctx *ExecContext) {
    if ctx == nil || ctx.Cache == nil { return }
    ctx.Cache.javaActionsWithContainerGen = nil
}

func invalidateJavaScriptActionsCache(ctx *ExecContext) {
    if ctx == nil || ctx.Cache == nil { return }
    ctx.Cache.javaScriptActionsWithContainerGen = nil
}
```

#### A0.S5: Wire ctx.JavaActions and ctx.JavaScriptActions in BackendFactory

Find the constructor that assigns `Microflows`/`Nanoflows`/`Security` (search `ctx.Microflows = mprrepos.NewMicroflowRepository`); add the two parallel lines.

#### A0.S6: GREEN + build + full test gate

```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./...
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./... -count=1 -timeout 240s
```

#### A0.S7: Commit

```bash
git add mdl/repos/javaactions.go mdl/backend/mpr/repos/javaactions.go mdl/backend/mpr/repos/javaactions_test.go \
        mdl/executor/exec_context.go mdl/executor/executor.go \
        mdl/executor/helpers_javaactions_gen.go mdl/executor/helpers_javaactions_gen_test.go
git commit -m "$(cat <<'EOF'
feat(executor,repos): Stage 3.3.2.A0 — javaactions repo + cache helpers + ctx wiring

Adds mdl/repos/javaactions.go (JavaActionRepository, JavaScriptActionRepository)
with reader-only methods (writer stubs return "not implemented (Phase D)").
Direct-mode implementation in mdl/backend/mpr/repos/javaactions.go uses
mprread.ListUnitsByType for listing and r.GetUnitContainerID for container
linkage.

Wires ExecContext.JavaActions and ExecContext.JavaScriptActions, plus the
listJavaActionsWithContainerGen / listJavaScriptActionsWithContainerGen
cache helpers.

Mirrors the security/microflow pattern from Stage 3.2 / Stage 3.3.1.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task A1: `listJavaActionsGen`

**Files:**
- Modify: `mdl/executor/cmd_javaactions_gen.go` (NEW)
- Modify: `mdl/executor/cmd_javaactions_gen_test.go` (NEW)

#### A1.S1: Failing test

```go
// mdl/executor/cmd_javaactions_gen_test.go
func TestListJavaActionsGen_OutputsQualifiedName(t *testing.T) {
    ctx := newJavaActionsTestContext(t)
    var buf bytes.Buffer
    ctx.Output = &buf
    ctx.Format = FormatTable
    if err := listJavaActionsGen(ctx, ""); err != nil {
        t.Fatalf("listJavaActionsGen: %v", err)
    }
    if !strings.Contains(buf.String(), "Qualified Name") {
        t.Errorf("expected header 'Qualified Name' in output: %q", buf.String())
    }
    if !strings.Contains(buf.String(), "java actions)") {
        t.Errorf("expected count summary in output: %q", buf.String())
    }
}
```

#### A1.S2: RED

```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/executor/ -run TestListJavaActionsGen_OutputsQualifiedName -v
```

#### A1.S3: Implement

```go
// mdl/executor/cmd_javaactions_gen.go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
    "fmt"
    "sort"
    "strings"

    mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

// listJavaActionsGen handles SHOW JAVA ACTIONS using gen-typed JavaAction
// units from listJavaActionsWithContainerGen. Mirrors listJavaActions in
// output shape; only the type source changes.
func listJavaActionsGen(ctx *ExecContext, moduleName string) error {
    h, err := getHierarchy(ctx)
    if err != nil { return mdlerrors.NewBackend("build hierarchy", err) }

    pairs, err := listJavaActionsWithContainerGen(ctx)
    if err != nil { return mdlerrors.NewBackend("list java actions", err) }

    type row struct {
        qualifiedName, module, name, folderPath string
    }
    var rows []row
    for _, p := range pairs {
        if p.Elem == nil { continue }
        modID := h.FindModuleID(model.ID(p.ContainerID))
        modName := h.GetModuleName(modID)
        if moduleName != "" && modName != moduleName { continue }
        qn := modName + "." + p.Elem.Name()
        folder := h.BuildFolderPath(model.ID(p.ContainerID))
        rows = append(rows, row{qn, modName, p.Elem.Name(), folder})
    }
    sort.Slice(rows, func(i, j int) bool {
        return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
    })

    result := &TableResult{
        Columns: []string{"Qualified Name", "Module", "Name", "Folder"},
        Summary: fmt.Sprintf("(%d java actions)", len(rows)),
    }
    for _, r := range rows {
        result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.folderPath})
    }
    return writeResult(ctx, result)
}
```

#### A1.S4: GREEN
#### A1.S5: Build + full test gate
#### A1.S6: Commit

```bash
git add mdl/executor/cmd_javaactions_gen.go mdl/executor/cmd_javaactions_gen_test.go
git commit -m "$(cat <<'EOF'
feat(executor): Stage 3.3.2.A1 — listJavaActionsGen (gen-typed)

Mirrors listJavaActions reading via ctx.JavaActions /
listJavaActionsWithContainerGen instead of ctx.Backend.ListJavaActions.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task A2: `formatJavaActionTypeGen` + `formatJavaActionReturnTypeGen` (polymorphic dispatch)

The hardest read-path piece. Replaces the sdk-typed `formatJavaActionType` (cmd_javaactions.go:226) and `formatJavaActionReturnType` (cmd_javaactions.go:241) with versions that dispatch on `elem.TypeName()` and walk gen-typed sub-elements.

**Files:**
- Modify: `mdl/executor/cmd_javaactions_gen.go` (extend)
- Modify: `mdl/executor/cmd_javaactions_gen_test.go` (extend)

#### A2.S1: Failing tests (cover the major polymorphism cases)

```go
func TestFormatJavaActionTypeGen_BooleanType(t *testing.T) {
    bt := &genJA.BooleanType{} // requires NewBooleanType-equivalent
    bt.SetTypeName("CodeActions$BooleanType")
    if got := formatJavaActionTypeGen(bt, nil); got != "Boolean" {
        t.Errorf("got %q, want Boolean", got)
    }
}

func TestFormatJavaActionTypeGen_EntityType(t *testing.T) {
    et := genJA.NewConcreteEntityType()
    et.SetEntityQualifiedName("Sales.Customer")
    if got := formatJavaActionTypeGen(et, nil); got != "Sales.Customer" {
        t.Errorf("got %q, want Sales.Customer", got)
    }
}

func TestFormatJavaActionTypeGen_ListType_WithEntity(t *testing.T) {
    inner := genJA.NewConcreteEntityType()
    inner.SetEntityQualifiedName("Sales.Order")
    lt := genJA.NewListType()
    lt.SetParameter(inner)
    if got := formatJavaActionTypeGen(lt, nil); got != "List of Sales.Order" {
        t.Errorf("got %q, want List of Sales.Order", got)
    }
}

func TestFormatJavaActionTypeGen_EntityTypeParameterType_ResolvesName(t *testing.T) {
    tp := genJA.NewTypeParameter()
    tp.SetName("pEntity")
    etp := genJA.NewEntityTypeParameterType()
    etp.SetTypeParameterID(tp.ID())
    typeParams := []element.Element{tp}
    if got := formatJavaActionTypeGen(etp, typeParams); got != "entity <pEntity>" {
        t.Errorf("got %q, want entity <pEntity>", got)
    }
}

func TestFormatJavaActionReturnTypeGen_VoidType(t *testing.T) {
    // If gen has VoidType: use NewVoidType().
    // If not: assert that nil ActionReturnType returns "Void" (caller convention).
    if got := formatJavaActionReturnTypeGen(nil, nil); got != "Void" {
        t.Errorf("got %q, want Void", got)
    }
}
```

#### A2.S2: RED
#### A2.S3: Implement

```go
// formatJavaActionTypeGen dispatches on elem.TypeName() to render the MDL
// type syntax for a gen-typed Java action parameter or return type.
//
// typeParams is the action's ActionTypeParametersItems() slice — needed to
// resolve EntityTypeParameterType.TypeParameterRefID() back to the displayed
// type-parameter name (gen does not store the resolved name on the use site).
func formatJavaActionTypeGen(elem element.Element, typeParams []element.Element) string {
    if elem == nil { return "Object" }
    switch elem.TypeName() {
    case "CodeActions$VoidType":
        return "Void"
    case "CodeActions$BooleanType":
        return "Boolean"
    case "CodeActions$IntegerType":
        return "Integer"
    case "CodeActions$LongType":
        return "Long"
    case "CodeActions$DecimalType":
        return "Decimal"
    case "CodeActions$StringType":
        return "String"
    case "CodeActions$DateTimeType":
        return "DateTime"
    case "CodeActions$FileDocumentType":
        return "FileDocument"
    case "CodeActions$ConcreteEntityType":
        if et, ok := elem.(*genJA.ConcreteEntityType); ok && et.EntityQualifiedName() != "" {
            return et.EntityQualifiedName()
        }
        return "Object"
    case "CodeActions$EntityType":
        // Abstract; fall through with raw read in case Studio Pro emitted bare EntityType
        if name := codec.ReadBSONFieldString(elem.Raw(), "Entity"); name != "" {
            return name
        }
        return "Object"
    case "CodeActions$ListType":
        lt, ok := elem.(*genJA.ListType)
        if !ok { return "List" }
        inner := lt.Parameter()
        if inner == nil { return "List" }
        return "List of " + formatJavaActionTypeGen(inner, typeParams)
    case "CodeActions$EnumerationType":
        if et, ok := elem.(*genJA.EnumerationType); ok && et.EnumerationQualifiedName() != "" {
            return "Enum " + et.EnumerationQualifiedName()
        }
        return "Enumeration"
    case "CodeActions$EntityTypeParameterType":
        etp, ok := elem.(*genJA.EntityTypeParameterType)
        if !ok { return "entity <>" }
        name := resolveTypeParamName(etp.TypeParameterRefID(), typeParams)
        if name != "" { return "entity <" + name + ">" }
        return "entity <>"
    case "CodeActions$ParameterizedEntityType":
        pt, ok := elem.(*genJA.ParameterizedEntityType)
        if !ok { return "T" }
        if name := resolveTypeParamName(pt.TypeParameterRefID(), typeParams); name != "" {
            return name
        }
        return "T"
    case "CodeActions$MicroflowParameterType", "CodeActions$MicroflowJavaActionParameterType":
        return "Microflow"
    case "CodeActions$NanoflowType":
        return "Nanoflow"
    case "CodeActions$StringTemplateParameterType":
        if grammar := codec.ReadBSONFieldString(elem.Raw(), "Grammar"); grammar != "" {
            return "StringTemplate(" + grammar + ")"
        }
        return "StringTemplate"
    default:
        // Strip "CodeActions$" and "Type" suffix as a fallback display
        n := elem.TypeName()
        n = strings.TrimPrefix(n, "CodeActions$")
        n = strings.TrimSuffix(n, "Type")
        return n
    }
}

func formatJavaActionReturnTypeGen(elem element.Element, typeParams []element.Element) string {
    if elem == nil { return "Void" }
    return formatJavaActionTypeGen(elem, typeParams)
}

// resolveTypeParamName looks up a *TypeParameter in the def list by its ID
// and returns its Name. Returns empty string if not found.
func resolveTypeParamName(id element.ID, typeParams []element.Element) string {
    for _, tp := range typeParams {
        if tp == nil { continue }
        if typed, ok := tp.(*genJA.TypeParameter); ok && typed.ID() == id {
            return typed.Name()
        }
    }
    return ""
}
```

NOTE: schema gap fallback patterns ("schema gap: X returns ..."): if `genJA.LongType` is missing, the `case "CodeActions$LongType"` branch still works because TypeName() comes from the BSON, not the gen registry. Document each case here.

#### A2.S4: GREEN
#### A2.S5: Build + full test gate
#### A2.S6: Commit

```bash
git add mdl/executor/cmd_javaactions_gen.go mdl/executor/cmd_javaactions_gen_test.go
git commit -m "$(cat <<'EOF'
feat(executor): Stage 3.3.2.A2 — formatJavaActionType{,Return}Gen polymorphic dispatch

Renders gen-typed CodeActions$* sub-elements via TypeName() switch.
Resolves EntityTypeParameterType.TypeParameterRefID back to type-param
display names by scanning the action's ActionTypeParametersItems list.

Schema gap notes inline per memory project_gen_schema_gaps.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task A3: `describeJavaActionGen`

**Files:**
- Modify: `mdl/executor/cmd_javaactions_gen.go` (extend)
- Modify: `mdl/executor/cmd_javaactions_gen_test.go` (extend)

#### A3.S1: Failing test

```go
func TestDescribeJavaActionGen_OutputsCreateStatement(t *testing.T) {
    ctx := newJavaActionsTestContext(t)
    var buf bytes.Buffer
    ctx.Output = &buf
    if err := describeJavaActionGen(ctx, ast.QualifiedName{Module: "TestModule", Name: "MyAction"}); err != nil {
        t.Fatalf("describeJavaActionGen: %v", err)
    }
    out := buf.String()
    if !strings.Contains(out, "create java action TestModule.MyAction") {
        t.Errorf("missing create statement: %q", out)
    }
}
```

#### A3.S2: RED
#### A3.S3: Implement

```go
func describeJavaActionGen(ctx *ExecContext, name ast.QualifiedName) error {
    qn := name.Module + "." + name.Name
    ja, err := ctx.JavaActions.FindByQualifiedName(qn)
    if err != nil || ja == nil {
        return mdlerrors.NewNotFound("java action", qn)
    }

    var sb strings.Builder
    // Documentation (JavaDoc style)
    doc := strings.ReplaceAll(ja.Documentation(), "\r\n", "\n")
    doc = strings.ReplaceAll(doc, "\r", "\n")
    if doc != "" {
        sb.WriteString("/**\n")
        for line := range strings.SplitSeq(doc, "\n") {
            sb.WriteString(" * ")
            sb.WriteString(line)
            sb.WriteString("\n")
        }
        sb.WriteString(" */\n")
    }

    sb.WriteString("create java action ")
    sb.WriteString(qn)
    sb.WriteString("(")

    // Resolve type-parameters list once for use in type formatting
    typeParams := ja.ActionTypeParametersItems()

    params := ja.ActionParametersItems()
    hasDescriptions := false
    for _, p := range params {
        if pp, ok := p.(*genJA.JavaActionParameter); ok && pp.Description() != "" {
            hasDescriptions = true
            break
        }
    }

    for i, p := range params {
        pp, ok := p.(*genJA.JavaActionParameter)
        if !ok { continue }
        if i > 0 { sb.WriteString(", ") }
        if hasDescriptions { sb.WriteString("\n    ") }
        sb.WriteString(pp.Name())
        sb.WriteString(": ")
        sb.WriteString(formatJavaActionTypeGen(pp.ActionParameterType(), typeParams))
        if pp.IsRequired() { sb.WriteString(" not null") }
        if pp.Description() != "" {
            firstLine, _, _ := strings.Cut(strings.ReplaceAll(pp.Description(), "\r\n", "\n"), "\n")
            sb.WriteString("  -- ")
            sb.WriteString(firstLine)
        }
    }
    if hasDescriptions { sb.WriteString("\n") }
    sb.WriteString(")")

    if rt := ja.ActionReturnType(); rt != nil {
        sb.WriteString(" returns ")
        sb.WriteString(formatJavaActionReturnTypeGen(rt, typeParams))
    }

    if rn := ja.ActionDefaultReturnName(); rn != "" {
        sb.WriteString("\n-- return NAME: '")
        sb.WriteString(rn)
        sb.WriteString("'")
    }

    if info := ja.ModelerActionInfo(); info != nil {
        if mi, ok := info.(*genJA.MicroflowActionInfo); ok && mi.Caption() != "" {
            sb.WriteString("\nexposed as '")
            sb.WriteString(mi.Caption())
            sb.WriteString("' in '")
            sb.WriteString(mi.Category())
            sb.WriteString("'")
            if icon := mi.IconQualifiedName(); icon != "" {
                sb.WriteString("\n-- icon: ")
                sb.WriteString(icon)
            }
        }
    }

    if javaCode := readJavaActionUserCode(ctx.MprPath, name.Module, name.Name); javaCode != "" {
        sb.WriteString("\nas $$\n")
        sb.WriteString(javaCode)
        sb.WriteString("\n$$")
    }
    sb.WriteString(";")
    fmt.Fprintln(ctx.Output, sb.String())

    if el := ja.ExportLevel(); el != "" && el != "Hidden" {
        fmt.Fprintf(ctx.Output, "-- export level: %s\n", el)
    }
    if ja.Excluded() {
        fmt.Fprintln(ctx.Output, "-- EXCLUDED: true")
    }
    return nil
}
```

#### A3.S4: GREEN
#### A3.S5: Build + full test gate
#### A3.S6: Commit

```bash
git commit -m "feat(executor): Stage 3.3.2.A3 — describeJavaActionGen (gen-typed) ..."
```

### Task A4: `listJavaScriptActionsGen`

Mirror task A1 with `genJSA` package and `JavaScriptActions$JavaScriptAction` storage type. Renders qualifiedName/module/name/platform/folder.

- [ ] **Step 1: Test** — `TestListJavaScriptActionsGen_OutputsPlatform`
- [ ] **Step 2: RED**
- [ ] **Step 3: Implement** (same shape as A1, swap to `ctx.JavaScriptActions` + `Platform()`)
- [ ] **Step 4: GREEN, Step 5: gate, Step 6: Commit `feat(executor): Stage 3.3.2.A4 — listJavaScriptActionsGen (gen-typed)`**

### Task A5: `describeJavaScriptActionGen`

Mirror A3 with gen `JavaScriptAction`. Note: `JavaScriptAction` does not have a `MicroflowActionInfo` (it has `ModelerActionInfo()` only — verify in §S2.4 update). Also reads JS source file via `readJavaScriptActionUserCode` helper (already pure file I/O).

- [ ] **Step 1: Test** — `TestDescribeJavaScriptActionGen_OutputsCreateStatement`
- [ ] **Step 2: RED**
- [ ] **Step 3: Implement**
- [ ] **Step 4: GREEN, Step 5: gate, Step 6: Commit `feat(executor): Stage 3.3.2.A5 — describeJavaScriptActionGen (gen-typed)`**

### Task A6: Dispatcher cutover (executor_query.go)

Switch every `ShowJavaActions`, `ShowJavaScriptActions`, `DescribeJavaAction`, `DescribeJavaScriptAction` dispatch to `*Gen` variant.

**Files:**
- Modify: `mdl/executor/executor_query.go` (lines 43, 45, 183, 185)
- Modify: `mdl/executor/cmd_modules.go` (line 676 — `describeJavaAction` call inside MODULES dispatcher)

#### A6.S1: Locate
```bash
grep -nE "listJavaActions\b|listJavaScriptActions\b|describeJavaAction\b|describeJavaScriptAction\b" \
  mdl/executor/executor_query.go mdl/executor/register_stubs.go mdl/executor/cmd_modules.go
```

#### A6.S2: Replace each with `*Gen` variant

#### A6.S3: Build + full test gate

#### A6.S4: Commit

```bash
git add mdl/executor/executor_query.go mdl/executor/cmd_modules.go
git commit -m "$(cat <<'EOF'
refactor(executor): Stage 3.3.2.A6 — dispatch all SHOW/DESCRIBE javaactions to gen variants

Cuts over executor_query.go and cmd_modules.go to call listJavaActionsGen,
listJavaScriptActionsGen, describeJavaActionGen, describeJavaScriptActionGen
instead of the legacy sdk-typed counterparts.

Legacy functions stay in cmd_javaactions{,_javascript}.go until Phase E
deletes them; this commit only flips the routing.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## §5 Phase B — Visualization

**Skipped.** JavaActions domain has no `cmd_javaactions_elk.go` / `cmd_javaactions_mermaid.go`. Verified by `find mdl/executor/ -name "*java*"` — only show/describe/create/structure files exist. The structure helpers (`outputJavaActions` in `cmd_structure.go`) are covered in Phase C (consumer migration).

---

## §6 Phase C — Consumer Migration

Walk through every non-executor file that imports `sdk/javaactions`. One commit per file (group only when ≤5 lines per file).

### Task C1: `mdl/types/java.go` — restructure JavaScriptAction

**Strategy:** retire `types.JavaScriptAction`'s use of sdk types entirely. The lite descriptor `types.JavaAction` is already pure (no sdk import). For `JavaScriptAction`, two choices:

- **C1a (preferred):** Delete `types.JavaScriptAction` entirely. Consumers (`cmd_javascript_actions.go`, mock tests) switch to `*genJSA.JavaScriptAction` directly.
- **C1b (fallback):** Keep `types.JavaScriptAction` but redefine its rich-typed fields with self-contained types or `element.Element`. Higher maintenance cost.

This plan picks **C1a**. The consumers (cmd_javascript_actions.go) are migrated in Phase A4/A5 already; only the `types.JavaScriptAction` definition remains.

**Files:**
- Modify: `mdl/types/java.go` — delete JavaScriptAction; keep JavaAction (lite)
- Modify: `mdl/backend/mpr/convert.go` — delete `convertJavaScriptActionSlice`, `convertJavaScriptActionPtr`
- Modify: `mdl/backend/mpr/convert_roundtrip_test.go` — delete the JS-side test cases
- Modify: `mdl/backend/java.go` — change `ListJavaScriptActions` and `ReadJavaScriptActionByName` interface to return gen-typed pointers (or delete the methods if no caller remains after A4/A5 cutover)

#### C1.S1: Test
After A4/A5 land, run `grep -rn 'types.JavaScriptAction' mdl/ --include="*.go"`. Expected: only `mdl/backend/mpr/convert.go` and `mdl/backend/mpr/backend.go` (the shim methods). All consumer call sites should be gone.

#### C1.S2: RED — Delete and confirm everything that imports `types.JavaScriptAction` breaks. List the breakages.

#### C1.S3: Implement — for each broken site, route to gen-typed `ctx.JavaScriptActions.ListAll() / FindByQualifiedName`. Delete `types.JavaScriptAction`, the convert helpers, and the legacy backend methods. Add gen-typed `ListJavaScriptActionsGen() ([]*genJSA.JavaScriptAction, error)` and `ReadJavaScriptActionByNameGen(qn) (*genJSA.JavaScriptAction, error)` to `JavaBackend` (additive; legacy methods are removed in E1).

NOTE: `mdl/backend/java.go::JavaBackend.ListJavaScriptActions` returning `[]*types.JavaScriptAction` is a public domain interface change. Per master plan §3.8, it's NOT a `modelsdk.go` / `api/` public API change, so doesn't require user approval — but flag in commit message.

#### C1.S4: GREEN, Step 5: Build + test gate, Step 6: Commit

```bash
git add mdl/types/java.go mdl/backend/mpr/convert.go mdl/backend/mpr/convert_roundtrip_test.go \
        mdl/backend/java.go mdl/backend/mpr/backend.go
git commit -m "$(cat <<'EOF'
refactor(types,backend): Stage 3.3.2.C1 — retire types.JavaScriptAction

Deletes the sdk-typed wrapper. Consumers in mdl/executor/ and the catalog
now go through *genJSA.JavaScriptAction directly via ctx.JavaScriptActions
(introduced in A0). The convert.go helpers and ListJavaScriptActions /
ReadJavaScriptActionByName legacy backend methods are replaced by their
*Gen siblings.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task C2: `mdl/catalog/builder.go` — interface return type

The catalog's reader interface line 59 references `[]*javaactions.JavaAction`. Switch to `[]*genJA.JavaAction` and update the implementation to use `ctx.JavaActions.ListAll()` (or call into the new `ListJavaActionsGen` backend method added in C3).

**Files:**
- Modify: `mdl/catalog/builder.go` (line 59 + caller body)

- [ ] **Step 1: Test** — pick a catalog test that exercises `ListJavaActionsFull` and adapt the assertion to gen accessors
- [ ] **Step 2: RED**
- [ ] **Step 3: Replace `ListJavaActionsFull() ([]*javaactions.JavaAction, error)` with `ListJavaActionsGen() ([]*genJA.JavaAction, error)`. Update any call site that consumed `ja.Name`, `ja.ContainerID`, `ja.ID` to use `ja.Name()`, container resolved via the cache helper, `ja.ID()`**
- [ ] **Step 4: GREEN, Step 5: gate, Step 6: Commit `refactor(catalog): Stage 3.3.2.C2 — Reader.ListJavaActionsFull returns gen-typed *JavaAction`**

### Task C3: `mdl/backend/java.go` + `mdl/backend/mpr/backend.go` — additive gen-typed methods

Add gen-typed siblings BEFORE retiring legacy. Same pattern as security plan C1.

**Files:**
- Modify: `mdl/backend/java.go` (add `ListJavaActionsGen`, `ReadJavaActionByNameGen`, `CreateJavaActionGen`, `UpdateJavaActionGen`, `WriteJavaSourceFileGen`; same for JS where applicable)
- Modify: `mdl/backend/mpr/backend.go` (implement via `b.msdkWriter` + `mprrepos.NewJavaActionRepository`)
- Modify: `mdl/backend/mock/backend.go` + `mock_java.go` (Func-field stubs)

#### C3.S1: Failing test (`mdl/backend/mpr/services_modelsdk_test.go` extension)

```go
func TestListJavaActionsGen_ReturnsGenTyped(t *testing.T) {
    mprPath := makeFixtureWithJavaAction(t)
    b := New()
    if err := b.Connect(mprPath); err != nil { t.Fatalf("Connect: %v", err) }
    defer b.Disconnect()

    actions, err := b.ListJavaActionsGen()
    if err != nil { t.Fatalf("ListJavaActionsGen: %v", err) }
    if len(actions) == 0 { t.Fatal("ListJavaActionsGen empty") }
    if actions[0].Name() == "" { t.Errorf("Name() empty") }
}
```

#### C3.S2: RED
#### C3.S3: Implement
```go
// mdl/backend/java.go
type JavaBackend interface {
    // ... existing legacy methods stay until E1 ...
    ListJavaActionsGen() ([]*genJA.JavaAction, error)
    ReadJavaActionByNameGen(qn string) (*genJA.JavaAction, error)
    CreateJavaActionGen(parentUUID, containmentName string, ja *genJA.JavaAction) error
    UpdateJavaActionGen(ja *genJA.JavaAction) error
    DeleteJavaAction(id model.ID) error  // unchanged
    WriteJavaSourceFileGen(mod, action, code string, params []*genJA.JavaActionParameter, returnType element.Element, extraImports []string, extraCode string) error
    DeleteJavaSourceFile(mod, action) error    // unchanged
    RenameJavaSourceFile(...) error            // unchanged
    ReadJavaSourceFile(...) (string, error)    // unchanged
    ListJavaScriptActionsGen() ([]*genJSA.JavaScriptAction, error)
    ReadJavaScriptActionByNameGen(qn string) (*genJSA.JavaScriptAction, error)
}

// mdl/backend/mpr/backend.go
func (b *MprBackend) ListJavaActionsGen() ([]*genJA.JavaAction, error) {
    return mprrepos.NewJavaActionRepository(b.msdkWriter).ListAll()
}
func (b *MprBackend) ReadJavaActionByNameGen(qn string) (*genJA.JavaAction, error) {
    return mprrepos.NewJavaActionRepository(b.msdkWriter).FindByQualifiedName(qn)
}
// ... CreateJavaActionGen, UpdateJavaActionGen — wired in Phase D.S4 to gen-native serializer
```

Mock backend gets descriptive error stubs:

```go
// mdl/backend/mock/backend.go
ListJavaActionsGenFunc       func() ([]*genJA.JavaAction, error)
ReadJavaActionByNameGenFunc  func(qn string) (*genJA.JavaAction, error)
CreateJavaActionGenFunc      func(parentUUID, containmentName string, ja *genJA.JavaAction) error
UpdateJavaActionGenFunc      func(ja *genJA.JavaAction) error
WriteJavaSourceFileGenFunc   func(mod, action, code string, params []*genJA.JavaActionParameter, returnType element.Element, extraImports []string, extraCode string) error
ListJavaScriptActionsGenFunc func() ([]*genJSA.JavaScriptAction, error)
ReadJavaScriptActionByNameGenFunc func(qn string) (*genJSA.JavaScriptAction, error)

// mdl/backend/mock/mock_java.go
func (m *MockBackend) ListJavaActionsGen() ([]*genJA.JavaAction, error) {
    if m.ListJavaActionsGenFunc != nil { return m.ListJavaActionsGenFunc() }
    return nil, fmt.Errorf("MockBackend.ListJavaActionsGen not configured")
}
// ... and similar for the other four
```

#### C3.S4: GREEN, Step 5: gate, Step 6: Commit `feat(backend): Stage 3.3.2.C3 — add gen-typed javaactions read/write methods to FullBackend`

### Task C4: `mdl/executor/cmd_structure.go` — outputJavaActions signature

Migrate `outputJavaActions(ctx, mod, []*javaactions.JavaAction, withNames)` to `outputJavaActionsGen(ctx, mod, []*genJA.JavaAction, withNames)`. The function body uses `ja.Name`, `ja.Parameters`, `ja.ReturnType` etc. — all need gen accessor swaps. `formatJavaActionSignature(ja, withNames)` similarly needs a gen sibling.

**Files:**
- Modify: `mdl/executor/cmd_structure.go`
- Modify: `mdl/executor/cmd_structure_gen.go` (line 507 — change `structureJaMapGen` value type to `[]*genJA.JavaAction`)

- [ ] **Step 1: Test** — extend `cmd_structure_test.go` to assert structure output for a project with one Java action mentions the action name + signature
- [ ] **Step 2: RED**
- [ ] **Step 3: Implement** `outputJavaActionsGen` and `formatJavaActionSignatureGen`. Update `cmd_structure_gen.go::loadGenJaByModule` (or wherever the JA fetch happens, see line 334) to call `ctx.Backend.ListJavaActionsGen()` (added in C3) and update the `structureJaMapGen` type alias
- [ ] **Step 4: GREEN, Step 5: gate, Step 6: Commit `refactor(executor): Stage 3.3.2.C4 — structure outputJavaActions on gen types`**

### Task C5: MockBackend audit

Bring `mdl/backend/mock/mock_java.go` into compliance: every Func-field stub returns `"MockBackend.X not configured"` instead of `nil`. (master plan §5 P3 for javaactions.)

**Files:** `mdl/backend/mock/mock_java.go`

- [ ] **Step 1: Test** — verify a freshly-constructed `MockBackend` returns descriptive errors when calling each java method without setting the Func
- [ ] **Step 2: RED** (tests today pass because nil-return is permissive — flip them to expect the new error format)
- [ ] **Step 3: Replace every `return nil, nil` / `return nil` with `return …, fmt.Errorf("MockBackend.X not configured")` (where `X` matches the method name)**
- [ ] **Step 4: GREEN** — fix every test that broke because it relied on permissive `nil, nil`. The fix is to set the Func explicitly, not to relax the mock.
- [ ] **Step 5: Build + test gate**
- [ ] **Step 6: Commit `refactor(mock): Stage 3.3.2.C5 — MockBackend java stubs return descriptive errors`**

### Task C6: Mock test fixture migration

Migrate `*javaactions.JavaAction` fixtures in:
- `mdl/executor/cmd_javaactions_mock_test.go`
- `mdl/executor/validate_system_javaaction_test.go`
- `mdl/backend/mpr/services_modelsdk_test.go` (one fixture at line 588)

to gen builder pattern (`genJA.NewJavaAction()` + setters).

**Strategy:** these are scattered fixtures; do all THREE files in ONE commit so cross-file fixture references stay consistent.

- [ ] **Step 1: Run test files RED first** — once C3 lands the FullBackend signatures (or removes them in E1), these tests fail with type mismatches
- [ ] **Step 2: Confirm RED**
- [ ] **Step 3: Replace each `&javaactions.JavaAction{...}` literal with `genJA.NewJavaAction()` + setters:**

```go
ja := genJA.NewJavaAction()
ja.SetName("MyAction")
ja.SetDocumentation("Doc")
ja.SetExportLevel("Public")
// Parameters via gen builder + AddActionParameters
p1 := genJA.NewJavaActionParameter()
p1.SetName("p1")
p1.SetIsRequired(true)
st := genJA.NewStringType()
p1.SetActionParameterType(st)
ja.AddActionParameters(p1)
```

- [ ] **Step 4: GREEN, Step 5: full gate, Step 6: Commit `test(executor,backend): Stage 3.3.2.C6 — migrate javaactions fixtures to gen builders`**

---

## §7 Phase D — Write Path Migration

Goal: replace `execCreateJavaAction` and `execDropJavaAction` with `*Gen` twins that build `*genJA.JavaAction` from AST and route through `CreateJavaActionGen` / `UpdateJavaActionGen`.

### Task D1: AST → gen converters (`astDataTypeToJavaActionParamTypeGen`, `astDataTypeToJavaActionReturnTypeGen`)

**Files:**
- Modify: `mdl/executor/cmd_javaactions_gen.go` (extend with converters)
- Modify: `mdl/executor/cmd_javaactions_gen_test.go` (extend)

#### D1.S1: Failing tests (one per AST kind)

```go
func TestAstDataTypeToJavaActionParamTypeGen_String(t *testing.T) {
    dt := ast.DataType{Kind: ast.TypeString}
    elem := astDataTypeToJavaActionParamTypeGen(dt, nil)
    if elem.TypeName() != "CodeActions$StringType" {
        t.Errorf("got %s, want CodeActions$StringType", elem.TypeName())
    }
}

func TestAstDataTypeToJavaActionParamTypeGen_EntityList(t *testing.T) {
    dt := ast.DataType{
        Kind: ast.TypeListOf,
        EntityRef: &ast.QualifiedName{Module: "Sales", Name: "Order"},
    }
    elem := astDataTypeToJavaActionParamTypeGen(dt, nil)
    lt, ok := elem.(*genJA.ListType)
    if !ok { t.Fatalf("got %T, want *ListType", elem) }
    inner := lt.Parameter().(*genJA.ConcreteEntityType)
    if inner.EntityQualifiedName() != "Sales.Order" {
        t.Errorf("got %s, want Sales.Order", inner.EntityQualifiedName())
    }
}

func TestAstDataTypeToJavaActionParamTypeGen_TypeParamRef(t *testing.T) {
    typeParamNames := map[string]bool{"pEntity": true}
    typeParamIDs := map[string]element.ID{"pEntity": element.ID("tp-uuid")}
    dt := ast.DataType{Kind: ast.TypeEntityTypeParam, TypeParamName: "pEntity"}
    elem := astDataTypeToJavaActionParamTypeGen(dt, map[string]element.ID{"pEntity": "tp-uuid"})
    etp, ok := elem.(*genJA.EntityTypeParameterType)
    if !ok { t.Fatalf("got %T, want *EntityTypeParameterType", elem) }
    if etp.TypeParameterRefID() != "tp-uuid" {
        t.Errorf("got %s, want tp-uuid", etp.TypeParameterRefID())
    }
}
```

#### D1.S2: RED
#### D1.S3: Implement

```go
// astDataTypeToJavaActionParamTypeGen mirrors astDataTypeToJavaActionParamType
// (cmd_javaactions.go:455) but returns gen element.Element. The
// typeParamIDs map resolves type-parameter names (e.g., "pEntity") to the
// TypeParameter unit IDs created in the same execCreateJavaActionGen call.
func astDataTypeToJavaActionParamTypeGen(dt ast.DataType, typeParamIDs map[string]element.ID) element.Element {
    switch dt.Kind {
    case ast.TypeBoolean:    return genJA.NewBooleanType()
    case ast.TypeInteger:    return genJA.NewIntegerType()
    case ast.TypeLong:
        // schema gap: gen has no LongType in the listing surveyed in §S2.4;
        // verify in A0.S1. If gap confirmed, allocate via element.New("CodeActions$LongType")
        // OR fall back to BasicParameterType. Update this branch when gen narrows.
        return newGenElementByType("CodeActions$LongType")
    case ast.TypeDecimal:    return genJA.NewDecimalType()
    case ast.TypeString:     return genJA.NewStringType()
    case ast.TypeDateTime, ast.TypeDate: return genJA.NewDateTimeType()
    case ast.TypeEntityTypeParam:
        etp := genJA.NewEntityTypeParameterType()
        etp.SetTypeParameterID(typeParamIDs[dt.TypeParamName])
        return etp
    case ast.TypeEntity, ast.TypeEnumeration:
        entityName := ""
        if dt.EntityRef != nil { entityName = dt.EntityRef.Module + "." + dt.EntityRef.Name }
        if dt.EnumRef != nil   { entityName = dt.EnumRef.Module + "." + dt.EnumRef.Name }
        et := genJA.NewConcreteEntityType()
        et.SetEntityQualifiedName(entityName)
        return et
    case ast.TypeListOf:
        entityName := ""
        if dt.EntityRef != nil { entityName = dt.EntityRef.Module + "." + dt.EntityRef.Name }
        inner := genJA.NewConcreteEntityType()
        inner.SetEntityQualifiedName(entityName)
        lt := genJA.NewListType()
        lt.SetParameter(inner)
        return lt
    default:
        return genJA.NewStringType()
    }
}

// astDataTypeToJavaActionReturnTypeGen — same logic, with TypeVoid → NewVoidType (or nil if absent)
```

NOTE: `newGenElementByType(name string) element.Element` is a small helper that allocates a `*element.Generic` (or whatever stub element type the gen package uses for unknown $Types). May need to be added to `modelsdk/element/` as part of D1 — coordinate with the codec team if the helper doesn't exist.

#### D1.S4: GREEN, Step 5: gate, Step 6: Commit `feat(executor): Stage 3.3.2.D1 — AST → gen JavaAction param/return type converters`

### Task D2: `execCreateJavaActionGen`

**Files:**
- Modify: `mdl/executor/cmd_javaactions_gen.go`
- Modify: `mdl/executor/cmd_javaactions_gen_test.go`

#### D2.S1: Test — round-trip a CREATE JAVA ACTION; assert via `mx check` + reading back via `describeJavaActionGen`

```go
func TestExecCreateJavaActionGen_Roundtrip(t *testing.T) {
    ctx := newWritableJavaActionsTestContext(t)
    stmt := &ast.CreateJavaActionStmt{
        Name: ast.QualifiedName{Module: "TestModule", Name: "MyAction"},
        Documentation: "test",
        Parameters: []ast.JavaActionParam{
            {Name: "p1", Type: ast.DataType{Kind: ast.TypeString}, IsRequired: true},
        },
        ReturnType: ast.DataType{Kind: ast.TypeBoolean},
    }
    if err := execCreateJavaActionGen(ctx, stmt); err != nil { t.Fatalf("create: %v", err) }
    ja, err := ctx.JavaActions.FindByQualifiedName("TestModule.MyAction")
    if err != nil { t.Fatalf("read back: %v", err) }
    if ja.Name() != "MyAction" { t.Errorf("Name mismatch") }
    if len(ja.ActionParametersItems()) != 1 { t.Errorf("Parameters count mismatch") }
}
```

#### D2.S2: RED
#### D2.S3: Implement

Mirror `execCreateJavaAction` (cmd_javaactions.go:285) substituting:
- `&javaactions.JavaAction{...}` → `genJA.NewJavaAction()` + setters
- `&javaactions.TypeParameterDef{...}` → `genJA.NewTypeParameter()` + `SetName`
- `&javaactions.JavaActionParameter{...}` → `genJA.NewJavaActionParameter()` + setters + `SetActionParameterType`
- `&javaactions.MicroflowActionInfo{...}` → `genJA.NewMicroflowActionInfo()` + setters (use `SetIconQualifiedName` not `SetIcon`)
- `ctx.Backend.CreateJavaAction(ja)` → `ctx.Backend.CreateJavaActionGen(parentUUID, "Documents", ja)` (parent = module unit ID; containment name = `"Documents"` per gen schema)
- `ctx.Backend.UpdateJavaAction(ja)` → `ctx.Backend.UpdateJavaActionGen(ja)`
- `ctx.Backend.WriteJavaSourceFile(mod, name, code, ja.Parameters, ja.ReturnType, ...)` → `ctx.Backend.WriteJavaSourceFileGen(mod, name, code, gatherGenParams(ja), ja.ActionReturnType(), ...)` where `gatherGenParams(ja)` casts `ja.ActionParametersItems()` to `[]*genJA.JavaActionParameter`

After write, call `invalidateJavaActionsCache(ctx)`.

#### D2.S4: GREEN, Step 5: gate, Step 6: Commit `feat(executor): Stage 3.3.2.D2 — execCreateJavaActionGen`

### Task D3: `execDropJavaActionGen`

**Files:** same as D2

#### D3.S1: Test — drop existing action; assert it's gone from `ctx.JavaActions.ListAll()` and the .java file is gone
#### D3.S2: RED
#### D3.S3: Implement (mirror cmd_javaactions.go:248)
- List via `listJavaActionsWithContainerGen(ctx)` instead of `ctx.Backend.ListJavaActions()`
- Delete via `ctx.Backend.DeleteJavaAction(id)` (unchanged) + `ctx.Backend.DeleteJavaSourceFile(mod, name)` (unchanged)
- Call `invalidateJavaActionsCache(ctx)` after

#### D3.S4: GREEN, Step 5: gate, Step 6: Commit `feat(executor): Stage 3.3.2.D3 — execDropJavaActionGen`

### Task D4: Wire write dispatchers (`register_stubs.go`)

**Files:**
- Modify: `mdl/executor/register_stubs.go` (lines 119, 122)

#### D4.S1: Locate
```bash
grep -nE "execCreateJavaAction\b|execDropJavaAction\b" mdl/executor/register_stubs.go
```

#### D4.S2: Replace each with `*Gen` variant
#### D4.S3: Build + full test gate
#### D4.S4: Commit `refactor(executor): Stage 3.3.2.D4 — dispatch CREATE/DROP JAVA ACTION to gen variants`

### Task D5: Backend gen-native serializer (replace sdk-typed `createJavaActionViaModelsdk`)

**Files:**
- Modify: `mdl/backend/mpr/repos/javaactions.go` (fill in `Create` writer method)
- Modify: `mdl/backend/mpr/backend.go::CreateJavaActionGen` to delegate to `mprrepos.NewJavaActionRepository(b.msdkWriter).Create(parentUUID, "Documents", ja)`
- Modify: `mdl/backend/mpr/backend.go::UpdateJavaActionGen` similarly
- Eventually: deprecate `createJavaActionViaModelsdk` and `updateJavaActionViaModelsdk` (Phase E)

#### D5.S1: Test — `TestJavaActionRepository_Create_Roundtrip`
```go
func TestJavaActionRepository_Create_Roundtrip(t *testing.T) {
    w, cleanup := openFixtureWriter(t, "testdata/blank-fixture.mpr")
    defer cleanup()
    repo := NewJavaActionRepository(w)
    moduleUUID := lookupModuleUUID(t, w, "TestModule")
    ja := genJA.NewJavaAction()
    ja.SetName("MyJA")
    ja.SetExportLevel("Public")
    if err := repo.Create(moduleUUID, "Documents", ja); err != nil {
        t.Fatalf("Create: %v", err)
    }
    got, err := repo.FindByQualifiedName("TestModule.MyJA")
    if err != nil || got == nil {
        t.Fatalf("FindByQualifiedName: got=%v err=%v", got, err)
    }
}
```

#### D5.S2: RED
#### D5.S3: Implement
- `Create` calls `codec.Encode(ja)` to produce BSON bytes, then `r.w.InsertUnit(ja.ID(), parentUUID, containmentName, javaActionTypeName, bytes)`. Mirrors microflow `Create` exactly.
- `Update` calls `r.w.UpdateUnit(ja.ID(), bytes)` (or whatever sink-aware writer the microflow repo uses)
- `Delete` calls `r.w.DeleteUnit(string(id))`

#### D5.S4: Run `mx check` smoke test on the round-tripped MPR. Assert no CE0463 / CE0066 errors.

#### D5.S5: GREEN, gate, Commit `feat(backend/mpr/repos): Stage 3.3.2.D5 — javaactions repo Create/Update/Delete (gen-native)`

### Task D6: `WriteJavaSourceFileGen` (gen-typed file I/O signature)

**Files:**
- Modify: `mdl/backend/mpr/java_files.go` (add `writeJavaSourceFileViaPathGen` accepting `[]*genJA.JavaActionParameter` + `element.Element` returnType)
- Modify: `mdl/backend/mpr/backend.go::WriteJavaSourceFileGen` to delegate

The function generates Java source code; its only sdk dep is the formatter for parameter/return types. Use `formatJavaActionTypeGen` (already in cmd_javaactions_gen.go) by exporting/relocating it OR by writing a dedicated `genElementToJavaTypeName(elem)` helper inside `java_files.go`.

#### D6.S1–D6.S6: same TDD pattern, Commit `feat(backend/mpr): Stage 3.3.2.D6 — WriteJavaSourceFileGen accepts gen types`

---

## §8 Phase E — Cleanup

### Task E1: Retire `FullBackend` deprecated sdk-typed javaactions methods

**Files:**
- Modify: `mdl/backend/java.go` (delete `ListJavaActionsFull`, `ReadJavaActionByName`, `CreateJavaAction`, `UpdateJavaAction`, `WriteJavaSourceFile`, plus `ListJavaScriptActions`, `ReadJavaScriptActionByName` from interfaces)
- Modify: `mdl/backend/mpr/backend.go` (delete the corresponding shim methods)
- Modify: `mdl/backend/mock/backend.go` + `mock_java.go` (delete corresponding Func fields and shims)
- Delete: `mdl/backend/mpr/create_services_modelsdk.go::createJavaActionViaModelsdk` (now unused)
- Delete: `mdl/backend/mpr/update_services_modelsdk.go::updateJavaActionViaModelsdk` (now unused)

#### E1.S1: Build to confirm no remaining callers
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./...
```
If anything fails, route the failing caller through `*Gen` first.

#### E1.S2: Delete + commit `refactor(backend): Stage 3.3.2.E1 — retire FullBackend java sdk-typed methods + ViaModelsdk wrappers`

### Task E2: Delete legacy executor javaactions files

**Files:**
- Delete: `mdl/executor/cmd_javaactions.go`
- Delete: `mdl/executor/cmd_javascript_actions.go`

#### E2.S1: Final grep
```bash
grep -rn '"github.com/mendixlabs/mxcli/sdk/javaactions"' mdl/executor/cmd_javaactions.go mdl/executor/cmd_javascript_actions.go
```

#### E2.S2: `git rm`
#### E2.S3: Build + full test gate
#### E2.S4: Commit `refactor(executor): Stage 3.3.2.E2 — delete legacy cmd_javaactions{,_javascript}.go`

### Task E3: Delete `sdk/javaactions` package (conditional on Stage 4)

**Files:**
- Delete: `sdk/javaactions/javaactions.go`

#### E3.S1: Final acceptance grep
```bash
grep -rln '"github.com/mendixlabs/mxcli/sdk/javaactions"' . --include="*.go" | grep -v "^./sdk/mpr/"
```
Expected: empty.

#### E3.S2: Verify Stage 4 boundary
```bash
git log -1 --oneline -- sdk/mpr/parser_javaactions.go sdk/mpr/system_java_actions.go sdk/mpr/writer_javaactions.go sdk/mpr/parser_misc.go sdk/mpr/serialize_exports.go sdk/mpr/parser_javaactions_test.go
```
If Stage 4 has removed the sdk/mpr importers, proceed. If not, the package stays in place with a deprecation header (escape hatch — same pattern as security plan §11).

#### E3.S3: Delete + build
```bash
git rm -r sdk/javaactions/
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./...
```

If `sdk/mpr` fails to build, BACK OUT the deletion and leave deprecation header.

#### E3.S4: Run full test gate
#### E3.S5: Commit `refactor: Stage 3.3.2.E3 — delete sdk/javaactions package`

```
Aggregate Stage 3.3.2 stats:
- Commits: ~25
- LoC delta: -2700 approx (sdk/javaactions 269 + cmd_javaactions 667 + cmd_javascript_actions 278 + types/java.go ~30 + sdk wrappers ~150)
                  +1200 approx (gen helpers + cmd_*_gen.go + extended tests)
```

### Task E4: Final acceptance verification

#### E4.S1: Acceptance greps
```bash
# Outside sdk/mpr — must be 0
grep -rln '"github.com/mendixlabs/mxcli/sdk/javaactions"' . --include="*.go" | grep -v "^./sdk/mpr/"

# modelsdk — must be 0
grep -rln '"github.com/mendixlabs/mxcli/sdk/javaactions"' modelsdk/ --include="*.go"

# api — must be 0
grep -rln '"github.com/mendixlabs/mxcli/sdk/javaactions"' api/ --include="*.go"
```
All three: empty output.

#### E4.S2: Run javaactions-affecting tests
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/executor/ ./mdl/backend/mpr/ ./mdl/backend/mpr/repos/ -run "JavaAction|JavaScript|java" -count=1 -v
```

#### E4.S3: Verify cache helpers exist — `listJavaActionsWithContainerGen` in `mdl/executor/helpers_javaactions_gen.go`

#### E4.S4: Update memory `project_stage_3_3_javaactions_complete.md` with final stats; cross-link in `MEMORY.md` index

#### E4.S5: No commit needed

---

## §9 Acceptance Criteria

- [ ] `grep -rln '"github.com/mendixlabs/mxcli/sdk/javaactions"' . --include="*.go" | grep -v "^./sdk/mpr/"` returns 0 lines
- [ ] All `Test*Java*` and `Test*JavaScript*` tests pass:
      `GOPROXY=... ~/go1.26/bin/go test ./mdl/executor/ ./mdl/backend/mpr/ ./mdl/backend/mpr/repos/ -run "Java" -count=1 -v`
- [ ] `listJavaActionsWithContainerGen` (the cache helper) is the only path used in `mdl/executor/` for JavaAction listing; no `ctx.Backend.ListJavaActionsGen()` raw calls outside that helper
- [ ] `ctx.JavaActions` repo is the only path for JavaAction reads in `mdl/executor/`
- [ ] `mx check` smoke test passes on a round-tripped MPR with: scalar CREATE, parameterized CREATE (`<pEntity>`), entity-list CREATE, MicroflowActionInfo CREATE, DROP
- [ ] Full repo build: `GOPROXY=... ~/go1.26/bin/go build ./...`
- [ ] Full repo test suite: `GOPROXY=... ~/go1.26/bin/go test ./... -count=1 -timeout 240s`
- [ ] `sdk/javaactions/` directory is gone (or, if Stage 4 hasn't landed, kept with explicit deprecation header)
- [ ] `mdl/types/java.go::JavaScriptAction` no longer exists (deleted in C1)

---

## §10 Estimated Commit Count + Sequencing

| Phase | Tasks | Commits | Cumulative |
|---|---|---|---|
| A — Read path | A0, A1, A2, A3, A4, A5, A6 | 7 | 7 |
| B — Visualization | (skipped) | 0 | 7 |
| C — Consumer migration | C1, C2, C3, C4, C5, C6 | 6 | 13 |
| D — Write path | D1, D2, D3, D4, D5, D6 | 6 | 19 |
| E — Cleanup | E1, E2, E3, E4 | 3 (E4 verify-only) | 22 |

**Estimated total: ~22 commits** (within master plan §6 row #2's 30–40 range; revised down because the surface is smaller than initially projected and infrastructure is already in place).

**Sequencing rationale:**
- A0 first (repo + cache + ctx wiring) is load-bearing for everything else
- A1–A5 add gen-typed read functions; A6 cuts over the dispatcher AFTER all gen-typed targets exist
- C1 (retire types.JavaScriptAction) MUST come after A4/A5 so consumers are already on gen
- C3 (additive backend gen-typed methods) BEFORE D5 (write path) which depends on `ctx.Backend.CreateJavaActionGen`
- D6a (`WriteJavaSourceFileGen`) is bundled into D6 (no separate "fix the encoder" subtask because the version-prefix bug from Stage 3.3.1 D6a is ALREADY FIXED globally)
- E1–E3 strictly after both A6 and D4 dispatchers route to gen

**Per-session checkpoints:** commit after each numbered task. If interrupted mid-D2, the partial converters in D1 are independently reviewable.

---

## §11 Coordination With Stage 4 Team

### Stage 4's territory
- `sdk/mpr/parser_javaactions.go`, `sdk/mpr/parser_javaactions_test.go`
- `sdk/mpr/parser_misc.go` (parses JavaScriptAction)
- `sdk/mpr/serialize_exports.go` (re-exports `SerializeJavaAction`)
- `sdk/mpr/system_java_actions.go`
- `sdk/mpr/writer_javaactions.go`

Six files in `sdk/mpr/` import `sdk/javaactions`. Stage 4 owns rewriting them.

### Stage 3.3.2 commitment
**Stage 3.3.2 will NOT modify any file under `sdk/mpr/`.** Specifically:
- E3 deletion is conditional on the package being unreachable from `sdk/mpr/`. If Stage 4 has removed all 6 importers before E3 runs, E3 deletes the package. If not, E3 leaves the package with a deprecation header; Stage 4's PR will trigger the final delete.
- The acceptance grep in §9 explicitly excludes `^./sdk/mpr/` matches.

### Risk of merge collision
Low. Stage 3.3.2 touches:
- `mdl/executor/cmd_javaactions*.go`, `mdl/executor/cmd_javascript_actions*.go`
- `mdl/backend/java.go`, `mdl/backend/mpr/backend.go`, `mdl/backend/mpr/create_services_modelsdk.go`, `mdl/backend/mpr/update_services_modelsdk.go`, `mdl/backend/mpr/java_files.go`, `mdl/backend/mpr/services_modelsdk_test.go`
- `mdl/backend/mock/backend.go`, `mdl/backend/mock/mock_java.go`
- `mdl/types/java.go`
- `mdl/catalog/builder.go`
- `mdl/executor/cmd_structure.go`, `mdl/executor/cmd_structure_gen.go`
- `mdl/repos/javaactions.go` (new)
- `mdl/backend/mpr/repos/javaactions.go` (new)

Stage 4 touches `sdk/mpr/` files. **No file is touched by both teams** — collision risk is zero.

### Communication
The team-lead should mention to Stage 4:
1. Phase E3 may NOT actually `git rm sdk/javaactions/` if Stage 4 hasn't removed the 6 sdk/mpr importers. Same escape-hatch pattern as security E3.
2. The `mdl/types/java.go::JavaScriptAction` deletion in C1 means the `sdk/mpr.JavaScriptAction` type can also be deleted by Stage 4 once their `parser_misc.go` rewrite is done. Coordinate timing.

---

## §12 Self-Review Checklist (skill-required)

**Spec coverage:** §4 (Phase A) covers all 4 read funcs in `cmd_javaactions.go` + `cmd_javascript_actions.go` plus dispatcher cutover (A6). §6 (Phase C) covers all 8 in-scope consumer files (types/java, catalog, structure, mock backend, mock test, execution test fixture). §7 (Phase D) covers the 2 write funcs + AST converters + backend gen-native serializer. §8 (Phase E) covers the three cleanup commits + verification. §11 spells out Stage 4 boundary. ✓

**Type consistency:** All gen-typed accessor names in implementation snippets verified against `modelsdk/gen/javaactions/types.go` and `modelsdk/gen/javascriptactions/types.go` per §S2.4. The TBD entries (LongType, VoidType, NanoflowType, StringTemplateParameterType, FileDocumentType) are flagged with R4 risk + explicit A0.S1 verification step. ✓

**Risk surfacing:** R1 (dual accessor families) → A1 verification step. R2 (polymorphic discrimination) → A2 uses `TypeName()` switch. R3 (`MicroflowActionInfo.ImageData` gap) → flagged with raw-BSON fallback. R4 (missing types) → A0.S1 verification + raw-string switch fallback. R5–R7 (TypeParameter/ListType/EntityTypeParameterType quirks) → addressed in A2/A3 with explicit lookup helpers. R8 (JS reader still in sdk/mpr) → mprread bypass. R9 (types.JavaScriptAction restructure) → C1 dedicated task with retire-vs-keep decision. R10 (Stage 4 boundary) → §11. ✓

**TDD discipline:** Every task A0–A6, C1–C6, D1–D6 starts with "Step 1: Write failing test" + "Step 2: Confirm RED". No "similar to A1" shortcuts on critical paths (A6, C1, C5, D2 fully expanded). ✓

**Commit hygiene:** Each commit single-concern; D1 (converters) split from D2 (execCreateJavaAction) because they're independently reviewable; D5 (backend gen-native serializer) split from D6 (Java source file gen sig) because they touch different files. Commit messages use HEREDOC. No `--no-verify`. ✓

**No public-API break without approval:** No `modelsdk.go` or `api/` files touched. The `mdl/backend/java.go` interface change in C3 is internal — additive (`*Gen` methods) before subtractive (E1 retire). The `mdl/types/java.go::JavaScriptAction` deletion (C1) is also internal — `mdl/types/` is not a stable public API surface. ✓

**Cache discipline:** `listJavaActionsWithContainerGen` and `listJavaScriptActionsWithContainerGen` cache helpers added in A0 BEFORE any consumer migration; every Phase D write commit pairs with `invalidateJavaActionsCache(ctx)` per memory `feedback_executor_cache_pattern`. ✓

---

## §13 Execution Notes (Wave Concurrency)

Lead executes the plan with the following concurrency strategy:

### Strictly serial (lead does directly)
- **A0** (repo + cache + ctx wiring) — single-file blast radius, downstream tasks depend on it. Lead owns.
- **A6** (read dispatcher cutover) — small but needs all A1–A5 commits landed first. Lead owns.
- **C3** (FullBackend + MockBackend additive) — same. Lead owns.
- **D4** (write dispatcher cutover) — Lead owns.
- **E1, E2, E3** (cleanup) — strictly serial deletion + verification. Lead owns.

### Concurrent-safe (dispatch parallel teammates writing independent new files; lead serializes commits)
- **A1, A2, A3** (`cmd_javaactions_gen.go` formatters) — touch the SAME new file. Run SERIALLY by one teammate.
- **A4, A5** (`cmd_javaactions_gen.go` JS additions) — same file as A1–A3. Same teammate, sequential.
- **C1, C2, C4, C5, C6** — independent files; can be dispatched in parallel to up to 3 teammates simultaneously, EACH writing only their assigned file. Lead reviews + commits in dependency order (C1 may need to land before C4 because C4 may consume the new gen-typed JS path, depending on what types.JavaScriptAction still exposes).

### Critical / complex (single teammate + reviewer)
- **D2** (`execCreateJavaActionGen`) — ~150 LoC migration with type-parameter resolution + MicroflowActionInfo edge cases + Java source file write integration. One teammate writes; lead requests `codex review` on the diff before committing.
- **D5** (gen-native repo writer + `mx check` smoke test) — gen schema gaps may surface here. One teammate, manual verification by lead.

### Safety rule
- Per memory `feedback_multi_agent_worktree_concurrency`: NEVER run two teammates in this same worktree concurrently if both might `git add` overlapping files. The plan's parallel C-phase dispatch only assigns ONE file per teammate.
- Per master plan §3.4: NEVER push during execution.

---

## §14 Open Questions for the User Before Execution Starts

1. **gen schema verification for VoidType / LongType / etc.** — should A0.S1 produce a tracked memory `project_gen_schema_gaps_javaactions.md` with the verified table, or roll the gap notes into existing `project_gen_schema_gaps.md`? Default = roll into existing.
2. **C1 decision (retire vs. restructure `types.JavaScriptAction`)** — plan defaults to retire (delete entirely). Confirm before execution begins. If keep-and-restructure preferred, C1 swells from 1 commit to ~3.
3. **WriteJavaSourceFileGen signature** (D6) — `returnType` parameter is `element.Element` (gen) vs the legacy `javaactions.CodeActionReturnType`. The Java code generation logic inside `java_files.go` needs to dispatch on `elem.TypeName()` to pick the right Java type name (`String`, `Boolean`, etc.). Confirm this is the right boundary — alternative is to pre-resolve in the executor and pass a plain `string javaTypeName` parameter, simpler but less symmetric with read path.

These should be resolved BEFORE Phase D commits land. A0–C6 can proceed without resolving (1) and (2), but (3) blocks D6.

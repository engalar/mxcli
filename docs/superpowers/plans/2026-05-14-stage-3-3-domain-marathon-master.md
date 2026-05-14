# Stage 3.3 Domain Marathon Master Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Scope notice (per writing-plans skill):** This is a **strategic master plan** spanning 6 independent subsystems (domains). Each domain in §6 is itself a marathon (~30–60 commits) and **MUST get its own dedicated detailed plan generated before execution** (per the writing-plans `Scope Check` rule). The master plan defines the reusable template, priority order, acceptance criteria, and shared invariants. The §7 reference (domainmodel) is provided as a complete worked example to clone into a new file when starting that domain.

**Goal:** Migrate `mdl/executor/` and adjacent layers off the legacy hand-written `sdk/<domain>` packages onto the auto-generated `modelsdk/gen/<domain>` types for the 6 domains still importing legacy SDK packages, mirroring the Stage 3.2 (sdk/microflows → modelsdk/gen/microflows) playbook completed 2026-05-13/14.

**Architecture:** Each domain follows the proven 4-phase decomposition (read/format → viz → consumer migration → write path) plus a cleanup phase. New `_gen.go` files implement the gen-typed twin in parallel with legacy code; legacy is removed only in a final cleanup commit per domain. Consumers migrate file-by-file via `*_gen.go` siblings, with `FullBackend` exposing a deprecated sdk-typed surface during transition. Cache helpers (`listXxxWithContainerGen`) prevent O(N²) regressions at scale.

**Tech Stack:**
- Go 1.26 (`~/go1.26/bin/go`), `GOPROXY=https://mirrors.aliyun.com/goproxy/,direct`
- Source of truth: `modelsdk/gen/<domain>/types.go` (auto-generated, 53 domains, 1500+ types)
- BSON codec: `modelsdk/codec` (already used by mdl/backend/mpr/repos/)
- Read-only listers: `modelsdk/mprread/` (added 2026-05-14, commit `bb435644`)
- Write transactions: `mdl/backend/mpr/repos/<domain>.go` (gen-typed repository pattern)
- Test fixtures: `mdl/backend/mpr/repos/testdata/`, `cmd/mxcli/testdata/`

---

## 1. Background — Why Stage 3.3 Exists

Stage 3.2 (2026-05-13/14, 71 commits) completed the migration for `sdk/microflows`. The same migration pressure applies to 6 sibling domains where `mdl/executor` still imports `sdk/<domain>`:

| Domain | sdk LoC | Total non-`./sdk/` callers | mdl/executor callers | Complexity tier |
|---|---|---|---|---|
| domainmodel | 607 | 61 | 35 | High (entity tree, attributes, associations, indexes, access rules) |
| pages | 1478 | 60 | 26 | High (widget tree, 50+ widget types, ALTER PAGE mutator) |
| workflows | 384 | 35 | 14 | Medium (workflow steps, decisions, paths, boundary events) |
| agenteditor | 216 | 22 | ~10 | Medium (custom blob mechanism, requires AgentEditorCommons) |
| javaactions | 269 | 22 | 6 | Low-Medium (parameters, return types) |
| security | 160 | 16 | 7 | Low (module roles, user roles, access rules) |

**Total**: ~3,114 sdk LoC + 216 callers to migrate. Aggregate estimate: 200–360 commits.

## 2. Why a Master Plan + Per-Domain Sub-Plans

A single plan covering all 6 domains would have ~1,500 individual steps and produce a document no human can review. Each domain meets the writing-plans `Scope Check` "independent subsystem" definition and gets its own plan. The master plan provides:

1. **Shared invariants** (§3) — rules that apply to every domain and never change
2. **Reusable phase template** (§4) — the 4-phase decomposition extracted from Stage 3.2
3. **Cross-domain prep tasks** (§5) — work that benefits all 6 domains, done once
4. **Priority + sequencing** (§6) — which domain to attempt first and why
5. **One worked reference** (§7) — `domainmodel` fully decomposed as a template
6. **Per-domain stubs** (§8) — checklist + size estimate for the other 5 domains
7. **Final cross-cutting cleanup** (§9) — what to delete after all 6 domains land

---

## 3. Shared Invariants (apply to every domain)

These rules MUST hold for every commit in every domain:

### 3.1 TDD discipline
- Write the failing test first; run it to confirm RED
- Implement minimum code to GREEN
- Never skip a step. Memory `feedback_tdd.md` documents this as user feedback

### 3.2 Build + test gate per commit
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./...
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./... -count=1 -timeout 240s
```
Both MUST be green before commit. No `--no-verify`.

### 3.3 Cache discipline (avoids O(N²) at scale)
Per memory `feedback_executor_cache_pattern.md`: any `ctx.Backend.List*()` call in `mdl/executor/` MUST go through a cache helper. Stage 3.2 added `listMicroflowsWithContainerGen` for ContainerUUID join; analogous helpers MUST exist before mass migration:
- `listEntitiesWithContainerGen` (domainmodel)
- `listPagesWithContainerGen` (pages)
- `listWorkflowsWithContainerGen` (workflows)
- (security/javaactions/agenteditor: assess per-domain at start)

### 3.4 No `git push`
The user manages `origin` push externally. Plans never push.

### 3.5 Single-concern commits
Per CLAUDE.md "Scope & atomicity" checklist:
- One feature OR one bugfix OR one refactor per commit
- Refactors that touch many files = own commit, not bundled with feature

### 3.6 BSON storage names vs qualified names
Per CLAUDE.md "BSON Storage Names vs Qualified Names": always verify storage name in `reference/mendixmodellib/reflection-data/` and `sdk/mpr/parser_*.go` before adding new types. The gen package usually has the right alias but schema gaps (`project_gen_schema_gaps.md`) require fallback to `codec.ReadBSONFieldString(obj.Raw(), "<storageKey>")`.

### 3.7 Container linkage chain
Per memory `mprread_package` and Stage 3.2 commit `6937aa37`: gen-typed elements lose `Container()` linkage after codec roundtrip. Use `mreader.ListUnitsByType("Domain$Type")` + `elem.ID()` join to recover ContainerID.

### 3.8 No public-API breakage without explicit user approval
`modelsdk.go` and `api/<domain>.go` files are public surface. Any signature change MUST be approved before commit. Stage 3.2 task B1+B2 (deleting `api/microflows.go` + retargeting `modelsdk.Microflow` alias) used this gate.

---

## 4. Reusable 4-Phase Template (per domain)

Every domain follows this phase decomposition. Generate the per-domain plan by instantiating this template with the domain-specific file paths.

### Phase A — Read path (formatters + queries)
Migrate `mdl/executor/cmd_<domain>_show.go` and helpers.

**Sub-stages** (each = 1 commit):
1. `cmd_<domain>_show_gen.go` skeleton with `Describe<Type>GenToString` returning a stub string
2. Per-family formatters (one commit per coherent group): properties, sub-elements, references
3. Replace internal references from `obj.Field()` to gen-typed accessors with raw-BSON fallback per `gen_schema_gaps`

### Phase B — Visualization
Migrate `cmd_<domain>_elk.go`, `cmd_<domain>_mermaid.go`, `cmd_<domain>_structure.go`.

**Sub-stages**:
1. `*_gen.go` twin with same public function signatures
2. Per-viz format migration (ELK, Mermaid, Structure tree)
3. Switch consumer in `cmd_visualize.go` (or equivalent dispatcher) to gen variant

### Phase C — Consumer migration
Walk through `mdl/executor/` and adjacent packages (`mdl/catalog/`, `mdl/linter/`) replacing `sdk/<domain>` imports file-by-file.

**Sub-stages**:
- One commit per consumer file (or coherent group of <5 files)
- `FullBackend` retains deprecated sdk-typed methods until last consumer is migrated
- Each consumer migration goes through cache helpers (§3.3)

### Phase D — Write path
Migrate `cmd_<domain>_create.go`, `cmd_<domain>_alter.go`, `cmd_<domain>_drop.go`, `flowbuilder_*.go` equivalents.

**Sub-stages** (most complex, follow Stage 3.2.3.a–l pattern):
1. Test scaffolding using `MockBackend` from `mdl/backend/mock/`
2. Per-mutation gen helpers (e.g., `setRawBSONField` for schema gaps)
3. Replace builder family one-by-one
4. Container linkage via §3.7 pattern

### Phase E — Cleanup
Final per-domain cleanup commit(s):
1. Retire `FullBackend.<sdk-typed methods>` (deprecated surface)
2. Delete legacy `cmd_<domain>_*.go` files (non-`_gen` versions)
3. Delete `sdk/<domain>/` package
4. Final verification: `grep -rln '"github.com/mendixlabs/mxcli/sdk/<domain>"' . --include="*.go"` → 0 (or only `modelsdk.go` if alias still exists)

---

## 5. Cross-Domain Prep Tasks (do once, benefits all 6)

These prep tasks MUST be in the first per-domain plan executed; they unblock all subsequent domains.

### Task P1: Extend `modelsdk/mprread` with generic unit lister

**Files:**
- Modify: `modelsdk/mprread/mprread.go` — add `ListUnitsByType[T element.Element](r *mpr.Reader, typeName string) ([]T, error)` generic helper
- Create: `modelsdk/mprread/generic_test.go`

- [ ] **Step 1: Write failing test for generic lister**

```go
// modelsdk/mprread/generic_test.go
func TestListUnitsByType_Microflows(t *testing.T) {
    r := openFixture(t)
    defer r.Close()
    mfs, err := mprread.ListUnitsByType[*genMf.Microflow](r, "Microflows$Microflow")
    require.NoError(t, err)
    require.NotEmpty(t, mfs)
}
```

- [ ] **Step 2: Run to confirm RED**

```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./modelsdk/mprread/ -run TestListUnitsByType_Microflows -v
```
Expected: FAIL ("undefined: ListUnitsByType")

- [ ] **Step 3: Implement generic lister**

```go
// modelsdk/mprread/mprread.go
func ListUnitsByType[T element.Element](r *mpr.Reader, typeName string) ([]T, error) {
    refs, err := r.ListUnitsByType(typeName)
    if err != nil { return nil, err }
    dec := codec.NewDecoder()
    out := make([]T, 0, len(refs))
    for _, ref := range refs {
        bytes, err := r.GetRawUnitBytes(ref.ID)
        if err != nil { return nil, fmt.Errorf("read unit %s: %w", ref.ID, err) }
        elem, err := dec.Decode(bytes)
        if err != nil { return nil, fmt.Errorf("decode unit %s: %w", ref.ID, err) }
        typed, ok := elem.(T)
        if !ok { return nil, fmt.Errorf("unit %s decoded as %T, want %T", ref.ID, elem, *new(T)) }
        out = append(out, typed)
    }
    return out, nil
}
```

- [ ] **Step 4: Run test to confirm GREEN**

- [ ] **Step 5: Refactor existing `ListMicroflows`/`ListNanoflows` to delegate to generic**

```go
func ListMicroflows(r *mpr.Reader) ([]*genMf.Microflow, error) {
    return ListUnitsByType[*genMf.Microflow](r, "Microflows$Microflow")
}
```

- [ ] **Step 6: Re-run full mprread test to confirm no regression**

- [ ] **Step 7: Commit**

```bash
git add modelsdk/mprread/
git commit -m "feat(mprread): add generic ListUnitsByType for cross-domain reuse

Extracts the type-filter pattern from ListMicroflows/ListNanoflows
into a generic helper so domain marathons (Stage 3.3) can list pages,
entities, workflows, security units, etc., without per-domain
boilerplate. Existing typed wrappers delegate to the generic.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### Task P2: Add cache helper template `listUnitsWithContainerGen` factory

**Files:**
- Create: `mdl/executor/helpers_gen_container.go`
- Modify: `mdl/executor/executor.go` — add cache fields per domain in `ExecutorCache`

- [ ] **Step 1: Write failing test using existing fixture**

```go
// mdl/executor/helpers_gen_container_test.go
func TestListEntitiesWithContainerGen_CachesAcrossCalls(t *testing.T) {
    ctx := newTestContext(t)
    list1, err := listEntitiesWithContainerGen(ctx)
    require.NoError(t, err)
    list2, _ := listEntitiesWithContainerGen(ctx)
    require.Same(t, &list1[0], &list2[0], "cache must return same slice header")
}
```

- [ ] **Step 2: Confirm RED**

- [ ] **Step 3: Implement factory pattern**

```go
// mdl/executor/helpers_gen_container.go
type genWithContainer[T element.Element] struct {
    Elem        T
    ContainerID model.ID
}

func listUnitsWithContainerGen[T element.Element](
    ctx *Context,
    typeName string,
    cacheGet func() ([]genWithContainer[T], bool),
    cachePut func([]genWithContainer[T]),
) ([]genWithContainer[T], error) {
    if cached, ok := cacheGet(); ok { return cached, nil }
    units, err := mprread.ListUnitsByType[T](ctx.Reader, typeName)
    if err != nil { return nil, err }
    refs, _ := ctx.Reader.ListUnitsByType(typeName)
    refMap := make(map[string]model.ID, len(refs))
    for _, ref := range refs { refMap[ref.ID] = model.ID(ref.ContainerID) }
    out := make([]genWithContainer[T], 0, len(units))
    for _, u := range units {
        out = append(out, genWithContainer[T]{Elem: u, ContainerID: refMap[string(u.ID())]})
    }
    cachePut(out)
    return out, nil
}

// Concrete wrappers per domain go in cmd_<domain>_helpers_gen.go
```

- [ ] **Step 4: GREEN**
- [ ] **Step 5: Commit `feat(executor): add generic listUnitsWithContainerGen factory for Stage 3.3 domains`**

### Task P3: `MockBackend` audit for gen-typed mutator stubs

Inspect `mdl/backend/mock/` and ensure for every domain in §6 there's a `Func`-field stub for gen-typed Create/Update/Delete. Memory: every new backend method needs `"MockBackend.X not configured"` error default per CLAUDE.md.

- [ ] One commit per domain stub addition

---

## 6. Domain Priority + Recommended Sequence

Domains ordered by **value × tractability** (highest first). Each row links to the per-domain detailed plan that MUST be generated before starting.

| # | Domain | Why this priority | LoC saved | Est commits | Sub-plan file (TBD) |
|---|---|---|---|---|---|
| 1 | **security** | Smallest LoC + lowest complexity; best learning ground; Stage 3.2 cache pattern (`listMicroflowsWithContainerGen`) already covers half the cross-cutting work | 160 | 25–35 | `2026-XX-XX-stage-3-3-security.md` |
| 2 | **javaactions** | Small surface; mostly leaf data (parameters, return types); no widget tree complexity | 269 | 30–40 | `2026-XX-XX-stage-3-3-javaactions.md` |
| 3 | **workflows** | Medium complexity; activity types overlap microflow patterns (Stage 3.2 reusable); ALTER WORKFLOW already exists as reference | 384 | 40–55 | `2026-XX-XX-stage-3-3-workflows.md` |
| 4 | **domainmodel** | High complexity but **highest value**: 35 executor files unblocked; reference §7 details all phases | 607 | 50–70 | `2026-XX-XX-stage-3-3-domainmodel.md` (template at §7) |
| 5 | **pages** | Highest complexity (widget tree, 50+ widgets, ALTER PAGE); should ride momentum from domainmodel | 1478 | 60–90 | `2026-XX-XX-stage-3-3-pages.md` |
| 6 | **agenteditor** | Custom blob mechanism = unique BSON; requires AgentEditorCommons; defer last | 216 | 30–45 | `2026-XX-XX-stage-3-3-agenteditor.md` |

**Reasoning for ordering** (not alphabetical):
- §1–§3 build muscle memory and prove the template works on simple shapes
- §4 (domainmodel) unblocks the largest consumer fanout (35 files), justifying the §7 deep-dive
- §5 (pages) is the riskiest; tackling it after domainmodel reuses entity-related infra
- §6 (agenteditor) has the most unique BSON shape; isolated last so its quirks don't pollute earlier work

---

## 7. Reference Detailed Sub-Plan: domainmodel

This section is a **complete worked example**. When starting the domainmodel domain, copy this section into `docs/superpowers/plans/2026-XX-XX-stage-3-3-domainmodel.md`, fill in TBD file paths from a fresh survey, and execute it standalone.

### 7.1 Pre-flight survey (do BEFORE writing the per-domain plan)

```bash
# Confirm current call surface
grep -rln '"github.com/mendixlabs/mxcli/sdk/domainmodel"' . --include="*.go" | grep -v "^./sdk/" | tee /tmp/domainmodel-callers.txt

# Per-file caller density
grep -rc '"github.com/mendixlabs/mxcli/sdk/domainmodel"' mdl/executor/ | sort -t: -k2 -rn | head -20

# Identify visualization files
find mdl/executor/ -name "*domainmodel*" -o -name "*entity*" | grep -v _test
```

Record the caller list in `/tmp/domainmodel-callers.txt`. The per-domain plan tasks are derived from this list.

### 7.2 Domain-specific task graph

```
Phase A (Read path)         Phase B (Viz)         Phase C (Consumers)        Phase D (Write)         Phase E (Cleanup)
  A1: show skeleton           B1: elk_gen           C1-C35: per-file walk      D1: test scaffolds       E1: retire FullBackend
  A2: entity formatter        B2: mermaid_gen        (5 commits per group)     D2: CreateEntity         E2: delete legacy files
  A3: attr formatter          B3: structure_gen                                D3: AddAttribute         E3: delete sdk/domainmodel
  A4: assoc formatter                                                          D4: CreateAssociation
  A5: index formatter                                                          D5: AlterEntity rename
  A6: access rule formatter                                                    D6: AlterEntity persist
  A7: doc/comments formatter                                                   D7: SetIndexes
                                                                              D8: AddEntityAccessRule
                                                                              D9: SetGeneralization
```

### 7.3 Phase A example task — A1: show skeleton

**Files:**
- Create: `mdl/executor/cmd_domainmodel_show_gen.go`
- Test: `mdl/executor/cmd_domainmodel_show_gen_test.go`

- [ ] **Step 1: Write failing test for entity describe**

```go
// mdl/executor/cmd_domainmodel_show_gen_test.go
func TestDescribeEntityGen_OutputsName(t *testing.T) {
    ctx := newTestContext(t, "testdata/domainmodel-fixture.mpr")
    out, err := DescribeEntityGenToString(ctx, "MyModule.Customer")
    require.NoError(t, err)
    require.Contains(t, out, "ENTITY MyModule.Customer")
}
```

- [ ] **Step 2: RED**

```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/executor/ -run TestDescribeEntityGen_OutputsName -v
```
Expected: FAIL (DescribeEntityGenToString undefined)

- [ ] **Step 3: Skeleton implementation**

```go
// mdl/executor/cmd_domainmodel_show_gen.go
package executor

import (
    "fmt"
    "strings"

    "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

func DescribeEntityGenToString(ctx *Context, qualifiedName string) (string, error) {
    entity, err := findEntityByQualifiedNameGen(ctx, qualifiedName)
    if err != nil { return "", err }
    var sb strings.Builder
    fmt.Fprintf(&sb, "ENTITY %s\n", qualifiedName)
    return sb.String(), nil
}
```

(Add `findEntityByQualifiedNameGen` stub — returns first entity matching name; full implementation deferred to A2)

- [ ] **Step 4: GREEN**
- [ ] **Step 5: Commit**

```bash
git add mdl/executor/cmd_domainmodel_show_gen.go mdl/executor/cmd_domainmodel_show_gen_test.go
git commit -m "feat(executor): Stage 3.3.1.A1 — entity describe skeleton (gen)

Adds DescribeEntityGenToString returning a stub line. Subsequent
A2-A7 commits incrementally add entity attribute, association,
index, access rule, and documentation formatters.

Mirrors Stage 3.2.1 PoC pattern from microflow migration.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### 7.4 Phase A — A2: entity attribute formatter

**Files:**
- Modify: `mdl/executor/cmd_domainmodel_show_gen.go`
- Test: `mdl/executor/cmd_domainmodel_show_gen_test.go` (extend)

- [ ] **Step 1: Add failing test for attribute output**

```go
func TestDescribeEntityGen_RendersAttributes(t *testing.T) {
    ctx := newTestContext(t, "testdata/domainmodel-fixture.mpr")
    out, _ := DescribeEntityGenToString(ctx, "MyModule.Customer")
    require.Contains(t, out, "Name STRING")
    require.Contains(t, out, "Age INTEGER")
}
```

- [ ] **Step 2: RED**

- [ ] **Step 3: Implement attribute walk**

```go
func formatAttributeGen(attr *domainmodels.Attribute) string {
    typeName := formatAttributeTypeGen(attr.Type())
    return fmt.Sprintf("  %s %s", attr.Name(), typeName)
}

func formatAttributeTypeGen(t domainmodels.AttributeType) string {
    if t == nil { return "UNKNOWN" }
    switch t.TypeName() {
    case "DomainModels$StringAttributeType": return "STRING"
    case "DomainModels$IntegerAttributeType": return "INTEGER"
    case "DomainModels$BooleanAttributeType": return "BOOLEAN"
    // ... full type list per modelsdk/gen/domainmodels/types.go
    default: return strings.TrimPrefix(t.TypeName(), "DomainModels$")
    }
}
```

Update DescribeEntityGenToString to walk `entity.Attributes()`.

- [ ] **Step 4: GREEN**
- [ ] **Step 5: Commit `feat(executor): Stage 3.3.1.A2 — entity attribute formatter (gen)`**

### 7.5 Phase A through E task list (placeholder — full enumeration in actual sub-plan)

Each task in the actual sub-plan follows the same 5-step (write test / confirm RED / implement / confirm GREEN / commit) pattern. A complete domainmodel sub-plan should enumerate ~50–70 tasks; the above two are illustrative.

When generating the sub-plan, **for each task**:
1. Identify exactly which file is created or modified
2. Write the test code in full
3. Show the implementation code in full
4. Show the exact commit command with message

Do NOT write "similar to A1" — repeat the 5-step structure per task.

### 7.6 Phase D — Write path special considerations

DomainModel write path has unique complexity:
- **Entity access rules**: per CLAUDE.md "Association Parent/Child Pointer Semantics", MemberAccess entries must only be added to the FROM entity (the one in `ParentPointer`)
- **AllowedModuleRoles version prefix bug**: per memory `project_allowedmoduleroles_version_prefix`, gen-native writes are missing the `int32(1)` version prefix; either fix in the writer or use `setRawBSONField` workaround
- **Index ordering deterministic**: per CLAUDE.md "Map iteration is deterministic" rule
- **System module dependency**: per memory `project_sdk_domainmodel_status`, sdk/domainmodel deletion has been blocked on system_module BSON-native rewrite — assess if Stage 3.3.4 can land without resolving this, or if it must come first

### 7.7 Phase E — Cleanup commits

After all consumers are migrated:
1. `refactor(backend): Stage 3.3.4.E1 — retire FullBackend domainmodel sdk-typed methods`
2. `refactor(executor): Stage 3.3.4.E2 — delete legacy cmd_domainmodel_*.go files`
3. `refactor: Stage 3.3.4.E3 — delete sdk/domainmodel package (Stage 3.3 domain 4 complete)`

### 7.8 Acceptance criteria for domainmodel domain

- `grep -rln '"github.com/mendixlabs/mxcli/sdk/domainmodel"' . --include="*.go"` → 0 (or only `modelsdk.go` if alias retained)
- All 35 executor files migrated without test regression
- `go test ./... -count=1 -timeout 240s` green throughout
- New `_gen.go` test count ≥ original sdk-typed test count
- Memory entry `project_stage_3_3_domainmodel.md` created with commit count + LoC delta

---

## 8. Per-Domain Sub-Plan Stubs (other 5 domains)

For each domain below, a dedicated detailed plan MUST be written before execution. Each plan instantiates §4 with that domain's specifics.

### 8.1 security (priority #1 — start here)

**Pre-survey commands:**
```bash
grep -rln '"github.com/mendixlabs/mxcli/sdk/security"' . --include="*.go" | grep -v "^./sdk/"
find mdl/executor -name "*security*" -not -name "*_test*"
```

**Special considerations:**
- AllowedModuleRoles version prefix bug (memory: `project_allowedmoduleroles_version_prefix`) — fix or workaround required in Phase D
- Module roles vs user roles split — verify gen package boundary (`gen/security` or split package)
- Demo users handling — minimal complexity but check fixture coverage

**Acceptance**: `grep sdk/security .` → 0; security_matrix tests pass; cache helper `listSecurityWithContainerGen` exists

### 8.2 javaactions (priority #2)

**Pre-survey:** same pattern as §8.1 with `javaactions`

**Special considerations:**
- Java file system writes (per memory `modelsdk_migration_pattern`): 4 cases out of 32 remaining `b.writer.*` calls are Java FS, no SQLite — these are NOT in scope for Stage 3.3
- Parameters + return type mostly leaf data, smallest gen-typed surface area

### 8.3 workflows (priority #3)

**Pre-survey:** same pattern

**Special considerations:**
- ALTER WORKFLOW already implemented (per CLAUDE.md "Implemented" list) — likely reusable patterns
- Activity type overlap with microflows (Stage 3.2.2 family formatters) — consider extracting shared abstraction
- Boundary events + parallel splits = unique BSON shapes, verify gen accessors

### 8.4 pages (priority #5)

**Pre-survey:** same pattern + widget enumeration:
```bash
ls sdk/pages/  # 10 files
ls sdk/widgets/templates/
```

**Special considerations:**
- Widget tree mutator (`OpenPageForMutation`) is already in `mdl/backend/mpr/page_mutator.go` — Phase D follows existing pattern
- Pluggable widget templates (CLAUDE.md "Pluggable Widget Templates"): CE0463 risk if Object/Type schema diverges
- 50+ widget types = large but flat fanout; family formatters per widget category (input/display/list/container)
- `ALTER PAGE/SNIPPET` complexity — most write-path commits will be here

### 8.5 agenteditor (priority #6 — last)

**Pre-survey:** same pattern

**Special considerations:**
- Custom blob mechanism (per memory `modelsdk_migration_pattern`: 8 of 32 remaining `b.writer.*` are AgentEditor)
- Requires AgentEditorCommons module + Mendix 11.9+ (per CLAUDE.md "Implemented")
- Dollar-quoted multi-line prompts — verify codec roundtrip preserves quoting
- Defer until after pages so the heavy widget-tree logic patterns are already proven

---

## 9. Cross-Cutting Final Cleanup (after all 6 domains land)

### Task F1: Delete `sdk/microflows` package (deferred from Stage 3.2)

After Stage 4 (sdk/mpr microflow retire) lands, sdk/microflows is unreachable. Delete the package and verify nothing breaks.

- [ ] **Step 1: Confirm 0 imports**

```bash
grep -rln '"github.com/mendixlabs/mxcli/sdk/microflows"' . --include="*.go"
```
Expected: empty

- [ ] **Step 2: `git rm -r sdk/microflows/`**
- [ ] **Step 3: `go build ./... && go test ./... -count=1`**
- [ ] **Step 4: Commit `refactor: Stage 3.3 cleanup — delete sdk/microflows package`**

### Task F2: Verify modelsdk.go has zero sdk/* imports

After all per-domain Phase E commits, modelsdk.go should only import gen packages. Audit:
```bash
grep '"github.com/mendixlabs/mxcli/sdk/' modelsdk.go
```
Expected: empty (or only `sdk/mpr` if reader path still needed)

### Task F3: Memory updates

Create memory entry `project_stage_3_3_complete.md` summarizing:
- Total commits per domain
- Aggregate LoC delta (sdk deletion + gen wrappers)
- Schema gaps discovered + filed
- Pattern reusability notes for future domain marathons

Update `MEMORY.md` index.

### Task F4: Documentation

- Update `CLAUDE.md` "Implemented" / "Not Yet Implemented" lists
- Update `docs/SDK_EQUIVALENCE.md` showing post-3.3 state
- Update any `.claude/skills/` files referencing sdk/<domain> paths

---

## 10. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Schema gap in gen type without setter (e.g., AllowedModuleRoles version prefix) | High | Medium | `setRawBSONField` pattern (per `gen_schema_gaps`); accumulate gap list per domain |
| Cache invalidation bug on write → stale reads | Medium | High | Pair every Create/Update/Delete with `invalidate*Cache` (per `executor_cache_pattern`); test cache invalidation explicitly |
| O(N²) regression at scale | Medium | High | Mandatory cache helpers (§3.3); fixture with ≥100 entities for performance smoke test |
| Studio Pro rejects MPR after migration | Medium | Critical | Each Phase D commit verified by `mx check` on test fixture; round-trip test |
| Public API break unnoticed | Low | High | §3.8 invariant enforced; `modelsdk.go` and `api/` files require explicit user approval per change |
| Cross-domain dependency creates circular plan | Medium | Medium | §6 ordering chosen to minimize cross-deps; system_module dependency for domainmodel flagged in §7.6 |
| Plan execution drift across multi-session work | High | Medium | Per-domain sub-plan committed to `docs/superpowers/plans/`; resume via `/superpowers:executing-plans` |

---

## 11. Acceptance Criteria (overall Stage 3.3)

- [ ] All 6 domain sub-plans written and stored in `docs/superpowers/plans/`
- [ ] Each sub-plan executed; final domain Phase E commit landed
- [ ] `grep -rln '"github.com/mendixlabs/mxcli/sdk/' mdl/ --include="*.go"` returns 0 (or only `sdk/mpr` if read paths legitimately need it)
- [ ] All 6 sdk legacy packages (domainmodel, pages, workflows, security, javaactions, agenteditor) deleted from `sdk/` directory
- [ ] `modelsdk.go` has 0 `sdk/<domain>` imports for these 6 domains
- [ ] `api/<domain>.go` deleted or migrated to gen for these 6 domains (per Stage 4 task B pattern)
- [ ] Full test suite green: `GOPROXY=... go test ./... -count=1 -timeout 240s`
- [ ] Memory `project_stage_3_3_complete.md` created
- [ ] No new untested code paths introduced (test count ≥ pre-migration baseline)

---

## 12. Self-Review Checklist (skill-required)

**Spec coverage:** Each of the 6 domains has a dedicated section (§7 reference + §8.1–§8.5 stubs); shared invariants in §3; phase template in §4; cross-domain prep in §5; cleanup in §9. ✓

**Placeholder scan:** §7 contains 2 fully-worked tasks (A1, A2) + §7.5 explicit "placeholder — full enumeration in actual sub-plan". This is intentional per the master-plan-vs-sub-plan split announced in §2 and is consistent with the writing-plans `Scope Check` rule. The §8 stubs are explicitly stubs (not deceptive placeholders). ✓

**Type consistency:** `DescribeEntityGenToString` (§7.3) consistent with `Describe<Type>GenToString` template (§4 Phase A). Function naming `listUnitsWithContainerGen[T]` (§5 P2) consistent with cache field convention. `setRawBSONField` references match memory `gen_schema_gaps`. ✓

**Sub-plan generation reminder:** §6 table flags each domain's sub-plan as TBD. §7 explicitly tells the engineer to copy that section into a new file before executing domainmodel. §8 each subsection says "a dedicated detailed plan MUST be written before execution." ✓

---

## 13. Execution Handoff

**This master plan is complete and saved to `docs/superpowers/plans/2026-05-14-stage-3-3-domain-marathon-master.md`.**

Two execution options:

**1. Subagent-Driven (recommended)** — Generate the first sub-plan (security domain per §6 priority), then dispatch a fresh subagent per task, review between tasks, fast iteration.

**2. Inline Execution** — Generate the first sub-plan, then execute its tasks in this session using executing-plans, batch execution with checkpoints.

**Either way, the workflow is:**
1. Pick the next domain from §6 (start with security)
2. Write that domain's detailed sub-plan using this master plan + §7 template
3. Save to `docs/superpowers/plans/2026-XX-XX-stage-3-3-<domain>.md`
4. Execute the sub-plan with chosen approach
5. After Phase E lands, return here, mark domain done in §6, repeat for next domain
6. After all 6 domains done, execute §9 cross-cutting cleanup

**Which approach?**

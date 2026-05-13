# Modelsdk-Native Stage 2.6: Per-Domain Repo Implementations

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the 15 stub domain repos that Stage 2 left as interface-only (so Stage 3 cutover can compile). Pattern is established by `mdl/backend/mpr/repos/microflows.go` (standard CRUD path) and `mdl/backend/mpr/repos/pages.go` (CRUD + Mutator). One commit per domain. All work is additive — DO NOT modify legacy `MprBackend` / `MockBackend` / `*ViaModelsdk*` / `mdl/executor/`.

**Architecture:** Each domain follows the same template:

```go
// mdl/backend/mpr/repos/{domain}.go (new file)
type {domain}Repo struct {
    r     *mmpr.Reader
    w     *mmpr.Writer
    sink  writeSink     // existing sink abstraction (sink.go)
    enc   *codec.Encoder
}

// Reader half
func (r *{domain}Repo) Get(id model.ID) (*gen{D}.{Type}, error) { /* GetRawUnitBytes + decode */ }
func (r *{domain}Repo) List(moduleID model.ID) ([]*gen{D}.{Type}, error) { /* ListUnitsByType + filter by moduleID + decode */ }

// Writer half
func (r *{domain}Repo) Create(parentUUID, containmentName string, e *gen{D}.{Type}) error { /* enc.Encode + InsertUnit */ }
func (r *{domain}Repo) Update(e *gen{D}.{Type}) error { /* enc.Encode + sink.WriteUnit */ }
func (r *{domain}Repo) Delete(id model.ID) error { /* w.DeleteUnit */ }
func (r *{domain}Repo) Move(id model.ID, newParentUUID string) error { /* w.UpdateUnitContainer */ }

func New{Domain}Repository(r *mmpr.Reader, w *mmpr.Writer, sink writeSink) repos.{Domain}Repository { ... }
```

```go
// mdl/repos/testing/{domain}.go (new file)
type Recording{Domain}Repository struct {
    GetCalls    []model.ID
    ListCalls   []model.ID
    CreateCalls []{Domain}CreateCall
    UpdateCalls []*gen{D}.{Type}
    DeleteCalls []model.ID
    MoveCalls   []{Domain}MoveCall
    // optional Func overrides for each method
}
// + interface conformance: var _ repos.{Domain}Repository = (*Recording{Domain}Repository)(nil)
```

**Final wiring task** (Task 16): update `mdl/backend/mpr/factory.go` `NewExecutorContext` to wire all 15 new repos into the returned `*repos.ExecutorContext` (add 15 fields + 15 builder calls). Also update `mdl/repos/context.go` ExecutorContext struct definition.

---

## Domain inventory (15 stubs to implement)

Mapped to BSON storage names (verified via fixture probe):

| Task | Domain pkg | Storage type name(s) | Fixture units | Test approach |
|---|---|---|---|---|
| T1 | `gen/microflows` (Nanoflow) | `Microflows$Nanoflow` | 13 | Real fixture |
| T2 | `gen/pages` (Layout) | `Forms$Layout` | 22 | Real fixture |
| T3 | `gen/pages` (Snippet) | `Forms$Snippet` | 4 | Real fixture |
| T4 | `gen/domainmodels` (DomainModel) | `DomainModels$DomainModel` | 8 | Real fixture |
| T5 | `gen/projects` (Module) | `Projects$Module` (note: prefix-match yields 16 but exact may differ — probe at task time, may be `Projects$ModuleSettings` or similar; if no exact match, fall back to fresh-create + roundtrip) | 16 prefix / 0 exact | Probe + adapt |
| T6 | `gen/projects` (Folder) | `Projects$Folder` | 80 | Real fixture |
| T7 | `gen/enumerations` (Enumeration) | `Enumerations$Enumeration` | 7 | Real fixture |
| T8 | `gen/constants` (Constant) | `Constants$Constant` | 2 | Real fixture |
| T9 | `gen/workflows` (Workflow) | `Workflows$Workflow` | 0 | Fresh-create + roundtrip |
| T10 | `gen/services` (umbrella) | `Services$Consumed*Service`, etc. — domain is umbrella; treat element type as `element.Element` and accept the gen interface | 0 | Fresh-create only; document as "umbrella domain — Stage 3 may split" |
| T11 | `gen/mappings` (umbrella) | `Mappings$ImportMapping`, `Mappings$ExportMapping` — also umbrella | 0 | Fresh-create only |
| T12 | `gen/projectsettings` etc. (Settings umbrella) | `Settings$ProjectSettings`(1) + others | 1 | Real fixture for Project; fresh for others |
| T13 | `gen/security` | `Security$ProjectSecurity`(1), `Security$ModuleSecurity`(8) | 9 | Real fixture |
| T14 | `gen/images` (ImageCollection) | `Images$ImageCollection` (7); `Images$Image` prefix-only (0 exact — need to probe actual storage name) | 7 | Real fixture for Collection |
| T15 | `gen/agents` | `Agents$Agent`, `Agents$KnowledgeBase`, `Agents$Model` — umbrella | 0 | Fresh-create only |
| T16 | (factory wiring) | — | — | Smoke tests |

Tasks T1-T15 are independent (each touches one new file pair). T16 depends on all of T1-T15 (factory wires them all).

---

## Per-task template (apply to T1-T15)

For each domain task:

- [ ] **Step 1: Read the existing stub interface** at `mdl/repos/{domain}.go` to confirm method signatures + gen package import alias. If signatures need adjustment (e.g., umbrella domain wants `element.Element` not a specific type), update the interface as a minimal additive change.

- [ ] **Step 2: Probe the BSON type name** if not 100% verified:
   ```bash
   GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go run -tags ignore - <<EOF
   ... (use the same probe pattern as Stage 2.5 — open testdata/expr-checker/minimal.mpr, ConcreteReader, ListUnitsByType, count exact matches)
   EOF
   ```
   Document the actual storage name string in the implementation.

- [ ] **Step 3: Implement** at `mdl/backend/mpr/repos/{domain}.go`. Mirror `microflows.go` structure exactly. Key points:
   - Reader uses `r.GetRawUnitBytes(string(id))` + `codec.NewDecoder(codec.DefaultRegistry).Decode(bson.Raw(b))` + type assert
   - List uses `r.ListUnitsByType(typeName)` with **strict `ref.Type == typeName`** filter (Forms$Page burns Forms$PageTemplate prefix collision — same defense)
   - moduleID filter via `r.BuildContainerParent()` walk (parents map)
   - Create uses `&codec.Encoder{}.Encode(elem)` + `w.InsertUnit(string(elem.ID()), parentUUID, containmentName, typeName, contents)`
   - Update uses `enc.Encode(elem)` + `r.sink.WriteUnit(string(elem.ID()), contents)` (sink abstracts direct vs txn)
   - Delete uses `w.DeleteUnit(string(id))`
   - Move uses `w.UpdateUnitContainer(string(id), string(newParentUUID))`
   - Constructor: `New{Domain}Repository(r *mmpr.Reader, w *mmpr.Writer, sink writeSink) repos.{Domain}Repository`
   - Compile-time interface check: `var _ repos.{Domain}Repository = (*{domain}Repo)(nil)`

- [ ] **Step 4: Write Recording mock** at `mdl/repos/testing/{domain}.go`. Mirror `microflows.go` shape:
   - Per-method call slice + Func override field
   - `RecordingX{Domain}Repository.Get/List/Create/Update/Delete/Move`
   - Compile-time check: `var _ repos.{Domain}Repository = (*Recording{Domain}Repository)(nil)`

- [ ] **Step 5: Test** at `mdl/backend/mpr/repos/{domain}_test.go`:
   - **Fixture-backed domains**: TestList_FixtureCount (assert exact count), TestGetRoundTrip (decode → access Name → encode → bytes match), TestCreateRoundTrip (fresh + InsertUnit + read-back), TestUpdate_RoundTrip (modify + commit + read-back), TestDelete_Vanishes (delete + List should not see)
   - **Fresh-only domains** (Workflow / Services / Mappings / Agents): Skip Get/List with `t.Skip("no fixture units")`. Test only Create + Update + Delete + Move via fresh-construct + InsertUnit roundtrip.
   - Mock test in `mdl/repos/testing/{domain}_test.go`: instantiate, call methods, assert Recording arrays.

- [ ] **Step 6: Run** `GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/backend/mpr/repos/... ./mdl/repos/testing/... -run Test{Domain} -v` — all PASS or SKIP.

- [ ] **Step 7: Commit**: `feat(mprrepos): {Domain}Repository implementation + Recording mock (Stage 2.6 task N)`

---

## Task 16: ExecutorContext factory wiring

After T1-T15 land:

- [ ] **Step 1: Update** `mdl/repos/context.go` `ExecutorContext` struct: add 15 fields, one per domain (name follows existing pattern: `Microflows`, `Pages`, `Nanoflows`, `Layouts`, etc.).

- [ ] **Step 2: Update** `mdl/backend/mpr/factory.go` `NewExecutorContext`: instantiate each repo via its `New{Domain}Repository` constructor, assign to corresponding context field.

- [ ] **Step 3: Smoke tests** at `mdl/backend/mpr/factory_test.go`: extend `TestNewExecutorContext_AllFieldsWired` to assert each new field is non-nil; add `TestNewExecutorContext_RepoSmoke{Domain}` for each domain that has fixtures (call List, assert count > 0).

- [ ] **Step 4: Verify**: `GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/backend/mpr/... -run TestNewExecutorContext -v` — all PASS.

- [ ] **Step 5: Commit**: `feat(mprbackend): wire all 15 Stage 2.6 repos into NewExecutorContext (Stage 2.6 task 16)`

---

## Task 17: Final verification + 0-additive self-check

- [ ] `GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./...` — no errors
- [ ] `GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./... -count=1 -timeout 120s` — 0 fails (full wall < 30s with the perf-fix landed)
- [ ] **0-additive verification**:
   - `git diff --stat e5c1c163..HEAD -- mdl/backend/mpr/backend.go mdl/backend/mock/ mdl/executor/ mdl/backend/microflow.go mdl/backend/page.go mdl/backend/mutation.go` → empty
   - **Exception**: `mdl/executor/` was modified by `15f99f29 perf(executor)` for the cache fix — that's lead's parallel work, not Stage 2.6's. Subtract that diff before reporting.
   - `/usr/bin/grep -rln "mxcli/mdl/repos\|mxcli/mdl/backend/mpr/repos" mdl/executor/` → exit 1 (no matches; executor still on legacy path)

- [ ] Report all 17 commit SHAs + total Recording mocks count + total tests added to lead.

---

## Out of scope (explicit)

- Stage 3 cutover (replace executor's Backend dependency, switch handlers to repos)
- Mutator interfaces for non-Page/non-Workflow domains (no perf justification)
- Cross-domain Services (CascadeService / ReferenceService — Stage 3+)
- Auxiliary services beyond what Stage 2.5 already shipped
- API surface changes in `api/`, `modelsdk.go`, or `mxcli` CLI

---

## Self-review checklist

- [ ] All 15 stub domains have `mdl/backend/mpr/repos/{domain}.go` implementation
- [ ] All 15 domains have `mdl/repos/testing/{domain}.go` Recording mock
- [ ] Each implementation has `var _ repos.{Domain}Repository = (*...)(nil)` compile-time check
- [ ] Each Recording mock has the same compile-time check
- [ ] `NewExecutorContext` wires all 15 new repos
- [ ] No legacy file in `mdl/backend/microflow.go` / `mdl/backend/page.go` / `mdl/backend/mutation.go` / `mdl/backend/mpr/backend.go` / `mdl/backend/mock/` / `mdl/executor/*` modified by this plan (perf fix exception above)
- [ ] No executor handler imports `mdl/repos/*` (Stage 3 cutover comes later)
- [ ] Tests against real fixtures use **strict `ref.Type ==` match**, not prefix
- [ ] Each commit message follows the convention: `feat(mprrepos): {Domain}Repository implementation + Recording mock (Stage 2.6 task N)`
- [ ] Total wall-clock for `go test ./...` stays under 60s

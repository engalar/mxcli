# Stage 3.3 AgentEditor Domain — Detailed Sub-Plan

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans`. Steps use checkbox `- [ ]` syntax for trackability.
>
> Generated from `2026-05-14-stage-3-3-domain-marathon-master.md` §6 priority #6 (last) + §4 phase template + §8.5 stub. Builds on the security/javaactions/workflows/domainmodel/pages sub-plans. AgentEditor is the **most architecturally unique** Stage 3.3 domain: there is NO `modelsdk/gen/agenteditor` package. This plan diverges from the standard "swap sdk → gen" template.

**Goal:** Migrate the `agenteditor` domain off the legacy hand-written `sdk/agenteditor` package by **relocating the inner JSON-payload types to `mdl/types/agenteditor.go`** (where they belong, since they are not BSON gen types). Final state: zero `sdk/agenteditor` imports outside `sdk/mpr/` (Stage 4 territory) and zero in `modelsdk/`, `api/`, `mdl/` (except `mdl/types/`).

---

## §1 Background — Status Snapshot

The agenteditor domain has a **fundamentally different architecture** from the other five Stage 3.3 domains:

### The custom-blob mechanism (per CLAUDE.md "Implemented" + `sdk/agenteditor/types.go:1-15`)

All four agent-editor document types (Agent, Model, KnowledgeBase, ConsumedMCPService) share the same outer BSON wrapper: `CustomBlobDocuments$CustomBlobDocument`. They are distinguished by:
- `wrapper.CustomDocumentType` field — values like `"agenteditor.agent"`, `"agenteditor.model"`, `"agenteditor.knowledgebase"`, `"agenteditor.consumedMCPService"`
- `wrapper.Contents` field — a **JSON string** payload whose schema depends on the type

Reading an agent document is a **two-step decode**:
1. BSON decode of the unit → `*genCBD.CustomBlobDocument` (gen exists)
2. JSON decode of `wrapper.Contents()` string → inner Go struct (`*sdk/agenteditor.Agent` etc. — sdk-defined)

The inner types are **not BSON-mapped**; they are JSON-mapped via standard `encoding/json` tags. This is why **no `modelsdk/gen/agenteditor` package exists**.

### Two migration options

**Option I — Relocate inner types to `mdl/types/agenteditor.go`**:
The 4 inner types (`Agent`, `Model`, `KnowledgeBase`, `ConsumedMCPService`) plus their sub-types (`AgentVar`, `AgentTool`, `AgentKBTool`, `DocRef`, `ConstantRef`) move to `mdl/types/agenteditor.go`. The outer wrapper becomes `*genCBD.CustomBlobDocument` (already gen). `sdk/agenteditor` is deleted (no Stage 4 dependency on the inner JSON types — Stage 4 only uses the outer wrapper which is gen-typed).

**Option II — Build `modelsdk/gen/agenteditor`**:
Hand-write a gen-equivalent package. Higher one-time cost; doesn't match the "auto-generated from reflection data" pattern (the JSON inner types are NOT in Mendix reflection metadata since they're extension-defined).

**Plan picks Option I.** Rationale:
- Zero gen-codegen support for these types (Mendix reflection doesn't model extension-defined JSON schemas)
- The inner types are pure data shapes (no BSON, no element.Element machinery needed)
- `mdl/types/` already hosts other lite descriptors (`mdl/types/java.go::JavaAction`, `mdl/types/UnitInfo`, etc.)
- One-step relocation cleaner than parallel gen package

### Already migrated — DO NOT redo

- `mdl/backend/mpr/agenteditor_modelsdk.go` (193 LoC, 8 sdk refs) — write helpers that JSON-encode the inner type then BSON-encode the outer wrapper via `b.msdkWriter.InsertUnit("CustomBlobDocuments$CustomBlobDocument", ...)`. **Inner serialization is already JSON via `encoding/json`** — no BSON gen path needed. The migration is purely a **type-import swap** (`sdk/agenteditor` → `mdl/types`).
- `modelsdk/gen/customblobdocuments/{types,enums,refs,version}.go` — outer wrapper gen types. Verified at `customblobdocuments/types.go:26 (CustomBlobDocument)` + `:128 (CustomBlobDocumentMetadata)`.

### Still to migrate (this plan's scope)

| File | LoC | What stays | What leaves |
|---|---|---|---|
| `sdk/agenteditor/types.go` | 216 | nothing | full file (relocated to `mdl/types/agenteditor.go`) |
| `mdl/backend/infrastructure.go` | 88 | interface methods | `*agenteditor.*` types in 12 method sigs |
| `mdl/backend/mock/backend.go` | 309 | Func decls | `*agenteditor.*` in Func types |
| `mdl/backend/mock/mock_infrastructure.go` | (size TBD) | shim | sdk types in method sigs |
| `mdl/backend/mpr/backend.go` | 881 | shim wrappers | sdk types in agent-editor method sigs |
| `mdl/backend/mpr/agenteditor_modelsdk.go` | 193 | inner JSON+BSON write logic | sdk-typed inputs in 8 sites (just the import + type-name swaps) |
| `mdl/executor/cmd_agenteditor_agents.go` | 260 | nothing | full file (4 refs) |
| `mdl/executor/cmd_agenteditor_kbs.go` | 147 | nothing | full file (2 refs) |
| `mdl/executor/cmd_agenteditor_mcpservices.go` | 131 | nothing | full file (2 refs) |
| `mdl/executor/cmd_agenteditor_models.go` | 257 | nothing | full file (6 refs) |
| `mdl/executor/cmd_agenteditor_write.go` | 293 | nothing | full file (11 refs) |
| `mdl/executor/cmd_agenteditor_mock_test.go` | (size TBD) | needs migration | sdk fixtures |
| `mdl/executor/cmd_error_mock_test.go` | (small) | needs migration | one fixture |
| `mdl/executor/cmd_json_mock_test.go` | (small) | needs migration | one fixture |
| `mdl/executor/validate_duplicates_test.go` | (small) | needs migration | one fixture |
| `sdk/mpr/parser_customblob.go` | (Stage 4) | — | DO NOT touch |
| `sdk/mpr/reader_agenteditor.go` | (Stage 4) | — | DO NOT touch |
| `sdk/mpr/serialize_exports.go` | (Stage 4) | — | DO NOT touch |
| `sdk/mpr/writer_agenteditor_agent.go` | (Stage 4) | — | DO NOT touch |
| `sdk/mpr/writer_agenteditor_kb.go` | (Stage 4) | — | DO NOT touch |
| `sdk/mpr/writer_agenteditor_mcpservice.go` | (Stage 4) | — | DO NOT touch |
| `sdk/mpr/writer_agenteditor_model.go` | (Stage 4) | — | DO NOT touch |
| `sdk/mpr/writer_customblob.go` | (Stage 4) | — | DO NOT touch |

### Why this domain is priority #6 (last)

- **Custom blob mechanism** is unique — defer until other 5 domains' patterns proven so the migration team knows when to deviate from the template
- **Inner JSON types** mean the dual-encode (BSON outer + JSON inner) doesn't fit the standard "gen accessor" pattern
- **Smallest in raw caller density** (~14 in-scope files) but architecturally distinct
- **Mendix 11.9+ requirement** + AgentEditorCommons module dependency (per CLAUDE.md)
- **Defer last** so heavy widget-tree patterns from pages (priority #5) don't need to coordinate with this unusual shape

---

## §2 Pre-Flight Survey Results

### S2.1 sdk/agenteditor importers (14 in-scope, 8 Stage 4) — listed in §1 table

### S2.2 sdk/agenteditor surface

`sdk/agenteditor/types.go` (216 LoC):
- Constants: `CustomTypeAgent`, `CustomTypeModel`, `CustomTypeKnowledgeBase`, `CustomTypeConsumedMCPService` (BSON `CustomDocumentType` values)
- Constants: `ReadableAgent`, `ReadableModel`, etc. (BSON Metadata.ReadableTypeName)
- Constant: `CreatedByExtensionID = "extension/agent-editor"`
- Type: `DocRef` (cross-document reference: ID + qualified name)
- Type: `ConstantRef` (constant reference)
- Type: `Model` (LLM definition: name, provider, parameters)
- Type: `KnowledgeBase` (vector store config)
- Type: `ConsumedMCPService` (MCP service definition)
- Type: `Agent` (agent definition: model, prompt, tools, KB tools)
- Type: `AgentVar`, `AgentTool`, `AgentKBTool` (agent sub-types)

All types use standard `json:"name"` field tags for JSON encoding. They embed `model.BaseElement` for Mendix unit infrastructure.

### S2.3 Read funcs (per executor file)

`cmd_agenteditor_agents.go` (260 LoC):
- `listAgents(ctx, moduleName)` — calls `ctx.Backend.ListAgents()` returning `[]*agenteditor.Agent`
- `describeAgent(ctx, name)` — calls `ctx.Backend.GetAgentByName(qn)` returning `*agenteditor.Agent`
- `formatAgent`, `formatAgentVars`, `formatAgentTools` helpers

`cmd_agenteditor_models.go` (257 LoC):
- `listModels`, `describeModel` + format helpers

`cmd_agenteditor_kbs.go` (147 LoC):
- `listKnowledgeBases`, `describeKnowledgeBase`

`cmd_agenteditor_mcpservices.go` (131 LoC):
- `listConsumedMCPServices`, `describeConsumedMCPService`

### S2.4 Write funcs (`cmd_agenteditor_write.go` 293 LoC, 11 refs)

- `execCreateAgent`, `execDropAgent` — top-level dispatch
- `execCreateModel`, `execDropModel`
- `execCreateKnowledgeBase`, `execDropKnowledgeBase`
- `execCreateConsumedMCPService`, `execDropConsumedMCPService`
- All call into `mdl/backend/mpr/agenteditor_modelsdk.go::create<X>ViaModelsdk` helpers

### S2.5 Backend interface methods (`mdl/backend/infrastructure.go`)

```go
InfrastructureBackend (or similar — verify name in A0):
  ListAgents() ([]*agenteditor.Agent, error)
  GetAgentByName(qn string) (*agenteditor.Agent, error)
  GetAgentByID(id model.ID) (*agenteditor.Agent, error)
  CreateAgent(a *agenteditor.Agent) error
  UpdateAgent(a *agenteditor.Agent) error
  DeleteAgent(id model.ID) error

  ListModels(), GetModelByName(...), CreateModel(...), DeleteModel(...)
  ListKnowledgeBases(), GetKnowledgeBaseByName(...), Create..., Delete...
  ListConsumedMCPServices(), Get..., Create..., Delete...
```

~16 sdk-typed methods. After Option I migration, signatures change from `*agenteditor.Agent` to `*types.Agent` (pure type-name swap; no behavior change).

### S2.6 ExecContext field

No `ctx.AgentEditor` field today. Phase A0 evaluation: do we need a repo abstraction for these read paths?

**Decision: NO repo needed.** The reads go through `ListAgents`/`GetAgentByName` etc. which already exist on the backend. The two-step decode (BSON wrapper → JSON inner) lives in `mdl/backend/mpr/agenteditor_modelsdk.go` — there's no benefit from an additional `ctx.AgentEditor` abstraction since the gen `*CustomBlobDocument` is the outer wrapper (gen-typed) and the inner types are JSON pure-Go structs (not gen-needing).

This is the **second major divergence** from the standard plan template (the first being "no gen package").

### S2.7 MockBackend conformance — same `nil, nil` pattern; Phase C5 brings into compliance.

### S2.8 Storage type names

- `CustomBlobDocuments$CustomBlobDocument` — outer wrapper (gen-typed, line 26 of `modelsdk/gen/customblobdocuments/types.go`)
- Inner type discriminated by `CustomDocumentType` field: `agenteditor.agent`, `agenteditor.model`, `agenteditor.knowledgebase`, `agenteditor.consumedMCPService`

### S2.9 AgentEditorCommons module dependency

Per CLAUDE.md: agent-editor documents require the `AgentEditorCommons` module + Mendix 11.9+. The version-awareness skill (`.claude/skills/version-awareness.md`) handles the `min_version` check at execute time. Migration does NOT change this; the version gate is in `sdk/versions/mendix-{10,11}.yaml` and `cmd_agenteditor_*.go` calls `checkFeature()` before any write.

---

## §3 Risks Specific to AgentEditor Domain

| # | Risk | Impact | Mitigation |
|---|---|---|---|
| R1 | **No gen package exists** (§S2.6) — standard "sdk → gen" template doesn't apply | Critical — plan structure diverges from template | Plan picks **Option I (relocate inner types)** explicitly. §1 + §2.6 documents the rationale. Phase A0 is much shorter than other domains (no repo creation) |
| R2 | **JSON encoding behavior preserved** — relocation is type-import swap; field tags + behaviors must stay byte-identical. Adding/removing/reordering fields breaks Mendix Studio Pro JSON parser | Critical — silent data corruption | `git mv` (well, `git rm + git add`) preserves blame. Tests for round-trip JSON identity in A0.S2 |
| R3 | **Dollar-quoted multi-line prompts** (per CLAUDE.md "Implemented") — Agent prompt strings can contain `$$...$$` MDL syntax. The inner JSON-decoded `Agent.Prompt` field already handles this. Migration must NOT touch prompt parsing | Medium | Verify in cmd_agenteditor_agents_test.go that a fixture agent with dollar-quoted prompt round-trips correctly |
| R4 | **Stage 4 boundary** — 8 sdk/mpr files import `sdk/agenteditor` for the BSON parser side. Stage 4 owns the BSON dispatch (`parser_customblob.go` reads the wrapper, calls `parseAgent`/`parseModel`/etc. via the `CustomDocumentType` switch) | Medium | Acceptance grep excludes `^./sdk/mpr/`; deprecation header on `sdk/agenteditor` if Stage 4 hasn't landed (same escape hatch as other domains) |
| R5 | **AgentEditorCommons module + version gate** — the migration must not break the version check | Low — the check is in executor write funcs (preserved) | Tests still gate on `checkFeature("agent_editor")` per `.claude/skills/version-awareness.md` |
| R6 | **JSON wire format compatibility with Studio Pro** — the inner JSON payload IS readable by Mendix Studio Pro's Agent Editor extension. Field renames or tag changes break that round-trip | Critical | Migration is type-relocation only. NO field renames, NO tag changes. Diff the relocated file against the original to verify zero semantic delta |
| R7 | **DocRef and ConstantRef share types with other Mendix concepts**? — verify these types are agenteditor-specific (not used outside `sdk/agenteditor`) | Low | Survey (§S2.10) confirms |

### S2.10 DocRef/ConstantRef usage check

```bash
grep -rn '"github.com/mendixlabs/mxcli/sdk/agenteditor"' . --include="*.go" | grep -v "^./sdk/" | grep -E "DocRef|ConstantRef"
```

Verified: usage of `DocRef` / `ConstantRef` is confined to:
- The 4 inner type definitions inside `sdk/agenteditor/types.go` (their own embedded use)
- The 5 executor `cmd_agenteditor_*.go` files (consumers)
- `mdl/backend/mpr/agenteditor_modelsdk.go` (write helpers)
- The test files

No external code references these types. Safe to relocate to `mdl/types/agenteditor.go`.

---

## §4 Phase A — Type Relocation (NOT a read-path migration like other domains)

Goal: relocate `sdk/agenteditor/types.go` to `mdl/types/agenteditor.go` with **zero semantic change**. All consumers update their import path. No new gen accessors, no formatter rewrites.

### Task A0: Create `mdl/types/agenteditor.go` as identical-content sibling

**Files:**
- Create: `mdl/types/agenteditor.go` — full copy of `sdk/agenteditor/types.go`, same constants, same struct definitions, same JSON tags, same methods. Only difference: package declaration changes from `package agenteditor` to `package types`.

#### A0.S1: Verify byte-level diff = "package + import only"

```bash
# After creating mdl/types/agenteditor.go:
diff -u sdk/agenteditor/types.go mdl/types/agenteditor.go
# Expected diff: only the package line + import path normalization
```

#### A0.S2: JSON round-trip test

```go
// mdl/types/agenteditor_test.go
func TestAgent_JsonRoundtrip_PreservesFields(t *testing.T) {
    fixture := loadFixtureAgent(t) // any existing agent fixture
    bytes, err := json.Marshal(fixture)
    if err != nil { t.Fatalf("marshal: %v", err) }
    var got types.Agent
    if err := json.Unmarshal(bytes, &got); err != nil { t.Fatalf("unmarshal: %v", err) }
    if !reflect.DeepEqual(fixture, &got) { t.Errorf("roundtrip drift: got %+v, want %+v", &got, fixture) }
}

func TestModel_JsonRoundtrip(t *testing.T) { /* same pattern */ }
func TestKnowledgeBase_JsonRoundtrip(t *testing.T) { /* same */ }
func TestConsumedMCPService_JsonRoundtrip(t *testing.T) { /* same */ }
```

#### A0.S3: RED — these tests should pass immediately (since `mdl/types/agenteditor.go` is byte-identical)

#### A0.S4: GREEN

#### A0.S5: Build + full test gate

#### A0.S6: Commit

```bash
git add mdl/types/agenteditor.go mdl/types/agenteditor_test.go
git commit -m "$(cat <<'EOF'
feat(types): Stage 3.3.6.A0 — clone agenteditor types to mdl/types/

Adds mdl/types/agenteditor.go as a byte-identical sibling of
sdk/agenteditor/types.go (only the package declaration differs).
No consumer migrated yet; subsequent A1-A6 commits switch one
consumer file at a time. Phase E removes sdk/agenteditor.

Tests verify JSON round-trip identity for all 4 inner types
(Agent, Model, KnowledgeBase, ConsumedMCPService).

Note: this domain has NO modelsdk/gen/agenteditor package — agent
documents use a custom-blob mechanism where the outer wrapper is
gen (CustomBlobDocuments$CustomBlobDocument) and the inner Contents
is JSON. The 4 inner types are JSON-mapped Go structs, not BSON gen
types, so they belong in mdl/types/ rather than modelsdk/gen/.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

(There is NO Phase A1–A6 like other domains — see §6 Phase C for the consumer migration. Phase A is just one task: type relocation prep.)

---

## §5 Phase B — Visualization

**SKIPPED.** AgentEditor domain has no `cmd_agenteditor_elk.go` / `cmd_agenteditor_mermaid.go`. Verified by `find mdl/executor/ -name "*agenteditor*"` — only show/describe/write files exist.

---

## §6 Phase C — Consumer Migration (the bulk of the work for this domain)

For each non-executor file that imports `sdk/agenteditor`, switch import path. Per file ONE commit.

### Task C1: `mdl/backend/infrastructure.go` + `mdl/backend/mpr/backend.go`

Switch interface return types and shim wrappers from `*agenteditor.Agent` → `*types.Agent` etc.

**Strategy:** Unlike previous plans, this is NOT additive (no `*Gen` siblings) — it's a direct import swap. Reason: there are no gen types to add; the new path is identical except for the import.

**Files:**
- Modify: `mdl/backend/infrastructure.go` (12 method sigs)
- Modify: `mdl/backend/mpr/backend.go` (shim methods)

#### C1.S1: Test — open one existing test that calls `b.ListAgents()`; the type assertion changes from `*agenteditor.Agent` to `*types.Agent`
#### C1.S2: RED if not changed
#### C1.S3: Replace import + type names
#### C1.S4: GREEN
#### C1.S5: Build + full test gate
#### C1.S6: Commit

```bash
git commit -m "refactor(backend): Stage 3.3.6.C1 — InfrastructureBackend uses types.Agent etc."
```

### Task C2: `mdl/backend/mpr/agenteditor_modelsdk.go` (8 refs)

Switch all `agenteditor.X` → `types.X`. The JSON encode + BSON wrapper write logic is unchanged.

**Files:** `mdl/backend/mpr/agenteditor_modelsdk.go`

#### C2.S1–C2.S6: TDD, Commit `refactor(backend/mpr): Stage 3.3.6.C2 — agenteditor_modelsdk uses types.Agent`

### Task C3: `mdl/backend/mock/mock_infrastructure.go` + `backend.go`

Switch Func-field types + method sigs.

**Files:** both mock files

#### C3.S1–C3.S6: TDD, Commit `refactor(mock): Stage 3.3.6.C3 — MockBackend infrastructure uses types.Agent`

### Task C4: MockBackend conformance audit (master plan §5 P3)

Bring `mdl/backend/mock/mock_infrastructure.go` into compliance: `nil, nil` → `nil, fmt.Errorf("MockBackend.X not configured")`.

**Files:** `mdl/backend/mock/mock_infrastructure.go`

#### C4.S1–C4.S6: TDD, Commit `refactor(mock): Stage 3.3.6.C4 — MockBackend infrastructure stubs return descriptive errors`

### Task C5: `mdl/executor/cmd_agenteditor_agents.go`

Switch import + 4 type refs.

#### C5.S1–C5.S6: TDD, Commit `refactor(executor): Stage 3.3.6.C5 — cmd_agenteditor_agents uses types.Agent`

### Task C6: `mdl/executor/cmd_agenteditor_models.go` (6 refs)

Same pattern.

#### C6.S1–C6.S6: TDD, Commit `refactor(executor): Stage 3.3.6.C6 — cmd_agenteditor_models uses types.Model`

### Task C7: `mdl/executor/cmd_agenteditor_kbs.go` (2 refs)

#### C7.S1–C7.S6: TDD, Commit `refactor(executor): Stage 3.3.6.C7 — cmd_agenteditor_kbs uses types.KnowledgeBase`

### Task C8: `mdl/executor/cmd_agenteditor_mcpservices.go` (2 refs)

#### C8.S1–C8.S6: TDD, Commit `refactor(executor): Stage 3.3.6.C8 — cmd_agenteditor_mcpservices uses types.ConsumedMCPService`

### Task C9: `mdl/executor/cmd_agenteditor_write.go` (11 refs)

Largest single executor consumer. Switch all 11 refs.

#### C9.S1–C9.S6: TDD with full round-trip test (CREATE AGENT, CREATE MODEL, etc. through `mx check`), Commit `refactor(executor): Stage 3.3.6.C9 — cmd_agenteditor_write uses types.* (full round-trip)`

### Task C10: Mock test fixture migration

Migrate sdk-typed fixtures in:
- `mdl/executor/cmd_agenteditor_mock_test.go`
- `mdl/executor/cmd_error_mock_test.go`
- `mdl/executor/cmd_json_mock_test.go`
- `mdl/executor/validate_duplicates_test.go`

ONE commit (cross-file consistency).

#### C10.S1–C10.S6: TDD, Commit `test(executor): Stage 3.3.6.C10 — migrate agenteditor mock fixtures to types.* package`

---

## §7 Phase D — Write Path Migration

**MOSTLY SKIPPED.** Unlike other domains, the write path migration is just the type-import swap (already covered by C2 + C9 above). There are no AST → gen builders to write because there is no gen package.

### Task D1: Verify dispatcher routing still works

After C2 + C9, the executor's `register_stubs.go` continues to call the same `execCreateAgent` etc. functions — those functions internally now accept `*types.Agent`. No dispatcher changes needed.

#### D1.S1: Locate
```bash
grep -nE "execCreateAgent\b|execDropAgent\b|execCreateModel\b|execDropModel\b|execCreateKnowledgeBase\b|execDropKnowledgeBase\b|execCreateConsumedMCPService\b|execDropConsumedMCPService\b" mdl/executor/register_stubs.go
```

#### D1.S2: Verify build green — no changes needed
#### D1.S3: NO COMMIT (verification only)

### Task D2: `mx check` smoke test on round-tripped MPR

Create a test fixture (or use existing) that:
1. CREATE MODEL "GPT-4" with parameters
2. CREATE KNOWLEDGE BASE "Docs" with vector store config
3. CREATE CONSUMED MCP SERVICE "WebSearch"
4. CREATE AGENT "ResearchBot" with above model/KB/MCP refs + a multi-line dollar-quoted prompt + AgentVar + AgentTool + AgentKBTool

After CREATE, run `mx check` on the round-tripped MPR. Assert no errors.

**Files:** `mdl-examples/agenteditor-tests/agent-roundtrip.test.mdl` (new)

#### D2.S1: Test exists (write the .test.mdl)
#### D2.S2: Run via `mxcli test`; expected GREEN
#### D2.S3: If RED, debug; if GREEN, no commit needed (test file should have been added in C9 anyway)

---

## §8 Phase E — Cleanup

### Task E1: Retire `FullBackend` deprecated sdk-typed methods

(Optional — since this domain didn't add `*Gen` siblings, there's nothing to retire on the FullBackend interface. The interface methods themselves stay; only their argument types changed.)

**SKIPPED — no commits.**

### Task E2: Delete `sdk/agenteditor` package — CONDITIONAL on Stage 4

**Files:**
- Delete: `sdk/agenteditor/types.go`

#### E2.S1: Final acceptance grep
```bash
grep -rln '"github.com/mendixlabs/mxcli/sdk/agenteditor"' . --include="*.go" | grep -v "^./sdk/mpr/"
```
Expected: empty.

#### E2.S2: Verify Stage 4 boundary
```bash
git log -1 --oneline -- sdk/mpr/parser_customblob.go sdk/mpr/reader_agenteditor.go sdk/mpr/writer_agenteditor_*.go sdk/mpr/writer_customblob.go
```
If Stage 4 has rewritten the parser/writer side BSON-natively + dropped the sdk import, proceed.

#### E2.S3a (Stage 4 done): Delete + build + test gate + Commit `refactor: Stage 3.3.6.E2 — delete sdk/agenteditor package`

#### E2.S3b (Stage 4 NOT done): Add deprecation header

```go
// sdk/agenteditor/types.go top:
// DEPRECATED — Stage 3.3.6 (2026-05-XX): all in-scope consumers (mdl/, api/, modelsdk/)
// migrated to mdl/types/agenteditor.go. The package is kept ONLY because
// sdk/mpr/parser_customblob.go and sdk/mpr/writer_agenteditor_*.go still
// import it. Will be deleted after Stage 4 rewrites those files.
```

```bash
git commit -m "refactor: Stage 3.3.6.E2 — deprecation header on sdk/agenteditor (Stage 4 boundary)"
```

### Task E3: Final acceptance verification

#### E3.S1: Acceptance greps
```bash
grep -rln '"github.com/mendixlabs/mxcli/sdk/agenteditor"' . --include="*.go" | grep -v "^./sdk/mpr/"
grep -rln '"github.com/mendixlabs/mxcli/sdk/agenteditor"' modelsdk/ --include="*.go"
grep -rln '"github.com/mendixlabs/mxcli/sdk/agenteditor"' api/ --include="*.go"
```
All three: empty.

#### E3.S2: Run agenteditor-affecting tests
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/executor/ ./mdl/backend/mpr/ ./mdl/types/ -run "Agent|Model|KnowledgeBase|MCP|agenteditor" -count=1 -v
```

#### E3.S3: Update memory `project_stage_3_3_agenteditor_complete.md` with final stats

#### E3.S4: No commit needed

---

## §9 Acceptance Criteria

- [ ] `grep -rln '"github.com/mendixlabs/mxcli/sdk/agenteditor"' . --include="*.go" | grep -v "^./sdk/mpr/"` returns 0 lines
- [ ] All `Test*Agent*`, `Test*Model*`, `Test*KnowledgeBase*`, `Test*MCP*` tests pass:
      `GOPROXY=... ~/go1.26/bin/go test ./mdl/executor/ ./mdl/backend/mpr/ ./mdl/types/ -run "agenteditor|Agent|Model|KnowledgeBase|MCP" -count=1 -v`
- [ ] **`mx check` smoke test passes** (D2.S2) on a round-tripped MPR with: CREATE MODEL, CREATE KNOWLEDGE BASE, CREATE CONSUMED MCP SERVICE, CREATE AGENT (with model/KB/MCP refs + dollar-quoted prompt + AgentVar + AgentTool + AgentKBTool)
- [ ] **JSON round-trip identity** (A0.S2): all 4 inner types preserve all fields after JSON marshal+unmarshal cycle
- [ ] **No JSON wire-format change**: `git diff mdl/types/agenteditor.go sdk/agenteditor/types.go` shows only `package` declaration delta + import path
- [ ] **Studio Pro Agent Editor still opens** documents created by `mxcli create agent` (manual verification — sniff test on a fresh project)
- [ ] Full repo build: `GOPROXY=... ~/go1.26/bin/go build ./...`
- [ ] Full repo test suite: `GOPROXY=... ~/go1.26/bin/go test ./... -count=1 -timeout 240s`
- [ ] `sdk/agenteditor/` directory is gone (or, if Stage 4 hasn't landed, kept with deprecation header)

---

## §10 Estimated Commit Count + Sequencing

| Phase | Tasks | Commits | Cumulative |
|---|---|---|---|
| A — Type relocation | A0 | 1 | 1 |
| B — Visualization | (skipped) | 0 | 1 |
| C — Consumer migration | C1, C2, C3, C4, C5, C6, C7, C8, C9, C10 | 10 | 11 |
| D — Write path | D1 (verify-only), D2 (test-only — bundled into C9) | 0 | 11 |
| E — Cleanup | E1 (skipped), E2, E3 | 1 (E3 verify-only) | 12 |

**Estimated total: ~12 commits** (within master plan §6 row #6's 30–45 range — much lower because the migration is type-relocation, not gen-rebuild). The 30–45 estimate in the master plan assumed the standard sdk→gen template; this domain's unique architecture cuts the work in half.

**Sequencing rationale:**
- A0 first — load-bearing relocation
- C1 (interface change) BEFORE consumer file migrations C2–C9
- C4 (mock conformance audit) BEFORE C10 (fixtures consume new mock contract)
- C9 (the heaviest write consumer) bundled with the D2 round-trip smoke test
- C10 mock fixture migration after all consumer files migrated
- E2 conditional on Stage 4

**Per-session checkpoints:** commit after each task; if interrupted mid-C, the partial state is fine — both old (`sdk/agenteditor`) and new (`mdl/types`) packages coexist until E2.

---

## §11 Coordination With Stage 4 Team

### Stage 4's territory (8 sdk/mpr files)
- `sdk/mpr/parser_customblob.go` — outer wrapper BSON parser
- `sdk/mpr/reader_agenteditor.go` — agent document reader (calls into customblob parser, then JSON-decodes)
- `sdk/mpr/writer_customblob.go` — outer wrapper BSON writer
- `sdk/mpr/writer_agenteditor_agent.go`, `writer_agenteditor_kb.go`, `writer_agenteditor_mcpservice.go`, `writer_agenteditor_model.go` — type-specific JSON-encode + BSON-write helpers
- `sdk/mpr/serialize_exports.go` — re-exports

### Stage 3.3.6 commitment
**Stage 3.3.6 will NOT modify any file under `sdk/mpr/`.** E2 deletion is conditional on Stage 4 having rewritten the 8 sdk/mpr files BSON-natively (using `gen/customblobdocuments` outer wrapper + JSON inner via `mdl/types`). If Stage 4 hasn't, the package stays with deprecation header.

### Risk of merge collision
**Zero.** Stage 3.3.6 touches `mdl/`, `mdl/types/agenteditor.go` (new), nothing else. Stage 4 touches `sdk/mpr/`. **No file overlap.** The cleanest collision-risk profile of any Stage 3.3 sub-plan.

### Communication
Team-lead should mention to Stage 4:
1. The 8 sdk/mpr files are the longest list per agenteditor (relative to its tiny in-scope surface). Coordinate timing.
2. Stage 4's rewrite should consume `mdl/types/agenteditor.go` (after this plan lands) and the gen `customblobdocuments` package — NOT `sdk/agenteditor` (which will be deleted/deprecated).
3. JSON wire format must be byte-identical (R6) — Stage 4's parser tests should diff against pre-migration fixtures.

---

## §12 Self-Review Checklist (skill-required)

**Spec coverage:** §4 covers the type relocation (A0). §5 visualization N/A. §6 covers all 14 in-scope consumer files: backend × 2 (interface + shim), mpr × 1 (write helpers), mock × 2 (declarations + shims), executor × 5 (4 read + 1 write), test × 4 (mock fixtures). §7 write path mostly skipped (no gen rebuild needed). §8 cleanup. §11 Stage 4 boundary. ✓

**Type consistency:** No gen accessor names to verify (no gen package — §S2.6). Inner types stay JSON-encoded structs; only the package import changes. The byte-level diff verification in A0.S1 ensures no semantic drift. ✓

**Risk surfacing:** R1 unique architecture → §1 + §2.6 explicit explanation; plan structure adapted. R2 JSON encoding preservation → A0.S2 round-trip test. R3 dollar-quoted prompts → C9 fixture covers. R4 Stage 4 boundary → §11 + E2 escape hatch. R5 version gate preserved → no change to executor write funcs. R6 Studio Pro JSON wire format → manual sniff test in §9. R7 DocRef/ConstantRef external usage → §S2.10 verified. ✓

**TDD discipline:** Every task A0, C1–C10, E2 starts with TDD. C9 explicitly includes full round-trip with `mx check` smoke test. ✓

**Commit hygiene:** Each commit single-concern (1 file or coherent group). 10 consumer files = 10 commits (no bundling because each file is independent). C10 (test fixtures) bundled because cross-file consistency required. Commit messages use HEREDOC. No `--no-verify`. ✓

**No public-API break without approval:** No `modelsdk.go` or `api/` files touched. `mdl/backend/infrastructure.go` interface change in C1 is internal (return-type rename from sdk to types — semantically equivalent). ✓

**Cache discipline:** N/A — no `ctx.AgentEditor` repo added (per §S2.6 decision). The existing `ctx.Backend.ListAgents()` etc. paths are unchanged in semantic. ✓

**Plan-template adaptation:** Plan diverges from the standard sdk→gen template because this domain has no gen package. Divergence is documented explicitly in §1 + §2.6 + §10 sequencing notes. ✓

---

## §13 Execution Notes (Wave Concurrency)

### Strictly serial (lead does directly)
- **A0** (type relocation) — load-bearing
- **C1** (interface change) — must precede consumer migrations
- **C4** (mock conformance) — must precede C10 fixtures
- **E2** (sdk delete) — strictly serial

### Concurrent-safe (parallel teammates writing independent files)
- **C2 (mpr/agenteditor_modelsdk.go), C5 (cmd_agenteditor_agents.go), C6 (cmd_agenteditor_models.go), C7 (cmd_agenteditor_kbs.go), C8 (cmd_agenteditor_mcpservices.go)** — independent files; up to 4 parallel teammates, each writing only one file
- **C3, C9** — touch different mock + executor files; parallel-safe with each other and with C2/C5–C8

### Critical / complex (single teammate + reviewer)
- **A0** — byte-level diff verification + JSON round-trip tests. Single teammate + lead `diff -u` inspection.
- **C9** — round-trip with `mx check` on real fixture. Single teammate + lead Studio Pro sniff test on output project.

### Safety rule
- Per memory `feedback_multi_agent_worktree_concurrency`: NEVER concurrent edits to same file.
- Per master plan §3.4: NEVER push during execution.

---

## §14 Open Questions for the User Before Execution Starts

1. **Option I (relocate to mdl/types) vs Option II (build modelsdk/gen/agenteditor)** — plan picks Option I (§1). Confirm before A0. If Option II picked, plan grows from 12 commits to ~30–45 commits (matching master plan §6 estimate) and requires custom codegen support for extension-defined JSON schemas. **Strongly recommend Option I.**

2. **Stage 4 dependency** — same E2 escape-hatch pattern as other plans. Confirm acceptable.

3. **`mx check` smoke test on Studio Pro round-trip** (R6) — should we require manual Studio Pro inspection (open the .mpr in actual Studio Pro 11.9+ with AgentEditor extension) before E2, or is `mx check` headless validation sufficient? Default = `mx check` only; manual sniff test deferred to user discretion since Studio Pro requires a Windows or macOS box.

4. **AgentEditorCommons module fixture** (R5) — D2 smoke test requires a fixture project that has AgentEditorCommons installed + Mendix 11.9+ MxBuild. Confirm the test infrastructure has access; if not, add a setup step in `mdl-examples/agenteditor-tests/README.md`.

These should be resolved BEFORE A0. (1) is the most critical (changes plan scope by ~3x).

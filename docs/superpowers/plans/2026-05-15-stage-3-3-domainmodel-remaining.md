# Stage 3.3.4 Domainmodel — Remaining Work Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the remaining 48 sdk/domainmodel importers (out of original 49) via 5 sequential phases. C9 (executor cache → gen) is the highest-priority unblocker for the parallel pages plan's Phase 2. system_module.go BSON-native rewrite (E3) is the final Stage 4 boundary breakthrough.

**Architecture:** Domainmodel team's Stage 3.3.4 work paused 2026-05-14 21:31 after C2.e. Read path + early write path (D1.a-e + D2 + D4 + D5.assoc + D8.entity + C2.b/c/d/e + C3 + C8.oql + D9.drop) landed but didn't fully retire the legacy sdk surface. Many writes still bridge gen→sdk via createXxxViaModelsdk helpers. This plan completes the C-phase consumer migration (especially C7/C9 which unblock pages R9), finishes Phase D write path, runs Phase E cleanup including the system_module.go rewrite that's been deferred since project start.

**Tech Stack:**
- Go 1.26 (`~/go1.26/bin/go`), `GOPROXY=https://mirrors.aliyun.com/goproxy/,direct`
- `modelsdk/gen/domainmodels` types (already in tree)
- `mdl/backend/mpr/repos/domainmodels.go` (gen-native repo, real Create/Update/Delete impl per nanoflow template)
- BSON codec via `modelsdk/codec`

---

## §1 Status Snapshot (2026-05-15)

**Already done** by stage-3-3-domainmodel team (~14 commits between 2026-05-14 ~14:00 and 21:31):
- A0/A1/A2/A3/A4/A5/A6 read path complete
- B1/B2/B3 visualization (ELK + Mermaid + dispatcher)
- C1 additive gen-typed FullBackend methods
- C2.a/b/c/d/e read consumers (catalog Reader + buildEntityPermissions + listModules + describeModule + execDropModule)
- C3 security matrix entity walk on gen
- C5 MockBackend domainmodel stubs descriptive errors
- C8.oql OqlQueryPlanELK takes oqlQuery string (no longer needs domain model)
- D1.a/b/c/d/e AST→gen builders (attribute type, attribute, entity, generalization, association, event handler, index, validation rules)
- D2 execCreateEntityGen
- D4 execDropEntityGen
- D5.assoc CreateAssociationGen via gen→sdk bridge
- D8.entity CreateEntityGen + UpdateEntityGen via gen→sdk bridge
- D9.drop DropEntity dispatcher cuts to execDropEntityGen

**Remaining 48 importers split into 6 categories**:

| Category | Importers | Files |
|---|---|---|
| **Cat-A C7 unblocks pages R9** | 5 | cmd_pages_builder.go, cmd_pages_builder_v3.go, cmd_pages_builder_input_filters.go, executor.go (executorCache.domainModels), helpers.go (genDomainModelHelpers) |
| **Cat-B C-phase consumer remainder** | 14 | cmd_modules.go, cmd_entities.go, cmd_entities_describe.go, cmd_entities_access.go, cmd_associations.go, cmd_domainmodel_elk.go, cmd_mermaid.go, cmd_move.go, cmd_diff_mdl.go, cmd_odata.go, cmd_import.go, cmd_contract.go, cmd_security_gen.go, cmd_structure.go, cmd_structure_gen.go, oql_type_inference.go, flowbuilder_assoc_lookup_gen.go, flowbuilder_actions_retrieve_gen.go |
| **Cat-C catalog migration** | 4 | catalog/builder.go, builder_modules.go, builder_permissions.go, builder_permissions_test.go |
| **Cat-D Phase D write path remainder** | 4 | mpr/domainmodel_modelsdk.go (createXxxViaModelsdk legacy), mpr/domainmodel_modelsdk_gen.go, mpr/modules_modelsdk.go, mpr/delete_move_modelsdk.go, mpr/mf_page_modelsdk.go |
| **Cat-E E1 backend interface retirement** | 4 | backend/domainmodel.go, backend/mpr/backend.go, backend/mock/backend.go, backend/mock/mock_domainmodel.go |
| **Cat-F Test fixtures** | 12 | cmd_modules_mock_test, cmd_mermaid_mock_test, cmd_associations_mock_test, cmd_entities_mock_test, cmd_odata_mock_test, cmd_rename_mock_test, cmd_security_mock_test, cmd_write_handlers_mock_test, validate_duplicates_test, mock_test_helpers_test, mpr/domainmodel_modelsdk_test, mpr/security_entity_access_gen_test |
| **Stage 4 sdk/mpr boundary** | (excluded from 48 grep) | sdk/mpr/system_module.go + parser_domainmodel.go + writer_domainmodel.go (Stage 4 territory; Phase E3 rewrites system_module.go BSON-natively per §14 Q2 in-plan decision) |

---

## §2 Phase 1 — C7+C9 (Pages R9 Unblock, ~5hr) — TOP PRIORITY

**Why first:** The parallel `2026-05-15-stage-3-3-pages-remaining.md` plan's Phase 2 (R9 cmd_pages_builder family, 6 importers) is fully blocked until this lands. Doing this first unlocks parallel pages work.

### Task 1.1: C9 — executorCache.domainModels gen field (1 commit, ~1hr)

**Files:**
- Modify: `mdl/executor/executor.go` (executorCache.domainModels field type)
- Modify: `mdl/executor/helpers.go` (any executorCache.domainModels accessor)

- [ ] **Step 1.1.1: Inspect current cache field**

```bash
grep -n "domainModels\s*\[\]" mdl/executor/executor.go | head -5
grep -n "ctx.Cache.domainModels" mdl/executor/*.go | head -10
```

Expected: `domainModels []*domainmodel.DomainModel` field + 5-10 callers.

- [ ] **Step 1.1.2: Add gen sibling field**

In `mdl/executor/executor.go::executorCache` struct:

```go
type executorCache struct {
    // existing
    domainModels []*domainmodel.DomainModel

    // ── Stage 3.3.4.C9 gen-typed sibling ─────────────────────────────
    domainModelsGen []*genDm.DomainModel
}
```

(Don't delete legacy field yet — Phase 2 consumer migration switches callers, then Phase E retires the field.)

- [ ] **Step 1.1.3: Add cache helper**

In `mdl/executor/helpers_domainmodels_gen.go` (likely already exists per prior C2 work):

```go
func cachedDomainModelsGen(ctx *ExecContext) ([]*genDm.DomainModel, error) {
    if ctx.Cache != nil && ctx.Cache.domainModelsGen != nil {
        return ctx.Cache.domainModelsGen, nil
    }
    list, err := ctx.Backend.ListDomainModelsGen()
    if err != nil { return nil, err }
    if ctx.Cache != nil {
        ctx.Cache.domainModelsGen = list
    }
    return list, nil
}

func invalidateDomainModelsGenCache(ctx *ExecContext) {
    if ctx == nil || ctx.Cache == nil { return }
    ctx.Cache.domainModelsGen = nil
    ctx.Cache.domainModels = nil // also invalidate legacy until Phase E removes
}
```

- [ ] **Step 1.1.4: Build + test**

```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./...
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/executor/ -count=1 -timeout 60s
```

- [ ] **Step 1.1.5: Commit**

```bash
git add mdl/executor/executor.go mdl/executor/helpers_domainmodels_gen.go
git commit -m "$(cat <<'EOF'
feat(executor): Stage 3.3.4.C9 — executorCache.domainModelsGen field + cache helper

Adds additive gen-typed cache field paralleling executorCache.domainModels,
plus cachedDomainModelsGen + invalidateDomainModelsGenCache helpers per
the cache-discipline rule (memory feedback_executor_cache_pattern).
Unblocks pages R9 migration: cmd_pages_builder.go::getDomainModels can
now switch to gen via this cache helper.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 1.2: C7 — pages-builder shim adapters (1 commit, ~2hr)

**Files:**
- Modify: `mdl/executor/cmd_pages_builder.go::getDomainModels` (return gen)
- Modify: `mdl/executor/cmd_pages_builder_v3.go` (callers)
- Modify: `mdl/executor/cmd_pages_builder_input_filters.go` (callers)

This is THE unblock for the parallel pages plan Phase 2. After this commit lands, the pages teammate can resume cmd_pages_builder family migration.

- [ ] **Step 1.2.1-N**: Migrate per file. Use `cachedDomainModelsGen(ctx)`.

- [ ] **Step 1.2.final: Commit `feat(executor): Stage 3.3.4.C7 — pages-builder reads on gen DomainModel (R9 unblock)`**

---

## §3 Phase 2 — Cat-B Consumer Migration (14 importers, ~10hr)

Per file commit. Each migration drops 1 sdk/domainmodel importer. Use `cachedDomainModelsGen` (Phase 1) for cache discipline.

### Task 2.1: cmd_modules.go (1 commit)
### Task 2.2: cmd_entities.go + cmd_entities_describe.go + cmd_entities_access.go (3 commits, separate per file)
### Task 2.3: cmd_associations.go (1 commit)
### Task 2.4: cmd_domainmodel_elk.go + cmd_mermaid.go (2 commits — or bundled if small)
### Task 2.5: cmd_move.go + cmd_diff_mdl.go (2 commits)
### Task 2.6: cmd_odata.go + cmd_contract.go + cmd_import.go (3 commits)
### Task 2.7: cmd_security_gen.go (1 commit)
### Task 2.8: cmd_structure.go + cmd_structure_gen.go (2 commits)
### Task 2.9: oql_type_inference.go + flowbuilder_assoc_lookup_gen.go + flowbuilder_actions_retrieve_gen.go (3 commits — or bundled)

For each file, the pattern is:
- [ ] Inspect callers: `grep -n "domainmodel\." <file>`
- [ ] Replace `[]*domainmodel.DomainModel` with `[]*genDm.DomainModel`
- [ ] Replace `dm.Entities` with `dm.EntitiesItems()` (gen accessor)
- [ ] Replace `entity.Attributes` with `entity.AttributesItems()` etc.
- [ ] Use raw-BSON fallback for any schema gap (per memory `gen_schema_gaps`)
- [ ] Drop `sdk/domainmodel` import
- [ ] Build + test
- [ ] Commit `refactor(executor): Stage 3.3.4.C-followup.<file> — <what>`

---

## §4 Phase 3 — Cat-C Catalog Migration (4 importers, ~3hr)

### Task 3.1: catalog/builder.go (1 commit)
### Task 3.2: catalog/builder_modules.go (1 commit)
### Task 3.3: catalog/builder_permissions.go + builder_permissions_test.go (1 commit)

Pattern same as Phase 2.

---

## §5 Phase 4 — Cat-D Phase D Write Path Remainder (4 importers, ~8hr)

### Task 4.1: domainmodel_modelsdk_gen.go consolidation (1 commit, ~2hr)

**Files:**
- Modify: `mdl/backend/mpr/domainmodel_modelsdk_gen.go` (consolidate gen-typed write path)
- Modify: `mdl/backend/mpr/domainmodel_modelsdk.go` (deprecate legacy createXxxViaModelsdk if no callers after Phase 2 migration)

- [ ] **Step 4.1.1: Audit createXxxViaModelsdk callers**

```bash
grep -rn "createDomainModelViaModelsdk\|updateDomainModelViaModelsdk" mdl/ --include="*.go"
```

If only mdl/backend/mpr/backend.go's legacy wrappers call them, schedule for E1 deletion.

- [ ] **Step 4.1.2-N**: Per helper rebuild on gen-native repo (per javaActionRepo Phase D template — see memory `project_stage_3_3_marathon_progress` Phase D template section).

### Task 4.2: modules_modelsdk.go gen migration (1 commit, ~2hr)

**Files:**
- Modify: `mdl/backend/mpr/modules_modelsdk.go` (drop sdk/domainmodel import; module-level operations now use gen DomainModel)

### Task 4.3: delete_move_modelsdk.go gen migration (1 commit, ~2hr)

**Files:**
- Modify: `mdl/backend/mpr/delete_move_modelsdk.go` (entity/association delete + move via gen)

### Task 4.4: mf_page_modelsdk.go gen migration (1 commit, ~2hr)

**Files:**
- Modify: `mdl/backend/mpr/mf_page_modelsdk.go` (page→domainmodel ref resolution on gen)

---

## §6 Phase 5 — Cat-F Test Fixtures (12 importers, ~6hr)

Per workflows E0 commit `60890815` pattern. Bundled OR per-file.

### Task 5.1-5.N: Test fixture migration

- [ ] mock_test_helpers_test.go (mkDomainModel/mkEntity/mkAttribute helpers) — 1 commit
- [ ] cmd_modules_mock_test.go — 1 commit
- [ ] cmd_entities_mock_test.go + cmd_associations_mock_test.go — 1-2 commits
- [ ] cmd_mermaid_mock_test.go + cmd_odata_mock_test.go — 1 commit (small)
- [ ] cmd_rename_mock_test.go + cmd_security_mock_test.go + cmd_write_handlers_mock_test.go — 1-2 commits
- [ ] validate_duplicates_test.go — 1 commit
- [ ] mpr/domainmodel_modelsdk_test.go (gen path test) — 1 commit
- [ ] mpr/security_entity_access_gen_test.go — 1 commit

Pattern per file:
- Replace `&domainmodel.DomainModel{...}` with `genDm.NewDomainModel()` + setters
- Replace `ListDomainModelsFunc` with `ListDomainModelsGenFunc`
- Use `RecordingDomainModelRepository` from `mdl/repos/testing/` if available
- Build + test
- Commit

---

## §7 Phase 6 — Cat-E E1 Backend Interface Retirement (4 importers, ~3hr)

**ORDERING WARNING (per workflows learning):** Verify E2 not needed first. If any executor file still imports sdk/domainmodel after Phase 2, that's an E2 deletion candidate before E1.

### Task 6.1: Audit + E2 (if any orphan executor files)

```bash
grep -l '"github.com/mendixlabs/mxcli/sdk/domainmodel"' mdl/executor/*.go
# Expected: 0 after Phase 2 + Phase 5 fixture migration
```

If non-zero, create E2 commit deleting orphan executor files BEFORE Task 6.2.

### Task 6.2: E1 retire FullBackend sdk-typed DomainModel surface (1 commit, ~2hr)

**Files:**
- Modify: `mdl/backend/domainmodel.go` (drop ListDomainModels/GetDomainModel/UpdateDomainModel/CreateEntity/UpdateEntity/etc. legacy methods — keep *Gen siblings)
- Modify: `mdl/backend/mpr/backend.go` (drop legacy impls)
- Modify: `mdl/backend/mpr/domainmodel_modelsdk.go` (delete createXxxViaModelsdk + updateXxxViaModelsdk — no callers after Phase 2+4)
- Modify: `mdl/backend/mock/backend.go` (drop Func fields)
- Modify: `mdl/backend/mock/mock_domainmodel.go` (drop legacy mock impls)

- [ ] **Step 6.2.1: Inventory current legacy methods**

```bash
grep -n "^func.*MprBackend.*DomainModel\|^func.*MprBackend.*Entity\|^func.*MprBackend.*Association" mdl/backend/mpr/backend.go | head -20
```

- [ ] **Step 6.2.2-6**: Standard retirement pattern per workflows E1 commit `9a2f298b`

- [ ] **Step 6.2.7: Commit `refactor(backend,mock): Stage 3.3.4.E1 — retire FullBackend sdk-typed DomainModel surface`**

---

## §8 Phase 7 — E3 system_module.go BSON-Native Rewrite (CRITICAL, ~10hr)

**Per master plan §14 Q2 in-plan decision:** sdk/mpr/system_module.go has been the single Stage 4 territory file kept in scope for this plan. Rewrite it BSON-natively (no sdk/domainmodel dependency) so the sdk/domainmodel package can be deleted entirely.

**Files:**
- Rewrite: `sdk/mpr/system_module.go` (BSON-native: read/write System module BSON via codec, no sdk/domainmodel types)

### Task 7.1: Inventory current system_module.go surface

- [ ] **Step 7.1.1**: Read full file:

```bash
wc -l sdk/mpr/system_module.go
grep -n "domainmodel\." sdk/mpr/system_module.go | head -20
```

- [ ] **Step 7.1.2**: Identify external callers

```bash
grep -rn "system_module\|systemModule\|SystemModule" sdk/ mdl/ cmd/ --include="*.go" | head -20
```

### Task 7.2: BSON-native rewrite

- [ ] Convert sdk/domainmodel.X usage to raw BSON via `bsonutil` or `codec`
- [ ] Preserve external API signatures so callers don't break
- [ ] Add roundtrip test verifying System module BSON byte-identity vs current implementation
- [ ] Build + test
- [ ] Commit `refactor(sdk/mpr): Stage 3.3.4.E3.system_module — BSON-native rewrite (drop sdk/domainmodel dep)`

### Task 7.3: Delete sdk/domainmodel package

- [ ] **Step 7.3.1: Final acceptance grep**

```bash
grep -rln '"github.com/mendixlabs/mxcli/sdk/domainmodel"' . --include="*.go"
```

Expected: 0 lines (system_module.go no longer imports after Task 7.2).

- [ ] **Step 7.3.2**: `git rm -r sdk/domainmodel/`

- [ ] **Step 7.3.3**: Build + full test gate green

- [ ] **Step 7.3.4: Commit `refactor: Stage 3.3.4.E3 — delete sdk/domainmodel package (Stage 4 boundary cleared)`**

---

## §9 Acceptance Criteria

- [ ] `grep -rln '"github.com/mendixlabs/mxcli/sdk/domainmodel"' . --include="*.go"` returns 0 lines (after E3 system_module rewrite)
- [ ] `mdl/`, `modelsdk/`, `cmd/` all gen-typed for DomainModel
- [ ] Pages R9 unblocked: parallel `2026-05-15-stage-3-3-pages-remaining.md` Phase 2 can start after Task 1.2 (C7)
- [ ] All `Test*DomainModel*`, `Test*Entity*`, `Test*Association*` tests pass:
      `GOPROXY=... ~/go1.26/bin/go test ./mdl/... ./modelsdk/... -run "DomainModel|Entity|Association|Module" -count=1 -v`
- [ ] System module roundtrip test (Task 7.2) verifies BSON byte-identity
- [ ] Full repo build green: `GOPROXY=... ~/go1.26/bin/go build ./...`
- [ ] Full repo test suite green: `GOPROXY=... ~/go1.26/bin/go test ./mdl/... ./modelsdk/... -count=1 -timeout 240s`

---

## §10 Estimated Commit Count + Time

| Phase | Tasks | Commits | Time |
|---|---|---|---|
| 1 — C9 + C7 (R9 unblock) | 1.1, 1.2 | 2 | ~5hr |
| 2 — Cat-B consumer migration | 2.1-2.9 | ~14 | ~10hr |
| 3 — Cat-C catalog | 3.1-3.3 | 3 | ~3hr |
| 4 — Cat-D Phase D remainder | 4.1-4.4 | 4 | ~8hr |
| 5 — Cat-F test fixtures | 5.1-5.N | ~10 | ~6hr |
| 6 — Cat-E E1 retirement | 6.1, 6.2 | 1-2 | ~3hr |
| 7 — E3 system_module rewrite | 7.1, 7.2, 7.3 | 2 | ~10hr |
| **Total** | | **~36-37 commits** | **~45hr** |

**Cannot fit in single session.** Recommend:
1. Session A: Phase 1 (C9 + C7) — ~2 commits, ~5hr — **TOP PRIORITY (unlocks pages plan Phase 2)**
2. Session B: Phase 2 (Cat-B consumers) — ~14 commits, ~10hr
3. Session C: Phase 3 + 4 + 5 (catalog + write path + fixtures) — ~17 commits, ~17hr
4. Session D: Phase 6 + 7 (E1 retirement + system_module rewrite + sdk delete) — ~4 commits, ~13hr

---

## §11 Coordination

- **Parallel plan:** `2026-05-15-stage-3-3-pages-remaining.md` Phase 2 blocked until §2 Task 1.2 (C7) lands
- **Stage 4 boundary:** §8 Phase 7 explicitly rewrites sdk/mpr/system_module.go BSON-natively (per master plan §14 Q2 in-plan decision); no other sdk/mpr files touched
- **Memory references:** `gen_schema_gaps`, `feedback_executor_cache_pattern`, `feedback_tdd`, `multi-agent-worktree-concurrency`, `project_stage_3_3_marathon_progress`, `project_stage_3_3_domainmodel_progress` (other team's progress checkpoints), Phase D template (nanoflow → javaactions, replicable)
- **Restore stage-3-3-domainmodel team coordination:** This plan is for stage-3-3-marathon team to execute. The original stage-3-3-domainmodel team paused 2026-05-14 21:31 — may resume independently. Backup-restore worktree pattern recommended if both active.

---

## §12 Risk surface

| Risk | Impact | Mitigation |
|---|---|---|
| Phase 2 cmd_security_gen.go has gen schema gaps for Entity AccessRule (per memory `project_stage_3_3_domainmodel_progress` D6 work attempted earlier) | Medium | Use raw-BSON fallback per memory `gen_schema_gaps`; reference impl-pages-d's `entityRuleRoleStringsGen` / `entityRuleRightStringsGen` from earlier session |
| Phase 4 Cat-D Phase D write path uses gen→sdk bridge in current state (D5.assoc + D8.entity per stage-3-3-domainmodel team's commits) | Medium | Phase 4 rebuilds these helpers gen-native per nanoflowRepo template; bridge can stay or go depending on caller migration progress |
| Phase 7 system_module.go BSON-native rewrite needs careful BSON shape parity | Critical | Codex review of system_module rewrite per plan §14 Q2 expectations; round-trip test against real System module fixtures (e.g., Administration module entities like User/UserRole) |
| stage-3-3-domainmodel team may resume between sessions | Medium | Backup-restore worktree pattern (per memory `multi-agent-worktree-concurrency`); coordinate via SendMessage if active simultaneously |

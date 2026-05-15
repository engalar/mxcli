# Stage 3.3.5 Pages — Remaining Work Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the remaining 28 sdk/pages importers (out of original 41) via 4 sequential phases that respect upstream blockers (R9 domainmodel C9 + R3 CE0463 fixture pipeline + §14 Q1 modelsdk.go user approval).

**Architecture:** Phases are gated by external dependencies. Phase 0 establishes CE0463 round-trip pipeline (one-time setup, ~3hr). Phase 1 (R3 widget tree) is the heavy lift requiring fixture roundtrip per commit. Phase 2 (R9 cmd_pages_builder family) is blocked until the parallel domainmodel-remaining plan lands C9. Phase 3 (E1 cleanup) is blocked on Phase 1+2 complete. Phase 4 (test fixtures + E2/E3) is final cleanup.

**Tech Stack:**
- Go 1.26 (`~/go1.26/bin/go`), `GOPROXY=https://mirrors.aliyun.com/goproxy/,direct`
- `modelsdk/gen/pages` types
- `modelsdk/widgets/templates/mendix-11.6/{datagrid,combobox,gallery}.json` (CE0463 baseline)
- `mdl-examples/{doctype-tests,bug-tests,widgetdemo}/*.mdl` (round-trip test scripts)
- `testdata/expr-checker/minimal.mpr` (clean baseline fixture)

---

## §1 Status Snapshot (2026-05-15)

**Already done** (impl-pages 13 commits + impl-pages-d 5 + impl-pages-d2 9 = 27 commits, 41→28 importers, -32%):
- Phase A read path complete (A0a/A0b/A1/A2/A6 dispatcher cutover)
- Phase B partial (B1 mermaid, B2 wireframe)
- Phase C consumer migration (C1 FullBackend gen methods, C3-C8 catalog/structure/security/fragments/describe-container, C7e catalog/builder)
- Phase D5 partial (alter_page lookup, delete/move backends, mock fixture migration for SHOW path + alter/move tests)
- 已记录 strategic intelligence in memory `project_pages_d5_boundary.md`

**Remaining 28 importers split into 5 categories**:

| Category | Importers | Files |
|---|---|---|
| **Cat-A R3 widget tree** | 11 | widget_engine.go, widget_property.go, widget_builder.go, datagrid_builder.go, page_mutator.go, mutation.go, mock_mutation.go, mock_page_mutator.go, cmd_alter_page.go, cmd_styling.go, builder_references.go |
| **Cat-B R9 cmd_pages_builder family** | 6 | cmd_pages_builder.go, cmd_pages_builder_input_filters.go, cmd_pages_builder_input_test.go, cmd_pages_builder_v3.go, cmd_pages_builder_v3_layout.go, cmd_pages_builder_v3_widgets.go |
| **Cat-C E1-blocked legacy backend** | 5 | page.go, backend/mpr/backend.go, mf_page_modelsdk.go, delete_move_modelsdk.go, mock/backend.go, mock_page.go |
| **Cat-D Test fixtures driving Cat-A/B paths** | 4 | mock_test_helpers_test.go, cmd_error_mock_test.go, cmd_snippetcall_params_test.go, cmd_write_handlers_mock_test.go |
| **Cat-E catalog/builder.go** | 1 | builder.go (only legacy 1 ref left after C7e) |
| **§14 Q1 user approval** | 2 (NOT in 28) | modelsdk.go + examples/create_page/main.go |

---

## §2 Pre-Flight Gate (REQUIRED before Phase 1)

### Task 0: CE0463 round-trip baseline (~3hr setup, lead direct or fresh teammate)

**Goal:** Establish a repeatable `mxcli test` round-trip pipeline that exercises DataGrid2 + ComboBox + Gallery pluggable widgets against `testdata/expr-checker/minimal.mpr`, capturing baseline state. Without this, every Phase 1 commit risks silently breaking client projects (CE0463 "widget definition changed").

**Files:**
- Create: `mdl-examples/widget-roundtrip/datagrid-roundtrip.test.mdl`
- Create: `mdl-examples/widget-roundtrip/combobox-roundtrip.test.mdl`
- Create: `mdl-examples/widget-roundtrip/gallery-roundtrip.test.mdl`
- Create: `mdl-examples/widget-roundtrip/README.md` (documents how to invoke)
- Reference (read-only): `mdl-examples/doctype-tests/29-datagrid-examples.mdl`, `17-custom-widget-examples.mdl`, `03-page-examples.mdl`, `widgetdemo/03-showcase-page.mdl`

- [x] **Step 0.1: Build mxcli binary**

```bash
make build
ls -la bin/mxcli
```

Expected: `bin/mxcli` is built (~30s).

- [x] **Step 0.2: Verify minimal.mpr fixture is intact**

```bash
ls -la testdata/expr-checker/minimal.mpr
ls testdata/expr-checker/mprcontents/ | head -5
./bin/mxcli -p testdata/expr-checker/minimal.mpr -c "show modules"
```

Expected: minimal.mpr is non-empty, mprcontents has unit JSON files, `show modules` returns module list without error.

- [x] **Step 0.3: Create datagrid roundtrip script**

`mdl-examples/widget-roundtrip/datagrid-roundtrip.test.mdl`:

```
-- DataGrid2 round-trip baseline for Stage 3.3.5 Phase 1 R3 verification.
-- Creates a page with the canonical DataGrid2 widget shape, then ALTERs
-- it through several mutations. Each mutation must round-trip through
-- the mpr without triggering CE0463 in Studio Pro.

create module DgRT;

create persistent entity DgRT.Item (
  Name: string(200),
  Status: string(50)
);

create page DgRT.ItemList (
  Title: 'Items',
  Layout: Atlas_Core.Atlas_Default
) {
  DATAGRID dgItems (DataSource: DATABASE DgRT.Item) {
    COLUMN colName   (Attribute: Name,   Caption: 'Name')
    COLUMN colStatus (Attribute: Status, Caption: 'Status')
  }
};

-- Mutation 1: Hide a column
alter page DgRT.ItemList {
  set widget dgItems.colStatus (Visible: false)
};

-- Mutation 2: Add a column
alter page DgRT.ItemList {
  insert column dgItems.colExtra (Caption: 'Extra')
};

-- Mutation 3: Drop a column
alter page DgRT.ItemList {
  drop column dgItems.colExtra
};
```

- [x] **Step 0.4: Create combobox + gallery roundtrip scripts**

Mirror datagrid-roundtrip.test.mdl shape for combobox + gallery. Use `modelsdk/widgets/templates/mendix-11.6/{combobox,gallery}.json` as schema reference for property names + types.

- [x] **Step 0.5: Run baseline test (pre-Phase-1, current state)**

```bash
scripts/run-widget-roundtrip.sh testdata/expr-checker/minimal.mpr testdata/expr-checker/mprcontents "mdl-examples/widget-roundtrip/*.test.mdl" /tmp/widget-roundtrip-baseline.txt
echo "---"
echo "exit: $?"
```

Expected: exit 0. Output captured to /tmp for diff against post-Phase-1.

Note: current `widget-roundtrip/*.test.mdl` scripts are exec-style mutation scripts (no `@test` blocks), so baseline execution uses `scripts/run-widget-roundtrip.sh` (`mxcli exec` against a fresh fixture copy per script).

- [x] **Step 0.6: Document baseline in README**

`mdl-examples/widget-roundtrip/README.md`:

```markdown
# Widget Round-Trip Baseline

CE0463 ("widget definition changed") regression suite for Stage 3.3.5
Phase 1 (R3 widget tree gen migration). Run after every commit that
touches:

- mdl/backend/mpr/widget_builder.go
- mdl/backend/mpr/datagrid_builder.go
- mdl/backend/mpr/page_mutator.go
- mdl/executor/widget_engine.go
- mdl/executor/widget_property.go
- mdl/executor/cmd_alter_page.go
- mdl/executor/cmd_styling.go

## Run

scripts/run-widget-roundtrip.sh

## Pass criteria

Exit 0 + output diff against baseline (/tmp/widget-roundtrip-baseline.txt
or git-tracked golden file) shows zero unexpected BSON shape changes.
```

- [ ] **Step 0.7: Commit pipeline (no source changes yet)**

```bash
git add mdl-examples/widget-roundtrip/
git commit -m "$(cat <<'EOF'
test: Stage 3.3.5 pre-flight — CE0463 round-trip baseline for R3 work

Adds 3 .test.mdl scripts that exercise canonical DataGrid2/ComboBox/Gallery
pluggable widget mutations. Stage 3.3.5 Phase 1 (R3 widget tree gen
migration) MUST run this suite after each commit touching widget
builders/mutators/engines.

Baseline pipeline established in README. Output captured to
/tmp/widget-roundtrip-baseline.txt (gitignored — regenerate locally).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## §3 Phase 1 — R3 Widget Tree (Cat-A, 11 importers, ~30-50hr)

**Risk gate:** EVERY commit in this phase must pass §2 Step 0.5 round-trip suite. Schema drift in any widget builder = CE0463 in client projects.

### Task 1.1: D0 page_mutator.go signature gen siblings (2-3 commits, ~3hr)

**Files:**
- Modify: `mdl/backend/mutation.go` (add *Gen sigs to PageMutator interface)
- Modify: `mdl/backend/mpr/page_mutator.go` (add *Gen impl methods, raw-bson logic stays)
- Modify: `mdl/backend/mock/mock_page_mutator.go` (add *Gen Func fields + impls)
- Modify: `mdl/backend/mock/backend.go` (add MockBackend Func fields)
- Modify: `mdl/backend/mpr/page_mutator_test.go` (gen sibling tests, BSON shape parity)

- [x] **Step 1.1.1: Inventory PageMutator legacy methods**

```bash
grep -n "^func.*pageMutator.*(" mdl/backend/mpr/page_mutator.go | head -20
```

Expected: list of methods like `SetProperty`, `InsertWidget`, `DropWidget`, `ReplaceWidget`, `SetWidgetProperty`, `MoveWidget`, etc.

- [x] **Step 1.1.2: Add Gen sigs to PageMutator interface**

In `mdl/backend/mutation.go`, find `type PageMutator interface` block. For each legacy method that takes `[]pages.Widget` or `*pages.Widget`, add a `*Gen` sibling taking `[]element.Element` or `*element.Element`:

```go
type PageMutator interface {
    // existing legacy methods stay
    SetProperty(prop, value string) error
    InsertWidget(widgetRef string, atPos int, widgets []pages.Widget) error
    // ... etc

    // ── Stage 3.3.5.D0 gen-typed siblings ─────────────────────────────
    InsertWidgetGen(widgetRef string, atPos int, widgets []element.Element) error
    DropWidgetGen(widgetRef string, atPos int) error
    ReplaceWidgetGen(widgetRef string, atPos int, widgets []element.Element) error
    SetWidgetPropertyGen(widgetRef string, atPos int, prop, value string) error
    MoveWidgetGen(widgetRef string, atPos int, newParentRef string) error
    Save() error // shared
}
```

- [x] **Step 1.1.3: Implement gen siblings on mprPageMutator**

In `mdl/backend/mpr/page_mutator.go`, for each `*Gen` method, mirror the legacy method's raw-bson logic but accept gen `element.Element` inputs. The internal BSON serialization path is unchanged — only the input shape differs:

```go
func (m *mprPageMutator) InsertWidgetGen(widgetRef string, atPos int, widgets []element.Element) error {
    // Encode each gen widget to BSON via codec.Encoder, then proceed
    // through existing serializeWidgets BSON shaping pipeline.
    for _, w := range widgets {
        contents, err := codec.NewEncoder().Encode(w)
        if err != nil { return err }
        m.insertWidgetRaw(widgetRef, atPos, contents)
    }
    return nil
}
```

- [x] **Step 1.1.4: Add MockBackend gen Func fields**

In `mdl/backend/mock/backend.go`, add Func fields for each *Gen method (per audit rule: descriptive errors, not nil):

```go
InsertWidgetGenFunc    func(widgetRef string, atPos int, widgets []element.Element) error
DropWidgetGenFunc      func(widgetRef string, atPos int) error
// ... etc
```

- [x] **Step 1.1.5: Implement Mock methods returning descriptive errors**

In `mdl/backend/mock/mock_page_mutator.go`:

```go
func (m *MockPageMutator) InsertWidgetGen(widgetRef string, atPos int, widgets []element.Element) error {
    if m.InsertWidgetGenFunc != nil {
        return m.InsertWidgetGenFunc(widgetRef, atPos, widgets)
    }
    return fmt.Errorf("MockPageMutator.InsertWidgetGen not configured")
}
```

- [x] **Step 1.1.6: Write BSON shape parity test**

In `mdl/backend/mpr/page_mutator_gen_test.go` (new file):

```go
func TestPageMutator_InsertWidgetGen_ProducesIdenticalBSON(t *testing.T) {
    w := openTestWriter(t)
    pm := openPageMutator(t, w, "TestPage")

    // Insert via legacy
    legacyContents := pm.snapshotPage()
    pm.InsertWidget("rootContainer", 0, []pages.Widget{makeLegacyTextBox()})
    legacyAfter := pm.snapshotPage()

    // Reset + insert via gen
    pm2 := openPageMutator(t, w, "TestPage")
    pm2.InsertWidgetGen("rootContainer", 0, []element.Element{makeGenTextBox()})
    genAfter := pm2.snapshotPage()

    if !bytes.Equal(legacyAfter, genAfter) {
        t.Errorf("BSON shape divergence: legacy=%x gen=%x", legacyAfter, genAfter)
    }
}
```

- [x] **Step 1.1.7: Run tests + round-trip suite**

```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/backend/mpr/ -count=1 -timeout 120s
scripts/run-widget-roundtrip.sh
```

Both must exit 0.

- [ ] **Step 1.1.8: Commit**

```bash
git add mdl/backend/mutation.go mdl/backend/mpr/page_mutator.go mdl/backend/mpr/page_mutator_gen_test.go mdl/backend/mock/mock_page_mutator.go mdl/backend/mock/backend.go
git commit -m "feat(backend): Stage 3.3.5.D0 — additive PageMutator gen siblings"
```

### Task 1.2: D1 widget_engine + widget_property gen (2-3 commits, ~6hr)

**Files:**
- Modify: `mdl/executor/widget_engine.go` (replace `*pages.Widget` parameters with gen `element.Element`)
- Modify: `mdl/executor/widget_property.go` (gen widget property accessors)
- Test: `mdl/executor/widget_engine_test.go` (round-trip every widget family)

- [x] **Step 1.2.1: Inventory current widget_engine.go surface**

```bash
grep -n "pages\." mdl/executor/widget_engine.go | head -30
```

Expected: ~20-30 references to `*pages.Widget`, `pages.WidgetType`, `pages.WidgetObject` etc.

Observed in current tree: 0 `pages.*` references in `widget_engine.go`, 0 in `widget_property.go`.

- [ ] **Step 1.2.2-N: Per family migration with round-trip**

Migrate one widget family at a time (input/display/list/container/action/advanced). For each family:
1. Replace `*pages.X` with `element.Element` + type assertion to gen widget type
2. Use raw-BSON fallback for gen schema gaps (per memory `gen_schema_gaps`)
3. Run `scripts/run-widget-roundtrip.sh` after each family migration
4. Commit per family: `refactor(executor): Stage 3.3.5.D1.<family> — widget_engine on gen <family>`

Current D1 slice (in progress): child-slot/default-slot plumbing now uses backend-opaque serialized children via `WidgetObjectBuilder.SetChildWidgetsOpaque`, removing direct `[]pages.Widget` collections from `widget_engine.go`.
Current D1 slice (in progress): DataSource and Action flow in `widget_engine.go` now uses backend-opaque values via `SetDataSourceOpaque` / `SetActionOpaque`; remaining `pages.*` usage in `widget_engine.go` is only the `Build()` return type (`*pages.CustomWidget`).

### Task 1.3: D2 widget_builder + datagrid_builder gen (5-8 commits, ~15-25hr — heaviest)

**Files:**
- Modify: `mdl/backend/mpr/widget_builder.go` (941 LoC pluggable widget construction)
- Modify: `mdl/backend/mpr/datagrid_builder.go` (1287 LoC DataGrid2 builder)

This is R3+R5 critical territory. Each commit MUST run §2 Step 0.5 round-trip + golden BSON diff.

- [ ] **Step 1.3.1: Capture golden BSON for current state**

```bash
scripts/run-widget-roundtrip.sh testdata/expr-checker/minimal.mpr testdata/expr-checker/mprcontents "mdl-examples/widget-roundtrip/datagrid-roundtrip.test.mdl" /tmp/widget-roundtrip-datagrid-baseline.txt
hexdump -C testdata/expr-checker/mprcontents/<dgRT-page-uuid>.json > /tmp/datagrid-baseline.hex
```

- [ ] **Step 1.3.2-N: Per builder rebuild with golden BSON diff**

For each builder method (e.g., `buildColumn`, `buildAction`, `buildContainer`):
1. Rewrite to take gen widget input
2. Run round-trip
3. `hexdump -C testdata/.../...json > /tmp/datagrid-after.hex && diff /tmp/datagrid-baseline.hex /tmp/datagrid-after.hex`
4. Expected: zero diff (BSON byte-identical)
5. If diff: fix or document schema-gap
6. Commit per coherent group

### Task 1.4: D3 cmd_alter_page.go gen widget pipeline (2-3 commits, ~5hr)

**Files:**
- Modify: `mdl/executor/cmd_alter_page.go` (AST→gen widget builders, 50+ widget AST mappings)

- [ ] **Step 1.4.1-N**: Build gen-side AST→widget converters mirroring legacy. Per family commits.

Progress: `cmd_alter_page.go` now routes INSERT/REPLACE through `InsertWidgetGen` / `ReplaceWidgetGen` and converts builder output to `element.Element` via serializer+codec decode; `cmd_alter_page.go` no longer imports `sdk/pages`.

### Task 1.5: D4 cmd_styling.go ALTER reflection rewrite (2-3 commits, ~6hr)

**Files:**
- Modify: `mdl/executor/cmd_styling.go` (reflection-based mutation switched to gen widget property containers)

Gen widgets are `element.Element` property containers, not plain Go structs. Replace `reflect.Value.FieldByName` with gen accessor calls + `setRawBSONField` fallback per `gen_schema_gaps` memory.

### Task 1.6: D5 catalog/builder_references extractWidgetRefs gen (1-2 commits, ~3hr)

**Files:**
- Modify: `mdl/catalog/builder_references.go` (50+ widget polymorphic ref extractor)

50+ widget type-switch from sdk to gen. Mechanical but voluminous. Run round-trip + catalog test suite after.

### Task 1.7: D6 catalog/builder.go cleanup (1 commit, ~30min)

**Files:**
- Modify: `mdl/catalog/builder.go` (drop residual sdk/pages import after Task 1.6)

---

## §4 Phase 2 — R9 cmd_pages_builder Family (Cat-B, 6 importers, ~15hr) — BLOCKED on domainmodel C9

**Blocker:** This phase cannot start until the parallel `2026-05-15-stage-3-3-domainmodel-remaining.md` plan lands C9 (executorCache.domainModels → gen). Verify before starting:

```bash
grep -n "domainModelsGen\|cachedDomainModelsGen" mdl/executor/executor.go
# Expected: gen-typed cache field present
```

### Task 2.1: cmd_pages_builder.go::getDomainModels gen migration (1 commit, ~2hr)

**Files:**
- Modify: `mdl/executor/cmd_pages_builder.go` (getDomainModels returns []*genDm.DomainModel)

- [x] **Step 2.1.1: Verify domainmodel C9 is in HEAD**

```bash
grep "Stage 3.3.4.C9" $(git log --format=%h | head -50 | xargs -I {} git log -1 --format="%h %s" {}) | head -1
```

Expected: a commit subject like "Stage 3.3.4.C9 — executorCache.domainModels → gen".

- [x] **Step 2.1.2-N**: Migrate getDomainModels + downstream callers per `cmd_pages_builder_v3.go`/`v3_layout.go`/`v3_widgets.go`/`input_filters.go`/`input_test.go`. Each commit: 1 file or coherent group.

Progress: `cmd_pages_builder_v3.go` downstream domainmodel resolution now uses `getDomainModelsWithContainer()` (`resolveAssociationDestination` + `entityQNByID`); legacy `backend.ListDomainModels()` usage removed from the pages-builder family.
Progress: consolidated `cmd_pages_builder_v3_layout.go` + `cmd_pages_builder_v3_widgets.go` into `cmd_pages_builder_v3.go`; pages-builder family now has one remaining executor file importing `sdk/pages`.
Progress: introduced `mdl/backend/pages_aliases.go` and rewired `cmd_pages_builder_v3.go` from `pages.*` to `backend.*` aliases; executor pages-builder family now has 0 direct `sdk/pages` imports.

### Task 2.2: cmd_pages_builder_input_filters/input_test/v3_*.go (4 commits, ~10hr)

Per file commit, mirror Task 2.1 pattern. Each migration drops 1 sdk/pages importer.

---

## §5 Phase 3 — Cat-D Test Fixture Migration (4 importers, ~3hr)

**Files:**
- Modify: `mdl/executor/mock_test_helpers_test.go` (mkPage/mkSnippet/mkLayout helpers off sdk/pages)
- Modify: `mdl/executor/cmd_error_mock_test.go` (execDropPage tests)
- Modify: `mdl/executor/cmd_snippetcall_params_test.go`
- Modify: `mdl/executor/cmd_write_handlers_mock_test.go`

Pattern per workflows E0 commit `60890815` and pages D5.b commit `02412ef8`:
- Replace `&pages.Page{ID, Name, ...}` with `genPg.NewPage()` + setters
- Replace `ListPagesFunc` mock wirings with `ListPagesGenFunc`
- Use `RecordingPageRepository` from `mdl/repos/testing/`

### Task 3.1: mock_test_helpers_test.go (1 commit)

- [ ] Replace mkPage/mkSnippet/mkLayout helpers' return types
- [ ] Update all in-package callers
- [ ] Build + test
- [ ] Commit `test(executor): Stage 3.3.5.E0.helpers — mkPage/mkSnippet/mkLayout on gen`

### Task 3.2-3.4: Per fixture file (1 commit each)

Same pattern.

---

## §6 Phase 4 — E2 + E1 + E3 Cleanup (5 importers + final, ~5hr)

**ORDERING WARNING (per workflows learning, commit `893ca3b7` vs `9a2f298b`):** E2 MUST come before E1. Legacy executor files import sdk/pages for handler functions; deleting interface methods first breaks build.

### Task 4.1: E2 — delete legacy executor pages files (1 commit, ~1hr)

**Files:**
- Delete: any remaining `mdl/executor/cmd_pages_legacy_*.go` (audit first)
- Verify no caller via grep before each delete

- [ ] **Step 4.1.1: Audit which executor files are now orphans**

```bash
for f in $(grep -l '"github.com/mendixlabs/mxcli/sdk/pages"' mdl/executor/*.go); do
  echo "=== $f ==="
  grep -c "pages\." "$f"
  echo "callers of public funcs in $f:"
  grep -h "^func [A-Z]" "$f" | head -5
done
```

Expected: identify files with ONLY orphan funcs (no external callers).

- [ ] **Step 4.1.2: git rm orphan files**

- [ ] **Step 4.1.3: Build + test gate green**

- [ ] **Step 4.1.4: Commit `refactor(executor): Stage 3.3.5.E2 — delete legacy pages executor files`**

### Task 4.2: E1 — retire FullBackend sdk-typed Page surface (1 commit, ~2hr)

**Files:**
- Modify: `mdl/backend/page.go` (drop sdk-typed PageBackend/LayoutBackend/SnippetBackend methods)
- Modify: `mdl/backend/mpr/backend.go` (drop corresponding impls)
- Modify: `mdl/backend/mpr/mf_page_modelsdk.go` (delete createPageViaModelsdk + updatePageViaModelsdk + Layout + Snippet helpers — no callers after E1)
- Modify: `mdl/backend/mpr/delete_move_modelsdk.go` (drop sdk import if only Page-related)
- Modify: `mdl/backend/mock/{backend,mock_page,mock_mutation,mock_page_mutator}.go` (drop legacy Func fields + impls)
- Modify: `mdl/backend/mutation.go` (drop sdk-typed mutator interface methods, keep *Gen)

- [ ] **Step 4.2.1: Audit legacy method callers**

```bash
for method in CreatePage UpdatePage CreateLayout UpdateLayout CreateSnippet UpdateSnippet; do
  echo "=== $method ==="
  grep -rn "\.${method}(" mdl/ --include="*.go" | grep -v "_test\|Gen("
done
```

Expected: zero callers (Phase 1+2 should have migrated all).

- [ ] **Step 4.2.2: Delete legacy interface methods + impls + mocks**

- [ ] **Step 4.2.3: Drop sdk/pages imports from all 6 modified files**

- [ ] **Step 4.2.4: Build + full test gate green**

- [ ] **Step 4.2.5: Commit `refactor(backend,mock): Stage 3.3.5.E1 — retire FullBackend sdk-typed Page surface`**

### Task 4.3: E3 — sdk/pages boundary (1 commit, ~30min)

- [ ] **Step 4.3.1: Check sdk/mpr Stage 4 boundary**

```bash
grep -l '"github.com/mendixlabs/mxcli/sdk/pages"' sdk/mpr/*.go
```

If non-empty (sdk/mpr still imports), add deprecation header per workflows `c59f3727` pattern. Otherwise `git rm -r sdk/pages/`.

- [ ] **Step 4.3.2: Commit `refactor(sdk/pages): Stage 3.3.5.E3 — deprecation header (Stage 4 boundary)`** OR `refactor: Stage 3.3.5.E3 — delete sdk/pages package`

---

## §7 §14 Q1 modelsdk.go (Out-of-Scope, User Approval Required)

**Files (NOT touched by this plan):**
- `modelsdk.go` lines 171-177 (`type Page = sdk/pages.Page` + Layout + Snippet aliases)
- `examples/create_page/main.go` (public example using the aliases)

**Decision required from user before next session:**
- **Option A (recommended):** Retire the aliases. Switch any external consumer to `modelsdk/gen/pages.Page` directly. 1 commit + example update.
- **Option B:** Keep aliases (new alias `type Page = genpages.Page`). Maintains BC.

After user approval, single commit. Out of this plan.

---

## §8 Acceptance Criteria

- [ ] `grep -rln '"github.com/mendixlabs/mxcli/sdk/pages"' . --include="*.go" | grep -v "^./sdk/mpr/"` returns:
  - 1 line (`mdl/backend/pages_aliases.go`) if sdk/pages compatibility boundary file is retained
  - 2 lines (modelsdk.go + examples/create_page) if §7 deferred and boundary file retired
- [ ] All `Test*Page*`, `Test*Widget*`, `Test*Layout*`, `Test*Snippet*` tests pass
- [ ] `scripts/run-widget-roundtrip.sh` exits 0
- [ ] Studio Pro Manual sniff test: open mpr post-mutation, no CE0463
- [ ] Full repo build green: `GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./...`
- [ ] Full repo test suite green: `GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/... ./modelsdk/... -count=1 -timeout 240s`

---

## §9 Estimated Commit Count + Time

| Phase | Tasks | Commits | Time |
|---|---|---|---|
| 0 — Pre-flight CE0463 baseline | 1 | 1 | ~3hr |
| 1 — R3 widget tree | 1.1-1.7 | ~15-20 | ~30-50hr |
| 2 — R9 cmd_pages_builder | 2.1-2.2 | ~5 | ~15hr (after dm-C9 lands) |
| 3 — Cat-D test fixtures | 3.1-3.4 | 4 | ~3hr |
| 4 — E2/E1/E3 cleanup | 4.1-4.3 | 3 | ~5hr |
| **Total** | | **~28-33 commits** | **~56-76hr** |

**Cannot fit in single session.** Phase 1 alone is multi-session. Recommend:
1. Session A: §2 Phase 0 + Phase 1 task 1.1 + 1.2 (D0 page_mutator + D1 widget_engine, ~10 commits, ~12hr)
2. Session B: Phase 1 task 1.3 (D2 widget_builder + datagrid_builder, ~8 commits, ~25hr — split if needed)
3. Session C: Phase 1 task 1.4-1.7 (D3-D6, ~7 commits, ~15hr)
4. Session D (after dm-C9 lands): Phase 2 + 3 + 4 (~12 commits, ~23hr)

---

## §10 Coordination

- **Parallel plan:** `2026-05-15-stage-3-3-domainmodel-remaining.md` lands C9 + C7 unblocking Phase 2
- **Stage 4 boundary:** sdk/mpr files (parser_page/layout/snippet/widgets, reader_documents, writer_*) are NOT touched by this plan
- **§14 Q1 user approval:** modelsdk.go + examples/create_page deferred until user decides
- **Memory references:** `gen_schema_gaps`, `feedback_executor_cache_pattern`, `feedback_tdd`, `multi-agent-worktree-concurrency`, `project_pages_d5_boundary`, `project_stage_3_3_marathon_progress`

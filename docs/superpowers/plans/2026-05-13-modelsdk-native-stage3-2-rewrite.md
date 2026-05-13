# Modelsdk-Native Stage 3.2: Rewrite Microflow Handlers from sdk to gen Types

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace every consumer of `*sdk/microflows.Microflow` in `mdl/executor/` with code that consumes `*modelsdk/gen/microflows.Microflow` end-to-end, allowing Stage 3.x to ultimately drop the sdk/microflows package and route every read/write through the modelsdk-native repos.

**Architectural choice (vs Stage 3.1's `type adapter` alternative):** Direct rewrite of handlers and emitters. No round-trip-BSON conversion shim. The rewrite is mechanical (sdk struct fields → gen method calls) but pervasive: ~22 production files + 57 test files + ~22,000 lines impacted.

**Phasing (this plan = 6 sub-stages):**

| Sub-stage | Scope | Files (approx) | Risk |
|---|---|---|---|
| **3.2.1** | Read-path entry + structural skeleton (DescribeMicroflowToString + traverseFlow framing, no activity-body formatting) | cmd_microflows_show.go, cmd_microflows_show_helpers.go (skeleton only) | Medium — proves gen API ergonomics |
| **3.2.2** | Activity formatters (60+ activity types, each ~50 lines) | cmd_microflows_format_action.go, cmd_microflows_helpers.go | High — many small rewrites; test gold-files validate |
| **3.2.3** | Write path: cmd_microflows_create, flowBuilder family (~5000 lines) | cmd_microflows_create, cmd_microflows_builder*.go | High — flowBuilder is dense |
| **3.2.4** | Visualizations: ELK / mermaid / structure / catalog | cmd_microflow_elk, cmd_mermaid, cmd_structure, cmd_catalog | Medium — render-only consumers |
| **3.2.5** | Misc consumers (security, diff, autocomplete, nanoflow parity) | ~10 files | Medium |
| **3.2.6** | Delete sdk/microflows imports; switch nanoflows similarly; verify | All test files; sdk/microflows package | Low (cleanup) |

This plan covers the **tasks** for each phase. Execution may span multiple sessions / PRs.

---

## Sub-stage 3.2.1: Read-path entry + structural skeleton

**Scope this commit:** A new file `cmd_microflows_show_gen.go` providing `DescribeMicroflowGenToString(ctx *ExecContext, mf *genMf.Microflow) (string, error)` plus a structural traverseFlow that walks the activity graph using gen types and emits MDL framing (header, sequence flows, splits, loops, end-event) but NOT individual activity bodies. Activity bodies render as `// TODO Stage 3.2.2: <typename>` placeholders. The legacy sdk path stays untouched and continues to be the production caller.

**Test strategy:** A new test (`cmd_microflows_show_gen_test.go`) loads a fixture microflow via `ctx.Microflows.Get`, calls the new `DescribeMicroflowGenToString`, and asserts output contains expected structural markers (microflow header, expected counts of `if/else/end if`, `case`/`end case`, `loop`/`end loop` keywords). Activity-body coverage is the 3.2.2 deliverable.

**Files:**
- Create: `mdl/executor/cmd_microflows_show_gen.go`
- Create: `mdl/executor/cmd_microflows_show_gen_test.go`
- Possibly modify: `mdl/repos/microflows.go` (if Reader needs ListByQualifiedName-style helpers — only if absent)

### Tasks

- [ ] **Step 1: Verify gen API surface**
   ```bash
   grep -nE "^func \(o \*Microflow\) " modelsdk/gen/microflows/types.go | head -30
   ```
   Confirm: `Name() string`, `ObjectCollection() element.Element`, `FlowsItems() []element.Element`, `MicroflowReturnType() element.Element`. Note any name differences vs sdk and adapt the implementation.

- [ ] **Step 2: Implement the new entry point + skeleton traverseFlow**
   - `DescribeMicroflowGenToString(ctx, mf) → MDL string`
   - Walk `mf.ObjectCollection().Units()` (or analog) into local `activityMap map[model.ID]element.Element` and `flowsByOrigin/Dest`.
   - Build `splitMergeMap` mirroring legacy `findSplitMergePoints` semantics — port the algorithm node-by-node from `cmd_microflows_show_helpers.go:findSplitMergePoints` using the gen API (this is the structural backbone).
   - Implement `traverseFlowGen` with the same control structure as legacy `traverseFlow`: handle `*genMf.StartEvent`, `*genMf.EndEvent`, `*genMf.ExclusiveSplit`, `*genMf.InheritanceSplit`, `*genMf.LoopedActivity`, `*genMf.ExclusiveMerge`. Activity nodes that don't match any of these emit a placeholder line `// TODO Stage 3.2.2: format <typeName>`.
   - Emit standard MDL framing: `microflow ModuleName.Name (params...)` header, indented body, `end microflow;` footer.

- [ ] **Step 3: Write structural-skeleton test**
   ```go
   func TestDescribeMicroflowGenToString_StructuralSkeleton(t *testing.T) {
       w := openTestWriter(t)
       repo := mprrepos.NewMicroflowRepository(w)
       refs, _ := w.ConcreteReader().ListUnitsByType("Microflows$Microflow")
       // Pick a fixture flow with control structures (if + loop) — find by name
       var pickID model.ID
       for _, ref := range refs {
           if ref.Type == "Microflows$Microflow" {
               // pick first; refine name match if needed
               pickID = model.ID(ref.ID)
               break
           }
       }
       mf, err := repo.Get(pickID)
       if err != nil { t.Fatal(err) }
       ctx := &ExecContext{Microflows: repo /* + minimal stubs */}
       out, err := DescribeMicroflowGenToString(ctx, mf)
       if err != nil { t.Fatal(err) }
       // structural assertions
       if !strings.Contains(out, "microflow ") || !strings.Contains(out, "end microflow;") { t.Fatal("missing framing") }
       if !strings.Contains(out, "// TODO Stage 3.2.2:") { t.Fatal("placeholders not emitted") }
   }
   ```

- [ ] **Step 4: Build + run**
   ```bash
   GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./...
   GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/executor/ -run TestDescribeMicroflowGenToString -v -count=1
   ```

- [ ] **Step 5: Full regression sweep — must stay 0 fails**
   ```bash
   GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./... -count=1 -timeout 180s
   ```

- [ ] **Step 6: Commit**
   ```bash
   git commit -m "feat(executor): Stage 3.2.1 — gen-typed DescribeMicroflowGenToString skeleton + traverseFlow"
   ```

**Deliverable:** Working entry point that consumes gen types end-to-end and emits MDL framing for any microflow. Activity bodies are placeholders; 3.2.2 fills them.

**Out of scope for 3.2.1:**
- Activity-specific formatters (60+ types) — that's 3.2.2
- Replacing the legacy `DescribeMicroflowToString` callers — 3.2.2 final step
- Nanoflow path — same pattern in 3.2.5

---

## Sub-stage 3.2.2: Activity body formatters

**Scope:** Port `formatActivity` + 60+ subtype formatters from sdk-typed `cmd_microflows_format_action.go` to gen-typed equivalents. Round-trip golden tests verify each activity type renders identically.

(Detailed task breakdown deferred — write at execution time per activity domain: ActionActivity family, FlowActivity family, MicroflowCallAction family, etc.)

---

## Sub-stage 3.2.3: Write path rewrite

**Scope:** `cmd_microflows_create.go` + the flowBuilder family (~5000 lines across 7 files). Each handler that constructs sdk Microflow / Activity types → constructs gen equivalents → calls `ctx.Microflows.Create` (already wired via Stage 2). The flowBuilder gets a parallel `flowBuilderGen` or its sdk-based fields gain gen siblings.

(Detailed tasks at execution time.)

---

## Sub-stage 3.2.4: Visualizations

**Scope:** cmd_microflow_elk.go (ELK graph), cmd_mermaid.go, cmd_structure.go, cmd_nanoflow_elk.go. Render-only consumers — easier than write path; mostly type-substitution mechanical.

---

## Sub-stage 3.2.5: Misc consumers + Nanoflows

**Scope:**
- cmd_security_write.go, cmd_security.go, cmd_diff.go, autocomplete.go, helpers.go, cmd_workflows_write.go, cmd_odata.go (where they touch microflows)
- Nanoflow parity: replicate 3.2.1–3.2.4 work for `*sdk/microflows.Nanoflow` → `*gen/microflows.Nanoflow`

---

## Sub-stage 3.2.6: Final cleanup

- [ ] Delete every remaining `import "github.com/mendixlabs/mxcli/sdk/microflows"` in `mdl/executor/`
- [ ] Verify: `/usr/bin/grep -rln "sdk/microflows" mdl/executor/ --include="*.go"` returns no matches
- [ ] Verify: `ctx.Backend.{List,Get,Create,Update,Move}(Microflow|Nanoflow)` callsites all gone
- [ ] Optionally remove the bridge functions in repo_extract.go (they were the Stage 3.1 transition — once 3.2 lands, all callsites use ctx.Microflows directly)
- [ ] Stage 3.3 prereq: with microflow + nanoflow done, the pattern is ready to replicate for the other 13 domains

---

## Out of scope (across all of 3.2)

- Stage 3.3+ (other domain rewrites: pages, domainmodels, etc.)
- Stage 3.N (drop ExecContext.Backend field, retire MockBackend) — different commit, after every 3.2.N–3.16 lands
- Stage 4 (delete sdk/mpr Writer + serializers)
- API changes in `api/` or `modelsdk.go`

---

## Self-review checklist (per sub-stage)

- [ ] No regressions in `go test ./... -count=1 -timeout 180s`
- [ ] Each rewritten function has at least one gen-typed unit test
- [ ] Gold-file or string-equality test confirms gen-rewritten output matches legacy output (until 3.2.6 deletes legacy)
- [ ] No new sdk/microflows imports introduced
- [ ] Commit message references the sub-stage number

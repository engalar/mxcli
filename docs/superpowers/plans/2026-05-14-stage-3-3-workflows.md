# Stage 3.3 Workflows Domain — Detailed Sub-Plan

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans`. Steps use checkbox `- [ ]` syntax for trackability.
>
> Generated from `2026-05-14-stage-3-3-domain-marathon-master.md` §6 priority #3 + §4 phase template + §8.3 stub. Builds on the security plan's playbook (`2026-05-14-stage-3-3-security.md`) and the javaactions sub-plan (`2026-05-14-stage-3-3-javaactions.md`); inherits Stage 3.2 + Stage 3.3.1 infrastructure (`mprread.ListUnitsByType`, `helpers_gen_container.go`, `ByNameRefList` versioned-array fix). The workflow domain reuses the **microflow activity formatters from Stage 3.2** wherever activity shapes overlap.

**Goal:** Migrate the `workflows` domain off the legacy hand-written `sdk/workflows` package onto the auto-generated `modelsdk/gen/workflows` types. Final state: zero `sdk/workflows` imports outside `sdk/mpr/` (Stage 4 territory) and zero in `modelsdk/`, `api/`, `mdl/`.

---

## §1 Background — Status Snapshot

The `workflows` domain is the **largest read-side surface** of the six Stage 3.3 domains: 14 polymorphic activity types, 4 condition outcome types, 5 user-source types, plus boundary events and event sub-processes. The gen package exposes **92 types** (vs 384 LoC of legacy sdk), so the migration is "narrow legacy → wide gen", with most work concentrated in formatters + dispatchers, not new infrastructure.

The good news: **ALTER WORKFLOW already exists** as a raw-bson mutator (`mprWorkflowMutator` in `mdl/backend/mpr/workflow_mutator.go`, 817 LoC). It manipulates `bson.D` directly without going through `sdk/workflows` types **except** for the `[]workflows.WorkflowActivity` parameter shape on the public mutator interface. The migration there is a **signature change** (accept `[]element.Element` from gen instead) plus a serialize-input adapter.

### Already migrated — DO NOT redo

- `mdl/backend/mpr/workflow_mutator.go` — raw-bson mutator implementing `WorkflowMutator`. Uses `bson.D` + `bsonutil` for the actual mutation; only the **input parameter type** (`[]workflows.WorkflowActivity`) is sdk-typed. Mutation logic does NOT need to change — just the input adapter.
- `mdl/executor/cmd_workflows_write_gen.go` (126 LoC) — **NOT a workflow gen migration**. It's a Stage 3.2 helper (`autoBindCallMicroflowGen`) that resolves microflow parameters via `ctx.Microflows` (gen) but still uses `*workflows.CallMicroflowTask` for the workflow-side data. Stays in place; rebuilt in Phase D2 to use gen `CallMicroflowActivity`.
- `mdl/executor/cmd_workflows_write_gen_test.go` — covers the Stage 3.2 helper.
- `modelsdk/gen/workflows/{types,enums,refs,version,ext}.go` — auto-generated, 8689 LoC total, 92 types. Verified in §S2.4 below.

### Still to migrate (this plan's scope)

| File | LoC | What stays | What leaves |
|---|---|---|---|
| `sdk/workflows/workflow.go` | 384 | nothing | full package (after E3) |
| `mdl/backend/workflow.go` | 17 | interface methods | `*workflows.Workflow` from 4 method sigs |
| `mdl/backend/mutation.go` | 354 | PageMutator etc. | `[]workflows.WorkflowActivity` from 6 mutator sigs + `WorkflowActivity` from `SerializeWorkflowActivity` |
| `mdl/backend/mpr/backend.go` | 881 | shim wrappers | `*workflows.*` types in 5 method sigs |
| `mdl/backend/mpr/workflow_mutator.go` | 817 | raw-bson logic | sdk types in 9 sites — input params + `serializeAndDedup` + `buildSubFlowBson` + `deduplicateNewActivityName` |
| `mdl/backend/mpr/workflow_mutator_test.go` | (size TBD) | test infra | sdk-typed fixtures |
| `mdl/backend/mpr/mf_page_modelsdk.go` | (size TBD) | mf/page logic | 1 incidental ref |
| `mdl/backend/mock/backend.go` | 309 | Func decls | `*workflows.*` in 4 Func types |
| `mdl/backend/mock/mock_workflow.go` | 118 | shim layer | sdk types in method sigs |
| `mdl/backend/mock/mock_workflow_mutator.go` | 136 | shim layer | sdk types in 12 sites |
| `mdl/backend/mock/mock_mutation.go` | 112 | infra | 1 ref |
| `mdl/catalog/builder.go` | 585 | reader interface | one method `ListWorkflows() ([]*workflows.Workflow, error)` |
| `mdl/catalog/builder_workflows.go` | 126 | counter helpers | sdk type-switch in 5 cases (`*workflows.UserTask` etc.) |
| `mdl/catalog/builder_references.go` | 701 | other extractors | `extractWorkflowFlowRefs` + condition-outcome ref extractor |
| `mdl/catalog/builder_strings.go` | 295 | other string extractors | `extractWorkflowFlowStrings` + 5 type-switch cases |
| `mdl/catalog/builder_workflows_test.go` | (size TBD) | test infra | sdk fixtures |
| `mdl/executor/cmd_workflows.go` | 669 | nothing | full file (45 sdk refs, 16 funcs) |
| `mdl/executor/cmd_alter_workflow.go` | 188 | dispatch logic | 1 incidental ref (calls `OpenWorkflowForMutation` returning `WorkflowMutator`) |
| `mdl/executor/cmd_workflows_write.go` | 790 | nothing | full file (87 sdk refs, 21 funcs) |
| `mdl/executor/cmd_workflows_write_gen.go` | 126 | autoBindCallMicroflowGen helper | rebuild to gen-typed CallMicroflowActivity |
| `mdl/executor/cmd_workflows_write_gen_test.go` | (size TBD) | test infra | sdk fixtures |
| `mdl/executor/cmd_workflows_describe_test.go` | (size TBD) | needs migration | sdk fixtures |
| `mdl/executor/cmd_workflows_mock_test.go` | (size TBD) | needs migration | sdk fixtures |
| `mdl/executor/cmd_structure.go` | 751 | structure infra | `*workflows.*` in `structureWfMap` value type + `outputWorkflows` body |
| `mdl/executor/cmd_structure_gen.go` | 509 | gen infra | `structureWfMapGen` value type currently `[]*workflows.Workflow` (sdk) |
| `mdl/executor/cmd_error_mock_test.go` | (small) | needs migration | one fixture |
| `mdl/executor/cmd_json_mock_test.go` | (small) | needs migration | one fixture |
| `mdl/executor/cmd_rename_mock_test.go` | (small) | needs migration | one fixture |
| `mdl/executor/mock_test_helpers_test.go` | (small) | needs migration | helper using sdk types |
| `mdl/executor/validate_duplicates_test.go` | (small) | needs migration | one fixture |
| `mdl/executor/validate_duplicates.go` | 425+ | duplicate-detection | 1 ref (`ListWorkflows` caller — purely consumes wf.Name) |
| `mdl/executor/cmd_rename.go` | (size TBD) | rename logic | 1 ref (`ListWorkflows` caller) |
| `sdk/mpr/parser_workflow.go` | (Stage 4 territory) | — | DO NOT touch |
| `sdk/mpr/reader_documents.go` | (Stage 4 territory) | — | DO NOT touch |
| `sdk/mpr/serialize_exports.go` | (Stage 4 territory) | — | DO NOT touch (re-exports SerializeWorkflow) |
| `sdk/mpr/workflow_parse_test.go` | (Stage 4 territory) | — | DO NOT touch |
| `sdk/mpr/workflow_write_test.go` | (Stage 4 territory) | — | DO NOT touch |
| `sdk/mpr/writer_workflow.go` | (Stage 4 territory) | — | DO NOT touch |

### Why this domain is priority #3

- **Medium complexity**: more activity types than security/javaactions but proven mutator pattern from Stage 3.1
- **Activity formatter overlap with microflows**: Stage 3.2 already migrated 60+ microflow activity types. Sub-types like ParameterMapping, Microflow refs are reusable; the polymorphic dispatch pattern in `formatActivityGen` is the template for `formatWorkflowActivityGen`.
- **ALTER WORKFLOW already raw-bson**: the heavy mutator logic doesn't need to change — only input param adapter (Phase D)
- **Largest read-side surface**: 16 read funcs + 21 write funcs = ~37 commits in Phases A + D alone, justifying a per-task TDD breakdown

---

## §2 Pre-Flight Survey Results

### S2.1 sdk/workflows importers (29 in-scope, 6 Stage 4 — exact list)

```
mdl/backend/mock/backend.go
mdl/backend/mock/mock_mutation.go
mdl/backend/mock/mock_workflow.go
mdl/backend/mock/mock_workflow_mutator.go
mdl/backend/mpr/backend.go
mdl/backend/mpr/mf_page_modelsdk.go
mdl/backend/mpr/workflow_mutator.go
mdl/backend/mpr/workflow_mutator_test.go
mdl/backend/mutation.go
mdl/backend/workflow.go
mdl/catalog/builder.go
mdl/catalog/builder_references.go
mdl/catalog/builder_strings.go
mdl/catalog/builder_workflows.go
mdl/catalog/builder_workflows_test.go
mdl/executor/cmd_alter_workflow.go
mdl/executor/cmd_error_mock_test.go
mdl/executor/cmd_json_mock_test.go
mdl/executor/cmd_rename_mock_test.go
mdl/executor/cmd_structure.go
mdl/executor/cmd_structure_gen.go
mdl/executor/cmd_workflows.go             ← 45 references, 16 funcs
mdl/executor/cmd_workflows_describe_test.go
mdl/executor/cmd_workflows_mock_test.go
mdl/executor/cmd_workflows_write.go        ← 87 references, 21 funcs
mdl/executor/cmd_workflows_write_gen.go    ← 4 (Stage 3.2 helper)
mdl/executor/cmd_workflows_write_gen_test.go
mdl/executor/mock_test_helpers_test.go
mdl/executor/validate_duplicates_test.go
sdk/mpr/parser_workflow.go             ← Stage 4 territory
sdk/mpr/reader_documents.go            ← Stage 4 territory
sdk/mpr/serialize_exports.go           ← Stage 4 territory
sdk/mpr/workflow_parse_test.go         ← Stage 4 territory
sdk/mpr/workflow_write_test.go         ← Stage 4 territory
sdk/mpr/writer_workflow.go             ← Stage 4 territory
```

### S2.2 Read funcs in `cmd_workflows.go`

- `listWorkflows(ctx, moduleName)` — `ctx.Backend.ListWorkflows()` returning `[]*workflows.Workflow`. Renders qualifiedName + activity counts via `countWorkflowActivities`.
- `countWorkflowActivities(wf *workflows.Workflow) (total, userTasks, decisions int)` — recursive walker over Flow + Outcomes.
- `countFlowActivities(flow *workflows.Flow, total, userTasks, decisions *int)` — flow descent.
- `describeWorkflow(ctx, name)` — calls `describeWorkflowToString` and prints.
- `describeWorkflowToString(ctx, name) (string, map[string]elkSourceRange, error)` — **the heavy one**. Returns the rendered MDL plus an ELK source-range map (used by `cmd_visualize.go` for SVG generation). Walks every activity recursively.
- `formatAnnotation`, `boundaryEventKeyword` — pure helpers.
- `formatBoundaryEvents([]*workflows.BoundaryEvent, indent) []string` — sdk-typed.
- `formatWorkflowActivities(flow *workflows.Flow, indent) []string` — flow-level dispatch.
- `formatUserTask(*workflows.UserTask, indent) []string` — 93 LoC; renders task page, name, description, allowed roles, target, completion criteria, outcomes, boundary events.
- `formatCallMicroflowTask(*workflows.CallMicroflowTask, indent)`.
- `formatSystemTask(*workflows.SystemTask, indent)`.
- `formatCallWorkflowActivity(*workflows.CallWorkflowActivity, indent)`.
- `formatExclusiveSplit(*workflows.ExclusiveSplitActivity, indent)`.
- `formatParallelSplit(*workflows.ParallelSplitActivity, indent)`.
- `formatConditionOutcomes([]workflows.ConditionOutcome, indent)`.

### S2.3 Write funcs in `cmd_workflows_write.go`

- `execCreateWorkflow(ctx, s)` — builds `*workflows.Workflow` from AST; calls `CreateWorkflow` or `UpdateWorkflow`.
- `execDropWorkflow(ctx, s)` — list + filter + delete.
- `generateWorkflowUUID()` — pure helper.
- `buildWorkflowActivities([]ast.WorkflowActivityNode) []workflows.WorkflowActivity` — top-level builder.
- `buildBoundaryEvents([]ast.WorkflowBoundaryEventNode) []*workflows.BoundaryEvent`
- `buildWorkflowActivity(ast.WorkflowActivityNode) workflows.WorkflowActivity` — dispatch.
- `buildUserTask(*ast.WorkflowUserTaskNode) *workflows.UserTask`
- `buildCallMicroflowTask(*ast.WorkflowCallMicroflowNode) *workflows.CallMicroflowTask`
- `buildCallWorkflowActivity(*ast.WorkflowCallWorkflowNode) *workflows.CallWorkflowActivity`
- `buildExclusiveSplit(*ast.WorkflowDecisionNode) *workflows.ExclusiveSplitActivity`
- `buildConditionOutcome(ast.WorkflowConditionOutcomeNode) workflows.ConditionOutcome`
- `buildParallelSplit(*ast.WorkflowParallelSplitNode) *workflows.ParallelSplitActivity`
- `buildJumpTo`, `buildWaitForTimer`, `buildWaitForNotification`, `buildEndWorkflow`, `buildAnnotationActivity` — leaf builders.
- `deduplicateActivityNames`, `deduplicateActivityNamesInFlow`, `uniqueName`, `sanitizeActivityName` — name-management helpers; pure on sdk types but trivially port to gen `*element.Element` walks.
- `autoBindWorkflowParameters([]workflows.WorkflowActivity)` — pre-fill Parameter mappings.
- `autoBindActivitiesInFlow([]workflows.WorkflowActivity)` — dispatch to autoBindCallMicroflow / autoBindCallWorkflow.
- `autoBindCallMicroflow(*workflows.CallMicroflowTask)` — **already has gen sibling** in `cmd_workflows_write_gen.go::autoBindCallMicroflowGen` (Stage 3.2). Phase D rebuilds the workflow side too.
- `autoBindCallWorkflow(*workflows.CallWorkflowActivity)`.

### S2.4 Gen accessor name map (CRITICAL — verified against `modelsdk/gen/workflows/types.go`)

**Workflow (line 5232):**

| sdk field/method | gen accessor | Notes |
|---|---|---|
| `wf.Name` | `Name() / SetName(string)` | exact |
| `wf.Documentation` | `Documentation() / SetDocumentation` | exact |
| `wf.Excluded` | `Excluded() / SetExcluded` | exact |
| `wf.ExportLevel` | `ExportLevel() / SetExportLevel` | exact |
| `wf.Title` (display name) | `Title() / SetTitle` AND `WorkflowName() element.Element` (Texts$Text wrapper) | **Two parallel accessors** — see R1 |
| `wf.ContextEntity` | `ContextEntityQualifiedName() / SetContextEntityQualifiedName` | + `WorkflowEntityQualifiedName()` (legacy)|
| `wf.WorkflowEntity` | `WorkflowEntityQualifiedName()` | exact |
| `wf.OverviewPage` | `OverviewPageQualifiedName() / SetOverviewPageQualifiedName` | exact |
| `wf.AdminPage` | `AdminPage() element.Element / SetAdminPage` | wraps a PageReference part |
| `wf.Parameter` (`*WorkflowParameter`) | `Parameter() element.Element / SetParameter` | gen Parameter is a Part — must cast to `*genWf.Parameter` (line 4004) and use `EntityQualifiedName()` |
| `wf.Flow` (`*Flow`) | `Flow() element.Element / SetFlow` | wraps `*genWf.Flow` (line 2050) |
| `wf.AllowedModuleRoles []string` | `AllowedModuleRolesQualifiedNames() []string / Set...` + `AddAllowedModuleRoles(string)` | uses `ByNameRefList` — **versioned-array fix from Stage 3.3.1 D6a applies** |
| `wf.OnWorkflowEvent` (`[]*WorkflowEventHandler`) | `OnWorkflowEventItems() []element.Element / AddOnWorkflowEvent / RemoveOnWorkflowEvent` | PartList |
| `wf.EventSubProcesses` | `EventSubProcessesItems() []element.Element` | PartList |
| `wf.WorkflowName` (display Title text) | `WorkflowName() element.Element` | wraps Texts$Text |
| `wf.WorkflowDescription` | `WorkflowDescription() element.Element` | wraps Texts$Text |
| `wf.DueDate` | `DueDate() / SetDueDate` | string |
| `wf.WorkflowMetaData` | `WorkflowMetaData() element.Element` | wraps `*genWf.WorkflowMetaData` (line 5779) |
| `wf.Annotation` | `Annotation() element.Element / SetAnnotation` | wraps `*genWf.Annotation` (line 597) |
| `wf.WorkflowOnStateChangeEvent` | `WorkflowOnStateChangeEvent() element.Element` | event handler ref |
| `wf.UsertaskOnStateChangeEvent` | `UsertaskOnStateChangeEvent() element.Element` | event handler ref |
| `wf.WorkflowV2` | `WorkflowV2() bool / SetWorkflowV2` | exact |

**Flow (line 2050):**

| sdk field | gen accessor |
|---|---|
| `flow.Activities []WorkflowActivity` | `ActivitiesItems() []element.Element / AddActivities / RemoveActivities` |

**Activity Type Map (sdk → gen, with storage names):**

| sdk type | gen type (line) | Storage `$Type` | Notes |
|---|---|---|---|
| `StartWorkflowActivity` | `StartWorkflowActivity` (4330) | `Workflows$StartWorkflowActivity` | exact rename |
| `EndWorkflowActivity` | `EndWorkflowActivity` (1667) | `Workflows$EndWorkflowActivity` | exact |
| `UserTask` | `UserTaskActivity` (2956) **AND** `UserTask` (4598) | `Workflows$UserTaskActivity` | **DUAL TYPE** — see R2. The activity placed inflow is `UserTaskActivity`. The bare `UserTask` type is a sub-element (likely TaskName/TaskDescription holder). PLUS `MultiUserTaskActivity` (3192) and `SingleUserTaskActivity` (4094) — gen splits the user-task hierarchy into multi vs single. sdk had only the unified `UserTask`. **Migration must handle all three subtypes.** |
| `SystemTask` | gen has NO `SystemTask` (verified by `^type SystemTask` grep) | (storage TBD — likely `Workflows$SystemTaskActivity` — VERIFY in A0) | Schema gap risk. Possibly merged into `MicroflowBasedActivity` (line 219) which is an abstract parent class for tasks that delegate to microflows |
| `CallMicroflowTask` | `CallMicroflowTask` (1096) **AND** `CallMicroflowActivity` (931) | `Workflows$CallMicroflowActivity` | **DUAL TYPE** — see R2. The inflow activity is `CallMicroflowActivity` (line 931); `CallMicroflowTask` is the legacy property name |
| `CallWorkflowActivity` | `CallWorkflowActivity` (1261) | `Workflows$CallWorkflowActivity` | exact |
| `ExclusiveSplitActivity` | `ExclusiveSplitActivity` (1883) | `Workflows$ExclusiveSplitActivity` | exact |
| `ParallelSplitActivity` | `ParallelSplitActivity` (3859) | `Workflows$ParallelSplitActivity` | exact |
| `JumpToActivity` | `JumpToActivity` (2334) | `Workflows$JumpToActivity` | exact |
| `WaitForTimerActivity` | `WaitForTimerActivity` (5134) | `Workflows$WaitForTimerActivity` | exact |
| `WaitForNotificationActivity` | `WaitForNotificationActivity` (5027) | `Workflows$WaitForNotificationActivity` | exact |
| `EndOfParallelSplitPathActivity` | `EndOfParallelSplitPathActivity` (1581) | `Workflows$EndOfParallelSplitPathActivity` | exact |
| `EndOfBoundaryEventPathActivity` | `EndOfBoundaryEventPathActivity` (1495) | `Workflows$EndOfBoundaryEventPathActivity` | exact |
| `WorkflowAnnotationActivity` | gen has `Annotation` (597) and `FloatingAnnotation` (2002) | (storage TBD — VERIFY in A0) | sdk's `WorkflowAnnotationActivity` likely maps to `FloatingAnnotation` (the inflow positioned annotation) vs `Annotation` (sub-element) |
| `GenericWorkflowActivity` | (no gen type — used as sdk fallback for unknown $Types) | N/A | gen handles via raw element fallback |

**Outcome Type Map:**

| sdk type | gen type (line) | Storage `$Type` | Notes |
|---|---|---|---|
| `UserTaskOutcome` | `UserTaskOutcome` (4873) | `Workflows$UserTaskOutcome` | exact |
| `BooleanConditionOutcome` | `BooleanConditionOutcome` (781) | `Workflows$BooleanConditionOutcome` | exact |
| `EnumerationValueConditionOutcome` | `EnumerationValueConditionOutcome` (1753) | `Workflows$EnumerationValueConditionOutcome` | exact |
| `VoidConditionOutcome` | `VoidConditionOutcome` (4989) | `Workflows$VoidConditionOutcome` | exact |
| `ParallelSplitOutcome` | `ParallelSplitOutcome` (3966) | `Workflows$ParallelSplitOutcome` | exact |
| `ConditionOutcome` (interface) | `ConditionOutcome` (743) **AND** `ConditionOutcomeActivity` (112) | `Workflows$ConditionOutcome` | abstract base — gen exposes both |
| `ExclusiveSplitOutcome` | `ExclusiveSplitOutcome` (5978) | `Workflows$ExclusiveSplitOutcome` | gen-only addition |

**User Source Map (`UserSource` interface in sdk):**

| sdk type | gen type (line) | Storage `$Type` | Notes |
|---|---|---|---|
| `NoUserSource` | `EmptyUserSource` (1483) | `Workflows$EmptyUserSource` | **NAMING FLIP** — sdk `NoUserSource` ↔ gen `EmptyUserSource` |
| `MicroflowBasedUserSource` | `MicroflowBasedUserSource` (2712) | `Workflows$MicroflowBasedUserSource` | exact |
| `XPathBasedUserSource` | `XPathBasedUserSource` (5906) | `Workflows$XPathBasedUserSource` | exact |
| `MicroflowGroupSource` | (gen has `MicroflowGroupTargeting` (2836) — semantics check) | (storage TBD — VERIFY in A0) | Likely renamed: gen separates "user source" (which user) from "targeting" (whom to assign). VERIFY |
| `XPathGroupSource` | (gen has `XPathGroupTargeting` (5930)) | (storage TBD — VERIFY in A0) | Same naming pattern |

**Boundary Event Map:**

| sdk type | gen type (line) | Storage `$Type` |
|---|---|---|
| `BoundaryEvent` | `BoundaryEvent` (831) abstract + `InterruptingTimerBoundaryEvent` (2222) + `NonInterruptingTimerBoundaryEvent` (3618) + `TimerBoundaryEvent` (4498) | `Workflows$InterruptingTimerBoundaryEvent` etc. — **gen splits into concrete subtypes** |

**ParameterMapping:**

| sdk | gen |
|---|---|
| `ParameterMapping{Parameter, Expression}` | `MicroflowCallParameterMapping` (2740) for `CallMicroflowActivity`; `WorkflowCallParameterMapping` (5602) for `CallWorkflowActivity`; `PageParameterMapping` (3791) for UserTask page params | gen splits by call-site |

### S2.5 Backend interface methods (`mdl/backend/workflow.go` + `mdl/backend/mutation.go`)

```go
WorkflowBackend (workflow.go):
  ListWorkflows() ([]*workflows.Workflow, error)            // sdk-typed read — REPLACE
  GetWorkflow(id model.ID) (*workflows.Workflow, error)     // sdk-typed read — REPLACE
  CreateWorkflow(wf *workflows.Workflow) error              // sdk-typed write — REPLACE
  UpdateWorkflow(wf *workflows.Workflow) error              // sdk-typed write — REPLACE

WorkflowMutator (mutation.go):
  SetProperty(prop, value string) error                     // ok
  SetPropertyWithEntity(prop, value, entity string) error  // ok
  SetActivityProperty(activityRef string, atPos int, prop, value string) error // ok
  InsertAfterActivity(activityRef string, atPos int, activities []workflows.WorkflowActivity) error  // sdk-typed input — REPLACE input slice type
  DropActivity(activityRef string, atPos int) error         // ok
  ReplaceActivity(activityRef string, atPos int, activities []workflows.WorkflowActivity) error  // sdk-typed input — REPLACE
  InsertOutcome(activityRef string, atPos int, outcomeName string, activities []workflows.WorkflowActivity) error  // REPLACE
  DropOutcome(...) error  // ok
  InsertPath(activityRef string, atPos int, pathCaption string, activities []workflows.WorkflowActivity) error  // REPLACE
  DropPath(...) error  // ok
  InsertBranch(activityRef string, atPos int, condition string, activities []workflows.WorkflowActivity) error  // REPLACE
  DropBranch(...) error  // ok
  InsertBoundaryEvent(activityRef string, atPos int, eventType, delay string, activities []workflows.WorkflowActivity) error  // REPLACE
  DropBoundaryEvent(...) error  // ok
  Save() error  // ok

WidgetSerializationBackend (mutation.go):
  SerializeWorkflowActivity(a workflows.WorkflowActivity) (any, error)  // sdk-typed input — REPLACE
```

The **4 read/write methods** in `WorkflowBackend` and the **6 mutator inputs** in `WorkflowMutator` plus **1 serialize method** = 11 sites needing parallel `*Gen` siblings.

### S2.6 ExecContext field

`ExecContext` does not have a `ctx.Workflows` field today. Phase A0 adds it, mirroring `ctx.Microflows`/`ctx.Security`/`ctx.JavaActions`.

### S2.7 MockBackend conformance

`mdl/backend/mock/mock_workflow.go`, `mock_workflow_mutator.go`, `mock_mutation.go` likely return `nil, nil` for unset Funcs (per master plan §5 P3 / CLAUDE.md). Phase C5 brings them into compliance.

### S2.8 Existing repos infrastructure

No `mdl/repos/workflows.go` or `mdl/backend/mpr/repos/workflows.go` exists. Phase A0 creates both.

### S2.9 Storage type names verified

- `Workflows$Workflow` — confirmed at `sdk/mpr/parser_workflow.go:28`
- All activity storage types verified by grep against `modelsdk/gen/workflows/types.go` line numbers in §S2.4

### S2.10 Container linkage

Workflow units live under module containment. Container UUID = module ID (verified by `cmd_workflows.go:24 (h.FindModuleID(wf.ContainerID))` style usage). Codec-decoded gen `*Workflow` does NOT carry `ContainerID` (per master plan §3.7). Resolution: cache helper `listWorkflowsWithContainerGen` (Phase A0).

### S2.11 ELK source-range integration

`describeWorkflowToString` returns a `(string, map[string]elkSourceRange, error)` triple. The map is consumed by `cmd_visualize.go` to pin SVG node positions. Migration to gen MUST preserve the exact `elkSourceRange` shape — the consumer is the visualization layer, NOT the workflow domain. This means the gen-typed `describeWorkflowToStringGen` keeps the same triple signature and the same range-map keys (`activity.Name + "@" + position`).

---

## §3 Risks Specific to Workflows Domain

| # | Risk | Impact | Mitigation |
|---|---|---|---|
| R1 | **Workflow.Title vs WorkflowName dual storage**: gen exposes both `Title() string` (legacy property) and `WorkflowName() element.Element` (Texts$Text wrapper). Setting only one leaves the other stale and Studio Pro may render the wrong value | High — silent UI mismatch | A0.S1 verification: open existing workflow with non-empty Title, assert which gen accessor is non-empty after decode. The mutator already handles this dual write (workflow_mutator.go:60–72) — replicate the same dual-write in execCreateWorkflowGen + describeWorkflowGen |
| R2 | **DUAL polymorphic types** in gen: `UserTaskActivity` vs `UserTask` (4598 vs 2956), `CallMicroflowActivity` vs `CallMicroflowTask` (931 vs 1096). Plus `MultiUserTaskActivity` (3192) + `SingleUserTaskActivity` (4094) — sdk only had `UserTask` | High — picking wrong type silently produces wrong storage `$Type` | A1 verification step: write a smoke test that opens a fixture with a UserTask. Walk `wf.Flow().(*genWf.Flow).ActivitiesItems()` and assert the actual decoded type is `*UserTaskActivity` (the inflow form). Pin in helpers |
| R3 | **gen `SystemTask` does NOT exist** in the listing (verify in A0). Likely subsumed into `MicroflowBasedActivity` (219) or stored as raw `Workflows$SystemTaskActivity` with no narrow gen type. The legacy formatter `formatSystemTask` knows about `Microflow` field; gen path may need to read raw BSON | Medium | A0.S1 verification grep + raw BSON fallback if no gen type. Document in `formatWorkflowActivityGen`'s switch: `case "Workflows$SystemTaskActivity": ... codec.ReadBSONFieldString(elem.Raw(), "Microflow")` |
| R4 | **NoUserSource ↔ EmptyUserSource naming flip**: sdk uses `NoUserSource`, gen uses `EmptyUserSource` (line 1483) | Low — caught at compile/grep | Pin in §S2.4 and helpers; rename in execCreateWorkflowGen |
| R5 | **MicroflowGroupSource / XPathGroupSource ↔ MicroflowGroupTargeting / XPathGroupTargeting**: sdk has "Source", gen has "Targeting" (different semantic separation: source = which user, targeting = whom to assign) | Medium | A0.S1 verification — find an existing fixture with a group source and verify which gen type gets emitted by storage `$Type`. The storage name decides; the Go type name is just the gen alias |
| R6 | **BoundaryEvent split into multiple gen types**: sdk had unified `BoundaryEvent`, gen has `BoundaryEvent` (abstract, 831) + `InterruptingTimerBoundaryEvent` (2222) + `NonInterruptingTimerBoundaryEvent` (3618) + `TimerBoundaryEvent` (4498) | Medium | Phase D `buildBoundaryEvents` becomes a switch on event-type discriminator; formatter walks `elem.TypeName()` |
| R7 | **ParameterMapping splits into 3 gen types**: `MicroflowCallParameterMapping` (CallMicroflowActivity), `WorkflowCallParameterMapping` (CallWorkflowActivity), `PageParameterMapping` (UserTask). sdk used a single `ParameterMapping{Parameter, Expression}` | Medium | Phase D builders pick the right gen type per call-site context. Re-use `MicroflowCallParameterMapping` in `autoBindCallMicroflowGen` rebuild |
| R8 | **AllowedModuleRoles version-prefix bug**: gen `Workflow.AllowedModuleRolesQualifiedNames` uses `ByNameRefList`. Stage 3.3.1 D6a fixed the encoder globally, but verify the fix applies to Workflow round-trips by smoke test in Phase D | Critical if not fixed | Phase D2 round-trip test asserts `[int32(1), "Module.Role"]` shape after CREATE WORKFLOW with allowed roles |
| R9 | **WorkflowMutator already raw-bson** — the mutation logic does NOT need migration; only input parameter sigs change. But the `serializeAndDedup` helper (workflow_mutator.go:781) and `buildSubFlowBson` (798) call `b.SerializeWorkflowActivity(a)` which expects sdk-typed input | High blast radius if not careful | Phase D adds gen-aware serializer in workflow_mutator.go: input is `[]element.Element`, dispatch on `elem.TypeName()` to BSON shape. The raw-bson output stays identical. Old SerializeWorkflowActivity stays for legacy dispatchers until E1 |
| R10 | **describeWorkflowToString dual return** — `map[string]elkSourceRange` is consumed by the visualization layer. Migration MUST preserve the exact key format. Breaking this silently breaks SVG generation | High — silent visualization regression | A2 explicit assertion: after migration, `cmd_visualize` integration test must produce identical SVG output. Pin a snapshot test in cmd_visualize_test.go (or create one) |
| R11 | **Stage 4 boundary at sdk/mpr**: 6 sdk/mpr files import sdk/workflows. Same escape-hatch pattern as security E3 (deprecation header if Stage 4 hasn't landed) | Medium | Acceptance grep in §9 explicitly excludes `^./sdk/mpr/` matches |
| R12 | **Activity formatter overlap with microflows** (per master plan §8.3 hint): some sub-types (Microflow refs, ParameterMapping, qualified-name resolution) were already migrated in Stage 3.2 microflow path. NOT a risk per se, but a **reuse opportunity** — surface in Phase A2 by importing helpers from `cmd_microflows_show_gen.go` (e.g., `formatMicroflowParameterMappings`) | (positive) | A2 audits Stage 3.2 helpers and reuses |
| R13 | **WorkflowParameter shape change**: sdk had `WorkflowParameter{Name, EntityQualifiedName}`. gen `Parameter` (4004) is a single-field Part wrapping the entity QName. The `Name` may have moved to `Parameter.Name()` accessor — verify in A0 | Low | A0 verifies `*genWf.Parameter` accessor names |

---

## §4 Phase A — Read Path Migration

Goal: every read function in `cmd_workflows.go` has a `*Gen` twin reading from `ctx.Workflows` (via `mprread.ListUnitsByType`), and the dispatcher routes to the gen variants. ELK source-range output is bit-identical with the legacy.

### Task A0: `mdl/repos/workflows.go` + `mdl/backend/mpr/repos/workflows.go` + cache helpers + ctx wiring

**Files:**
- Create: `mdl/repos/workflows.go` (interface)
- Create: `mdl/backend/mpr/repos/workflows.go` (direct-mode implementation)
- Create: `mdl/backend/mpr/repos/workflows_test.go`
- Create: `mdl/executor/helpers_workflows_gen.go`
- Create: `mdl/executor/helpers_workflows_gen_test.go`
- Modify: `mdl/executor/exec_context.go` — add `Workflows repos.WorkflowRepository`
- Modify: `mdl/executor/executor.go` — add `workflowsWithContainerGen []ContainerWithGen[*genWf.Workflow]` to `executorCache`
- Modify: BackendFactory wiring

#### A0.S1: Pre-flight gen verification (one-off, NO commit)

```bash
# Verify gen types referenced in §S2.4 actually exist with their listed accessors
grep -nE "^type (SystemTask|MicroflowBasedActivity|MicroflowGroupSource|MicroflowGroupTargeting|XPathGroupSource|XPathGroupTargeting|EmptyUserSource|NoUserSource|FloatingAnnotation|WorkflowAnnotationActivity|Parameter)" modelsdk/gen/workflows/types.go > /tmp/wf-gen-types.txt
cat /tmp/wf-gen-types.txt

# Verify Workflow accessor existence
grep -nE "^func \(o \*Workflow\) (Name|Title|WorkflowName|ContextEntity|Flow|Parameter|AllowedModuleRoles|OnWorkflow|EventSubProcesses|WorkflowV2)" modelsdk/gen/workflows/types.go

# Verify activity union via storage $Type — open an existing fixture and dump
~/go1.26/bin/go run ./cmd/mxcli/ -p reference/test-projects/workflow-fixture/app.mpr -c "describe workflow Demo.Approve"
```

Update §S2.4 storage-name TBD entries (SystemTask, FloatingAnnotation, MicroflowGroupSource, XPathGroupSource) with verified values. Decide whether each gap requires raw-BSON fallback.

#### A0.S2: Failing tests

```go
// mdl/backend/mpr/repos/workflows_test.go
func TestWorkflowRepository_ListAll_DecodesGenTypes(t *testing.T) {
    w, cleanup := openFixtureWriter(t, "testdata/workflow-fixture.mpr")
    defer cleanup()
    repo := NewWorkflowRepository(w)
    wfs, err := repo.ListAll()
    if err != nil { t.Fatalf("ListAll: %v", err) }
    if len(wfs) == 0 { t.Fatal("ListAll empty") }
    if wfs[0].Name() == "" { t.Errorf("Name() empty") }
}

func TestWorkflowRepository_GetContainerUUID_Resolves(t *testing.T) {
    /* mirror microflows_test.go pattern */
}
```

```go
// mdl/executor/helpers_workflows_gen_test.go
func TestListWorkflowsWithContainerGen_CachesAcrossCalls(t *testing.T) {
    ctx := newWorkflowsTestContext(t)
    list1, err := listWorkflowsWithContainerGen(ctx)
    if err != nil { t.Fatalf("listWorkflowsWithContainerGen: %v", err) }
    list2, _ := listWorkflowsWithContainerGen(ctx)
    if len(list1) != len(list2) { t.Fatalf("cache mismatch") }
}
```

#### A0.S3: RED

```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/backend/mpr/repos/ -run TestWorkflowRepository_ -v
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/executor/ -run TestListWorkflowsWithContainerGen_CachesAcrossCalls -v
```
Expected: FAIL.

#### A0.S4: Implement

```go
// mdl/repos/workflows.go
package repos

import (
    "github.com/mendixlabs/mxcli/model"
    genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

type WorkflowReader interface {
    Get(id model.ID) (*genWf.Workflow, error)
    List(moduleID model.ID) ([]*genWf.Workflow, error)
    ListAll() ([]*genWf.Workflow, error)
    FindByQualifiedName(qn string) (*genWf.Workflow, error)
    GetContainerUUID(id model.ID) (model.ID, error)
}

type WorkflowWriter interface {
    Create(parentUUID, containmentName string, wf *genWf.Workflow) error
    Update(wf *genWf.Workflow) error
    Delete(id model.ID) error
}

type WorkflowRepository interface {
    WorkflowReader
    WorkflowWriter
}
```

```go
// mdl/backend/mpr/repos/workflows.go (mirror microflows.go)
package mprrepos

const workflowTypeName = "Workflows$Workflow"

type workflowRepo struct {
    w   *mmpr.Writer
    r   *mmpr.Reader
    dec *decoder
}

func NewWorkflowRepository(w *mmpr.Writer) repos.WorkflowRepository { ... }
func (r *workflowRepo) ListAll() ([]*genWf.Workflow, error) {
    return mprread.ListUnitsByType[*genWf.Workflow](r.r, workflowTypeName)
}
// Get, List, FindByQualifiedName, GetContainerUUID — clones of microflow patterns
// Writer methods stub to fmt.Errorf("not implemented in Phase A"); Phase D fills.

var _ repos.WorkflowRepository = (*workflowRepo)(nil)
```

```go
// mdl/executor/helpers_workflows_gen.go
package executor

import (
    "github.com/mendixlabs/mxcli/model"
    "github.com/mendixlabs/mxcli/modelsdk/element"
    genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

func listWorkflowsWithContainerGen(ctx *ExecContext) ([]ContainerWithGen[*genWf.Workflow], error) {
    if ctx == nil || ctx.Workflows == nil { return nil, nil }
    return listUnitsWithContainerGen(
        func() ([]*genWf.Workflow, error) { return ctx.Workflows.ListAll() },
        func(id element.ID) (element.ID, error) {
            c, err := ctx.Workflows.GetContainerUUID(model.ID(id))
            return element.ID(c), err
        },
        func() ([]ContainerWithGen[*genWf.Workflow], bool) {
            if ctx.Cache != nil && ctx.Cache.workflowsWithContainerGen != nil {
                return ctx.Cache.workflowsWithContainerGen, true
            }
            return nil, false
        },
        func(s []ContainerWithGen[*genWf.Workflow]) {
            if ctx.Cache != nil { ctx.Cache.workflowsWithContainerGen = s }
        },
    )
}

func invalidateWorkflowsCache(ctx *ExecContext) {
    if ctx == nil || ctx.Cache == nil { return }
    ctx.Cache.workflowsWithContainerGen = nil
}
```

#### A0.S5: Wire ctx.Workflows in BackendFactory (find `ctx.Microflows = mprrepos.NewMicroflowRepository`; add parallel line)

#### A0.S6: GREEN + build + full test gate

#### A0.S7: Commit

```bash
git add mdl/repos/workflows.go mdl/backend/mpr/repos/workflows.go mdl/backend/mpr/repos/workflows_test.go \
        mdl/executor/exec_context.go mdl/executor/executor.go \
        mdl/executor/helpers_workflows_gen.go mdl/executor/helpers_workflows_gen_test.go
git commit -m "feat(executor,repos): Stage 3.3.3.A0 — workflows repo + cache helpers + ctx wiring ..."
```

### Task A1: `listWorkflowsGen` + activity counters

**Files:**
- Create: `mdl/executor/cmd_workflows_gen.go`
- Create: `mdl/executor/cmd_workflows_gen_test.go`

#### A1.S1: Failing test

```go
func TestListWorkflowsGen_OutputsActivityCounts(t *testing.T) {
    ctx := newWorkflowsTestContext(t)
    var buf bytes.Buffer
    ctx.Output = &buf
    ctx.Format = FormatTable
    if err := listWorkflowsGen(ctx, ""); err != nil {
        t.Fatalf("listWorkflowsGen: %v", err)
    }
    if !strings.Contains(buf.String(), "Activities") {
        t.Errorf("expected Activities column header: %q", buf.String())
    }
}
```

#### A1.S2: RED
#### A1.S3: Implement (mirror `listWorkflows` cmd_workflows.go:17 + `countWorkflowActivities`)

```go
// listWorkflowsGen handles SHOW WORKFLOWS using gen-typed Workflow units.
func listWorkflowsGen(ctx *ExecContext, moduleName string) error {
    h, err := getHierarchy(ctx)
    if err != nil { return mdlerrors.NewBackend("build hierarchy", err) }
    pairs, err := listWorkflowsWithContainerGen(ctx)
    if err != nil { return mdlerrors.NewBackend("list workflows", err) }

    type row struct { qn, mod, name string; total, ut, dec int }
    var rows []row
    for _, p := range pairs {
        modName := h.GetModuleName(h.FindModuleID(model.ID(p.ContainerID)))
        if moduleName != "" && modName != moduleName { continue }
        total, ut, dec := countWorkflowActivitiesGen(p.Elem)
        rows = append(rows, row{modName + "." + p.Elem.Name(), modName, p.Elem.Name(), total, ut, dec})
    }
    /* sort + render — same shape as listWorkflows */
}

func countWorkflowActivitiesGen(wf *genWf.Workflow) (total, userTasks, decisions int) {
    if wf == nil { return }
    flow, _ := wf.Flow().(*genWf.Flow)
    if flow == nil { return }
    countFlowActivitiesGen(flow, &total, &userTasks, &decisions)
    return
}

func countFlowActivitiesGen(flow *genWf.Flow, total, ut, dec *int) {
    if flow == nil { return }
    for _, a := range flow.ActivitiesItems() {
        if a == nil { continue }
        *total++
        switch a.TypeName() {
        case "Workflows$UserTaskActivity", "Workflows$MultiUserTaskActivity", "Workflows$SingleUserTaskActivity":
            *ut++
        case "Workflows$ExclusiveSplitActivity", "Workflows$ParallelSplitActivity":
            *dec++
        }
        // Recurse into outcomes (ExclusiveSplit/ParallelSplit)
        countNestedFlowsGen(a, total, ut, dec)
    }
}
```

`countNestedFlowsGen` walks ExclusiveSplitActivity outcome list + each outcome's `Flow()` + ParallelSplitActivity paths similarly.

#### A1.S4: GREEN, A1.S5: gate, A1.S6: Commit

### Task A2: `formatWorkflowActivitiesGen` polymorphic dispatch

The largest commit of Phase A. Replaces `formatWorkflowActivities` (cmd_workflows.go:286) with gen-typed dispatch over `flow.ActivitiesItems()`, dispatching by `elem.TypeName()`.

**Files:**
- Modify: `mdl/executor/cmd_workflows_gen.go`
- Modify: `mdl/executor/cmd_workflows_gen_test.go`

#### A2.S1: Tests (one per major activity type)

```go
func TestFormatWorkflowActivitiesGen_UserTask(t *testing.T) {
    flow := genWf.NewFlow()
    ut := genWf.NewUserTaskActivity()
    ut.SetName("ApprovalTask")
    ut.SetTaskPage(/* page ref ... */)
    flow.AddActivities(ut)

    lines := formatWorkflowActivitiesGen(flow, "  ", nil)
    found := false
    for _, l := range lines {
        if strings.Contains(l, "user task ApprovalTask") { found = true; break }
    }
    if !found { t.Errorf("missing user task line: %v", lines) }
}

// Mirror tests for: CallMicroflowActivity, SystemTask, CallWorkflowActivity,
// ExclusiveSplitActivity, ParallelSplitActivity, JumpToActivity, WaitForTimer,
// WaitForNotification, EndWorkflow, FloatingAnnotation, EndOfParallelSplitPath,
// EndOfBoundaryEventPath
```

#### A2.S2: RED
#### A2.S3: Implement

```go
// formatWorkflowActivitiesGen mirrors formatWorkflowActivities (cmd_workflows.go:286)
// but iterates gen Flow.ActivitiesItems() and dispatches on elem.TypeName().
//
// indent: prefix string applied to each line
// typeParams: workflow's parameter list (for resolving entity-type-param references in activity bodies)
func formatWorkflowActivitiesGen(flow *genWf.Flow, indent string, typeParams []element.Element) []string {
    if flow == nil { return nil }
    var lines []string
    for _, a := range flow.ActivitiesItems() {
        if a == nil { continue }
        switch a.TypeName() {
        case "Workflows$StartWorkflowActivity":
            // typically rendered as start marker — see legacy
        case "Workflows$EndWorkflowActivity":
            lines = append(lines, formatEndWorkflowGen(a, indent)...)
        case "Workflows$UserTaskActivity":
            ut, _ := a.(*genWf.UserTaskActivity)
            lines = append(lines, formatUserTaskGen(ut, indent)...)
        case "Workflows$MultiUserTaskActivity":
            mt, _ := a.(*genWf.MultiUserTaskActivity)
            lines = append(lines, formatMultiUserTaskGen(mt, indent)...)
        case "Workflows$SingleUserTaskActivity":
            st, _ := a.(*genWf.SingleUserTaskActivity)
            lines = append(lines, formatSingleUserTaskGen(st, indent)...)
        case "Workflows$CallMicroflowActivity":
            cm, _ := a.(*genWf.CallMicroflowActivity)
            lines = append(lines, formatCallMicroflowActivityGen(cm, indent)...)
        case "Workflows$SystemTaskActivity":
            // schema gap: gen has no SystemTask narrow type — read raw BSON
            lines = append(lines, formatSystemTaskGen(a, indent)...)
        case "Workflows$CallWorkflowActivity":
            cw, _ := a.(*genWf.CallWorkflowActivity)
            lines = append(lines, formatCallWorkflowActivityGen(cw, indent)...)
        case "Workflows$ExclusiveSplitActivity":
            es, _ := a.(*genWf.ExclusiveSplitActivity)
            lines = append(lines, formatExclusiveSplitGen(es, indent)...)
        case "Workflows$ParallelSplitActivity":
            ps, _ := a.(*genWf.ParallelSplitActivity)
            lines = append(lines, formatParallelSplitGen(ps, indent)...)
        case "Workflows$JumpToActivity":
            jt, _ := a.(*genWf.JumpToActivity)
            lines = append(lines, formatJumpToGen(jt, indent)...)
        case "Workflows$WaitForTimerActivity":
            wt, _ := a.(*genWf.WaitForTimerActivity)
            lines = append(lines, formatWaitForTimerGen(wt, indent)...)
        case "Workflows$WaitForNotificationActivity":
            wn, _ := a.(*genWf.WaitForNotificationActivity)
            lines = append(lines, formatWaitForNotificationGen(wn, indent)...)
        case "Workflows$EndOfParallelSplitPathActivity",
             "Workflows$EndOfBoundaryEventPathActivity":
            // sentinel — usually rendered as end-of-block marker
        case "Workflows$FloatingAnnotation":
            lines = append(lines, formatAnnotationGen(a, indent)...)
        default:
            // Generic fallback — render type name + name field
            lines = append(lines, fmt.Sprintf("%s-- unknown activity: %s", indent, a.TypeName()))
        }
    }
    return lines
}
```

The per-activity formatters (`formatUserTaskGen`, etc.) are each individual sub-tasks (A2.a–A2.l). One commit each, OR group small ones (~3 per commit) — author's discretion based on reviewability.

#### A2.S4: GREEN per sub-task
#### A2.S5: gate
#### A2.S6: Commit per sub-task

```bash
git commit -m "feat(executor): Stage 3.3.3.A2.<x> — format<X>Gen polymorphic dispatch"
```

### Task A3: `formatUserTaskGen` (the heavy-lift formatter)

UserTask is the most complex activity (93 LoC in legacy `formatUserTask`). Renders:
- Task page reference + Task name (Texts$Text wrapper)
- Task description (Texts$Text wrapper)
- Allowed module roles
- User source (None / Microflow / XPath / GroupMicroflow / GroupXPath)
- User targeting (Multi: completion criteria, threshold)
- Outcomes
- Boundary events
- Annotations

#### A3.S1: Tests (one per major rendering case — about 10 sub-tests)
#### A3.S2: RED
#### A3.S3: Implement

```go
func formatUserTaskGen(ut *genWf.UserTaskActivity, indent string) []string {
    if ut == nil { return nil }
    var lines []string
    lines = append(lines, fmt.Sprintf("%suser task %s", indent, ut.Name()))
    if name := readTextElement(ut.TaskName()); name != "" {
        lines = append(lines, fmt.Sprintf("%s  page name: '%s'", indent, name))
    }
    if desc := readTextElement(ut.TaskDescription()); desc != "" {
        lines = append(lines, fmt.Sprintf("%s  description: '%s'", indent, desc))
    }
    if pq := ut.PageQualifiedName(); pq != "" {
        lines = append(lines, fmt.Sprintf("%s  page: %s", indent, pq))
    }
    if dueDate := ut.DueDate(); dueDate != "" {
        lines = append(lines, fmt.Sprintf("%s  due_date: '%s'", indent, dueDate))
    }
    // Allowed module roles (uses ByNameRefList — versioned-array fix from Stage 3.3.1 D6a)
    if roles := ut.AllowedModuleRolesQualifiedNames(); len(roles) > 0 {
        lines = append(lines, fmt.Sprintf("%s  allowed: %s", indent, strings.Join(roles, ", ")))
    }
    // User source (NoUserSource → None, MicroflowBased → microflow:, XPathBased → xpath:)
    if src := ut.UserSource(); src != nil {
        lines = append(lines, formatUserSourceGen(src, indent + "  ")...)
    }
    // User targeting (single user vs multi user)
    if tgt := ut.UserTargeting(); tgt != nil {
        lines = append(lines, formatUserTargetingGen(tgt, indent + "  ")...)
    }
    // Outcomes (UserTaskOutcome)
    for _, oc := range ut.OutcomesItems() {
        lines = append(lines, formatUserTaskOutcomeGen(oc, indent + "  ")...)
    }
    // Boundary events
    for _, ev := range ut.BoundaryEventsItems() {
        lines = append(lines, formatBoundaryEventGen(ev, indent + "  ")...)
    }
    return lines
}

// readTextElement extracts the inner Text scalar from a Texts$Text wrapper element.
// Returns empty string if elem is nil or has no Text field.
func readTextElement(elem element.Element) string {
    if elem == nil { return "" }
    return codec.ReadBSONFieldString(elem.Raw(), "Text")
}

func formatUserSourceGen(src element.Element, indent string) []string {
    switch src.TypeName() {
    case "Workflows$EmptyUserSource":
        return []string{fmt.Sprintf("%s  source: none", indent)}
    case "Workflows$MicroflowBasedUserSource":
        if mf, ok := src.(*genWf.MicroflowBasedUserSource); ok {
            return []string{fmt.Sprintf("%s  source: microflow %s", indent, mf.MicroflowQualifiedName())}
        }
    case "Workflows$XPathBasedUserSource":
        if xp, ok := src.(*genWf.XPathBasedUserSource); ok {
            return []string{fmt.Sprintf("%s  source: xpath '%s'", indent, xp.XPathConstraint())}
        }
    }
    return nil
}
```

(`UserSource()` accessor name MUST be verified in A0.S1 — gen may use a different name.)

#### A3.S4: GREEN per sub-test
#### A3.S5: gate
#### A3.S6: Commit `feat(executor): Stage 3.3.3.A3 — formatUserTaskGen (gen-typed, all sub-fields)`

### Task A4: `describeWorkflowGen` + ELK source-range preservation

**Files:**
- Modify: `mdl/executor/cmd_workflows_gen.go`
- Modify: `mdl/executor/cmd_workflows_gen_test.go`

#### A4.S1: Test — assert MDL output + ELK source-range map

```go
func TestDescribeWorkflowGen_OutputAndSourceRanges(t *testing.T) {
    ctx := newWorkflowsTestContext(t)
    out, ranges, err := describeWorkflowToStringGen(ctx, ast.QualifiedName{Module: "Demo", Name: "Approve"})
    if err != nil { t.Fatalf("describeWorkflowToStringGen: %v", err)}
    if !strings.Contains(out, "create workflow Demo.Approve") {
        t.Errorf("missing create statement: %q", out)
    }
    if len(ranges) == 0 {
        t.Errorf("expected ELK source ranges, got empty")
    }
    // Snapshot test: ELK ranges must match legacy
    legacyOut, legacyRanges, _ := describeWorkflowToString(ctx, ast.QualifiedName{Module: "Demo", Name: "Approve"})
    if !mapsEqual(ranges, legacyRanges) {
        t.Errorf("ELK ranges drift:\nlegacy: %v\ngen: %v", legacyRanges, ranges)
    }
    if out != legacyOut {
        t.Errorf("MDL output drift:\nlegacy:\n%s\ngen:\n%s", legacyOut, out)
    }
}
```

#### A4.S2: RED
#### A4.S3: Implement

```go
func describeWorkflowGen(ctx *ExecContext, name ast.QualifiedName) error {
    out, _, err := describeWorkflowToStringGen(ctx, name)
    if err != nil { return err }
    fmt.Fprintln(ctx.Output, out)
    return nil
}

func describeWorkflowToStringGen(ctx *ExecContext, name ast.QualifiedName) (string, map[string]elkSourceRange, error) {
    qn := name.Module + "." + name.Name
    wf, err := ctx.Workflows.FindByQualifiedName(qn)
    if err != nil || wf == nil {
        return "", nil, mdlerrors.NewNotFound("workflow", qn)
    }
    var sb strings.Builder
    ranges := make(map[string]elkSourceRange)

    // Documentation (JavaDoc style)
    if doc := wf.Documentation(); doc != "" { writeDocCommentGen(&sb, doc) }

    // CREATE WORKFLOW header
    fmt.Fprintf(&sb, "create workflow %s\n", qn)
    if title := wf.Title(); title != "" {
        fmt.Fprintf(&sb, "  display: '%s'\n", title)
    } else if name := readTextElement(wf.WorkflowName()); name != "" {
        fmt.Fprintf(&sb, "  display: '%s'\n", name)
    }
    if desc := readTextElement(wf.WorkflowDescription()); desc != "" {
        fmt.Fprintf(&sb, "  description: '%s'\n", desc)
    }
    if entity := wf.WorkflowEntityQualifiedName(); entity != "" {
        fmt.Fprintf(&sb, "  entity: %s\n", entity)
    }
    if param := wf.Parameter(); param != nil {
        if p, ok := param.(*genWf.Parameter); ok {
            fmt.Fprintf(&sb, "  parameter: %s\n", p.EntityQualifiedName())
        }
    }
    if op := wf.OverviewPageQualifiedName(); op != "" {
        fmt.Fprintf(&sb, "  overview: %s\n", op)
    }
    if dueDate := wf.DueDate(); dueDate != "" {
        fmt.Fprintf(&sb, "  due_date: '%s'\n", dueDate)
    }
    if roles := wf.AllowedModuleRolesQualifiedNames(); len(roles) > 0 {
        fmt.Fprintf(&sb, "  allowed: %s\n", strings.Join(roles, ", "))
    }

    // Flow body
    if flow, ok := wf.Flow().(*genWf.Flow); ok {
        sb.WriteString("flow\n")
        lines := formatWorkflowActivitiesGen(flow, "  ", nil)
        // Walk lines and record ELK ranges as we go
        for i, line := range lines {
            sb.WriteString(line + "\n")
            if isActivityLine(line) {
                ranges[activityKeyFromLine(line)] = elkSourceRange{Start: i, End: i+1}
            }
        }
        sb.WriteString("end\n")
    }
    sb.WriteString(";\n")
    return sb.String(), ranges, nil
}
```

Key constraint per R10: the ranges map MUST match the legacy. The simplest path is **build the legacy + gen output from the same fixture** in a test and assert byte-identity. Any drift becomes a test failure.

#### A4.S4: GREEN, A4.S5: gate, A4.S6: Commit

### Task A5: Dispatcher cutover (executor_query.go)

Switch every dispatch site from legacy to `*Gen` variant.

**Files:**
- Modify: `mdl/executor/executor_query.go`
- Modify: `mdl/executor/executor_describe.go` (if it has DESCRIBE WORKFLOW)
- Modify: any other dispatcher

```bash
grep -nE "listWorkflows\b|describeWorkflow\b|describeWorkflowToString\b" \
  mdl/executor/executor_query.go mdl/executor/register_stubs.go mdl/executor/executor_describe.go
```

Replace each with `*Gen`. Build + test gate. Commit.

```bash
git commit -m "refactor(executor): Stage 3.3.3.A5 — dispatch all SHOW/DESCRIBE workflow to gen variants"
```

---

## §5 Phase B — Visualization

The workflow ELK SVG generation is **threaded through `describeWorkflowToString`** (it returns the source-range map as second return value). Phase A4 already preserves the source-range output bit-identically.

**Standalone Phase B tasks:**

### Task B1: Visualization integration test

**Files:**
- Modify: `cmd_visualize_test.go` (or create one if missing)

#### B1.S1: Test — assert SVG identical for `mxcli visualize workflow Demo.Approve` before vs after migration
#### B1.S2: RED if it drifts
#### B1.S3: Fix `formatWorkflowActivitiesGen` source-range emission to match
#### B1.S4: Commit `test(executor): Stage 3.3.3.B1 — workflow visualization SVG snapshot test`

(If SVG already byte-identical from A4, B1 is a verify-only commit; no implementation change.)

---

## §6 Phase C — Consumer Migration

For each non-executor file that imports `sdk/workflows`, switch to gen types.

### Task C1: `mdl/backend/workflow.go` + `mdl/backend/mpr/backend.go` — additive gen-typed methods

Mirror security plan C1: add `ListWorkflowsGen`, `GetWorkflowGen`, `CreateWorkflowGen`, `UpdateWorkflowGen` BEFORE retiring legacy.

**Files:**
- Modify: `mdl/backend/workflow.go` (add 4 gen-typed methods)
- Modify: `mdl/backend/mpr/backend.go` (impl via `mprrepos.NewWorkflowRepository(b.msdkWriter)`)
- Modify: `mdl/backend/mock/backend.go` + `mock_workflow.go` (Func-field stubs with descriptive errors)

#### C1.S1: Test, C1.S2: RED, C1.S3: Implement, C1.S4: GREEN, C1.S5: gate, C1.S6: Commit `feat(backend): Stage 3.3.3.C1 — add gen-typed workflow read/write methods to FullBackend`

### Task C2: `mdl/backend/mutation.go` — WorkflowMutator inputs

Add gen-typed `*Gen` siblings for the 6 mutator methods accepting `[]workflows.WorkflowActivity`:
- `InsertAfterActivityGen(activityRef, atPos, activities []element.Element) error`
- `ReplaceActivityGen(...)`, `InsertOutcomeGen(...)`, `InsertPathGen(...)`, `InsertBranchGen(...)`, `InsertBoundaryEventGen(...)`

Plus `SerializeWorkflowActivityGen(a element.Element) (any, error)` on `WidgetSerializationBackend`.

**Files:**
- Modify: `mdl/backend/mutation.go`
- Modify: `mdl/backend/mpr/workflow_mutator.go` (add `*Gen` impls; old impls call new ones via type-conversion adapter)
- Modify: `mdl/backend/mock/mock_workflow_mutator.go`, `mock_mutation.go`

#### C2.S1–C2.S6: TDD per method, Commit `feat(backend,mock): Stage 3.3.3.C2 — gen-typed WorkflowMutator + serializer methods`

### Task C3: `mdl/catalog/builder_workflows.go` + `builder_references.go` + `builder_strings.go`

Migrate the activity-type type switches from `*workflows.UserTask` etc. to `*genWf.UserTaskActivity` (and friends).

**Files:**
- Modify: `mdl/catalog/builder_workflows.go` (5 cases)
- Modify: `mdl/catalog/builder_references.go` (extractWorkflowFlowRefs + condition-outcome ref extractor)
- Modify: `mdl/catalog/builder_strings.go` (extractWorkflowFlowStrings + 5 cases)
- Modify: `mdl/catalog/builder.go` (Reader.ListWorkflows return type)
- Modify: `mdl/catalog/builder_workflows_test.go` (fixture migration)

#### C3.S1: Test — pick one catalog test that exercises workflow extraction; adapt to gen accessors
#### C3.S2: RED
#### C3.S3: Implement — switch `case *workflows.UserTask:` → `case *genWf.UserTaskActivity:`. Replace `a.UserSource.(*workflows.MicroflowBasedUserSource).Microflow` style accesses with gen equivalents (`a.UserSource().(*genWf.MicroflowBasedUserSource).MicroflowQualifiedName()`)
#### C3.S4: GREEN, C3.S5: gate, C3.S6: Commit `refactor(catalog): Stage 3.3.3.C3 — workflow extractors on gen types`

### Task C4: `mdl/executor/cmd_structure.go` + `cmd_structure_gen.go` — outputWorkflows signature

Migrate `outputWorkflows(ctx, mod, []*workflows.Workflow, ...)` to gen-typed equivalent. Update `structureWfMapGen` value type from `[]*workflows.Workflow` to `[]*genWf.Workflow`. The `cmd_structure_gen.go::loadGenStructureMaps` call to `ctx.Backend.ListWorkflows()` becomes `ctx.Backend.ListWorkflowsGen()` (added in C1).

**Files:**
- Modify: `mdl/executor/cmd_structure.go`
- Modify: `mdl/executor/cmd_structure_gen.go` (line 342 + structureWfMapGen type)

#### C4.S1: Test, C4.S2: RED, C4.S3: Implement, C4.S4: GREEN, C4.S5: gate, C4.S6: Commit `refactor(executor): Stage 3.3.3.C4 — structure outputWorkflows on gen types`

### Task C5: MockBackend audit (mock_workflow.go, mock_workflow_mutator.go, mock_mutation.go)

Bring all three mock files into compliance: `nil, nil` → `nil, fmt.Errorf("MockBackend.X not configured")`.

**Files:** `mdl/backend/mock/mock_workflow.go`, `mock_workflow_mutator.go`, `mock_mutation.go`

#### C5.S1–C5.S6: TDD, Commit `refactor(mock): Stage 3.3.3.C5 — MockBackend workflow stubs return descriptive errors`

### Task C6: Mock test fixture migration

Migrate sdk-typed fixtures in:
- `mdl/executor/cmd_workflows_describe_test.go`
- `mdl/executor/cmd_workflows_mock_test.go`
- `mdl/executor/cmd_workflows_write_gen_test.go` (only the workflow-side fixtures)
- `mdl/executor/cmd_error_mock_test.go`, `cmd_json_mock_test.go`, `cmd_rename_mock_test.go`
- `mdl/executor/mock_test_helpers_test.go`
- `mdl/executor/validate_duplicates_test.go`
- `mdl/backend/mpr/workflow_mutator_test.go`
- `mdl/catalog/builder_workflows_test.go`

ALL in ONE commit so cross-file fixture references stay consistent.

#### C6.S1–C6.S6: TDD, Commit `test(executor,backend,catalog): Stage 3.3.3.C6 — migrate workflow fixtures to gen builders`

### Task C7: `mdl/executor/cmd_alter_workflow.go` + `validate_duplicates.go` + `cmd_rename.go`

Three small consumers that each have one or two `ctx.Backend.ListWorkflows()` calls. Switch to `ctx.Workflows.ListAll()` (or `listWorkflowsWithContainerGen` if container UUID is needed).

**Files:** the three files listed

#### C7.S1: Test — at least one for each file ensuring the renamed/altered/validated workflow's name flows through
#### C7.S2: RED
#### C7.S3: Implement — replace the call sites
#### C7.S4: GREEN, C7.S5: gate, C7.S6: Commit `refactor(executor): Stage 3.3.3.C7 — alter/rename/validate workflow callers on gen`

### Task C8: `mdl/backend/mpr/mf_page_modelsdk.go` — incidental import

One ref to `workflows.` in this file. Likely a forward-declared type or stub. Verify and remove the import.

**Files:** `mdl/backend/mpr/mf_page_modelsdk.go`

#### C8: Single-file verify-and-fix commit. `refactor(backend): Stage 3.3.3.C8 — drop sdk/workflows from mf_page_modelsdk.go`

---

## §7 Phase D — Write Path Migration

Goal: replace `cmd_workflows_write.go` with gen-typed equivalents that build `*genWf.*` activities from AST and route through `CreateWorkflowGen`/`UpdateWorkflowGen` and the gen-typed mutator inputs from C2.

### Task D1: AST → gen activity converters

**Files:**
- Create: `mdl/executor/cmd_workflows_write_gen2.go` (or extend existing `cmd_workflows_write_gen.go`)
- Modify: `mdl/executor/cmd_workflows_write_gen_test.go`

#### D1.S1: Tests — one per AST kind (UserTask, CallMicroflow, CallWorkflow, ExclusiveSplit, ParallelSplit, JumpTo, WaitForTimer, WaitForNotification, EndWorkflow, FloatingAnnotation)
#### D1.S2: RED
#### D1.S3: Implement

```go
// buildWorkflowActivitiesGen mirrors buildWorkflowActivities (cmd_workflows_write.go:170)
// returning []element.Element. The dispatch is by AST node Go type.
func buildWorkflowActivitiesGen(nodes []ast.WorkflowActivityNode) []element.Element {
    var out []element.Element
    for _, n := range nodes {
        if elem := buildWorkflowActivityGen(n); elem != nil {
            out = append(out, elem)
        }
    }
    return out
}

func buildWorkflowActivityGen(node ast.WorkflowActivityNode) element.Element {
    switch n := node.(type) {
    case *ast.WorkflowUserTaskNode:
        return buildUserTaskGen(n)
    case *ast.WorkflowCallMicroflowNode:
        return buildCallMicroflowActivityGen(n)
    case *ast.WorkflowCallWorkflowNode:
        return buildCallWorkflowActivityGen(n)
    case *ast.WorkflowDecisionNode:
        return buildExclusiveSplitGen(n)
    case *ast.WorkflowParallelSplitNode:
        return buildParallelSplitGen(n)
    case *ast.WorkflowJumpToNode:
        return buildJumpToGen(n)
    case *ast.WorkflowWaitForTimerNode:
        return buildWaitForTimerGen(n)
    case *ast.WorkflowWaitForNotificationNode:
        return buildWaitForNotificationGen(n)
    case *ast.WorkflowEndNode:
        return buildEndWorkflowGen(n)
    case *ast.WorkflowAnnotationActivityNode:
        return buildAnnotationActivityGen(n)
    }
    return nil
}

func buildUserTaskGen(n *ast.WorkflowUserTaskNode) *genWf.UserTaskActivity {
    ut := genWf.NewUserTaskActivity()
    ut.SetName(sanitizeActivityName(n.Name))
    if n.Caption != "" { ut.SetCaption(n.Caption) }
    if n.PageQualifiedName != "" { ut.SetPageQualifiedName(n.PageQualifiedName) }
    if n.DueDate != "" { ut.SetDueDate(n.DueDate) }
    if len(n.AllowedRoles) > 0 {
        ut.SetAllowedModuleRolesQualifiedNames(n.AllowedRoles)
    }
    // UserSource: dispatch on n.UserSource kind → NoUser/Microflow/XPath
    if n.UserSource != nil {
        ut.SetUserSource(buildUserSourceGen(n.UserSource))
    }
    // Outcomes
    for _, oc := range n.Outcomes {
        ut.AddOutcomes(buildUserTaskOutcomeGen(oc))
    }
    // Boundary events
    for _, ev := range n.BoundaryEvents {
        ut.AddBoundaryEvents(buildBoundaryEventGen(ev))
    }
    return ut
}

// buildUserSourceGen — dispatch on n.Kind:
//   "none" → genWf.NewEmptyUserSource()  (note R4 naming flip)
//   "microflow" → genWf.NewMicroflowBasedUserSource() + SetMicroflowQualifiedName
//   "xpath" → genWf.NewXPathBasedUserSource() + SetXPathConstraint
func buildUserSourceGen(n *ast.WorkflowUserSourceNode) element.Element { ... }
```

(Each builder is independently testable; group ≤3 per commit per master plan §3.5 single-concern guideline.)

#### D1.S4: GREEN per group, D1.S5: gate, D1.S6: Commit per group

### Task D2: `execCreateWorkflowGen`

Replaces `execCreateWorkflow` (cmd_workflows_write.go:20). Uses `buildWorkflowActivitiesGen` from D1.

**Files:**
- Modify: `mdl/executor/cmd_workflows_write_gen2.go`

#### D2.S1: Test — round-trip CREATE WORKFLOW with: title, description, entity, allowed roles, parameter, flow with UserTask + ExclusiveSplit. Assert via gen describe + `mx check` smoke
#### D2.S2: RED
#### D2.S3: Implement (mirror execCreateWorkflow with gen builders + `ctx.Backend.CreateWorkflowGen` from C1)
#### D2.S4: GREEN, D2.S5: gate (include `mx check` per R8), D2.S6: Commit `feat(executor): Stage 3.3.3.D2 — execCreateWorkflowGen`

### Task D3: `execDropWorkflowGen`

Mirror `execDropWorkflow`; list via `listWorkflowsWithContainerGen`, delete via `ctx.Backend.DeleteWorkflow(id)` (unchanged).

#### D3.S1–D3.S6: TDD, Commit `feat(executor): Stage 3.3.3.D3 — execDropWorkflowGen`

### Task D4: Rebuild `autoBindCallMicroflowGen` for gen workflow side

The existing `cmd_workflows_write_gen.go::autoBindCallMicroflowGen` consumes microflows via gen but mutates a sdk-typed `*workflows.CallMicroflowTask`. Rebuild to take gen `*genWf.CallMicroflowActivity` and append `*genWf.MicroflowCallParameterMapping` to its `ParameterMappingsItems()`.

**Files:** `mdl/executor/cmd_workflows_write_gen.go`

#### D4.S1: Test — assert gen CallMicroflowActivity has correct ParameterMappings after auto-bind
#### D4.S2: RED, D4.S3: Implement (rename old to `autoBindCallMicroflowGenLegacy`; new fn takes `*genWf.CallMicroflowActivity`)
#### D4.S4: GREEN, D4.S5: gate, D4.S6: Commit `refactor(executor): Stage 3.3.3.D4 — autoBindCallMicroflowGen on gen workflow types`

### Task D5: Rebuild `autoBindCallWorkflowGen` (parallel to D4 for CallWorkflowActivity + WorkflowCallParameterMapping)

#### D5.S1–D5.S6: TDD, Commit `feat(executor): Stage 3.3.3.D5 — autoBindCallWorkflowGen on gen types`

### Task D6: Wire write dispatchers (`register_stubs.go`)

Switch `execCreateWorkflow` and `execDropWorkflow` registrations to `*Gen` variants.

#### D6.S1–D6.S4: locate / replace / build+test / commit `refactor(executor): Stage 3.3.3.D6 — dispatch CREATE/DROP WORKFLOW to gen variants`

### Task D7: WorkflowMutator gen-aware serializer

The mutator's `serializeAndDedup` and `buildSubFlowBson` (workflow_mutator.go:781, 798) call `b.SerializeWorkflowActivity(a)` with sdk types. Add gen-aware path: input is `[]element.Element`; dispatch on `elem.TypeName()` to BSON shape using `codec.Encode`.

**Files:** `mdl/backend/mpr/workflow_mutator.go`

#### D7.S1–D7.S6: TDD, Commit `feat(backend/mpr): Stage 3.3.3.D7 — WorkflowMutator gen-aware activity serializer`

### Task D8: Wire ALTER WORKFLOW to use gen-typed mutator inputs

`cmd_alter_workflow.go::execAlterWorkflow` builds `[]workflows.WorkflowActivity` and passes to mutator methods. Replace with `[]element.Element` (gen).

**Files:** `mdl/executor/cmd_alter_workflow.go`

#### D8.S1–D8.S6: TDD, Commit `refactor(executor): Stage 3.3.3.D8 — ALTER WORKFLOW uses gen mutator inputs`

---

## §8 Phase E — Cleanup

### Task E1: Retire `FullBackend.{ListWorkflows,GetWorkflow,CreateWorkflow,UpdateWorkflow}` + sdk-typed mutator inputs

**Files:**
- Modify: `mdl/backend/workflow.go` (delete legacy methods)
- Modify: `mdl/backend/mutation.go` (drop sdk types from `WorkflowMutator` + `WidgetSerializationBackend.SerializeWorkflowActivity`)
- Modify: `mdl/backend/mpr/backend.go` (delete shim methods)
- Modify: `mdl/backend/mpr/workflow_mutator.go` (delete sdk-input methods; rename `*Gen` versions)
- Modify: `mdl/backend/mock/*.go` (delete corresponding Func fields and shims)

#### E1.S1: Build to confirm 0 callers
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./...
```

#### E1.S2: Delete + commit `refactor(backend): Stage 3.3.3.E1 — retire FullBackend workflow sdk-typed methods + mutator inputs`

### Task E2: Delete legacy executor workflow files

**Files:**
- Delete: `mdl/executor/cmd_workflows.go`
- Delete: `mdl/executor/cmd_workflows_write.go`
- Modify: `mdl/executor/cmd_workflows_write_gen.go` (drop the legacy autoBindCallMicroflow shim if unused)

#### E2.S1: Final grep — no caller in mdl/executor uses anything from those files
#### E2.S2: `git rm`
#### E2.S3: Build + full test gate
#### E2.S4: Commit `refactor(executor): Stage 3.3.3.E2 — delete legacy cmd_workflows{,_write}.go`

### Task E3: Delete `sdk/workflows` package (conditional on Stage 4)

**Files:** `sdk/workflows/workflow.go`

#### E3.S1: Final acceptance grep
```bash
grep -rln '"github.com/mendixlabs/mxcli/sdk/workflows"' . --include="*.go" | grep -v "^./sdk/mpr/"
```
Expected: empty.

#### E3.S2: Verify Stage 4 boundary status — same escape-hatch logic as security plan §11

#### E3.S3: Delete + build + test gate

#### E3.S4: Commit `refactor: Stage 3.3.3.E3 — delete sdk/workflows package`

```
Aggregate Stage 3.3.3 stats:
- Commits: ~45
- LoC delta: -2270 approx (sdk/workflows 384 + cmd_workflows 669 + cmd_workflows_write 790 + sdk-typed mutator inputs ~50 + catalog sdk ~30 + structure sdk ~30 + small consumers ~20)
                  +1800 approx (gen helpers + cmd_workflows_gen.go + cmd_workflows_write_gen2.go + repo + extended tests)
```

### Task E4: Final acceptance verification

#### E4.S1: Acceptance greps
```bash
grep -rln '"github.com/mendixlabs/mxcli/sdk/workflows"' . --include="*.go" | grep -v "^./sdk/mpr/"
grep -rln '"github.com/mendixlabs/mxcli/sdk/workflows"' modelsdk/ --include="*.go"
grep -rln '"github.com/mendixlabs/mxcli/sdk/workflows"' api/ --include="*.go"
```
All three: empty.

#### E4.S2: Run workflow-affecting tests
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/executor/ ./mdl/backend/mpr/ ./mdl/backend/mpr/repos/ ./mdl/catalog/ -run "Workflow|workflow" -count=1 -v
```

#### E4.S3: Verify cache helpers exist — `listWorkflowsWithContainerGen` in `mdl/executor/helpers_workflows_gen.go`

#### E4.S4: Update memory `project_stage_3_3_workflows_complete.md` with final stats

#### E4.S5: No commit needed

---

## §9 Acceptance Criteria

- [ ] `grep -rln '"github.com/mendixlabs/mxcli/sdk/workflows"' . --include="*.go" | grep -v "^./sdk/mpr/"` returns 0 lines
- [ ] All `Test*Workflow*` and `Test*workflow*` tests pass:
      `GOPROXY=... ~/go1.26/bin/go test ./mdl/executor/ ./mdl/backend/mpr/ ./mdl/backend/mpr/repos/ ./mdl/catalog/ -run "Workflow" -count=1 -v`
- [ ] `listWorkflowsWithContainerGen` (cache helper) is the only path used in `mdl/executor/` for workflow listing; no `ctx.Backend.ListWorkflowsGen()` raw calls outside that helper
- [ ] `ctx.Workflows` repo is the only path for Workflow reads in `mdl/executor/`
- [ ] `mx check` smoke test passes on a round-tripped MPR with: scalar CREATE WORKFLOW, ALTER (Insert/Drop/Replace activity, Insert/Drop outcome, Insert/Drop path, Insert/Drop branch, Insert/Drop boundary event), DROP
- [ ] **Visualization parity**: `mxcli visualize workflow Demo.Approve --format svg` produces byte-identical output before vs after migration (snapshot test in B1)
- [ ] **AllowedModuleRoles version-prefix correct**: every CREATE WORKFLOW with allowed roles produces BSON with `[int32(1), "Module.Role"]` shape (verified by D2 round-trip test)
- [ ] Full repo build: `GOPROXY=... ~/go1.26/bin/go build ./...`
- [ ] Full repo test suite: `GOPROXY=... ~/go1.26/bin/go test ./... -count=1 -timeout 240s`
- [ ] `sdk/workflows/` directory is gone (or, if Stage 4 hasn't landed, kept with deprecation header)

---

## §10 Estimated Commit Count + Sequencing

| Phase | Tasks | Commits | Cumulative |
|---|---|---|---|
| A — Read path | A0, A1, A2 (12 sub-commits per activity-type), A3, A4, A5 | 17 | 17 |
| B — Visualization | B1 (snapshot + verify) | 1 | 18 |
| C — Consumer migration | C1, C2, C3, C4, C5, C6, C7, C8 | 8 | 26 |
| D — Write path | D1 (10 sub-commits per activity builder, grouped to ~4), D2, D3, D4, D5, D6, D7, D8 | 11 | 37 |
| E — Cleanup | E1, E2, E3, E4 | 3 (E4 verify-only) | 40 |

**Estimated total: ~40 commits** (within master plan §6 row #3's 40–55 range; lower end because activity formatters can be grouped).

**Sequencing rationale:**
- A0 first (repo + ctx wiring) — load-bearing
- A1–A2 add gen-typed read functions; A2 sub-tasks per activity type (testable independently)
- A3 (UserTask) is the heaviest single formatter — own commit
- A4 (describeWorkflowGen + ELK ranges) MUST come after A2/A3 (consumes them)
- A5 dispatcher cutover after A1–A4 land
- B1 snapshot test verifies A4 didn't drift the visualization
- C1–C2 (additive backend gen methods) BEFORE D2/D8 (which depend on `CreateWorkflowGen` + gen mutator inputs)
- C5 (mock audit) BEFORE C6 (fixture migration consumes the new mock contract)
- D6 dispatcher AFTER D2/D3
- D7 (mutator gen serializer) BEFORE D8 (ALTER WORKFLOW migration consumes it)
- E1–E3 strictly after both A5 and D6 dispatchers route to gen

**Per-session checkpoints:** commit after each numbered task. Mid-A2 interruption is fine because each activity formatter is independent.

---

## §11 Coordination With Stage 4 Team

### Stage 4's territory
- `sdk/mpr/parser_workflow.go`, `sdk/mpr/workflow_parse_test.go`
- `sdk/mpr/writer_workflow.go`, `sdk/mpr/workflow_write_test.go`
- `sdk/mpr/reader_documents.go` (incidental import)
- `sdk/mpr/serialize_exports.go` (re-exports SerializeWorkflow)

### Stage 3.3.3 commitment
**Stage 3.3.3 will NOT modify any file under `sdk/mpr/`.** E3 deletion conditional on Stage 4 having removed all 6 sdk/mpr importers; otherwise deprecation header.

### Risk of merge collision
Low. Stage 3.3.3 touches `mdl/`, `modelsdk/property/` is unchanged (the D6a fix from Stage 3.3.1 already landed and is reused). Stage 4 touches `sdk/mpr/`. **No file is touched by both teams.**

### Communication
The team-lead should mention to Stage 4:
1. Phase E3 conditional-delete pattern (same as security E3, javaactions E3).
2. The `mdl/backend/mpr/workflow_mutator.go::serializeAndDedup` rebuild in D7 may surface BSON-shape differences against `sdk/mpr.SerializeWorkflowActivity`. Coordinate the test-update if needed.
3. `mdl/types/` does NOT have a `Workflow` type — there's nothing to retire on the types side (unlike javaactions which had `types.JavaScriptAction`).

---

## §12 Self-Review Checklist (skill-required)

**Spec coverage:** §4 Phase A covers all 8 read funcs in `cmd_workflows.go` plus dispatcher (A5). §6 Phase C covers 13 in-scope consumer files (catalog × 4, mock × 3, structure × 2, executor × 4). §7 Phase D covers 21 write funcs from `cmd_workflows_write.go` plus mutator/serializer migration. §8 Phase E covers 3 cleanup commits + verification. §11 Stage 4 boundary. ✓

**Type consistency:** All gen accessor names verified against `modelsdk/gen/workflows/types.go` per §S2.4 with line numbers. TBD entries (`SystemTask`, `MicroflowGroupSource`, `XPathGroupSource`, `FloatingAnnotation` storage names) flagged with R3/R5 risks + explicit A0.S1 verification step. The dual-type families (UserTask/UserTaskActivity, CallMicroflowTask/CallMicroflowActivity) flagged with R2 + per-activity smoke test. ✓

**Risk surfacing:** R1 dual Title/WorkflowName storage → A0 verifies + A4 dual-write. R2 dual UserTask types → A2 smoke test. R3 missing SystemTask gen → raw BSON fallback. R4 NoUserSource ↔ EmptyUserSource flip → pinned in helpers. R5 Group source/targeting flip → A0 verification. R6 BoundaryEvent split → Phase D switch. R7 ParameterMapping split → per-call-site builder. R8 AllowedModuleRoles version prefix → D2 round-trip test. R9 mutator gen serializer → D7 dedicated task. R10 ELK source-range preservation → A4 snapshot test + B1 verify. R11 Stage 4 boundary → §11. R12 microflow formatter reuse → A2 audits Stage 3.2 helpers. R13 WorkflowParameter shape → A0 verify. ✓

**TDD discipline:** Every task A0–A5, B1, C1–C8, D1–D8, E1–E3 starts with "Step 1: Write failing test" + "Step 2: Confirm RED". A2 explicitly per-activity-type sub-tasks (no "similar to" shortcuts). ✓

**Commit hygiene:** Each commit single-concern. A2 split per activity type. D1 split per builder group. Commit messages use HEREDOC. No `--no-verify`. ✓

**No public-API break without approval:** No `modelsdk.go` or `api/` files touched. `mdl/backend/{workflow,mutation}.go` interface changes are internal — additive `*Gen` first (C1, C2), subtractive after migration (E1). ✓

**Cache discipline:** `listWorkflowsWithContainerGen` cache helper added in A0 BEFORE any consumer migration; every Phase D write commit pairs with `invalidateWorkflowsCache(ctx)` per memory `feedback_executor_cache_pattern`. ✓

---

## §13 Execution Notes (Wave Concurrency)

Lead executes the plan with the following concurrency strategy:

### Strictly serial (lead does directly)
- **A0** (repo + cache + ctx wiring) — single-file blast radius on shared infra. Lead owns.
- **A5, D6** (dispatcher cutovers) — small but need all upstream commits. Lead owns.
- **C1, C2** (additive FullBackend + WorkflowMutator gen methods) — interface change. Lead owns.
- **E1, E2, E3** (cleanup) — strictly serial. Lead owns.

### Concurrent-safe (parallel teammates writing independent NEW files; lead serializes commits)
- **A1, A2 sub-tasks (per activity type), A3, A4** — write into the SAME new file `cmd_workflows_gen.go`. Single teammate, sequential.
- **C3, C4, C5, C7, C8** — independent files; up to 3 parallel teammates, each writing only one assigned file.

### Critical / complex (single teammate + reviewer)
- **A2.UserTaskActivity + A3 (formatUserTaskGen)** — heaviest per-activity formatter. Single teammate. Lead requests `codex review` on the diff before committing.
- **A4 (describeWorkflowToStringGen + ELK source-range preservation)** — visualization byte-parity is critical. Single teammate + manual SVG diff inspection by lead before commit.
- **D2 (execCreateWorkflowGen)** — round-trip with all activity types. Single teammate + `mx check` smoke verification by lead.
- **D7 (mutator gen serializer)** — raw BSON shape parity with `SerializeWorkflowActivity`. Single teammate + manual hex diff inspection.

### Safety rule
- Per memory `feedback_multi_agent_worktree_concurrency`: NEVER run two teammates concurrently with overlapping file targets.
- Per master plan §3.4: NEVER push during execution.

---

## §14 Open Questions for the User Before Execution Starts

1. **gen schema verification for SystemTask / FloatingAnnotation / Group Source-vs-Targeting** — A0.S1 produces verified storage names + accessors. Should A0.S1 produce a tracked memory `project_gen_schema_gaps_workflows.md` with the verified table, or roll into existing `project_gen_schema_gaps.md`? Default = roll into existing.
2. **A2 commit granularity** — plan estimates 12 sub-commits per activity type. Reviewer-friendlier to bundle small ones (Start/End/JumpTo/WaitFor/EndOf*) into one or two commits, leaving UserTask/CallMicroflow/CallWorkflow/ExclusiveSplit/ParallelSplit as their own commits. Confirm preference. Default = group small ones (~4 commits total for activity formatters).
3. **D7 mutator serializer scope** — the gen-aware serializer must produce BSON shapes byte-identical to `mpr.SerializeWorkflowActivity`. Should D7 add a hex-diff test that compares the two outputs on every supported activity type? Default = yes (essential for ALTER WORKFLOW correctness).
4. **B1 visualization test** — the SVG output depends on graphviz/ELK rendering. Should we snapshot the full SVG bytes, the ELK source-range JSON, or just the activity name/position tuples? Default = ELK source-range JSON (most stable; SVG bytes drift on layout-engine version bumps).

These should be resolved BEFORE execution starts.

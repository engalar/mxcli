# MDL Workflow Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add full MDL read support for Mendix workflows — fix feature matrix gaps (SHOW/DESCRIBE already exist but undocumented), add catalog cross-references, source generation, string extraction, and context/navigation integration.

**Architecture:** Workflow read-only support is 80% done (SDK types, parser, reader, SHOW/DESCRIBE, catalog table all exist). This plan fills the remaining 20%: catalog cross-references already implemented but need testing, source generation needs workflow inclusion, string extraction needs workflow coverage, and context/navigation commands need verification. Feature matrix and docs need updating.

**Tech Stack:** Go, ANTLR4, SQLite (catalog), existing mxcli infrastructure

---

## Current State Analysis

### Already Implemented
| Component | File | Status |
|-----------|------|--------|
| SDK types | `sdk/workflows/workflow.go` | Complete (301 lines, 12 activity types) |
| BSON parser | `sdk/mpr/parser_workflow.go` | Complete (550 lines, all activity dispatch) |
| Reader | `sdk/mpr/reader_documents.go` | Complete (`ListWorkflows`, `GetWorkflow`) |
| SHOW WORKFLOWS | `mdl/executor/cmd_workflows.go` | Complete |
| DESCRIBE WORKFLOW | `mdl/executor/cmd_workflows.go` | Complete (MDL-like output) |
| Grammar tokens | `mdl/grammar/MDLLexer.g4` | `WORKFLOW`, `WORKFLOWS` tokens exist |
| Grammar rules | `mdl/grammar/MDLParser.g4` | `SHOW WORKFLOWS`, `DESCRIBE WORKFLOW` rules exist |
| AST types | `mdl/ast/ast_query.go` | `ShowWorkflows`, `DescribeWorkflow` exist |
| Visitor | `mdl/visitor/visitor_query.go` | Integrated |
| Catalog table | `mdl/catalog/tables.go:357-379` | `workflows` table with 21 columns |
| Catalog builder | `mdl/catalog/builder_workflows.go` | Complete (127 lines) |
| Cross-references | `mdl/catalog/builder_references.go:258-291,578-675` | Already implemented! |
| Context detection | `mdl/executor/cmd_context.go:67-94` | `workflow` type registered |
| Help topic | `cmd/mxcli/help_topics/workflow.txt` | Exists |

### Gaps to Fill
| Gap | Description | Effort |
|-----|-------------|--------|
| Source generation | Workflows not included in `buildSource()` FTS table | Low |
| String extraction | Workflow strings not extracted for SEARCH | Low |
| Feature matrix | Shows N for SHOW/DESCRIBE (should be Y) | Trivial |
| Integration tests | No test coverage for workflow commands | Medium |
| SHOW CONTEXT OF | `assembleWorkflowContext` may not exist yet | Low |
| Documentation | Help topic may need updates | Low |

### npm Model SDK Reference (workflows.d.ts)

Key type hierarchy from `mendixmodelsdk` (installed locally):

```
Workflow extends projects.Document
  - parameter: Parameter (entity: IEntity)
  - flow: Flow (activities: WorkflowActivity[])
  - workflowName: StringTemplate
  - workflowDescription: StringTemplate
  - dueDate: string (expression)
  - adminPage: PageReference
  - onWorkflowEvent: WorkflowEventHandler[]
  - eventSubProcesses: EventSubProcess[] (v11.8+)
  - annotation: Annotation
  - workflowMetaData: WorkflowMetaData (v11.1+)

WorkflowActivity (abstract)
  ├── StartWorkflowActivity (v11.1+)
  ├── EndWorkflowActivity
  ├── ConditionOutcomeActivity (abstract, has outcomes: ConditionOutcome[])
  │   ├── CallMicroflowTask (microflow, parameterMappings, boundaryEvents)
  │   └── ExclusiveSplitActivity (expression)
  ├── UserTaskActivity (abstract, v10.12+)
  │   ├── SingleUserTaskActivity
  │   └── MultiUserTaskActivity (targetUserInput, completionCriteria)
  │   Common: taskPage, taskName, taskDescription, dueDate, userSource/userTargeting, outcomes, boundaryEvents, onCreatedEvent
  ├── CallWorkflowActivity (workflow, parameterMappings, boundaryEvents, executeAsync)
  ├── ParallelSplitActivity (outcomes: ParallelSplitOutcome[])
  ├── MergeActivity
  ├── JumpToActivity (target)
  ├── WaitForTimerActivity (delayExpression)
  └── WaitForNotificationActivity (boundaryEvents)

Outcome (abstract, flow: Flow)
  ├── ConditionOutcome (abstract)
  │   ├── BooleanConditionOutcome (value: boolean)
  │   └── EnumerationValueConditionOutcome (value: string)
  ├── UserTaskOutcome (name, caption)
  └── ParallelSplitOutcome

UserSource (abstract, v11.2 renamed to UserTargeting)
  ├── NoUserSource / NoUserTargeting
  ├── MicroflowBasedUserSource / MicroflowUserTargeting
  ├── XPathBasedUserSource / XPathUserTargeting
  └── (Group variants: MicroflowGroupTargeting, XPathGroupTargeting)

BoundaryEvent (v10.14+)
  ├── InterruptingTimerBoundaryEvent (v10.20+)
  └── NonInterruptingTimerBoundaryEvent (v10.20+)
```

---

## Task 1: Verify Source Generation Includes Workflows

**Files:**
- Check: `mdl/catalog/builder_source.go`
- Modify (if needed): `mdl/catalog/builder_source.go`

**Step 1: Read builder_source.go and check if workflows are already included**

Look for `cachedWorkflows()` or `WORKFLOW` in the source builder. Based on the explore agent's findings, workflows are already collected in items via `cachedWorkflows()` and dispatched through `describeFunc`.

Run: `grep -n "workflow\|WORKFLOW" mdl/catalog/builder_source.go`

If workflows ARE already included, mark this task as done.

**Step 2: If not included, add workflow source generation**

Follow the existing pattern for microflows:
```go
// In buildSource() items collection:
wfList, err := b.cachedWorkflows()
for _, wf := range wfList {
    moduleID := b.hierarchy.findModuleID(wf.ContainerID)
    moduleName := b.hierarchy.getModuleName(moduleID)
    items = append(items, sourceItem{
        "WORKFLOW",
        moduleName + "." + wf.Name,
        moduleName,
    })
}
```

**Step 3: Verify describeFunc handles "WORKFLOW" type**

Check that `describeFunc` in `builder_source.go` or wherever it's defined dispatches `"WORKFLOW"` to `describeWorkflowToString`.

**Step 4: Build and test**

Run: `make build && make test`

---

## Task 2: Verify String Extraction Includes Workflows

**Files:**
- Check: `mdl/catalog/builder_strings.go`
- Modify (if needed): `mdl/catalog/builder_strings.go`

**Step 1: Read builder_strings.go and check if workflows are included**

Run: `grep -n "workflow\|WORKFLOW" mdl/catalog/builder_strings.go`

**Step 2: If not included, add workflow string extraction**

Extract strings from workflow activity captions, task names, descriptions:

```go
// In buildStrings():
wfList, err := b.cachedWorkflows()
for _, wf := range wfList {
    moduleID := b.hierarchy.findModuleID(wf.ContainerID)
    moduleName := b.hierarchy.getModuleName(moduleID)
    qn := moduleName + "." + wf.Name

    // Workflow-level strings
    if wf.WorkflowName != "" {
        insert(qn, "WORKFLOW", wf.WorkflowName, "WorkflowName", moduleName)
    }
    if wf.WorkflowDescription != "" {
        insert(qn, "WORKFLOW", wf.WorkflowDescription, "WorkflowDescription", moduleName)
    }
    if wf.Documentation != "" {
        insert(qn, "WORKFLOW", wf.Documentation, "Documentation", moduleName)
    }

    // Extract activity strings recursively
    if wf.Flow != nil {
        extractWorkflowFlowStrings(wf.Flow, qn, "WORKFLOW", moduleName, insert)
    }
}
```

**Step 3: Build and test**

Run: `make build && make test`

---

## Task 3: Verify Context/Navigation Commands

**Files:**
- Check: `mdl/executor/cmd_context.go`

**Step 1: Read cmd_context.go and find assembleWorkflowContext**

Run: `grep -n "assembleWorkflowContext\|workflow" mdl/executor/cmd_context.go`

**Step 2: If assembleWorkflowContext exists, verify it works**

It should show:
- Workflow parameter entity
- Called microflows
- Called sub-workflows
- Referenced pages
- User targeting microflows

**Step 3: If missing, implement assembleWorkflowContext**

```go
func (e *Executor) assembleWorkflowContext(output *strings.Builder, name ast.QualifiedName, depth int) {
    // Get workflow describe output
    text, _, err := e.describeWorkflowToString(name)
    if err == nil {
        output.WriteString(text)
        output.WriteString("\n")
    }
}
```

**Step 4: Build and test**

Run: `make build && make test`

---

## Task 4: Update Feature Matrix

**Files:**
- Modify: `docs/01-project/MDL_FEATURE_MATRIX.md`

**Step 1: Update the Workflows row**

Change from:
```
| **Workflows** | N | N | N | N | N | N | N | N | N | N | N | N | N | N | N | Y | N | Workflow definitions |
```

To (update SHOW and DESCRIBE columns to Y):
Check which column positions are SHOW and DESCRIBE by reading the header row, then update accordingly.

**Step 2: Commit**

```bash
git add docs/01-project/MDL_FEATURE_MATRIX.md
git commit -m "docs: update feature matrix with workflow SHOW/DESCRIBE support"
```

---

## Task 5: Integration Test with Real Projects

**Files:**
- Test projects in `mx-test-projects/`

**Step 1: Build mxcli**

Run: `make build`

**Step 2: Test SHOW WORKFLOWS**

```bash
# Find projects with workflows
find mx-test-projects -name "*.mpr" -exec sh -c '
  echo "=== {} ==="
  ./bin/mxcli -p "{}" -c "SHOW WORKFLOWS" 2>/dev/null
' \;
```

**Step 3: Test DESCRIBE WORKFLOW**

For each workflow found in Step 2:
```bash
./bin/mxcli -p <project.mpr> -c "DESCRIBE WORKFLOW <Module.WorkflowName>"
```

**Step 4: Test catalog queries**

```bash
./bin/mxcli -p <project.mpr> -c "REFRESH CATALOG FULL FORCE; SELECT * FROM CATALOG.WORKFLOWS"
```

**Step 5: Test cross-references**

```bash
./bin/mxcli -p <project.mpr> -c "REFRESH CATALOG FULL FORCE; SHOW CALLERS OF <microflow-called-by-workflow>"
./bin/mxcli -p <project.mpr> -c "SHOW CALLEES OF <workflow-name>"
```

**Step 6: Test SEARCH**

```bash
./bin/mxcli -p <project.mpr> -c "REFRESH CATALOG FULL FORCE; SEARCH 'workflow-related-keyword'"
```

**Step 7: Test SHOW CONTEXT OF**

```bash
./bin/mxcli -p <project.mpr> -c "REFRESH CATALOG FULL FORCE; SHOW CONTEXT OF <workflow-name>"
```

---

## Task 6: Fix Any Issues Found in Testing

Address any bugs or gaps discovered during integration testing.

**Step 1: For each issue, write a failing test or reproduce**
**Step 2: Fix the issue**
**Step 3: Verify the fix**
**Step 4: Commit**

```bash
git add <changed-files>
git commit -m "fix: <description of fix>"
```

---

## Task 7: Final Commit and Summary

**Step 1: Run full test suite**

```bash
make build && make test
```

**Step 2: Commit all remaining changes**

```bash
git add <files>
git commit -m "feat: complete workflow MDL read support (source, strings, context, docs)"
```

---

## Verification Checklist

- [ ] `SHOW WORKFLOWS` lists all workflows with correct counts
- [ ] `SHOW WORKFLOWS IN <Module>` filters by module
- [ ] `DESCRIBE WORKFLOW <Module.Name>` shows full MDL-like output
- [ ] `CATALOG.WORKFLOWS` table populated with correct data
- [ ] Cross-references: `SHOW CALLERS OF <microflow>` shows workflows that call it
- [ ] Cross-references: `SHOW CALLEES OF <workflow>` shows microflows/pages it references
- [ ] `SEARCH '<keyword>'` finds workflow content
- [ ] `SHOW CONTEXT OF <workflow>` shows workflow details
- [ ] Feature matrix updated
- [ ] `make build && make test` passes

# Workflow Improvements: ALTER WORKFLOW + Cross-References

**Date:** 2026-04-03
**Status:** Proposal
**Author:** @anthropics/claude-code

## Problem

Workflow support in mxcli has full CREATE/DESCRIBE/DROP/SHOW coverage with 13 activity types and BSON round-trip fidelity. Two significant gaps remain:

1. **No ALTER WORKFLOW** — any workflow change requires a full `CREATE OR MODIFY` rebuild. For small edits (change a task page, add an outcome, insert an activity), this is disproportionate effort and error-prone.

2. **No cross-reference tracking** — the catalog `refs` table does not track workflow references. `SHOW CALLERS OF Module.SomeMicroflow` will not show workflows that call it. Impact analysis is incomplete.

## Design

### Part 1: ALTER WORKFLOW

#### 1.1 Property Operations

Modify workflow-level and activity-level metadata without touching the flow graph:

```sql
-- Workflow-level properties
ALTER WORKFLOW Module.MyWorkflow
  SET DISPLAY 'New Display Name';

ALTER WORKFLOW Module.MyWorkflow
  SET DESCRIPTION 'Updated description';

ALTER WORKFLOW Module.MyWorkflow
  SET EXPORT LEVEL API;

ALTER WORKFLOW Module.MyWorkflow
  SET DUE DATE '[%CurrentDateTime%] + 7 * 24 * 60 * 60 * 1000';

ALTER WORKFLOW Module.MyWorkflow
  SET OVERVIEW PAGE Module.NewOverviewPage;

ALTER WORKFLOW Module.MyWorkflow
  SET PARAMETER $WorkflowContext: Module.NewEntity;

-- Activity-level properties
ALTER WORKFLOW Module.MyWorkflow
  SET ACTIVITY userTask1 PAGE Module.NewTaskPage;

ALTER WORKFLOW Module.MyWorkflow
  SET ACTIVITY userTask1 DESCRIPTION 'Updated task description';

ALTER WORKFLOW Module.MyWorkflow
  SET ACTIVITY userTask1 TARGETING MICROFLOW Module.NewTargeting;

ALTER WORKFLOW Module.MyWorkflow
  SET ACTIVITY userTask1 TARGETING XPATH '[Status = "Active"]';
```

**Implementation**: Uses `readPatchWrite` pattern from ALTER PAGE. Reads raw BSON as `bson.D`, modifies specific fields, writes back.

#### 1.2 Graph Operations

The workflow flow graph is stored as two flat arrays:
- `Flow.Activities`: all activity objects
- `Flow.SequenceFlows`: directed edges between activities

Each graph operation maps to a well-defined transformation on these arrays.

##### INSERT AFTER (linear position)

**Precondition**: Target activity has exactly one outgoing non-error edge.

```sql
ALTER WORKFLOW Module.MyWorkflow
  INSERT AFTER activityName
    CALL MICROFLOW validate Module.Validate;
```

**BSON transformation**:
```
Before: A ──edge1──→ B
Step 1: Create activity C, append to Activities array
Step 2: Set edge1.Dest = C
Step 3: Create edge2: {Origin: C, Dest: B}
After:  A ──edge1──→ C ──edge2──→ B
```

**Error conditions**:
- Target has multiple outgoing edges → reject with "activity is a split point, use INSERT OUTCOME/PATH/BRANCH"
- Target is End → reject with "cannot insert after end activity"
- Target not found → error

##### INSERT OUTCOME ON UserTask

**Structure**: UserTask has an `Outcomes` array. Each outcome maps to outgoing edges via `ConditionValue` matching the outcome `$ID`.

```sql
ALTER WORKFLOW Module.MyWorkflow
  INSERT OUTCOME 'NeedMoreInfo' ON userTask1 {
    CALL MICROFLOW requestInfo Module.RequestInfo
  };
```

**BSON transformation**:
```
Step 1: Create new Outcome {Name: "NeedMoreInfo", $ID: id3}, append to Outcomes array
Step 2: Create activities from the block, chained in sequence
Step 3: Find merge point (see algorithm below)
Step 4: Add edge {Origin: userTask1, Dest: firstNewActivity, ConditionValue: id3}
Step 5: Add edge {Origin: lastNewActivity, Dest: mergePoint}
```

**Error conditions**:
- Duplicate outcome name → reject
- Empty block `{}` → create direct edge to merge point
- Merge point not found → error (malformed graph)

##### INSERT PATH ON ParallelSplit

Same as INSERT OUTCOME but without ConditionValue:

```sql
ALTER WORKFLOW Module.MyWorkflow
  INSERT PATH ON parallelSplit1 {
    USER TASK review3 'Third Review'
      PAGE Module.ReviewPage
  };
```

**BSON transformation**: Same as INSERT OUTCOME, but edges have no ConditionValue. The merge target is a ParallelMerge node.

##### INSERT BRANCH ON Decision

```sql
ALTER WORKFLOW Module.MyWorkflow
  INSERT BRANCH ON decision1 CONDITION '$ctx/Status = "Special"' {
    CALL MICROFLOW handleSpecial Module.HandleSpecial
  };
```

**BSON transformation**: Creates a new ConditionOutcome on the Decision, then adds edges and activities as in INSERT OUTCOME.

##### INSERT BOUNDARY EVENT

```sql
ALTER WORKFLOW Module.MyWorkflow
  INSERT BOUNDARY EVENT INTERRUPTING TIMER '86400000' ON userTask1 {
    CALL MICROFLOW escalate Module.Escalate
  };
```

##### DROP ACTIVITY (linear node)

**Precondition**: Target has exactly one incoming edge and one outgoing edge.

```sql
ALTER WORKFLOW Module.MyWorkflow
  DROP ACTIVITY callMf1;
```

**BSON transformation**:
```
Before: A ──e1──→ C ──e2──→ B
Step 1: Set e1.Dest = B
Step 2: Delete e2 from SequenceFlows array
Step 3: Delete C from Activities array
After:  A ──e1──→ B
```

**Error conditions**:
- Multiple in/out edges → reject with "activity is a split/merge point, use DROP OUTCOME/PATH instead"

##### DROP OUTCOME / PATH / BRANCH

```sql
ALTER WORKFLOW Module.MyWorkflow
  DROP OUTCOME 'NeedMoreInfo' ON userTask1;

ALTER WORKFLOW Module.MyWorkflow
  DROP PATH 2 ON parallelSplit1;  -- by index (0-based)

ALTER WORKFLOW Module.MyWorkflow
  DROP BRANCH 'Special' ON decision1;
```

**BSON transformation**:
```
Step 1: Find the outgoing edge for this outcome/path/branch
Step 2: Collect all activities on this path (BFS from edge.Dest to merge point)
Step 3: Delete collected activities and their edges from arrays
Step 4: Delete the outcome/branch entry from the split activity
```

**Path activity collection algorithm**:
```
collectPathActivities(startId, mergeId):
  queue = [startId]
  result = []
  while queue not empty:
    id = queue.pop()
    if id == mergeId: continue  // don't delete merge point
    result.append(id)
    for edge in outgoingEdges(id):
      queue.append(edge.Dest)
  return result
```

#### 1.3 Merge Point Discovery Algorithm

Several operations require finding the merge/convergence point of a split node. All branches from a split eventually converge at a single merge point (Mendix enforces structured workflows).

```
findMergePoint(splitActivityId, sequenceFlows):
  // Get all outgoing edges from the split
  outEdges = filter(sequenceFlows, edge.Origin == splitActivityId)

  // Follow each branch, collecting visited nodes per branch
  branchPaths = []
  for each edge in outEdges:
    path = set()
    current = edge.Dest
    while current has exactly one outgoing edge:
      path.add(current)
      current = next(current)  // follow the single outgoing edge
    path.add(current)  // add the multi-input node (potential merge)
    branchPaths.append(path)

  // Find first common node across all branches
  if len(branchPaths) == 0: error
  common = branchPaths[0]
  for path in branchPaths[1:]:
    common = common.intersect(path)

  // Return the first (closest to split) common node
  return closest(common, splitActivityId)
```

**Note**: This simplified algorithm works for structured workflows (which Mendix enforces). Arbitrary graph structures would require proper post-dominator analysis.

#### 1.4 Operations NOT Supported

These require `CREATE OR MODIFY` to rebuild the workflow:
- Moving activities between branches
- Converting a linear activity into a Decision/ParallelSplit
- Merging or splitting branches
- Reordering activities within a branch

#### 1.5 Grammar

```antlr
alterWorkflowStatement
    : ALTER WORKFLOW qualifiedName alterWorkflowAction+
    ;

alterWorkflowAction
    : SET workflowProperty                         // workflow-level property
    | SET ACTIVITY IDENTIFIER activityProperty      // activity-level property
    | INSERT AFTER IDENTIFIER workflowActivity      // linear insert
    | INSERT OUTCOME STRING_LITERAL ON IDENTIFIER workflowBlock  // user task outcome
    | INSERT PATH ON IDENTIFIER workflowBlock       // parallel path
    | INSERT BRANCH ON IDENTIFIER CONDITION STRING_LITERAL workflowBlock  // decision branch
    | INSERT BOUNDARY EVENT boundaryEventSpec ON IDENTIFIER workflowBlock
    | DROP ACTIVITY IDENTIFIER                      // linear delete
    | DROP OUTCOME STRING_LITERAL ON IDENTIFIER     // outcome delete
    | DROP PATH INT ON IDENTIFIER                   // parallel path delete (by index)
    | DROP BRANCH STRING_LITERAL ON IDENTIFIER      // decision branch delete
    | DROP BOUNDARY EVENT ON IDENTIFIER             // boundary event delete
    ;

workflowProperty
    : DISPLAY STRING_LITERAL
    | DESCRIPTION STRING_LITERAL
    | EXPORT LEVEL IDENTIFIER
    | DUE DATE STRING_LITERAL
    | OVERVIEW PAGE qualifiedName
    | PARAMETER VARIABLE COLON qualifiedName
    ;

activityProperty
    : PAGE qualifiedName
    | DESCRIPTION STRING_LITERAL
    | TARGETING MICROFLOW qualifiedName
    | TARGETING XPATH STRING_LITERAL
    | DUE DATE STRING_LITERAL
    ;

workflowBlock
    : '{' workflowActivity* '}'
    ;
```

### Part 2: Workflow Cross-References

#### 2.1 Reference Types

Extend `catalog/builder.go` `buildRefs()` to track workflow references in the `refs` table:

| SourceType | TargetType | RefKind | Scenario |
|------------|------------|---------|----------|
| workflow | microflow | `call` | CALL MICROFLOW activity |
| workflow | workflow | `call` | CALL WORKFLOW activity |
| workflow | entity | `uses` | Parameter entity |
| workflow | page | `uses` | UserTask PAGE, OverviewPage |
| workflow | microflow | `uses` | UserTask TARGETING MICROFLOW |

**Note**: The reverse direction (microflow → workflow via CallWorkflowAction) should already be tracked if microflow refs are complete. Verify and add if missing.

#### 2.2 Implementation

In `catalog/builder_workflows.go`, extend the existing `buildWorkflows()` function to also emit `refs` rows:

```go
func (b *Builder) buildWorkflowRefs(wf *workflows.Workflow, moduleName string) {
    qn := moduleName + "." + wf.Name

    // Parameter entity reference
    if wf.Parameter != nil && wf.Parameter.Entity != "" {
        b.insertRef(qn, wf.Parameter.Entity, "uses", "workflow", "entity")
    }

    // Overview page reference
    if wf.OverviewPage != "" {
        b.insertRef(qn, wf.OverviewPage, "uses", "workflow", "page")
    }

    // Recursively scan activities
    b.buildActivityRefs(qn, wf.Flow.Activities)
}

func (b *Builder) buildActivityRefs(workflowQN string, activities []workflows.Activity) {
    for _, a := range activities {
        switch act := a.(type) {
        case *workflows.CallMicroflowTask:
            b.insertRef(workflowQN, act.MicroflowQN, "call", "workflow", "microflow")
        case *workflows.CallWorkflowTask:
            b.insertRef(workflowQN, act.WorkflowQN, "call", "workflow", "workflow")
        case *workflows.UserTask:
            if act.Page != "" {
                b.insertRef(workflowQN, act.Page, "uses", "workflow", "page")
            }
            if act.TargetingMicroflow != "" {
                b.insertRef(workflowQN, act.TargetingMicroflow, "uses", "workflow", "microflow")
            }
            // Recurse into outcome sub-flows
            for _, outcome := range act.Outcomes {
                b.buildActivityRefs(workflowQN, outcome.Flow.Activities)
            }
        }
        // Recurse into boundary event sub-flows
        for _, be := range a.BoundaryEvents() {
            b.buildActivityRefs(workflowQN, be.Flow.Activities)
        }
    }
}
```

#### 2.3 Effect on Existing Commands

No new commands needed. Existing commands automatically pick up workflow refs:

```sql
-- Shows workflows that call this microflow
SHOW CALLERS OF Module.SomeMicroflow;

-- Shows all microflows/workflows/pages referenced by this workflow
SHOW CALLEES OF Module.MyWorkflow;

-- Shows workflows that reference this entity
SHOW REFERENCES TO Module.SomeEntity;

-- Includes workflows in impact analysis
SHOW IMPACT OF Module.SomeEntity;
```

## Implementation Phases

| Phase | Scope | Complexity | Dependencies |
|-------|-------|------------|--------------|
| **P1** | Workflow Cross-References in catalog builder | Low | None — independent, quick win |
| **P2** | ALTER WORKFLOW SET (properties only) | Low | Grammar + executor |
| **P3** | ALTER WORKFLOW SET ACTIVITY (activity properties) | Low | P2 |
| **P4** | INSERT AFTER / DROP ACTIVITY (linear graph ops) | Medium | P2 + merge point algorithm |
| **P5** | INSERT/DROP OUTCOME, PATH, BRANCH (split graph ops) | High | P4 + merge point discovery |
| **P6** | INSERT/DROP BOUNDARY EVENT | Medium | P4 |

**Recommended order**: P1 first (quick, independent), then P2→P3→P4→P5→P6.

## Testing Strategy

- **Cross-References**: Add workflow entries to existing `buildRefs` test suite. Verify `SHOW CALLERS/CALLEES` output includes workflows after `REFRESH CATALOG FULL`.
- **ALTER SET**: Roundtrip test — `DESCRIBE` → `ALTER SET` → `DESCRIBE` → compare.
- **Graph ops**: Create workflow via `CREATE WORKFLOW`, apply ALTER operations, `mx check` to validate, `DESCRIBE` to verify structure.
- **Edge cases**: INSERT on split nodes (must reject), DROP on merge nodes (must reject), duplicate outcome names, empty blocks.

## Compatibility

- **Backward compatible**: No existing syntax changes.
- **Forward compatible**: ALTER WORKFLOW statements fail gracefully on older mxcli versions with a parse error.
- **Studio Pro**: All BSON output must pass `mx check` validation.

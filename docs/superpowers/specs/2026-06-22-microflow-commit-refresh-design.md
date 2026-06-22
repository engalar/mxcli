# Microflow Activity Commit/Refresh Design

> **Status:** Draft
> **Date:** 2026-06-22
> **Research basis:** `docs/research/2026-06-22-modelsdk-bson-investigation-methodology.md`
> **BSON data:** `docs/research/2026-06-22-microflow-commit-refresh-bson.md`

## Goal

Add MDL support for the three built-in properties (Commit, WithEvents/WithoutEvents, RefreshInClient) across all five microflow activities: CREATE, CHANGE, COMMIT, DELETE, ROLLBACK. All changes must round-trip (create → describe → recreate matches original).

## Design

### Syntax

```sql
-- CREATE
$Obj = create Module.Entity (Name = 'x');
$Obj = create Module.Entity (Name = 'x') with commit;
$Obj = create Module.Entity (Name = 'x') with commit refresh;
$Obj = create Module.Entity (Name = 'x') with commit without events;
$Obj = create Module.Entity (Name = 'x') with commit without events refresh;

-- CHANGE
change $Obj (Name = 'new');
change $Obj (Name = 'new') with commit;
change $Obj (Name = 'new') with commit refresh;
change $Obj (Name = 'new') with commit without events;
change $Obj (Name = 'new') with commit without events refresh;

-- COMMIT (unchanged)
commit $Obj;
commit $Obj with events;
commit $Obj refresh;
commit $Obj with events refresh;

-- DELETE (new: refresh)
delete $Obj;
delete $Obj refresh;

-- ROLLBACK (unchanged, already has refresh)
rollback $Obj;
rollback $Obj refresh;
```

### Grammar Changes

File: `mdl/grammar/domains/MDLMicroflow.g4`

**createObjectStatement** (add commit clause):
```antlr
createObjectStatement
    : (VARIABLE EQUALS)? CREATE nonListDataType (LPAREN memberAssignmentList? RPAREN)?
      (WITH COMMIT (WITHOUT EVENTS)?)? REFRESH? onErrorClause?
    ;
```

**changeObjectStatement** (add commit clause):
```antlr
changeObjectStatement
    : CHANGE VARIABLE (LPAREN memberAssignmentList? RPAREN)?
      (WITH COMMIT (WITHOUT EVENTS)?)? REFRESH? onErrorClause?
    ;
```

**commitStatement** (no change):
```antlr
commitStatement
    : COMMIT VARIABLE (WITH EVENTS)? REFRESH? onErrorClause?
    ;
```

**deleteObjectStatement** (add refresh):
```antlr
deleteObjectStatement
    : DELETE VARIABLE REFRESH? onErrorClause?
    ;
```

**rollbackStatement** (no change):
```antlr
rollbackStatement
    : ROLLBACK VARIABLE REFRESH? onErrorClause?
    ;
```

**New tokens** (add to lexer): no new tokens needed — all keywords are already defined.

### AST Changes

File: `mdl/ast/ast_microflow.go`

**CreateObjectStmt — add fields:**
```go
type CreateObjectStmt struct {
    Variable        string               // $Var =
    EntityType      NonListDataType
    Changes         []MemberChange
    WithCommit      bool                 // NEW: true = "with commit"
    WithoutEvents   bool                 // NEW: true = "without events"
    RefreshInClient bool                 // NEW: true = "refresh"
    ErrorHandling   *ErrorHandlingClause
    Annotations     *ActivityAnnotations
}
```

**ChangeObjectStmt — add fields:**
```go
type ChangeObjectStmt struct {
    Variable        string               // $Var
    Changes         []MemberChange
    WithCommit      bool                 // NEW: true = "with commit"
    WithoutEvents   bool                 // NEW: true = "without events"
    RefreshInClient bool                 // existing, but currently parsed from REFRESH? token
    ErrorHandling   *ErrorHandlingClause
    Annotations     *ActivityAnnotations
}
```

**MfCommitStmt — no change needed** (already has `WithEvents`, `RefreshInClient`).

**MfDeleteStmt — add field:**
```go
type MfDeleteStmt struct {
    Variable        string
    RefreshInClient bool                 // NEW: true = "refresh"
    ErrorHandling   *ErrorHandlingClause
    Annotations     *ActivityAnnotations
}
```

**RollbackStmt — no change needed** (already has `RefreshInClient`).

### Visitor Changes

File: `mdl/visitor/visitor_microflow_statements.go`

**`buildCreateObjectStatement` — add parsing:**
```go
stmt.WithCommit = createCtx.COMMIT() != nil
stmt.WithoutEvents = createCtx.EVENTS() == nil && createCtx.COMMIT() != nil  // "with commit" → WithCommit=true; "with commit without events" → WithCommit=true, WithoutEvents=true
```
Wait — need to handle the ambiguity. The grammar has `WITH COMMIT (WITHOUT EVENTS)?`. So:
- `WITH COMMIT` present → `WithCommit = true`
- `WITHOUT EVENTS` present → `WithoutEvents = true`
- No `WITH` clause → both false

```go
if createCtx.COMMIT() != nil {
    stmt.WithCommit = true
    if createCtx.EVENTS() != nil { // WITHOUT EVENTS
        stmt.WithoutEvents = true
    }
}
stmt.RefreshInClient = createCtx.REFRESH() != nil
```

**`buildChangeObjectStatement` — add parsing:**
```go
if changeCtx.COMMIT() != nil {
    stmt.WithCommit = true
    if changeCtx.EVENTS() != nil { // WITHOUT EVENTS
        stmt.WithoutEvents = true
    }
}
stmt.RefreshInClient = changeCtx.REFRESH() != nil
```

**`buildDeleteObjectStatement` — add refresh parsing:**
```go
stmt.RefreshInClient = deleteCtx.REFRESH() != nil
```

### Executor (BSON Write) Changes

**File: `mdl/executor/flowbuilder_actions_change_v2.go`**

**`addCreateObjectActionGen`:**
```go
func (fb *flowBuilderGen) addCreateObjectActionGen(s *ast.CreateObjectStmt) element.ID {
    action := genMf.NewCreateObjectAction()
    assignFreshID(action)
    action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
    action.SetOutputVariableName(s.Variable)
    
    // Commit + RefreshInClient mapping
    commitVal := "No"
    if s.WithCommit {
        if s.WithoutEvents {
            commitVal = "YesWithoutEvents"
        } else {
            commitVal = "Yes"
        }
    }
    action.SetCommit(commitVal)
    action.SetRefreshInClient(s.RefreshInClient)
    
    // Entity
    entityQN := ""
    if s.EntityType.Module != "" && s.EntityType.Name != "" {
        entityQN = s.EntityType.Module + "." + s.EntityType.Name
        action.SetEntityQualifiedName(entityQN)
    }
    // ... rest unchanged
}
```

**`addChangeObjectActionGen`:** Same logic for Commit/RefreshInClient. Replace the hardcoded `action.SetCommit("No")` with the same commit value mapping.

**File: `mdl/executor/flowbuilder_actions_v2.go`**

**`addDeleteActionGen`:** Add `action.SetRefreshInClient(s.RefreshInClient)` after setting DeleteVariableName.

**`addCommitActionGen`:** No change (already sets `WithEvents` and `RefreshInClient`).

**`addRollbackActionGen`:** No change (already sets `RefreshInClient`).

### Formatter (BSON Read → MDL) Changes

**File: `mdl/executor/microflows_format_action_v2.go`**

**`formatCreateObjectActionGen`:** Append commit clause:
```go
func formatCreateObjectActionGen(a *genMf.CreateObjectAction) string {
    // ... existing format logic ...
    commitClause := ""
    switch a.Commit() {
    case "Yes":
        commitClause = " with commit"
    case "YesWithoutEvents":
        commitClause = " with commit without events"
    }
    refreshClause := ""
    if a.RefreshInClient() {
        refreshClause = " refresh"
    }
    return fmt.Sprintf("$%s = create %s (%s)%s%s;", outputVar, entityName, members, commitClause, refreshClause)
}
```

**`formatChangeObjectActionGen`:** Same pattern — append commit clause and refresh.

**`formatCommitActionGen`:** No change needed (already handles WithEvents + RefreshInClient).

**`formatDeleteActionGen`:** Add refresh:
```go
func formatDeleteActionGen(a *genMf.DeleteAction) string {
    suffix := ""
    if a.RefreshInClient() {
        suffix = " refresh"
    }
    return fmt.Sprintf("delete $%s%s;", varName, suffix)
}
```

**`formatRollbackActionGen`:** No change needed (already handles refresh).

### BSON Field Mapping Reference

| MDL clause | CreateChangeAction | ChangeAction | CommitAction | DeleteAction | RollbackAction |
|---|---|---|---|---|---|
| *(default)* | Commit="No" | Commit="No" | WithEvents=false | Refresh=n/a | Refresh=n/a |
| `with commit` | Commit="Yes" | Commit="Yes" | — | — | — |
| `without events` | Commit="YesWithoutEvents" | Commit="YesWithoutEvents" | WithEvents=false | — | — |
| `with events` | — | — | WithEvents=true | — | — |
| `refresh` | RefreshInClient=true | RefreshInClient=true | RefreshInClient=true | RefreshInClient=true | RefreshInClient=true |

### Round-Trip Test Cases

File: `mdl/executor/roundtrip_microflow_test.go`

Each test creates a microflow via MDL, describes it, and asserts the describe output matches the input:

```go
func TestRoundtripMicroflow_CreateWithCommit(t *testing.T) {
    // Input:  $Obj = create Module.Entity (Name = 'x') with commit refresh;
    // Expect: $Obj = create Module.Entity (Name = 'x') with commit refresh;
}

func TestRoundtripMicroflow_CreateWithCommitWithoutEvents(t *testing.T) {
    // Input:  $Obj = create Module.Entity (Name = 'x') with commit without events;
    // Expect: $Obj = create Module.Entity (Name = 'x') with commit without events;
}

func TestRoundtripMicroflow_ChangeWithCommitRefresh(t *testing.T) {
    // Input:  change $Obj (Name = 'new') with commit refresh;
    // Expect: change $Obj (Name = 'new') with commit refresh;
}

func TestRoundtripMicroflow_DeleteWithRefresh(t *testing.T) {
    // Input:  delete $Obj refresh;
    // Expect: delete $Obj refresh;
}
```

### Files to Modify

| # | File | Change |
|---|------|--------|
| 1 | `mdl/grammar/domains/MDLMicroflow.g4` | 4 rule modifications |
| 2 | `mdl/ast/ast_microflow.go` | AST field additions |
| 3 | `mdl/visitor/visitor_microflow_statements.go` | 3 builder modifications |
| 4 | `mdl/executor/flowbuilder_actions_change_v2.go` | Create/Change executor |
| 5 | `mdl/executor/flowbuilder_actions_v2.go` | Delete executor |
| 6 | `mdl/executor/microflows_format_action_v2.go` | 3 formatter modifications |
| 7 | `mdl/executor/microflows_format_action_v2_test.go` | Formatter tests |
| 8 | `mdl/executor/flowbuilder_actions_change_v2_test.go` | Builder tests |
| 9 | `mdl/executor/roundtrip_microflow_test.go` | Roundtrip tests |

### Out of Scope

- Nanoflows have different activity types and restrictions; commit/refresh behavior is different. Not covered here.
- Expresssion-level refresh validation (e.g., "commit on non-persistent entity must have refresh"). Not required per user feedback.

# Microflow Activity Commit/Refresh Design

> **Status:** Draft
> **Date:** 2026-06-22
> **Research basis:** `docs/research/2026-06-22-modelsdk-bson-investigation-methodology.md`
> **BSON data:** `docs/research/2026-06-22-microflow-commit-refresh-bson.md`

## Goal

Add MDL support for the three built-in properties (Commit, WithEvents/WithoutEvents, RefreshInClient) across all five microflow activities: CREATE, CHANGE, COMMIT, DELETE, ROLLBACK. All changes must round-trip (create → describe → recreate matches original).

## Design

### Syntax

**Core rule:** `refresh` must always follow a `with` clause — `with commit`, `with commit without events`, `with events`. Standalone `refresh` is only valid for DELETE and ROLLBACK (which have no commit/event sub-clause).

Describe output must be consistent across CREATE/CHANGE/COMMIT — same clause structure.

```sql
-- CREATE
$Obj = create Module.Entity (Name = 'x');                          -- Commit=No
$Obj = create Module.Entity (Name = 'x') with commit;              -- Commit=Yes
$Obj = create Module.Entity (Name = 'x') with commit refresh;      -- Commit=Yes, Refresh=true
$Obj = create Module.Entity (Name = 'x') with commit without events; -- YesWithoutEvents
$Obj = create Module.Entity (Name = 'x') with commit without events refresh;

-- CHANGE
change $Obj (Name = 'new');                                         -- Commit=No
change $Obj (Name = 'new') with commit;
change $Obj (Name = 'new') with commit refresh;
change $Obj (Name = 'new') with commit without events;
change $Obj (Name = 'new') with commit without events refresh;

-- COMMIT (refresh requires with events — grammar change ⚠️)
commit $Obj;                                                       -- WithEvents=false
commit $Obj with events;                                           -- WithEvents=true
commit $Obj with events refresh;                                   -- WithEvents=true, Refresh=true

-- DELETE
delete $Obj;
delete $Obj refresh;

-- ROLLBACK
rollback $Obj;
rollback $Obj refresh;
```

> ⚠️ **COMMIT grammar change:** `(WITH EVENTS)? REFRESH?` → `(WITH EVENTS REFRESH?)?`. `commit $V refresh;` is no longer valid — use `commit $V with events refresh;`. This matches the CREATE/CHANGE pattern where `refresh` requires `with`.

### Round-Trip Table (BSON ↔ MDL)

Every line in the table below must round-trip: MDL → BSON → describe → identical MDL.

**CREATE:**
| Commit | RefreshInClient | MDL output |
|--------|----------------|------------|
| `"No"` | `false` | `$X = create Module.Entity (attr);` |
| `"Yes"` | `false` | `$X = create Module.Entity (attr) with commit;` |
| `"Yes"` | `true` | `$X = create Module.Entity (attr) with commit refresh;` |
| `"YesWithoutEvents"` | `false` | `$X = create Module.Entity (attr) with commit without events;` |
| `"YesWithoutEvents"` | `true` | `$X = create Module.Entity (attr) with commit without events refresh;` |

> Note: `Commit="No"` + `RefreshInClient=true` (17 instances in real data) cannot be expressed in MDL. The formatter drops `refresh` silently when Commit="No". During round-trip, the describe output will drop this flag — acceptable as the runtime ignores RefreshInClient when there is no commit.

**CHANGE:**
| Commit | RefreshInClient | MDL output |
|--------|----------------|------------|
| `"No"` | `false` | `change $X (attr);` |
| `"Yes"` | `false` | `change $X (attr) with commit;` |
| `"Yes"` | `true` | `change $X (attr) with commit refresh;` |
| `"YesWithoutEvents"` | `false` | `change $X (attr) with commit without events;` |
| `"YesWithoutEvents"` | `true` | `change $X (attr) with commit without events refresh;` |

**COMMIT:**
| WithEvents | RefreshInClient | MDL output |
|------------|----------------|------------|
| `false` | `false` | `commit $X;` |
| `true` | `false` | `commit $X with events;` |
| `true` | `true` | `commit $X with events refresh;` |

> Note: `refresh` is only valid inside `with events`, so `WithEvents=false` + `RefreshInClient=true` drops refresh silently — consistent with CREATE/CHANGE rule.

**DELETE:**
| RefreshInClient | MDL output |
|----------------|------------|
| `false` | `delete $X;` |
| `true` | `delete $X refresh;` |

**ROLLBACK:**
| RefreshInClient | MDL output |
|----------------|------------|
| `false` | `rollback $X;` |
| `true` | `rollback $X refresh;` |

### Grammar Changes

File: `mdl/grammar/domains/MDLMicroflow.g4`

**createObjectStatement** (replace old clause with commit block):
```antlr
createObjectStatement
    : (VARIABLE EQUALS)? CREATE nonListDataType (LPAREN memberAssignmentList? RPAREN)?
      (WITH COMMIT (WITHOUT EVENTS)? REFRESH?)? onErrorClause?
    ;
```

**changeObjectStatement** (replace old `REFRESH?` with commit block):
```antlr
changeObjectStatement
    : CHANGE VARIABLE (LPAREN memberAssignmentList? RPAREN)?
      (WITH COMMIT (WITHOUT EVENTS)? REFRESH?)? onErrorClause?
    ;
```

> The commit block `(WITH COMMIT (WITHOUT EVENTS)? REFRESH?)?` is a single atomic clause: if any part appears, `WITH COMMIT` is the head. Standalone `REFRESH` is not valid for CREATE/CHANGE.

**commitStatement** (change to match CREATE/CHANGE pattern — refresh requires with):
```antlr
commitStatement
    : COMMIT VARIABLE (WITH EVENTS REFRESH?)? onErrorClause?
    ;
```

> Previously `(WITH EVENTS)? REFRESH?` which allowed `commit $V refresh;`. Now `REFRESH` is nested inside the `WITH EVENTS` clause, consistent with CREATE/CHANGE's `WITH COMMIT (WITHOUT EVENTS)? REFRESH?`.

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

The formatter must produce MDL that, when parsed back, produces identical BSON (round-trip). All six combinations per activity type from the Round-Trip Table above must be covered.

**File: `mdl/executor/microflows_format_action_v2.go`**

> **Consistency rule:** CREATE/CHANGE/COMMIT must produce the same clause pattern in describe output: `with <type> [(without events)] [refresh]`. If Commit/WithEvents is "No"/false, never emit `refresh` even if RefreshInClient=true (the runtime ignores it).

**`formatCreateObjectActionGen`:** Replace hardcoded output with round-trip-aware clause:
```go
func formatCreateObjectActionGen(a *genMf.CreateObjectAction) string {
    // ... existing format logic (entityName, outputVar, members) ...
    commitClause := ""
    refreshClause := ""
    switch a.Commit() {
    case "Yes":
        commitClause = " with commit"
        if a.RefreshInClient() {
            refreshClause = " refresh"
        }
    case "YesWithoutEvents":
        commitClause = " with commit without events"
        if a.RefreshInClient() {
            refreshClause = " refresh"
        }
    }
    // Commit="No": never emit refresh
    return fmt.Sprintf("$%s = create %s (%s)%s%s;", outputVar, entityName, members, commitClause, refreshClause)
}
```

**`formatChangeObjectActionGen`:** Same pattern. Currently outputs `change $V (members)[refresh];`. Replace with:
```go
func formatChangeObjectActionGen(a *genMf.ChangeObjectAction) string {
    // ... existing format logic ...
    commitClause := ""
    refreshClause := ""
    switch a.Commit() {
    case "Yes":
        commitClause = " with commit"
        if a.RefreshInClient() {
            refreshClause = " refresh"
        }
    case "YesWithoutEvents":
        commitClause = " with commit without events"
        if a.RefreshInClient() {
            refreshClause = " refresh"
        }
    }
    return fmt.Sprintf("change $%s (%s)%s%s;", varName, members, commitClause, refreshClause)
}
```

**`formatCommitActionGen`:** Already handles `with events` + `refresh`. Verify the pattern matches CREATE/CHANGE:
```go
func formatCommitActionGen(a *genMf.CommitAction) string {
    varName := strings.TrimSpace(a.CommitVariableName())
    if varName == "" { varName = "Object" }
    suffix := ""
    if a.WithEvents() {
        suffix += " with events"          // same `with <type>` head as CREATE/CHANGE
        if a.RefreshInClient() {
            suffix += " refresh"           // refresh only inside `with` clause
        }
    }
    // No `with events` → no refresh (consistent with CREATE/CHANGE rule)
    if a.ErrorHandlingType() == "Continue" {
        suffix += " on error continue"
    }
    return fmt.Sprintf("commit $%s%s;", varName, suffix)
}
```

**`formatDeleteActionGen`:** Add refresh support:
```go
func formatDeleteActionGen(a *genMf.DeleteAction) string {
    suffix := ""
    if a.RefreshInClient() {
        suffix = " refresh"
    }
    return fmt.Sprintf("delete $%s%s;", a.DeleteVariableName(), suffix)
}
```

**`formatRollbackActionGen`:** No change needed (already handles `refresh`).

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

Each test creates a microflow via MDL, describes it, and asserts the describe output matches the input. Every row in the Round-Trip Table must be covered:

```go
// CREATE — 4 valid combinations
func TestRoundtripMicroflow_CreateDefault(t *testing.T) {
    // Input:  $Obj = create Module.Entity (Name = 'x');
    // Expect: describe output must not add "with commit" or "refresh"
}
func TestRoundtripMicroflow_CreateWithCommit(t *testing.T) {
    // Input:  $Obj = create Module.Entity (Name = 'x') with commit;
    // Expect: $Obj = create Module.Entity (Name = 'x') with commit;
}
func TestRoundtripMicroflow_CreateWithCommitRefresh(t *testing.T) {
    // Input:  $Obj = create Module.Entity (Name = 'x') with commit refresh;
    // Expect: $Obj = create Module.Entity (Name = 'x') with commit refresh;
}
func TestRoundtripMicroflow_CreateWithCommitWithoutEvents(t *testing.T) {
    // Input:  $Obj = create Module.Entity (Name = 'x') with commit without events;
    // Expect: $Obj = create Module.Entity (Name = 'x') with commit without events;
}
func TestRoundtripMicroflow_CreateWithCommitWithoutEventsRefresh(t *testing.T) {
    // Input:  $Obj = create Module.Entity (Name = 'x') with commit without events refresh;
    // Expect: $Obj = create Module.Entity (Name = 'x') with commit without events refresh;
}
// Note: Commit="No"+Refresh=true cannot be expressed in MDL; BSON
// coming from Studio Pro with this combo drops refresh on describe.

// CHANGE — 4 valid combinations (same pattern as CREATE)
func TestRoundtripMicroflow_ChangeDefault(t *testing.T) { /* ... */ }
func TestRoundtripMicroflow_ChangeWithCommit(t *testing.T) { /* ... */ }
func TestRoundtripMicroflow_ChangeWithCommitRefresh(t *testing.T) { /* ... */ }
func TestRoundtripMicroflow_ChangeWithCommitWithoutEvents(t *testing.T) { /* ... */ }
func TestRoundtripMicroflow_ChangeWithCommitWithoutEventsRefresh(t *testing.T) { /* ... */ }

// COMMIT — 3 valid combinations (with events required for refresh)
func TestRoundtripMicroflow_CommitDefault(t *testing.T) { /* ... */ }
func TestRoundtripMicroflow_CommitWithEvents(t *testing.T) { /* ... */ }
func TestRoundtripMicroflow_CommitWithEventsRefresh(t *testing.T) { /* ... */ }

// DELETE — 2 combinations
func TestRoundtripMicroflow_DeleteDefault(t *testing.T) {
    // Input:  delete $Obj;
    // Expect: delete $Obj;
}
func TestRoundtripMicroflow_DeleteWithRefresh(t *testing.T) {
    // Input:  delete $Obj refresh;
    // Expect: delete $Obj refresh;
}

// ROLLBACK — 2 combinations (no change, existing tests)
func TestRoundtripMicroflow_RollbackDefault(t *testing.T) { /* ... */ }
func TestRoundtripMicroflow_RollbackWithRefresh(t *testing.T) { /* ... */ }
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

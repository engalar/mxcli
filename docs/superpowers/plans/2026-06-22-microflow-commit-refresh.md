# Microflow Activity Commit/Refresh Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add MDL support for Commit, WithEvents/WithoutEvents, and RefreshInClient properties across CREATE/CHANGE/COMMIT/DELETE/ROLLBACK microflow activities, with full round-trip (create → describe → recreate).

**Architecture:** Grammar → AST → Visitor → Executor (BSON write) → Formatter (BSON read → MDL). Each layer is modified in order, with tests at each step.

**Tech Stack:** ANTLR4 grammar, Go gen-typed elements, BSON codec

## Global Constraints

- REFRESH always follows `with commit` / `with commit without events` / `with events` — never standalone (except DELETE/ROLLBACK)
- Describe output must be identical across CREATE/CHANGE/COMMIT (same `with <type> [without events] [refresh]` pattern)
- When Commit="No" or WithEvents=false, never emit `refresh` even if RefreshInClient=true
- COMMIT grammar: `(WITH EVENTS REFRESH?)?` not `(WITH EVENTS)? REFRESH?`
- All valid BSON combinations from the Round-Trip Table must round-trip
- Existing formatter tests for COMMIT/ROLLBACK must not break

---
### Task 1: Grammar (MDLMicroflow.g4)

**Files:**
- Modify: `mdl/grammar/domains/MDLMicroflow.g4` (3 rules)

- [ ] **Step 1:** Change `createObjectStatement` rule — replace existing with commit clause:

```antlr
createObjectStatement
    : (VARIABLE EQUALS)? CREATE nonListDataType (LPAREN memberAssignmentList? RPAREN)?
      (WITH COMMIT (WITHOUT EVENTS)? REFRESH?)? onErrorClause?
    ;
```

- [ ] **Step 2:** Change `changeObjectStatement` rule — replace existing `REFRESH?` with commit clause:

```antlr
changeObjectStatement
    : CHANGE VARIABLE (LPAREN memberAssignmentList? RPAREN)?
      (WITH COMMIT (WITHOUT EVENTS)? REFRESH?)? onErrorClause?
    ;
```

- [ ] **Step 3:** Change `commitStatement` rule — nest REFRESH inside WITH EVENTS:

```antlr
commitStatement
    : COMMIT VARIABLE (WITH EVENTS REFRESH?)? onErrorClause?
    ;
```

- [ ] **Step 4:** Change `deleteObjectStatement` rule — add REFRESH:

```antlr
deleteObjectStatement
    : DELETE VARIABLE REFRESH? onErrorClause?
    ;
```

- [ ] **Step 5:** Regenerate parser:

```bash
make grammar
```

- [ ] **Step 6:** Build to verify grammar compiles:

```bash
go build ./mdl/...
```

- [ ] **Step 7:** Commit:

```bash
git add mdl/grammar/
git commit -m "grammar: add commit clause to create/change, refresh to delete, nest refresh in with events for commit"
```

### Task 2: AST (ast_microflow.go)

**Files:**
- Modify: `mdl/ast/ast_microflow.go`

- [ ] **Step 1:** Add fields to `CreateObjectStmt`:

```go
type CreateObjectStmt struct {
    Variable        string
    EntityType      NonListDataType
    Changes         []MemberChange
    WithCommit      bool                 // true = "with commit"
    WithoutEvents   bool                 // true = "without events"
    RefreshInClient bool                 // true = "refresh"
    ErrorHandling   *ErrorHandlingClause
    Annotations     *ActivityAnnotations
}
```

- [ ] **Step 2:** Add fields to `ChangeObjectStmt`:

```go
type ChangeObjectStmt struct {
    Variable        string
    Changes         []MemberChange
    WithCommit      bool                 // true = "with commit"
    WithoutEvents   bool                 // true = "without events"
    RefreshInClient bool                 // existing
    ErrorHandling   *ErrorHandlingClause
    Annotations     *ActivityAnnotations
}
```

- [ ] **Step 3:** Add `RefreshInClient` field to `MfDeleteStmt`:

```go
type MfDeleteStmt struct {
    Variable        string
    RefreshInClient bool                 // true = "refresh"
    ErrorHandling   *ErrorHandlingClause
    Annotations     *ActivityAnnotations
}
```

- [ ] **Step 4:** Build to verify:

```bash
go build ./mdl/...
```

- [ ] **Step 5:** Commit:

```bash
git add mdl/ast/ast_microflow.go
git commit -m "ast: add WithCommit/WithoutEvents to CreateObjectStmt/ChangeObjectStmt, RefreshInClient to MfDeleteStmt"
```

### Task 3: Visitor (visitor_microflow_statements.go)

**Files:**
- Modify: `mdl/visitor/visitor_microflow_statements.go`

- [ ] **Step 1:** Update `buildCreateObjectStatement` — parse commit clause:

```go
func buildCreateObjectStatement(ctx parser.ICreateObjectStatementContext) *ast.CreateObjectStmt {
    stmt := &ast.CreateObjectStmt{}
    // ... existing parsing for Variable, EntityType, Changes, ErrorHandling ...
    if createCtx.COMMIT() != nil {
        stmt.WithCommit = true
        if createCtx.EVENTS() != nil { // WITHOUT EVENTS
            stmt.WithoutEvents = true
        }
    }
    if createCtx.REFRESH() != nil {
        stmt.RefreshInClient = true
    }
    return stmt
}
```

- [ ] **Step 2:** Update `buildChangeObjectStatement` — parse commit clause:

```go
func buildChangeObjectStatement(ctx parser.IChangeObjectStatementContext) *ast.ChangeObjectStmt {
    stmt := &ast.ChangeObjectStmt{}
    // ... existing parsing for Variable, Changes, ErrorHandling ...
    if changeCtx.COMMIT() != nil {
        stmt.WithCommit = true
        if changeCtx.EVENTS() != nil { // WITHOUT EVENTS
            stmt.WithoutEvents = true
        }
    }
    if changeCtx.REFRESH() != nil {
        stmt.RefreshInClient = true
    }
    return stmt
}
```

- [ ] **Step 3:** Update `buildDeleteObjectStatement` — parse refresh:

```go
func buildDeleteObjectStatement(ctx parser.IDeleteObjectStatementContext) *ast.DeleteObjectStmt {
    stmt := &ast.DeleteObjectStmt{}
    // ... existing ...
    if deleteCtx.REFRESH() != nil {
        stmt.RefreshInClient = true
    }
    return stmt
}
```

- [ ] **Step 4:** Build to verify:

```bash
go build ./mdl/...
```

- [ ] **Step 5:** Commit:

```bash
git add mdl/visitor/visitor_microflow_statements.go
git commit -m "visitor: parse commit clause in create/change, refresh in delete"
```

### Task 4: Executor — CreateObjectAction and ChangeObjectAction

**Files:**
- Modify: `mdl/executor/flowbuilder_actions_change_v2.go`

- [ ] **Step 1:** Update `addCreateObjectActionGen` — set Commit+RefreshInClient from AST:

```go
func (fb *flowBuilderGen) addCreateObjectActionGen(s *ast.CreateObjectStmt) element.ID {
    action := genMf.NewCreateObjectAction()
    assignFreshID(action)
    action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
    action.SetOutputVariableName(s.Variable)

    // Commit mapping
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

    // Entity (unchanged)
    entityQN := ""
    if s.EntityType.Module != "" && s.EntityType.Name != "" {
        entityQN = s.EntityType.Module + "." + s.EntityType.Name
        action.SetEntityQualifiedName(entityQN)
    }
    if fb.varTypes != nil && entityQN != "" {
        fb.varTypes[s.Variable] = entityQN
    }
    for _, change := range s.Changes {
        mc := genMf.NewMemberChange()
        assignFreshID(mc)
        mc.SetType("Set")
        mc.SetValue(mendixExprValue(fb.memberExpressionToStringGen(change.Value, entityQN, change.Attribute)))
        applyResolvedMemberChangeGen(mc, fb.resolveMemberChangeGen(change.Attribute, entityQN))
        action.AddItems(mc)
    }
    return fb.genActivityWrap(action, s.ErrorHandling, s.Variable)
}
```

- [ ] **Step 2:** Update `addChangeObjectActionGen` — set Commit from AST:

```go
func (fb *flowBuilderGen) addChangeObjectActionGen(s *ast.ChangeObjectStmt) element.ID {
    action := genMf.NewChangeObjectAction()
    assignFreshID(action)
    action.SetErrorHandlingType(fb.ehTypeGen(nil))
    action.SetChangeVariableName(s.Variable)

    commitVal := "No"
    if s.WithCommit {
        if s.WithoutEvents {
            commitVal = "YesWithoutEvents"
        } else {
            commitVal = "Yes"
        }
    }
    action.SetCommit(commitVal)
    action.SetRefreshInClient(s.RefreshInClient || len(s.Changes) == 0)

    // ... rest unchanged (entity QN, member changes) ...
}
```

- [ ] **Step 3:** Build to verify:

```bash
go build ./mdl/executor/
```

- [ ] **Step 4:** Commit:

```bash
git add mdl/executor/flowbuilder_actions_change_v2.go
git commit -m "executor: set Commit+RefreshInClient from AST for Create/Change actions"
```

### Task 5: Executor — CommitAction, DeleteAction, RollbackAction

**Files:**
- Modify: `mdl/executor/flowbuilder_actions_v2.go`

- [ ] **Step 1:** Verify `addCommitActionGen` already reads `s.WithEvents` and `s.RefreshInClient` (no change needed).

- [ ] **Step 2:** Update `addDeleteActionGen` — set RefreshInClient:

```go
// Inside existing addDeleteActionGen (find the function):
action.SetRefreshInClient(s.RefreshInClient)
```

- [ ] **Step 3:** Verify `addRollbackActionGen` already reads `s.RefreshInClient` (no change needed).

- [ ] **Step 4:** Build to verify:

```bash
go build ./mdl/executor/
```

- [ ] **Step 5:** Commit:

```bash
git add mdl/executor/flowbuilder_actions_v2.go
git commit -m "executor: set RefreshInClient for DeleteAction"
```

### Task 6: Formatter (BSON → MDL describe)

**Files:**
- Modify: `mdl/executor/microflows_format_action_v2.go`

- [ ] **Step 1:** Update `formatCreateObjectActionGen` — emit commit clause:

```go
func formatCreateObjectActionGen(a *genMf.CreateObjectAction) string {
    entityName := strings.TrimSpace(a.EntityQualifiedName())
    if entityName == "" { entityName = "Entity" }
    outputVar := strings.TrimSpace(a.OutputVariableName())
    if outputVar == "" { outputVar = "NewObject" }
    entityModule := ""
    if parts := strings.SplitN(entityName, ".", 2); len(parts) == 2 {
        entityModule = parts[0]
    }
    members := collectInitialMemberAssignmentsGen(a.ItemsItems(), entityModule)

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

    if len(members) > 0 {
        return fmt.Sprintf("$%s = create %s (%s)%s%s;", outputVar, entityName,
            strings.Join(members, ", "), commitClause, refreshClause)
    }
    return fmt.Sprintf("$%s = create %s%s%s;", outputVar, entityName, commitClause, refreshClause)
}
```

- [ ] **Step 2:** Update `formatChangeObjectActionGen` — emit commit clause:

```go
func formatChangeObjectActionGen(a *genMf.ChangeObjectAction) string {
    varName := strings.TrimSpace(a.ChangeVariableName())
    if varName == "" { varName = "Object" }
    items := a.ItemsItems()
    var members []string
    for _, item := range items {
        mc, ok := item.(*genMf.MemberChange)
        if !ok || mc == nil { continue }
        members = append(members, formatMemberChangeGen(mc))
    }

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

    return fmt.Sprintf("change $%s (%s)%s%s;", varName,
        strings.Join(members, ", "), commitClause, refreshClause)
}
```

- [ ] **Step 3:** Verify `formatCommitActionGen` pattern matches (refresh inside `with events` block). Add a guard:

```go
// Ensure refresh only emitted inside `with events`:
if a.WithEvents() {
    suffix += " with events"
    if a.RefreshInClient() {
        suffix += " refresh"
    }
}
```

- [ ] **Step 4:** Update `formatDeleteActionGen` — emit refresh:

```go
func formatDeleteActionGen(a *genMf.DeleteAction) string {
    suffix := ""
    if a.RefreshInClient() {
        suffix = " refresh"
    }
    return fmt.Sprintf("delete $%s%s;", a.DeleteVariableName(), suffix)
}
```

- [ ] **Step 5:** Build to verify:

```bash
go build ./mdl/executor/
```

- [ ] **Step 6:** Commit:

```bash
git add mdl/executor/microflows_format_action_v2.go
git commit -m "formatter: emit commit/refresh clauses for create/change/delete describe output"
```

### Task 7: Formatter Unit Tests

**Files:**
- Modify: `mdl/executor/microflows_format_action_v2_test.go`

- [ ] **Step 1:** Add/update `TestFormatActionGen_CreateObjectAction` — cover all 4 commit combinations:

```go
func TestFormatActionGen_CreateWithCommitDefault(t *testing.T) {
    a := newGenAction(t, "Microflows$CreateObjectAction").(*genMf.CreateObjectAction)
    a.SetOutputVariableName("Obj")
    a.SetEntityQualifiedName("Mod.Entity")
    a.SetCommit("No")
    got := formatCreateObjectActionGen(a)
    if !strings.Contains(got, "$Obj = create Mod.Entity") {
        t.Errorf("expected create with default commit, got: %s", got)
    }
    if strings.Contains(got, "with commit") || strings.Contains(got, "refresh") {
        t.Errorf("no commit clause expected for Commit=No, got: %s", got)
    }
}

func TestFormatActionGen_CreateWithCommit(t *testing.T) {
    a := newGenAction(t, "Microflows$CreateObjectAction").(*genMf.CreateObjectAction)
    a.SetOutputVariableName("Obj")
    a.SetEntityQualifiedName("Mod.Entity")
    a.SetCommit("Yes")
    got := formatCreateObjectActionGen(a)
    if !strings.Contains(got, "with commit") || strings.Contains(got, "without events") {
        t.Errorf("expected 'with commit' without 'without events', got: %s", got)
    }
}

func TestFormatActionGen_CreateWithCommitRefresh(t *testing.T) {
    a := newGenAction(t, "Microflows$CreateObjectAction").(*genMf.CreateObjectAction)
    a.SetOutputVariableName("Obj")
    a.SetEntityQualifiedName("Mod.Entity")
    a.SetCommit("Yes")
    a.SetRefreshInClient(true)
    got := formatCreateObjectActionGen(a)
    if !strings.Contains(got, "with commit") || !strings.Contains(got, "refresh") {
        t.Errorf("expected 'with commit refresh', got: %s", got)
    }
}

func TestFormatActionGen_CreateWithCommitWithoutEvents(t *testing.T) {
    a := newGenAction(t, "Microflows$CreateObjectAction").(*genMf.CreateObjectAction)
    a.SetOutputVariableName("Obj")
    a.SetEntityQualifiedName("Mod.Entity")
    a.SetCommit("YesWithoutEvents")
    got := formatCreateObjectActionGen(a)
    if !strings.Contains(got, "without events") {
        t.Errorf("expected 'without events', got: %s", got)
    }
}
```

- [ ] **Step 2:** Add `TestFormatActionGen_DeleteWithRefresh`:

```go
func TestFormatActionGen_DeleteWithRefresh(t *testing.T) {
    a := newGenAction(t, "Microflows$DeleteAction").(*genMf.DeleteAction)
    a.SetDeleteVariableName("Obj")
    a.SetRefreshInClient(true)
    got := formatDeleteActionGen(a)
    if !strings.Contains(got, "refresh") {
        t.Errorf("expected 'delete $Obj refresh', got: %s", got)
    }
}
```

- [ ] **Step 3:** Run formatter tests:

```bash
go test ./mdl/executor/ -run "TestFormatActionGen" -v -count=1
```

- [ ] **Step 4:** Commit:

```bash
git add mdl/executor/microflows_format_action_v2_test.go
git commit -m "test: add formatter tests for create/delete commit/refresh clauses"
```

### Task 8: Builder Unit Tests

**Files:**
- Modify: `mdl/executor/flowbuilder_actions_change_v2_test.go`

- [ ] **Step 1:** Add test for `addCreateObjectActionGen` with commit flags:

```go
func TestAddCreateObjectActionGenWithCommit(t *testing.T) {
    fb, stmt := newCreateActionTestContext(t)
    stmt.WithCommit = true
    stmt.WithoutEvents = false
    stmt.RefreshInClient = true
    fb.addCreateObjectActionGen(stmt)
    act := actionFromObjects(t, fb).(*genMf.CreateObjectAction)
    if act.Commit() != "Yes" {
        t.Errorf("expected Commit=Yes, got %q", act.Commit())
    }
    if !act.RefreshInClient() {
        t.Error("expected RefreshInClient=true")
    }
}
```

- [ ] **Step 2:** Run builder tests:

```bash
go test ./mdl/executor/ -run "TestAddCreateObjectActionGen|TestAddChangeObjectActionGen" -v -count=1
```

- [ ] **Step 3:** Commit:

```bash
git add mdl/executor/flowbuilder_actions_change_v2_test.go
git commit -m "test: add builder tests for create/change commit/refresh flags"
```

### Task 9: Round-Trip Integration Tests

**Files:**
- Modify: `mdl/executor/roundtrip_microflow_test.go`

- [ ] **Step 1:** Add round-trip tests for CREATE with all 4 valid combinations:

```go
func TestRoundtripMicroflow_CreateWithCommitRefresh(t *testing.T) {
    script := `
        create microflow Mod.ACT_Test returns Nothing begin
            $Obj = create Mod.Entity (Name = 'x') with commit refresh;
        end;
    /
    `
    createAndDescribeMatch(t, script,
        "$Obj = create Mod.Entity (Name = 'x') with commit refresh;")
}

func TestRoundtripMicroflow_CreateWithCommitWithoutEventsRefresh(t *testing.T) {
    // Similar pattern
}
```

- [ ] **Step 2:** Add round-trip test for CHANGE with commit clause:

```go
func TestRoundtripMicroflow_ChangeWithCommit(t *testing.T) {
    script := `
        create microflow Mod.ACT_Test returns Nothing begin
            change $Obj (Name = 'x') with commit;
        end;
    /
    `
    createAndDescribeMatch(t, script,
        "change $Obj (Name = 'x') with commit;")
}
```

- [ ] **Step 3:** Add round-trip test for DELETE with refresh:

```go
func TestRoundtripMicroflow_DeleteWithRefresh(t *testing.T) {
    script := `
        create microflow Mod.ACT_Test returns Nothing begin
            delete $Obj refresh;
        end;
    /
    `
    createAndDescribeMatch(t, script,
        "delete $Obj refresh;")
}
```

- [ ] **Step 4:** Run all round-trip tests:

```bash
go test ./mdl/executor/ -run "TestRoundtripMicroflow" -v -count=1
```

- [ ] **Step 5:** Run full test suite:

```bash
go test ./mdl/executor/ -count=1 -timeout 120s
```

- [ ] **Step 6:** Commit:

```bash
git add mdl/executor/roundtrip_microflow_test.go
git commit -m "test: add round-trip tests for create/change/delete commit/refresh"
```

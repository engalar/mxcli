# Adding New Statements

Step-by-step guide for adding a new MDL statement to the parser and executor.

## Overview

Adding a new statement requires changes in five layers:

1. **Grammar** -- lexer tokens and parser rules
2. **AST** -- typed node struct
3. **Visitor** -- parse tree to AST conversion
4. **Executor** -- AST to SDK call
5. **Tests** -- parser and execution tests

## Step 1: Grammar Rules

### Add Tokens (if needed)

In `mdl/grammar/MDLLexer.g4`, add any new keywords:

```antlr
WORKFLOW : W O R K F L O W ;
```

### Add Parser Rule

In `mdl/grammar/MDLParser.g4`, add the statement rule:

```antlr
// Add to the statement alternatives
statement
    : ...
    | createWorkflowStatement
    ;

// Define the new statement
createWorkflowStatement
    : CREATE WORKFLOW qualifiedName
      '(' workflowBody ')'
    ;

workflowBody
    : workflowActivity (workflowActivity)*
    ;

workflowActivity
    : USER_TASK IDENTIFIER STRING_LITERAL
    ;
```

### Regenerate Parser

```bash
make grammar
```

## Step 2: AST Node

In `mdl/ast/`, create or extend an AST file:

```go
// ast/ast_workflow.go
package ast

type CreateWorkflowStmt struct {
    Name       QualifiedName
    Activities []*WorkflowActivity
}

type WorkflowActivity struct {
    Type string
    Name string
    Label string
}
```

## Step 3: Visitor Implementation

In `mdl/visitor/visitor.go`, implement the listener method:

```go
func (v *MDLVisitor) EnterCreateWorkflowStatement(ctx *parser.CreateWorkflowStatementContext) {
    stmt := &ast.CreateWorkflowStmt{
        Name: v.visitQualifiedName(ctx.QualifiedName()),
    }

    for _, actCtx := range ctx.AllWorkflowActivity() {
        activity := &ast.WorkflowActivity{
            Type:  "UserTask",
            Name:  actCtx.IDENTIFIER().GetText(),
            Label: unquote(actCtx.STRING_LITERAL().GetText()),
        }
        stmt.Activities = append(stmt.Activities, activity)
    }

    v.statements = append(v.statements, stmt)
}
```

## Step 4: Executor Handler

In `mdl/executor/`, create the execution handler:

```go
// executor/cmd_workflows.go
func (e *Executor) execCreateWorkflow(stmt *ast.CreateWorkflowStmt) error {
    module, err := e.findModule(stmt.Name.Module)
    if err != nil {
        return err
    }

    // Build the workflow using SDK types
    workflow := &workflows.Workflow{
        Name: stmt.Name.Name,
        // ... populate from stmt
    }

    return e.writer.CreateWorkflow(module.ID, workflow)
}
```

Register the handler in `executor.go`:

```go
func (e *Executor) Execute(stmt ast.Statement) error {
    switch s := stmt.(type) {
    // ... existing cases
    case *ast.CreateWorkflowStmt:
        return e.execCreateWorkflow(s)
    }
}
```

## Step 5: Tests

### Parser Test

Verify the grammar parses correctly:

```go
func TestParseCreateWorkflow(t *testing.T) {
    input := `CREATE WORKFLOW Sales.ApprovalProcess (
        USER_TASK reviewTask 'Review Order'
    );`
    stmts, err := parser.Parse(input)
    require.NoError(t, err)
    require.Len(t, stmts, 1)

    ws := stmts[0].(*ast.CreateWorkflowStmt)
    assert.Equal(t, "Sales", ws.Name.Module)
    assert.Equal(t, "ApprovalProcess", ws.Name.Name)
}
```

### Execution Test

Test against a real MPR file:

```go
func TestExecuteCreateWorkflow(t *testing.T) {
    writer := setupTestProject(t)
    defer writer.Close()

    err := executor.ExecuteString(writer,
        `CREATE WORKFLOW TestModule.MyWorkflow (...);`)
    require.NoError(t, err)
}
```

## Checklist

- [ ] Lexer tokens added (if new keywords)
- [ ] Parser rule added
- [ ] Parser regenerated (`make grammar`)
- [ ] AST node defined
- [ ] Visitor listener implemented
- [ ] Executor handler registered
- [ ] Parser tests pass
- [ ] Execution tests pass
- [ ] `mxcli check` validates the new syntax
- [ ] Studio Pro opens the modified project without errors

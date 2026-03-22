# Workflow Metadata Gaps: DISPLAY NAME, DESCRIPTION, EXPORT LEVEL

## Problem

DESCRIBE WORKFLOW outputs WorkflowName, WorkflowDescription, and ExportLevel as comments (`-- Display Name: X`), which CREATE cannot parse back. This breaks round-trip fidelity. The `bson discover` tool confirms 0% semantic field coverage for Workflows$Workflow.

## Solution

Add three optional MDL clauses to CREATE/ALTER WORKFLOW: `DISPLAY NAME`, `DESCRIPTION`, `EXPORT LEVEL`.

## Syntax

```sql
CREATE WORKFLOW Module.MyWorkflow
  PARAMETER $WorkflowContext: Module.Entity
  DISPLAY NAME 'My Workflow'
  DESCRIPTION 'Handles the approval process'
  EXPORT LEVEL Hidden
  DUE DATE 'addDays([%CurrentDateTime%], 7)'
  OVERVIEW PAGE Module.OverviewPage

BEGIN
  ...
END WORKFLOW
```

All three clauses are optional. Position: workflow header section (between PARAMETER and BEGIN), alongside existing DUE DATE and OVERVIEW PAGE.

### EXPORT LEVEL values

From `WorkflowsExportLevel` enum: `Hidden` (default), `Usable`.

## Changes Required

### 1. Grammar (MDLParser.g4)

Add three new rules in the `workflowHeader` section:

```antlr
workflowDisplayName: DISPLAY NAME stringLiteral;
workflowDescription: DESCRIPTION stringLiteral;
workflowExportLevel: EXPORT LEVEL (HIDDEN | USABLE);
```

New tokens needed: `DISPLAY`, `DESCRIPTION` (may already exist), `HIDDEN`, `USABLE`.

Add these as optional children of the `createWorkflowStatement` rule, after `workflowParameter` and before `BEGIN`.

### 2. AST (ast/ast_workflow.go)

Add fields to `CreateWorkflowStatement`:

```go
DisplayName  string // from DISPLAY NAME 'text'
Description  string // from DESCRIPTION 'text'
ExportLevel  string // "Hidden" or "Usable", default ""
```

### 3. Visitor (visitor/visitor_workflow.go)

Handle the new grammar rules in `ExitCreateWorkflowStatement` or dedicated listener methods. Extract string values and populate AST fields.

### 4. Executor CREATE path (executor/cmd_workflows_write.go)

Pass DisplayName, Description, ExportLevel from AST to the workflow SDK struct:

```go
wf.WorkflowName = stmt.DisplayName
wf.WorkflowDescription = stmt.Description
if stmt.ExportLevel != "" {
    wf.ExportLevel = stmt.ExportLevel
}
```

### 5. DESCRIBE output (executor/cmd_workflows.go)

Change from comment format to MDL clauses:

```go
// Before:
// lines = append(lines, fmt.Sprintf("-- Display Name: %s", targetWf.WorkflowName))

// After:
if targetWf.WorkflowName != "" {
    lines = append(lines, fmt.Sprintf("  DISPLAY NAME '%s'", escape(targetWf.WorkflowName)))
}
```

### 6. BSON writer (sdk/mpr/writer_workflow.go)

Already handles WorkflowName, WorkflowDescription, ExportLevel in BSON serialization. The SDK workflow struct already has these fields. No writer changes needed — just need to populate the struct correctly from the executor.

## Scope

### In scope
- DISPLAY NAME, DESCRIPTION, EXPORT LEVEL in CREATE WORKFLOW
- Updated DESCRIBE output (clauses instead of comments)
- Grammar, AST, visitor, executor changes
- Round-trip test: DESCRIBE → CREATE → mx check → DESCRIBE (should match)

### Out of scope
- Annotation (UI sticky note, low round-trip value)
- AdminPage (rarely used)
- Excluded flag (rarely used)
- ALTER WORKFLOW support (future)
- UserTask-level TaskName/TaskDescription gaps (separate effort)

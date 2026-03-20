# Proposal: DESCRIBE Nanoflow (Enhancement)

## Overview

**Document type:** `Microflows$Nanoflow`
**Prevalence:** 227 across test projects (79 Enquiries, 97 Evora, 51 Lato)
**Priority:** High — nanoflows are heavily used, SHOW works but DESCRIBE is missing

Nanoflows execute client-side in the browser or native app. They share the same BSON structure as microflows (`MicroflowObjectCollection`, `MicroflowObject`, `SequenceFlow`) but run on the client. Currently `SHOW NANOFLOWS` works but `DESCRIBE NANOFLOW` does not.

## What Already Exists

| Layer | Status | Location |
|-------|--------|----------|
| **Go type** | Yes | `sdk/microflows/microflows.go` — `Nanoflow` struct (has `ObjectCollection` field) |
| **Parser** | **Partial** | `sdk/mpr/parser_nanoflow.go` — parses metadata and parameters only, **does NOT parse ObjectCollection** |
| **Reader** | Yes | `ListNanoflows()`, `GetNanoflow()` |
| **SHOW** | Yes | `showNanoflows()` in executor (activity count is always 0) |
| **DESCRIBE** | **No** | No `DescribeNanoflow` AST type or handler |
| **DROP** | **No** | No `DropNanoflowStmt` |

## Critical Gap: Nanoflow Parser Ignores Activities

The current `parseNanoflow()` in `sdk/mpr/parser_nanoflow.go` only parses:
- Name, Documentation, MarkAsUsed, Excluded (basic metadata)
- Parameters

It does **NOT** call `parseMicroflowObjectCollection()`, so the `ObjectCollection` field is always nil. This means:
- `SHOW NANOFLOWS` always shows 0 activities
- Any DESCRIBE handler would output an empty body
- The nanoflow serializer similarly ignores activities

The microflow parser (`parser_microflow.go`) **does** parse `ObjectCollection` via `parseMicroflowObjectCollection()`, which handles all flow objects and action types.

## Nanoflow vs Microflow Activities

Nanoflows use the same `MicroflowObject` types as microflows, but Mendix restricts which activities are available client-side:

### Activities available in nanoflows

| Activity | MDL Keyword | Notes |
|----------|-------------|-------|
| CreateVariable | `CREATE VARIABLE` | Same as microflow |
| ChangeVariable | `CHANGE VARIABLE` | Same as microflow |
| CreateObject | `CREATE` | Same as microflow |
| ChangeObject | `CHANGE` | Same as microflow |
| RetrieveAction | `RETRIEVE` | From database or by association |
| CreateList | `CREATE LIST` | Same as microflow |
| ChangeList | `CHANGE LIST` | Same as microflow |
| ListOperation | `LIST OPERATION` | Same as microflow |
| AggregateList | `AGGREGATE LIST` | Same as microflow |
| ShowPage | `SHOW PAGE` | Same as microflow |
| ClosePage | `CLOSE PAGE` | Same as microflow |
| ShowMessage | `SHOW MESSAGE` | Same as microflow |
| ValidationFeedback | `VALIDATION FEEDBACK` | Same as microflow |
| CallNanoflow | `CALL NANOFLOW` | Uses MicroflowCallAction with nanoflow target |
| CallJavaScriptAction | `CALL JAVASCRIPT ACTION` | Via JavaActionCallAction with JS action target |
| ExclusiveSplit | `IF ... THEN` | Same as microflow |
| ExclusiveMerge | (merge point) | Same as microflow |
| LoopedActivity | `LOOP` | Same as microflow |
| StartEvent | (implicit) | Same as microflow |
| EndEvent | `RETURN` | Same as microflow |
| ErrorEvent | (error handler) | Same as microflow |
| ShowHomePage | `SHOW HOME PAGE` | Same as microflow |

### Activities NOT available in nanoflows (microflow-only)

| Activity | MDL Keyword | Reason |
|----------|-------------|--------|
| CommitAction | `COMMIT` | Server-side database operation |
| DeleteAction | `DELETE` | Server-side database operation |
| RollbackAction | `ROLLBACK` | Server-side transaction |
| JavaActionCallAction | `CALL JAVA ACTION` | Server-side JVM execution |
| RestCallAction | `CALL REST` | Server-side HTTP (nanoflows use JS actions for HTTP) |
| LogMessageAction | `LOG MESSAGE` | Server-side logging |
| DownloadFileAction | `DOWNLOAD FILE` | Server-side file streaming |
| CastAction | `CAST` | Server-side type casting |
| ExecuteDatabaseQueryAction | `EXECUTE DATABASE QUERY` | Server-side SQL |
| CallExternalAction | `CALL EXTERNAL` | Server-side external calls |

### Nanoflow-specific considerations

- **CallNanoflow**: Uses the same `MicroflowCallAction` BSON type but targets a nanoflow. The formatter should output `CALL NANOFLOW` instead of `CALL MICROFLOW` when the target is a nanoflow.
- **CallJavaScriptAction**: Uses `JavaActionCallAction` BSON type but targets a JS action. The formatter should output `CALL JAVASCRIPT ACTION` instead of `CALL JAVA ACTION`.
- **Offline retrieval**: Nanoflows retrieve from the client-side database, which may have different behavior than server-side retrieves.

## Proposed MDL Syntax

### DESCRIBE NANOFLOW

```
DESCRIBE NANOFLOW Module.Name
```

Output format (same as DESCRIBE MICROFLOW but with NANOFLOW keyword):

```
/**
 * Validates the customer form before saving
 */
CREATE NANOFLOW MyModule.ValidateCustomerForm (
  $Customer: MyModule.Customer
)
RETURNS Boolean
BEGIN
  IF $Customer/Name = '' THEN
    VALIDATION FEEDBACK $Customer ATTRIBUTE Name MESSAGE 'Name is required';
    RETURN false;
  END IF;
  RETURN true;
END;
/
```

This matches the existing microflow DESCRIBE format exactly — parameters are inline in parentheses with `$` prefix, comma-separated, one per line.

### DROP NANOFLOW (optional, lower priority)

```
DROP NANOFLOW Module.Name;
```

## Implementation Steps

### 1. Enhance Nanoflow Parser (sdk/mpr/parser_nanoflow.go) — **Critical**

Add `ObjectCollection` parsing to `parseNanoflow()`:

```go
func parseNanoflow(data map[string]any) (*microflows.Nanoflow, error) {
    nf := &microflows.Nanoflow{...}
    // ... existing metadata parsing ...

    // ADD: Parse ObjectCollection (reuse microflow parsing)
    if objColl, ok := data["ObjectCollection"]; ok {
        nf.ObjectCollection = parseMicroflowObjectCollection(objColl)
    }

    // ADD: Parse Flows
    if flows, ok := data["Flows"]; ok {
        nf.Flows = parseMicroflowFlows(flows)
    }

    // ADD: Parse ReturnType
    if rt, ok := data["MicroflowReturnType"]; ok {
        nf.ReturnType = parseMicroflowReturnType(rt)
    }

    return nf, nil
}
```

This reuses the existing `parseMicroflowObjectCollection()` and related functions from `parser_microflow.go` — no new parsing code is needed since nanoflows use the same BSON structure.

### 2. Add AST Types (mdl/ast/ast_query.go)

```go
// In DescribeObjectType enum:
DescribeNanoflow

// Add String() case:
case DescribeNanoflow:
    return "NANOFLOW"
```

For DROP (optional):
```go
type DropNanoflowStmt struct {
    Name QualifiedName
}
```

### 3. Add Grammar Rules (MDLParser.g4)

The grammar likely already has `DESCRIBE NANOFLOW` syntax — the visitor just needs to wire it to a new AST type instead of silently ignoring it.

### 4. Add Visitor Mapping

Map `DESCRIBE NANOFLOW qualifiedName` to `DescribeStmt{ObjectType: DescribeNanoflow}`.

### 5. Add Executor Handler (mdl/executor/cmd_microflows_show.go)

```go
func (e *Executor) describeNanoflow(name ast.QualifiedName) error {
    // Look up nanoflow
    nanoflows, err := e.reader.ListNanoflows()
    // ... find by qualified name ...

    // Convert to Microflow (they share the same structure) and
    // delegate to describeMicroflowToString with "NANOFLOW" keyword
}
```

The existing `formatActivity()` and `formatAction()` in `cmd_microflows_format_action.go` handle all `MicroflowObject` types and work unchanged for nanoflows since they share the same types.

Consideration: when formatting a `MicroflowCallAction` that targets a nanoflow, output `CALL NANOFLOW` instead of `CALL MICROFLOW`. This requires checking whether the call target is a nanoflow (by looking it up in the nanoflow list) or adding a helper to distinguish.

### 6. Wire into Executor Dispatcher

```go
case ast.DescribeNanoflow:
    return e.describeNanoflow(s.Name)
```

### 7. Fix SHOW NANOFLOWS activity count

With the parser enhancement, `SHOW NANOFLOWS` will correctly report activity counts instead of always showing 0.

## Complexity

**Medium** — The main work is enhancing the nanoflow parser to parse ObjectCollection:
- Parser enhancement: ~20 lines (calls to existing functions)
- AST + Grammar + Visitor wiring: ~15 lines
- Executor handler: ~40 lines (find nanoflow, delegate to formatter)
- CALL NANOFLOW vs CALL MICROFLOW distinction: ~15 lines
- Total: ~90 lines of code

The formatter code in `cmd_microflows_format_action.go` requires **no changes** since nanoflows use the same `MicroflowObject` types.

## Testing

- Add `DESCRIBE NANOFLOW` examples to existing test files
- Verify against all 3 test projects (especially Evora with 97 nanoflows)
- Test activities: CreateObject, ChangeObject, Retrieve, ShowPage, ClosePage, ValidationFeedback, CallNanoflow, CallJavaScriptAction, ExclusiveSplit, Loop
- Verify that microflow-only activities (if somehow present) don't cause errors

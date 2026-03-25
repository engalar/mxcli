# SHOW CALLERS / CALLEES

These commands trace the call graph of your Mendix project, showing what calls a given element and what that element calls.

## Prerequisites

Both commands require a full catalog refresh:

```sql
REFRESH CATALOG FULL;
```

## SHOW CALLERS OF

Shows all elements that call or reference a given element.

**Syntax:**

```sql
SHOW CALLERS OF <qualified-name>
```

**Examples:**

```sql
-- Find everything that calls a microflow
SHOW CALLERS OF Sales.ACT_ProcessOrder;

-- Find what references an entity
SHOW CALLERS OF Sales.Customer;

-- Find what uses a page
SHOW CALLERS OF Sales.CustomerOverview;
```

### CLI Usage

```bash
# Basic caller lookup
mxcli callers -p app.mpr Module.MyMicroflow

# Transitive callers (full call chain)
mxcli callers -p app.mpr Module.MyMicroflow --transitive
```

The `--transitive` flag follows the call chain recursively, showing not just direct callers but also callers of callers.

## SHOW CALLEES OF

Shows all elements that a given element calls or references.

**Syntax:**

```sql
SHOW CALLEES OF <qualified-name>
```

**Examples:**

```sql
-- Find what a microflow calls
SHOW CALLEES OF Sales.ACT_ProcessOrder;

-- Find what entities a microflow uses
SHOW CALLEES OF Sales.SubmitOrder;
```

### CLI Usage

```bash
mxcli callees -p app.mpr Module.MyMicroflow
```

## Use Cases

### Before Refactoring

Check what calls a microflow before renaming or modifying it:

```sql
SHOW CALLERS OF Sales.ACT_OldName;
-- Review the list of callers before making changes
```

### Understanding Dependencies

Trace the full call chain of a complex microflow:

```sql
SHOW CALLEES OF Sales.ACT_ProcessOrder;
-- Shows: Sales.ValidateOrder, Sales.UpdateInventory, Sales.SendConfirmation
```

### Finding Dead Code

If `SHOW CALLERS` returns no results, the element may be unused:

```sql
SHOW CALLERS OF Sales.ACT_UnusedMicroflow;
-- Empty result = potentially dead code
```

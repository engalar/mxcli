# SHOW REFERENCES / IMPACT

These commands provide broader reference tracking and impact analysis, helping you understand how changes propagate through a project.

## Prerequisites

Both commands require a full catalog refresh:

```sql
REFRESH CATALOG FULL;
```

## SHOW REFERENCES OF

Shows all references to and from a given element, combining both incoming and outgoing relationships.

**Syntax:**

```sql
SHOW REFERENCES OF <qualified-name>
```

**Examples:**

```sql
-- Find all references to/from an entity
SHOW REFERENCES OF Sales.Customer;

-- Find all references to/from a microflow
SHOW REFERENCES OF Sales.ACT_ProcessOrder;

-- Find all references to/from a page
SHOW REFERENCES OF Sales.CustomerOverview;
```

### CLI Usage

```bash
mxcli refs -p app.mpr Module.Customer
```

### Difference from CALLERS/CALLEES

While `SHOW CALLERS` and `SHOW CALLEES` focus on the call graph direction, `SHOW REFERENCES` combines both directions into a single view. It shows every element that either references or is referenced by the target element.

## SHOW IMPACT OF

Performs impact analysis on an element, showing what would be affected if you changed or removed it.

**Syntax:**

```sql
SHOW IMPACT OF <qualified-name>
```

**Examples:**

```sql
-- Analyze impact of changing an entity
SHOW IMPACT OF Sales.Customer;

-- Analyze impact before removing a microflow
SHOW IMPACT OF Sales.ACT_CalculateTotal;

-- Check impact before moving an element
SHOW IMPACT OF Sales.CustomerOverview;
```

### CLI Usage

```bash
mxcli impact -p app.mpr Module.Customer
```

## Use Cases

### Pre-Change Impact Assessment

Before modifying an entity, check what would be affected:

```sql
-- What depends on the Customer entity?
SHOW IMPACT OF Sales.Customer;

-- Review results: pages, microflows, associations, access rules
-- Then decide if the change is safe to make
```

### Before Moving Elements

Moving elements across modules changes their qualified name and can break references:

```sql
-- Check impact before moving
SHOW IMPACT OF OldModule.Customer;

-- If impact is acceptable, proceed
MOVE ENTITY OldModule.Customer TO NewModule;
```

### Finding All Usages of an Enumeration

```sql
SHOW REFERENCES OF Sales.OrderStatus;
-- Shows: entities with attributes of this type, microflows that use it, pages that display it
```

### Dependency Mapping

Understand the full dependency web of a complex module:

```sql
SHOW REFERENCES OF Sales.Order;
SHOW REFERENCES OF Sales.OrderLine;
SHOW REFERENCES OF Sales.Order_Customer;
```

# Indexes

Indexes improve query performance for frequently searched or sorted attributes. They are defined after the attribute list in an entity definition.

## Syntax

```sql
INDEX (<column> [ASC|DESC] [, <column> [ASC|DESC] ...])
```

Indexes are placed after the closing parenthesis of the attribute list:

```sql
CREATE PERSISTENT ENTITY Sales.Order (
  OrderId: AutoNumber NOT NULL UNIQUE,
  OrderNumber: String(50) NOT NULL,
  CustomerId: Long,
  OrderDate: DateTime,
  Status: Enumeration(Sales.OrderStatus)
)
-- Single column index
INDEX (OrderNumber)
-- Composite index with sort direction
INDEX (CustomerId, OrderDate DESC)
-- Another single column index
INDEX (Status);
```

## Sort Direction

Each column in an index can specify a sort direction:

| Direction | Keyword | Default |
|-----------|---------|---------|
| Ascending | `ASC` | Yes (default) |
| Descending | `DESC` | No |

```sql
INDEX (Name)                    -- ascending (default)
INDEX (Name ASC)                -- explicit ascending
INDEX (CreatedAt DESC)          -- descending
INDEX (Name, CreatedAt DESC)    -- mixed: Name ascending, CreatedAt descending
```

## Adding and Removing Indexes

Use `ALTER ENTITY` to add or remove indexes on existing entities:

```sql
-- Add an index
ALTER ENTITY Sales.Customer
  ADD INDEX (Email);

-- Remove an index
ALTER ENTITY Sales.Customer
  DROP INDEX (Email);
```

## Guidelines

1. **Primary lookups** -- index columns used in WHERE clauses
2. **Foreign keys** -- index columns used in association joins
3. **Sorting** -- index columns used in ORDER BY clauses
4. **Composite order** -- put high-selectivity columns first in composite indexes

## See Also

- [Entities](./entities.md) -- entity definitions that contain indexes
- [ALTER ENTITY](./alter-entity.md) -- adding/removing indexes on existing entities

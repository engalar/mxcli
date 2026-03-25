# Entities

Entities are the primary data structures in a Mendix domain model. Each entity corresponds to a database table (for persistent entities) or an in-memory object (for non-persistent entities).

## Entity Types

| Type | MDL Keyword | Description |
|------|-------------|-------------|
| Persistent | `PERSISTENT` | Stored in the database with a corresponding table |
| Non-Persistent | `NON-PERSISTENT` | In-memory only, scoped to the user session |
| View | `VIEW` | Based on an OQL query, read-only |
| External | `EXTERNAL` | From an external data source (OData, etc.) |

## CREATE ENTITY

```sql
[/** <documentation> */]
[@Position(<x>, <y>)]
CREATE [OR MODIFY] <entity-type> ENTITY <Module>.<Name> (
  <attribute-definitions>
)
[INDEX (<column-list>)]
[;|/]
```

### Persistent Entity

The most common type. Data is stored in the application database:

```sql
/** Customer entity */
@Position(100, 200)
CREATE PERSISTENT ENTITY Sales.Customer (
  CustomerId: AutoNumber NOT NULL UNIQUE DEFAULT 1,
  Name: String(200) NOT NULL ERROR 'Name is required',
  Email: String(200) UNIQUE ERROR 'Email must be unique',
  Balance: Decimal DEFAULT 0,
  IsActive: Boolean DEFAULT TRUE,
  CreatedDate: DateTime,
  Status: Enumeration(Sales.CustomerStatus) DEFAULT 'Active'
)
INDEX (Name)
INDEX (Email);
```

### Non-Persistent Entity

Used for helper objects, filter parameters, and UI state that does not need database storage:

```sql
CREATE NON-PERSISTENT ENTITY Sales.CustomerFilter (
  SearchName: String(200),
  MinBalance: Decimal,
  MaxBalance: Decimal
);
```

### View Entity

Defined by an OQL query. View entities are read-only:

```sql
CREATE VIEW ENTITY Reports.CustomerSummary (
  CustomerName: String,
  TotalOrders: Integer,
  TotalAmount: Decimal
) AS
  SELECT
    c.Name AS CustomerName,
    COUNT(o.OrderId) AS TotalOrders,
    SUM(o.Amount) AS TotalAmount
  FROM Sales.Customer c
  LEFT JOIN Sales.Order o ON o.Customer = c
  GROUP BY c.Name;
```

## CREATE OR MODIFY

Creates the entity if it does not exist, or updates it if it does. New attributes are added; existing attributes are preserved:

```sql
CREATE OR MODIFY PERSISTENT ENTITY Sales.Customer (
  CustomerId: AutoNumber NOT NULL UNIQUE,
  Name: String(200) NOT NULL,
  Email: String(200),
  Phone: String(50)  -- new attribute added on modify
);
```

## Annotations

### @Position

Controls where the entity appears in the domain model diagram:

```sql
@Position(100, 200)
CREATE PERSISTENT ENTITY Sales.Customer ( ... );
```

### Documentation

A `/** ... */` comment before the entity becomes its documentation in Studio Pro:

```sql
/** Customer master data.
 *  Stores both active and inactive customers.
 */
CREATE PERSISTENT ENTITY Sales.Customer ( ... );
```

## DROP ENTITY

Removes an entity from the domain model:

```sql
DROP ENTITY Sales.Customer;
```

## See Also

- [Attributes](./attributes.md) -- attribute definitions within entities
- [Indexes](./indexes.md) -- adding indexes to entities
- [Generalization](./generalization.md) -- entity inheritance with EXTENDS
- [ALTER ENTITY](./alter-entity.md) -- modifying existing entities
- [Associations](./associations.md) -- relationships between entities

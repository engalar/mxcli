# Domain Model

The domain model is the data layer of a Mendix application. It defines the entities (data tables), their attributes (columns), relationships between entities (associations), and inheritance hierarchies (generalizations).

## Core Concepts

| Concept | MDL Statement | Description |
|---------|---------------|-------------|
| **Entity** | `CREATE ENTITY` | A data structure, similar to a database table |
| **Attribute** | Defined inside entity | A field on an entity, similar to a column |
| **Association** | `CREATE ASSOCIATION` | A relationship between two entities |
| **Generalization** | `EXTENDS` | Inheritance -- one entity extends another |
| **Index** | `INDEX (...)` | Performance optimization for queries |
| **Enumeration** | `CREATE ENUMERATION` | A fixed set of named values for an attribute |

## Module Scope

Every domain model element belongs to a module. Elements are always referenced by their [qualified name](./qualified-names.md) in `Module.Element` format:

```sql
Sales.Customer           -- entity
Sales.OrderStatus        -- enumeration
Sales.Order_Customer     -- association
```

## Complete Example

This example creates a small Sales domain model with related entities, an enumeration, and an association:

```sql
-- Enumeration for order statuses
CREATE ENUMERATION Sales.OrderStatus (
  Draft 'Draft',
  Pending 'Pending',
  Confirmed 'Confirmed',
  Shipped 'Shipped',
  Delivered 'Delivered',
  Cancelled 'Cancelled'
);

-- Customer entity
/** Customer master data */
@Position(100, 100)
CREATE PERSISTENT ENTITY Sales.Customer (
  CustomerId: AutoNumber NOT NULL UNIQUE DEFAULT 1,
  Name: String(200) NOT NULL ERROR 'Customer name is required',
  Email: String(200) UNIQUE ERROR 'Email already registered',
  Phone: String(50),
  IsActive: Boolean DEFAULT TRUE,
  CreatedAt: DateTime
)
INDEX (Name)
INDEX (Email);

-- Order entity
/** Sales order */
@Position(300, 100)
CREATE PERSISTENT ENTITY Sales.Order (
  OrderId: AutoNumber NOT NULL UNIQUE DEFAULT 1,
  OrderNumber: String(50) NOT NULL UNIQUE,
  OrderDate: DateTime NOT NULL,
  TotalAmount: Decimal DEFAULT 0,
  Status: Enumeration(Sales.OrderStatus) DEFAULT 'Draft',
  Notes: String(unlimited)
)
INDEX (OrderNumber)
INDEX (OrderDate DESC);

-- Association: Order belongs to Customer
CREATE ASSOCIATION Sales.Order_Customer
  FROM Sales.Customer
  TO Sales.Order
  TYPE Reference
  OWNER Default
  DELETE_BEHAVIOR DELETE_BUT_KEEP_REFERENCES;
```

## Further Reading

- [Entities](./entities.md) -- entity types and CREATE ENTITY syntax
- [Attributes](./attributes.md) -- attribute definitions and validation
- [Associations](./associations.md) -- relationships between entities
- [Generalization](./generalization.md) -- entity inheritance
- [Indexes](./indexes.md) -- index creation
- [Enumerations](./enumerations.md) -- enumeration types
- [ALTER ENTITY](./alter-entity.md) -- modifying existing entities

# Associations

Associations define relationships between entities. They determine how objects reference each other and control behavior on deletion.

## Association Types

| Type | MDL Keyword | Cardinality | Description |
|------|-------------|-------------|-------------|
| Reference | `Reference` | Many-to-One | Many objects on the FROM side reference one object on the TO side |
| ReferenceSet | `ReferenceSet` | Many-to-Many | Objects on both sides can reference multiple objects |

## CREATE ASSOCIATION

```sql
[/** <documentation> */]
CREATE ASSOCIATION <Module>.<Name>
  FROM <ParentEntity>
  TO <ChildEntity>
  TYPE <Reference|ReferenceSet>
  [OWNER <Default|Both|Parent|Child>]
  [DELETE_BEHAVIOR <behavior>]
```

### Reference (Many-to-One)

A `Reference` association means each object on the FROM side can reference one object on the TO side. Multiple FROM objects can reference the same TO object.

```sql
/** Order belongs to Customer (many-to-one) */
CREATE ASSOCIATION Sales.Order_Customer
  FROM Sales.Customer
  TO Sales.Order
  TYPE Reference
  OWNER Default
  DELETE_BEHAVIOR DELETE_BUT_KEEP_REFERENCES;
```

### ReferenceSet (Many-to-Many)

A `ReferenceSet` association allows multiple objects on both sides to reference each other:

```sql
/** Order has many Products (many-to-many) */
CREATE ASSOCIATION Sales.Order_Product
  FROM Sales.Order
  TO Sales.Product
  TYPE ReferenceSet
  OWNER Both;
```

## Owner Options

The owner determines which side of the association can modify the reference.

| Owner | Description |
|-------|-------------|
| `Default` | Child (FROM side) owns the association |
| `Both` | Both sides can modify the association |
| `Parent` | Only parent (TO side) can modify |
| `Child` | Only child (FROM side) can modify |

## Delete Behavior

Controls what happens when an associated object is deleted.

| Behavior | MDL Keyword | Description |
|----------|-------------|-------------|
| Keep references | `DELETE_BUT_KEEP_REFERENCES` | Delete the object, set references to null |
| Cascade | `DELETE_CASCADE` | Delete associated objects as well |

```sql
/** Invoice must be deleted with Order */
CREATE ASSOCIATION Sales.Order_Invoice
  FROM Sales.Order
  TO Sales.Invoice
  TYPE Reference
  DELETE_BEHAVIOR DELETE_CASCADE;
```

## Naming Convention

Association names typically follow the pattern `Module.FromEntity_ToEntity`:

```sql
Sales.Order_Customer       -- Order references Customer
Sales.Order_Product        -- Order references Product
Sales.Order_Invoice        -- Order references Invoice
```

## DROP ASSOCIATION

Remove an association:

```sql
DROP ASSOCIATION Sales.Order_Customer;
```

## See Also

- [Domain Model](./domain-model.md) -- overview of domain model concepts
- [Entities](./entities.md) -- the entities that associations connect
- [Generalization](./generalization.md) -- inheritance (a different kind of relationship)

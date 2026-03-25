# Enumerations

Enumerations define a fixed set of named values. They are used as attribute types to restrict a field to a specific list of options.

## CREATE ENUMERATION

```sql
[/** <documentation> */]
CREATE ENUMERATION <Module>.<Name> (
  <value-name> '<caption>' [, ...]
)
```

Each value has an identifier (used in code) and a caption (displayed in the UI):

```sql
/** Order status enumeration */
CREATE ENUMERATION Sales.OrderStatus (
  Draft 'Draft',
  Pending 'Pending Approval',
  Approved 'Approved',
  Shipped 'Shipped',
  Delivered 'Delivered',
  Cancelled 'Cancelled'
);
```

## Using Enumerations in Attributes

Reference an enumeration type in an attribute definition with `Enumeration(Module.Name)`:

```sql
CREATE PERSISTENT ENTITY Sales.Order (
  Status: Enumeration(Sales.OrderStatus) DEFAULT 'Draft',
  Priority: Enumeration(Core.Priority) NOT NULL
);
```

The default value is the **name** of the enumeration value (not the caption). The fully qualified form is preferred:

```sql
-- Fully qualified (preferred)
Status: Enumeration(Sales.OrderStatus) DEFAULT Sales.OrderStatus.Draft

-- Legacy string literal form
Status: Enumeration(Sales.OrderStatus) DEFAULT 'Draft'
```

## ALTER ENUMERATION

Add or remove values from an existing enumeration:

```sql
-- Add a new value
ALTER ENUMERATION Sales.OrderStatus
  ADD VALUE OnHold 'On Hold';

-- Remove a value
ALTER ENUMERATION Sales.OrderStatus
  REMOVE VALUE Draft;
```

## DROP ENUMERATION

Remove an enumeration entirely:

```sql
DROP ENUMERATION Sales.OrderStatus;
```

Dropping an enumeration that is still referenced by entity attributes will cause errors in Studio Pro.

## See Also

- [Primitive Types](./primitive-types.md) -- all attribute types including enumeration references
- [Data Types](./data-types.md) -- type system overview
- [Entities](./entities.md) -- using enumerations in entity definitions

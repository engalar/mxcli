# Data Types

MDL has a type system that maps to the Mendix metamodel attribute types. Every attribute in an entity definition has a type that determines how values are stored and validated.

## Type Categories

| Category | Types |
|----------|-------|
| **Text** | `String`, `String(n)`, `HashedString` |
| **Numeric** | `Integer`, `Long`, `Decimal`, `AutoNumber` |
| **Logical** | `Boolean` |
| **Temporal** | `DateTime`, `Date` |
| **Binary** | `Binary` |
| **Reference** | `Enumeration(Module.Name)` |

## Quick Example

```sql
CREATE PERSISTENT ENTITY Demo.AllTypes (
  Id: AutoNumber NOT NULL UNIQUE DEFAULT 1,
  Code: String(10) NOT NULL UNIQUE,
  Name: String(200) NOT NULL,
  Description: String(unlimited),
  Counter: Integer DEFAULT 0,
  BigNumber: Long,
  Amount: Decimal DEFAULT 0.00,
  IsActive: Boolean DEFAULT TRUE,
  CreatedAt: DateTime,
  BirthDate: Date,
  Attachment: Binary,
  Status: Enumeration(Demo.Status) DEFAULT 'Active'
);
```

## No Implicit Conversions

MDL does not perform implicit type conversions. Types must match exactly when assigning defaults or modifying attributes.

Compatible type changes (when using `CREATE OR MODIFY`):

- `String(100)` to `String(200)` -- increasing length
- `Integer` to `Long` -- widening numeric type

Incompatible type changes:

- `String` to `Integer`
- `Boolean` to `String`
- Any type to `AutoNumber` (if data exists)

## Further Reading

- [Primitive Types](./primitive-types.md) -- detailed reference for each type
- [Constraints](./constraints.md) -- NOT NULL, UNIQUE, DEFAULT
- [Type Mapping](./type-mapping.md) -- MDL to Mendix to database mapping
- [Enumerations](./enumerations.md) -- enumeration type definitions

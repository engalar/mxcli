# Constraints

Constraints restrict the values an attribute can hold. They are specified after the type in an attribute definition.

## Syntax

```sql
<name>: <type> [NOT NULL [ERROR '<message>']] [UNIQUE [ERROR '<message>']] [DEFAULT <value>]
```

Constraints must appear in this order:

1. `NOT NULL` (optionally with `ERROR`)
2. `UNIQUE` (optionally with `ERROR`)
3. `DEFAULT`

## NOT NULL

Marks the attribute as required. The value cannot be empty.

```sql
Name: String(200) NOT NULL
```

With a custom error message displayed to the user:

```sql
Name: String(200) NOT NULL ERROR 'Name is required'
```

## UNIQUE

Enforces that the value must be unique across all objects of the entity.

```sql
Email: String(200) UNIQUE
```

With a custom error message:

```sql
Email: String(200) UNIQUE ERROR 'Email already exists'
```

## DEFAULT

Sets the initial value when a new object is created.

```sql
Count: Integer DEFAULT 0
IsActive: Boolean DEFAULT TRUE
Status: Enumeration(Sales.OrderStatus) DEFAULT 'Draft'
Name: String(200) DEFAULT 'Unknown'
```

Default value syntax varies by type:

| Type | Default Syntax | Examples |
|------|----------------|----------|
| String | `DEFAULT 'value'` | `DEFAULT ''`, `DEFAULT 'Unknown'` |
| Integer | `DEFAULT n` | `DEFAULT 0`, `DEFAULT -1` |
| Long | `DEFAULT n` | `DEFAULT 0` |
| Decimal | `DEFAULT n.n` | `DEFAULT 0`, `DEFAULT 0.00`, `DEFAULT 99.99` |
| Boolean | `DEFAULT TRUE/FALSE` | `DEFAULT TRUE`, `DEFAULT FALSE` |
| AutoNumber | `DEFAULT n` | `DEFAULT 1` (starting value) |
| Enumeration | `DEFAULT Module.Enum.Value` | `DEFAULT Shop.Status.Active`, `DEFAULT 'Pending'` |

Omitting the `DEFAULT` clause means no default value is set:

```sql
OptionalField: String(200)
```

## Combining Constraints

All three constraints can be used together:

```sql
Email: String(200) NOT NULL ERROR 'Email is required'
                   UNIQUE ERROR 'Email already exists'
                   DEFAULT ''
```

More examples:

```sql
CREATE PERSISTENT ENTITY Sales.Product (
  -- Required only
  Name: String(200) NOT NULL,

  -- Required with custom error
  SKU: String(50) NOT NULL ERROR 'SKU is required for all products',

  -- Unique only
  Barcode: String(50) UNIQUE,

  -- Required and unique with custom errors
  ProductCode: String(20) NOT NULL ERROR 'Product code required'
                          UNIQUE ERROR 'Product code must be unique',

  -- Default only
  Quantity: Integer DEFAULT 0,

  -- No constraints
  Description: String(unlimited)
);
```

## Validation Rule Mapping

Under the hood, constraints map to Mendix validation rules:

| MDL Constraint | Mendix Validation Rule |
|----------------|----------------------|
| `NOT NULL` | `DomainModels$RequiredRuleInfo` |
| `UNIQUE` | `DomainModels$UniqueRuleInfo` |

## See Also

- [Primitive Types](./primitive-types.md) -- type reference
- [Attributes](./attributes.md) -- full attribute definition syntax
- [ALTER ENTITY](./alter-entity.md) -- modifying constraints on existing entities

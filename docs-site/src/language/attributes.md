# Attributes and Validation Rules

Attributes define the fields on an entity. Each attribute has a name, a type, and optional constraints.

## Attribute Syntax

```sql
[/** <documentation> */]
<name>: <type> [NOT NULL [ERROR '<message>']] [UNIQUE [ERROR '<message>']] [DEFAULT <value>]
```

| Component | Description |
|-----------|-------------|
| Name | Attribute identifier (follows [identifier rules](./qualified-names.md)) |
| Type | One of the [primitive types](./primitive-types.md) or an [enumeration](./enumerations.md) reference |
| `NOT NULL` | Value is required |
| `UNIQUE` | Value must be unique across all objects |
| `DEFAULT` | Initial value on object creation |
| `/** ... */` | Documentation comment |

## Attribute Ordering

Attributes are listed in order, separated by commas. The last attribute has no trailing comma:

```sql
CREATE PERSISTENT ENTITY Module.Entity (
  FirstAttr: String(200),      -- comma after
  SecondAttr: Integer,         -- comma after
  LastAttr: Boolean DEFAULT FALSE  -- no comma
);
```

## Validation Rules

Validation rules are expressed as attribute constraints. When validation fails, Mendix displays the error message (if provided) or a default message.

| Validation | MDL Syntax | Description |
|------------|------------|-------------|
| Required | `NOT NULL` | Attribute must have a value |
| Required with message | `NOT NULL ERROR 'message'` | Custom error message on empty |
| Unique | `UNIQUE` | Value must be unique across all objects |
| Unique with message | `UNIQUE ERROR 'message'` | Custom error message on duplicate |

### Example with Validation

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

  -- Optional field (no validation)
  Description: String(unlimited)
);
```

## Documentation Comments

Attach documentation to individual attributes with `/** ... */`:

```sql
CREATE PERSISTENT ENTITY Sales.Customer (
  /** Unique customer identifier, auto-generated */
  CustomerId: AutoNumber NOT NULL UNIQUE DEFAULT 1,

  /** Full legal name of the customer */
  Name: String(200) NOT NULL,

  /** Primary contact email address */
  Email: String(200) UNIQUE
);
```

## CALCULATED Attributes

An attribute can be marked as calculated, meaning its value is computed by a microflow rather than stored:

```sql
FullName: String(400) CALCULATED
```

Calculated attributes use a `CALCULATED BY` clause to specify the source microflow. See the MDL quick reference for details.

## See Also

- [Primitive Types](./primitive-types.md) -- all available attribute types
- [Constraints](./constraints.md) -- NOT NULL, UNIQUE, DEFAULT in detail
- [Entities](./entities.md) -- entity definitions that contain attributes
- [ALTER ENTITY](./alter-entity.md) -- adding and modifying attributes on existing entities

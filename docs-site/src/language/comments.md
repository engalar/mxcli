# Comments and Documentation

MDL supports three comment styles for annotating scripts and attaching documentation to model elements.

## Single-Line Comments

Use `--` (SQL style) or `//` (C style) for single-line comments:

```sql
-- This is a single-line comment
SHOW MODULES

// This is also a single-line comment
SHOW ENTITIES IN Sales
```

Everything after the comment marker to the end of the line is ignored.

## Multi-Line Comments

Use `/* ... */` for comments that span multiple lines:

```sql
/* This comment spans
   multiple lines and is useful
   for longer explanations */
CREATE PERSISTENT ENTITY Sales.Customer (
  Name: String(200) NOT NULL
);
```

## Documentation Comments

Use `/** ... */` to attach documentation to model elements. Documentation comments are stored in the Mendix model and appear in Studio Pro:

```sql
/** Customer entity stores master customer data.
 *  Used by the Sales and Support modules.
 */
@Position(100, 200)
CREATE PERSISTENT ENTITY Sales.Customer (
  /** Unique customer identifier, auto-generated */
  CustomerId: AutoNumber NOT NULL UNIQUE DEFAULT 1,

  /** Full legal name of the customer */
  Name: String(200) NOT NULL,

  /** Primary contact email address */
  Email: String(200) UNIQUE
);
```

Documentation comments can be placed before:

- Entity definitions (becomes entity documentation)
- Attribute definitions (becomes attribute documentation)
- Enumeration definitions (becomes enumeration documentation)
- Association definitions (becomes association documentation)

### Updating Documentation

You can also set documentation on existing entities with `ALTER ENTITY`:

```sql
ALTER ENTITY Sales.Customer
  SET DOCUMENTATION 'Customer master data for the Sales module';
```

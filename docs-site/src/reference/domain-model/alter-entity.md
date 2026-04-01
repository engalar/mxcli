# ALTER ENTITY

## Synopsis

    ALTER ENTITY module.name ADD ( attribute_definition [, ...] )

    ALTER ENTITY module.name DROP ( attribute_name [, ...] )

    ALTER ENTITY module.name MODIFY ( attribute_definition [, ...] )

    ALTER ENTITY module.name RENAME old_name TO new_name

    ALTER ENTITY module.name ADD INDEX ( column [ ASC | DESC ] [, ...] )

    ALTER ENTITY module.name DROP INDEX ( column [, ...] )

    ALTER ENTITY module.name SET DOCUMENTATION 'text'

## Description

`ALTER ENTITY` modifies an existing entity without replacing it entirely. This is useful for incremental changes to a domain model -- adding a new attribute, removing an obsolete one, renaming for clarity, or managing indexes.

Each `ALTER ENTITY` statement performs exactly one operation. To make multiple changes, issue multiple statements.

The `ADD` operation appends one or more new attributes to the entity. Attribute definitions follow the same syntax as in `CREATE ENTITY`: a name, a data type, and optional constraints (`NOT NULL`, `UNIQUE`, `DEFAULT`).

The `DROP` operation removes one or more attributes by name. Dropping an attribute also removes any validation rules and index entries that reference it.

The `MODIFY` operation changes the type or constraints of existing attributes. The attribute name must already exist in the entity. The full attribute definition (type and constraints) replaces the current one.

The `RENAME` operation changes an attribute's name. This updates references within the entity but does not automatically update microflows, pages, or access rules that reference the old name.

The `ADD INDEX` and `DROP INDEX` operations manage database indexes. Index columns may include an optional `ASC` or `DESC` sort direction.

The `SET DOCUMENTATION` operation replaces the entity's documentation string.

## Parameters

**module.name**
: The qualified name of the entity to modify, in the form `Module.EntityName`.

**attribute_definition**
: An attribute name followed by a colon, data type, and optional constraints. Same syntax as in `CREATE ENTITY`.

**attribute_name**
: The name of an existing attribute to drop.

**old_name**
: The current name of the attribute to rename.

**new_name**
: The new name for the attribute.

**column**
: An attribute name for the index, optionally followed by `ASC` or `DESC`.

**text**
: A single-quoted string to use as the entity's documentation.

## Examples

### Add new attributes

```sql
ALTER ENTITY Sales.Customer
    ADD (Phone: String(50), Notes: String(unlimited));
```

### Drop an attribute

```sql
ALTER ENTITY Sales.Customer
    DROP (Notes);
```

### Modify an attribute's type

```sql
ALTER ENTITY Sales.Customer
    MODIFY (Phone: String(100) NOT NULL);
```

### Rename an attribute

```sql
ALTER ENTITY Sales.Customer
    RENAME Phone TO PhoneNumber;
```

### Add an index

```sql
ALTER ENTITY Sales.Customer
    ADD INDEX (Email);
```

### Add a composite index with sort direction

```sql
ALTER ENTITY Sales.Order
    ADD INDEX (CustomerId, OrderDate DESC);
```

### Drop an index

```sql
ALTER ENTITY Sales.Customer
    DROP INDEX (Email);
```

### Set documentation

```sql
ALTER ENTITY Sales.Customer
    SET DOCUMENTATION 'Customer master data for the sales module.';
```

### Set position

Reposition an entity on the domain model canvas:

```sql
ALTER ENTITY Sales.Customer
    SET POSITION (100, 200);
```

## Notes

- Each `ALTER ENTITY` statement performs a single operation. Chain multiple statements for multiple changes.
- `RENAME` does not update references in microflows, pages, or access rules. Update those separately or use `SHOW IMPACT OF` to find affected elements.
- `DROP` removes the attribute's validation rules and index entries automatically.

## See Also

[CREATE ENTITY](create-entity.md), [DROP ENTITY](drop-entity.md)

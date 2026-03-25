# Entity Access

Entity access rules control which module roles can create, read, write, and delete objects of a given entity. Rules can restrict access to specific attributes and apply XPath constraints to limit which rows are visible.

## GRANT on Entities

```sql
GRANT <Module>.<Role> ON <Module>.<Entity> (<rights>) [WHERE '<xpath>'];
```

The `<rights>` list is a comma-separated combination of:

| Right | Syntax | Description |
|-------|--------|-------------|
| Create | `CREATE` | Allow creating new objects |
| Delete | `DELETE` | Allow deleting objects |
| Read all | `READ *` | Read access to all attributes and associations |
| Read specific | `READ (<attr>, ...)` | Read access to listed members only |
| Write all | `WRITE *` | Write access to all attributes and associations |
| Write specific | `WRITE (<attr>, ...)` | Write access to listed members only |

## Examples

### Full Access

Grant all operations on all members:

```sql
GRANT Shop.Admin ON Shop.Customer (CREATE, DELETE, READ *, WRITE *);
```

### Read-Only Access

```sql
GRANT Shop.Viewer ON Shop.Customer (READ *);
```

### Selective Member Access

Restrict read and write to specific attributes:

```sql
GRANT Shop.User ON Shop.Customer (READ (Name, Email, Status), WRITE (Email));
```

### XPath Constraints

Limit which objects a role can see or modify using an XPath expression in the `WHERE` clause:

```sql
-- Users can only access their own orders
GRANT Shop.User ON Shop.Order (READ *, WRITE *)
  WHERE '[Sales.Order_Customer/Sales.Customer/Name = $currentUser]';

-- Only open orders are editable
GRANT Shop.User ON Shop.Order (READ *, WRITE *)
  WHERE '[Status = ''Open'']';
```

Note that single quotes inside XPath expressions must be doubled (`''`), since the entire expression is wrapped in single quotes.

### Multiple Roles on the Same Entity

Each GRANT creates a separate access rule. An entity can have rules for multiple roles:

```sql
GRANT Shop.Admin ON Shop.Order (CREATE, DELETE, READ *, WRITE *);
GRANT Shop.User ON Shop.Order (READ *, WRITE *) WHERE '[Status = ''Open'']';
GRANT Shop.Viewer ON Shop.Order (READ *);
```

## REVOKE on Entities

Remove an entity access rule for a role:

```sql
REVOKE <Module>.<Role> ON <Module>.<Entity>;
```

Example:

```sql
REVOKE Shop.Viewer ON Shop.Customer;
```

This removes the entire access rule for that role on that entity.

## Viewing Entity Access

```sql
-- See which roles have access to an entity
SHOW ACCESS ON Shop.Customer;

-- Full matrix across a module
SHOW SECURITY MATRIX IN Shop;
```

## See Also

- [Security](./security.md) -- overview of the security model
- [Module Roles and User Roles](./roles.md) -- defining the roles referenced in access rules
- [Document Access](./document-access.md) -- microflow and page access
- [GRANT / REVOKE](./grant-revoke.md) -- complete GRANT and REVOKE reference

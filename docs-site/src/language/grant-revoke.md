# GRANT / REVOKE

The `GRANT` and `REVOKE` statements control all permissions in a Mendix project. They work on three targets: entities (CRUD access), microflows (execute access), and pages (view access).

## Entity Access

### GRANT

```sql
GRANT <Module>.<Role> ON <Module>.<Entity> (<rights>) [WHERE '<xpath>'];
```

Where `<rights>` is a comma-separated list of:

| Right | Description |
|-------|-------------|
| `CREATE` | Allow creating new objects |
| `DELETE` | Allow deleting objects |
| `READ *` | Read all members |
| `READ (<attr>, ...)` | Read specific members only |
| `WRITE *` | Write all members |
| `WRITE (<attr>, ...)` | Write specific members only |

Examples:

```sql
-- Full access
GRANT Shop.Admin ON Shop.Customer (CREATE, DELETE, READ *, WRITE *);

-- Read-only
GRANT Shop.Viewer ON Shop.Customer (READ *);

-- Selective members
GRANT Shop.User ON Shop.Customer (READ (Name, Email), WRITE (Email));

-- With XPath constraint (doubled single quotes for string literals)
GRANT Shop.User ON Shop.Order (READ *, WRITE *)
  WHERE '[Status = ''Open'']';
```

### REVOKE

Remove an entity access rule entirely:

```sql
REVOKE <Module>.<Role> ON <Module>.<Entity>;
```

Example:

```sql
REVOKE Shop.Viewer ON Shop.Customer;
```

## Microflow Access

### GRANT EXECUTE ON MICROFLOW

```sql
GRANT EXECUTE ON MICROFLOW <Module>.<Name> TO <Module>.<Role> [, ...];
```

Example:

```sql
GRANT EXECUTE ON MICROFLOW Shop.ACT_ProcessOrder TO Shop.User, Shop.Admin;
```

### REVOKE EXECUTE ON MICROFLOW

```sql
REVOKE EXECUTE ON MICROFLOW <Module>.<Name> FROM <Module>.<Role> [, ...];
```

Example:

```sql
REVOKE EXECUTE ON MICROFLOW Shop.ACT_ProcessOrder FROM Shop.User;
```

## Page Access

### GRANT VIEW ON PAGE

```sql
GRANT VIEW ON PAGE <Module>.<Name> TO <Module>.<Role> [, ...];
```

Example:

```sql
GRANT VIEW ON PAGE Shop.Order_Overview TO Shop.User, Shop.Admin;
```

### REVOKE VIEW ON PAGE

```sql
REVOKE VIEW ON PAGE <Module>.<Name> FROM <Module>.<Role> [, ...];
```

Example:

```sql
REVOKE VIEW ON PAGE Shop.Admin_Dashboard FROM Shop.User;
```

## Nanoflow Access

```sql
GRANT EXECUTE ON NANOFLOW <Module>.<Name> TO <Module>.<Role> [, ...];
REVOKE EXECUTE ON NANOFLOW <Module>.<Name> FROM <Module>.<Role> [, ...];
```

## Workflow Access

```sql
GRANT EXECUTE ON WORKFLOW <Module>.<Name> TO <Module>.<Role> [, ...];
REVOKE EXECUTE ON WORKFLOW <Module>.<Name> FROM <Module>.<Role> [, ...];
```

## OData Service Access

```sql
GRANT ACCESS ON ODATA SERVICE <Module>.<Name> TO <Module>.<Role> [, ...];
REVOKE ACCESS ON ODATA SERVICE <Module>.<Name> FROM <Module>.<Role> [, ...];
```

## Complete Example

A typical security setup script:

```sql
-- Module roles
CREATE MODULE ROLE Shop.Admin DESCRIPTION 'Full access';
CREATE MODULE ROLE Shop.User DESCRIPTION 'Standard access';
CREATE MODULE ROLE Shop.Viewer DESCRIPTION 'Read-only access';

-- User roles
CREATE USER ROLE Administrator (Shop.Admin, System.Administrator) MANAGE ALL ROLES;
CREATE USER ROLE Employee (Shop.User);
CREATE USER ROLE Guest (Shop.Viewer);

-- Entity access
GRANT Shop.Admin ON Shop.Customer (CREATE, DELETE, READ *, WRITE *);
GRANT Shop.User ON Shop.Customer (READ *, WRITE (Email, Phone));
GRANT Shop.Viewer ON Shop.Customer (READ *);

GRANT Shop.Admin ON Shop.Order (CREATE, DELETE, READ *, WRITE *);
GRANT Shop.User ON Shop.Order (CREATE, READ *, WRITE *)
  WHERE '[Status = ''Open'']';
GRANT Shop.Viewer ON Shop.Order (READ *);

-- Microflow access
GRANT EXECUTE ON MICROFLOW Shop.ACT_ProcessOrder TO Shop.Admin;
GRANT EXECUTE ON MICROFLOW Shop.ACT_CreateOrder TO Shop.User, Shop.Admin;
GRANT EXECUTE ON MICROFLOW Shop.ACT_ViewOrders TO Shop.User, Shop.Admin, Shop.Viewer;

-- Page access
GRANT VIEW ON PAGE Shop.Order_Overview TO Shop.User, Shop.Admin, Shop.Viewer;
GRANT VIEW ON PAGE Shop.Order_Edit TO Shop.User, Shop.Admin;
GRANT VIEW ON PAGE Shop.Admin_Dashboard TO Shop.Admin;

-- Demo users
CREATE DEMO USER 'demo_admin' PASSWORD 'Admin123!' (Administrator);
CREATE DEMO USER 'demo_user' PASSWORD 'User123!' (Employee);

-- Enable demo users
ALTER PROJECT SECURITY DEMO USERS ON;
ALTER PROJECT SECURITY LEVEL PROTOTYPE;
```

## See Also

- [Security](./security.md) -- overview of the security model
- [Entity Access](./entity-access.md) -- details on entity CRUD permissions and XPath constraints
- [Document Access](./document-access.md) -- microflow, page, and nanoflow access patterns
- [Module Roles and User Roles](./roles.md) -- creating and managing roles
- [Demo Users](./demo-users.md) -- creating test accounts

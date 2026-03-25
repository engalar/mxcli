# Security

Mendix applications use a layered security model with three levels: project security settings, role definitions, and access rules. MDL provides complete control over all three layers.

## Security Levels

The project security level determines how strictly the runtime enforces access rules.

| Level | MDL Keyword | Description |
|-------|-------------|-------------|
| Off | `OFF` | No security enforcement (development only) |
| Prototype | `PROTOTYPE` | Security enforced but incomplete configurations allowed |
| Production | `PRODUCTION` | Full enforcement, all access rules must be complete |

```sql
ALTER PROJECT SECURITY LEVEL PRODUCTION;
```

## Security Architecture

Security in Mendix is organized in layers:

1. **Module Roles** -- defined per module, they represent permissions within that module (e.g., `Shop.Admin`, `Shop.Viewer`)
2. **User Roles** -- project-level roles that aggregate module roles across modules (e.g., `AppAdmin` combines `Shop.Admin` and `System.Administrator`)
3. **Entity Access** -- CRUD permissions and XPath constraints per entity per module role
4. **Document Access** -- execute/view permissions on microflows, nanoflows, and pages per module role
5. **Demo Users** -- test accounts with assigned user roles for development

## Inspecting Security

MDL provides several commands for viewing the current security configuration:

```sql
-- Project-wide settings
SHOW PROJECT SECURITY;

-- Roles
SHOW MODULE ROLES;
SHOW MODULE ROLES IN Shop;
SHOW USER ROLES;

-- Access rules
SHOW ACCESS ON MICROFLOW Shop.ACT_ProcessOrder;
SHOW ACCESS ON PAGE Shop.Order_Edit;
SHOW ACCESS ON Shop.Customer;

-- Full matrix
SHOW SECURITY MATRIX;
SHOW SECURITY MATRIX IN Shop;

-- Demo users
SHOW DEMO USERS;
```

## Modifying Project Security

Toggle the security level and demo user visibility:

```sql
ALTER PROJECT SECURITY LEVEL PRODUCTION;
ALTER PROJECT SECURITY DEMO USERS ON;
ALTER PROJECT SECURITY DEMO USERS OFF;
```

## See Also

- [Module Roles and User Roles](./roles.md) -- defining roles
- [Entity Access](./entity-access.md) -- CRUD permissions per entity
- [Document Access](./document-access.md) -- microflow, page, and nanoflow permissions
- [GRANT / REVOKE](./grant-revoke.md) -- granting and revoking permissions
- [Demo Users](./demo-users.md) -- creating test accounts

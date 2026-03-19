# Security Management Skill

This skill covers Mendix security configuration via MDL: module roles, user roles, access control (microflows, pages, entities), project security settings, and demo users.

## When to Use This Skill

Use when the user asks to:
- Set up security for a module or project
- Create or manage module roles / user roles
- Grant or revoke access to microflows, pages, or entities
- Configure project security level or demo users
- Review existing security configuration

## Security Concepts

- **Module Roles** define permissions within a single module (e.g., `Shop.Admin`, `Shop.Viewer`)
- **User Roles** aggregate module roles from multiple modules (e.g., `Administrator` includes `Shop.Admin` + `System.Administrator`)
- **Access Rules** control CRUD rights on entities per module role
- **Microflow/Page Access** controls which module roles can execute/view specific elements
- **Project Security Level** determines enforcement: `OFF`, `PROTOTYPE`, or `PRODUCTION`

## Syntax Reference

### Show Commands (Read-Only)

```sql
-- Project-wide security overview
SHOW PROJECT SECURITY;

-- Module roles (all or filtered)
SHOW MODULE ROLES;
SHOW MODULE ROLES IN MyModule;

-- User roles and demo users
SHOW USER ROLES;
SHOW DEMO USERS;

-- Access on specific elements
SHOW ACCESS ON MICROFLOW MyModule.ProcessOrder;
SHOW ACCESS ON PAGE MyModule.CustomerOverview;
SHOW ACCESS ON MyModule.Customer;

-- Full security matrix
SHOW SECURITY MATRIX;
SHOW SECURITY MATRIX IN MyModule;
```

### Describe Commands

```sql
-- Describe individual roles and users (MDL output)
DESCRIBE MODULE ROLE MyModule.Admin;
DESCRIBE USER ROLE Administrator;
DESCRIBE DEMO USER 'demo_admin';
```

### Catalog Queries (SQL)

Security data is available in catalog tables for advanced querying. Use `REFRESH CATALOG FULL` to populate permissions and role mappings.

```sql
-- All permissions (entity, microflow, page, OData access)
SELECT * FROM CATALOG.PERMISSIONS WHERE ModuleRoleName = 'MyModule.Admin';

-- Filter by type
SELECT ElementName, AccessType FROM CATALOG.PERMISSIONS
  WHERE ElementType = 'ENTITY' AND ModuleName = 'MyModule';

SELECT ElementName FROM CATALOG.PERMISSIONS
  WHERE ElementType = 'MICROFLOW' AND AccessType = 'EXECUTE';

-- User role to module role mappings
SELECT * FROM CATALOG.ROLE_MAPPINGS;
SELECT ModuleRoleName FROM CATALOG.ROLE_MAPPINGS WHERE UserRoleName = 'Administrator';

-- Which user roles have access to a module?
SELECT DISTINCT UserRoleName FROM CATALOG.ROLE_MAPPINGS WHERE ModuleName = 'MyModule';

-- Describe catalog table schema
DESCRIBE CATALOG.PERMISSIONS;
DESCRIBE CATALOG.ROLE_MAPPINGS;
```

**Catalog tables:**
| Table | Contents | Build mode |
|-------|----------|------------|
| `CATALOG.PERMISSIONS` | Entity CRUD, microflow EXECUTE, page VIEW, OData ACCESS | `REFRESH CATALOG FULL` |
| `CATALOG.ROLE_MAPPINGS` | User role → module role assignments | `REFRESH CATALOG` |

### Module Roles

```sql
-- Create module roles
CREATE MODULE ROLE MyModule.Admin DESCRIPTION 'Full administrative access';
CREATE MODULE ROLE MyModule.User;
CREATE MODULE ROLE MyModule.Viewer DESCRIPTION 'Read-only access';

-- Remove a module role
DROP MODULE ROLE MyModule.Viewer;
```

### Microflow Access

```sql
-- Grant execute access (multiple roles supported)
GRANT EXECUTE ON MICROFLOW MyModule.ACT_Customer_Create TO MyModule.User, MyModule.Admin;

-- Revoke from specific roles
REVOKE EXECUTE ON MICROFLOW MyModule.ACT_Customer_Create FROM MyModule.User;
```

### Page Access

```sql
-- Grant view access
GRANT VIEW ON PAGE MyModule.Customer_Overview TO MyModule.User, MyModule.Admin;

-- Revoke from specific roles
REVOKE VIEW ON PAGE MyModule.Customer_Overview FROM MyModule.User;
```

### Entity Access (CRUD)

```sql
-- Full access (all CRUD + all members)
GRANT MyModule.Admin ON MyModule.Customer (CREATE, DELETE, READ *, WRITE *);

-- Read-only (all members)
GRANT MyModule.Viewer ON MyModule.Customer (READ *);

-- Selective member access
GRANT MyModule.User ON MyModule.Customer (READ (Name, Email), WRITE (Email));

-- With XPath constraint
GRANT MyModule.User ON MyModule.Order (READ *, WRITE *) WHERE '[Status = ''Open'']';

-- Revoke entity access for a role
REVOKE MyModule.Viewer ON MyModule.Customer;
```

### User Roles

```sql
-- Create with module roles
CREATE USER ROLE RegularUser (MyModule.User, OtherModule.Reader);

-- Create with manage all roles permission
CREATE USER ROLE SuperAdmin (MyModule.Admin) MANAGE ALL ROLES;

-- Add/remove module roles
ALTER USER ROLE RegularUser ADD MODULE ROLES (MyModule.Viewer);
ALTER USER ROLE RegularUser REMOVE MODULE ROLES (MyModule.Viewer);

-- Remove user role
DROP USER ROLE RegularUser;
```

### Project Security Settings

```sql
-- Set security level
ALTER PROJECT SECURITY LEVEL OFF;
ALTER PROJECT SECURITY LEVEL PROTOTYPE;
ALTER PROJECT SECURITY LEVEL PRODUCTION;

-- Enable/disable demo users
ALTER PROJECT SECURITY DEMO USERS ON;
ALTER PROJECT SECURITY DEMO USERS OFF;
```

### Demo Users

```sql
-- Create demo user (auto-detects entity that generalizes System.User)
CREATE DEMO USER 'demo_admin' PASSWORD 'Admin123!' (Administrator, SuperAdmin);

-- Create demo user with explicit entity
CREATE DEMO USER 'demo_admin' PASSWORD 'Admin123!' ENTITY Administration.Account (Administrator, SuperAdmin);

-- Remove demo user
DROP DEMO USER 'demo_admin';
```

The ENTITY clause specifies which entity (generalizing `System.User`) to use. If omitted, it auto-detects the unique System.User subtype in the project. If multiple subtypes exist, you must specify ENTITY explicitly.

## Starlark Lint Rule APIs

Security data is available in Starlark lint rules (`.star` files):

| Function | Returns | Description |
|----------|---------|-------------|
| `permissions()` | list of permission | All permissions across all element types |
| `permissions_for(qn)` | list of permission | Permissions for a specific entity |
| `user_roles()` | list of user_role | User roles with module role assignments |
| `module_roles()` | list of module_role | Distinct module roles |
| `role_mappings()` | list of role_mapping | User role → module role mappings |
| `project_security()` | struct or None | Security level, guest access, password policy |

See `write-lint-rules.md` for object property details.

## Common Workflow: Setting Up Module Security

A typical security setup follows this order:

```sql
-- 1. Create module roles
CREATE MODULE ROLE Shop.User DESCRIPTION 'Regular user access';
CREATE MODULE ROLE Shop.Admin DESCRIPTION 'Administrative access';
CREATE MODULE ROLE Shop.Viewer DESCRIPTION 'Read-only access';

-- 2. Grant entity access
GRANT Shop.Admin ON Shop.Customer (CREATE, DELETE, READ *, WRITE *);
GRANT Shop.User ON Shop.Customer (READ (Name, Email), WRITE (Email));
GRANT Shop.Viewer ON Shop.Customer (READ *);

-- 3. Grant microflow access
GRANT EXECUTE ON MICROFLOW Shop.ACT_Customer_Create TO Shop.User, Shop.Admin;
GRANT EXECUTE ON MICROFLOW Shop.ACT_Customer_Delete TO Shop.Admin;

-- 4. Grant page access
GRANT VIEW ON PAGE Shop.Customer_Overview TO Shop.User, Shop.Admin, Shop.Viewer;
GRANT VIEW ON PAGE Shop.Customer_Edit TO Shop.User, Shop.Admin;

-- 5. Create user roles (project-level)
CREATE USER ROLE AppUser (Shop.User);
CREATE USER ROLE AppAdmin (Shop.Admin) MANAGE ALL ROLES;

-- 6. Verify
SHOW SECURITY MATRIX IN Shop;
DESCRIBE USER ROLE AppAdmin;
```

## Common Mistakes

1. **Creating module roles before the module exists** — `CREATE MODULE` must come first
2. **Referencing non-existent roles in GRANT** — create the module role before granting access
3. **Forgetting qualified names** — roles use `Module.Role` format in GRANT/REVOKE
4. **User roles without System module roles** — in Production security, user roles need at least one System module role (CE0156)
5. **Entity access without proper member rights** — use `READ *` for all members or `READ (Attr1, Attr2)` for specific ones

## Validation

After setting up security, verify with:
```bash
# Check security matrix
mxcli -p app.mpr -c "SHOW SECURITY MATRIX IN MyModule"

# Validate with Mendix
~/.mxcli/mxbuild/*/modeler/mx check app.mpr
```

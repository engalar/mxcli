# Demo Users

Demo users are test accounts created for development and testing. They appear on the login screen when demo users are enabled, allowing quick access without manual user setup.

## CREATE DEMO USER

```sql
CREATE DEMO USER '<username>' PASSWORD '<password>'
  [ENTITY <Module>.<Entity>]
  (<UserRole> [, ...]);
```

| Parameter | Description |
|-----------|-------------|
| `<username>` | Login name for the demo user |
| `<password>` | Password (visible in development only) |
| `ENTITY` | Optional. The entity that generalizes `System.User` (e.g., `Administration.Account`). If omitted, the system auto-detects the unique `System.User` subtype. |
| `<UserRole>` | One or more project-level user roles to assign |

### Examples

```sql
-- Basic demo user
CREATE DEMO USER 'demo_admin' PASSWORD 'Admin123!' (Administrator);

-- With explicit entity
CREATE DEMO USER 'demo_admin' PASSWORD 'Admin123!'
  ENTITY Administration.Account (Administrator);

-- Multiple roles
CREATE DEMO USER 'demo_manager' PASSWORD 'Manager1!'
  (Manager, Reporting);

-- Standard user
CREATE DEMO USER 'demo_user' PASSWORD 'User1234!' (Employee);
```

## DROP DEMO USER

```sql
DROP DEMO USER '<username>';
```

Example:

```sql
DROP DEMO USER 'demo_admin';
```

## Enabling Demo Users

Demo users only appear on the login screen when enabled in project security:

```sql
ALTER PROJECT SECURITY DEMO USERS ON;
```

To hide them:

```sql
ALTER PROJECT SECURITY DEMO USERS OFF;
```

## Listing Demo Users

```sql
SHOW DEMO USERS;
```

## Typical Setup

```sql
-- Enable demo users and set prototype security
ALTER PROJECT SECURITY LEVEL PROTOTYPE;
ALTER PROJECT SECURITY DEMO USERS ON;

-- Create demo accounts for each role
CREATE DEMO USER 'demo_admin' PASSWORD 'Admin123!'
  ENTITY Administration.Account (Administrator);
CREATE DEMO USER 'demo_user' PASSWORD 'User1234!'
  ENTITY Administration.Account (Employee);
CREATE DEMO USER 'demo_guest' PASSWORD 'Guest123!'
  ENTITY Administration.Account (Guest);
```

## See Also

- [Security](./security.md) -- overview of the security model
- [Module Roles and User Roles](./roles.md) -- defining the user roles assigned to demo users
- [GRANT / REVOKE](./grant-revoke.md) -- complete permission management reference

# Microflow, Page, and Nanoflow Access

Document-level access controls which module roles can execute microflows and nanoflows or view pages. Without an explicit grant, a document is inaccessible to all roles.

## Microflow Access

### GRANT EXECUTE ON MICROFLOW

```sql
GRANT EXECUTE ON MICROFLOW <Module>.<Name> TO <Module>.<Role> [, ...];
```

Examples:

```sql
-- Single role
GRANT EXECUTE ON MICROFLOW Shop.ACT_ProcessOrder TO Shop.Admin;

-- Multiple roles
GRANT EXECUTE ON MICROFLOW Shop.ACT_ViewOrders TO Shop.User, Shop.Admin;
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

Examples:

```sql
GRANT VIEW ON PAGE Shop.Order_Overview TO Shop.User, Shop.Admin;
GRANT VIEW ON PAGE Shop.Admin_Dashboard TO Shop.Admin;
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

Nanoflow access uses the same syntax as microflow access:

```sql
GRANT EXECUTE ON NANOFLOW Shop.NAV_Filter TO Shop.User, Shop.Admin;
REVOKE EXECUTE ON NANOFLOW Shop.NAV_Filter FROM Shop.User;
```

## Viewing Document Access

```sql
SHOW ACCESS ON MICROFLOW Shop.ACT_ProcessOrder;
SHOW ACCESS ON PAGE Shop.Order_Overview;
```

## Typical Pattern

After creating documents, grant access as part of the same script:

```sql
-- Create the microflow
CREATE MICROFLOW Shop.ACT_CreateOrder
BEGIN
  DECLARE $Order Shop.Order;
  $Order = CREATE Shop.Order (Status = 'Draft');
  COMMIT $Order;
  RETURN $Order;
END;

-- Grant access
GRANT EXECUTE ON MICROFLOW Shop.ACT_CreateOrder TO Shop.User, Shop.Admin;

-- Create the page
CREATE PAGE Shop.Order_Edit (
  Params: { $Order: Shop.Order },
  Title: 'Edit Order',
  Layout: Atlas_Core.PopupLayout
) { ... }

-- Grant access
GRANT VIEW ON PAGE Shop.Order_Edit TO Shop.User, Shop.Admin;
```

## See Also

- [Security](./security.md) -- overview of the security model
- [Entity Access](./entity-access.md) -- CRUD permissions on entities
- [GRANT / REVOKE](./grant-revoke.md) -- complete GRANT and REVOKE reference
- [Module Roles and User Roles](./roles.md) -- defining the roles

# Home Pages and Menus

Each navigation profile has a default home page, optional role-specific home pages, and a menu tree. These determine what users see when they first open the application and how they navigate between pages.

## Home Pages

### Default Home Page

Every profile requires a default home page. This is shown to users whose role has no specific home page assignment:

```sql
CREATE OR REPLACE NAVIGATION Responsive
  HOME PAGE MyModule.Home_Web;
```

### Role-Specific Home Pages

Use `HOME PAGE ... FOR` to direct users to different pages based on their module role:

```sql
CREATE OR REPLACE NAVIGATION Responsive
  HOME PAGE MyModule.Home_Web
  HOME PAGE MyModule.AdminDashboard FOR MyModule.Administrator
  HOME PAGE MyModule.ManagerDashboard FOR MyModule.Manager;
```

When a user logs in, the runtime checks their roles and redirects to the most specific matching home page. If no role-specific page matches, the default home page is used.

### Login Page

The login page is shown to unauthenticated users:

```sql
CREATE OR REPLACE NAVIGATION Responsive
  HOME PAGE MyModule.Home_Web
  LOGIN PAGE Administration.Login;
```

### Not Found Page

An optional custom 404 page:

```sql
CREATE OR REPLACE NAVIGATION Responsive
  HOME PAGE MyModule.Home_Web
  NOT FOUND PAGE MyModule.Custom404;
```

## Menus

The `MENU` block defines the navigation menu as a tree of items and submenus.

### Menu Items

A menu item links a label to a page:

```sql
MENU ITEM '<label>' PAGE <Module>.<Page>;
```

### Submenus

Nest items inside a `MENU '<label>' (...)` block:

```sql
MENU '<label>' (
  <menu-items>
);
```

### Complete Example

```sql
CREATE OR REPLACE NAVIGATION Responsive
  HOME PAGE Shop.Home
  LOGIN PAGE Administration.Login
  MENU (
    MENU ITEM 'Home' PAGE Shop.Home;
    MENU ITEM 'Products' PAGE Shop.Product_Overview;
    MENU ITEM 'Orders' PAGE Shop.Order_Overview;
    MENU 'Administration' (
      MENU ITEM 'Users' PAGE Administration.Account_Overview;
      MENU ITEM 'Roles' PAGE Administration.Role_Overview;
      MENU 'System' (
        MENU ITEM 'Logs' PAGE Administration.Log_Overview;
        MENU ITEM 'Settings' PAGE Shop.Settings;
      );
    );
  );
```

### Inspecting Menus

```sql
-- View menu tree
SHOW NAVIGATION MENU;
SHOW NAVIGATION MENU Responsive;

-- View home page assignments
SHOW NAVIGATION HOMES;
```

## See Also

- [Navigation and Settings](./navigation.md) -- overview of navigation concepts
- [Navigation Profiles](./navigation-profiles.md) -- profile types and CREATE OR REPLACE syntax
- [Project Settings](./project-settings.md) -- runtime and configuration settings

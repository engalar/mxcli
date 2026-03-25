# Navigation Profiles

A navigation profile defines the navigation structure for a specific device type. Each profile has a default home page, optional role-specific home pages, an optional login page, and a menu.

## Profile Types

| Profile | MDL Name | Description |
|---------|----------|-------------|
| Responsive web | `Responsive` | Default browser navigation |
| Tablet web | `Tablet` | Tablet-optimized browser navigation |
| Phone web | `Phone` | Phone-optimized browser navigation |
| Native mobile | `NativePhone` | React Native mobile navigation |

## CREATE OR REPLACE NAVIGATION

Replaces an entire navigation profile:

```sql
CREATE OR REPLACE NAVIGATION <Profile>
  HOME PAGE <Module>.<Page>
  [HOME PAGE <Module>.<Page> FOR <Module>.<Role>]
  [LOGIN PAGE <Module>.<Page>]
  [NOT FOUND PAGE <Module>.<Page>]
  [MENU (
    <menu-items>
  )]
```

### Full Example

```sql
CREATE OR REPLACE NAVIGATION Responsive
  HOME PAGE MyModule.Home_Web
  HOME PAGE MyModule.AdminHome FOR MyModule.Administrator
  LOGIN PAGE Administration.Login
  NOT FOUND PAGE MyModule.Custom404
  MENU (
    MENU ITEM 'Home' PAGE MyModule.Home_Web;
    MENU ITEM 'Orders' PAGE Shop.Order_Overview;
    MENU 'Administration' (
      MENU ITEM 'Users' PAGE Administration.Account_Overview;
      MENU ITEM 'Settings' PAGE MyModule.Settings;
    );
  );
```

### Minimal Example

A profile only requires a home page:

```sql
CREATE OR REPLACE NAVIGATION Phone
  HOME PAGE MyModule.Home_Phone;
```

## DESCRIBE NAVIGATION

View the current navigation profile in MDL syntax:

```sql
-- All profiles
DESCRIBE NAVIGATION;

-- Single profile
DESCRIBE NAVIGATION Responsive;
```

The output is round-trippable -- you can copy it, modify it, and execute it as a `CREATE OR REPLACE NAVIGATION` statement.

## See Also

- [Navigation and Settings](./navigation.md) -- overview of navigation and settings
- [Home Pages and Menus](./home-pages.md) -- details on home page and menu configuration
- [Project Settings](./project-settings.md) -- runtime and configuration settings

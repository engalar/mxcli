# Navigation and Settings

Navigation defines how users move through a Mendix application. Each device type has its own navigation profile with a home page, optional role-specific home pages, a login page, and a menu structure. Project settings control runtime behavior, configurations, languages, and workflows.

## Navigation Overview

A Mendix application has one or more **navigation profiles**, each targeting a device type. MDL supports inspecting and replacing entire profiles.

### Inspecting Navigation

```sql
-- Summary of all profiles
SHOW NAVIGATION;

-- Menu tree for a specific profile
SHOW NAVIGATION MENU Responsive;

-- Home page assignments across all profiles
SHOW NAVIGATION HOMES;

-- Full MDL output (round-trippable)
DESCRIBE NAVIGATION;
DESCRIBE NAVIGATION Responsive;
```

### Navigation Profiles

| Profile | Description |
|---------|-------------|
| `Responsive` | Web browser (desktop, laptop) |
| `Tablet` | Tablet browser |
| `Phone` | Phone browser |
| `NativePhone` | Native mobile (React Native) |

See [Navigation Profiles](./navigation-profiles.md) for full syntax.

## Settings Overview

Project settings control runtime configuration, database connections, languages, and workflow behavior.

```sql
-- Overview of all settings
SHOW SETTINGS;

-- Full MDL output
DESCRIBE SETTINGS;
```

See [Project Settings](./project-settings.md) for the ALTER SETTINGS syntax.

## See Also

- [Navigation Profiles](./navigation-profiles.md) -- creating and replacing navigation profiles
- [Home Pages and Menus](./home-pages.md) -- home page assignments and menu structures
- [Project Settings](./project-settings.md) -- runtime, configuration, and language settings

# SHOW CONSTANTS

## Synopsis

    SHOW CONSTANTS [IN <module>]

    SHOW CONSTANT VALUES [IN <module>]

## Description

Lists constants defined in the project. Without the `IN` clause, lists all constants across all modules. With `IN <module>`, restricts the listing to constants in the specified module.

Constants are named values (strings, integers, booleans, etc.) that can be configured per deployment environment. They are typically used for API URLs, feature flags, and configuration values.

`SHOW CONSTANT VALUES` shows one row per constant per configuration, making it easy to compare constant overrides across configurations (e.g., Default, Acceptance, Production). Each constant's default value is shown first, followed by any per-configuration overrides.

## Parameters

*module*
: The name of the module to filter by. Only constants belonging to this module are shown.

## Examples

List all constants in the project:

```sql
SHOW CONSTANTS
```

List constants in a specific module:

```sql
SHOW CONSTANTS IN MyModule
```

Compare constant values across all configurations:

```sql
SHOW CONSTANT VALUES
```

Compare constant values for a specific module:

```sql
SHOW CONSTANT VALUES IN MyModule
```

## See Also

[SHOW MODULES](show-modules.md), [SHOW STRUCTURE](show-structure.md), [SHOW / DESCRIBE SETTINGS](../settings/show-settings.md)

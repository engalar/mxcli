# SHOW / DESCRIBE SETTINGS

## Synopsis

    SHOW SETTINGS

    DESCRIBE SETTINGS [ category ]

## Description

Displays project settings. `SHOW SETTINGS` provides a compact overview of all settings across all categories. `DESCRIBE SETTINGS` outputs the full settings in round-trippable MDL syntax that can be used to recreate the same configuration.

When `DESCRIBE SETTINGS` is called without a category, it outputs all settings. When a category is specified, only that category is shown.

The available settings categories are:

| Category | Contents |
|----------|----------|
| `MODEL` | Application-level settings: AfterStartupMicroflow, BeforeShutdownMicroflow, HashAlgorithm, JavaVersion, etc. |
| `CONFIGURATION` | Runtime configurations: DatabaseType, DatabaseUrl, HttpPortNumber, etc. Each named configuration is listed separately. |
| `CONSTANT` | Constant value overrides per configuration. Shows which constants have non-default values in each configuration. |
| `LANGUAGE` | Localization settings: DefaultLanguageCode and available languages. |
| `WORKFLOWS` | Workflow engine settings: UserEntity, DefaultTaskParallelism, etc. |

## Parameters

**category** (DESCRIBE only)
: One of `MODEL`, `CONFIGURATION`, `CONSTANT`, `LANGUAGE`, or `WORKFLOWS`. If omitted, all categories are shown.

## Examples

### Show settings overview

```sql
SHOW SETTINGS;
```

### Describe all settings in MDL format

```sql
DESCRIBE SETTINGS;
```

### Describe model settings only

```sql
DESCRIBE SETTINGS MODEL;
```

### Describe runtime configurations

```sql
DESCRIBE SETTINGS CONFIGURATION;
```

### Describe workflow settings

```sql
DESCRIBE SETTINGS WORKFLOWS;
```

## See Also

[ALTER SETTINGS](alter-settings.md), [SHOW CONSTANT VALUES](../query/show-constants.md)

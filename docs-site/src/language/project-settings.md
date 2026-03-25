# Project Settings

Project settings control application runtime behavior, server configurations, language settings, and workflow configuration. MDL provides commands to inspect and modify all settings categories.

## Inspecting Settings

```sql
-- Overview of all settings
SHOW SETTINGS;

-- Full MDL output (round-trippable)
DESCRIBE SETTINGS;
```

## ALTER SETTINGS

Settings are organized into categories. Each `ALTER SETTINGS` command targets one category.

### Model Settings

Runtime-level settings such as the after-startup microflow, hashing algorithm, and Java version:

```sql
ALTER SETTINGS MODEL <Key> = <Value>;
```

Examples:

```sql
ALTER SETTINGS MODEL AfterStartupMicroflow = 'MyModule.ACT_Startup';
ALTER SETTINGS MODEL HashAlgorithm = 'BCrypt';
ALTER SETTINGS MODEL JavaVersion = '17';
```

### Configuration Settings

Server configuration settings like database type, URL, and HTTP port. Each configuration is identified by name (commonly `'default'`):

```sql
ALTER SETTINGS CONFIGURATION '<Name>' <Key> = <Value>;
```

Examples:

```sql
ALTER SETTINGS CONFIGURATION 'default' DatabaseType = 'POSTGRESQL';
ALTER SETTINGS CONFIGURATION 'default' DatabaseUrl = 'jdbc:postgresql://localhost:5432/myapp';
ALTER SETTINGS CONFIGURATION 'default' HttpPortNumber = '8080';
```

### Constant Overrides

Override a constant value within a specific configuration:

```sql
ALTER SETTINGS CONSTANT '<ConstantName>' VALUE '<value>' IN CONFIGURATION '<cfg>';
```

Example:

```sql
ALTER SETTINGS CONSTANT 'MyModule.ApiBaseUrl' VALUE 'https://staging.example.com' IN CONFIGURATION 'default';
```

### Language Settings

Set the default language for the application:

```sql
ALTER SETTINGS LANGUAGE <Key> = <Value>;
```

Example:

```sql
ALTER SETTINGS LANGUAGE DefaultLanguageCode = 'en_US';
```

### Workflow Settings

Configure workflow behavior such as the user entity and task parallelism:

```sql
ALTER SETTINGS WORKFLOWS <Key> = <Value>;
```

Examples:

```sql
ALTER SETTINGS WORKFLOWS UserEntity = 'Administration.Account';
ALTER SETTINGS WORKFLOWS DefaultTaskParallelism = '5';
```

## See Also

- [Navigation and Settings](./navigation.md) -- overview of navigation and settings
- [Navigation Profiles](./navigation-profiles.md) -- navigation profile configuration
- [Workflows](./workflows.md) -- workflow definitions

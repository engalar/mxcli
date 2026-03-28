# Project Settings

## When to Use This Skill

Use this skill when the user wants to:
- Configure database connections (PostgreSQL, SQLServer, etc.)
- Set up Kafka endpoints or other constant overrides
- Change the after-startup or before-shutdown microflows
- Modify hash algorithms, Java versions, or rounding modes
- View or modify language settings
- Configure workflow settings (user entity, parallelism)

## Commands

### View Settings

```sql
-- Overview table of all settings parts
SHOW SETTINGS;

-- Full MDL output (round-trippable ALTER SETTINGS statements)
DESCRIBE SETTINGS;
```

### Modify Model Settings

```sql
ALTER SETTINGS MODEL AfterStartupMicroflow = 'Module.MF_Startup';
ALTER SETTINGS MODEL BeforeShutdownMicroflow = 'Module.MF_Shutdown';
ALTER SETTINGS MODEL HealthCheckMicroflow = 'Module.MF_HealthCheck';
ALTER SETTINGS MODEL HashAlgorithm = 'BCrypt';
ALTER SETTINGS MODEL BcryptCost = 12;
ALTER SETTINGS MODEL JavaVersion = 'Java21';
ALTER SETTINGS MODEL RoundingMode = 'HalfUp';
ALTER SETTINGS MODEL AllowUserMultipleSessions = true;
ALTER SETTINGS MODEL ScheduledEventTimeZoneCode = 'Etc/UTC';
```

### Modify Configuration Settings

```sql
-- Full database configuration
ALTER SETTINGS CONFIGURATION 'Default'
  DatabaseType = 'PostgreSql',
  DatabaseUrl = 'localhost:5432',
  DatabaseName = 'mydb',
  DatabaseUserName = 'mendix',
  DatabasePassword = 'mendix',
  HttpPortNumber = 8080,
  ServerPortNumber = 8090;

-- Update a single field
ALTER SETTINGS CONFIGURATION 'Default'
  DatabaseUrl = 'newhost:5432';
```

### Constant Overrides

```sql
-- View constant values across all configurations
SHOW CONSTANT VALUES;
SHOW CONSTANT VALUES IN MyModule;    -- Filter by module

-- Override a constant value in a configuration
ALTER SETTINGS CONSTANT 'BusinessEvents.ServerUrl' VALUE 'kafka:9092'
  IN CONFIGURATION 'Default';

-- Without IN CONFIGURATION (uses first configuration)
ALTER SETTINGS CONSTANT 'MyModule.ApiKey' VALUE 'abc123';
```

### Language and Workflow Settings

```sql
ALTER SETTINGS LANGUAGE DefaultLanguageCode = 'en_US';

ALTER SETTINGS WORKFLOWS
  UserEntity = 'System.User',
  DefaultTaskParallelism = 3;
```

## Common Patterns

### PostgreSQL Configuration
```sql
ALTER SETTINGS CONFIGURATION 'Default'
  DatabaseType = 'PostgreSql',
  DatabaseUrl = 'localhost:5432',
  DatabaseName = 'myapp',
  DatabaseUserName = 'mendix',
  DatabasePassword = 'mendix',
  HttpPortNumber = 8080;
```

### SQL Server Configuration
```sql
ALTER SETTINGS CONFIGURATION 'Default'
  DatabaseType = 'SqlServer',
  DatabaseUrl = 'localhost:1433',
  DatabaseName = 'myapp',
  DatabaseUserName = 'sa',
  DatabasePassword = 'MyPassword',
  HttpPortNumber = 8080;
```

## Checklist

- [ ] Always run `SHOW SETTINGS` or `DESCRIBE SETTINGS` first to see current values
- [ ] Verify changes after modification with `SHOW SETTINGS`
- [ ] There is always exactly one ProjectSettings document; it cannot be created or deleted
- [ ] Model setting key names are case-sensitive (e.g., `JavaVersion`, not `javaversion`)
- [ ] Configuration names are case-insensitive (e.g., `'Default'` matches `'default'`)

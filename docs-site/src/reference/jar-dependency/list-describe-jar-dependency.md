# LIST / DESCRIBE JAR DEPENDENCY

## Synopsis

    LIST JAR DEPENDENCIES;
    LIST JAR DEPENDENCIES IN module;
    DESCRIBE JAR DEPENDENCY module 'group:artifact';

## Description

`LIST JAR DEPENDENCIES` shows all Maven/JAR dependencies declared across the
project, optionally filtered to a single module. `DESCRIBE JAR DEPENDENCY`
outputs the full definition of a single dependency as a re-executable
`ALTER MODULE … ADD JAR DEPENDENCY` statement.

## Parameters

**IN module**
: Filters the listing to a single module.

**module**
: The module name containing the dependency (for `DESCRIBE`).

**'group:artifact'**
: The Maven coordinate identifying the dependency, in `groupId:artifactId`
  format (no version). Example: `'org.duckdb:duckdb_jdbc'`.

## Examples

### List all JAR dependencies

```sql
LIST JAR DEPENDENCIES;
```

Output:
```
module      group                         artifact           version   included
MyModule    com.fasterxml.jackson.core    jackson-databind   2.21.2    true
MyModule    org.duckdb                    duckdb_jdbc        1.1.3     true
```

### Filter by module

```sql
LIST JAR DEPENDENCIES IN MyModule;
```

### Describe a specific dependency

```sql
DESCRIBE JAR DEPENDENCY MyModule 'com.fasterxml.jackson.core:jackson-databind';
```

Output:
```sql
alter module MyModule
  add jar dependency (
    group    = 'com.fasterxml.jackson.core',
    artifact = 'jackson-databind',
    version  = '2.21.2',
    included = true,
  );
```

The output is suitable for copying into a script and re-executing.

## See Also

[ALTER MODULE JAR DEPENDENCY](alter-module-jar-dependency.md)

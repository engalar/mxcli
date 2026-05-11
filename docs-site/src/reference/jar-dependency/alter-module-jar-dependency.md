# ALTER MODULE … JAR DEPENDENCY

## Synopsis

    ALTER MODULE module_name
      ADD JAR DEPENDENCY (
        group    = 'groupId',
        artifact = 'artifactId',
        version  = 'version',
        included = true|false,
      );

    ALTER MODULE module_name
      SET JAR DEPENDENCY 'group:artifact' VERSION 'new-version';

    ALTER MODULE module_name
      SET JAR DEPENDENCY 'group:artifact' INCLUDED true|false;

    ALTER MODULE module_name
      SET JAR DEPENDENCY 'group:artifact' ADD EXCLUSION 'group:artifact';

    ALTER MODULE module_name
      SET JAR DEPENDENCY 'group:artifact' DROP EXCLUSION 'group:artifact';

    ALTER MODULE module_name
      DROP JAR DEPENDENCY 'group:artifact';

## Description

Manages Maven/JAR dependencies stored in a module's `Projects$ModuleSettings`
document. These are the same dependencies shown in Studio Pro under
**Module Settings → Java dependencies**.

Multiple actions can be chained in a single `ALTER MODULE` statement.

## Parameters

**module_name**
: The name of the module to modify.

**'group:artifact'**
: The Maven coordinate identifying the dependency, in `groupId:artifactId`
  format (no version). Example: `'org.duckdb:duckdb_jdbc'`.

**group, artifact, version**
: The three components of a Maven coordinate. All three are required for `ADD`.

**included**
: When `false`, the dependency is declared but excluded from the classpath
  at build time. Defaults to `true`.

## Examples

### Add a new dependency

```sql
ALTER MODULE MyModule
  ADD JAR DEPENDENCY (
    group    = 'org.duckdb',
    artifact = 'duckdb_jdbc',
    version  = '1.1.3',
    included = true,
  );
```

### Update the version

```sql
ALTER MODULE MyModule
  SET JAR DEPENDENCY 'org.duckdb:duckdb_jdbc' VERSION '1.2.0';
```

### Disable a dependency without removing it

```sql
ALTER MODULE MyModule
  SET JAR DEPENDENCY 'org.duckdb:duckdb_jdbc' INCLUDED false;
```

### Add a transitive exclusion

```sql
ALTER MODULE MyModule
  SET JAR DEPENDENCY 'org.duckdb:duckdb_jdbc'
    ADD EXCLUSION 'com.example:conflicting-lib';
```

### Remove a transitive exclusion

```sql
ALTER MODULE MyModule
  SET JAR DEPENDENCY 'org.duckdb:duckdb_jdbc'
    DROP EXCLUSION 'com.example:conflicting-lib';
```

### Remove a dependency entirely

```sql
ALTER MODULE MyModule
  DROP JAR DEPENDENCY 'org.duckdb:duckdb_jdbc';
```

### Chain multiple actions

```sql
ALTER MODULE MyModule
  ADD JAR DEPENDENCY (
    group    = 'com.fasterxml.jackson.core',
    artifact = 'jackson-databind',
    version  = '2.21.2',
    included = true,
  )
  ADD JAR DEPENDENCY (
    group    = 'com.fasterxml.jackson.core',
    artifact = 'jackson-core',
    version  = '2.21.2',
    included = true,
  );
```

## Notes

Writing the dependency updates the `.mpr` file immediately, but Gradle must
still resolve and download the JAR into `vendorlib/`. This happens automatically
when the project is opened in Studio Pro or built with `mxcli docker build`.

## See Also

[LIST / DESCRIBE JAR DEPENDENCY](list-describe-jar-dependency.md)

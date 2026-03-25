# Database Connector Generation

The `SQL GENERATE CONNECTOR` command auto-generates Mendix Database Connector MDL from an external database schema. This creates the entities, microflows, and configuration needed to query external databases from within a Mendix application.

## Syntax

```sql
SQL <alias> GENERATE CONNECTOR INTO <module> [TABLES (<table1>, <table2>, ...)] [VIEWS (<view1>, ...)] [EXEC]
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<alias>` | Named connection established with `SQL CONNECT` |
| `INTO <module>` | Target Mendix module to generate the connector in |
| `TABLES (...)` | Optional: specific tables to include (default: all) |
| `VIEWS (...)` | Optional: specific views to include |
| `EXEC` | Execute the generated MDL immediately (otherwise just output it) |

## Examples

### Generate for All Tables

```sql
SQL CONNECT postgres 'postgres://user:pass@localhost:5432/mydb' AS source;

-- Preview the generated MDL (output only, no changes)
SQL source GENERATE CONNECTOR INTO HRModule;
```

### Generate for Specific Tables

```sql
-- Generate for selected tables only
SQL source GENERATE CONNECTOR INTO HRModule TABLES (employees, departments);
```

### Generate and Execute

```sql
-- Generate and immediately apply to the project
SQL source GENERATE CONNECTOR INTO HRModule
  TABLES (employees, departments)
  EXEC;
```

The `EXEC` flag executes the generated MDL against the connected project, creating the entities and microflows in one step.

## What Gets Generated

The connector generation inspects the external database schema and produces:

- **Non-persistent entities** for each table/view, with attributes matching column types
- **Microflow activities** using `EXECUTE DATABASE QUERY` for querying the external database
- **SQL type mapping** from database column types to Mendix attribute types

## Type Mapping

External database types are mapped to Mendix types:

| SQL Type | Mendix Type |
|----------|-------------|
| `VARCHAR`, `TEXT`, `CHAR` | String |
| `INTEGER`, `INT`, `SMALLINT` | Integer |
| `BIGINT` | Long |
| `NUMERIC`, `DECIMAL`, `FLOAT`, `DOUBLE` | Decimal |
| `BOOLEAN`, `BIT` | Boolean |
| `DATE`, `TIMESTAMP`, `DATETIME` | DateTime |
| `BYTEA`, `BLOB` | Binary |

## Workflow

```sql
-- 1. Connect to the external database
SQL CONNECT postgres 'postgres://user:pass@host:5432/db' AS source;

-- 2. Explore the schema
SQL source SHOW TABLES;
SQL source DESCRIBE employees;

-- 3. Preview the generated MDL
SQL source GENERATE CONNECTOR INTO HRModule TABLES (employees, departments);

-- 4. Review the output, then execute
SQL source GENERATE CONNECTOR INTO HRModule TABLES (employees, departments) EXEC;

-- 5. Disconnect
SQL DISCONNECT source;
```

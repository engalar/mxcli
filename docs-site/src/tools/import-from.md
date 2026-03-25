# IMPORT FROM

The `IMPORT` command imports data from an external database into a Mendix application's PostgreSQL database. It supports column mapping, association linking, batch processing, and row limits.

## Syntax

```sql
IMPORT FROM <alias> QUERY '<sql>'
  INTO <Module.Entity>
  MAP (<source-col> AS <AttrName> [, ...])
  [LINK (<source-col> TO <AssocName> ON <MatchAttr>) [, ...]]
  [BATCH <size>]
  [LIMIT <count>]
```

## Parameters

| Parameter | Description |
|-----------|-------------|
| `<alias>` | Named connection established with `SQL CONNECT` |
| `QUERY '<sql>'` | SQL query to execute against the external database |
| `INTO <Module.Entity>` | Target Mendix entity to insert into |
| `MAP (...)` | Column-to-attribute mapping |
| `LINK (...)` | Association linking via lookup |
| `BATCH <size>` | Number of rows per batch insert (default varies) |
| `LIMIT <count>` | Maximum number of rows to import |

## Basic Import

Map source columns to entity attributes:

```sql
SQL CONNECT postgres 'postgres://user:pass@localhost:5432/mydb' AS source;

IMPORT FROM source QUERY 'SELECT name, email FROM employees'
  INTO HR.Employee
  MAP (name AS Name, email AS Email);
```

## Import with Association Linking

The `LINK` clause creates associations by looking up related entities:

```sql
IMPORT FROM source QUERY 'SELECT name, dept_name FROM employees'
  INTO HR.Employee
  MAP (name AS Name)
  LINK (dept_name TO Employee_Department ON Name);
```

This matches the `dept_name` value from the source query against the `Name` attribute of the entity on the other side of the `Employee_Department` association, creating the association link for each imported row.

## Batch and Limit Control

```sql
IMPORT FROM source QUERY 'SELECT * FROM products'
  INTO Catalog.Product
  MAP (name AS Name, price AS Price)
  BATCH 500
  LIMIT 1000;
```

- `BATCH 500` -- Insert 500 rows at a time (reduces memory usage)
- `LIMIT 1000` -- Stop after importing 1000 rows

## How It Works

The IMPORT pipeline:

1. **Executes the source query** against the external database
2. **Generates Mendix IDs** for each new row (proper Mendix UUID format)
3. **Maps columns** to entity attributes based on the MAP clause
4. **Resolves associations** by looking up matching entities for LINK clauses
5. **Batch inserts** into the Mendix application's PostgreSQL database
6. **Tracks sequences** to ensure auto-number fields continue correctly

## Auto-Connection to Mendix DB

The IMPORT command automatically connects to the Mendix application's PostgreSQL database based on the project's configuration settings. You do not need to manually connect to the Mendix database.

## Complete Example

```sql
-- Connect to external source
SQL CONNECT postgres 'postgres://user:pass@legacy-db:5432/hrdb' AS legacy;

-- Preview the data
SQL legacy SELECT name, email, department FROM employees LIMIT 5;

-- Import employees
IMPORT FROM legacy QUERY 'SELECT name, email FROM employees WHERE active = true'
  INTO HR.Employee
  MAP (name AS FullName, email AS EmailAddress)
  BATCH 500;

-- Import with association linking
IMPORT FROM legacy QUERY 'SELECT name, dept_name FROM employees WHERE active = true'
  INTO HR.Employee
  MAP (name AS FullName)
  LINK (dept_name TO Employee_Department ON DepartmentName)
  BATCH 500
  LIMIT 10000;

-- Disconnect
SQL DISCONNECT legacy;
```

## Notes

- The target entity must already exist in the Mendix project
- Column types are automatically converted where possible
- Association linking requires the target entity to have matching values in the lookup attribute
- The Mendix application database must be a PostgreSQL instance accessible from the mxcli host

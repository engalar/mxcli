# SQL Queries

The catalog supports standard SQL SELECT syntax for querying project metadata. Queries use the `CATALOG.` prefix to identify catalog tables.

## Syntax

```sql
SELECT <columns> FROM CATALOG.<table> [WHERE <condition>] [ORDER BY <column>] [LIMIT <n>]
```

Standard SQL features are supported: `SELECT`, `WHERE`, `JOIN`, `GROUP BY`, `ORDER BY`, `HAVING`, `UNION`, `LIMIT`, aggregate functions (`COUNT`, `SUM`, `AVG`, `MIN`, `MAX`), and subqueries.

## Basic Queries

### Listing Elements

```sql
-- All entities in a module
SELECT Name, EntityType, Description
FROM CATALOG.ENTITIES
WHERE ModuleName = 'Sales';

-- All microflows in a module
SELECT Name, ReturnType
FROM CATALOG.MICROFLOWS
WHERE ModuleName = 'Sales';

-- Pages with URLs
SELECT Name, ModuleName, URL
FROM CATALOG.PAGES
WHERE URL IS NOT NULL;
```

### Filtering with LIKE

```sql
-- Find entities by name pattern
SELECT QualifiedName FROM CATALOG.ENTITIES
WHERE Name LIKE '%Customer%';

-- Find microflows referencing an entity
SELECT Name, ModuleName
FROM CATALOG.MICROFLOWS
WHERE Description LIKE '%Customer%'
   OR Name LIKE '%Customer%';

-- Find pages by URL pattern
SELECT Name, ModuleName, URL
FROM CATALOG.PAGES
WHERE URL LIKE '/admin%';
```

### Joining Tables

```sql
-- Attributes of a specific entity
SELECT a.Name, a.AttributeType
FROM CATALOG.ATTRIBUTES a
JOIN CATALOG.ENTITIES e ON a.EntityId = e.Id
WHERE e.QualifiedName = 'Sales.Customer';

-- Entity relationships
SELECT Name, ParentEntity, ChildEntity, AssociationType
FROM CATALOG.ASSOCIATIONS
WHERE ParentEntity LIKE '%Order%'
   OR ChildEntity LIKE '%Order%';
```

### Aggregation

```sql
-- Count entities per module
SELECT ModuleName, COUNT(*) AS EntityCount
FROM CATALOG.ENTITIES
GROUP BY ModuleName
ORDER BY EntityCount DESC;

-- Documentation coverage by module
SELECT
    ModuleName,
    COUNT(*) AS TotalEntities,
    SUM(CASE WHEN Description != '' THEN 1 ELSE 0 END) AS Documented,
    ROUND(100.0 * SUM(CASE WHEN Description != '' THEN 1 ELSE 0 END) / COUNT(*), 1) AS CoveragePercent
FROM CATALOG.ENTITIES
GROUP BY ModuleName
ORDER BY CoveragePercent ASC;
```

### Subqueries and UNION

```sql
-- Complete data flow for an entity
SELECT 'Page' AS Source, p.QualifiedName AS Element, 'displays' AS Action
FROM CATALOG.PAGES p
WHERE p.DataSource LIKE '%Customer%'
UNION ALL
SELECT 'Microflow' AS Source, m.QualifiedName AS Element, 'processes' AS Action
FROM CATALOG.MICROFLOWS m
WHERE m.ObjectUsage LIKE '%Customer%';
```

## CLI Usage

```bash
# Run a catalog query
mxcli -p app.mpr -c "SELECT Name FROM CATALOG.ENTITIES WHERE ModuleName = 'Sales'"

# Query with limit
mxcli -p app.mpr -c "SELECT Name FROM CATALOG.MICROFLOWS LIMIT 10"
```

## Notes

- Always run `REFRESH CATALOG` (or `REFRESH CATALOG FULL`) before querying
- Table and column names are case-sensitive in some contexts
- String comparisons use SQLite semantics (case-sensitive by default with `=`, use `LIKE` for case-insensitive matching)
- The catalog is read-only; INSERT, UPDATE, and DELETE are not supported

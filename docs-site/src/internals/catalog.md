# Catalog System

The catalog is an in-memory SQLite database that indexes Mendix project metadata for fast querying, full-text search, and cross-reference navigation.

## Purpose

Reading a Mendix project requires parsing BSON documents from the MPR file. The catalog pre-processes this data into relational tables, enabling:

- **SQL queries** against project metadata (`SELECT ... FROM CATALOG.ENTITIES`)
- **Full-text search** across all strings and source (`SEARCH 'keyword'`)
- **Cross-reference tracking** (callers, callees, references, impact analysis)
- **Lint rule evaluation** (Starlark rules query catalog tables)

## Building the Catalog

The catalog is populated by the `REFRESH CATALOG` command:

```sql
-- Quick refresh: basic tables (modules, entities, microflows, pages, etc.)
REFRESH CATALOG;

-- Full refresh: adds cross-references, widgets, and source tables
REFRESH CATALOG FULL;
```

The full refresh is required for:

- `SHOW CALLERS` / `SHOW CALLEES` / `SHOW REFERENCES` / `SHOW IMPACT`
- `SHOW WIDGETS` / `UPDATE WIDGETS`
- `SELECT ... FROM CATALOG.REFS`
- `SELECT ... FROM CATALOG.SOURCE`

## Querying the Catalog

Use standard SQL syntax with the `CATALOG.` prefix:

```sql
-- Find large microflows
SELECT Name, ActivityCount
FROM CATALOG.MICROFLOWS
WHERE ActivityCount > 10
ORDER BY ActivityCount DESC;

-- Find all entity usages
SELECT SourceName, RefKind, TargetName
FROM CATALOG.REFS
WHERE TargetName = 'MyModule.Customer';

-- Search the strings table
SELECT * FROM CATALOG.STRINGS
WHERE strings MATCH 'error'
LIMIT 10;
```

## Available Tables

| Table | Contents | Populated By |
|-------|----------|--------------|
| `MODULES` | Module names and IDs | `REFRESH CATALOG` |
| `ENTITIES` | Entity names, persistence, attribute counts | `REFRESH CATALOG` |
| `MICROFLOWS` | Microflow names, activity counts, parameters | `REFRESH CATALOG` |
| `NANOFLOWS` | Nanoflow names and metadata | `REFRESH CATALOG` |
| `PAGES` | Page names, layouts, URLs | `REFRESH CATALOG` |
| `SNIPPETS` | Snippet names | `REFRESH CATALOG` |
| `ENUMERATIONS` | Enumeration names and value counts | `REFRESH CATALOG` |
| `CONSTANTS` | Constant names, types, default values | `REFRESH CATALOG` |
| `CONSTANT_VALUES` | Per-configuration constant overrides | `REFRESH CATALOG` |
| `WORKFLOWS` | Workflow names and activity counts | `REFRESH CATALOG` |
| `ACTIVITIES` | Individual microflow activities | `REFRESH CATALOG FULL` |
| `WIDGETS` | Widget instances across all pages | `REFRESH CATALOG FULL` |
| `REFS` | Cross-references between documents | `REFRESH CATALOG FULL` |
| `PERMISSIONS` | Security permissions (entity access, microflow access) | `REFRESH CATALOG FULL` |
| `STRINGS` | FTS5 full-text search index | `REFRESH CATALOG FULL` |
| `SOURCE` | MDL source text for all documents | `REFRESH CATALOG FULL` |

## Implementation

The catalog is implemented in `mdl/catalog/catalog.go`. It:

1. Creates an in-memory SQLite database
2. Iterates all modules and documents in the project
3. Inserts rows into the appropriate tables
4. Builds FTS5 indexes for full-text search
5. Builds the reference graph for cross-reference queries

The catalog lives for the duration of the mxcli session and is discarded on exit.

## Related Pages

- [SQLite Schema](./catalog-schema.md) -- table definitions and indexes
- [FTS5 Full-Text Search](./fts5.md) -- how full-text search works
- [Reference Tracking](./reference-tracking.md) -- callers, callees, and impact analysis

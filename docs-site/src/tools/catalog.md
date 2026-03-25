# Catalog Queries

The catalog is a SQLite-based query system that provides SQL access to project metadata. It indexes all elements in your Mendix project -- entities, microflows, pages, associations, attributes, and more -- into queryable tables that support standard SQL syntax.

## How It Works

The catalog builds an in-memory SQLite database from the project's MPR file. This database is cached in `.mxcli/catalog.db` next to the MPR file for fast subsequent access.

There are two levels of catalog refresh:

| Command | What It Builds | Use Case |
|---------|---------------|----------|
| `REFRESH CATALOG` | Basic metadata tables (entities, microflows, pages, etc.) | Quick queries about project structure |
| `REFRESH CATALOG FULL` | Metadata + cross-references + source content | Code navigation, impact analysis, full-text search |

## Quick Start

```sql
-- Build the catalog
REFRESH CATALOG;

-- See what tables are available
SHOW CATALOG TABLES;

-- Query entities in a module
SELECT Name, EntityType FROM CATALOG.ENTITIES WHERE ModuleName = 'Sales';

-- Find microflows that reference an entity
SELECT Name, ModuleName FROM CATALOG.MICROFLOWS
WHERE Description LIKE '%Customer%' OR Name LIKE '%Customer%';
```

## Key Features

- **Standard SQL syntax** -- SELECT, WHERE, JOIN, GROUP BY, ORDER BY, HAVING, UNION
- **Multiple tables** -- Entities, attributes, associations, microflows, pages, and more
- **Cross-reference data** -- Available after `REFRESH CATALOG FULL`
- **Cached for performance** -- Stored in `.mxcli/catalog.db` next to the MPR file
- **AI-friendly** -- Structured data that AI assistants can query programmatically

## CLI Usage

```bash
# Refresh catalog
mxcli -p app.mpr -c "REFRESH CATALOG"
mxcli -p app.mpr -c "REFRESH CATALOG FULL"

# Query catalog
mxcli -p app.mpr -c "SELECT Name FROM CATALOG.MICROFLOWS LIMIT 10"
mxcli -p app.mpr -c "SHOW CATALOG TABLES"
```

## Related Pages

- [REFRESH CATALOG](refresh-catalog.md) -- Building and refreshing the catalog
- [Available Tables](catalog-tables.md) -- What tables are available
- [SQL Queries](catalog-sql.md) -- Query syntax and examples
- [Use Cases](catalog-use-cases.md) -- Practical analysis examples

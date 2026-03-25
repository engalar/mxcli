# REFRESH CATALOG

The `REFRESH CATALOG` command builds or rebuilds the catalog index from the current project state. The catalog must be refreshed before running catalog queries or cross-reference navigation commands.

## Syntax

```sql
-- Basic refresh: builds metadata tables
REFRESH CATALOG

-- Full refresh: builds metadata + cross-references + source
REFRESH CATALOG FULL

-- Force rebuild: ignores cached catalog
REFRESH CATALOG FULL FORCE
```

## Refresh Levels

### Basic Refresh

```sql
REFRESH CATALOG;
```

Builds the core metadata tables: entities, attributes, associations, microflows, pages, enumerations, and other element types. This is sufficient for structural queries like "list all entities in a module" or "find pages with a specific URL."

### Full Refresh

```sql
REFRESH CATALOG FULL;
```

Builds everything in the basic refresh plus:

- **Cross-reference data** -- Which elements reference which other elements
- **Source content** -- Microflow activity content, expression strings, widget properties
- **Call graph** -- Caller/callee relationships between elements

This level is required for:
- `SHOW CALLERS OF`
- `SHOW CALLEES OF`
- `SHOW REFERENCES OF`
- `SHOW IMPACT OF`
- `SHOW CONTEXT OF`
- `SEARCH`
- Full-text catalog queries

### Force Rebuild

```sql
REFRESH CATALOG FULL FORCE;
```

Ignores the cached `.mxcli/catalog.db` file and rebuilds from scratch. Use this when the catalog appears stale or corrupt.

## CLI Usage

```bash
# Quick refresh
mxcli -p app.mpr -c "REFRESH CATALOG"

# Full refresh with cross-references
mxcli -p app.mpr -c "REFRESH CATALOG FULL"

# Force rebuild
mxcli -p app.mpr -c "REFRESH CATALOG FULL FORCE"
```

## Caching

The catalog is cached in `.mxcli/catalog.db` next to the MPR file. On subsequent refreshes, mxcli checks whether the project has changed and only rebuilds if necessary (unless `FORCE` is specified).

## When to Refresh

- **After opening a project** -- Run `REFRESH CATALOG` to enable queries
- **Before cross-reference navigation** -- Run `REFRESH CATALOG FULL` to enable CALLERS/CALLEES/IMPACT
- **After making changes** -- Refresh to update the catalog with new elements
- **If results seem stale** -- Use `REFRESH CATALOG FULL FORCE` to force a rebuild

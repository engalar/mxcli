# Full-Text Search

The `SEARCH` command performs full-text search across all strings and source code in a Mendix project. It searches element names, documentation, captions, expressions, and other string content.

## Syntax

```sql
SEARCH '<keyword>'
```

## Examples

```sql
-- Search for a keyword
SEARCH 'validation';

-- Search for an entity name
SEARCH 'Customer';

-- Search for error messages
SEARCH 'cannot be empty';

-- Search for specific patterns
SEARCH 'CurrentDateTime';
```

## CLI Usage

```bash
# Basic search
mxcli search -p app.mpr "validation"

# Quiet mode (no headers/formatting), suitable for piping
mxcli search -p app.mpr "validation" -q

# Output as names (type<TAB>name per line)
mxcli search -p app.mpr "validation" -q --format names

# Output as JSON array
mxcli search -p app.mpr "validation" -q --format json
```

## Output Formats

| Format | Description |
|--------|-------------|
| `names` (default) | `type<TAB>name` per line, suitable for piping |
| `json` | JSON array of search results |

The `-q` (quiet) flag suppresses headers and formatting, making output suitable for piping to other commands.

## Piping Search Results

Search results can be piped to other mxcli commands for powerful workflows:

```bash
# Find the first microflow matching "error" and describe it
mxcli search -p app.mpr "error" -q --format names | head -1 | awk '{print $2}' | \
  xargs mxcli describe -p app.mpr microflow
```

## What Gets Searched

The search command looks through:

- Element names (entities, microflows, pages, etc.)
- Documentation comments
- Attribute names and captions
- Expression strings in microflows
- Widget captions and labels
- Enumeration value captions
- Log message strings
- Validation feedback messages

## Comparison with Catalog Queries

`SEARCH` provides quick keyword-based search, while catalog queries (`SELECT FROM CATALOG.*`) provide structured SQL-based queries with filtering, joining, and aggregation. Use `SEARCH` for quick discovery and catalog queries for precise analysis.

```sql
-- Quick search: find anything mentioning "Customer"
SEARCH 'Customer';

-- Precise query: find entities named Customer with their module
SELECT Name, ModuleName FROM CATALOG.ENTITIES WHERE Name LIKE '%Customer%';
```

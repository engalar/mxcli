# FTS5 Full-Text Search

The catalog uses SQLite's FTS5 extension to provide fast full-text search across all project content.

## Overview

Two FTS5 virtual tables are created during `REFRESH CATALOG FULL`:

| Table | Indexes | Use Case |
|-------|---------|----------|
| `STRINGS` | Validation messages, captions, labels, log messages | `SEARCH 'keyword'` |
| `SOURCE` | MDL source representation of all documents | Code search |

Both use the `porter unicode61` tokenizer, which provides:

- **Porter stemming** -- matches word variants (e.g., "validate" matches "validation")
- **Unicode support** -- handles non-ASCII characters correctly

## Building the Index

During `REFRESH CATALOG FULL`, the catalog:

1. Iterates all documents in the project
2. Extracts all text content (messages, captions, names, expressions)
3. Generates the MDL source representation for each document
4. Inserts into the FTS5 tables

## Querying

### From MDL

```sql
-- Simple keyword search
SEARCH 'validation';

-- Phrase search
SEARCH 'error message';
```

### From SQL

```sql
-- Basic match
SELECT name, kind FROM CATALOG.STRINGS
WHERE strings MATCH 'customer';

-- Phrase match
SELECT name, kind FROM CATALOG.STRINGS
WHERE strings MATCH '"email validation"';

-- Boolean operators
SELECT name, kind FROM CATALOG.STRINGS
WHERE strings MATCH 'customer AND NOT deleted';

-- With snippets (highlighted context)
SELECT name, kind,
    snippet(STRINGS, 2, '>>>', '<<<', '...', 20) as context
FROM CATALOG.STRINGS
WHERE strings MATCH 'error'
LIMIT 10;

-- Search MDL source
SELECT name, kind FROM CATALOG.SOURCE
WHERE source MATCH 'RETRIEVE FROM';
```

## FTS5 Query Syntax

FTS5 supports a rich query syntax:

| Syntax | Meaning |
|--------|---------|
| `word` | Match documents containing "word" |
| `"exact phrase"` | Match exact phrase |
| `word1 AND word2` | Both words must appear |
| `word1 OR word2` | Either word may appear |
| `NOT word` | Exclude documents with this word |
| `word*` | Prefix match (e.g., "valid*" matches "validation") |
| `NEAR(word1 word2, 5)` | Words within 5 tokens of each other |

## CLI Integration

The `SEARCH` command and `mxcli search` subcommand are wrappers around FTS5 queries:

```bash
# Search from command line
mxcli search -p app.mpr "validation"

# Pipe-friendly output
mxcli search -p app.mpr "error" -q --format names

# JSON output
mxcli search -p app.mpr "Customer" -q --format json
```

The search results include:

- Document type (Microflow, Page, Entity, etc.)
- Qualified name
- Matching context snippet

## Performance

FTS5 indexes are built in memory and provide sub-millisecond query times for typical project sizes. The index construction itself takes a few seconds for large projects (1000+ documents), which is why it requires the explicit `REFRESH CATALOG FULL` command.

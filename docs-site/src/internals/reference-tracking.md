# Reference Tracking

The cross-reference tracking system powers the callers, callees, references, and impact analysis queries. It is built during `REFRESH CATALOG FULL` and stored in the `REFS` table.

## How References Are Collected

During a full catalog refresh, every document is analyzed for outgoing references:

| Document Type | Reference Sources |
|--------------|-------------------|
| **Microflows** | CALL MICROFLOW actions, RETRIEVE data sources, entity parameters, SHOW PAGE actions, association traversals |
| **Nanoflows** | Same as microflows (client-side) |
| **Pages** | Data source entities, microflow data sources, SHOW_PAGE actions, association paths, snippet calls |
| **Snippets** | Same as pages |
| **Domain Models** | Generalization references, association endpoints |

Each reference is stored as a row in the `REFS` table:

```sql
INSERT INTO REFS (SourceName, SourceKind, TargetName, TargetKind, RefKind)
VALUES ('Sales.ProcessOrder', 'Microflow', 'Sales.Customer', 'Entity', 'Retrieve');
```

## Query Commands

### SHOW CALLERS

Find what calls a given element:

```sql
SHOW CALLERS OF Sales.ProcessOrder;
```

Returns all microflows, nanoflows, and pages that reference `Sales.ProcessOrder`.

**Transitive callers** follow the call chain recursively:

```sql
SHOW CALLERS OF Sales.ProcessOrder TRANSITIVE;
```

### SHOW CALLEES

Find what a microflow calls:

```sql
SHOW CALLEES OF Sales.ProcessOrder;
```

Returns all microflows, entities, and pages referenced by `Sales.ProcessOrder`.

### SHOW REFERENCES

Find all references to an element:

```sql
SHOW REFERENCES TO Sales.Customer;
```

Returns every document that references `Sales.Customer` in any way (data source, parameter, association, etc.).

### SHOW IMPACT

Analyze the impact of changing an element:

```sql
SHOW IMPACT OF Sales.Customer;
```

Returns all direct and transitive dependents -- everything that would potentially be affected by changing or removing the element.

### SHOW CONTEXT

Assemble context for understanding a document:

```sql
SHOW CONTEXT OF Sales.ProcessOrder DEPTH 3;
```

Returns the element itself plus its callers and callees up to the specified depth, providing a focused view of the element's neighborhood in the dependency graph.

## CLI Commands

The same queries are available as CLI subcommands:

```bash
# Direct callers
mxcli callers -p app.mpr Sales.ProcessOrder

# Transitive callers
mxcli callers -p app.mpr Sales.ProcessOrder --transitive

# Callees
mxcli callees -p app.mpr Sales.ProcessOrder

# All references
mxcli refs -p app.mpr Sales.Customer

# Impact analysis
mxcli impact -p app.mpr Sales.Customer

# Context assembly
mxcli context -p app.mpr Sales.ProcessOrder --depth 3
```

## Reference Kinds

The `RefKind` column in the `REFS` table categorizes the relationship:

| RefKind | Meaning |
|---------|---------|
| `Call` | Microflow/nanoflow call action |
| `Retrieve` | Database retrieve with entity data source |
| `DataSource` | Page/widget data source reference |
| `ShowPage` | Show page action |
| `Parameter` | Microflow parameter type reference |
| `Association` | Association traversal |
| `Generalization` | Entity generalization (inheritance) |
| `SnippetCall` | Snippet embedded in a page |
| `Enumeration` | Enumeration type reference |

## Implementation

The reference tracker is implemented in the catalog package (`mdl/catalog/catalog.go`). It:

1. Parses each document's BSON structure
2. Extracts all qualified name references
3. Classifies each reference by kind
4. Inserts into the `REFS` table with source/target/kind

Transitive queries use recursive SQL CTEs (Common Table Expressions) to follow reference chains.

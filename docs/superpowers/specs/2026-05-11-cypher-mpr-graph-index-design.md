# Cypher MPR Graph Index Design

**Date:** 2026-05-11  
**Status:** Approved

## Problem Statement

The existing `mdl/catalog/` system requires explicit `refresh catalog` commands, persists a separate SQLite file, and only supports SQL queries. For AI-driven bulk refactoring and impact analysis, a graph query language (Cypher) is significantly more expressive — especially for multi-hop traversals like "find all microflows that use any attribute of Entity X through any association path."

MDL remains the right tool for 0→1 creation (new entities, microflows, pages). Cypher becomes the right tool for querying and bulk modification of existing elements.

## Core Insight

MPR files are SQLite databases. The `Units` table already contains `UnitID`, `ContainerUnitID`, `UnitType`, and `contents` (BSON). We can build an in-memory graph index from modelsdk element walks, expose it via PuppyGraph, and run full native Cypher without implementing a parser ourselves.

The "batch update in memory, write back to BSON once" pattern:
- Cypher queries find elements → return UUIDs
- modelsdk loads those elements → mutations accumulated in UnitCache (MarkDirty)
- Single `Flush()` writes all dirty elements atomically

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    Consumer Layer                    │
│   Maia tool calls  │  mxcli cypher  │  MDL executor │
└──────────────────────────┬──────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────┐
│              GraphDB Interface (new: mdl/graphdb/)   │
│  type GraphDB interface {                            │
│    RunCypher(ctx, cypher, params) ([]Record, error)  │
│    Close() error                                     │
│  }                                                   │
│  PuppyGraphAdapter | NebulaAdapter (fallback)        │
└──────────────────────────┬──────────────────────────┘
                           │ Cypher returns element UUIDs
┌──────────────────────────▼──────────────────────────┐
│     In-memory Graph Index SQLite (:memory:, new)     │
│  graph_nodes(id, label, name, module, props_json)    │
│  graph_edges(from_id, to_id, rel_type)               │
│  ← Built by modelsdk Walk at MPR open time           │
│  ← PuppyGraph queries this as its data source        │
└──────────────────────────┬──────────────────────────┘
                 ▲         │
                 │         │ PuppyGraph reads here
┌────────────────┴─────────▼──────────────────────────┐
│              modelsdk (unchanged)                    │
│  Open/OpenForWriting → LoadUnit → SetXxx/MarkDirty   │
│  Flush() → encode dirty elements → MPR SQLite        │
└──────────────────────────┬──────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────┐
│            MPR SQLite (ground truth)                 │
│  Units(UnitID, ContainerUnitID, UnitType, contents)  │
└─────────────────────────────────────────────────────┘
```

## Data Flow

### Query + Bulk Modify

1. `mxcli open MPR` → modelsdk walks all units → fills `:memory:` SQLite `graph_nodes`/`graph_edges` → PuppyGraph connects to this data source
2. Cypher query (Maia or CLI) → `GraphDB.RunCypher(...)` → PuppyGraph executes on `:memory:` DB → returns `[]Record` with element UUIDs
3. Batch in-memory mutation: `for uuid in records → modelsdk.LoadUnit(uuid) → elem.SetXxx() → elem.MarkDirty()`
4. Single write-back: `modelsdk.Flush()` → encodes all dirty elements → batch writes to MPR SQLite

### MDL vs Cypher Division of Labor

| Operation | Tool | Reason |
|-----------|------|--------|
| Create Entity/Microflow/Page | MDL (`create entity ...`) | 0→1, semantic clarity |
| Find / impact analysis | Cypher (`MATCH ... RETURN`) | Multi-hop graph traversal |
| Bulk rename / retype | Cypher find + modelsdk mutate + Flush | In-memory batch, single write-back |
| Fine-grained BSON structure edits | MDL (`alter entity ... add attribute`) | Already fully implemented |

## Graph Schema

### Node Labels

```cypher
(:Module    {id, name, sortIndex})
(:Entity    {id, name, module, isPersistable, generalization})
(:Attribute {id, name, module, entity, dataType, length})
(:Microflow {id, name, module, returnType})
(:NanoFlow  {id, name, module, returnType})
(:Page      {id, name, module, layout})
(:Enumeration  {id, name, module})
(:EnumValue    {id, name, caption})
(:Association  {id, name, module, type, parent, child})
```

### Relationship Types

```cypher
(:Module)-[:HAS_ENTITY]->(:Entity)
(:Module)-[:HAS_MICROFLOW]->(:Microflow)
(:Entity)-[:HAS_ATTRIBUTE]->(:Attribute)
(:Entity)-[:HAS_ASSOCIATION]->(:Association)
(:Association)-[:FROM_ENTITY]->(:Entity)      // ParentPointer (FK owner)
(:Association)-[:TO_ENTITY]->(:Entity)        // ChildPointer (referenced)
(:Microflow)-[:USES_ENTITY]->(:Entity)        // entity references in activities
(:Microflow)-[:CALLS_MICROFLOW]->(:Microflow) // sub-microflow calls
(:Page)-[:USES_ENTITY]->(:Entity)            // page data source
```

### Edge Sources

- **Containment edges**: read directly from `Units.ContainerUnitID` — no BSON parsing needed
- **Cross-reference edges**: extracted by modelsdk Walk at index build time (UUID fields in BSON: AssociationPointer, EntityRef, MicroflowRef, etc.)

### SQLite Schema (in-memory)

```sql
CREATE TABLE graph_nodes (
    id        TEXT PRIMARY KEY,
    label     TEXT NOT NULL,       -- Cypher label (e.g. "Entity")
    name      TEXT,
    module    TEXT,
    props_json TEXT                -- JSON blob for additional properties
);

CREATE TABLE graph_edges (
    from_id   TEXT NOT NULL,
    to_id     TEXT NOT NULL,
    rel_type  TEXT NOT NULL        -- relationship type (e.g. "HAS_ATTRIBUTE")
);

CREATE INDEX idx_edges_from ON graph_edges(from_id);
CREATE INDEX idx_edges_to   ON graph_edges(to_id);
CREATE INDEX idx_nodes_label ON graph_nodes(label);
```

## Example Cypher Queries

```cypher
-- Find all microflows that use the Customer entity
MATCH (mf:Microflow)-[:USES_ENTITY]->(e:Entity {name:'Customer'})
RETURN mf.name, mf.module

-- Impact analysis: what references Customer?
MATCH (x)-[r]->(e:Entity {name:'Customer'})
RETURN type(r), labels(x), x.name

-- Find attributes named 'CreatedDate' across all entities (for bulk rename)
MATCH (a:Attribute {name:'CreatedDate'})<-[:HAS_ATTRIBUTE]-(e:Entity)
RETURN a.id, e.name, a.module

-- Find microflows that call other microflows in the same module (coupling analysis)
MATCH (caller:Microflow)-[:CALLS_MICROFLOW]->(callee:Microflow)
WHERE caller.module = callee.module
RETURN caller.name, callee.name, caller.module
```

## Package Structure

```
mdl/graphdb/
├── interface.go     // GraphDB interface, Record type definition
├── puppygraph.go    // PuppyGraphAdapter (HTTP API to PuppyGraph)
├── nebula.go        // NebulaAdapter (nGQL adapter, fallback)
├── indexer.go       // modelsdk → :memory: SQLite graph index builder
├── mutator.go       // BatchMutator: RunCypherMutation() + Commit()
└── schema.go        // graph_nodes/graph_edges DDL constants
```

### Key Types

```go
// GraphDB is the unified Cypher execution interface.
type GraphDB interface {
    RunCypher(ctx context.Context, cypher string, params map[string]any) ([]Record, error)
    Close() error
}

// Record is one row from a Cypher RETURN clause.
type Record map[string]any

// PuppyGraphAdapter connects to PuppyGraph via its HTTP API.
type PuppyGraphAdapter struct {
    endpoint string
    apiKey   string
}

// Indexer builds the in-memory SQLite graph index from a modelsdk.Model.
type Indexer struct {
    db *sql.DB // :memory:
}

// BatchMutator coordinates Cypher query → in-memory modelsdk mutation → single Flush.
type BatchMutator struct {
    model *modelsdk.Model
    db    GraphDB
}

// RunCypherMutation finds elements via Cypher, applies mutateFn to each in memory.
// Does NOT write to disk — call Commit() when all mutations are done.
func (bm *BatchMutator) RunCypherMutation(
    ctx context.Context,
    findCypher string,
    mutateFn func(elem element.Element) error,
) (int, error)

// Commit writes all dirty in-memory elements back to MPR in a single Flush.
func (bm *BatchMutator) Commit() (int, error)
```

## What We Do Not Implement

- **Cypher parser/lexer**: PuppyGraph handles this entirely
- **Graph traversal engine**: PuppyGraph handles this entirely
- **Persistent catalog**: Replaced by the ephemeral `:memory:` index built at open time

## What Replaces the Catalog

| Old `mdl/catalog/` | New `mdl/graphdb/` |
|--------------------|--------------------|
| Explicit `refresh catalog` command | Auto-built when MPR opens |
| Persistent SQLite file | In-memory SQLite `:memory:` |
| SQL queries via `select ... from CATALOG.table` | Cypher via PuppyGraph |
| Builder files per domain (builder_entities.go, etc.) | Single Indexer walk via modelsdk |

## Implementation Note: SQLite File vs In-Memory

PuppyGraph is a separate process and cannot open an in-memory SQLite DB (`:memory:`) from inside the mxcli process. The graph index SQLite must be a temporary file:

- Location: `os.CreateTemp("", "mxcli-graph-*.db")`
- Lifetime: tied to the mxcli session; deleted on `Close()`
- PuppyGraph connects via JDBC SQLite driver pointing to this temp file path
- mxcli writes the graph_nodes/graph_edges tables, then signals PuppyGraph to connect

This is still "in-memory" from the user's perspective (no persistent catalog file), but physically backed by a temp file for inter-process access.

## Constraints and Decisions

- **PuppyGraph as primary engine**: chosen for native Cypher support and ability to query existing SQLite data sources without ETL
- **NebulaAdapter is optional**: NebulaGraph uses nGQL (Cypher-like), requires auto-conversion; only needed if PuppyGraph is unavailable
- **modelsdk is the write path**: the graph index is read-only; all mutations go through modelsdk setters + MarkDirty + Flush; graph index is rebuilt/updated after Flush if needed
- **Catalog is removed**: `mdl/catalog/` is superseded; existing `show callers/callees/references/impact` commands are re-implemented on top of Cypher

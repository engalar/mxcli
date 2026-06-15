# mxgraph — File-Based Graph Index Framework Design

**Date:** 2026-06-15
**Status:** Design approved — implementation plan pending
**Scope:** A unified, extensible file-based graph index framework (`mxgraph`) providing in-memory graph storage with incremental file persistence, event-driven adapter architecture, and AI-grade graph query capabilities (path schema discovery). First two index types: MPR model + Widget configuration.

---

## 1. Goal

mxcli needs efficient graph-based indexing across multiple data domains: MPR project models, widget configurations, codebase knowledge, Java runtime APIs, and Mendix documentation. Currently each domain has its own ad-hoc indexing (graphify for knowledge, cypher-mpr-graph-index for models). This design unifies them into a single extensible framework:

- **In-memory graph engine** with adjacency lists, label indexes, and property indexes for millisecond traversal
- **Incremental persistence** via append-only delta log + periodic snapshot compaction — avoids full rebuild on every change
- **Event-driven adapter architecture** — each index type is a plugin that emits graph events; engine subscribes and maintains state
- **Path schema discovery** (`FindPathSchemas`) — finds all topological routes between two nodes without loading instance data, enabling AI-driven model exploration

---

## 2. Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Application Layer                       │
│   mxcli commands  │  Maia/AI tools  │  Go API consumers     │
└────────────────────────────┬────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────┐
│                     IndexManager                             │
│  Registers adapters, manages lifecycle, coordinates Build   │
│  /Load/Save across all indexes                              │
└──────┬──────────────────────────────────┬───────────────────┘
       │ GetAdapter(name)                 │ Query()
┌──────▼──────────┐            ┌─────────▼───────────────────┐
│   Adapter Layer │            │      Query API               │
│  mpr/adapter    │            │  FindPathSchemas(from, to)   │
│  widget/adapter │  events →  │  ExplorePath(from, schema)   │
│  (future: ...)  │            │  Traverse(id, rel, depth)    │
└──────┬──────────┘            │  FindNodes(label, props)     │
       │ []Event               │  Neighbors(id, rel...)       │
┌──────▼──────────────────────────────────▼───────────────────┐
│                     Graph Engine                             │
│  NodeIndex:  map[NodeID]*Node                               │
│  Adjacency:  map[NodeID]map[RelType][]NodeID (in/out)       │
│  LabelIndex: map[Label]map[NodeID]bool                      │
│  PropIndex:  map[Label]map[Key]map[Value]map[NodeID]bool    │
│  DirtyTracker: tracks changed nodes/edges                   │
└──────┬──────────────────────────────────────────────────────┘
       │ Apply(events)                 │ Snapshot / Delta
┌──────▼──────────────────────────────▼───────────────────────┐
│                     Persistence Layer                        │
│  Load()   → read snapshot + replay delta.log                │
│  Save()   → write snapshot (protobuf)                       │
│  Compact() → merge delta into snapshot, truncate delta      │
└─────────────────────────────────────────────────────────────┘
```

### File Layout

```
{project_root}/.mxcli/mxgraph/
  snapshot              # GraphSnapshot protobuf (全量)
  delta.log             # append-only 变更日志
  meta.json             # 版本信息、时间戳
```

---

## 3. Graph Engine — Data Model

```go
type NodeID string
type RelType string
type Label string

type Node struct {
    ID       NodeID
    Label    Label
    Props    map[string]any
}

type Edge struct {
    ID       NodeID
    From     NodeID
    To       NodeID
    Type     RelType
    Props    map[string]any
}

type Graph struct {
    mu       sync.RWMutex
    nodes    map[NodeID]*Node
    edges    map[NodeID]*Edge
    outEdges map[NodeID]map[RelType][]NodeID   // outbound adjacency
    inEdges  map[NodeID]map[RelType][]NodeID    // inbound adjacency
    byLabel  map[Label]map[NodeID]bool
    propIdx  map[Label]map[string]map[any]map[NodeID]bool  // label→key→value→{ids}
    dirty    DirtyTracker
}
```

### Event System

```go
type EventType uint8
const (
    NodeCreated EventType = iota
    NodeUpdated
    NodeDeleted
    EdgeCreated
    EdgeDeleted
)

type Event struct {
    Type  EventType
    Node  *Node
    Edge  *Edge
}

func (g *Graph) Apply(events []Event)
// 1. Update in-memory structures (nodes, adjacency, indexes)
// 2. Append to delta.log (via EventSink bridge)
// 3. Mark dirty in DirtyTracker
```

---

## 4. Query API

```go
type GraphQuery interface {
    // Basic navigation
    GetNode(id NodeID) *Node
    Neighbors(id NodeID, relTypes ...RelType) []*Node
    Edges(id NodeID, direction Direction, relTypes ...RelType) []*Edge

    // Path schema discovery (core AI capability)
    FindPathSchemas(from, to NodeID, depthLimit int) []PathSchema
    ExplorePath(from NodeID, schema PathSchema) []PathNode

    // Traversal
    Traverse(start NodeID, rel RelType, depth int) []*Node

    // Filter
    FindNodes(label Label, props map[string]any) []*Node
}

type PathSchema struct {
    Steps []PathStep
    Label string   // e.g. "Entity→Attribute→Microflow"
}

type PathStep struct {
    NodeLabel Label
    RelType   RelType
}

type PathNode struct {
    Node   *Node
    RelOut RelType   // edge to next step; empty for last
}

type Direction uint8
const (Outbound Direction = iota; Inbound; Both)
```

### FindPathSchemas Algorithm

BFS from start node with bounded depth. For each path reaching target, record the (label, rel_type) sequence, deduplicate. Returns topology only (no instance data) — AI chooses which path to explore.

---

### EventSink → Graph + Persistence Bridge

The `EventSink` interface is implemented by `IndexManager`:

```go
func (m *IndexManager) Emit(events []Event) error {
    m.graph.Apply(events)          // update in-memory structures
    m.persist.AppendDelta(events)  // append to delta.log
    return nil
}
```

This ensures every event is simultaneously applied to memory and persisted, keeping them in sync.

---

## 5. Persistence

### Snapshot (Protocol Buffers)

```protobuf
message GraphSnapshot {
    uint64 version = 1;
    int64  timestamp = 2;
    repeated NodeProto nodes = 3;
    repeated EdgeProto edges = 4;
}

message NodeProto {
    string id = 1;
    string label = 2;
    map<string, bytes> props = 3;  // JSON-encoded values
}

message EdgeProto {
    string id = 1;
    string from = 2;
    string to = 3;
    string rel_type = 4;
    map<string, bytes> props = 5;
}
```

### Delta Log (append-only binary)

```
[record_type:1byte] [body_length:4bytes LE] [body]

record_type: 0x01 NodeCreated / 0x02 NodeUpdated / 0x03 NodeDeleted
             0x04 EdgeCreated / 0x05 EdgeDeleted / 0xFF Checkpoint
```

### Lifecycle

```
Startup:
  1. Load meta.json → validate version
  2. Load snapshot → protobuf deserialize → fill Graph
  3. Replay delta.log → apply remaining events to Graph

Runtime:
  1. Adapter produces events → Graph.Apply() updates memory
  2. Persistence appends to delta.log (async buffered)

Shutdown / Periodic:
  1. Serialize full Graph → write temp snapshot file
  2. Append Checkpoint(0xFF) to delta.log
  3. Atomic rename temp → snapshot
  4. Truncate delta.log (entries before Checkpoint)
```

---

## 6. Adapter System

```go
type IndexAdapter interface {
    Name() string
    Schema() *GraphSchema
    Build(ctx context.Context, sink EventSink) error
    Watch(ctx context.Context, sink EventSink) (stop func(), error)
}

type EventSink interface {
    Emit(events []Event) error
}

type GraphSchema struct {
    NodeLabels []Label
    EdgeTypes  []struct { Type RelType; From Label; To Label }
}

type IndexManager struct {
    graph    *Graph
    persist  *Persistence
    adapters map[string]IndexAdapter
}

func (m *IndexManager) RegisterAdapter(a IndexAdapter)
func (m *IndexManager) BuildAll(ctx context.Context) error
func (m *IndexManager) Load() error
func (m *IndexManager) Save() error
func (m *IndexManager) Query() GraphQuery
```

### MPR Adapter

Walks all model units via modelsdk.Produces nodes for Module/Entity/Attribute/Microflow/Page/Association/Enumeration, edges for containment and references.

**Incremental:** Listens to modelsdk dirty callbacks. On entity rename → NodeUpdated. On attribute delete → NodeDeleted + cascading edge deletes.

### Widget Adapter

Reads widget XML/JSON from themesource/. Produces nodes for WidgetPackage/Widget/WidgetProperty. Edges for HAS_WIDGET/DEPENDS_ON/HAS_PROPERTY.

**Incremental:** File watcher on widget directory. File change → re-parse single widget → diff old vs new → emit targeted events.

---

## 7. Package Structure

```
internal/
  mxgraph/                        # 核心引擎
    engine.go                     # Graph struct, Apply, AddNode/Edge, NodeIndex/Adjacency
    query.go                      # GraphQuery impl: FindPathSchemas, ExplorePath, Traverse
    types.go                      # Node, Edge, Event, NodeID, RelType, Label, PathSchema

    persist.go                    # Snapshot read/write, Delta read/write/append, Compact
    persist_test.go

    adapter.go                    # IndexAdapter interface, EventSink, IndexManager

  mxgraph/adapter/
    mpr/
      adapter.go                  # MprAdapter: walks modelsdk, emits events
      schema.go                   # MPR-specific schema (labels, edge types)
      watch.go                    # modelsdk dirty callback → incremental events

    widget/
      adapter.go                  # WidgetAdapter: reads widget configs
      schema.go                   # Widget-specific schema

  mxgraph/                        # (future adapters)
    knowledge/
    doc/
    javaapi/
```

---

## 8. SOLID Compliance

| Principle | How the design satisfies it |
|-----------|---------------------------|
| **S**ingle Responsibility | `Graph` = graph data + operations. `Persistence` = file I/O. `IndexAdapter` = source→events. `IndexManager` = lifecycle. One job each. |
| **O**pen/Closed | New index type = new `IndexAdapter` implementation. Zero changes to engine/query/persistence. |
| **L**iskov Substitution | Adapters all return `[]Event`. Engine doesn't know or care which adapter produced them. |
| **I**nterface Segregation | `IndexAdapter` has 4 methods. `EventSink` has 1. `GraphQuery` has 7. No fat interfaces. |
| **D**ependency Inversion | Engine depends on `Event` (abstraction), not on MPR or Widget types. Adapters depend on `EventSink` (abstraction). |

---

## 9. First Implementation (Phase 1)

Deliver: `internal/mxgraph/` engine + persistence + MPR adapter.

1. `types.go` — Node, Edge, Event, PathSchema, NodeID types
2. `engine.go` — Graph struct with AddNode/AddEdge/Neighbors/Apply
3. `query.go` — FindPathSchemas, ExplorePath, Traverse, FindNodes
4. `persist.go` — Snapshot write/read, Delta append/replay, Compact
5. `adapter.go` — IndexAdapter interface, EventSink, IndexManager
6. `adapter/mpr/adapter.go` — MprAdapter: walk modelsdk, build graph
7. Integration test: MPR → mxgraph → FindPathSchemas between two entities
8. Benchmark: build time, query latency, delta size vs snapshot

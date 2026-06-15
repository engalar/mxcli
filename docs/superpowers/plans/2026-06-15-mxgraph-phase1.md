# mxgraph Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver `internal/mxgraph/` — in-memory graph engine with adjacency indexes, path schema discovery (FindPathSchemas), gob-based snapshot/delta persistence, event-driven adapter system, and MPR model adapter as the first concrete index type.

**Architecture:** Pure in-memory graph with map-based adjacency lists and label/property indexes. Event-driven: adapters emit `[]Event`, engine consumes via `Apply()`. Persistence via gob snapshot + append-only delta log. Query API provides FindPathSchemas (topology-only), ExplorePath (data along selected schema), and basic traversal.

**Tech Stack:** Go standard library (`encoding/gob` for snapshots), modelsdk, `google.golang.org/protobuf` (future migration)

**Spec:** `docs/superpowers/specs/2026-06-15-mxgraph-index-framework-design.md`

---

## File Structure

| Task | File | Role |
|------|------|------|
| 1 | `internal/mxgraph/types.go` | Node, Edge, Event, NodeID, RelType, Label, PathSchema, PathStep, PathNode, Direction |
| 2 | `internal/mxgraph/engine.go` | Graph struct, AddNode/AddEdge/RemoveNode/Apply, adjacency indexes, Neighbors, FindNodes |
| 3 | `internal/mxgraph/query.go` | FindPathSchemas (BFS all-paths), ExplorePath, Traverse |
| 4 | `internal/mxgraph/persist.go` | SnapshotReader/SnapshotWriter (gob), DeltaWriter/DeltaReader, Compact |
| 5 | `internal/mxgraph/adapter.go` | IndexAdapter interface, EventSink, IndexManager |
| 6 | `internal/mxgraph/adapter/mpr/adapter.go` | MprAdapter: modelsdk walk → events |
| 7 | `internal/mxgraph/mxgraph_test.go` | Unit + integration tests for all components |

---

### Task 1: Core types

**Files:**
- Create: `internal/mxgraph/types.go`
- Create: `internal/mxgraph/mxgraph_test.go`

- [ ] **Step 1: Write failing test for type constants**

```go
// internal/mxgraph/mxgraph_test.go
package mxgraph

import "testing"

func TestDirectionValues(t *testing.T) {
    if Outbound != 0 || Inbound != 1 || Both != 2 {
        t.Error("unexpected Direction iota values")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd internal\mxgraph && go test -run TestDirectionValues -v`
Expected: FAIL (package does not exist)

- [ ] **Step 3: Create directory and write types**

```go
// internal/mxgraph/types.go
package mxgraph

type NodeID string
type RelType string
type Label string

type Node struct {
    ID    NodeID
    Label Label
    Props map[string]any
}

type Edge struct {
    ID   NodeID
    From NodeID
    To   NodeID
    Type RelType
    Props map[string]any
}

type EventType uint8

const (
    NodeCreated EventType = iota
    NodeUpdated
    NodeDeleted
    EdgeCreated
    EdgeDeleted
)

type Event struct {
    Type EventType
    Node *Node
    Edge *Edge
}

type PathStep struct {
    NodeLabel Label
    RelType   RelType
}

type PathSchema struct {
    Steps []PathStep
    Label string
}

type PathNode struct {
    Node   *Node
    RelOut RelType
}

type Direction uint8

const (
    Outbound Direction = iota
    Inbound
    Both
)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd internal\mxgraph && go test -run TestDirectionValues -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mxgraph/
git commit -m "feat(mxgraph): core types — Node, Edge, Event, PathSchema, Direction"
```

---

### Task 2: Graph Engine

**Files:**
- Create: `internal/mxgraph/engine.go`

- [ ] **Step 1: Write failing test for Graph operations**

```go
// internal/mxgraph/mxgraph_test.go
func TestGraphAddNode(t *testing.T) {
    g := New()
    g.AddNode("n1", "Entity", map[string]any{"Name": "Payer"})
    n := g.GetNode("n1")
    if n == nil {
        t.Fatal("GetNode returned nil")
    }
    if n.Label != "Entity" {
        t.Errorf("label = %q, want Entity", n.Label)
    }
    if n.Props["Name"] != "Payer" {
        t.Errorf("Name = %v, want Payer", n.Props["Name"])
    }
}

func TestGraphAddEdge(t *testing.T) {
    g := New()
    g.AddNode("m1", "Module", nil)
    g.AddNode("e1", "Entity", nil)
    g.AddEdge("edge1", "m1", "e1", "HAS_ENTITY", nil)

    neighbors := g.Neighbors("m1", "HAS_ENTITY")
    if len(neighbors) != 1 {
        t.Fatalf("Neighbors returned %d, want 1", len(neighbors))
    }
    if neighbors[0].ID != "e1" {
        t.Errorf("neighbor ID = %q, want e1", neighbors[0].ID)
    }
}

func TestGraphRemoveNode(t *testing.T) {
    g := New()
    g.AddNode("e1", "Entity", nil)
    g.AddNode("a1", "Attribute", nil)
    g.AddEdge("edge1", "e1", "a1", "HAS_ATTRIBUTE", nil)

    g.RemoveNode("e1")
    if g.GetNode("e1") != nil {
        t.Error("e1 should be nil after RemoveNode")
    }
    // Edge should be cascaded
    edges := g.Edges("e1", Outbound)
    if len(edges) != 0 {
        t.Errorf("expected 0 edges after node removal, got %d", len(edges))
    }
}

func TestGraphApplyEvents(t *testing.T) {
    g := New()
    g.Apply([]Event{
        {Type: NodeCreated, Node: &Node{ID: "e1", Label: "Entity"}},
        {Type: NodeCreated, Node: &Node{ID: "a1", Label: "Attribute"}},
        {Type: EdgeCreated, Edge: &Edge{ID: "edge1", From: "e1", To: "a1", Type: "HAS_ATTRIBUTE"}},
    })
    if g.GetNode("e1") == nil {
        t.Fatal("node e1 not found after Apply")
    }
    if g.GetNode("a1") == nil {
        t.Fatal("node a1 not found after Apply")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd internal\mxgraph && go test -run "TestGraphAddNode|TestGraphAddEdge|TestGraphRemoveNode|TestGraphApplyEvents" -v`
Expected: FAIL (New, GetNode, AddNode, etc. not defined)

- [ ] **Step 3: Implement Graph struct**

```go
// internal/mxgraph/engine.go
package mxgraph

import "sync"

type Graph struct {
    mu       sync.RWMutex
    nodes    map[NodeID]*Node
    edges    map[NodeID]*Edge
    outEdges map[NodeID]map[RelType][]NodeID
    inEdges  map[NodeID]map[RelType][]NodeID
    byLabel  map[Label]map[NodeID]bool
    propIdx  map[Label]map[string]map[any]map[NodeID]bool
}

func New() *Graph {
    return &Graph{
        nodes:    map[NodeID]*Node{},
        edges:    map[NodeID]*Edge{},
        outEdges: map[NodeID]map[RelType][]NodeID{},
        inEdges:  map[NodeID]map[RelType][]NodeID{},
        byLabel:  map[Label]map[NodeID]bool{},
        propIdx:  map[Label]map[string]map[any]map[NodeID]bool{},
    }
}

func (g *Graph) GetNode(id NodeID) *Node {
    g.mu.RLock()
    defer g.mu.RUnlock()
    return g.nodes[id]
}

func (g *Graph) AddNode(id NodeID, label Label, props map[string]any) {
    g.mu.Lock()
    defer g.mu.Unlock()
    n := &Node{ID: id, Label: label, Props: props}
    g.nodes[id] = n
    if g.byLabel[label] == nil {
        g.byLabel[label] = map[NodeID]bool{}
    }
    g.byLabel[label][id] = true
    g.indexProps(n)
}

func (g *Graph) AddEdge(id, from, to NodeID, rel RelType, props map[string]any) {
    g.mu.Lock()
    defer g.mu.Unlock()
    e := &Edge{ID: id, From: from, To: to, Type: rel, Props: props}
    g.edges[id] = e
    if g.outEdges[from] == nil {
        g.outEdges[from] = map[RelType][]NodeID{}
    }
    g.outEdges[from][rel] = append(g.outEdges[from][rel], to)
    if g.inEdges[to] == nil {
        g.inEdges[to] = map[RelType][]NodeID{}
    }
    g.inEdges[to][rel] = append(g.inEdges[to][rel], from)
}

func (g *Graph) RemoveNode(id NodeID) {
    g.mu.Lock()
    defer g.mu.Unlock()
    n := g.nodes[id]
    if n == nil {
        return
    }
    // Remove from label index
    delete(g.byLabel[n.Label], id)
    // Remove all outbound edges
    for rel, targets := range g.outEdges[id] {
        for _, t := range targets {
            g.removeEdgeFromIndex(t, id, rel)
        }
    }
    delete(g.outEdges, id)
    // Remove all inbound edges
    for rel, sources := range g.inEdges[id] {
        for _, s := range sources {
            g.removeEdgeFromIndex(id, s, rel)
        }
    }
    delete(g.inEdges, id)
    // Remove property index
    g.unindexProps(n)
    delete(g.nodes, id)
}

func (g *Graph) removeEdgeFromIndex(from, to NodeID, rel RelType) {
    if edges, ok := g.outEdges[from]; ok {
        if targets, ok := edges[rel]; ok {
            for i, t := range targets {
                if t == to {
                    edges[rel] = append(targets[:i], targets[i+1:]...)
                    break
                }
            }
        }
    }
}

func (g *Graph) Apply(events []Event) {
    for _, ev := range events {
        switch ev.Type {
        case NodeCreated:
            g.AddNode(ev.Node.ID, ev.Node.Label, ev.Node.Props)
        case NodeUpdated:
            g.mu.Lock()
            if existing := g.nodes[ev.Node.ID]; existing != nil {
                g.unindexProps(existing)
                existing.Props = ev.Node.Props
                g.indexProps(existing)
            }
            g.mu.Unlock()
        case NodeDeleted:
            g.RemoveNode(ev.Node.ID)
        case EdgeCreated:
            g.AddEdge(ev.Edge.ID, ev.Edge.From, ev.Edge.To, ev.Edge.Type, ev.Edge.Props)
        case EdgeDeleted:
            g.mu.Lock()
            delete(g.edges, ev.Edge.ID)
            g.mu.Unlock()
        }
    }
}

func (g *Graph) Neighbors(id NodeID, relTypes ...RelType) []*Node {
    g.mu.RLock()
    defer g.mu.RUnlock()
    var result []*Node
    edges := g.outEdges[id]
    if len(relTypes) == 0 {
        for _, targets := range edges {
            for _, t := range targets {
                if n := g.nodes[t]; n != nil {
                    result = append(result, n)
                }
            }
        }
    } else {
        for _, rel := range relTypes {
            for _, t := range edges[rel] {
                if n := g.nodes[t]; n != nil {
                    result = append(result, n)
                }
            }
        }
    }
    return result
}

func (g *Graph) Edges(id NodeID, dir Direction, relTypes ...RelType) []*Edge {
    g.mu.RLock()
    defer g.mu.RUnlock()
    var result []*Edge
    var edgeMap map[NodeID]map[RelType][]NodeID
    switch dir {
    case Outbound:
        edgeMap = g.outEdges
    case Inbound:
        edgeMap = g.inEdges
    case Both:
        result = append(result, g.Edges(id, Outbound, relTypes...)...)
        result = append(result, g.Edges(id, Inbound, relTypes...)...)
        return result
    }
    filter := len(relTypes) > 0
    for _, e := range g.edges {
        if (dir == Outbound && e.From == id) || (dir == Inbound && e.To == id) {
            if filter {
                for _, rt := range relTypes {
                    if e.Type == rt {
                        result = append(result, e)
                        break
                    }
                }
            } else {
                result = append(result, e)
            }
        }
    }
    _ = edgeMap
    return result
}

func (g *Graph) FindNodes(label Label, props map[string]any) []*Node {
    g.mu.RLock()
    defer g.mu.RUnlock()
    // With props: use propIdx intersection
    if len(props) > 0 {
        var sets []map[NodeID]bool
        for k, v := range props {
            if idx, ok := g.propIdx[label]; ok {
                if vals, ok := idx[k]; ok {
                    if nodes, ok := vals[v]; ok {
                        sets = append(sets, nodes)
                    }
                }
            }
        }
        if len(sets) == 0 {
            return nil
        }
        // Intersection: start with smallest set
        minIdx := 0
        for i := 1; i < len(sets); i++ {
            if len(sets[i]) < len(sets[minIdx]) {
                minIdx = i
            }
        }
        var result []*Node
        for id := range sets[minIdx] {
            match := true
            for i, s := range sets {
                if i == minIdx {
                    continue
                }
                if !s[id] {
                    match = false
                    break
                }
            }
            if match {
                if n := g.nodes[id]; n != nil {
                    result = append(result, n)
                }
            }
        }
        return result
    }
    // Without props: return all nodes with label
    var result []*Node
    for id := range g.byLabel[label] {
        if n := g.nodes[id]; n != nil {
            result = append(result, n)
        }
    }
    return result
}

func (g *Graph) indexProps(n *Node) {
    if g.propIdx[n.Label] == nil {
        g.propIdx[n.Label] = map[string]map[any]map[NodeID]bool{}
    }
    for k, v := range n.Props {
        if g.propIdx[n.Label][k] == nil {
            g.propIdx[n.Label][k] = map[any]map[NodeID]bool{}
        }
        if g.propIdx[n.Label][k][v] == nil {
            g.propIdx[n.Label][k][v] = map[NodeID]bool{}
        }
        g.propIdx[n.Label][k][v][n.ID] = true
    }
}

func (g *Graph) unindexProps(n *Node) {
    for k, v := range n.Props {
        delete(g.propIdx[n.Label][k][v], n.ID)
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd internal\mxgraph && go test -run "TestGraphAddNode|TestGraphAddEdge|TestGraphRemoveNode|TestGraphApplyEvents" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mxgraph/engine.go internal/mxgraph/mxgraph_test.go
git commit -m "feat(mxgraph): Graph engine with adjacency, label/property indexes, event Apply"
```

---

### Task 3: Query API — FindPathSchemas, ExplorePath, Traverse

**Files:**
- Create: `internal/mxgraph/query.go`

- [ ] **Step 1: Write failing test for path queries**

```go
// internal/mxgraph/mxgraph_test.go
func TestFindPathSchemas(t *testing.T) {
    g := buildTestGraph()

    schemas := g.FindPathSchemas("e1", "mf1", 5)
    if len(schemas) == 0 {
        t.Fatal("FindPathSchemas returned 0 schemas")
    }
    // Expected path: Entity →[HAS_ATTRIBUTE]→ Attribute →[USED_IN_MICROFLOW]→ Microflow
    found := false
    for _, s := range schemas {
        if s.Label == "Entity→HAS_ATTRIBUTE→Attribute→USED_IN_MICROFLOW→Microflow" {
            found = true
            break
        }
    }
    if !found {
        t.Errorf("expected path Entity→Attribute→Microflow, got %v", schemas)
    }
}

func TestExplorePath(t *testing.T) {
    g := buildTestGraph()
    schemas := g.FindPathSchemas("e1", "mf1", 5)
    if len(schemas) == 0 {
        t.Fatal("no schemas found")
    }
    path := g.ExplorePath("e1", schemas[0])
    if len(path) < 2 {
        t.Fatalf("ExplorePath returned %d nodes, want >= 2", len(path))
    }
    if path[0].Node.ID != "e1" {
        t.Errorf("first node = %q, want e1", path[0].Node.ID)
    }
    if path[len(path)-1].Node.ID != "mf1" {
        t.Errorf("last node = %q, want mf1", path[len(path)-1].Node.ID)
    }
}

func TestTraverse(t *testing.T) {
    g := buildTestGraph()
    results := g.Traverse("m1", "HAS_ENTITY", 1)
    if len(results) == 0 {
        t.Fatal("Traverse returned 0 results")
    }
    if results[0].ID != "e1" {
        t.Errorf("first result = %q, want e1", results[0].ID)
    }
}

func buildTestGraph() *Graph {
    g := New()
    g.AddNode("m1", "Module", map[string]any{"Name": "MyModule"})
    g.AddNode("e1", "Entity", map[string]any{"Name": "Payer"})
    g.AddNode("a1", "Attribute", map[string]any{"Name": "Status"})
    g.AddNode("mf1", "Microflow", map[string]any{"Name": "ACT_Process"})
    g.AddEdge("edge1", "m1", "e1", "HAS_ENTITY", nil)
    g.AddEdge("edge2", "e1", "a1", "HAS_ATTRIBUTE", nil)
    g.AddEdge("edge3", "a1", "mf1", "USED_IN_MICROFLOW", nil)
    return g
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd internal\mxgraph && go test -run "TestFindPathSchemas|TestExplorePath|TestTraverse" -v`
Expected: FAIL (FindPathSchemas not defined)

- [ ] **Step 3: Implement query methods**

```go
// internal/mxgraph/query.go
package mxgraph

type pathState struct {
    current      NodeID
    visitedNodes map[NodeID]bool
    steps        []PathStep
}

func (g *Graph) FindPathSchemas(from, to NodeID, depthLimit int) []PathSchema {
    g.mu.RLock()
    defer g.mu.RUnlock()

    if g.nodes[from] == nil || g.nodes[to] == nil {
        return nil
    }

    var schemas []PathSchema
    seen := map[string]bool{} // dedup by label sequence

    var dfs func(state pathState, depth int)
    dfs = func(state pathState, depth int) {
        if depth > depthLimit {
            return
        }
        if state.current == to && len(state.steps) > 0 {
            // Build label sequence for dedup
            var labelSeq string
            for _, s := range state.steps {
                labelSeq += string(s.NodeLabel) + "→" + string(s.RelType) + "→"
            }
            if !seen[labelSeq] {
                seen[labelSeq] = true
                schema := PathSchema{
                    Steps: append([]PathStep{}, state.steps...),
                    Label: labelSeq,
                }
                if len(schema.Label) > 0 && schema.Label[len(schema.Label)-1] == '→' {
                    schema.Label = schema.Label[:len(schema.Label)-1]
                }
                schemas = append(schemas, schema)
            }
            return
        }

        for rel, targets := range g.outEdges[state.current] {
            for _, nextID := range targets {
                if state.visitedNodes[nextID] {
                    continue
                }
                nextNode := g.nodes[nextID]
                if nextNode == nil {
                    continue
                }
                state.visitedNodes[nextID] = true
                state.steps = append(state.steps, PathStep{
                    NodeLabel: nextNode.Label,
                    RelType:   rel,
                })
                dfs(pathState{
                    current:      nextID,
                    visitedNodes: state.visitedNodes,
                    steps:        state.steps,
                }, depth+1)
                state.steps = state.steps[:len(state.steps)-1]
                delete(state.visitedNodes, nextID)
            }
        }
    }

    startVisited := map[NodeID]bool{from: true}
    dfs(pathState{
        current:      from,
        visitedNodes: startVisited,
        steps:        nil,
    }, 0)

    return schemas
}

func (g *Graph) ExplorePath(from NodeID, schema PathSchema) []PathNode {
    g.mu.RLock()
    defer g.mu.RUnlock()

    result := []PathNode{
        {Node: g.nodes[from], RelOut: schema.Steps[0].RelType},
    }

    current := from
    for _, step := range schema.Steps {
        // Find the next node along rel
        for _, nextID := range g.outEdges[current][step.RelType] {
            nextNode := g.nodes[nextID]
            if nextNode != nil && nextNode.Label == step.NodeLabel {
                current = nextID
                break
            }
        }
        result = append(result, PathNode{Node: g.nodes[current]})
    }

    return result
}

func (g *Graph) Traverse(start NodeID, rel RelType, depth int) []*Node {
    g.mu.RLock()
    defer g.mu.RUnlock()

    visited := map[NodeID]bool{start: true}
    var result []*Node
    var queue []struct {
        id    NodeID
        depth int
    }
    queue = append(queue, struct {
        id    NodeID
        depth int
    }{start, 0})

    for len(queue) > 0 {
        cur := queue[0]
        queue = queue[1:]
        for _, nextID := range g.outEdges[cur.id][rel] {
            if !visited[nextID] {
                visited[nextID] = true
                n := g.nodes[nextID]
                if n != nil {
                    result = append(result, n)
                }
                if cur.depth+1 < depth {
                    queue = append(queue, struct {
                        id    NodeID
                        depth int
                    }{nextID, cur.depth + 1})
                }
            }
        }
    }
    return result
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd internal\mxgraph && go test -run "TestFindPathSchemas|TestExplorePath|TestTraverse" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mxgraph/query.go
git commit -m "feat(mxgraph): FindPathSchemas, ExplorePath, Traverse query API"
```

---

### Task 4: Persistence — Snapshot + Delta

**Files:**
- Create: `internal/mxgraph/persist.go`

- [ ] **Step 1: Write failing persistence tests**

```go
// internal/mxgraph/mxgraph_test.go
func TestSnapshotRoundtrip(t *testing.T) {
    g := buildTestGraph()
    data, err := MarshalSnapshot(g)
    if err != nil {
        t.Fatalf("MarshalSnapshot: %v", err)
    }
    g2, err := UnmarshalSnapshot(data)
    if err != nil {
        t.Fatalf("UnmarshalSnapshot: %v", err)
    }
    if g2.GetNode("e1") == nil {
        t.Error("e1 missing after roundtrip")
    }
    if len(g2.Neighbors("e1")) != 1 {
        t.Errorf("expected 1 neighbor, got %d", len(g2.Neighbors("e1")))
    }
}

func TestDeltaAppendReplay(t *testing.T) {
    var buf bytes.Buffer
    w := NewDeltaWriter(&buf)
    w.WriteEvent(Event{Type: NodeCreated, Node: &Node{ID: "n1", Label: "Entity"}})
    w.WriteEvent(Event{Type: NodeCreated, Node: &Node{ID: "n2", Label: "Attribute"}})
    w.WriteEvent(Event{Type: EdgeCreated, Edge: &Edge{ID: "e1", From: "n1", To: "n2", Type: "HAS_ATTRIBUTE"}})
    w.Close()

    g := New()
    r := NewDeltaReader(&buf)
    for {
        ev, err := r.ReadEvent()
        if err == io.EOF {
            break
        }
        if err != nil {
            t.Fatalf("ReadEvent: %v", err)
        }
        g.Apply([]Event{ev})
    }
    if g.GetNode("n1") == nil || g.GetNode("n2") == nil {
        t.Error("nodes not restored after delta replay")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd internal\mxgraph && go test -run "TestSnapshotRoundtrip|TestDeltaAppendReplay" -v`
Expected: FAIL (MarshalSnapshot not defined)

- [ ] **Step 3: Implement persistence**

```go
// internal/mxgraph/persist.go
package mxgraph

import (
    "bytes"
    "encoding/binary"
    "encoding/gob"
    "fmt"
    "io"
)

func init() {
    gob.Register(&Node{})
    gob.Register(&Edge{})
    gob.Register(map[string]any{})
    gob.Register([]any{})
}

// ── Snapshot ────────────────────────────────────────────────

type graphSnapshot struct {
    Version   int64
    Nodes     []*Node
    Edges     []*Edge
}

func MarshalSnapshot(g *Graph) ([]byte, error) {
    g.mu.RLock()
    defer g.mu.RUnlock()

    snap := graphSnapshot{
        Version: 1,
        Nodes:   make([]*Node, 0, len(g.nodes)),
        Edges:   make([]*Edge, 0, len(g.edges)),
    }
    for _, n := range g.nodes {
        snap.Nodes = append(snap.Nodes, n)
    }
    for _, e := range g.edges {
        snap.Edges = append(snap.Edges, e)
    }

    var buf bytes.Buffer
    enc := gob.NewEncoder(&buf)
    if err := enc.Encode(snap); err != nil {
        return nil, fmt.Errorf("encode snapshot: %w", err)
    }
    return buf.Bytes(), nil
}

func UnmarshalSnapshot(data []byte) (*Graph, error) {
    var snap graphSnapshot
    dec := gob.NewDecoder(bytes.NewReader(data))
    if err := dec.Decode(&snap); err != nil {
        return nil, fmt.Errorf("decode snapshot: %w", err)
    }

    g := New()
    for _, n := range snap.Nodes {
        g.AddNode(n.ID, n.Label, n.Props)
    }
    for _, e := range snap.Edges {
        g.AddEdge(e.ID, e.From, e.To, e.Type, e.Props)
    }
    return g, nil
}

// ── Delta Log ─────────────────────────────────────────────

type DeltaWriter struct {
    w io.Writer
}

func NewDeltaWriter(w io.Writer) *DeltaWriter {
    return &DeltaWriter{w: w}
}

func (dw *DeltaWriter) WriteEvent(ev Event) error {
    var buf bytes.Buffer
    enc := gob.NewEncoder(&buf)
    if err := enc.Encode(ev); err != nil {
        return fmt.Errorf("encode event: %w", err)
    }
    // Format: [record_type:1] [length:4 LE] [body]
    recordType := byte(ev.Type + 1) // 1-5 to avoid 0
    body := buf.Bytes()
    header := make([]byte, 5)
    header[0] = recordType
    binary.LittleEndian.PutUint32(header[1:5], uint32(len(body)))
    if _, err := dw.w.Write(header); err != nil {
        return err
    }
    _, err := dw.w.Write(body)
    return err
}

func (dw *DeltaWriter) WriteCheckpoint() error {
    header := make([]byte, 5)
    header[0] = 0xFF
    binary.LittleEndian.PutUint32(header[1:5], 0)
    _, err := dw.w.Write(header)
    return err
}

func (dw *DeltaWriter) Close() error {
    if c, ok := dw.w.(io.Closer); ok {
        return c.Close()
    }
    return nil
}

type DeltaReader struct {
    r io.Reader
}

func NewDeltaReader(r io.Reader) *DeltaReader {
    return &DeltaReader{r: r}
}

func (dr *DeltaReader) ReadEvent() (Event, error) {
    header := make([]byte, 5)
    if _, err := io.ReadFull(dr.r, header); err != nil {
        return Event{}, err
    }
    recordType := header[0]
    length := binary.LittleEndian.Uint32(header[1:5])
    if recordType == 0xFF {
        return Event{}, io.EOF // checkpoint marker
    }
    body := make([]byte, length)
    if _, err := io.ReadFull(dr.r, body); err != nil {
        return Event{}, err
    }
    var ev Event
    dec := gob.NewDecoder(bytes.NewReader(body))
    if err := dec.Decode(&ev); err != nil {
        return Event{}, fmt.Errorf("decode event: %w", err)
    }
    return ev, nil
}
```

- [ ] **Step 4: Add imports to test file**

```go
// internal/mxgraph/mxgraph_test.go add:
import (
    "bytes"
    "io"
)
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd internal\mxgraph && go test -run "TestSnapshotRoundtrip|TestDeltaAppendReplay" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/mxgraph/persist.go
git commit -m "feat(mxgraph): gob-based snapshot and delta log persistence"
```

---

### Task 5: Adapter System — IndexAdapter, EventSink, IndexManager

**Files:**
- Create: `internal/mxgraph/adapter.go`
- Modify: `internal/mxgraph/mxgraph_test.go`

- [ ] **Step 1: Write failing test for IndexManager**

```go
// internal/mxgraph/mxgraph_test.go
type testAdapter struct {
    name   string
    schema *GraphSchema
    events []Event
}

func (a *testAdapter) Name() string { return a.name }
func (a *testAdapter) Schema() *GraphSchema { return a.schema }
func (a *testAdapter) Build(ctx context.Context, sink EventSink) error {
    return sink.Emit(a.events)
}
func (a *testAdapter) Watch(ctx context.Context, sink EventSink) (stop func(), error) {
    return func() {}, nil
}

func TestIndexManagerBuild(t *testing.T) {
    m := NewIndexManager()
    m.RegisterAdapter(&testAdapter{
        name: "test",
        schema: &GraphSchema{NodeLabels: []Label{"Entity"}},
        events: []Event{
            {Type: NodeCreated, Node: &Node{ID: "e1", Label: "Entity"}},
        },
    })

    ctx := context.Background()
    if err := m.BuildAll(ctx); err != nil {
        t.Fatalf("BuildAll: %v", err)
    }

    n := m.Query().GetNode("e1")
    if n == nil {
        t.Error("e1 not found after BuildAll")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd internal\mxgraph && go test -run TestIndexManagerBuild -v`
Expected: FAIL (IndexManager not defined)

- [ ] **Step 3: Implement adapter system**

```go
// internal/mxgraph/adapter.go
package mxgraph

import (
    "context"
    "fmt"
)

type GraphSchema struct {
    NodeLabels []Label
    EdgeTypes  []struct {
        Type RelType
        From Label
        To   Label
    }
}

type EventSink interface {
    Emit(events []Event) error
}

type IndexAdapter interface {
    Name() string
    Schema() *GraphSchema
    Build(ctx context.Context, sink EventSink) error
    Watch(ctx context.Context, sink EventSink) (stop func(), error)
}

type IndexManager struct {
    graph    *Graph
    adapters map[string]IndexAdapter
}

func NewIndexManager() *IndexManager {
    return &IndexManager{
        graph:    New(),
        adapters: map[string]IndexAdapter{},
    }
}

func (m *IndexManager) RegisterAdapter(a IndexAdapter) {
    m.adapters[a.Name()] = a
}

func (m *IndexManager) BuildAll(ctx context.Context) error {
    for name, a := range m.adapters {
        if err := a.Build(ctx, m); err != nil {
            return fmt.Errorf("adapter %q Build: %w", name, err)
        }
    }
    return nil
}

func (m *IndexManager) BuildOne(ctx context.Context, name string) error {
    a, ok := m.adapters[name]
    if !ok {
        return fmt.Errorf("adapter %q not registered", name)
    }
    return a.Build(ctx, m)
}

func (m *IndexManager) Emit(events []Event) error {
    m.graph.Apply(events)
    return nil
}

func (m *IndexManager) Query() *Graph {
    return m.graph
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd internal\mxgraph && go test -run TestIndexManagerBuild -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mxgraph/adapter.go
git commit -m "feat(mxgraph): IndexAdapter interface, EventSink, IndexManager"
```

---

### Task 6: MPR Adapter

**Files:**
- Create: `internal/mxgraph/adapter/mpr/adapter.go`
- Test: `internal/mxgraph/adapter/mpr/adapter_test.go`

- [ ] **Step 1: Write test for MprAdapter**

```go
// internal/mxgraph/adapter/mpr/adapter_test.go
package mpr

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/mendixlabs/mxcli/internal/mxgraph/adapter/mpr"
    "github.com/mendixlabs/mxcli/modelsdk"
    _ "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
    _ "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
)

func findTestMPR(t testing.TB) string {
    t.Helper()
    patterns := []string{
        "testdata/corpus-a/app.mpr",
        "testdata/*/app.mpr",
    }
    root := filepath.Join("..", "..", "..", "..", "..")
    for _, p := range patterns {
        matches, _ := filepath.Glob(filepath.Join(root, p))
        if len(matches) > 0 {
            return matches[0]
        }
    }
    return ""
}

type recordingSink struct {
    events []mxgraph.Event
}

func (s *recordingSink) Emit(events []mxgraph.Event) error {
    s.events = append(s.events, events...)
    return nil
}

func TestMprAdapterBuild(t *testing.T) {
    mprPath := findTestMPR(t)
    if mprPath == "" {
        t.Skip("no test MPR found")
    }
    m, err := modelsdk.Open(mprPath)
    if err != nil {
        t.Fatalf("Open: %v", err)
    }
    defer m.Close()

    a := &mpr.Adapter{Model: m}
    sink := &recordingSink{}

    ctx := context.Background()
    if err := a.Build(ctx, sink); err != nil {
        t.Fatalf("Build: %v", err)
    }

    t.Logf("MPR adapter produced %d events", len(sink.events))
    if len(sink.events) == 0 {
        t.Error("expected at least some events from MPR build")
    }

    // Verify at least one DomainModel node was created
    hasDomainModel := false
    for _, ev := range sink.events {
        if ev.Type == mxgraph.NodeCreated && ev.Node.Label == "DomainModel" {
            hasDomainModel = true
            break
        }
    }
    if !hasDomainModel {
        t.Error("expected at least one DomainModel node")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd internal\mxgraph\adapter\mpr && go test -run TestMprAdapterBuild -v`
Expected: FAIL (package does not exist)

- [ ] **Step 3: Create package directory**

```bash
mkdir -p internal\mxgraph\adapter\mpr
```

- [ ] **Step 4: Implement MprAdapter**

```go
// internal/mxgraph/adapter/mpr/adapter.go
package mpr

import (
    "context"
    "fmt"

    "github.com/mendixlabs/mxcli/internal/mxgraph"
    "github.com/mendixlabs/mxcli/modelsdk"
    "github.com/mendixlabs/mxcli/modelsdk/element"
)

type Adapter struct {
    Model *modelsdk.Model
}

func (a *Adapter) Name() string { return "mpr" }

func (a *Adapter) Schema() *mxgraph.GraphSchema {
    return &mxgraph.GraphSchema{
        NodeLabels: []mxgraph.Label{
            "Module", "DomainModel", "Entity", "Attribute", "Microflow",
            "Page", "Layout", "Association", "Enumeration", "EnumValue",
        },
        EdgeTypes: []struct {
            Type mxgraph.RelType
            From mxgraph.Label
            To   mxgraph.Label
        }{
            {"HAS_DOMAIN_MODEL", "Module", "DomainModel"},
            {"HAS_ENTITY", "DomainModel", "Entity"},
            {"HAS_ATTRIBUTE", "Entity", "Attribute"},
            {"HAS_ASSOCIATION", "DomainModel", "Association"},
        },
    }
}

func (a *Adapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
    var events []mxgraph.Event

    for _, unit := range a.Model.Units() {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        elem, err := a.Model.LoadUnit(unit.ID)
        if err != nil {
            continue
        }

        typeName := elem.TypeName()
        switch typeName {
        case "DomainModels$DomainModel":
            n := nodeForUnit(unit.ID, "DomainModel", elem)
            events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: n})

            // Entities
            for _, prop := range elem.Properties() {
                if prop.Name() == "Entities" {
                    if cl, ok := prop.(element.ChildListProperty); ok {
                        for _, child := range cl.ChildElements() {
                            childType := child.TypeName()
                            if childType == "DomainModels$Entity" || childType == "DomainModels$EntityImpl" {
                                cn := nodeForElement(child, "Entity")
                                events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: cn})
                                events = append(events, mxgraph.Event{
                                    Type: mxgraph.EdgeCreated,
                                    Edge: &mxgraph.Edge{
                                        ID: mxgraph.NodeID(fmt.Sprintf("%s->%s", unit.ID, child.ID())),
                                        From: mxgraph.NodeID(unit.ID),
                                        To:   mxgraph.NodeID(child.ID()),
                                        Type: "HAS_ENTITY",
                                    },
                                })

                                // Attributes
                                for _, ap := range child.Properties() {
                                    if ap.Name() == "Attributes" {
                                        if cl2, ok := ap.(element.ChildListProperty); ok {
                                            for _, attr := range cl2.ChildElements() {
                                                an := nodeForElement(attr, "Attribute")
                                                events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: an})
                                                events = append(events, mxgraph.Event{
                                                    Type: mxgraph.EdgeCreated,
                                                    Edge: &mxgraph.Edge{
                                                        ID:   mxgraph.NodeID(fmt.Sprintf("%s->%s", child.ID(), attr.ID())),
                                                        From: mxgraph.NodeID(child.ID()),
                                                        To:   mxgraph.NodeID(attr.ID()),
                                                        Type: "HAS_ATTRIBUTE",
                                                    },
                                                })
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    if len(events) > 0 {
        if err := sink.Emit(events); err != nil {
            return fmt.Errorf("emit events: %w", err)
        }
    }
    return nil
}

func (a *Adapter) Watch(ctx context.Context, sink mxgraph.EventSink) (stop func(), error) {
    return func() {}, nil
}

func nodeForUnit(id element.ID, label mxgraph.Label, elem element.Element) *mxgraph.Node {
    props := map[string]any{}
    props["$Type"] = elem.TypeName()
    for _, p := range elem.Properties() {
        if wp, ok := p.(element.WritableProperty); ok {
            val := wp.BSONValue()
            if val != nil {
                props[p.Name()] = val
            }
        }
    }
    return &mxgraph.Node{ID: mxgraph.NodeID(id), Label: label, Props: props}
}

func nodeForElement(elem element.Element, label mxgraph.Label) *mxgraph.Node {
    props := map[string]any{}
    props["$Type"] = elem.TypeName()
    for _, p := range elem.Properties() {
        if wp, ok := p.(element.WritableProperty); ok {
            val := wp.BSONValue()
            if val != nil {
                props[p.Name()] = val
            }
        }
    }
    return &mxgraph.Node{ID: mxgraph.NodeID(elem.ID()), Label: label, Props: props}
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd internal\mxgraph\adapter\mpr && go test -run TestMprAdapterBuild -v`
Expected: PASS (or SKIP if no MPR found)

- [ ] **Step 6: Commit**

```bash
git add internal/mxgraph/adapter/
git commit -m "feat(mxgraph): MPR adapter — walks modelsdk, emits Entity/Attribute nodes and edges"
```

---

### Task 7: Integration — MPR → mxgraph → Path Query

**Files:**
- Create: `internal/mxgraph/adapter/mpr/adapter_test.go` (add to above)

- [ ] **Step 1: Write end-to-end integration test**

```go
// internal/mxgraph/adapter/mpr/adapter_test.go — add
func TestMprAdapterFindPath(t *testing.T) {
    mprPath := findTestMPR(t)
    if mprPath == "" {
        t.Skip("no test MPR found")
    }
    m, err := modelsdk.Open(mprPath)
    if err != nil {
        t.Fatalf("Open: %v", err)
    }
    defer m.Close()

    // Build graph via IndexManager
    mgr := mxgraph.NewIndexManager()
    mgr.RegisterAdapter(&Adapter{Model: m})
    ctx := context.Background()
    if err := mgr.BuildAll(ctx); err != nil {
        t.Fatalf("BuildAll: %v", err)
    }

    graph := mgr.Query()
    t.Logf("Graph: %d nodes, %d edges", len(graph.AllNodes()), len(graph.AllEdges()))

    // Find all Entities
    entities := graph.FindNodes("Entity", nil)
    if len(entities) == 0 {
        t.Skip("no entities in graph")
    }
    t.Logf("Found %d entities", len(entities))

    // Find path schemas between first entity and its attributes
    entityID := entities[0].ID
    attributes := graph.Neighbors(entityID, "HAS_ATTRIBUTE")
    if len(attributes) == 0 {
        t.Skip("entity has no attributes")
    }

    schemas := graph.FindPathSchemas(entityID, attributes[0].ID, 5)
    t.Logf("Found %d path schemas", len(schemas))
    for i, s := range schemas {
        t.Logf("  Schema[%d]: %s", i, s.Label)
    }

    if len(schemas) == 0 {
        t.Error("expected at least one path schema from entity to its attribute")
    }
}

// Add helper methods for counting (used in test)
func (g *Graph) AllNodes() map[NodeID]*Node { return g.nodes }
func (g *Graph) AllEdges() map[NodeID]*Edge { return g.edges }
```

- [ ] **Step 2: Add AllNodes/AllEdges to engine.go**

Add to `internal/mxgraph/engine.go`:
```go
func (g *Graph) AllNodes() map[NodeID]*Node { return g.nodes }
func (g *Graph) AllEdges() map[NodeID]*Edge { return g.edges }
```

- [ ] **Step 3: Run integration test**

Run: `cd internal\mxgraph\adapter\mpr && go test -run TestMprAdapterFindPath -v`
Expected: PASS with log output showing graph stats and path schemas

- [ ] **Step 4: Run all mxgraph tests**

Run: `cd internal\mxgraph && go test ./... -v -count=1`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mxgraph/
git commit -m "feat(mxgraph): end-to-end integration — MPR → graph → FindPathSchemas"
```

---

### Self-Review

**1. Spec coverage:**
- Graph Engine (types + data model) → Task 1, 2
- Query API (FindPathSchemas, ExplorePath, Traverse, FindNodes) → Task 3
- Persistence (snapshot + delta) → Task 4
- Adapter system (IndexAdapter, EventSink, IndexManager) → Task 5
- MPR Adapter → Task 6
- Integration test → Task 7
- Widget Adapter → deferred to Phase 2 (spec explicitly phases it)

**2. Placeholder scan:** All code blocks contain complete Go code. No TBD/TODO. Each step has exact commands and expected output.

**3. Type consistency:** `NodeID`, `RelType`, `Label`, `PathSchema`, `PathStep`, `PathNode`, `Direction`, `Event`, `EventType` are consistent across all tasks. `Graph.AddNode/AddEdge/RemoveNode/Apply/Neighbors/FindNodes/Edges` signatures match between engine.go and query.go. `IndexAdapter.Name()/Schema()/Build()/Watch()` match adapter.go and the MPR adapter.

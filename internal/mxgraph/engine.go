// Package mxgraph provides an in-memory graph engine with adjacency indexes,
// path schema discovery, event-driven adapter system, and gob-based persistence.
// Adapters produce Event streams consumed by Graph.Apply to build and update the graph.
package mxgraph

import (
	"reflect"
	"sync"
)

type Graph struct {
	mu       sync.RWMutex
	nodes    map[NodeID]*Node
	edges    map[NodeID]*Edge
	outEdges map[NodeID]map[RelType][]NodeID
	inEdges  map[NodeID]map[RelType][]NodeID
	byLabel  map[Label]map[NodeID]bool
	propIdx  map[Label]map[string]map[any]map[NodeID]bool
	// edge ID 索引，用于 O(degree) 的 Edges() 查询。
	// key: node ID, value: relType → []edgeID（NodeID 类型复用，值是边的 ID）
	outEdgeIDs map[NodeID]map[RelType][]NodeID
	inEdgeIDs  map[NodeID]map[RelType][]NodeID
}

func New() *Graph {
	return &Graph{
		nodes:      map[NodeID]*Node{},
		edges:      map[NodeID]*Edge{},
		outEdges:   map[NodeID]map[RelType][]NodeID{},
		inEdges:    map[NodeID]map[RelType][]NodeID{},
		byLabel:    map[Label]map[NodeID]bool{},
		propIdx:    map[Label]map[string]map[any]map[NodeID]bool{},
		outEdgeIDs: map[NodeID]map[RelType][]NodeID{},
		inEdgeIDs:  map[NodeID]map[RelType][]NodeID{},
	}
}

func (g *Graph) AllNodes() map[NodeID]*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.nodes
}

func (g *Graph) AllEdges() map[NodeID]*Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.edges
}

func (g *Graph) GetNode(id NodeID) *Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.nodes[id]
}

func (g *Graph) AddNode(id NodeID, label Label, props map[string]any) {
	g.mu.Lock()
	defer g.mu.Unlock()
	// Clean up existing node with same ID to prevent index corruption
	if existing := g.nodes[id]; existing != nil {
		delete(g.byLabel[existing.Label], id)
		g.unindexProps(existing)
	}
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
	if g.outEdgeIDs[from] == nil {
		g.outEdgeIDs[from] = map[RelType][]NodeID{}
	}
	g.outEdgeIDs[from][rel] = append(g.outEdgeIDs[from][rel], id)
	if g.inEdgeIDs[to] == nil {
		g.inEdgeIDs[to] = map[RelType][]NodeID{}
	}
	g.inEdgeIDs[to][rel] = append(g.inEdgeIDs[to][rel], id)
}

func (g *Graph) RemoveNode(id NodeID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	n := g.nodes[id]
	if n == nil {
		return
	}
	delete(g.byLabel[n.Label], id)
	// 用 outEdgeIDs 删除所有出边（O(out-degree)）
	for rel, eids := range g.outEdgeIDs[id] {
		for _, eid := range eids {
			if e := g.edges[eid]; e != nil {
				g.removeInEdgeFromIndex(e.To, id, rel)
				g.removeFromInEdgeIDs(e.To, rel, eid)
			}
			delete(g.edges, eid)
		}
	}
	delete(g.outEdgeIDs, id)
	delete(g.outEdges, id)
	// 用 inEdgeIDs 删除所有入边（O(in-degree)）
	for rel, eids := range g.inEdgeIDs[id] {
		for _, eid := range eids {
			if e := g.edges[eid]; e != nil {
				g.removeEdgeFromIndex(e.From, id, rel)
				g.removeFromOutEdgeIDs(e.From, rel, eid)
			}
			delete(g.edges, eid)
		}
	}
	delete(g.inEdgeIDs, id)
	delete(g.inEdges, id)
	g.unindexProps(n)
	delete(g.nodes, id)
}

func (g *Graph) removeFromOutEdgeIDs(from NodeID, rel RelType, eid NodeID) {
	if m, ok := g.outEdgeIDs[from]; ok {
		eids := m[rel]
		for i, e := range eids {
			if e == eid {
				m[rel] = append(eids[:i], eids[i+1:]...)
				return
			}
		}
	}
}

func (g *Graph) removeFromInEdgeIDs(to NodeID, rel RelType, eid NodeID) {
	if m, ok := g.inEdgeIDs[to]; ok {
		eids := m[rel]
		for i, e := range eids {
			if e == eid {
				m[rel] = append(eids[:i], eids[i+1:]...)
				return
			}
		}
	}
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

func (g *Graph) removeInEdgeFromIndex(to, from NodeID, rel RelType) {
	if edges, ok := g.inEdges[to]; ok {
		if sources, ok := edges[rel]; ok {
			for i, s := range sources {
				if s == from {
					edges[rel] = append(sources[:i], sources[i+1:]...)
					break
				}
			}
		}
	}
}

func (g *Graph) removeEdgeFromAdj(e *Edge) {
	g.removeEdgeFromIndex(e.From, e.To, e.Type)
	if inEdges, ok := g.inEdges[e.To]; ok {
		if targets, ok := inEdges[e.Type]; ok {
			for i, s := range targets {
				if s == e.From {
					inEdges[e.Type] = append(targets[:i], targets[i+1:]...)
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
			e := g.edges[ev.Edge.ID]
			if e != nil {
				g.removeEdgeFromAdj(e)
				g.removeFromOutEdgeIDs(e.From, e.Type, e.ID)
				g.removeFromInEdgeIDs(e.To, e.Type, e.ID)
			}
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

	relFilter := make(map[RelType]bool, len(relTypes))
	for _, rt := range relTypes {
		relFilter[rt] = true
	}
	useFilter := len(relTypes) > 0

	var result []*Edge
	seen := map[NodeID]bool{} // 防止 Both 模式下重复（自环情况）

	collect := func(idx map[NodeID]map[RelType][]NodeID) {
		for rel, eids := range idx[id] {
			if useFilter && !relFilter[rel] {
				continue
			}
			for _, eid := range eids {
				if seen[eid] {
					continue
				}
				seen[eid] = true
				if e := g.edges[eid]; e != nil {
					result = append(result, e)
				}
			}
		}
	}

	if dir == Outbound || dir == Both {
		collect(g.outEdgeIDs)
	}
	if dir == Inbound || dir == Both {
		collect(g.inEdgeIDs)
	}
	return result
}

func (g *Graph) FindNodes(label Label, props map[string]any) []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if len(props) > 0 {
		var sets []map[NodeID]bool
		for k, v := range props {
			idx, ok := g.propIdx[label]
			if !ok {
				return nil // label not indexed → no match
			}
			vals, ok := idx[k]
			if !ok {
				return nil // key not indexed → no match
			}
			nodes, ok := vals[v]
			if !ok {
				return nil // value not found → no match
			}
			sets = append(sets, nodes)
		}
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
		// Only scalar (comparable) values can be map keys. Slice/map-valued
		// props (e.g. StringList attributes) are not filterable via FindNodes
		// and would panic if used as a key, so skip indexing them.
		if !isHashable(v) {
			continue
		}
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
		if !isHashable(v) {
			continue
		}
		if idx, ok := g.propIdx[n.Label]; ok {
			if vals, ok := idx[k]; ok {
				delete(vals[v], n.ID)
			}
		}
	}
}

// isHashable reports whether v can be used as a Go map key (i.e. is comparable).
// Slices, maps, and functions are not hashable and would panic if used as keys.
func isHashable(v any) bool {
	switch v.(type) {
	case nil:
		return false
	case string, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, uintptr,
		float32, float64,
		complex64, complex128:
		return true
	default:
		return reflect.TypeOf(v).Comparable()
	}
}

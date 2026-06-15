// Package mxgraph provides an in-memory graph engine with adjacency indexes,
// path schema discovery, event-driven adapter system, and gob-based persistence.
// Adapters produce Event streams consumed by Graph.Apply to build and update the graph.
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
	delete(g.byLabel[n.Label], id)
	for rel, targets := range g.outEdges[id] {
		for _, t := range targets {
			g.removeEdgeFromIndex(t, id, rel)
		}
	}
	delete(g.outEdges, id)
	for rel, sources := range g.inEdges[id] {
		for _, s := range sources {
			g.removeEdgeFromIndex(id, s, rel)
		}
	}
	delete(g.inEdges, id)
	g.unindexProps(n)
	for eid, e := range g.edges {
		if e.From == id || e.To == id {
			delete(g.edges, eid)
		}
	}
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
	var result []*Edge
	filter := len(relTypes) > 0
	for _, e := range g.edges {
		match := (dir == Outbound && e.From == id) || (dir == Inbound && e.To == id) || (dir == Both && (e.From == id || e.To == id))
		if !match {
			continue
		}
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
	return result
}

func (g *Graph) FindNodes(label Label, props map[string]any) []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
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
		if idx, ok := g.propIdx[n.Label]; ok {
			if vals, ok := idx[k]; ok {
				delete(vals[v], n.ID)
			}
		}
	}
}

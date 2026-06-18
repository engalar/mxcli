// internal/fkg/fkg.go
package fkg

import (
	"context"
	"fmt"
	"sort"

	"github.com/mendixlabs/mxcli/internal/fkg/concepts"
	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

// New builds the feature knowledge graph from all registered concept adapters
// and returns a Querier over the result.
func New() (Querier, error) {
	mgr := mxgraph.NewIndexManager()
	for _, a := range concepts.All() {
		mgr.RegisterAdapter(a)
	}
	if err := mgr.BuildAll(context.Background(), mgr); err != nil {
		return nil, fmt.Errorf("fkg.New: build: %w", err)
	}
	return &fkgQuerier{graph: mgr.Query()}, nil
}

type fkgQuerier struct {
	graph *mxgraph.Graph
}

var _ Querier           = (*fkgQuerier)(nil)
var _ GuidanceQuerier   = (*fkgQuerier)(nil)
var _ CurriculumQuerier = (*fkgQuerier)(nil)
var _ Orchestrator      = (*fkgQuerier)(nil)

// nodeToSummary converts a raw graph node to the public NodeSummary type.
func nodeToSummary(n *mxgraph.Node) NodeSummary {
	name, _ := n.Props["Name"].(string)
	summary, _ := n.Props["Summary"].(string)
	return NodeSummary{
		ID:      string(n.ID),
		Label:   string(n.Label),
		Name:    name,
		Summary: summary,
	}
}

// Explore returns the seed node plus all nodes and edges reachable within depth hops.
func (q *fkgQuerier) Explore(id string, depth int) (*ExploreResult, error) {
	seedID := mxgraph.NodeID(id)
	seed := q.graph.GetNode(seedID)
	if seed == nil {
		return nil, fmt.Errorf("node %q not found in feature knowledge graph", id)
	}

	result := &ExploreResult{Seed: nodeToSummary(seed)}
	seenNodes := map[mxgraph.NodeID]bool{seedID: true}
	seenEdges := map[mxgraph.NodeID]bool{}
	frontier := []mxgraph.NodeID{seedID}

	for d := 0; d < depth && len(frontier) > 0; d++ {
		var next []mxgraph.NodeID
		for _, cur := range frontier {
			for _, e := range q.graph.Edges(cur, mxgraph.Both) {
				if seenEdges[e.ID] {
					continue
				}
				seenEdges[e.ID] = true
				result.Edges = append(result.Edges, EdgeSummary{
					RelType: string(e.Type),
					From:    string(e.From),
					To:      string(e.To),
				})
				neighborID := e.To
				if e.To == cur {
					neighborID = e.From
				}
				if !seenNodes[neighborID] {
					seenNodes[neighborID] = true
					if n := q.graph.GetNode(neighborID); n != nil {
						result.Nodes = append(result.Nodes, nodeToSummary(n))
						next = append(next, neighborID)
					}
				}
			}
		}
		frontier = next
	}
	return result, nil
}

// conceptEdges are edge types that represent concept topology.
// Curriculum/teaching edges (TEACHES, DEPENDS) are excluded to keep paths focused on domain relationships.
var conceptEdges = map[mxgraph.RelType]bool{
	concepts.Specializes: true,
	concepts.Requires:    true,
	concepts.RelatedTo:   true,
	concepts.HasSyntax:   true,
	concepts.HasSkill:    true,
	concepts.HasPattern:  true,
	concepts.HasExt:      true,
}

// Path discovers concrete paths between two nodes (up to depth 5).
// Only traverses concept topology edges (excludes TEACHES, DEPENDS).
func (q *fkgQuerier) Path(from, to string) ([]PathSchema, error) {
	fromID := mxgraph.NodeID(from)
	toID := mxgraph.NodeID(to)
	if q.graph.GetNode(fromID) == nil || q.graph.GetNode(toID) == nil {
		return nil, nil
	}

	var result []PathSchema
	seenLabels := map[string]bool{}

	var dfs func(current mxgraph.NodeID, steps []PathStep, visited map[mxgraph.NodeID]bool)
	dfs = func(current mxgraph.NodeID, steps []PathStep, visited map[mxgraph.NodeID]bool) {
		if len(steps) >= 5 {
			return
		}
		if current == toID && len(steps) > 0 {
			labelSeq := string(q.graph.GetNode(fromID).Label)
			for _, s := range steps {
				labelSeq += "→" + s.RelType + "→" + s.NodeLabel
			}
			if !seenLabels[labelSeq] {
				seenLabels[labelSeq] = true
				pathSteps := make([]PathStep, len(steps))
				copy(pathSteps, steps)
				result = append(result, PathSchema{Steps: pathSteps, Label: labelSeq})
			}
			return
		}
		for _, e := range q.graph.Edges(current, mxgraph.Both) {
			if !conceptEdges[e.Type] {
				continue
			}
			nextID := e.To
			if e.To == current {
				nextID = e.From
			}
			if visited[nextID] {
				continue
			}
			nextNode := q.graph.GetNode(nextID)
			if nextNode == nil {
				continue
			}
			nextName, _ := nextNode.Props["Name"].(string)

			visited[nextID] = true
			steps = append(steps, PathStep{
				NodeLabel: string(nextNode.Label),
				RelType:   string(e.Type),
				NodeID:    string(nextID),
				NodeName:  nextName,
			})
			dfs(nextID, steps, visited)
			steps = steps[:len(steps)-1]
			delete(visited, nextID)
		}
	}

	dfs(fromID, nil, map[mxgraph.NodeID]bool{fromID: true})
	return result, nil
}

// Schema returns the full ontology skeleton: node types, edge types, and root concepts.
func (q *fkgQuerier) Schema() *SchemaResult {
	allNodes := q.graph.AllNodes()
	allEdges := q.graph.AllEdges()

	nodeCounts := map[mxgraph.Label]int{}
	for _, n := range allNodes {
		nodeCounts[n.Label]++
	}
	edgeCounts := map[mxgraph.RelType]int{}
	for _, e := range allEdges {
		edgeCounts[e.Type]++
	}

	result := &SchemaResult{}
	for label, count := range nodeCounts {
		result.NodeTypes = append(result.NodeTypes, NodeTypeInfo{
			Label: string(label),
			Count: count,
		})
	}
	for rel, count := range edgeCounts {
		result.EdgeTypes = append(result.EdgeTypes, EdgeTypeInfo{
			RelType: string(rel),
			Count:   count,
		})
	}

	// Roots: Concept nodes with no inbound SPECIALIZES edges.
	for _, n := range allNodes {
		if n.Label != concepts.LabelConcept {
			continue
		}
		if len(q.graph.Edges(n.ID, mxgraph.Outbound, concepts.Specializes)) == 0 {
			result.Roots = append(result.Roots, nodeToSummary(n))
		}
	}

	sort.Slice(result.NodeTypes, func(i, j int) bool { return result.NodeTypes[i].Label < result.NodeTypes[j].Label })
	sort.Slice(result.EdgeTypes, func(i, j int) bool { return result.EdgeTypes[i].RelType < result.EdgeTypes[j].RelType })
	sort.Slice(result.Roots, func(i, j int) bool { return result.Roots[i].ID < result.Roots[j].ID })

	return result
}

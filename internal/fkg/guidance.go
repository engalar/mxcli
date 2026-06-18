// internal/fkg/guidance.go
package fkg

import (
	"fmt"
	"sort"

	"github.com/mendixlabs/mxcli/internal/fkg/concepts"
	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

// Guide returns implementation guidance for a concept.
func (q *fkgQuerier) Guide(conceptID string) (*GuidanceResult, error) {
	seedID := mxgraph.NodeID(conceptID)
	seed := q.graph.GetNode(seedID)
	if seed == nil {
		return nil, fmt.Errorf("concept %q not found", conceptID)
	}
	result := &GuidanceResult{Concept: nodeToSummary(seed)}

	for _, e := range q.graph.Edges(seedID, mxgraph.Outbound) {
		n := q.graph.GetNode(e.To)
		if n == nil {
			continue
		}
		switch n.Label {
		case concepts.LabelPattern:
			result.Patterns = append(result.Patterns, nodeToSummary(n))
		case concepts.LabelSyntaxFeature:
			result.SyntaxRefs = append(result.SyntaxRefs, nodeToSummary(n))
		case concepts.LabelSkill:
			result.Skills = append(result.Skills, nodeToSummary(n))
		case concepts.LabelCodeExtension:
			result.Extensions = append(result.Extensions, nodeToSummary(n))
		}
	}

	// Extract ordered steps from pattern → implDetail edges with StepOrder property.
	for _, p := range result.Patterns {
		pid := mxgraph.NodeID(p.ID) // p.ID already includes prefix (e.g. "pattern:overview-page")
		for _, e := range q.graph.Edges(pid, mxgraph.Outbound) {
			n := q.graph.GetNode(e.To)
			if n == nil || n.Label != concepts.LabelImplDetail {
				continue
			}
			if order, ok := n.Props["StepOrder"].(int); ok {
				action, _ := n.Props["StepAction"].(string)
				targetType, _ := n.Props["TargetType"].(string)
				targetName, _ := n.Props["TargetName"].(string)
				desc, _ := n.Props["Summary"].(string)
				hint, _ := n.Props["SyntaxHint"].(string)
				result.Steps = append(result.Steps, GuidanceStep{
					Order:       order,
					Action:      action,
					TargetType:  targetType,
					TargetName:  targetName,
					Description: desc,
					SyntaxHint:  hint,
				})
			}
		}
	}
	sort.Slice(result.Steps, func(i, j int) bool { return result.Steps[i].Order < result.Steps[j].Order })

	return result, nil
}

// Plan returns a curriculum module's concept/skill mapping.
func (q *fkgQuerier) Plan(moduleName string) (*CurriculumPlan, error) {
	mid := mxgraph.NodeID(moduleName)
	m := q.graph.GetNode(mid)
	if m == nil {
		return nil, fmt.Errorf("curriculum module %q not found", moduleName)
	}
	result := &CurriculumPlan{Module: nodeToSummary(m)}

	for _, e := range q.graph.Edges(mid, mxgraph.Outbound) {
		n := q.graph.GetNode(e.To)
		if n == nil {
			continue
		}
		switch n.Label {
		case concepts.LabelCurriculum:
			result.Prerequisites = append(result.Prerequisites, nodeToSummary(n))
		case concepts.LabelConcept:
			result.Concepts = append(result.Concepts, nodeToSummary(n))
		case concepts.LabelSkill:
			result.Skills = append(result.Skills, nodeToSummary(n))
		case concepts.LabelPattern:
			result.Patterns = append(result.Patterns, nodeToSummary(n))
		case concepts.LabelCodeExtension:
			result.Extensions = append(result.Extensions, nodeToSummary(n))
		}
	}
	return result, nil
}

// Orchestrate returns an ordered implementation plan for multiple concepts.
func (q *fkgQuerier) Orchestrate(conceptIDs []string) (*OrchestrationPlan, error) {
	type depInfo struct {
		index     int
		node      *mxgraph.Node
		deps      map[string]bool // concept IDs this depends on
	}

	nodes := make([]*depInfo, 0, len(conceptIDs))
	nodeIndex := map[string]int{}

	for i, id := range conceptIDs {
		n := q.graph.GetNode(mxgraph.NodeID(id))
		if n == nil {
			return nil, fmt.Errorf("concept %q not found", id)
		}
		nodeIndex[id] = i
		nodes = append(nodes, &depInfo{
			index: i,
			node:  n,
			deps:  map[string]bool{},
		})
	}

	// Build dependency graph from REQUIRES inbound edges.
	// If A→REQUIRES→B, then A must be implemented before B's security rules can be configured.
	// So B depends on A (A comes first). We use Inbound to capture the "depends on who requires me" relationship.
	for _, info := range nodes {
		for _, e := range q.graph.Edges(info.node.ID, mxgraph.Inbound, concepts.Requires) {
			sourceID := string(e.From)
			if _, ok := nodeIndex[sourceID]; ok {
				info.deps[sourceID] = true
			}
		}
	}

	// Topological sort (Kahn's algorithm).
	inDegree := map[string]int{}
	for _, info := range nodes {
		id := string(info.node.ID)
		if _, ok := inDegree[id]; !ok {
			inDegree[id] = 0
		}
		for dep := range info.deps {
			inDegree[dep]++
		}
	}

	// Build dependents map for Kahn's algorithm.
	dependents := map[string][]string{}
	for _, info := range nodes {
		id := string(info.node.ID)
		for dep := range info.deps {
			dependents[dep] = append(dependents[dep], id)
		}
	}

	// Reset inDegree: count of unmet dependencies.
	inDeg := map[string]int{}
	for _, info := range nodes {
		id := string(info.node.ID)
		inDeg[id] = len(info.deps)
	}

	queue := []string{}
	for _, info := range nodes {
		id := string(info.node.ID)
		if inDeg[id] == 0 {
			queue = append(queue, id)
		}
	}

	order := []string{}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		order = append(order, cur)
		for _, dep := range dependents[cur] {
			inDeg[dep]--
			if inDeg[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	// Build result.
	plan := &OrchestrationPlan{}
	orderMap := map[string]int{}
	for i, id := range order {
		orderMap[id] = i
	}

	for _, id := range order {
		info := nodes[nodeIndex[id]]
		ns := nodeToSummary(info.node)

		var depNames []string
		for dep := range info.deps {
			depNames = append(depNames, dep)
		}

		// Collect patterns and skills via Guide.
		guide, _ := q.Guide(id)

		step := OrchestrationStep{
			Concept:   ns,
			Order:     orderMap[id] + 1,
			DependsOn: depNames,
		}
		if guide != nil {
			step.Patterns = guide.Patterns
			step.Skills = guide.Skills
		}
		plan.Steps = append(plan.Steps, step)
	}

	return plan, nil
}

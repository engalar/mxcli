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
	seen := map[string]bool{}

	var dfs func(state pathState, depth int)
	dfs = func(state pathState, depth int) {
		if depth > depthLimit {
			return
		}
		if state.current == to && len(state.steps) > 0 {
			labelSeq := string(g.nodes[from].Label)
			for _, s := range state.steps {
				labelSeq += "\u2192" + string(s.RelType) + "\u2192" + string(s.NodeLabel)
			}
			if !seen[labelSeq] {
				seen[labelSeq] = true
				schema := PathSchema{
					Steps: append([]PathStep{}, state.steps...),
					Label: labelSeq,
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
				visitedCopy := make(map[NodeID]bool, len(state.visitedNodes)+1)
				for k, v := range state.visitedNodes {
					visitedCopy[k] = v
				}
				visitedCopy[nextID] = true
				stepsCopy := append([]PathStep{}, state.steps...)
				stepsCopy = append(stepsCopy, PathStep{NodeLabel: nextNode.Label, RelType: rel})
				newState := pathState{current: nextID, visitedNodes: visitedCopy, steps: stepsCopy}
				dfs(newState, depth+1)
			}
		}
	}

	startVisited := map[NodeID]bool{from: true}
	dfs(pathState{current: from, visitedNodes: startVisited, steps: nil}, 0)
	return schemas
}

func (g *Graph) ExplorePath(from NodeID, schema PathSchema) []PathNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := []PathNode{{Node: g.nodes[from]}}
	if len(schema.Steps) == 0 {
		return result
	}
	result[0].RelOut = schema.Steps[0].RelType

	current := from
	for _, step := range schema.Steps {
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
	type queueItem struct {
		id    NodeID
		depth int
	}
	queue := []queueItem{{start, 0}}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, nextID := range g.outEdges[cur.id][rel] {
			if !visited[nextID] {
				visited[nextID] = true
				if n := g.nodes[nextID]; n != nil {
					result = append(result, n)
				}
				if cur.depth+1 < depth {
					queue = append(queue, queueItem{nextID, cur.depth + 1})
				}
			}
		}
	}
	return result
}

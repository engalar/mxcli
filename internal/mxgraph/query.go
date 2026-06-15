package mxgraph

func (g *Graph) FindPathSchemas(from, to NodeID, depthLimit int) []PathSchema {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.nodes[from] == nil || g.nodes[to] == nil {
		return nil
	}

	var schemas []PathSchema
	seen := map[string]bool{}

	// \u5171\u4eab\u56de\u6eaf\u72b6\u6001\uff0c\u4e0d\u518d\u6bcf\u6b65\u590d\u5236 map
	visited := map[NodeID]bool{from: true}
	steps := make([]PathStep, 0, depthLimit)

	var dfs func(current NodeID, depth int)
	dfs = func(current NodeID, depth int) {
		if depth > depthLimit {
			return
		}
		if current == to && len(steps) > 0 {
			labelSeq := string(g.nodes[from].Label)
			for _, s := range steps {
				labelSeq += "\u2192" + string(s.RelType) + "\u2192" + string(s.NodeLabel)
			}
			if !seen[labelSeq] {
				seen[labelSeq] = true
				schemas = append(schemas, PathSchema{
					Steps: append([]PathStep{}, steps...), // \u4ec5\u5728\u627e\u5230\u8def\u5f84\u65f6\u590d\u5236
					Label: labelSeq,
				})
			}
			return
		}

		for rel, targets := range g.outEdges[current] {
			for _, nextID := range targets {
				if visited[nextID] {
					continue
				}
				nextNode := g.nodes[nextID]
				if nextNode == nil {
					continue
				}
				// \u56de\u6eaf\uff1a\u52a0\u5165 \u2192 \u9012\u5f52 \u2192 \u79fb\u9664
				visited[nextID] = true
				steps = append(steps, PathStep{NodeLabel: nextNode.Label, RelType: rel})
				dfs(nextID, depth+1)
				steps = steps[:len(steps)-1]
				delete(visited, nextID)
			}
		}
	}

	dfs(from, 0)
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

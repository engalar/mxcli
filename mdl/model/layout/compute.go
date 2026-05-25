// SPDX-License-Identifier: Apache-2.0

package layout

import (
	"image"
	"sort"
)

// Node is an entity the layout algorithm must place.
type Node struct {
	ID        string // string(genDm.Entity.ID())
	AttrCount int    // len(entity.AttributesItems())
}

// Edge is an association between two entities (undirected for layout purposes).
type Edge struct{ From, To string }

const (
	colSpacing      = 350 // horizontal pixels between columns
	componentGap    = 200 // horizontal pixels between disconnected components
	startY          = 50  // top margin
	rowGap          = 20  // vertical pixels between entities in the same column
	entityHeaderPx  = 40  // fixed header height
	entityAttrPx    = 20  // pixels per attribute
	entityPaddingPx = 30  // bottom padding per entity
	entityMinHeight = 90  // minimum entity height
)

// Compute returns an (X, Y) canvas position for every node.
// The result is deterministic for the same input order.
// Pure function — no I/O, no side effects.
func Compute(nodes []Node, edges []Edge) map[string]image.Point {
	if len(nodes) == 0 {
		return map[string]image.Point{}
	}

	adj := buildAdj(nodes, edges)
	components := connectedComponents(nodes, adj)

	result := map[string]image.Point{}
	nodeByID := map[string]Node{}
	for _, n := range nodes {
		nodeByID[n.ID] = n
	}

	// Sort components largest-first for stable left-to-right placement.
	sort.Slice(components, func(i, j int) bool {
		return len(components[i]) > len(components[j])
	})

	// Collect singleton (isolated) components into one pile so they share a column.
	var multi [][]string
	var singletons []string
	for _, comp := range components {
		if len(comp) == 1 {
			singletons = append(singletons, comp[0])
		} else {
			multi = append(multi, comp)
		}
	}
	if len(singletons) > 0 {
		sort.Strings(singletons)
		multi = append(multi, singletons)
	}

	xOffset := 50 // left margin for the first component
	for _, comp := range multi {
		colAssign := bfsColumns(comp, adj)
		sortedCols := barycenterSort(colAssign, adj)
		pts := assignCoordinates(sortedCols, nodeByID, xOffset)
		for id, pt := range pts {
			result[id] = pt
		}
		// Advance xOffset past the rightmost column of this component.
		maxCol := 0
		for _, c := range colAssign {
			if c > maxCol {
				maxCol = c
			}
		}
		xOffset += (maxCol+1)*colSpacing + componentGap
	}
	return result
}

// buildAdj constructs an undirected adjacency list.
func buildAdj(nodes []Node, edges []Edge) map[string][]string {
	adj := map[string][]string{}
	for _, n := range nodes {
		adj[n.ID] = nil // ensure all nodes present even if isolated
	}
	for _, e := range edges {
		if _, ok := adj[e.From]; !ok {
			return adj // skip edges referencing unknown nodes
		}
		if _, ok := adj[e.To]; !ok {
			return adj
		}
		adj[e.From] = append(adj[e.From], e.To)
		adj[e.To] = append(adj[e.To], e.From)
	}
	return adj
}

// connectedComponents returns groups of node IDs using BFS.
func connectedComponents(nodes []Node, adj map[string][]string) [][]string {
	visited := map[string]bool{}
	var components [][]string
	// Deterministic order: sort node IDs.
	ids := make([]string, len(nodes))
	for i, n := range nodes {
		ids[i] = n.ID
	}
	sort.Strings(ids)

	for _, id := range ids {
		if visited[id] {
			continue
		}
		var comp []string
		queue := []string{id}
		visited[id] = true
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			comp = append(comp, cur)
			neighbours := adj[cur]
			sort.Strings(neighbours) // determinism
			for _, nb := range neighbours {
				if !visited[nb] {
					visited[nb] = true
					queue = append(queue, nb)
				}
			}
		}
		sort.Strings(comp)
		components = append(components, comp)
	}
	return components
}

// bfsColumns assigns a column index (BFS depth) to each node in the component.
// The root is the lowest-degree node (typically a leaf), so BFS forms a
// natural hierarchy left-to-right. Ties broken by ID for determinism.
func bfsColumns(comp []string, adj map[string][]string) map[string]int {
	// Find the root: lowest-degree node (a leaf if possible).
	root := comp[0]
	minDeg := len(adj[root])
	for _, id := range comp[1:] {
		d := len(adj[id])
		if d < minDeg || (d == minDeg && id < root) {
			minDeg = d
			root = id
		}
	}

	col := map[string]int{}
	queue := []string{root}
	col[root] = 0
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		neighbours := adj[cur]
		sort.Strings(neighbours)
		for _, nb := range neighbours {
			if _, seen := col[nb]; !seen {
				col[nb] = col[cur] + 1
				queue = append(queue, nb)
			}
		}
	}
	// Any node not reached by BFS (isolated within the component) gets col 0.
	for _, id := range comp {
		if _, ok := col[id]; !ok {
			col[id] = 0
		}
	}
	return col
}

// barycenterSort orders nodes within each column to minimise edge crossings.
// Uses two passes (left→right then right→left) of the barycenter heuristic.
// Returns columns as [][]string in column-index order.
func barycenterSort(colAssign map[string]int, adj map[string][]string) [][]string {
	// Group by column.
	maxCol := 0
	for _, c := range colAssign {
		if c > maxCol {
			maxCol = c
		}
	}
	cols := make([][]string, maxCol+1)
	for id, c := range colAssign {
		cols[c] = append(cols[c], id)
	}
	for i := range cols {
		sort.Strings(cols[i]) // initial deterministic order
	}

	// rankMap returns the position (0-based) of each id within its column.
	rankMap := func() map[string]float64 {
		rm := map[string]float64{}
		for _, col := range cols {
			for i, id := range col {
				rm[id] = float64(i)
			}
		}
		return rm
	}

	// sortByBarycenter reorders col[c] by the median rank of each node's
	// neighbours in the reference column (adjCol).
	sortByBarycenter := func(c int, rm map[string]float64) {
		type item struct {
			id string
			bc float64
		}
		items := make([]item, len(cols[c]))
		for i, id := range cols[c] {
			var sum float64
			count := 0
			for _, nb := range adj[id] {
				if rank, ok := rm[nb]; ok {
					sum += rank
					count++
				}
			}
			bc := float64(i) // default: keep current position
			if count > 0 {
				bc = sum / float64(count)
			}
			items[i] = item{id, bc}
		}
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].bc != items[j].bc {
				return items[i].bc < items[j].bc
			}
			return items[i].id < items[j].id
		})
		for i, it := range items {
			cols[c][i] = it.id
		}
	}

	// Pass 1: left to right (use left-neighbour ranks).
	for c := 1; c <= maxCol; c++ {
		rm := rankMap()
		sortByBarycenter(c, rm)
	}
	// Pass 2: right to left (use right-neighbour ranks).
	for c := maxCol - 1; c >= 0; c-- {
		rm := rankMap()
		sortByBarycenter(c, rm)
	}

	return cols
}

// entityHeight returns the canvas height of an entity with the given attribute count.
func entityHeight(attrCount int) int {
	h := entityHeaderPx + attrCount*entityAttrPx + entityPaddingPx
	if h < entityMinHeight {
		return entityMinHeight
	}
	return h
}

// assignCoordinates converts ordered columns into (X, Y) positions.
func assignCoordinates(cols [][]string, nodeByID map[string]Node, xOffset int) map[string]image.Point {
	result := map[string]image.Point{}
	for colIdx, col := range cols {
		x := xOffset + colIdx*colSpacing
		y := startY
		for _, id := range col {
			result[id] = image.Point{X: x, Y: y}
			n := nodeByID[id]
			y += entityHeight(n.AttrCount) + rowGap
		}
	}
	return result
}

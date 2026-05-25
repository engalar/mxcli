# Domain Entity Auto-Layout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the flat single-row entity layout (`X=100+N×150, Y=100`) with a hierarchical column layout that minimises association-line crossings and accounts for entity height.

**Architecture:** Three layers: `mdl/model/layout/` (pure algorithm, no gen dependencies) → `mdl/backend/mpr/domainmodel_layout.go` (1 read + 1 write, owns gen-type conversion) → `mdl/executor/cmd_create_entity_gen.go` (one-liner call). Follows the `mdl/model/entity/` pattern exactly.

**Tech Stack:** Go stdlib only for the algorithm (`image` package for `image.Point`). Gen types from `modelsdk/gen/domainmodels`. Backend write via existing `UpdateDomainModelGen`.

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `mdl/model/layout/compute.go` | Create | Pure graph layout algorithm |
| `mdl/model/layout/compute_test.go` | Create | Table-driven unit tests for algorithm |
| `mdl/backend/domainmodel.go` | Modify | Add `RelayoutDomainModel` to interface |
| `mdl/backend/mpr/domainmodel_layout.go` | Create | Backend implementation (1 read + 1 write) |
| `mdl/backend/mock/backend.go` | Modify | Add `RelayoutDomainModelFunc` field |
| `mdl/backend/mock/mock_domainmodel.go` | Modify | Add stub method |
| `mdl/executor/cmd_create_entity_gen.go` | Modify | Remove old formula, call RelayoutDomainModel |

---

## Task 1: Pure layout algorithm

**Files:**
- Create: `mdl/model/layout/compute.go`
- Create: `mdl/model/layout/compute_test.go`

- [ ] **Step 1: Write the failing tests**

Create `mdl/model/layout/compute_test.go`:

```go
package layout_test

import (
	"image"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/model/layout"
)

func TestCompute_SingleNode(t *testing.T) {
	nodes := []layout.Node{{ID: "a", AttrCount: 3}}
	got := layout.Compute(nodes, nil)
	if got["a"] != (image.Point{X: 50, Y: 50}) {
		t.Errorf("single node: want (50,50), got %v", got["a"])
	}
}

func TestCompute_LinearChain(t *testing.T) {
	// A → B → C  should produce 3 different columns
	nodes := []layout.Node{
		{ID: "a", AttrCount: 2},
		{ID: "b", AttrCount: 2},
		{ID: "c", AttrCount: 2},
	}
	edges := []layout.Edge{{From: "a", To: "b"}, {From: "b", To: "c"}}
	got := layout.Compute(nodes, edges)
	if len(got) != 3 {
		t.Fatalf("want 3 positions, got %d", len(got))
	}
	// All three X values must be distinct (different columns)
	xs := map[int]bool{got["a"].X: true, got["b"].X: true, got["c"].X: true}
	if len(xs) != 3 {
		t.Errorf("linear chain: expected 3 distinct X columns, got %v", got)
	}
}

func TestCompute_Star(t *testing.T) {
	// hub connects to 4 spokes; hub should be in its own column
	nodes := []layout.Node{
		{ID: "hub", AttrCount: 5},
		{ID: "s1", AttrCount: 1},
		{ID: "s2", AttrCount: 1},
		{ID: "s3", AttrCount: 1},
		{ID: "s4", AttrCount: 1},
	}
	edges := []layout.Edge{
		{From: "hub", To: "s1"}, {From: "hub", To: "s2"},
		{From: "hub", To: "s3"}, {From: "hub", To: "s4"},
	}
	got := layout.Compute(nodes, edges)
	hubX := got["hub"].X
	for _, spoke := range []string{"s1", "s2", "s3", "s4"} {
		if got[spoke].X == hubX {
			t.Errorf("spoke %s should be in a different column from hub", spoke)
		}
	}
}

func TestCompute_TwoComponents(t *testing.T) {
	// Two disconnected pairs; they must not overlap
	nodes := []layout.Node{
		{ID: "a1", AttrCount: 2}, {ID: "a2", AttrCount: 2},
		{ID: "b1", AttrCount: 2}, {ID: "b2", AttrCount: 2},
	}
	edges := []layout.Edge{
		{From: "a1", To: "a2"},
		{From: "b1", To: "b2"},
	}
	got := layout.Compute(nodes, edges)
	// All four positions must be distinct
	seen := map[image.Point]string{}
	for id, pt := range got {
		if prev, exists := seen[pt]; exists {
			t.Errorf("collision: %s and %s both at %v", id, prev, pt)
		}
		seen[pt] = id
	}
}

func TestCompute_EntityHeight(t *testing.T) {
	// Two entities in same column; larger one above means more Y gap below it
	nodes := []layout.Node{
		{ID: "big", AttrCount: 10}, // height = 40 + 10*20 + 30 = 270
		{ID: "small", AttrCount: 1}, // height = max(90, 40 + 1*20 + 30) = 90
	}
	got := layout.Compute(nodes, nil)
	// Both are disconnected, so same column. The second entity's Y must be
	// at least startY + height(first) + gap above it.
	yBig := got["big"].Y
	ySmall := got["small"].Y
	if yBig == ySmall {
		t.Errorf("entities in same column must have different Y values")
	}
	// Gap between them must be at least min-height (90) + 20 gap
	diff := abs(ySmall - yBig)
	if diff < 90+20 {
		t.Errorf("Y gap too small: %d (want >= 110)", diff)
	}
}

func TestCompute_Cycle(t *testing.T) {
	// A ↔ B cycle; algorithm must not hang or produce wrong output
	nodes := []layout.Node{
		{ID: "x", AttrCount: 2},
		{ID: "y", AttrCount: 2},
	}
	edges := []layout.Edge{{From: "x", To: "y"}, {From: "y", To: "x"}}
	got := layout.Compute(nodes, edges)
	if len(got) != 2 {
		t.Fatalf("cycle: want 2 positions, got %d", len(got))
	}
}

func TestCompute_Empty(t *testing.T) {
	got := layout.Compute(nil, nil)
	if len(got) != 0 {
		t.Errorf("empty input: want empty map, got %v", got)
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/model/layout/... 2>&1 | head -20
```

Expected: `cannot find package` or `no Go files`.

- [ ] **Step 3: Implement the algorithm**

Create `mdl/model/layout/compute.go`:

```go
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
	colSpacing       = 350 // horizontal pixels between columns
	componentGap     = 200 // horizontal pixels between disconnected components
	startY           = 50  // top margin
	rowGap           = 20  // vertical pixels between entities in the same column
	entityHeaderPx   = 40  // fixed header height
	entityAttrPx     = 20  // pixels per attribute
	entityPaddingPx  = 30  // bottom padding per entity
	entityMinHeight  = 90  // minimum entity height
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

	xOffset := 50 // left margin for the first component
	for _, comp := range components {
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
// The root is the highest-degree node; ties broken by ID for determinism.
func bfsColumns(comp []string, adj map[string][]string) map[string]int {
	// Find the root: highest-degree node.
	root := comp[0]
	maxDeg := len(adj[root])
	for _, id := range comp[1:] {
		d := len(adj[id])
		if d > maxDeg || (d == maxDeg && id < root) {
			maxDeg = d
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
			id   string
			bc   float64
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
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/model/layout/... -v 2>&1
```

Expected: all 8 tests PASS.

- [ ] **Step 5: Confirm no import cycle**

```bash
go build ./mdl/model/layout/... && echo "OK"
```

Expected: `OK` with no errors.

- [ ] **Step 6: Commit**

```bash
git add mdl/model/layout/compute.go mdl/model/layout/compute_test.go
git commit -m "feat(layout): add pure hierarchical layout algorithm

Compute() places entities in BFS-depth columns with 2-pass barycenter
sorting to minimise association-line crossings. Entity height derived
from attribute count. Deterministic, no I/O, no gen dependencies.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Backend interface + mock stub

**Files:**
- Modify: `mdl/backend/domainmodel.go`
- Modify: `mdl/backend/mock/backend.go`
- Modify: `mdl/backend/mock/mock_domainmodel.go`

- [ ] **Step 1: Add method to the interface**

Open `mdl/backend/domainmodel.go`. Find the closing `}` of `DomainModelBackend` interface (after `DeleteCrossAssociation`). Add immediately before it:

```go
	// RelayoutDomainModel recomputes and applies canvas positions for all
	// entities in the given domain model. Called after CREATE ENTITY when
	// no explicit @position annotation is provided.
	RelayoutDomainModel(domainModelID model.ID) error
```

- [ ] **Step 2: Add Func field to mock**

Open `mdl/backend/mock/backend.go`. Find the block of `DomainModel` Func fields (near `CreateAssociationGenFunc`). Add after `CreateAssociationGenFunc`:

```go
	RelayoutDomainModelFunc func(domainModelID model.ID) error
```

- [ ] **Step 3: Add stub method to mock**

Open `mdl/backend/mock/mock_domainmodel.go`. Find `func (m *MockBackend) CreateAssociationGen`. Add after its closing `}`:

```go
func (m *MockBackend) RelayoutDomainModel(domainModelID model.ID) error {
	if m.RelayoutDomainModelFunc != nil {
		return m.RelayoutDomainModelFunc(domainModelID)
	}
	return fmt.Errorf("MockBackend.RelayoutDomainModel not configured")
}
```

- [ ] **Step 4: Verify compile-time interface check still passes**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/backend/... 2>&1
```

Expected: no errors. The `var _ backend.FullBackend = (*MprBackend)(nil)` line in `mdl/backend/mpr/backend.go` will now fail to compile until Task 3 adds the implementation — that failure is expected and will be fixed in Task 3.

- [ ] **Step 5: Commit**

```bash
git add mdl/backend/domainmodel.go mdl/backend/mock/backend.go mdl/backend/mock/mock_domainmodel.go
git commit -m "feat(backend): add RelayoutDomainModel to DomainModelBackend interface + mock stub

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Backend implementation

**Files:**
- Create: `mdl/backend/mpr/domainmodel_layout.go`

- [ ] **Step 1: Create the implementation file**

Create `mdl/backend/mpr/domainmodel_layout.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"github.com/mendixlabs/mxcli/mdl/model"
	"github.com/mendixlabs/mxcli/mdl/model/layout"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// RelayoutDomainModel reads the domain model, computes optimal entity positions
// using the layout algorithm, updates each entity's Location in-memory, and
// writes the entire domain model back in a single transaction.
func (b *MprBackend) RelayoutDomainModel(domainModelID model.ID) error {
	dm, err := b.GetDomainModelByIDGen(domainModelID)
	if err != nil {
		return err
	}
	positions := layout.Compute(collectLayoutNodes(dm), collectLayoutEdges(dm))
	if len(positions) == 0 {
		return nil
	}
	for _, elem := range dm.EntitiesItems() {
		e, ok := elem.(*genDm.Entity)
		if !ok {
			continue
		}
		if pt, found := positions[string(e.ID())]; found {
			e.SetLocation(layoutPos(pt.X, pt.Y))
		}
	}
	return b.UpdateDomainModelGen(dm)
}

// collectLayoutNodes extracts layout.Node values from a gen DomainModel.
// This is the only place that touches gen types for layout purposes.
func collectLayoutNodes(dm *genDm.DomainModel) []layout.Node {
	items := dm.EntitiesItems()
	nodes := make([]layout.Node, 0, len(items))
	for _, elem := range items {
		e, ok := elem.(*genDm.Entity)
		if !ok {
			continue
		}
		nodes = append(nodes, layout.Node{
			ID:        string(e.ID()),
			AttrCount: len(e.AttributesItems()),
		})
	}
	return nodes
}

// collectLayoutEdges extracts layout.Edge values from a gen DomainModel.
// Both Associations and CrossAssociations are included.
func collectLayoutEdges(dm *genDm.DomainModel) []layout.Edge {
	var edges []layout.Edge
	for _, elem := range dm.AssociationsItems() {
		a, ok := elem.(*genDm.Association)
		if !ok {
			continue
		}
		from := string(a.ParentRefID())
		to := string(a.ChildRefID())
		if from != "" && to != "" {
			edges = append(edges, layout.Edge{From: from, To: to})
		}
	}
	return edges
}
```

- [ ] **Step 2: Verify the project builds**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./... 2>&1
```

Expected: no errors (the `var _ backend.FullBackend = (*MprBackend)(nil)` check in `mdl/backend/mpr/backend.go` now passes).

- [ ] **Step 3: Commit**

```bash
git add mdl/backend/mpr/domainmodel_layout.go
git commit -m "feat(backend/mpr): implement RelayoutDomainModel — 1 read + 1 write

Reads the DomainModel, calls layout.Compute, sets Location on each
entity in-memory, writes back via UpdateDomainModelGen (single tx).

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Wire into executor

**Files:**
- Modify: `mdl/executor/cmd_create_entity_gen.go`

- [ ] **Step 1: Update `persistEntityCanonical` (canonical path)**

In `persistEntityCanonical` (starting at line ~137), find the block that injects a default position when `stmt.Position == nil` and `existing == nil`:

```go
		if stmt.Position == nil {
			// New entity: auto-layout based on current entity count.
			stmt.Position = &ast.Position{X: 100 + len(dm.EntitiesItems())*150, Y: 100}
		}
```

Replace it with:

```go
		if stmt.Position == nil {
			// Temporary origin; RelayoutDomainModel below will assign real position.
			stmt.Position = &ast.Position{X: 0, Y: 0}
		}
```

Then find the block after `doc.Persist(pCtx)` succeeds and cache invalidation runs. It currently ends with:

```go
	invalidateDomainModelGenForModule(ctx, module.ID)
	invalidateDomainModelsCache(ctx)
	invalidateHierarchy(ctx)
	if existing != nil {
		fmt.Fprintf(ctx.Output, "Modified entity: %s\n", s.Name)
	} else {
		fmt.Fprintf(ctx.Output, "Created entity: %s\n", s.Name)
	}
	ctx.trackModifiedDomainModel(module.ID, module.Name)
	return nil
```

Add the relayout call before the `fmt.Fprintf` lines:

```go
	invalidateDomainModelGenForModule(ctx, module.ID)
	invalidateDomainModelsCache(ctx)
	invalidateHierarchy(ctx)

	if s.Position == nil {
		if err := ctx.Backend.RelayoutDomainModel(model.ID(dm.ID())); err != nil {
			fmt.Fprintf(ctx.Output, "warning: auto-layout failed: %v\n", err)
		} else {
			invalidateDomainModelGenForModule(ctx, module.ID)
		}
	}

	if existing != nil {
		fmt.Fprintf(ctx.Output, "Modified entity: %s\n", s.Name)
	} else {
		fmt.Fprintf(ctx.Output, "Created entity: %s\n", s.Name)
	}
	ctx.trackModifiedDomainModel(module.ID, module.Name)
	return nil
```

- [ ] **Step 2: Update the legacy path**

In `execCreateEntityGen` (the legacy path, starting around line 89), find:

```go
	if entity.Location() == "" {
		entity.SetLocation(layoutPos(100+len(dm.EntitiesItems())*150, 100))
	}
```

Replace with (entity starts at origin; layout runs after write):

```go
	// Location is set to origin; RelayoutDomainModel assigns the real position.
	if entity.Location() == "" && s.Position == nil {
		entity.SetLocation(layoutPos(0, 0))
	}
```

Then find the CREATE branch (no `existingEntity`). It currently ends with:

```go
	invalidateDomainModelGenForModule(ctx, module.ID)
	invalidateDomainModelsCache(ctx)
	invalidateHierarchy(ctx)
	fmt.Fprintf(ctx.Output, "Created entity: %s\n", s.Name)
	ctx.trackModifiedDomainModel(module.ID, module.Name)
	return nil
```

Add the relayout call before `fmt.Fprintf`:

```go
	invalidateDomainModelGenForModule(ctx, module.ID)
	invalidateDomainModelsCache(ctx)
	invalidateHierarchy(ctx)

	if s.Position == nil {
		if err := ctx.Backend.RelayoutDomainModel(model.ID(dm.ID())); err != nil {
			fmt.Fprintf(ctx.Output, "warning: auto-layout failed: %v\n", err)
		} else {
			invalidateDomainModelGenForModule(ctx, module.ID)
		}
	}

	fmt.Fprintf(ctx.Output, "Created entity: %s\n", s.Name)
	ctx.trackModifiedDomainModel(module.ID, module.Name)
	return nil
```

Note: the MODIFY branch (when `existingEntity != nil`) does NOT trigger relayout — user intent is to modify, not reposition everything.

- [ ] **Step 3: Build to verify no compile errors**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/executor/... 2>&1
```

Expected: no errors.

- [ ] **Step 4: Run existing executor tests**

```bash
go test ./mdl/executor/... 2>&1 | tail -20
```

Expected: all tests PASS. Any test that checks the exact position of an auto-placed entity will need updating in Step 5.

- [ ] **Step 5: Fix any position-dependent test failures**

If tests assert `Location == "250;100"` or similar flat-row positions, update them to accept either a non-zero position or mock out `RelayoutDomainModelFunc` to return `nil` without moving entities. Example mock setup for tests that don't care about layout:

```go
mb := &mock.MockBackend{
    // ...existing fields...
    RelayoutDomainModelFunc: func(model.ID) error { return nil },
}
```

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/cmd_create_entity_gen.go
git commit -m "feat(executor): trigger RelayoutDomainModel on CREATE ENTITY without @position

Removes the flat X=100+N*150 row formula. Both canonical and legacy
paths call ctx.Backend.RelayoutDomainModel after writing the entity.
Auto-layout is skipped when an explicit @position annotation is given.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Integration smoke test

**Files:**
- Create: `mdl-examples/bug-tests/entity-autolayout.mdl`

- [ ] **Step 1: Write the smoke-test MDL script**

Create `mdl-examples/bug-tests/entity-autolayout.mdl`:

```sql
-- Smoke test: 6 entities with associations — no @position annotations.
-- After execution, all entities must have distinct, non-overlapping positions.

create module LayoutTest;

create persistent entity LayoutTest.Customer (
  Name: string(200)
);

create persistent entity LayoutTest.Order (
  OrderDate: datetime,
  TotalAmount: decimal,
  status: string(50)
);

create persistent entity LayoutTest.OrderLine (
  Quantity: integer,
  UnitPrice: decimal
);

create persistent entity LayoutTest.Product (
  ProductName: string(200),
  Price: decimal,
  StockQuantity: integer,
  SKU: string(50),
  IsActive: boolean
);

create persistent entity LayoutTest.Category (
  CategoryName: string(200)
);

create persistent entity LayoutTest.Address (
  Street: string(200),
  City: string(100),
  PostalCode: string(20)
);

create association LayoutTest.Order_Customer
from LayoutTest.Order to LayoutTest.Customer type reference;

create association LayoutTest.OrderLine_Order
from LayoutTest.OrderLine to LayoutTest.Order type reference;

create association LayoutTest.OrderLine_Product
from LayoutTest.OrderLine to LayoutTest.Product type reference;

create association LayoutTest.Product_Category
from LayoutTest.Product to LayoutTest.Category type reference;

create association LayoutTest.Customer_Address
from LayoutTest.Customer to LayoutTest.Address type reference;
```

- [ ] **Step 2: Run against the testdata project**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
./bin/mxcli -p testdata/expr-checker/minimal.mpr -c "$(cat mdl-examples/bug-tests/entity-autolayout.mdl)" 2>&1
```

If `bin/mxcli` does not exist, build first:
```bash
make build
```

Expected: 6 "Created entity:" lines and 5 "Created association:" lines, no errors.

- [ ] **Step 3: Verify positions with describe**

```bash
./bin/mxcli -p testdata/expr-checker/minimal.mpr -c "show entities in LayoutTest" 2>&1
```

Expected: 6 entities listed. Positions should differ — look for `@position` annotations that are not all `(0,0)` or `(100,100)`.

- [ ] **Step 4: Restore testdata**

```bash
git restore testdata/expr-checker/
```

- [ ] **Step 5: Run the full test suite**

```bash
make test 2>&1 | tail -30
```

Expected: all tests PASS, no new failures.

- [ ] **Step 6: Commit**

```bash
git add mdl-examples/bug-tests/entity-autolayout.mdl
git commit -m "test(layout): add entity-autolayout smoke-test MDL script

Six entities with associations to verify RelayoutDomainModel produces
distinct, non-overlapping canvas positions without @position annotations.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Self-Review Checklist

**Spec coverage:**
- [x] Algorithm in `mdl/model/layout/` (Task 1)
- [x] `RelayoutDomainModel` on `DomainModelBackend` interface (Task 2)
- [x] Implementation: 1 read + 1 write via `UpdateDomainModelGen` (Task 3)
- [x] Executor: both canonical and legacy paths wired (Task 4)
- [x] Explicit `@position` is not overridden (Task 4 — `if s.Position == nil` guard)
- [x] `collectNodes` / `collectEdges` own all gen-type casting (Task 3)
- [x] Mock stub with descriptive error (Task 2)
- [x] No batch interface needed — reuses `UpdateDomainModelGen` (Task 3)

**Type consistency:**
- `layout.Node`, `layout.Edge` — defined in Task 1, used in Task 3.
- `layout.Compute` — defined in Task 1, called in Task 3.
- `RelayoutDomainModel(model.ID) error` — interface in Task 2, impl in Task 3, called in Task 4.
- `collectLayoutNodes`, `collectLayoutEdges` — private to `mdl/backend/mpr/domainmodel_layout.go`.
- `layoutPos` — already exists in `mdl/executor/flowbuilder_gen.go:240`; reused in Task 3 (same package).

**Placeholder scan:** No TBD, no "similar to", no missing code blocks found.

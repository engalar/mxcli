# Domain Entity Auto-Layout Design

**Date**: 2026-05-25
**Status**: Approved

## Problem

When `CREATE ENTITY` is executed without an explicit `@position` annotation, entities are placed at
`X = 100 + N×150, Y = 100` — a flat single row with 150 px gaps. With more than a handful of
entities the canvas becomes an unreadable horizontal strip. Associations draw long diagonal lines
with many crossings.

The skill doc (`generate-domain-model.md`) already describes a column-based layout with
attribute-height awareness, but the code never implemented it.

## Goal

Every `CREATE ENTITY` without an explicit `@position` triggers a full re-layout of all entities
in the domain model. The algorithm places related entities in adjacent columns, minimises
association-line crossings, and accounts for entity height (derived from attribute count).

## Non-Goals

- Re-layout when `@position` is explicitly supplied (user intent is preserved).
- Re-layout on `CREATE ASSOCIATION` (associations are written after entities; the layout at
  entity-creation time already includes any associations that exist in the DM at that moment).
- Providing a standalone `ALTER DOMAIN MODEL LAYOUT` command (out of scope for this change).

## Architecture: Three-Layer Design

Follows the canonical `mdl/model/entity/` pattern exactly:

```
Algorithm layer  mdl/model/layout/   pure function, stdlib-only, no gen types
Backend layer    mdl/backend/mpr/    1 read + 1 write per call, owns gen conversion
Executor layer   mdl/executor/       one-liner: ctx.Backend.RelayoutDomainModel(dmID)
```

### Layer 1 — Algorithm (`mdl/model/layout/compute.go`)

New package, parallel to `mdl/model/entity/`. Zero external dependencies.

```go
package layout

type Node struct {
    ID        string // string(genDm.Entity.ID())
    AttrCount int    // len(entity.AttributesItems())
}

type Edge struct{ From, To string } // Node IDs

// Compute is a pure function: deterministic, no I/O, no gen types.
func Compute(nodes []Node, edges []Edge) map[string]image.Point
```

Internal steps (each a private function):

| Function | Input | Output |
|---|---|---|
| `buildAdj` | `[]Node`, `[]Edge` | `map[string][]string` undirected adjacency |
| `connectedComponents` | adjacency map | `[][]string` grouped by component |
| `bfsColumns` | one component, adjacency | `map[id]int` column index (BFS depth from highest-degree root) |
| `barycenterSort` | column→[]id, adjacency | columns sorted to minimise crossings (2-pass barycenter) |
| `assignCoordinates` | sorted columns, `[]Node` | `map[id]image.Point` |

Coordinate rules (aligned with `generate-domain-model.md`):

```
entityHeight(n) = max(90, 40 + n.AttrCount×20 + 30)
colX            = componentStartX + colIndex × 350
startY          = 50
rowY            = prevY + prevEntityHeight + 20   // 20 px gap between entities
componentGap    = 200 px horizontal
```

Barycenter heuristic (Sugiyama standard, 2 passes):
- Pass 1 left→right: order entities in column c by median rank of their col c-1 neighbours.
- Pass 2 right→left: order entities in column c by median rank of their col c+1 neighbours.
- Two passes reduce crossings by ~30% vs one pass.

Special cases:

| Scenario | Handling |
|---|---|
| Single entity, no edges | `(50, 50)` |
| All entities disconnected | Groups of 4 per column, stacked vertically |
| Cycles (bidirectional refs) | BFS on undirected graph; cycles not a problem |
| NPE with 0 attributes | `entityHeight = 90` (minimum) |

### Layer 2 — Backend Interface + Implementation

**Interface** (`mdl/backend/domainmodel.go`):

```go
type DomainModelBackend interface {
    // existing methods: ListDomainModelsGen, GetDomainModelGen, GetDomainModelByIDGen,
    // UpdateDomainModelGen, CreateEntityGen, UpdateEntityGen, MoveEntityGen,
    // CreateAssociationGen, DeleteEntity, DeleteAttribute, DeleteAssociation, etc.
    RelayoutDomainModel(domainModelID model.ID) error
}
```

**Implementation** (`mdl/backend/mpr/domainmodel_layout.go`, new file):

```go
func (b *MprBackend) RelayoutDomainModel(dmID model.ID) error {
    dm, err := b.GetDomainModelByIDGen(dmID)       // 1 read
    if err != nil { return err }

    positions := layout.Compute(collectNodes(dm), collectEdges(dm))

    for _, elem := range dm.EntitiesItems() {
        e, ok := elem.(*genDm.Entity)
        if !ok { continue }
        if pt, ok := positions[string(e.ID())]; ok {
            e.SetLocation(layoutPos(pt.X, pt.Y))
        }
    }
    return b.UpdateDomainModelGen(dm)              // 1 write
}

// collectNodes and collectEdges are private helpers in the same file.
// They own all gen-type casting, keeping layout.Compute free of gen imports.
func collectNodes(dm *genDm.DomainModel) []layout.Node { ... }
func collectEdges(dm *genDm.DomainModel) []layout.Edge { ... }
```

`UpdateDomainModelGen` already writes the entire DomainModel unit atomically — no new batch
interface is needed.

**Mock stub** (`mdl/backend/mock/`):

```go
// backend.go
RelayoutDomainModelFunc func(model.ID) error

// mock_domainmodel.go
func (m *MockBackend) RelayoutDomainModel(id model.ID) error {
    if m.RelayoutDomainModelFunc != nil { return m.RelayoutDomainModelFunc(id) }
    return fmt.Errorf("MockBackend.RelayoutDomainModel not configured")
}
```

### Layer 3 — Executor (`mdl/executor/cmd_create_entity_gen.go`)

Only the canonical path (`persistEntityCanonical`) and the legacy path need changes.

**Canonical path** — after `doc.Persist(pCtx)` and cache invalidation:

```go
if s.Position == nil {
    if err := ctx.Backend.RelayoutDomainModel(model.ID(dm.ID())); err != nil {
        fmt.Fprintf(ctx.Output, "warning: auto-layout: %v\n", err)
    }
    invalidateDomainModelGenForModule(ctx, module.ID)
}
```

**Legacy path** — after `ctx.Backend.CreateEntityGen(...)` and cache invalidation:

```go
if s.Position == nil {
    if err := ctx.Backend.RelayoutDomainModel(model.ID(dm.ID())); err != nil {
        fmt.Fprintf(ctx.Output, "warning: auto-layout: %v\n", err)
    }
    invalidateDomainModelGenForModule(ctx, module.ID)
}
```

The old inline formula `X = 100 + len(dm.EntitiesItems())*150, Y = 100` is removed from both
paths. New entities get `(0, 0)` as a temporary position before `RelayoutDomainModel` overwrites
it — Studio Pro renders `(0, 0)` entities at the origin, which is fine because the layout call
immediately follows.

## File Changelist

| File | Action | Notes |
|---|---|---|
| `mdl/model/layout/compute.go` | Create | Pure algorithm, stdlib-only |
| `mdl/model/layout/compute_test.go` | Create | Table-driven unit tests |
| `mdl/backend/domainmodel.go` | Modify | Add `RelayoutDomainModel` to interface |
| `mdl/backend/mpr/domainmodel_layout.go` | Create | 1-read + 1-write implementation |
| `mdl/backend/mock/backend.go` | Modify | Add `RelayoutDomainModelFunc` field |
| `mdl/backend/mock/mock_domainmodel.go` | Modify | Add stub method |
| `mdl/executor/cmd_create_entity_gen.go` | Modify | Remove old formula; call `RelayoutDomainModel` |

## Testing

**Unit tests** (`mdl/model/layout/compute_test.go`):
- Empty graph → single entity at `(50, 50)`
- Linear chain A→B→C → three columns
- Star topology (hub + 4 spokes) → hub in col 0, spokes in col 1
- Two disconnected components → placed side by side with 200 px gap
- Large entity (20 attrs) followed by small entity (2 attrs) → correct Y cumulation
- Cycle (A→B, B→A) → deterministic output (BFS on undirected graph)

**Integration** (existing `mdl-examples/doctype-tests/`):
- Existing entity tests must pass unchanged (no regression on `@position` entities).
- New `mdl-examples/bug-tests/` script: create 6 entities with associations, verify all
  positions differ and none overlap.

## Design Decisions

**Q: Should entities with explicit `@position` be excluded from relayout?**
A: No — all entities are relayouted every time. If the user wants to pin an entity, they call
`ALTER ENTITY set position` after all entities are created. This keeps the algorithm simple and
results consistent.

**Q: Why `mdl/model/layout/` not `mdl/executor/`?**
A: The executor imports the backend, not the other way around. Placing the algorithm in
`mdl/model/layout/` lets both the backend (for `RelayoutDomainModel`) and tests import it without
a cycle.

**Q: Why not a new `BatchUpdateEntityLocationsGen` backend method?**
A: `UpdateDomainModelGen` already writes the entire DomainModel unit atomically (all entities +
associations in one BSON blob, one SQLite transaction). Mutating entity locations in-memory before
calling it achieves the same result with no new interface surface.

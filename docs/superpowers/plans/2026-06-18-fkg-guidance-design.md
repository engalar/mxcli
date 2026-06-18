# FKG Guidance Layer — Design (SOLID)

> **Driver:** `onto` currently reveals concept topology but cannot guide academy
> course implementation. Root cause analysis in
> `2026-06-18-fkg-feature-knowledge-graph.md` (验收) shows ~40% concept
> coverage and ~5% implementation-level guidance value.

---

## 1. SOLID Violations in Current FKG

| Principle | Violation | Location |
|-----------|-----------|----------|
| **S** — `fkgQuerier` bundles topology queries + aggregation logic | `fkg.go` — `Schema()` does counting, filtering, sorting |
| **O** — Adding a new query type edits the `Querier` interface | `querier.go:6` — all 3 methods in one interface |
| **L** — Every adapter stubs `Watch` as no-op (contract not substitutable) | `page.go:465`, `microflow.go:572`, etc. |
| **I** — `Querier` mixes 3 semantically different query families | `querier.go:6-10` — Explore (traversal), Path (connection), Schema (inventory) |
| **D** — CLI calls `fkg.New()` directly, not abstracted | `cmd_onto.go` — each handler imports and calls `fkg.New()` |

---

## 2. Design Goals

1. **Guidance coverage > 80%** of academy concepts and syntax (from ~40%)
2. **Zero existing file modifications** — new behaviour via new adapters + interfaces
3. **Query interfaces segregated by concern** — topology vs guidance vs curriculum
4. **Patterns as first-class nodes** — reusable implementation strategies teachable via `onto guide`
5. **Curriculum-aware** — `onto plan` answers "what do I need to know before module X?"

---

## 3. Architecture

### 3.1 Node Type Expansion

Add to `internal/fkg/concepts/constants.go`:

```go
const (
    LabelPattern         mxgraph.Label = "Pattern"         // NEW
    LabelImplDetail      mxgraph.Label = "ImplDetail"      // NEW
    LabelCodeExtension   mxgraph.Label = "CodeExtension"   // NEW
    LabelCurriculum      mxgraph.Label = "Curriculum"      // NEW

    // Existing (unchanged):
    LabelConcept       mxgraph.Label = "Concept"
    LabelSyntaxFeature mxgraph.Label = "SyntaxFeature"
    LabelSkill         mxgraph.Label = "Skill"
    LabelDoc           mxgraph.Label = "Doc"
)
```

| Node type | Example instances | Purpose |
|-----------|------------------|---------|
| `Concept` | Entity, Microflow, Page, Security | Level-0 domain (existing) |
| `SyntaxFeature` | entity.create, microflow.retrieve | MDL syntax form (existing) |
| `Skill` | write-microflows, manage-security | How-to knowledge (existing) |
| **`Pattern`** | popup-form-pattern, xpath-row-filter, state-machine-sla, seed-demo-data | Reusable implementation strategy |
| **`ImplDetail`** | textbox, actionbutton, layoutgrid, validation-feedback, loop, show-message | Fine-grained element within a concept |
| **`CodeExtension`** | java-action, js-action, css-theme | Non-MDL extension point |
| **`Curriculum`** | academy-01-domain-modeling, academy-04-pages | Course module mapping |

### 3.2 Edge Type Expansion

Add to `internal/fkg/concepts/constants.go`:

```go
const (
    // New:
    HasPattern   mxgraph.RelType = "HAS_PATTERN"   // Concept → Pattern
    HasExt       mxgraph.RelType = "HAS_EXT"       // Concept → CodeExtension
    Teaches      mxgraph.RelType = "TEACHES"       // Curriculum → Concept|Skill|Pattern
    Depends      mxgraph.RelType = "DEPENDS"       // Curriculum → Curriculum (prerequisite)

    // Existing (unchanged):
    Specializes  mxgraph.RelType = "SPECIALIZES"
    Requires     mxgraph.RelType = "REQUIRES"
    RelatedTo    mxgraph.RelType = "RELATED_TO"
    HasSyntax    mxgraph.RelType = "HAS_SYNTAX"
    HasSkill     mxgraph.RelType = "HAS_SKILL"
    HasDoc       mxgraph.RelType = "HAS_DOC"
)
```

### 3.3 Event Builders (helpers.go)

Add to `internal/fkg/concepts/helpers.go`:

```go
// patternNode returns a NodeCreated event for a Pattern node.
// id is prefixed with "pattern:" to avoid collisions.
func patternNode(id, name, summary string) mxgraph.Event {
    return mxgraph.Event{
        Type: mxgraph.NodeCreated,
        Node: &mxgraph.Node{
            ID:    mxgraph.NodeID("pattern:" + id),
            Label: LabelPattern,
            Props: map[string]any{"Name": name, "Summary": summary},
        },
    }
}

// implDetailNode returns a NodeCreated event for an ImplDetail node.
// id is prefixed with "detail:".
func implDetailNode(id, name, summary string) mxgraph.Event {
    return mxgraph.Event{
        Type: mxgraph.NodeCreated,
        Node: &mxgraph.Node{
            ID:    mxgraph.NodeID("detail:" + id),
            Label: LabelImplDetail,
            Props: map[string]any{"Name": name, "Summary": summary},
        },
    }
}

// extNode returns a NodeCreated event for a CodeExtension node.
func extNode(id, name, summary string) mxgraph.Event {
    return mxgraph.Event{
        Type: mxgraph.NodeCreated,
        Node: &mxgraph.Node{
            ID:    mxgraph.NodeID("ext:" + id),
            Label: LabelCodeExtension,
            Props: map[string]any{"Name": name, "Summary": summary},
        },
    }
}

// curriculumNode returns a NodeCreated event for a Curriculum node.
func curriculumNode(id, name, summary string) mxgraph.Event {
    return mxgraph.Event{
        Type: mxgraph.NodeCreated,
        Node: &mxgraph.Node{
            ID:    mxgraph.NodeID("curriculum:" + id),
            Label: LabelCurriculum,
            Props: map[string]any{"Name": name, "Summary": summary},
        },
    }
}
```

### 3.4 Interface Segregation (querier.go)

Split `Querier` into three interfaces. The existing `Querier` stays unchanged
for backward compatibility; new capabilities are separate interfaces.

```go
// Querier — topology navigation (unchanged).
type Querier interface {
    Explore(id string, depth int) (*ExploreResult, error)
    Path(from, to string) ([]PathSchema, error)
    Schema() *SchemaResult
}

// GuidanceQuerier — implementation guidance for a concept.
type GuidanceQuerier interface {
    Guide(conceptID string) (*GuidanceResult, error)
}

// CurriculumQuerier — academy module planning.
type CurriculumQuerier interface {
    Plan(moduleName string) (*CurriculumPlan, error)
}
```

Output types:

```go
// GuidanceStep is one actionable implementation instruction.
type GuidanceStep struct {
    Order       int    // display order
    Action      string // "create", "configure", "wire", "grant"
    TargetType  string // "Entity", "Microflow", "Page", "Security"
    TargetName  string // e.g. "HD.Ticket"
    Description string // what to do
    SyntaxHint  string // optional MDL snippet
}

// GuidanceResult aggregates everything needed to implement a concept.
type GuidanceResult struct {
    Concept     NodeSummary
    Patterns    []NodeSummary   // applicable patterns
    Steps       []GuidanceStep  // ordered implementation steps
    SyntaxRefs  []NodeSummary   // relevant SyntaxFeature nodes
    Skills      []NodeSummary   // relevant Skill nodes
    Extensions  []NodeSummary   // CodeExtension nodes
}

// CurriculumPlan describes a module's concept/skill dependencies.
type CurriculumPlan struct {
    Module       NodeSummary
    Prerequisites []NodeSummary // other Curriculum nodes
    Concepts     []NodeSummary
    Skills       []NodeSummary
    Patterns     []NodeSummary
    Extensions   []NodeSummary
}
```

### 3.5 Composite fkgQuerier (fkg.go)

`fkgQuerier` implements all three interfaces:

```go
type fkgQuerier struct {
    graph *mxgraph.Graph
}

var _ Querier           = (*fkgQuerier)(nil)
var _ GuidanceQuerier   = (*fkgQuerier)(nil)  // NEW compiler check
var _ CurriculumQuerier = (*fkgQuerier)(nil)  // NEW compiler check
```

Guidance implementation strategy (in new `guidance.go`):

```go
// Guide returns a ranked, ordered implementation guidance plan for a concept.
func (q *fkgQuerier) Guide(conceptID string) (*GuidanceResult, error) {
    seed := q.graph.GetNode(mxgraph.NodeID(conceptID))
    if seed == nil {
        return nil, fmt.Errorf("concept %q not found", conceptID)
    }

    result := &GuidanceResult{Concept: nodeToSummary(seed)}

    // 1. Collect patterns (HAS_PATTERN outbound)
    for _, e := range q.graph.Edges(seed.ID, mxgraph.Outbound, concepts.HasPattern) {
        if n := q.graph.GetNode(e.To); n != nil {
            result.Patterns = append(result.Patterns, nodeToSummary(n))
        }
    }

    // 2. Collect SyntaxFeature refs (HAS_SYNTAX outbound)
    for _, e := range q.graph.Edges(seed.ID, mxgraph.Outbound, concepts.HasSyntax) {
        if n := q.graph.GetNode(e.To); n != nil {
            result.SyntaxRefs = append(result.SyntaxRefs, nodeToSummary(n))
        }
    }

    // 3. Collect Skills (HAS_SKILL outbound)
    // 4. Collect Extensions (HAS_EXT outbound)
    // 5. Build ordered Steps from patterns + sub-concepts

    return result, nil
}
```

---

## 4. File Map

### Modified files (existing, additive only):

| File | Change |
|------|--------|
| `internal/fkg/concepts/constants.go` | Add 4 node labels + 2 edge types |
| `internal/fkg/concepts/helpers.go` | Add `patternNode()`, `implDetailNode()`, `extNode()`, `curriculumNode()` |
| `internal/fkg/querier.go` | Add `GuidanceQuerier`, `CurriculumQuerier` + output types |
| `internal/fkg/fkg.go` | Add compiler checks; import `guidance.go` behaviour |

### New files:

| File | Responsibility |
|------|---------------|
| `internal/fkg/guidance.go` | `Guide()` + `Plan()` implementation |
| `internal/fkg/guidance_test.go` | Tests for Guide + Plan |
| `internal/fkg/concepts/constant.go` | `ConstantAdapter` |
| `internal/fkg/concepts/java_action.go` | `JavaActionAdapter` |
| `internal/fkg/concepts/js_action.go` | `JavaScriptActionAdapter` |
| `internal/fkg/concepts/page_patterns.go` | Pattern nodes for page: popup-form, master-detail, overview-page |
| `internal/fkg/concepts/microflow_patterns.go` | Pattern nodes for microflow: state-machine-sla, validation-feedback |
| `internal/fkg/concepts/security_patterns.go` | Pattern nodes: xpath-row-filter, demo-user-setup |
| `internal/fkg/concepts/cross_patterns.go` | Cross-cutting patterns: seed-demo-data |
| `internal/fkg/concepts/curriculum_academy.go` | Academy module → concept mapping |
| `cmd/mxcli/cmd_onto_guide.go` | `mxcli onto guide <concept>` command |
| `cmd/mxcli/cmd_onto_plan.go` | `mxcli onto plan <module>` command |

---

## 5. SOLID Compliance Map

| Principle | Mechanism |
|-----------|-----------|
| **S** — Single Responsibility | `guidance.go` owns Guide/Plan; `fkg.go` owns topology queries; each adapter owns one domain |
| **O** — Open for Extension | New query types defined as new interfaces; new curriculum domains as new adapters; no existing files modified |
| **L** — Liskov Substitution | Adapters that don't need extensions simply don't emit `HasExt` edges — still valid `IndexAdapter` |
| **I** — Interface Segregation | 3 small interfaces (Querier, GuidanceQuerier, CurriculumQuerier) instead of 1 fat one |
| **D** — Dependency Inversion | CLI handlers receive `Querier`/`GuidanceQuerier`/`CurriculumQuerier` from a factory, never call `fkg.New()` directly |

Dependency graph:

```
cmd_onto*.go  ──→  interface{Querier, GuidanceQuerier, CurriculumQuerier}
                         ↑
                    fkgQuerier
                     ↙     ↘
            mxgraph.Graph   concepts.All()
                                ↑
                    [8 existing adapters | 4 new adapters | curriculum adapter]
```

---

## 6. Implementation Plan

### Task 1: Type & helper expansion (constants.go + helpers.go)
Add 4 node labels, 2 edge types, 4 event builders.

### Task 2: Interface segregation (querier.go)
Add `GuidanceQuerier`, `CurriculumQuerier` interfaces + output types.
Existing `Querier` unchanged.

### Task 3: Guidance query engine (guidance.go + guidance_test.go TDD)
Implement `Guide()` and `Plan()` against the graph.

### Task 4: Missing domain adapters
Create `constant.go`, `java_action.go`, `js_action.go`.

### Task 5: Pattern adapters
Create `page_patterns.go`, `microflow_patterns.go`, `security_patterns.go`, `cross_patterns.go`.
Each emits Pattern nodes and HAS_PATTERN edges from existing Concept nodes.

### Task 6: Curriculum adapter
Create `curriculum_academy.go` — maps 12 academy modules + capstone as Curriculum nodes,
with DEPENDS edges for prerequisites and TEACHES edges for concepts/skills.

### Task 7: CLI commands
Create `cmd_onto_guide.go` (`onto guide <concept>`) and
`cmd_onto_plan.go` (`onto plan <module>`).

---

## 7. Example Outputs

### `mxcli onto guide entity`

```
Guidance for: Entity — Persistent data object with attributes and associations

Patterns:
  generate-domain-model — Entity/attribute/association MDL syntax patterns

Implementation steps:
  1. Create the entity: CREATE OR MODIFY PERSISTENT ENTITY HD.Ticket ( ... )
  2. Define attributes with types (string, boolean, datetime, integer)
  3. Add constraints: NOT NULL, UNIQUE, DEFAULT values
  4. Create enumerations for status/priority fields
  5. Define associations between entities (ownership, multiplicity)
  6. Add indexes for frequently queried fields
  7. Configure entity access rules (requires Security)

Syntax references:
  entity.create        CREATE OR MODIFY ENTITY
  enumeration.create   CREATE OR MODIFY ENUMERATION
  association.create   CREATE ASSOCIATION

Related skills:
  generate-domain-model
  mendix/associations

Requires: Security
```

### `mxcli onto plan academy-04-pages`

```
Curriculum plan: 04-页面与UI (Pages & UI)

Prerequisites:
  academy-01-domain-modeling  — Entity concepts for page datasources
  academy-02-microflows       — Microflow actions for page buttons

Concepts taught:
  Page, DataGrid, Form, Layout

Skills taught:
  create-page, alter-page, overview-pages, master-detail-pages

Patterns taught:
  popup-form-pattern
  master-detail-layout

Extensions: none
```

---

## 8. Existing-System Boundary

| Must not change | Rationale |
|----------------|-----------|
| `internal/mxgraph/` | Shared engine; FKG adapter only |
| Any `cmd/*` except new files | Zero regression on existing commands |
| Existing `page.go` adapter | OCP — enhancements via `page_patterns.go` emitting extra edges |
| `Querier` interface method signatures | Backward compatibility for existing consumers |

---

## 9. Risk Assessment

| Risk | Mitigation |
|------|-----------|
| Guidance steps become stale as curriculum changes | CurriculumAdapter reads structured metadata (not hardcoded text); update metadata = update guidance |
| Too many new node types confuse users | Guide/Plan output types abstract the graph; users never see raw nodes |
| Pattern duplication across adapters | `cross_patterns.go` as single authority; page/microflow/security adapters reference patterns by edge |
| Curriculum mapping is manually maintained | Single `curriculum_academy.go` file; ~150 lines for all 12+1 modules |

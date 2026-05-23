# Canonical Model Layer — Design Spec

**Date**: 2026-05-23  
**Status**: Draft  
**Goal**: Eliminate MDL round-trip drift by introducing a single canonical model layer that both the write path (parse → BSON) and the read path (BSON → describe) must pass through.

---

## Problem Statement

The MDL pipeline currently has two independent serialization paths that diverge:

**Write path**: MDL text → ANTLR4 parse → AST (strong-typed Go structs) → executor → gen `element.Element` (dynamic, `TypeName()`-based) → BSON

**Read path**: BSON → gen `element.Element` → describe via `switch TypeName()` string matching → MDL text

Because the two paths make independent semantic decisions, `describe(execute(mdl))` can produce output that differs from the original `mdl`. This is round-trip drift. The `diff` command is equally affected — its "proposed" and "current" views use separate serialization functions (`entityStmtToMDL` vs `entityToMDLGen` in `cmd_diff_mdl.go`), so diff accuracy degrades whenever the two diverge.

---

## Solution: Canonical Model Layer

Insert a new `mdl/model/` layer as the **single semantic center**. Both paths must pass through it. Serialization logic lives here once, not twice.

```
                    ┌─────────────────────────────────────────┐
MDL 文本 ──parse──→ AST                                        │
                    │  Registry.LiftFrom()                    │
                    ↓                                         │
               Persistable ──── Persist(ctx) ──────────────→ BSON
                    │                                         │
               Document    ←─── HydrateFrom() ←── gen ←───── BSON
                    │
                    └── ToMDL() ──→ MDL 文本
```

The canonical model layer exposes four operations per document type:

| Operation | Direction | Pure? | Notes |
|-----------|-----------|-------|-------|
| `Lift(ast.Statement) → Persistable` | AST → model | Yes | No ctx, no project access |
| `Hydrate(element.Element) → Document` | gen → model | Yes | Defensive; returns warnings |
| `ToMDL() string` | model → text | Yes | Deterministic, sorted output |
| `Persist(PersistContext) error` | model → BSON | No | Via backend abstraction |

---

## SOLID Design

### Package Structure

```
mdl/model/
├── doc.go          # Document, Persistable, Diffable interfaces; Warning type
├── registry.go     # DocumentRegistry interface + Codec registration (OCP)
├── context.go      # LiftContext, HydrateContext, PersistContext value types
│
├── entity/
│   ├── model.go    # EntityModel — data only, no behavior
│   ├── lift.go     # liftEntity(ast) → *EntityModel
│   ├── hydrate.go  # hydrateEntity(gen) → *EntityModel, []Warning
│   ├── serialize.go# (*EntityModel).ToMDL() string
│   ├── persist.go  # (*EntityModel).Persist(ctx) error
│   └── init.go     # model.Register[*ast.CreateEntityStmt](Codec{...})
│
├── microflow/
│   └── ...         # same structure
│
└── page/
    └── ...         # same structure
```

### S — Single Responsibility

Each file has exactly one concern. `entity/model.go` holds data. `entity/serialize.go` holds serialization. No file does two things. `EntityModel` itself has no methods beyond `ToMDL()` and `Persist()` — it does not validate, does not call backend, does not parse.

### O — Open/Closed

The framework (`registry.go`) never mentions `entity`, `microflow`, or `page` by name. Each document type registers itself via `init()`:

```go
// entity/init.go
func init() {
    model.Register[*ast.CreateEntityStmt](model.Codec{
        LiftFn:    func(s ast.Statement) (model.Persistable, error) {
            return liftEntity(s.(*ast.CreateEntityStmt))
        },
        HydrateFn: func(el element.Element) (model.Document, []model.Warning, error) {
            return hydrateEntity(el)
        },
    })
}
```

Adding a new document type = new sub-package + `init()` registration. Zero changes to framework code.

### L — Liskov Substitution

The `Document` interface contract: `ToMDL()` must return syntactically valid MDL that, if executed against the same project, produces an equivalent result. Implementations that can only provide best-effort output (e.g., Studio Pro objects with unknown fields) must still return valid MDL — they may omit unknown properties but must not emit invalid syntax or panic.

### I — Interface Segregation

Three focused interfaces; no fat interface:

```go
// doc.go

// Document is the minimum contract: any canonical model can serialize to MDL.
// All document types implement this. Even read-only catalog entries.
type Document interface {
    ToMDL() string
}

// Persistable extends Document for types that can be written to an MPR file.
// Not all documents are writable (e.g., read-only system module entries).
type Persistable interface {
    Document
    Persist(ctx PersistContext) error
}

// Diffable is reserved for future use (delta computation).
// Defined now to avoid breaking changes later.
type Diffable interface {
    Document
    DiffFrom(other Document) string
}

// Warning is a non-fatal issue encountered during Hydrate.
// Unknown BSON fields, missing optional values, etc.
type Warning struct {
    Field   string
    Message string
}
```

Executor checks capability via type assertion rather than assuming it:

```go
p, ok := doc.(model.Persistable)
if !ok {
    return fmt.Errorf("%T is not persistable", doc)
}
return p.Persist(ctx.PersistContext())
```

### D — Dependency Inversion

The executor depends on the `DocumentRegistry` interface, not on concrete model sub-packages:

```go
// registry.go
type DocumentRegistry interface {
    LiftFrom(stmt ast.Statement) (Persistable, error)
    HydrateFrom(el element.Element) (Document, []Warning, error)
}
```

`ExecContext` holds a `DocumentRegistry` field. The concrete registry (`model.DefaultRegistry`) is wired at startup. Tests inject a mock registry. Executor handlers never import `model/entity` or `model/microflow` directly.

---

## Key Design Constraints

### Canonical model = MDL-complete, not BSON-complete

The canonical model captures exactly what MDL can express. Non-MDL BSON fields (internal IDs, layout positions, timestamps, version prefixes) are **not** in the canonical model. They are the responsibility of `Persist()` and the backend:

- **CREATE**: `Persist()` generates fresh defaults for non-MDL fields.
- **UPDATE** (CREATE OR MODIFY, ALTER): `Persist()` reads the existing BSON object first and copies non-MDL fields before writing back.

This boundary is explicit and documented. Consumers of the canonical model must not assume it is a complete BSON representation.

### Lift() is a pure function with no project context

`Lift()` converts AST to canonical model without accessing the backend or project file. It must handle the `TypeEnumeration` vs `TypeEntity` ambiguity (CLAUDE.md §TypeEnumeration vs TypeEntity Ambiguity) by producing an "unresolved ref" DataType:

```go
type DataType struct {
    Kind    DataTypeKind  // includes KindUnresolvedRef for ambiguous cases
    Ref     string        // "Module.Name" when Kind == KindUnresolvedRef
    // ...
}
```

`Persist()` resolves ambiguous refs using the backend (check if ref is an entity or enum). `ToMDL()` emits the ref string as-is regardless — the MDL text is valid in both cases because the parser already accepts it.

### Hydrate() is defensive

`Hydrate()` never returns an error for unknown or missing BSON fields. Unknown fields are silently skipped. Missing optional fields become zero values in the canonical model. Non-fatal structural issues are returned as `[]Warning` for logging, not as errors. Only genuine data corruption (e.g., a required field that is both missing and has no sensible default) returns an error.

### Persist() preserves non-MDL fields on update

When updating an existing object (CREATE OR MODIFY, ALTER):

```go
// persist.go pattern
func (m *EntityModel) Persist(ctx PersistContext) error {
    existing, _ := ctx.Backend.GetEntityGen(m.Name)  // nil if new
    gen := buildGenEntity(m, existing)               // copies IDs, positions from existing
    return ctx.Backend.WriteEntityGen(gen)
}
```

### ToMDL() produces deterministic output

`ToMDL()` sorts all collections before serializing:
- Attributes: by declaration order (preserved in model)
- Indexes: by name
- Validation rules: by attribute name
- Access rules: by role name

Deterministic output is a hard requirement. Non-deterministic describe output causes flaky diffs and breaks round-trip stability tests.

### ALTER operations reuse the Hydrate → merge → Persist path

ALTER is not a separate code path. It is:
1. `Hydrate(existing gen object)` → canonical model of current state
2. Apply the ALTER delta (a structured merge, not raw BSON patching)
3. `Persist()` → write back

The ALTER AST node is converted to a typed delta struct (e.g., `EntityDelta{AddAttributes: [...], DropAttributes: [...]}`), not directly to gen types. The merge function in `entity/persist.go` applies the delta to the hydrated model.

### diff command improvement (free benefit)

After migration, the diff command's two serialization paths collapse to one:

```go
// Before: two diverging functions
Proposed: entityStmtToMDL(ctx, s)         // AST path
Current:  entityToMDLGen(ctx, modName, e) // gen path

// After: single ToMDL() on both sides
Proposed: registry.LiftFrom(s).ToMDL()
Current:  registry.HydrateFrom(e).ToMDL()
```

Diff accuracy improves as a side effect of the unification.

---

## Boundary Scenarios

| Scenario | Handling |
|----------|----------|
| TypeEnumeration vs TypeEntity ambiguity | `Lift()` emits `KindUnresolvedRef`; `Persist()` resolves; `ToMDL()` emits ref string unchanged |
| Studio Pro-created objects | `Hydrate()` skips unknown fields, returns warnings; `ToMDL()` is best-effort |
| ALTER (incremental update) | Hydrate existing → apply typed delta → Persist (copies non-MDL fields) |
| BSON fields with no MDL expression | Not in canonical model; `Persist()` generates defaults or copies from existing |
| Non-deterministic gen iteration order | `ToMDL()` normalizes all collections before serializing |
| MPR v1 vs v2 format | Handled in backend/gen layer, transparent to canonical model |
| CREATE OR MODIFY (upsert) | `Persist()` checks existence; same code path as ALTER update |

---

## Migration Strategy

The canonical model layer is introduced incrementally. Existing executor code is not broken during migration.

**Priority order** (most drift impact first):

1. **Entity + Attribute** — validation rules (NotNull/Unique) currently use `TypeName()` switch in describe; highest drift risk
2. **Association** — ownership direction and multiplicity encoding
3. **Microflow** — parameter types and return type encoding
4. **Page** — largest, most complex; done last

**Migration pattern per document type**:

1. Create `mdl/model/<type>/` sub-package with model, lift, hydrate, serialize, persist
2. Add `init()` registration
3. Update executor CREATE handler: `Lift() + Persist()` replaces direct gen manipulation
4. Update executor DESCRIBE handler: `Hydrate() + ToMDL()` replaces `switch TypeName()` 
5. Update diff command to use `ToMDL()` on both sides
6. Add round-trip stability test: `ToMDL(Lift(parse(mdl))) == normalize(mdl)` and `ToMDL(Hydrate(Write(Lift(parse(mdl))))) == normalize(mdl)`
7. Verify no regression: existing `TestMxCheck_DoctypeScripts` integration tests must still pass

---

## Testing Requirements

Each document type requires two round-trip stability tests:

**Test 1 — Parser round-trip** (no MPR file needed):
```
parse(mdl) → Lift() → ToMDL() == normalize(mdl)
```

**Test 2 — Full round-trip** (requires MPR file, integration tag):
```
parse(mdl) → Lift() → Persist() → Hydrate() → ToMDL() == normalize(mdl)
```

`normalize(mdl)` is a canonical formatter that strips comments, normalizes whitespace, and sorts where order is semantically irrelevant. It runs on the input before comparison so the test is not sensitive to formatting style.

Both tests are table-driven, using the `mdl-examples/doctype-tests/` corpus as the source of MDL inputs.

---

## Out of Scope

- `mxcli check` syntax validation: continues to operate at AST level, no change
- `mxcli check --references`: continues to use backend queries directly, no change
- Catalog-only read operations (SHOW ENTITIES, etc.): no canonical model needed
- Phase 4 read-path migration (15 domains with complex converters): separate spec

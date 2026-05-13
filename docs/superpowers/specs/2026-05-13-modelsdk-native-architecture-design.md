# Modelsdk-Native Architecture Design

**Date:** 2026-05-13
**Status:** PoC validated (2026-05-13) — Stage 2 plan pending
**Scope:** Full end-to-end migration from `sdk/*` types to `modelsdk/gen/*` types, with SOLID-compliant per-domain repository architecture
**Addendum:** [`2026-05-13-modelsdk-native-architecture-design-addendum.md`](2026-05-13-modelsdk-native-architecture-design-addendum.md) — empirical PoC findings; resolves Section 9 Blockers and Section 11 Open Decisions. Section 5 below incorporates the addendum's amendments.

---

## 1. Goal

Replace the parallel `sdk/*` type system with `modelsdk/gen/*` as the single source of truth for Mendix domain types. Decompose the 145-method `MprBackend` interface into focused per-domain repositories. Eliminate `sdk/mpr.SerializeXxx` from all write paths.

**End state:** `executor → mdl/repos (interfaces) → mdl/backend/mpr/repos (implementations) → modelsdk/codec → modelsdk/mpr → SQLite/mprcontents`. The `sdk/microflows`, `sdk/pages`, `sdk/domainmodel` etc. packages are deleted. `sdk/mpr` retains only the low-level SQLite operations.

---

## 2. Current State and Problems

### Existing architecture
- **`sdk/*` packages** (`sdk/microflows`, `sdk/pages`, `sdk/domainmodel`, etc.): hand-written domain types
- **`sdk/mpr/Serialize*`**: 28 package-level serializer functions converting `sdk/*` → BSON
- **`MprBackend`**: 145-method backend interface that handlers depend on
- **`*ViaModelsdk` functions** (145 of them, in 24 files): mediate between executor and write path; currently call `mpr.SerializeXxx(sdk_obj)` then `msdkWriter.InsertUnit(bytes)`

### SOLID problems

| Principle | Violation | Impact |
|---|---|---|
| **S** | Monolithic `MprBackend` mixes 145 unrelated operations | Hard to reason about, hard to test |
| **O** | Adding a new Mendix domain requires modifying `MprBackend` | Existing handlers recompile, churn |
| **L** | `MockBackend` must stub all 145 methods even for tiny tests | Mock noise, brittleness |
| **I** | A handler that needs `Get` depends on the same interface as one that needs `Delete` | Test injection coarse, dependency unclear |
| **D** | Concrete `sdk/microflows.Microflow` baked into 145 callsites | Every gen-type regeneration cascades; type system duplicated |

### Parallel-type tax
`sdk/microflows.Microflow` and `modelsdk/gen/microflows.Microflow` are two representations of the same Mendix concept. The codec exists to translate `gen/*` ↔ BSON but the executor uses `sdk/*`. We pay maintenance cost on both type systems.

### `ViaModelsdk` is a transitional name
The naming acknowledges these functions are the new path, but the implementation is hybrid: serialization through `sdk/mpr`, write through `msdkWriter`. The "Via" prefix is a smell — the canonical path should not need a qualifier.

---

## 3. Architecture

### Four-layer chain (target)

```
MDL AST
  ↓
Executor              (constructs and operates on gen/* types)
  ↓
mdl/repos/            (interface layer — buffer between executor and gen)
  ↓
mdl/backend/mpr/repos/ (MPR implementations)
  ↓
modelsdk/codec + modelsdk/mpr  (encoder/decoder + storage)
```

### Why a buffer interface layer

`modelsdk/gen/*` is **auto-generated**. Every Mendix version regenerates the types — fields can be added, renamed, or removed. `mdl/repos/` interfaces are **hand-written** and stable. They absorb gen-type drift so executor handlers don't break on each regeneration.

Repo implementations (`mdl/backend/mpr/repos/`) are the only code directly coupled to gen-type internals. After codegen, only the implementation layer needs review.

---

## 4. Package Structure

```
mdl/
  repos/                          # Interfaces only — zero implementations
    # Domain repositories
    microflows.go                 # MicroflowReader + MicroflowWriter + MicroflowRepository
    nanoflows.go
    pages.go
    layouts.go
    snippets.go
    domainmodels.go
    modules.go
    enumerations.go
    constants.go
    workflows.go
    services.go                   # JavaAction, REST, OData, BusinessEvent, etc.
    mappings.go                   # Import/Export mappings
    settings.go                   # Project + Module settings
    security.go
    folders.go
    images.go                     # ImageCollection
    agents.go                     # AI Agent Editor

    # Cross-domain coordination services
    reference.go                  # ReferenceService
    cascade.go                    # CascadeService

    # Transactions
    uow.go                        # UnitOfWork + TransactionFactory

    # Auxiliary services (avoid circular deps)
    resolver.go                   # QualifiedNameResolver
    id.go                         # IDGenerator
    version.go                    # VersionProvider
    cache.go                      # ReaderCache

    testing/                      # Recording mocks (part of design, not afterthought)
      microflows.go               # RecordingMicroflowWriter, etc.
      ...

  backend/mpr/
    repos/                        # MPR implementations (one file per domain)
      microflows.go
      pages.go
      ...
    factory.go                    # NewExecutorContext(path) → ctx, closer

modelsdk/gen/*/
  constructors.go                 # NEW: exported New*() constructor functions
```

**Dependency direction (single, no cycles):**

```
executor  →  mdl/repos (interfaces)  ←  mdl/backend/mpr/repos (implementations)
              ↑                               ↓
         modelsdk/gen/*                modelsdk/codec + modelsdk/mpr
```

---

## 5. Interface Design

### Reader/Writer split (Interface Segregation)

Each domain has a Reader interface (queries) and a Writer interface (mutations), composed into a `Repository` interface for callers needing both:

```go
// mdl/repos/microflows.go
package repos

import (
    genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
    "github.com/mendixlabs/mxcli/model"
)

type MicroflowReader interface {
    Get(id model.ID) (*genMf.Microflow, error)
    List(moduleID model.ID) ([]*genMf.Microflow, error)
    ListAll() ([]*genMf.Microflow, error)
    FindByQualifiedName(qn string) (*genMf.Microflow, error)
    IsRule(qn string) (bool, error)
}

type MicroflowWriter interface {
    // Create inserts a new microflow. parentUUID is the container unit's UUID
    // (typically a Folder or Module); containmentName is the BSON containment
    // slot ("Documents" for module-level documents). Lineage is supplied as
    // call parameters because gen types do not expose SetContainerID — see
    // addendum Blocker 2.
    Create(parentUUID string, containmentName string, mf *genMf.Microflow) error
    Update(mf *genMf.Microflow) error
    Delete(id model.ID) error
    Move(id model.ID, targetModuleID model.ID) error
}

type MicroflowRepository interface {
    MicroflowReader
    MicroflowWriter
}
```

The same pattern applies to all ~16 domain repos. Every domain's `Writer.Create`
takes `(parentUUID, containmentName, elem)` for the same reason — the BSON
encoder consumes only the element's own properties; container lineage is data
on the write call, not state on the element (addendum Blocker 2).

### Why three categories of interface

| Category | Examples | Lives in |
|---|---|---|
| **Repository** (CRUD per domain) | MicroflowRepository, PageRepository | `mdl/repos/` |
| **Service** (cross-domain coordination) | ReferenceService, CascadeService | `mdl/repos/` |
| **Auxiliary** (infrastructure) | QualifiedNameResolver, IDGenerator, VersionProvider, ReaderCache | `mdl/repos/` |

This three-tier split is required because not every operation maps to a single domain. Renames cross domains (microflow text references entity qualified names). Module deletion cascades across 6+ domains. Both belong in Service interfaces, not Repository interfaces.

### Cross-domain Services

```go
// mdl/repos/cascade.go
type CascadeService interface {
    DeleteModule(moduleID model.ID) error  // cascades to mf, pages, dm, security, settings
    DeleteFolder(folderID model.ID) error
}

// mdl/repos/reference.go
type ReferenceService interface {
    ScanRename(oldQN, newQN string) error
    PatchNavigationProfile(nav *genNav.NavigationProfile) error
    UpdateEnumerationRefsInAllDomainModels(oldQN, newQN string) error
}
```

### Auxiliary Services

`QualifiedNameResolver` exists to break circular dependencies. If `MprDomainModelRepo` depended on `ModuleReader` to resolve `Module.Entity` names, and `MprModuleRepo` depended on `DomainModelReader` to list module contents, the implementation graph cycles. The resolver bypasses repos and queries SQLite directly:

```go
// mdl/repos/resolver.go
type QualifiedNameResolver interface {
    ModuleNameByID(id model.ID) (string, error)
    ResolveQualifiedName(qn string) (domainID model.ID, kind string, err error)
}
```

`IDGenerator` exists because `Create()` needs an ID before the executor calls it; the executor would otherwise import `modelsdkmpr.GenerateID` directly (DIP violation):

```go
// mdl/repos/id.go
type IDGenerator interface {
    NewID() model.ID
}
```

`ReaderCache` makes the write-then-read consistency contract explicit:

```go
// mdl/repos/cache.go
type ReaderCache interface {
    Invalidate()
    InvalidateUnit(id model.ID)
}
```

Repos invalidate the cache after Write operations. **Note (post-PoC):** the
underlying `modelsdk/mpr.Writer` already invalidates its own reader cache on
both `InsertUnit` and `WriteTransaction.Commit` (addendum Blocker 4), so the
default repo implementations do not need to call `Invalidate()` explicitly.
The `ReaderCache` interface stays for explicit cross-process invalidation and
for tests that bypass `Writer`.

### Mutator interfaces for large units (post-PoC amendment)

PoC Blocker 3 measured that `Encoder.Encode()` rebuilds the entire root
document whenever any root-level dirty bit is set — incremental vs full
encoding came in at **1.06×**, not the 5× the design assumed (addendum
Blocker 3, 419 µs / 168 KB / 3 591 allocs per encode on a 25 KB microflow).
For microflow-sized units this is acceptable. For pages and workflows whose
widget trees often exceed 100 KB, the standard `Update(elem)` path becomes
performance-prohibitive on `ALTER PAGE` / `ALTER WORKFLOW`.

To preserve interactive ALTER throughput on large units, the design adds
mutator interfaces that operate on raw BSON sub-trees rather than re-encoding
the whole element:

```go
// mdl/repos/page_mutator.go
type PageMutator interface {
    SetWidgetProperty(widgetID model.ID, prop string, value any) error
    InsertWidget(parentID model.ID, slot string, widget element.Element) error
    DeleteWidget(widgetID model.ID) error
    ReplaceWidget(widgetID model.ID, replacement element.Element) error
    SetLayout(layoutQN string) error
    Commit() error  // single InsertUnit/WriteTransaction at the end
}

type PageRepository interface {
    PageReader
    PageWriter
    OpenForMutation(pageID model.ID) (PageMutator, error)
}

// mdl/repos/workflow_mutator.go — analogous shape for WorkflowRepository
```

Routing rule: handlers that perform whole-element CRUD (`CREATE PAGE`,
`DROP PAGE`) use the standard `PageWriter`; handlers that perform localized
in-place edits (`ALTER PAGE`, `ALTER WORKFLOW`) use the mutator. This mirrors
the existing `OpenPageForMutation` pattern in the legacy `MprBackend` and
keeps the spec's overall layering intact.

Microflow / domain-model / smaller domains do **not** need a mutator — the
standard `Update(elem)` path is fast enough at their typical unit sizes.

### UnitOfWork for atomic cross-domain operations

```go
// mdl/repos/uow.go
type UnitOfWork interface {
    Microflows()   MicroflowWriter
    Pages()        PageWriter
    DomainModels() DomainModelWriter
    Modules()      ModuleWriter
    // ... etc
    Commit() error
    Rollback()
}

type TransactionFactory interface {
    Begin() (UnitOfWork, error)
}
```

Multi-step operations (`create module` writes module + domain model + security + settings atomically) use `ctx.Tx.Begin()` instead of individual repos.

---

## 6. ExecutorContext

```go
// mdl/executor/context.go
type ExecutorContext struct {
    // Domain repositories
    Microflows    repos.MicroflowRepository
    Nanoflows     repos.NanoflowRepository
    Pages         repos.PageRepository
    Layouts       repos.LayoutRepository
    Snippets      repos.SnippetRepository
    DomainModels  repos.DomainModelRepository
    Modules       repos.ModuleRepository
    Enumerations  repos.EnumerationRepository
    Constants     repos.ConstantRepository
    Workflows     repos.WorkflowRepository
    Services      repos.ServiceRepository
    Mappings      repos.MappingRepository
    Settings      repos.SettingsRepository
    Security      repos.SecurityRepository
    Folders       repos.FolderRepository
    Images        repos.ImageCollectionRepository
    Agents        repos.AgentRepository

    // Cross-domain services
    Refs          repos.ReferenceService
    ModuleCascade repos.CascadeService

    // Transactions
    Tx            repos.TransactionFactory

    // Auxiliary services (all interfaces, not concrete types)
    QN            repos.QualifiedNameResolver
    IDs           repos.IDGenerator
    Version       repos.VersionProvider

    // Existing (unchanged)
    Catalog       catalog.CatalogReader
    Linter        *linter.Linter
}
```

Every field is an interface. Tests inject mock implementations from `mdl/repos/testing/`.

### Handler dependency granularity

Handlers continue to receive `*ExecutorContext`. For high-frequency test scenarios, handlers can declare a per-handler dependency interface:

```go
type CreateMicroflowDeps interface {
    Microflows() repos.MicroflowWriter
    Modules() repos.ModuleReader
}

// ExecutorContext implicitly satisfies CreateMicroflowDeps via methods
func handleCreateMicroflow(deps CreateMicroflowDeps, stmt *ast.Stmt) error { ... }
```

This is opt-in per-handler — no global signature change required.

---

## 7. Single Construction Entry Point

All entry points (REPL, CLI exec, LSP, test, benchmark) construct `ExecutorContext` through one factory:

```go
// mdl/backend/mpr/factory.go
func NewExecutorContext(path string) (*executor.ExecutorContext, io.Closer, error)
```

This factory:
- Opens `mpr.Reader`
- Constructs `codec.Decoder` and `codec.Encoder`
- Constructs `modelsdk/mpr.UnitWriter`
- Constructs all 16 domain repos sharing the above infrastructure
- Constructs services (Reference, Cascade) and auxiliaries (Resolver, IDGen, Version, Cache)
- Wires everything into ExecutorContext
- Returns a `Closer` that shuts down all underlying connections

Entry points become one line:

```go
ctx, closer, err := mprbackend.NewExecutorContext(path)
defer closer.Close()
```

Adding a new domain requires updating exactly one place: the factory.

---

## 8. MockRepository Strategy

`mdl/repos/testing/` contains reusable Recording mocks for every Repository, Service, and Auxiliary interface. Tests import these instead of writing one-off mock structs.

```go
// mdl/repos/testing/microflows.go
type RecordingMicroflowWriter struct {
    Created []*genMf.Microflow
    Updated []*genMf.Microflow
    Deleted []model.ID
    Moved   []MoveCall

    // Optional override hooks
    CreateFunc func(*genMf.Microflow) error
    UpdateFunc func(*genMf.Microflow) error
}

func (m *RecordingMicroflowWriter) Create(mf *genMf.Microflow) error {
    m.Created = append(m.Created, mf)
    if m.CreateFunc != nil {
        return m.CreateFunc(mf)
    }
    return nil
}
```

Tests verify behavior by inspecting `mock.Created` rather than coding assertion handlers. Recording mocks make tests shorter and more declarative.

This package is part of the design, not an afterthought. It must be created in lockstep with the interfaces.

---

## 9. Blockers Requiring PoC Validation Before Implementation

> **Status (2026-05-13): all four blockers validated.** See
> [`addendum`](2026-05-13-modelsdk-native-architecture-design-addendum.md).
> Three PASS, Blocker 3 MARGINAL (1.06×) — resolved by adding the
> `PageMutator` / `WorkflowMutator` interfaces in Section 5. The original
> blocker text below is preserved as historical context for the PoC scope.

Four assumptions must be empirically confirmed before committing to the design. Any failure forces a redesign loop.

### Blocker 1: Encoding freshly-constructed gen objects

```go
mf := genMf.NewMicroflow()
mf.Name.Set("MyFlow")
contents, err := encoder.Encode(mf)  // does this produce valid BSON?
```

`property.Primitive[T]` is designed for lazy decode (raw BSON → typed value on demand). When constructed without raw BSON and only `Set()` is called, the encoder must still produce correct output.

**Validation:** Round-trip test. Construct minimal `genMf.Microflow` from scratch, encode, decode, verify shape matches Studio Pro's output for a manually-created microflow.

### Blocker 2: gen type ID setters

```go
mf := genMf.NewMicroflow()
mf.SetID(ctx.IDs.NewID())             // exists?
mf.SetContainerID(moduleID)            // exists?
```

Currently the `initMicroflow()` factory and field initialization are private. The design requires public setters or constructors that accept ID/ContainerID.

**Validation:** Inspect `modelsdk/gen/microflows/types.go` for ID/ContainerID setters. If absent, codegen must be modified to emit them.

### Blocker 3: Incremental encoding (dirty tracking)

```go
mf, _ := repo.Get(id)            // mf has 200 activities, lazily loaded
mf.Parameters.Append(param)      // only one field modified
contents, _ := encoder.Encode(mf)
// Does encoder re-encode all 200 activities, or only the dirty Parameters field?
```

If `Encode()` always re-serializes the full object, ALTER on large pages/microflows becomes performance-prohibitive. The `OpenForMutation` pattern would have to survive as a separate Writer method.

**Validation:** Benchmark `Encode()` on a 300-widget page after modifying one widget. If linear in total widget count rather than dirty-field count, design must include `PageMutator` interface for surgical edits.

### Blocker 4: Write-then-read cache consistency

```go
ctx.Microflows.Create(mf)              // writes via msdkWriter
flows, _ := ctx.Microflows.List(mid)   // reads via mpr.Reader
// Does flows include the just-created mf?
```

`mpr.Reader` has internal caches (catalog, units). The msdkWriter writes the same SQLite. If they share the connection, SQLite's WAL ensures consistency. If they don't, the Reader cache may be stale.

**Validation:** Integration test — Create then immediately List, assert presence. Test both shared-connection and separate-connection scenarios.

---

## 10. Migration Strategy (Big Bang, Four Stages)

### Stage 1 — PoC validation (✅ complete, 2026-05-13)

Plan: `docs/superpowers/plans/2026-05-13-modelsdk-native-stage1-poc.md`.
Tests: `modelsdk/codec/poc/` (5 tests + 3 benchmarks). Findings: addendum.

Outcome: PROCEED with two spec amendments (already folded into Section 5):
1. `repo.Create` takes `(parentUUID, containmentName, elem)` — addendum
   Blocker 2.
2. New `PageMutator` / `WorkflowMutator` interfaces for large-unit ALTER —
   addendum Blocker 3.

No deal-breakers; Stage 2 is unblocked.

### Stage 2 — Infrastructure (additive, non-breaking)

Build the new layers alongside the old:
- Add public `New*()` constructors to `modelsdk/gen/*/` packages
- Create `mdl/repos/` with all interfaces
- Create `mdl/backend/mpr/repos/` with all implementations
- Create `mdl/repos/testing/` with Recording mocks
- Create `mprbackend.NewExecutorContext()` factory
- Old `MprBackend` and `ViaModelsdk` functions remain untouched

Build still passes. Tests still pass. Nothing user-visible changes.

### Stage 3 — Big Bang switchover

In a single coordinated change:
- Update `ExecutorContext` struct (replace `Backend` field with all repo fields)
- Update every executor handler to use the new repo fields
- Update every entry point (REPL, CLI exec, LSP, test runner, benchmark) to construct context via factory
- Replace `MockBackend` usage with composed Recording mocks from `mdl/repos/testing/`
- Remove all `b.writer.SerializeXxx` and `*ViaModelsdk` calls
- Build and tests must pass at the end of this stage

### Stage 4 — Cleanup (deletion)

After Stage 3 lands and stabilizes:
- Delete `sdk/microflows`, `sdk/pages`, `sdk/domainmodel`, etc. (16 packages)
- Delete `sdk/mpr/Serialize*` package-level functions
- Delete `sdk/mpr/Writer.Create/Update/Delete` for migrated domains
- Delete `MprBackend` struct and the 145-method backend interface
- `sdk/mpr` retains only the low-level SQLite read operations (until those too can be replaced by `modelsdk/mpr`)

---

## 11. Open Decisions Deferred to Plan

> **All resolved by the 2026-05-13 PoC addendum. Summary below; full
> rationale and benchmark numbers in the addendum's "Open Decisions
> Resolved" table.**

- **PageMutator interface presence** — ✅ **Yes, required.** Blocker 3
  measured 1.06× (not 5×); large-unit ALTER needs a mutator. Interface
  defined in Section 5 above.
- **Cache invalidation strategy** — ✅ **Automatic in `modelsdk/mpr.Writer`**;
  explicit `ReaderCache.Invalidate()` retained as optional escape hatch.
  Blocker 4.
- **gen type constructor location** — ✅ **Already in `modelsdk/gen/*`**
  (`NewMicroflow()` etc.); no codegen change needed. Blocker 1 + 2.
- **TransactionFactory implementation** — ✅ **Direct wrap** of
  `Writer.BeginWriteTransaction()`; commit handles cache invalidation.
  Blocker 4 transactional path PASS.
- **`repo.Create` lineage** — ✅ **Parameters** `(parentUUID, containmentName)`,
  not state on the element. Blocker 2; reflected in Section 5.

---

## 12. Out of Scope

- **`api/` high-level fluent API package** — separate migration plan
- **`modelsdk.go` public top-level API** — separate migration plan
- **VS Code extension** — unaffected (uses LSP protocol, not Go types)
- **`sdk/mpr` low-level SQLite operations** — retained until `modelsdk/mpr` provides full coverage (separate plan)
- **MPR v2 file-storage internals** — abstracted by `modelsdk/mpr.UnitWriter` already
- **AgentEditor types** — already standalone in `sdk/mpr/serialize_exports.go`, will be migrated as part of the regular domain migration in Stage 3

---

## 13. Success Criteria

After Stage 4:
- Zero imports of `sdk/microflows`, `sdk/pages`, `sdk/domainmodel`, `sdk/widgets` in `mdl/` or `cmd/`
- Zero `sdk/mpr.SerializeXxx` calls anywhere in the codebase
- Zero `b.writer.*` calls outside `sdk/mpr` itself
- `MprBackend` struct removed; replaced by per-domain repos
- All executor handlers compile against `repos.*Repository` interfaces only
- Full test suite passes (sdk/mpr, mdl/, modelsdk/)
- Studio Pro can open MPR files created and modified through the new path (manual validation, plus automated round-trip tests)

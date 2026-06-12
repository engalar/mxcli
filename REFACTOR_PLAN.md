# Architecture Refactoring Plan

## Phase 1 — Foundation: Role Interfaces

**Goal**: Break `FullBackend`'s monolithic interface into segregated role interfaces that consumers can depend on instead of the full surface.

### 1a. Define role interfaces in `mdl/backend/`

Create granular, single-purpose interfaces alongside the existing ones:

| Current | New role interfaces |
|---------|--------------------|
| `FullBackend` (30+ methods) | `EntityReader`, `MicroflowReader`, `PageReader`, `SecurityReader`, `NavigationReader`, `WorkflowReader`, `EntityWriter`, `MicroflowWriter`, `PageWriter`, `SecurityWriter`, `WorkflowWriter`, `ModuleLister`, `FolderLister`, etc. |

Each role interface should group __only__ the methods that logically belong together and are typically consumed together.

```go
// Example: isolate the "list/describe modules" concern
type ModuleLister interface {
    ListModules(ctx context.Context) ([]*model.Module, error)
}

// Example: isolate entity reading
type EntityReader interface {
    ListDomainModelsGen(ctx context.Context) ([]*genDm.DomainModel, error)
    GetDomainModelGen(ctx context.Context, unitID model.ID) (*genDm.DomainModel, error)
}
```

**Strategy**: Extract existing method signatures from each sub-interface in `backend.go` (e.g. `DomainModelBackend`, `MicroflowBackend`). The sub-interfaces already exist — the problem is `FullBackend` composes all of them. The first step is to stop composing them into one.

### 1b. Make `FullBackend` compose role interfaces instead

Change `mdl/backend/backend.go` from:
```go
type FullBackend interface {
    ConnectionBackend
    ModuleBackend
    // ... 25+ more
}
```
to a struct or keep the interface but make it clear it's only for construction:
```go
// FullBackend is the union of all backends. Only used at construction time.
// Handler code should accept role interfaces instead.
type FullBackend interface {
    ConnectionBackend
    ModuleLister
    EntityReader
    EntityWriter
    // ... etc
}
```

No functional change yet — just reorganizing the composition.

### 1c. Extract the mock backend from role interfaces

Update `mdl/backend/mock/mock_backend.go` to implement each role interface separately, so consumers can mock only what they need.

---

## Phase 2 — DIP: Wire ExecContext to Role Interfaces

**Goal**: Stop passing `FullBackend` to every handler — pass only what each handler needs.

### 2a. Audit every handler's actual backend usage

For each `register*Handler` function (30+ in `mdl/executor/register_stubs.go`), trace which `Backend.*` methods are called inside the handler. Create a map:

| Handler | Backend methods used | Role interface needed |
|---------|---------------------|----------------------|
| `CreateEntityStmt` | `ListDomainModelsGen`, `UpdateDomainModelGen` | `EntityReader`, `EntityWriter` |
| `DropMicroflowStmt` | `ListMicroflowsGen`, `DeleteUnit` | `MicroflowReader`, `UnitWriter` |
| ... | ... | ... |

### 2b. Add role-interface fields to `ExecContext`

```go
type ExecContext struct {
    context.Context
    Backend backend.FullBackend  // keep for backward compat

    // Role interfaces — populated lazily from Backend
    EntityReader   backend.EntityReader
    EntityWriter   backend.EntityWriter
    MicroflowReader backend.MicroflowReader
    // ... etc
}
```

### 2c. Populate role interfaces in `newExecContext`

In `newExecContext()`, cast or wrap `Backend` into the role interfaces:

```go
func newExecContext(e *Executor, ctx context.Context) *ExecContext {
    ec := &ExecContext{
        Context: ctx,
        Backend: e.backend,
    }
    if e.backend != nil {
        ec.EntityReader = e.backend
        ec.EntityWriter = e.backend
        // ...
    }
    // ...
}
```

### 2d. Migrate handlers one at a time

Start with the simplest handlers. For each:
1. Change handler signature to accept the role interface directly (via `ExecContext.EntityReader` instead of `ExecContext.Backend`)
2. Verify compilation
3. Run tests

Do this in 5-10 handler batches across multiple commits.

---

## Phase 3 — SRP: Split MprBackend

**Goal**: Break `MprBackend` from a single god struct with 30+ responsibilities into focused backend implementations that can be composed.

### 3a. Map the territory

`MprBackend` currently implements 30+ interfaces across 80 files in `mdl/backend/mpr/`. Each file implements a subset of methods.

Categorize:

| Domain | Files | Est. methods |
|--------|-------|-------------|
| Module/Folder | `modules.go`, `repos_provider.go` | 8 |
| Entity/Domain model | `domainmodel.go`, `domainmodel_v2.go`, `domainmodel_layout.go` | 12 |
| Microflow | `services_create_v2.go`, `convert.go`, `convert_reader.go` | 15 |
| Page/Layout/Snippet | `page_model.go`, `page_mutator.go`, `page_mutator_v2.go` | 20 |
| Workflow | `workflow_mutator.go`, `workflow_mutator_v2.go` | 15 |
| Security | `security.go`, `security_module.go`, `security_project.go`, `security_entity_access.go` | 10 |
| Java actions | `java_files.go`, `java_source_v2.go` | 8 |
| Settings | `settings.go`, `settings_legacy.go` | 6 |
| Navigation | `navigation.go`, `navigation_legacy.go` | 4 |
| Image | `image_legacy.go` | 4 |
| Agent editor | `agenteditor.go` | 6 |
| Translation | `translation_backend.go`, `translation_microflow.go`, `translation_page_mutator.go`, `translation_writer.go` | 12 |
| Serialization/builder | `widget_builder.go`, `agenteditor_serialize.go`, `datagrid_builder.go` | 8 |
| Script/transaction | `script_buf.go`, `script_tx.go`, `import_buf.go` | 6 |
| Infrastructure | `factory.go`, `constants.go`, `refs.go`, `rename.go`, `update_services.go`, `create_services.go`, `delete_move.go`, `write_helpers.go`, `unit_hash.go` | 20 |

### 3b. Extract domain-specific backend files into separate types

For each domain, extract the methods into a dedicated struct. Keep the `MprBackend` as a facade initially.

**Target structure:**
```
mdl/backend/mpr/
  backend.go              # MprBackend facade (keeps existing API)
  mpr_module.go           # moduleBackend
  mpr_entity.go           # entityBackend
  mpr_microflow.go        # microflowBackend
  mpr_page.go             # pageBackend
  mpr_workflow.go         # workflowBackend
  mpr_security.go         # securityBackend
  mpr_java.go             # javaBackend
  ...
```

### 3c. Pattern for extraction

Each extracted type receives only the dependencies it needs:

```go
// Before: method on MprBackend
func (b *MprBackend) ListModules(ctx context.Context) ([]*model.Module, error) {
    return b.msdkReader.ListModules(ctx)
}

// After: extracted to focused type
type moduleBackend struct {
    reader *modelsdkmpr.Reader
}

func (b *moduleBackend) ListModules(ctx context.Context) ([]*model.Module, error) {
    return b.reader.ListModules(ctx)
}

// MprBackend delegates
func (b *MprBackend) ListModules(ctx context.Context) ([]*model.Module, error) {
    return b.module.ListModules(ctx)
}
```

### 3d. Replace callers

Once all methods are extracted behind the facade:
1. Create `NewMprBackend()` that instantiates all sub-backends
2. Change callers that only need a subset to accept the role interface directly
3. When no callers use `MprBackend` for a given method, the facade method can be removed

---

## Phase 4 — LSP: Fix Unchecked Type Assertions

**Goal**: Every type assertion from `element.Element` to a concrete `*gen*` type checks the `ok` value and handles failure gracefully.

### 4a. Production code (non-test files)

| File | Line(s) | Pattern | Fix |
|------|---------|---------|-----|
| `mdl/executor/cmd_associations.go` | 417 | `db, _ := elem.(...)` | Return nil, caller checks |
| `mdl/executor/cmd_odata.go` | 629, 640 | `oc, _ := mf.ObjectCollection().(...)` | Guard with `if oc == nil { continue }` |
| `mdl/executor/cmd_odata.go` | 730 | `src, _ := entity.Source().(...)` | Guard with nil check |
| `mdl/executor/cmd_workflows_write_v2.go` | 271, 940-944, 1126-1130 | `f, _ := voc.Flow().(...)` | Guard with nil check or continue |

### 4b. Test code

Same pattern — add `, ok` and skip/assert as appropriate.

---

## Phase 5 — executorCache and Executor

**Goal**: Reduce the surface area of `executorCache` without breaking the lazy-population pattern.

### 5a. Add domain-specific cache accessors

Instead of splitting into multiple structs (which forces multi-nil checks), add typed accessor methods:

```go
type executorCache struct {
    // Private fields stay co-located

    // lazily-populated accessors
}

func (c *executorCache) Microflows() []MicroflowGenWithContainer {
    return c.microflowsWithContainerGen
}

func (c *executorCache) SetMicroflows(v []MicroflowGenWithContainer) {
    c.microflowsWithContainerGen = v
}
```

### 5b. Invalidation becomes domain-scoped

Replace the monolithic `invalidateAllDocumentCaches()` with per-domain invalidation that callers invoke explicitly:

```go
func invalidateMicroflowCache(ctx *ExecContext) { ... }
func invalidatePageCache(ctx *ExecContext) { ... }
func invalidateWorkflowCache(ctx *ExecContext) { ... }
```

---

## Migration Order

```
Phase 1a ──► 1b ──► 1c          (safe, no callers change)
   │
   ▼
Phase 2a ──► 2b ──► 2c ──► 2d  (incremental, handler-by-handler)
   │
   ▼
Phase 3a ──► 3b ──► 3c ──► 3d  (facade pattern, no breaking changes)
   │
   ▼
Phase 4a ──► 4b                 (mechanical, low risk)
   │
   ▼
Phase 5a ──► 5b                 (optional cleanup)
```

Each phase compiles independently and can be committed separately. No phase breaks the build.

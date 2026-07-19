# Plan 4: repos Package Activation — Repository Layer Unification

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate the overlap between `mdl/backend/role.go` query interfaces (`DomainModelReader`, `MicroflowReader`, etc.) and `mdl/repos/` repository interfaces. Queries go through `repos`, mutations stay in `backend`. Delete ~10 Reader interfaces from `backend/role.go`.

**Architecture:** Split each current `backend.RoleReader+Writer` into pure Writer (stay in backend) + Query interface (add to repos, or use existing repos methods). Repos already has most of the needed interfaces — the work is wiring executor handlers to use repos for reads instead of backend.

**Tech Stack:** Go, `mdl/backend/`, `mdl/repos/`, `mdl/executor/`

## Global Constraints

- Every commit must compile: `go build ./...`
- Every commit must pass: `go test ./mdl/... -count=1`
- Repos interfaces already exist — do NOT redesign them, just wire them
- Handler migration pattern: `deps.DomainModelReader.ListDomainModelsGen()` → `deps.DomainModels.ListDomainModels(ctx)`
- The `context.Context` parameter is the key API difference — repos takes ctx, backend interfaces don't

---

### Task 1: Audit current usage of backend Reader interfaces

**Files:** 
- Read-only analysis

- [ ] **Step 1: Find all production uses of backend query interfaces**

```bash
# Find every reference to each Reader/Writer interface in non-test code
for iface in DomainModelReader MicroflowReader WorkflowReader PageReader ModuleLister EnumerationReader ConstantReader SettingsReader MappingReader NavigationReader; do
    echo "=== $iface ==="
    rg -n "$iface" mdl/executor/ --no-filename | grep -v '_test.go' | grep -v '^\s*//' | head -10
done
```

Catalog the results — this tells us which handlers need migration.

- [ ] **Step 2: Map each Reader interface to the equivalent repos interface**

| Backend interface | Equivalent repos interface | Status |
|-------------------|---------------------------|--------|
| `DomainModelReader` | `DomainModelRepository` (repos/domainmodels.go) | Exists |
| `MicroflowReader` | `MicroflowRepository` (repos/microflows.go) | Exists |
| `WorkflowReader` | `WorkflowRepository` (repos/workflows.go) | Exists |
| `PageReader` | `PageRepository` (repos/pages.go) | Exists |
| `ModuleLister` | `ModuleRepository` or similar | Check |
| `EnumerationReader` | Check repos/ | Stub |
| `ConstantReader` | Check repos/ | Stub |
| `SettingsReader` | Check repos/ | Stub |
| `MappingReader` | Check repos/ | Stub |
| `NavigationReader` | Check repos/ | Stub |

- [ ] **Step 3: Commit findings**

```bash
# No code change — just document the analysis
```

---

### Task 2: Add missing repos implementations

**Files:**
- Modify: various files in `mdl/repos/` and `mdl/backend/mpr/repos/`

Based on Task 1 findings, implement any repos interfaces that are only stubs.

For each missing repos interface, ensure the implementation in `mdl/backend/mpr/repos/` is complete, not a `panic("TODO")` stub.

- [ ] **Step 1: Check each repos interface for TODO stubs**

```bash
rg -n "TODO\|panic(" mdl/repos/ mdl/backend/mpr/repos/
```

- [ ] **Step 2: Implement each stub**

For each stub, implement using the same backend reader pattern that the current `DomainModelReader` methods use:

```go
// Example: if EnumerationReader exists as stub, implement:
func (r *enumRepo) ListEnumerations(ctx context.Context) ([]*model.Enumeration, error) {
    return r.reader.ListEnumerations()
}
```

- [ ] **Step 3: Build and test**

```bash
go build ./... && go test ./mdl/... -count=1 -timeout 120s
```

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "feat: complete repos interface implementations"
```

---

### Task 3-12: Migrate handlers from backend Reader to repos (one domain per task)

**Pattern for each migration:**

```
Find all handlers that use deps.DomainModelReader.ListDomainModelsGen()
Replace with: deps.DomainModels.ListDomainModels(ctx)

Find all handlers that use deps.MicroflowReader.ListMicroflowsGen()
Replace with: deps.MicroflowRepo.List(ctx)
```

- [ ] **Task 3: Migrate entity/domainmodel queries**

Replace `deps.DomainModelReader` usage with `deps.DomainModels` in all handlers.

```bash
rg -n 'deps\.DomainModelReader\.' mdl/executor/ --no-filename | grep -v '//'
```

- [ ] **Task 4: Migrate microflow queries**

Replace `deps.MicroflowReader` usage with `deps.MicroflowRepo`.

- [ ] **Task 5: Migrate workflow queries**

Replace `deps.WorkflowReader` usage with `deps.WorkflowRepo`.

- [ ] **Task 6: Migrate page queries**

Replace `deps.PageReader` usage with `deps.PageRepo`.

- [ ] **Task 7: Migrate module queries**

Replace `deps.ModuleLister` usage with `deps.Modules` (or add to repos if missing).

- [ ] **Task 8: Migrate enumeration queries**

Replace `deps.EnumerationReader` usage with repos equivalent.

- [ ] **Task 9: Migrate constant queries**

Replace `deps.ConstantReader` usage with repos equivalent.

- [ ] **Task 10: Migrate settings queries**

Replace `deps.SettingsReader` usage with repos equivalent.

- [ ] **Task 11: Migrate mapping queries**

Replace `deps.MappingReader` usage with repos equivalent.

- [ ] **Task 12: Migrate navigation queries**

Replace `deps.NavigationReader` usage with repos equivalent.

**Each migration commit:**

```bash
git add -A
git commit -m "refactor: migrate <domain> queries from backend.Reader to repos"
```

---

### Task 13: Remove migrated Reader interfaces from backend/role.go

**Files:**
- Modify: `mdl/backend/role.go`
- Modify: `mdl/backend/backend.go` (FullBackend composition)
- Modify: `mdl/executor/handler_deps.go`
- Modify: `mdl/backend/mock/mock_backend.go`

- [ ] **Step 1: Verify zero production references to each Reader interface**

```bash
for iface in DomainModelReader MicroflowReader WorkflowReader PageReader ModuleLister; do
    echo "Checking $iface..."
    rg -n "$iface" mdl/ --include='*.go' | grep -v '_test.go' | grep -v '^\s*//' | grep -v "type.*interface" | grep -v "role.go" | grep -v "backend.go"
done
```

Should return zero matches for each.

- [ ] **Step 2: Remove Reader interface definitions from `role.go`**

Delete:
- `DomainModelReader` interface definition
- `MicroflowReader` interface definition
- `WorkflowReader` interface definition
- `PageReader` interface definition
- `ModuleLister` interface definition
- `EnumerationReader` interface definition
- `ConstantReader` interface definition
- `SettingsReader` interface definition
- `MappingReader` interface definition
- `NavigationReader` interface definition

Keep Writer interfaces that have write methods:
- `DomainModelWriter` (UpdateEntityGen, CreateEntityGen, etc.)
- `MicroflowWriter` (DeleteMicroflow, DeleteNanoflow)
- `WorkflowWriter` (CreateWorkflowGen, UpdateWorkflowGen, DeleteWorkflow)
- `PageWriter` (existing write methods)
- `ModuleWriter`
- etc.

- [ ] **Step 3: Remove from `FullBackend` composition**

In `mdl/backend/backend.go`, remove the embedded Reader interfaces from `FullBackend`.

- [ ] **Step 4: Remove from `HandlerDeps`**

In `mdl/executor/handler_deps.go`, delete:
```go
ModuleLister         backend.ModuleLister
DomainModelReader    backend.DomainModelReader
MicroflowReader      backend.MicroflowReader
WorkflowReader       backend.WorkflowReader
PageReader           backend.PageReader
EnumerationReader    backend.EnumerationReader
ConstantReader       backend.ConstantReader
SettingsReader       backend.SettingsReader
MapperReader         backend.MappingReader
NavigationReader     backend.NavigationReader
```

- [ ] **Step 5: Remove from `MockBackend`**

In `mdl/backend/mock/mock_backend.go`, remove the compiled-time checks and method stubs for each deleted interface.

- [ ] **Step 6: Build and test**

```bash
go build ./... && go test ./mdl/... -count=1 -timeout 180s
```

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "refactor!: remove Reader interfaces from backend/role.go

All query operations now go through mdl/repos interfaces.
Backend only holds Writer interfaces for mutations."
```

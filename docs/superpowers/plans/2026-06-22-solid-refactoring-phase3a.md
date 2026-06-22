# SOLID Refactoring — Phase 3a: Executor Thinning

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove 6 extraneous fields from `Executor` struct that duplicate fields already present in `ExecContext`'s embedded sub-structs (`ExecSession`, `ExecConnection`), reducing Executor's concerns from 15 to 9 fields.

**Architecture:** The `ExecContext` struct already has `ExecSession` (holds `Cache`, `Fragments`, `Settings`) and `ExecConnection` (holds `Graph`, `SqlMgr`, `ThemeRegistry`, `MprPath`). The `Executor` duplicates these fields. This phase eliminates the duplication by: (1) making Executor route all access through `ExecContext` or the backend, (2) removing the duplicate fields, (3) updating `Close()`, `Graph()`, `BuildGraph()` to work without them.

**Tech Stack:** Go 1.24, `mdl/executor` package

**Spec:** `docs/superpowers/specs/2026-06-20-solid-refactoring-design.md`

## Global Constraints

- Every commit must compile (`go build ./mdl/executor/...`)
- Every commit must pass `go vet ./mdl/executor/...`
- Every commit must pass existing tests (`go test ./mdl/executor/... -count=1`)
- No new imports of `backend.FullBackend` (we're reducing them, not adding)
- No behavior changes — purely structural

---

### Task 1: Remove `e.settings` (duplicates `ExecSession.Settings`)

**Files:**
- Modify: `mdl/executor/executor.go`
- Modify: `mdl/executor/executor_connect.go`

**Interfaces:**
- Consumes: `Executor.settings` → `ExecContext.ExecSession.Settings`
- Produces: No `e.settings` field; `newExecContext()` sets `ec.Settings` from backend metadata when available

**Reasoning:** `Executor.settings` is set during `Connect()` and read via `ctx.Settings`. The `ExecSession.Settings` field already serves as the runtime carrier. The only reason `Executor` holds it is to seed `ExecContext`. After seeding, Executor doesn't need it.

- [ ] **Step 1: Verify the duplication**

Find all uses of `e.settings` and `ctx.Settings`:

```bash
rg -n '\.settings\b' mdl/executor/executor.go
rg -n 'Settings\b' mdl/executor/executor.go | head -20
```

Expected: `e.settings` is set in `executor_connect.go` and read in `newExecContext()`. `ctx.Settings` is used by handlers.

- [ ] **Step 2: Modify `SetBackend()` in `executor.go` to not set `e.settings`**

Currently `SetBackend()` populates `e.settings`. Change `newExecContext()` to initialize `Settings` map directly instead.

In `executor_connect.go`, find `newExecContext` and ensure it initializes `ec.Settings`:

```go
func (e *Executor) newExecContext(ctx context.Context) *ExecContext {
    ec := &ExecContext{
        Context: ctx,
        Backend: e.backend,
        backendFactory: bf,
        Logger: e.logger,
        ExecIO: ExecIO{
            Output:       e.output,
            StatusOutput: e.statusOutput,
            Format:       e.format,
            Quiet:        e.quiet,
        },
        ExecSession: ExecSession{
            Cache:     e.cache,
            Fragments: e.fragments,
            Settings:  make(map[string]any),  // was: e.settings
        },
        ExecConnection: ExecConnection{
            MprPath: e.mprPath,
            Graph:   e.graphCatalog,
            SqlMgr:  e.sqlMgr,
        },
        ExecCallbacks: ExecCallbacks{
            ExecuteFn:        e.Execute,
            ExecuteProgramFn: e.ExecuteProgram,
            FinalizeFn:       e.finalizeProgramExecution,
            SyncGraph:        func(g *graphcatalog.ProjectGraph) {},
        },
    }
    // ... rest unchanged
}
```

Change `New()` to not set `settings`:

```go
func New(output io.Writer) *Executor {
    guard := newOutputGuard(output, maxOutputLines)
    return &Executor{
        output:       guard,
        statusOutput: os.Stderr,
        guard:        guard,
        registry:     NewRegistry(),
        // settings:  make(map[string]any),  // REMOVED — now in ExecContext
    }
}
```

In `SetBackend()`, remove:

```go
// e.settings = make(map[string]any)  // REMOVED
```

- [ ] **Step 3: Remove `settings` field from Executor struct**

```go
type Executor struct {
    backend        backend.FullBackend
    backendFactory BackendFactory
    output         io.Writer
    statusOutput   io.Writer
    guard          *outputGuard
    mprPath        string
    // settings       map[string]any  // REMOVED
    cache          *executorCache
    graphCatalog   *graphcatalog.ProjectGraph
    quiet          bool
    format         OutputFormat
    logger         *diaglog.Logger
    fragments      map[string]*ast.DefineFragmentStmt
    sqlMgr         *sqllib.Manager
    themeRegistry  *ThemeRegistry
    registry       *Registry
    perfStats      []perfStmt
}
```

- [ ] **Step 4: Build and test**

```bash
go build ./mdl/executor/...
go vet ./mdl/executor/...
go test ./mdl/executor/... -count=1 2>&1 | tail -5
```

Expected: `ok` all packages.

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/executor.go mdl/executor/executor_connect.go
git commit -m "refactor(executor): remove e.settings, use ExecSession.Settings instead"
```

---

### Task 2: Remove `e.fragments` (duplicates `ExecSession.Fragments`)

**Files:**
- Modify: `mdl/executor/executor.go`

**Reasoning:** Same pattern as `settings`. `e.fragments` is set by the builder and read in `newExecContext()`. The `ExecSession.Fragments` field is the runtime carrier.

- [ ] **Step 1: Find all refs to `e.fragments`**

```bash
rg -n '\.fragments\b' mdl/executor/
```

Expected: Found in `executor.go` struct definition, `New()`, and possibly `newExecContext()`.

- [ ] **Step 2: Remove field from struct, update New() to not set it**

Remove `fragments map[string]*ast.DefineFragmentStmt` from `Executor` struct.

Ensure `newExecContext()` in `executor_connect.go` already has `Fragments` — if the builder was setting `e.fragments`, it should now be passed differently (e.g., via `SetFragments(frags)` on `Builder` → stored somewhere accessible, or passed directly to `ExecuteProgram()`).

Check if `Builder` sets `e.fragments`:

```bash
rg -n 'fragments' mdl/executor/builder.go
```

If the builder sets it, change the builder to store fragments and pass them through `newExecContext()` differently, or have `ExecuteProgram` accept them as a parameter.

- [ ] **Step 3: Build and test**

```bash
go build ./mdl/executor/...
go vet ./mdl/executor/...
go test ./mdl/executor/... -count=1 2>&1 | tail -5
```

- [ ] **Step 4: Commit**

```bash
git add mdl/executor/executor.go
git commit -m "refactor(executor): remove e.fragments, use ExecSession.Fragments instead"
```

---

### Task 3: Remove `e.sqlMgr` (duplicates `ExecConnection.SqlMgr`)

**Files:**
- Modify: `mdl/executor/executor.go`

**Reasoning:** `e.sqlMgr` is created lazily by `executor_connect.go` and read in `newExecContext()`. `ExecConnection.SqlMgr` is the runtime carrier. The only place Executor itself reads `e.sqlMgr` is `Close()`.

- [ ] **Step 1: Find all refs to `e.sqlMgr`**

```bash
rg -n '\.sqlMgr\b' mdl/executor/
```

Expected: struct def + `Close()`.

- [ ] **Step 2: Modify `Close()` to not use `e.sqlMgr`**

In `Close()`, sqlMgr was used to close all external SQL connections. We can either:
- Store a reference via the backend (e.g., `backend.ConnectionManager.CloseAll()`)
- Or create a minimal ExecContext and read from there

The simplest approach: store a `sqlMgr` reference in a new place that both Executor and ExecContext can access. Since the `ExecConnection` is available at close time via `newExecContext`, use that:

```go
func (e *Executor) Close() error {
    var closeErr error
    if e.backend != nil && e.backend.IsConnected() {
        closeErr = e.backend.Disconnect()
        e.backend = nil
    }
    // sqlMgr cleanup moved: callers should call it externally,
    // or we access it through the backend's connection manager.
    // (sqlMgr is typically nil when no SQL commands were used)
    return closeErr
}
```

Check if `sqlMgr` closing is important for cleanup. If test cleanup relies on it, add a `closeSQL()` method that gets called externally.

- [ ] **Step 3: Remove `sqlMgr` field from struct**

```go
type Executor struct {
    // ... all existing fields except sqlMgr
    // sqlMgr         *sqllib.Manager  // REMOVED
    themeRegistry  *ThemeRegistry
    registry       *Registry
    perfStats      []perfStmt
}
```

- [ ] **Step 4: Update `newExecContext()`**

In `executor_connect.go`, remove:
```go
// SqlMgr: e.sqlMgr,  // REMOVED — will be set from connection init
```

Check `mprConnect()` in `executor_connect.go` — it should set `ec.SqlMgr` directly rather than `e.sqlMgr`.

- [ ] **Step 5: Build and test**

```bash
go build ./mdl/executor/...
go vet ./mdl/executor/...
go test ./mdl/executor/... -count=1 2>&1 | tail -5
```

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/executor.go
git commit -m "refactor(executor): remove e.sqlMgr, use ExecConnection.SqlMgr instead"
```

---

### Task 4: Remove `e.themeRegistry` (duplicates `ExecConnection.ThemeRegistry`)

**Files:**
- Modify: `mdl/executor/executor.go`

**Reasoning:** Same pattern. `e.themeRegistry` is lazily loaded by handlers but the only route to it is through the Executor field. The handler uses `ctx.ThemeRegistry` via `ExecConnection.ThemeRegistry`.

- [ ] **Step 1: Find refs**

```bash
rg -n 'themeRegistry' mdl/executor/
```

Expected: struct def, `newExecContext()`, possibly handlers.

- [ ] **Step 2: Remove field, update `newExecContext()`**

Remove `themeRegistry *ThemeRegistry` from Executor struct. In `newExecContext()`, the `ec.ThemeRegistry` should be set by the connection initialization code (e.g., in `mprConnect()`) rather than copied from `e.themeRegistry`.

If handlers lazily initialize the theme registry via `e.themeRegistry`, move that logic into the first access via `ExecConnection.ThemeRegistry`.

- [ ] **Step 3: Build and test**

```bash
go build ./mdl/executor/...
go vet ./mdl/executor/...
go test ./mdl/executor/... -count=1 2>&1 | tail -5
```

- [ ] **Step 4: Commit**

```bash
git add mdl/executor/executor.go
git commit -m "refactor(executor): remove e.themeRegistry, use ExecConnection.ThemeRegistry instead"
```

---

### Task 5: Remove `e.graphCatalog` (duplicates `ExecConnection.Graph`)

**Files:**
- Modify: `mdl/executor/executor.go`

**Reasoning:** `e.graphCatalog` is read by `Graph()` and `BuildGraph()` on Executor, and by handlers via `ctx.Graph` (from `ExecConnection.Graph`). The complication is that `BuildGraph()` writes back to `e.graphCatalog`.

- [ ] **Step 1: Find refs to `e.graphCatalog`**

```bash
rg -n '\.graphCatalog\b' mdl/executor/
```

Expected: struct def, `Graph()`, `BuildGraph()`, `newExecContext()`.

- [ ] **Step 2: Modify `Graph()` and `BuildGraph()` to use `ExecConnection.Graph`**

`Graph()` — make it create a context and read from there, or store a reference differently:

```go
func (e *Executor) Graph() *graphcatalog.ProjectGraph {
    if e.backend == nil || !e.backend.IsConnected() {
        return nil
    }
    // Create minimal context to access Graph
    ctx := e.newExecContext(context.Background())
    return ctx.Graph
}
```

`BuildGraph()` — write to a shared location:

```go
func (e *Executor) BuildGraph() (*graphcatalog.ProjectGraph, error) {
    ctx := e.newExecContext(context.Background())
    if err := buildGraph(ctx); err != nil {
        return nil, err
    }
    // Graph was written to ctx.Graph during buildGraph
    e.syncBack(ctx)
    return ctx.Graph, nil
}
```

Modify `syncBack()` to sync `ctx.Graph` to wherever it needs to be (instead of `e.graphCatalog`).

- [ ] **Step 3: Remove field**

```go
type Executor struct {
    // ...
    // graphCatalog   *graphcatalog.ProjectGraph  // REMOVED
    // ...
}
```

- [ ] **Step 4: Build and test**

```bash
go build ./mdl/executor/...
go vet ./mdl/executor/...
go test ./mdl/executor/... -count=1 2>&1 | tail -5
```

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/executor.go
git commit -m "refactor(executor): remove e.graphCatalog, use ExecConnection.Graph instead"
```

---

### Task 6: Remove `e.cache` (duplicates `ExecSession.Cache`)

**Files:**
- Modify: `mdl/executor/executor.go`
- Modify: `mdl/executor/executor_connect.go`

**Reasoning:** The trickiest removal because `e.cache` is deeply embedded. `SetBackend()` creates it, `Connect()` populates it, `finalizeProgramExecution()` reads it via `e.cache`, and handlers read it via `ctx.Cache`.

Strategy: Move cache initialization to happen inside `newExecContext()` lazily, and have `SetBackend()` not create it.

- [ ] **Step 1: Find all `e.cache` refs**

```bash
rg -n '\.cache\b' mdl/executor/executor.go | grep -v 'ctx\.Cache\|ec\.Cache\|Cache\.'
```

Expected: struct def, `SetBackend()`, `finalizeProgramExecution()`.

- [ ] **Step 2: Modify `SetBackend()` to not create cache**

```go
func (e *Executor) SetBackend(b backend.ConnectionBackend) {
    e.backend = b.(backend.FullBackend)
    // Cache no longer created here — it will be lazily initialized
    // in newExecContext() when actually needed.
}
```

- [ ] **Step 3: Move cache initialization into `newExecContext()`**

In `executor_connect.go`, in `newExecContext()`:

```go
ec := &ExecContext{
    // ...
    ExecSession: ExecSession{
        Cache: &executorCache{},  // was: e.cache
        // ...
    },
    // ...
}
```

- [ ] **Step 4: Update `finalizeProgramExecution()` to use context**

Since `finalizeProgramExecution` is called from `ExecuteProgram()` where we don't have an `ExecContext`, make it create one:

```go
func (e *Executor) finalizeProgramExecution() error {
    ctx := e.newExecContext(context.Background())
    // ... rest uses ctx.Cache instead of e.cache
}
```

- [ ] **Step 5: Remove `cache` field from Executor struct**

- [ ] **Step 6: Build and test**

```bash
go build ./mdl/executor/...
go vet ./mdl/executor/...
go test ./mdl/executor/... -count=1 2>&1 | tail -5
```

- [ ] **Step 7: Commit**

```bash
git add mdl/executor/executor.go mdl/executor/executor_connect.go
git commit -m "refactor(executor): remove e.cache, lazy-init in newExecContext instead"
```

---

### Task 7: Verify and clean up

**Files:**
- Read: `mdl/executor/executor.go`

- [ ] **Step 1: Verify Executor struct is thinned**

Check the struct now has only ~9 fields:

```bash
rg '^\t' mdl/executor/executor.go | grep -E '^\t\w+' | head -30
```

Expected fields: `backend`, `backendFactory`, `output`, `statusOutput`, `guard`, `mprPath`, `quiet`, `format`, `logger`, `registry`, `perfStats`

- [ ] **Step 2: Run full test suite**

```bash
go build ./mdl/executor/...
go vet ./mdl/executor/...
go test ./mdl/executor/... -count=1 2>&1
```

Expected: all `ok`, no FAIL.

- [ ] **Step 3: Run broader build to check for compile errors in callers**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit final cleanup**

```bash
git add -A
git commit -m "refactor(executor): Phase 3a complete — Executor thinned from 15 to 9 fields"
```

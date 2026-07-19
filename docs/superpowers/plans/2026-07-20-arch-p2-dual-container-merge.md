# Plan 2: Dual Container Merge — ExecContext Retirement

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete `ExecContext` struct, its 5 embedded sub-structs (`ExecRepos`, `ExecIO`, `ExecSession`, `ExecConnection`, `ExecCallbacks`), and the two bidirectional bridge functions (`execContextToDeps`, `NewExecContext`). All ~68 handlers use `*HandlerDeps` directly.

**Architecture:** Currently ~35 handler functions still create a temporary `ExecContext` via `NewExecContext(deps)`. Each one must be migrated to accept `*HandlerDeps` directly. Once all are migrated, the bridge code (240 lines) and ExecContext (177 lines) are deleted.

**Tech Stack:** Go, `mdl/executor/`, `mdl/executor/domainmodel/`

## Global Constraints

- Every commit must compile: `go build ./...`
- Every commit must pass: `go test ./mdl/executor/...`
- Each handler migration is a separate commit for clean revertability
- Handler signature change pattern: `(ectx *ExecContext, stmt)` → `(ctx context.Context, stmt, deps *HandlerDeps)`
- Inside handler: `ectx.Xxx` → `deps.Xxx`, `ectx.Backend.Xxx` → `deps.RoleInterface.Xxx`, `ectx.Context` → `ctx`

---

### Task 1: Flatten ExecContext sub-structs into HandlerDeps

**Files:**
- Modify: `mdl/executor/executor.go:725-776`

- [ ] **Step 1: Verify all ExecRepos/ExecIO/ExecSession/ExecConnection/ExecCallbacks fields already exist in HandlerDeps**

Compare `executor.go:725-776` with `handler_deps.go:20-105`. Confirm:
- `ExecRepos` fields → `HandlerDeps` repo fields
- `ExecIO` fields → `HandlerDeps` `Output`, `StatusOutput`, `Format`, `Quiet`
- `ExecSession` fields → `HandlerDeps` `Cache`, `Session`, `Fragments`, `Settings`
- `ExecConnection` fields → `HandlerDeps` `MprPath`, `BackendFactory`, `Graph`, `Perf`
- `ExecCallbacks` fields → `HandlerDeps` `ExecuteFn`, `ExecuteProgramFn`, `FinalizeFn`, `SyncGraph`

All match. No data loss.

- [ ] **Step 2: Remove sub-struct type definitions**

Delete from `executor.go:725-776`:
```go
type ExecRepos struct { ... }
type ExecIO struct { ... }
type ExecSession struct { ... }
type ExecConnection struct { ... }
type ExecCallbacks struct { ... }
```

Update `ExecContext` to embed fields directly:

```go
type ExecContext struct {
	context.Context

	backendFactory backend.BackendFactory
	Logger         *diaglog.Logger

	// Role-specific backend interfaces (all from HandlerDeps)
	// ... keep ALL role fields, they just move from being promoted
	// from sub-structs to being direct fields
}
```

The role interface fields (ModuleLister, ModuleWriter, etc.) are already direct fields on ExecContext — they don't change. Only the sub-struct embeddings are removed. The fields within those sub-structs just move up.

- [ ] **Step 3: Update `NewExecContext` to use direct field assignment**

In `handlers_future.go:2901-2994`, replace sub-struct populating:
```go
// Before:
ExecRepos: ExecRepos{
	DomainModels: deps.DomainModels,
	...
},
ExecIO: ExecIO{
	Output: deps.Output,
	...
},

// After (direct field assignment):
ectx.Cache = deps.Cache
ectx.Session = deps.Session
ectx.Fragments = deps.Fragments
ectx.Settings = deps.Settings
ectx.Output = deps.Output
ectx.StatusOutput = deps.StatusOutput
ectx.Format = deps.Format
ectx.Quiet = deps.Quiet
ectx.DomainModels = deps.DomainModels
// ... etc for ALL fields
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor: flatten ExecContext sub-structs into direct fields"
```

---

### Task 2: Create a universal handler migration wrapper

**Files:**
- Create: `mdl/executor/handler_migrate.go` (temporary, deleted after all handlers migrated)

- [ ] **Step 1: Create migration automation pattern**

Create a temporary helper that makes the bridge pattern explicit:

```go
// handler_migrate.go — temporary file, deleted after migration complete
package executor

// migrateHandler converts an old-style handler that expects *ExecContext
// into a new-style handler that uses *HandlerDeps directly.
//
// Usage:
//
//	// Old handler bridge:
//	r.RegisterFuture("Foo", func(ctx context.Context, stmt ast.Statement) error {
//	    ectx := NewExecContext(ctx, deps)
//	    return execFoo(ectx, stmt)
//	})
//
//	// New:
//	r.RegisterFuture("Foo", func(ctx context.Context, stmt ast.Statement) error {
//	    return ExecFooFn(ctx, stmt.(*ast.FooStmt), deps)
//	})
//
//	func ExecFooFn(ctx context.Context, s *ast.FooStmt, deps *HandlerDeps) error {
//	    // replace: ectx.Xxx → deps.Xxx, ectx.Context → ctx
//	    ...
//	}
```

This file is documentation-only; actual migration is done per-handler.

- [ ] **Step 2: Commit**

```bash
git add -A
git commit -m "docs: add handler migration guide"
```

---

### Tasks 3-37: Migrate each handler from NewExecContext to HandlerDeps

**Pattern for each handler migration:**

```
For each ExecXxxFuture function in handlers_future.go that calls NewExecContext:
  1. Create the HandlerDeps version: ExecXxxFn(ctx, stmt, deps)
  2. Inside: replace ectx.Xxx → deps.Xxx
  3. Replace the bridge in registerFutureOverlays()
  4. Build and test
  5. Commit
```

Here's the complete list of ~35 handlers to migrate, grouped by similarity:

**Group A: OData/Service handlers (Task 3, file scope)**

- [ ] **Task 3: ExecCreateODataClientFuture** (`handlers_future.go:3083`)
- [ ] **Task 4: ExecAlterODataClientFuture** (`handlers_future.go:3088`)
- [ ] **Task 5: ExecDropODataClientFuture** (`handlers_future.go:3093`)
- [ ] **Task 6: ExecCreateODataServiceFuture** (`handlers_future.go:3098`)
- [ ] **Task 7: ExecAlterODataServiceFuture** (`handlers_future.go:3103`)
- [ ] **Task 8: ExecDropODataServiceFuture** (`handlers_future.go:3108`)
- [ ] **Task 9: ExecCreateRestClientFuture** (`handlers_future.go:3137`)
- [ ] **Task 10: ExecDropRestClientFuture** (`handlers_future.go:3142`)
- [ ] **Task 11: ExecDescribeContractFromOpenAPIFuture** (`handlers_future.go:3147`)
- [ ] **Task 12: ExecCreateExternalEntityFuture** (`handlers_future.go:3164`)
- [ ] **Task 13: ExecCreateExternalEntitiesFuture** (`handlers_future.go:3169`)

**Group B: Configuration/Event handlers (Tasks 14-18)**

- [ ] **Task 14: ExecCreateConfigurationFuture** (`handlers_future.go:3063`)
- [ ] **Task 15: ExecDropConfigurationFuture** (`handlers_future.go:3068`)
- [ ] **Task 16: ExecCreateBusinessEventServiceFuture** (`handlers_future.go:3073`)
- [ ] **Task 17: ExecDropBusinessEventServiceFuture** (`handlers_future.go:3078`)
- [ ] **Task 18: ExecAlterSettingsFuture** (`handlers_future.go:3049`)

**Group C: Translation/Metadata handlers (Tasks 19-23)**

- [ ] **Task 19: ExecTranslateFuture** (`handlers_future.go:3054`)
- [ ] **Task 20: ExecDescribeTranslationsFuture** (`handlers_future.go:3199`)
- [ ] **Task 21: ExecShowFeaturesFuture** (`handlers_future.go:3208`)
- [ ] **Task 22: ExecSearchFuture** (`handlers_future.go:3229`)
- [ ] **Task 23: ExecRefreshCatalogFuture** (`handlers_future.go:3234`)

**Group D: Describe/Fragment handlers (Tasks 24-26)**

- [ ] **Task 24: execDescribeFragmentFromFuture** (`handlers_future.go:3250`)
- [ ] **Task 25: ExecAlterModuleJarDepFuture** (`handlers_future.go:3000`)
- [ ] **Task 26: ExecSelectFuture** (`handlers_future.go:3194`)

**Group E: Agent Editor handlers (Tasks 27-28)**

- [ ] **Task 27: ExecCreateModelFuture** (`handlers_future.go:3295`)
- [ ] **Task 28: ExecDropModelFuture** (`handlers_future.go:3300`)

**Group F: XxxFn bridge functions (Tasks 29-35)**

These are more complex — they bridge to functions like `execCreateModule` etc.

- [ ] **Task 29: ExecCreateModuleFn** (`handlers_future.go:3331`)
- [ ] **Task 30: ExecDropModuleFn** (`handlers_future.go:3336`)
- [ ] **Task 31: ExecCreatePageV3Fn** (`handlers_future.go:3341`)
- [ ] **Task 32: ExecCreateSnippetV3Fn** (`handlers_future.go:3346`)
- [ ] **Task 33: ExecAlterPageFn** (`handlers_future.go:3351`)
- [ ] **Task 34: `listODataClientsFn` etc** (`handlers_future.go:3356-3393`)
- [ ] **Task 35: `describeODataClientFn` etc** (`handlers_future.go:3380-3393`)

**Migration template for a typical handler (Task 3 example):**

```
File: mdl/executor/handlers_future.go:3083-3085

Before:
func ExecCreateODataClientFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
    ectx := NewExecContext(ctx, deps)
    return createODataClient(ectx, stmt.(*ast.CreateODataClientStmt))
}

After:
func ExecCreateODataClientFuture(ctx context.Context, stmt ast.Statement, deps *HandlerDeps) error {
    return ExecCreateODataClientFn(ctx, stmt.(*ast.CreateODataClientStmt), deps)
}

// Add new function:
func ExecCreateODataClientFn(ctx context.Context, s *ast.CreateODataClientStmt, deps *HandlerDeps) error {
    // Move the body of createODataClient() here, replacing:
    //   ectx.Xxx → deps.Xxx
    //   ectx.Context → ctx
    ...
}
```

For Tasks 29-35 (Fn patterns), the existing `ExecXxxFn()` bridges need to be inlined — the function they call (`execCreateModule` etc.) needs to be refactored to accept `*HandlerDeps` instead of `*ExecContext`.

---

### Task 36: Delete `execContextToDeps` bridge function

**Files:**
- Delete: content in `mdl/executor/executor_dispatch.go:889-973`

- [ ] **Step 1: Verify zero callers**

```bash
rg -n 'execContextToDeps' mdl/executor/
```

Must return zero results besides the definition.

- [ ] **Step 2: Delete the function**

Delete lines 889-973 from `executor_dispatch.go`.

- [ ] **Step 3: Build and test**

```bash
go build ./... && go test ./mdl/executor/... -count=1 -timeout 120s
```

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "refactor: remove execContextToDeps bridge (zero callers)"
```

---

### Task 37: Delete `NewExecContext` bridge function

**Files:**
- Delete: `mdl/executor/handlers_future.go:2898-2994`

- [ ] **Step 1: Verify zero callers**

```bash
rg -n 'NewExecContext' mdl/executor/
```

Must return zero results besides the definition.

- [ ] **Step 2: Delete the function**

Delete lines 2898-2994.

- [ ] **Step 3: Delete the temporary `handler_migrate.go` file**

```bash
rm mdl/executor/handler_migrate.go
```

- [ ] **Step 4: Build and test**

```bash
go build ./... && go test ./mdl/executor/... -count=1 -timeout 120s
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor: remove NewExecContext bridge (zero callers)"
```

---

### Task 38: Delete `ExecContext` struct + `initRoles()`

**Files:**
- Modify: `mdl/executor/executor.go:778-960`

- [ ] **Step 1: Verify zero production references to `ExecContext`**

```bash
rg -n 'ExecContext' mdl/executor/ --no-filename | grep -v '^\s*//' | grep -v '_test.go' | sort -u
```

Should show only the definition and `newExecContext()`.

- *Note: `domainmodel/handler.go` still references `executor.NewExecContext()` — this must be resolved in earlier tasks before proceeding.*

- [ ] **Step 2: Remove `ExecContext` struct definition**

Delete lines 778-850 (the struct definition) and lines 855-960 (initRoles).

Replace `newExecContext()` at `executor_dispatch.go:18-61` with `newHandlerDeps(ctx, deps)` that returns `*HandlerDeps`.

- [ ] **Step 3: Remove `ExecContext` type alias in `domainmodel/handler.go`**

If `domainmodel/handler.go` still uses `executor.NewExecContext`, resolve by passing `*HandlerDeps` directly.

- [ ] **Step 4: Build and test**

```bash
go build ./... && go test ./mdl/... -count=1 -timeout 180s
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor!: delete ExecContext struct

ExecContext and its initRoles() are removed. All handler code
uses HandlerDeps directly."
```

# Plan 1: FullBackend Dissolution — Narrow Interface Injection

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete `HandlerDeps.Backend` field (the deprecated `FullBackend`), replacing its 12 remaining call sites with narrow role interfaces.

**Architecture:** `FullBackend` (50-interface composition) is currently referenced in only 6 production code locations (non-comment, non-test). Replace each with the appropriate role interface that already exists in `HandlerDeps`. Then delete the `Backend` field, which proves no handler depends on the megainterface.

**Tech Stack:** Go, `mdl/backend/`, `mdl/executor/`

## Global Constraints

- Every commit must compile: `go build ./...`
- Every commit must pass existing tests: `go test ./mdl/...`
- Do NOT change test file behavior — only refactor production code paths
- The `Backend` field in `HandlerDeps` is deleted only at the end when all references are gone

---

### Task 1: Create `CacheInvalidator` interface + integrate into HandlerDeps

**Files:**
- Create: `mdl/backend/infrastructure.go`
- Modify: `mdl/executor/handler_deps.go:26`
- Modify: `mdl/executor/executor.go:980-993`

**Interfaces:**
- Consumes: `backend.FullBackend` (will be removed)
- Produces: `backend.CacheInvalidator` interface with `InvalidateCache()` method

- [ ] **Step 1: Create `mdl/backend/infrastructure.go`**

```go
package backend

// CacheInvalidator provides cache invalidation for backend storage.
// Extracted from FullBackend because it cross-cuts all domains — any
// mutation operation may need to invalidate storage-level caches.
type CacheInvalidator interface {
	InvalidateCache()
}
```

- [ ] **Step 2: Add role interface to MockBackend**

Read `mdl/backend/mock/mock_backend.go`. Add `InvalidateCacheFunc func()` field and implement `InvalidateCache()`:

In the MockBackend struct, add:
```go
InvalidateCacheFunc func()
```

Add the method:
```go
func (m *MockBackend) InvalidateCache() {
	if m.InvalidateCacheFunc != nil {
		m.InvalidateCacheFunc()
	}
}
```

Add compile-time check:
```go
var _ backend.CacheInvalidator = (*MockBackend)(nil)
```

- [ ] **Step 3: Add `CacheInvalidator` field to HandlerDeps**

In `mdl/executor/handler_deps.go`, add after existing role interfaces:

```go
CacheInvalidator backend.CacheInvalidator
```

- [ ] **Step 4: Update `ExecContext.InvalidateCache()` to use CacheInvalidator**

In `mdl/executor/executor.go:982-992`, replace `ctx.Backend` with dedicated role field:

```go
func (ctx *ExecContext) InvalidateCache() {
	ctx.Cache = nil
	if ctx.MetadataReader != nil {
		ctx.MetadataReader.InvalidateCache()
	}
	if ctx.CacheInvalidator != nil {
		ctx.CacheInvalidator.InvalidateCache()
	}
}
```

- [ ] **Step 5: Wire `CacheInvalidator` in `NewExecContext`**

In `mdl/executor/handlers_future.go:2943-2993`, add after the role field population block:

```go
ectx.CacheInvalidator = deps.CacheInvalidator
```

Also keep the fallback:
```go
if deps.CacheInvalidator == nil && deps.Backend != nil {
	ectx.CacheInvalidator = deps.Backend
}
```

- [ ] **Step 6: Run tests to verify no regression**

```bash
go build ./...
go test ./mdl/executor/... -count=1 -timeout 120s
```

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "refactor: extract CacheInvalidator interface from FullBackend"
```

---

### Task 2: Replace `ctx.Backend.(backend.BackendFactory)` type assertion

**Files:**
- Modify: `mdl/executor/executor.go:862`

- [ ] **Step 1: Replace type assertion with role interface check**

In `mdl/executor/executor.go:862` inside `initRoles()`:

```go
// Before:
bf, _ = ctx.Backend.(backend.BackendFactory)

// After (simplified — backendFactory is set directly in executor_connect.go):
// Remove this line entirely. backendFactory field is set directly.
```

The `backendFactory` field at `executor.go:792` is already set by `executor_connect.go` before `initRoles()` is called. The cast is dead code.

Verify by reading `executor_connect.go`:
```go
ctx.backendFactory = bf  // line ~30
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "refactor: remove dead BackendFactory type assertion in initRoles"
```

---

### Task 3: Replace `ctx.Backend.IsConnected()` with `ConnectionManager`

**Files:**
- Modify: `mdl/executor/executor.go:967-969`

- [ ] **Step 1: Remove the Backend fallback in Connected()**

In `mdl/executor/executor.go:963-971`:

```go
func (ctx *ExecContext) Connected() bool {
	if ctx.ConnectionManager != nil {
		return ctx.ConnectionManager.IsConnected()
	}
	// Fallback removed — ConnectionManager is always populated
	return false
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "refactor: remove Backend fallback in Connected()"
```

---

### Task 4: Replace `ectx.Backend.InvalidateCache()` in domainmodel/handler.go

**Files:**
- Modify: `mdl/executor/domainmodel/handler.go:77-79,87-89`

- [ ] **Step 1: Replace two `ectx.Backend.InvalidateCache()` calls**

In `mdl/executor/domainmodel/handler.go`:

Line 77:
```go
// Before:
if ectx.Backend != nil {
	ectx.Backend.InvalidateCache()
}

// After:
if ectx.CacheInvalidator != nil {
	ectx.CacheInvalidator.InvalidateCache()
}
```

Line 87 (same pattern):
```go
// Before:
if ectx.Backend != nil {
	ectx.Backend.InvalidateCache()
}

// After:
if ectx.CacheInvalidator != nil {
	ectx.CacheInvalidator.InvalidateCache()
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "refactor: use CacheInvalidator in domainmodel handler"
```

---

### Task 5: Remove `Backend` field from `HandlerDeps`

**Files:**
- Modify: `mdl/executor/handler_deps.go:26`
- Modify: `mdl/executor/handlers_future.go:2944-2949` (NewExecContext fallback)
- Modify: `mdl/executor/executor_dispatch.go:889-1071` (execContextToDeps)

- [ ] **Step 1: Verify no remaining references**

Run to confirm zero production references:
```bash
rg -n 'deps\.Backend\.' mdl/executor/ --no-filename | grep -v '^\s*//' | grep -v 'tests'
```

This must return zero results (the only match should be the `handlers_future.go:2944` comment).

- [ ] **Step 2: Remove `Backend` field from `HandlerDeps`**

In `mdl/executor/handler_deps.go:26`, delete:
```go
Backend backend.FullBackend
```

- [ ] **Step 3: Remove `Backend` field from `NewExecContext`**

In `mdl/executor/handlers_future.go:2904`, delete:
```go
Backend: deps.Backend,
```

And remove the fallback block at `2944-2949`:
```go
// Each field uses the deps value when non-nil, falling back to deps.Backend.
if deps.ConnectionManager != nil {
	ectx.ConnectionManager = deps.ConnectionManager
} else if deps.Backend != nil {
	ectx.ConnectionManager = deps.Backend
}
```

Replace with:
```go
ectx.ConnectionManager = deps.ConnectionManager
```

- [ ] **Step 4: Check for compilation errors**

```bash
go build ./... 2>&1
```

If compilation fails, fix each error by replacing `deps.Backend` with the appropriate role field. Each error identifies a hidden dependency we missed.

- [ ] **Step 5: Run tests**

```bash
go test ./mdl/... -count=1 -timeout 120s
```

- [ ] **Step 6: Add deprecation comment to `FullBackend` interface**

In `mdl/backend/backend.go:19`, update the doc comment to note that `HandlerDeps` no longer carries this interface.

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "refactor!: remove Backend field from HandlerDeps

HandlerDeps no longer carries backend.FullBackend. All handlers
now use narrow role interfaces instead.

Closes #FullBackend-dissolution"
```

---

### Task 6: Remove `ExecContext.Backend` field

**Files:**
- Modify: `mdl/executor/executor.go:789`

- [ ] **Step 1: Verify no remaining production references to `ctx.Backend`**

```bash
rg -n 'ctx\.Backend' mdl/executor/ --no-filename | grep -v '^\s*//' | grep -v '_test.go'
```

Should show only:
- `executor.go:789` (the field declaration itself)
- `executor.go:940-959` (the `initRoles()` fallback block assigning `ctx.Backend` to role fields)

- [ ] **Step 2: Remove `Backend` field from `ExecContext`**

In `mdl/executor/executor.go:786-789`, delete:
```go
Backend backend.FullBackend
```

and update the comment on `backendFactory`.
Also remove the `initRoles()` fallback block at lines ~940-959 that does:
```go
ctx.ModuleLister = ctx.Backend
ctx.ModuleWriter = ctx.Backend
// ... etc
```

Replace with a simplified fallback that does nothing (since backendFactory is always set):

```go
// Keep initRoles with only the bf path:
func (ctx *ExecContext) initRoles() {
	if ctx == nil {
		return
	}
	bf := ctx.backendFactory
	if bf == nil {
		return
	}
	// ... bf.ModuleLister() etc — keep these lines
}
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "refactor: remove Backend field from ExecContext"
```

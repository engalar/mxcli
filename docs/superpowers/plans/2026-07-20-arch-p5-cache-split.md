# Plan 5: executorCache Split — Domain-Scoped Invalidation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the monolithic `executorCache` with per-domain `domainCache[T]` wrappers. Replace `invalidateAllDocumentCaches()` with domain-scoped `Invalidate(CacheDomainEntity, CacheDomainMicroflow)`.

**Architecture:** A generic `domainCache[T]` struct encapsulates lazy-load + invalidate logic. `executorCache` holds one `*domainCache[T]` per domain. Write handlers call `deps.Cache.Invalidate(CacheDomainMicroflows)` instead of `invalidateAllDocumentCaches()`.

**Tech Stack:** Go, `mdl/executor/`

## Global Constraints

- Every commit must compile: `go build ./...`
- Every commit must pass: `go test ./mdl/executor/... -count=1`
- Cache behavior must be identical before and after — the only change is scope of invalidation
- Cache keys (struct field names) must NOT change, only the underlying type

---

### Task 1: Define `domainCache[T]` and `CacheDomain` type

**Files:**
- Create: `mdl/executor/domain_cache.go`

- [ ] **Step 1: Create the generic cache wrapper**

```go
package executor

import "sync"

// CacheDomain identifies a domain for cache invalidation.
type CacheDomain int

const (
	CacheDomainModules    CacheDomain = iota
	CacheDomainEntities
	CacheDomainMicroflows
	CacheDomainNanoflows
	CacheDomainPages
	CacheDomainSnippets
	CacheDomainLayouts
	CacheDomainWorkflows
	CacheDomainEnumerations
	CacheDomainConstants
	CacheDomainJavaActions
	CacheDomainJavaScriptActions
	CacheDomainSettings
	CacheDomainNavigation
)

// domainCache provides lazy-load + invalidate for a single domain's data.
type domainCache[T any] struct {
	mu     sync.RWMutex
	data   []T
	loaded bool
	loadFn func() ([]T, error)
}

func newDomainCache[T any](loadFn func() ([]T, error)) *domainCache[T] {
	return &domainCache[T]{loadFn: loadFn}
}

func (c *domainCache[T]) Get() ([]T, error) {
	c.mu.RLock()
	if c.loaded {
		defer c.mu.RUnlock()
		return c.data, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	// Double-check after acquiring write lock
	if c.loaded {
		return c.data, nil
	}
	var err error
	c.data, err = c.loadFn()
	c.loaded = true
	return c.data, err
}

func (c *domainCache[T]) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loaded = false
	c.data = nil
}
```

- [ ] **Step 2: Build and test**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "feat: add domainCache[T] generic wrapper"
```

---

### Task 2: Refactor `executorCache` to use `domainCache`

**Files:**
- Modify: `mdl/executor/executor_cache.go` (or the file where `executorCache` is defined)

- [ ] **Step 1: Find executorCache definition**

```bash
rg -n 'type executorCache struct' mdl/executor/
```

Read the current struct to map all cache fields.

- [ ] **Step 2: Replace plain fields with `*domainCache[T]`**

```go
type executorCache struct {
	// Before:
	microflowsGen []*genMf.Microflow

	// After:
	microflowsGen *domainCache[*genMf.Microflow]
}
```

Initialize each in a constructor:
```go
func newExecutorCache(backend backend.FullBackend) *executorCache {
	return &executorCache{
		microflowsGen: newDomainCache(func() ([]*genMf.Microflow, error) {
			return backend.ListMicroflowsGen()
		}),
		// ... per domain
	}
}
```

- [ ] **Step 3: Update accessor methods**

Find the `executorCache.MicroflowsGen()` (or similar) accessor. Replace:

```go
// Before:
func (c *executorCache) MicroflowsGen() []*genMf.Microflow {
	return c.microflowsGen
}

// After:
func (c *executorCache) MicroflowsGen() ([]*genMf.Microflow, error) {
	return c.microflowsGen.Get()
}
```

Note: signature changes from `() []T` to `() ([]T, error)` — update all callers.

- [ ] **Step 4: Build and test**

```bash
go build ./... && go test ./mdl/executor/... -count=1 -timeout 120s
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor: executorCache fields use domainCache[T] wrappers"
```

---

### Task 3: Replace `invalidateAllDocumentCaches` with scoped invalidation

**Files:**
- Modify: `mdl/executor/executor_cache.go` (or the registration file)

- [ ] **Step 1: Find all callers of `invalidateAllDocumentCaches`**

```bash
rg -n 'invalidateAllDocumentCaches' mdl/executor/
```

- [ ] **Step 2: Add scoped `Invalidate` method**

```go
func (c *executorCache) Invalidate(domains ...CacheDomain) {
	for _, d := range domains {
		switch d {
		case CacheDomainModules:
		case CacheDomainEntities:
			c.entities.Invalidate()
		case CacheDomainMicroflows:
			c.microflowsGen.Invalidate()
		case CacheDomainNanoflows:
			c.nanoflowsGen.Invalidate()
		case CacheDomainPages:
			c.pages.Invalidate()
		// ... per domain
		}
	}
}
```

- [ ] **Step 3: Replace each `invalidateAllDocumentCaches()` call with scoped version**

```go
// Before: cache invalidation after microflow write
invalidateAllDocumentCaches(deps)

// After:
deps.Cache.Invalidate(CacheDomainMicroflows, CacheDomainModules)
```

Determine the correct domain for each call site by examining what was written:

| Write operation | Cache domains to invalidate |
|----------------|---------------------------|
| CreateEntity | `CacheDomainEntities, CacheDomainModules` |
| DeleteMicroflow | `CacheDomainMicroflows` |
| CreatePage | `CacheDomainPages, CacheDomainModules` |
| CreateWorkflow | `CacheDomainWorkflows` |
| AlterSecurity | `CacheDomainEntities` (entity access rules are cached with entities) |

- [ ] **Step 4: Remove `invalidateAllDocumentCaches` function**

After all callers are migrated, delete the function definition.

```go
// Delete:
func invalidateAllDocumentCaches(ctx *ExecContext) { ... }
```

- [ ] **Step 5: Build and test**

```bash
go build ./... && go test ./mdl/executor/... -count=1 -timeout 120s
```

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "refactor: replace full cache invalidation with domain-scoped"

All write handlers now only invalidate the cache domain(s) they
actually modify. No unnecessary reloads of unrelated domains.
```

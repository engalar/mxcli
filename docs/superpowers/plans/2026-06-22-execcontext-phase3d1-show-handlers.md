# ExecContext Elimination — Phase 3d-1: Pure Query Handlers

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate all show/describe handlers from `StmtHandler` (`*ExecContext`) to `StmtHandlerFunc` (`context.Context` + explicit deps). ~55 handlers total across `show_registry.go` and `describe_registry.go`.

**Architecture:** Each handler function currently takes `*ExecContext` and accesses fields like `ctx.Output`, `ctx.ModuleLister`, etc. Migrated versions take `context.Context` + explicit interfaces. Registration uses `RegisterFuture` with closure-captured deps from `HandlerDeps`.

**Tech Stack:** Go 1.24, `mdl/executor` package

**Spec:** `docs/superpowers/specs/2026-06-20-solid-refactoring-design.md` Phase 3d

## Global Constraints

- Every commit must compile (`go build ./mdl/executor/...`)
- Every commit must pass `go vet ./mdl/executor/...`
- Every commit must pass existing tests (`go test ./mdl/executor/... -count=1`)
- New handler functions go in `mdl/executor/handlers_future.go`
- Old handler functions stay unchanged (fallback in Dispatch)
- Registration in `registerFutureOverlays()` in `handler_deps.go`
- Each migrated handler must be added to TestRegistry_AllStatementTypesCovered (already covered)

## Migration Template

For each handler function `func listXxx(ctx *ExecContext, ...) error`:

### Step 1: Identify ExecContext fields used
Most show handlers use:
- `ctx.Output` (io.Writer) — for formatted output
- `ctx.Format` (OutputFormat) — for output format (table/json)
- Domain-specific reader (e.g., `ctx.ModuleLister`, `ctx.EnumerationReader`)

### Step 2: Create new function signature
```go
func listXxxFuture(ctx context.Context, ...args, output io.Writer, format OutputFormat, reader backend.XxxReader) error {
```

### Step 3: Replace `ctx.Output` → `output`, `ctx.Format` → `format`
### Step 4: Replace `ctx.XxxReader.SomeMethod()` → `reader.SomeMethod()`
### Step 5: Register in `registerFutureOverlays()` via `RegisterFuture`

---

### Task 1: Module/Version/Catalog show handlers (5 handlers)

**Files:**
- Modify: `mdl/executor/handlers_future.go`
- Modify: `mdl/executor/handler_deps.go`
- Read: `mdl/executor/show_registry.go`

**Handlers to migrate:**
```
ast.ShowModules:           listModules(ctx)
ast.ShowCatalogTables:     execShowCatalogTables(ctx)
ast.ShowCatalogStatus:     execShowCatalogStatus(ctx)
ast.ShowVersion:           listVersion(ctx)
ast.ShowFragments:         listFragments(ctx)
```

**Deps needed from HandlerDeps:** Output, Format, ModuleLister, ConnectionManager, MetadataReader

- [ ] **Step 1: Read the existing handlers**

```bash
grep -n 'listModules\|execShowCatalogTables\|execShowCatalogStatus\|listVersion\|listFragments' mdl/executor/ --include='*.go' | grep -v _test | head -20
```

- [ ] **Step 2: Create migrated functions in handlers_future.go**

```go
func listModulesFuture(ctx context.Context, output io.Writer, format OutputFormat, ml backend.ModuleLister) error {
    modules, err := ml.ListModules()
    if err != nil {
        return err
    }
    return printModules(output, format, modules)
}
```

For each of the 5 handlers, follow the same pattern: identify the ExecContext fields used, convert to explicit deps.

- [ ] **Step 3: Register in handler_deps.go**

In `registerFutureOverlays()`, add:
```go
r.RegisterFuture("Show", func(ctx context.Context, stmt ast.Statement) error {
    s := stmt.(*ast.ShowStmt)
    switch s.ObjectType {
    case ast.ShowModules:
        return listModulesFuture(ctx, deps.Output, e.format, deps.ModuleLister)
    // ... etc
    default:
        return fmt.Errorf("unknown show type: %s", s.ObjectType)
    }
})
```

- [ ] **Step 4: Build and test**

```bash
go build ./mdl/executor/... && go vet ./mdl/executor/... && go test ./mdl/executor/... -count=1 2>&1 | tail -5
```

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "refactor(executor): migrate 5 module/version show handlers (Phase 3d-1a)"
```

---

### Task 2: Enumeration/Constant show handlers (4 handlers)

**Files:**
- Modify: `mdl/executor/handlers_future.go`
- Modify: `mdl/executor/handler_deps.go`
- Read: `mdl/executor/show_registry.go`

**Handlers:**
```
ast.ShowEnumerations:      listEnumerations(ctx, s.InModule)
ast.ShowConstants:         listConstants(ctx, s.InModule)
ast.ShowConstantValues:    listConstantValues(ctx, s.InModule)
```

**Deps:** Output, Format, EnumerationReader, ConstantReader

- [ ] **Step 1-5: Follow template pattern**

---

### Task 3: Entity/Association show handlers (4 handlers)

**Files:**
- Modify: `mdl/executor/handlers_future.go`
- Modify: `mdl/executor/handler_deps.go`

**Handlers:**
```
ast.ShowEntities:          listEntitiesGen(ctx, s.InModule)
ast.ShowEntity:            listEntity(ctx, s.Name)
ast.ShowAssociations:      listAssociations(ctx, s.InModule)
ast.ShowAssociation:       listAssociation(ctx, s.Name)
```

**Deps:** Output, Format, DomainModelReader, ModuleLister

---

### Task 4: Microflow/Nanoflow/Page show handlers (6 handlers)

**Files:**
- Modify: `mdl/executor/handlers_future.go`
- Modify: `mdl/executor/handler_deps.go`

**Handlers:**
```
ast.ShowMicroflows:        listMicroflows(ctx, s.InModule)
ast.ShowNanoflows:         listNanoflows(ctx, s.InModule)
ast.ShowPages:             listPagesGen(ctx, s.InModule)
ast.ShowSnippets:          listSnippetsGen(ctx, s.InModule)
ast.ShowLayouts:           listLayoutsGen(ctx, s.InModule)
ast.ShowJavaActions:       listJavaActionsGen(ctx, s.InModule)
ast.ShowJavaScriptActions: listJavaScriptActionsGen(ctx, s.InModule)
```

**Deps:** Output, Format, MicroflowReader, PageReader, JavaActionReader, JavaScriptActionReader

---

### Task 5: Workflow/BusinessEvent show handlers (4 handlers)

**Files:**
- Modify: `mdl/executor/handlers_future.go`
- Modify: `mdl/executor/handler_deps.go`

**Handlers:**
```
ast.ShowWorkflows:                listWorkflowsGen(ctx, s.InModule)
ast.ShowBusinessEventServices:    listBusinessEventServices(ctx, s.InModule)
ast.ShowBusinessEventClients:     listBusinessEventClients(ctx, s.InModule)
ast.ShowBusinessEvents:           listBusinessEvents(ctx, s.InModule)
```

**Deps:** Output, Format, WorkflowReader

---

### Task 6: Security show handlers (10 handlers)

**Files:**
- Modify: `mdl/executor/handlers_future.go`
- Modify: `mdl/executor/handler_deps.go`

**Handlers:**
```
ast.ShowProjectSecurity:   listProjectSecurityGen(ctx)
ast.ShowModuleRoles:       listModuleRolesGen(ctx, s.InModule)
ast.ShowUserRoles:         listUserRolesGen(ctx)
ast.ShowDemoUsers:         listDemoUsersGen(ctx)
ast.ShowAccessOn:          listAccessOnEntityGen(ctx, s.Name)
ast.ShowAccessOnMicroflow: listAccessOnMicroflowGen(ctx, s.Name)
ast.ShowAccessOnPage:      listAccessOnPageGen(ctx, s.Name)
ast.ShowAccessOnWorkflow:  listAccessOnWorkflow(ctx, s.Name)
ast.ShowAccessOnNanoflow:  listAccessOnNanoflowGen(ctx, s.Name)
ast.ShowSecurityMatrix:    listSecurityMatrixGen(ctx, s.InModule)
```

**Deps:** Output, Format, SecurityReader, ModuleLister

---

### Task 7: Navigation/Settings show handlers (7 handlers)

**Files:**
- Modify: `mdl/executor/handlers_future.go`
- Modify: `mdl/executor/handler_deps.go`

**Handlers:**
```
ast.ShowNavigation:        listNavigation(ctx)
ast.ShowNavigationMenu:    listNavigationMenu(ctx, s.Name)
ast.ShowNavigationHomes:   listNavigationHomes(ctx)
ast.ShowSettings:          listSettings(ctx)
ast.ShowLanguages:         listLanguages(ctx)
ast.ShowSupportedLanguages: listSupportedLanguages(ctx)
ast.ShowStructure:         execShowStructureGen(ctx, s)
```

**Deps:** Output, Format, NavigationReader, SettingsReader

---

### Task 8: Describe handlers (first 5)

**Files:**
- Modify: `mdl/executor/handlers_future.go`
- Modify: `mdl/executor/handler_deps.go`
- Read: `mdl/executor/describe_registry.go`

**Handlers:**
```
ast.DescribeModule:        describe...   (in describe_registry.go)
ast.DescribeEntities:      describe...
ast.DescribeMicroflow:     describe...
ast.DescribePage:          describe...
ast.DescribeEnumeration:   describe...
```

**Deps:** Output, Format, domain-specific readers

---

### Task 9: Verify and full test

- [ ] **Run full test suite**

```bash
go build ./mdl/executor/... && go vet ./mdl/executor/... && go test ./mdl/executor/... -count=1 2>&1 | tail -5
```

- [ ] **Check coverage test still passes (new handler types not accidentally dropped)**

```bash
go test ./mdl/executor/... -run TestRegistry_AllStatementTypesCovered -v 2>&1
```

- [ ] **Final commit**

```bash
git add -A && git commit -m "refactor(executor): Phase 3d-1 complete — all show/describe handlers migrated"
```

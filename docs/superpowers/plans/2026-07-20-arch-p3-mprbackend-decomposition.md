# Plan 3: MprBackend Decomposition — Domain-Specific Backends

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Break the single `MprBackend` struct (100 files, 30+ interfaces) into ~15 focused domain backend structs, each with explicit constructor dependencies.

**Architecture:** Each domain (Module, Entity, Microflow, Page, Workflow, Security, etc.) gets its own private struct type. `MprBackend` becomes a facade that composes them. `BackendFactory` returns the specific domain backend for each role interface. No external callers change — only internal file shuffling.

**Tech Stack:** Go, `mdl/backend/mpr/`

## Global Constraints

- Zero external API changes — `MprBackend` still implements all role interfaces
- Zero test changes — mock backend is unchanged
- Every commit must pass: `go build ./... && go test ./mdl/backend/...`
- Do NOT move or rename existing methods during extraction — preserve method signatures exactly
- Each extracted domain is a separate commit for auditability

---

### Task 1: Extract module backend

**Files:**
- Create: `mdl/backend/mpr/mpr_module.go`
- Modify: `mdl/backend/mpr/backend.go` (add `module *moduleBackend` field)

- [ ] **Step 1: Read the existing module methods**

```bash
rg -n 'func \(b \*MprBackend\)' mdl/backend/mpr/modules.go
```

Identify all methods that belong to `ModuleLister`/`ModuleWriter`/`ModuleSettingsReader`/`ModuleSettingsWriter`.

- [ ] **Step 2: Create `mpr_module.go`**

```go
package mprbackend

type moduleBackend struct {
	reader *modelsdkmpr.Reader
}

func (b *moduleBackend) ListModules() ([]*model.Module, error) {
	return b.reader.ListModules()
}

func (b *moduleBackend) GetModule(id model.ID) (*model.Module, error) {
	return b.reader.ListModulesByID(id) // or equivalent
}

// ... all ModuleLister, ModuleWriter, ModuleSettingsReader, ModuleSettingsWriter methods
```

Exact method signatures are copied from existing `MprBackend` methods in `modules.go`.

- [ ] **Step 3: Wire into MprBackend**

In `backend.go`, add field:
```go
type MprBackend struct {
	module *moduleBackend
	// ... existing fields kept for now
}
```

In `NewMprBackend` (or the existing factory function):
```go
module: &moduleBackend{reader: reader},
```

- [ ] **Step 4: Delegate facade methods**

For each method on `MprBackend` that belongs to Module domain, replace the implementation with a delegation:

```go
func (b *MprBackend) ListModules() ([]*model.Module, error) {
	return b.module.ListModules()
}
```

- [ ] **Step 5: Build and test**

```bash
go build ./...
go test ./mdl/backend/... -count=1 -timeout 120s
```

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "refactor: extract moduleBackend from MprBackend"
```

---

### Tasks 2-11: Extract remaining domain backends (same pattern)

Each task follows the same steps as Task 1: read existing methods, create new file+struct, wire into facade, delegate.

- [ ] **Task 2: Extract entity/domainmodel backend**

Create `mpr_entity.go` with `entityBackend` struct. Move methods from `domainmodel.go`, `domainmodel_v2.go`, `domainmodel_layout.go`.

```go
type entityBackend struct {
	reader *modelsdkmpr.Reader
	writer *modelsdkmpr.Writer
	mdl    *modelsdk.Model
}

// DomainModelReader, DomainModelWriter methods
```

- [ ] **Task 3: Extract microflow backend**

Create `mpr_microflow.go` with `microflowBackend` struct. Move methods from `services_create_v2.go`, `convert.go`, `convert_reader.go`.

```go
type microflowBackend struct {
	reader *modelsdkmpr.Reader
	writer *modelsdkmpr.Writer
}
```

- [ ] **Task 4: Extract page backend**

Create `mpr_page.go` with `pageBackend` struct. Move methods from `page_model.go`, `page_mutator.go`, `page_mutator_v2.go`.

```go
type pageBackend struct {
	reader        *modelsdkmpr.Reader
	writer        *modelsdkmpr.Writer
	widgetBuilder *widgetBuilder
}
```

- [ ] **Task 5: Extract workflow backend**

Create `mpr_workflow.go` with `workflowBackend`.

- [ ] **Task 6: Extract security backend**

Create `mpr_security.go` with `securityBackend`.

- [ ] **Task 7: Extract Java actions backend**

Create `mpr_java.go` with `javaBackend`.

- [ ] **Task 8: Extract settings backend**

Create `mpr_settings.go` with `settingsBackend`. Only needs `*mpr.Reader`.

- [ ] **Task 9: Extract navigation backend**

Create `mpr_navigation.go` with `navigationBackend`.

- [ ] **Task 10: Extract agent editor backend**

Create `mpr_agent.go` with `agentEditorBackend`.

- [ ] **Task 11: Extract translation backend**

Create `mpr_translation.go` with `translationBackend`.

---

### Task 12: Remove MprBackend obsolete fields

**Files:**
- Modify: `mdl/backend/mpr/backend.go`

- [ ] **Step 1: Audit which fields `MprBackend` still needs directly**

Methods that haven't been delegated (e.g., constructors, wiring) should become constructor-only locals.

- [ ] **Step 2: Shrink `MprBackend` struct**

Remove fields that are now owned by sub-backends. Each sub-backend holds its own Reader/Writer references, so `MprBackend` no longer needs them as fields.

```go
type MprBackend struct {
	// Only sub-backends remain:
	module    *moduleBackend
	entity    *entityBackend
	microflow *microflowBackend
	page      *pageBackend
	workflow  *workflowBackend
	security  *securityBackend
	java      *javaBackend
	settings  *settingsBackend
	navigation *navigationBackend
	agent     *agentEditorBackend
	translation *translationBackend
}
```

- [ ] **Step 3: Build and test**

```bash
go build ./... && go test ./mdl/backend/... -count=1 -timeout 120s
```

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "refactor: MprBackend holds sub-backends, no raw reader/writer fields"
```

---

### Task 13: Update BackendFactory to return sub-backends

**Files:**
- Modify: `mdl/backend/mpr/factory.go`

- [ ] **Step 1: Update factory methods**

In the `mprBackendFactory` struct (or equivalent `NewMprBackend` return), ensure each `BackendFactory` method returns the appropriate sub-backend:

```go
func (b *MprBackend) ModuleLister() backend.ModuleLister {
	return b.module
}

func (b *MprBackend) DomainModelReader() backend.DomainModelReader {
	return b.entity
}
```

- [ ] **Step 2: Build and test**

```bash
go build ./... && go test ./mdl/backend/... -count=1 -timeout 120s
```

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "refactor: BackendFactory returns sub-backend instances"
```

# MOVE Command SOLID Refactor + New Document Types

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the `execMove` switch-statement with an OCP-compliant `documentMover` registry pattern, extract a generic `MoveDocumentGen` backend method to eliminate 3 identical thin wrappers, and add MOVE support for JAVA ACTION, JAVASCRIPT ACTION, LAYOUT, and WORKFLOW.

**Architecture:** Introduce `documentMover` interface in a new `cmd_move_registry.go` file; each document type is a small struct implementing `find / moveToContainer / crossModuleHook / invalidate / label`. The `execMove` dispatcher shrinks to pure orchestration — no type-specific logic. Entity stays a special case (embedded in domain model, not a standalone unit). New types only need a registry entry + grammar + AST + visitor; no new backend methods because all gen-typed moves reuse `MoveDocumentGen`.

**Tech Stack:** Go (executor/backend/AST/visitor), ANTLR4 (grammar)

---

## File Map

| Action | File | What changes |
|--------|------|-------------|
| Modify | `mdl/backend/page.go` | Add `MoveDocumentGen` to `PageBackend` interface |
| Modify | `mdl/backend/mpr/backend.go` | Implement `MoveDocumentGen`; deprecate `MovePageGen`, `MoveSnippetGen`, `MoveLayoutGen` |
| Modify | `mdl/backend/mock/mock_backend.go` | Add `MoveDocumentGenFunc` field |
| Modify | `mdl/backend/mock/mock_page.go` | Add mock impl for `MoveDocumentGen` |
| Create | `mdl/executor/cmd_move_registry.go` | `documentMover` interface + registry + all 11 mover structs |
| Modify | `mdl/executor/cmd_move.go` | Remove per-type functions; keep `execMove`, `moveEntity`, `updateQualifiedNameRefs` |
| Modify | `mdl/ast/ast.go` | Add 4 new `DocumentType` constants |
| Modify | `mdl/grammar/MDLParser.g4` | Add JAVA ACTION, JAVASCRIPT ACTION, LAYOUT, WORKFLOW to `moveStatement` |
| Modify | `mdl/visitor/visitor_entity.go` | Add 4 new type detections in `ExitMoveStatement` |
| Modify | `mdl/executor/cmd_move_test.go` | Tests for new types (or new file if absent) |

---

### Task 1: Generic MoveDocumentGen backend method

**Files:**
- Modify: `mdl/backend/page.go`
- Modify: `mdl/backend/mpr/backend.go`
- Modify: `mdl/backend/mock/mock_backend.go`
- Modify: `mdl/backend/mock/mock_page.go`

The three existing methods (`MovePageGen`, `MoveSnippetGen`, `MoveLayoutGen`) all call `b.msdkWriter.UpdateUnitContainer(string(id), string(containerID))`. Replace them with one generic method. Keep the three old methods as deprecated aliases pointing to `MoveDocumentGen` so callers in `cmd_move.go` are migrated atomically in Task 2.

- [ ] **Step 1: Add MoveDocumentGen to the PageBackend interface**

In `mdl/backend/page.go`, find the block containing `MovePageGen` and add before it:

```go
// MoveDocumentGen moves any gen-typed document unit to a new container.
// This is the single low-level move operation; type-specific helpers are aliases.
MoveDocumentGen(id, containerID model.ID) error
```

- [ ] **Step 2: Implement MoveDocumentGen in MprBackend**

In `mdl/backend/mpr/backend.go`, add before `MovePageGen`:

```go
func (b *MprBackend) MoveDocumentGen(id, containerID model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("MoveDocumentGen: no modelsdk writer")
	}
	return b.msdkWriter.UpdateUnitContainer(string(id), string(containerID))
}
```

- [ ] **Step 3: Add mock stub**

In `mdl/backend/mock/mock_backend.go`, in the struct fields block near `MovePageGenFunc`, add:
```go
MoveDocumentGenFunc func(id, containerID model.ID) error
```

In `mdl/backend/mock/mock_page.go`, add:
```go
func (m *MockBackend) MoveDocumentGen(id, containerID model.ID) error {
	if m.MoveDocumentGenFunc != nil {
		return m.MoveDocumentGenFunc(id, containerID)
	}
	return fmt.Errorf("MockBackend.MoveDocumentGen not configured")
}
```

- [ ] **Step 4: Verify it compiles**

```bash
go build ./mdl/... 2>&1
```

Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add mdl/backend/page.go mdl/backend/mpr/backend.go \
    mdl/backend/mock/mock_backend.go mdl/backend/mock/mock_page.go
git commit -m "feat(backend): add generic MoveDocumentGen to replace per-type move wrappers"
```

---

### Task 2: documentMover interface + registry + all mover structs

**Files:**
- Create: `mdl/executor/cmd_move_registry.go`

This file contains the interface, the registry map, and all 11 mover structs (8 existing + 4 new, minus Entity which stays special). Entity is NOT in the registry.

- [ ] **Step 1: Write a registry coverage test**

Create `mdl/executor/cmd_move_registry_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0
package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// TestDocumentMoverRegistry_Coverage confirms every supported non-Entity document type
// has a registered mover with a non-empty label.
func TestDocumentMoverRegistry_Coverage(t *testing.T) {
	expected := []ast.DocumentType{
		ast.DocumentTypePage,
		ast.DocumentTypeMicroflow,
		ast.DocumentTypeSnippet,
		ast.DocumentTypeNanoflow,
		ast.DocumentTypeEnumeration,
		ast.DocumentTypeConstant,
		ast.DocumentTypeDatabaseConnection,
		ast.DocumentTypeJavaAction,
		ast.DocumentTypeJavaScriptAction,
		ast.DocumentTypeLayout,
		ast.DocumentTypeWorkflow,
	}
	for _, dt := range expected {
		m, ok := documentMoverRegistry[dt]
		if !ok {
			t.Errorf("documentMoverRegistry missing entry for %q", dt)
			continue
		}
		if m.label() == "" {
			t.Errorf("mover for %q has empty label", dt)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./mdl/executor/ -run TestDocumentMoverRegistry_Coverage -v 2>&1
```

Expected: FAIL — `documentMoverRegistry` undefined, `DocumentTypeJavaAction` etc. undefined.

- [ ] **Step 3: Add new DocumentType constants to ast.go**

In `mdl/ast/ast.go`, add 4 constants after `DocumentTypeDatabaseConnection`:

```go
DocumentTypeJavaAction         DocumentType = "JAVA ACTION"
DocumentTypeJavaScriptAction   DocumentType = "JAVASCRIPT ACTION"
DocumentTypeLayout             DocumentType = "LAYOUT"
DocumentTypeWorkflow           DocumentType = "WORKFLOW"
```

- [ ] **Step 4: Create cmd_move_registry.go**

Create `mdl/executor/cmd_move_registry.go`:

```go
// SPDX-License-Identifier: Apache-2.0

// cmd_move_registry.go — OCP-compliant move registry.
//
// Each document type implements documentMover. To add a new moveable type:
//   1. Add a DocumentType constant in mdl/ast/ast.go
//   2. Add a mover struct here (find/moveToContainer/crossModuleHook/invalidate/label)
//   3. Register it in documentMoverRegistry
//   4. Add the type to the grammar + visitor

package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
)

// documentMover abstracts the move operation for a single document type.
type documentMover interface {
	find(ctx *ExecContext, name ast.QualifiedName) (model.ID, error)
	moveToContainer(ctx *ExecContext, id model.ID, targetContainerID model.ID) error
	crossModuleHook(ctx *ExecContext, id model.ID, name ast.QualifiedName, targetModule *model.Module) error
	invalidate(ctx *ExecContext)
	label() string
}

// noopCrossModule is embedded by movers that require no extra cross-module work.
type noopCrossModule struct{}

func (noopCrossModule) crossModuleHook(_ *ExecContext, _ model.ID, _ ast.QualifiedName, _ *model.Module) error {
	return nil
}

// documentMoverRegistry maps each moveable document type to its mover.
// Entity is excluded — it has a fundamentally different data structure.
var documentMoverRegistry = map[ast.DocumentType]documentMover{
	ast.DocumentTypePage:               pageMoverImpl{},
	ast.DocumentTypeMicroflow:          microflowMoverImpl{},
	ast.DocumentTypeSnippet:            snippetMoverImpl{},
	ast.DocumentTypeNanoflow:           nanoflowMoverImpl{},
	ast.DocumentTypeEnumeration:        enumerationMoverImpl{},
	ast.DocumentTypeConstant:           constantMoverImpl{},
	ast.DocumentTypeDatabaseConnection: databaseConnectionMoverImpl{},
	ast.DocumentTypeJavaAction:         javaActionMoverImpl{},
	ast.DocumentTypeJavaScriptAction:   javaScriptActionMoverImpl{},
	ast.DocumentTypeLayout:             layoutMoverImpl{},
	ast.DocumentTypeWorkflow:           workflowMoverImpl{},
}

// ── pageMoverImpl ────────────────────────────────────────────────────────────

type pageMoverImpl struct{}

func (pageMoverImpl) label() string { return "page" }

func (pageMoverImpl) find(ctx *ExecContext, name ast.QualifiedName) (model.ID, error) {
	pairs, err := listPagesWithContainerGen(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("list pages", err)
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("build hierarchy", err)
	}
	for _, pair := range pairs {
		if pair.Elem == nil {
			continue
		}
		modName := h.GetModuleName(h.FindModuleID(model.ID(pair.ContainerID)))
		if modName == name.Module && pair.Elem.Name() == name.Name {
			return model.ID(pair.Elem.ID()), nil
		}
	}
	return "", mdlerrors.NewNotFound("page", name.String())
}

func (pageMoverImpl) moveToContainer(ctx *ExecContext, id, containerID model.ID) error {
	return ctx.Backend.MoveDocumentGen(id, containerID)
}

func (pageMoverImpl) crossModuleHook(ctx *ExecContext, id model.ID, _ ast.QualifiedName, targetModule *model.Module) error {
	pairs, err := listPagesWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list pages for role remap", err)
	}
	for _, pair := range pairs {
		if pair.Elem == nil || model.ID(pair.Elem.ID()) != id {
			continue
		}
		current := pair.Elem.AllowedRolesQualifiedNames()
		currentIDs := make([]model.ID, len(current))
		for i, qn := range current {
			currentIDs[i] = model.ID(qn)
		}
		remapped := remapDocumentAccessRoles(ctx, targetModule, currentIDs)
		return ctx.Backend.UpdateAllowedRoles(id, documentRoleStrings(remapped))
	}
	return nil
}

func (pageMoverImpl) invalidate(ctx *ExecContext) { invalidatePagesGenCache(ctx) }

// ── microflowMoverImpl ───────────────────────────────────────────────────────

type microflowMoverImpl struct{}

func (microflowMoverImpl) label() string { return "microflow" }

func (microflowMoverImpl) find(ctx *ExecContext, name ast.QualifiedName) (model.ID, error) {
	if ctx.Microflows == nil {
		return "", mdlerrors.NewBackend("microflows repo unavailable", nil)
	}
	mfs, err := listMicroflowsWithContainerGen(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("list microflows", err)
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("build hierarchy", err)
	}
	for _, item := range mfs {
		modName := h.GetModuleName(h.FindModuleID(item.ContainerUUID))
		if modName == name.Module && item.MF.Name() == name.Name {
			return model.ID(item.MF.ID()), nil
		}
	}
	return "", mdlerrors.NewNotFound("microflow", name.String())
}

func (microflowMoverImpl) moveToContainer(ctx *ExecContext, id, containerID model.ID) error {
	return ctx.Microflows.Move(id, string(containerID))
}

func (microflowMoverImpl) crossModuleHook(ctx *ExecContext, id model.ID, _ ast.QualifiedName, targetModule *model.Module) error {
	mfs, err := listMicroflowsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list microflows for role remap", err)
	}
	for _, item := range mfs {
		if model.ID(item.MF.ID()) != id {
			continue
		}
		existing := item.MF.AllowedModuleRolesQualifiedNames()
		currentIDs := make([]model.ID, len(existing))
		for i, qn := range existing {
			currentIDs[i] = model.ID(qn)
		}
		remapped := remapDocumentAccessRoles(ctx, targetModule, currentIDs)
		item.MF.SetAllowedModuleRolesQualifiedNames(documentRoleStrings(remapped))
		return ctx.Microflows.Update(item.MF)
	}
	return nil
}

func (microflowMoverImpl) invalidate(ctx *ExecContext) { invalidateMicroflowsCache(ctx) }

// ── snippetMoverImpl ─────────────────────────────────────────────────────────

type snippetMoverImpl struct{ noopCrossModule }

func (snippetMoverImpl) label() string { return "snippet" }

func (snippetMoverImpl) find(ctx *ExecContext, name ast.QualifiedName) (model.ID, error) {
	pairs, err := listSnippetsWithContainerGen(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("list snippets", err)
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("build hierarchy", err)
	}
	for _, pair := range pairs {
		if pair.Elem == nil {
			continue
		}
		modName := h.GetModuleName(h.FindModuleID(model.ID(pair.ContainerID)))
		if modName == name.Module && pair.Elem.Name() == name.Name {
			return model.ID(pair.Elem.ID()), nil
		}
	}
	return "", mdlerrors.NewNotFound("snippet", name.String())
}

func (snippetMoverImpl) moveToContainer(ctx *ExecContext, id, containerID model.ID) error {
	return ctx.Backend.MoveDocumentGen(id, containerID)
}

func (snippetMoverImpl) invalidate(ctx *ExecContext) { invalidatePagesGenCache(ctx) }

// ── nanoflowMoverImpl ────────────────────────────────────────────────────────

type nanoflowMoverImpl struct{ noopCrossModule }

func (nanoflowMoverImpl) label() string { return "nanoflow" }

func (nanoflowMoverImpl) find(ctx *ExecContext, name ast.QualifiedName) (model.ID, error) {
	if ctx.Nanoflows == nil {
		return "", mdlerrors.NewBackend("nanoflows repo unavailable", nil)
	}
	nfs, err := listNanoflowsWithContainerGen(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("list nanoflows", err)
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("build hierarchy", err)
	}
	for _, item := range nfs {
		modName := h.GetModuleName(h.FindModuleID(item.ContainerUUID))
		if modName == name.Module && item.NF.Name() == name.Name {
			return model.ID(item.NF.ID()), nil
		}
	}
	return "", mdlerrors.NewNotFound("nanoflow", name.String())
}

func (nanoflowMoverImpl) moveToContainer(ctx *ExecContext, id, containerID model.ID) error {
	return ctx.Nanoflows.Move(id, string(containerID))
}

func (nanoflowMoverImpl) invalidate(ctx *ExecContext) { invalidateMicroflowsCache(ctx) }

// ── enumerationMoverImpl ─────────────────────────────────────────────────────

type enumerationMoverImpl struct{}

func (enumerationMoverImpl) label() string { return "enumeration" }

func (enumerationMoverImpl) find(ctx *ExecContext, name ast.QualifiedName) (model.ID, error) {
	enum := findEnumeration(ctx, name.Module, name.Name)
	if enum == nil {
		return "", mdlerrors.NewNotFound("enumeration", name.String())
	}
	return enum.ID, nil
}

func (enumerationMoverImpl) moveToContainer(ctx *ExecContext, id, containerID model.ID) error {
	// findEnumeration only works by name; iterate all enumerations to find by ID.
	// This is acceptable: enum lists are small (typically < 100 per project).
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}
	allEnums := listAllEnumerations(ctx) // same helper used by findEnumeration
	for _, enum := range allEnums {
		if enum.ID == id {
			enum.ContainerID = containerID
			return ctx.Backend.MoveEnumeration(enum)
		}
	}
	// Fallback: also check by iterating modules (hierarchy-based).
	_ = h
	return mdlerrors.NewNotFound("enumeration", string(id))
}
// Note: `listAllEnumerations` may not exist by that name. Check `cmd_enumerations.go`
// for the actual helper that returns []*model.Enumeration or similar. The real call
// site in the original moveEnumeration uses:
//   enum := findEnumeration(ctx, name.Module, name.Name)
// Since we lose the name after find(), the alternative is to list all enums via
//   ctx.Backend.ListEnumerations() (if it exists) or reconstruct from hierarchy.
// Simplest real implementation: store module+name in the struct:
//   type enumerationMoverImpl struct{ name ast.QualifiedName }
// Then both find() and moveToContainer() use name-based lookup.
// The registry entry becomes: ast.DocumentTypeEnumeration: enumerationMoverImpl{}

func (enumerationMoverImpl) crossModuleHook(ctx *ExecContext, _ model.ID, name ast.QualifiedName, targetModule *model.Module) error {
	if targetModule.Name == name.Module {
		return nil
	}
	oldQN := name.String()
	newQN := targetModule.Name + "." + name.Name
	if err := ctx.Backend.UpdateEnumerationRefsInAllDomainModels(oldQN, newQN); err != nil {
		fmt.Fprintf(ctx.Output, "Warning: Could not update enumeration references: %v\n", err)
	} else {
		fmt.Fprintf(ctx.Output, "Updated enumeration references: %s -> %s\n", oldQN, newQN)
	}
	return nil
}

func (enumerationMoverImpl) invalidate(_ *ExecContext) {}

// ── constantMoverImpl ────────────────────────────────────────────────────────

type constantMoverImpl struct{ noopCrossModule }

func (constantMoverImpl) label() string { return "constant" }

func (constantMoverImpl) find(ctx *ExecContext, name ast.QualifiedName) (model.ID, error) {
	constants, err := ctx.Backend.ListConstants()
	if err != nil {
		return "", mdlerrors.NewBackend("list constants", err)
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("build hierarchy", err)
	}
	for _, c := range constants {
		modName := h.GetModuleName(h.FindModuleID(c.ContainerID))
		if modName == name.Module && c.Name == name.Name {
			return c.ID, nil
		}
	}
	return "", mdlerrors.NewNotFound("constant", name.String())
}

func (constantMoverImpl) moveToContainer(ctx *ExecContext, id, containerID model.ID) error {
	constants, err := ctx.Backend.ListConstants()
	if err != nil {
		return mdlerrors.NewBackend("list constants", err)
	}
	for _, c := range constants {
		if c.ID == id {
			c.ContainerID = containerID
			return ctx.Backend.MoveConstant(c)
		}
	}
	return mdlerrors.NewNotFound("constant", string(id))
}

func (constantMoverImpl) invalidate(_ *ExecContext) {}

// ── databaseConnectionMoverImpl ──────────────────────────────────────────────

type databaseConnectionMoverImpl struct{ noopCrossModule }

func (databaseConnectionMoverImpl) label() string { return "database connection" }

func (databaseConnectionMoverImpl) find(ctx *ExecContext, name ast.QualifiedName) (model.ID, error) {
	connections, err := ctx.Backend.ListDatabaseConnections()
	if err != nil {
		return "", mdlerrors.NewBackend("list database connections", err)
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("build hierarchy", err)
	}
	for _, conn := range connections {
		modName := h.GetModuleName(h.FindModuleID(conn.ContainerID))
		if modName == name.Module && conn.Name == name.Name {
			return conn.ID, nil
		}
	}
	return "", mdlerrors.NewNotFound("database connection", name.String())
}

func (databaseConnectionMoverImpl) moveToContainer(ctx *ExecContext, id, containerID model.ID) error {
	connections, err := ctx.Backend.ListDatabaseConnections()
	if err != nil {
		return mdlerrors.NewBackend("list database connections", err)
	}
	for _, conn := range connections {
		if conn.ID == id {
			conn.ContainerID = containerID
			return ctx.Backend.MoveDatabaseConnection(conn)
		}
	}
	return mdlerrors.NewNotFound("database connection", string(id))
}

func (databaseConnectionMoverImpl) invalidate(_ *ExecContext) {}

// ── javaActionMoverImpl ──────────────────────────────────────────────────────

type javaActionMoverImpl struct{ noopCrossModule }

func (javaActionMoverImpl) label() string { return "java action" }

func (javaActionMoverImpl) find(ctx *ExecContext, name ast.QualifiedName) (model.ID, error) {
	pairs, err := listJavaActionsWithContainerGen(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("list java actions", err)
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("build hierarchy", err)
	}
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modName := h.GetModuleName(h.FindModuleID(modelIDFromElementID(p.ContainerID)))
		if modName == name.Module && p.Elem.Name() == name.Name {
			return model.ID(p.Elem.ID()), nil
		}
	}
	return "", mdlerrors.NewNotFound("java action", name.String())
}

func (javaActionMoverImpl) moveToContainer(ctx *ExecContext, id, containerID model.ID) error {
	return ctx.Backend.MoveDocumentGen(id, containerID)
}

func (javaActionMoverImpl) invalidate(ctx *ExecContext) {
	if ctx.Cache != nil {
		ctx.Cache.javaActionsWithContainerGen = nil
	}
}

// ── javaScriptActionMoverImpl ────────────────────────────────────────────────

type javaScriptActionMoverImpl struct{ noopCrossModule }

func (javaScriptActionMoverImpl) label() string { return "javascript action" }

func (javaScriptActionMoverImpl) find(ctx *ExecContext, name ast.QualifiedName) (model.ID, error) {
	pairs, err := listJavaScriptActionsWithContainerGen(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("list javascript actions", err)
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("build hierarchy", err)
	}
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modName := h.GetModuleName(h.FindModuleID(modelIDFromElementID(p.ContainerID)))
		if modName == name.Module && p.Elem.Name() == name.Name {
			return model.ID(p.Elem.ID()), nil
		}
	}
	return "", mdlerrors.NewNotFound("javascript action", name.String())
}

func (javaScriptActionMoverImpl) moveToContainer(ctx *ExecContext, id, containerID model.ID) error {
	return ctx.Backend.MoveDocumentGen(id, containerID)
}

func (javaScriptActionMoverImpl) invalidate(ctx *ExecContext) {
	if ctx.Cache != nil {
		ctx.Cache.javaScriptActionsWithContainerGen = nil
	}
}

// ── layoutMoverImpl ──────────────────────────────────────────────────────────

type layoutMoverImpl struct{ noopCrossModule }

func (layoutMoverImpl) label() string { return "layout" }

func (layoutMoverImpl) find(ctx *ExecContext, name ast.QualifiedName) (model.ID, error) {
	pairs, err := listLayoutsWithContainerGen(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("list layouts", err)
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("build hierarchy", err)
	}
	for _, pair := range pairs {
		if pair.Elem == nil {
			continue
		}
		modName := h.GetModuleName(h.FindModuleID(model.ID(pair.ContainerID)))
		if modName == name.Module && pair.Elem.Name() == name.Name {
			return model.ID(pair.Elem.ID()), nil
		}
	}
	return "", mdlerrors.NewNotFound("layout", name.String())
}

func (layoutMoverImpl) moveToContainer(ctx *ExecContext, id, containerID model.ID) error {
	return ctx.Backend.MoveDocumentGen(id, containerID)
}

func (layoutMoverImpl) invalidate(ctx *ExecContext) { invalidatePagesGenCache(ctx) }

// ── workflowMoverImpl ────────────────────────────────────────────────────────

type workflowMoverImpl struct{ noopCrossModule }

func (workflowMoverImpl) label() string { return "workflow" }

func (workflowMoverImpl) find(ctx *ExecContext, name ast.QualifiedName) (model.ID, error) {
	pairs, err := listWorkflowsWithContainerGen(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("list workflows", err)
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("build hierarchy", err)
	}
	for _, pair := range pairs {
		if pair.Elem == nil {
			continue
		}
		modName := h.GetModuleName(h.FindModuleID(model.ID(pair.ContainerID)))
		if modName == name.Module && pair.Elem.Name() == name.Name {
			return model.ID(pair.Elem.ID()), nil
		}
	}
	return "", mdlerrors.NewNotFound("workflow", name.String())
}

func (workflowMoverImpl) moveToContainer(ctx *ExecContext, id, containerID model.ID) error {
	return ctx.Backend.MoveDocumentGen(id, containerID)
}

func (workflowMoverImpl) invalidate(ctx *ExecContext) {
	// Workflows cache field: check if ctx.Cache.workflowsWithContainerGen exists.
	// If it doesn't (workflows listing may not be cached), this is a no-op.
	// grep for workflowsWithContainerGen in exec_context.go to confirm.
	if ctx.Cache != nil {
		ctx.Cache.workflowsWithContainerGen = nil // set to nil if field exists; remove line if not
	}
}
```

**Note on enumerationMoverImpl**: It calls `findEnumerationByID(ctx, id)` — verify this helper exists by grepping for `findEnumeration` in the executor package. If only `findEnumeration(ctx, module, name)` exists (not by ID), use it differently: call `findEnumeration(ctx, name.Module, name.Name)` directly in `moveToContainer` by adding `name ast.QualifiedName` to the method, OR store the found enum object in a small wrapper struct. Check the actual helper signatures first and adapt.

**Note on workflowMoverImpl**: Verify `listWorkflowsWithContainerGen` returns `[]ContainerWithGen[*genWf.Workflow]` with `Elem.Name()` and `Elem.ID()` available. If the Workflow gen type uses different field accessors, adapt accordingly. Check `mdl/executor/helpers_workflows_v2.go`.

**Note on Cache fields**: Verify `ctx.Cache.workflowsWithContainerGen` exists by checking the `ExecutorCache` struct. If it doesn't, use a no-op invalidate (workflows listing isn't cached yet).

- [ ] **Step 5: Run coverage test to verify it passes**

```bash
go test ./mdl/executor/ -run TestDocumentMoverRegistry_Coverage -v 2>&1
```

Expected: PASS (all 11 types have movers).

- [ ] **Step 6: Commit**

```bash
git add mdl/ast/ast.go mdl/executor/cmd_move_registry.go mdl/executor/cmd_move_registry_test.go
git commit -m "feat(executor): add documentMover registry with 11 types (OCP refactor + JAVA/JS/LAYOUT/WORKFLOW)"
```

---

### Task 3: Simplify cmd_move.go using the registry

**Files:**
- Modify: `mdl/executor/cmd_move.go`

Replace the per-type switch + individual move functions with a single dispatch through `documentMoverRegistry`. Keep only `execMove`, `moveEntity`, and `updateQualifiedNameRefs`.

- [ ] **Step 1: Write a smoke test for execMove via mock**

In `mdl/executor/cmd_move_registry_test.go`, add:

```go
func TestExecMove_UnsupportedType_ReturnsError(t *testing.T) {
	// Verify the registry-based dispatch returns an error for unknown types.
	// Uses a write-connected mock context.
	ctx := &ExecContext{}
	// Mark as connected for write
	ctx.connectedForWrite = true  // check actual field name in ExecContext
	stmt := &ast.MoveStmt{
		DocumentType: ast.DocumentType("UNKNOWNTYPE"),
		Name:         ast.QualifiedName{Module: "Mod", Name: "Name"},
	}
	err := execMove(ctx, stmt)
	if err == nil {
		t.Fatal("expected error for unsupported document type")
	}
}
```

Note: `ctx.connectedForWrite` is an internal field — check the actual way to test this in the package. Look at existing move tests or mock setup to understand the pattern.

- [ ] **Step 2: Rewrite execMove**

Replace the entire body of `execMove` in `mdl/executor/cmd_move.go` with:

```go
func execMove(ctx *ExecContext, s *ast.MoveStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnected()
	}

	sourceModule, err := findModule(ctx, s.Name.Module)
	if err != nil {
		return mdlerrors.NewBackend("find source module", err)
	}

	var targetModule *model.Module
	isCrossModuleMove := false
	if s.TargetModule != "" {
		targetModule, err = findModule(ctx, s.TargetModule)
		if err != nil {
			return mdlerrors.NewBackend("find target module", err)
		}
		isCrossModuleMove = targetModule.ID != sourceModule.ID
	} else {
		targetModule = sourceModule
	}

	// Entity has a fundamentally different data structure (embedded in DomainModel).
	if s.DocumentType == ast.DocumentTypeEntity {
		return moveEntity(ctx, s.Name, sourceModule, targetModule)
	}

	m, ok := documentMoverRegistry[s.DocumentType]
	if !ok {
		return mdlerrors.NewUnsupported("unsupported document type: " + string(s.DocumentType))
	}

	var targetContainerID model.ID
	if s.Folder != "" {
		targetContainerID, err = resolveFolder(ctx, targetModule.ID, s.Folder)
		if err != nil {
			return mdlerrors.NewBackend("resolve target folder", err)
		}
	} else {
		targetContainerID = targetModule.ID
	}

	id, err := m.find(ctx, s.Name)
	if err != nil {
		return err
	}

	if err := m.moveToContainer(ctx, id, targetContainerID); err != nil {
		return err
	}

	if isCrossModuleMove {
		if err := m.crossModuleHook(ctx, id, s.Name, targetModule); err != nil {
			return err
		}
		if err := updateQualifiedNameRefs(ctx, s.Name, targetModule.Name); err != nil {
			return err
		}
	}

	m.invalidate(ctx)
	fmt.Fprintf(ctx.Output, "Moved %s %s to new location\n", m.label(), s.Name.String())
	return nil
}
```

- [ ] **Step 3: Delete the 8 per-type move functions**

Delete from `cmd_move.go`:
- `movePage`
- `moveMicroflow`
- `moveSnippet`
- `moveNanoflow`
- `moveEnumeration`
- `moveConstant`
- `moveDatabaseConnection`

Keep: `execMove`, `updateQualifiedNameRefs`, `moveEntity`.

Also remove unused imports (e.g., `genDm` is still needed for `moveEntity`). Run `goimports` or check imports manually.

- [ ] **Step 4: Fix imports**

```bash
go build ./mdl/executor/ 2>&1
```

Fix any import errors (unused or missing).

- [ ] **Step 5: Run full executor tests**

```bash
go test ./mdl/executor/ -timeout 120s -count=1 2>&1 | tail -5
```

Expected: `ok github.com/mendixlabs/mxcli/mdl/executor`

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/cmd_move.go mdl/executor/cmd_move_registry_test.go
git commit -m "refactor(executor): simplify execMove to registry dispatch (remove 7 per-type functions)"
```

---

### Task 4: Grammar — add new types to moveStatement

**Files:**
- Modify: `mdl/grammar/MDLParser.g4`

- [ ] **Step 1: Update moveStatement rule**

In `mdl/grammar/MDLParser.g4`, find the `moveStatement` rule (around line 360) and expand the token alternation to include the 4 new types:

```antlr
moveStatement
    : MOVE (PAGE | MICROFLOW | SNIPPET | NANOFLOW | ENUMERATION | CONSTANT | DATABASE CONNECTION | LAYOUT | WORKFLOW | JAVA ACTION | JAVASCRIPT ACTION) qualifiedName TO FOLDER STRING_LITERAL (IN (qualifiedName | IDENTIFIER))?
    | MOVE (PAGE | MICROFLOW | SNIPPET | NANOFLOW | ENUMERATION | CONSTANT | DATABASE CONNECTION | LAYOUT | WORKFLOW | JAVA ACTION | JAVASCRIPT ACTION) qualifiedName TO (qualifiedName | IDENTIFIER)
    | MOVE ENTITY qualifiedName TO (qualifiedName | IDENTIFIER)
    | MOVE FOLDER qualifiedName TO FOLDER STRING_LITERAL (IN (qualifiedName | IDENTIFIER))?
    | MOVE FOLDER qualifiedName TO (qualifiedName | IDENTIFIER)
    ;
```

Note: `LAYOUT` and `WORKFLOW` tokens should already exist in the lexer. Verify with:
```bash
grep -n "^LAYOUT:\|^WORKFLOW:" /mnt/data_sdd/gh/mxcli-wt-02/mdl/grammar/MDLLexer.g4
```

If they don't exist, add them.

- [ ] **Step 2: Regenerate the parser**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02 && make grammar 2>&1
```

Expected: no errors.

- [ ] **Step 3: Verify it compiles**

```bash
go build ./mdl/... 2>&1
```

Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add mdl/grammar/MDLParser.g4 mdl/grammar/parser/
git commit -m "feat(grammar): add JAVA ACTION, JAVASCRIPT ACTION, LAYOUT, WORKFLOW to moveStatement"
```

---

### Task 5: Visitor — parse new document types

**Files:**
- Modify: `mdl/visitor/visitor_entity.go`

- [ ] **Step 1: Write a failing parse test**

In `mdl/visitor/visitor_entity.go` test file (or a new test file), add:

```go
func TestParse_MoveJavaAction(t *testing.T) {
	stmts, errs := Build("MOVE JAVA ACTION MyModule.MyAction TO FOLDER 'actions' IN MyModule;")
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts))
	}
	s, ok := stmts[0].(*ast.MoveStmt)
	if !ok {
		t.Fatalf("expected *ast.MoveStmt, got %T", stmts[0])
	}
	if s.DocumentType != ast.DocumentTypeJavaAction {
		t.Errorf("DocumentType = %q, want %q", s.DocumentType, ast.DocumentTypeJavaAction)
	}
	if s.Name.Module != "MyModule" || s.Name.Name != "MyAction" {
		t.Errorf("Name = %+v", s.Name)
	}
	if s.Folder != "actions" {
		t.Errorf("Folder = %q", s.Folder)
	}
}

func TestParse_MoveJavaScriptAction(t *testing.T) {
	stmts, errs := Build("MOVE JAVASCRIPT ACTION MyModule.MyJSAction TO MyModule;")
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	s, ok := stmts[0].(*ast.MoveStmt)
	if !ok || s.DocumentType != ast.DocumentTypeJavaScriptAction {
		t.Fatalf("expected JAVASCRIPT ACTION move, got %T / %q", stmts[0], s.DocumentType)
	}
}

func TestParse_MoveLayout(t *testing.T) {
	stmts, errs := Build("MOVE LAYOUT MyModule.MyLayout TO FOLDER 'layouts' IN MyModule;")
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	s, ok := stmts[0].(*ast.MoveStmt)
	if !ok || s.DocumentType != ast.DocumentTypeLayout {
		t.Fatalf("expected LAYOUT move, got %T / %q", stmts[0], s.DocumentType)
	}
}

func TestParse_MoveWorkflow(t *testing.T) {
	stmts, errs := Build("MOVE WORKFLOW MyModule.MyWorkflow TO MyModule;")
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	s, ok := stmts[0].(*ast.MoveStmt)
	if !ok || s.DocumentType != ast.DocumentTypeWorkflow {
		t.Fatalf("expected WORKFLOW move, got %T / %q", stmts[0], s.DocumentType)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./mdl/visitor/ -run "TestParse_Move(JavaAction|JavaScriptAction|Layout|Workflow)" -v 2>&1
```

Expected: FAIL (grammar not yet regenerated OR visitor not yet updated).

- [ ] **Step 3: Add visitor cases**

In `mdl/visitor/visitor_entity.go`, in `ExitMoveStatement`, find the chain of `if ctx.PAGE() != nil` conditions and add 4 new else-if branches **before the final `else`** (or wherever the chain ends):

```go
} else if ctx.JAVA() != nil && ctx.ACTION() != nil {
    stmt.DocumentType = ast.DocumentTypeJavaAction
} else if ctx.JAVASCRIPT() != nil {
    stmt.DocumentType = ast.DocumentTypeJavaScriptAction
} else if ctx.LAYOUT() != nil {
    stmt.DocumentType = ast.DocumentTypeLayout
} else if ctx.WORKFLOW() != nil {
    stmt.DocumentType = ast.DocumentTypeWorkflow
}
```

Note: The generated ANTLR context methods depend on the grammar tokens. `JAVASCRIPT ACTION` in grammar generates `ctx.JAVASCRIPT()` (for the JAVASCRIPT token) + `ctx.ACTION()` is shared. Verify by checking the generated parser in `mdl/grammar/parser/` after Task 4.

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./mdl/visitor/ -run "TestParse_Move(JavaAction|JavaScriptAction|Layout|Workflow)" -v 2>&1
```

Expected: PASS.

- [ ] **Step 5: Run full visitor tests**

```bash
go test ./mdl/visitor/ -timeout 60s -count=1 2>&1 | tail -5
```

Expected: `ok github.com/mendixlabs/mxcli/mdl/visitor`

- [ ] **Step 6: Commit**

```bash
git add mdl/visitor/visitor_entity.go  # and test file
git commit -m "feat(visitor): parse JAVA ACTION, JAVASCRIPT ACTION, LAYOUT, WORKFLOW in MOVE statement"
```

---

### Task 6: Full test sweep

- [ ] **Step 1: Run all tests**

```bash
go test ./... -timeout 180s -count=1 2>&1 | grep -E "FAIL|ok" | tail -20
```

Expected: all packages `ok`.

- [ ] **Step 2: Verify grammar is round-trippable with mxcli check**

```bash
./bin/mxcli check - <<'EOF'
MOVE JAVA ACTION FeedbackModule.ACT_Test TO FOLDER 'actions' IN FeedbackModule;
MOVE JAVASCRIPT ACTION FeedbackModule.JS_isStrictMode TO FeedbackModule;
MOVE LAYOUT Atlas_Core.PopupLayout TO Atlas_Core;
MOVE WORKFLOW MyModule.Order_Workflow TO FOLDER 'workflows' IN MyModule;
EOF
```

Expected: no syntax errors (semantic errors are OK — we don't have a project loaded).

- [ ] **Step 3: Commit**

```bash
git commit --allow-empty -m "test: verify SOLID MOVE refactor + new types pass full suite"
```

Actually only commit if there are remaining file changes. Otherwise just record the test result.

---

## Self-Review Checklist

- [ ] `go test ./... -count=1` passes clean
- [ ] `make build` succeeds
- [ ] `execMove` contains no type-specific logic (only orchestration)
- [ ] `cmd_move.go` contains only 3 functions: `execMove`, `moveEntity`, `updateQualifiedNameRefs`
- [ ] `cmd_move_registry.go` contains 11 mover structs with consistent interface signatures
- [ ] `MovePageGen`, `MoveSnippetGen`, `MoveLayoutGen` are either deleted or clearly deprecated; `MoveDocumentGen` is the single call-site
- [ ] `MOVE JAVA ACTION`, `MOVE JAVASCRIPT ACTION`, `MOVE LAYOUT`, `MOVE WORKFLOW` parse without errors
- [ ] All 11 types covered in `TestDocumentMoverRegistry_Coverage`
- [ ] New grammar additions committed alongside regenerated parser files

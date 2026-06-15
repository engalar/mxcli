// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// documentMover encapsulates the per-document-type behaviour of the MOVE
// command. Each registered type knows how to locate its unit, relocate it to a
// new container, optionally remap cross-module references, and invalidate its
// caches. execMove drives every type through this single interface (OCP: adding
// a new movable document type is a registry entry, not a new switch case).
type documentMover interface {
	// find locates the unit by qualified name and returns its ID.
	find(ctx *ExecContext, name ast.QualifiedName) (model.ID, error)
	// moveToContainer relocates the unit (identified by id) into targetContainerID.
	// name is passed for movers (e.g. enumeration) that re-resolve by name.
	moveToContainer(ctx *ExecContext, id model.ID, name ast.QualifiedName, targetContainerID model.ID) error
	// crossModuleHook performs cross-module reference remapping. Called only for
	// cross-module moves, after moveToContainer succeeds.
	crossModuleHook(ctx *ExecContext, id model.ID, name ast.QualifiedName, targetModule *model.Module) error
	// invalidate clears any caches affected by the move.
	invalidate(ctx *ExecContext)
	// label is the human-readable document type, used in the success message.
	label() string
}

// noopCrossModule is embedded by movers that have no cross-module references to
// remap (the container change alone is sufficient).
type noopCrossModule struct{}

func (noopCrossModule) crossModuleHook(_ *ExecContext, _ model.ID, _ ast.QualifiedName, _ *model.Module) error {
	return nil
}

// documentMoverRegistry maps each movable document type to its mover.
// Entity moves are handled separately in execMove (entities are embedded in
// domain models, not top-level units).
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

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

type pageMoverImpl struct{}

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

func (pageMoverImpl) moveToContainer(ctx *ExecContext, id model.ID, _ ast.QualifiedName, targetContainerID model.ID) error {
	if err := ctx.PageWriter.MoveDocumentGen(id, targetContainerID); err != nil {
		return mdlerrors.NewBackend("move page", err)
	}
	return nil
}

func (pageMoverImpl) crossModuleHook(ctx *ExecContext, id model.ID, _ ast.QualifiedName, targetModule *model.Module) error {
	pairs, err := listPagesWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}
	for _, pair := range pairs {
		if pair.Elem == nil || model.ID(pair.Elem.ID()) != id {
			continue
		}
		currentRoles := pair.Elem.AllowedRolesQualifiedNames()
		currentRoleIDs := make([]model.ID, len(currentRoles))
		for i, qn := range currentRoles {
			currentRoleIDs[i] = model.ID(qn)
		}
		remappedIDs := remapDocumentAccessRoles(ctx, targetModule, currentRoleIDs)
		if err := ctx.SecurityEntityAccessManager.UpdateAllowedRoles(id, documentRoleStrings(remappedIDs)); err != nil {
			return mdlerrors.NewBackend("remap page access", err)
		}
		return nil
	}
	return nil
}

func (pageMoverImpl) invalidate(ctx *ExecContext) { invalidatePagesGenCache(ctx) }
func (pageMoverImpl) label() string               { return "page" }

// ---------------------------------------------------------------------------
// Microflow
// ---------------------------------------------------------------------------

type microflowMoverImpl struct{}

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

func (microflowMoverImpl) moveToContainer(ctx *ExecContext, id model.ID, _ ast.QualifiedName, targetContainerID model.ID) error {
	if err := ctx.Microflows.Move(id, string(targetContainerID)); err != nil {
		return mdlerrors.NewBackend("move microflow", err)
	}
	return nil
}

func (microflowMoverImpl) crossModuleHook(ctx *ExecContext, id model.ID, _ ast.QualifiedName, targetModule *model.Module) error {
	mfs, err := listMicroflowsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list microflows", err)
	}
	for _, item := range mfs {
		mf := item.MF
		if model.ID(mf.ID()) != id {
			continue
		}
		existing := mf.AllowedModuleRolesQualifiedNames()
		currentRoleIDs := make([]model.ID, len(existing))
		for i, qn := range existing {
			currentRoleIDs[i] = model.ID(qn)
		}
		remappedIDs := remapDocumentAccessRoles(ctx, targetModule, currentRoleIDs)
		mf.SetAllowedModuleRolesQualifiedNames(documentRoleStrings(remappedIDs))
		if err := ctx.Microflows.Update(mf); err != nil {
			return mdlerrors.NewBackend("remap microflow access", err)
		}
		return nil
	}
	return nil
}

func (microflowMoverImpl) invalidate(ctx *ExecContext) { invalidateMicroflowsCache(ctx) }
func (microflowMoverImpl) label() string               { return "microflow" }

// ---------------------------------------------------------------------------
// Snippet
// ---------------------------------------------------------------------------

type snippetMoverImpl struct {
	noopCrossModule
}

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

func (snippetMoverImpl) moveToContainer(ctx *ExecContext, id model.ID, _ ast.QualifiedName, targetContainerID model.ID) error {
	if err := ctx.PageWriter.MoveDocumentGen(id, targetContainerID); err != nil {
		return mdlerrors.NewBackend("move snippet", err)
	}
	return nil
}

func (snippetMoverImpl) invalidate(ctx *ExecContext) { invalidatePagesGenCache(ctx) }
func (snippetMoverImpl) label() string               { return "snippet" }

// ---------------------------------------------------------------------------
// Nanoflow
// ---------------------------------------------------------------------------

type nanoflowMoverImpl struct {
	noopCrossModule
}

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

func (nanoflowMoverImpl) moveToContainer(ctx *ExecContext, id model.ID, _ ast.QualifiedName, targetContainerID model.ID) error {
	if err := ctx.Nanoflows.Move(id, string(targetContainerID)); err != nil {
		return mdlerrors.NewBackend("move nanoflow", err)
	}
	return nil
}

func (nanoflowMoverImpl) invalidate(ctx *ExecContext) { invalidateMicroflowsCache(ctx) }
func (nanoflowMoverImpl) label() string               { return "nanoflow" }

// ---------------------------------------------------------------------------
// Enumeration
// ---------------------------------------------------------------------------

type enumerationMoverImpl struct{}

func (enumerationMoverImpl) find(ctx *ExecContext, name ast.QualifiedName) (model.ID, error) {
	enum := findEnumeration(ctx, name.Module, name.Name)
	if enum == nil {
		return "", mdlerrors.NewNotFound("enumeration", name.String())
	}
	return enum.ID, nil
}

func (enumerationMoverImpl) moveToContainer(ctx *ExecContext, _ model.ID, name ast.QualifiedName, targetContainerID model.ID) error {
	enum := findEnumeration(ctx, name.Module, name.Name)
	if enum == nil {
		return mdlerrors.NewNotFound("enumeration", name.String())
	}
	enum.ContainerID = targetContainerID
	if err := ctx.EnumerationWriter.MoveEnumeration(enum); err != nil {
		return mdlerrors.NewBackend("move enumeration", err)
	}
	return nil
}

func (enumerationMoverImpl) crossModuleHook(ctx *ExecContext, _ model.ID, name ast.QualifiedName, targetModule *model.Module) error {
	oldQualifiedName := name.String()
	newQualifiedName := targetModule.Name + "." + name.Name
	if err := ctx.DomainModelWriter.UpdateEnumerationRefsInAllDomainModels(oldQualifiedName, newQualifiedName); err != nil {
		return mdlerrors.NewBackend("update enumeration references", err)
	}
	return nil
}

func (enumerationMoverImpl) invalidate(_ *ExecContext) {}
func (enumerationMoverImpl) label() string             { return "enumeration" }

// ---------------------------------------------------------------------------
// Constant
// ---------------------------------------------------------------------------

type constantMoverImpl struct {
	noopCrossModule
}

func (constantMoverImpl) find(ctx *ExecContext, name ast.QualifiedName) (model.ID, error) {
	constants, err := ctx.ConstantReader.ListConstants()
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

func (constantMoverImpl) moveToContainer(ctx *ExecContext, id model.ID, name ast.QualifiedName, targetContainerID model.ID) error {
	constants, err := ctx.ConstantReader.ListConstants()
	if err != nil {
		return mdlerrors.NewBackend("list constants", err)
	}
	for _, c := range constants {
		if c.ID != id {
			continue
		}
		c.ContainerID = targetContainerID
		if err := ctx.ConstantWriter.MoveConstant(c); err != nil {
			return mdlerrors.NewBackend("move constant", err)
		}
		return nil
	}
	return mdlerrors.NewNotFound("constant", name.String())
}

func (constantMoverImpl) invalidate(_ *ExecContext) {}
func (constantMoverImpl) label() string             { return "constant" }

// ---------------------------------------------------------------------------
// Database connection
// ---------------------------------------------------------------------------

type databaseConnectionMoverImpl struct {
	noopCrossModule
}

func (databaseConnectionMoverImpl) find(ctx *ExecContext, name ast.QualifiedName) (model.ID, error) {
	connections, err := ctx.ServiceLister.ListDatabaseConnections()
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

func (databaseConnectionMoverImpl) moveToContainer(ctx *ExecContext, id model.ID, name ast.QualifiedName, targetContainerID model.ID) error {
	connections, err := ctx.ServiceLister.ListDatabaseConnections()
	if err != nil {
		return mdlerrors.NewBackend("list database connections", err)
	}
	for _, conn := range connections {
		if conn.ID != id {
			continue
		}
		conn.ContainerID = targetContainerID
		if err := ctx.ServiceWriter.MoveDatabaseConnection(conn); err != nil {
			return mdlerrors.NewBackend("move database connection", err)
		}
		return nil
	}
	return mdlerrors.NewNotFound("database connection", name.String())
}

func (databaseConnectionMoverImpl) invalidate(_ *ExecContext) {}
func (databaseConnectionMoverImpl) label() string             { return "database connection" }

// ---------------------------------------------------------------------------
// Java action
// ---------------------------------------------------------------------------

type javaActionMoverImpl struct {
	noopCrossModule
}

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

func (javaActionMoverImpl) moveToContainer(ctx *ExecContext, id model.ID, _ ast.QualifiedName, targetContainerID model.ID) error {
	if err := ctx.PageWriter.MoveDocumentGen(id, targetContainerID); err != nil {
		return mdlerrors.NewBackend("move java action", err)
	}
	return nil
}

func (javaActionMoverImpl) invalidate(ctx *ExecContext) {
	if ctx.Cache != nil {
		ctx.Cache.javaActionsWithContainerGen = nil
	}
}
func (javaActionMoverImpl) label() string { return "java action" }

// ---------------------------------------------------------------------------
// JavaScript action
// ---------------------------------------------------------------------------

type javaScriptActionMoverImpl struct {
	noopCrossModule
}

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

func (javaScriptActionMoverImpl) moveToContainer(ctx *ExecContext, id model.ID, _ ast.QualifiedName, targetContainerID model.ID) error {
	if err := ctx.PageWriter.MoveDocumentGen(id, targetContainerID); err != nil {
		return mdlerrors.NewBackend("move javascript action", err)
	}
	return nil
}

func (javaScriptActionMoverImpl) invalidate(ctx *ExecContext) {
	if ctx.Cache != nil {
		ctx.Cache.javaScriptActionsWithContainerGen = nil
	}
}
func (javaScriptActionMoverImpl) label() string { return "javascript action" }

// ---------------------------------------------------------------------------
// Layout
// ---------------------------------------------------------------------------

type layoutMoverImpl struct {
	noopCrossModule
}

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

func (layoutMoverImpl) moveToContainer(ctx *ExecContext, id model.ID, _ ast.QualifiedName, targetContainerID model.ID) error {
	if err := ctx.PageWriter.MoveDocumentGen(id, targetContainerID); err != nil {
		return mdlerrors.NewBackend("move layout", err)
	}
	return nil
}

func (layoutMoverImpl) invalidate(ctx *ExecContext) { invalidatePagesGenCache(ctx) }
func (layoutMoverImpl) label() string               { return "layout" }

// ---------------------------------------------------------------------------
// Workflow
// ---------------------------------------------------------------------------

type workflowMoverImpl struct {
	noopCrossModule
}

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

func (workflowMoverImpl) moveToContainer(ctx *ExecContext, id model.ID, _ ast.QualifiedName, targetContainerID model.ID) error {
	if err := ctx.PageWriter.MoveDocumentGen(id, targetContainerID); err != nil {
		return mdlerrors.NewBackend("move workflow", err)
	}
	return nil
}

func (workflowMoverImpl) invalidate(ctx *ExecContext) {
	if ctx.Cache != nil {
		ctx.Cache.workflowsWithContainerGen = nil
	}
}
func (workflowMoverImpl) label() string { return "workflow" }

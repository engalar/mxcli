// SPDX-License-Identifier: Apache-2.0

// Package executor - MOVE command
package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
)

// execMove handles MOVE PAGE/MICROFLOW/SNIPPET/NANOFLOW/ENTITY/ENUMERATION statements.
func execMove(ctx *ExecContext, s *ast.MoveStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnected()
	}

	// Find the source module
	sourceModule, err := findModule(ctx, s.Name.Module)
	if err != nil {
		return mdlerrors.NewBackend("find source module", err)
	}

	// Determine target module
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

	// Entity moves are handled specially (entities are embedded in domain models, not top-level units)
	if s.DocumentType == ast.DocumentTypeEntity {
		return moveEntity(ctx, s.Name, sourceModule, targetModule)
	}

	// Resolve target container (folder or module root)
	var targetContainerID model.ID
	if s.Folder != "" {
		targetContainerID, err = resolveFolder(ctx, targetModule.ID, s.Folder)
		if err != nil {
			return mdlerrors.NewBackend("resolve target folder", err)
		}
	} else {
		targetContainerID = targetModule.ID
	}

	// Execute move based on document type
	switch s.DocumentType {
	case ast.DocumentTypePage:
		if err := movePage(ctx, s.Name, targetContainerID, targetModule, isCrossModuleMove); err != nil {
			return err
		}
	case ast.DocumentTypeMicroflow:
		if err := moveMicroflow(ctx, s.Name, targetContainerID, targetModule, isCrossModuleMove); err != nil {
			return err
		}
	case ast.DocumentTypeSnippet:
		if err := moveSnippet(ctx, s.Name, targetContainerID); err != nil {
			return err
		}
	case ast.DocumentTypeNanoflow:
		if err := moveNanoflow(ctx, s.Name, targetContainerID); err != nil {
			return err
		}
	case ast.DocumentTypeEnumeration:
		return moveEnumeration(ctx, s.Name, targetContainerID, targetModule.Name)
	case ast.DocumentTypeConstant:
		if err := moveConstant(ctx, s.Name, targetContainerID); err != nil {
			return err
		}
	case ast.DocumentTypeDatabaseConnection:
		if err := moveDatabaseConnection(ctx, s.Name, targetContainerID); err != nil {
			return err
		}
	default:
		return mdlerrors.NewUnsupported("unsupported document type: " + string(s.DocumentType))
	}

	// For cross-module moves, update all BY_NAME references throughout the project
	if isCrossModuleMove {
		if err := updateQualifiedNameRefs(ctx, s.Name, targetModule.Name); err != nil {
			return err
		}
	}

	return nil
}

// updateQualifiedNameRefs updates all BY_NAME references to an element after a cross-module move.
func updateQualifiedNameRefs(ctx *ExecContext, name ast.QualifiedName, newModule string) error {
	oldQN := name.String()               // "OldModule.ElementName"
	newQN := newModule + "." + name.Name // "NewModule.ElementName"
	updated, err := ctx.Backend.UpdateQualifiedNameInAllUnits(oldQN, newQN)
	if err != nil {
		return mdlerrors.NewBackend("update references", err)
	}
	if updated > 0 {
		fmt.Fprintf(ctx.Output, "Updated references in %d document(s): %s → %s\n", updated, oldQN, newQN)
	}
	return nil
}

// movePage moves a page to a new container.
func movePage(ctx *ExecContext, name ast.QualifiedName, targetContainerID model.ID, targetModule *model.Module, isCrossModuleMove bool) error {
	// Find the page
	pages, err := ctx.Backend.ListPages()
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	for _, p := range pages {
		modID := h.FindModuleID(p.ContainerID)
		modName := h.GetModuleName(modID)
		if modName == name.Module && p.Name == name.Name {
			// Update container ID and move the unit
			p.ContainerID = targetContainerID
			if err := ctx.Backend.MovePage(p); err != nil {
				return mdlerrors.NewBackend("move page", err)
			}
			if isCrossModuleMove {
				p.AllowedRoles = remapDocumentAccessRoles(ctx, targetModule, p.AllowedRoles)
				if err := ctx.Backend.UpdateAllowedRoles(p.ID, documentRoleStrings(p.AllowedRoles)); err != nil {
					return mdlerrors.NewBackend("remap page access", err)
				}
			}
			fmt.Fprintf(ctx.Output, "Moved page %s to new location\n", name.String())
			return nil
		}
	}

	return mdlerrors.NewNotFound("page", name.String())
}

// moveMicroflow moves a microflow to a new container.
//
// Uses the gen-typed flow listing (Followup E4): container linkage
// lives in the SQL Unit row, not the gen Microflow object, so the move
// flows through ctx.Microflows.Move(id, parentUUID) — there is no
// "set ContainerID then call MoveMicroflow" path on the gen surface.
// Cross-module remap also goes through ctx.Microflows.Update because
// the legacy Backend.UpdateAllowedRoles took an mfID + []string and
// mutated through the sdk type.
func moveMicroflow(ctx *ExecContext, name ast.QualifiedName, targetContainerID model.ID, targetModule *model.Module, isCrossModuleMove bool) error {
	if ctx.Microflows == nil {
		return mdlerrors.NewBackend("microflows repo unavailable", nil)
	}

	mfs, err := listMicroflowsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list microflows", err)
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	for _, item := range mfs {
		mf := item.MF
		modID := h.FindModuleID(item.ContainerUUID)
		modName := h.GetModuleName(modID)
		if modName != name.Module || mf.Name() != name.Name {
			continue
		}
		mfID := model.ID(mf.ID())
		if err := ctx.Microflows.Move(mfID, string(targetContainerID)); err != nil {
			return mdlerrors.NewBackend("move microflow", err)
		}
		if isCrossModuleMove {
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
		}
		invalidateMicroflowsCache(ctx)
		fmt.Fprintf(ctx.Output, "Moved microflow %s to new location\n", name.String())
		return nil
	}

	return mdlerrors.NewNotFound("microflow", name.String())
}

// moveSnippet moves a snippet to a new container.
func moveSnippet(ctx *ExecContext, name ast.QualifiedName, targetContainerID model.ID) error {
	// Find the snippet
	snippets, err := ctx.Backend.ListSnippets()
	if err != nil {
		return mdlerrors.NewBackend("list snippets", err)
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	for _, s := range snippets {
		modID := h.FindModuleID(s.ContainerID)
		modName := h.GetModuleName(modID)
		if modName == name.Module && s.Name == name.Name {
			// Update container ID and move the unit
			s.ContainerID = targetContainerID
			if err := ctx.Backend.MoveSnippet(s); err != nil {
				return mdlerrors.NewBackend("move snippet", err)
			}
			fmt.Fprintf(ctx.Output, "Moved snippet %s to new location\n", name.String())
			return nil
		}
	}

	return mdlerrors.NewNotFound("snippet", name.String())
}

// moveNanoflow moves a nanoflow to a new container.
//
// See moveMicroflow for the gen-flow Move rationale (Followup E4): the
// container update is a SQL Unit row mutation, not a BSON field
// rewrite, so we go through ctx.Nanoflows.Move(id, parentUUID).
func moveNanoflow(ctx *ExecContext, name ast.QualifiedName, targetContainerID model.ID) error {
	if ctx.Nanoflows == nil {
		return mdlerrors.NewBackend("nanoflows repo unavailable", nil)
	}

	nfs, err := listNanoflowsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list nanoflows", err)
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	for _, item := range nfs {
		nf := item.NF
		modID := h.FindModuleID(item.ContainerUUID)
		modName := h.GetModuleName(modID)
		if modName != name.Module || nf.Name() != name.Name {
			continue
		}
		if err := ctx.Nanoflows.Move(model.ID(nf.ID()), string(targetContainerID)); err != nil {
			return mdlerrors.NewBackend("move nanoflow", err)
		}
		invalidateMicroflowsCache(ctx)
		fmt.Fprintf(ctx.Output, "Moved nanoflow %s to new location\n", name.String())
		return nil
	}

	return mdlerrors.NewNotFound("nanoflow", name.String())
}

// moveEntity moves an entity from one domain model to another.
// Entities are embedded inside DomainModel documents, so we must remove from source DM and add to target DM.
// Associations referencing the entity are converted to CrossAssociations.
// ViewEntitySourceDocuments for view entities are also moved.
func moveEntity(ctx *ExecContext, name ast.QualifiedName, sourceModule, targetModule *model.Module) error {
	// Get source domain model
	sourceDM, err := ctx.Backend.GetDomainModel(sourceModule.ID)
	if err != nil {
		return mdlerrors.NewBackend("get source domain model", err)
	}

	// Find the entity in the source domain model
	var entity *domainmodel.Entity
	for _, ent := range sourceDM.Entities {
		if ent.Name == name.Name {
			entity = ent
			break
		}
	}
	if entity == nil {
		return mdlerrors.NewNotFound("entity", name.String())
	}

	// Get target domain model
	targetDM, err := ctx.Backend.GetDomainModel(targetModule.ID)
	if err != nil {
		return mdlerrors.NewBackend("get target domain model", err)
	}

	// Move entity via writer (converts associations to CrossAssociations, updates validation rule refs)
	convertedAssocs, err := ctx.Backend.MoveEntity(entity, sourceDM.ID, targetDM.ID, sourceModule.Name, targetModule.Name)
	if err != nil {
		return mdlerrors.NewBackend("move entity", err)
	}

	// Move ViewEntitySourceDocument for view entities
	if entity.Source == "DomainModels$OqlViewEntitySource" && entity.SourceDocumentRef != "" {
		// The SourceDocumentRef was already updated by MoveEntity to use the new module name.
		// Extract the original doc name (before the module prefix was changed).
		docName := name.Name // ViewEntitySourceDocument name matches the entity name
		if err := ctx.Backend.MoveViewEntitySourceDocument(sourceModule.Name, targetModule.ID, docName); err != nil {
			fmt.Fprintf(ctx.Output, "Warning: Could not move ViewEntitySourceDocument: %v\n", err)
		}
	}

	// Update OQL queries in all ViewEntitySourceDocuments that reference the moved entity
	oldQualifiedName := name.String()                       // e.g., "DmTest.Customer"
	newQualifiedName := targetModule.Name + "." + name.Name // e.g., "DmTest2.Customer"
	if oqlUpdated, err := ctx.Backend.UpdateOqlQueriesForMovedEntity(oldQualifiedName, newQualifiedName); err != nil {
		fmt.Fprintf(ctx.Output, "Warning: Could not update OQL queries: %v\n", err)
	} else if oqlUpdated > 0 {
		fmt.Fprintf(ctx.Output, "Updated %d OQL query(ies) referencing %s\n", oqlUpdated, oldQualifiedName)
	}

	fmt.Fprintf(ctx.Output, "Moved entity %s to %s\n", name.String(), targetModule.Name)
	if len(convertedAssocs) > 0 {
		fmt.Fprintf(ctx.Output, "Converted %d association(s) to cross-module associations:\n", len(convertedAssocs))
		for _, assocName := range convertedAssocs {
			fmt.Fprintf(ctx.Output, "  - %s\n", assocName)
		}
	}
	return nil
}

// moveEnumeration moves an enumeration to a new container.
// For cross-module moves, updates all EnumerationAttributeType references across all domain models.
func moveEnumeration(ctx *ExecContext, name ast.QualifiedName, targetContainerID model.ID, targetModuleName string) error {
	enum := findEnumeration(ctx, name.Module, name.Name)
	if enum == nil {
		return mdlerrors.NewNotFound("enumeration", name.String())
	}

	oldQualifiedName := name.String() // e.g., "DmTest.Country"
	enum.ContainerID = targetContainerID
	if err := ctx.Backend.MoveEnumeration(enum); err != nil {
		return mdlerrors.NewBackend("move enumeration", err)
	}

	// For cross-module moves, update enumeration references in all domain models
	if targetModuleName != "" && targetModuleName != name.Module {
		newQualifiedName := targetModuleName + "." + name.Name
		if err := ctx.Backend.UpdateEnumerationRefsInAllDomainModels(oldQualifiedName, newQualifiedName); err != nil {
			fmt.Fprintf(ctx.Output, "Warning: Could not update enumeration references: %v\n", err)
		} else {
			fmt.Fprintf(ctx.Output, "Updated enumeration references: %s -> %s\n", oldQualifiedName, newQualifiedName)
		}
	}

	fmt.Fprintf(ctx.Output, "Moved enumeration %s to new location\n", name.String())
	return nil
}

// moveConstant moves a constant to a new container.
func moveConstant(ctx *ExecContext, name ast.QualifiedName, targetContainerID model.ID) error {
	constants, err := ctx.Backend.ListConstants()
	if err != nil {
		return mdlerrors.NewBackend("list constants", err)
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	for _, c := range constants {
		modID := h.FindModuleID(c.ContainerID)
		modName := h.GetModuleName(modID)
		if modName == name.Module && c.Name == name.Name {
			c.ContainerID = targetContainerID
			if err := ctx.Backend.MoveConstant(c); err != nil {
				return mdlerrors.NewBackend("move constant", err)
			}
			fmt.Fprintf(ctx.Output, "Moved constant %s to new location\n", name.String())
			return nil
		}
	}

	return mdlerrors.NewNotFound("constant", name.String())
}

// moveDatabaseConnection moves a database connection to a new container.
func moveDatabaseConnection(ctx *ExecContext, name ast.QualifiedName, targetContainerID model.ID) error {
	connections, err := ctx.Backend.ListDatabaseConnections()
	if err != nil {
		return mdlerrors.NewBackend("list database connections", err)
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	for _, conn := range connections {
		modID := h.FindModuleID(conn.ContainerID)
		modName := h.GetModuleName(modID)
		if modName == name.Module && conn.Name == name.Name {
			conn.ContainerID = targetContainerID
			if err := ctx.Backend.MoveDatabaseConnection(conn); err != nil {
				return mdlerrors.NewBackend("move database connection", err)
			}
			fmt.Fprintf(ctx.Output, "Moved database connection %s to new location\n", name.String())
			return nil
		}
	}

	return mdlerrors.NewNotFound("database connection", name.String())
}

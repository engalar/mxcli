// SPDX-License-Identifier: Apache-2.0

// Package executor - MOVE command
package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
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

	mover, ok := documentMoverRegistry[s.DocumentType]
	if !ok {
		return mdlerrors.NewUnsupported("unsupported document type: " + string(s.DocumentType))
	}

	// Resolve target container (folder or module root)
	var targetContainerID model.ID
	if s.Folder != "" {
		targetContainerID, err = resolveFolder(ctx, targetModule.ID, s.Folder, nil)
		if err != nil {
			return mdlerrors.NewBackend("resolve target folder", err)
		}
	} else {
		targetContainerID = targetModule.ID
	}

	id, err := mover.find(ctx, s.Name)
	if err != nil {
		return err
	}
	if err := mover.moveToContainer(ctx, id, s.Name, targetContainerID); err != nil {
		return err
	}
	if isCrossModuleMove {
		if err := mover.crossModuleHook(ctx, id, s.Name, targetModule); err != nil {
			return err
		}
		if err := updateQualifiedNameRefs(ctx, s.Name, targetModule.Name); err != nil {
			return err
		}
	}
	mover.invalidate(ctx)
	fmt.Fprintf(ctx.Output, "Moved %s %s to new location\n", mover.label(), s.Name.String())
	return nil
}

// updateQualifiedNameRefs updates all BY_NAME references to an element after a cross-module move.
func updateQualifiedNameRefs(ctx *ExecContext, name ast.QualifiedName, newModule string) error {
	oldQN := name.String()               // "OldModule.ElementName"
	newQN := newModule + "." + name.Name // "NewModule.ElementName"
	updated, err := ctx.RenameManager.UpdateQualifiedNameInAllUnits(oldQN, newQN)
	if err != nil {
		return mdlerrors.NewBackend("update references", err)
	}
	if updated > 0 {
		fmt.Fprintf(ctx.Output, "Updated references in %d document(s): %s → %s\n", updated, oldQN, newQN)
	}
	return nil
}
// moveEntity moves an entity from one domain model to another.
// Entities are embedded inside DomainModel documents, so we must remove from source DM and add to target DM.
// Associations referencing the entity are converted to CrossAssociations.
// ViewEntitySourceDocuments for view entities are also moved.
func moveEntity(ctx *ExecContext, name ast.QualifiedName, sourceModule, targetModule *model.Module) error {
	// Get source domain model
	sourceDM, err := getDomainModelGenCached(ctx, sourceModule.ID)
	if err != nil {
		return mdlerrors.NewBackend("get source domain model", err)
	}
	if sourceDM == nil {
		return mdlerrors.NewBackend("get source domain model", nil)
	}

	// Find the entity in the source domain model
	var entity *genDm.Entity
	for _, entElem := range sourceDM.EntitiesItems() {
		ent, ok := entElem.(*genDm.Entity)
		if ok && ent.Name() == name.Name {
			entity = ent
			break
		}
	}
	if entity == nil {
		return mdlerrors.NewNotFound("entity", name.String())
	}

	// Get target domain model
	targetDM, err := getDomainModelGenCached(ctx, targetModule.ID)
	if err != nil {
		return mdlerrors.NewBackend("get target domain model", err)
	}
	if targetDM == nil {
		return mdlerrors.NewBackend("get target domain model", nil)
	}

	// Move entity via writer (converts associations to CrossAssociations, updates validation rule refs)
	convertedAssocs, err := ctx.DomainModelWriter.MoveEntityGen(entity, model.ID(sourceDM.ID()), model.ID(targetDM.ID()), sourceModule.Name, targetModule.Name)
	if err != nil {
		return mdlerrors.NewBackend("move entity", err)
	}
	invalidateDomainModelGenForModule(ctx, sourceModule.ID)
	invalidateDomainModelGenForModule(ctx, targetModule.ID)

	// Move ViewEntitySourceDocument for view entities
	if source, ok := entity.Source().(*genDm.OqlViewEntitySource); ok && source != nil && source.SourceDocumentQualifiedName() != "" {
		// The SourceDocumentRef was already updated by MoveEntity to use the new module name.
		// Extract the original doc name (before the module prefix was changed).
		docName := name.Name // ViewEntitySourceDocument name matches the entity name
		if err := ctx.DomainModelWriter.MoveViewEntitySourceDocument(sourceModule.Name, targetModule.ID, docName); err != nil {
			fmt.Fprintf(ctx.Output, "Warning: Could not move ViewEntitySourceDocument: %v\n", err)
		}
	}

	// Update OQL queries in all ViewEntitySourceDocuments that reference the moved entity
	oldQualifiedName := name.String()                       // e.g., "DmTest.Customer"
	newQualifiedName := targetModule.Name + "." + name.Name // e.g., "DmTest2.Customer"
	if oqlUpdated, err := ctx.DomainModelWriter.UpdateOqlQueriesForMovedEntity(oldQualifiedName, newQualifiedName); err != nil {
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


// SPDX-License-Identifier: Apache-2.0

// Package executor - MOVE command
package executor

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// execMove handles MOVE PAGE/MICROFLOW/SNIPPET/NANOFLOW/ENTITY/ENUMERATION statements.

// execMoveFn is the HandlerDeps version of execMove.
func execMoveFn(ctx context.Context, s *ast.MoveStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	sourceModule, err := findModuleFn(deps.ModuleLister, s.Name.Module)
	if err != nil {
		return mdlerrors.NewBackend("find source module", err)
	}

	var targetModule *model.Module
	isCrossModuleMove := false
	if s.TargetModule != "" {
		targetModule, err = findModuleFn(deps.ModuleLister, s.TargetModule)
		if err != nil {
			return mdlerrors.NewBackend("find target module", err)
		}
		isCrossModuleMove = targetModule.ID != sourceModule.ID
	} else {
		targetModule = sourceModule
	}

	if s.DocumentType == ast.DocumentTypeEntity {
		return moveEntityFn(ctx, deps, s.Name, sourceModule, targetModule)
	}

	mover, ok := documentMoverRegistry[s.DocumentType]
	if !ok {
		return mdlerrors.NewUnsupported("unsupported document type: " + string(s.DocumentType))
	}

	var targetContainerID model.ID
	if s.Folder != "" {
		ectx := NewExecContext(ctx, deps)
		targetContainerID, err = resolveFolder(ectx, targetModule.ID, s.Folder, nil)
		if err != nil {
			return mdlerrors.NewBackend("resolve target folder", err)
		}
	} else {
		targetContainerID = targetModule.ID
	}

	ectx := NewExecContext(ctx, deps)
	id, err := mover.find(ectx, s.Name)
	if err != nil {
		return err
	}
	// Note: mover methods still take *ExecContext; pass a temp context
	if err := mover.moveToContainer(ectx, id, s.Name, targetContainerID); err != nil {
		return err
	}
	if isCrossModuleMove {
		if err := mover.crossModuleHook(ectx, id, s.Name, targetModule); err != nil {
			return err
		}
		if err := updateQualifiedNameRefsFn(ctx, deps, s.Name, targetModule.Name); err != nil {
			return err
		}
	}
	mover.invalidate(ectx)
	fmt.Fprintf(deps.Output, "Moved %s %s to new location\n", mover.label(), s.Name.String())
	return nil
}

// updateQualifiedNameRefsFn is the HandlerDeps version of updateQualifiedNameRefs.
func updateQualifiedNameRefsFn(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName, newModule string) error {
	oldQN := name.String()
	newQN := newModule + "." + name.Name
	updated, err := deps.RenameManager.UpdateQualifiedNameInAllUnits(oldQN, newQN)
	if err != nil {
		return mdlerrors.NewBackend("update references", err)
	}
	if updated > 0 {
		fmt.Fprintf(deps.Output, "Updated references in %d document(s): %s → %s\n", updated, oldQN, newQN)
	}
	return nil
}

// moveEntityFn is the HandlerDeps version of moveEntity.
func moveEntityFn(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName, sourceModule, targetModule *model.Module) error {
	ectx := NewExecContext(ctx, deps)
	sourceDM, err := getDomainModelGenCached(ectx, sourceModule.ID)
	if err != nil {
		return mdlerrors.NewBackend("get source domain model", err)
	}
	if sourceDM == nil {
		return mdlerrors.NewBackend("get source domain model", nil)
	}

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

	targetDM, err := getDomainModelGenCached(ectx, targetModule.ID)
	if err != nil {
		return mdlerrors.NewBackend("get target domain model", err)
	}
	if targetDM == nil {
		return mdlerrors.NewBackend("get target domain model", nil)
	}

	convertedAssocs, err := deps.DomainModelWriter.MoveEntityGen(entity, model.ID(sourceDM.ID()), model.ID(targetDM.ID()), sourceModule.Name, targetModule.Name)
	if err != nil {
		return mdlerrors.NewBackend("move entity", err)
	}
	invalidateDomainModelGenForModule(ectx, sourceModule.ID)
	invalidateDomainModelGenForModule(ectx, targetModule.ID)

	if source, ok := entity.Source().(*genDm.OqlViewEntitySource); ok && source != nil && source.SourceDocumentQualifiedName() != "" {
		docName := name.Name
		if err := deps.DomainModelWriter.MoveViewEntitySourceDocument(sourceModule.Name, targetModule.ID, docName); err != nil {
			fmt.Fprintf(deps.Output, "Warning: Could not move ViewEntitySourceDocument: %v\n", err)
		}
	}

	oldQualifiedName := name.String()
	newQualifiedName := targetModule.Name + "." + name.Name
	if oqlUpdated, err := deps.DomainModelWriter.UpdateOqlQueriesForMovedEntity(oldQualifiedName, newQualifiedName); err != nil {
		fmt.Fprintf(deps.Output, "Warning: Could not update OQL queries: %v\n", err)
	} else if oqlUpdated > 0 {
		fmt.Fprintf(deps.Output, "Updated %d OQL query(ies) referencing %s\n", oqlUpdated, oldQualifiedName)
	}

	fmt.Fprintf(deps.Output, "Moved entity %s to %s\n", name.String(), targetModule.Name)
	if len(convertedAssocs) > 0 {
		fmt.Fprintf(deps.Output, "Converted %d association(s) to cross-module associations:\n", len(convertedAssocs))
		for _, assocName := range convertedAssocs {
			fmt.Fprintf(deps.Output, "  - %s\n", assocName)
		}
	}
	return nil
}



// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// execDropAssociationGenDeps is the HandlerDeps implementation for DROP ASSOCIATION.
func execDropAssociationGenDeps(ctx context.Context, s *ast.DropAssociationStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	if s == nil || s.Name.Module == "" || s.Name.Name == "" {
		return mdlerrors.NewValidation("DROP ASSOCIATION requires a qualified name (Module.Association)")
	}

	module, err := findModuleDeps(ctx, deps, s.Name.Module)
	if err != nil {
		return err
	}
	dm, err := getDomainModelGenCachedDeps(ctx, deps, module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewBackend("get domain model", nil)
	}

	for _, item := range dm.AssociationsItems() {
		assoc, ok := item.(*genDm.Association)
		if !ok || assoc.Name() != s.Name.Name {
			continue
		}
		if err := deps.DomainModelWriter.DeleteAssociation(model.ID(dm.ID()), model.ID(assoc.ID())); err != nil {
			return mdlerrors.NewBackend("delete association", err)
		}
		invalidateDomainModelGenForModuleDeps(deps, module.ID)
		invalidateHierarchyDeps(deps)
		invalidateDomainModelsCacheDeps(deps)
		trackModifiedDomainModelDeps(deps, module.ID, module.Name)
		fmt.Fprintf(deps.Output, "Dropped association: %s\n", s.Name)
		return nil
	}
	for _, item := range dm.CrossAssociationsItems() {
		assoc, ok := item.(*genDm.CrossAssociation)
		if !ok || assoc.Name() != s.Name.Name {
			continue
		}
		if err := deps.DomainModelWriter.DeleteCrossAssociation(model.ID(dm.ID()), model.ID(assoc.ID())); err != nil {
			return mdlerrors.NewBackend("delete cross-module association", err)
		}
		invalidateDomainModelGenForModuleDeps(deps, module.ID)
		invalidateHierarchyDeps(deps)
		invalidateDomainModelsCacheDeps(deps)
		trackModifiedDomainModelDeps(deps, module.ID, module.Name)
		fmt.Fprintf(deps.Output, "Dropped cross-module association: %s\n", s.Name)
		return nil
	}

	return mdlerrors.NewNotFound("association", s.Name.String())
}

func ExecDropAssociationGen(ctx *ExecContext, s *ast.DropAssociationStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	if s == nil || s.Name.Module == "" || s.Name.Name == "" {
		return mdlerrors.NewValidation("DROP ASSOCIATION requires a qualified name (Module.Association)")
	}

	module, err := findModule(ctx, s.Name.Module)
	if err != nil {
		return err
	}
	dm, err := getDomainModelGenCached(ctx, module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewBackend("get domain model", nil)
	}

	for _, item := range dm.AssociationsItems() {
		assoc, ok := item.(*genDm.Association)
		if !ok || assoc.Name() != s.Name.Name {
			continue
		}
		if err := ctx.DomainModelWriter.DeleteAssociation(model.ID(dm.ID()), model.ID(assoc.ID())); err != nil {
			return mdlerrors.NewBackend("delete association", err)
		}
		invalidateDomainModelGenForModule(ctx, module.ID)
		invalidateHierarchy(ctx)
		invalidateDomainModelsCache(ctx)
		ctx.trackModifiedDomainModel(module.ID, module.Name)
		fmt.Fprintf(ctx.Output, "Dropped association: %s\n", s.Name)
		return nil
	}
	for _, item := range dm.CrossAssociationsItems() {
		assoc, ok := item.(*genDm.CrossAssociation)
		if !ok || assoc.Name() != s.Name.Name {
			continue
		}
		if err := ctx.DomainModelWriter.DeleteCrossAssociation(model.ID(dm.ID()), model.ID(assoc.ID())); err != nil {
			return mdlerrors.NewBackend("delete cross-module association", err)
		}
		invalidateDomainModelGenForModule(ctx, module.ID)
		invalidateHierarchy(ctx)
		invalidateDomainModelsCache(ctx)
		ctx.trackModifiedDomainModel(module.ID, module.Name)
		fmt.Fprintf(ctx.Output, "Dropped cross-module association: %s\n", s.Name)
		return nil
	}

	return mdlerrors.NewNotFound("association", s.Name.String())
}

func ExecDropAssociationGenDeps(ctx context.Context, s *ast.DropAssociationStmt, deps *HandlerDeps) error {
	return execDropAssociationGenDeps(ctx, s, deps)
}



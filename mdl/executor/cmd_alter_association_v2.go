// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// execAlterAssociationGenDeps is the HandlerDeps implementation for ALTER ASSOCIATION.
func execAlterAssociationGenDeps(ctx context.Context, s *ast.AlterAssociationStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	if s == nil || s.Name.Module == "" || s.Name.Name == "" {
		return mdlerrors.NewValidation("ALTER ASSOCIATION requires a qualified name (Module.Association)")
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

	apply := func(ownerSetter func(string), storageSetter func(string), commentSetter func(string), deleteSetter func(element.Element)) {
		switch s.Operation {
		case ast.AlterAssociationSetDeleteBehavior:
			dbe := genDm.NewAssociationDeleteBehavior()
			dbe.SetParentDeleteBehavior("DeleteMeButKeepReferences")
			switch s.DeleteBehavior {
			case ast.DeleteCascade:
				dbe.SetChildDeleteBehavior("DeleteMeAndReferences")
			case ast.DeleteIfNoReferences:
				dbe.SetChildDeleteBehavior("DeleteMeIfNoReferences")
			default:
				dbe.SetChildDeleteBehavior("DeleteMeButKeepReferences")
			}
			deleteSetter(dbe)
		case ast.AlterAssociationSetOwner:
			if s.Owner == ast.OwnerBoth {
				ownerSetter("Both")
			} else {
				ownerSetter("Default")
			}
		case ast.AlterAssociationSetStorage:
			if s.Storage == ast.StorageColumn {
				storageSetter("Column")
			} else {
				storageSetter("Table")
			}
		case ast.AlterAssociationSetComment:
			commentSetter(s.Comment)
		}
	}

	for _, item := range dm.AssociationsItems() {
		assoc, ok := item.(*genDm.Association)
		if !ok || assoc.Name() != s.Name.Name {
			continue
		}
		apply(assoc.SetOwner, assoc.SetStorageFormat, assoc.SetDocumentation, assoc.SetDeleteBehavior)
		if err := deps.DomainModelWriter.UpdateDomainModelGen(dm); err != nil {
			return mdlerrors.NewBackend("update association", err)
		}
		setDomainModelGenCachedDeps(deps, module.ID, dm)
		invalidateHierarchyDeps(deps)
		invalidateDomainModelsCacheDeps(deps)
		trackModifiedDomainModelDeps(deps, module.ID, module.Name)
		fmt.Fprintf(deps.Output, "Altered association: %s\n", s.Name)
		return nil
	}

	for _, item := range dm.CrossAssociationsItems() {
		assoc, ok := item.(*genDm.CrossAssociation)
		if !ok || assoc.Name() != s.Name.Name {
			continue
		}
		apply(assoc.SetOwner, assoc.SetStorageFormat, assoc.SetDocumentation, assoc.SetDeleteBehavior)
		if err := deps.DomainModelWriter.UpdateDomainModelGen(dm); err != nil {
			return mdlerrors.NewBackend("update cross-module association", err)
		}
		setDomainModelGenCachedDeps(deps, module.ID, dm)
		invalidateHierarchyDeps(deps)
		invalidateDomainModelsCacheDeps(deps)
		trackModifiedDomainModelDeps(deps, module.ID, module.Name)
		fmt.Fprintf(deps.Output, "Altered association: %s\n", s.Name)
		return nil
	}

	return mdlerrors.NewNotFound("association", s.Name.String())
}


func ExecAlterAssociationGenDeps(ctx context.Context, s *ast.AlterAssociationStmt, deps *HandlerDeps) error {
	return execAlterAssociationGenDeps(ctx, s, deps)
}



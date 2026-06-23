// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// execCreateAssociationGen handles the same-module pure-CREATE subset of
// CREATE ASSOCIATION on the gen-typed write path. Cross-module and
// CREATE OR MODIFY still fall back to the legacy executor until the
// remaining write surface is migrated.
func execCreateAssociationGen(ctx *ExecContext, s *ast.CreateAssociationStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	if s == nil || s.Name.Module == "" || s.Name.Name == "" {
		return mdlerrors.NewValidation("CREATE ASSOCIATION requires a qualified name (Module.Association)")
	}

	parentModule := s.Parent.Module
	if parentModule == "" {
		parentModule = s.Name.Module
	}
	childModule := s.Child.Module
	if childModule == "" {
		childModule = s.Name.Module
	}

	// Keep legacy path for cross-module and OR MODIFY until the gen
	// write bridge grows CrossAssociation + update support.
	if s.CreateOrModify || parentModule != childModule {
		return ExecCreateAssociation(ctx, s)
	}

	module, err := findOrCreateModule(ctx, s.Name.Module)
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
		assoc, ok := item.(interface{ Name() string })
		if ok && assoc.Name() == s.Name.Name {
			return mdlerrors.NewAlreadyExists("association", s.Name.String())
		}
	}
	for _, item := range dm.CrossAssociationsItems() {
		assoc, ok := item.(interface{ Name() string })
		if ok && assoc.Name() == s.Name.Name {
			return mdlerrors.NewAlreadyExists("association", s.Name.String())
		}
	}

	parentEntity, _, err := findEntityGen(ctx, ast.QualifiedName{Module: parentModule, Name: s.Parent.Name})
	if err != nil {
		return mdlerrors.NewBackend("get parent entity", err)
	}
	if parentEntity == nil {
		return mdlerrors.NewNotFound("parent entity", s.Parent.String())
	}
	childEntity, _, err := findEntityGen(ctx, ast.QualifiedName{Module: childModule, Name: s.Child.Name})
	if err != nil {
		return mdlerrors.NewBackend("get child entity", err)
	}
	if childEntity == nil {
		return mdlerrors.NewNotFound("child entity", s.Child.String())
	}

	assoc := astToAssociationGen(s, element.ID(parentEntity.ID()), element.ID(childEntity.ID()))
	if assoc == nil {
		return mdlerrors.NewValidation("failed to build gen association from AST")
	}

	if err := ctx.DomainModelWriter.CreateAssociationGen(model.ID(dm.ID()), assoc); err != nil {
		return mdlerrors.NewBackend("create association (gen)", err)
	}

	invalidateDomainModelGenForModule(ctx, module.ID)
	invalidateHierarchy(ctx)
	invalidateDomainModelsCache(ctx)

	if err := ctx.DomainModelWriter.RelayoutDomainModel(model.ID(dm.ID())); err != nil {
		fmt.Fprintf(ctx.Output, "warning: auto-layout failed: %v\n", err)
	} else {
		invalidateDomainModelGenForModule(ctx, module.ID)
	}

	if freshDM, err := getDomainModelGenCached(ctx, module.ID); err == nil && freshDM != nil {
		if msgs, err := ctx.SecurityEntityAccessManager.ReconcileMemberAccesses(model.ID(freshDM.ID()), module.Name); err == nil {
			for _, msg := range msgs {
				fmt.Fprintf(ctx.Output, "  reconciled: %s\n", msg)
			}
		}
	}

	ctx.trackModifiedDomainModel(module.ID, module.Name)
	fmt.Fprintf(ctx.Output, "Created association: %s\n", s.Name)
	return nil
}

package domainmodel

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

func ExecCreateAssociationFn(ctx context.Context, s *ast.CreateAssociationStmt, d DomainModelDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	module, err := d.FindOrCreateModule(s.Name.Module)
	if err != nil {
		return err
	}

	dm, err := d.GetDomainModelGenCached(module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewBackend("get domain model", nil)
	}

	parentModule := s.Parent.Module
	if parentModule == "" {
		parentModule = s.Name.Module
	}
	parentEntity, _, err := d.FindEntityGen(ast.QualifiedName{Module: parentModule, Name: s.Parent.Name})
	if err != nil {
		return mdlerrors.NewBackend("get parent entity", err)
	}
	if parentEntity == nil {
		return mdlerrors.NewNotFound("parent entity", s.Parent.String())
	}
	parentID := element.ID(parentEntity.ID())

	childModule := s.Child.Module
	if childModule == "" {
		childModule = s.Name.Module
	}
	childEntity, _, err := d.FindEntityGen(ast.QualifiedName{Module: childModule, Name: s.Child.Name})
	if err != nil {
		return mdlerrors.NewBackend("get child entity", err)
	}
	if childEntity == nil {
		return mdlerrors.NewNotFound("child entity", s.Child.String())
	}
	childID := element.ID(childEntity.ID())

	parentPersistable := d.EntityPersistable(parentEntity)
	childPersistable := d.EntityPersistable(childEntity)
	if parentPersistable && !childPersistable {
		return mdlerrors.NewValidationf(
			"association owner must be the non-persistent entity: %q is persistable but %q is non-persistent — reverse the direction: from %s to %s",
			s.Parent.String(), s.Child.String(), s.Child.String(), s.Parent.String(),
		)
	}

	isCrossModule := parentModule != childModule

	if s.CreateOrModify {
		if !isCrossModule {
			for _, assocElem := range dm.AssociationsItems() {
				assoc, ok := assocElem.(*genDm.Association)
				if !ok || assoc.Name() != s.Name.Name {
					continue
				}
				assoc.SetType(executor.AstAssociationTypeStringGen(s))
				assoc.SetOwner(executor.AstAssociationOwnerStringGen(s))
				assoc.SetStorageFormat(executor.AstAssociationStorageStringGen(s))
				assoc.SetDeleteBehavior(executor.AstAssociationDeleteBehaviorGen(s))
				if doc := executor.AssociationDocumentation(s); doc != "" {
					assoc.SetDocumentation(doc)
				}
				if err := d.DomainModelWriter.UpdateDomainModelGen(dm); err != nil {
					return mdlerrors.NewBackend("update association", err)
				}
				d.SetDomainModelGenCached(module.ID, dm)
				d.InvalidateHierarchy()
				d.InvalidateDomainModelsCache()
				d.TrackModifiedDomainModel(module.ID, module.Name)
				fmt.Fprintf(d.Output, "Modified association: %s\n", s.Name)
				return nil
			}
		} else {
			childRef := childModule + "." + s.Child.Name
			for _, crossElem := range dm.CrossAssociationsItems() {
				ca, ok := crossElem.(*genDm.CrossAssociation)
				if !ok || ca.Name() != s.Name.Name {
					continue
				}
				ca.SetType(executor.AstAssociationTypeStringGen(s))
				ca.SetOwner(executor.AstAssociationOwnerStringGen(s))
				ca.SetStorageFormat(executor.AstAssociationStorageStringGen(s))
				ca.SetDeleteBehavior(executor.AstAssociationDeleteBehaviorGen(s))
				ca.SetChildQualifiedName(childRef)
				if doc := executor.AssociationDocumentation(s); doc != "" {
					ca.SetDocumentation(doc)
				}
				if err := d.DomainModelWriter.UpdateDomainModelGen(dm); err != nil {
					return mdlerrors.NewBackend("update cross-module association", err)
				}
				d.SetDomainModelGenCached(module.ID, dm)
				d.InvalidateHierarchy()
				d.InvalidateDomainModelsCache()
				d.TrackModifiedDomainModel(module.ID, module.Name)
				fmt.Fprintf(d.Output, "Modified association: %s\n", s.Name)
				return nil
			}
		}
	}

	if isCrossModule {
		if !s.CreateOrModify {
			for _, crossElem := range dm.CrossAssociationsItems() {
				ca, ok := crossElem.(*genDm.CrossAssociation)
				if ok && ca.Name() == s.Name.Name {
					return mdlerrors.NewAlreadyExists("association", s.Name.String())
				}
			}
			for _, assocElem := range dm.AssociationsItems() {
				assoc, ok := assocElem.(*genDm.Association)
				if ok && assoc.Name() == s.Name.Name {
					return mdlerrors.NewAlreadyExists("association", s.Name.String())
				}
			}
		}
		ca := genDm.NewCrossAssociation()
		ca.SetName(s.Name.Name)
		ca.SetParentID(parentID)
		ca.SetChildQualifiedName(childModule + "." + s.Child.Name)
		ca.SetType(executor.AstAssociationTypeStringGen(s))
		ca.SetOwner(executor.AstAssociationOwnerStringGen(s))
		ca.SetStorageFormat(executor.AstAssociationStorageStringGen(s))
		ca.SetDeleteBehavior(executor.AstAssociationDeleteBehaviorGen(s))
		if doc := executor.AssociationDocumentation(s); doc != "" {
			ca.SetDocumentation(doc)
		}
		dm.AddCrossAssociations(ca)
		if err := d.DomainModelWriter.UpdateDomainModelGen(dm); err != nil {
			return mdlerrors.NewBackend("create cross-module association", err)
		}
		d.SetDomainModelGenCached(module.ID, dm)
	} else {
		if !s.CreateOrModify {
			for _, assocElem := range dm.AssociationsItems() {
				assoc, ok := assocElem.(*genDm.Association)
				if ok && assoc.Name() == s.Name.Name {
					return mdlerrors.NewAlreadyExists("association", s.Name.String())
				}
			}
			for _, crossElem := range dm.CrossAssociationsItems() {
				ca, ok := crossElem.(*genDm.CrossAssociation)
				if ok && ca.Name() == s.Name.Name {
					return mdlerrors.NewAlreadyExists("association", s.Name.String())
				}
			}
		}
		assoc := executor.AstToAssociationGen(s, parentID, childID)
		if doc := executor.AssociationDocumentation(s); doc != "" {
			assoc.SetDocumentation(doc)
		}
		if err := d.DomainModelWriter.CreateAssociationGen(model.ID(dm.ID()), assoc); err != nil {
			return mdlerrors.NewBackend("create association", err)
		}
		d.InvalidateDomainModelGenCache(module.ID)
	}

	d.InvalidateHierarchy()
	d.InvalidateDomainModelsCache()

	if freshDM, err := d.GetDomainModelGenCached(module.ID); err == nil && freshDM != nil {
		if err := d.DomainModelWriter.RelayoutDomainModel(model.ID(freshDM.ID())); err != nil {
			fmt.Fprintf(d.Output, "warning: auto-layout failed: %v\n", err)
		} else {
			d.InvalidateDomainModelGenCache(module.ID)
		}
	}

	if freshDM, err := d.GetDomainModelGenCached(module.ID); err == nil && freshDM != nil {
		if msgs, err := d.SecurityEntityAccessManager.ReconcileMemberAccesses(model.ID(freshDM.ID()), module.Name); err == nil {
			for _, msg := range msgs {
				fmt.Fprintf(d.Output, "  reconciled: %s\n", msg)
			}
		}
	}

	d.TrackModifiedDomainModel(module.ID, module.Name)
	fmt.Fprintf(d.Output, "Created association: %s\n", s.Name)
	return nil
}

func ExecAlterAssociationFn(ctx context.Context, s *ast.AlterAssociationStmt, d DomainModelDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	module, err := d.FindModule(s.Name.Module)
	if err != nil {
		return err
	}

	dm, err := d.GetDomainModelGenCached(module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewNotFound("association", s.Name.String())
	}

	for _, assocElem := range dm.AssociationsItems() {
		assoc, ok := assocElem.(*genDm.Association)
		if !ok || assoc.Name() != s.Name.Name {
			continue
		}
		switch s.Operation {
		case ast.AlterAssociationSetDeleteBehavior:
			assoc.SetDeleteBehavior(executor.NewAssociationDeleteBehaviorGen(s.DeleteBehavior))
		case ast.AlterAssociationSetOwner:
			assoc.SetOwner(s.Owner.String())
		case ast.AlterAssociationSetStorage:
			assoc.SetStorageFormat(s.Storage.String())
		case ast.AlterAssociationSetComment:
			assoc.SetDocumentation(s.Comment)
		}
		if err := d.DomainModelWriter.UpdateDomainModelGen(dm); err != nil {
			return mdlerrors.NewBackend("update association", err)
		}
		d.SetDomainModelGenCached(module.ID, dm)
		fmt.Fprintf(d.Output, "Altered association: %s\n", s.Name)
		return nil
	}

	for _, crossElem := range dm.CrossAssociationsItems() {
		ca, ok := crossElem.(*genDm.CrossAssociation)
		if !ok || ca.Name() != s.Name.Name {
			continue
		}
		switch s.Operation {
		case ast.AlterAssociationSetDeleteBehavior:
			ca.SetDeleteBehavior(executor.NewAssociationDeleteBehaviorGen(s.DeleteBehavior))
		case ast.AlterAssociationSetOwner:
			ca.SetOwner(s.Owner.String())
		case ast.AlterAssociationSetStorage:
			ca.SetStorageFormat(s.Storage.String())
		case ast.AlterAssociationSetComment:
			ca.SetDocumentation(s.Comment)
		}
		if err := d.DomainModelWriter.UpdateDomainModelGen(dm); err != nil {
			return mdlerrors.NewBackend("update cross-module association", err)
		}
		d.SetDomainModelGenCached(module.ID, dm)
		fmt.Fprintf(d.Output, "Altered association: %s\n", s.Name)
		return nil
	}

	return mdlerrors.NewNotFound("association", s.Name.String())
}

func ExecDropAssociationFn(ctx context.Context, s *ast.DropAssociationStmt, d DomainModelDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	module, err := d.FindModule(s.Name.Module)
	if err != nil {
		return err
	}

	dm, err := d.GetDomainModelGenCached(module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewNotFound("association", s.Name.String())
	}

	for _, assocElem := range dm.AssociationsItems() {
		assoc, ok := assocElem.(*genDm.Association)
		if !ok || assoc.Name() != s.Name.Name {
			continue
		}
		if err := d.DomainModelWriter.DeleteAssociation(model.ID(dm.ID()), model.ID(assoc.ID())); err != nil {
			return mdlerrors.NewBackend("delete association", err)
		}
		fmt.Fprintf(d.Output, "Dropped association: %s\n", s.Name)
		return nil
	}
	for _, crossElem := range dm.CrossAssociationsItems() {
		ca, ok := crossElem.(*genDm.CrossAssociation)
		if !ok || ca.Name() != s.Name.Name {
			continue
		}
		if err := d.DomainModelWriter.DeleteCrossAssociation(model.ID(dm.ID()), model.ID(ca.ID())); err != nil {
			return mdlerrors.NewBackend("delete cross-module association", err)
		}
		fmt.Fprintf(d.Output, "Dropped cross-module association: %s\n", s.Name)
		return nil
	}

	return mdlerrors.NewNotFound("association", s.Name.String())
}

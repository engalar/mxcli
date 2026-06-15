// SPDX-License-Identifier: Apache-2.0

// Package executor - Association commands (SHOW/DESCRIBE/CREATE/DROP ASSOCIATION)
package executor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// execCreateAssociation handles CREATE ASSOCIATION statements.
func execCreateAssociation(ctx *ExecContext, s *ast.CreateAssociationStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}

	// Find or auto-create module
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

	// Find parent and child entities (supports cross-module associations)
	parentModule := s.Parent.Module
	if parentModule == "" {
		parentModule = s.Name.Module
	}
	parentEntity, _, err := findEntityGen(ctx, ast.QualifiedName{Module: parentModule, Name: s.Parent.Name})
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
	childEntity, _, err := findEntityGen(ctx, ast.QualifiedName{Module: childModule, Name: s.Child.Name})
	if err != nil {
		return mdlerrors.NewBackend("get child entity", err)
	}
	if childEntity == nil {
		return mdlerrors.NewNotFound("child entity", s.Child.String())
	}
	childID := element.ID(childEntity.ID())

	// Validate persistability constraint: when one entity is non-persistent,
	// the non-persistent entity must be the FROM (owner) side.
	parentPersistable := entityPersistableGen(parentEntity)
	childPersistable := entityPersistableGen(childEntity)
	if parentPersistable && !childPersistable {
		return mdlerrors.NewValidationf(
			"association owner must be the non-persistent entity: %q is persistable but %q is non-persistent — reverse the direction: from %s to %s",
			s.Parent.String(), s.Child.String(), s.Child.String(), s.Parent.String(),
		)
	}

	// Create association
	// ParentID = FROM entity (the one with the FK)
	// ChildID = TO entity (the one being referenced)
	// Cross-module associations use BY_NAME for the child entity
	isCrossModule := parentModule != childModule

	// OR MODIFY: update existing association properties in place, preserving its UUID.
	if s.CreateOrModify {
		if !isCrossModule {
			for _, assocElem := range dm.AssociationsItems() {
				assoc, ok := assocElem.(*genDm.Association)
				if !ok || assoc.Name() != s.Name.Name {
					continue
				}
				assoc.SetType(astAssociationTypeStringGen(s))
				assoc.SetOwner(astAssociationOwnerStringGen(s))
				assoc.SetStorageFormat(astAssociationStorageStringGen(s))
				assoc.SetDeleteBehavior(astAssociationDeleteBehaviorGen(s))
				if doc := associationDocumentation(s); doc != "" {
					assoc.SetDocumentation(doc)
				}
				if err := ctx.DomainModelWriter.UpdateDomainModelGen(dm); err != nil {
					return mdlerrors.NewBackend("update association", err)
				}
				setDomainModelGenCached(ctx, module.ID, dm)
				invalidateHierarchy(ctx)
				invalidateDomainModelsCache(ctx)
				ctx.trackModifiedDomainModel(module.ID, module.Name)
				fmt.Fprintf(ctx.Output, "Modified association: %s\n", s.Name)
				return nil
			}
		} else {
			childRef := childModule + "." + s.Child.Name
			for _, crossElem := range dm.CrossAssociationsItems() {
				ca, ok := crossElem.(*genDm.CrossAssociation)
				if !ok || ca.Name() != s.Name.Name {
					continue
				}
				ca.SetType(astAssociationTypeStringGen(s))
				ca.SetOwner(astAssociationOwnerStringGen(s))
				ca.SetStorageFormat(astAssociationStorageStringGen(s))
				ca.SetDeleteBehavior(astAssociationDeleteBehaviorGen(s))
				ca.SetChildQualifiedName(childRef)
				if doc := associationDocumentation(s); doc != "" {
					ca.SetDocumentation(doc)
				}
				if err := ctx.DomainModelWriter.UpdateDomainModelGen(dm); err != nil {
					return mdlerrors.NewBackend("update cross-module association", err)
				}
				setDomainModelGenCached(ctx, module.ID, dm)
				invalidateHierarchy(ctx)
				invalidateDomainModelsCache(ctx)
				ctx.trackModifiedDomainModel(module.ID, module.Name)
				fmt.Fprintf(ctx.Output, "Modified association: %s\n", s.Name)
				return nil
			}
		}
		// Association not found — fall through to create it.
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
		ca.SetType(astAssociationTypeStringGen(s))
		ca.SetOwner(astAssociationOwnerStringGen(s))
		ca.SetStorageFormat(astAssociationStorageStringGen(s))
		ca.SetDeleteBehavior(astAssociationDeleteBehaviorGen(s))
		if doc := associationDocumentation(s); doc != "" {
			ca.SetDocumentation(doc)
		}
		dm.AddCrossAssociations(ca)
		if err := ctx.DomainModelWriter.UpdateDomainModelGen(dm); err != nil {
			return mdlerrors.NewBackend("create cross-module association", err)
		}
		setDomainModelGenCached(ctx, module.ID, dm)
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
		assoc := astToAssociationGen(s, parentID, childID)
		if doc := associationDocumentation(s); doc != "" {
			assoc.SetDocumentation(doc)
		}
		if err := ctx.DomainModelWriter.CreateAssociationGen(model.ID(dm.ID()), assoc); err != nil {
			return mdlerrors.NewBackend("create association", err)
		}
		invalidateDomainModelGenForModule(ctx, module.ID)
	}

	// Invalidate hierarchy cache so the new association's container is visible
	invalidateHierarchy(ctx)
	invalidateDomainModelsCache(ctx)

	if freshDM, err := getDomainModelGenCached(ctx, module.ID); err == nil && freshDM != nil {
		if err := ctx.DomainModelWriter.RelayoutDomainModel(model.ID(freshDM.ID())); err != nil {
			fmt.Fprintf(ctx.Output, "warning: auto-layout failed: %v\n", err)
		} else {
			invalidateDomainModelGenForModule(ctx, module.ID)
		}
	}

	// Reconcile MemberAccesses immediately — existing access rules on entities
	// in this DM need MemberAccess entries for the new association (CE0066).
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

func associationDocumentation(s *ast.CreateAssociationStmt) string {
	if s == nil {
		return ""
	}
	if s.Documentation != "" {
		return s.Documentation
	}
	if s.Comment != "" {
		return s.Comment
	}
	return ""
}

// execAlterAssociation handles ALTER ASSOCIATION statements.
func execAlterAssociation(ctx *ExecContext, s *ast.AlterAssociationStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
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
		return mdlerrors.NewNotFound("association", s.Name.String())
	}

	// Try intra-module associations first
	for _, assocElem := range dm.AssociationsItems() {
		assoc, ok := assocElem.(*genDm.Association)
		if !ok || assoc.Name() != s.Name.Name {
			continue
		}
		switch s.Operation {
		case ast.AlterAssociationSetDeleteBehavior:
			assoc.SetDeleteBehavior(newAssociationDeleteBehaviorGen(s.DeleteBehavior))
		case ast.AlterAssociationSetOwner:
			assoc.SetOwner(s.Owner.String())
		case ast.AlterAssociationSetStorage:
			assoc.SetStorageFormat(s.Storage.String())
		case ast.AlterAssociationSetComment:
			assoc.SetDocumentation(s.Comment)
		}
		if err := ctx.DomainModelWriter.UpdateDomainModelGen(dm); err != nil {
			return mdlerrors.NewBackend("update association", err)
		}
		setDomainModelGenCached(ctx, module.ID, dm)
		fmt.Fprintf(ctx.Output, "Altered association: %s\n", s.Name)
		return nil
	}

	// Try cross-module associations
	for _, crossElem := range dm.CrossAssociationsItems() {
		ca, ok := crossElem.(*genDm.CrossAssociation)
		if !ok || ca.Name() != s.Name.Name {
			continue
		}
		switch s.Operation {
		case ast.AlterAssociationSetDeleteBehavior:
			ca.SetDeleteBehavior(newAssociationDeleteBehaviorGen(s.DeleteBehavior))
		case ast.AlterAssociationSetOwner:
			ca.SetOwner(s.Owner.String())
		case ast.AlterAssociationSetStorage:
			ca.SetStorageFormat(s.Storage.String())
		case ast.AlterAssociationSetComment:
			ca.SetDocumentation(s.Comment)
		}
		if err := ctx.DomainModelWriter.UpdateDomainModelGen(dm); err != nil {
			return mdlerrors.NewBackend("update cross-module association", err)
		}
		setDomainModelGenCached(ctx, module.ID, dm)
		fmt.Fprintf(ctx.Output, "Altered association: %s\n", s.Name)
		return nil
	}

	return mdlerrors.NewNotFound("association", s.Name.String())
}

// execDropAssociation handles DROP ASSOCIATION statements.
func execDropAssociation(ctx *ExecContext, s *ast.DropAssociationStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}

	// Find module
	module, err := findModule(ctx, s.Name.Module)
	if err != nil {
		return err
	}

	dm, err := getDomainModelGenCached(ctx, module.ID)
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
		if err := ctx.DomainModelWriter.DeleteAssociation(model.ID(dm.ID()), model.ID(assoc.ID())); err != nil {
			return mdlerrors.NewBackend("delete association", err)
		}
		fmt.Fprintf(ctx.Output, "Dropped association: %s\n", s.Name)
		return nil
	}
	for _, crossElem := range dm.CrossAssociationsItems() {
		ca, ok := crossElem.(*genDm.CrossAssociation)
		if !ok || ca.Name() != s.Name.Name {
			continue
		}
		if err := ctx.DomainModelWriter.DeleteCrossAssociation(model.ID(dm.ID()), model.ID(ca.ID())); err != nil {
			return mdlerrors.NewBackend("delete cross-module association", err)
		}
		fmt.Fprintf(ctx.Output, "Dropped cross-module association: %s\n", s.Name)
		return nil
	}

	return mdlerrors.NewNotFound("association", s.Name.String())
}

// listAssociation handles SHOW ASSOCIATION command.
func listAssociation(ctx *ExecContext, name *ast.QualifiedName) error {
	if name == nil {
		return mdlerrors.NewValidation("association name required")
	}

	module, err := findModule(ctx, name.Module)
	if err != nil {
		return err
	}

	dm, err := getDomainModelGenCached(ctx, module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewNotFound("association", name.String())
	}

	// Build entity ID → qualified name map for this domain model.
	entityNames := make(map[model.ID]string)
	for _, e := range dm.EntitiesItems() {
		entity, ok := e.(*genDm.Entity)
		if !ok {
			continue
		}
		entityNames[model.ID(entity.ID())] = module.Name + "." + entity.Name()
	}

	for _, assocElem := range dm.AssociationsItems() {
		assoc, ok := assocElem.(*genDm.Association)
		if !ok || assoc.Name() != name.Name {
			continue
		}
		parent := entityNames[model.ID(assoc.ParentRefID())]
		child := entityNames[model.ID(assoc.ChildRefID())]
		fmt.Fprintf(ctx.Output, "Association: %s.%s\n", module.Name, assoc.Name())
		fmt.Fprintf(ctx.Output, "  Multiplicity: %s\n", associationMultiplicity(assoc.Type(), assoc.Owner()))
		fmt.Fprintf(ctx.Output, "  FROM (owner): %s\n", parent)
		fmt.Fprintf(ctx.Output, "  TO (referenced): %s\n", child)
		fmt.Fprintf(ctx.Output, "  Type: %s\n", assoc.Type())
		fmt.Fprintf(ctx.Output, "  Owner: %s\n", assoc.Owner())
		fmt.Fprintf(ctx.Output, "  Storage: %s\n", assoc.StorageFormat())
		return nil
	}
	for _, crossElem := range dm.CrossAssociationsItems() {
		ca, ok := crossElem.(*genDm.CrossAssociation)
		if !ok || ca.Name() != name.Name {
			continue
		}
		parent := entityNames[model.ID(ca.ParentRefID())]
		fmt.Fprintf(ctx.Output, "Association: %s.%s (cross-module)\n", module.Name, ca.Name())
		fmt.Fprintf(ctx.Output, "  Multiplicity: %s\n", associationMultiplicity(ca.Type(), ca.Owner()))
		fmt.Fprintf(ctx.Output, "  FROM (owner): %s\n", parent)
		fmt.Fprintf(ctx.Output, "  TO (referenced): %s\n", ca.ChildQualifiedName())
		fmt.Fprintf(ctx.Output, "  Type: %s\n", ca.Type())
		fmt.Fprintf(ctx.Output, "  Owner: %s\n", ca.Owner())
		fmt.Fprintf(ctx.Output, "  Storage: %s\n", ca.StorageFormat())
		return nil
	}

	return mdlerrors.NewNotFound("association", name.String())
}

type elementLikeDeleteBehavior interface {
	ChildDeleteBehavior() string
}

func associationDeleteBehaviorGen(elem interface{}) elementLikeDeleteBehavior {
	db, _ := elem.(*genDm.AssociationDeleteBehavior)
	return db
}

func newAssociationDeleteBehaviorGen(db ast.DeleteBehavior) *genDm.AssociationDeleteBehavior {
	out := genDm.NewAssociationDeleteBehavior()
	switch db {
	case ast.DeleteCascade:
		out.SetChildDeleteBehavior(genDm.DeletingBehaviorDeleteMeAndReferences)
	case ast.DeleteIfNoReferences:
		out.SetChildDeleteBehavior(genDm.DeletingBehaviorDeleteMeIfNoReferences)
	default:
		out.SetChildDeleteBehavior(genDm.DeletingBehaviorDeleteMeButKeepReferences)
	}
	return out
}

// describeAssociationComment returns a single-line MDL comment that describes
// the association in plain English, e.g. "-- one-to-many: Order *-->1 Customer".
func describeAssociationComment(assocType, owner, fromQN, toQN string) string {
	switch associationMultiplicity(assocType, owner) {
	case "1--1":
		return fmt.Sprintf("-- one-to-one: %s -- %s (both own)", fromQN, toQN)
	case "*-->1":
		return fmt.Sprintf("-- one-to-many: %s *-->1 %s (each %s belongs to one %s)", fromQN, toQN, fromQN, toQN)
	case "*--*":
		return fmt.Sprintf("-- many-to-many: %s *--* %s (both own)", fromQN, toQN)
	default:
		return fmt.Sprintf("-- many-to-many: %s *-->* %s (only %s owns)", fromQN, toQN, fromQN)
	}
}

// associationMultiplicity returns a directional multiplicity label.
//
// The `>` marker always appears on the TO (non-owning) side.
// When both sides own (owner=Both), no `>` appears — just `--`.
//
//	*-->1   Reference + Default  one-to-many  (FROM owns, TO does not)
//	1--1    Reference + Both     one-to-one   (both own)
//	*-->*   ReferenceSet + Default  many-to-many, only FROM owns
//	*--*    ReferenceSet + Both     many-to-many, both own
func associationMultiplicity(assocType, owner string) string {
	isRefSet := assocType == genDm.AssociationTypeReferenceSet
	isBoth := owner == genDm.AssociationOwnerBoth
	switch {
	case !isRefSet && isBoth:
		return "1--1"
	case !isRefSet:
		return "*-->1"
	case isBoth:
		return "*--*"
	default:
		return "*-->*"
	}
}

// listAssociations handles SHOW ASSOCIATIONS command.
func listAssociations(ctx *ExecContext, moduleName string) error {
	pairs, err := listDomainModelsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list domain models", err)
	}

	mods, err := ctx.ModuleLister.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("list modules", err)
	}
	moduleNames := make(map[model.ID]string, len(mods))
	for _, m := range mods {
		moduleNames[m.ID] = m.Name
	}

	entityNames := make(map[model.ID]string)
	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		modName := moduleNames[p.ContainerID]
		for _, e := range p.DM.EntitiesItems() {
			entity, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			entityNames[model.ID(entity.ID())] = modName + "." + entity.Name()
		}
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		parent        string
		child         string
		multiplicity  string
		assocType     string
		owner         string
		storage       string
	}
	var rows []row

	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		modName := moduleNames[p.ContainerID]
		if moduleName != "" && modName != moduleName {
			continue
		}
		for _, a := range p.DM.AssociationsItems() {
			assoc, ok := a.(*genDm.Association)
			if !ok {
				continue
			}
			parentID := model.ID(assoc.ParentRefID())
			childID := model.ID(assoc.ChildRefID())
			parent := entityNames[parentID]
			if parent == "" {
				parent = string(parentID)
			}
			child := entityNames[childID]
			if child == "" {
				child = string(childID)
			}
			rows = append(rows, row{
				qualifiedName: modName + "." + assoc.Name(),
				module:        modName,
				name:          assoc.Name(),
				parent:        parent,
				child:         child,
				multiplicity:  associationMultiplicity(assoc.Type(), assoc.Owner()),
				assocType:     assoc.Type(),
				owner:         assoc.Owner(),
				storage:       assoc.StorageFormat(),
			})
		}
		for _, c := range p.DM.CrossAssociationsItems() {
			ca, ok := c.(*genDm.CrossAssociation)
			if !ok {
				continue
			}
			parentID := model.ID(ca.ParentRefID())
			parent := entityNames[parentID]
			if parent == "" {
				parent = string(parentID)
			}
			rows = append(rows, row{
				qualifiedName: modName + "." + ca.Name(),
				module:        modName,
				name:          ca.Name(),
				parent:        parent,
				child:         ca.ChildQualifiedName(),
				multiplicity:  associationMultiplicity(ca.Type(), ca.Owner()),
				assocType:     ca.Type(),
				owner:         ca.Owner(),
				storage:       ca.StorageFormat(),
			})
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "FROM (owner)", "TO (referenced)", "Multiplicity", "Type", "Owner", "Storage"},
		Summary: fmt.Sprintf("(%d associations)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.parent, r.child, r.multiplicity, r.assocType, r.owner, r.storage})
	}
	return writeResult(ctx, result)
}

// describeAssociation handles DESCRIBE ASSOCIATION command.
func describeAssociation(ctx *ExecContext, name ast.QualifiedName) error {
	pairs, err := listDomainModelsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list domain models", err)
	}
	mods, err := ctx.ModuleLister.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("list modules", err)
	}
	moduleNames := make(map[model.ID]string, len(mods))
	for _, m := range mods {
		moduleNames[m.ID] = m.Name
	}
	// entityNames keyed by ID string for assocSpecFromGen compatibility.
	entityNamesByIDStr := make(map[string]string)
	// entityNames keyed by model.ID for cross-module lookup.
	entityNames := make(map[model.ID]string)
	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		modName := moduleNames[p.ContainerID]
		for _, e := range p.DM.EntitiesItems() {
			entity, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			qn := modName + "." + entity.Name()
			entityNamesByIDStr[string(entity.ID())] = qn
			entityNames[model.ID(entity.ID())] = qn
		}
	}

	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		modName := moduleNames[p.ContainerID]
		if modName != name.Module {
			continue
		}
		for _, a := range p.DM.AssociationsItems() {
			assoc, ok := a.(*genDm.Association)
			if !ok || assoc.Name() != name.Name {
				continue
			}
			spec := assocSpecFromGen(modName, assoc, entityNamesByIDStr)
			comment := describeAssociationComment(assoc.Type(), assoc.Owner(), spec.fromQN, spec.toQN)
			fmt.Fprintf(ctx.Output, "%s\n", comment)
			fmt.Fprintf(ctx.Output, "%s;\n/\n", renderAssocMDL(spec))
			return nil
		}
		for _, c := range p.DM.CrossAssociationsItems() {
			ca, ok := c.(*genDm.CrossAssociation)
			if !ok || ca.Name() != name.Name {
				continue
			}
			fromQN := entityNames[model.ID(ca.ParentRefID())]
			if fromQN == "" {
				fromQN = string(ca.ParentRefID())
			}
			toQN := ca.ChildQualifiedName()
			deleteBehavior := "DELETE_BUT_KEEP_REFERENCES"
			if dbe, ok := ca.DeleteBehavior().(*genDm.AssociationDeleteBehavior); ok && dbe != nil {
				deleteBehavior = genAssocDeleteBehaviorToMDL(dbe.ChildDeleteBehavior())
			}
			assocType := "Reference"
			if ca.Type() == "ReferenceSet" {
				assocType = "ReferenceSet"
			}
			owner := "Default"
			if ca.Owner() == "Both" {
				owner = "Both"
			}
			spec := assocMDLSpec{
				module:         modName,
				name:           ca.Name(),
				fromQN:         fromQN,
				toQN:           toQN,
				documentation:  ca.Documentation(),
				assocType:      assocType,
				owner:          owner,
				deleteBehavior: deleteBehavior,
			}
			comment := describeAssociationComment(ca.Type(), ca.Owner(), fromQN, toQN)
			fmt.Fprintf(ctx.Output, "%s\n", comment)
			fmt.Fprintf(ctx.Output, "%s;\n/\n", renderAssocMDL(spec))
			return nil
		}
	}
	return mdlerrors.NewNotFound("association", name.String())
}

// --- Executor method wrappers for callers not yet migrated ---

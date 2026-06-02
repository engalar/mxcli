// SPDX-License-Identifier: Apache-2.0

package association

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/canonical"
	mxID "github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// Persist writes the AssociationModel to the project via ctx.Backend.
// ctx.Backend must satisfy assocBackend (satisfied by backend.DomainModelBackend).
// The FROM and TO entity IDs are resolved from their qualified names via the
// backend; the resolved IDs become the ParentPointer (FROM) and ChildPointer
// (TO) respectively, matching Mendix's inverted pointer convention.
func (m *AssociationModel) Persist(ctx canonical.PersistContext) error {
	if m == nil {
		return fmt.Errorf("association.Persist: nil model")
	}
	if ctx.Backend == nil {
		return fmt.Errorf("association.Persist: nil backend")
	}
	if ctx.DomainModelID == "" {
		return fmt.Errorf("association.Persist: missing DomainModelID")
	}
	type assocBackend interface {
		CreateAssociationGen(domainModelID mxID.ID, assoc *genDm.Association) error
		GetEntityIDByQualifiedName(qualifiedName string) (element.ID, error)
	}
	b, ok := ctx.Backend.(assocBackend)
	if !ok {
		return fmt.Errorf("association.Persist: backend %T does not implement assocBackend", ctx.Backend)
	}

	fromID, err := b.GetEntityIDByQualifiedName(m.From.String())
	if err != nil {
		return fmt.Errorf("association.Persist: resolve FROM entity %s: %w", m.From, err)
	}
	toID, err := b.GetEntityIDByQualifiedName(m.To.String())
	if err != nil {
		return fmt.Errorf("association.Persist: resolve TO entity %s: %w", m.To, err)
	}

	a := genDm.NewAssociation()
	a.SetName(m.Name.Name)
	if m.Documentation != "" {
		a.SetDocumentation(m.Documentation)
	}
	a.SetParentID(fromID)
	a.SetChildID(toID)
	a.SetType(assocTypeToGen(m.Type))
	a.SetOwner(ownerToGen(m.Owner))
	a.SetStorageFormat(storageToGen(m.Storage))
	a.SetDeleteBehavior(deleteBehaviorToGen(m.DeleteBehavior))

	return b.CreateAssociationGen(ctx.DomainModelID, a)
}

func assocTypeToGen(t AssocType) string {
	if t == AssocReferenceSet {
		return "ReferenceSet"
	}
	return "Reference"
}

func ownerToGen(o OwnerType) string {
	if o == OwnerBoth {
		return "Both"
	}
	return "Default"
}

func storageToGen(s StorageType) string {
	if s == StorageColumn {
		return "AssociationStorageColumn"
	}
	return "AssociationStorageCrossTable"
}

func deleteBehaviorToGen(d DeleteBehaviorType) element.Element {
	db := genDm.NewAssociationDeleteBehavior()
	switch d {
	case DeleteCascade:
		db.SetChildDeleteBehavior("DeleteMeAndReferences")
		db.SetParentDeleteBehavior("Nothing")
	case DeleteBoth:
		db.SetChildDeleteBehavior("DeleteBoth")
		db.SetParentDeleteBehavior("DeleteBoth")
	case DeleteKeepParentDeleteChild:
		db.SetChildDeleteBehavior("KeepParentDeleteChild")
		db.SetParentDeleteBehavior("Nothing")
	case DeleteKeepChildDeleteParent:
		db.SetChildDeleteBehavior("KeepChildDeleteParent")
		db.SetParentDeleteBehavior("Nothing")
	case DeleteIfNoReferences:
		db.SetChildDeleteBehavior("DeleteIfNoReferences")
		db.SetParentDeleteBehavior("Nothing")
	default:
		db.SetChildDeleteBehavior("DeleteMeButKeepReferences")
		db.SetParentDeleteBehavior("Nothing")
	}
	return db
}

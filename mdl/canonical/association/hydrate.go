// SPDX-License-Identifier: Apache-2.0

package association

import (
	"github.com/mendixlabs/mxcli/mdl/canonical"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// Hydrate converts a gen-typed Association to an AssociationModel.
// ctx.ModuleName is the owning module (gen entity does not carry it).
// ctx.EntityNames maps element ID strings to simple entity names;
// when an ID is missing, the ID itself is used as a fallback and a Warning
// is emitted so callers can log the gap.
func Hydrate(ctx canonical.HydrateCtx, a *genDm.Association) (*AssociationModel, []canonical.Warning, error) {
	var warns []canonical.Warning

	fromName := resolveEntityName(ctx, string(a.ParentRefID()), &warns)
	toName := resolveEntityName(ctx, string(a.ChildRefID()), &warns)

	m := &AssociationModel{
		Name:           QualifiedName{Module: ctx.ModuleName, Name: a.Name()},
		From:           QualifiedName{Module: ctx.ModuleName, Name: fromName},
		To:             QualifiedName{Module: ctx.ModuleName, Name: toName},
		Type:           hydrateType(a.Type()),
		Owner:          hydrateOwner(a.Owner()),
		Storage:        hydrateStorage(a.StorageFormat()),
		DeleteBehavior: hydrateDelete(a),
		Documentation:  a.Documentation(),
	}
	return m, warns, nil
}

func resolveEntityName(ctx canonical.HydrateCtx, id string, warns *[]canonical.Warning) string {
	if name, ok := ctx.EntityNames[id]; ok {
		return name
	}
	*warns = append(*warns, canonical.Warning{
		Field:   "EntityID",
		Message: "entity ID " + id + " not found in EntityNames map; using ID as fallback",
	})
	return id
}

func hydrateType(t string) AssocType {
	if t == "ReferenceSet" {
		return AssocReferenceSet
	}
	return AssocReference
}

func hydrateOwner(o string) OwnerType {
	if o == "Both" {
		return OwnerBoth
	}
	return OwnerDefault
}

func hydrateStorage(s string) StorageType {
	if s == "AssociationStorageColumn" {
		return StorageColumn
	}
	return StorageTable
}

func hydrateDelete(a *genDm.Association) DeleteBehaviorType {
	dbe := a.DeleteBehavior()
	if dbe == nil {
		return DeleteKeepReferences
	}
	db, ok := dbe.(*genDm.AssociationDeleteBehavior)
	if !ok {
		return DeleteKeepReferences
	}
	switch db.ChildDeleteBehavior() {
	case "DeleteMeAndReferences":
		return DeleteCascade
	case "DeleteBoth":
		return DeleteBoth
	case "KeepParentDeleteChild":
		return DeleteKeepParentDeleteChild
	case "KeepChildDeleteParent":
		return DeleteKeepChildDeleteParent
	case "DeleteIfNoReferences":
		return DeleteIfNoReferences
	default:
		return DeleteKeepReferences
	}
}

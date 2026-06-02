// SPDX-License-Identifier: Apache-2.0

// Package association implements the canonical model for Mendix associations.
package association

// AssociationModel is the canonical in-memory representation of a Mendix association.
// It captures exactly what MDL can express — not the complete BSON state.
type AssociationModel struct {
	Name           QualifiedName
	From           QualifiedName // ParentPointer — FK owner entity
	To             QualifiedName // ChildPointer — referenced entity
	Type           AssocType
	Owner          OwnerType
	Storage        StorageType
	DeleteBehavior DeleteBehaviorType
	Documentation  string
}

// QualifiedName is a module-qualified identifier.
type QualifiedName struct {
	Module string
	Name   string
}

func (q QualifiedName) String() string {
	if q.Module == "" {
		return q.Name
	}
	return q.Module + "." + q.Name
}

// AssocType mirrors ast.AssociationType.
type AssocType int

const (
	AssocReference AssocType = iota
	AssocReferenceSet
)

// OwnerType mirrors ast.OwnerType.
type OwnerType int

const (
	OwnerDefault OwnerType = iota
	OwnerBoth
)

// StorageType mirrors ast.StorageType.
type StorageType int

const (
	StorageTable StorageType = iota
	StorageColumn
)

// DeleteBehaviorType mirrors ast.DeleteBehavior.
type DeleteBehaviorType int

const (
	DeleteKeepReferences DeleteBehaviorType = iota
	DeleteCascade
	DeleteBoth
	DeleteKeepParentDeleteChild
	DeleteKeepChildDeleteParent
	DeleteIfNoReferences
)

// SPDX-License-Identifier: Apache-2.0

package association

import "github.com/mendixlabs/mxcli/mdl/ast"

// Lift converts a parsed CREATE ASSOCIATION AST statement to an AssociationModel.
// It is a pure function: no backend access, no side effects.
func Lift(s *ast.CreateAssociationStmt) *AssociationModel {
	return &AssociationModel{
		Name:           QualifiedName{Module: s.Name.Module, Name: s.Name.Name},
		From:           QualifiedName{Module: s.Parent.Module, Name: s.Parent.Name},
		To:             QualifiedName{Module: s.Child.Module, Name: s.Child.Name},
		Type:           liftType(s.Type),
		Owner:          liftOwner(s.Owner),
		Storage:        liftStorage(s.Storage),
		DeleteBehavior: liftDelete(s.DeleteBehavior),
		Documentation:  s.Documentation,
	}
}

func liftType(t ast.AssociationType) AssocType {
	if t == ast.AssocReferenceSet {
		return AssocReferenceSet
	}
	return AssocReference
}

func liftOwner(o ast.OwnerType) OwnerType {
	if o == ast.OwnerBoth {
		return OwnerBoth
	}
	return OwnerDefault
}

func liftStorage(s ast.StorageType) StorageType {
	if s == ast.StorageColumn {
		return StorageColumn
	}
	return StorageTable
}

func liftDelete(d ast.DeleteBehavior) DeleteBehaviorType {
	switch d {
	case ast.DeleteCascade:
		return DeleteCascade
	case ast.DeleteBoth:
		return DeleteBoth
	case ast.DeleteKeepParentDeleteChild:
		return DeleteKeepParentDeleteChild
	case ast.DeleteKeepChildDeleteParent:
		return DeleteKeepChildDeleteParent
	case ast.DeleteIfNoReferences:
		return DeleteIfNoReferences
	default:
		return DeleteKeepReferences
	}
}

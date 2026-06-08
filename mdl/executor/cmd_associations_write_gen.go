// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.4 D1.c: AST → gen Entity / Generalization / Association
// builders. Continues cmd_entities_write_gen.go (D1.a + D1.b).

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// astToEntityGen builds a gen-typed *Entity from a CREATE ENTITY AST.
//
// Pseudo-types (AutoOwner / AutoChangedBy / AutoCreatedDate /
// AutoChangedDate) flip the corresponding flag on the entity's
// NoGeneralization element rather than producing real attributes,
// matching legacy execCreateEntity semantics.
func astToEntityGen(s *ast.CreateEntityStmt) *genDm.Entity {
	if s == nil {
		return nil
	}
	entity := genDm.NewEntity()
	entity.SetName(s.Name.Name)
	entity.SetDocumentation(s.Documentation)
	if s.Position != nil {
		entity.SetLocation(layoutPos(s.Position.X, s.Position.Y))
	}

	persistable := s.Kind != ast.EntityNonPersistent
	gen := astToGeneralizationGen(s, persistable)
	entity.SetGeneralization(gen)

	noGen, hasNoGen := gen.(*genDm.NoGeneralization)
	if hasNoGen {
		// Explicitly set all 4 system member flags (Mendix requires all 4 present in BSON
		// even when false; absent fields cause CE0161 for XPath constraints).
		enabled := make(map[string]bool, len(s.SystemMembers))
		for _, m := range s.SystemMembers {
			enabled[m] = true
		}
		noGen.SetHasOwner(enabled["owner"])
		noGen.SetHasCreatedDate(enabled["createdDate"])
		noGen.SetHasChangedDate(enabled["changedDate"])
		noGen.SetHasChangedBy(enabled["changedBy"])
	}
	attrNameToID := make(map[string]model.ID)
	for _, a := range s.Attributes {
		ac := a
		if ac.Type.Kind == ast.TypeBoolean && !ac.HasDefault {
			ac.HasDefault = true
			ac.DefaultValue = false
		}
		attr := astToAttributeGen(&ac)
		if attr != nil {
			// Pre-assign a UUID so IndexedAttribute.AttributePointer gets a
			// real binary ID when astToIndexGen runs below. Without this,
			// attr.ID() is "" and the index stores an empty string, which
			// Mendix rejects with InvalidCastException (String→Byte[]).
			// assignEntityIDsGen later skips elements with non-empty IDs.
			attr.SetID(element.ID(modelsdkmpr.GenerateID()))
			entity.AddAttributes(attr)
			attrNameToID[ac.Name] = model.ID(attr.ID())
			for _, vr := range astToValidationRulesGen(&ac, s.Name.String()) {
				entity.AddValidationRules(vr)
			}
		}
	}
	for _, idx := range s.Indexes {
		if genIdx := astToIndexGen(&idx, attrNameToID); genIdx != nil && len(genIdx.AttributesItems()) > 0 {
			entity.AddIndexes(genIdx)
		}
	}
	for _, eh := range s.EventHandlers {
		if genEh := astToEventHandlerGen(&eh); genEh != nil {
			entity.AddEventHandlers(genEh)
		}
	}
	return entity
}

// astToGeneralizationGen returns the entity's Generalization element:
// *Generalization when the AST signals "extends Module.Parent", else
// *NoGeneralization with Persistable set per the entity kind.
func astToGeneralizationGen(s *ast.CreateEntityStmt, persistable bool) element.Element {
	if s != nil && s.Generalization != nil && s.Generalization.String() != "" {
		g := genDm.NewGeneralization()
		g.SetGeneralizationQualifiedName(s.Generalization.String())
		return g
	}
	noGen := genDm.NewNoGeneralization()
	noGen.SetPersistable(persistable)
	return noGen
}

// astToAssociationGen builds a gen-typed *Association from a CREATE
// ASSOCIATION AST. Caller resolves from/to entity IDs before invoking.
//
// Per CLAUDE.md "Association Parent/Child Pointer Semantics":
// ParentRefID = FROM entity (FK owner), ChildRefID = TO entity.
func astToAssociationGen(s *ast.CreateAssociationStmt, fromID, toID element.ID) *genDm.Association {
	if s == nil {
		return nil
	}
	a := genDm.NewAssociation()
	a.SetName(s.Name.Name)
	a.SetParentID(fromID)
	a.SetChildID(toID)
	a.SetType(astAssociationTypeStringGen(s))
	a.SetOwner(astAssociationOwnerStringGen(s))
	a.SetStorageFormat(astAssociationStorageStringGen(s))
	a.SetDeleteBehavior(astAssociationDeleteBehaviorGen(s))
	if s.Documentation != "" {
		a.SetDocumentation(s.Documentation)
	}
	return a
}

func astAssociationTypeStringGen(s *ast.CreateAssociationStmt) string {
	if s == nil {
		return "Reference"
	}
	return astAssocTypeStr(s.Type)
}

func astAssociationOwnerStringGen(s *ast.CreateAssociationStmt) string {
	if s == nil {
		return "Default"
	}
	return astOwnerStr(s.Owner)
}

func astAssociationStorageStringGen(s *ast.CreateAssociationStmt) string {
	if s == nil {
		return "Table"
	}
	if s.Storage == ast.StorageColumn {
		return "Column"
	}
	return "Table"
}

func astAssociationDeleteBehaviorGen(s *ast.CreateAssociationStmt) element.Element {
	dbe := genDm.NewAssociationDeleteBehavior()
	dbe.SetParentDeleteBehavior("DeleteMeButKeepReferences")
	switch {
	case s == nil:
		dbe.SetChildDeleteBehavior("DeleteMeButKeepReferences")
	case s.DeleteBehavior == ast.DeleteCascade:
		dbe.SetChildDeleteBehavior("DeleteMeAndReferences")
	case s.DeleteBehavior == ast.DeleteIfNoReferences:
		dbe.SetChildDeleteBehavior("DeleteMeIfNoReferences")
	default:
		dbe.SetChildDeleteBehavior("DeleteMeButKeepReferences")
	}
	return dbe
}

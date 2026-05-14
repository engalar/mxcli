// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.6.4: standalone (no flowBuilder receiver) reverse-lookup
// helpers extracted from cmd_microflows_builder_actions.go (deleted).
//
// `lookupAssociation` and `entityIsSubtypeOf` only consume the
// backend interface — they do not touch sdk/microflows types — but
// the originals were methods on `flowBuilder` (the legacy sdk-typed
// builder). The gen-typed flow builders (flowbuilder_gen.go,
// flowbuilder_actions_retrieve_gen.go) used to fabricate a throwaway
// `flowBuilder{}` just to call them; this file inlines the logic so
// the legacy builder can be deleted.

package executor

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
)

// assocLookupResult holds resolved association metadata.
type assocLookupResult struct {
	Type              domainmodel.AssociationType
	Owner             domainmodel.AssociationOwner
	parentEntityQN    string // Qualified name of the parent (FROM/owner) entity
	childEntityQN     string // Qualified name of the child (TO/referenced) entity
	parentPersistable bool
	childPersistable  bool
}

// lookupAssociationGen finds an association by module and name, returning
// its type and the qualified names of its parent and child entities.
// Returns nil if the association cannot be found (e.g., backend is nil
// or module doesn't exist). Pure read — no mutation.
func lookupAssociationGen(b backend.FullBackend, moduleName, assocName string) *assocLookupResult {
	if b == nil {
		return nil
	}
	mod, err := b.GetModuleByName(moduleName)
	if err != nil || mod == nil {
		return nil
	}
	dm, err := b.GetDomainModel(mod.ID)
	if err != nil || dm == nil {
		return nil
	}

	entityNames := make(map[model.ID]string, len(dm.Entities))
	entityPersistable := make(map[model.ID]bool, len(dm.Entities))
	for _, e := range dm.Entities {
		entityNames[e.ID] = moduleName + "." + e.Name
		entityPersistable[e.ID] = e.Persistable
	}

	for _, a := range dm.Associations {
		if a.Name == assocName {
			return &assocLookupResult{
				Type:              a.Type,
				Owner:             a.Owner,
				parentEntityQN:    entityNames[a.ParentID],
				childEntityQN:     entityNames[a.ChildID],
				parentPersistable: entityPersistable[a.ParentID],
				childPersistable:  entityPersistable[a.ChildID],
			}
		}
	}
	return nil
}

// isEntityGen checks whether a (module, entity) pair refers to an
// entity in the domain model. Used by the gen builder to disambiguate
// `Module.Name` references that the parser parses as TypeEnumeration
// — when the name actually targets an entity we rewrite the type.
func isEntityGen(b backend.FullBackend, moduleName, entityName string) bool {
	if b == nil {
		return false
	}
	mod, err := b.GetModuleByName(moduleName)
	if err != nil || mod == nil {
		return false
	}
	dm, err := b.GetDomainModel(mod.ID)
	if err != nil || dm == nil {
		return false
	}
	for _, e := range dm.Entities {
		if e.Name == entityName {
			return true
		}
	}
	return false
}

// lookupEnumRefGen returns the enumeration qualified name (e.g.,
// "MyModule.ENUM_Status") for an attribute if it is an enumeration
// type. Returns "" if the attribute is not an enumeration or if the
// domain model is not available.
func lookupEnumRefGen(b backend.FullBackend, entityQN, attrName string) string {
	if b == nil || entityQN == "" || attrName == "" {
		return ""
	}
	parts := strings.SplitN(entityQN, ".", 2)
	if len(parts) != 2 {
		return ""
	}
	mod, err := b.GetModuleByName(parts[0])
	if err != nil || mod == nil {
		return ""
	}
	dm, err := b.GetDomainModel(mod.ID)
	if err != nil || dm == nil {
		return ""
	}
	for _, entity := range dm.Entities {
		if entity.Name == parts[1] {
			for _, attr := range entity.Attributes {
				if attr.Name == attrName {
					if enumType, ok := attr.Type.(*domainmodel.EnumerationAttributeType); ok {
						return enumType.EnumerationRef
					}
					return ""
				}
			}
			return ""
		}
	}
	return ""
}

// resolveAttributeInEntityHierarchyGen walks a generalization chain
// looking for an attribute named attrName, returning its fully
// qualified name (Module.Entity.Attribute) on the first match.
// Pure read.
func resolveAttributeInEntityHierarchyGen(b backend.FullBackend, entityQN, attrName string) (string, bool) {
	if b == nil || entityQN == "" || attrName == "" {
		return "", false
	}
	seen := make(map[string]bool)
	for currentQN := entityQN; currentQN != ""; {
		if seen[currentQN] {
			return "", false
		}
		seen[currentQN] = true

		parts := strings.SplitN(currentQN, ".", 2)
		if len(parts) != 2 {
			return "", false
		}
		mod, err := b.GetModuleByName(parts[0])
		if err != nil || mod == nil {
			return "", false
		}
		dm, err := b.GetDomainModel(mod.ID)
		if err != nil || dm == nil {
			return "", false
		}
		entity := dm.FindEntityByName(parts[1])
		if entity == nil {
			return "", false
		}
		for _, attr := range entity.Attributes {
			if attr != nil && attr.Name == attrName {
				return currentQN + "." + attrName, true
			}
		}
		currentQN = entity.GeneralizationRef
	}
	return "", false
}

// resolveMemberChangeGenStandalone classifies a member name as
// attribute or association, returning the matching qualified name
// pair (exactly one of attributeQN / associationQN set on success).
//
// Mirrors the legacy `flowBuilder.resolveMemberChange` algorithm:
//  1. memberName is `Module.Assoc` or bare; the qualifier (if any)
//     names the OWNING module of the association.
//  2. Lookup walks the authored module's domain model first.
//  3. Cross-module association table is consulted as fallback.
//  4. Two-dot author-qualified names like `Module.Entity.Attr` are
//     preserved as attribute QNs even when entity metadata is missing.
//
// Pure read; no flowBuilder receiver needed.
func resolveMemberChangeGenStandalone(b backend.FullBackend, memberName, entityQN string) resolvedMemberChange {
	if entityQN == "" {
		return memberChangeFallback(memberName, "")
	}

	parts := strings.SplitN(entityQN, ".", 2)
	if len(parts) != 2 {
		return resolvedMemberChange{attributeQN: entityQN + "." + memberName}
	}
	moduleName := parts[0]

	bareName := memberName
	qualifiedName := memberName
	lookupModule := moduleName
	if dot := strings.Index(memberName, "."); dot >= 0 {
		lookupModule = memberName[:dot]
		bareName = memberName[dot+1:]
	} else {
		qualifiedName = moduleName + "." + memberName
	}

	if b != nil {
		if mod, err := b.GetModuleByName(lookupModule); err == nil && mod != nil {
			if dm, err := b.GetDomainModel(mod.ID); err == nil && dm != nil {
				for _, a := range dm.Associations {
					if a.Name == bareName {
						return resolvedMemberChange{associationQN: qualifiedName}
					}
				}
				for _, a := range dm.CrossAssociations {
					if a.Name == bareName {
						return resolvedMemberChange{associationQN: qualifiedName}
					}
				}
				if strings.Contains(memberName, ".") {
					return resolvedMemberChange{attributeQN: memberName}
				}
				if attrQN, ok := resolveAttributeInEntityHierarchyGen(b, entityQN, memberName); ok {
					return resolvedMemberChange{attributeQN: attrQN}
				}
				return resolvedMemberChange{attributeQN: entityQN + "." + memberName}
			}
		}
	}

	return memberChangeFallback(memberName, entityQN)
}

// memberChangeFallback preserves the authored member-name shape when
// entity metadata is unavailable.
//
//   - 0 dots  => bare attribute name. If entityQN is known, qualify
//     as `Module.Entity.Attribute`; otherwise preserve bare.
//   - 1 dot   => association qualified by module (`Module.Association`).
//   - >=2 dots => fully qualified attribute (`Module.Entity.Attribute`).
func memberChangeFallback(memberName, entityQN string) resolvedMemberChange {
	if memberName == "" {
		return resolvedMemberChange{}
	}
	switch strings.Count(memberName, ".") {
	case 0:
		if entityQN == "" {
			return resolvedMemberChange{attributeQN: memberName}
		}
		return resolvedMemberChange{attributeQN: entityQN + "." + memberName}
	case 1:
		return resolvedMemberChange{associationQN: memberName}
	default:
		return resolvedMemberChange{attributeQN: memberName}
	}
}

// entityIsSubtypeOfGen walks the generalization chain from candidateQN
// upward, returning true if it ever reaches ancestorQN. Returns false
// on missing modules / entities / dangling refs (defensive).
func entityIsSubtypeOfGen(b backend.FullBackend, candidateQN, ancestorQN string) bool {
	if candidateQN == "" || ancestorQN == "" {
		return false
	}
	if candidateQN == ancestorQN {
		return true
	}
	if b == nil {
		return false
	}
	seen := make(map[string]bool)
	for currentQN := candidateQN; currentQN != ""; {
		if seen[currentQN] {
			return false
		}
		seen[currentQN] = true
		if currentQN == ancestorQN {
			return true
		}
		parts := strings.SplitN(currentQN, ".", 2)
		if len(parts) != 2 {
			return false
		}
		mod, err := b.GetModuleByName(parts[0])
		if err != nil || mod == nil {
			return false
		}
		dm, err := b.GetDomainModel(mod.ID)
		if err != nil || dm == nil {
			return false
		}
		entity := dm.FindEntityByName(parts[1])
		if entity == nil {
			return false
		}
		currentQN = entity.GeneralizationRef
	}
	return false
}

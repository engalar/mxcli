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

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// assocLookupResult holds resolved association metadata.
type assocLookupResult struct {
	Type              string
	Owner             string
	parentEntityQN    string // Qualified name of the parent (FROM/owner) entity
	childEntityQN     string // Qualified name of the child (TO/referenced) entity
	parentPersistable bool
	childPersistable  bool
}

// lookupAssociationGen finds an association by module and name, returning
// its type and the qualified names of its parent and child entities.
// Returns nil if the association cannot be found (e.g., backend is nil
// or module doesn't exist). Pure read — no mutation.
func lookupAssociationGen(lister backend.ModuleLister, reader backend.DomainModelReader, moduleName, assocName string) *assocLookupResult {
	if lister == nil || reader == nil {
		return nil
	}
	mod, err := lister.GetModuleByName(moduleName)
	if err != nil || mod == nil {
		return nil
	}
	dm, err := reader.GetDomainModelGen(mod.ID)
	if err != nil || dm == nil {
		return nil
	}

	entityNames := make(map[model.ID]string, len(dm.EntitiesItems()))
	entityPersistable := make(map[model.ID]bool, len(dm.EntitiesItems()))
	for _, item := range dm.EntitiesItems() {
		e, ok := item.(*genDm.Entity)
		if !ok {
			continue
		}
		entityNames[model.ID(e.ID())] = moduleName + "." + e.Name()
		entityPersistable[model.ID(e.ID())] = entityPersistableGen(e)
	}

	for _, item := range dm.AssociationsItems() {
		a, ok := item.(*genDm.Association)
		if !ok {
			continue
		}
		if a.Name() == assocName {
			return &assocLookupResult{
				Type:              a.Type(),
				Owner:             a.Owner(),
				parentEntityQN:    entityNames[model.ID(a.ParentRefID())],
				childEntityQN:     entityNames[model.ID(a.ChildRefID())],
				parentPersistable: entityPersistable[model.ID(a.ParentRefID())],
				childPersistable:  entityPersistable[model.ID(a.ChildRefID())],
			}
		}
	}
	return nil
}

// isEntityGen checks whether a (module, entity) pair refers to an
// entity in the domain model. Used by the gen builder to disambiguate
// `Module.Name` references that the parser parses as TypeEnumeration
// — when the name actually targets an entity we rewrite the type.
func isEntityGen(lister backend.ModuleLister, reader backend.DomainModelReader, moduleName, entityName string) bool {
	if lister == nil || reader == nil {
		return false
	}
	mod, err := lister.GetModuleByName(moduleName)
	if err != nil || mod == nil {
		return false
	}
	dm, err := reader.GetDomainModelGen(mod.ID)
	if err != nil || dm == nil {
		return false
	}
	for _, item := range dm.EntitiesItems() {
		e, ok := item.(*genDm.Entity)
		if ok && e.Name() == entityName {
			return true
		}
	}
	return false
}

// resolveAmbiguousDataType resolves the TypeEnumeration vs TypeEntity
// ambiguity for bare qualified names in both visitor contexts:
//
//   - In attribute context (buildDataType): visitor returns TypeEnumeration;
//     if the name is actually an entity, upgrade to TypeEntity.
//   - In microflow parameter context (buildMicroflowDataType): visitor always
//     returns TypeEntity for bare QN; if the name is NOT a known entity,
//     downgrade to TypeEnumeration (the QN must be an enum, constant, or
//     similar non-entity type).
//
// Safe to call with any DataType; non-ambiguous kinds are returned unchanged.
// Uses ctx to leverage the per-module DomainModel cache (getDomainModelGenCached)
// so that consecutive calls for the same module share one backend fetch.
func resolveAmbiguousDataType(ctx *ExecContext, lister backend.ModuleLister, reader backend.DomainModelReader, dt ast.DataType) ast.DataType {
	switch dt.Kind {
	case ast.TypeEnumeration:
		if dt.EnumRef != nil && lister != nil && reader != nil {
			if isEntityGenCached(ctx, lister, reader, dt.EnumRef.Module, dt.EnumRef.Name) {
				return ast.DataType{Kind: ast.TypeEntity, EntityRef: dt.EnumRef}
			}
		}
	case ast.TypeEntity:
		if dt.EntityRef != nil && lister != nil && reader != nil {
			if !isEntityGenCached(ctx, lister, reader, dt.EntityRef.Module, dt.EntityRef.Name) {
				return ast.DataType{Kind: ast.TypeEnumeration, EnumRef: dt.EntityRef}
			}
		}
	}
	return dt
}

// isEntityGenCached is the cache-aware variant of isEntityGen.
// It uses getDomainModelGenCached(ctx, moduleID) so that multiple
// lookups against the same module hit the in-memory cache instead
// of decoding BSON from the backend each time.
func isEntityGenCached(ctx *ExecContext, lister backend.ModuleLister, reader backend.DomainModelReader, moduleName, entityName string) bool {
	if ctx == nil {
		return isEntityGen(lister, reader, moduleName, entityName)
	}
	mod, err := lister.GetModuleByName(moduleName)
	if err != nil || mod == nil {
		return false
	}
	dm, err := getDomainModelGenCached(ctx, mod.ID)
	if err != nil || dm == nil {
		return false
	}
	for _, item := range dm.EntitiesItems() {
		e, ok := item.(*genDm.Entity)
		if ok && e.Name() == entityName {
			return true
		}
	}
	return false
}

// lookupEnumRefGen returns the enumeration qualified name (e.g.,
// "MyModule.ENUM_Status") for an attribute if it is an enumeration
// type. Returns "" if the attribute is not an enumeration or if the
// domain model is not available.
func lookupEnumRefGen(lister backend.ModuleLister, reader backend.DomainModelReader, entityQN, attrName string) string {
	if lister == nil || reader == nil || entityQN == "" || attrName == "" {
		return ""
	}
	parts := strings.SplitN(entityQN, ".", 2)
	if len(parts) != 2 {
		return ""
	}
	mod, err := lister.GetModuleByName(parts[0])
	if err != nil || mod == nil {
		return ""
	}
	dm, err := reader.GetDomainModelGen(mod.ID)
	if err != nil || dm == nil {
		return ""
	}
	for _, item := range dm.EntitiesItems() {
		entity, ok := item.(*genDm.Entity)
		if !ok || entity.Name() != parts[1] {
			continue
		}
		for _, attrItem := range entity.AttributesItems() {
			attr, ok := attrItem.(*genDm.Attribute)
			if !ok || attr.Name() != attrName {
				continue
			}
			if enumType, ok := attr.Type().(*genDm.EnumerationAttributeType); ok {
				return enumType.EnumerationQualifiedName()
			}
			return ""
		}
		return ""
	}
	return ""
}

// resolveAttributeInEntityHierarchyGen walks a generalization chain
// looking for an attribute named attrName, returning its fully
// qualified name (Module.Entity.Attribute) on the first match.
// Pure read.
func resolveAttributeInEntityHierarchyGen(lister backend.ModuleLister, reader backend.DomainModelReader, entityQN, attrName string, dm *genDm.DomainModel) (string, bool) {
	if lister == nil || reader == nil || entityQN == "" || attrName == "" {
		return "", false
	}
	seen := make(map[string]bool)
	firstIteration := true
	for currentQN := entityQN; currentQN != ""; {
		if seen[currentQN] {
			return "", false
		}
		seen[currentQN] = true

		parts := strings.SplitN(currentQN, ".", 2)
		if len(parts) != 2 {
			return "", false
		}

		// Use the pre-fetched dm for the first iteration (avoids the
		// double-fetch bug when called from resolveMemberChangeGenStandalone).
		// For subsequent generalization-chain iterations that may cross
		// modules, fall through to the backend.
		var entity *genDm.Entity
		if firstIteration && dm != nil {
			entity = findEntityInDomainModelGen(dm, parts[1])
		} else {
			mod, err := lister.GetModuleByName(parts[0])
			if err != nil || mod == nil {
				return "", false
			}
			resolvedDM, err := reader.GetDomainModelGen(mod.ID)
			if err != nil || resolvedDM == nil {
				return "", false
			}
			entity = findEntityInDomainModelGen(resolvedDM, parts[1])
		}
		firstIteration = false

		if entity == nil {
			return "", false
		}
		for _, item := range entity.AttributesItems() {
			attr, ok := item.(*genDm.Attribute)
			if ok && attr.Name() == attrName {
				return currentQN + "." + attrName, true
			}
		}
		currentQN = entityGeneralizationQNGen(entity)
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
// dm is an optional pre-fetched domain model. When non-nil the function
// uses it directly instead of calling reader.GetDomainModelGen, enabling
// callers to cache the DM across multiple member-change resolutions.
// Pass nil for the original fetch-from-backend behaviour.
//
// Pure read; no flowBuilder receiver needed.
func resolveMemberChangeGenStandalone(lister backend.ModuleLister, reader backend.DomainModelReader, memberName, entityQN string, dm *genDm.DomainModel) resolvedMemberChange {
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

	if lister != nil && reader != nil {
		if mod, err := lister.GetModuleByName(lookupModule); err == nil && mod != nil {
			// Use caller-provided DM when available (avoids re-fetching
			// the same domain model for every member change).
			if dm == nil {
				if fetchedDM, err := reader.GetDomainModelGen(mod.ID); err == nil && fetchedDM != nil {
					dm = fetchedDM
				}
			}
			if dm != nil {
				for _, item := range dm.AssociationsItems() {
					a, ok := item.(*genDm.Association)
					if ok && a.Name() == bareName {
						return resolvedMemberChange{associationQN: qualifiedName}
					}
				}
				for _, item := range dm.CrossAssociationsItems() {
					a, ok := item.(*genDm.CrossAssociation)
					if ok && a.Name() == bareName {
						return resolvedMemberChange{associationQN: qualifiedName}
					}
				}
				// Two-or-more dots means Module.Entity.Attr — preserve as
				// attribute even when entity metadata is missing.
				if strings.Count(memberName, ".") >= 2 {
					return resolvedMemberChange{attributeQN: memberName}
				}
				// Pass the already-fetched dm to avoid a second backend
				// call inside resolveAttributeInEntityHierarchyGen.
				if attrQN, ok := resolveAttributeInEntityHierarchyGen(lister, reader, entityQN, memberName, dm); ok {
					return resolvedMemberChange{attributeQN: attrQN}
				}
				// Neither attribute nor association found in the domain model
				// (common when the reader cache is stale after a write). Fall
				// back to the dot-count heuristic: a 1-dot name is a
				// Module.Association, a 0-dot name is a bare attribute.
				return memberChangeFallback(memberName, entityQN)
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
func entityIsSubtypeOfGen(lister backend.ModuleLister, reader backend.DomainModelReader, candidateQN, ancestorQN string) bool {
	if candidateQN == "" || ancestorQN == "" {
		return false
	}
	if candidateQN == ancestorQN {
		return true
	}
	if lister == nil || reader == nil {
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
		mod, err := lister.GetModuleByName(parts[0])
		if err != nil || mod == nil {
			return false
		}
		dm, err := reader.GetDomainModelGen(mod.ID)
		if err != nil || dm == nil {
			return false
		}
		entity := findEntityInDomainModelGen(dm, parts[1])
		if entity == nil {
			return false
		}
		currentQN = entityGeneralizationQNGen(entity)
	}
	return false
}

func entityPersistableGen(entity *genDm.Entity) bool {
	if entity == nil {
		return false
	}
	if g, ok := entity.Generalization().(*genDm.NoGeneralization); ok {
		return g.Persistable()
	}
	return true
}

func findEntityInDomainModelGen(dm *genDm.DomainModel, name string) *genDm.Entity {
	if dm == nil {
		return nil
	}
	for _, item := range dm.EntitiesItems() {
		entity, ok := item.(*genDm.Entity)
		if ok && entity.Name() == name {
			return entity
		}
	}
	return nil
}

// isNonPersistentEntity reports whether qualifiedName refers to a non-persistent entity.
// If fb.nonPersistentEntities is already populated (e.g. by tests), it is used directly.
// Otherwise it is lazily loaded from fb.domainModelReader; returns false when reader is nil.
func (fb *flowBuilderGen) isNonPersistentEntity(qualifiedName string) bool {
	if fb.nonPersistentEntities == nil {
		if fb.domainModelReader == nil {
			return false
		}
		fb.nonPersistentEntities = loadNonPersistentEntitySet(fb.domainModelReader, fb.hierarchy)
	}
	return fb.nonPersistentEntities[qualifiedName]
}

// loadNonPersistentEntitySet builds the set of non-persistent entity qualified names
// by walking all domain models via the backend.
func loadNonPersistentEntitySet(reader backend.DomainModelReader, h *ContainerHierarchy) map[string]bool {
	result := make(map[string]bool)
	dms, err := reader.ListDomainModelsGen()
	if err != nil || h == nil {
		return result
	}
	for _, dm := range dms {
		if dm == nil {
			continue
		}
		modName := h.GetModuleName(h.FindModuleID(model.ID(dm.ID())))
		if modName == "" {
			continue
		}
		for _, entityElem := range dm.EntitiesItems() {
			ent, ok := entityElem.(*genDm.Entity)
			if !ok {
				continue
			}
			if !entityPersistableGen(ent) {
				result[modName+"."+ent.Name()] = true
			}
		}
	}
	return result
}

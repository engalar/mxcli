// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDM "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// rolesMatch reports whether the two role-name slices contain the same set of
// elements (order-independent, no duplicates expected). Used to locate an
// existing AccessRule by its AllowedModuleRoles list.
func rolesMatch(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]struct{}, len(a))
	for _, r := range a {
		seen[r] = struct{}{}
	}
	for _, r := range b {
		if _, ok := seen[r]; !ok {
			return false
		}
	}
	return true
}

// addEntityAccessRuleViaModelsdk creates a fresh AccessRule on the named
// entity through the gen-native msdkWrite path. The encoder writes the
// AllowedModuleRoles BSON key (verified against Studio Pro behaviour).
func (b *MprBackend) addEntityAccessRuleViaModelsdk(unitID model.ID, entityName string, roleNames []string,
	allowCreate, allowDelete bool, defaultMemberAccess, xpathConstraint string,
	memberAccesses []types.EntityMemberAccess) error {
	return b.msdkWrite(unitID, func(elem element.Element) error {
		dm, ok := elem.(*genDM.DomainModel)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *DomainModel)", elem)
		}
		for _, e := range dm.EntitiesItems() {
			ent, ok := e.(*genDM.Entity)
			if !ok || ent.Name() != entityName {
				continue
			}
			rule := genDM.NewAccessRule()
			rule.SetAllowCreate(allowCreate)
			rule.SetAllowDelete(allowDelete)
			rule.SetDefaultMemberAccessRights(defaultMemberAccess)
			rule.SetXPathConstraint(xpathConstraint)
			// Documentation and XPathConstraintCaption must always be present (even
			// as empty strings) — Studio Pro's CE0066 checker requires these fields.
			rule.SetDocumentation("")
			rule.SetXPathConstraintCaption("")
			rule.SetModuleRolesQualifiedNames(roleNames)
			for _, ma := range memberAccesses {
				genMA := genDM.NewMemberAccess()
				genMA.SetAccessRights(ma.AccessRights)
				if ma.AttributeRef != "" {
					genMA.SetAttributeQualifiedName(ma.AttributeRef)
				} else if ma.AssociationRef != "" {
					genMA.SetAssociationQualifiedName(ma.AssociationRef)
				}
				rule.AddMemberAccesses(genMA)
			}
			ent.AddAccessRules(rule)
			return nil
		}
		return fmt.Errorf("entity not found: %s", entityName)
	})
}

// removeEntityAccessRuleViaModelsdk removes every AccessRule on the named
// entity whose AllowedModuleRoles list matches roleNames (order-independent).
// Iterates in reverse so RemoveAccessRules indices stay valid as entries drop.
func (b *MprBackend) removeEntityAccessRuleViaModelsdk(unitID model.ID, entityName string, roleNames []string) (int, error) {
	removed := 0
	err := b.msdkWrite(unitID, func(elem element.Element) error {
		dm, ok := elem.(*genDM.DomainModel)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *DomainModel)", elem)
		}
		for _, e := range dm.EntitiesItems() {
			ent, ok := e.(*genDM.Entity)
			if !ok || ent.Name() != entityName {
				continue
			}
			rules := ent.AccessRulesItems()
			for i := len(rules) - 1; i >= 0; i-- {
				rule, ok := rules[i].(*genDM.AccessRule)
				if !ok {
					continue
				}
				if rolesMatch(rule.ModuleRolesQualifiedNames(), roleNames) {
					ent.RemoveAccessRules(i)
					removed++
				}
			}
		}
		return nil
	})
	return removed, err
}

// removeRoleFromAllEntitiesViaModelsdk strips roleName from the
// AllowedModuleRoles list of every AccessRule on every entity in the domain
// model. Returns the number of rules whose role list shrank.
func (b *MprBackend) removeRoleFromAllEntitiesViaModelsdk(unitID model.ID, roleName string) (int, error) {
	removed := 0
	err := b.msdkWrite(unitID, func(elem element.Element) error {
		dm, ok := elem.(*genDM.DomainModel)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *DomainModel)", elem)
		}
		for _, e := range dm.EntitiesItems() {
			ent, ok := e.(*genDM.Entity)
			if !ok {
				continue
			}
			for _, r := range ent.AccessRulesItems() {
				rule, ok := r.(*genDM.AccessRule)
				if !ok {
					continue
				}
				roles := rule.ModuleRolesQualifiedNames()
				newRoles := make([]string, 0, len(roles))
				for _, ro := range roles {
					if ro != roleName {
						newRoles = append(newRoles, ro)
					}
				}
				if len(newRoles) != len(roles) {
					rule.SetModuleRolesQualifiedNames(newRoles)
					removed++
				}
			}
		}
		return nil
	})
	return removed, err
}

// revokeEntityMemberAccessViaModelsdk performs a partial revoke on every
// AccessRule on the named entity whose AllowedModuleRoles list matches
// roleNames. Mirrors sdk/mpr.Writer.RevokeEntityMemberAccess semantics:
// boolean flags downgrade AllowCreate/AllowDelete; RevokeReadAll forces
// DefaultMemberAccessRights to "NoAccess"; RevokeWriteAll forces "ReadOnly";
// per-member revokes downgrade matching MemberAccess entries.
func (b *MprBackend) revokeEntityMemberAccessViaModelsdk(unitID model.ID, entityName string, roleNames []string, revocation types.EntityAccessRevocation) (int, error) {
	revoked := 0
	err := b.msdkWrite(unitID, func(elem element.Element) error {
		dm, ok := elem.(*genDM.DomainModel)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *DomainModel)", elem)
		}
		readSet := make(map[string]struct{}, len(revocation.RevokeReadMembers))
		for _, m := range revocation.RevokeReadMembers {
			readSet[m] = struct{}{}
		}
		writeSet := make(map[string]struct{}, len(revocation.RevokeWriteMembers))
		for _, m := range revocation.RevokeWriteMembers {
			writeSet[m] = struct{}{}
		}
		for _, e := range dm.EntitiesItems() {
			ent, ok := e.(*genDM.Entity)
			if !ok || ent.Name() != entityName {
				continue
			}
			for _, r := range ent.AccessRulesItems() {
				rule, ok := r.(*genDM.AccessRule)
				if !ok {
					continue
				}
				if !rolesMatch(rule.ModuleRolesQualifiedNames(), roleNames) {
					continue
				}
				if revocation.RevokeCreate {
					rule.SetAllowCreate(false)
				}
				if revocation.RevokeDelete {
					rule.SetAllowDelete(false)
				}
				if revocation.RevokeReadAll {
					rule.SetDefaultMemberAccessRights("NoAccess")
				} else if revocation.RevokeWriteAll {
					rule.SetDefaultMemberAccessRights("ReadOnly")
				}
				for _, ma := range rule.MemberAccessesItems() {
					maTyped, ok := ma.(*genDM.MemberAccess)
					if !ok {
						continue
					}
					ref := maTyped.AttributeQualifiedName()
					if ref == "" {
						ref = maTyped.AssociationQualifiedName()
					}
					if revocation.RevokeReadAll {
						maTyped.SetAccessRights("NoAccess")
						continue
					}
					if _, ok := readSet[ref]; ok {
						maTyped.SetAccessRights("NoAccess")
						continue
					}
					if revocation.RevokeWriteAll {
						if maTyped.AccessRights() == "ReadWrite" {
							maTyped.SetAccessRights("ReadOnly")
						}
						continue
					}
					if _, ok := writeSet[ref]; ok {
						if maTyped.AccessRights() == "ReadWrite" {
							maTyped.SetAccessRights("ReadOnly")
						}
					}
				}
				revoked++
			}
		}
		return nil
	})
	return revoked, err
}

// reconcileMemberAccessesViaModelsdk stays on Patch* + writeUnitContents: the
// 200-LOC orphan-detection logic in sdk/mpr.Writer.ReconcileMemberAccesses is
// not yet ported to gen. Kept here so all entity-access funcs sit together.
func (b *MprBackend) reconcileMemberAccessesViaModelsdk(unitID model.ID, moduleName string) ([]modelsdkmpr.ReconcileChange, error) {
	if b.msdkWriter == nil {
		return nil, fmt.Errorf("modelsdk writer not initialized")
	}
	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
	if err != nil {
		return nil, fmt.Errorf("read unit: %w", err)
	}
	patched, changes, err := modelsdkmpr.PatchReconcileMemberAccesses(rawBytes, moduleName)
	if err != nil {
		return nil, err
	}
	if len(changes) > 0 {
		if err := b.writeUnitContents(unitID, patched); err != nil {
			return nil, err
		}
	}
	return changes, nil
}

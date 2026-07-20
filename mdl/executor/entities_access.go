// SPDX-License-Identifier: Apache-2.0

// Package executor - Entity access control (GRANT/REVOKE output and resolution)
package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// memberAccessLocalName resolves a MemberAccess BSON entry to the plain local
// name used in MDL GRANT statements.
//
// Mendix stores member references in two distinct formats depending on the
// member kind:
//   - Attributes: "Module.Entity.AttributeName"  (3-part, entity-scoped)
//   - Associations: "Module.AssociationName"     (2-part, module-scoped)
//
// When a BSON ID is stored instead, attrNames maps it to the plain name.
func memberAccessLocalName(ma *genDm.MemberAccess, attrNames map[string]string) string {
	if attrQN := ma.AttributeQualifiedName(); attrQN != "" {
		// Attribute: prefer the ID/name lookup built from the entity's own
		// attribute list; fall back to extracting the last segment of the
		// 3-part qualified name.
		if mapped, ok := attrNames[attrQN]; ok {
			return mapped
		}
		if parts := strings.Split(attrQN, "."); len(parts) == 3 {
			return parts[2]
		}
		return attrQN
	}

	// Association: return the full qualified name (Module.AssocName) so the
	// describe output matches the source MDL and roundtrips cleanly.
	return ma.AssociationQualifiedName()
}

// outputEntityAccessGrantsGen outputs GRANT statements for entity access rules.
func outputEntityAccessGrantsGen(ctx *ExecContext, entity *genDm.Entity, moduleName, entityName string) {
	if entity == nil || len(entity.AccessRulesItems()) == 0 {
		return
	}

	attrNames := make(map[string]string)
	for _, a := range entity.AttributesItems() {
		attr, ok := a.(*genDm.Attribute)
		if !ok {
			continue
		}
		attrNames[string(attr.ID())] = attr.Name()
		attrNames[attr.Name()] = attr.Name()
	}

	for _, r := range entity.AccessRulesItems() {
		rule, ok := r.(*genDm.AccessRule)
		if !ok {
			continue
		}
		roleStrs := entityRuleRoleStringsGen(rule)
		if len(roleStrs) == 0 {
			continue
		}

		rightsStr := formatAccessRuleRightsGen(rule, attrNames)
		if rightsStr == "" {
			continue
		}

		grantLine := fmt.Sprintf("\ngrant %s on %s.%s (%s)",
			strings.Join(roleStrs, ", "), moduleName, entityName, rightsStr)

		if rule.XPathConstraint() != "" {
			escaped := strings.ReplaceAll(rule.XPathConstraint(), "'", "''")
			grantLine += fmt.Sprintf(" where '%s'", escaped)
		}
		grantLine += ";"

		fmt.Fprintln(ctx.Output, grantLine)
	}
}

// resolveEntityMemberAccessGen determines per-member READ/WRITE access.
// Returns nil slices for "all members" (*), or specific member name lists.
func resolveEntityMemberAccessGen(rule *genDm.AccessRule, attrNames map[string]string) (readMembers []string, writeMembers []string) {
	if rule == nil || len(rule.MemberAccessesItems()) == 0 {
		return nil, nil
	}

	allMatchDefault := true
	for _, m := range rule.MemberAccessesItems() {
		ma, ok := m.(*genDm.MemberAccess)
		if !ok {
			continue
		}
		if ma.AccessRights() != rule.DefaultMemberAccessRights() {
			allMatchDefault = false
			break
		}
	}
	if allMatchDefault {
		return nil, nil
	}

	var readOnly, readWrite []string
	for _, m := range rule.MemberAccessesItems() {
		ma, ok := m.(*genDm.MemberAccess)
		if !ok {
			continue
		}
		memberName := memberAccessLocalName(ma, attrNames)
		switch ma.AccessRights() {
		case genDm.MemberAccessRightsReadWrite:
			readWrite = append(readWrite, memberName)
		case genDm.MemberAccessRightsReadOnly:
			readOnly = append(readOnly, memberName)
		}
	}

	allReadable := append(readOnly, readWrite...)
	if len(allReadable) == 0 {
		readMembers = nil
	} else {
		readMembers = allReadable
	}

	if len(readWrite) == 0 {
		writeMembers = []string{}
	} else {
		writeMembers = readWrite
	}

	return readMembers, writeMembers
}

// formatAccessRuleRightsGen formats the rights portion of an access rule as a string.
func formatAccessRuleRightsGen(rule *genDm.AccessRule, attrNames map[string]string) string {
	if rule == nil {
		return ""
	}

	var rights []string
	if rule.AllowCreate() {
		rights = append(rights, "create")
	}
	if rule.AllowDelete() {
		rights = append(rights, "delete")
	}

	hasRead := rule.DefaultMemberAccessRights() == genDm.MemberAccessRightsReadOnly ||
		rule.DefaultMemberAccessRights() == genDm.MemberAccessRightsReadWrite
	hasWrite := rule.DefaultMemberAccessRights() == genDm.MemberAccessRightsReadWrite
	if !hasRead || !hasWrite {
		for _, m := range rule.MemberAccessesItems() {
			ma, ok := m.(*genDm.MemberAccess)
			if !ok {
				continue
			}
			if ma.AccessRights() == genDm.MemberAccessRightsReadOnly ||
				ma.AccessRights() == genDm.MemberAccessRightsReadWrite {
				hasRead = true
			}
			if ma.AccessRights() == genDm.MemberAccessRightsReadWrite {
				hasWrite = true
			}
		}
	}

	readMembers, writeMembers := resolveEntityMemberAccessGen(rule, attrNames)

	if hasRead {
		if readMembers == nil {
			rights = append(rights, "read *")
		} else {
			rights = append(rights, fmt.Sprintf("read (%s)", strings.Join(readMembers, ", ")))
		}
	}
	if hasWrite {
		if writeMembers == nil {
			rights = append(rights, "write *")
		} else if len(writeMembers) > 0 {
			rights = append(rights, fmt.Sprintf("write (%s)", strings.Join(writeMembers, ", ")))
		}
	}

	return strings.Join(rights, ", ")
}

// formatAccessRuleResult re-reads the entity and formats the resulting access state
// for the given roles. Returns a string like "  Result: CREATE, READ (Name, Price)\n".
func formatAccessRuleResult(ctx *ExecContext, moduleName, entityName string, roleNames []string) string {
	invalidateDomainModelsCache(ctx)

	entity, _, err := findEntityGen(ctx, ast.QualifiedName{Module: moduleName, Name: entityName})
	if err != nil || entity == nil {
		return ""
	}

	attrNames := make(map[string]string)
	for _, a := range entity.AttributesItems() {
		attr, ok := a.(*genDm.Attribute)
		if !ok {
			continue
		}
		attrNames[string(attr.ID())] = attr.Name()
		attrNames[attr.Name()] = attr.Name()
	}

	roleSet := make(map[string]bool)
	for _, rn := range roleNames {
		roleSet[rn] = true
	}

	for _, r := range entity.AccessRulesItems() {
		rule, ok := r.(*genDm.AccessRule)
		if !ok {
			continue
		}
		matchCount := 0
		for _, rn := range rule.ModuleRolesQualifiedNames() {
			if roleSet[rn] {
				matchCount++
			}
		}
		if matchCount == 0 {
			continue
		}
		rightsStr := formatAccessRuleRightsGen(rule, attrNames)
		if rightsStr == "" {
			return "  Result: (no access)\n"
		}
		return fmt.Sprintf("  Result: %s\n", rightsStr)
	}

	return "  Result: (no access)\n"
}

// formatAccessRuleResultDeps is the HandlerDeps version of formatAccessRuleResult.
func formatAccessRuleResultDeps(deps *HandlerDeps, moduleName, entityName string, roleNames []string) string {
	invalidateDomainModelsCacheDeps(deps)

	entity, _, err := findEntityInDMsDeps(context.Background(), deps, ast.QualifiedName{Module: moduleName, Name: entityName})
	if err != nil || entity == nil {
		return ""
	}

	attrNames := make(map[string]string)
	for _, a := range entity.AttributesItems() {
		attr, ok := a.(*genDm.Attribute)
		if !ok {
			continue
		}
		attrNames[string(attr.ID())] = attr.Name()
		attrNames[attr.Name()] = attr.Name()
	}

	roleSet := make(map[string]bool)
	for _, rn := range roleNames {
		roleSet[rn] = true
	}

	for _, r := range entity.AccessRulesItems() {
		rule, ok := r.(*genDm.AccessRule)
		if !ok {
			continue
		}
		matchCount := 0
		for _, rn := range rule.ModuleRolesQualifiedNames() {
			if roleSet[rn] {
				matchCount++
			}
		}
		if matchCount == 0 {
			continue
		}
		rightsStr := formatAccessRuleRightsGen(rule, attrNames)
		if rightsStr == "" {
			return "  Result: (no access)\n"
		}
		return fmt.Sprintf("  Result: %s\n", rightsStr)
	}

	return "  Result: (no access)\n"
}

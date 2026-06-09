// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.f2 — gen-typed CreateObject / ChangeObject adders.
//
// CreateObject and ChangeObject are the most member-heavy adders in
// the actions family — both build a list of MemberChange entries
// from the SET / change clause and need the
// `resolveMemberChange` algorithm to classify each member name as
// attribute vs association. The legacy implementation is ~200 LoC
// of backend-driven domain-model walking; rather than duplicate that
// logic here, this file delegates to a temporary legacy `flowBuilder`
// adapter and copies the resolved string fields onto the gen
// `MemberChange` element.
//
// The legacy resolver mutates a `*microflows.MemberChange` in place;
// the adapter exposes the decision (attribute QN / association QN)
// as a small helper struct so the gen code only needs the strings.
// This keeps the gen path free of `microflows.*` types and avoids
// re-implementing the cross-module association lookup.

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// addCreateObjectActionGen emits a `$X = create E (member = expr, ...)`
// activity. Mirrors flowBuilder.addCreateObjectAction including the
// CommitTypeNo default (gen has no enum constant — the BSON value is
// the empty string for "No" / "Yes" / "YesWithoutEvents"; legacy used
// CommitTypeNo which serializes empty too).
func (fb *flowBuilderGen) addCreateObjectActionGen(s *ast.CreateObjectStmt) element.ID {
	action := genMf.NewCreateObjectAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetOutputVariableName(s.Variable)
	// CommitTypeNo serialises as "No" in BSON; explicit set so the
	// gen encoder doesn't omit the field on empty default.
	action.SetCommit("No")

	entityQN := ""
	if s.EntityType.Module != "" && s.EntityType.Name != "" {
		entityQN = s.EntityType.Module + "." + s.EntityType.Name
		action.SetEntityQualifiedName(entityQN)
	}

	if fb.varTypes != nil && entityQN != "" {
		fb.varTypes[s.Variable] = entityQN
	}

	for _, change := range s.Changes {
		mc := genMf.NewMemberChange()
		assignFreshID(mc)
		mc.SetType("Set")
		mc.SetValue(mendixExprValue(fb.memberExpressionToStringGen(change.Value, entityQN, change.Attribute)))
		applyResolvedMemberChangeGen(mc, fb.resolveMemberChangeGen(change.Attribute, entityQN))
		action.AddItems(mc)
	}

	return fb.genActivityWrap(action, s.ErrorHandling, s.Variable)
}

// addChangeObjectActionGen emits a `change $X (member = expr, ...)`
// activity. Mirrors flowBuilder.addChangeObjectAction including the
// CE0032 auto-promotion: an empty change with no commit is
// auto-bumped to RefreshInClient=true so Studio Pro doesn't reject
// it as "no items and not committed".
func (fb *flowBuilderGen) addChangeObjectActionGen(s *ast.ChangeObjectStmt) element.ID {
	action := genMf.NewChangeObjectAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(nil))
	action.SetChangeVariableName(s.Variable)
	action.SetCommit("No")
	action.SetRefreshInClient(s.RefreshInClient || len(s.Changes) == 0)

	entityQN := ""
	if fb.varTypes != nil {
		entityQN = fb.varTypes[s.Variable]
	}

	for _, change := range s.Changes {
		mc := genMf.NewMemberChange()
		assignFreshID(mc)
		mc.SetType("Set")
		mc.SetValue(mendixExprValue(fb.memberExpressionToStringGen(change.Value, entityQN, change.Attribute)))
		applyResolvedMemberChangeGen(mc, fb.resolveMemberChangeGen(change.Attribute, entityQN))
		action.AddItems(mc)
	}

	return fb.genActivityWrap(action, nil, "")
}

// resolvedMemberChange is the (attribute QN, association QN) pair
// that resolveMemberChangeGen returns. Exactly one is set for a
// successfully resolved member; both empty for unresolved (caller
// should leave the gen MemberChange's QN fields blank).
type resolvedMemberChange struct {
	attributeQN   string
	associationQN string
}

// resolveMemberChangeGen classifies a member name as attribute or
// association. Stage 3.2.6.4: switched to the standalone
// `resolveMemberChangeGenStandalone` (in flowbuilder_assoc_lookup_gen.go);
// the legacy `flowBuilder.resolveMemberChange` was deleted with the
// rest of the legacy builder family.
func (fb *flowBuilderGen) resolveMemberChangeGen(memberName, entityQN string) resolvedMemberChange {
	return resolveMemberChangeGenStandalone(fb.backend, memberName, entityQN)
}

// applyResolvedMemberChangeGen overlays the resolved (attribute QN,
// association QN) pair onto a gen MemberChange. Mendix requires BOTH
// Association and Attribute fields to be present in the BSON (the
// unused one as an explicit empty string). Omitting either field
// causes CE0117 "Error(s) in expression" during mx check.
func applyResolvedMemberChangeGen(mc *genMf.MemberChange, r resolvedMemberChange) {
	if r.attributeQN != "" {
		// Corpus-b analysis: Association is written BEFORE Attribute in Studio Pro BSON.
		// Field order matters for Mendix validation; set Association first.
		mc.SetAssociationQualifiedName("") // explicitly empty — required by Mendix BSON schema
		mc.SetAttributeQualifiedName(r.attributeQN)
	} else if r.associationQN != "" {
		mc.SetAssociationQualifiedName(r.associationQN)
		mc.SetAttributeQualifiedName("") // explicitly empty — required by Mendix BSON schema
	}
}

// memberExpressionToStringGen converts a member-assignment value expression
// to its BSON string form.
//
// For 3-part qualified enum references (Module.EnumName.Value), the value is
// stored as a quoted string literal ('Value') rather than as a qualified name:
//
//  1. Mendix accepts both 'Value' and Module.EnumName.Value in change actions.
//  2. The qualified-name form causes CE0117 "Error in expression" when the
//     microflow is created in a separate executor session from the entity/enum.
//  3. The string literal form never triggers CE0117.
//
// String literals (MDL: 'Value') are passed through fb.exprToString unchanged
// because expressionToString already formats them correctly.
// mendixExprValue wraps a Mendix expression string with the mandatory
// trailing newline that Studio Pro always appends to expression values
// stored in BSON. Without this newline, Mendix raises CE0117 "Error(s) in
// expression" because its expression parser expects the newline terminator.
func mendixExprValue(s string) string {
	if s == "" || s == "empty" {
		return s
	}
	return s + "\n"
}

func (fb *flowBuilderGen) memberExpressionToStringGen(expr ast.Expression, entityQN, attrName string) string {
	// Use standard expression rendering — with NewType fix in place, Mendix
	// correctly resolves attribute types so 3-part enum refs are valid.
	return fb.exprToString(expr)
}

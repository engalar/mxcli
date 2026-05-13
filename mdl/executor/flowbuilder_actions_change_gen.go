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
	"github.com/mendixlabs/mxcli/sdk/microflows"
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
		mc.SetValue(fb.memberExpressionToStringGen(change.Value, entityQN, change.Attribute))
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
		mc.SetValue(fb.memberExpressionToStringGen(change.Value, entityQN, change.Attribute))
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
// association by delegating to the legacy resolver — its behaviour
// is identical to what's needed here, and it has no SDK type
// references in the lookup logic itself (only the legacy MemberChange
// struct it mutates). We pass a throwaway *microflows.MemberChange
// purely as a destination buffer, then read the resolved fields out.
//
// This keeps the gen path free of `microflows.MemberChange` in its
// public surface, while sharing the cross-module association lookup
// algorithm without duplicating it.
func (fb *flowBuilderGen) resolveMemberChangeGen(memberName, entityQN string) resolvedMemberChange {
	mc := &microflows.MemberChange{}
	legacy := &flowBuilder{
		backend:   fb.backend,
		hierarchy: fb.hierarchy,
	}
	legacy.resolveMemberChange(mc, memberName, entityQN)
	return resolvedMemberChange{
		attributeQN:   mc.AttributeQualifiedName,
		associationQN: mc.AssociationQualifiedName,
	}
}

// applyResolvedMemberChangeGen overlays the resolved (attribute QN,
// association QN) pair onto a gen MemberChange. Exactly one of the
// two is expected to be non-empty per Mendix BSON contract; this
// helper is defensive in setting only the non-empty one.
func applyResolvedMemberChangeGen(mc *genMf.MemberChange, r resolvedMemberChange) {
	if r.attributeQN != "" {
		mc.SetAttributeQualifiedName(r.attributeQN)
	}
	if r.associationQN != "" {
		mc.SetAssociationQualifiedName(r.associationQN)
	}
}

// memberExpressionToStringGen ports flowBuilder.memberExpressionToString
// to the gen builder. The legacy lookup uses the backend's domain
// model, which the gen builder already exposes via fb.backend.
//
// String literals targeting an enumeration attribute are rewritten
// from `'Value'` to `Module.EnumName.Value` so Studio Pro recognises
// them as enum references.
func (fb *flowBuilderGen) memberExpressionToStringGen(expr ast.Expression, entityQN, attrName string) string {
	if lit, ok := expr.(*ast.LiteralExpr); ok && lit.Kind == ast.LiteralString {
		legacy := &flowBuilder{
			backend:   fb.backend,
			hierarchy: fb.hierarchy,
		}
		if enumRef := legacy.lookupEnumRef(entityQN, attrName); enumRef != "" {
			if v, ok := lit.Value.(string); ok {
				return enumRef + "." + v
			}
		}
	}
	return fb.exprToString(expr)
}

// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.4 D1.e: AST → gen-typed ValidationRule builders.
// Maps AST attribute constraints (NotNull / Unique with optional error
// messages) to gen ValidationRule + RuleInfo subtypes.

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// astToValidationRulesGen returns the list of *ValidationRule elements
// for a single AST Attribute. Each ValidationRule references the
// attribute by qualified name; the caller supplies entityQN to build
// the QN ("Module.Entity.AttrName"). Multiple rules per attribute (e.g.
// NotNull + Unique on the same column) produce one element each.
//
// gen ValidationRule.RuleInfo is a polymorphic Part holding one of the
// concrete RuleInfo subtypes — RequiredRuleInfo for NOT NULL,
// UniqueRuleInfo for UNIQUE. Other constraint kinds (range, regex,
// max-length) are out of scope for this builder; the AST does not
// expose them on Attribute today.
func astToValidationRulesGen(a *ast.Attribute, entityQN string) []*genDm.ValidationRule {
	if a == nil || entityQN == "" {
		return nil
	}
	attrQN := entityQN + "." + a.Name
	var out []*genDm.ValidationRule
	if a.NotNull {
		vr := genDm.NewValidationRule()
		vr.SetAttributeQualifiedName(attrQN)
		vr.SetRuleInfo(genDm.NewRequiredRuleInfo())
		// Error message rendering — gen ErrorMessage is a Text element;
		// modelling that requires the cross-package types.Text gen
		// equivalent. The AST's NotNullError string passthrough is
		// deferred until the gen Text builder lands; for now, the rule
		// emits without a custom error message (Mendix falls back to
		// the default localized message).
		_ = a.NotNullError
		out = append(out, vr)
	}
	if a.Unique {
		vr := genDm.NewValidationRule()
		vr.SetAttributeQualifiedName(attrQN)
		vr.SetRuleInfo(genDm.NewUniqueRuleInfo())
		_ = a.UniqueError
		out = append(out, vr)
	}
	return out
}

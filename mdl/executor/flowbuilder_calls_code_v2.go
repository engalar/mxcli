// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.g2 — gen-typed CALL JAVA ACTION / JAVASCRIPT ACTION adders.
//
// Both code-action variants wrap a list of parameter mappings carrying
// a polymorphic ParameterValue:
//
//   JavaActionCallAction
//     ├── JavaActionQualifiedName  string
//     └── ParameterMappings PartList of JavaActionParameterMapping
//                                            ├── ParameterQualifiedName string
//                                            └── Value (alias `ParameterValue`)
//                                                  → BasicCodeActionParameterValue
//                                                    │ EntityTypeCodeActionParameterValue
//                                                    │ MicroflowParameterValue
//
//   JavaScriptActionCallAction
//     ├── JavaScriptActionQualifiedName  string
//     └── ParameterMappings PartList of JavaScriptActionParameterMapping
//                                            ├── ParameterQualifiedName string
//                                            └── ParameterValue
//                                                  → BasicCodeActionParameterValue
//                                                    │ (only this in legacy)
//
// Scope of g2 (offline / common path):
//
//   - All Java + JavaScript parameters land as
//     *BasicCodeActionParameterValue with Argument = rendered Mendix
//     expression. The legacy Java path further classifies parameters
//     by reading the Java action definition via
//     `fb.backend.ReadJavaActionByName` to detect EntityTypeParameterType /
//     MicroflowType — that classification is **deferred to commit j**
//     (the dispatcher commit) where backend resolution lives. The
//     write path used by gen tests offline is parity-clean for the
//     basic-parameter case which is the dominant shape in the
//     fixtures.
//
//   - Empty / null arguments produce an empty Argument string ("") —
//     the "intentionally unbound" marker per
//     PROPOSAL_microflow_empty_java_action_argument.md. The legacy
//     "type-aware empty → Argument: \"empty\"" promotion needs the
//     Java action definition and is part of the deferred backend
//     classification.
//
// Schema-gap tracking: none new — JavaActionParameterMapping has both
// `Value` and `ParameterValue` setters (BSON aliases for the same
// field; gen describer reads `Value` first then falls back). We use
// `Value` for parity with the legacy/describer expectation.

package executor

import (
	"strings"
	"unicode"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// addCallJavaActionActionGen emits a `[$Y = ]call java action Mod.Act(p1=…, …);`
// activity. All parameters are emitted as BasicCodeActionParameterValue
// in this commit; entity-type and microflow-type classification is
// deferred to commit j when backend resolution is wired in.
func (fb *flowBuilderGen) addCallJavaActionActionGen(s *ast.CallJavaActionStmt) element.ID {
	actionQN := s.ActionName.Module + "." + s.ActionName.Name

	action := genMf.NewJavaActionCallAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetJavaActionQualifiedName(actionQN)
	action.SetOutputVariableName(s.OutputVariable)
	action.SetUseReturnVariable(s.OutputVariable != "")

	for _, arg := range s.Arguments {
		mapping := genMf.NewJavaActionParameterMapping()
		assignFreshID(mapping)
		mapping.SetParameterQualifiedName(actionQN + "." + arg.Name)

		var paramValue element.Element
		if isEntityTypeArgument(arg.Value) {
			// Entity-type parameters must use EntityTypeCodeActionParameterValue
			// with the entity qualified name (no quotes), not BasicCodeActionParameterValue.
			// mx check requires this distinction to validate callers' return variables.
			entityQN := extractEntityQN(fb.exprToString(arg.Value))
			ev := genMf.NewEntityTypeCodeActionParameterValue()
			assignFreshID(ev)
			ev.SetEntityQualifiedName(entityQN)
			paramValue = ev
		} else {
			bv := genMf.NewBasicCodeActionParameterValue()
			assignFreshID(bv)
			if !isEmptyJavaActionArgumentGen(arg.Value) {
				bv.SetArgument(fb.exprToString(arg.Value))
			}
			paramValue = bv
		}
		// Legacy uses .Value (long-standing BSON field); the gen
		// describer reads .Value first then falls back to
		// .ParameterValue. Set Value for legacy parity.
		mapping.SetValue(paramValue)

		action.AddParameterMappings(mapping)
	}

	// TODO Stage 3.2.3.j: backend-driven param classification —
	// MicroflowParameterValue for microflow-type params, the
	// "Argument: empty" promotion for typed BasicParameterTypes when
	// the AST argument is `empty`/`null`. Also output-variable type
	// tracking via javaActionReturnVarType + inferGenericJavaActionReturnType.

	return fb.genActivityWrap(action, s.ErrorHandling, s.OutputVariable)
}

// addCallJavaScriptActionActionGen emits a `[$Y = ]call javascript
// action Mod.Act(p1=…, …);` activity. JavaScript actions are
// BasicCodeActionParameterValue-only — no entity/microflow-type
// classification path exists in legacy.
func (fb *flowBuilderGen) addCallJavaScriptActionActionGen(s *ast.CallJavaScriptActionStmt) element.ID {
	actionQN := s.ActionName.Module + "." + s.ActionName.Name

	action := genMf.NewJavaScriptActionCallAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetJavaScriptActionQualifiedName(actionQN)
	action.SetOutputVariableName(s.OutputVariable)
	action.SetUseReturnVariable(s.OutputVariable != "")

	for _, arg := range s.Arguments {
		mapping := genMf.NewJavaScriptActionParameterMapping()
		assignFreshID(mapping)
		// Lowercase first letter to match Studio Pro convention
		argName := strings.ToLower(arg.Name[:1]) + arg.Name[1:]
		mapping.SetParameterQualifiedName(actionQN + "." + argName)

		value := genMf.NewBasicCodeActionParameterValue()
		assignFreshID(value)
		value.SetArgument(fb.exprToString(arg.Value))
		mapping.SetParameterValue(value)

		action.AddParameterMappings(mapping)
	}

	return fb.genActivityWrap(action, s.ErrorHandling, s.OutputVariable)
}

// isEmptyJavaActionArgumentGen reports whether the AST argument is an
// `empty` or `null` literal. Mirrors legacy isEmptyJavaActionArgument.
// Used by the Java action adder to distinguish "intentionally unbound"
// arguments from rendered expressions; matters for the
// type-aware-empty branch the dispatcher commit will add.
func isEmptyJavaActionArgumentGen(expr ast.Expression) bool {
	lit, ok := expr.(*ast.LiteralExpr)
	return ok && (lit.Kind == ast.LiteralEmpty || lit.Kind == ast.LiteralNull)
}

// isEntityTypeArgument reports whether the AST expression is a quoted
// string literal in the form 'Module.EntityName' — the conventional MDL
// representation for entity-type Java action parameters (EntityType: entity <>).
// If true, the call emitter uses EntityTypeCodeActionParameterValue instead
// of BasicCodeActionParameterValue so that mx check can resolve the type.
func isEntityTypeArgument(expr ast.Expression) bool {
	// Unwrap SourceExpr — buildExpression always wraps in SourceExpr.
	if se, ok := expr.(*ast.SourceExpr); ok {
		expr = se.Expression
	}
	lit, ok := expr.(*ast.LiteralExpr)
	if !ok || lit.Kind != ast.LiteralString {
		return false
	}
	s, ok := lit.Value.(string)
	if !ok {
		return false
	}
	return isQualifiedEntityName(s)
}

// isQualifiedEntityName returns true when s looks like "Module.EntityName":
// two dot-separated segments, each non-empty and starting with an uppercase
// letter, containing only letters and digits.
func isQualifiedEntityName(s string) bool {
	parts := strings.SplitN(s, ".", 2)
	if len(parts) != 2 {
		return false
	}
	return isIdentifier(parts[0]) && isIdentifier(parts[1])
}

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 && !unicode.IsUpper(r) {
			return false
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

// extractEntityQN strips surrounding single-quotes from the rendered
// expression string to get the bare entity qualified name.
func extractEntityQN(rendered string) string {
	s := strings.TrimSpace(rendered)
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}

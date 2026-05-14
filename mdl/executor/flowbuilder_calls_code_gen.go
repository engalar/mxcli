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

		value := genMf.NewBasicCodeActionParameterValue()
		assignFreshID(value)
		if !isEmptyJavaActionArgumentGen(arg.Value) {
			value.SetArgument(fb.exprToString(arg.Value))
		}
		// Legacy uses .Value (long-standing BSON field); the gen
		// describer reads .Value first then falls back to
		// .ParameterValue. Set Value for legacy parity.
		mapping.SetValue(value)

		action.AddParameterMappings(mapping)
	}

	// TODO Stage 3.2.3.j: backend-driven param classification —
	// EntityTypeCodeActionParameterValue for entity-type params,
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
		mapping.SetParameterQualifiedName(actionQN + "." + arg.Name)

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

// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.g1 — gen-typed CALL MICROFLOW / CALL NANOFLOW adders.
//
// Both wrap a nested *Call element (MicroflowCall / NanoflowCall)
// inside the outer *CallAction:
//
//   MicroflowCallAction
//     └── MicroflowCall
//           ├── MicroflowQualifiedName       string
//           └── ParameterMappings PartList   of MicroflowCallParameterMapping
//                                                  ├── ParameterQualifiedName string
//                                                  └── Argument               string
//
//   NanoflowCallAction
//     └── NanoflowCall
//           ├── NanoflowQualifiedName        string
//           └── ParameterMappings PartList   of NanoflowCallParameterMapping
//                                                  ├── ParameterQualifiedName string
//                                                  └── Argument               string
//
// Existence-of-callee validation:
//   - microflowExistsGen / nanoflowExistsGen (foundation a) reports
//     "found" by default in offline mode (no repo wired) — same
//     fall-through-to-true semantics as the legacy variant. With a
//     repo present and the callee absent, an `addError` surfaces a
//     CALL MICROFLOW / NANOFLOW '<QN>' not-found message that mirrors
//     legacy wording.
//
// Output-variable type tracking is intentionally skipped here: the
// legacy registerResultVariableType walks the called microflow's
// `microflows.DataType` interface, which the gen path can't yet
// resolve because gen Microflow's ReturnType is a string + an
// optional element. The dispatcher commit (j) will revisit this
// when addStatement is wired and a backend-driven type lookup can
// translate the gen return-type pair back into our varTypes map.

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDt "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// checkOutputVarCollision reports an error if varName is already declared in fb,
// and registers it in declaredVars if not. Returns true if a collision is detected,
// in which case the caller must not set the action's output variable (doing so would
// produce a CE0111 duplicate-variable error when the project opens in Studio Pro).
func checkOutputVarCollision(fb *flowBuilderGen, varName string) bool {
	if varName == "" {
		return false
	}
	if fb.isVariableDeclared("$" + varName) {
		fb.addError(
			"output variable $%s is already declared in this microflow; use a unique name (e.g. $%s2)",
			varName, varName,
		)
		return true
	}
	fb.declaredVars[varName] = "Unknown"
	return false
}

// addCallMicroflowActionGen emits a `call microflow Mod.MF (args)`
// activity. Reports a validation error when the called microflow
// can't be located via fb.microflowsRepo (offline mode skips the
// check — see microflowExistsGen for the semantics).
func (fb *flowBuilderGen) addCallMicroflowActionGen(s *ast.CallMicroflowStmt) element.ID {
	mfQN := s.MicroflowName.Module + "." + s.MicroflowName.Name

	if !fb.microflowExistsGen(mfQN) {
		fb.addError("CALL MICROFLOW '%s': microflow not found in the project (check module name and spelling)", mfQN)
	}

	call := genMf.NewMicroflowCall()
	assignFreshID(call)
	call.SetMicroflowQualifiedName(mfQN)

	for _, arg := range s.Arguments {
		mapping := genMf.NewMicroflowCallParameterMapping()
		assignFreshID(mapping)
		mapping.SetParameterQualifiedName(mfQN + "." + arg.Name)
		mapping.SetArgument(fb.exprToString(arg.Value))
		call.AddParameterMappings(mapping)
	}

	action := genMf.NewMicroflowCallAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetMicroflowCall(call)

	if !checkOutputVarCollision(fb, s.OutputVariable) {
		action.SetOutputVariableName(s.OutputVariable)
		action.SetUseReturnVariable(s.OutputVariable != "")
		if s.OutputVariable != "" && fb.microflowsRepo != nil && fb.varTypes != nil {
			if mf, err := fb.microflowsRepo.FindByQualifiedName(mfQN); err == nil && mf != nil {
				fb.registerCallResultVarType(s.OutputVariable, mf.MicroflowReturnType())
			}
		}
	}

	return fb.genActivityWrap(action, s.ErrorHandling, s.OutputVariable)
}

// addCallNanoflowActionGen emits a `call nanoflow Mod.NF (args)`
// activity. Mirror image of addCallMicroflowActionGen for the
// nanoflow side (separate gen types, same wiring shape).
func (fb *flowBuilderGen) addCallNanoflowActionGen(s *ast.CallNanoflowStmt) element.ID {
	nfQN := s.NanoflowName.Module + "." + s.NanoflowName.Name

	if !fb.nanoflowExistsGen(nfQN) {
		fb.addError("CALL NANOFLOW '%s': nanoflow not found in the project (check module name and spelling)", nfQN)
	}

	call := genMf.NewNanoflowCall()
	assignFreshID(call)
	call.SetNanoflowQualifiedName(nfQN)

	for _, arg := range s.Arguments {
		mapping := genMf.NewNanoflowCallParameterMapping()
		assignFreshID(mapping)
		mapping.SetParameterQualifiedName(nfQN + "." + arg.Name)
		mapping.SetArgument(fb.exprToString(arg.Value))
		call.AddParameterMappings(mapping)
	}

	action := genMf.NewNanoflowCallAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetNanoflowCall(call)

	if !checkOutputVarCollision(fb, s.OutputVariable) {
		action.SetOutputVariableName(s.OutputVariable)
		action.SetUseReturnVariable(s.OutputVariable != "")
		if s.OutputVariable != "" && fb.nanoflowsRepo != nil && fb.varTypes != nil {
			if nf, err := fb.nanoflowsRepo.FindByQualifiedName(nfQN); err == nil && nf != nil {
				fb.registerCallResultVarType(s.OutputVariable, nf.MicroflowReturnType())
			}
		}
	}

	return fb.genActivityWrap(action, s.ErrorHandling, s.OutputVariable)
}

// registerCallResultVarType sets fb.varTypes[varName] from a
// MicroflowReturnType element when it is an ObjectType or ListType
// with a non-empty EntityQualifiedName. No-op for other types or nil.
func (fb *flowBuilderGen) registerCallResultVarType(varName string, retType element.Element) {
	if retType == nil {
		return
	}
	switch t := retType.(type) {
	case *genDt.ObjectType:
		if entityQN := t.EntityQualifiedName(); entityQN != "" {
			fb.varTypes[varName] = entityQN
		}
	case *genDt.ListType:
		if entityQN := t.EntityQualifiedName(); entityQN != "" {
			fb.varTypes[varName] = "List of " + entityQN
		}
	}
}

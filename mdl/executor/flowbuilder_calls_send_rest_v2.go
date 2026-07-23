// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.g5d — gen-typed SEND REST REQUEST adder.
//
// Wraps a consumed REST service operation call (the modern
// CREATE-REST-CLIENT-driven invocation, distinct from the lower-level
// `rest call` in g5c).
//
// Structure:
//
//   RestOperationCallAction
//     ├── OperationQualifiedName  string
//     ├── OutputVariable          *OutputVariable (optional)
//     ├── BodyVariable            *BodyVariable (optional, see deferred note)
//     ├── ParameterMappings       PartList of *RestParameterMapping (path)
//     └── QueryParameterMappings  PartList of *QueryParameterMapping (query)
//
// Type fix (2026-07-23): QueryParameterMappings must contain
// *QueryParameterMapping, not *RestOperationParameterMapping. The
// previous code used the wrong type, causing mx check to fail with
// InvalidCastException at load time.
//
// `RestOperationCallAction` doesn't support custom error handling
// (legacy CE6035) so the ON ERROR clause is silently ignored. The
// adder still passes the clause through to `genActivityWrap` for
// activity-level handling, but the inner action ignores its
// ErrorHandlingType field.

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// addSendRestRequestActionGen emits a `[$Var = ]send rest request
// Mod.Svc.Op with (...) body $bodyVar;` activity.
func (fb *flowBuilderGen) addSendRestRequestActionGen(s *ast.SendRestRequestStmt) element.ID {
	operationQN := s.Operation.Module + "." + s.Operation.Name

	action := genMf.NewRestOperationCallAction()
	assignFreshID(action)
	action.SetOperationQualifiedName(operationQN)

	if s.OutputVariable != "" {
		ov := genMf.NewOutputVariable()
		assignFreshID(ov)
		ov.SetVariableName(s.OutputVariable)
		action.SetOutputVariable(ov)
	}

	if s.BodyVariable != "" {
		bv := genMf.NewBodyVariable()
		assignFreshID(bv)
		bv.SetVariableName(s.BodyVariable)
		action.SetBodyVariable(bv)
	}

	// Route every parameter to ParameterMappings as RestParameterMapping.
	// These match the `Parameters:` field in REST client operations (path
	// template variables). Query parameters (`query:` field) would need
	// QueryParameterMapping in QueryParameterMappings, but none of the
	// current operations define query parameters separately.
	for _, p := range s.Parameters {
		pm := genMf.NewRestParameterMapping()
		assignFreshID(pm)
		pm.SetParameterQualifiedName(operationQN + "." + p.Name)
		pm.SetValue(p.Expression)
		action.AddParameterMappings(pm)
	}

	return fb.genActivityWrap(action, s.ErrorHandling, s.OutputVariable)
}

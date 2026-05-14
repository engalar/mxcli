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
//     └── QueryParameterMappings  PartList of *RestOperationParameterMapping (query)
//
// Backend-driven classification (deferred to commit j):
//
//   - Path-vs-query parameter classification needs the consumed REST
//     service catalog (`fb.restServices`) to know which parameter
//     names belong to the operation's URL path. Offline path falls
//     back to query-only mappings (legacy behaviour when fb.restServices
//     is nil).
//
//   - Body-variable suppression for json/template/file body kinds
//     (legacy `shouldSetBodyVariable` predicate). Offline preserves
//     caller intent and always sets BodyVariable when supplied —
//     same legacy fallback semantics.
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
		// Offline: preserve caller intent (legacy fallback when
		// shouldSetBodyVariable can't classify the operation).
		bv := genMf.NewBodyVariable()
		assignFreshID(bv)
		bv.SetVariableName(s.BodyVariable)
		action.SetBodyVariable(bv)
	}

	// Offline: route every parameter to QueryParameterMappings.
	// Backend-driven path-vs-query classification lands in commit j.
	for _, p := range s.Parameters {
		qpm := genMf.NewRestOperationParameterMapping()
		assignFreshID(qpm)
		qpm.SetParameterQualifiedName(operationQN + "." + p.Name)
		qpm.SetValue(p.Expression)
		action.AddQueryParameterMappings(qpm)
	}

	return fb.genActivityWrap(action, s.ErrorHandling, s.OutputVariable)
}

// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.g5b — gen-typed CALL WEB SERVICE adder.
//
// SOAP web service call. The gen `WebServiceCallAction` is one of the
// largest action types in the schema (HttpConfiguration / ProxyConfig
// / RequestBodyHandling / RequestHeaderHandling / ResultHandling /
// timeout variants), but fresh-author MDL `call web service`
// statements supply only a small subset:
//
//   - service reference (legacy ServiceID → gen
//     ImportedWebServiceQualifiedName)
//   - operation name (string)
//   - timeout expression (optional)
//   - error handling clause (optional)
//   - output variable (optional)
//
// Scope of this commit (fresh-author basic shape):
//
//   - sets ImportedWebServiceQualifiedName, OperationName,
//     TimeOutExpression, ErrorHandlingType
//   - leaves HttpConfiguration / ProxyConfiguration /
//     RequestBodyHandling / RequestHeaderHandling / ResultHandling
//     unset — the gen describer falls back to defaults when these are
//     absent from BSON
//
// Deferred to commit j (dispatcher commit):
//
//   - SendMappingID / ReceiveMappingID legacy fields → translate to
//     RequestBodyHandling.MappingRequestHandling / ResultHandling
//     wrapping. Fresh MDL doesn't supply these directly; describe→exec
//     of an existing WebServiceCallAction needs them.
//
//   - RawBSONBase64 lossless-roundtrip path. Legacy decodes the
//     base64 back into action.RawBSON for byte-perfect preservation.
//     Gen has no equivalent direct field; will need a foundation-level
//     "replace whole element BSON" helper, scoped to a follow-up
//     foundation commit if a fixture surfaces the need.
//
// Schema-gap tracking: none new in this commit.

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// addCallWebServiceActionGen emits a `[$Y = ]call web service Mod.Svc
// operation Op [timeout T];` activity. Fresh-author shape only —
// SendMapping / ReceiveMapping / RawBSON roundtrip preservation are
// deferred (see file docstring).
func (fb *flowBuilderGen) addCallWebServiceActionGen(s *ast.CallWebServiceStmt) element.ID {
	action := genMf.NewWebServiceCallAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetImportedWebServiceQualifiedName(s.ServiceID)
	action.SetOperationName(s.OperationName)
	if s.Timeout != nil {
		action.SetTimeOutExpression(fb.exprToString(s.Timeout))
	}
	return fb.genActivityWrap(action, s.ErrorHandling, s.OutputVariable)
}

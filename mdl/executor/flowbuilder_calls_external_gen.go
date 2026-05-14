// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.g4 — gen-typed external/integration call adders.
//
// Three independent adders shipped here:
//
//   - addCallExternalActionGen — wraps a CallExternalAction
//     (consumed OData call) carrying ExternalActionParameterMapping
//     entries.
//
//   - addExecuteDatabaseQueryActionGen — wraps a Database Connector
//     ExecuteDatabaseQueryAction (lives in
//     `modelsdk/gen/databaseconnector`, distinct package). Carries
//     two parameter-mapping lists: parameters (QueryParameterMapping)
//     and connection-overrides (ConnectionParameterMapping).
//
//   - addTransformJsonActionGen — wraps a TransformJsonAction
//     (DataTransformer-backed JSON-to-JSON transformation).
//
// Schema-gap notes:
//
//   - `*genMf.CallExternalAction` has NO `OutputVariableName` and NO
//     `UseReturnVariable` getter/setter pair. The legacy
//     `ResultVariableName` maps to gen `VariableName`; legacy
//     `UseReturnVariable` has no gen equivalent (the gen reader
//     infers it from VariableName != "" — see show_gen).
//
// REST + SendRest + ImportXml + ExportXml + WebServiceCall are
// deferred to a follow-up commit (g5+) because they each carry deep
// nested polymorphic dispatch that warrants their own focused work.

package executor

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDb "github.com/mendixlabs/mxcli/modelsdk/gen/databaseconnector"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// addCallExternalActionGen emits a `[$Y = ]call external action
// ServiceQN.Action(args);` activity. The OData service is referenced
// by qualified name; arguments map to ExternalActionParameterMapping
// entries with their AST argument rendered into a Mendix expression.
func (fb *flowBuilderGen) addCallExternalActionGen(s *ast.CallExternalActionStmt) element.ID {
	serviceQN := s.ServiceName.Module + "." + s.ServiceName.Name

	action := genMf.NewCallExternalAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetConsumedODataServiceQualifiedName(serviceQN)
	action.SetName(s.ActionName)
	if s.OutputVariable != "" {
		action.SetVariableName(s.OutputVariable)
	}

	for _, arg := range s.Arguments {
		mapping := genMf.NewExternalActionParameterMapping()
		assignFreshID(mapping)
		mapping.SetParameterName(arg.Name)
		mapping.SetArgument(fb.exprToString(arg.Value))
		action.AddParameterMappings(mapping)
	}

	return fb.genActivityWrap(action, s.ErrorHandling, s.OutputVariable)
}

// addExecuteDatabaseQueryActionGen emits a `[$Y = ]execute database
// query Q (args) [over connection (...)];` activity.
//
// DynamicQuery is a Mendix expression — strings need single quotes
// with embedded `'` doubled per Mendix expression syntax. The legacy
// adder applies that wrapping when the value isn't already quoted;
// we mirror the same predicate (`!strings.HasPrefix(v, "'")`) so
// already-quoted authored values pass through unchanged.
func (fb *flowBuilderGen) addExecuteDatabaseQueryActionGen(s *ast.ExecuteDatabaseQueryStmt) element.ID {
	dynamicQuery := s.DynamicQuery
	if dynamicQuery != "" && !strings.HasPrefix(dynamicQuery, "'") {
		dynamicQuery = "'" + strings.ReplaceAll(dynamicQuery, "'", "''") + "'"
	}

	action := genDb.NewExecuteDatabaseQueryAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetOutputVariableName(s.OutputVariable)
	action.SetQueryQualifiedName(s.QueryName)
	action.SetDynamicQuery(dynamicQuery)

	for _, arg := range s.Arguments {
		pm := genDb.NewQueryParameterMapping()
		assignFreshID(pm)
		pm.SetParameterName(arg.Name)
		pm.SetValue(fb.exprToString(arg.Value))
		action.AddParameterMappings(pm)
	}

	for _, arg := range s.ConnectionArguments {
		cm := genDb.NewConnectionParameterMapping()
		assignFreshID(cm)
		cm.SetParameterName(arg.Name)
		cm.SetValue(fb.exprToString(arg.Value))
		action.AddConnectionParameterMappings(cm)
	}

	return fb.genActivityWrap(action, s.ErrorHandling, s.OutputVariable)
}

// addTransformJsonActionGen emits a `$Y = transform json $Input via
// Mod.Transformer;` activity.
func (fb *flowBuilderGen) addTransformJsonActionGen(s *ast.TransformJsonStmt) element.ID {
	action := genMf.NewTransformJsonAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetInputVariableName(s.InputVariable)
	action.SetOutputVariableName(s.OutputVariable)
	action.SetTransformationQualifiedName(s.Transformation.Module + "." + s.Transformation.Name)
	return fb.genActivityWrap(action, s.ErrorHandling, s.OutputVariable)
}

// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.g4 — external/integration call adder tests (TDD).
//
// Three independent adders covered here:
//
//   - addCallExternalActionGen — `[$Y = ]call external action ServiceQN.Action(args);`
//   - addExecuteDatabaseQueryActionGen — `[$Y = ]execute database query Query (args) [over connection (...)];`
//   - addTransformJsonActionGen — `$Y = transform json $Input via Mod.Transformer;`
//
// REST + SendRest + ImportXml + ExportXml + WebServiceCall are
// deferred to a follow-up commit because they have nested polymorphic
// dispatch (HttpConfiguration / RequestHandling / ResultHandling /
// OutputMethod) and warrant dedicated commits.

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genDb "github.com/mendixlabs/mxcli/modelsdk/gen/databaseconnector"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func TestAddCallExternalActionGenSetsServiceAndMappings(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CallExternalActionStmt{
		OutputVariable: "Result",
		ServiceName:    ast.QualifiedName{Module: "Sales", Name: "OrderService"},
		ActionName:     "GetOrders",
		Arguments: []ast.CallArgument{
			{Name: "From", Value: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "2026-01-01"}},
		},
	}
	fb.addCallExternalActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.CallExternalAction)
	if act.ConsumedODataServiceQualifiedName() != "Sales.OrderService" {
		t.Fatalf("service QN = %q", act.ConsumedODataServiceQualifiedName())
	}
	if act.Name() != "GetOrders" {
		t.Fatalf("name = %q", act.Name())
	}
	// gen CallExternalAction uses VariableName (not OutputVariableName).
	if act.VariableName() != "Result" {
		t.Fatalf("variable name = %q, want Result", act.VariableName())
	}
	mappings := act.ParameterMappingsItems()
	if len(mappings) != 1 {
		t.Fatalf("mappings = %d, want 1", len(mappings))
	}
	pm := mappings[0].(*genMf.ExternalActionParameterMapping)
	if pm.ParameterName() != "From" {
		t.Fatalf("param name = %q", pm.ParameterName())
	}
	if pm.Argument() != "'2026-01-01'" {
		t.Fatalf("argument = %q", pm.Argument())
	}
}

func TestAddCallExternalActionGenNoOutputVariable(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CallExternalActionStmt{
		ServiceName: ast.QualifiedName{Module: "M", Name: "S"},
		ActionName:  "Act",
	}
	fb.addCallExternalActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.CallExternalAction)
	if act.VariableName() != "" {
		t.Fatalf("variable name = %q, want empty", act.VariableName())
	}
}

func TestAddCallExternalActionGenContinueErrorHandlingType(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CallExternalActionStmt{
		ServiceName:   ast.QualifiedName{Module: "M", Name: "S"},
		ActionName:    "Act",
		ErrorHandling: &ast.ErrorHandlingClause{Type: ast.ErrorHandlingContinue},
	}
	fb.addCallExternalActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.CallExternalAction)
	if act.ErrorHandlingType() != "Continue" {
		t.Fatalf("eh = %q, want Continue", act.ErrorHandlingType())
	}
}

func TestAddExecuteDatabaseQueryActionGenStaticQuery(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.ExecuteDatabaseQueryStmt{
		OutputVariable: "Rows",
		QueryName:      "Sales.SalesDB.GetOpenOrders",
		Arguments: []ast.CallArgument{
			{Name: "MinAmount", Value: &ast.LiteralExpr{Kind: ast.LiteralInteger, Value: int64(100)}},
		},
	}
	fb.addExecuteDatabaseQueryActionGen(stmt)
	act := actionFromObjects(t, fb).(*genDb.ExecuteDatabaseQueryAction)
	if act.QueryQualifiedName() != "Sales.SalesDB.GetOpenOrders" {
		t.Fatalf("query QN = %q", act.QueryQualifiedName())
	}
	if act.OutputVariableName() != "Rows" {
		t.Fatalf("output var = %q", act.OutputVariableName())
	}
	mappings := act.ParameterMappingsItems()
	if len(mappings) != 1 {
		t.Fatalf("mappings = %d, want 1", len(mappings))
	}
	pm := mappings[0].(*genDb.QueryParameterMapping)
	if pm.ParameterName() != "MinAmount" {
		t.Fatalf("param name = %q", pm.ParameterName())
	}
	if pm.Value() != "100" {
		t.Fatalf("param value = %q", pm.Value())
	}
}

func TestAddExecuteDatabaseQueryActionGenDynamicQueryQuotesAndEscapes(t *testing.T) {
	// DynamicQuery is a Mendix expression — string literals need
	// single quotes, and embedded single quotes are doubled per
	// Mendix expression syntax.
	fb := newActionTestFb()
	stmt := &ast.ExecuteDatabaseQueryStmt{
		DynamicQuery: "select * from foo where x = 'open'",
	}
	fb.addExecuteDatabaseQueryActionGen(stmt)
	act := actionFromObjects(t, fb).(*genDb.ExecuteDatabaseQueryAction)
	want := "'select * from foo where x = ''open'''"
	if act.DynamicQuery() != want {
		t.Fatalf("dynamic query = %q, want %q", act.DynamicQuery(), want)
	}
}

func TestAddExecuteDatabaseQueryActionGenAlreadyQuotedDynamicQueryPasses(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.ExecuteDatabaseQueryStmt{
		DynamicQuery: "'pre quoted'",
	}
	fb.addExecuteDatabaseQueryActionGen(stmt)
	act := actionFromObjects(t, fb).(*genDb.ExecuteDatabaseQueryAction)
	// Already-quoted strings pass through unchanged (legacy parity).
	if act.DynamicQuery() != "'pre quoted'" {
		t.Fatalf("dynamic query = %q", act.DynamicQuery())
	}
}

func TestAddExecuteDatabaseQueryActionGenConnectionArguments(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.ExecuteDatabaseQueryStmt{
		QueryName: "M.C.Q",
		ConnectionArguments: []ast.CallArgument{
			{Name: "DSN", Value: &ast.VariableExpr{Name: "Conn"}},
		},
	}
	fb.addExecuteDatabaseQueryActionGen(stmt)
	act := actionFromObjects(t, fb).(*genDb.ExecuteDatabaseQueryAction)
	conns := act.ConnectionParameterMappingsItems()
	if len(conns) != 1 {
		t.Fatalf("connection mappings = %d, want 1", len(conns))
	}
	cpm := conns[0].(*genDb.ConnectionParameterMapping)
	if cpm.ParameterName() != "DSN" {
		t.Fatalf("conn param name = %q", cpm.ParameterName())
	}
	if cpm.Value() != "$Conn" {
		t.Fatalf("conn param value = %q", cpm.Value())
	}
}

func TestAddTransformJsonActionGenSetsAllFields(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.TransformJsonStmt{
		InputVariable:  "Input",
		OutputVariable: "Output",
		Transformation: ast.QualifiedName{Module: "Mod", Name: "Transformer"},
	}
	fb.addTransformJsonActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.TransformJsonAction)
	if act.InputVariableName() != "Input" {
		t.Fatalf("input var = %q", act.InputVariableName())
	}
	if act.OutputVariableName() != "Output" {
		t.Fatalf("output var = %q", act.OutputVariableName())
	}
	if act.TransformationQualifiedName() != "Mod.Transformer" {
		t.Fatalf("transformation QN = %q", act.TransformationQualifiedName())
	}
}

func TestAddTransformJsonActionGenContinueErrorHandling(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.TransformJsonStmt{
		InputVariable:  "I",
		OutputVariable: "O",
		Transformation: ast.QualifiedName{Module: "M", Name: "T"},
		ErrorHandling:  &ast.ErrorHandlingClause{Type: ast.ErrorHandlingContinue},
	}
	fb.addTransformJsonActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.TransformJsonAction)
	if act.ErrorHandlingType() != "Continue" {
		t.Fatalf("eh = %q, want Continue", act.ErrorHandlingType())
	}
}

func TestAddCallExternalActionGenAdvancesPos(t *testing.T) {
	fb := newActionTestFb()
	startX := fb.posX
	stmt := &ast.CallExternalActionStmt{
		ServiceName: ast.QualifiedName{Module: "M", Name: "S"},
		ActionName:  "Act",
	}
	fb.addCallExternalActionGen(stmt)
	if fb.posX != startX+HorizontalSpacing {
		t.Fatalf("posX = %d, want %d", fb.posX, startX+HorizontalSpacing)
	}
}

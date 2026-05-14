// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.g5c — REST CALL adder tests (TDD).
//
// gen `RestCallAction` is the largest single action in the schema.
// Tests cover the production-relevant fresh-author shape:
//
//   - HTTP method enum mapping
//   - URL template (StringTemplate with text + arguments)
//   - Header entries (HttpHeaderEntry list)
//   - Basic auth (HttpConfiguration auth fields)
//   - Custom body request handling
//   - Timeout expression (with default)
//   - Result handling discriminator (String / HttpResponse / Mapping / None)

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func newRestStmtBase() *ast.RestCallStmt {
	return &ast.RestCallStmt{
		Method: ast.HttpMethodGet,
		URL:    &ast.LiteralExpr{Kind: ast.LiteralString, Value: "https://example.com/api"},
		Result: ast.RestResult{Type: ast.RestResultString},
	}
}

func TestAddRestCallActionGenSetsHttpMethodAndUrl(t *testing.T) {
	fb := newActionTestFb()
	stmt := newRestStmtBase()
	stmt.Method = ast.HttpMethodPost
	fb.addRestCallActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RestCallAction)
	hc, ok := act.HttpConfiguration().(*genMf.HttpConfiguration)
	if !ok {
		t.Fatalf("HttpConfiguration = %T, want *HttpConfiguration", act.HttpConfiguration())
	}
	if hc.HttpMethod() != "Post" {
		t.Fatalf("http method = %q, want Post", hc.HttpMethod())
	}
}

func TestAddRestCallActionGenStringTemplateUrl(t *testing.T) {
	fb := newActionTestFb()
	stmt := newRestStmtBase()
	fb.addRestCallActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RestCallAction)
	hc := act.HttpConfiguration().(*genMf.HttpConfiguration)
	tmpl, ok := hc.CustomLocationTemplate().(*genMf.StringTemplate)
	if !ok {
		t.Fatalf("CustomLocationTemplate = %T, want *StringTemplate", hc.CustomLocationTemplate())
	}
	if tmpl.Text() != "https://example.com/api" {
		t.Fatalf("url text = %q", tmpl.Text())
	}
}

func TestAddRestCallActionGenHeaderEntries(t *testing.T) {
	fb := newActionTestFb()
	stmt := newRestStmtBase()
	stmt.Headers = []ast.RestHeader{
		{Name: "Accept", Value: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "application/json"}},
		{Name: "X-Trace", Value: &ast.VariableExpr{Name: "TraceID"}},
	}
	fb.addRestCallActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RestCallAction)
	hc := act.HttpConfiguration().(*genMf.HttpConfiguration)
	headers := hc.HeaderEntriesItems()
	if len(headers) != 2 {
		t.Fatalf("headers = %d, want 2", len(headers))
	}
	h1 := headers[0].(*genMf.HttpHeaderEntry)
	if h1.Key() != "Accept" || h1.Value() != "'application/json'" {
		t.Fatalf("header 0 = %q / %q", h1.Key(), h1.Value())
	}
	h2 := headers[1].(*genMf.HttpHeaderEntry)
	if h2.Key() != "X-Trace" || h2.Value() != "$TraceID" {
		t.Fatalf("header 1 = %q / %q", h2.Key(), h2.Value())
	}
}

func TestAddRestCallActionGenBasicAuth(t *testing.T) {
	fb := newActionTestFb()
	stmt := newRestStmtBase()
	stmt.Auth = &ast.RestAuth{
		Username: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "admin"},
		Password: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "secret"},
	}
	fb.addRestCallActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RestCallAction)
	hc := act.HttpConfiguration().(*genMf.HttpConfiguration)
	if !hc.UseAuthentication() {
		t.Fatal("UseAuthentication should be true when auth supplied")
	}
}

func TestAddRestCallActionGenCustomBodyRequestHandling(t *testing.T) {
	fb := newActionTestFb()
	stmt := newRestStmtBase()
	stmt.Body = &ast.RestBody{
		Type:     ast.RestBodyCustom,
		Template: &ast.LiteralExpr{Kind: ast.LiteralString, Value: `{"hello":"world"}`},
	}
	fb.addRestCallActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RestCallAction)
	if act.RequestHandlingType() != "Custom" {
		t.Fatalf("request handling type = %q, want Custom", act.RequestHandlingType())
	}
	rh, ok := act.RequestHandling().(*genMf.CustomRequestHandling)
	if !ok {
		t.Fatalf("RequestHandling = %T, want *CustomRequestHandling", act.RequestHandling())
	}
	tmpl, ok := rh.Template().(*genMf.StringTemplate)
	if !ok {
		t.Fatalf("Template = %T, want *StringTemplate", rh.Template())
	}
	if tmpl.Text() != `{"hello":"world"}` {
		t.Fatalf("body text = %q", tmpl.Text())
	}
}

func TestAddRestCallActionGenMappingBodyRequestHandling(t *testing.T) {
	fb := newActionTestFb()
	stmt := newRestStmtBase()
	stmt.Body = &ast.RestBody{
		Type:           ast.RestBodyMapping,
		MappingName:    ast.QualifiedName{Module: "Sales", Name: "OrderExport"},
		SourceVariable: "Order",
	}
	fb.addRestCallActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RestCallAction)
	if act.RequestHandlingType() != "Mapping" {
		t.Fatalf("request handling type = %q, want Mapping", act.RequestHandlingType())
	}
	rh, ok := act.RequestHandling().(*genMf.MappingRequestHandling)
	if !ok {
		t.Fatalf("RequestHandling = %T, want *MappingRequestHandling", act.RequestHandling())
	}
	if rh.MappingQualifiedName() != "Sales.OrderExport" {
		t.Fatalf("mapping QN = %q", rh.MappingQualifiedName())
	}
	if rh.MappingArgumentVariableName() != "Order" {
		t.Fatalf("mapping arg = %q", rh.MappingArgumentVariableName())
	}
}

func TestAddRestCallActionGenResultStringDiscriminator(t *testing.T) {
	fb := newActionTestFb()
	stmt := newRestStmtBase()
	stmt.Result = ast.RestResult{Type: ast.RestResultString}
	fb.addRestCallActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RestCallAction)
	if act.ResultHandlingType() != "String" {
		t.Fatalf("result handling type = %q, want String", act.ResultHandlingType())
	}
}

func TestAddRestCallActionGenResultHttpResponseDiscriminator(t *testing.T) {
	fb := newActionTestFb()
	stmt := newRestStmtBase()
	stmt.OutputVariable = "Resp"
	stmt.Result = ast.RestResult{Type: ast.RestResultResponse}
	fb.addRestCallActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RestCallAction)
	if act.ResultHandlingType() != "HttpResponse" {
		t.Fatalf("result handling type = %q, want HttpResponse", act.ResultHandlingType())
	}
	rh, ok := act.ResultHandling().(*genMf.ResultHandling)
	if !ok {
		t.Fatalf("ResultHandling = %T, want *ResultHandling", act.ResultHandling())
	}
	if rh.OutputVariableName() != "Resp" {
		t.Fatalf("output var = %q, want Resp", rh.OutputVariableName())
	}
}

func TestAddRestCallActionGenResultMappingDiscriminator(t *testing.T) {
	fb := newActionTestFb()
	stmt := newRestStmtBase()
	stmt.OutputVariable = "Order"
	stmt.Result = ast.RestResult{
		Type:         ast.RestResultMapping,
		MappingName:  ast.QualifiedName{Module: "Sales", Name: "OrderImport"},
		ResultEntity: ast.QualifiedName{Module: "Sales", Name: "Order"},
		IsList:       false,
	}
	fb.addRestCallActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RestCallAction)
	if act.ResultHandlingType() != "Mapping" {
		t.Fatalf("result handling type = %q, want Mapping", act.ResultHandlingType())
	}
	rh := act.ResultHandling().(*genMf.ResultHandling)
	if rh.OutputVariableName() != "Order" {
		t.Fatalf("output var = %q", rh.OutputVariableName())
	}
	call, ok := rh.ImportMappingCall().(*genMf.ImportMappingCall)
	if !ok {
		t.Fatalf("ImportMappingCall = %T, want *ImportMappingCall", rh.ImportMappingCall())
	}
	if call.MappingQualifiedName() != "Sales.OrderImport" {
		t.Fatalf("mapping QN = %q", call.MappingQualifiedName())
	}
	if !call.ForceSingleOccurrence() {
		t.Fatal("single object → ForceSingleOccurrence should be true")
	}
}

func TestAddRestCallActionGenResultMappingListDiscriminator(t *testing.T) {
	fb := newActionTestFb()
	stmt := newRestStmtBase()
	stmt.Result = ast.RestResult{
		Type:        ast.RestResultMapping,
		MappingName: ast.QualifiedName{Module: "M", Name: "Map"},
		IsList:      true,
	}
	fb.addRestCallActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RestCallAction)
	rh := act.ResultHandling().(*genMf.ResultHandling)
	call := rh.ImportMappingCall().(*genMf.ImportMappingCall)
	if call.ForceSingleOccurrence() {
		t.Fatal("list → ForceSingleOccurrence should be false")
	}
}

func TestAddRestCallActionGenResultNoneDiscriminator(t *testing.T) {
	fb := newActionTestFb()
	stmt := newRestStmtBase()
	stmt.Result = ast.RestResult{Type: ast.RestResultNone}
	fb.addRestCallActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RestCallAction)
	if act.ResultHandlingType() != "None" {
		t.Fatalf("result handling type = %q, want None", act.ResultHandlingType())
	}
}

func TestAddRestCallActionGenTimeoutDefault(t *testing.T) {
	fb := newActionTestFb()
	stmt := newRestStmtBase()
	fb.addRestCallActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RestCallAction)
	// Default 300s when no timeout supplied.
	if act.TimeOutExpression() != "300" {
		t.Fatalf("timeout default = %q, want 300", act.TimeOutExpression())
	}
}

func TestAddRestCallActionGenTimeoutExpression(t *testing.T) {
	fb := newActionTestFb()
	stmt := newRestStmtBase()
	stmt.Timeout = &ast.LiteralExpr{Kind: ast.LiteralInteger, Value: int64(60)}
	fb.addRestCallActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RestCallAction)
	if act.TimeOutExpression() != "60" {
		t.Fatalf("timeout = %q, want 60", act.TimeOutExpression())
	}
}

func TestAddRestCallActionGenContinueErrorHandling(t *testing.T) {
	fb := newActionTestFb()
	stmt := newRestStmtBase()
	stmt.ErrorHandling = &ast.ErrorHandlingClause{Type: ast.ErrorHandlingContinue}
	fb.addRestCallActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RestCallAction)
	if act.ErrorHandlingType() != "Continue" {
		t.Fatalf("eh = %q, want Continue", act.ErrorHandlingType())
	}
}

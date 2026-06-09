// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.g5b — CALL WEB SERVICE adder tests (TDD).
//
// gen `WebServiceCallAction` covers SOAP web service calls. Scope of
// this commit: fresh-author basic shape (service + operation + timeout
// + output var + error handling). The legacy `RawBSONBase64`
// roundtrip-preservation path (used by describe→exec of an existing
// WebServiceCallAction) is **deferred** to commit j when the gen
// dispatcher can wire raw-BSON injection of the entire action body.
//
// SendMapping / ReceiveMapping ID legacy fields are also deferred —
// they map onto gen `RequestBodyHandling` / `ResultHandling` element
// sub-types whose schemas need their own focused work, and fresh
// MDL `call web service` statements typically don't supply them.

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func TestAddCallWebServiceActionGenSetsAllFields(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CallWebServiceStmt{
		OutputVariable: "Resp",
		ServiceID:      "Mod.OrderSvc",
		OperationName:  "GetOrder",
		Timeout:        &ast.LiteralExpr{Kind: ast.LiteralInteger, Value: int64(60)},
	}
	fb.addCallWebServiceActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.WebServiceCallAction)
	if act.ImportedWebServiceQualifiedName() != "Mod.OrderSvc" {
		t.Fatalf("service QN = %q, want Mod.OrderSvc", act.ImportedWebServiceQualifiedName())
	}
	if act.OperationName() != "GetOrder" {
		t.Fatalf("operation name = %q, want GetOrder", act.OperationName())
	}
	if act.TimeOutExpression() != "60" {
		t.Fatalf("timeout expr = %q, want 60", act.TimeOutExpression())
	}
}

func TestAddCallWebServiceActionGenWithoutTimeout(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CallWebServiceStmt{
		ServiceID:     "Mod.Svc",
		OperationName: "Op",
	}
	fb.addCallWebServiceActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.WebServiceCallAction)
	if act.TimeOutExpression() != "" {
		t.Fatalf("timeout should be empty when nil, got %q", act.TimeOutExpression())
	}
}

func TestAddCallWebServiceActionGenContinueErrorHandling(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CallWebServiceStmt{
		ServiceID:     "Mod.Svc",
		OperationName: "Op",
		ErrorHandling: &ast.ErrorHandlingClause{Type: ast.ErrorHandlingContinue},
	}
	fb.addCallWebServiceActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.WebServiceCallAction)
	if act.ErrorHandlingType() != "Continue" {
		t.Fatalf("eh = %q, want Continue", act.ErrorHandlingType())
	}
}

func TestAddCallWebServiceActionGenAdvancesPos(t *testing.T) {
	fb := newActionTestFb()
	startX := fb.posX
	fb.addCallWebServiceActionGen(&ast.CallWebServiceStmt{
		ServiceID:     "M.S",
		OperationName: "Op",
	})
	if fb.posX != startX+HorizontalSpacing {
		t.Fatalf("posX = %d, want %d", fb.posX, startX+HorizontalSpacing)
	}
}

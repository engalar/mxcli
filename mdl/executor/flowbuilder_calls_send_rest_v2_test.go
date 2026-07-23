// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.g5d — SEND REST REQUEST adder tests (TDD).
//
// gen `RestOperationCallAction` wraps a consumed REST service call.
// Tests cover the offline path: operation reference + parameter
// mappings + output/body variables + error handling.
//
// Backend-driven parameter classification (path vs query) is
// **deferred** to commit j — the offline default is to route every
// parameter to QueryParameterMappings (matches legacy behaviour when
// fb.restServices is nil).

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func TestAddSendRestRequestActionGenSetsOperation(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.SendRestRequestStmt{
		Operation: ast.QualifiedName{Module: "Mod", Name: "ApiSvc.GetById"},
	}
	fb.addSendRestRequestActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RestOperationCallAction)
	if act.OperationQualifiedName() != "Mod.ApiSvc.GetById" {
		t.Fatalf("operation QN = %q, want Mod.ApiSvc.GetById", act.OperationQualifiedName())
	}
}

func TestAddSendRestRequestActionGenOutputVariable(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.SendRestRequestStmt{
		Operation:      ast.QualifiedName{Module: "M", Name: "S.Op"},
		OutputVariable: "Result",
	}
	fb.addSendRestRequestActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RestOperationCallAction)
	ov, ok := act.OutputVariable().(*genMf.OutputVariable)
	if !ok {
		t.Fatalf("OutputVariable = %T, want *OutputVariable", act.OutputVariable())
	}
	if ov.VariableName() != "Result" {
		t.Fatalf("output var = %q, want Result", ov.VariableName())
	}
}

func TestAddSendRestRequestActionGenNoOutputVariable(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.SendRestRequestStmt{
		Operation: ast.QualifiedName{Module: "M", Name: "S.Op"},
	}
	fb.addSendRestRequestActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RestOperationCallAction)
	if act.OutputVariable() != nil {
		t.Fatalf("OutputVariable should be nil when no output var set, got %T", act.OutputVariable())
	}
}

func TestAddSendRestRequestActionGenBodyVariable(t *testing.T) {
	// Without backend operation lookup (offline), legacy preserves
	// caller intent and sets BodyVariable. We mirror that fallback.
	fb := newActionTestFb()
	stmt := &ast.SendRestRequestStmt{
		Operation:    ast.QualifiedName{Module: "M", Name: "S.Op"},
		BodyVariable: "Body",
	}
	fb.addSendRestRequestActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RestOperationCallAction)
	bv, ok := act.BodyVariable().(*genMf.BodyVariable)
	if !ok {
		t.Fatalf("BodyVariable = %T, want *BodyVariable", act.BodyVariable())
	}
	if bv.VariableName() != "Body" {
		t.Fatalf("body var = %q, want Body", bv.VariableName())
	}
}

func TestAddSendRestRequestActionGenParameters(t *testing.T) {
	// All parameters route to ParameterMappings as RestParameterMapping,
	// matching the `Parameters:` field in REST client operations.
	fb := newActionTestFb()
	stmt := &ast.SendRestRequestStmt{
		Operation: ast.QualifiedName{Module: "Mod", Name: "Svc.Op"},
		Parameters: []ast.SendRestParamDef{
			{Name: "p1", Expression: "$X"},
			{Name: "p2", Expression: "'literal'"},
		},
	}
	fb.addSendRestRequestActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.RestOperationCallAction)
	pathParams := act.ParameterMappingsItems()
	if len(pathParams) != 2 {
		t.Fatalf("path params = %d, want 2", len(pathParams))
	}
	p1 := pathParams[0].(*genMf.RestParameterMapping)
	if p1.ParameterQualifiedName() != "Mod.Svc.Op.p1" {
		t.Fatalf("param 1 QN = %q", p1.ParameterQualifiedName())
	}
	if p1.Value() != "$X" {
		t.Fatalf("param 1 value = %q", p1.Value())
	}
}

func TestAddSendRestRequestActionGenAdvancesPos(t *testing.T) {
	fb := newActionTestFb()
	startX := fb.posX
	fb.addSendRestRequestActionGen(&ast.SendRestRequestStmt{
		Operation: ast.QualifiedName{Module: "M", Name: "S.Op"},
	})
	if fb.posX != startX+HorizontalSpacing {
		t.Fatalf("posX = %d, want %d", fb.posX, startX+HorizontalSpacing)
	}
}

// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.g2 — CALL JAVA ACTION / JAVASCRIPT ACTION adder tests (TDD).
//
// Both wrap a code-action call with parameter mappings carrying a
// polymorphic ParameterValue element. Offline tests cover the common
// path (BasicCodeActionParameterValue with the rendered Mendix
// expression). Backend-driven entity-type / microflow-type
// classification needs `fb.backend.ReadJavaActionByName` and is
// covered by integration tests under mdl/integration.

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func TestAddCallJavaActionActionGenSetsCallAndMappings(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CallJavaActionStmt{
		OutputVariable: "Result",
		ActionName:     ast.QualifiedName{Module: "Cust", Name: "DoThing"},
		Arguments: []ast.CallArgument{
			{Name: "Input", Value: &ast.VariableExpr{Name: "X"}},
		},
	}
	fb.addCallJavaActionActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.JavaActionCallAction)
	if act.JavaActionQualifiedName() != "Cust.DoThing" {
		t.Fatalf("java action QN = %q", act.JavaActionQualifiedName())
	}
	if act.OutputVariableName() != "Result" {
		t.Fatalf("output var = %q", act.OutputVariableName())
	}
	if !act.UseReturnVariable() {
		t.Fatal("UseReturnVariable must be true when output var set")
	}
	mappings := act.ParameterMappingsItems()
	if len(mappings) != 1 {
		t.Fatalf("mappings = %d, want 1", len(mappings))
	}
	pm := mappings[0].(*genMf.JavaActionParameterMapping)
	if pm.ParameterQualifiedName() != "Cust.DoThing.Input" {
		t.Fatalf("param QN = %q", pm.ParameterQualifiedName())
	}
	value, ok := pm.Value().(*genMf.BasicCodeActionParameterValue)
	if !ok {
		t.Fatalf("value = %T, want *BasicCodeActionParameterValue", pm.Value())
	}
	if value.Argument() != "$X" {
		t.Fatalf("value arg = %q", value.Argument())
	}
}

func TestAddCallJavaActionActionGenEmptyArgumentRendersAsBlankBasic(t *testing.T) {
	// Without backend resolution, an empty/null AST argument should
	// produce a BasicCodeActionParameterValue with Argument="" — the
	// "intentionally unbound" marker. Backend-aware classification
	// (`Argument: "empty"` for typed basic params) is deferred.
	fb := newActionTestFb()
	stmt := &ast.CallJavaActionStmt{
		ActionName: ast.QualifiedName{Module: "Cust", Name: "Act"},
		Arguments: []ast.CallArgument{
			{Name: "P", Value: &ast.LiteralExpr{Kind: ast.LiteralEmpty}},
		},
	}
	fb.addCallJavaActionActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.JavaActionCallAction)
	pm := act.ParameterMappingsItems()[0].(*genMf.JavaActionParameterMapping)
	value := pm.Value().(*genMf.BasicCodeActionParameterValue)
	if value.Argument() != "" {
		t.Fatalf("empty AST arg should produce blank Argument; got %q", value.Argument())
	}
}

func TestAddCallJavaActionActionGenNoOutputVariable(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CallJavaActionStmt{
		ActionName: ast.QualifiedName{Module: "M", Name: "Act"},
	}
	fb.addCallJavaActionActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.JavaActionCallAction)
	if act.OutputVariableName() != "" {
		t.Fatalf("output var = %q, want empty", act.OutputVariableName())
	}
	if act.UseReturnVariable() {
		t.Fatal("UseReturnVariable must be false when no output var")
	}
}

func TestAddCallJavaScriptActionActionGenSetsCallAndMappings(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CallJavaScriptActionStmt{
		OutputVariable: "Result",
		ActionName:     ast.QualifiedName{Module: "JS", Name: "DoIt"},
		Arguments: []ast.CallArgument{
			{Name: "Arg1", Value: &ast.LiteralExpr{Kind: ast.LiteralString, Value: "hello"}},
		},
	}
	fb.addCallJavaScriptActionActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.JavaScriptActionCallAction)
	if act.JavaScriptActionQualifiedName() != "JS.DoIt" {
		t.Fatalf("js action QN = %q", act.JavaScriptActionQualifiedName())
	}
	if act.OutputVariableName() != "Result" {
		t.Fatalf("output var = %q", act.OutputVariableName())
	}
	mappings := act.ParameterMappingsItems()
	if len(mappings) != 1 {
		t.Fatalf("mappings = %d, want 1", len(mappings))
	}
	pm := mappings[0].(*genMf.JavaScriptActionParameterMapping)
	if pm.ParameterQualifiedName() != "JS.DoIt.Arg1" {
		t.Fatalf("param QN = %q", pm.ParameterQualifiedName())
	}
	value, ok := pm.ParameterValue().(*genMf.BasicCodeActionParameterValue)
	if !ok {
		t.Fatalf("value = %T, want *BasicCodeActionParameterValue", pm.ParameterValue())
	}
	if value.Argument() != "'hello'" {
		t.Fatalf("value arg = %q", value.Argument())
	}
}

func TestAddCallJavaScriptActionActionGenContinueErrorHandling(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CallJavaScriptActionStmt{
		ActionName:    ast.QualifiedName{Module: "M", Name: "Act"},
		ErrorHandling: &ast.ErrorHandlingClause{Type: ast.ErrorHandlingContinue},
	}
	fb.addCallJavaScriptActionActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.JavaScriptActionCallAction)
	if act.ErrorHandlingType() != "Continue" {
		t.Fatalf("eh = %q, want Continue", act.ErrorHandlingType())
	}
}

func TestAddCallJavaActionActionGenAdvancesPos(t *testing.T) {
	fb := newActionTestFb()
	startX := fb.posX
	stmt := &ast.CallJavaActionStmt{
		ActionName: ast.QualifiedName{Module: "M", Name: "Act"},
	}
	fb.addCallJavaActionActionGen(stmt)
	if fb.posX != startX+HorizontalSpacing {
		t.Fatalf("posX = %d, want %d", fb.posX, startX+HorizontalSpacing)
	}
}

func TestIsEmptyJavaActionArgumentGenDetectsEmptyAndNull(t *testing.T) {
	cases := []struct {
		name string
		expr ast.Expression
		want bool
	}{
		{"empty literal", &ast.LiteralExpr{Kind: ast.LiteralEmpty}, true},
		{"null literal", &ast.LiteralExpr{Kind: ast.LiteralNull}, true},
		{"string literal", &ast.LiteralExpr{Kind: ast.LiteralString, Value: "x"}, false},
		{"variable", &ast.VariableExpr{Name: "X"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isEmptyJavaActionArgumentGen(tc.expr); got != tc.want {
				t.Fatalf("got %t, want %t", got, tc.want)
			}
		})
	}
}

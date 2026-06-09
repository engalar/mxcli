// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.g1 — CALL MICROFLOW / NANOFLOW adder tests (TDD).
//
// Both follow the same shape: the action wraps a nested *Call element
// (MicroflowCall / NanoflowCall) carrying the qualified name and a
// PartList of parameter mappings. Each ParameterMapping ties the
// callee's `<callee QN>.<param name>` to a Mendix expression
// (rendered from the AST argument value).

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func TestCheckOutputVarCollision_Duplicate(t *testing.T) {
	fb := &flowBuilderGen{
		declaredVars: map[string]string{"Result": "Unknown"},
		varTypes:     map[string]string{},
		errors:       nil,
	}

	collision := checkOutputVarCollision(fb, "Result")

	if !collision {
		t.Fatal("expected collision=true for duplicate variable, got false")
	}
	if len(fb.errors) == 0 {
		t.Fatal("expected error recorded, got none")
	}
	if !strings.Contains(fb.errors[0], "Result") {
		t.Errorf("error should mention the variable name, got: %q", fb.errors[0])
	}
}

func TestCheckOutputVarCollision_Fresh(t *testing.T) {
	fb := &flowBuilderGen{
		declaredVars: map[string]string{},
		varTypes:     map[string]string{},
		errors:       nil,
	}

	collision := checkOutputVarCollision(fb, "Result")

	if collision {
		t.Fatal("expected collision=false for fresh variable, got true")
	}
	if len(fb.errors) != 0 {
		t.Errorf("expected no error for fresh variable, got: %v", fb.errors)
	}
	if _, ok := fb.declaredVars["Result"]; !ok {
		t.Error("fresh output variable should be added to declaredVars")
	}
}

func TestCheckOutputVarCollision_Empty(t *testing.T) {
	fb := &flowBuilderGen{
		declaredVars: map[string]string{},
		varTypes:     map[string]string{},
		errors:       nil,
	}

	if checkOutputVarCollision(fb, "") {
		t.Fatal("expected collision=false for empty variable name")
	}
	if len(fb.errors) != 0 {
		t.Errorf("expected no error for empty variable, got: %v", fb.errors)
	}
}

func TestAddCallMicroflowActionGenSetsCallAndMappings(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CallMicroflowStmt{
		MicroflowName: ast.QualifiedName{Module: "Sales", Name: "ProcessOrder"},
		Arguments: []ast.CallArgument{
			{Name: "Order", Value: &ast.VariableExpr{Name: "Ord"}},
			{Name: "Now", Value: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true}},
		},
		OutputVariable: "Result",
	}
	fb.addCallMicroflowActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.MicroflowCallAction)
	if act.OutputVariableName() != "Result" {
		t.Fatalf("output var = %q", act.OutputVariableName())
	}
	if !act.UseReturnVariable() {
		t.Fatal("UseReturnVariable must be true when output var set")
	}
	call, ok := act.MicroflowCall().(*genMf.MicroflowCall)
	if !ok {
		t.Fatalf("MicroflowCall = %T, want *MicroflowCall", act.MicroflowCall())
	}
	if call.MicroflowQualifiedName() != "Sales.ProcessOrder" {
		t.Fatalf("microflow QN = %q", call.MicroflowQualifiedName())
	}
	mappings := call.ParameterMappingsItems()
	if len(mappings) != 2 {
		t.Fatalf("mappings = %d, want 2", len(mappings))
	}
	m1 := mappings[0].(*genMf.MicroflowCallParameterMapping)
	if m1.ParameterQualifiedName() != "Sales.ProcessOrder.Order" {
		t.Fatalf("param 1 QN = %q", m1.ParameterQualifiedName())
	}
	if m1.Argument() != "$Ord" {
		t.Fatalf("param 1 arg = %q", m1.Argument())
	}
	m2 := mappings[1].(*genMf.MicroflowCallParameterMapping)
	if m2.ParameterQualifiedName() != "Sales.ProcessOrder.Now" {
		t.Fatalf("param 2 QN = %q", m2.ParameterQualifiedName())
	}
	if m2.Argument() != "true" {
		t.Fatalf("param 2 arg = %q", m2.Argument())
	}
}

func TestAddCallMicroflowActionGenNoOutputVariable(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CallMicroflowStmt{
		MicroflowName: ast.QualifiedName{Module: "M", Name: "WF"},
	}
	fb.addCallMicroflowActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.MicroflowCallAction)
	if act.OutputVariableName() != "" {
		t.Fatalf("output var = %q, want empty", act.OutputVariableName())
	}
	if act.UseReturnVariable() {
		t.Fatal("UseReturnVariable must be false when no output var")
	}
}

func TestAddCallMicroflowActionGenDefaultErrorHandlingType(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CallMicroflowStmt{
		MicroflowName: ast.QualifiedName{Module: "M", Name: "WF"},
	}
	fb.addCallMicroflowActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.MicroflowCallAction)
	if act.ErrorHandlingType() != "Rollback" {
		t.Fatalf("default eh = %q, want Rollback", act.ErrorHandlingType())
	}
}

func TestAddCallMicroflowActionGenContinueErrorHandlingType(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CallMicroflowStmt{
		MicroflowName: ast.QualifiedName{Module: "M", Name: "WF"},
		ErrorHandling: &ast.ErrorHandlingClause{Type: ast.ErrorHandlingContinue},
	}
	fb.addCallMicroflowActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.MicroflowCallAction)
	if act.ErrorHandlingType() != "Continue" {
		t.Fatalf("eh = %q, want Continue", act.ErrorHandlingType())
	}
}

func TestAddCallNanoflowActionGenSetsCallAndMappings(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.CallNanoflowStmt{
		NanoflowName: ast.QualifiedName{Module: "UI", Name: "RefreshList"},
		Arguments: []ast.CallArgument{
			{Name: "List", Value: &ast.VariableExpr{Name: "Items"}},
		},
		OutputVariable: "Updated",
	}
	fb.addCallNanoflowActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.NanoflowCallAction)
	if act.OutputVariableName() != "Updated" {
		t.Fatalf("output var = %q", act.OutputVariableName())
	}
	if !act.UseReturnVariable() {
		t.Fatal("UseReturnVariable must be true when output var set")
	}
	call, ok := act.NanoflowCall().(*genMf.NanoflowCall)
	if !ok {
		t.Fatalf("NanoflowCall = %T, want *NanoflowCall", act.NanoflowCall())
	}
	if call.NanoflowQualifiedName() != "UI.RefreshList" {
		t.Fatalf("nanoflow QN = %q", call.NanoflowQualifiedName())
	}
	mappings := call.ParameterMappingsItems()
	if len(mappings) != 1 {
		t.Fatalf("mappings = %d, want 1", len(mappings))
	}
	m := mappings[0].(*genMf.NanoflowCallParameterMapping)
	if m.ParameterQualifiedName() != "UI.RefreshList.List" {
		t.Fatalf("param QN = %q", m.ParameterQualifiedName())
	}
	if m.Argument() != "$Items" {
		t.Fatalf("param arg = %q", m.Argument())
	}
}

func TestAddCallNanoflowActionGenDefaultsRollback(t *testing.T) {
	// Microflow + Nanoflow both default to "Rollback" error handling
	// — fb.isNanoflow controls the BUILDER context, not the inner
	// action's eh type, so a microflow CALLING a nanoflow keeps
	// "Rollback".
	fb := newActionTestFb()
	stmt := &ast.CallNanoflowStmt{
		NanoflowName: ast.QualifiedName{Module: "M", Name: "NF"},
	}
	fb.addCallNanoflowActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.NanoflowCallAction)
	if act.ErrorHandlingType() != "Rollback" {
		t.Fatalf("eh = %q, want Rollback", act.ErrorHandlingType())
	}
}

func TestAddCallMicroflowActionGenWrapsActivityAndAdvances(t *testing.T) {
	fb := newActionTestFb()
	startX := fb.posX
	stmt := &ast.CallMicroflowStmt{
		MicroflowName: ast.QualifiedName{Module: "M", Name: "WF"},
	}
	id := fb.addCallMicroflowActionGen(stmt)
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	if fb.posX != startX+HorizontalSpacing {
		t.Fatalf("posX = %d, want %d", fb.posX, startX+HorizontalSpacing)
	}
}

func TestAddCallMicroflowActionGenMissingMicroflowReportsError(t *testing.T) {
	// Without a microflowsRepo, microflowExistsGen falls through to
	// true (offline mode) — no error reported. With a repo whose
	// lookups all return empty, microflowExistsGen returns false and
	// the adder must surface a validation error.
	fb := newActionTestFb()
	fb.microflowsRepo = &fakeMicroflowRepo{}
	stmt := &ast.CallMicroflowStmt{
		MicroflowName: ast.QualifiedName{Module: "M", Name: "Missing"},
	}
	fb.addCallMicroflowActionGen(stmt)
	if len(fb.GetErrors()) == 0 {
		t.Fatal("expected validation error for missing microflow")
	}
}

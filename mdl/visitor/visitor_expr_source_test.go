// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// exprSourceOf walks the first body statement of the first microflow in prog
// and returns the Source field of any SourceExpr it finds.
func exprSourceOf(t *testing.T, mdlSrc string) string {
	t.Helper()
	prog, errs := Build(mdlSrc)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	// Find first microflow (statements[0] is CreateModuleStmt)
	var mf *ast.CreateMicroflowStmt
	for _, s := range prog.Statements {
		if m, ok := s.(*ast.CreateMicroflowStmt); ok {
			mf = m
			break
		}
	}
	if mf == nil {
		t.Fatal("no CreateMicroflowStmt found in program")
	}
	if len(mf.Body) == 0 {
		t.Fatal("microflow body is empty")
	}
	decl, ok := mf.Body[0].(*ast.DeclareStmt)
	if !ok {
		t.Fatalf("body[0] is %T, want *DeclareStmt", mf.Body[0])
	}
	se, ok := decl.InitialValue.(*ast.SourceExpr)
	if !ok {
		t.Fatalf("InitialValue is %T, want *SourceExpr", decl.InitialValue)
	}
	return se.Source
}

// TestExprSource_DivVarVar verifies that $x div $y preserves whitespace
// in the SourceExpr.Source field so that the exprcheck parser can lex it
// correctly as three separate tokens ($x, div, $y).
func TestExprSource_DivVarVar(t *testing.T) {
	src := `create module T;
create microflow T.F ($x: decimal, $y: decimal) returns decimal as $r
begin
  declare $r: decimal = $x div $y;
  return $r;
end;
/`
	source := exprSourceOf(t, src)
	if !strings.Contains(source, " ") {
		t.Errorf("Source = %q — whitespace stripped; exprcheck will mis-lex 'div' as part of adjacent variable", source)
	}
	if !strings.Contains(source, "div") {
		t.Errorf("Source = %q — 'div' keyword missing", source)
	}
}

// TestExprSource_ModVarVar verifies the same for $x mod $y.
func TestExprSource_ModVarVar(t *testing.T) {
	src := `create module T;
create microflow T.F ($x: integer, $y: integer) returns integer as $r
begin
  declare $r: integer = $x mod $y;
  return $r;
end;
/`
	source := exprSourceOf(t, src)
	if !strings.Contains(source, " ") {
		t.Errorf("Source = %q — whitespace stripped", source)
	}
	if !strings.Contains(source, "mod") {
		t.Errorf("Source = %q — 'mod' keyword missing", source)
	}
}

// TestExprSource_IfThenElse verifies that the inline if-then-else expression
// preserves whitespace so the exprcheck parser can find the 'if', 'then', 'else'
// keywords as separate tokens.
func TestExprSource_IfThenElse(t *testing.T) {
	src := `create module T;
create microflow T.F ($w: decimal) returns decimal as $fee
begin
  declare $fee: decimal = if $w < 1.0 then 0.0 else 5.0;
  return $fee;
end;
/`
	source := exprSourceOf(t, src)
	if !strings.Contains(source, "then") {
		t.Errorf("Source = %q — 'then' keyword missing; whitespace likely stripped", source)
	}
	if !strings.Contains(source, "else") {
		t.Errorf("Source = %q — 'else' keyword missing", source)
	}
	if !strings.Contains(source, " ") {
		t.Errorf("Source = %q — whitespace stripped", source)
	}
}

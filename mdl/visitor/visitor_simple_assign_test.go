// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// TestSimpleAssignStatement verifies that $x = expr (no SET keyword) is
// parsed and builds an MfSetStmt identical to SET $x = expr.
func TestSimpleAssignStatement_Variable(t *testing.T) {
	input := `create microflow Test.M () returns Nothing
{
  declare $count: integer = 0;
  $count = $count + 1;
}`

	prog, errs := Build(input)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	mf := prog.Statements[0].(*ast.CreateMicroflowStmt)
	if len(mf.Body) != 2 {
		t.Fatalf("body len = %d, want 2", len(mf.Body))
	}
	set, ok := mf.Body[1].(*ast.MfSetStmt)
	if !ok {
		t.Fatalf("body[1] = %T, want *ast.MfSetStmt", mf.Body[1])
	}
	if set.Target != "count" {
		t.Errorf("Target = %q, want %q", set.Target, "count")
	}
	if set.Value == nil {
		t.Error("Value is nil")
	}
}

// TestSimpleAssignStatement_AttributePath verifies $obj/Attr = expr form.
func TestSimpleAssignStatement_AttributePath(t *testing.T) {
	input := `create microflow Test.M ($order: Test.Order) returns Nothing
{
  $order/IsActive = true;
}`

	prog, errs := Build(input)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	mf := prog.Statements[0].(*ast.CreateMicroflowStmt)
	if len(mf.Body) != 1 {
		t.Fatalf("body len = %d, want 1", len(mf.Body))
	}
	set, ok := mf.Body[0].(*ast.MfSetStmt)
	if !ok {
		t.Fatalf("body[0] = %T, want *ast.MfSetStmt", mf.Body[0])
	}
	if set.Target == "" {
		t.Error("Target is empty")
	}
	if set.Value == nil {
		t.Error("Value is nil")
	}
}

// TestSimpleAssignStatement_ModernBodyForm verifies $x = expr inside {} body.
func TestSimpleAssignStatement_ModernBodyForm(t *testing.T) {
	input := `create microflow Test.M () returns Nothing {
  declare $r: integer = 0;
  $r = $r + 1;
}`

	prog, errs := Build(input)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	mf := prog.Statements[0].(*ast.CreateMicroflowStmt)
	if len(mf.Body) != 2 {
		t.Fatalf("body len = %d, want 2", len(mf.Body))
	}
	if _, ok := mf.Body[1].(*ast.MfSetStmt); !ok {
		t.Fatalf("body[1] = %T, want *ast.MfSetStmt", mf.Body[1])
	}
}

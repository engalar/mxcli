// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestQualifiedNameToXPath_EnumValue(t *testing.T) {
	// 3-part names (Module.EnumName.Value) should emit just the value in quotes
	expr := &ast.QualifiedNameExpr{
		QualifiedName: ast.QualifiedName{Module: "MyModule", Name: "ENUM_Status.Processing"},
	}
	got := qualifiedNameToXPath(expr)
	want := "'Processing'"
	if got != want {
		t.Errorf("qualifiedNameToXPath(%q) = %q, want %q", expr.QualifiedName.String(), got, want)
	}
}

func TestQualifiedNameToXPath_NonEnum(t *testing.T) {
	// 2-part names (Module.AssocName) should pass through as-is
	expr := &ast.QualifiedNameExpr{
		QualifiedName: ast.QualifiedName{Module: "MyModule", Name: "SomeAssoc"},
	}
	got := qualifiedNameToXPath(expr)
	want := "MyModule.SomeAssoc"
	if got != want {
		t.Errorf("qualifiedNameToXPath(%q) = %q, want %q", expr.QualifiedName.String(), got, want)
	}
}

func TestExpressionToXPath_EnumInComparison(t *testing.T) {
	// WHERE Status = Module.ENUM.Value should produce: Status = 'Value'
	expr := &ast.BinaryExpr{
		Left:     &ast.IdentifierExpr{Name: "Status"},
		Operator: "=",
		Right: &ast.QualifiedNameExpr{
			QualifiedName: ast.QualifiedName{Module: "BST", Name: "ComplianceStatus.Rectified"},
		},
	}
	got := expressionToXPath(expr)
	want := "Status = 'Rectified'"
	if got != want {
		t.Errorf("expressionToXPath = %q, want %q", got, want)
	}
}

func TestExpressionToXPath_StringLiteralPreserved(t *testing.T) {
	// WHERE Status = 'Pending' should stay as Status = 'Pending'
	expr := &ast.BinaryExpr{
		Left:     &ast.IdentifierExpr{Name: "Status"},
		Operator: "=",
		Right:    &ast.LiteralExpr{Value: "Pending", Kind: ast.LiteralString},
	}
	got := expressionToXPath(expr)
	want := "Status = 'Pending'"
	if got != want {
		t.Errorf("expressionToXPath = %q, want %q", got, want)
	}
}

func TestExpressionToString_QualifiedNameUnchanged(t *testing.T) {
	// In expression context, qualified names should remain as-is (correct for enum refs)
	expr := &ast.QualifiedNameExpr{
		QualifiedName: ast.QualifiedName{Module: "MyModule", Name: "ENUM_Status.Processing"},
	}
	got := expressionToString(expr)
	want := "MyModule.ENUM_Status.Processing"
	if got != want {
		t.Errorf("expressionToString = %q, want %q", got, want)
	}
}

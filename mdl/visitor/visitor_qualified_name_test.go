// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestBuildQualifiedNameThreeParts(t *testing.T) {
	// Bug #1: Module.Enum.Value was truncated to Module.Enum
	// The WHERE clause should preserve all 3 parts of an enum value reference.
	input := `CREATE MICROFLOW BST.Test ()
BEGIN
  RETRIEVE $Submissions FROM BST.ComplianceSubmission
    WHERE ComplianceResult = BST.ComplianceStatus.Rectified;
END;`

	prog, errs := Build(input)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("Parse error: %v", err)
		}
		return
	}

	if len(prog.Statements) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(prog.Statements))
	}

	stmt, ok := prog.Statements[0].(*ast.CreateMicroflowStmt)
	if !ok {
		t.Fatalf("Expected CreateMicroflowStmt, got %T", prog.Statements[0])
	}

	if len(stmt.Body) < 1 {
		t.Fatalf("Expected at least 1 body statement, got %d", len(stmt.Body))
	}

	retrieve, ok := stmt.Body[0].(*ast.RetrieveStmt)
	if !ok {
		t.Fatalf("Expected RetrieveStmt, got %T", stmt.Body[0])
	}

	if retrieve.Where == nil {
		t.Fatal("Expected WHERE clause, got nil")
	}

	// The WHERE clause should contain a QualifiedNameExpr with the full 3-part name
	// Walk the expression tree to find it
	found := findQualifiedNameExpr(retrieve.Where)
	if found == nil {
		t.Fatal("Expected QualifiedNameExpr in WHERE clause, not found")
	}

	got := found.QualifiedName.String()
	expected := "BST.ComplianceStatus.Rectified"
	if got != expected {
		t.Errorf("Enum value truncated: got %q, want %q", got, expected)
	}
}

func TestBuildQualifiedNameTwoParts(t *testing.T) {
	// Verify 2-part names still work correctly
	input := `CREATE MICROFLOW BST.Test ()
BEGIN
  RETRIEVE $Items FROM BST.Item
    WHERE Status = BST.SomeAssoc;
END;`

	prog, errs := Build(input)
	if len(errs) > 0 {
		t.Fatalf("Parse error: %v", errs[0])
	}

	stmt := prog.Statements[0].(*ast.CreateMicroflowStmt)
	retrieve := stmt.Body[0].(*ast.RetrieveStmt)

	found := findQualifiedNameExpr(retrieve.Where)
	if found == nil {
		t.Fatal("Expected QualifiedNameExpr in WHERE clause")
	}

	got := found.QualifiedName.String()
	if got != "BST.SomeAssoc" {
		t.Errorf("Two-part name broken: got %q, want %q", got, "BST.SomeAssoc")
	}
}

// findQualifiedNameExpr recursively searches an expression tree for a QualifiedNameExpr.
func findQualifiedNameExpr(expr ast.Expression) *ast.QualifiedNameExpr {
	switch e := expr.(type) {
	case *ast.QualifiedNameExpr:
		return e
	case *ast.BinaryExpr:
		if found := findQualifiedNameExpr(e.Left); found != nil {
			return found
		}
		return findQualifiedNameExpr(e.Right)
	case *ast.ParenExpr:
		return findQualifiedNameExpr(e.Inner)
	default:
		return nil
	}
}

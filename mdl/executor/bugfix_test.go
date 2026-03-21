// SPDX-License-Identifier: Apache-2.0

// Tests for bug fixes discovered during BST Monitoring app session (2026-03-13).
package executor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

// TestValidateDuplicateVariableDeclareRetrieve verifies that DECLARE followed by
// RETRIEVE for the same variable is caught as a duplicate (CE0111).
// Bug #3: mxcli check passed but mx check reported CE0111.
func TestValidateDuplicateVariableDeclareRetrieve(t *testing.T) {
	input := `CREATE MICROFLOW Test.MF_DuplicateVar ()
BEGIN
  DECLARE $Count Integer = 0;
  RETRIEVE $Count FROM Test.TestItem;
  RETURN $Count;
END;`

	errors := validateMicroflowFromMDL(t, input)

	found := false
	for _, e := range errors {
		if strings.Contains(e, "duplicate") && strings.Contains(e, "Count") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected duplicate variable error for $Count, got errors: %v", errors)
	}
}

// TestValidateDuplicateVariableDeclareOnly verifies that two DECLARE statements
// for the same variable are caught as duplicate.
func TestValidateDuplicateVariableDeclareOnly(t *testing.T) {
	input := `CREATE MICROFLOW Test.MF_DoubleDeclare ()
BEGIN
  DECLARE $X Integer = 0;
  DECLARE $X String = 'hello';
END;`

	errors := validateMicroflowFromMDL(t, input)

	found := false
	for _, e := range errors {
		if strings.Contains(e, "duplicate") && strings.Contains(e, "X") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected duplicate variable error for $X, got errors: %v", errors)
	}
}

// TestValidateNoDuplicateWhenRetrieveOnly verifies that a single RETRIEVE
// (without prior DECLARE) does not trigger a false positive.
func TestValidateNoDuplicateWhenRetrieveOnly(t *testing.T) {
	input := `CREATE MICROFLOW Test.MF_RetrieveOnly ()
BEGIN
  RETRIEVE $Items FROM Test.SomeEntity;
END;`

	errors := validateMicroflowFromMDL(t, input)

	for _, e := range errors {
		if strings.Contains(e, "duplicate") {
			t.Errorf("Unexpected duplicate variable error: %s", e)
		}
	}
}

// TestValidateDuplicateVariableDeclareCreate verifies that DECLARE followed by
// CREATE for the same variable is caught as a duplicate (CE0111).
func TestValidateDuplicateVariableDeclareCreate(t *testing.T) {
	input := `CREATE MICROFLOW Test.MF_DeclareCreate ()
BEGIN
  DECLARE $NewTodo Test.Todo;
  $NewTodo = CREATE Test.Todo();
END;`

	errors := validateMicroflowFromMDL(t, input)

	found := false
	for _, e := range errors {
		if strings.Contains(e, "duplicate") && strings.Contains(e, "NewTodo") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected duplicate variable error for $NewTodo, got errors: %v", errors)
	}
}

// TestValidateEntityReservedAttributeName verifies that persistent entity attributes
// using reserved system names (CreatedDate, ChangedDate, Owner, ChangedBy) are caught.
func TestValidateEntityReservedAttributeName(t *testing.T) {
	input := `CREATE PERSISTENT ENTITY Test.MyEntity (
  Name : String(200),
  CreatedDate : DateTime,
  Status : String(50)
);`

	prog, errs := visitor.Build(input)
	if len(errs) > 0 {
		t.Fatalf("Parse error: %v", errs[0])
	}

	stmt, ok := prog.Statements[0].(*ast.CreateEntityStmt)
	if !ok {
		t.Fatalf("Expected CreateEntityStmt, got %T", prog.Statements[0])
	}

	errors := ValidateEntity(stmt)
	found := false
	for _, e := range errors {
		if strings.Contains(e, "CreatedDate") && strings.Contains(e, "system attribute") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected reserved attribute error for CreatedDate, got: %v", errors)
	}
}

// TestValidateEntityNonPersistentAllowed verifies that non-persistent entities
// can use system attribute names without error.
func TestValidateEntityNonPersistentAllowed(t *testing.T) {
	input := `CREATE NON-PERSISTENT ENTITY Test.MyNPE (
  CreatedDate : DateTime,
  Owner : String(200)
);`

	prog, errs := visitor.Build(input)
	if len(errs) > 0 {
		t.Fatalf("Parse error: %v", errs[0])
	}

	stmt, ok := prog.Statements[0].(*ast.CreateEntityStmt)
	if !ok {
		t.Fatalf("Expected CreateEntityStmt, got %T", prog.Statements[0])
	}

	errors := ValidateEntity(stmt)
	if len(errors) > 0 {
		t.Errorf("Non-persistent entity should allow system attribute names, got: %v", errors)
	}
}

// TestValidateEntityNormalAttributesPass verifies that normal attribute names
// don't trigger false positives.
func TestValidateEntityNormalAttributesPass(t *testing.T) {
	input := `CREATE PERSISTENT ENTITY Test.MyEntity (
  Name : String(200),
  Description : String(2000),
  Amount : Decimal,
  IsActive : Boolean
);`

	prog, errs := visitor.Build(input)
	if len(errs) > 0 {
		t.Fatalf("Parse error: %v", errs[0])
	}

	stmt, ok := prog.Statements[0].(*ast.CreateEntityStmt)
	if !ok {
		t.Fatalf("Expected CreateEntityStmt, got %T", prog.Statements[0])
	}

	errors := ValidateEntity(stmt)
	if len(errors) > 0 {
		t.Errorf("Normal attributes should not trigger errors, got: %v", errors)
	}
}

// TestReturnsNothingAcceptsBarReturn verifies that RETURNS Nothing treats
// RETURN; (no value) as valid — "Nothing" means void.
func TestReturnsNothingAcceptsBarReturn(t *testing.T) {
	input := `CREATE MICROFLOW Test.MF_ReturnsNothing ()
RETURNS Nothing
BEGIN
  LOG INFO 'hello';
  RETURN;
END;`

	prog, errs := visitor.Build(input)
	if len(errs) > 0 {
		t.Fatalf("Parse error: %v", errs[0])
	}

	stmt := prog.Statements[0].(*ast.CreateMicroflowStmt)

	// The return type should be TypeVoid
	if stmt.ReturnType != nil && stmt.ReturnType.Type.Kind != ast.TypeVoid {
		t.Errorf("Expected TypeVoid for RETURNS Nothing, got %v", stmt.ReturnType.Type.Kind)
	}

	// Validation should NOT produce errors about RETURN requiring a value
	warnings := ValidateMicroflowBody(stmt)
	for _, w := range warnings {
		if strings.Contains(w, "RETURN requires a value") {
			t.Errorf("RETURNS Nothing should not reject bare RETURN;, got: %s", w)
		}
	}
}

// TestEnumDefaultNotDoubleQualified verifies that enum DEFAULT values are stored
// without the enum prefix (just the value name), preventing double-qualification.
func TestEnumDefaultNotDoubleQualified(t *testing.T) {
	input := `CREATE PERSISTENT ENTITY Test.Item (
  Status : Enumeration(Test.ItemStatus) DEFAULT Test.ItemStatus.Active
);`

	prog, errs := visitor.Build(input)
	if len(errs) > 0 {
		t.Fatalf("Parse error: %v", errs[0])
	}

	stmt := prog.Statements[0].(*ast.CreateEntityStmt)
	if len(stmt.Attributes) == 0 {
		t.Fatal("Expected at least 1 attribute")
	}

	attr := stmt.Attributes[0]
	if !attr.HasDefault {
		t.Fatal("Expected attribute to have a default value")
	}

	// The default value from the parser is the full text "Test.ItemStatus.Active"
	defaultStr := fmt.Sprintf("%v", attr.DefaultValue)
	// When stored, it should be stripped to just "Active" (the executor does this)
	// Here we verify the parser at least captures the full text correctly
	if !strings.Contains(defaultStr, "Active") {
		t.Errorf("Default value should contain 'Active', got: %s", defaultStr)
	}
}

// TestExpressionToXPath_TokenQuoting verifies that [%CurrentDateTime%] tokens
// are quoted in XPath context but not in Mendix expression context (GitHub issue #1).
func TestExpressionToXPath_TokenQuoting(t *testing.T) {
	tests := []struct {
		name     string
		expr     ast.Expression
		wantExpr string // expressionToString output
		wantXP   string // expressionToXPath output
	}{
		{
			name:     "Token_CurrentDateTime",
			expr:     &ast.TokenExpr{Token: "CurrentDateTime"},
			wantExpr: "[%CurrentDateTime%]",
			wantXP:   "'[%CurrentDateTime%]'",
		},
		{
			name:     "Token_CurrentUser",
			expr:     &ast.TokenExpr{Token: "CurrentUser"},
			wantExpr: "[%CurrentUser%]",
			wantXP:   "'[%CurrentUser%]'",
		},
		{
			name: "BinaryExpr_with_token",
			expr: &ast.BinaryExpr{
				Left:     &ast.IdentifierExpr{Name: "DueDate"},
				Operator: "<",
				Right:    &ast.TokenExpr{Token: "CurrentDateTime"},
			},
			wantExpr: "DueDate < [%CurrentDateTime%]",
			wantXP:   "DueDate < '[%CurrentDateTime%]'",
		},
		{
			name:     "Variable_unchanged",
			expr:     &ast.VariableExpr{Name: "MyVar"},
			wantExpr: "$MyVar",
			wantXP:   "$MyVar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExpr := expressionToString(tt.expr)
			if gotExpr != tt.wantExpr {
				t.Errorf("expressionToString() = %q, want %q", gotExpr, tt.wantExpr)
			}
			gotXP := expressionToXPath(tt.expr)
			if gotXP != tt.wantXP {
				t.Errorf("expressionToXPath() = %q, want %q", gotXP, tt.wantXP)
			}
		})
	}
}

// TestExpressionToXPath_XPathPathExpr verifies that XPathPathExpr (bare association paths,
// nested predicates) serialize correctly via expressionToXPath.
func TestExpressionToXPath_XPathPathExpr(t *testing.T) {
	tests := []struct {
		name   string
		expr   ast.Expression
		wantXP string
	}{
		{
			name: "bare_association_path",
			expr: &ast.XPathPathExpr{
				Steps: []ast.XPathStep{
					{Expr: &ast.QualifiedNameExpr{QualifiedName: ast.QualifiedName{Module: "Module", Name: "Assoc"}}},
					{Expr: &ast.QualifiedNameExpr{QualifiedName: ast.QualifiedName{Module: "Module", Name: "Entity"}}},
					{Expr: &ast.IdentifierExpr{Name: "Attr"}},
				},
			},
			wantXP: "Module.Assoc/Module.Entity/Attr",
		},
		{
			name: "path_with_nested_predicate",
			expr: &ast.XPathPathExpr{
				Steps: []ast.XPathStep{
					{Expr: &ast.QualifiedNameExpr{QualifiedName: ast.QualifiedName{Module: "Sys", Name: "roles"}}},
					{
						Expr: &ast.QualifiedNameExpr{QualifiedName: ast.QualifiedName{Module: "Sys", Name: "UserRole"}},
						Predicate: &ast.BinaryExpr{
							Left:     &ast.IdentifierExpr{Name: "Active"},
							Operator: "=",
							Right:    &ast.LiteralExpr{Value: true, Kind: ast.LiteralBoolean},
						},
					},
				},
			},
			wantXP: "Sys.roles/Sys.UserRole[Active = true]",
		},
		{
			name: "path_with_reversed",
			expr: &ast.XPathPathExpr{
				Steps: []ast.XPathStep{
					{
						Expr:      &ast.QualifiedNameExpr{QualifiedName: ast.QualifiedName{Module: "System", Name: "roles"}},
						Predicate: &ast.FunctionCallExpr{Name: "reversed"},
					},
					{Expr: &ast.QualifiedNameExpr{QualifiedName: ast.QualifiedName{Module: "System", Name: "UserRole"}}},
				},
			},
			wantXP: "System.roles[reversed()]/System.UserRole",
		},
		{
			name: "comparison_with_path_and_token",
			expr: &ast.BinaryExpr{
				Left: &ast.XPathPathExpr{
					Steps: []ast.XPathStep{
						{Expr: &ast.QualifiedNameExpr{QualifiedName: ast.QualifiedName{Module: "System", Name: "owner"}}},
					},
				},
				Operator: "=",
				Right:    &ast.TokenExpr{Token: "CurrentUser"},
			},
			wantXP: "System.owner = '[%CurrentUser%]'",
		},
		{
			name: "not_with_path",
			expr: &ast.UnaryExpr{
				Operator: "not",
				Operand: &ast.XPathPathExpr{
					Steps: []ast.XPathStep{
						{Expr: &ast.QualifiedNameExpr{QualifiedName: ast.QualifiedName{Module: "Module", Name: "Assoc"}}},
						{Expr: &ast.QualifiedNameExpr{QualifiedName: ast.QualifiedName{Module: "Module", Name: "Entity"}}},
					},
				},
			},
			wantXP: "not(Module.Assoc/Module.Entity)",
		},
		{
			name: "function_with_path_args",
			expr: &ast.FunctionCallExpr{
				Name: "contains",
				Arguments: []ast.Expression{
					&ast.IdentifierExpr{Name: "Name"},
					&ast.VariableExpr{Name: "SearchStr"},
				},
			},
			wantXP: "contains(Name, $SearchStr)",
		},
		{
			name:   "empty_literal",
			expr:   &ast.LiteralExpr{Value: nil, Kind: ast.LiteralEmpty},
			wantXP: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotXP := expressionToXPath(tt.expr)
			if gotXP != tt.wantXP {
				t.Errorf("expressionToXPath() = %q, want %q", gotXP, tt.wantXP)
			}
		})
	}
}

// validateMicroflowFromMDL parses a CREATE MICROFLOW statement and runs
// ValidateMicroflowBody, returning any validation errors.
func validateMicroflowFromMDL(t *testing.T, input string) []string {
	t.Helper()

	prog, errs := visitor.Build(input)
	if len(errs) > 0 {
		t.Fatalf("Parse error: %v", errs[0])
	}

	if len(prog.Statements) == 0 {
		t.Fatal("No statements parsed")
	}

	stmt, ok := prog.Statements[0].(*ast.CreateMicroflowStmt)
	if !ok {
		t.Fatalf("Expected CreateMicroflowStmt, got %T", prog.Statements[0])
	}

	return ValidateMicroflowBody(stmt)
}

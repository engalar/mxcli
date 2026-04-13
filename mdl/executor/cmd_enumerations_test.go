package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestValidateEnumeration_ReservedWords(t *testing.T) {
	tests := []struct {
		name       string
		values     []string
		wantErrors int
	}{
		{
			name:       "Empty is a Mendix reserved word",
			values:     []string{"Empty"},
			wantErrors: 1,
		},
		{
			name:       "Owner, Type, Object are reserved",
			values:     []string{"Owner", "Type", "Object"},
			wantErrors: 3,
		},
		{
			name:       "case insensitive: empty, EMPTY",
			values:     []string{"empty", "EMPTY"},
			wantErrors: 2,
		},
		{
			name:       "Java reserved: class, return, void",
			values:     []string{"class", "return", "void"},
			wantErrors: 3,
		},
		{
			name:       "non-reserved keywords are allowed",
			values:     []string{"Open", "Closed", "Data", "Filter", "Action"},
			wantErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var enumValues []ast.EnumValue
			for _, v := range tt.values {
				enumValues = append(enumValues, ast.EnumValue{Name: v})
			}
			stmt := &ast.CreateEnumerationStmt{
				Name:   ast.QualifiedName{Module: "Test", Name: "TestEnum"},
				Values: enumValues,
			}
			violations := ValidateEnumeration(stmt)
			if len(violations) != tt.wantErrors {
				t.Errorf("got %d violations, want %d", len(violations), tt.wantErrors)
				for _, v := range violations {
					t.Logf("  violation: %s", v.Message)
				}
			}
		})
	}
}

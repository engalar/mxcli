// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestValidateMicroflow_DollarPrefixParam_ReturnsError(t *testing.T) {
	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "Mod", Name: "Flow"},
		Parameters: []ast.MicroflowParam{
			{Name: "$BadName", Type: ast.DataType{Kind: ast.TypeString}},
		},
	}
	violations := ValidateMicroflow(stmt)
	found := false
	for _, v := range violations {
		if strings.Contains(v.Message, "$BadName") &&
			strings.Contains(v.Message, "must not include '$'") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected violation about '$' prefix in parameter name, got: %v", violations)
	}
}

func TestValidateMicroflow_NoDollarParam_NoError(t *testing.T) {
	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "Mod", Name: "Flow"},
		Parameters: []ast.MicroflowParam{
			{Name: "GoodName", Type: ast.DataType{Kind: ast.TypeString}},
		},
	}
	violations := ValidateMicroflow(stmt)
	for _, v := range violations {
		if strings.Contains(v.Message, "must not include '$'") {
			t.Errorf("unexpected $ violation for param without dollar: %v", v)
		}
	}
}

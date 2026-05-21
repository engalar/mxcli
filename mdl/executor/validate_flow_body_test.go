// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestValidateCE7247_SetOnListVariable(t *testing.T) {
	body := []ast.MicroflowStatement{
		&ast.CreateListStmt{
			Variable:   "ResultList",
			EntityType: ast.QualifiedName{Module: "M", Name: "Entity"},
		},
		&ast.MfSetStmt{
			Target: "ResultList",
			Value:  &ast.LiteralExpr{Value: "empty"},
		},
	}
	errs := validateFlowBody(nil, body)
	if len(errs) == 0 {
		t.Error("expected CE7247 error for set on list variable, got none")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e, "CE7247") || strings.Contains(e, "list") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected CE7247 mention in errors, got: %v", errs)
	}
}

func TestValidateCE0111_BothBranches(t *testing.T) {
	body := []ast.MicroflowStatement{
		&ast.IfStmt{
			Condition: &ast.LiteralExpr{Value: "true"},
			ThenBody: []ast.MicroflowStatement{
				&ast.CreateObjectStmt{
					Variable:   "Result",
					EntityType: ast.QualifiedName{Module: "M", Name: "CustomerBasicDto"},
				},
			},
			ElseBody: []ast.MicroflowStatement{
				&ast.CreateObjectStmt{
					Variable:   "Result",
					EntityType: ast.QualifiedName{Module: "M", Name: "CustomerBasicDto"},
				},
			},
		},
	}
	errs := validateFlowBody(nil, body)
	if len(errs) == 0 {
		t.Error("expected CE0111 for duplicate $Result in if/else branches, got none")
	}
}

func TestValidateCE0111_FlatScope_StillCaught(t *testing.T) {
	body := []ast.MicroflowStatement{
		&ast.CreateObjectStmt{Variable: "Result", EntityType: ast.QualifiedName{Module: "M", Name: "Dto"}},
		&ast.CreateObjectStmt{Variable: "Result", EntityType: ast.QualifiedName{Module: "M", Name: "Dto"}},
	}
	errs := validateFlowBody(nil, body)
	if len(errs) == 0 {
		t.Error("flat-scope CE0111 regression: duplicate $Result not detected")
	}
}

func TestValidateCE0111_ParameterAndBodyVariable_SameName(t *testing.T) {
	params := []ast.MicroflowParam{
		{Name: "Result", Type: ast.DataType{EntityRef: &ast.QualifiedName{Module: "M", Name: "Dto"}}},
	}
	body := []ast.MicroflowStatement{
		&ast.CreateObjectStmt{
			Variable:   "Result",
			EntityType: ast.QualifiedName{Module: "M", Name: "Dto"},
		},
	}
	errs := validateFlowBody(params, body)
	if len(errs) == 0 {
		t.Error("expected CE0111 when body variable shadows parameter name, got none")
	}
}

func TestValidateCE7247_SetOnPrimitiveVariable_NoError(t *testing.T) {
	body := []ast.MicroflowStatement{
		&ast.DeclareStmt{
			Variable: "Counter",
			Type:     ast.DataType{Kind: ast.TypeInteger},
		},
		&ast.MfSetStmt{
			Target: "Counter",
			Value:  &ast.LiteralExpr{Kind: ast.LiteralInteger, Value: int64(1)},
		},
	}
	errs := validateFlowBody(nil, body)
	// no CE7247 expected
	for _, e := range errs {
		if strings.Contains(e, "CE7247") {
			t.Errorf("unexpected CE7247 for primitive variable: %v", errs)
		}
	}
}

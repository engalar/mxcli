// SPDX-License-Identifier: Apache-2.0

package exprcheck

import "testing"

func TestAST_NodeTypesImplement(t *testing.T) {
	nodes := []RobustExpr{
		&StringLit{Value: "x"},
		&NumberLit{Value: "1", Kind: KindInteger},
		&BoolLit{Value: true},
		&EmptyExpr{},
		&VariableExpr{Name: "x"},
		&AttributePathExpr{Variable: "x", Path: []string{"a"}},
		&QNameExpr{Module: "M", Name: "E", Sub: "V"},
		&CallExpr{Name: "length"},
		&BinExpr{Op: "+", L: &StringLit{}, R: &StringLit{}},
		&UnaryExpr{Op: "-", Operand: &NumberLit{Value: "1"}},
		&ParenExpr{Inner: &BoolLit{}},
		&IfThenElseExpr{},
		&TokenExpr{Token: "CurrentDateTime"},
		&ConstantRef{QName: "M.C"},
		&RecoveredExpr{SourceFragment: "@@@"},
	}
	if len(nodes) == 0 {
		t.Fatal("no nodes")
	}
}

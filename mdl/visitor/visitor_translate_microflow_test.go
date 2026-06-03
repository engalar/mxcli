// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestTranslateMicroflowStatement_Parse(t *testing.T) {
	input := `TRANSLATE MICROFLOW MyModule.ACT_Save IN zh_CN
		SET ShowMessage[0].message = '已保存',
		    ShowMessage[1].message = '失败';`
	prog, errs := Build(input)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("parse error: %v", err)
		}
		return
	}
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}
	stmt, ok := prog.Statements[0].(*ast.TranslateMicroflowStmt)
	if !ok {
		t.Fatalf("expected TranslateMicroflowStmt, got %T", prog.Statements[0])
	}
	if stmt.QName.Module != "MyModule" || stmt.QName.Name != "ACT_Save" {
		t.Errorf("qname = %s", stmt.QName.String())
	}
	if stmt.Lang != "zh_CN" {
		t.Errorf("lang = %q", stmt.Lang)
	}
	if len(stmt.Ops) != 2 {
		t.Fatalf("expected 2 ops, got %d", len(stmt.Ops))
	}
	if stmt.Ops[0].ActionType != "ShowMessage" || stmt.Ops[0].Index != 0 ||
		stmt.Ops[0].Property != "message" || stmt.Ops[0].Text != "已保存" {
		t.Errorf("op[0] = %+v", stmt.Ops[0])
	}
	if stmt.Ops[1].Index != 1 || stmt.Ops[1].Text != "失败" {
		t.Errorf("op[1] = %+v", stmt.Ops[1])
	}
}

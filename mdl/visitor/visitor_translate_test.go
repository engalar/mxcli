// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestTranslateStatement_Page(t *testing.T) {
	input := `TRANSLATE PAGE MyModule.Home IN zh_CN
		SET Button_Submit.caption = '提交',
		    title = '首页';`
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
	stmt, ok := prog.Statements[0].(*ast.TranslateStmt)
	if !ok {
		t.Fatalf("expected TranslateStmt, got %T", prog.Statements[0])
	}
	if stmt.DocType != "PAGE" {
		t.Errorf("expected DocType PAGE, got %q", stmt.DocType)
	}
	if stmt.QName.Module != "MyModule" || stmt.QName.Name != "Home" {
		t.Errorf("expected MyModule.Home, got %s", stmt.QName.String())
	}
	if stmt.Lang != "zh_CN" {
		t.Errorf("expected lang zh_CN, got %q", stmt.Lang)
	}
	if len(stmt.Ops) != 2 {
		t.Fatalf("expected 2 ops, got %d", len(stmt.Ops))
	}
	if stmt.Ops[0].Path != "Button_Submit.caption" || stmt.Ops[0].Text != "提交" {
		t.Errorf("op[0] = %+v", stmt.Ops[0])
	}
	if stmt.Ops[1].Path != "title" || stmt.Ops[1].Text != "首页" {
		t.Errorf("op[1] = %+v", stmt.Ops[1])
	}
}

func TestTranslateStatement_Snippet(t *testing.T) {
	input := `TRANSLATE SNIPPET MyModule.Header IN nl_NL SET Label_X.caption = 'Hallo';`
	prog, errs := Build(input)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("parse error: %v", err)
		}
		return
	}
	stmt, ok := prog.Statements[0].(*ast.TranslateStmt)
	if !ok {
		t.Fatalf("expected TranslateStmt, got %T", prog.Statements[0])
	}
	if stmt.DocType != "SNIPPET" {
		t.Errorf("expected SNIPPET, got %q", stmt.DocType)
	}
}

// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestDescribeTranslations_WithLang(t *testing.T) {
	prog, errs := Build(`DESCRIBE TRANSLATIONS MyModule.Home IN zh_CN;`)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("Parse error: %v", err)
		}
		return
	}
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}
	stmt, ok := prog.Statements[0].(*ast.DescribeTranslationsStmt)
	if !ok {
		t.Fatalf("expected DescribeTranslationsStmt, got %T", prog.Statements[0])
	}
	if stmt.QName.Module != "MyModule" || stmt.QName.Name != "Home" {
		t.Errorf("expected MyModule.Home, got %s", stmt.QName.String())
	}
	if stmt.Lang != "zh_CN" {
		t.Errorf("expected lang zh_CN, got %q", stmt.Lang)
	}
}

func TestDescribeTranslations_NoLang(t *testing.T) {
	prog, errs := Build(`DESCRIBE TRANSLATIONS MyModule.Home;`)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("Parse error: %v", err)
		}
		return
	}
	stmt, ok := prog.Statements[0].(*ast.DescribeTranslationsStmt)
	if !ok {
		t.Fatalf("expected DescribeTranslationsStmt, got %T", prog.Statements[0])
	}
	if stmt.Lang != "" {
		t.Errorf("expected empty lang, got %q", stmt.Lang)
	}
}

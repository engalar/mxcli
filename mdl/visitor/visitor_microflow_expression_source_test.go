// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestBuildExpression_WrapsWithSource(t *testing.T) {
	src := `
CREATE MICROFLOW M.F ()
RETURNS Boolean AS $r
{
    DECLARE $r Boolean = false;
    IF $r = true THEN SET $r = false; END IF;
    RETURN $r;
}
`
	prog, errs := Build(src)
	if len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	var found bool
	for _, st := range prog.Statements {
		mf, ok := st.(*ast.CreateMicroflowStmt)
		if !ok {
			continue
		}
		for _, body := range mf.Body {
			if ifs, ok := body.(*ast.IfStmt); ok {
				if se, ok := ifs.Condition.(*ast.SourceExpr); ok {
					if se.Source != "$r = true" && se.Source != "$r=true" {
						t.Errorf("source = %q, want $r = true or $r=true", se.Source)
					}
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("IF condition was not wrapped in *ast.SourceExpr")
	}
}

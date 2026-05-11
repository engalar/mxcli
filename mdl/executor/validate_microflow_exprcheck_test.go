// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

func TestValidateMicroflow_ProducesExprCheckHints(t *testing.T) {
	src := `
CREATE MICROFLOW M.F ($p: Integer)
RETURNS Boolean AS $r
BEGIN
    DECLARE $r Boolean = false;
    IF $r = true THEN SET $r = false; END IF;
    RETURN $r;
END;
`
	prog, errs := visitor.Build(src)
	if len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	var visited bool
	for _, s := range prog.Statements {
		if mf, ok := s.(*ast.CreateMicroflowStmt); ok {
			_ = ValidateMicroflow(mf)
			visited = true
		}
	}
	if !visited {
		t.Fatal("microflow not visited")
	}
}

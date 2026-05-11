// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/linter"
)

func TestCheckAdapter_ConvertsHintsToViolations(t *testing.T) {
	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "M", Name: "F"},
		Body: []ast.MicroflowStatement{},
	}
	v := NewCheckAdapter(nil)
	out := v.CheckMicroflow(stmt)
	var got []linter.Violation = out.AsViolations()
	if len(got) != 0 {
		t.Errorf("empty microflow should produce 0 violations, got %d", len(got))
	}
}

// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
)

// TestValidateMicroflowReferences_DetectsMissingEntityInDeclareStmt verifies
// that mxcli check --references catches entity references in DECLARE statements
// where the referenced entity does not exist in the project.
//
// Regression: flowRefCollector.collectFromStatements did not handle
// *ast.DeclareStmt, so declare $X list of Missing.Entity = empty silently
// passed reference validation and produced invalid BSON (CE1613 at load time).
func TestValidateMicroflowReferences_DetectsMissingEntityInDeclareStmt(t *testing.T) {
	moduleID := model.ID("module-1")
	backend := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{{
				BaseElement: model.BaseElement{ID: moduleID},
				Name:        "SyntheticAudit",
			}}, nil
		},
	}
	ctx, _ := newMockCtx(t, withBackend(backend))

	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "SyntheticAudit", Name: "ListOrders"},
		Body: []ast.MicroflowStatement{
			&ast.DeclareStmt{
				Variable: "orders",
				Type: ast.DataType{
					Kind:      ast.TypeListOf,
					EntityRef: &ast.QualifiedName{Module: "MissingMod", Name: "MissingEntity"},
				},
			},
		},
	}

	err := validate(ctx, stmt)
	if err == nil {
		t.Fatal("expected reference error for missing entity in DECLARE statement")
	}
	if !strings.Contains(err.Error(), "entity not found: MissingMod.MissingEntity") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "referenced by declare") {
		t.Fatalf("error should mention 'declare' source: %v", err)
	}
}

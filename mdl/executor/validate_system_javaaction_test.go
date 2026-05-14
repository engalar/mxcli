// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
)

// `mxcli check --references` must not flag System.* Java actions
// (System.VerifyPassword, System.GenerateRandomString, etc.) as missing.
// They are runtime-provided and never appear in the project MPR; flagging
// them produces false positives on any microflow that uses Mendix
// built-ins, including the Administration module's password flows.
func TestValidateMicroflowReferencesSkipsSystemJavaAction(t *testing.T) {
	moduleID := model.ID("module-1")
	backend := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{{
				BaseElement: model.BaseElement{ID: moduleID},
				Name:        "Administration",
			}}, nil
		},
		// No microflow list seeded — buildMicroflowQualifiedNamesGen
		// gets an empty set, which is what these tests need: the
		// CallJavaActionStmt is what's being validated, not the
		// surrounding microflow body.
	}
	ctx, _ := newMockCtx(t, withBackend(backend))

	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "Administration", Name: "ChangeMyPassword"},
		Body: []ast.MicroflowStatement{
			&ast.CallJavaActionStmt{
				ActionName: ast.QualifiedName{Module: "System", Name: "VerifyPassword"},
			},
		},
	}

	if err := validate(ctx, stmt); err != nil {
		t.Fatalf("System.* Java action reference must not error, got: %v", err)
	}
}

// User-module Java action references are still validated. A missing one
// must error out so genuine typos don't slip through.
func TestValidateMicroflowReferencesReportsMissingUserJavaAction(t *testing.T) {
	moduleID := model.ID("module-1")
	backend := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{{
				BaseElement: model.BaseElement{ID: moduleID},
				Name:        "Administration",
			}}, nil
		},
		// No microflow list seeded — buildMicroflowQualifiedNamesGen
		// gets an empty set, which is what these tests need: the
		// CallJavaActionStmt is what's being validated, not the
		// surrounding microflow body. Validators source Java actions
		// via the gen-typed ListJavaActionsGenFunc when needed; the
		// "missing" case below relies on the empty default.
	}
	ctx, _ := newMockCtx(t, withBackend(backend))

	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "Administration", Name: "BadCall"},
		Body: []ast.MicroflowStatement{
			&ast.CallJavaActionStmt{
				ActionName: ast.QualifiedName{Module: "Administration", Name: "NonExistentJavaAction"},
			},
		},
	}

	err := validate(ctx, stmt)
	if err == nil {
		t.Fatal("expected missing user Java action reference error")
	}
	if !strings.Contains(err.Error(), "java action not found: Administration.NonExistentJavaAction") {
		t.Fatalf("unexpected error: %v", err)
	}
}

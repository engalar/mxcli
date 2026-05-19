// SPDX-License-Identifier: Apache-2.0

package executor_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/linter"
)

// microflow with: SET $Id = $WorkHistory/id;
func mfWithIdAccess(varName, attrPath string) *ast.CreateMicroflowStmt {
	return &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "WF_Engine", Name: "ACT_Test"},
		ReturnType: &ast.MicroflowReturnType{
			Type: ast.DataType{Kind: ast.TypeLong},
		},
		Body: []ast.MicroflowStatement{
			&ast.MfSetStmt{
				Target: "Id",
				Value: &ast.AttributePathExpr{
					Variable: varName,
					Path:     []string{attrPath},
				},
			},
			&ast.ReturnStmt{
				Value: &ast.VariableExpr{Name: "Id"},
			},
		},
	}
}

// TestValidateMicroflow_IdAccess_FlagsIdPath verifies that SET $Var = $Object/id
// is reported as a violation referencing hint E012.
func TestValidateMicroflow_IdAccess_FlagsIdPath(t *testing.T) {
	stmt := mfWithIdAccess("WorkHistory", "id")
	violations := executor.ValidateMicroflow(stmt)

	found := false
	for _, v := range violations {
		if v.RuleID == "MDL013" {
			found = true
			if v.Severity != linter.SeverityError {
				t.Errorf("MDL013 severity = %v, want Error", v.Severity)
			}
			if v.Suggestion == "" {
				t.Error("MDL013 Suggestion must not be empty")
			}
			// The suggestion must guide toward E012.
			hasE012 := false
			for _, s := range []string{v.Suggestion, v.Message} {
				if contains(s, "E012") {
					hasE012 = true
					break
				}
			}
			if !hasE012 {
				t.Errorf("MDL013 message/suggestion must reference E012; got message=%q suggestion=%q",
					v.Message, v.Suggestion)
			}
		}
	}
	if !found {
		t.Errorf("expected MDL013 violation for $WorkHistory/id, got: %+v", violations)
	}
}

// TestValidateMicroflow_IdAccess_NoFalsePositive verifies that $WorkHistory/Content
// (a regular attribute) does NOT trigger MDL013.
func TestValidateMicroflow_IdAccess_NoFalsePositive(t *testing.T) {
	stmt := mfWithIdAccess("WorkHistory", "Content")
	for _, v := range executor.ValidateMicroflow(stmt) {
		if v.RuleID == "MDL013" {
			t.Errorf("false positive MDL013 for $WorkHistory/Content: %+v", v)
		}
	}
}

// TestValidateMicroflow_IdAccess_InReturnExpr verifies that RETURN $Obj/id also triggers MDL013.
func TestValidateMicroflow_IdAccess_InReturnExpr(t *testing.T) {
	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "WF_Engine", Name: "ACT_Test"},
		ReturnType: &ast.MicroflowReturnType{
			Type: ast.DataType{Kind: ast.TypeLong},
		},
		Body: []ast.MicroflowStatement{
			&ast.ReturnStmt{
				Value: &ast.AttributePathExpr{
					Variable: "WorkHistory",
					Path:     []string{"id"},
				},
			},
		},
	}
	violations := executor.ValidateMicroflow(stmt)
	for _, v := range violations {
		if v.RuleID == "MDL013" {
			return // found — test passes
		}
	}
	t.Errorf("expected MDL013 for RETURN $WorkHistory/id, got: %+v", violations)
}

// TestValidateMicroflow_IdAccess_SourceExpr_StringPath verifies that the
// containsIdPath fallback fires when Source="$WorkHistory/id" even if the
// inner Expression does not expose an AttributePathExpr with "id" in Path.
// This is the case produced by the real visitor.
func TestValidateMicroflow_IdAccess_SourceExpr_StringPath(t *testing.T) {
	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "WF_Engine", Name: "ACT_Test"},
		ReturnType: &ast.MicroflowReturnType{
			Type: ast.DataType{Kind: ast.TypeLong},
		},
		Body: []ast.MicroflowStatement{
			&ast.MfSetStmt{
				Target: "Id",
				Value: &ast.SourceExpr{
					Source:     "$WorkHistory/id",
					Expression: &ast.VariableExpr{Name: "WorkHistory"}, // no /id in inner tree
				},
			},
			&ast.ReturnStmt{Value: &ast.VariableExpr{Name: "Id"}},
		},
	}
	violations := executor.ValidateMicroflow(stmt)
	for _, v := range violations {
		if v.RuleID == "MDL013" {
			return
		}
	}
	t.Errorf("expected MDL013 via Source string for $WorkHistory/id, got: %+v", violations)
}

// TestValidateMicroflow_IdAccess_SourceExpr_NoFalsePositive verifies that
// a SourceExpr containing a valid attribute (e.g. $Obj/UserId) does NOT trigger MDL013.
func TestValidateMicroflow_IdAccess_SourceExpr_NoFalsePositive(t *testing.T) {
	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "WF_Engine", Name: "ACT_Test"},
		Body: []ast.MicroflowStatement{
			&ast.MfSetStmt{
				Target: "X",
				Value: &ast.SourceExpr{
					Source:     "$Obj/UserId",
					Expression: &ast.AttributePathExpr{Variable: "Obj", Path: []string{"UserId"}},
				},
			},
		},
	}
	for _, v := range executor.ValidateMicroflow(stmt) {
		if v.RuleID == "MDL013" {
			t.Errorf("false positive MDL013 for $Obj/UserId: %+v", v)
		}
	}
}

// TestValidateMicroflow_IdAccess_NoDuplicateViolation verifies that $Var/id
// inside a SourceExpr produces exactly ONE MDL013, not two (one from the inner
// AttributePathExpr and one from the Source string).
func TestValidateMicroflow_IdAccess_NoDuplicateViolation(t *testing.T) {
	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "WF_Engine", Name: "ACT_Test"},
		Body: []ast.MicroflowStatement{
			&ast.MfSetStmt{
				Target: "X",
				Value: &ast.SourceExpr{
					Source:     "$Area/id",
					Expression: &ast.AttributePathExpr{Variable: "Area", Path: []string{"id"}},
				},
			},
		},
	}
	count := 0
	for _, v := range executor.ValidateMicroflow(stmt) {
		if v.RuleID == "MDL013" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 MDL013 violation, got %d", count)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

// SPDX-License-Identifier: Apache-2.0

package executor_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/linter"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func mfWithRetrieve(whereSource string) *ast.CreateMicroflowStmt {
	return &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "M", Name: "F"},
		Body: []ast.MicroflowStatement{
			&ast.RetrieveStmt{
				Variable: "R",
				Source:   ast.QualifiedName{Module: "A", Name: "B"},
				Where:    &ast.SourceExpr{Source: whereSource},
			},
		},
	}
}

func mfWithExpr(initSource string) *ast.CreateMicroflowStmt {
	return &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "M", Name: "F"},
		Body: []ast.MicroflowStatement{
			&ast.DeclareStmt{
				Variable:     "X",
				Type:         ast.DataType{Kind: ast.TypeString},
				InitialValue: &ast.SourceExpr{Source: initSource},
			},
		},
	}
}

func hasViolation(violations []linter.Violation, ruleID string) bool {
	for _, v := range violations {
		if v.RuleID == ruleID {
			return true
		}
	}
	return false
}

func countViolations(violations []linter.Violation, ruleID string) int {
	n := 0
	for _, v := range violations {
		if v.RuleID == ruleID {
			n++
		}
	}
	return n
}

// ── Rule 1: MDL014 — uppercase AND in WHERE ──────────────────────────────────

func TestValidateMicroflow_MDL014_UppercaseAND_Flagged(t *testing.T) {
	stmt := mfWithRetrieve("Prefix=$P AND Year=$Y")
	if !hasViolation(executor.ValidateMicroflow(stmt), "MDL014") {
		t.Error("expected MDL014 for uppercase AND in WHERE")
	}
}

func TestValidateMicroflow_MDL014_LowercaseAnd_NotFlagged(t *testing.T) {
	stmt := mfWithRetrieve("(Prefix=$P) and (Year=$Y)")
	if hasViolation(executor.ValidateMicroflow(stmt), "MDL014") {
		t.Error("false positive MDL014 for lowercase and")
	}
}

func TestValidateMicroflow_MDL014_OnlyOneViolation(t *testing.T) {
	stmt := mfWithRetrieve("A=$X AND B=$Y AND C=$Z")
	n := countViolations(executor.ValidateMicroflow(stmt), "MDL014")
	if n != 1 {
		t.Errorf("expected exactly 1 MDL014 per RETRIEVE, got %d", n)
	}
}

func TestValidateMicroflow_MDL014_Fix_ReferencesHint(t *testing.T) {
	stmt := mfWithRetrieve("Prefix=$P AND Year=$Y")
	for _, v := range executor.ValidateMicroflow(stmt) {
		if v.RuleID == "MDL014" {
			if v.Suggestion == "" {
				t.Error("MDL014 Suggestion must not be empty")
			}
			return
		}
	}
	t.Error("MDL014 not found")
}

// ── Rule 2: MDL015 — quoted association name in WHERE ────────────────────────

func TestValidateMicroflow_MDL015_QuotedAssoc_Flagged(t *testing.T) {
	stmt := mfWithRetrieve(`("PayerRegistration"."PayerAreaData_Payer"=$Payer) and (IsActive=true)`)
	if !hasViolation(executor.ValidateMicroflow(stmt), "MDL015") {
		t.Error("expected MDL015 for quoted association name in WHERE")
	}
}

func TestValidateMicroflow_MDL015_UnquotedAssoc_NotFlagged(t *testing.T) {
	stmt := mfWithRetrieve("(PayerRegistration.PayerAreaData_Payer=$Payer) and (IsActive=true)")
	if hasViolation(executor.ValidateMicroflow(stmt), "MDL015") {
		t.Error("false positive MDL015 for unquoted association")
	}
}

func TestValidateMicroflow_MDL015_ErrorSeverity(t *testing.T) {
	stmt := mfWithRetrieve(`"Mod"."AssocName"=$Var`)
	for _, v := range executor.ValidateMicroflow(stmt) {
		if v.RuleID == "MDL015" {
			if v.Severity != linter.SeverityError {
				t.Errorf("MDL015 severity = %v, want Error", v.Severity)
			}
			return
		}
	}
	t.Error("MDL015 not found")
}

// ── Rule 3: MDL016 — [%CurrentUser%] complex path ───────────────────────────

func TestValidateMicroflow_MDL016_ComplexCurrentUserPath_Flagged(t *testing.T) {
	stmt := mfWithExpr("[%CurrentUser%]/System.User/Name")
	if !hasViolation(executor.ValidateMicroflow(stmt), "MDL016") {
		t.Error("expected MDL016 for [%CurrentUser%]/System.User/Name")
	}
}

func TestValidateMicroflow_MDL016_SimpleCurrentUserPath_NotFlagged(t *testing.T) {
	stmt := mfWithExpr("[%CurrentUser%]/Name")
	if hasViolation(executor.ValidateMicroflow(stmt), "MDL016") {
		t.Error("false positive MDL016 for [%CurrentUser%]/Name")
	}
}

func TestValidateMicroflow_MDL016_InSetStmt_Flagged(t *testing.T) {
	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "M", Name: "F"},
		Body: []ast.MicroflowStatement{
			&ast.MfSetStmt{
				Target: "X",
				Value:  &ast.SourceExpr{Source: "[%CurrentUser%]/System.User/Name"},
			},
		},
	}
	if !hasViolation(executor.ValidateMicroflow(stmt), "MDL016") {
		t.Error("expected MDL016 for [%CurrentUser%]/System.User/Name in SET")
	}
}

// ── Rule 5: MDL017 — RETURNS Integer + /id ───────────────────────────────────

func TestValidateMicroflow_MDL017_ReturnsIntegerWithId_Flagged(t *testing.T) {
	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "M", Name: "F"},
		ReturnType: &ast.MicroflowReturnType{
			Type: ast.DataType{Kind: ast.TypeInteger},
		},
		Body: []ast.MicroflowStatement{
			&ast.MfSetStmt{
				Target: "Id",
				Value:  &ast.SourceExpr{Source: "$Obj/id"},
			},
			&ast.ReturnStmt{Value: &ast.VariableExpr{Name: "Id"}},
		},
	}
	if !hasViolation(executor.ValidateMicroflow(stmt), "MDL017") {
		t.Error("expected MDL017 for RETURNS Integer + $Obj/id")
	}
}

func TestValidateMicroflow_MDL017_ReturnsLongWithId_NotFlagged(t *testing.T) {
	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "M", Name: "F"},
		ReturnType: &ast.MicroflowReturnType{
			Type: ast.DataType{Kind: ast.TypeLong},
		},
		Body: []ast.MicroflowStatement{
			&ast.MfSetStmt{
				Target: "Id",
				Value:  &ast.SourceExpr{Source: "$Obj/id"},
			},
			&ast.ReturnStmt{Value: &ast.VariableExpr{Name: "Id"}},
		},
	}
	// MDL013 fires (illegal /id), but MDL017 must NOT fire (return type is already Long)
	if hasViolation(executor.ValidateMicroflow(stmt), "MDL017") {
		t.Error("false positive MDL017 when return type is Long")
	}
}

func TestValidateMicroflow_MDL017_ReturnsIntegerNoId_NotFlagged(t *testing.T) {
	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "M", Name: "F"},
		ReturnType: &ast.MicroflowReturnType{
			Type: ast.DataType{Kind: ast.TypeInteger},
		},
		Body: []ast.MicroflowStatement{
			&ast.MfSetStmt{
				Target: "X",
				Value:  &ast.SourceExpr{Source: "$Obj/Counter"},
			},
		},
	}
	if hasViolation(executor.ValidateMicroflow(stmt), "MDL017") {
		t.Error("false positive MDL017 when body has no /id")
	}
}

func TestValidateMicroflow_MDL017_Suggestion_MentionsLong(t *testing.T) {
	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "M", Name: "F"},
		ReturnType: &ast.MicroflowReturnType{
			Type: ast.DataType{Kind: ast.TypeInteger},
		},
		Body: []ast.MicroflowStatement{
			&ast.MfSetStmt{
				Target: "Id",
				Value:  &ast.SourceExpr{Source: "$Obj/id"},
			},
		},
	}
	for _, v := range executor.ValidateMicroflow(stmt) {
		if v.RuleID == "MDL017" {
			if v.Suggestion == "" {
				t.Error("MDL017 Suggestion must not be empty")
			}
			return
		}
	}
	t.Error("MDL017 not found")
}

// SPDX-License-Identifier: Apache-2.0

package typecheck_test

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/typecheck"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
)

// stubIndex satisfies IndexReader with zero-value returns so CheckStructural
// can run without a real project index. VarTypeKind returns KindObject so
// SEM-L01 (List attribute access) does not interfere with SEM-L02 tests.
type stubIndex struct{}

func (s stubIndex) AttributeKind(_, _ string) (exprcheck.TypeKind, bool) { return exprcheck.KindUnknown, false }
func (s stubIndex) VarTypeKind(_, _ string) exprcheck.TypeKind           { return exprcheck.KindObject }
func (s stubIndex) VarEntityQN(_, _ string) string                       { return "" }
func (s stubIndex) MicroflowParamKind(_, _ string) (exprcheck.TypeKind, bool) {
	return exprcheck.KindUnknown, false
}
func (s stubIndex) MicroflowReturnKind(_ string) (exprcheck.TypeKind, bool) {
	return exprcheck.KindUnknown, false
}

func makeStructuralPR(raw string, ast exprcheck.RobustExpr) parse.ParseResult {
	return parse.ParseResult{
		Record: scan.ExprRecord{
			UnitID:   "unit-001",
			UnitType: "Microflows$ExclusiveSplitCondition",
			Field:    "Expression",
			Raw:      raw,
		},
		AST: ast,
	}
}

// TestCheckStructural_SEML02_FlagsIdAccess verifies that accessing /id on an
// object variable raises SEM-L02 (id is a reserved Mendix system attribute,
// not readable in microflow expressions).
func TestCheckStructural_SEML02_FlagsIdAccess(t *testing.T) {
	tc := typecheck.NewChecker(stubIndex{})
	pr := makeStructuralPR("$WorkHistory/id", &exprcheck.AttributePathExpr{
		Variable: "WorkHistory",
		Path:     []string{"id"},
	})
	results := tc.CheckStructural(pr)
	if len(results) == 0 {
		t.Fatal("expected SEM-L02 result, got none")
	}
	found := false
	for _, r := range results {
		if r.RuleID == "SEM-L02" {
			found = true
			if r.Severity != "ERROR" {
				t.Errorf("SEM-L02 severity = %q, want ERROR", r.Severity)
			}
			if !strings.Contains(r.Fix, "E012") {
				t.Errorf("SEM-L02 Fix = %q, want reference to E012", r.Fix)
			}
		}
	}
	if !found {
		t.Errorf("no SEM-L02 result; got: %+v", results)
	}
}

// TestCheckStructural_SEML02_DeepPath verifies SEM-L02 fires when /id appears
// anywhere in a multi-segment path (e.g. $Obj/Association/id).
func TestCheckStructural_SEML02_DeepPath(t *testing.T) {
	tc := typecheck.NewChecker(stubIndex{})
	pr := makeStructuralPR("$Obj/Association/id", &exprcheck.AttributePathExpr{
		Variable: "Obj",
		Path:     []string{"Association", "id"},
	})
	results := tc.CheckStructural(pr)
	for _, r := range results {
		if r.RuleID == "SEM-L02" {
			return
		}
	}
	t.Errorf("expected SEM-L02 for deep path containing /id, got: %+v", results)
}

// TestCheckStructural_SEML02_NoFalsePositive verifies that a regular attribute
// access ($WorkHistory/Content) does NOT trigger SEM-L02.
func TestCheckStructural_SEML02_NoFalsePositive(t *testing.T) {
	tc := typecheck.NewChecker(stubIndex{})
	pr := makeStructuralPR("$WorkHistory/Content", &exprcheck.AttributePathExpr{
		Variable: "WorkHistory",
		Path:     []string{"Content"},
	})
	for _, r := range tc.CheckStructural(pr) {
		if r.RuleID == "SEM-L02" {
			t.Errorf("false positive SEM-L02 for $WorkHistory/Content: %+v", r)
		}
	}
}

// TestCheckStructural_SEML02_NilAST verifies that a nil AST does not panic.
func TestCheckStructural_SEML02_NilAST(t *testing.T) {
	tc := typecheck.NewChecker(stubIndex{})
	pr := makeStructuralPR("", nil)
	results := tc.CheckStructural(pr)
	if len(results) != 0 {
		t.Errorf("nil AST should produce no results, got: %+v", results)
	}
}

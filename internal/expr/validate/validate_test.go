// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/validate"
	"github.com/stretchr/testify/assert"
)

func makeRec(raw, unitType, slot string) scan.ExprRecord {
	return scan.ExprRecord{Raw: raw, UnitType: unitType, Category: "microflow"}
}

func TestValidate_CleanExpression_NoIssues(t *testing.T) {
	r := makeRec("trim($X) = ''", "Microflows$ExpressionSplitCondition", "IfStmt.Condition")
	pr := parse.ParseExpression(r)
	issues := validate.ValidateSyntax(pr)
	assert.Empty(t, issues, "clean expression must have no issues")
}

func TestValidateSYN02_URL(t *testing.T) {
	r := makeRec("https://www.mendix.com/", "Forms$StaticOrDynamicString", "")
	pr := parse.ParseExpression(r)
	issues := validate.ValidateSyntax(pr)
	found := false
	for _, i := range issues {
		if i.RuleID == "SYN-02" {
			assert.Equal(t, "INFO", i.Severity)
			found = true
		}
	}
	assert.True(t, found, "should flag URL as SYN-02")
}

func TestValidateSYN03_MissingElse(t *testing.T) {
	r := makeRec("if $X then $Y", "Microflows$ExpressionSplitCondition", "IfStmt.Condition")
	pr := parse.ParseExpression(r)
	issues := validate.ValidateSyntax(pr)
	found := false
	for _, i := range issues {
		if i.RuleID == "SYN-03" {
			assert.Equal(t, "WARNING", i.Severity)
			assert.NotEmpty(t, i.Fix)
			found = true
		}
	}
	assert.True(t, found, "should flag missing else as SYN-03")
}

func TestValidate_ExprcheckHints_Surfaced(t *testing.T) {
	// "not x" without parens triggers E011 from exprcheck
	r := makeRec("not $IsValid", "Microflows$ExpressionSplitCondition", "IfStmt.Condition")
	pr := parse.ParseExpression(r)
	issues := validate.ValidateSyntax(pr)
	found := false
	for _, i := range issues {
		if i.RuleID == "E011" {
			assert.NotEmpty(t, i.Fix)
			found = true
		}
	}
	assert.True(t, found, "E011 hint should be surfaced for 'not x' without parens")
}

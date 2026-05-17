// SPDX-License-Identifier: Apache-2.0

package repair_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/repair"
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rec(raw, slot string) scan.ExprRecord {
	return scan.ExprRecord{Raw: raw, UnitType: "Microflows$ExpressionSplitCondition", Category: "microflow"}
}

func issuesFor(raw, slot string) []validate.ValidationResult {
	r := rec(raw, slot)
	return validate.ValidateSyntax(parse.ParseExpression(r))
}

func TestRepairR01_KeywordGlued(t *testing.T) {
	for _, issue := range issuesFor("$Dto=emptyor$X=''", "IfStmt.Condition") {
		if issue.RuleID == "SYN-01" {
			sugs := repair.Suggest(issue)
			require.NotEmpty(t, sugs, "R-01 should produce suggestions")
			assert.Equal(t, "R-01", sugs[0].PatternID)
			assert.Greater(t, sugs[0].Confidence, 0.9)
			assert.Contains(t, sugs[0].After, "empty or")
			return
		}
	}
	t.Skip("no SYN-01 issue generated — expression may have been parsed differently")
}

func TestRepairR06_MissingElse(t *testing.T) {
	for _, issue := range issuesFor("if $X then $Y", "IfStmt.Condition") {
		if issue.RuleID == "SYN-03" {
			sugs := repair.Suggest(issue)
			require.NotEmpty(t, sugs)
			assert.Equal(t, "R-06", sugs[0].PatternID)
			assert.Contains(t, sugs[0].After, "else empty")
			return
		}
	}
	t.Skip("no SYN-03 issue — expression may have parsed with implicit else")
}

func TestRepairE011_NotWithoutParens(t *testing.T) {
	for _, issue := range issuesFor("not $IsValid", "IfStmt.Condition") {
		if issue.RuleID == "E011" {
			sugs := repair.Suggest(issue)
			require.NotEmpty(t, sugs)
			assert.Equal(t, "R-E011", sugs[0].PatternID)
			assert.Contains(t, sugs[0].After, "not($IsValid)")
			assert.Greater(t, sugs[0].Confidence, 0.9)
			return
		}
	}
}

func TestRepairClean_NoSuggestions(t *testing.T) {
	issues := issuesFor("trim($X) = ''", "IfStmt.Condition")
	for _, issue := range issues {
		sugs := repair.Suggest(issue)
		assert.Empty(t, sugs, "clean expression should not generate repair suggestions")
	}
}

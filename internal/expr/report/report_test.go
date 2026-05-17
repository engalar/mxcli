// SPDX-License-Identifier: Apache-2.0

package report_test

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/report"
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleIssues() []validate.ValidationResult {
	r := scan.ExprRecord{
		Raw: "not $IsValid", UnitType: "Microflows$ExpressionSplitCondition",
		SlotPath: "IfStmt.Condition", Project: "test", Category: "microflow",
	}
	return validate.ValidateSyntax(parse.ParseExpression(r))
}

func TestRenderJSON(t *testing.T) {
	issues := sampleIssues()
	out, err := report.Render(issues, report.Options{Format: "json"})
	require.NoError(t, err)
	assert.Contains(t, string(out), "E011")
}

func TestRenderHTML(t *testing.T) {
	issues := sampleIssues()
	out, err := report.Render(issues, report.Options{Format: "html"})
	require.NoError(t, err)
	body := string(out)
	assert.True(t, strings.HasPrefix(body, "<!DOCTYPE html>"))
	assert.Contains(t, body, "MEMV Validation Report")
	assert.Contains(t, body, "E011")
}

func TestRenderText(t *testing.T) {
	issues := sampleIssues()
	out, err := report.Render(issues, report.Options{Format: "text"})
	require.NoError(t, err)
	assert.Contains(t, string(out), "Total:")
}

func TestRenderFilter_ErrorOnly(t *testing.T) {
	r := scan.ExprRecord{
		Raw: "if $X then $Y", UnitType: "Microflows$ExpressionSplitCondition",
		SlotPath: "IfStmt.Condition", Project: "test",
	}
	issues := validate.ValidateSyntax(parse.ParseExpression(r))
	out, err := report.Render(issues, report.Options{Format: "json", Severity: "ERROR"})
	require.NoError(t, err)
	assert.NotContains(t, string(out), "WARNING")
}

func TestRenderEmpty(t *testing.T) {
	out, err := report.Render(nil, report.Options{Format: "html"})
	require.NoError(t, err)
	assert.Contains(t, string(out), "0 issues")
}

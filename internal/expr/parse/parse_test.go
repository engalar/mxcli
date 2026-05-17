// SPDX-License-Identifier: Apache-2.0

package parse_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rec(raw, unitType, slotPath string) scan.ExprRecord {
	return scan.ExprRecord{
		Raw:      raw,
		UnitType: unitType,
		SlotPath: slotPath,
		Category: "microflow",
		Project:  "test",
	}
}

// Canonical cases drawn directly from real corpus samples confirmed to parse cleanly.
var canonicalCases = []scan.ExprRecord{
	rec("$WFInstance = empty", "Microflows$ExpressionSplitCondition", "IfStmt.Condition"),
	rec("trim($Workbook) = '' or trim($Sheet) = ''", "Microflows$ExpressionSplitCondition", "IfStmt.Condition"),
	rec("(if trim($InvalidFields) = '' then '' else $InvalidFields + ', ')", "Microflows$ChangeVariableAction", "MfSetStmt.Value"),
	rec("if $Feedback/_showEmail then true else false", "Microflows$ExpressionSplitCondition", "IfStmt.Condition"),
	rec("addDays([%CurrentDateTime%], 5)", "Workflows$SingleUserTaskActivity", "RetrieveStmt.LimitExpr"),
	rec("@FeedbackModule.LocalStorageKey", "Microflows$BasicCodeActionParameterValue", "CallArgument.Value"),
	rec("Common_Utils.BatchExecStatus.Completed", "Microflows$ExpressionSplitCondition", "IfStmt.Condition"),
	rec("not(isMatch($Cleaned, '^[0-9]+$'))", "Microflows$ExpressionSplitCondition", "IfStmt.Condition"),
	rec("$currentObject/ImageB64 != empty", "Forms$ConditionalVisibilitySettings", "IfStmt.Condition"),
	rec("length(toString($value)) > 0", "Forms$WidgetValidation", "IfStmt.Condition"),
	rec("false", "Microflows$CreateVariableAction", "DeclareStmt.InitialValue"),
	rec("$Dto", "Microflows$MicroflowCallParameterMapping", "CallArgument.Value"),
	rec("$Total : $Count", "Microflows$ChangeVariableAction", "MfSetStmt.Value"),
}

func TestParseExpression_CanonicalCases(t *testing.T) {
	for _, tc := range canonicalCases {
		t.Run(tc.Raw[:min(len(tc.Raw), 50)], func(t *testing.T) {
			result := parse.ParseExpression(tc)
			assert.True(t, result.OK,
				"expected OK for %q, hints: %+v", tc.Raw, result.Hints)
		})
	}
}

func TestParseExpression_SetsRecord(t *testing.T) {
	r := rec("$X = empty", "Microflows$ExpressionSplitCondition", "IfStmt.Condition")
	result := parse.ParseExpression(r)
	assert.Equal(t, r.Raw, result.Record.Raw)
	assert.Equal(t, r.SlotPath, result.Record.SlotPath)
}

func TestBatchParse_CorpusCoverage(t *testing.T) {
	records, err := scan.ScanMprcontents(
		"/mnt/data_sdd/macnica/mendix-app/mprcontents", scan.Options{})
	require.NoError(t, err)

	results := parse.BatchParse(records)
	pass := 0
	for _, r := range results {
		if r.OK {
			pass++
		}
	}
	coverage := float64(pass) / float64(len(results)) * 100
	t.Logf("macnica coverage: %.1f%% (%d/%d)", coverage, pass, len(results))
	assert.Greater(t, coverage, 85.0, "coverage must be >85%% (hand-written parser may differ from ANTLR4)")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

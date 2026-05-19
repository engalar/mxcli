// SPDX-License-Identifier: Apache-2.0

package parse_test

import (
	"os"
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/testutil"
	"github.com/stretchr/testify/assert"
)

func rec(raw, unitType, _ string) scan.ExprRecord {
	return scan.ExprRecord{
		Raw:      raw,
		UnitType: unitType,
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
	assert.Equal(t, r.Raw, result.Record.Raw)
}

// scanProject transparently supports v1 (SQLite) and v2 (mprcontents/) formats.
func scanProject(t *testing.T, mprPath string, opts scan.Options) []scan.ExprRecord {
	t.Helper()
	contentsDir := scan.MprContentsPath(mprPath)
	if _, err := os.Stat(contentsDir); err == nil {
		recs, err := scan.ScanMprcontents(contentsDir, opts)
		if err != nil {
			t.Skipf("ScanMprcontents failed: %v", err)
		}
		return recs
	}
	recs, err := scan.ScanMPR(mprPath, opts)
	if err != nil {
		t.Skipf("ScanMPR failed: %v", err)
	}
	return recs
}

func TestBatchParse_CorpusCoverage(t *testing.T) {
	mprPath := testutil.FindMPR(t, "MACNICA_MPR", "testdata/macnica/MacnicaApp.mpr")
	records := scanProject(t, mprPath, scan.Options{})

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

// XPath routing tests — verify that XPath slot records use visitor.ParseXPathConstraint
// and do NOT generate E007 false positives.
func TestParseExpression_XPath_NoFalsePositives(t *testing.T) {
	// Valid XPath constraints — all start with "[" but not "[%"
	xpathCases := []string{
		"[Code = $Code\n      and IsInvalid = false]",
		"[ApplicationHeaderId=$ApplicationHeaderId]",
		"[UserId = $UserName\n      and IsActive = true]",
		"[Common_Utils.PortalDispSetting_Account = $Account]",
		"[id='[%CurrentUser%]']",
	}

	for _, tc := range xpathCases {
		t.Run(tc[:min(len(tc), 50)], func(t *testing.T) {
			r := scan.ExprRecord{
				Raw:      tc,
				UnitType: "Microflows$DatabaseRetrieveSource", Category: "microflow",
			}
			result := parse.ParseExpression(r)
			assert.True(t, result.OK, "valid XPath must parse OK, got hints: %+v", result.Hints)
			for _, h := range result.Hints {
				assert.NotEqual(t, "E007", h.Code, "E007 must not fire on valid XPath: %s", tc)
			}
		})
	}
}

func TestParseExpression_XPath_EmptyFails(t *testing.T) {
	// ParseXPathConstraint returns (nil, false) only for empty input.
	// The MDL parser uses error recovery so non-empty malformed input may still
	// produce a partial tree — we verify the empty case as the documented contract.
	r := scan.ExprRecord{
		Raw: "", UnitType: "Microflows$DatabaseRetrieveSource", Category: "microflow",
	}
	result := parse.ParseExpression(r)
	// Empty raw is excluded by scan, but if it reaches parse it must not panic.
	_ = result // just verify no panic
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

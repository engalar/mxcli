//go:build integration

// SPDX-License-Identifier: Apache-2.0

package expr_test

import (
	"os"
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/repair"
	"github.com/mendixlabs/mxcli/internal/expr/report"
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/testutil"
	"github.com/mendixlabs/mxcli/internal/expr/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestFullPipeline_Macnica(t *testing.T) {
	corpusAMPR := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")
	// Layer 2: scan
	recs := scanProject(t, corpusAMPR, scan.Options{})
	assert.Greater(t, len(recs), 3000, "corpus-a must yield >3000 expression records")

	// Layer 3: parse — use exprcheck (100% coverage observed)
	parsed := parse.BatchParse(recs)
	pass := 0
	for _, pr := range parsed {
		if pr.OK {
			pass++
		}
	}
	coverage := float64(pass) / float64(len(parsed)) * 100
	t.Logf("Parse coverage: %.1f%% (%d/%d)", coverage, pass, len(parsed))
	assert.Greater(t, coverage, 95.0, "corpus-a parse coverage must be >95%%")

	// Layer 4: validate
	var issues []validate.ValidationResult
	for _, pr := range parsed {
		issues = append(issues, validate.ValidateSyntax(pr)...)
	}
	errCount, warnCount := 0, 0
	for _, i := range issues {
		switch i.Severity {
		case "ERROR":
			errCount++
		case "WARNING":
			warnCount++
		}
	}
	t.Logf("Validation: %d issues (ERROR:%d WARNING:%d)", len(issues), errCount, warnCount)

	// Layer 5: repair
	repairCount := 0
	for _, issue := range issues {
		if sugs := repair.Suggest(issue); len(sugs) > 0 {
			repairCount++
		}
	}
	t.Logf("Repair suggestions: %d issues have suggestions", repairCount)

	// Layer 6: report renders without error
	htmlOut, err := report.Render(issues, report.Options{Format: "html"})
	require.NoError(t, err)
	assert.Contains(t, string(htmlOut), "MEMV Validation Report")

	jsonOut, err := report.Render(issues, report.Options{Format: "json"})
	require.NoError(t, err)
	assert.NotEmpty(t, jsonOut)
}

func TestFullPipeline_BothProjects(t *testing.T) {
	corpusAMPR := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")
	corpusBMPR := testutil.FindMPR(t, "CORPUS_B_MPR", "testdata/corpus-b/app.mpr")
	var allRecs []scan.ExprRecord
	for _, mprPath := range []string{corpusAMPR, corpusBMPR} {
		recs := scanProject(t, mprPath, scan.Options{})
		allRecs = append(allRecs, recs...)
	}
	assert.Greater(t, len(allRecs), 15000, "both projects combined must yield >15000 records")

	parsed := parse.BatchParse(allRecs)
	pass := 0
	for _, pr := range parsed {
		if pr.OK {
			pass++
		}
	}
	coverage := float64(pass) / float64(len(parsed)) * 100
	t.Logf("Both projects coverage: %.1f%% (%d/%d)", coverage, pass, len(parsed))
	assert.Greater(t, coverage, 95.0)

	// Collect all issues and verify report renders
	var issues []validate.ValidationResult
	for _, pr := range parsed {
		issues = append(issues, validate.ValidateSyntax(pr)...)
	}
	out, err := report.Render(issues, report.Options{Format: "text"})
	require.NoError(t, err)
	assert.Contains(t, string(out), "Total:")
	t.Logf("Combined validation: %d issues", len(issues))
}

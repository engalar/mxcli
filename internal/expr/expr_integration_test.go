//go:build integration

// SPDX-License-Identifier: Apache-2.0

package expr_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/repair"
	"github.com/mendixlabs/mxcli/internal/expr/report"
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	macnicaMprcontents = "/mnt/data_sdd/macnica/mendix-app/mprcontents"
	mx2026Mprcontents  = "/mnt/data_sdd/gh/Mx2026AIDay/mprcontents"
)

func TestFullPipeline_Macnica(t *testing.T) {
	// Layer 2: scan
	recs, err := scan.ScanMprcontents(macnicaMprcontents, scan.Options{})
	require.NoError(t, err)
	assert.Greater(t, len(recs), 3000, "macnica must yield >3000 expression records")

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
	assert.Greater(t, coverage, 95.0, "macnica parse coverage must be >95%%")

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
	var allRecs []scan.ExprRecord
	for _, path := range []string{macnicaMprcontents, mx2026Mprcontents} {
		recs, err := scan.ScanMprcontents(path, scan.Options{})
		require.NoError(t, err)
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

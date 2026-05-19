// SPDX-License-Identifier: Apache-2.0

//go:build integration

package typecheck_test

import (
	"os"
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/meta"
	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/testutil"
	"github.com/mendixlabs/mxcli/internal/expr/typecheck"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
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

func runSEM03(t *testing.T, mprPath string) []typecheck.Result {
	t.Helper()
	b, err := mprbackend.NewFromPath(mprPath)
	if err != nil {
		t.Skipf("MPR not accessible: %v", err)
	}
	idx, err := meta.BuildFromBackend(b)
	if err != nil {
		t.Fatalf("BuildFromBackend: %v", err)
	}

	records := scanProject(t, mprPath, scan.Options{})

	parseResults := parse.BatchParseWithCatalog(records, idx)
	checker := typecheck.NewChecker(idx)

	var results []typecheck.Result
	for _, pr := range parseResults {
		results = append(results, checker.Check(pr)...)
	}
	return results
}

func TestSEM03_Macnica_HasDetections(t *testing.T) {
	mprPath := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")
	results := runSEM03(t, mprPath)
	sem03 := 0
	for _, r := range results {
		if r.RuleID == "SEM-03" {
			sem03++
		}
	}
	t.Logf("SEM-03 detections: %d", sem03)
	if sem03 == 0 {
		t.Error("expected at least 1 SEM-03 detection in corpus-a")
	}
}

func TestSEM03_CorpusB_NoFalsePositives(t *testing.T) {
	mprPath := testutil.FindMPR(t, "CORPUS_B_MPR", "testdata/corpus-b/app.mpr")
	results := runSEM03(t, mprPath)
	for _, r := range results {
		if r.RuleID == "SEM-03" {
			t.Errorf("false positive SEM-03 in corpus-b: %s on %s/%s [%s]",
				r.Message, r.UnitType, r.Field, r.Raw)
		}
	}
}

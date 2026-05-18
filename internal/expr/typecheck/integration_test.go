// SPDX-License-Identifier: Apache-2.0

//go:build integration

package typecheck_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/meta"
	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/typecheck"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
)

const macnicaMPR = "/mnt/data_sdd/macnica/mendix-app/MacnicaApp.mpr"
const mx2026MPR = "/mnt/data_sdd/gh/Mx2026AIDay/Factory Management.mpr"

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

	mprcontentsPath := scan.MprContentsPath(mprPath)
	records, err := scan.ScanMprcontents(mprcontentsPath, scan.Options{})
	if err != nil {
		t.Fatalf("ScanMprcontents: %v", err)
	}

	parseResults := parse.BatchParseWithCatalog(records, idx)
	checker := typecheck.NewChecker(idx)

	var results []typecheck.Result
	for _, pr := range parseResults {
		results = append(results, checker.Check(pr)...)
	}
	return results
}

func TestSEM03_Macnica_HasDetections(t *testing.T) {
	results := runSEM03(t, macnicaMPR)
	sem03 := 0
	for _, r := range results {
		if r.RuleID == "SEM-03" {
			sem03++
		}
	}
	t.Logf("SEM-03 detections: %d", sem03)
	if sem03 == 0 {
		t.Error("expected at least 1 SEM-03 detection in macnica")
	}
}

func TestSEM03_Mx2026AIDay_NoFalsePositives(t *testing.T) {
	results := runSEM03(t, mx2026MPR)
	for _, r := range results {
		if r.RuleID == "SEM-03" {
			t.Errorf("false positive SEM-03 in Mx2026AIDay: %s on %s/%s [%s]",
				r.Message, r.UnitType, r.Field, r.Raw)
		}
	}
}

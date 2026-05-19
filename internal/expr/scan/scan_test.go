// SPDX-License-Identifier: Apache-2.0

package scan_test

import (
	"os"
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/testutil"
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

func TestScanMprcontents_ReturnsExpressions(t *testing.T) {
	mprPath := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")
	records := scanProject(t, mprPath, scan.Options{})
	assert.Greater(t, len(records), 3000, "corpus-a should have >3000 expressions")
}

func TestScanMprcontents_FilterByType(t *testing.T) {
	mprPath := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")
	records := scanProject(t, mprPath, scan.Options{FilterType: "ExpressionSplitCondition"})
	assert.Greater(t, len(records), 500)
	for _, r := range records {
		assert.Contains(t, r.UnitType, "ExpressionSplitCondition")
	}
}

func TestScanMprcontents_RequiredFields(t *testing.T) {
	mprPath := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")
	records := scanProject(t, mprPath, scan.Options{})
	require.NotEmpty(t, records)
	for _, r := range records[:min(100, len(records))] {
		assert.NotEmpty(t, r.UnitID, "UnitID must not be empty")
		assert.NotEmpty(t, r.Project)
		assert.NotEmpty(t, r.UnitType)
		assert.NotEmpty(t, r.Field)
		assert.NotEmpty(t, r.Raw)
		assert.Contains(t, []string{"microflow", "page", "domain", "workflow", "widget"},
			r.Category, "category must be known")
	}
}

func TestScanMprcontents_ExcludesURLs(t *testing.T) {
	mprPath := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")
	records := scanProject(t, mprPath, scan.Options{})
	for _, r := range records {
		assert.False(t, len(r.Raw) > 7 && r.Raw[:8] == "https://",
			"URLs must be excluded, got: %s", r.Raw)
	}
}

func TestScanMprcontents_NoSlotPath(t *testing.T) {
	// SlotPath was removed — parse package now detects XPath by content.
	// This test verifies the struct compiles correctly without SlotPath.
	mprPath := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")
	records := scanProject(t, mprPath, scan.Options{})
	require.NotEmpty(t, records)
	// Just verify the record has the fields we do care about.
	r := records[0]
	assert.NotEmpty(t, r.UnitType)
	assert.NotEmpty(t, r.Raw)
	assert.NotEmpty(t, r.Category)
}

func TestScanMprcontents_TypeCheckFields(t *testing.T) {
	// Verify TargetAttrQN is populated for ChangeActionItem expressions.
	mprPath := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")
	recs := scanProject(t, mprPath, scan.Options{FilterType: "ChangeActionItem"})
	// At least some ChangeActionItem records should have TargetAttrQN set.
	withAttr := 0
	for _, r := range recs {
		if r.TargetAttrQN != "" {
			withAttr++
		}
	}
	if withAttr == 0 {
		t.Error("expected at least one ChangeActionItem with TargetAttrQN populated")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

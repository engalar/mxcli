// SPDX-License-Identifier: Apache-2.0

package scan_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const macnicaMpr = "/mnt/data_sdd/macnica/mendix-app/mprcontents"

func TestScanMprcontents_ReturnsExpressions(t *testing.T) {
	records, err := scan.ScanMprcontents(macnicaMpr, scan.Options{})
	require.NoError(t, err)
	assert.Greater(t, len(records), 3000, "macnica should have >3000 expressions")
}

func TestScanMprcontents_FilterByType(t *testing.T) {
	records, err := scan.ScanMprcontents(macnicaMpr, scan.Options{FilterType: "ExpressionSplitCondition"})
	require.NoError(t, err)
	assert.Greater(t, len(records), 500)
	for _, r := range records {
		assert.Contains(t, r.UnitType, "ExpressionSplitCondition")
	}
}

func TestScanMprcontents_RequiredFields(t *testing.T) {
	records, err := scan.ScanMprcontents(macnicaMpr, scan.Options{})
	require.NoError(t, err)
	require.NotEmpty(t, records)
	for _, r := range records[:min(100, len(records))] {
		assert.NotEmpty(t, r.UnitID, "UnitID must not be empty")
		assert.NotEmpty(t, r.Project)
		assert.NotEmpty(t, r.UnitType)
		assert.NotEmpty(t, r.Field)
		assert.NotEmpty(t, r.Raw)
		assert.Contains(t, []string{"microflow", "page", "domain", "workflow", "widget"},
			r.Category, "category must be known")
		assert.NotEmpty(t, r.SlotPath, "SlotPath must be set for %s.%s", r.UnitType, r.Field)
	}
}

func TestScanMprcontents_ExcludesURLs(t *testing.T) {
	records, err := scan.ScanMprcontents(macnicaMpr, scan.Options{})
	require.NoError(t, err)
	for _, r := range records {
		assert.False(t, len(r.Raw) > 7 && r.Raw[:8] == "https://",
			"URLs must be excluded, got: %s", r.Raw)
	}
}

func TestScanMprcontents_SlotPathsKnown(t *testing.T) {
	records, err := scan.ScanMprcontents(macnicaMpr, scan.Options{})
	require.NoError(t, err)
	for _, r := range records {
		assert.NotEmpty(t, r.SlotPath,
			"every record needs a SlotPath, missing for %s.%s", r.UnitType, r.Field)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

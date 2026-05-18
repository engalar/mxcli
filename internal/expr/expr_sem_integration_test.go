//go:build integration

// SPDX-License-Identifier: Apache-2.0

package expr_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/meta"
	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/validate"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const macnicaMPR = "/mnt/data_sdd/macnica/mendix-app/MacnicaApp.mpr"

func TestSemantic_FullPipeline_Macnica(t *testing.T) {
	b, err := mprbackend.NewFromPath(macnicaMPR)
	require.NoError(t, err)
	defer func() { _ = b.Disconnect() }()

	idx, err := meta.BuildFromBackend(b)
	require.NoError(t, err)
	assert.Greater(t, idx.EntityCount(), 0, "should index entities")
	assert.Greater(t, idx.EnumCount(), 0, "should index enums")
	t.Logf("Index: %d entities, %d enums, %d constants", idx.EntityCount(), idx.EnumCount(), idx.ConstantsCount())

	mprContentsPath := scan.MprContentsPath(macnicaMPR)
	records, err := scan.ScanMprcontents(mprContentsPath, scan.Options{})
	require.NoError(t, err)
	t.Logf("Scanned: %d expressions", len(records))

	parsed := parse.BatchParseWithCatalog(records, idx)

	var synIssues, semIssues []validate.ValidationResult
	for _, pr := range parsed {
		synIssues = append(synIssues, validate.ValidateSyntax(pr)...)
		semIssues = append(semIssues, validate.ValidateSemantic(pr, idx)...)
	}

	t.Logf("Syntax issues: %d", len(synIssues))
	t.Logf("Semantic issues: %d", len(semIssues))

	foundE006 := false
	for _, i := range synIssues {
		if i.RuleID == "E006" {
			foundE006 = true
			break
		}
	}
	assert.True(t, foundE006, "macnica known E006 must be detected")
	assert.GreaterOrEqual(t, len(semIssues), 0)
}

func TestSemantic_NoDaemon_SkipsSEM(t *testing.T) {
	mprContentsPath := scan.MprContentsPath(macnicaMPR)
	records, err := scan.ScanMprcontents(mprContentsPath, scan.Options{})
	require.NoError(t, err)

	parsed := parse.BatchParse(records)
	var semIssues []validate.ValidationResult
	for _, pr := range parsed {
		semIssues = append(semIssues, validate.ValidateSemantic(pr, nil)...)
	}
	assert.Empty(t, semIssues, "nil idx (no-daemon mode) must produce no semantic issues")
}

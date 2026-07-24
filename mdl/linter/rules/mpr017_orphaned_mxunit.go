// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/linter"
)

// OrphanedMxUnitRule (MPR017) detects .mxunit files in mprcontents/ that have
// no corresponding record in the MPR SQLite Unit table. These orphaned files
// accumulate from incomplete refactorings, branch switches, or tool operations
// and waste space. Use --fix to remove them.
type OrphanedMxUnitRule struct{}

// NewOrphanedMxUnitRule creates a new OrphanedMxUnitRule.
func NewOrphanedMxUnitRule() *OrphanedMxUnitRule {
	return &OrphanedMxUnitRule{}
}

func (r *OrphanedMxUnitRule) ID() string                       { return "MPR017" }
func (r *OrphanedMxUnitRule) Name() string                     { return "OrphanedMxUnit" }
func (r *OrphanedMxUnitRule) Category() string                 { return "integrity" }
func (r *OrphanedMxUnitRule) DefaultSeverity() linter.Severity { return linter.SeverityWarning }
func (r *OrphanedMxUnitRule) Description() string {
	return ".mxunit file in mprcontents/ has no corresponding Unit record in the MPR database"
}

// Check scans mprcontents/ and cross-references against the Unit table.
func (r *OrphanedMxUnitRule) Check(ctx *linter.LintContext) []linter.Violation {
	reader := ctx.Reader()
	if reader == nil {
		return nil
	}

	// 1. Get all valid unit UUIDs from SQLite.
	validIDs, err := reader.ListAllUnitIDs()
	if err != nil {
		return nil
	}
	validSet := make(map[string]struct{}, len(validIDs))
	for _, id := range validIDs {
		validSet[id] = struct{}{}
	}

	// 2. Get mprcontents directory path.
	contentsDir := reader.ContentsDir()
	if contentsDir == "" {
		return nil
	}

	// 3. Walk mprcontents/ for .mxunit files.
	var violations []linter.Violation
	_ = filepath.Walk(contentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".mxunit") {
			return nil
		}

		// Extract UUID from filename ("<UUID>.mxunit").
		uuid := strings.TrimSuffix(info.Name(), ".mxunit")
		if _, exists := validSet[uuid]; exists {
			return nil
		}

		// Orphaned: file on disk but no Unit record.
		violations = append(violations, linter.Violation{
			RuleID:     r.ID(),
			Severity:   r.DefaultSeverity(),
			Message:    fmt.Sprintf("orphaned .mxunit file: %s", info.Name()),
			Location: linter.Location{
				Module:       "(project)",
				DocumentType: "orphaned-mxunit",
				DocumentName: info.Name(),
				DocumentID:   uuid,
			},
			Suggestion: "remove the file with --fix or verify the unit should exist",
			Extra:      path,
		})
		return nil
	})

	return violations
}

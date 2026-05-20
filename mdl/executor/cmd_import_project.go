// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"path/filepath"
	"sort"
	"strings"
)

// ImportOptions controls the behaviour of ImportProject.
type ImportOptions struct {
	Module     string
	DryRun     bool
	SkipErrors bool
	Progress   func(line string)
}

// importDocumentOrder lists ordered substring matches against the relative
// MDL file path. Lower priority numbers run first.
var importDocumentOrder = []struct {
	pattern  string
	priority int
}{
	{"_marketplace.mdl", 0},
	{"_module.mdl", 1},
	{"Domain/", 2},
	{"_associations.mdl", 3},
	{"Constants/", 4},
	{"_module_roles.mdl", 5},
	{"JavaActions/", 6},
	{"JavaScriptActions/", 7},
	{"Microflows/", 8},
	{"Nanoflows/", 9},
	{"Layouts/", 10},
	{"Snippets/", 11},
	{"Pages/", 12},
	{"Workflows/", 13},
	{"_project/navigation", 14},
	{"_project/security", 15},
	{"_project/settings", 16},
}

func fileImportPriority(path string) int {
	norm := filepath.ToSlash(path)
	for _, entry := range importDocumentOrder {
		if strings.Contains(norm, entry.pattern) || strings.HasSuffix(norm, entry.pattern) {
			return entry.priority
		}
	}
	return 50
}

// sortMDLFiles returns paths sorted by document type (entities before
// associations, microflows before pages, project settings last), with
// lexicographic order as the tie-breaker for stable diffs.
func sortMDLFiles(paths []string) []string {
	out := make([]string, len(paths))
	copy(out, paths)
	sort.SliceStable(out, func(i, j int) bool {
		pi := fileImportPriority(out[i])
		pj := fileImportPriority(out[j])
		if pi != pj {
			return pi < pj
		}
		return out[i] < out[j]
	})
	return out
}

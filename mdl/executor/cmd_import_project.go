// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/visitor"
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

// ImportProject scans inputDir for .mdl files, sorts them in dependency
// order, and executes each against the connected project. _marketplace.mdl
// is always skipped (it is informational only).
func (e *Executor) ImportProject(inputDir string, opts ImportOptions) error {
	ctx := e.newExecContext(context.Background())
	if !ctx.Connected() {
		return fmt.Errorf("not connected to a project")
	}

	progress := opts.Progress
	if progress == nil {
		progress = func(string) {}
	}

	var allFiles []string
	if err := filepath.WalkDir(inputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".mdl") {
			rel, _ := filepath.Rel(inputDir, path)
			allFiles = append(allFiles, rel)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("scan %s: %w", inputDir, err)
	}

	if opts.Module != "" {
		filtered := allFiles[:0]
		for _, f := range allFiles {
			norm := filepath.ToSlash(f)
			if strings.HasPrefix(norm, opts.Module+"/") || strings.HasPrefix(norm, "_project/") {
				filtered = append(filtered, f)
			}
		}
		allFiles = filtered
	}

	filtered := allFiles[:0]
	for _, f := range allFiles {
		if filepath.Base(f) != "_marketplace.mdl" {
			filtered = append(filtered, f)
		}
	}
	allFiles = filtered

	sorted := sortMDLFiles(allFiles)

	var errs []string
	for _, rel := range sorted {
		fullPath := filepath.Join(inputDir, rel)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			msg := fmt.Sprintf("read %s: %v", rel, err)
			if !opts.SkipErrors {
				return fmt.Errorf("%s", msg)
			}
			errs = append(errs, msg)
			continue
		}

		prog, parseErrs := visitor.Build(string(content))
		if len(parseErrs) > 0 {
			for _, pe := range parseErrs {
				msg := fmt.Sprintf("parse %s: %v", rel, pe)
				if !opts.SkipErrors {
					return fmt.Errorf("%s", msg)
				}
				errs = append(errs, msg)
			}
			continue
		}

		if opts.DryRun {
			progress(fmt.Sprintf("  [dry-run]  %s", rel))
			continue
		}

		progress(fmt.Sprintf("  [exec]     %s", rel))
		if err := e.ExecuteProgram(prog); err != nil {
			msg := fmt.Sprintf("exec %s: %v", rel, err)
			if !opts.SkipErrors {
				return fmt.Errorf("%s", msg)
			}
			errs = append(errs, msg)
		}
	}

	if len(errs) > 0 {
		progress(fmt.Sprintf("  [warn]     %d file(s) had errors:", len(errs)))
		for _, e := range errs {
			progress(fmt.Sprintf("    - %s", e))
		}
	}

	return nil
}

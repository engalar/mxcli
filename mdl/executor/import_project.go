// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
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
	{"_marketplace.mdl", 0},  // informational only — skipped
	{"_module.mdl", 1},       // CREATE MODULE must come first
	{"Enumerations/", 2},     // enumerations before entities (attrs ref enums)
	{"_module_roles.mdl", 3}, // module roles BEFORE entities so GRANTs resolve
	{"Domain/", 4},           // entities (within-module order preserved by export)
	{"_associations.mdl", 5}, // associations after all entities
	{"Constants/", 6},        // constants
	{"JavaActions/", 7},
	{"JavaScriptActions/", 8},
	{"Microflows/", 9},
	{"Nanoflows/", 10},
	{"Layouts/", 11},  // layouts before pages
	{"Snippets/", 12}, // snippets before pages
	{"Pages/", 13},
	{"Workflows/", 14},
	{"_project/navigation", 15}, // navigation references pages
	{"_project/security", 16},   // user roles reference module roles
	{"_project/settings", 17},
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

// importFileBuffer is the per-file parse+cache result used across passes.
type importFileBuffer struct {
	rel     string
	content string
	prog    *ast.Program
}

// parseImportFile parses a single MDL file, skipping on parse errors.
func parseImportFile(inputDir, rel string, opts ImportOptions, progress func(string), errs *[]string) *importFileBuffer {
	fullPath := filepath.Join(inputDir, rel)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		msg := fmt.Sprintf("read %s: %v", rel, err)
		if !opts.SkipErrors {
			panic(msg) // caller handles
		}
		*errs = append(*errs, msg)
		return nil
	}

	prog, parseErrs := visitor.Build(string(content))
	if len(parseErrs) > 0 {
		for _, pe := range parseErrs {
			msg := fmt.Sprintf("parse %s: %v", rel, pe)
			if !opts.SkipErrors {
				panic(msg)
			}
			*errs = append(*errs, msg)
		}
		return nil
	}
	return &importFileBuffer{rel: rel, content: string(content), prog: prog}
}

// executeImportFile runs a single parsed file against the project.
// Panics from the executor (e.g., nil pointer dereference in page builder
// when a referenced microflow doesn't exist yet) are caught and converted
// to errors, allowing retry in a later pass.
func executeImportFile(fb *importFileBuffer, e *Executor, importBuf backend.ImportBuffer, opts ImportOptions, progress func(string), errs *[]string) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			if importBuf != nil {
				importBuf.Discard()
			}
			msg := fmt.Sprintf("panic %s: %v", fb.rel, r)
			*errs = append(*errs, msg)
			ok = false
		}
	}()

	if opts.DryRun {
		progress(fmt.Sprintf("  [dry-run]  %s", fb.rel))
		return true
	}

	progress(fmt.Sprintf("  [exec]     %s", fb.rel))
	if err := e.ExecuteProgram(fb.prog); err != nil {
		if importBuf != nil {
			importBuf.Discard()
		}
		msg := fmt.Sprintf("exec %s: %v", fb.rel, err)
		if !opts.SkipErrors {
			panic(msg)
		}
		*errs = append(*errs, msg)
		return false
	}

	if importBuf != nil {
		if flushErr := importBuf.Flush(); flushErr != nil {
			msg := fmt.Sprintf("flush %s: %v", fb.rel, flushErr)
			if !opts.SkipErrors {
				panic(msg)
			}
			*errs = append(*errs, msg)
			return false
		}
	}
	return true
}

// calleeQNFromPath extracts the qualified name (Module.Name) from an
// MDL file path under a Microflows/ or Nanoflows/ directory, e.g.
// "ss_bootstrap/Microflows/_Common/ss_bootstrap.ASU_LoadConfigOnStartup.mdl"
// → "ss_bootstrap.ASU_LoadConfigOnStartup".
func calleeQNFromPath(rel string) string {
	norm := filepath.ToSlash(rel)
	base := filepath.Base(norm)
	base = strings.TrimSuffix(base, ".mdl")
	// Base format is Module.MicroflowName — return as-is
	if strings.Contains(base, ".") {
		return base
	}
	return ""
}

// callQNsFromContent scans MDL text for CALL MICROFLOW / CALL NANOFLOW
// references and returns the set of called qualified names. Uses a simple
// regex-free scan over the statement surface.
func callQNsFromContent(content string) map[string]bool {
	calls := map[string]bool{}
	// Patterns to match: CALL MICROFLOW QN (args) or CALL NANOFLOW QN (args)
	// QN is Module.Name — two identifier segments separated by a dot.
	markers := []string{"CALL MICROFLOW ", "CALL NANOFLOW "}
	for _, marker := range markers {
		rest := content
		for {
			idx := strings.Index(rest, marker)
			if idx < 0 {
				break
			}
			after := rest[idx+len(marker):]
			// Qualified name ends at '(' or whitespace.
			end := strings.IndexAny(after, " (")
			if end < 0 {
				end = len(after)
			}
			qn := strings.TrimSpace(after[:end])
			if strings.Count(qn, ".") == 1 && strings.Contains(qn, ".") {
				calls[qn] = true
			}
			rest = after[end:]
		}
	}
	return calls
}

// dependencySortFiles reorders files so that, within each priority group,
// files defining a called microflow come before the files that call it.
// This resolves the common case where microflow A references microflow B
// but B sorts after A alphabetically.
func dependencySortFiles(bufs []*importFileBuffer, inputDir string) []*importFileBuffer {
	// Build map: qualified name → file buffer (callee index).
	calleeIndex := map[string]*importFileBuffer{}
	for _, fb := range bufs {
		if qn := calleeQNFromPath(fb.rel); qn != "" {
			calleeIndex[qn] = fb
		}
	}

	// For each file, find its callees and move the callee before it.
	// Use Kahn's algorithm: link callees before callers.
	inDegree := map[*importFileBuffer]int{}
	dependents := map[*importFileBuffer][]*importFileBuffer{} // caller → callees
	dependencyOf := map[*importFileBuffer][]*importFileBuffer{} // callee → callers

	for _, fb := range bufs {
		inDegree[fb] = 0 // init
		if _, ok := dependents[fb]; !ok {
			dependents[fb] = nil
		}
		if _, ok := dependencyOf[fb]; !ok {
			dependencyOf[fb] = nil
		}
	}

	for _, fb := range bufs {
		if qn := calleeQNFromPath(fb.rel); qn != "" {
			if _, exists := calleeIndex[qn]; !exists {
				calleeIndex[qn] = fb
			}
		}
	}

	for _, fb := range bufs {
		calls := callQNsFromContent(fb.content)
		for qn := range calls {
			callee, ok := calleeIndex[qn]
			if !ok || callee == fb {
				continue
			}
			// fb calls callee → callee must come before fb
			dependents[callee] = append(dependents[callee], fb)
			dependencyOf[fb] = append(dependencyOf[fb], callee)
			inDegree[fb]++
		}
	}

	// Kahn's algorithm: start with nodes that have no incoming edges.
	var queue []*importFileBuffer
	for _, fb := range bufs {
		if inDegree[fb] == 0 {
			queue = append(queue, fb)
		}
	}

	var result []*importFileBuffer
	visited := map[*importFileBuffer]bool{}
	for len(queue) > 0 {
		fb := queue[0]
		queue = queue[1:]
		if visited[fb] {
			continue
		}
		visited[fb] = true
		result = append(result, fb)

		for _, caller := range dependents[fb] {
			inDegree[caller]--
			if inDegree[caller] == 0 {
				queue = append(queue, caller)
			}
		}
	}

	// Append any nodes not reached (cycle or no dependencies) in original order.
	for _, fb := range bufs {
		if !visited[fb] {
			result = append(result, fb)
		}
	}

	return result
}

// ImportProject scans inputDir for .mdl files, sorts them in dependency
// order, and executes each against the connected project. _marketplace.mdl
// is always skipped (it is informational only).
//
// A second pass retries files that failed on the first pass due to
// validation errors (typically cross-module dependencies that hadn't
// been created yet). This handles the common pattern where microflow A
// calls microflow B defined later in the sort order.
func (e *Executor) ImportProject(inputDir string, opts ImportOptions) error {
	ctx := execContextFromDeps(e.buildHandlerDeps())
	if !ctx.Connected() {
		return fmt.Errorf("not connected to a project")
	}

	progress := opts.Progress
	if progress == nil {
		progress = func(string) {}
	}

	// Activate the import buffer: all updateUnit calls are buffered in memory
	// and flushed to disk as a single SQLite transaction per .mdl file.
	// This eliminates ~50 per-statement transactions per file (5-10x fewer I/O ops).
	var importBuf backend.ImportBuffer
	if bufBE, ok := ctx.Backend.(backend.ImportBufferBackend); ok {
		importBuf = bufBE.BeginImportBuffer()
		defer bufBE.DisableImportBuffer()
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

	// Phase 1: parse all files, build dependency-graph-aware order.
	var fileBufs []*importFileBuffer
	var errs []string
	for _, rel := range sorted {
		fb := parseImportFile(inputDir, rel, opts, progress, &errs)
		if fb != nil {
			fileBufs = append(fileBufs, fb)
		}
	}

	// Sort files so that callees come before callers within each priority
	// group. This resolves cross-module microflow→microflow dependencies
	// that the type-based sort alone misses.
	fileBufs = dependencySortFiles(fileBufs, inputDir)

	// Phase 2: execute all files. Failed files are collected for retry.
	var failed []*importFileBuffer
	for _, fb := range fileBufs {
		if executeImportFile(fb, e, importBuf, opts, progress, &errs) {
			continue
		}
		// First-pass file-level error: collect for retry.
		if opts.SkipErrors {
			failed = append(failed, fb)
		}
	}

	// Phase 3: retry failed files. By now all first-pass files have been
	// committed, so cross-file dependencies should be resolvable.
	if len(failed) > 0 {
		progress(fmt.Sprintf("  [retry]    %d file(s) failed on first pass — retrying...", len(failed)))
		for _, fb := range failed {
			if executeImportFile(fb, e, importBuf, opts, progress, &errs) {
				progress(fmt.Sprintf("  [retry-ok] %s", fb.rel))
			}
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

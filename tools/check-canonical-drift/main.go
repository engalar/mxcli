// SPDX-License-Identifier: Apache-2.0

// check-canonical-drift warns when staged changes touch serialization functions
// not yet migrated to the canonical model layer (mdl/model/).
// Reads git diff --cached --unified=0 from stdin. Always exits 0.
package main

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	hunkPattern  = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)
	addedFuncPat = regexp.MustCompile(`^\+func \w*(StmtToMDL|ToMDLGen)\b`)
	funcPattern  = regexp.MustCompile(`(StmtToMDL|ToMDLGen)$`)
)

type lineRange struct{ start, end int }

// unmgrFunc describes a serialization function not yet migrated to the canonical model layer.
// Populated in Task 2 (scanSource) and consumed in Task 3 (crossMatch).
type unmgrFunc struct {
	file       string
	name       string
	start, end int
}

// violation is a cross-match hit: a staged change touched an unmigtated function.
// Produced by crossMatch in Task 3.
type violation struct {
	file   string
	name   string
	reason string // "modified" or "added"
}

// parseDiff parses unified diff text (--unified=0 format) and returns:
//   - changed: executor file path → new-side changed line ranges
//   - newUnmigrated: names of newly added *StmtToMDL/*ToMDLGen functions without .ToMDL()
func parseDiff(diffText string) (changed map[string][]lineRange, newUnmigrated []string) {
	changed = make(map[string][]lineRange)

	var currentFile string
	var addedFuncName string
	var addedFuncHasToMDL bool

	finishAddedFunc := func() {
		if addedFuncName != "" && !addedFuncHasToMDL {
			newUnmigrated = append(newUnmigrated, addedFuncName)
		}
		addedFuncName = ""
		addedFuncHasToMDL = false
	}

	scanner := bufio.NewScanner(strings.NewReader(diffText))
	for scanner.Scan() {
		line := scanner.Text()

		// File header: +++ b/mdl/executor/foo.go
		if strings.HasPrefix(line, "+++ b/") {
			finishAddedFunc()
			path := strings.TrimPrefix(line, "+++ b/")
			if strings.HasPrefix(path, "mdl/executor/") &&
				strings.HasSuffix(path, ".go") &&
				!strings.HasSuffix(path, "_test.go") {
				currentFile = path
			} else {
				currentFile = ""
			}
			continue
		}

		if currentFile == "" {
			continue
		}

		// Hunk header: @@ -old[,count] +new[,count] @@
		if m := hunkPattern.FindStringSubmatch(line); m != nil {
			finishAddedFunc()
			start, _ := strconv.Atoi(m[1])
			count := 1
			if m[2] != "" {
				count, _ = strconv.Atoi(m[2])
			}
			if count > 0 {
				changed[currentFile] = append(changed[currentFile],
					lineRange{start, start + count - 1})
			}
			continue
		}

		if !strings.HasPrefix(line, "+") {
			continue
		}
		content := line[1:]

		// Detect a newly added function matching the pattern
		if addedFuncPat.MatchString(line) {
			finishAddedFunc()
			// Extract name: "func <name>(" — skip "func " (5 chars), take up to "("
			idx := strings.Index(content, "(")
			if idx > 5 {
				addedFuncName = strings.TrimSpace(content[5:idx])
			}
		}

		if addedFuncName != "" && strings.Contains(content, ".ToMDL()") {
			addedFuncHasToMDL = true
		}
	}
	finishAddedFunc()
	return
}

// scanSource parses Go source text and returns unmigtated serialization functions.
// A function is unmigtated if its name matches *StmtToMDL/*ToMDLGen and its
// body contains no .ToMDL() call.
func scanSource(file, src string) []unmgrFunc {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, src, 0)
	if err != nil {
		return nil
	}
	var results []unmgrFunc
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		if !funcPattern.MatchString(fn.Name.Name) {
			continue
		}
		if hasToMDLCall(fn.Body) {
			continue
		}
		results = append(results, unmgrFunc{
			file:  file,
			name:  fn.Name.Name,
			start: fset.Position(fn.Pos()).Line,
			end:   fset.Position(fn.End()).Line,
		})
	}
	return results
}

// hasToMDLCall reports whether the AST subtree contains any .ToMDL() selector call.
func hasToMDLCall(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}
		sel, ok := n.(*ast.SelectorExpr)
		if ok && sel.Sel.Name == "ToMDL" {
			found = true
		}
		return !found
	})
	return found
}

func main() {
	// TODO: wire parseDiff + scanExecutor + crossMatch + printWarning (Task 3).
	os.Exit(0)
}

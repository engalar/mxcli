# Canonical Model Drift Guard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a non-blocking pre-commit check that warns when staged changes touch serialization functions not yet migrated to the canonical model layer (`mdl/model/`).

**Architecture:** A Go tool (`tools/check-canonical-drift/main.go`) reads `git diff --cached --unified=0` from stdin, uses `go/ast` to find unmigtated `*StmtToMDL`/`*ToMDLGen` functions in `mdl/executor/`, cross-matches changed line ranges, and prints a warning to stderr. A shell wrapper (`06-canonical-model-drift.sh`) invokes it; both always exit 0.

**Tech Stack:** Go 1.26, standard library (`go/ast`, `go/parser`, `go/token`, `bufio`, `io/fs`, `regexp`), `github.com/stretchr/testify` for tests, module `github.com/mendixlabs/mxcli`.

**Spec:** `docs/superpowers/specs/2026-05-31-canonical-drift-guard-design.md`

---

## File Map

| File | Responsibility |
|------|---------------|
| `tools/check-canonical-drift/main.go` | Types, diff parser, AST scanner, cross-matcher, warning printer, `main()` — built incrementally across tasks |
| `tools/check-canonical-drift/drift_test.go` | Unit tests — one `TestX_*` block per function, added before its implementation |
| `.githooks/checks/06-canonical-model-drift.sh` | Shell entry point, always exit 0 |

---

## Task 1: Types + diff parser (TDD)

**Files:**
- Create: `tools/check-canonical-drift/drift_test.go` (parseDiff tests only)
- Create: `tools/check-canonical-drift/main.go` (types + parseDiff + stub main)

- [ ] **Step 1: Write failing parseDiff tests**

```go
// tools/check-canonical-drift/drift_test.go
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDiff_ChangedLines(t *testing.T) {
	diff := "diff --git a/mdl/executor/cmd_diff_mdl.go b/mdl/executor/cmd_diff_mdl.go\n" +
		"index abc..def 100644\n" +
		"--- a/mdl/executor/cmd_diff_mdl.go\n" +
		"+++ b/mdl/executor/cmd_diff_mdl.go\n" +
		"@@ -93,3 +93,3 @@\n" +
		" unchanged\n" +
		"-old\n" +
		"+new\n"
	changed, _ := parseDiff(diff)
	ranges := changed["mdl/executor/cmd_diff_mdl.go"]
	require.Len(t, ranges, 1)
	assert.Equal(t, 93, ranges[0].start)
	assert.Equal(t, 95, ranges[0].end)
}

func TestParseDiff_MultipleHunks(t *testing.T) {
	diff := "+++ b/mdl/executor/cmd_diff_mdl.go\n" +
		"@@ -10,2 +10,2 @@\n" +
		"-old\n" +
		"+new\n" +
		"@@ -50,1 +50,1 @@\n" +
		"-old2\n" +
		"+new2\n"
	changed, _ := parseDiff(diff)
	ranges := changed["mdl/executor/cmd_diff_mdl.go"]
	require.Len(t, ranges, 2)
	assert.Equal(t, 10, ranges[0].start)
	assert.Equal(t, 11, ranges[0].end)
	assert.Equal(t, 50, ranges[1].start)
	assert.Equal(t, 50, ranges[1].end)
}

func TestParseDiff_SkipsNonExecutor(t *testing.T) {
	diff := "+++ b/cmd/mxcli/main.go\n@@ -1 +1 @@\n+line\n"
	changed, _ := parseDiff(diff)
	assert.Empty(t, changed)
}

func TestParseDiff_SkipsTestFiles(t *testing.T) {
	diff := "+++ b/mdl/executor/cmd_diff_mdl_test.go\n@@ -1 +1 @@\n+line\n"
	changed, _ := parseDiff(diff)
	assert.Empty(t, changed)
}

func TestParseDiff_ZeroCountHunk(t *testing.T) {
	// @@ -5,3 +5,0 @@ means 3 lines deleted, 0 added — no new-side range
	diff := "+++ b/mdl/executor/cmd_diff_mdl.go\n@@ -5,3 +5,0 @@\n-del1\n-del2\n-del3\n"
	changed, _ := parseDiff(diff)
	assert.Empty(t, changed["mdl/executor/cmd_diff_mdl.go"])
}

func TestParseDiff_NewUnmigratedFunc(t *testing.T) {
	diff := "+++ b/mdl/executor/cmd_diff_mdl.go\n" +
		"@@ -0,0 +1,4 @@\n" +
		"+func fooStmtToMDL(ctx *ExecContext) string {\n" +
		"+\treturn \"foo\"\n" +
		"+}\n"
	_, newFuncs := parseDiff(diff)
	require.Len(t, newFuncs, 1)
	assert.Equal(t, "fooStmtToMDL", newFuncs[0])
}

func TestParseDiff_NewMigratedFunc(t *testing.T) {
	diff := "+++ b/mdl/executor/cmd_diff_mdl.go\n" +
		"@@ -0,0 +1,4 @@\n" +
		"+func fooStmtToMDL(ctx *ExecContext) string {\n" +
		"+\treturn m.ToMDL()\n" +
		"+}\n"
	_, newFuncs := parseDiff(diff)
	assert.Empty(t, newFuncs)
}

func TestParseDiff_NewToMDLGenFunc(t *testing.T) {
	diff := "+++ b/mdl/executor/cmd_entities_gen.go\n" +
		"@@ -0,0 +1,3 @@\n" +
		"+func barToMDLGen(ctx *ExecContext) string {\n" +
		"+\treturn \"bar\"\n" +
		"+}\n"
	_, newFuncs := parseDiff(diff)
	require.Len(t, newFuncs, 1)
	assert.Equal(t, "barToMDLGen", newFuncs[0])
}
```

- [ ] **Step 2: Run tests — expect compile failure**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02 && go test ./tools/check-canonical-drift/... 2>&1 | head -5
```

Expected: `cannot find package` — the package doesn't exist yet.

- [ ] **Step 3: Create `main.go` with types + `parseDiff` + stub `main`**

```go
// SPDX-License-Identifier: Apache-2.0

// check-canonical-drift warns when staged changes touch serialization functions
// not yet migrated to the canonical model layer (mdl/model/).
// Reads git diff --cached --unified=0 from stdin. Always exits 0.
package main

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	hunkPattern  = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)
	addedFuncPat = regexp.MustCompile(`^\+func \w*(StmtToMDL|ToMDLGen)\b`)
)

type lineRange struct{ start, end int }

type unmgrFunc struct {
	file       string
	name       string
	start, end int
}

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

func main() {
	os.Exit(0)
}
```

- [ ] **Step 4: Run parseDiff tests — expect pass**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02 && go test ./tools/check-canonical-drift/... -run TestParseDiff -v
```

Expected: all 8 `TestParseDiff_*` tests pass. If `TestParseDiff_ChangedLines` fails, check the hunk line-range arithmetic: `start + count - 1` where count defaults to 1 when the `,count` part is absent.

- [ ] **Step 5: Commit**

```bash
git add tools/check-canonical-drift/main.go tools/check-canonical-drift/drift_test.go
git commit -m "feat(tools): add canonical drift guard skeleton + diff parser (TDD)"
```

---

## Task 2: AST scanner (TDD)

**Files:**
- Modify: `tools/check-canonical-drift/drift_test.go` (add scanSource tests)
- Modify: `tools/check-canonical-drift/main.go` (add scanSource, hasToMDLCall)

- [ ] **Step 1: Append scanSource tests to `drift_test.go`**

Add after the last `TestParseDiff_*` function:

```go
// --- scanSource tests ---

const srcMigrated = `package executor

func entityStmtToMDL(s *ast.CreateEntityStmt) string {
	m, _ := entity.Lift(s)
	return m.ToMDL()
}
`

const srcUnmigrated = `package executor

import "strings"

func associationStmtToMDL(s *ast.CreateAssociationStmt) string {
	var sb strings.Builder
	sb.WriteString("create association ")
	return sb.String()
}
`

const srcToMDLGen = `package executor

func viewEntityToMDLGen(mod string, e *genDm.Entity) string {
	var sb strings.Builder
	sb.WriteString(e.Name())
	return sb.String()
}
`

const srcNonMatching = `package executor

func execCreateEntity(ctx *ExecContext, s *ast.CreateEntityStmt) error {
	return nil
}
`

const srcMultiple = `package executor

import "strings"

func enumerationStmtToMDL(s *ast.CreateEnumerationStmt) string {
	var sb strings.Builder
	sb.WriteString("create enumeration")
	return sb.String()
}

func microflowStmtToMDL(s *ast.CreateMicroflowStmt) string {
	var sb strings.Builder
	sb.WriteString("create microflow")
	return sb.String()
}

func entityStmtToMDL(s *ast.CreateEntityStmt) string {
	m, _ := entity.Lift(s)
	return m.ToMDL()
}
`

func TestScanSource_MigratedNotReported(t *testing.T) {
	fns := scanSource("fake.go", srcMigrated)
	assert.Empty(t, fns)
}

func TestScanSource_UnmigratedReported(t *testing.T) {
	fns := scanSource("fake.go", srcUnmigrated)
	require.Len(t, fns, 1)
	assert.Equal(t, "associationStmtToMDL", fns[0].name)
	assert.Greater(t, fns[0].end, fns[0].start)
}

func TestScanSource_ToMDLGenReported(t *testing.T) {
	fns := scanSource("fake.go", srcToMDLGen)
	require.Len(t, fns, 1)
	assert.Equal(t, "viewEntityToMDLGen", fns[0].name)
}

func TestScanSource_NonMatchingSkipped(t *testing.T) {
	fns := scanSource("fake.go", srcNonMatching)
	assert.Empty(t, fns)
}

func TestScanSource_MultipleOnlyUnmigrated(t *testing.T) {
	fns := scanSource("fake.go", srcMultiple)
	require.Len(t, fns, 2)
	names := []string{fns[0].name, fns[1].name}
	assert.Contains(t, names, "enumerationStmtToMDL")
	assert.Contains(t, names, "microflowStmtToMDL")
}

func TestScanSource_LineRangesPopulated(t *testing.T) {
	fns := scanSource("fake.go", srcUnmigrated)
	require.Len(t, fns, 1)
	assert.Greater(t, fns[0].start, 0)
	assert.GreaterOrEqual(t, fns[0].end, fns[0].start)
}
```

- [ ] **Step 2: Run tests — expect compile failure**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02 && go test ./tools/check-canonical-drift/... -run TestScanSource 2>&1 | head -5
```

Expected: `scanSource undefined` compile error.

- [ ] **Step 3: Add `scanSource` and `hasToMDLCall` to `main.go`**

Add these imports to the existing import block in `main.go`:

```go
"go/ast"
"go/parser"
"go/token"
```

Add these functions after `parseDiff`:

```go
var funcPattern = regexp.MustCompile(`(StmtToMDL|ToMDLGen)$`)

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
```

- [ ] **Step 4: Run scanSource tests — expect pass**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02 && go test ./tools/check-canonical-drift/... -run TestScanSource -v
```

Expected: all 6 `TestScanSource_*` tests pass. If `TestScanSource_MultipleOnlyUnmigrated` returns 3 instead of 2, `hasToMDLCall` is not detecting `m.ToMDL()` in `entityStmtToMDL` — check the `SelectorExpr` walk.

- [ ] **Step 5: Commit**

```bash
git add tools/check-canonical-drift/main.go tools/check-canonical-drift/drift_test.go
git commit -m "feat(tools): add AST scanner — scanSource + hasToMDLCall (TDD)"
```

---

## Task 3: Cross-match + wire main (TDD)

**Files:**
- Modify: `tools/check-canonical-drift/drift_test.go` (add crossMatch tests)
- Modify: `tools/check-canonical-drift/main.go` (add crossMatch, scanExecutor, printWarning, wire main)

- [ ] **Step 1: Append crossMatch tests to `drift_test.go`**

Add after the last `TestScanSource_*` function:

```go
// --- crossMatch tests ---

func TestCrossMatch_TouchedFunction(t *testing.T) {
	fns := []unmgrFunc{{
		file: "mdl/executor/cmd_diff_mdl.go",
		name: "associationStmtToMDL", start: 93, end: 130,
	}}
	changed := map[string][]lineRange{
		"mdl/executor/cmd_diff_mdl.go": {{100, 105}},
	}
	v := crossMatch(fns, changed)
	require.Len(t, v, 1)
	assert.Equal(t, "associationStmtToMDL", v[0].name)
	assert.Equal(t, "modified", v[0].reason)
}

func TestCrossMatch_UntouchedFunction(t *testing.T) {
	fns := []unmgrFunc{{
		file: "mdl/executor/cmd_diff_mdl.go",
		name: "associationStmtToMDL", start: 93, end: 130,
	}}
	changed := map[string][]lineRange{
		"mdl/executor/cmd_diff_mdl.go": {{200, 210}},
	}
	v := crossMatch(fns, changed)
	assert.Empty(t, v)
}

func TestCrossMatch_DifferentFile(t *testing.T) {
	fns := []unmgrFunc{{
		file: "mdl/executor/cmd_diff_mdl.go",
		name: "associationStmtToMDL", start: 93, end: 130,
	}}
	changed := map[string][]lineRange{
		"mdl/executor/cmd_entities_gen.go": {{100, 110}},
	}
	v := crossMatch(fns, changed)
	assert.Empty(t, v)
}

func TestCrossMatch_BoundaryStart(t *testing.T) {
	fns := []unmgrFunc{{
		file: "mdl/executor/cmd_diff_mdl.go",
		name: "microflowStmtToMDL", start: 50, end: 80,
	}}
	changed := map[string][]lineRange{
		"mdl/executor/cmd_diff_mdl.go": {{50, 50}},
	}
	v := crossMatch(fns, changed)
	require.Len(t, v, 1)
}

func TestCrossMatch_BoundaryEnd(t *testing.T) {
	fns := []unmgrFunc{{
		file: "mdl/executor/cmd_diff_mdl.go",
		name: "microflowStmtToMDL", start: 50, end: 80,
	}}
	changed := map[string][]lineRange{
		"mdl/executor/cmd_diff_mdl.go": {{80, 85}},
	}
	v := crossMatch(fns, changed)
	require.Len(t, v, 1)
}

func TestCrossMatch_NoDoubleCounting(t *testing.T) {
	// Two overlapping ranges both touching the same function → one violation only
	fns := []unmgrFunc{{
		file: "mdl/executor/cmd_diff_mdl.go",
		name: "associationStmtToMDL", start: 93, end: 130,
	}}
	changed := map[string][]lineRange{
		"mdl/executor/cmd_diff_mdl.go": {{100, 105}, {110, 115}},
	}
	v := crossMatch(fns, changed)
	require.Len(t, v, 1)
}
```

- [ ] **Step 2: Run tests — expect compile failure**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02 && go test ./tools/check-canonical-drift/... -run TestCrossMatch 2>&1 | head -5
```

Expected: `crossMatch undefined` compile error.

- [ ] **Step 3: Add `crossMatch`, `scanExecutor`, `printWarning`, and wire `main` in `main.go`**

Add these imports to the import block:

```go
"fmt"
"io"
"io/fs"
"path/filepath"
```

Add these functions after `hasToMDLCall`:

```go
// crossMatch returns violations where staged changed lines intersect unmigtated functions.
func crossMatch(fns []unmgrFunc, changed map[string][]lineRange) []violation {
	var violations []violation
	for _, fn := range fns {
		for _, r := range changed[fn.file] {
			if r.end >= fn.start && r.start <= fn.end {
				violations = append(violations, violation{
					file: fn.file, name: fn.name, reason: "modified",
				})
				break
			}
		}
	}
	return violations
}

// scanExecutor walks dir and collects all unmigtated functions in non-test .go files.
func scanExecutor(dir string) ([]unmgrFunc, error) {
	var results []unmgrFunc
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		results = append(results, scanSource(path, string(src))...)
		return nil
	})
	return results, err
}

func printWarning(violations []violation) {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "⚠ CANONICAL MODEL DRIFT WARNING (non-blocking)")
	fmt.Fprintln(os.Stderr)
	for _, v := range violations {
		if v.reason == "added" {
			fmt.Fprintf(os.Stderr, "  [new] %s\n", v.name)
		} else {
			short := strings.TrimPrefix(v.file, "mdl/executor/")
			fmt.Fprintf(os.Stderr, "  %s: %s\n", short, v.name)
		}
		fmt.Fprintln(os.Stderr, "    Not yet migrated to the canonical model layer.")
		fmt.Fprintln(os.Stderr, "    Editing risks divergence between diff and describe paths.")
		fmt.Fprintln(os.Stderr)
	}
	fmt.Fprintln(os.Stderr, "Migration plan: docs/superpowers/plans/2026-05-23-canonical-model-layer-phase1.md")
	fmt.Fprintln(os.Stderr, "Silence: migrate the domain (add .ToMDL() delegation in function body).")
}
```

Replace the stub `main()` with the real one:

```go
func main() {
	diffBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "check-canonical-drift: read stdin:", err)
		os.Exit(0)
	}

	changed, newUnmigrated := parseDiff(string(diffBytes))

	fns, err := scanExecutor("mdl/executor")
	if err != nil {
		fmt.Fprintln(os.Stderr, "check-canonical-drift: scan executor:", err)
		os.Exit(0)
	}

	violations := crossMatch(fns, changed)
	for _, name := range newUnmigrated {
		violations = append(violations, violation{file: "staged", name: name, reason: "added"})
	}

	if len(violations) > 0 {
		printWarning(violations)
	}
	os.Exit(0)
}
```

- [ ] **Step 4: Run all tests — expect full pass**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02 && go test ./tools/check-canonical-drift/... -v
```

Expected: all tests pass (8 parseDiff + 6 scanSource + 6 crossMatch = 20 tests). Fix any import errors first.

- [ ] **Step 5: Verify tool builds**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02 && go build -o /tmp/check-canonical-drift ./tools/check-canonical-drift/ && echo "build ok" && rm /tmp/check-canonical-drift
```

Expected: `build ok`.

- [ ] **Step 6: Commit**

```bash
git add tools/check-canonical-drift/main.go tools/check-canonical-drift/drift_test.go
git commit -m "feat(tools): add crossMatch, scanExecutor, printWarning — wire main() (TDD)"
```

---

## Task 4: Shell wrapper + smoke tests

**Files:**
- Create: `.githooks/checks/06-canonical-model-drift.sh`

- [ ] **Step 1: Create the shell wrapper**

```bash
cat > /mnt/data_sdd/gh/mxcli-wt-02/.githooks/checks/06-canonical-model-drift.sh << 'EOF'
#!/bin/sh
# Warn when staged changes touch non-migrated canonical model serialization functions.
# Non-blocking: always exit 0.
# Spec: docs/superpowers/specs/2026-05-31-canonical-drift-guard-design.md

STAGED=$(git diff --cached --name-only | grep "^mdl/executor/.*\.go$" | grep -v "_test\.go")
[ -z "$STAGED" ] && exit 0

git diff --cached --unified=0 | go run ./tools/check-canonical-drift/ >&2
exit 0
EOF
chmod +x /mnt/data_sdd/gh/mxcli-wt-02/.githooks/checks/06-canonical-model-drift.sh
```

- [ ] **Step 2: Smoke test A — no executor files staged → silent**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
echo "test" > /tmp/not_executor.txt
# Pipe an empty diff (no executor files) through the tool
echo "" | go run ./tools/check-canonical-drift/ 2>&1
```

Expected: no output.

- [ ] **Step 3: Smoke test B — change inside unmigtated function → warning**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02

# Find the line number of enumerationStmtToMDL's opening brace
LINE=$(grep -n "^func enumerationStmtToMDL" mdl/executor/cmd_diff_mdl.go | cut -d: -f1)
echo "enumerationStmtToMDL is at line $LINE"

# Insert a harmless comment line inside the function
sed -i "$((LINE+1))s/^/\t\/\/ drift-guard smoke test\n/" mdl/executor/cmd_diff_mdl.go

# Stage and run the check tool directly
git add mdl/executor/cmd_diff_mdl.go
git diff --cached --unified=0 | go run ./tools/check-canonical-drift/ 2>&1

# Restore
git checkout -- mdl/executor/cmd_diff_mdl.go
```

Expected output contains:
```
⚠ CANONICAL MODEL DRIFT WARNING (non-blocking)

  cmd_diff_mdl.go: enumerationStmtToMDL
    Not yet migrated to the canonical model layer.
```

- [ ] **Step 4: Smoke test C — change inside migrated function → no warning**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02

LINE=$(grep -n "^func entityStmtToMDL" mdl/executor/cmd_diff_mdl.go | cut -d: -f1)
echo "entityStmtToMDL is at line $LINE"

sed -i "$((LINE+1))s/^/\t\/\/ drift-guard migrated smoke test\n/" mdl/executor/cmd_diff_mdl.go
git add mdl/executor/cmd_diff_mdl.go
git diff --cached --unified=0 | go run ./tools/check-canonical-drift/ 2>&1

git checkout -- mdl/executor/cmd_diff_mdl.go
```

Expected: no output (entity is already migrated).

- [ ] **Step 5: Commit**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add .githooks/checks/06-canonical-model-drift.sh
git commit -m "feat(githooks): add check 06 — canonical model drift warning (non-blocking)"
```

---

## Self-Review Checklist

After completing all tasks, verify:

- [ ] `go test ./tools/check-canonical-drift/... -v` — all 20 tests pass
- [ ] `go build ./tools/check-canonical-drift/...` — no errors
- [ ] Smoke test B triggers warning for `enumerationStmtToMDL`
- [ ] Smoke test C produces no output for `entityStmtToMDL`
- [ ] Shell script is executable and exits 0 with no executor changes staged
- [ ] Adding `+func fooStmtToMDL` without `.ToMDL()` triggers warning (covered by `TestParseDiff_NewUnmigratedFunc`)
- [ ] Adding `+func fooStmtToMDL` with `.ToMDL()` produces no warning (covered by `TestParseDiff_NewMigratedFunc`)
- [ ] `go test ./...` from repo root — no regressions in other packages

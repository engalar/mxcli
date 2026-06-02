# Canonical Model Layer — Architectural Guards Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a SOLID `internal/archtest` rule engine and wire five architectural guard tests that catch drift in the `mdl/model/` canonical layer before it reaches `main`.

**Architecture:** `internal/archtest` defines a `Rule` interface + `Check()` entry point. Three concrete rules (`NoImport`, `PackageName`, `CodecComplete`) live in separate files with no shared state. Five guard test files invoke `Check()` with domain-specific hints; failing tests are self-describing fix guides.

**Tech Stack:** Go 1.26, `go/parser`, `go/token`, `go/ast`, `path/filepath`, standard `testing` package. No third-party tools.

**Spec:** `docs/superpowers/specs/2026-06-02-canonical-guards-design.md`

**Red/green baseline:** Guards 1 and 4 start RED immediately (current violations in `context.go` and `cmd_entities_gen.go`/`cmd_diff_mdl.go`). Guards 2, 3, and 5 start GREEN as regression sentinels — they turn RED if someone breaks the contract going forward, or when the `HydrateFn` interface is upgraded in a future plan.

---

## File Map

**New files:**

| File | Responsibility |
|------|----------------|
| `internal/archtest/archtest.go` | `Rule`, `Violation`, `Package`, `File`, `Check()` |
| `internal/archtest/archtest_test.go` | Tests for `Package.Files()`, `Check()` formatting |
| `internal/archtest/no_import.go` | `NoImport` rule |
| `internal/archtest/no_import_test.go` | Tests for `NoImport` |
| `internal/archtest/package_name.go` | `PackageName` rule |
| `internal/archtest/package_name_test.go` | Tests for `PackageName` |
| `internal/archtest/codec_complete.go` | `CodecComplete` rule |
| `internal/archtest/codec_complete_test.go` | Tests for `CodecComplete` |
| `mdl/model/import_guard_test.go` | Guard 1 — root package import direction (RED) |
| `mdl/model/entity/comply_test.go` | Guard 2 — interface compliance (GREEN baseline) |
| `mdl/model/codec_guard_test.go` | Guard 3 — codec completeness (GREEN baseline) |
| `mdl/executor/boundary_guard_test.go` | Guard 4 — executor boundary (RED) |
| `mdl/model/naming_guard_test.go` | Guard 5 — package structure (GREEN baseline) |

**Modified files:**

| File | Change |
|------|--------|
| `mdl/model/registry.go` | Add `Lookup(genTypeName string) (Codec, bool)` method |

---

## Task 1: `internal/archtest` core — `Rule`, `Violation`, `Package`, `File`, `Check()`

**Files:**
- Create: `internal/archtest/archtest.go`
- Create: `internal/archtest/archtest_test.go`

- [ ] **Step 1: Create `internal/archtest/archtest.go`**

```go
// SPDX-License-Identifier: Apache-2.0

// Package archtest provides a lightweight rule engine for architectural
// guard tests. Each Rule implementation checks one structural property;
// failing tests carry actionable Hints so implementers know exactly what
// to fix. See docs/superpowers/specs/2026-06-02-canonical-guards-design.md.
package archtest

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Rule is the single abstraction for an architectural constraint.
// Each implementation checks one property and returns all violations found.
type Rule interface {
	Name() string
	Check(pkg Package) []Violation
}

// Violation is one failed rule instance.
// Message describes what is wrong; Hint is a concrete fix guide for the
// implementer (including AI agents) — it is written by the guard-file
// author, not by the rule implementation.
type Violation struct {
	File    string
	Line    int
	Message string
	Hint    string
}

// Package is the input to file-scanning rules.
// Files() returns only root-level non-test .go files.
// Walk() recurses into subdirectories.
type Package struct {
	Dir string
}

// Files returns all non-test .go files in the root directory (not recursive).
func (p Package) Files() []File {
	entries, err := os.ReadDir(p.Dir)
	if err != nil {
		return nil
	}
	var files []File
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if f, ok := parseGoFile(filepath.Join(p.Dir, name), name); ok {
			files = append(files, f)
		}
	}
	return files
}

// Walk returns all non-test .go files recursively, including subdirectories.
func (p Package) Walk() []File {
	var files []File
	_ = filepath.WalkDir(p.Dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		if f, ok := parseGoFile(path, name); ok {
			files = append(files, f)
		}
		return nil
	})
	return files
}

// File is a parsed non-test Go source file.
type File struct {
	Name    string   // basename (e.g. "context.go")
	Path    string   // full filesystem path
	Imports []string // import paths, e.g. "github.com/foo/bar"
	lineOf  map[string]int
}

// Line returns the source line number of the given import path, or 0 if not found.
func (f File) Line(importPath string) int { return f.lineOf[importPath] }

// Check runs rules against pkg and reports all violations via t.Errorf.
// The Hint (if non-empty) is indented and appended to the error message.
func Check(t testing.TB, dir string, rules ...Rule) {
	t.Helper()
	pkg := Package{Dir: dir}
	for _, rule := range rules {
		for _, v := range rule.Check(pkg) {
			if v.Hint != "" {
				t.Errorf("%s:%d: %s [%s]\n      HINT: %s",
					v.File, v.Line, v.Message, rule.Name(),
					strings.ReplaceAll(v.Hint, "\n", "\n      "))
			} else {
				t.Errorf("%s:%d: %s [%s]", v.File, v.Line, v.Message, rule.Name())
			}
		}
	}
}

// parseGoFile parses a Go source file (imports only) and returns a File.
func parseGoFile(path, name string) (File, bool) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return File{}, false
	}
	lineOf := make(map[string]int, len(f.Imports))
	var imports []string
	for _, imp := range f.Imports {
		p := strings.Trim(imp.Path.Value, `"`)
		imports = append(imports, p)
		lineOf[p] = fset.Position(imp.Path.Pos()).Line
	}
	return File{Name: name, Path: path, Imports: imports, lineOf: lineOf}, true
}

// Sprintf is a convenience alias exposed for rule implementations.
var Sprintf = fmt.Sprintf
```

- [ ] **Step 2: Create `internal/archtest/archtest_test.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package archtest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/internal/archtest"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestPackage_Files_excludesTestFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "foo.go", "package foo\n")
	writeFile(t, dir, "foo_test.go", "package foo_test\n")

	files := archtest.Package{Dir: dir}.Files()
	if len(files) != 1 {
		t.Fatalf("want 1 file, got %d", len(files))
	}
	if files[0].Name != "foo.go" {
		t.Errorf("want foo.go, got %s", files[0].Name)
	}
}

func TestPackage_Files_excludesSubdirs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "root.go", "package root\n")
	sub := filepath.Join(dir, "sub")
	_ = os.Mkdir(sub, 0755)
	writeFile(t, sub, "sub.go", "package sub\n")

	files := archtest.Package{Dir: dir}.Files()
	if len(files) != 1 {
		t.Fatalf("Files() must not recurse into subdirs; got %d files", len(files))
	}
}

func TestPackage_Walk_includesSubdirs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "root.go", "package root\n")
	sub := filepath.Join(dir, "sub")
	_ = os.Mkdir(sub, 0755)
	writeFile(t, sub, "sub.go", "package sub\n")

	files := archtest.Package{Dir: dir}.Walk()
	if len(files) != 2 {
		t.Fatalf("Walk() should include sub dir files; got %d", len(files))
	}
}

func TestFile_Line_returnsImportLineNumber(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", `package a

import "github.com/foo/bar"
`)
	files := archtest.Package{Dir: dir}.Files()
	if len(files) != 1 {
		t.Fatal("expected 1 file")
	}
	line := files[0].Line("github.com/foo/bar")
	if line != 3 {
		t.Errorf("want line 3, got %d", line)
	}
}
```

- [ ] **Step 3: Compile and test**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./internal/archtest/... -v -run TestPackage
```

Expected: all `TestPackage_*` and `TestFile_*` tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/archtest/archtest.go internal/archtest/archtest_test.go
git commit -m "feat(archtest): add Rule engine core — Package, File, Check()"
```

---

## Task 2: `NoImport` rule

**Files:**
- Create: `internal/archtest/no_import.go`
- Create: `internal/archtest/no_import_test.go`

- [ ] **Step 1: Write failing tests in `no_import_test.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package archtest_test

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/internal/archtest"
)

func TestNoImport_findsViolation(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bad.go", `package foo
import "github.com/forbidden/pkg"
`)
	rule := archtest.NoImport{
		Forbidden: []string{"github.com/forbidden/pkg"},
		Hint:      "remove forbidden import",
	}
	violations := rule.Check(archtest.Package{Dir: dir})
	if len(violations) != 1 {
		t.Fatalf("want 1 violation, got %d", len(violations))
	}
	if !strings.Contains(violations[0].Message, "github.com/forbidden/pkg") {
		t.Errorf("violation message should name the import: %s", violations[0].Message)
	}
	if violations[0].Hint != "remove forbidden import" {
		t.Errorf("hint not propagated: %q", violations[0].Hint)
	}
}

func TestNoImport_prefixMatch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", `package foo
import "github.com/forbidden/pkg/sub"
`)
	rule := archtest.NoImport{Forbidden: []string{"github.com/forbidden/"}}
	violations := rule.Check(archtest.Package{Dir: dir})
	if len(violations) != 1 {
		t.Fatalf("prefix match should catch sub-packages; got %d violations", len(violations))
	}
}

func TestNoImport_allowlist_exempts_file(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "exempt.go", `package foo
import "github.com/forbidden/pkg"
`)
	rule := archtest.NoImport{
		Forbidden: []string{"github.com/forbidden/pkg"},
		Allowlist: map[string]bool{"exempt.go": true},
	}
	if violations := rule.Check(archtest.Package{Dir: dir}); len(violations) != 0 {
		t.Errorf("allowlisted file should not produce violations")
	}
}

func TestNoImport_clean_file_passes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "ok.go", `package foo
import "fmt"
`)
	rule := archtest.NoImport{Forbidden: []string{"github.com/forbidden/"}}
	if v := rule.Check(archtest.Package{Dir: dir}); len(v) != 0 {
		t.Errorf("clean file should have no violations")
	}
}
```

- [ ] **Step 2: Run tests — expect compile failure**

```bash
go test ./internal/archtest/... -run TestNoImport 2>&1 | head -10
```

Expected: `archtest.NoImport undefined`.

- [ ] **Step 3: Create `internal/archtest/no_import.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package archtest

import "strings"

// NoImport fails for any non-test source file in pkg.Files() that imports
// a path matching any Forbidden prefix, unless the file basename is in Allowlist.
// Hint is propagated to every Violation so test output is a fix guide.
type NoImport struct {
	Forbidden []string
	Allowlist map[string]bool
	Hint      string
}

func (r NoImport) Name() string { return "NoImport" }

func (r NoImport) Check(pkg Package) []Violation {
	var violations []Violation
	for _, f := range pkg.Files() {
		if r.Allowlist[f.Name] {
			continue
		}
		for _, imp := range f.Imports {
			for _, forbidden := range r.Forbidden {
				if imp == forbidden || strings.HasPrefix(imp, forbidden) {
					violations = append(violations, Violation{
						File:    f.Name,
						Line:    f.Line(imp),
						Message: Sprintf("imports %q", imp),
						Hint:    r.Hint,
					})
				}
			}
		}
	}
	return violations
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
go test ./internal/archtest/... -run TestNoImport -v
```

Expected: all `TestNoImport_*` tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/archtest/no_import.go internal/archtest/no_import_test.go
git commit -m "feat(archtest): add NoImport rule"
```

---

## Task 3: `PackageName` rule

**Files:**
- Create: `internal/archtest/package_name.go`
- Create: `internal/archtest/package_name_test.go`

- [ ] **Step 1: Write failing tests in `package_name_test.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package archtest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/internal/archtest"
)

func TestPackageName_nameMatchesDir_passes(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "mypkg")
	_ = os.Mkdir(sub, 0755)
	writeFile(t, sub, "a.go", "package mypkg\n")

	rule := archtest.PackageName{MaxDepth: 1, NameMatchesDir: true, Hint: "fix pkg name"}
	if v := rule.Check(archtest.Package{Dir: dir}); len(v) != 0 {
		t.Errorf("matching name should pass, got %d violations", len(v))
	}
}

func TestPackageName_nameDoesNotMatchDir_fails(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "mydir")
	_ = os.Mkdir(sub, 0755)
	writeFile(t, sub, "a.go", "package wrongname\n")

	rule := archtest.PackageName{MaxDepth: 1, NameMatchesDir: true, Hint: "match name to dir"}
	violations := rule.Check(archtest.Package{Dir: dir})
	if len(violations) != 1 {
		t.Fatalf("want 1 violation, got %d", len(violations))
	}
}

func TestPackageName_excessiveDepth_fails(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "lvl1", "lvl2")
	_ = os.MkdirAll(nested, 0755)
	writeFile(t, nested, "a.go", "package lvl2\n")

	rule := archtest.PackageName{MaxDepth: 1, NameMatchesDir: true, Hint: "keep flat"}
	violations := rule.Check(archtest.Package{Dir: dir})
	if len(violations) == 0 {
		t.Error("depth 2 should violate MaxDepth=1")
	}
}

func TestPackageName_exactlyMaxDepth_passes(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "pkg")
	_ = os.Mkdir(sub, 0755)
	writeFile(t, sub, "a.go", "package pkg\n")

	rule := archtest.PackageName{MaxDepth: 1, NameMatchesDir: true}
	if v := rule.Check(archtest.Package{Dir: dir}); len(v) != 0 {
		t.Errorf("depth 1 == MaxDepth should pass")
	}
}
```

- [ ] **Step 2: Run tests — expect compile failure**

```bash
go test ./internal/archtest/... -run TestPackageName 2>&1 | head -10
```

Expected: `archtest.PackageName undefined`.

- [ ] **Step 3: Create `internal/archtest/package_name.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package archtest

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// PackageName checks that subdirectories of pkg.Dir:
//  1. Do not exceed MaxDepth levels of nesting.
//  2. When NameMatchesDir is true, the declared package name equals the directory basename.
type PackageName struct {
	MaxDepth       int
	NameMatchesDir bool
	Hint           string
}

func (r PackageName) Name() string { return "PackageName" }

func (r PackageName) Check(pkg Package) []Violation {
	var violations []Violation
	_ = filepath.WalkDir(pkg.Dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == pkg.Dir {
			return nil
		}
		rel, _ := filepath.Rel(pkg.Dir, path)
		depth := len(strings.Split(rel, string(filepath.Separator)))

		if depth > r.MaxDepth {
			violations = append(violations, Violation{
				File:    rel,
				Message: Sprintf("directory depth %d exceeds MaxDepth %d", depth, r.MaxDepth),
				Hint:    r.Hint,
				})
				return filepath.SkipDir
			}

		if r.NameMatchesDir {
			dirName := d.Name()
			if pkgName, ok := readPackageName(path); ok && pkgName != dirName {
				violations = append(violations, Violation{
					File:    rel,
					Message: Sprintf("package %q does not match directory name %q", pkgName, dirName),
					Hint:    r.Hint,
				})
			}
		}
		return nil
	})
	return violations
}

// readPackageName returns the declared package name from the first non-test
// .go file found in dir, or ("", false) if none exists.
func readPackageName(dir string) (string, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasSuffix(name, "_test.go") || !strings.HasSuffix(name, ".go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.PackageClauseOnly)
		if err != nil {
			continue
		}
		return f.Name.Name, true
	}
	return "", false
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
go test ./internal/archtest/... -run TestPackageName -v
```

Expected: all `TestPackageName_*` tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/archtest/package_name.go internal/archtest/package_name_test.go
git commit -m "feat(archtest): add PackageName rule"
```

---

## Task 4: `Registry.Lookup()` + `CodecComplete` rule

**Files:**
- Modify: `mdl/model/registry.go`
- Create: `internal/archtest/codec_complete.go`
- Create: `internal/archtest/codec_complete_test.go`

- [ ] **Step 1: Add `Lookup` to `mdl/model/registry.go`**

After the `RegisterGenType` method, add:

```go
// Lookup returns the Codec registered for the given gen TypeName.
// Used by archtest.CodecComplete to verify codec completeness.
func (r *DefaultRegistry) Lookup(genTypeName string) (Codec, bool) {
	c, ok := r.byGenType[genTypeName]
	return c, ok
}
```

- [ ] **Step 2: Verify registry compiles**

```bash
go build ./mdl/model/...
```

Expected: no errors.

- [ ] **Step 3: Write failing tests in `codec_complete_test.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package archtest_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/archtest"
	"github.com/mendixlabs/mxcli/mdl/model"
)

func buildTestRegistry(liftFn func(any) (model.Persistable, error), hydrateFn func(any) (model.Document, []model.Warning, error)) *model.DefaultRegistry {
	r := model.NewDefaultRegistry()
	r.RegisterGenType("TestType$Foo", model.Codec{
		LiftFn:    liftFn,
		HydrateFn: hydrateFn,
	})
	return r
}

func TestCodecComplete_allPresent_passes(t *testing.T) {
	r := buildTestRegistry(
		func(any) (model.Persistable, error) { return nil, nil },
		func(any) (model.Document, []model.Warning, error) { return nil, nil, nil },
	)
	rule := archtest.CodecComplete{
		BuildRegistry: func() *model.DefaultRegistry { return r },
		Required:      []string{"TestType$Foo"},
	}
	if v := rule.Check(archtest.Package{}); len(v) != 0 {
		t.Errorf("complete codec should have no violations")
	}
}

func TestCodecComplete_nilLiftFn_fails(t *testing.T) {
	r := buildTestRegistry(
		nil,
		func(any) (model.Document, []model.Warning, error) { return nil, nil, nil },
	)
	rule := archtest.CodecComplete{
		BuildRegistry: func() *model.DefaultRegistry { return r },
		Required:      []string{"TestType$Foo"},
		Hint:          "add LiftFn",
	}
	violations := rule.Check(archtest.Package{})
	if len(violations) != 1 {
		t.Fatalf("want 1 violation for nil LiftFn, got %d", len(violations))
	}
	if violations[0].Hint != "add LiftFn" {
		t.Errorf("hint not propagated")
	}
}

func TestCodecComplete_nilHydrateFn_fails(t *testing.T) {
	r := buildTestRegistry(
		func(any) (model.Persistable, error) { return nil, nil },
		nil,
	)
	rule := archtest.CodecComplete{
		BuildRegistry: func() *model.DefaultRegistry { return r },
		Required:      []string{"TestType$Foo"},
	}
	if v := rule.Check(archtest.Package{}); len(v) != 1 {
		t.Fatalf("want 1 violation for nil HydrateFn, got %d", len(v))
	}
}

func TestCodecComplete_missingRequiredType_fails(t *testing.T) {
	r := model.NewDefaultRegistry() // empty
	rule := archtest.CodecComplete{
		BuildRegistry: func() *model.DefaultRegistry { return r },
		Required:      []string{"Missing$Type"},
		Hint:          "register missing type",
	}
	violations := rule.Check(archtest.Package{})
	if len(violations) != 1 {
		t.Fatalf("want 1 violation for unregistered type, got %d", len(violations))
	}
}
```

- [ ] **Step 4: Run tests — expect compile failure**

```bash
go test ./internal/archtest/... -run TestCodecComplete 2>&1 | head -10
```

Expected: `archtest.CodecComplete undefined`.

- [ ] **Step 5: Create `internal/archtest/codec_complete.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package archtest

import "github.com/mendixlabs/mxcli/mdl/model"

// CodecComplete verifies that every Required gen TypeName is registered in
// the registry returned by BuildRegistry, and that both LiftFn and HydrateFn
// are non-nil. The Package argument is ignored — this rule inspects runtime
// registry state, not source files.
type CodecComplete struct {
	BuildRegistry func() *model.DefaultRegistry
	Required      []string
	Hint          string
}

func (r CodecComplete) Name() string { return "CodecComplete" }

func (r CodecComplete) Check(_ Package) []Violation {
	reg := r.BuildRegistry()
	var violations []Violation
	for _, name := range r.Required {
		codec, ok := reg.Lookup(name)
		if !ok {
			violations = append(violations, Violation{
				File:    "codec registration",
				Message: Sprintf("gen type %q not registered", name),
				Hint:    r.Hint,
			})
			continue
		}
		if codec.LiftFn == nil {
			violations = append(violations, Violation{
				File:    "codec registration",
				Message: Sprintf("%q: LiftFn is nil", name),
				Hint:    r.Hint,
			})
		}
		if codec.HydrateFn == nil {
			violations = append(violations, Violation{
				File:    "codec registration",
				Message: Sprintf("%q: HydrateFn is nil", name),
				Hint:    r.Hint,
			})
		}
	}
	return violations
}
```

- [ ] **Step 6: Run tests — expect pass**

```bash
go test ./internal/archtest/... -v
```

Expected: all `TestCodecComplete_*` tests pass. Full archtest suite green.

- [ ] **Step 7: Commit**

```bash
git add internal/archtest/codec_complete.go internal/archtest/codec_complete_test.go \
        mdl/model/registry.go
git commit -m "feat(archtest): add CodecComplete rule; add Registry.Lookup()"
```

---

## Task 5: Guard 1 — root package import direction (starts RED)

**Files:**
- Create: `mdl/model/import_guard_test.go`

- [ ] **Step 1: Create `mdl/model/import_guard_test.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package model_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/archtest"
)

// TestImportDirection verifies that the mdl/model root package — the stable
// canonical center — does not import volatile layers (AST, gen types, backend).
//
// RED state: context.go imports "github.com/mendixlabs/mxcli/mdl/backend".
// GREEN when: PersistContext.Backend is replaced by a Writer interface defined
// in this package (see Hint below).
func TestImportDirection(t *testing.T) {
	archtest.Check(t, ".",
		archtest.NoImport{
			Forbidden: []string{
				"github.com/mendixlabs/mxcli/mdl/ast",
				"github.com/mendixlabs/mxcli/modelsdk/gen/",
				"github.com/mendixlabs/mxcli/mdl/backend",
			},
			Hint: `mdl/model/ (future: mdl/canonical/) is the stable center.
It must not depend on any volatile layer (AST parser, gen types, backend).
Fix for context.go:
  1. Define a Writer interface in this package:
       type Writer interface {
           CreateDoc(domainModelID model.ID, doc Persistable) error
           UpdateDoc(existingID model.ID, domainModelID model.ID, doc Persistable) error
       }
  2. Change PersistContext.Backend field type from backend.DomainModelBackend to Writer.
  3. backend/mpr's MprBackend implements Writer; executor passes it in.
  4. Remove the "mdl/backend" import from context.go.`,
		},
	)
}
```

- [ ] **Step 2: Run guard — confirm RED**

```bash
go test ./mdl/model/ -run TestImportDirection -v
```

Expected output:
```
--- FAIL: TestImportDirection
    context.go:9: imports "github.com/mendixlabs/mxcli/mdl/backend" [NoImport]
      HINT: mdl/model/ (future: mdl/canonical/) is the stable center.
      ...
```

- [ ] **Step 3: Commit**

```bash
git add mdl/model/import_guard_test.go
git commit -m "test(model): add Guard 1 — import direction (RED: context.go imports backend)"
```

---

## Task 6: Guard 2 — interface compliance (GREEN baseline)

**Files:**
- Create: `mdl/model/entity/comply_test.go`

- [ ] **Step 1: Create `mdl/model/entity/comply_test.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package entity_test

import (
	"github.com/mendixlabs/mxcli/mdl/model"
	"github.com/mendixlabs/mxcli/mdl/model/entity"
)

// Compile-time interface compliance assertions.
// These have no runtime cost — they are checked at compile time by go test.
//
// GREEN baseline: EntityModel currently satisfies both interfaces.
// Turns RED if someone removes ToMDL() or Persist() from EntityModel, or if
// the Document/Persistable interface contract changes.
//
// When a new domain is added (e.g. mdl/model/association/), create
// mdl/model/association/comply_test.go with the equivalent two lines.
// No other file needs modification.
var _ model.Document    = (*entity.EntityModel)(nil)
var _ model.Persistable = (*entity.EntityModel)(nil)
```

- [ ] **Step 2: Verify GREEN**

```bash
go test ./mdl/model/entity/ -run "^$" -v 2>&1 | head -5
```

Expected: package compiles without error (the `var _` declarations are compile-time only).

- [ ] **Step 3: Commit**

```bash
git add mdl/model/entity/comply_test.go
git commit -m "test(model/entity): add Guard 2 — interface compliance (GREEN baseline)"
```

---

## Task 7: Guard 3 — codec completeness (GREEN baseline)

**Files:**
- Create: `mdl/model/codec_guard_test.go`

- [ ] **Step 1: Create `mdl/model/codec_guard_test.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package model_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/archtest"
	"github.com/mendixlabs/mxcli/mdl/model"
	"github.com/mendixlabs/mxcli/mdl/model/entity"
)

// TestCodecComplete verifies that all registered gen TypeNames have complete
// codecs (non-nil LiftFn and HydrateFn).
//
// GREEN baseline: entity codecs are complete (both fns non-nil).
// Turns RED if: a new domain registers a codec with a nil function, or
// a Required TypeName is removed from the registry without updating this file.
//
// When adding a new domain (Phase 2+):
//   1. Add: domainpkg.RegisterCodec(r) to BuildRegistry below.
//   2. Add: "DomainModels$NewType" to Required.
//   Both must be updated together — a Required entry with no RegisterCodec
//   call will immediately fail this test.
func TestCodecComplete(t *testing.T) {
	archtest.Check(t, ".",
		archtest.CodecComplete{
			BuildRegistry: func() *model.DefaultRegistry {
				r := model.NewDefaultRegistry()
				entity.RegisterCodec(r)
				// Phase 2: association.RegisterCodec(r)
				// Phase 3: microflow.RegisterCodec(r)
				return r
			},
			Required: []string{
				"DomainModels$Entity",
				"DomainModels$EntityImpl",
				// Phase 2: "DomainModels$Association"
			},
			Hint: `Every Required gen TypeName must be registered with non-nil LiftFn and HydrateFn.
Fix:
  1. Confirm domain/codec.go calls r.Register(...) or r.RegisterGenType(...) for all listed TypeNames.
  2. LiftFn  — func(stmt any) (Persistable, error): converts AST stmt to canonical Model.
  3. HydrateFn — func(el any) (Document, []Warning, error): converts gen element to canonical Model.
     Note: current HydrateFn passes "" as moduleName — a known gap tracked in the spec.
     A future plan will upgrade HydrateFn to accept HydrateCtx{ModuleName string}.
  4. When adding a new domain: add RegisterCodec call to BuildRegistry AND
     add TypeName(s) to Required. Both must be updated together.`,
		},
	)
}
```

- [ ] **Step 2: Run guard — confirm GREEN**

```bash
go test ./mdl/model/ -run TestCodecComplete -v
```

Expected: `PASS` — both entity TypeNames are registered with non-nil fns.

- [ ] **Step 3: Commit**

```bash
git add mdl/model/codec_guard_test.go
git commit -m "test(model): add Guard 3 — codec completeness (GREEN baseline)"
```

---

## Task 8: Guard 4 — executor boundary (starts RED)

**Files:**
- Create: `mdl/executor/boundary_guard_test.go`

- [ ] **Step 1: Create `mdl/executor/boundary_guard_test.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/archtest"
)

// TestExecutorBoundary verifies that cmd_*.go files do not import
// mdl/model/entity (or any future mdl/model/{domain} subpackage) directly.
// All canonical model operations must go through ctx.ModelCodecs dispatch.
//
// RED state: cmd_entities_gen.go and cmd_diff_mdl.go import mdl/model/entity
// and call entity.HydrateWithModule() directly, bypassing the codec registry.
//
// GREEN when: describe and create paths route through ctx.ModelCodecs.HydrateFrom()
// and ctx.ModelCodecs.LiftFrom() respectively. executor.go (RegisterCodec site)
// remains in the Allowlist.
func TestExecutorBoundary(t *testing.T) {
	archtest.Check(t, ".",
		archtest.NoImport{
			Forbidden: []string{
				"github.com/mendixlabs/mxcli/mdl/model/entity",
				// Phase 2: "github.com/mendixlabs/mxcli/mdl/model/association"
				// Phase 3: "github.com/mendixlabs/mxcli/mdl/model/microflow"
			},
			Allowlist: map[string]bool{
				// executor.go is the sole file permitted to import domain subpackages.
				// It uses them only to call RegisterCodec(). All other files must
				// access canonical models through ctx.ModelCodecs.
				"executor.go": true,
			},
			Hint: `cmd_*.go files must not import mdl/model/{domain}/ subpackages directly.
Fix for cmd_entities_gen.go (describe path):
  Replace:
    m, warns, err := entityModel.HydrateWithModule(modName, entity)
  With:
    doc, warns, err := ctx.ModelCodecs.HydrateFrom(entity)
    m := doc.(*entitypkg.EntityModel)  // type-assert only if domain fields needed
    _ = modName                        // modName must be overlaid on m.Name.Module

Fix for cmd_diff_mdl.go:
  Replace direct entityModel.Lift(s) / entityModel.HydrateWithModule calls
  with ctx.ModelCodecs.LiftFrom(s) and ctx.ModelCodecs.HydrateFrom(el).

The codec registry (executor.go) is the ONLY file that may import domain subpackages.`,
		},
	)
}
```

- [ ] **Step 2: Run guard — confirm RED**

```bash
go test ./mdl/executor/ -run TestExecutorBoundary -v
```

Expected output (two violations):
```
--- FAIL: TestExecutorBoundary
    cmd_entities_gen.go:21: imports "github.com/mendixlabs/mxcli/mdl/model/entity" [NoImport]
      HINT: cmd_*.go files must not import ...
    cmd_diff_mdl.go:11: imports "github.com/mendixlabs/mxcli/mdl/model/entity" [NoImport]
      HINT: cmd_*.go files must not import ...
```

- [ ] **Step 3: Commit**

```bash
git add mdl/executor/boundary_guard_test.go
git commit -m "test(executor): add Guard 4 — executor boundary (RED: cmd_entities_gen + cmd_diff_mdl bypass registry)"
```

---

## Task 9: Guard 5 — package structure (GREEN baseline)

**Files:**
- Create: `mdl/model/naming_guard_test.go`

- [ ] **Step 1: Create `mdl/model/naming_guard_test.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package model_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/archtest"
)

// TestPackageStructure verifies that mdl/model/ (future: mdl/canonical/) uses
// a flat one-level subpackage structure where each subdirectory name matches
// its declared package name.
//
// GREEN baseline: entity/ and layout/ both satisfy the constraints.
// Turns RED if:
//   - A new domain creates nested subdirectories (e.g. entity/conversion/)
//   - A subdirectory declares a package name that does not match its dirname
//
// Rule: add files within the same directory to split large files; do not
// create sub-subdirectories.
func TestPackageStructure(t *testing.T) {
	archtest.Check(t, ".",
		archtest.PackageName{
			MaxDepth:       1,
			NameMatchesDir: true,
			Hint: `mdl/model/ allows exactly one level of subpackages (entity/, association/, etc.).
Do not create nested subdirectories inside domain packages (e.g. entity/conversion/).
Reason: nesting makes import paths verbose and cross-domain references complex.
To split a large file: add files in the same directory with the same package name.
Example: entity/lift.go, entity/hydrate.go, entity/persist.go — all "package entity".`,
		},
	)
}
```

- [ ] **Step 2: Run guard — confirm GREEN**

```bash
go test ./mdl/model/ -run TestPackageStructure -v
```

Expected: `PASS` — current structure (`entity/`, `layout/`) satisfies MaxDepth=1 and names match.

- [ ] **Step 3: Run full test suite — confirm no regressions**

```bash
go test ./internal/archtest/... ./mdl/model/... ./mdl/executor/... -count=1 2>&1 | grep -E "FAIL|ok" | head -20
```

Expected:
- `internal/archtest`: PASS
- `mdl/model/`: FAIL (Guard 1 RED — expected)
- `mdl/executor/`: FAIL (Guard 4 RED — expected)
- Other packages: no new failures

- [ ] **Step 4: Commit**

```bash
git add mdl/model/naming_guard_test.go
git commit -m "test(model): add Guard 5 — package structure (GREEN baseline)"
```

---

## Self-Review Checklist

**Spec coverage:**

| Spec section | Covered by |
|---|---|
| `internal/archtest` Rule interface | Task 1 |
| `internal/archtest` Violation + Hint | Task 1 |
| `internal/archtest` Package.Files/Walk | Task 1 |
| `NoImport` rule | Task 2 |
| `PackageName` rule | Task 3 |
| `CodecComplete` rule | Task 4 |
| `Registry.Lookup()` | Task 4 |
| Guard 1 import direction | Task 5 |
| Guard 2 interface compliance | Task 6 |
| Guard 3 codec completeness | Task 7 |
| Guard 4 executor boundary | Task 8 |
| Guard 5 package structure | Task 9 |

All spec requirements covered. No gaps.

**Type consistency:** `model.Codec`, `model.DefaultRegistry`, `model.Persistable`, `model.Document` are used consistently across Tasks 4, 7, and the guard files. `archtest.Package`, `archtest.Violation`, `archtest.NoImport`, `archtest.PackageName`, `archtest.CodecComplete` are defined in Task 1-4 and referenced identically in Tasks 5-9.

**Red/green states documented:** Guard 1 (RED), Guard 2 (GREEN), Guard 3 (GREEN), Guard 4 (RED), Guard 5 (GREEN) — all noted explicitly in test comments so future readers understand expected state.

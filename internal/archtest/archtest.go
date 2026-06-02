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

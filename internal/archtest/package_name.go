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

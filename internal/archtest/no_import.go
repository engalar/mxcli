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

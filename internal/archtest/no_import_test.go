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

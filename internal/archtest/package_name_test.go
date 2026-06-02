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

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

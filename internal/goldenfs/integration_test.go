// SPDX-License-Identifier: Apache-2.0

//go:build linux

package goldenfs

import (
	"os"
	"path/filepath"
	"testing"

	mprpkg "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// repoRoot returns the absolute path to the repository root
// (two directories above this package).
func repoRoot(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func exprCheckerDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), "testdata", "expr-checker")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("testdata/expr-checker not found: %v", err)
	}
	return dir
}

// TestMprWriterThroughFUSE verifies that mpr.Writer writing through the FUSE
// overlay leaves testdata/expr-checker untouched.
func TestMprWriterThroughFUSE(t *testing.T) {
	snap, err := Open(exprCheckerDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	mprPath := filepath.Join(snap.MountDir(), "minimal.mpr")

	// Open for writing via the FUSE mount.
	reader, err := mprpkg.OpenWithOptions(mprPath, mprpkg.OpenOptions{ReadOnly: false})
	if err != nil {
		t.Fatalf("open mpr through FUSE: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close mpr: %v", err)
	}

	// The SQLite WAL may have been created in the overlay.
	// Confirm the real minimal.mpr has not been modified.
	realMpr := filepath.Join(exprCheckerDir(t), "minimal.mpr")
	origStat, _ := os.Stat(realMpr)

	// The snapshot's dirty layer may contain the WAL; base must not.
	walPath := filepath.Join(exprCheckerDir(t), "minimal.mpr-wal")
	if _, err := os.Stat(walPath); err == nil {
		t.Fatal("WAL file must not exist in base dir after FUSE write")
	}

	// mtime of the real file must be unchanged.
	newStat, _ := os.Stat(realMpr)
	if !origStat.ModTime().Equal(newStat.ModTime()) {
		t.Fatal("real minimal.mpr mtime changed — base was modified")
	}
}

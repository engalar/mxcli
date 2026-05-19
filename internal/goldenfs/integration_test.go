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
	// Capture baseline BEFORE Open() — otherwise both stats are post-write
	// and the comparison is trivially true.
	realMpr := filepath.Join(exprCheckerDir(t), "minimal.mpr")
	origStat, err := os.Stat(realMpr)
	if err != nil {
		t.Fatalf("stat real mpr: %v", err)
	}

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

	// Neither WAL, SHM, nor rollback journal must appear in the real base dir.
	// minimal.mpr uses DELETE journal mode (not WAL), so the rollback journal
	// is the most likely leak from a write+rollback.
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if _, err := os.Stat(realMpr + suffix); err == nil {
			t.Fatalf("%s file must not exist in base dir after FUSE write", suffix)
		}
	}

	// mtime of the real file must be unchanged.
	newStat, err := os.Stat(realMpr)
	if err != nil {
		t.Fatalf("stat real mpr after: %v", err)
	}
	if !origStat.ModTime().Equal(newStat.ModTime()) {
		t.Fatal("real minimal.mpr mtime changed — base was modified")
	}
}

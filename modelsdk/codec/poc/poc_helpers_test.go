//go:build poc
// SPDX-License-Identifier: Apache-2.0

// Package poc_test contains the Stage 1 proof-of-concept tests for the
// modelsdk-native architecture migration. See
// docs/superpowers/specs/2026-05-13-modelsdk-native-architecture-design.md
// Section 9 for the design assumptions these tests validate.
package poc_test

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// fixturePath is the canonical PoC fixture: a v2 MPR with mprcontents/ folder
// and ~16 Microflows$Microflow units. Path is relative to this test file's
// directory (modelsdk/codec/poc/).
const fixturePath = "../../../testdata/expr-checker/minimal.mpr"

// openTestMPR opens the canonical PoC fixture in read-write mode under a
// per-test temp directory. Returns the writer and the concrete reader so
// tests can call ListUnitsByType / GetRawUnitBytes directly without going
// through the UnitReader interface (which exposes neither).
func openTestMPR(t *testing.T) (*mmpr.Writer, *mmpr.Reader) {
	t.Helper()
	dst := copyFixture(t, fixturePath, t.TempDir())
	w, err := mmpr.NewWriter(dst)
	if err != nil {
		t.Fatalf("NewWriter(%s): %v", dst, err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w, w.ConcreteReader()
}

// openTestMPRForBench is the *testing.B variant of openTestMPR.
func openTestMPRForBench(b *testing.B) (*mmpr.Writer, *mmpr.Reader) {
	b.Helper()
	dst := copyFixtureB(b, fixturePath, b.TempDir())
	w, err := mmpr.NewWriter(dst)
	if err != nil {
		b.Fatalf("NewWriter(%s): %v", dst, err)
	}
	b.Cleanup(func() { _ = w.Close() })
	return w, w.ConcreteReader()
}

// copyFixture copies the .mpr file and its sibling mprcontents/ tree (v2
// format) into dstDir. Returns the path of the copied .mpr file.
func copyFixture(t *testing.T, srcMPR, dstDir string) string {
	t.Helper()
	dst, err := copyMPRTree(srcMPR, dstDir)
	if err != nil {
		t.Fatalf("copy fixture %s -> %s: %v", srcMPR, dstDir, err)
	}
	return dst
}

func copyFixtureB(b *testing.B, srcMPR, dstDir string) string {
	b.Helper()
	dst, err := copyMPRTree(srcMPR, dstDir)
	if err != nil {
		b.Fatalf("copy fixture %s -> %s: %v", srcMPR, dstDir, err)
	}
	return dst
}

// copyMPRTree copies srcMPR into dstDir, plus any sibling "mprcontents"
// directory (v2 format). Returns the destination .mpr path.
func copyMPRTree(srcMPR, dstDir string) (string, error) {
	dstMPR := filepath.Join(dstDir, filepath.Base(srcMPR))
	if err := copyOneFile(srcMPR, dstMPR); err != nil {
		return "", err
	}

	srcContents := filepath.Join(filepath.Dir(srcMPR), "mprcontents")
	if info, err := os.Stat(srcContents); err == nil && info.IsDir() {
		dstContents := filepath.Join(dstDir, "mprcontents")
		if err := copyDir(srcContents, dstContents); err != nil {
			return "", err
		}
	}
	return dstMPR, nil
}

func copyOneFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyOneFile(p, target)
	})
}

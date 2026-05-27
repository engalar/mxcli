// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"github.com/mendixlabs/mxcli/model"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// fixtureOpenSem limits concurrent fixture open operations at GOMAXPROCS
// to prevent I/O storms when many tests run in parallel.
var fixtureOpenSem = make(chan struct{}, runtime.GOMAXPROCS(0))

// typedID is a 1-line cast helper so test bodies stay readable.
func typedID(s string) model.ID { return model.ID(s) }

// fixturePath is the canonical Stage 2 fixture: a v2 MPR with
// mprcontents/ folder and 16 Microflows$Microflow units. Path is
// relative to test files in mdl/backend/mpr/repos/.
const fixturePath = "../../../../testdata/expr-checker/minimal.mpr"

// openTestWriter sets up an isolated writable copy of the fixture for
// a test and opens it as a *mmpr.Writer.
//
// Isolation strategy for mprcontents/ (370 immutable .mxunit files):
//   - Hard-link instead of copy: O(1) per file, no data written
//   - The Writer never modifies existing .mxunit files; it only creates
//     new ones with fresh UUIDs, so hard links are safe
//
// The .mpr SQLite file (68 KB) must be a real copy because the Writer
// commits WAL changes back into it.
func openTestWriter(t *testing.T) *mmpr.Writer {
	t.Helper()
	t.Parallel()
	fixtureOpenSem <- struct{}{}
	defer func() { <-fixtureOpenSem }()
	dst := copyFixture(t, fixturePath, t.TempDir())
	w, err := mmpr.NewWriter(dst)
	if err != nil {
		t.Fatalf("NewWriter(%s): %v", dst, err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w
}

func copyFixture(t *testing.T, srcMPR, dstDir string) string {
	t.Helper()
	dst, err := copyMPRTree(srcMPR, dstDir)
	if err != nil {
		t.Fatalf("copy fixture %s -> %s: %v", srcMPR, dstDir, err)
	}
	return dst
}

// copyMPRTree copies the .mpr SQLite file and hard-links the mprcontents/
// directory tree into dstDir. Hard links are used for mprcontents/ because
// those files are immutable once written — the Writer adds new files rather
// than modifying existing ones.
func copyMPRTree(srcMPR, dstDir string) (string, error) {
	dstMPR := filepath.Join(dstDir, filepath.Base(srcMPR))
	if err := copyOneFile(srcMPR, dstMPR); err != nil {
		return "", err
	}
	srcContents := filepath.Join(filepath.Dir(srcMPR), "mprcontents")
	if info, err := os.Stat(srcContents); err == nil && info.IsDir() {
		dstContents := filepath.Join(dstDir, "mprcontents")
		if err := hardLinkDir(srcContents, dstContents); err != nil {
			return "", err
		}
	}
	return dstMPR, nil
}

// hardLinkDir mirrors src into dst using hard links when possible.
// Falls back to copyOneFile on cross-device setups (EXDEV). Hard links
// are safe for mprcontents/ because the Writer only creates new files
// with fresh UUIDs — it never modifies existing .mxunit files in place.
func hardLinkDir(src, dst string) error {
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
		if linkErr := os.Link(p, target); linkErr != nil {
			if errors.Is(linkErr, syscall.EXDEV) {
				return copyOneFile(p, target)
			}
			return linkErr
		}
		return nil
	})
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

// copyDir is retained for tests that explicitly need a real on-disk copy.
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

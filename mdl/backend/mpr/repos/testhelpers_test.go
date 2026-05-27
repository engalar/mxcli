// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/model"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// typedID is a 1-line cast helper so test bodies stay readable.
func typedID(s string) model.ID { return model.ID(s) }

// fixturePath is the canonical Stage 2 fixture: a v2 MPR with
// mprcontents/ folder and 16 Microflows$Microflow units. Path is
// relative to test files in mdl/backend/mpr/repos/.
const fixturePath = "../../../../testdata/expr-checker/minimal.mpr"

// openTestWriter copies the canonical Stage 2 fixture into a per-test
// temp directory and opens it as a *mmpr.Writer. It calls t.Parallel()
// so the 112 independent repo tests run concurrently.
func openTestWriter(t *testing.T) *mmpr.Writer {
	t.Helper()
	t.Parallel()
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

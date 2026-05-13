// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const fixturePath = "../../../testdata/expr-checker/minimal.mpr"

func openTestWriter(t *testing.T) *mmpr.Writer {
	t.Helper()
	dst := copyFixture(t, fixturePath, t.TempDir())
	w, err := mmpr.NewWriter(dst)
	if err != nil {
		t.Fatalf("NewWriter(%s): %v", dst, err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w
}

func TestNewExecutorContext_AllFieldsWired(t *testing.T) {
	w := openTestWriter(t)
	ctx := NewExecutorContext(w)
	if ctx == nil {
		t.Fatal("NewExecutorContext returned nil")
	}
	if ctx.Microflows == nil {
		t.Error("ctx.Microflows = nil")
	}
	if ctx.Pages == nil {
		t.Error("ctx.Pages = nil")
	}
	if ctx.IDs == nil {
		t.Error("ctx.IDs = nil")
	}
	if ctx.Tx == nil {
		t.Error("ctx.Tx = nil")
	}
	if ctx.Names == nil {
		t.Error("ctx.Names = nil")
	}
	if ctx.Cache == nil {
		t.Error("ctx.Cache = nil")
	}
}

func TestNewExecutorContext_RepoSmokeMicroflows(t *testing.T) {
	w := openTestWriter(t)
	ctx := NewExecutorContext(w)
	mfs, err := ctx.Microflows.ListAll()
	if err != nil {
		t.Fatalf("ctx.Microflows.ListAll: %v", err)
	}
	if len(mfs) != 16 {
		t.Errorf("ctx.Microflows.ListAll: got %d, want 16 (fixture)", len(mfs))
	}
}

func TestNewExecutorContext_RepoSmokePages(t *testing.T) {
	w := openTestWriter(t)
	ctx := NewExecutorContext(w)
	pages, err := ctx.Pages.ListAll()
	if err != nil {
		t.Fatalf("ctx.Pages.ListAll: %v", err)
	}
	if len(pages) != 16 {
		t.Errorf("ctx.Pages.ListAll: got %d, want 16 (fixture Forms$Page)", len(pages))
	}
}

func TestNewExecutorContext_TxBeginRollback(t *testing.T) {
	w := openTestWriter(t)
	ctx := NewExecutorContext(w)
	uow, err := ctx.Tx.Begin()
	if err != nil {
		t.Fatalf("ctx.Tx.Begin: %v", err)
	}
	if err := uow.Rollback(); err != nil {
		t.Fatalf("Rollback: %v", err)
	}
}

func TestNewExecutorContext_IDsProduceFreshUUIDs(t *testing.T) {
	w := openTestWriter(t)
	ctx := NewExecutorContext(w)
	a, b := ctx.IDs.NewID(), ctx.IDs.NewID()
	if a == b {
		t.Errorf("two consecutive NewIDs identical: %s", a)
	}
	if len(string(a)) != 36 {
		t.Errorf("NewID len = %d, want 36", len(string(a)))
	}
}

// --- helpers (mirror mdl/backend/mpr/repos/testhelpers_test.go) ---

func copyFixture(t *testing.T, srcMPR, dstDir string) string {
	t.Helper()
	dst, err := copyMPRTree(srcMPR, dstDir)
	if err != nil {
		t.Fatalf("copy fixture %s -> %s: %v", srcMPR, dstDir, err)
	}
	return dst
}

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

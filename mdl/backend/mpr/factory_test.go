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

	// Stage 2.7 — Cascade always wired by NewExecutorContext; References
	// requires the with-references constructor.
	if ctx.Cascade == nil {
		t.Error("ctx.Cascade = nil (Stage 2.7 always-wired service)")
	}
	if ctx.References != nil {
		t.Error("ctx.References = non-nil from NewExecutorContext (use NewExecutorContextWithReferences for that)")
	}

	// Stage 2.6 — 15 newly wired domains.
	if ctx.Nanoflows == nil {
		t.Error("ctx.Nanoflows = nil")
	}
	if ctx.Layouts == nil {
		t.Error("ctx.Layouts = nil")
	}
	if ctx.Snippets == nil {
		t.Error("ctx.Snippets = nil")
	}
	if ctx.DomainModels == nil {
		t.Error("ctx.DomainModels = nil")
	}
	if ctx.Modules == nil {
		t.Error("ctx.Modules = nil")
	}
	if ctx.Enumerations == nil {
		t.Error("ctx.Enumerations = nil")
	}
	if ctx.Constants == nil {
		t.Error("ctx.Constants = nil")
	}
	if ctx.Workflows == nil {
		t.Error("ctx.Workflows = nil")
	}
	if ctx.Services == nil {
		t.Error("ctx.Services = nil")
	}
	if ctx.Mappings == nil {
		t.Error("ctx.Mappings = nil")
	}
	if ctx.ProjectSet == nil {
		t.Error("ctx.ProjectSet = nil")
	}
	if ctx.ModuleSet == nil {
		t.Error("ctx.ModuleSet = nil")
	}
	if ctx.Security == nil {
		t.Error("ctx.Security = nil")
	}
	if ctx.Folders == nil {
		t.Error("ctx.Folders = nil")
	}
	if ctx.Images == nil {
		t.Error("ctx.Images = nil")
	}
	if ctx.Agents == nil {
		t.Error("ctx.Agents = nil")
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

// Stage 2.6 fixture-backed smoke tests: per-domain List against the
// canonical fixture must return the expected count. Numbers come
// from the same probes the per-domain implementations were written
// against (see commit messages).

func TestNewExecutorContext_RepoSmokeNanoflows(t *testing.T) {
	w := openTestWriter(t)
	ctx := NewExecutorContext(w)
	got, err := ctx.Nanoflows.List("")
	if err != nil {
		t.Fatalf("Nanoflows.List: %v", err)
	}
	if len(got) != 13 {
		t.Errorf("Nanoflows.List: got %d, want 13", len(got))
	}
}

func TestNewExecutorContext_RepoSmokeLayouts(t *testing.T) {
	w := openTestWriter(t)
	ctx := NewExecutorContext(w)
	got, err := ctx.Layouts.List("")
	if err != nil {
		t.Fatalf("Layouts.List: %v", err)
	}
	if len(got) != 22 {
		t.Errorf("Layouts.List: got %d, want 22", len(got))
	}
}

func TestNewExecutorContext_RepoSmokeSnippets(t *testing.T) {
	w := openTestWriter(t)
	ctx := NewExecutorContext(w)
	got, err := ctx.Snippets.List("")
	if err != nil {
		t.Fatalf("Snippets.List: %v", err)
	}
	if len(got) != 4 {
		t.Errorf("Snippets.List: got %d, want 4", len(got))
	}
}

func TestNewExecutorContext_RepoSmokeDomainModels(t *testing.T) {
	w := openTestWriter(t)
	ctx := NewExecutorContext(w)
	got, err := ctx.DomainModels.List("")
	if err != nil {
		t.Fatalf("DomainModels.List: %v", err)
	}
	if len(got) != 8 {
		t.Errorf("DomainModels.List: got %d, want 8", len(got))
	}
}

func TestNewExecutorContext_RepoSmokeModules(t *testing.T) {
	w := openTestWriter(t)
	ctx := NewExecutorContext(w)
	got, err := ctx.Modules.ListAll()
	if err != nil {
		t.Fatalf("Modules.ListAll: %v", err)
	}
	if len(got) != 8 {
		t.Errorf("Modules.ListAll: got %d, want 8", len(got))
	}
}

func TestNewExecutorContext_RepoSmokeEnumerations(t *testing.T) {
	w := openTestWriter(t)
	ctx := NewExecutorContext(w)
	got, err := ctx.Enumerations.List("")
	if err != nil {
		t.Fatalf("Enumerations.List: %v", err)
	}
	if len(got) != 7 {
		t.Errorf("Enumerations.List: got %d, want 7", len(got))
	}
}

func TestNewExecutorContext_RepoSmokeConstants(t *testing.T) {
	w := openTestWriter(t)
	ctx := NewExecutorContext(w)
	got, err := ctx.Constants.List("")
	if err != nil {
		t.Fatalf("Constants.List: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("Constants.List: got %d, want 2", len(got))
	}
}

func TestNewExecutorContext_RepoSmokeProjectSet(t *testing.T) {
	w := openTestWriter(t)
	ctx := NewExecutorContext(w)
	ps, err := ctx.ProjectSet.Get()
	if err != nil {
		t.Fatalf("ProjectSet.Get: %v", err)
	}
	if ps == nil {
		t.Error("ProjectSet.Get returned nil singleton")
	}
}

func TestNewExecutorContext_RepoSmokeSecurity(t *testing.T) {
	w := openTestWriter(t)
	ctx := NewExecutorContext(w)
	ps, err := ctx.Security.Get()
	if err != nil {
		t.Fatalf("Security.Get: %v", err)
	}
	if ps == nil {
		t.Error("Security.Get returned nil singleton")
	}
}

func TestNewExecutorContext_RepoSmokeFolders(t *testing.T) {
	w := openTestWriter(t)
	ctx := NewExecutorContext(w)
	got, err := ctx.Folders.List("")
	if err != nil {
		t.Fatalf("Folders.List: %v", err)
	}
	if len(got) != 80 {
		t.Errorf("Folders.List: got %d, want 80", len(got))
	}
}

func TestNewExecutorContext_RepoSmokeImages(t *testing.T) {
	w := openTestWriter(t)
	ctx := NewExecutorContext(w)
	got, err := ctx.Images.ListCollections("")
	if err != nil {
		t.Fatalf("Images.ListCollections: %v", err)
	}
	if len(got) != 7 {
		t.Errorf("Images.ListCollections: got %d, want 7", len(got))
	}
}

// Stage 2.7: NewExecutorContextWithReferences wires both Cascade
// (always wired) and References (requires sdk/mpr Writer).
func TestNewExecutorContextWithReferences_BothServicesWired(t *testing.T) {
	dst := copyFixture(t, fixturePath, t.TempDir())
	mw, err := mmpr.NewWriter(dst)
	if err != nil {
		t.Fatalf("mmpr.NewWriter: %v", err)
	}
	t.Cleanup(func() { _ = mw.Close() })
	scanner, err := mmpr.Open(dst)
	if err != nil {
		t.Fatalf("mmpr.Open: %v", err)
	}
	t.Cleanup(func() { _ = scanner.Close() })

	ctx := NewExecutorContextWithReferences(mw, scanner)
	if ctx == nil {
		t.Fatal("NewExecutorContextWithReferences returned nil")
	}
	if ctx.Cascade == nil {
		t.Error("ctx.Cascade = nil")
	}
	if ctx.References == nil {
		t.Error("ctx.References = nil")
	}
	// Smoke: ScanRename with non-matching name should be a no-op (zero hits).
	hits, err := ctx.References.ScanRename("Module.NoSuchEntity_xyz", "Module.NewName_xyz")
	if err != nil {
		t.Fatalf("ScanRename: %v", err)
	}
	if len(hits) != 0 {
		t.Errorf("ScanRename hits = %d, want 0 for non-matching name", len(hits))
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

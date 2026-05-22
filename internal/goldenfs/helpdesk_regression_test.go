// SPDX-License-Identifier: Apache-2.0

//go:build linux && integration

package goldenfs

import (
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// updateGolden: go test ... -update-golden -run TestHelpdeskGolden_Update
var updateGolden = flag.Bool("update-golden", false,
	"overwrite testdata/helpdesk-golden/ with the current MDL execution result")

// helpdeskBlankDir returns the directory containing the blank base MPR (A).
// Uses testdata/expr-checker which is already committed and v2-format.
func helpdeskBlankDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), "testdata", "expr-checker")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("testdata/expr-checker not found: %v", err)
	}
	return dir
}

// helpdeskBlankMPR returns the path to the blank base MPR file.
func helpdeskBlankMPR(t *testing.T) string {
	t.Helper()
	p := filepath.Join(helpdeskBlankDir(t), "minimal.mpr")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("testdata/expr-checker/minimal.mpr not found: %v", err)
	}
	return p
}

// helpdeskGoldenDir returns the path to the committed B1 golden directory.
func helpdeskGoldenDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "testdata", "helpdesk-golden")
}

// helpdeskGoldenMPR returns the MPR path inside the golden directory.
func helpdeskGoldenMPR(t *testing.T) string {
	t.Helper()
	return filepath.Join(helpdeskGoldenDir(t), "minimal.mpr")
}

// helpdeskMDLScript reads mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
// and returns its content as a string.
func helpdeskMDLScript(t *testing.T) string {
	t.Helper()
	p := filepath.Join(repoRoot(t), "mdl-examples", "use-cases", "helpdesk", "helpdesk-app.mdl")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read helpdesk-app.mdl: %v", err)
	}
	return string(data)
}

// copyDir copies src directory tree to dst, creating dst if needed.
// Overwrites existing files in dst.
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, 0o644)
	})
}

// TestHelpdeskGolden_Update 生成或更新 testdata/helpdesk-golden/。
// 只在 -update-golden flag 存在时有效；否则直接 Skip。
// 运行方式：
//
//	go test ./internal/goldenfs/ -tags linux,integration \
//	       -run TestHelpdeskGolden_Update -update-golden -v
func TestHelpdeskGolden_Update(t *testing.T) {
	if !*updateGolden {
		t.Skip("pass -update-golden to regenerate testdata/helpdesk-golden/")
	}

	blankDir := helpdeskBlankDir(t)
	goldenDir := helpdeskGoldenDir(t)
	script := helpdeskMDLScript(t)

	// Open FUSE overlay on top of blank A.
	snap, err := Open(blankDir)
	if err != nil {
		t.Fatalf("goldenfs.Open: %v", err)
	}
	defer func() {
		if err := snap.Close(); err != nil {
			t.Logf("snap.Close: %v", err)
		}
	}()

	mountMPR := filepath.Join(snap.MountDir(), "minimal.mpr")

	// Execute helpdesk-app.mdl — produces B2 in the FUSE overlay.
	runMDL(t, mountMPR, script)

	// Copy entire FUSE mount (A + dirty layer = B2) to testdata/helpdesk-golden/.
	// NOTE: do NOT call snap.Commit() — that would write back to blankDir (A).
	if err := os.RemoveAll(goldenDir); err != nil {
		t.Fatalf("remove old golden: %v", err)
	}
	if err := copyDir(snap.MountDir(), goldenDir); err != nil {
		t.Fatalf("copy to golden: %v", err)
	}

	t.Logf("Golden updated: %s", goldenDir)
	t.Logf("Next step: git add testdata/helpdesk-golden/ && git commit")
}

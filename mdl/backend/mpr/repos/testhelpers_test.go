// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mendixlabs/mxcli/internal/goldenfs"
	"github.com/mendixlabs/mxcli/internal/testfsutil"
	"github.com/mendixlabs/mxcli/model"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// fixtureOpenSem limits concurrent fixture open operations to GOMAXPROCS
// to prevent I/O storms when many tests run in parallel.
var fixtureOpenSem = make(chan struct{}, runtime.GOMAXPROCS(0))

// typedID is a 1-line cast helper so test bodies stay readable.
func typedID(s string) model.ID { return model.ID(s) }

// fixturePath is the canonical Stage 2 fixture: a v2 MPR with
// mprcontents/ folder and 16 Microflows$Microflow units. Path is
// relative to test files in mdl/backend/mpr/repos/.
const fixturePath = "../../../../testdata/expr-checker/minimal.mpr"

// openTestWriter sets up an isolated writable copy of the fixture and
// opens it as a *mmpr.Writer.
//
// The .mpr SQLite file is real-copied (Writer writes WAL back to it).
// mprcontents/ is hard-linked via testfsutil.HardLinkDir — O(1) per
// file with automatic EXDEV fallback to byte-copy on cross-device setups.
func openTestWriter(t *testing.T) *mmpr.Writer {
	t.Helper()
	t.Parallel()
	fixtureOpenSem <- struct{}{}
	defer func() { <-fixtureOpenSem }()
	dst := copyFixture(t, fixturePath)
	w, err := mmpr.NewWriter(dst)
	if err != nil {
		t.Fatalf("NewWriter(%s): %v", dst, err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w
}

func copyFixture(t *testing.T, srcMPR string) string {
	t.Helper()
	snap, err := goldenfs.Open(filepath.Dir(srcMPR))
	if err == nil {
		t.Cleanup(func() { snap.Close() })
		return filepath.Join(snap.MountDir(), filepath.Base(srcMPR))
	}
	if !errors.Is(err, goldenfs.ErrNotSupported) {
		t.Fatalf("goldenfs.Open: %v", err)
	}
	dstDir := testfsutil.SameFSTempDir(t)
	dst, err := copyMPRTree(srcMPR, dstDir)
	if err != nil {
		t.Fatalf("copy fixture %s -> %s: %v", srcMPR, dstDir, err)
	}
	return dst
}

func copyMPRTree(srcMPR, dstDir string) (string, error) {
	dstMPR := filepath.Join(dstDir, filepath.Base(srcMPR))
	if err := testfsutil.CopyFile(srcMPR, dstMPR); err != nil {
		return "", err
	}
	srcContents := filepath.Join(filepath.Dir(srcMPR), "mprcontents")
	if info, err := os.Stat(srcContents); err == nil && info.IsDir() {
		dstContents := filepath.Join(dstDir, "mprcontents")
		if err := testfsutil.HardLinkDir(srcContents, dstContents); err != nil {
			return "", err
		}
	}
	return dstMPR, nil
}

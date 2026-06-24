// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mendixlabs/mxcli/internal/goldenfs"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const fixtureMprPath = "../../testdata/expr-checker/minimal.mpr"

// fixtureDir is the directory containing the fixture MPR file.
var fixtureDir = filepath.Dir(fixtureMprPath)

// fixtureMPRBase is the base name of the fixture MPR file.
var fixtureMPRBase = filepath.Base(fixtureMprPath)

var openFixtureSem = make(chan struct{}, runtime.GOMAXPROCS(0))

// openMprWriterForTest copies the fixture into a per-test temp dir
// (hard-linking mprcontents/ when source and /tmp share a filesystem)
// and returns an mmpr.Writer. Uses goldenfs FUSE overlay when available.
// parallelOnce() ensures t.Parallel() fires exactly once even when this
// helper is invoked multiple times in the same test (e.g. via helper chains).
// openFixtureSem throttles concurrent I/O to GOMAXPROCS to prevent I/O storms.
func openMprWriterForTest(t *testing.T) *mmpr.Writer {
	t.Helper()
	parallelOnce(t)

	if w := tryOpenWriterWithOverlay(t); w != nil {
		return w
	}

	openFixtureSem <- struct{}{}
	defer func() { <-openFixtureSem }()
	dst := copyMPRFixture(t, fixtureMprPath, t.TempDir())
	w, err := mmpr.NewWriter(dst)
	if err != nil {
		t.Fatalf("mmpr.NewWriter(%s): %v", dst, err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w
}

// tryOpenWriterWithOverlay attempts to open a goldenfs FUSE overlay over the
// fixture directory. Returns nil if FUSE is unavailable (non-Linux, etc.).
func tryOpenWriterWithOverlay(t *testing.T) *mmpr.Writer {
	t.Helper()
	overlay, err := goldenfs.Open(fixtureDir)
	if err != nil {
		return nil
	}
	t.Cleanup(func() { overlay.Rollback(); overlay.Close() })
	mprPath := filepath.Join(overlay.MountDir(), fixtureMPRBase)
	w, err := mmpr.NewWriter(mprPath)
	if err != nil {
		t.Fatalf("mmpr.NewWriter(%s): %v", mprPath, err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w
}

// tryOpenBackendWithOverlay attempts to open an MprBackend via goldenfs FUSE
// overlay. Returns nil if FUSE is unavailable.
func tryOpenBackendWithOverlay(t *testing.T) *mprbackend.MprBackend {
	t.Helper()
	overlay, err := goldenfs.Open(fixtureDir)
	if err != nil {
		return nil
	}
	t.Cleanup(func() { overlay.Rollback(); overlay.Close() })
	mprPath := filepath.Join(overlay.MountDir(), fixtureMPRBase)
	be, err := mprbackend.NewFromPath(mprPath)
	if err != nil {
		t.Fatalf("mprbackend.NewFromPath(%s): %v", mprPath, err)
	}
	t.Cleanup(func() { _ = be.Disconnect() })
	return be
}

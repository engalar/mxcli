// SPDX-License-Identifier: Apache-2.0

//go:build linux

package executor

import (
	"runtime"
	"testing"

	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const fixtureMprPath = "../../testdata/expr-checker/minimal.mpr"

var openFixtureSem = make(chan struct{}, runtime.GOMAXPROCS(0))

// openMprWriterForTest copies (or hard-links) the fixture into a per-test
// temp dir and returns an mmpr.Writer. parallelOnce() ensures t.Parallel()
// fires exactly once even when this helper is invoked multiple times.
// openFixtureSem throttles concurrent I/O to GOMAXPROCS.
func openMprWriterForTest(t *testing.T) *mmpr.Writer {
	t.Helper()
	parallelOnce(t)
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

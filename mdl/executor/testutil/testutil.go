// SPDX-License-Identifier: Apache-2.0

// Package testutil provides test helpers for testing executor commands
// from external packages (package executor_test or other packages).
//
// For tests WITHIN the executor package, continue using the existing
// newMockCtx pattern in mock_test_helpers_test.go — this package
// complements that pattern, it does not replace it.
package testutil

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

// TestExec wraps an Executor for use in tests from external packages.
// Obtain via New, NewWithProject, or (future) NewWithMPRBytes.
type TestExec struct {
	// Mock is the underlying MockBackend; non-nil only when created via New.
	// Configure Func fields before calling Run to control backend responses.
	Mock *mock.MockBackend

	t    *testing.T
	exec *executor.Executor
	buf  *strings.Builder

	// mount is non-nil only when created via NewWithMPRBytes (linux/darwin).
	// Typed as an interface so this struct compiles on platforms without FUSE.
	mount mprMounter
}

// mprMounter is satisfied by *MPRMount (defined in fuse.go, build-tagged for
// linux/darwin). Declaring the field as an interface keeps TestExec buildable
// on platforms where MPRMount does not exist.
type mprMounter interface {
	Bytes() []byte
}

// New creates a TestExec backed by a MockBackend. The mock's IsConnectedFunc
// is pre-set to return true so callers can skip CONNECT and test commands directly.
// All other Func fields start nil (return zero values / nil error by default).
func New(t *testing.T) *TestExec {
	t.Helper()
	m := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	var buf strings.Builder
	exec := executor.Build().
		Out(&buf).
		WithBackend(m).
		Quiet().
		Create()
	t.Cleanup(func() { exec.Close() })
	return &TestExec{Mock: m, t: t, exec: exec, buf: &buf}
}

// NewWithProject creates a TestExec connected to a real MPR file at mprPath.
// Suitable for integration tests; guard with testing.Short() in CI.
func NewWithProject(t *testing.T, mprPath string) *TestExec {
	t.Helper()
	be := mprbackend.New()
	if err := be.Connect(mprPath); err != nil {
		t.Fatalf("testutil.NewWithProject: connect %s: %v", mprPath, err)
	}
	var buf strings.Builder
	exec := executor.Build().
		Out(&buf).
		WithBackend(be).
		Quiet().
		Create()
	t.Cleanup(func() {
		exec.Close()
		_ = be.Disconnect()
	})
	return &TestExec{t: t, exec: exec, buf: &buf}
}

// Run executes mdl and returns (stdout, nil) on success, ("", error) on failure.
// Does NOT call t.Fatal — the caller decides how to handle errors.
func (te *TestExec) Run(mdl string) (string, error) {
	te.t.Helper()
	te.buf.Reset()
	prog, errs := visitor.Build(mdl)
	if len(errs) > 0 {
		return "", fmt.Errorf("parse: %v", errs[0])
	}
	err := te.exec.ExecuteProgram(prog)
	if errors.Is(err, executor.ErrExit) {
		err = nil
	}
	return te.buf.String(), err
}

// RunError is an alias for Run, emphasising the caller expects an error path.
func (te *TestExec) RunError(mdl string) (string, error) {
	return te.Run(mdl)
}

// Executor returns the underlying *executor.Executor for advanced use.
func (te *TestExec) Executor() *executor.Executor {
	return te.exec
}

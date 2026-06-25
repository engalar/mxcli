// SPDX-License-Identifier: Apache-2.0

//go:build linux && integration

package goldenfs

import (
	"io"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/executor/domainmodel"
	"github.com/mendixlabs/mxcli/mdl/executor/microflow"
	"github.com/mendixlabs/mxcli/mdl/executor/misc"
	"github.com/mendixlabs/mxcli/mdl/executor/page"
	"github.com/mendixlabs/mxcli/mdl/executor/query"
	"github.com/mendixlabs/mxcli/mdl/executor/security"
	"github.com/mendixlabs/mxcli/mdl/executor/workflow"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

// runMDL executes a script against the executor, prefixing it with
// `connect local '<mprPath>';` so the backend wires to the FUSE-mounted file.
// A fresh executor + backend is used per call to avoid stale list-cache issues
// (matches workflow_integration_test.go pattern).
//
// The executor is closed before runMDL returns so that the SQLite connection is
// fully released before bsoncompare opens the same file. A close failure is
// reported as a test error (not just a warning) because an unreleased handle
// could cause the subsequent ReadAllUnits call to see a locked or inconsistent
// file.
func runMDL(t *testing.T, mprPath, script string) {
	t.Helper()
	e := executor.New(io.Discard)
	e.SetQuiet(true)
	e.SetBackendFactory(func() backend.ConnectionBackend { return mprbackend.New() })
	deps := e.BuildHandlerDeps()
	microflow.RegisterHandlers(e.Registry(), deps)
	page.RegisterHandlers(e.Registry(), deps)
	workflow.RegisterHandlers(e.Registry(), deps)
	domainmodel.RegisterHandlers(e.Registry(), deps)
	security.RegisterHandlers(e.Registry(), deps)
	query.RegisterHandlers(e.Registry(), deps)
	misc.RegisterHandlers(e.Registry(), deps)
	e.AddReregister(func(fresh *executor.HandlerDeps) {
		microflow.RegisterHandlers(e.Registry(), fresh)
		page.RegisterHandlers(e.Registry(), fresh)
		workflow.RegisterHandlers(e.Registry(), fresh)
		domainmodel.RegisterHandlers(e.Registry(), fresh)
		security.RegisterHandlers(e.Registry(), fresh)
		query.RegisterHandlers(e.Registry(), fresh)
		misc.RegisterHandlers(e.Registry(), fresh)
	})
	defer func() {
		if err := e.Close(); err != nil {
			t.Errorf("runMDL: executor close: %v", err)
		}
	}()

	full := "connect local '" + mprPath + "';\n" + script
	prog, errs := visitor.Build(full)
	if len(errs) > 0 {
		t.Fatalf("MDL parse error: %v", errs)
	}
	if err := e.ExecuteProgram(prog); err != nil {
		t.Fatalf("executor error: %v", err)
	}
}

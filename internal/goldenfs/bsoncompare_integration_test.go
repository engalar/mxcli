// SPDX-License-Identifier: Apache-2.0

//go:build linux && integration

package goldenfs

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
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
	// Register subpackage handlers so Connect's registerFutureOverlays
	// + reRegisterAll have proper deps to work with.
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

// TestBsonCompare_CreateMicroflow drives an MDL script that creates a single
// microflow through the FUSE overlay, then uses bsoncompare to verify that the
// only BSON-level change between the pristine baseline and the post-mutation
// snapshot is the addition of the new microflow unit.
func TestBsonCompare_CreateMicroflow(t *testing.T) {
	realDir := exprCheckerDir(t)
	realMpr := filepath.Join(realDir, "minimal.mpr")
	origStat, err := os.Stat(realMpr)
	if err != nil {
		t.Fatalf("stat real mpr: %v", err)
	}

	snap, err := Open(realDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := snap.Close(); err != nil {
			t.Logf("snap.Close warning: %v", err)
		}
	}()

	mountMpr := filepath.Join(snap.MountDir(), "minimal.mpr")

	runMDL(t, mountMpr, `create or modify microflow MyFirstModule.ACT_BsonCompareTest ()
  returns Nothing
  {
    return;
  }`)

	// A = pristine baseline (read-only via real path)
	// B = post-mutation FUSE mount (overlay reflects the write)
	bsoncompare.AssertEqual(t, realMpr, mountMpr, bsoncompare.DefaultOptions(),
		bsoncompare.ExpectAdded("MyFirstModule.ACT_BsonCompareTest"),
		bsoncompare.ExpectNoOtherChanges(),
	)

	// Base must be untouched — overlay isolation matches other goldenfs tests.
	checkFUSEIsolation(t, realMpr, origStat)
	snap.Rollback()
}

// TestBsonCompare_CreateEnumeration verifies that creating an enumeration adds
// exactly one new unit (the enumeration itself) and nothing else.
func TestBsonCompare_CreateEnumeration(t *testing.T) {
	realDir := exprCheckerDir(t)
	realMpr := filepath.Join(realDir, "minimal.mpr")
	origStat, err := os.Stat(realMpr)
	if err != nil {
		t.Fatalf("stat real mpr: %v", err)
	}

	snap, err := Open(realDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := snap.Close(); err != nil {
			t.Logf("snap.Close warning: %v", err)
		}
	}()

	mountMpr := filepath.Join(snap.MountDir(), "minimal.mpr")

	runMDL(t, mountMpr, `create or modify enumeration MyFirstModule.BsonTest_Status (
  Active   'Active',
  Inactive 'Inactive'
);`)

	bsoncompare.AssertEqual(t, realMpr, mountMpr, bsoncompare.DefaultOptions(),
		bsoncompare.ExpectAdded("MyFirstModule.BsonTest_Status"),
		bsoncompare.ExpectNoOtherChanges(),
	)

	checkFUSEIsolation(t, realMpr, origStat)
	snap.Rollback()
}

// TestBsonCompare_CreatePage verifies that creating a minimal page adds exactly
// one new Forms$Page unit and no other changes.
func TestBsonCompare_CreatePage(t *testing.T) {
	realDir := exprCheckerDir(t)
	realMpr := filepath.Join(realDir, "minimal.mpr")
	origStat, err := os.Stat(realMpr)
	if err != nil {
		t.Fatalf("stat real mpr: %v", err)
	}

	snap, err := Open(realDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := snap.Close(); err != nil {
			t.Logf("snap.Close warning: %v", err)
		}
	}()

	mountMpr := filepath.Join(snap.MountDir(), "minimal.mpr")

	runMDL(t, mountMpr, `create page MyFirstModule.BsonTest_Page
  (title: 'BsonTest Page', layout: Atlas_Core.Atlas_TopBar) { }`)

	bsoncompare.AssertEqual(t, realMpr, mountMpr, bsoncompare.DefaultOptions(),
		bsoncompare.ExpectAdded("MyFirstModule.BsonTest_Page"),
		bsoncompare.ExpectNoOtherChanges(),
	)

	checkFUSEIsolation(t, realMpr, origStat)
	snap.Rollback()
}

// TestBsonCompare_DropMicroflow verifies the full create-then-drop roundtrip:
// after dropping a microflow that was created in the same session, bsoncompare
// must see zero net changes relative to the pristine baseline.
func TestBsonCompare_DropMicroflow(t *testing.T) {
	realDir := exprCheckerDir(t)
	realMpr := filepath.Join(realDir, "minimal.mpr")
	origStat, err := os.Stat(realMpr)
	if err != nil {
		t.Fatalf("stat real mpr: %v", err)
	}

	snap, err := Open(realDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := snap.Close(); err != nil {
			t.Logf("snap.Close warning: %v", err)
		}
	}()

	mountMpr := filepath.Join(snap.MountDir(), "minimal.mpr")

	// Step 1: create the microflow and verify it appears as added.
	runMDL(t, mountMpr, `create or modify microflow MyFirstModule.ACT_BsonDropTest ()
  returns Nothing
  {
    return;
  }`)

	bsoncompare.AssertEqual(t, realMpr, mountMpr, bsoncompare.DefaultOptions(),
		bsoncompare.ExpectAdded("MyFirstModule.ACT_BsonDropTest"),
		bsoncompare.ExpectNoOtherChanges(),
	)

	// Step 2: drop it and verify the unit disappears — net diff from baseline is zero.
	runMDL(t, mountMpr, `drop microflow MyFirstModule.ACT_BsonDropTest;`)

	bsoncompare.AssertEqual(t, realMpr, mountMpr, bsoncompare.DefaultOptions(),
		bsoncompare.ExpectNoOtherChanges(),
	)

	checkFUSEIsolation(t, realMpr, origStat)
	snap.Rollback()
}

// TestBsonCompare_NoOpScript verifies that connecting + disconnecting without
// any mutating statements produces zero BSON diffs between the baseline and
// the overlay snapshot. Detects spurious writes from the executor / backend
// connect path that would mask real mutations in other tests.
func TestBsonCompare_NoOpScript(t *testing.T) {
	realDir := exprCheckerDir(t)
	realMpr := filepath.Join(realDir, "minimal.mpr")
	origStat, err := os.Stat(realMpr)
	if err != nil {
		t.Fatalf("stat real mpr: %v", err)
	}

	snap, err := Open(realDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := snap.Close(); err != nil {
			t.Logf("snap.Close warning: %v", err)
		}
	}()

	mountMpr := filepath.Join(snap.MountDir(), "minimal.mpr")

	// show modules; is a pure read: the backend opens the MPR in read-write
	// mode but commits no write transaction, so bsoncompare must see zero diffs.
	// Any spurious write from the connect path would be caught here.
	runMDL(t, mountMpr, `show modules;`)

	bsoncompare.AssertEqual(t, realMpr, mountMpr, bsoncompare.DefaultOptions(),
		bsoncompare.ExpectNoOtherChanges(),
	)

	checkFUSEIsolation(t, realMpr, origStat)
	snap.Rollback()
}

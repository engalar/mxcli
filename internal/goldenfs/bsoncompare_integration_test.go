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
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

// runMDL executes a script against the executor, prefixing it with
// `connect local '<mprPath>';` so the backend wires to the FUSE-mounted file.
// A fresh executor + backend is used per call to avoid stale list-cache issues
// (matches workflow_integration_test.go pattern).
func runMDL(t *testing.T, mprPath, script string) {
	t.Helper()
	e := executor.New(io.Discard)
	e.SetQuiet(true)
	e.SetBackendFactory(func() backend.FullBackend { return mprbackend.New() })
	defer func() {
		if err := e.Close(); err != nil {
			t.Logf("executor close warning: %v", err)
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
	defer snap.Close()

	mountMpr := filepath.Join(snap.MountDir(), "minimal.mpr")

	runMDL(t, mountMpr, `create or modify microflow MyFirstModule.ACT_BsonCompareTest ()
  returns Nothing
  begin
    return;
  end;`)

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
	defer snap.Close()

	mountMpr := filepath.Join(snap.MountDir(), "minimal.mpr")

	// connect-only script; no create/alter/drop.
	runMDL(t, mountMpr, `show modules;`)

	bsoncompare.AssertEqual(t, realMpr, mountMpr, bsoncompare.DefaultOptions(),
		bsoncompare.ExpectNoOtherChanges(),
	)

	checkFUSEIsolation(t, realMpr, origStat)
	snap.Rollback()
}

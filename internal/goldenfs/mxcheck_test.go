// SPDX-License-Identifier: Apache-2.0

//go:build linux && integration

package goldenfs

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

// TestGoldenFS_ExecAndMxCheck is the full pipeline: MDL → executor →
// MprBackend → FUSE overlay → SQLite write → mx check verification.
// Built behind the `integration` tag because it depends on the mx binary
// being installed on the runner.
func TestGoldenFS_ExecAndMxCheck(t *testing.T) {
	mxBin := findMxBinaryForTest()
	if mxBin == "" {
		t.Skip("mx binary not available (set MX_BINARY or install mxbuild)")
	}

	// Use helpdesk-clean MPR: a blank project with Atlas Core that passes
	// mx check with 0 errors, so any FUSE-corruption signature in the output
	// is from our new microflow, not pre-existing project issues.
	realDir := helpdeskBlankDir(t)
	realMpr := filepath.Join(realDir, "minimal.mpr")
	origMprStat, err := os.Stat(realMpr)
	if err != nil {
		t.Fatalf("stat real mpr: %v", err)
	}

	snap, err := Open(realDir)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	mprPath := filepath.Join(snap.MountDir(), "minimal.mpr")

	// Drive the executor against the FUSE mount via the public API: factory
	// produces an unconnected MprBackend; `connect` in the script wires it
	// to the FUSE-mounted .mpr. ExecContext.Cache is unexported so we don't
	// touch it directly.
	out := &bytes.Buffer{}
	e := executor.New(out)
	e.SetQuiet(true)
	e.SetBackendFactory(func() backend.ConnectionBackend { return mprbackend.New() })
	defer func() {
		if err := e.Close(); err != nil {
			t.Logf("executor close warning: %v", err)
		}
	}()

	script := fmt.Sprintf(`connect local '%s';
create or modify microflow MyFirstModule.ACT_GoldenTest () returns Nothing { return; }`, mprPath)

	prog, errs := visitor.Build(script)
	if len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	if err := e.ExecuteProgram(prog); err != nil {
		t.Fatalf("exec:\n%s\n%v", out.String(), err)
	}

	// Disconnect the executor so SQLite releases its handles on the mount;
	// `mx check` will open the same file via Mono.
	if err := e.Close(); err != nil {
		t.Fatalf("close executor: %v", err)
	}

	cmd := exec.Command(mxBin, "check", mprPath)
	output, err := cmd.CombinedOutput()
	t.Logf("mx check output (%d bytes, err=%v):\n%s", len(output), err, output)

	assertNoFUSECorruption(t, string(output), "ACT_GoldenTest")
	checkFUSEIsolation(t, realMpr, origMprStat)

	snap.Rollback()
}

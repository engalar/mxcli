// SPDX-License-Identifier: Apache-2.0

//go:build linux && integration

package goldenfs

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

// findMxBinaryForTest returns the path to an `mx` binary, preferring
// MX_BINARY if set, otherwise the highest-version mxbuild under ~/.mxcli.
func findMxBinaryForTest() string {
	if p := os.Getenv("MX_BINARY"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		pattern := filepath.Join(home, ".mxcli", "mxbuild", "*", "modeler", "mx")
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			// Last match = highest sort order (e.g. 11.6.6 after 11.6.4).
			return matches[len(matches)-1]
		}
	}
	return ""
}

// TestGoldenFS_ExecAndMxCheck is the full pipeline: MDL → executor →
// MprBackend → FUSE overlay → SQLite write → mx check verification.
// Built behind the `integration` tag because it depends on the mx binary
// being installed on the runner.
func TestGoldenFS_ExecAndMxCheck(t *testing.T) {
	mxBin := findMxBinaryForTest()
	if mxBin == "" {
		t.Skip("mx binary not available (set MX_BINARY or install mxbuild)")
	}

	// Snapshot the testdata before anything else so a base-dir tampering
	// check later is meaningful.
	realDir := exprCheckerDir(t)
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
	e.SetBackendFactory(func() backend.FullBackend { return mprbackend.New() })
	defer func() {
		if err := e.Close(); err != nil {
			t.Logf("executor close warning: %v", err)
		}
	}()

	script := fmt.Sprintf(`connect local '%s';
create or modify microflow MyFirstModule.ACT_GoldenTest () returns Nothing begin return; end;`, mprPath)

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

	// mx check on the FUSE mount. The testdata project has many pre-existing
	// CE6083 design-property errors that have nothing to do with our write;
	// we only assert that no FUSE-corruption signatures appear and that our
	// new microflow is not flagged as broken.
	cmd := exec.Command(mxBin, "check", mprPath)
	output, err := cmd.CombinedOutput()
	t.Logf("mx check output (%d bytes, err=%v):\n%s", len(output), err, output)

	// Fatal signatures that would indicate the FUSE overlay corrupted the file
	// or the .NET runtime couldn't parse what we wrote.
	fatalSignatures := []string{
		"StorageLoadException",   // SQLite file corruption / mprcontents desync
		"TypeCacheUnknownType",   // BSON $Type written that Mendix doesn't recognise
		"CE0066",                 // Entity access out of date
		"CE0463",                 // Widget definition changed (BSON shape mismatch)
		"CE1613",                 // Layout no longer exists
		"Invalid file format",    // SQLite header damage
	}
	outStr := string(output)
	for _, sig := range fatalSignatures {
		if strings.Contains(outStr, sig) {
			t.Fatalf("mx check reported FUSE-corruption signature %q in output", sig)
		}
	}
	if strings.Contains(outStr, "ACT_GoldenTest") {
		// If the microflow appears in errors, the executor wrote bad BSON.
		t.Fatalf("our newly-created microflow ACT_GoldenTest is flagged by mx check")
	}

	// Base verification — overlay must have absorbed the write completely.
	newMprStat, err := os.Stat(realMpr)
	if err != nil {
		t.Fatalf("stat real mpr after: %v", err)
	}
	if !origMprStat.ModTime().Equal(newMprStat.ModTime()) {
		t.Fatalf("real minimal.mpr mtime changed (%v → %v) — overlay leaked",
			origMprStat.ModTime(), newMprStat.ModTime())
	}
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if _, err := os.Stat(realMpr + suffix); err == nil {
			t.Fatalf("%s file must not exist in base dir after FUSE write", suffix)
		}
	}

	// Explicit rollback for clarity. Snapshot is closed by the defer anyway.
	snap.Rollback()
}

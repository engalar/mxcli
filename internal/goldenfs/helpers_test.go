// SPDX-License-Identifier: Apache-2.0

//go:build linux && integration

package goldenfs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// findMxBinaryForTest returns the path to an mx binary, or "" if unavailable.
// Prefers MX_BINARY env; falls back to highest lexicographic version under
// ~/.mxcli/mxbuild/*/modeler/mx.
func findMxBinaryForTest() string {
	if p := os.Getenv("MX_BINARY"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		pattern := filepath.Join(home, ".mxcli", "mxbuild", "*", "modeler", "mx")
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			// Last in lexicographic order — set MX_BINARY to override when version
			// layout causes incorrect selection (e.g. 11.10.x < 11.9.x lexicographically).
			return matches[len(matches)-1]
		}
	}
	return ""
}

// assertNoFUSECorruption fails the test if mx check output contains any
// load-time crash signature, or if any of ourObjects appears in the output.
// Pre-existing CE6083 design-property errors are allowed through.
//
// A non-zero exit code alone is not sufficient — mx check exits non-zero for
// ordinary model errors (CE*) too. We specifically look for signatures that
// indicate the FUSE overlay corrupted the file OR our BSON writes produced
// data Mendix cannot deserialise.
func assertNoFUSECorruption(t *testing.T, output string, ourObjects ...string) {
	t.Helper()
	fatalSignatures := []string{
		"StorageLoadException",      // SQLite file corruption / mprcontents desync
		"TypeCacheUnknownType",      // BSON $Type written that Mendix doesn't recognise
		"InvalidOperationException", // type mismatch when setting a BSON field (e.g. wrong wrapper type)
		"ArgumentException",         // inner cause of many InvalidOperationException crashes
		"CE0066",                    // Entity access out of date
		"CE0463",                    // Widget definition changed (BSON shape mismatch)
		"CE1613",                    // Layout no longer exists
		"Invalid file format",       // SQLite header damage
	}
	for _, sig := range fatalSignatures {
		if strings.Contains(output, sig) {
			t.Fatalf("mx check reported FUSE-corruption signature %q in output", sig)
		}
	}
	for _, obj := range ourObjects {
		if strings.Contains(output, obj) {
			t.Fatalf("our newly-created object %q is flagged by mx check", obj)
		}
	}
}

// checkFUSEIsolation verifies that the real MPR file was not modified and
// no SQLite auxiliary files leaked to the base directory.
func checkFUSEIsolation(t *testing.T, realMpr string, origStat os.FileInfo) {
	t.Helper()
	newStat, err := os.Stat(realMpr)
	if err != nil {
		t.Fatalf("stat real mpr after: %v", err)
	}
	if !origStat.ModTime().Equal(newStat.ModTime()) {
		t.Fatalf("real minimal.mpr mtime changed (%v → %v) — overlay leaked",
			origStat.ModTime(), newStat.ModTime())
	}
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if _, err := os.Stat(realMpr + suffix); err == nil {
			t.Fatalf("%s must not exist in base dir after FUSE write", suffix)
		}
	}
}

// SPDX-License-Identifier: Apache-2.0

//go:build linux && integration

package goldenfs

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// parseSemver splits a "MAJOR.MINOR.PATCH" string into three ints.
// Returns false if the string cannot be parsed.
func parseSemver(v string) (major, minor, patch int, ok bool) {
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	var err error
	if major, err = strconv.Atoi(parts[0]); err != nil {
		return 0, 0, 0, false
	}
	if minor, err = strconv.Atoi(parts[1]); err != nil {
		return 0, 0, 0, false
	}
	if patch, err = strconv.Atoi(parts[2]); err != nil {
		return 0, 0, 0, false
	}
	return major, minor, patch, true
}

// semverGT returns true when a > b in semver order.
func semverGT(aMaj, aMin, aPatch, bMaj, bMin, bPatch int) bool {
	if aMaj != bMaj {
		return aMaj > bMaj
	}
	if aMin != bMin {
		return aMin > bMin
	}
	return aPatch > bPatch
}

// findMxBinaryForVersion returns the mx binary for a specific Mendix version.
// It checks MX_BINARY env first (exact binary path), then looks for an exact
// version-specific binary under ~/.mxcli/mxbuild/{version}/modeler/mx, then
// falls back to the highest semver version available.
//
// Using lexicographic ordering is incorrect: "11.10.0" < "11.6.6" lex-wise,
// so earlier code using the lex-last entry would pick 11.6.6 even when 11.10.0
// is installed. Semver ordering is the correct fallback.
func findMxBinaryForVersion(projectVersion string) string {
	if p := os.Getenv("MX_BINARY"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	// Exact version match first.
	if projectVersion != "" {
		exact := filepath.Join(home, ".mxcli", "mxbuild", projectVersion, "modeler", "mx")
		if _, err := os.Stat(exact); err == nil {
			return exact
		}
	}
	// Fall back: pick the highest semver version installed.
	pattern := filepath.Join(home, ".mxcli", "mxbuild", "*", "modeler", "mx")
	matches, _ := filepath.Glob(pattern)
	if len(matches) == 0 {
		return ""
	}
	best := ""
	var bestMaj, bestMin, bestPatch int
	for _, m := range matches {
		// Extract version from path: ~/.mxcli/mxbuild/<version>/modeler/mx
		versionDir := filepath.Base(filepath.Dir(filepath.Dir(m)))
		maj, min, patch, ok := parseSemver(versionDir)
		if !ok {
			continue
		}
		if best == "" || semverGT(maj, min, patch, bestMaj, bestMin, bestPatch) {
			best = m
			bestMaj, bestMin, bestPatch = maj, min, patch
		}
	}
	if best != "" {
		return best
	}
	return matches[0]
}

// findMxBinaryForTest returns the path to an mx binary, or "" if unavailable.
// Prefers MX_BINARY env; falls back to the highest semver version under
// ~/.mxcli/mxbuild/*/modeler/mx.
func findMxBinaryForTest() string {
	return findMxBinaryForVersion("")
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

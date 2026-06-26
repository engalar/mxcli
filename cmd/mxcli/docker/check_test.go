// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCheck_SkipUpdateWidgets(t *testing.T) {
	// This test verifies the SkipUpdateWidgets option is wired through.
	// Since we don't have a real mx binary in CI, we just verify the
	// function returns the expected "mx not found" error.
	opts := CheckOptions{
		ProjectPath:       "/nonexistent/app.mpr",
		SkipUpdateWidgets: true,
		Stdout:            &bytes.Buffer{},
		Stderr:            &bytes.Buffer{},
	}

	err := Check(opts)
	if err == nil {
		t.Fatal("expected error when project file not found")
	}
	t.Logf("got expected error: %s", err)
}

// createFakeMxDir creates a temp directory with fake mx and mxbuild scripts
// that log their first argument to a file.
func createFakeMxDir(t *testing.T) (dir, logFile string) {
	t.Helper()
	dir = t.TempDir()
	logFile = filepath.Join(dir, "commands.log")

	script := `#!/bin/sh
echo "$1" >> ` + logFile + "\n"

	for _, name := range []string{"mx", "mxbuild"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(script), 0755); err != nil {
			t.Fatal(err)
		}
	}
	return dir, logFile
}

// fakeMPRPath creates a temp directory with an empty fake .mpr file.
// The file must exist on disk so Check() can validate the path before
// invoking mx (JVM startup is ~2.5s). A zero-byte file is sufficient
// because the test only verifies mx invocation order, not MPR parsing.
// Using t.TempDir() avoids pollution from ambient mprcontents/ directories
// (e.g. /tmp/mprcontents/) that would confuse the MPR v2 detection logic.
func fakeMPRPath(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fake.mpr")
	if err := os.WriteFile(p, nil, 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

// fakeMPRPathV2 creates a temp directory that looks like an MPR v2 project:
// the directory contains an mprcontents/ sub-directory so that isMPRv2 returns
// true. The .mpr file must also exist on disk so Check() can validate the path.
func fakeMPRPathV2(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "mprcontents"), 0755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "fake.mpr")
	if err := os.WriteFile(p, nil, 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCheck_UpdateWidgetsBeforeCheck(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on Windows")
	}

	mxDir, logFile := createFakeMxDir(t)

	var stdout, stderr bytes.Buffer
	opts := CheckOptions{
		ProjectPath: fakeMPRPath(t), // v1: no mprcontents/ sibling
		MxBuildPath: mxDir,
		Stdout:      &stdout,
		Stderr:      &stderr,
	}

	Check(opts)

	logBytes, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read command log: %v", err)
	}

	log := string(logBytes)
	if !bytes.Contains(logBytes, []byte("update-widgets\n")) {
		t.Errorf("update-widgets was not called, got log:\n%s", log)
	}
	if !bytes.Contains(logBytes, []byte("check\n")) {
		t.Errorf("check was not called, got log:\n%s", log)
	}

	// Verify order: update-widgets before check
	uwIdx := bytes.Index(logBytes, []byte("update-widgets"))
	chIdx := bytes.Index(logBytes, []byte("check"))
	if uwIdx >= chIdx {
		t.Errorf("update-widgets should run before check, got log:\n%s", log)
	}
}

func TestCheck_SkipUpdateWidgetsFlag(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on Windows")
	}

	mxDir, logFile := createFakeMxDir(t)

	var stdout, stderr bytes.Buffer
	opts := CheckOptions{
		ProjectPath:       fakeMPRPath(t), // v1: no mprcontents/ sibling
		MxBuildPath:       mxDir,
		SkipUpdateWidgets: true,
		Stdout:            &stdout,
		Stderr:            &stderr,
	}

	Check(opts)

	logBytes, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read command log: %v", err)
	}

	if bytes.Contains(logBytes, []byte("update-widgets")) {
		t.Error("update-widgets should NOT be called when SkipUpdateWidgets=true")
	}
	if !bytes.Contains(logBytes, []byte("check")) {
		t.Error("check should still be called")
	}
}

func TestCheck_V2SkipsUpdateWidgetsByDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on Windows")
	}

	mxDir, logFile := createFakeMxDir(t)

	var stdout, stderr bytes.Buffer
	opts := CheckOptions{
		ProjectPath: fakeMPRPathV2(t), // v2: has mprcontents/ sibling
		MxBuildPath: mxDir,
		Stdout:      &stdout,
		Stderr:      &stderr,
	}

	Check(opts)

	logBytes, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read command log: %v", err)
	}

	log := string(logBytes)
	if bytes.Contains(logBytes, []byte("update-widgets")) {
		t.Errorf("update-widgets should NOT be called for MPR v2 by default, got log:\n%s", log)
	}
	if !bytes.Contains(logBytes, []byte("check")) {
		t.Errorf("check should still be called, got log:\n%s", log)
	}

	// Warning should appear in stderr
	if !bytes.Contains(stderr.Bytes(), []byte("MPR v2 format detected")) {
		t.Errorf("expected MPR v2 warning in stderr, got: %s", stderr.String())
	}
}

func TestCheck_V2ForceUpdateWidgets(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on Windows")
	}

	mxDir, logFile := createFakeMxDir(t)

	var stdout, stderr bytes.Buffer
	opts := CheckOptions{
		ProjectPath:        fakeMPRPathV2(t), // v2: has mprcontents/ sibling
		MxBuildPath:        mxDir,
		ForceUpdateWidgets: true,
		Stdout:             &stdout,
		Stderr:             &stderr,
	}

	Check(opts)

	logBytes, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read command log: %v", err)
	}

	log := string(logBytes)
	if !bytes.Contains(logBytes, []byte("update-widgets\n")) {
		t.Errorf("update-widgets should be called when ForceUpdateWidgets=true, got log:\n%s", log)
	}
	if !bytes.Contains(logBytes, []byte("check\n")) {
		t.Errorf("check should still be called, got log:\n%s", log)
	}

	// Warning about v1 conversion should appear in stderr
	if !bytes.Contains(stderr.Bytes(), []byte("--update-widgets specified")) {
		t.Errorf("expected --update-widgets warning in stderr, got: %s", stderr.String())
	}
}

func TestResolveMxForVersion_PrefersExactCachedVersion(t *testing.T) {
	dir := t.TempDir()
	setTestHomeDir(t, dir)
	setTestApplicationsDir(t, t.TempDir()) // prevent real macOS Studio Pro from matching
	// Point PATH at an empty temp dir (rather than clearing it) so exec.LookPath
	// still works for any other testing infrastructure but can't find mx.
	t.Setenv("PATH", t.TempDir())

	versions := []string{"9.24.40.80973", "11.6.3", "11.9.0"}
	var expected string
	for _, version := range versions {
		modelerDir := filepath.Join(dir, ".mxcli", "mxbuild", version, "modeler")
		if err := os.MkdirAll(modelerDir, 0755); err != nil {
			t.Fatal(err)
		}
		bin := filepath.Join(modelerDir, mxBinaryName())
		if err := os.WriteFile(bin, []byte("fake"), 0755); err != nil {
			t.Fatal(err)
		}
		if version == "11.9.0" {
			expected = bin
		}
	}

	result, err := ResolveMxForVersion("", "11.9.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected exact cached mx %s, got %s", expected, result)
	}
}

// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestBuildMxMetadata_CompactJSON(t *testing.T) {
	got := buildMxMetadata("10.6.0.0", "Version2")

	// Must be valid JSON
	if !json.Valid([]byte(got)) {
		t.Fatalf("not valid JSON: %q", got)
	}

	// Must be compact (no spaces after separators)
	if strings.Contains(got, ": ") || strings.Contains(got, ", ") {
		t.Errorf("JSON is not compact (contains spaces): %q", got)
	}

	// Must not have trailing newline (libgit2 compatibility)
	if strings.HasSuffix(got, "\n") {
		t.Errorf("JSON must not have trailing newline")
	}

	// Must contain correct fields
	var m map[string]any
	_ = json.Unmarshal([]byte(got), &m)
	if m["ModelerVersion"] != "10.6.0.0" {
		t.Errorf("ModelerVersion = %v, want 10.6.0.0", m["ModelerVersion"])
	}
	if m["MPRFormatVersion"] != "Version2" {
		t.Errorf("MPRFormatVersion = %v, want Version2", m["MPRFormatVersion"])
	}
	if m["HasModelerVersion"] != true {
		t.Errorf("HasModelerVersion = %v, want true", m["HasModelerVersion"])
	}
	// ModelChanges and RelatedStories must be [] not null
	changes, ok := m["ModelChanges"].([]any)
	if !ok || changes == nil {
		t.Errorf("ModelChanges must be [] array, got %T %v", m["ModelChanges"], m["ModelChanges"])
	}
}

func TestBuildMxMetadata_Version1(t *testing.T) {
	got := buildMxMetadata("9.24.0.0", "Version1")
	var m map[string]any
	_ = json.Unmarshal([]byte(got), &m)
	if m["MPRFormatVersion"] != "Version1" {
		t.Errorf("MPRFormatVersion = %v, want Version1", m["MPRFormatVersion"])
	}
}

func TestHashObjectAndAddNote_Success(t *testing.T) {
	orig := gitExecCommand
	defer func() { gitExecCommand = orig }()

	const fakeSHA = "abc1234abc1234abc1234abc1234abc1234abc1234"
	const fakeBlob = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	var gotArgs [][]string

	gitExecCommand = func(name string, args ...string) *exec.Cmd {
		gotArgs = append(gotArgs, args)
		switch {
		case len(args) > 0 && args[0] == "hash-object":
			// Return fake blob hash (no trailing newline — printf not echo)
			return exec.Command("sh", "-c", "printf '"+fakeBlob+"'")
		case len(args) > 0 && args[0] == "notes":
			return exec.Command("sh", "-c", "exit 0")
		}
		return exec.Command("sh", "-c", "exit 1")
	}

	err := hashObjectAndAddNote(fakeSHA, `{"test":true}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify hash-object was called with -w --stdin
	if len(gotArgs) < 2 {
		t.Fatalf("expected 2 git calls, got %d", len(gotArgs))
	}
	if !contains(gotArgs[0], "hash-object") || !contains(gotArgs[0], "-w") {
		t.Errorf("first call must be hash-object -w --stdin, got %v", gotArgs[0])
	}
	// Verify notes add used the blob hash
	if !contains(gotArgs[1], "notes") || !contains(gotArgs[1], fakeBlob) {
		t.Errorf("second call must be notes add with blob hash, got %v", gotArgs[1])
	}
}

func TestHashObjectAndAddNote_NotesExist_UsesForce(t *testing.T) {
	orig := gitExecCommand
	defer func() { gitExecCommand = orig }()

	gitExecCommand = func(name string, args ...string) *exec.Cmd {
		switch {
		case contains(args, "hash-object"):
			return exec.Command("sh", "-c", "printf 'aabbccdd1122aabbccdd1122aabbccdd1122aabb'")
		case contains(args, "notes") && !contains(args, "-f"):
			// First notes add fails (note already exists)
			return exec.Command("sh", "-c", "exit 1")
		case contains(args, "notes") && contains(args, "-f"):
			// Force add succeeds
			return exec.Command("sh", "-c", "exit 0")
		}
		return exec.Command("sh", "-c", "exit 1")
	}

	err := hashObjectAndAddNote("sha123", `{"test":true}`)
	if err != nil {
		t.Fatalf("force retry must succeed, got: %v", err)
	}
}

// contains checks if a string slice contains a value.
func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func TestDetectMendixVersion_ExplicitFlag(t *testing.T) {
	v, fmt_, err := detectMendixVersion("10.6.0.0", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "10.6.0.0" {
		t.Errorf("version = %q, want 10.6.0.0", v)
	}
	if fmt_ != "Version2" {
		t.Errorf("format = %q, want Version2", fmt_)
	}
}

func TestDetectMendixVersion_NotesScanning(t *testing.T) {
	orig := gitExecCommand
	defer func() { gitExecCommand = orig }()

	// Simulate: git notes list returns two entries; first note has valid JSON
	gitExecCommand = func(name string, args ...string) *exec.Cmd {
		if contains(args, "list") {
			return exec.Command("sh", "-c",
				`printf 'noteblob1 commitsha1\nnoteblob2 commitsha2'`)
		}
		if contains(args, "show") && contains(args, "commitsha1") {
			note := `{"BranchName":"","ModelerVersion":"11.2.0.0","ModelChanges":[],"RelatedStories":[],"SolutionVersion":"","MPRFormatVersion":"Version2","HasModelerVersion":true}`
			return exec.Command("sh", "-c", "printf '"+note+"'")
		}
		return exec.Command("sh", "-c", "exit 1")
	}

	v, _, err := detectMendixVersion("", "/nonexistent-no-mpr-here.mpr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "11.2.0.0" {
		t.Errorf("version = %q, want 11.2.0.0", v)
	}
}

func TestDetectMendixVersion_NoVersionFound_Error(t *testing.T) {
	orig := gitExecCommand
	defer func() { gitExecCommand = orig }()

	// No MPR file, notes list is empty
	gitExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 1")
	}

	_, _, err := detectMendixVersion("", "/nonexistent-no-mpr-here.mpr")
	if err == nil {
		t.Fatal("expected error when version cannot be detected")
	}
}

func TestGitCommitWrapper_AddsNoteAfterCommit(t *testing.T) {
	orig := gitExecCommand
	defer func() { gitExecCommand = orig }()

	const commitSHA = "7fa3b2cabc1234567fa3b2cabc1234567fa3b2ca"

	gitExecCommand = func(name string, args ...string) *exec.Cmd {
		switch {
		case contains(args, "commit"):
			return exec.Command("sh", "-c", `echo '[main 7fa3b2c] Test commit'`)
		case contains(args, "rev-parse"):
			return exec.Command("sh", "-c", "printf '"+commitSHA+"'")
		case contains(args, "hash-object"):
			return exec.Command("sh", "-c", "printf 'blobhash123blobhash123blobhash123blobhash'")
		case contains(args, "notes"):
			return exec.Command("sh", "-c", "exit 0")
		}
		return exec.Command("sh", "-c", "exit 1")
	}

	var buf strings.Builder
	err := runGitCommit([]string{"-m", "Test commit"}, "10.6.0.0", "", &buf)
	if err != nil {
		t.Fatalf("runGitCommit returned error: %v", err)
	}

	out := buf.String()
	// Output must contain note confirmation
	if !strings.Contains(out, "mx_metadata note added") {
		t.Errorf("output missing note confirmation, got:\n%s", out)
	}
	// Output must contain next-step hint
	if !strings.Contains(out, "mxcli git notes push") {
		t.Errorf("output missing next-step hint, got:\n%s", out)
	}
}

func TestGitCommitWrapper_GitFails_NoNote(t *testing.T) {
	orig := gitExecCommand
	defer func() { gitExecCommand = orig }()

	noteAdded := false
	gitExecCommand = func(name string, args ...string) *exec.Cmd {
		if contains(args, "commit") {
			return exec.Command("sh", "-c", "exit 1") // git commit fails
		}
		if contains(args, "notes") {
			noteAdded = true
		}
		return exec.Command("sh", "-c", "exit 0")
	}

	var buf strings.Builder
	err := runGitCommit([]string{"-m", "bad"}, "10.6.0.0", "", &buf)
	if err == nil {
		t.Fatal("expected error when git commit fails")
	}
	if noteAdded {
		t.Error("must not write note when git commit fails")
	}
}

func TestRunGitNotesPush_Success(t *testing.T) {
	orig := gitExecCommand
	defer func() { gitExecCommand = orig }()

	var pushArgs []string
	gitExecCommand = func(name string, args ...string) *exec.Cmd {
		switch {
		case contains(args, "rev-parse") && contains(args, "--abbrev-ref"):
			// tracking remote detection → returns "origin/main", so remote = "origin"
			return exec.Command("sh", "-c", "printf 'origin/main'")
		case contains(args, "push"):
			pushArgs = args
			return exec.Command("sh", "-c", "exit 0")
		}
		return exec.Command("sh", "-c", "exit 1")
	}

	var buf strings.Builder
	err := runGitNotesPush("", false, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !contains(pushArgs, "refs/notes/mx_metadata") {
		t.Errorf("push must target refs/notes/mx_metadata, got: %v", pushArgs)
	}
}

func TestRunGitNotesPush_ForceFlag(t *testing.T) {
	orig := gitExecCommand
	defer func() { gitExecCommand = orig }()

	var pushArgs []string
	gitExecCommand = func(name string, args ...string) *exec.Cmd {
		if contains(args, "push") {
			pushArgs = args
			return exec.Command("sh", "-c", "exit 0")
		}
		return exec.Command("sh", "-c", "printf 'origin/main'")
	}

	var buf strings.Builder
	_ = runGitNotesPush("origin", true, &buf)
	if !contains(pushArgs, "--force") && !contains(pushArgs, "-f") {
		t.Errorf("--force must be passed to git push, got: %v", pushArgs)
	}
}

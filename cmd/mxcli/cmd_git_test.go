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

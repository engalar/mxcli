// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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

func TestDoctorCheck_GitConfig_Pass(t *testing.T) {
	orig := gitExecCommand
	defer func() { gitExecCommand = orig }()

	gitExecCommand = func(name string, args ...string) *exec.Cmd {
		// All config keys return correct values
		key := args[len(args)-1]
		switch key {
		case "core.autocrlf":
			return exec.Command("sh", "-c", "printf 'false'")
		case "mendix.commits-since-gc":
			return exec.Command("sh", "-c", "printf '0'")
		case "mendix.lineEndingResetDone":
			return exec.Command("sh", "-c", "printf 'true'")
		}
		return exec.Command("sh", "-c", "exit 1")
	}

	result := checkGitConfig()
	if !result.OK {
		t.Errorf("expected config check to pass, got: %s", result.Detail)
	}
}

func TestDoctorCheck_GitConfig_Fail_MissingKey(t *testing.T) {
	orig := gitExecCommand
	defer func() { gitExecCommand = orig }()

	gitExecCommand = func(name string, args ...string) *exec.Cmd {
		// All keys missing
		return exec.Command("sh", "-c", "exit 1")
	}

	result := checkGitConfig()
	if result.OK {
		t.Error("expected config check to fail when keys missing")
	}
	if !strings.Contains(result.Detail, "core.autocrlf") {
		t.Errorf("detail should mention missing key, got: %s", result.Detail)
	}
}

func TestDoctorCheck_RemoteURL_HTTPS_Pass(t *testing.T) {
	orig := gitExecCommand
	defer func() { gitExecCommand = orig }()

	gitExecCommand = func(name string, args ...string) *exec.Cmd {
		if contains(args, "get-url") {
			return exec.Command("sh", "-c", "printf 'https://github.com/org/repo'")
		}
		return exec.Command("sh", "-c", "printf 'origin'")
	}

	result := checkRemoteURL("origin")
	if !result.OK {
		t.Errorf("HTTPS URL should pass, got: %s", result.Detail)
	}
}

func TestDoctorCheck_RemoteURL_SSH_Fail(t *testing.T) {
	orig := gitExecCommand
	defer func() { gitExecCommand = orig }()

	gitExecCommand = func(name string, args ...string) *exec.Cmd {
		if contains(args, "get-url") {
			return exec.Command("sh", "-c", "printf 'git@github.com:org/repo.git'")
		}
		return exec.Command("sh", "-c", "printf 'origin'")
	}

	result := checkRemoteURL("origin")
	if result.OK {
		t.Error("SSH URL should fail")
	}
}

func TestRunDoctor_OutputContainsSummary(t *testing.T) {
	orig := gitExecCommand
	defer func() { gitExecCommand = orig }()

	// All checks pass
	gitExecCommand = func(name string, args ...string) *exec.Cmd {
		switch {
		case contains(args, "config"):
			key := args[len(args)-1]
			switch key {
			case "core.autocrlf":
				return exec.Command("sh", "-c", "printf 'false'")
			case "mendix.commits-since-gc":
				return exec.Command("sh", "-c", "printf '0'")
			case "mendix.lineEndingResetDone":
				return exec.Command("sh", "-c", "printf 'true'")
			}
		case contains(args, "get-url"):
			return exec.Command("sh", "-c", "printf 'https://github.com/org/repo'")
		case contains(args, "ls-remote"):
			return exec.Command("sh", "-c", "printf 'abc123\trefs/notes/mx_metadata'")
		case contains(args, "log"):
			return exec.Command("sh", "-c", "exit 0") // no commits = all have notes
		case contains(args, "list"):
			return exec.Command("sh", "-c", "exit 0")
		}
		return exec.Command("sh", "-c", "printf 'origin'")
	}

	var buf strings.Builder
	_ = runDoctor("origin", "", &buf)
	out := buf.String()
	if !strings.Contains(out, "Diagnosis:") {
		t.Errorf("output must contain Diagnosis summary, got:\n%s", out)
	}
}

func TestRunGitFix_SetsConfigKeys(t *testing.T) {
	orig := gitExecCommand
	defer func() { gitExecCommand = orig }()

	var setKeys []string
	gitExecCommand = func(name string, args ...string) *exec.Cmd {
		switch {
		case contains(args, "config") && contains(args, "--local") && len(args) == 4:
			// git config --local <key> <value>  (set): [config --local key value]
			setKeys = append(setKeys, args[2])
			return exec.Command("sh", "-c", "exit 0")
		case contains(args, "config") && contains(args, "--local") && len(args) == 3:
			// git config --local <key>  (get): [config --local key] — empty to force set
			return exec.Command("sh", "-c", "exit 1")
		case contains(args, "get-url"):
			return exec.Command("sh", "-c", "printf 'https://github.com/org/repo'")
		case contains(args, "log"):
			return exec.Command("sh", "-c", "exit 0")
		case contains(args, "list"):
			return exec.Command("sh", "-c", "exit 0")
		case contains(args, "push"):
			return exec.Command("sh", "-c", "exit 0")
		}
		return exec.Command("sh", "-c", "printf 'origin'")
	}

	var buf strings.Builder
	err := runGitFix("", "10.6.0.0", "origin", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantKeys := []string{"core.autocrlf", "mendix.commits-since-gc", "mendix.lineEndingResetDone"}
	for _, k := range wantKeys {
		if !contains(setKeys, k) {
			t.Errorf("fix must set config key %q, set keys: %v", k, setKeys)
		}
	}
}

func TestRunGitFix_ConvertsSSHToHTTPS(t *testing.T) {
	orig := gitExecCommand
	defer func() { gitExecCommand = orig }()

	var newURL string
	gitExecCommand = func(name string, args ...string) *exec.Cmd {
		switch {
		case contains(args, "config"):
			return exec.Command("sh", "-c", "printf 'false'") // already set
		case contains(args, "get-url"):
			return exec.Command("sh", "-c", "printf 'git@github.com:org/repo.git'")
		case contains(args, "set-url"):
			newURL = args[len(args)-1]
			return exec.Command("sh", "-c", "exit 0")
		case contains(args, "log"):
			return exec.Command("sh", "-c", "exit 0")
		case contains(args, "list"):
			return exec.Command("sh", "-c", "exit 0")
		case contains(args, "push"):
			return exec.Command("sh", "-c", "exit 0")
		}
		return exec.Command("sh", "-c", "printf 'origin'")
	}

	var buf strings.Builder
	_ = runGitFix("", "10.6.0.0", "origin", &buf)

	if !strings.HasPrefix(newURL, "https://") {
		t.Errorf("SSH URL must be converted to HTTPS, got: %q", newURL)
	}
}

// TestGitCommitIntegration runs a real git init + mxcli git commit flow
// using a temporary git repository. Requires git in PATH.
func TestGitCommitIntegration(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH, skipping integration test")
	}

	dir := t.TempDir()

	for _, cmd := range [][]string{
		{"git", "-C", dir, "init"},
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
	} {
		if out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("setup %v: %v\n%s", cmd, err, out)
		}
	}

	testFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", dir, "add", ".").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	var buf strings.Builder
	err := runGitCommit([]string{"-m", "Initial commit"}, "10.6.0.0", "", &buf)
	if err != nil {
		t.Fatalf("runGitCommit failed: %v\nOutput:\n%s", err, buf.String())
	}

	sha, _ := exec.Command("git", "rev-parse", "HEAD").Output()
	commitSHA := strings.TrimSpace(string(sha))

	noteOut, err := exec.Command("git", "notes", "--ref=mx_metadata", "show", commitSHA).Output()
	if err != nil {
		t.Fatalf("note not found on commit %s: %v", commitSHA[:7], err)
	}

	var m map[string]any
	if err := json.Unmarshal(noteOut, &m); err != nil {
		t.Fatalf("note is not valid JSON: %v\nNote content: %q", err, string(noteOut))
	}
	if m["ModelerVersion"] != "10.6.0.0" {
		t.Errorf("ModelerVersion = %v, want 10.6.0.0", m["ModelerVersion"])
	}
	if strings.HasSuffix(string(noteOut), "\n") {
		t.Error("note must not have trailing newline (libgit2 compatibility)")
	}
}

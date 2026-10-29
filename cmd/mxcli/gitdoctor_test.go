// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// `mxcli git doctor` audits a repository the way Studio Pro needs it, and
// `mxcli git commit` does what Studio Pro's commit does: hashes, staging,
// the mx_metadata note. Both are factored as doctorRun / commitRun so the
// tests drive them without the cobra wiring.

func doctorEnv(t *testing.T, dir string) {
	t.Helper()
	requireGit(t)
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	t.Setenv("GIT_AUTHOR_NAME", "t")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@example.com")
}

func configLocal(t *testing.T, dir, key string) string {
	t.Helper()
	out, err := gitOut(dir, "config", "--local", key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// --- doctor: git local config ----------------------------------------------

func TestGitDoctor_ConfigMissingReported(t *testing.T) {
	dir := t.TempDir()
	doctorEnv(t, dir)
	gitIn(t, dir, "init", "-q", "-b", "main")

	var out bytes.Buffer
	err := doctorRun(dir, "", "", "", false, &out)
	if !errors.Is(err, errDoctorIssuesRemain) {
		t.Fatalf("want errDoctorIssuesRemain, got %v", err)
	}
	for _, key := range []string{"core.autocrlf", "mendix.commits-since-gc", "mendix.lineEndingResetDone"} {
		if !strings.Contains(out.String(), key) {
			t.Errorf("report must name %s:\n%s", key, out.String())
		}
	}
}

func TestGitDoctor_FixWritesConfigAndPasses(t *testing.T) {	dir := t.TempDir()
	doctorEnv(t, dir)
	gitIn(t, dir, "init", "-q", "-b", "main")

	var out bytes.Buffer
	if err := doctorRun(dir, "", "", "", true, &out); err != nil {
		t.Fatalf("--fix must leave no issue: %v\n%s", err, out.String())
	}
	if got := configLocal(t, dir, "core.autocrlf"); !strings.EqualFold(got, "False") {
		t.Errorf("core.autocrlf = %q, want False", got)
	}
	if got := configLocal(t, dir, "mendix.commits-since-gc"); got != "0" {
		t.Errorf("mendix.commits-since-gc = %q, want 0", got)
	}
	if got := configLocal(t, dir, "mendix.lineEndingResetDone"); !strings.EqualFold(got, "True") {
		t.Errorf("mendix.lineEndingResetDone = %q, want True", got)
	}
	// A second run with nothing to fix stays clean.
	out.Reset()
	if err := doctorRun(dir, "", "", "", false, &out); err != nil {
		t.Fatalf("clean repo must pass: %v\n%s", err, out.String())
	}
	// A repo without a remote is reported, but not broken.
	if !strings.Contains(out.String(), "remote") {
		t.Errorf("report must mention the remote check:\n%s", out.String())
	}
}

// Measured on a real Studio Pro checkout (the mx-git-lab lab, S0): Studio Pro
// writes core.autocrlf=false in lowercase, counts commits-since-gc up between
// garbage collections (2 after two commits) and writes lineendingresetdone in
// lowercase. A healthy Studio Pro checkout must pass untouched.
func TestGitDoctor_StudioProValuesPass(t *testing.T) {
	dir := t.TempDir()
	doctorEnv(t, dir)
	gitIn(t, dir, "init", "-q", "-b", "main")
	gitIn(t, dir, "config", "--local", "core.autocrlf", "false")
	gitIn(t, dir, "config", "--local", "mendix.commits-since-gc", "2")
	gitIn(t, dir, "config", "--local", "mendix.lineendingresetdone", "true")

	var out bytes.Buffer
	if err := doctorRun(dir, "", "", "", false, &out); err != nil {
		t.Fatalf("a healthy Studio Pro checkout must pass: %v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), "mendix.commits-since-gc — ") {
		t.Errorf("the gc counter must not be judged:\n%s", out.String())
	}
}

// core.autocrlf=true is the one config value that damages a Mendix checkout:
// a checkout rewrites line endings inside .mxunit files on Windows.
func TestGitDoctor_AutoCRLFTrueIsFixed(t *testing.T) {
	dir := t.TempDir()
	doctorEnv(t, dir)
	gitIn(t, dir, "init", "-q", "-b", "main")
	gitIn(t, dir, "config", "--local", "core.autocrlf", "true")

	var out bytes.Buffer
	if err := doctorRun(dir, "", "", "", true, &out); err != nil {
		t.Fatalf("--fix must repair autocrlf=true: %v\n%s", err, out.String())
	}
	if got := configLocal(t, dir, "core.autocrlf"); !strings.EqualFold(got, "false") {
		t.Errorf("core.autocrlf = %q, want false", got)
	}
}

// --- doctor: remote URL scheme ----------------------------------------------

func TestGitDoctor_SSHRemoteRewrittenByFix(t *testing.T) {
	dir := t.TempDir()
	doctorEnv(t, dir)
	gitIn(t, dir, "init", "-q", "-b", "main")
	gitIn(t, dir, "remote", "add", "origin", "git@github.com:org/app.git")

	var out bytes.Buffer
	if err := doctorRun(dir, "", "", "", false, &out); !errors.Is(err, errDoctorIssuesRemain) {
		t.Fatalf("SSH remote must be an issue: %v", err)
	}
	if !strings.Contains(out.String(), "git@github.com:org/app.git") {
		t.Errorf("report must name the URL:\n%s", out.String())
	}

	out.Reset()
	if err := doctorRun(dir, "", "", "", true, &out); err != nil {
		t.Fatalf("--fix must repair the remote: %v\n%s", err, out.String())
	}
	got, err := gitOut(dir, "remote", "get-url", "origin")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != "https://github.com/org/app" {
		t.Errorf("remote url = %q, want https://github.com/org/app", got)
	}
}

func TestGitDoctor_SSHURLFormRewrittenByFix(t *testing.T) {
	dir := t.TempDir()
	doctorEnv(t, dir)
	gitIn(t, dir, "init", "-q", "-b", "main")
	gitIn(t, dir, "remote", "add", "origin", "ssh://git@github.com/org/app.git")

	var out bytes.Buffer
	if err := doctorRun(dir, "", "", "", true, &out); err != nil {
		t.Fatalf("--fix must repair ssh:// remotes: %v\n%s", err, out.String())
	}
	got, err := gitOut(dir, "remote", "get-url", "origin")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != "https://github.com/org/app" {
		t.Errorf("remote url = %q, want https://github.com/org/app", got)
	}
}

// --- doctor: mx_metadata notes ----------------------------------------------

// doctorNoteRepo makes a two-commit repository holding an MPR v2 project:
// the base commit, then a page retitle. Returns dir and the .mpr path.
func doctorNoteRepo(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q", "-b", "feature")
	units := baseUnits(t)
	writeProject(t, dir, "11.13.0", units)
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "base")
	units[tPage] = tUnit{tFolder, unitDoc(t, "Forms$Page", "Home", bson.E{Key: "Title", Value: "Welcome"})}
	writeProject(t, dir, "11.13.0", units)
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "retitle home")
	return dir, filepath.Join(dir, "App.mpr")
}

func noteOn(t *testing.T, dir, rev string) string {
	t.Helper()
	out, err := gitOut(dir, "notes", "--ref="+mxMetadataRef, "show", rev)
	if err != nil {
		return ""
	}
	return out
}

func TestGitDoctor_MissingAndMalformedNotesReportedAndFixed(t *testing.T) {
	dir, mprPath := doctorNoteRepo(t)
	doctorEnv(t, dir)
	gitIn(t, dir, "notes", "--ref="+mxMetadataRef, "add", "-f", "-m", "not json", "HEAD~1")

	var out bytes.Buffer
	if err := doctorRun(dir, mprPath, "", "", false, &out); !errors.Is(err, errDoctorIssuesRemain) {
		t.Fatalf("missing and malformed notes must be issues: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "malformed") || !strings.Contains(out.String(), "without a note") {
		t.Errorf("report must name both findings:\n%s", out.String())
	}

	out.Reset()
	if err := doctorRun(dir, mprPath, "", "", true, &out); err != nil {
		t.Fatalf("--fix must repair the notes: %v\n%s", err, out.String())
	}
	var head, base mxNote
	if err := json.Unmarshal([]byte(noteOn(t, dir, "HEAD")), &head); err != nil {
		t.Fatalf("HEAD note is not valid JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(noteOn(t, dir, "HEAD~1")), &base); err != nil {
		t.Fatalf("HEAD~1 note (was malformed) is not valid JSON: %v", err)
	}
	if !head.HasModelerVersion || head.ModelerVersion != "11.13.0" {
		t.Errorf("HEAD note header: %+v", head)
	}
	if len(head.ModelChanges) != 1 || head.ModelChanges[0].UnitID != tPage {
		t.Errorf("HEAD note must list the retitled page: %+v", head.ModelChanges)
	}
	if len(base.ModelChanges) == 0 {
		t.Errorf("base commit note must list the added units: %+v", base)
	}
}

func TestGitDoctor_FixKeepsRealNotes(t *testing.T) {
	dir, mprPath := doctorNoteRepo(t)
	doctorEnv(t, dir)
	gitIn(t, dir, "notes", "--ref="+mxMetadataRef, "add", "-f", "-m", realStudioProNote, "HEAD")

	var out bytes.Buffer
	doctorRun(dir, mprPath, "", "", true, &out) // config findings remain; the note must not
	if got := strings.TrimSpace(noteOn(t, dir, "HEAD")); got != realStudioProNote {
		t.Fatalf("real note was changed:\n got %s\nwant %s", got, realStudioProNote)
	}
}

func TestGitDoctor_TemplateWithoutProjectUsesVersionFlag(t *testing.T) {
	dir := t.TempDir()
	doctorEnv(t, dir)
	gitIn(t, dir, "init", "-q", "-b", "main")
	gitIn(t, dir, "commit", "-q", "--allow-empty", "-m", "docs")

	// Without --version there is nothing to derive a ModelerVersion from.
	var out bytes.Buffer
	err := doctorRun(dir, "", "", "", true, &out)
	if err == nil || !strings.Contains(err.Error(), "--version") {
		t.Fatalf("want an error naming --version, got %v", err)
	}

	out.Reset()
	if err := doctorRun(dir, "", "", "10.6.0", true, &out); err != nil {
		t.Fatalf("--fix --version must fill the notes: %v\n%s", err, out.String())
	}
	var n mxNote
	if err := json.Unmarshal([]byte(noteOn(t, dir, "HEAD")), &n); err != nil {
		t.Fatal(err)
	}
	if n.ModelerVersion != "10.6.0" || n.BranchName != "" || !n.HasModelerVersion || len(n.ModelChanges) != 0 {
		t.Errorf("template note: %+v", n)
	}
}

func TestGitDoctor_TemplateDetectsVersionFromHistory(t *testing.T) {
	dir, _ := doctorNoteRepo(t)
	doctorEnv(t, dir)
	gitIn(t, dir, "notes", "--ref="+mxMetadataRef, "add", "-f", "-m", realStudioProNote, "HEAD~1")

	// No -p: the note cannot be computed from the model, so the template is
	// written with the version detected from the newest valid note.
	var out bytes.Buffer
	if err := doctorRun(dir, "", "", "", true, &out); err != nil {
		t.Fatalf("--fix: %v\n%s", err, out.String())
	}
	var n mxNote
	if err := json.Unmarshal([]byte(noteOn(t, dir, "HEAD")), &n); err != nil {
		t.Fatal(err)
	}
	if n.ModelerVersion != "11.13.0" {
		t.Errorf("template ModelerVersion = %q, want 11.13.0 from the newest valid note", n.ModelerVersion)
	}
}

// --- git commit ---------------------------------------------------------------

func commitSubject(t *testing.T, dir string) string {
	t.Helper()
	out, err := gitOut(dir, "show", "-s", "--format=%s", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(out)
}

func committedFiles(t *testing.T, dir string) string {
	t.Helper()
	out, err := gitOut(dir, "show", "--name-only", "--format=", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestGitCommit_HappyPathWritesNoteAndSkipsJunk(t *testing.T) {
	dir, mprPath := doctorNoteRepo(t)
	doctorEnv(t, dir)
	// A real source change, plus one junk file that must never be committed.
	units := baseUnits(t)
	units[tPage] = tUnit{tFolder, unitDoc(t, "Forms$Page", "Home", bson.E{Key: "Title", Value: "Again"})}
	writeProject(t, dir, "11.13.0", units)
	if err := os.WriteFile(filepath.Join(dir, "project-settings.user.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := commitRun(mprPath, "retitle again", false, false, &out); err != nil {
		t.Fatalf("commitRun: %v\n%s", err, out.String())
	}
	if subj := commitSubject(t, dir); subj != "retitle again" {
		t.Errorf("HEAD subject = %q", subj)
	}
	files := committedFiles(t, dir)
	if strings.Contains(files, "project-settings.user.json") {
		t.Errorf("user settings must not be committed:\n%s", files)
	}
	var n mxNote
	if err := json.Unmarshal([]byte(noteOn(t, dir, "HEAD")), &n); err != nil {
		t.Fatalf("HEAD has no valid note: %v", err)
	}
	if n.BranchName != "feature" || !n.HasModelerVersion {
		t.Errorf("note header: %+v", n)
	}
	// The junk file is still on disk, untracked.
	if _, err := os.Stat(filepath.Join(dir, "project-settings.user.json")); err != nil {
		t.Error("junk file was deleted")
	}
	if !strings.Contains(out.String(), "git push origin feature "+mxMetadataRef) {
		t.Errorf("must tell how to push branch and notes:\n%s", out.String())
	}
}

func TestGitCommit_RepairsHashDrift(t *testing.T) {
	dir, mprPath := doctorNoteRepo(t)
	doctorEnv(t, dir)
	// Rewrite a unit file behind the index's back: the classic git checkout drift.
	page := filepath.Join(dir, "mprcontents", tPage[:2], tPage[2:4], tPage+".mxunit")
	raw, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw[:len(raw)-2], raw[len(raw)-2]+1, raw[len(raw)-1])
	if err := os.WriteFile(page, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := commitRun(mprPath, "drift", false, false, &out); err != nil {
		t.Fatalf("commitRun must repair mismatched hashes and commit: %v\n%s", err, out.String())
	}
	var out2 bytes.Buffer
	if err := fixHashes(mprPath, false, &out2); err != nil {
		t.Errorf("hashes still drift after commit: %v\n%s", err, out2.String())
	}
	if subj := commitSubject(t, dir); subj != "drift" {
		t.Errorf("HEAD subject = %q", subj)
	}
}

func TestGitCommit_RefusedWhileStudioProOpen(t *testing.T) {
	dir, mprPath := doctorNoteRepo(t)
	doctorEnv(t, dir)
	lock := mprPath + ".lock"
	if err := os.WriteFile(lock, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := commitRun(mprPath, "nope", false, false, &out); err == nil {
		t.Fatal("commit must be refused while the lock file exists")
	}
	if subj := commitSubject(t, dir); subj != "retitle home" {
		t.Errorf("a commit was made anyway: %q", subj)
	}
}

func TestGitCommit_DryRunCommitsNothing(t *testing.T) {
	dir, mprPath := doctorNoteRepo(t)
	doctorEnv(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, "theme"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "theme", "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := commitSubject(t, dir)

	var out bytes.Buffer
	if err := commitRun(mprPath, "dry", false, true, &out); err != nil {
		t.Fatalf("dry run failed: %v\n%s", err, out.String())
	}
	if got := commitSubject(t, dir); got != before {
		t.Errorf("dry run committed: %q", got)
	}
	staged, err := gitOut(dir, "diff", "--cached", "--name-only")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(staged) != "" {
		t.Errorf("dry run left changes staged:\n%s", staged)
	}
}

func TestGitCommit_PushesBranchAndNotes(t *testing.T) {
	dir, mprPath := doctorNoteRepo(t)
	doctorEnv(t, dir)
	units := baseUnits(t)
	units[tPage] = tUnit{tFolder, unitDoc(t, "Forms$Page", "Home", bson.E{Key: "Title", Value: "Ship"})}
	writeProject(t, dir, "11.13.0", units)
	remote := t.TempDir()
	gitIn(t, remote, "init", "-q", "--bare")
	gitIn(t, dir, "remote", "add", "origin", remote)

	var out bytes.Buffer
	if err := commitRun(mprPath, "push me", true, false, &out); err != nil {
		t.Fatalf("commitRun --push: %v\n%s", err, out.String())
	}
	ls, err := gitOut(dir, "ls-remote", "origin")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ls, "refs/heads/feature") {
		t.Errorf("branch was not pushed:\n%s", ls)
	}
	if !strings.Contains(ls, mxMetadataRef) {
		t.Errorf("notes ref was not pushed:\n%s", ls)
	}
}

func TestGitCommit_NothingToCommit(t *testing.T) {
	dir, mprPath := doctorNoteRepo(t)
	doctorEnv(t, dir)

	var out bytes.Buffer
	if err := commitRun(mprPath, "empty", false, false, &out); err != nil {
		t.Fatalf("nothing to commit must not be an error: %v", err)
	}
	if !strings.Contains(out.String(), "nothing to commit") {
		t.Errorf("must say so:\n%s", out.String())
	}
}

// gitInE runs git and lets it fail, for the commands whose failure is the
// fixture (a merge that conflicts).
func gitInE(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	b, err := cmd.CombinedOutput()
	return string(b), err
}

// A staged path with spaces must reach the commit whole, and junk under a
// spaced path must still be left out.
func TestGitCommit_SpacedPathsStagedWholeAndJunkLeft(t *testing.T) {
	dir, mprPath := doctorNoteRepo(t)
	doctorEnv(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, "my docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "my docs", "read me.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "deployment"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "deployment", "old log.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := commitRun(mprPath, "spacey", false, false, &out); err != nil {
		t.Fatalf("commitRun: %v\n%s", err, out.String())
	}
	files := committedFiles(t, dir)
	if !strings.Contains(files, "my docs/read me.txt") {
		t.Errorf("spaced path not committed whole:\n%s", files)
	}
	if strings.Contains(files, "deployment/old log.txt") {
		t.Errorf("junk was committed:\n%s", files)
	}
}

// An unresolved merge must refuse the commit: `git add -A` would otherwise
// stage the conflict markers and commit them.
func TestGitCommit_RefusedDuringUnresolvedMerge(t *testing.T) {
	dir, mprPath := doctorNoteRepo(t)
	doctorEnv(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("feature side\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", "notes.txt")
	gitIn(t, dir, "commit", "-q", "-m", "notes feature")
	gitIn(t, dir, "switch", "-q", "-c", "other", "HEAD~1")
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("other side\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", "notes.txt")
	gitIn(t, dir, "commit", "-q", "-m", "notes other")
	gitIn(t, dir, "switch", "-q", "feature")
	if _, err := gitInE(t, dir, "merge", "other"); err == nil {
		t.Fatal("fixture: the merge was expected to conflict")
	}

	var out bytes.Buffer
	if err := commitRun(mprPath, "conflicted", false, false, &out); err == nil {
		t.Fatal("commit must be refused during an unresolved merge")
	}
	if subj := commitSubject(t, dir); subj != "notes feature" {
		t.Errorf("a commit was made anyway: %q", subj)
	}
}

// The lock guard honours the same override every other mxcli write honours,
// for the crash-left-a-lock case.
func TestGitCommit_LockOverrideEnvCommits(t *testing.T) {
	dir, mprPath := doctorNoteRepo(t)
	doctorEnv(t, dir)
	if err := os.WriteFile(mprPath+".lock", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	units := baseUnits(t)
	units[tPage] = tUnit{tFolder, unitDoc(t, "Forms$Page", "Home", bson.E{Key: "Title", Value: "Crashed"})}
	writeProject(t, dir, "11.13.0", units)
	t.Setenv("MXCLI_ALLOW_STUDIO_PRO_OPEN", "1")

	var out bytes.Buffer
	if err := commitRun(mprPath, "override", false, false, &out); err != nil {
		t.Fatalf("commitRun with the override env: %v\n%s", err, out.String())
	}
	if subj := commitSubject(t, dir); subj != "override" {
		t.Errorf("HEAD subject = %q", subj)
	}
}

// A dry run writes nothing — not even the ContentsHash repair.
func TestGitCommit_DryRunLeavesHashDriftUntouched(t *testing.T) {
	dir, mprPath := doctorNoteRepo(t)
	doctorEnv(t, dir)
	page := filepath.Join(dir, "mprcontents", tPage[:2], tPage[2:4], tPage+".mxunit")
	raw, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw[:len(raw)-2], raw[len(raw)-2]+1, raw[len(raw)-1])
	if err := os.WriteFile(page, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := commitRun(mprPath, "dry over drift", false, true, &out); err != nil {
		t.Fatalf("dry run failed: %v\n%s", err, out.String())
	}
	var out2 bytes.Buffer
	if err := fixHashes(mprPath, false, &out2); !errors.Is(err, errHashIssuesRemain) {
		t.Errorf("the dry run repaired the hashes (err %v):\n%s", err, out2.String())
	}
}

// The staged-files summary must list what the commit will hold, not the junk
// that was just unstaged (measured on the mx-git-lab lab: the list opened with
// 10 .mxcli/ paths while the commit held 2 model files).
func TestGitCommit_StagedListOmitsJunk(t *testing.T) {
	dir, mprPath := doctorNoteRepo(t)
	doctorEnv(t, dir)
	units := baseUnits(t)
	units[tPage] = tUnit{tFolder, unitDoc(t, "Forms$Page", "Home", bson.E{Key: "Title", Value: "List"})}
	writeProject(t, dir, "11.13.0", units)
	if err := os.WriteFile(filepath.Join(dir, "project-settings.user.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := commitRun(mprPath, "clean list", false, false, &out); err != nil {
		t.Fatalf("commitRun: %v\n%s", err, out.String())
	}
	section := out.String()[strings.Index(out.String(), "Staged"):]
	if strings.Contains(section, "project-settings.user.json") {
		t.Errorf("the staged list names junk that was left out:\n%s", section)
	}
	if !strings.Contains(section, "App.mpr") {
		t.Errorf("the staged list must name the model files:\n%s", section)
	}
}

func TestGitCommit_DryRunWithPushRefused(t *testing.T) {	dir, mprPath := doctorNoteRepo(t)
	doctorEnv(t, dir)

	var out bytes.Buffer
	if err := commitRun(mprPath, "both", true, true, &out); err == nil {
		t.Fatal("--dry-run with --push must be refused")
	}
}

// A misspelled -p must be refused, not turned into a "real" note with an
// empty ModelerVersion that no later run would ever touch again.
func TestGitDoctor_BadProjectPathRefused(t *testing.T) {
	dir, mprPath := doctorNoteRepo(t)
	doctorEnv(t, dir)

	var out bytes.Buffer
	err := doctorRun(dir, mprPath+"-typo", "", "10.6.0", true, &out)
	if err == nil || !strings.Contains(err.Error(), "project not found") {
		t.Fatalf("want a project-not-found error, got %v", err)
	}
	if note := noteOn(t, dir, "HEAD"); strings.Contains(note, `"ModelerVersion":""`) {
		t.Fatalf("a note with an empty version was written:\n%s", note)
	}
}

func TestGitDoctor_VersionFlagValidated(t *testing.T) {
	dir := t.TempDir()
	doctorEnv(t, dir)
	gitIn(t, dir, "init", "-q", "-b", "main")
	gitIn(t, dir, "commit", "-q", "--allow-empty", "-m", "docs")

	var out bytes.Buffer
	err := doctorRun(dir, "", "", "banana", true, &out)
	if err == nil || !strings.Contains(err.Error(), "--version") {
		t.Fatalf("want a --version format error, got %v", err)
	}
}

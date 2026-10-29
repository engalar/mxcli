// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func bsonKeyValue(key, value string) bson.E {
	return bson.E{Key: key, Value: value}
}

// `mxcli git merge <ref>` — merge a branch with model-aware resolution: git
// merge first, and on a model-file conflict `mx git-merge` (the same engine
// Studio Pro uses). The mx run is injected as runMx so the orchestration is
// testable without a Mendix installation; the real runner shells out to mx.

// modelConflictRepo makes a repository whose merge of "side" into "feature"
// conflicts on both model files: each side changed the same DomainModel unit.
func modelConflictRepo(t *testing.T) (string, string) {
	t.Helper()
	dir, mprPath := doctorNoteRepo(t)
	units := baseUnits(t)
	units[tDM] = tUnit{tMod, unitDoc(t, "DomainModels$DomainModel", "", mainSideKey)}
	writeProject(t, dir, "11.13.0", units)
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "main side")
	gitIn(t, dir, "switch", "-q", "-c", "side", "HEAD~1")
	side := baseUnits(t)
	side[tDM] = tUnit{tMod, unitDoc(t, "DomainModels$DomainModel", "", sideKey)}
	writeProject(t, dir, "11.13.0", side)
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "side change")
	gitIn(t, dir, "switch", "-q", "feature")
	_, err := gitInE(t, dir, "merge", "side")
	if err != nil && !strings.Contains(err.Error(), "exit status") {
		t.Fatalf("fixture: expected a conflicted merge, got %v", err)
	}
	if uu := gitOutT(t, dir, "diff", "--name-only", "--diff-filter=U"); !strings.Contains(uu, ".mxunit") {
		t.Fatalf("fixture: expected unresolved model files, got %q", uu)
	}
	return dir, mprPath
}

var (
	mainSideKey = bsonKeyValue("DocA", "main")
	sideKey     = bsonKeyValue("DocB", "side")
)

func okRunner(out string) mxMergeRunner {
	return func(base, mine, theirs string, shas [3]string) (int, string, error) {
		return 0, out, nil
	}
}

func TestGitMerge_CleanMergeTextFileWritesNote(t *testing.T) {
	dir, mprPath := doctorNoteRepo(t)
	doctorEnv(t, dir)
	gitIn(t, dir, "branch", "docs", "HEAD~1")
	gitIn(t, dir, "switch", "-q", "docs")
	if err := os.WriteFile(filepath.Join(dir, "NOTES.txt"), []byte("docs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", "NOTES.txt")
	gitIn(t, dir, "commit", "-q", "-m", "docs side")
	gitIn(t, dir, "switch", "-q", "feature")

	noMx := func(base, mine, theirs string, shas [3]string) (int, string, error) {
		t.Fatal("mx must not run on a merge git resolves itself")
		return 0, "", nil
	}
	var out bytes.Buffer
	if err := mergeRun(mprPath, "docs", "", "", false, noMx, &out); err != nil {
		t.Fatalf("mergeRun: %v\n%s", err, out.String())
	}
	parents := strings.Fields(gitOutT(t, dir, "rev-list", "--parents", "-n", "1", "HEAD"))
	if len(parents) != 3 {
		t.Fatalf("want a two-parent merge commit, got %v", parents)
	}
	var n mxNote
	if err := json.Unmarshal([]byte(noteOn(t, dir, "HEAD")), &n); err != nil {
		t.Fatalf("merge commit has no valid note: %v", err)
	}
	if !n.HasModelerVersion || n.BranchName != "feature" {
		t.Errorf("note header: %+v", n)
	}
}

func TestGitMerge_ModelConflictAutoResolvedByMx(t *testing.T) {
	dir, mprPath := modelConflictRepo(t)
	doctorEnv(t, dir)

	var gotBase, gotMine, gotTheirs string
	var gotShas [3]string
	runner := func(base, mine, theirs string, shas [3]string) (int, string, error) {
		gotBase, gotMine, gotTheirs, gotShas = base, mine, theirs, shas
		return 0, "Merge successful.", nil
	}
	var out bytes.Buffer
	if err := mergeRun(mprPath, "side", "", "", false, runner, &out); err != nil {
		t.Fatalf("mergeRun: %v\n%s", err, out.String())
	}
	if !strings.Contains(gotBase, "mx-git-lab") && !strings.Contains(filepath.Base(gotBase), ".mpr") {
		// the base path must be an .mpr inside a temporary checkout
		if filepath.Ext(gotBase) != ".mpr" {
			t.Errorf("runner got base %q, want the base checkout's .mpr", gotBase)
		}
	}
	if filepath.Base(gotMine) != "App.mpr" {
		t.Errorf("runner's mine = %q, want the working .mpr", gotMine)
	}
	if filepath.Base(gotTheirs) != "App.mpr" {
		t.Errorf("runner's theirs = %q, want the theirs checkout's .mpr", gotTheirs)
	}
	if gotShas[0] == "" || gotShas[1] == gotShas[0] || gotShas[2] == gotShas[1] {
		t.Errorf("shas must be base/ours/theirs distinct: %v", gotShas)
	}
	parents := strings.Fields(gitOutT(t, dir, "rev-list", "--parents", "-n", "1", "HEAD"))
	if len(parents) != 3 {
		t.Fatalf("want a two-parent merge commit, got %v", parents)
	}
	uu := gitOutT(t, dir, "diff", "--name-only", "--diff-filter=U")
	if strings.TrimSpace(uu) != "" {
		t.Errorf("conflicts still unresolved:\n%s", uu)
	}
	if !strings.Contains(noteOn(t, dir, "HEAD"), `"HasModelerVersion":true`) {
		t.Errorf("merge commit has no note:\n%s", noteOn(t, dir, "HEAD"))
	}
}

func TestGitMerge_ModelConflictUnresolvedAborts(t *testing.T) {
	dir, mprPath := modelConflictRepo(t)
	doctorEnv(t, dir)
	before := commitSubject(t, dir)

	fails := func(base, mine, theirs string, shas [3]string) (int, string, error) {
		return 2, `[{"ConflictResolutionDetails":"ManualResolutionRequired"}]`, nil
	}
	var out bytes.Buffer
	err := mergeRun(mprPath, "side", "", "", false, fails, &out)
	if err == nil || !strings.Contains(err.Error(), "Studio Pro") {
		t.Fatalf("want a Studio Pro referral, got %v", err)
	}
	if got := commitSubject(t, dir); got != before {
		t.Errorf("HEAD moved despite the abort: %q", got)
	}
	if _, err := gitOut(dir, "rev-parse", "--verify", "--quiet", "MERGE_HEAD"); err == nil {
		t.Error("MERGE_HEAD still present: the merge was not aborted")
	}
	uu := gitOutT(t, dir, "diff", "--name-only", "--diff-filter=U")
	if strings.TrimSpace(uu) != "" {
		t.Errorf("conflicted files remain:\n%s", uu)
	}
}

func TestGitMerge_NonModelConflictLeftAlone(t *testing.T) {
	dir, mprPath := doctorNoteRepo(t)
	doctorEnv(t, dir)
	gitIn(t, dir, "checkout", "-q", "-b", "side", "HEAD~1")
	if err := os.WriteFile(filepath.Join(dir, "NOTES.txt"), []byte("side\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", "NOTES.txt")
	gitIn(t, dir, "commit", "-q", "-m", "side notes")
	gitIn(t, dir, "checkout", "-q", "feature")
	if err := os.WriteFile(filepath.Join(dir, "NOTES.txt"), []byte("main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", "NOTES.txt")
	gitIn(t, dir, "commit", "-q", "-m", "main notes")
	_, err := gitInE(t, dir, "merge", "side")
	if err != nil && !strings.Contains(err.Error(), "exit status") {
		t.Fatalf("fixture: expected a conflict, got %v", err)
	}

	calls := 0
	counting := func(base, mine, theirs string, shas [3]string) (int, string, error) {
		calls++
		return 0, "", nil
	}
	var out bytes.Buffer
	merr := mergeRun(mprPath, "side", "", "", false, counting, &out)
	if merr == nil {
		t.Fatal("a non-model conflict must be reported, not silently passed")
	}
	if calls != 0 {
		t.Errorf("mx must not run for non-model conflicts (ran %d times)", calls)
	}
	uu := gitOutT(t, dir, "diff", "--name-only", "--diff-filter=U")
	if !strings.Contains(uu, "NOTES.txt") {
		t.Errorf("the conflicted text file must be left for manual resolution:\n%s", uu)
	}
	gitInE(t, dir, "merge", "--abort") // clean up the fixture
}

func TestGitMerge_AlreadyUpToDate(t *testing.T) {
	dir, mprPath := doctorNoteRepo(t)
	doctorEnv(t, dir)
	before := commitSubject(t, dir)

	noMx := func(base, mine, theirs string, shas [3]string) (int, string, error) {
		t.Fatal("mx must not run")
		return 0, "", nil
	}
	var out bytes.Buffer
	if err := mergeRun(mprPath, "HEAD~1", "", "", false, noMx, &out); err != nil {
		t.Fatalf("mergeRun: %v", err)
	}
	if got := commitSubject(t, dir); got != before {
		t.Errorf("HEAD moved: %q", got)
	}
	if !strings.Contains(out.String(), "up to date") {
		t.Errorf("must say so:\n%s", out.String())
	}
}

// The conflict report is the review material: on ManualResolutionRequired the
// merge is aborted and both sides' changes are printed, so a human can decide
// before anything lands. mx is not even needed for the report.
func TestGitMerge_ConflictReportShowsBothSidesAndAborts(t *testing.T) {
	dir, mprPath := modelConflictRepo(t)
	doctorEnv(t, dir)
	before := commitSubject(t, dir)

	noMx := func(base, mine, theirs string, shas [3]string) (int, string, error) {
		return 2, `[{"ConflictResolutionDetails":"ManualResolutionRequired"}]`, nil
	}
	var out bytes.Buffer
	err := mergeRun(mprPath, "side", "", "", false, noMx, &out)
	if err == nil || !strings.Contains(err.Error(), "--ours") {
		t.Fatalf("the report must point at the resolution flags, got %v", err)
	}
	report := out.String()
	for _, want := range []string{"DomainModels$DomainModel", "also changed on the other side", "Yours", "Theirs"} {
		if !strings.Contains(report, want) {
			t.Errorf("report must contain %q:\n%s", want, report)
		}
	}
	if got := commitSubject(t, dir); got != before {
		t.Errorf("HEAD moved: %q", got)
	}
	if _, err := gitOut(dir, "rev-parse", "--verify", "--quiet", "MERGE_HEAD"); err == nil {
		t.Error("MERGE_HEAD still present")
	}
	if uu := gitOutT(t, dir, "diff", "--name-only", "--diff-filter=U"); strings.TrimSpace(uu) != "" {
		t.Errorf("working tree not restored:\n%s", uu)
	}
}

// --ours / --theirs take the decided side for every conflicted model file,
// commit the merge and write the note — deterministically, without mx.
func TestGitMerge_SideTakingResolves(t *testing.T) {
	dir, mprPath := modelConflictRepo(t)
	doctorEnv(t, dir)

	noMx := func(base, mine, theirs string, shas [3]string) (int, string, error) {
		t.Fatal("--ours must not consult mx")
		return 0, "", nil
	}
	var out bytes.Buffer
	if err := mergeRun(mprPath, "side", "", "ours", false, noMx, &out); err != nil {
		t.Fatalf("mergeRun --ours: %v\n%s", err, out.String())
	}
	parents := strings.Fields(gitOutT(t, dir, "rev-list", "--parents", "-n", "1", "HEAD"))
	if len(parents) != 3 {
		t.Fatalf("want a two-parent merge commit, got %v", parents)
	}
	if !strings.Contains(noteOn(t, dir, "HEAD"), `"HasModelerVersion":true`) {
		t.Errorf("merge commit has no note")
	}
	page := filepath.Join(dir, "mprcontents", tDM[:2], tDM[2:4], tDM+".mxunit")
	raw, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("DocA")) || bytes.Contains(raw, []byte("DocB")) {
		t.Error("--ours must leave our side of the unit on disk")
	}

	// --theirs flips to the other side, deterministically.
	dir2, mprPath2 := modelConflictRepo(t)
	theirs := func(base, mine, theirs string, shas [3]string) (int, string, error) {
		t.Fatal("--theirs must not consult mx")
		return 0, "", nil
	}
	var out2 bytes.Buffer
	if err := mergeRun(mprPath2, "side", "", "theirs", false, theirs, &out2); err != nil {
		t.Fatalf("mergeRun --theirs: %v\n%s", err, out2.String())
	}
	page2 := filepath.Join(dir2, "mprcontents", tDM[:2], tDM[2:4], tDM+".mxunit")
	raw2, err := os.ReadFile(page2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw2, []byte("DocB")) || bytes.Contains(raw2, []byte("DocA")) {
		t.Error("--theirs must leave their side of the unit on disk")
	}
}

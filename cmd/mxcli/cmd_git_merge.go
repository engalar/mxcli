// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
	"github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/spf13/cobra"
)

// cmd_git_merge.go: `mxcli git merge <ref>` — merge a branch the way Studio
// Pro does. `git merge` runs first; when it stops on the model files (the
// .mpr is SQLite and every .mxunit is BSON, so any two-sided model change is
// a binary conflict), the conflict is handed to `mx git-merge` — the same
// model-aware engine Studio Pro's own merge uses — with the three revisions
// checked out into temporary worktrees. What mx resolves is committed as a
// normal two-parent merge commit with the mx_metadata note; what it reports
// as ManualResolutionRequired aborts the merge with the conflict list and a
// pointer to Studio Pro, leaving nothing behind.

var gitMergeCmd = &cobra.Command{
	Use:   "merge <ref>",
	Short: "Merge a branch with model-aware conflict resolution (mx git-merge)",
	Long: `Merge <ref> into the current branch, resolving model conflicts the way
Studio Pro does.

'git merge <ref>' runs first. A merge git resolves itself (disjoint files, a
fast-forward) is committed and annotated like 'mxcli git commit' does. When
git stops on the model files — the .mpr and the .mxunit units are binary, so
two sides changing the model always conflict — the three revisions are
checked out into temporary worktrees and 'mx git-merge BASE MINE THEIRS'
(Studio Pro's own model-aware merge, resolved for the project's Mendix
version) decides:

  - everything mx auto-resolves is staged and committed as the merge commit,
    with the mx_metadata note attached (this keeps both sides — the usual
    best outcome);
  - a conflict mx reports as ManualResolutionRequired is never merged
    silently: the merge is aborted, and both sides' changes are printed for
    review — at property level, from 'mx diff' (e.g.
    "New type > Length: 100 → 150" on ours, "→ 200" on theirs). Then decide:
      --plan <file>  resolve per unit from the plan template written beside
                     the .mpr (<mpr>.merge-plan.json): set "take" to
                     "ours"/"theirs" per unit; the .mpr index is repaired
                     from the mixed files. Modified-on-both units only —
                     added/deleted/moved conflicts go to Studio Pro.
      --ours / --theirs   take one side of every conflicted model file
                     (deterministic; mx is not consulted again)
      Studio Pro (Version Control › Merge Changes Here) for mixed
                     within-unit resolutions — still the tool when a decision
                     must mix both sides inside one unit;
  - a conflict in non-model files (text sources) is left in place for manual
    resolution, like plain git.

The merge is refused while Studio Pro has the project open (.mpr.lock).
MPR v2 projects only. The mx binary is resolved for the project's Mendix
version (PATH, the mxcli cache, or the Studio Pro installation).`,
	Example: `  mxcli git merge feature/orders -p app.mpr
  mxcli git merge origin/main -p app.mpr --push
  mxcli git merge feature/orders -p app.mpr --theirs   # after reviewing the report`,
	Args:          cobra.ExactArgs(1),
	RunE:          runGitMerge,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	gitMergeCmd.Flags().StringP("project", "p", "", "path to the Mendix project (.mpr)")
	_ = gitMergeCmd.MarkFlagRequired("project")
	gitMergeCmd.Flags().StringP("message", "m", "", "merge commit message (default: git's)")
	gitMergeCmd.Flags().Bool("ours", false, "on a model conflict mx cannot resolve, take our side of every conflicted model file")
	gitMergeCmd.Flags().Bool("theirs", false, "on a model conflict mx cannot resolve, take their side of every conflicted model file")
	gitMergeCmd.Flags().String("plan", "", "resolve per unit from a plan file written by a previous run (<mpr>.merge-plan.json)")
	gitMergeCmd.Flags().Bool("push", false, "push the branch and the notes ref to their remote afterwards")
	gitCmd.AddCommand(gitMergeCmd)
}

func runGitMerge(cmd *cobra.Command, args []string) error {
	ref := args[0]
	message, _ := cmd.Flags().GetString("message")
	ours, _ := cmd.Flags().GetBool("ours")
	theirs, _ := cmd.Flags().GetBool("theirs")
	plan, _ := cmd.Flags().GetString("plan")
	push, _ := cmd.Flags().GetBool("push")
	project, _ := cmd.Flags().GetString("project")
	strategy := ""
	switch {
	case ours && theirs:
		return errors.New("--ours and --theirs cannot be combined")
	case ours && plan != "", theirs && plan != "":
		return errors.New("--plan cannot be combined with --ours/--theirs: the plan is the finer decision")
	case ours:
		strategy = "ours"
	case theirs:
		strategy = "theirs"
	}
	return mergeRun(project, ref, message, strategy, plan, push, mxRunners{merge: realMxGitMerge, diff: realMxDiff}, cmd.OutOrStdout())
}

// mxRunners are the mx invocations, injected for tests.
type mxMergeRunner func(base, mine, theirs string, shas [3]string) (int, string, error)
type mxDiffRunner func(base, mine, outJSON string) error

type mxRunners struct {
	merge mxMergeRunner
	diff  mxDiffRunner
}

// mergeRun is the command without its wiring. runMx is injected for tests.
// mergeRun is the command without its wiring. runMx is injected for tests.
//
// Conflict handling is a three-step flow a human stays in charge of:
//
//  1. `git merge` runs; a merge it resolves itself is annotated and done.
//  2. A model conflict goes to `mx git-merge` first — what it auto-resolves
//     keeps both sides, which is almost always the best outcome — and is
//     committed with the note.
//  3. What mx reports as ManualResolutionRequired is never merged silently:
//     with no strategy the merge is aborted and both sides' changes are
//     printed for review; `--ours`/`--theirs` then take one side wholesale
//     (mx is not consulted — the human has decided).
func mergeRun(mprPath, ref, message, strategy, plan string, push bool, runners mxRunners, out io.Writer) error {
	switch strategy {
	case "", "ours", "theirs":
	default:
		return fmt.Errorf("unknown strategy %q (use --ours or --theirs)", strategy)
	}
	abs, err := filepath.Abs(mprPath)
	if err != nil {
		return err
	}
	if !mpr.AllowWritesWhileStudioProOpen && os.Getenv(mpr.AllowStudioProOpenEnv) != "1" {
		if lock, mod := mpr.StudioProLockFile(abs); lock != "" {
			return &mpr.StudioProOpenError{MprPath: abs, LockPath: lock, LockTime: mod}
		}
	}
	repo, err := openGitNoteRepo(mprPath)
	if err != nil {
		return err
	}
	defer repo.Close()
	root := repo.Root

	branch := strings.TrimSpace(gitOutOr(root, "symbolic-ref", "--quiet", "--short", "HEAD"))
	if branch == "" {
		return errors.New("detached HEAD: check out a branch before merging")
	}

	if plan != "" {
		return applyPlan(root, repo, abs, mprPath, ref, plan, message, branch, push, out)
	}

	mergeOut, mergeErr := gitOutIn(root, nil, "merge", "--no-edit", ref)
	if mergeErr == nil {
		if strings.Contains(mergeOut, "Already up to date") {
			fmt.Fprintln(out, "Already up to date.")
			return nil
		}
		head := strings.TrimSpace(gitOutOr(root, "rev-parse", "HEAD"))
		fmt.Fprintf(out, "Merged %s without model conflicts (%s).\n", ref, shortCommit(head))
		return finishMerge(out, repo, head, branch, push)
	}

	// A ref that does not resolve fails before any conflict exists.
	if _, mhErr := gitOut(root, "rev-parse", "--verify", "--quiet", "MERGE_HEAD"); mhErr != nil {
		return fmt.Errorf("git merge %s: %s", ref, firstLineOf(mergeOut))
	}

	conflicted := nulFields(gitOutOr(root, "diff", "--name-only", "--diff-filter=U", "-z"))
	var model, other []string
	for _, p := range conflicted {
		if isModelPath(repo, p) {
			model = append(model, p)
		} else {
			other = append(other, p)
		}
	}
	if len(other) > 0 {
		fmt.Fprintln(out, "Non-model conflicts, resolve by hand, then 'git add' and 'git commit':")
		for _, p := range other {
			fmt.Fprintf(out, "  %s\n", p)
		}
		return errors.New("merge has non-model conflicts left for manual resolution")
	}

	baseSha := strings.TrimSpace(gitOutOr(root, "merge-base", "HEAD", "MERGE_HEAD"))
	oursSha := strings.TrimSpace(gitOutOr(root, "rev-parse", "HEAD"))
	theirsSha := strings.TrimSpace(gitOutOr(root, "rev-parse", "MERGE_HEAD"))

	// The human has decided: take one side of every conflicted model file.
	// Deterministic, and mx is not consulted — `git checkout --ours/--theirs`
	// reads the conflicted stages directly.
	if strategy != "" {
		side := "--" + strategy
		for _, p := range model {
			if _, err := gitOutIn(root, nil, "checkout", side, "--", p); err != nil {
				return fmt.Errorf("take the %s version of %s: %w", strategy, p, err)
			}
		}
		args := []string{"add", "--"}
		args = append(args, model...)
		if _, err := gitOutIn(root, nil, args...); err != nil {
			return fmt.Errorf("stage the resolved model files: %w", err)
		}
		return commitTheMerge(root, repo, model, message, branch, push, out)
	}

	// mx first: what it auto-resolves keeps both sides, which is almost
	// always better than taking one side wholesale.
	// Check out the base and theirs revisions; MINE is the working tree.
	tmp, err := os.MkdirTemp("", "mxcli-gitmerge-")
	if err != nil {
		return err
	}
	baseDir := filepath.Join(tmp, "base")
	theirsDir := filepath.Join(tmp, "theirs")
	if _, err := gitOutIn(root, nil, "worktree", "add", "--detach", baseDir, baseSha); err != nil {
		os.RemoveAll(tmp)
		return fmt.Errorf("check out the merge base: %w", err)
	}
	if _, err := gitOutIn(root, nil, "worktree", "add", "--detach", theirsDir, theirsSha); err != nil {
		_, _ = gitOutIn(root, nil, "worktree", "remove", "--force", baseDir)
		os.RemoveAll(tmp)
		return fmt.Errorf("check out %s: %w", ref, err)
	}
	cleanWorktrees := func() {
		_, _ = gitOutIn(root, nil, "worktree", "remove", "--force", baseDir)
		_, _ = gitOutIn(root, nil, "worktree", "remove", "--force", theirsDir)
		_ = os.RemoveAll(tmp)
	}

	version := projectMendixVersion(mprPath)
	fmt.Fprintf(out, "Model files conflict (%s); merging the model with mx %s…\n",
		strings.Join(shortNames(model), ", "), version)
	code, mxOut, runErr := runners.merge(
		filepath.Join(baseDir, filepath.FromSlash(repo.MPRPath)),
		abs,
		filepath.Join(theirsDir, filepath.FromSlash(repo.MPRPath)),
		[3]string{baseSha, oursSha, theirsSha},
	)
	if s := strings.TrimSpace(mxOut); s != "" {
		fmt.Fprintln(out, s)
	}
	if runErr != nil {
		cleanWorktrees()
		return fmt.Errorf("mx git-merge failed: %w", runErr)
	}
	if code != 0 {
		// The review step: what each side changed, at property level (mx
		// diff), plus a per-unit plan template. Then leave nothing behind —
		// the human decides.
		if runners.diff != nil {
			oursJSON := filepath.Join(tmp, "ours-diff.json")
			theirsJSON := filepath.Join(tmp, "theirs-diff.json")
			d1 := runners.diff(filepath.Join(baseDir, filepath.FromSlash(repo.MPRPath)), abs, oursJSON)
			d2 := runners.diff(filepath.Join(baseDir, filepath.FromSlash(repo.MPRPath)), filepath.Join(theirsDir, filepath.FromSlash(repo.MPRPath)), theirsJSON)
			if d1 == nil && d2 == nil {
				fmt.Fprintln(out, propertyReport(oursJSON, theirsJSON, conflictedUnitIDs(model)))
			}
		}
		if err := writePlanTemplate(mprPath, baseSha, oursSha, theirsSha, model); err != nil {
			fmt.Fprintf(out, "  (could not write the plan template: %v)\n", err)
		} else {
			fmt.Fprintf(out, "Plan template: %s — set \"take\" per unit, re-run with --plan.\n", filepath.Base(mprPath)+".merge-plan.json")
		}
		cleanWorktrees()
		if _, abortErr := gitOutIn(root, nil, "merge", "--abort"); abortErr != nil {
			return fmt.Errorf("the model merge has conflicts and aborting it failed (%v); resolve in Studio Pro", abortErr)
		}
		fmt.Fprintln(out, "Merge aborted; the working tree is back to the pre-merge state.")
		return errors.New("the model merge needs a decision: re-run with --plan to resolve per unit, --ours/--theirs to take one side wholesale, or resolve in Studio Pro (Version Control › Merge Changes Here)")
	}

	args := []string{"add", "--"}
	args = append(args, model...)
	if _, err := gitOutIn(root, nil, args...); err != nil {
		cleanWorktrees()
		return fmt.Errorf("stage the resolved model files: %w", err)
	}
	if err := commitTheMerge(root, repo, nil, message, branch, push, out); err != nil {
		cleanWorktrees()
		return err
	}
	cleanWorktrees()
	return nil
}

// commitTheMerge commits the resolved merge (MERGE_HEAD is present), writes
// the note and pushes. stagedPaths, when given, are staged first.
func commitTheMerge(root string, repo *gitNoteRepo, stagedPaths []string, message, branch string, push bool, out io.Writer) error {
	if stagedPaths != nil {
		args := []string{"add", "--"}
		args = append(args, stagedPaths...)
		if _, err := gitOutIn(root, nil, args...); err != nil {
			return fmt.Errorf("stage the resolved model files: %w", err)
		}
	}
	commitArgs := []string{"commit", "--no-edit"}
	if message != "" {
		commitArgs = []string{"commit", "-m", message}
	}
	if _, err := gitOutIn(root, nil, commitArgs...); err != nil {
		return fmt.Errorf("commit the merge: %w", err)
	}

	head := strings.TrimSpace(gitOutOr(root, "rev-parse", "HEAD"))
	fmt.Fprintf(out, "Model merge committed as %s.\n", shortCommit(head))
	return finishMerge(out, repo, head, branch, push)
}

// finishMerge attaches the mx_metadata note to the finished merge and prints
// (or, with push, runs) the branch-and-notes push.
func finishMerge(out io.Writer, repo *gitNoteRepo, head, branch string, push bool) error {
	if err := annotateCommits(out, repo, []string{head}, branch, true, false); err != nil {
		return fmt.Errorf("committed %s, but attaching the note failed (%v) — run 'mxcli git note -p … --write' to attach it", shortCommit(head), err)
	}
	return pushBranchAndNotes(out, repo.Root, branch, push)
}

func pushBranchAndNotes(out io.Writer, root, branch string, push bool) error {
	if push {
		if branch == "" {
			return errors.New("detached HEAD; cannot push — check out a branch and push manually")
		}
		if _, err := gitOutIn(root, nil, "push", "origin", branch, mxMetadataRef); err != nil {
			return fmt.Errorf("push %s and %s: %w", branch, mxMetadataRef, err)
		}
		fmt.Fprintf(out, "Pushed %s and %s to origin.\n", branch, mxMetadataRef)
		return nil
	}
	if branch != "" {
		fmt.Fprintf(out, "Push with: git push origin %s %s\n", branch, mxMetadataRef)
	}
	return nil
}

// isModelPath reports whether a repo-relative path is part of the model:
// the .mpr itself or an .mxunit under mprcontents.
func isModelPath(repo *gitNoteRepo, rel string) bool {
	f := filepath.ToSlash(rel)
	if f == repo.MPRPath {
		return true
	}
	return strings.HasPrefix(f, repo.Contents+"/") && strings.HasSuffix(f, ".mxunit")
}

// projectMendixVersion reads the project's _ProductVersion for the mx
// resolution; "" on any error (ResolveMxForVersion then falls back to PATH).
func projectMendixVersion(mprPath string) string {
	r, err := mpr.Open(mprPath)
	if err != nil {
		return ""
	}
	defer r.Close()
	v, err := r.GetMendixVersion()
	if err != nil {
		return ""
	}
	return v
}

// realMxGitMerge shells out to `mx git-merge -v …`, resolved for the
// project's Mendix version.
func realMxGitMerge(base, mine, theirs string, shas [3]string) (int, string, error) {
	version := projectMendixVersion(mine)
	mxPath, err := docker.ResolveMxForVersion("", version)
	if err != nil {
		return 1, "", fmt.Errorf("no mx for Mendix %s: %w (model conflicts need it)", version, err)
	}
	cmd := exec.Command(mxPath, "git-merge", "-v", base, mine, theirs, shas[0], shas[1], shas[2])
	b, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
		} else {
			return 1, string(b), err
		}
	}
	return code, string(b), nil
}

func shortNames(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, filepath.Base(p))
	}
	return out
}

// conflictedUnitIDs extracts the unit ids of conflicted .mxunit paths.
func conflictedUnitIDs(model []string) []string {
	var ids []string
	for _, p := range model {
		if strings.HasSuffix(p, ".mxunit") {
			ids = append(ids, strings.TrimSuffix(filepath.Base(p), ".mxunit"))
		}
	}
	return ids
}

// --- property-level conflict report and the per-unit plan -------------------

type mxDiffChange struct {
	Change            string         `json:"change"`
	ChangeDescription string         `json:"changeDescription"`
	Property          string         `json:"property"`
	BaseValue         string         `json:"baseValue"`
	MineValue         string         `json:"mineValue"`
	ChangeDetails     []mxDiffChange `json:"changeDetails"`
}

type mxDiffUnit struct {
	ID            string         `json:"id"`
	Type          string         `json:"type"`
	Change        string         `json:"change"`
	ChangeDetails []mxDiffChange `json:"changeDetails"`
}

type mxDiffFile struct {
	UnitDifferences []mxDiffUnit `json:"unitDifferences"`
}

// propertyReport renders the two mx diff files as one property-level report
// over the conflicted units: what each side changed.
func propertyReport(oursJSON, theirsJSON string, unitIDs []string) string {
	want := map[string]bool{}
	for _, id := range unitIDs {
		want[id] = true
	}
	lines := map[string][]string{}
	for _, f := range []struct{ side, path string }{{"Yours", oursJSON}, {"Theirs", theirsJSON}} {
		raw, err := os.ReadFile(f.path)
		if err != nil {
			lines[f.side] = append(lines[f.side], fmt.Sprintf("  (diff unavailable: %v)", err))
			continue
		}
		var d mxDiffFile
		if err := json.Unmarshal(bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf")), &d); err != nil {
			lines[f.side] = append(lines[f.side], fmt.Sprintf("  (diff unreadable: %v)", err))
			continue
		}
		var side []string
		for _, u := range d.UnitDifferences {
			if !want[u.ID] {
				continue
			}
			side = append(side, flattenChanges(u.ChangeDetails, "  ")...)
		}
		if len(side) == 0 {
			side = append(side, "  (no change details)")
		}
		lines[f.side] = side
	}
	var b strings.Builder
	b.WriteString("\nProperty-level changes on the conflicted units:\n")
	for _, side := range []string{"Yours", "Theirs"} {
		fmt.Fprintf(&b, "%s:\n", side)
		for _, l := range lines[side] {
			b.WriteString(l)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// flattenChanges walks mx diff's change tree and yields one line per leaf:
// the description, and for property leaves the value movement.
func flattenChanges(changes []mxDiffChange, indent string) []string {
	var out []string
	for _, c := range changes {
		if c.Property != "" {
			out = append(out, fmt.Sprintf("%s%s: %s → %s", indent, c.Property, c.BaseValue, c.MineValue))
			continue
		}
		desc := c.ChangeDescription
		if desc == "" {
			desc = c.Change
		}
		out = append(out, fmt.Sprintf("%s%s", indent, desc))
		out = append(out, flattenChanges(c.ChangeDetails, indent+"    ")...)
	}
	return out
}

// mergePlanTemplate is the file a ManualResolutionRequired run writes beside
// the .mpr; the human sets "take" per unit and re-runs with --plan.
type mergePlanTemplate struct {
	Base   string     `json:"base"`
	Ours   string     `json:"ours"`
	Theirs string     `json:"theirs"`
	Units  []planUnit `json:"units"`
}

type planUnit struct {
	ID   string `json:"id"`
	Take string `json:"take"`
}

func writePlanTemplate(mprPath, baseSha, oursSha, theirsSha string, model []string) error {
	tmpl := mergePlanTemplate{Base: baseSha, Ours: oursSha, Theirs: theirsSha}
	for _, id := range conflictedUnitIDs(model) {
		tmpl.Units = append(tmpl.Units, planUnit{ID: id, Take: "ours"})
	}
	b, err := json.MarshalIndent(tmpl, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(mprPath+".merge-plan.json", b, 0o644)
}

// applyPlan re-runs the merge (it must conflict the same way again), checks
// the plan against those revisions, resolves per unit, repairs the .mpr index
// from the mixed files and commits — with the note.
func applyPlan(root string, repo *gitNoteRepo, mprAbs, mprPath, ref, planPath, message, branch string, push bool, out io.Writer) error {
	raw, err := os.ReadFile(planPath)
	if err != nil {
		return fmt.Errorf("read plan: %w", err)
	}
	var plan struct {
		Base   string     `json:"base"`
		Ours   string     `json:"ours"`
		Theirs string     `json:"theirs"`
		Units  []planUnit `json:"units"`
	}
	if err := json.Unmarshal(raw, &plan); err != nil {
		return fmt.Errorf("plan %s is not valid JSON: %w", planPath, err)
	}

	mergeOut, mergeErr := gitOutIn(root, nil, "merge", "--no-edit", ref)
	if mergeErr == nil {
		return errors.New("the merge no longer conflicts — the plan is unnecessary; nothing was done")
	}
	if _, mhErr := gitOut(root, "rev-parse", "--verify", "--quiet", "MERGE_HEAD"); mhErr != nil {
		return fmt.Errorf("git merge %s: %s", ref, firstLineOf(mergeOut))
	}
	baseSha := strings.TrimSpace(gitOutOr(root, "merge-base", "HEAD", "MERGE_HEAD"))
	oursSha := strings.TrimSpace(gitOutOr(root, "rev-parse", "HEAD"))
	theirsSha := strings.TrimSpace(gitOutOr(root, "rev-parse", "MERGE_HEAD"))
	if plan.Base != baseSha || plan.Ours != oursSha || plan.Theirs != theirsSha {
		_, _ = gitOutIn(root, nil, "merge", "--abort")
		return errors.New("stale plan: the merge revisions moved since it was written; delete it and re-run the merge to regenerate")
	}

	conflicted := nulFields(gitOutOr(root, "diff", "--name-only", "--diff-filter=U", "-z"))
	var model []string
	for _, p := range conflicted {
		if isModelPath(repo, p) {
			model = append(model, p)
		}
	}

	// Every planned unit must be modified on both sides; anything else
	// (added on one side, deleted, moved) only Studio Pro can decide.
	baseSnap, err := repo.snapshot(baseSha)
	if err != nil {
		return err
	}
	oursSnap, err := repo.snapshot(oursSha)
	if err != nil {
		return err
	}
	theirsSnap, err := repo.snapshot(theirsSha)
	if err != nil {
		return err
	}
	oursStatus := changeStatuses(computeModelChanges(baseSnap, oursSnap, nil))
	theirsStatus := changeStatuses(computeModelChanges(baseSnap, theirsSnap, nil))
	take := map[string]string{}
	for _, u := range plan.Units {
		if u.Take != "ours" && u.Take != "theirs" {
			return fmt.Errorf("plan: unit %s has take %q (ours|theirs)", shortCommit(u.ID), u.Take)
		}
		if oursStatus[u.ID] != "Modified" || theirsStatus[u.ID] != "Modified" {
			return fmt.Errorf("plan: unit %s is %s on ours and %s on theirs — only modified-on-both conflicts can be planned; resolve it in Studio Pro", shortCommit(u.ID), statusOrNone(oursStatus[u.ID]), statusOrNone(theirsStatus[u.ID]))
		}
		take[u.ID] = u.Take
	}

	// Ours everywhere, then the planned units from theirs.
	for _, p := range model {
		if _, err := gitOutIn(root, nil, "checkout", "--ours", "--", p); err != nil {
			return fmt.Errorf("take the ours version of %s: %w", p, err)
		}
	}
	for _, id := range conflictedUnitIDs(model) {
		if take[id] != "theirs" {
			continue
		}
		p := path.Join(repo.Contents, id[:2], id[2:4], id+".mxunit")
		if _, err := gitOutIn(root, nil, "checkout", "--theirs", "--", p); err != nil {
			return fmt.Errorf("take the theirs version of %s: %w", p, err)
		}
		fmt.Fprintf(out, "  %s: theirs\n", shortCommit(id))
	}
	// The .mpr index is ours; the mixed unit files have different bytes, so
	// the hashes are rewritten from the files — files are the truth.
	if err := fixHashes(mprPath, true, out); err != nil {
		if !errors.Is(err, errHashIssuesRemain) {
			return err
		}
		return errors.New("refusing to commit: the ContentsHash index has MISSING or ORPHAN entries (see above)")
	}

	args := []string{"add", "--"}
	args = append(args, model...)
	if _, err := gitOutIn(root, nil, args...); err != nil {
		return fmt.Errorf("stage the resolved model files: %w", err)
	}
	return commitTheMerge(root, repo, nil, message, branch, push, out)
}

func changeStatuses(changes []mxModelChange) map[string]string {
	out := map[string]string{}
	for _, c := range changes {
		out[c.UnitID] = c.Status
	}
	return out
}

func statusOrNone(s string) string {
	if s == "" {
		return "unchanged"
	}
	return s
}

// realMxDiff shells out to `mx diff BASE MINE OUTPUT` (resolved like git-merge).
func realMxDiff(base, mine, outJSON string) error {
	version := projectMendixVersion(mine)
	mxPath, err := docker.ResolveMxForVersion("", version)
	if err != nil {
		return fmt.Errorf("no mx for Mendix %s: %w", version, err)
	}
	if b, err := exec.Command(mxPath, "diff", base, mine, outJSON).CombinedOutput(); err != nil {
		return fmt.Errorf("mx diff: %v: %s", err, strings.TrimSpace(string(b)))
	}
	return nil
}

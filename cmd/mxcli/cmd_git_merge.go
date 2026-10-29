// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
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
    review. Then decide:
      --ours   take our side of every conflicted model file
      --theirs take their side
    (deterministic; mx is not consulted again), or resolve for real in
    Studio Pro (Version Control › Merge Changes Here), which stays the tool
    for mixed resolutions;
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
	gitMergeCmd.Flags().Bool("push", false, "push the branch and the notes ref to their remote afterwards")
	gitCmd.AddCommand(gitMergeCmd)
}

func runGitMerge(cmd *cobra.Command, args []string) error {
	ref := args[0]
	message, _ := cmd.Flags().GetString("message")
	ours, _ := cmd.Flags().GetBool("ours")
	theirs, _ := cmd.Flags().GetBool("theirs")
	push, _ := cmd.Flags().GetBool("push")
	project, _ := cmd.Flags().GetString("project")
	strategy := ""
	switch {
	case ours && theirs:
		return errors.New("--ours and --theirs cannot be combined")
	case ours:
		strategy = "ours"
	case theirs:
		strategy = "theirs"
	}
	return mergeRun(project, ref, message, strategy, push, realMxGitMerge, cmd.OutOrStdout())
}

// mxMergeRunner runs `mx git-merge BASE MINE THEIRS BASE_COMMIT MINE_COMMIT
// THEIRS_COMMIT` and returns mx's exit code and output.
type mxMergeRunner func(base, mine, theirs string, shas [3]string) (int, string, error)

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
func mergeRun(mprPath, ref, message, strategy string, push bool, runMx mxMergeRunner, out io.Writer) error {
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
	code, mxOut, runErr := runMx(
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
		// The review step: show what each side changed, then leave nothing
		// behind. The human decides — --ours/--theirs or Studio Pro.
		cleanWorktrees()
		printConflictReport(root, repo, baseSha, oursSha, theirsSha, out)
		if _, abortErr := gitOutIn(root, nil, "merge", "--abort"); abortErr != nil {
			return fmt.Errorf("the model merge has conflicts and aborting it failed (%v); resolve in Studio Pro", abortErr)
		}
		fmt.Fprintln(out, "Merge aborted; the working tree is back to the pre-merge state.")
		return errors.New("the model merge needs a decision: re-run with --ours or --theirs to take one side wholesale, or resolve in Studio Pro (Version Control › Merge Changes Here)")
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

// printConflictReport shows what each side changed — the review material a
// human needs before deciding how the conflict is resolved.
func printConflictReport(root string, repo *gitNoteRepo, baseSha, oursSha, theirsSha string, out io.Writer) {
	base, err := repo.snapshot(baseSha)
	if err != nil {
		fmt.Fprintf(out, "  (could not read the merge base: %v)\n", err)
		return
	}
	ours, err := repo.snapshot(oursSha)
	if err != nil {
		fmt.Fprintf(out, "  (could not read our side: %v)\n", err)
		return
	}
	theirs, err := repo.snapshot(theirsSha)
	if err != nil {
		fmt.Fprintf(out, "  (could not read their side: %v)\n", err)
		return
	}
	oursChanges := computeModelChanges(base, ours, nil)
	theirsChanges := computeModelChanges(base, theirs, nil)
	theirsIDs := map[string]bool{}
	for _, c := range theirsChanges {
		theirsIDs[c.UnitID] = true
	}
	oursIDs := map[string]bool{}
	for _, c := range oursChanges {
		oursIDs[c.UnitID] = true
	}
	fmt.Fprintf(out, "\nBoth sides changed — a decision is needed on:\n")
	for _, c := range oursChanges {
		if theirsIDs[c.UnitID] {
			fmt.Fprintf(out, "  %s %s [%s] (unit %s)\n", c.UnitType, c.UnitName, c.Module, shortCommit(c.UnitID))
		}
	}
	fmt.Fprintf(out, "\nYours (%s):\n", shortCommit(oursSha))
	for _, c := range oursChanges {
		fmt.Fprintf(out, "  %-9s %s %s [%s]%s\n", c.Status, c.UnitType, c.UnitName, c.Module, oursTheirsMark(theirsIDs, c.UnitID))
	}
	fmt.Fprintf(out, "\nTheirs (%s):\n", shortCommit(theirsSha))
	for _, c := range theirsChanges {
		fmt.Fprintf(out, "  %-9s %s %s [%s]%s\n", c.Status, c.UnitType, c.UnitName, c.Module, oursTheirsMark(oursIDs, c.UnitID))
	}
}

func oursTheirsMark(other map[string]bool, unitID string) string {
	if other[unitID] {
		return "  ◄ also changed on the other side"
	}
	return ""
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

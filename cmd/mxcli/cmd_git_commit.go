// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/spf13/cobra"
)

// cmd_git_commit.go: `mxcli git commit` is the commit Studio Pro would have
// made, for when the agent (or CI) commits instead: it repairs the
// ContentsHash index, stages everything except the files Studio Pro never
// commits, commits, and attaches the mx_metadata note — the one sequence that
// leaves the repository the way Studio Pro expects to find it. The single
// steps are `mxcli fix hashes`, plain git, and `mxcli git note`.

var gitCommitCmd = &cobra.Command{
	Use:   "commit -m <message> -p <project.mpr>",
	Short: "Commit the working tree the way Studio Pro does: hashes, staging, note",
	Long: `Commit every change with the steps Studio Pro's own commit wraps:

  1. the .mpr's ContentsHash index is repaired against the unit files first —
     a hash that drifted (typically from a git checkout of a single file) is
     rewritten from the file, which is the source of truth; MISSING or ORPHAN
     entries have no automatic repair and refuse the commit;
  2. everything is staged, then the files Studio Pro never commits are
     unstaged again: project-settings.user.json, *.mpr.lock, *.mpr.bak,
     .mendix-cache/, deployment/, theme-cache/, .mxcli/;
  3. git commit runs with your message and your git identity;
  4. the mx_metadata note is computed from the new commit and attached, so
     Studio Pro can version-check it and show what changed.

--push pushes the branch and the notes ref together (git push alone loses the
note the next time Studio Pro fetches). The commit is refused while Studio Pro
has the project open (<project>.mpr.lock).

Exits non-zero when the commit did not happen, or when its note could not be
attached afterwards (the commit itself stays; re-run 'mxcli git note').
MPR v2 projects only (the note needs the unit index; Mendix 10.18 and later).`,
	Example: `  mxcli git commit -m "Add order history page" -p app.mpr
  mxcli git commit -m "Fix stock filter" -p app.mpr --push
  mxcli git commit -m "WIP" -p app.mpr --dry-run`,
	Args:          cobra.NoArgs,
	RunE:          runGitCommit,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	gitCommitCmd.Flags().StringP("message", "m", "", "commit message")
	_ = gitCommitCmd.MarkFlagRequired("message")
	gitCommitCmd.Flags().StringP("project", "p", "", "path to the Mendix project (.mpr)")
	_ = gitCommitCmd.MarkFlagRequired("project")
	gitCommitCmd.Flags().Bool("push", false, "push the branch and the notes ref to their remote afterwards")
	gitCommitCmd.Flags().Bool("dry-run", false, "stage and report, then unstage; commit nothing")
	gitCmd.AddCommand(gitCommitCmd)
}

func runGitCommit(cmd *cobra.Command, _ []string) error {
	message, _ := cmd.Flags().GetString("message")
	push, _ := cmd.Flags().GetBool("push")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	project, _ := cmd.Flags().GetString("project")
	return commitRun(project, message, push, dryRun, cmd.OutOrStdout())
}

// commitRun is the command without its wiring.
func commitRun(mprPath, message string, push, dryRun bool, out io.Writer) error {
	if dryRun && push {
		return errors.New("--dry-run and --push cannot be combined: a dry run commits nothing to push")
	}
	abs, err := filepath.Abs(mprPath)
	if err != nil {
		return err
	}
	// The lock guard, honouring the same overrides every mxcli write honours:
	// a lock Studio Pro left behind on a crash must not deadlock the user.
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

	// An unresolved merge must stop here: `git add -A` below would stage the
	// conflict markers and commit them, which plain git refuses.
	if unmerged := strings.TrimSpace(gitOutOr(root, "ls-files", "--unmerged")); unmerged != "" {
		return errors.New("refusing to commit: the merge has unresolved conflicts (git ls-files --unmerged lists the files); resolve them first, or commit the merge from Studio Pro")
	}

	// 1. Hashes first: the repair rewrites the .mpr, which is staged below.
	//    A dry run writes nothing, so it verifies only and reports the drift.
	repair := !dryRun
	if err := fixHashes(mprPath, repair, out); err != nil {
		switch {
		case errors.Is(err, errHashIssuesRemain) && dryRun:
			fmt.Fprintln(out, "ContentsHash drift found; a real commit would repair it from the files on disk.")
		case errors.Is(err, errHashIssuesRemain):
			return errors.New("refusing to commit: the ContentsHash index has MISSING or ORPHAN entries, which have no automatic repair (see above)")
		default:
			return err
		}
	}

	// 2. Stage everything, then unstage what Studio Pro never commits.
	if _, err := gitOutIn(root, nil, "add", "-A"); err != nil {
		return fmt.Errorf("stage changes: %w", err)
	}
	// -z: paths come back NUL-separated, so spaces and quotes stay inside one path.
	staged := nulFields(gitOutOr(root, "diff", "--cached", "--name-only", "-z"))
	var junk []string
	for _, f := range staged {
		if isJunkPath(f) {
			junk = append(junk, f)
		}
	}
	if len(junk) > 0 {
		args := append([]string{"reset", "-q", "--"}, junk...)
		if _, err := gitOutIn(root, nil, args...); err != nil {
			return fmt.Errorf("unstage files Studio Pro never commits: %w", err)
		}
		fmt.Fprintf(out, "Left out %d file(s) Studio Pro never commits (project-settings.user.json, *.mpr.lock, caches).\n", len(junk))
	}
	if len(staged) == len(junk) {
		fmt.Fprintln(out, "nothing to commit, working tree clean")
		return nil
	}
	kept := make([]string, 0, len(staged)-len(junk))
	for _, f := range staged {
		if !isJunkPath(f) {
			kept = append(kept, f)
		}
	}
	fmt.Fprintf(out, "Staged %d file(s):\n", len(kept))
	shown := kept
	if len(shown) > 10 {
		shown = shown[:10]
	}
	for _, f := range shown {
		fmt.Fprintf(out, "  %s\n", f)
	}
	if len(kept) > 10 {
		fmt.Fprintf(out, "  … and %d more\n", len(kept)-10)
	}

	if dryRun {
		if _, err := gitOutIn(root, nil, "reset", "-q"); err != nil {
			return fmt.Errorf("unstage: %w", err)
		}
		fmt.Fprintln(out, "Dry run: staged nothing permanently, committed nothing.")
		return nil
	}

	// 3. The commit itself, with the user's message and identity.
	if _, err := gitOutIn(root, nil, "commit", "-m", message); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	head := strings.TrimSpace(gitOutOr(root, "rev-parse", "HEAD"))
	branch := strings.TrimSpace(gitOutOr(root, "symbolic-ref", "--quiet", "--short", "HEAD"))
	fmt.Fprintf(out, "Committed %s %q.\n", shortCommit(head), message)

	// 4. The note, computed from the commit that now exists. The commit is
	// already made, so a failure here must say so — the note is attachable
	// later, and a CI retry must not make a second commit.
	if err := annotateCommits(out, repo, []string{head}, branch, true, false); err != nil {
		return fmt.Errorf("committed %s, but attaching the note failed (%v) — run 'mxcli git note -p %s --write' to attach it", shortCommit(head), err, mprPath)
	}

	if push {
		if branch == "" {
			return errors.New("detached HEAD; cannot push — check out a branch and push manually")
		}
		if _, err := gitOutIn(root, nil, "push", "origin", branch, mxMetadataRef); err != nil {
			return fmt.Errorf("push %s and %s: %w", branch, mxMetadataRef, err)
		}
		fmt.Fprintf(out, "Pushed %s and %s to origin.\n", branch, mxMetadataRef)
	}
	return nil
}

// nulFields splits git's -z output (NUL-separated, no quoting) into paths.
func nulFields(s string) []string {
	var out []string
	for _, f := range strings.Split(s, "\x00") {
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

// isJunkPath reports whether a repo-relative path is one of the files Studio
// Pro never commits. Matched on any path level and without case, the way
// Studio Pro matches its own lock file (whose .gitignore entry is lowercase).
func isJunkPath(rel string) bool {
	f := filepath.ToSlash(rel)
	base := path.Base(f)
	switch {
	case strings.EqualFold(base, "project-settings.user.json"):
		return true
	case strings.HasSuffix(strings.ToLower(base), ".mpr.lock"), strings.HasSuffix(strings.ToLower(base), ".mpr.bak"):
		return true
	}
	for _, seg := range strings.Split(f, "/") {
		switch strings.ToLower(seg) {
		case ".mendix-cache", "deployment", "theme-cache", ".mxcli":
			return true
		}
	}
	return false
}

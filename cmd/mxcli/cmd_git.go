// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/spf13/cobra"
)

// gitExecCommand is a package-level variable so tests can replace it with a stub.
var gitExecCommand = func(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// mxMetadata is the JSON structure written as a git note on every Mendix commit.
// Field order matches Studio Pro / libgit2 output exactly.
type mxMetadata struct {
	BranchName        string `json:"BranchName"`
	ModelerVersion    string `json:"ModelerVersion"`
	ModelChanges      []any  `json:"ModelChanges"`
	RelatedStories    []any  `json:"RelatedStories"`
	SolutionVersion   string `json:"SolutionVersion"`
	MPRFormatVersion  string `json:"MPRFormatVersion"`
	HasModelerVersion bool   `json:"HasModelerVersion"`
}

// buildMxMetadata produces the compact JSON string for a mx_metadata git note.
// The result has no trailing newline, matching libgit2 blob format.
func buildMxMetadata(mendixVersion, mprFormatVersion string) string {
	m := mxMetadata{
		ModelerVersion:    mendixVersion,
		ModelChanges:      []any{},
		RelatedStories:    []any{},
		MPRFormatVersion:  mprFormatVersion,
		HasModelerVersion: true,
	}
	b, _ := json.Marshal(m) // json.Marshal never returns error for this struct
	return string(b)        // no trailing newline — json.Marshal doesn't add one
}

// hashObjectAndAddNote writes metadata as a git blob and attaches it as a
// mx_metadata note on the given commit SHA. Uses git hash-object to create
// a blob without a trailing newline (libgit2 compatibility), then git notes
// add to associate it. If a note already exists, retries with -f.
func hashObjectAndAddNote(commitSHA, metadata string) error {
	// Step 1: write blob
	hashCmd := gitExecCommand("git", "hash-object", "-w", "--stdin")
	hashCmd.Stdin = strings.NewReader(metadata)
	out, err := hashCmd.Output()
	if err != nil {
		return fmt.Errorf("git hash-object: %w", err)
	}
	blobHash := strings.TrimSpace(string(out))

	// Step 2: associate note (no -f first, then retry with -f if note exists)
	addCmd := gitExecCommand("git", "notes", "--ref=mx_metadata", "add", "-C", blobHash, commitSHA)
	if err := addCmd.Run(); err != nil {
		addCmd = gitExecCommand("git", "notes", "--ref=mx_metadata", "add", "-f", "-C", blobHash, commitSHA)
		if err := addCmd.Run(); err != nil {
			return fmt.Errorf("git notes add: %w", err)
		}
	}
	return nil
}

// detectMendixVersion resolves the Mendix version string and MPRFormatVersion
// string needed for mx_metadata notes. Detection priority:
//  1. versionFlag — explicit --version flag value
//  2. projectPath — open MPR file and read _MetaData
//  3. current dir — auto-discover *.mpr and read it
//  4. existing notes — scan git notes for ModelerVersion
//  5. error
func detectMendixVersion(versionFlag, projectPath string) (mendixVersion, mprFormatVersion string, err error) {
	// 1. Explicit --version flag
	if versionFlag != "" {
		return versionFlag, "Version2", nil
	}

	// 2. Try MPR file (explicit path or auto-discovered)
	mprPath := projectPath
	if mprPath == "" {
		mprPath = discoverProjectPath() // reuses main.go function
	}
	if mprPath != "" {
		if v, f, ok := readVersionFromMPR(mprPath); ok {
			return v, f, nil
		}
	}

	// 3. Scan existing mx_metadata notes
	if v := scanVersionFromNotes(); v != "" {
		return v, "Version2", nil
	}

	return "", "", fmt.Errorf("cannot detect Mendix version: no --version flag, no MPR file found, and no existing mx_metadata notes\n\n  Fix:\n    mxcli git commit -p app.mpr -m \"...\"\n    mxcli git commit --version 10.6.0.0 -m \"...\"")
}

// readVersionFromMPR opens an MPR file and reads the Mendix version.
func readVersionFromMPR(path string) (version, formatVersion string, ok bool) {
	r, err := mmpr.Open(path)
	if err != nil {
		return "", "", false
	}
	defer r.Close()

	v, err := r.GetMendixVersion()
	if err != nil || v == "" {
		return "", "", false
	}
	fmtVer := "Version2"
	if r.Version() == mmpr.MPRVersionV1 {
		fmtVer = "Version1"
	}
	return v, fmtVer, true
}

// scanVersionFromNotes reads the git notes list and extracts ModelerVersion
// from the first note that is valid JSON with that field set.
func scanVersionFromNotes() string {
	listCmd := gitExecCommand("git", "notes", "--ref=mx_metadata", "list")
	out, err := listCmd.Output()
	if err != nil || len(out) == 0 {
		return ""
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		commitSHA := parts[1]
		showCmd := gitExecCommand("git", "notes", "--ref=mx_metadata", "show", commitSHA)
		noteOut, err := showCmd.Output()
		if err != nil {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(noteOut, &m); err != nil {
			continue
		}
		if v, ok := m["ModelerVersion"].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// ──────────────────────────────────────────────
// Cobra command group
// ──────────────────────────────────────────────

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Mendix-aware git helpers (commit, doctor, fix)",
	Long: `Git helpers that maintain Mendix Studio Pro compatibility.

Studio Pro requires mx_metadata git notes on every commit and specific
git config keys. These commands ensure AI agents and native-git users
don't break Studio Pro's Version Control panel.

Commands:
  commit      Commit changes and automatically add mx_metadata note
  notes push  Push mx_metadata notes to remote
  doctor      Diagnose git/Mendix compatibility issues
  fix         Repair missing or malformed mx_metadata notes
`,
}

var gitCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit and auto-add mx_metadata git note",
	Long: `Run 'git commit' with all provided flags, then automatically write
an mx_metadata git note on the new commit. This makes commits from
AI agents or native git clients compatible with Mendix Studio Pro.

After committing, run:
  mxcli git notes push

Examples:
  mxcli git commit -m "Add OrderItem entity"
  mxcli git commit -a -m "Fix microflow logic"
  mxcli git commit --amend
  mxcli git commit -p app.mpr -m "Update entity"
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		versionFlag, _ := cmd.Flags().GetString("version")
		projectPath, _ := cmd.Flags().GetString("project")
		return runGitCommit(args, versionFlag, projectPath, cmd.OutOrStdout())
	},
}

func init() {
	gitCommitCmd.Flags().String("version", "", "Mendix version number (e.g. 10.6.0.0), skips MPR detection")
	gitCmd.AddCommand(gitCommitCmd)
}

// runGitCommit executes git commit with the given args, then adds mx_metadata note.
// It is a separate function (not inline in RunE) so tests can call it directly.
// The out parameter is an io.Writer: strings.Builder (tests) and
// cmd.OutOrStdout() (runtime) both satisfy it.
func runGitCommit(gitArgs []string, versionFlag, projectPath string, out io.Writer) error {
	// Build git commit command, forwarding all user-supplied args.
	commitArgs := append([]string{"commit"}, gitArgs...)
	commitCmd := gitExecCommand("git", commitArgs...)
	commitCmd.Stdout = out
	commitCmd.Stderr = out
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	// Get new commit SHA
	shaCmd := gitExecCommand("git", "rev-parse", "HEAD")
	shaOut, err := shaCmd.Output()
	if err != nil {
		return fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	commitSHA := strings.TrimSpace(string(shaOut))

	// Detect Mendix version
	mendixVersion, mprFmtVersion, err := detectMendixVersion(versionFlag, projectPath)
	if err != nil {
		fmt.Fprintf(out, "\n[mendix] WARNING: %v\n", err)
		return nil // commit succeeded; note failure is non-fatal but warned
	}

	// Build and write note
	metadata := buildMxMetadata(mendixVersion, mprFmtVersion)
	if err := hashObjectAndAddNote(commitSHA, metadata); err != nil {
		fmt.Fprintf(out, "\n[mendix] WARNING: could not write mx_metadata note: %v\n", err)
		return nil
	}

	shortSHA := commitSHA
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}
	fmt.Fprintf(out,
		"\n[mendix] mx_metadata note added to %s (Mendix %s)\n\n  Push notes when ready:\n    mxcli git notes push\n\n  Or push code + notes together:\n    git push && mxcli git notes push\n",
		shortSHA, mendixVersion)
	return nil
}

// ──────────────────────────────────────────────
// notes push command
// ──────────────────────────────────────────────

var gitNotesCmd = &cobra.Command{
	Use:   "notes",
	Short: "Manage mx_metadata git notes",
}

var gitNotesPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push mx_metadata notes to remote",
	Long: `Push refs/notes/mx_metadata to the remote repository so Mendix Studio Pro
can read the version history written by 'mxcli git commit'.

Examples:
  mxcli git notes push
  mxcli git notes push --remote origin
  mxcli git notes push --force
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		remote, _ := cmd.Flags().GetString("remote")
		force, _ := cmd.Flags().GetBool("force")
		return runGitNotesPush(remote, force, cmd.OutOrStdout())
	},
}

// runGitNotesPush pushes refs/notes/mx_metadata to the resolved remote.
func runGitNotesPush(remoteOverride string, force bool, out io.Writer) error {
	remote := resolveRemote(remoteOverride)
	if remote == "" {
		return fmt.Errorf("no remote found — specify with --remote <name>")
	}

	pushArgs := []string{"push", remote, "refs/notes/mx_metadata"}
	if force {
		pushArgs = append(pushArgs, "--force")
	}
	pushCmd := gitExecCommand("git", pushArgs...)
	pushCmd.Stdout = out
	pushCmd.Stderr = out
	if err := pushCmd.Run(); err != nil {
		return fmt.Errorf("git push notes: %w\n\n  If remote has diverged, retry with --force", err)
	}

	fmt.Fprintf(out, "\n[mendix] notes pushed to %s/refs/notes/mx_metadata\n\n  Studio Pro can now read version history.\n  Remember to also push your commits:\n    git push\n", remote)
	return nil
}

// resolveRemote returns the git remote to use: override → tracking remote → "origin".
func resolveRemote(override string) string {
	if override != "" {
		return override
	}
	// Detect tracking remote from current branch
	cmd := gitExecCommand("git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	out, err := cmd.Output()
	if err == nil {
		parts := strings.SplitN(strings.TrimSpace(string(out)), "/", 2)
		if len(parts) == 2 {
			return parts[0]
		}
	}
	// Fallback: check if "origin" exists
	listCmd := gitExecCommand("git", "remote")
	listOut, err := listCmd.Output()
	if err == nil {
		remotes := strings.Fields(strings.TrimSpace(string(listOut)))
		for _, r := range remotes {
			if r == "origin" {
				return "origin"
			}
		}
		if len(remotes) > 0 {
			return remotes[0]
		}
	}
	return ""
}

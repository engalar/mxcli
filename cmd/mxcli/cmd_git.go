// SPDX-License-Identifier: Apache-2.0

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
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

All git commit flags (-m, -a, --amend, ...) are passed straight through.
Only --version and -p/--project are consumed by mxcli.

After committing, run:
  mxcli git notes push

Examples:
  mxcli git commit -m "Add OrderItem entity"
  mxcli git commit -a -m "Fix microflow logic"
  mxcli git commit --amend
  mxcli git commit -p app.mpr -m "Update entity"
`,
	// DisableFlagParsing so git's own flags (-m, -a, --amend, etc.) pass through
	// untouched. mxcli's --version and -p/--project are extracted manually below.
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var versionFlag, projectPath string
		var gitArgs []string
		for i := 0; i < len(args); i++ {
			switch {
			case args[i] == "--version" && i+1 < len(args):
				versionFlag = args[i+1]
				i++
			case args[i] == "-p" && i+1 < len(args):
				projectPath = args[i+1]
				i++
			case args[i] == "--project" && i+1 < len(args):
				projectPath = args[i+1]
				i++
			case strings.HasPrefix(args[i], "--project="):
				projectPath = strings.TrimPrefix(args[i], "--project=")
			case strings.HasPrefix(args[i], "--version="):
				versionFlag = strings.TrimPrefix(args[i], "--version=")
			default:
				gitArgs = append(gitArgs, args[i])
			}
		}
		return runGitCommit(gitArgs, versionFlag, projectPath, cmd.OutOrStdout())
	},
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

// ──────────────────────────────────────────────
// Doctor command
// ──────────────────────────────────────────────

type doctorCheck struct {
	Name   string
	OK     bool
	Detail string
}

var gitDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose Mendix git compatibility (read-only)",
	Long: `Run 7 read-only health checks on the current git repository:

  1. Git local config (core.autocrlf, mendix.*)
  2. Remote URL protocol (SSH blocked by Studio Pro; HTTP/HTTPS OK)
  3. Remote refs/notes/mx_metadata existence
  4. Local commits mx_metadata completeness
  5. Notes JSON format validity
  6. MPR ↔ mprcontents consistency (missing/orphan .mxunit files, merge conflicts)
  7. Duplicate note blobs (raw-git commits that bypassed Studio Pro)

Examples:
  mxcli git doctor
  mxcli git doctor --remote origin
  mxcli git doctor -p app.mpr
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		remote, _ := cmd.Flags().GetString("remote")
		projectPath, _ := cmd.Flags().GetString("project")
		return runDoctor(remote, projectPath, cmd.OutOrStdout())
	},
}

// loadNotesList runs 'git notes --ref=mx_metadata list' once and returns a
// blobHash→[]commitSHA map. Multiple commits may share the same blob hash
// (raw-git attach), so the value is a slice to preserve all entries.
// All doctor checks share this single result to avoid redundant subprocess
// invocations per doctor run.
func loadNotesList() map[string][]string {
	listCmd := gitExecCommand("git", "notes", "--ref=mx_metadata", "list")
	out, err := listCmd.Output()
	result := make(map[string][]string)
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return result
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 {
			result[parts[0]] = append(result[parts[0]], parts[1]) // blobHash → []commitSHA
		}
	}
	return result
}

func runDoctor(remoteOverride, projectPath string, out io.Writer) error {
	remote := resolveRemote(remoteOverride)
	fmt.Fprintf(out, "Diagnosing Git repo\n\n")

	notesList := loadNotesList()

	checks := []doctorCheck{
		checkGitConfig(),
		checkRemoteURL(remote),
		checkRemoteNotesRef(remote),
		checkCommitsHaveNotesFromList(notesList),
		checkNotesJSONFormatFromList(notesList),
		checkMPRConsistency(projectPath),
		checkDuplicateNoteBlobsFromList(notesList),
	}

	failed := 0
	for _, c := range checks {
		sym := "[✓]"
		if !c.OK {
			sym = "[✗]"
			failed++
		}
		fmt.Fprintf(out, "  %s %s\n", sym, c.Detail)
	}

	fmt.Fprintf(out, "\nDiagnosis: ")
	if failed == 0 {
		fmt.Fprintf(out, "all checks passed.\n")
	} else {
		fmt.Fprintf(out, "%d issue(s) found.\n  Run 'mxcli git fix' to repair.\n", failed)
	}
	return nil
}

func checkGitConfig() doctorCheck {
	wanted := map[string]string{
		"core.autocrlf":              "false",
		"mendix.commits-since-gc":    "0",
		"mendix.lineEndingResetDone": "true",
	}
	var missing []string
	for key, want := range wanted {
		cmd := gitExecCommand("git", "config", "--local", key)
		out, err := cmd.Output()
		got := strings.ToLower(strings.TrimSpace(string(out)))
		if err != nil || got != want {
			missing = append(missing, key)
		}
	}
	if len(missing) == 0 {
		return doctorCheck{Name: "config", OK: true, Detail: "Git local config (core.autocrlf, mendix.*)"}
	}
	sort.Strings(missing)
	return doctorCheck{Name: "config", OK: false, Detail: fmt.Sprintf("Git local config — missing: %s", strings.Join(missing, ", "))}
}

func checkRemoteURL(remote string) doctorCheck {
	if remote == "" {
		return doctorCheck{Name: "remote-url", OK: false, Detail: "Remote URL — no remote found"}
	}
	cmd := gitExecCommand("git", "remote", "get-url", remote)
	out, err := cmd.Output()
	if err != nil {
		return doctorCheck{Name: "remote-url", OK: false, Detail: fmt.Sprintf("Remote URL — cannot get URL for '%s'", remote)}
	}
	url := strings.TrimSpace(string(out))
	// Studio Pro requires non-SSH. HTTP is fine for internal Gitea/GitLab servers.
	if strings.HasPrefix(url, "git@") {
		return doctorCheck{Name: "remote-url", OK: false, Detail: fmt.Sprintf("Remote URL: %s (SSH not supported by Studio Pro — convert to HTTPS)", url)}
	}
	protocol := "HTTPS"
	if strings.HasPrefix(url, "http://") {
		protocol = "HTTP (internal)"
	}
	return doctorCheck{Name: "remote-url", OK: true, Detail: fmt.Sprintf("Remote URL: %s (%s)", url, protocol)}
}

func checkRemoteNotesRef(remote string) doctorCheck {
	if remote == "" {
		return doctorCheck{Name: "remote-notes", OK: false, Detail: "Remote refs/notes/mx_metadata — no remote"}
	}
	cmd := gitExecCommand("git", "ls-remote", remote, "refs/notes/mx_metadata")
	out, err := cmd.Output()
	if err == nil && strings.Contains(string(out), "refs/notes/mx_metadata") {
		return doctorCheck{Name: "remote-notes", OK: true, Detail: fmt.Sprintf("Remote refs/notes/mx_metadata exists on '%s'", remote)}
	}
	return doctorCheck{Name: "remote-notes", OK: false, Detail: fmt.Sprintf("Remote refs/notes/mx_metadata missing on '%s'", remote)}
}

func checkCommitsHaveNotes() doctorCheck {
	return checkCommitsHaveNotesFromList(loadNotesList())
}

func checkCommitsHaveNotesFromList(notesList map[string][]string) doctorCheck {
	logCmd := gitExecCommand("git", "log", "--format=%H")
	logOut, err := logCmd.Output()
	if err != nil || len(strings.TrimSpace(string(logOut))) == 0 {
		return doctorCheck{Name: "commits-notes", OK: true, Detail: "Commits mx_metadata — no commits"}
	}
	allCommits := strings.Split(strings.TrimSpace(string(logOut)), "\n")

	// Build noted set from pre-loaded list (blobHash→[]commitSHA map).
	noted := make(map[string]bool, len(notesList))
	for _, commitSHAs := range notesList {
		for _, commitSHA := range commitSHAs {
			noted[commitSHA] = true
		}
	}

	var missing []string
	for _, sha := range allCommits {
		sha = strings.TrimSpace(sha)
		if sha != "" && !noted[sha] {
			missing = append(missing, sha[:minInt(len(sha), 7)])
		}
	}
	if len(missing) == 0 {
		return doctorCheck{Name: "commits-notes", OK: true, Detail: fmt.Sprintf("Commits mx_metadata — all %d have notes", len(allCommits))}
	}
	detail := fmt.Sprintf("Commits mx_metadata — %d/%d missing notes: %s",
		len(missing), len(allCommits), strings.Join(missing[:minInt(len(missing), 3)], ", "))
	if len(missing) > 3 {
		detail += fmt.Sprintf(" ... (+%d more)", len(missing)-3)
	}
	return doctorCheck{Name: "commits-notes", OK: false, Detail: detail}
}

func checkNotesJSONFormat() doctorCheck {
	return checkNotesJSONFormatFromList(loadNotesList())
}

func checkNotesJSONFormatFromList(notesList map[string][]string) doctorCheck {
	if len(notesList) == 0 {
		return doctorCheck{Name: "notes-json", OK: true, Detail: "Notes JSON format — no notes to check"}
	}
	malformed := 0
	for _, commitSHAs := range notesList {
		for _, commitSHA := range commitSHAs {
			showCmd := gitExecCommand("git", "notes", "--ref=mx_metadata", "show", commitSHA)
			noteOut, err := showCmd.Output()
			if err != nil {
				continue
			}
			var m map[string]any
			if err := json.Unmarshal(noteOut, &m); err != nil {
				malformed++
			}
		}
	}
	if malformed == 0 {
		return doctorCheck{Name: "notes-json", OK: true, Detail: "Notes JSON format — all valid"}
	}
	return doctorCheck{Name: "notes-json", OK: false, Detail: fmt.Sprintf("Notes JSON format — %d malformed notes", malformed)}
}

// ──────────────────────────────────────────────
// P0: MPR ↔ mprcontents consistency
// ──────────────────────────────────────────────

// blobToUUIDForPath converts a 16-byte SQLite GUID blob to the UUID string used
// for mxunit file paths (Microsoft GUID format: first 3 groups little-endian).
func blobToUUIDForPath(blob []byte) string {
	if len(blob) != 16 {
		return ""
	}
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		blob[3], blob[2], blob[1], blob[0],
		blob[5], blob[4],
		blob[7], blob[6],
		blob[8], blob[9],
		blob[10], blob[11], blob[12], blob[13], blob[14], blob[15])
}

// checkMPRConsistency verifies that every Unit row in the .mpr SQLite database
// has a corresponding .mxunit file on disk (v2 format), and flags:
//   - missing .mxunit files (mx check crash trigger)
//   - orphan .mxunit files not referenced by any Unit row
//   - Unit rows with TreeConflict != 0 or non-empty ContentsConflicts
func checkMPRConsistency(projectPath string) doctorCheck {
	if projectPath == "" {
		return doctorCheck{Name: "mpr-consistency", OK: true, Detail: "MPR consistency — skipped (no -p flag)"}
	}

	mprDir := filepath.Dir(projectPath)
	contentsDir := filepath.Join(mprDir, "mprcontents")
	if _, err := os.Stat(contentsDir); os.IsNotExist(err) {
		return doctorCheck{Name: "mpr-consistency", OK: true, Detail: "MPR consistency — v1 format (single-file, no mprcontents/)"}
	}

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", projectPath))
	if err != nil {
		return doctorCheck{Name: "mpr-consistency", OK: false, Detail: fmt.Sprintf("MPR consistency — cannot open SQLite: %v", err)}
	}
	defer db.Close()

	rows, err := db.Query("SELECT UnitID, TreeConflict, ContentsConflicts FROM Unit")
	if err != nil {
		return doctorCheck{Name: "mpr-consistency", OK: false, Detail: fmt.Sprintf("MPR consistency — cannot query Unit table: %v", err)}
	}
	defer rows.Close()

	seenUUIDs := make(map[string]bool)
	var missingFiles, conflictUnits int
	var missingExamples []string

	for rows.Next() {
		var blob []byte
		var treeConflict int64
		var contentsConflicts sql.NullString
		if err := rows.Scan(&blob, &treeConflict, &contentsConflicts); err != nil {
			continue
		}
		uuid := blobToUUIDForPath(blob)
		if uuid == "" {
			continue
		}
		seenUUIDs[uuid] = true

		unitPath := filepath.Join(contentsDir, uuid[0:2], uuid[2:4], uuid+".mxunit")
		if _, err := os.Stat(unitPath); os.IsNotExist(err) {
			missingFiles++
			if len(missingExamples) < 3 {
				missingExamples = append(missingExamples, uuid[:8]+"…")
			}
		}
		if treeConflict != 0 || (contentsConflicts.Valid && strings.TrimSpace(contentsConflicts.String) != "") {
			conflictUnits++
		}
	}
	if err := rows.Err(); err != nil {
		return doctorCheck{Name: "mpr-consistency", OK: false, Detail: fmt.Sprintf("MPR consistency — row scan error: %v", err)}
	}

	// Count orphan .mxunit files (on disk but not in Unit table).
	orphanFiles := 0
	_ = filepath.Walk(contentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".mxunit") {
			return nil
		}
		base := strings.TrimSuffix(filepath.Base(path), ".mxunit")
		if !seenUUIDs[base] {
			orphanFiles++
		}
		return nil
	})

	totalUnits := len(seenUUIDs)
	var issues []string
	if missingFiles > 0 {
		msg := fmt.Sprintf("%d missing .mxunit file(s)", missingFiles)
		if len(missingExamples) > 0 {
			msg += " (" + strings.Join(missingExamples, ", ") + ")"
		}
		issues = append(issues, msg)
	}
	if conflictUnits > 0 {
		issues = append(issues, fmt.Sprintf("%d unit(s) with unresolved merge conflicts (TreeConflict/ContentsConflicts)", conflictUnits))
	}
	if orphanFiles > 0 {
		issues = append(issues, fmt.Sprintf("%d orphan .mxunit file(s) on disk (not in Unit table)", orphanFiles))
	}

	if len(issues) == 0 {
		return doctorCheck{Name: "mpr-consistency", OK: true,
			Detail: fmt.Sprintf("MPR consistency — OK (%d units, no missing files or conflicts)", totalUnits)}
	}
	return doctorCheck{Name: "mpr-consistency", OK: false,
		Detail: fmt.Sprintf("MPR consistency — %d unit(s): %s", totalUnits, strings.Join(issues, "; "))}
}

// ──────────────────────────────────────────────
// P1: Duplicate note blob detection
// ──────────────────────────────────────────────

// checkDuplicateNoteBlobs detects commits that share identical mx_metadata note
// blobs — a reliable indicator that the notes were attached after the fact
// (e.g. by mxcli git fix) rather than generated by Studio Pro at commit time.
// When Studio Pro commits, it generates a unique blob per commit because
// ModelChanges lists exactly what changed; blob reuse means ModelChanges is
// empty/generic, hiding which documents were actually modified.
func checkDuplicateNoteBlobs() doctorCheck {
	return checkDuplicateNoteBlobsFromList(loadNotesList())
}

func checkDuplicateNoteBlobsFromList(notesList map[string][]string) doctorCheck {
	if len(notesList) == 0 {
		return doctorCheck{Name: "note-blobs", OK: true, Detail: "Note blobs — no notes to check"}
	}

	// notesList is already blobHash→[]commitSHA; blobs with >1 commit are duplicates.
	var dupBlobs, dupCommits int
	var examples []string
	for _, commitSHAs := range notesList {
		if len(commitSHAs) > 1 {
			dupBlobs++
			dupCommits += len(commitSHAs)
			if len(examples) < 2 {
				// Truncate commit SHAs to 7 chars for display.
				shorts := make([]string, len(commitSHAs))
				for i, s := range commitSHAs {
					if len(s) > 7 {
						s = s[:7]
					}
					shorts[i] = s
				}
				examples = append(examples, fmt.Sprintf("%s (shared by %s)", shorts[0], strings.Join(shorts[1:minInt(len(shorts), 4)], ", ")))
			}
		}
	}

	if dupBlobs == 0 {
		return doctorCheck{Name: "note-blobs", OK: true, Detail: "Note blobs — all unique (Studio Pro generated)"}
	}
	detail := fmt.Sprintf("Note blobs — %d blob(s) shared across %d commits (raw-git suspected): %s",
		dupBlobs, dupCommits, strings.Join(examples, "; "))
	if dupBlobs > 2 {
		detail += fmt.Sprintf(" … (+%d more)", dupBlobs-2)
	}
	return doctorCheck{Name: "note-blobs", OK: false, Detail: detail}
}

// notedCommits returns the set of commit SHAs that have an mx_metadata note.
func notedCommits() map[string]bool {
	listCmd := gitExecCommand("git", "notes", "--ref=mx_metadata", "list")
	listOut, _ := listCmd.Output()
	noted := map[string]bool{}
	for _, line := range strings.Split(string(listOut), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 {
			noted[parts[1]] = true
		}
	}
	return noted
}

// minInt returns the smaller of a and b. Named to avoid conflict with Go 1.21+ builtin min.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ──────────────────────────────────────────────
// Fix command
// ──────────────────────────────────────────────

var gitFixCmd = &cobra.Command{
	Use:   "fix",
	Short: "Repair Mendix git compatibility issues",
	Long: `Fix all issues detected by 'mxcli git doctor':

  1. Add missing git local config (core.autocrlf, mendix.*)
  2. Convert SSH remote URL to HTTPS (auto, no confirmation required)
  3. Add mx_metadata notes to commits that lack them
  4. Repair malformed (invalid JSON) notes with -f override
  5. Push notes to remote

After fix, restart Mendix Studio Pro to verify.

Examples:
  mxcli git fix
  mxcli git fix --version 10.6.0.0
  mxcli git fix --remote origin
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		remote, _ := cmd.Flags().GetString("remote")
		versionFlag, _ := cmd.Flags().GetString("version")
		projectPath, _ := cmd.Flags().GetString("project")
		return runGitFix(projectPath, versionFlag, remote, cmd.OutOrStdout())
	},
}

func runGitFix(projectPath, versionFlag, remoteOverride string, out io.Writer) error {
	remote := resolveRemote(remoteOverride)
	fmt.Fprintf(out, "Fixing Git repo\n\n")

	// Step 1: Git config
	fmt.Fprintf(out, "Step 1: Git local config\n")
	fixGitConfig(out)

	// Step 2: Remote URL
	fmt.Fprintf(out, "Step 2: Remote URL\n")
	fixRemoteURL(remote, out)

	// Step 3: Missing notes
	fmt.Fprintf(out, "Step 3: mx_metadata notes\n")
	mendixVersion, mprFmtVersion, err := detectMendixVersion(versionFlag, projectPath)
	if err != nil {
		fmt.Fprintf(out, "  WARNING: %v\n  Skipping notes repair.\n", err)
	} else {
		fixed, skipped := fixMissingNotes(mendixVersion, mprFmtVersion, out)
		fmt.Fprintf(out, "  Fixed: %d, Skipped: %d (already valid)\n", fixed, skipped)
	}

	// Step 4: Push notes
	if remote != "" {
		fmt.Fprintf(out, "Step 4: Push notes to %s\n", remote)
		if err := runGitNotesPush(remoteOverride, false, out); err != nil {
			fmt.Fprintf(out, "  Push failed: %v\n  Retry with: mxcli git notes push --force\n", err)
		}
	}

	fmt.Fprintf(out, "\nDone! Restart Mendix Studio Pro to verify.\n")
	return nil
}

func fixGitConfig(out io.Writer) {
	configs := map[string]string{
		"core.autocrlf":              "False",
		"mendix.commits-since-gc":    "0",
		"mendix.lineEndingResetDone": "True",
	}
	keys := make([]string, 0, len(configs))
	for k := range configs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		val := configs[key]
		getCmd := gitExecCommand("git", "config", "--local", key)
		current, err := getCmd.Output()
		if err != nil || strings.TrimSpace(string(current)) == "" {
			setCmd := gitExecCommand("git", "config", "--local", key, val)
			if err := setCmd.Run(); err != nil {
				fmt.Fprintf(out, "  FAILED to set %s: %v\n", key, err)
			} else {
				fmt.Fprintf(out, "  Set: %s = %s\n", key, val)
			}
		} else {
			fmt.Fprintf(out, "  OK:  %s = %s\n", key, strings.TrimSpace(string(current)))
		}
	}
}

var sshURLRe = regexp.MustCompile(`^git@([^:]+):(.+?)(?:\.git)?$`)

func fixRemoteURL(remote string, out io.Writer) {
	if remote == "" {
		fmt.Fprintf(out, "  No remote found, skipping.\n")
		return
	}
	getCmd := gitExecCommand("git", "remote", "get-url", remote)
	urlOut, err := getCmd.Output()
	if err != nil {
		fmt.Fprintf(out, "  Cannot get URL for '%s'\n", remote)
		return
	}
	url := strings.TrimSpace(string(urlOut))
	m := sshURLRe.FindStringSubmatch(url)
	if m == nil {
		fmt.Fprintf(out, "  OK:  %s uses HTTPS\n", remote)
		return
	}
	httpsURL := fmt.Sprintf("https://%s/%s", m[1], m[2])
	setCmd := gitExecCommand("git", "remote", "set-url", remote, httpsURL)
	if err := setCmd.Run(); err != nil {
		fmt.Fprintf(out, "  Failed to convert SSH URL: %v\n", err)
		return
	}
	fmt.Fprintf(out, "  Converted: %s → %s\n", url, httpsURL)
}

func fixMissingNotes(mendixVersion, mprFmtVersion string, out io.Writer) (fixed, skipped int) {
	logCmd := gitExecCommand("git", "log", "--format=%H")
	logOut, err := logCmd.Output()
	if err != nil || len(strings.TrimSpace(string(logOut))) == 0 {
		return 0, 0
	}
	allCommits := strings.Split(strings.TrimSpace(string(logOut)), "\n")

	noted := notedCommits()

	metadata := buildMxMetadata(mendixVersion, mprFmtVersion)
	for _, sha := range allCommits {
		sha = strings.TrimSpace(sha)
		if sha == "" {
			continue
		}
		short := sha[:minInt(len(sha), 7)]
		if noted[sha] {
			// Verify JSON is valid; if not, repair with force.
			showCmd := gitExecCommand("git", "notes", "--ref=mx_metadata", "show", sha)
			noteOut, err := showCmd.Output()
			if err == nil {
				var m map[string]any
				if json.Unmarshal(noteOut, &m) == nil {
					skipped++
					continue
				}
			}
			// Malformed — force overwrite.
			fmt.Fprintf(out, "  Repair: %s (malformed JSON)\n", short)
			if err := hashObjectAndAddNote(sha, metadata); err == nil {
				fixed++
			}
		} else {
			fmt.Fprintf(out, "  Add: %s\n", short)
			if err := hashObjectAndAddNote(sha, metadata); err == nil {
				fixed++
			}
		}
	}
	return fixed, skipped
}

// ──────────────────────────────────────────────
// Wiring
// ──────────────────────────────────────────────

func init() {
	// Flags (gitCommitCmd uses DisableFlagParsing, so no flags registered on it)
	gitNotesPushCmd.Flags().String("remote", "", "Remote name (default: auto-detect tracking remote, then 'origin')")
	gitNotesPushCmd.Flags().Bool("force", false, "Force push (overwrites remote notes)")
	gitDoctorCmd.Flags().String("remote", "", "Remote to check (default: auto-detect)")
	gitFixCmd.Flags().String("remote", "", "Remote name (default: auto-detect)")
	gitFixCmd.Flags().String("version", "", "Mendix version number (e.g. 10.6.0.0)")

	// Subcommand tree
	gitNotesCmd.AddCommand(gitNotesPushCmd)
	gitCmd.AddCommand(gitCommitCmd)
	gitCmd.AddCommand(gitNotesCmd)
	gitCmd.AddCommand(gitDoctorCmd)
	gitCmd.AddCommand(gitFixCmd)
}

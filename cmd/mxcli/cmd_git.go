// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
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

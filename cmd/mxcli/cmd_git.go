// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"os/exec"
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

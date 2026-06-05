// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
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

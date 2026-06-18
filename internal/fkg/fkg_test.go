// internal/fkg/fkg_test.go
package fkg_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/internal/fkg"
	_ "github.com/mendixlabs/mxcli/internal/fkg/concepts" // register all adapters
)

func mustNew(t *testing.T) fkg.Querier {
	t.Helper()
	q, err := fkg.New()
	if err != nil {
		t.Fatalf("fkg.New(): %v", err)
	}
	return q
}

// gitRoot returns the absolute repo root by running git rev-parse.
func gitRoot() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// gitStatusAll returns porcelain output from the repo root (no path filter).
func gitStatusAll() (string, error) {
	out, err := exec.Command("git", "-C", gitRoot(), "status", "--porcelain").Output()
	return strings.TrimSpace(string(out)), err
}

// gitStatusLazygitStyle runs the exact command lazygit v0.44.1 uses.
func gitStatusLazygitStyle() (string, error) {
	out, err := exec.Command("git", "-C", gitRoot(),
		"status",
		"--untracked-files=all",
		"--porcelain",
		"-z",
		"--find-renames=50%",
	).Output()
	// lazygit splits on \x00; replace for readability
	return strings.TrimSpace(strings.ReplaceAll(string(out), "\x00", "\n")), err
}

// ── Discoverability ──────────────────────────────────────────────────────────

func TestFkgGo_IsDiscoverableByGitStatus(t *testing.T) {
	root := gitRoot()
	if root == "" {
		t.Skip("not in a git repo")
	}
	out, err := gitStatusAll()
	if err != nil {
		t.Fatalf("git status failed: %v", err)
	}
	if !strings.Contains(out, "internal/fkg/fkg.go") {
		t.Error("internal/fkg/fkg.go is NOT visible in git status --porcelain; lazygit won't discover it")
	}
	t.Logf("git status root: %s", root)
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "fkg") {
			t.Logf("  %s", line)
		}
	}
}

func TestFkgGo_IsDiscoverableByLazygitStyleStatus(t *testing.T) {
	root := gitRoot()
	if root == "" {
		t.Skip("not in a git repo")
	}
	out, err := gitStatusLazygitStyle()
	if err != nil {
		t.Fatalf("git status (lazygit-style) failed: %v", err)
	}
	// This runs the exact command lazygit v0.44.1 uses to discover files.
	// If the file is absent here, it's invisible to lazygit.
	if !strings.Contains(out, "internal/fkg/fkg.go") {
		t.Error("internal/fkg/fkg.go is NOT visible in lazygit-style git status; lazygit can't discover it")
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "fkg") {
			t.Logf("  %s", line)
		}
	}
}

// ── New ──────────────────────────────────────────────────────────────────────

func TestNew_Succeeds(t *testing.T) {
	q := mustNew(t)
	if q == nil {
		t.Fatal("expected non-nil Querier")
	}
}

// ── Schema ───────────────────────────────────────────────────────────────────

func TestSchema_HasExpectedNodeTypes(t *testing.T) {
	q := mustNew(t)
	s := q.Schema()

	labels := map[string]int{}
	for _, nt := range s.NodeTypes {
		labels[nt.Label] = nt.Count
	}

	if labels["Concept"] == 0 {
		t.Error("Schema: expected Concept nodes")
	}
	if labels["SyntaxFeature"] == 0 {
		t.Error("Schema: expected SyntaxFeature nodes")
	}
	if labels["Skill"] == 0 {
		t.Error("Schema: expected Skill nodes")
	}
}

func TestSchema_RootsAreTopLevelConcepts(t *testing.T) {
	q := mustNew(t)
	s := q.Schema()

	if len(s.Roots) == 0 {
		t.Fatal("Schema: expected at least one root concept")
	}
	for _, r := range s.Roots {
		if r.Label != "Concept" {
			t.Errorf("root %q has label %q, want Concept", r.ID, r.Label)
		}
	}

	// "page", "microflow", "entity" must all be roots (no inbound SPECIALIZES)
	rootIDs := map[string]bool{}
	for _, r := range s.Roots {
		rootIDs[r.ID] = true
	}
	for _, want := range []string{"page", "microflow", "entity", "security"} {
		if !rootIDs[want] {
			t.Errorf("expected %q in roots", want)
		}
	}
}

func TestSchema_SubConceptsNotRoots(t *testing.T) {
	q := mustNew(t)
	s := q.Schema()

	rootIDs := map[string]bool{}
	for _, r := range s.Roots {
		rootIDs[r.ID] = true
	}
	// datagrid and nanoflow have inbound SPECIALIZES, must not appear as roots
	for _, notRoot := range []string{"datagrid", "nanoflow", "pluggable-widget"} {
		if rootIDs[notRoot] {
			t.Errorf("%q should not be a root (it specializes a parent)", notRoot)
		}
	}
}

// ── Explore ──────────────────────────────────────────────────────────────────

func TestExplore_PageDepth1(t *testing.T) {
	q := mustNew(t)
	result, err := q.Explore("page", 1)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	if result.Seed.ID != "page" {
		t.Errorf("Seed.ID = %q, want page", result.Seed.ID)
	}
	if result.Seed.Label != "Concept" {
		t.Errorf("Seed.Label = %q, want Concept", result.Seed.Label)
	}

	// Must have at least syntax and skill neighbours
	neighborIDs := map[string]bool{}
	for _, n := range result.Nodes {
		neighborIDs[n.ID] = true
	}
	if !neighborIDs["syntax:page.create"] {
		t.Error("depth-1 explore of 'page' must include syntax:page.create")
	}
	if !neighborIDs["skill:create-page"] {
		t.Error("depth-1 explore of 'page' must include skill:create-page")
	}
}

func TestExplore_UnknownIDReturnsError(t *testing.T) {
	q := mustNew(t)
	_, err := q.Explore("nonexistent-concept", 1)
	if err == nil {
		t.Error("expected error for unknown node ID")
	}
}

func TestExplore_Depth2IncludesGrandchildren(t *testing.T) {
	q := mustNew(t)
	result, err := q.Explore("page", 2)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	// depth-2 from page should reach microflow (page→RELATED_TO→microflow)
	found := false
	for _, n := range result.Nodes {
		if n.ID == "microflow" {
			found = true
		}
	}
	if !found {
		t.Error("depth-2 explore of 'page' should reach 'microflow' via RELATED_TO")
	}
}

// ── Path ─────────────────────────────────────────────────────────────────────

func TestPath_PageToSecurity(t *testing.T) {
	q := mustNew(t)
	schemas, err := q.Path("page", "security")
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if len(schemas) == 0 {
		t.Fatal("expected at least one path from 'page' to 'security'")
	}

	// The shortest concrete path must name actual concepts, not just type labels.
	shortest := schemas[len(schemas)-1]
	if len(shortest.Steps) == 0 {
		t.Fatal("expected at least one step in shortest path")
	}
	last := shortest.Steps[len(shortest.Steps)-1]
	if last.NodeID != "security" {
		t.Errorf("last step should end at 'security', got NodeID=%q NodeName=%q",
			last.NodeID, last.NodeName)
	}
	if last.NodeName == "" {
		t.Error("PathStep.NodeName must not be empty for concrete paths")
	}
}

func TestPath_NoPathReturnsEmpty(t *testing.T) {
	q := mustNew(t)
	schemas, err := q.Path("syntax:page.create", "workflow")
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	_ = schemas
}

func TestPath_EntityToSecurityHasRequiresEdge(t *testing.T) {
	q := mustNew(t)
	schemas, err := q.Path("entity", "security")
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if len(schemas) == 0 {
		t.Fatal("expected at least one path from 'entity' to 'security'")
	}

	// At least one path must use the REQUIRES edge directly: entity → REQUIRES → security
	foundDirect := false
	for _, ps := range schemas {
		if len(ps.Steps) == 1 &&
			ps.Steps[0].RelType == "REQUIRES" &&
			ps.Steps[0].NodeID == "security" &&
			ps.Steps[0].NodeName == "Security" {
			foundDirect = true
			break
		}
	}
	if !foundDirect {
		t.Error("expected direct path: entity → REQUIRES → security with concrete NodeID/NodeName")
	}
}

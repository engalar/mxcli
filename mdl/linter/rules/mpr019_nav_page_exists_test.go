package rules

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog/mock"
	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/mdl/types"
)

// TestNavPageExists_DetectsMissingPage verifies MPR019 detects a navigation
// reference to a page that doesn't exist (CE1613).
func TestNavPageExists_DetectsMissingPage(t *testing.T) {
	g := &mock.MockProjectGraph{
		PagesFunc: func(module string) []graphcatalog.PageNode {
			return []graphcatalog.PageNode{
				{ID: "p1", Name: "ExistingPage", QualifiedName: "Mod.ExistingPage", Module: "Mod"},
			}
		},
		SnippetsFunc: func(module string) []graphcatalog.SnippetNode { return nil },
	}

	// Navigation references a non-existent page.
	reader := &mockNavReader{
		nav: &types.NavigationDocument{
			Profiles: []*types.NavigationProfile{
				{
					Kind: "Responsive",
					MenuItems: []*types.NavMenuItem{
						{Caption: "Unit Tests", Page: "UnitTesting.UnitTestOverview"},
					},
				},
			},
		},
	}

	ctx := newGraphContext(g, reader)
	rule := NewNavPageExistsRule()
	violations := rule.Check(ctx)

	if len(violations) != 1 {
		t.Fatalf("got %d violations, want 1", len(violations))
	}
	v := violations[0]
	if v.RuleID != "MPR019" {
		t.Errorf("RuleID = %q, want MPR019", v.RuleID)
	}
	if !containsStr(v.Message, "UnitTesting.UnitTestOverview") {
		t.Errorf("message = %q, should mention the missing page", v.Message)
	}
}

// TestNavPageExists_NoViolationForExistingPages verifies no false positives
// when navigation references point to existing pages.
func TestNavPageExists_NoViolationForExistingPages(t *testing.T) {
	g := &mock.MockProjectGraph{
		PagesFunc: func(module string) []graphcatalog.PageNode {
			return []graphcatalog.PageNode{
				{ID: "p1", Name: "ExistingPage", QualifiedName: "Mod.ExistingPage", Module: "Mod"},
				{ID: "p2", Name: "AnotherPage", QualifiedName: "Mod.AnotherPage", Module: "Mod"},
			}
		},
		SnippetsFunc: func(module string) []graphcatalog.SnippetNode { return nil },
	}

	reader := &mockNavReader{
		nav: &types.NavigationDocument{
			Profiles: []*types.NavigationProfile{
				{
					Kind: "Responsive",
					HomePage: &types.NavHomePage{Page: "Mod.ExistingPage"},
					MenuItems: []*types.NavMenuItem{
						{Caption: "Test", Page: "Mod.AnotherPage"},
					},
				},
			},
		},
	}

	ctx := newGraphContext(g, reader)
	rule := NewNavPageExistsRule()
	violations := rule.Check(ctx)

	if len(violations) != 0 {
		t.Errorf("got %d violations for correct nav refs, want 0", len(violations))
	}
}

// TestNavPageExists_Deduplicates verifies multiple references to the same
// missing page produce only one violation.
func TestNavPageExists_Deduplicates(t *testing.T) {
	g := &mock.MockProjectGraph{
		PagesFunc: func(module string) []graphcatalog.PageNode { return nil },
		SnippetsFunc: func(module string) []graphcatalog.SnippetNode { return nil },
	}

	reader := &mockNavReader{
		nav: &types.NavigationDocument{
			Profiles: []*types.NavigationProfile{
				{
					Kind: "Responsive",
					HomePage: &types.NavHomePage{Page: "Mod.GonePage"},
					RoleBasedHomePages: []*types.NavRoleBasedHome{
						{Page: "Mod.GonePage", UserRole: "Admin"},
					},
				},
			},
		},
	}

	ctx := newGraphContext(g, reader)
	rule := NewNavPageExistsRule()
	violations := rule.Check(ctx)

	if len(violations) != 1 {
		t.Errorf("got %d violations for same missing page, want 1 (deduplicated)", len(violations))
	}
}

// mockNavReader provides a fake LintReader that returns a custom navigation document.
type mockNavReader struct {
	linter.LintReader
	nav       *types.NavigationDocument
}

func (r *mockNavReader) GetNavigation() (*types.NavigationDocument, error) {
	return r.nav, nil
}

// containsStr checks if needle appears in haystack.
func containsStr(haystack, needle string) bool {
	return len(haystack) >= len(needle) &&
		len(needle) > 0 &&
		containsStrImpl(haystack, needle)
}

func containsStrImpl(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

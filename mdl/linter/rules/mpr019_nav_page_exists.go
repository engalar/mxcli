package rules

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/mdl/types"
)

// NavPageExistsRule (MPR019) checks that every page referenced in navigation
// profiles actually exists in the project. Pages referenced by qualified name
// in navigation (BY_NAME references) are NOT automatically updated when a page
// is renamed — this rule catches stale navigation references to pages that no
// longer exist or were renamed without using mxcli's rename cascade.
type NavPageExistsRule struct{}

// NewNavPageExistsRule creates a new MPR019 rule.
func NewNavPageExistsRule() *NavPageExistsRule {
	return &NavPageExistsRule{}
}

func (r *NavPageExistsRule) ID() string                       { return "MPR019" }
func (r *NavPageExistsRule) Name() string                     { return "NavPageExists" }
func (r *NavPageExistsRule) Category() string                 { return "correctness" }
func (r *NavPageExistsRule) DefaultSeverity() linter.Severity { return linter.SeverityError }

func (r *NavPageExistsRule) Description() string {
	return "Checks that navigation page references point to actual pages — " +
		"stale BY_NAME references cause CE1613 and prevent Studio Pro from loading the navigation"
}

// navPageRef describes a single page reference found in navigation.
type navPageRef struct {
	QualifiedName string
	Profile       string
	Context       string
}

// Check collects all page references from navigation profiles and verifies
// each one points to a page that exists in the project.
func (r *NavPageExistsRule) Check(ctx *linter.LintContext) []linter.Violation {
	reader := ctx.Reader()
	if reader == nil {
		return nil
	}

	nav, err := reader.GetNavigation()
	if err != nil || nav == nil {
		return nil
	}

	// Build set of actual page qualified names in the project
	pages := ctx.Pages()
	actualPages := make(map[string]bool, len(pages))
	for _, p := range pages {
		actualPages[p.QualifiedName] = true
	}

	// Collect all page references from navigation
	var refs []navPageRef

	for _, profile := range nav.Profiles {
		pName := profile.Kind
		if pName == "" {
			pName = profile.Name
		}

		if profile.HomePage != nil && profile.HomePage.Page != "" {
			refs = append(refs, navPageRef{
				QualifiedName: profile.HomePage.Page,
				Profile:       pName, Context: "home page",
			})
		}

		for _, rbh := range profile.RoleBasedHomePages {
			if rbh.Page != "" {
				refs = append(refs, navPageRef{
					QualifiedName: rbh.Page,
					Profile:       pName,
					Context:       fmt.Sprintf("role-based home page for %s", rbh.UserRole),
				})
			}
		}

		if profile.LoginPage != "" {
			refs = append(refs, navPageRef{
				QualifiedName: profile.LoginPage,
				Profile:       pName, Context: "login page",
			})
		}

		if profile.NotFoundPage != "" {
			refs = append(refs, navPageRef{
				QualifiedName: profile.NotFoundPage,
				Profile:       pName, Context: "not-found page",
			})
		}

		collectMenuPageRefs(profile.MenuItems, pName, &refs)
	}

	if len(refs) == 0 {
		return nil
	}

	// Deduplicate: only one violation per missing page
	seen := make(map[string]bool)
	var violations []linter.Violation

	for _, ref := range refs {
		if seen[ref.QualifiedName] {
			continue
		}
		if actualPages[ref.QualifiedName] {
			continue
		}

		moduleName := moduleFromQualified(ref.QualifiedName)
		if ctx.IsExcluded(moduleName) {
			continue
		}
		seen[ref.QualifiedName] = true

		violations = append(violations, linter.Violation{
			RuleID:   r.ID(),
			Severity: r.DefaultSeverity(),
			Message: fmt.Sprintf(
				"Navigation %s references page '%s' which does not exist (CE1613)",
				ref.Context, ref.QualifiedName,
			),
			Location: linter.Location{
				Module:       moduleName,
				DocumentType: "navigation",
				DocumentName: "Navigation",
			},
			Suggestion: fmt.Sprintf(
				"Update the navigation reference: "+
					"the page may have been renamed. Use mxcli rename cascade "+
					"or manually update the Form field in %s navigation profile",
				ref.Profile,
			),
		})
	}

	return violations
}

// collectMenuPageRefs recursively collects page references from menu items.
func collectMenuPageRefs(items []*types.NavMenuItem, profileName string, refs *[]navPageRef) {
	for _, item := range items {
		if item.Page != "" {
			*refs = append(*refs, navPageRef{
				QualifiedName: item.Page,
				Profile:       profileName,
				Context:       fmt.Sprintf("menu item '%s'", item.Caption),
			})
		}
		collectMenuPageRefs(item.Items, profileName, refs)
	}
}



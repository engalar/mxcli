// SPDX-License-Identifier: Apache-2.0

// Package rules contains built-in lint rules.
package rules

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/mendixlabs/mxcli/mdl/linter"
)

// DefaultEntityPattern matches PascalCase entity names.
var DefaultEntityPattern = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)

// DefaultMicroflowPattern matches microflow naming conventions with optional prefix.
// Prefixes from Mendix Best Practices:
//
//	ACT_ (page action), SUB_ (sub-microflow), DS_ (data source), VAL_ (validation),
//	SCH_ (scheduled), IVK_ (invoked), BCO_ (before commit), ACO_ (after commit),
//	BCR_ (before create), ACR_ (after create), BDE_ (before delete), ADE_ (after delete),
//	BRO_ (before rollback), ARO_ (after rollback), OCH_ (on change), SE_ (security/event),
//	DL_ (delete), PWS_ (published web service), ASU_ (after startup), NAV_ (navigation),
//	LOGIN_ (login)
var DefaultMicroflowPattern = regexp.MustCompile(`^(ACT_|SUB_|DS_|VAL_|SCH_|IVK_|BCO_|ACO_|BCR_|ACR_|BDE_|ADE_|BRO_|ARO_|OCH_|SE_|DL_|PWS_|ASU_|NAV_|LOGIN_)?[A-Z][a-zA-Z0-9_]*$`)

// DefaultPagePattern matches page naming conventions.
var DefaultPagePattern = regexp.MustCompile(`^[A-Z][a-zA-Z0-9_]*$`)

// DefaultEnumerationPattern matches enumeration naming conventions.
var DefaultEnumerationPattern = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)

// NamingConventionRule checks naming conventions for entities, microflows, etc.
type NamingConventionRule struct {
	EntityPattern      *regexp.Regexp
	MicroflowPattern   *regexp.Regexp
	PagePattern        *regexp.Regexp
	EnumerationPattern *regexp.Regexp
}

// NewNamingConventionRule creates a new naming convention rule with default patterns.
func NewNamingConventionRule() *NamingConventionRule {
	return &NamingConventionRule{
		EntityPattern:      DefaultEntityPattern,
		MicroflowPattern:   DefaultMicroflowPattern,
		PagePattern:        DefaultPagePattern,
		EnumerationPattern: DefaultEnumerationPattern,
	}
}

func (r *NamingConventionRule) ID() string                       { return "MPR001" }
func (r *NamingConventionRule) Name() string                     { return "NamingConvention" }
func (r *NamingConventionRule) Category() string                 { return "style" }
func (r *NamingConventionRule) DefaultSeverity() linter.Severity { return linter.SeverityWarning }

func (r *NamingConventionRule) Description() string {
	return "Checks that entities, microflows, pages, and enumerations follow naming conventions"
}

// Check runs the naming convention checks.
func (r *NamingConventionRule) Check(ctx *linter.LintContext) []linter.Violation {
	var violations []linter.Violation

	// Check entity names
	for _, entity := range ctx.Entities() {
		if !r.EntityPattern.MatchString(entity.Name) {
			violations = append(violations, linter.Violation{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("Entity name '%s' should use PascalCase", entity.Name),
				Location: linter.Location{
					Module:       entity.Module,
					DocumentType: "entity",
					DocumentName: entity.Name,
					DocumentID:   entity.ID,
				},
				Suggestion: toPascalCase(entity.Name),
			})
		}
	}

	// Check microflow names
	for _, mf := range ctx.Microflows() {
		if !r.MicroflowPattern.MatchString(mf.Name) {
			violations = append(violations, linter.Violation{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("Microflow name '%s' should use PascalCase with optional prefix (ACT_, SUB_, DS_, VAL_, SCH_, IVK_, BCO_, ACO_, BCR_, ACR_, BDE_, ADE_, BRO_, ARO_, OCH_, SE_, DL_, PWS_, ASU_, NAV_, LOGIN_)", mf.Name),
				Location: linter.Location{
					Module:       mf.Module,
					DocumentType: "microflow",
					DocumentName: mf.Name,
					DocumentID:   mf.ID,
				},
				Suggestion: suggestMicroflowName(mf.Name),
			})
		}
	}

	// Check page names
	for _, page := range ctx.Pages() {
		if !r.PagePattern.MatchString(page.Name) {
			violations = append(violations, linter.Violation{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("Page name '%s' should use PascalCase", page.Name),
				Location: linter.Location{
					Module:       page.Module,
					DocumentType: "page",
					DocumentName: page.Name,
					DocumentID:   page.ID,
				},
				Suggestion: toPascalCase(page.Name),
			})
		}
	}

	// Check enumeration names
	for _, enum := range ctx.Enumerations() {
		if !r.EnumerationPattern.MatchString(enum.Name) {
			violations = append(violations, linter.Violation{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("Enumeration name '%s' should use PascalCase", enum.Name),
				Location: linter.Location{
					Module:       enum.Module,
					DocumentType: "enumeration",
					DocumentName: enum.Name,
					DocumentID:   enum.ID,
				},
				Suggestion: toPascalCase(enum.Name),
			})
		}
	}

	return violations
}

// toPascalCase converts a string to PascalCase.
func toPascalCase(s string) string {
	if s == "" {
		return s
	}

	// Split on non-alphanumeric characters
	words := splitWords(s)
	if len(words) == 0 {
		return s
	}

	var result strings.Builder
	for _, word := range words {
		if word == "" {
			continue
		}
		// Capitalize first letter, lowercase rest
		runes := []rune(word)
		runes[0] = unicode.ToUpper(runes[0])
		for i := 1; i < len(runes); i++ {
			runes[i] = unicode.ToLower(runes[i])
		}
		result.WriteString(string(runes))
	}

	return result.String()
}

// splitWords splits a string into words based on common separators and case changes.
func splitWords(s string) []string {
	var words []string
	var currentWord strings.Builder

	runes := []rune(s)
	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			// Non-alphanumeric: end current word
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
			continue
		}

		if i > 0 && unicode.IsUpper(r) && unicode.IsLower(runes[i-1]) {
			// camelCase boundary
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
		}

		currentWord.WriteRune(r)
	}

	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}

	return words
}

// suggestMicroflowName suggests a better microflow name.
func suggestMicroflowName(name string) string {
	// Try to preserve common prefixes
	prefixes := []string{"ACT_", "SUB_", "DS_", "VAL_", "SCH_", "IVK_",
		"BCO_", "ACO_", "BCR_", "ACR_", "BDE_", "ADE_",
		"BRO_", "ARO_", "OCH_", "SE_", "DL_", "PWS_",
		"ASU_", "NAV_", "LOGIN_"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToUpper(name), prefix) {
			rest := name[len(prefix):]
			return prefix + toPascalCase(rest)
		}
	}
	return toPascalCase(name)
}

// IsPascalCase checks if a string is in PascalCase.
func IsPascalCase(s string) bool {
	if s == "" {
		return false
	}
	runes := []rune(s)
	if !unicode.IsUpper(runes[0]) {
		return false
	}
	for _, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// IsCamelCase checks if a string is in camelCase.
func IsCamelCase(s string) bool {
	if s == "" {
		return false
	}
	runes := []rune(s)
	if !unicode.IsLower(runes[0]) {
		return false
	}
	for _, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

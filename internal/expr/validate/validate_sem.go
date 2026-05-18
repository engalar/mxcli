// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/scan"
)

// IndexReader is the minimal metadata surface required by semantic rules.
// meta.Index and meta.MockIndex both satisfy it.
type IndexReader interface {
	EnumCases(enumQN string) ([]string, bool)
	HasConstant(ref string) bool
	HasEntity(entityQN string) bool
}

// ValidateSemantic applies SEM-04/05/07 rules to a parse result.
// When idx is nil (no-daemon mode without an opened MPR), returns nil.
func ValidateSemantic(pr parse.ParseResult, idx IndexReader) []ValidationResult {
	if idx == nil {
		return nil
	}
	var out []ValidationResult
	rec := pr.Record

	out = append(out, checkEnumRefs(rec.Raw, rec, idx)...)
	out = append(out, checkConstantRefs(rec.Raw, rec, idx)...)

	if strings.HasPrefix(strings.TrimSpace(rec.Raw), "[") {
		out = append(out, checkXPathEntities(rec.Raw, rec, idx)...)
	}

	return out
}

// enumRefPattern matches Module.Enum.Value triples.
// Module/Enum must start with uppercase; Value can start with uppercase
// or underscore-prefixed identifier (lookbehind not supported in RE2,
// so we anchor on word boundaries).
var enumRefPattern = regexp.MustCompile(`\b([A-Z][A-Za-z0-9_]*)\.([A-Z][A-Za-z0-9_]*)\.([A-Z][A-Za-z0-9_][A-Za-z0-9_]*)\b`)

func checkEnumRefs(raw string, rec scan.ExprRecord, idx IndexReader) []ValidationResult {
	var out []ValidationResult
	for _, m := range enumRefPattern.FindAllStringSubmatch(raw, -1) {
		moduleName, enumName, valueName := m[1], m[2], m[3]
		enumQN := moduleName + "." + enumName
		vals, ok := idx.EnumCases(enumQN)
		if !ok {
			continue
		}
		found := false
		for _, v := range vals {
			if v == valueName {
				found = true
				break
			}
		}
		if !found {
			out = append(out, ValidationResult{
				UnitID: rec.UnitID, Project: rec.Project, UnitType: rec.UnitType,
				Field: rec.Field, Raw: raw,
				RuleID:   "SEM-04",
				Severity: "ERROR",
				Message:  fmt.Sprintf("Enum value '%s.%s.%s' not found in '%s'.", moduleName, enumName, valueName, enumQN),
				Fix:      fmt.Sprintf("Available values: %s", strings.Join(vals, ", ")),
			})
		}
	}
	return out
}

// constantRefPattern matches @Module.Name references.
var constantRefPattern = regexp.MustCompile(`@([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)`)

func checkConstantRefs(raw string, rec scan.ExprRecord, idx IndexReader) []ValidationResult {
	var out []ValidationResult
	for _, m := range constantRefPattern.FindAllString(raw, -1) {
		if !idx.HasConstant(m) {
			out = append(out, ValidationResult{
				UnitID: rec.UnitID, Project: rec.Project, UnitType: rec.UnitType,
				Field: rec.Field, Raw: raw,
				RuleID:   "SEM-05",
				Severity: "ERROR",
				Message:  fmt.Sprintf("Constant '%s' not found in project.", m),
				Fix:      "Check the constant name and module — it may have been renamed or the module changed.",
			})
		}
	}
	return out
}

// xpathEntityPattern matches Module.Entity within an XPath constraint.
var xpathEntityPattern = regexp.MustCompile(`\b([A-Z][A-Za-z0-9_]*)\.([A-Z][A-Za-z0-9_]*)\b`)

func checkXPathEntities(raw string, rec scan.ExprRecord, idx IndexReader) []ValidationResult {
	inner := strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(raw), "]"), "[")
	var out []ValidationResult
	seen := map[string]bool{}
	for _, m := range xpathEntityPattern.FindAllStringSubmatch(inner, -1) {
		candidate := m[1] + "." + m[2]
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		if strings.HasPrefix(m[1], "System") {
			continue
		}
		if _, isEnum := idx.EnumCases(candidate); isEnum {
			continue
		}
		if !idx.HasEntity(candidate) {
			// Be conservative: only flag heuristically entity-shaped names.
			// Short identifiers like "X.Y" are more likely typos than entities.
			if len(candidate) > 10 || strings.Contains(candidate, "_") {
				out = append(out, ValidationResult{
					UnitID: rec.UnitID, Project: rec.Project, UnitType: rec.UnitType,
					Field: rec.Field, Raw: raw,
					RuleID:   "SEM-07",
					Severity: "WARNING",
					Message:  fmt.Sprintf("XPath entity '%s' not found in domain model.", candidate),
					Fix:      "Verify the entity qualified name (Module.EntityName) is correct.",
				})
			}
		}
	}
	return out
}

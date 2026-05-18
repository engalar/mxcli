// Package validate maps exprcheck Hints to structured ValidationResult values,
// applying the SYN-01~03 rules documented in the MEMV design spec.
package validate

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/mdl/exprcheck/hints"
)

// ValidationResult is one validation finding for an expression.
type ValidationResult struct {
	UnitID   string `json:"UnitID"`
	Project  string `json:"Project,omitempty"`
	UnitType string `json:"UnitType"`
	Field    string `json:"Field"`
	Raw      string `json:"Raw"`
	RuleID   string `json:"RuleID"`
	Severity string `json:"Severity"`
	Message  string `json:"Message"`
	YouWrote string `json:"YouWrote,omitempty"`
	Fix      string `json:"Fix,omitempty"`
}

// ValidateSyntax applies SYN rules to a ParseResult.
// It also surfaces all exprcheck Hint codes directly.
func ValidateSyntax(pr parse.ParseResult) []ValidationResult {
	var out []ValidationResult
	rec := pr.Record

	// SYN-02: URL stored in expression field (not an expression)
	if strings.HasPrefix(rec.Raw, "https://") || strings.HasPrefix(rec.Raw, "http://") {
		out = append(out, ValidationResult{
			UnitID: rec.UnitID, Project: rec.Project, UnitType: rec.UnitType,
			Field: rec.Field, Raw: rec.Raw, 
			RuleID: "SYN-02", Severity: "INFO",
			Message: "Field contains a URL, not a Mendix expression",
		})
		return out
	}

	// SYN-03: if-then without else (heuristic on raw text).
	// Normalize all whitespace to single space before checking: Mendix developers
	// write else on new lines with tabs (e.g. "if X then Y\n\telse Z"), which
	// would otherwise defeat a simple " else " substring check.
	wsNorm := strings.NewReplacer("\n", " ", "\t", " ", "\r", " ")
	normalized := strings.ToLower(wsNorm.Replace(rec.Raw))
	// Collapse runs of spaces to avoid " else " being split by double-spaces
	for strings.Contains(normalized, "  ") {
		normalized = strings.ReplaceAll(normalized, "  ", " ")
	}
	// Skip SYN-03 for plain string literals: English text inside quotes often
	// contains "if ... then ..." as natural language, not Mendix keywords.
	isStringLiteral := strings.HasPrefix(strings.TrimSpace(normalized), "'")
	if !isStringLiteral &&
		strings.Contains(normalized, "if ") && strings.Contains(normalized, " then ") &&
		!strings.Contains(normalized, " else ") {
		out = append(out, ValidationResult{
			UnitID: rec.UnitID, Project: rec.Project, UnitType: rec.UnitType,
			Field: rec.Field, Raw: rec.Raw,
			RuleID: "SYN-03", Severity: "WARNING",
			Message: "if-then expression is missing else branch (Mendix requires else)",
			Fix:     "Add 'else <value>' after the then branch",
		})
	}

	// Surface all exprcheck hints as SYN-01 (parse errors) or their own code
	for _, h := range pr.Hints {
		sev := severityString(h.Severity)
		ruleID := h.Code
		if ruleID == "" {
			ruleID = "SYN-01"
		}
		out = append(out, ValidationResult{
			UnitID: rec.UnitID, Project: rec.Project, UnitType: rec.UnitType,
			Field: rec.Field, Raw: rec.Raw, 
			RuleID:   ruleID,
			Severity: sev,
			Message:  h.Problem,
			YouWrote: h.YouWrote,
			Fix:      h.Fix,
		})
	}

	return out
}

func severityString(s hints.Severity) string {
	switch s {
	case hints.SeverityError:
		return "ERROR"
	case hints.SeverityWarning:
		return "WARNING"
	default:
		return "INFO"
	}
}

// Summary returns a one-line summary of a ValidationResult for text output.
func (v ValidationResult) Summary() string {
	return fmt.Sprintf("[%s] %s: %s (in %s.%s)", v.Severity, v.RuleID, v.Message, v.UnitType, v.Field)
}

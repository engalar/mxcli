// Package repair matches ValidationResults to concrete repair suggestions.
// R-01 through R-06 patterns correspond to the MEMV design spec.
package repair

import (
	"regexp"
	"strings"

	"github.com/mendixlabs/mxcli/internal/expr/validate"
)

// RepairSuggestion is one ranked repair candidate.
type RepairSuggestion struct {
	PatternID  string  // R-01 .. R-10
	Before     string  // original expression
	After      string  // suggested replacement
	Confidence float64 // 0.0 – 1.0
	Note       string
}

// Suggest returns ranked repair candidates for a single ValidationResult.
// Returns nil if no pattern matches.
func Suggest(v validate.ValidationResult) []RepairSuggestion {
	switch v.RuleID {
	case "SYN-01":
		return repairSYN01(v.Raw)
	case "SYN-03":
		return repairSYN03(v.Raw)
	case "E011": // not without parens — from exprcheck
		return repairE011(v.Raw)
	}
	// Surface exprcheck Fix text as a generic suggestion when available
	if v.Fix != "" && v.RuleID != "SYN-02" {
		return []RepairSuggestion{{
			PatternID:  "R-" + v.RuleID,
			Before:     v.Raw,
			After:      v.Fix,
			Confidence: 0.60,
			Note:       v.Fix,
		}}
	}
	return nil
}

// R-01: keyword-adjacent token (missing whitespace).
var gluedKW = regexp.MustCompile(`(empty|true|false)(or|and)|(or|and)(empty|true|false|\$)`)

func repairSYN01(raw string) []RepairSuggestion {
	if !gluedKW.MatchString(raw) {
		return nil
	}
	fixed := raw
	for _, kw := range []string{"empty", "true", "false"} {
		fixed = strings.ReplaceAll(fixed, kw+"or", kw+" or")
		fixed = strings.ReplaceAll(fixed, kw+"and", kw+" and")
		fixed = strings.ReplaceAll(fixed, "or"+kw, "or "+kw)
		fixed = strings.ReplaceAll(fixed, "and"+kw, "and "+kw)
	}
	if fixed == raw {
		return nil
	}
	return []RepairSuggestion{{
		PatternID:  "R-01",
		Before:     raw,
		After:      fixed,
		Confidence: 0.95,
		Note:       "Inserted spaces between keyword and adjacent token",
	}}
}

// R-06: if-then missing else branch.
func repairSYN03(raw string) []RepairSuggestion {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(strings.ToLower(trimmed), "if ") {
		return nil
	}
	return []RepairSuggestion{{
		PatternID:  "R-06",
		Before:     raw,
		After:      raw + " else empty",
		Confidence: 0.90,
		Note:       "Added 'else empty' — replace 'empty' with the desired value",
	}}
}

// R-E011: not without parentheses (exprcheck E011).
func repairE011(raw string) []RepairSuggestion {
	lower := strings.TrimSpace(raw)
	if !strings.HasPrefix(lower, "not ") {
		return nil
	}
	inner := strings.TrimPrefix(lower, "not ")
	return []RepairSuggestion{{
		PatternID:  "R-E011",
		Before:     raw,
		After:      "not(" + inner + ")",
		Confidence: 0.92,
		Note:       "Wrap operand in parentheses: Mendix requires not(expr) form",
	}}
}

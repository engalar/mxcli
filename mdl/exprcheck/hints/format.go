// SPDX-License-Identifier: Apache-2.0

package hints

import (
	"encoding/json"
	"fmt"
	"strings"
)

func SeverityString(s Severity) string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	}
	return "unknown"
}

func FormatText(h Hint) string {
	var b strings.Builder
	fmt.Fprintf(&b, "HINT [%s %s] %s\n", h.Code, h.Slug, SeverityString(h.Severity))
	fmt.Fprintf(&b, "  WHERE:\n    %s line %d, in %s of microflow\n    %s\n",
		h.Where.File, h.Where.Line, h.Where.Context, h.Where.Microflow)
	fmt.Fprintf(&b, "\n  YOU WROTE:\n    %s\n", h.YouWrote)
	fmt.Fprintf(&b, "\n  PROBLEM:\n    %s\n", indent(h.Problem))
	fmt.Fprintf(&b, "\n  FIX:\n    %s\n", h.Fix)
	if h.Reference != nil && len(h.Reference.EnumValues) > 0 {
		fmt.Fprintf(&b, "\n  LEGAL VALUES for %s:\n    %s\n",
			h.Reference.Enum, strings.Join(h.Reference.EnumValues, ", "))
	}
	return b.String()
}

func indent(s string) string {
	return strings.ReplaceAll(s, "\n", "\n    ")
}

func FormatJSON(h Hint) string {
	payload := map[string]any{
		"code":     h.Code,
		"slug":     h.Slug,
		"severity": SeverityString(h.Severity),
		"where": map[string]any{
			"file":      h.Where.File,
			"line":      h.Where.Line,
			"column":    h.Where.Column,
			"microflow": h.Where.Microflow,
			"context":   h.Where.Context,
		},
		"you_wrote": h.YouWrote,
		"problem":   h.Problem,
		"fix":       h.Fix,
	}
	if h.Reference != nil {
		payload["reference"] = h.Reference
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	return string(b)
}

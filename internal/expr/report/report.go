// Package report assembles ValidationResult slices into HTML, JSON, or text output.
package report

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/internal/expr/validate"
)

// Options controls report format and filtering.
type Options struct {
	Format   string // "json" | "html" | "text"  (default: "json")
	Severity string // filter: "" = all, "ERROR", "WARNING", "INFO"
}

// Render assembles a report from validation results.
func Render(issues []validate.ValidationResult, opts Options) ([]byte, error) {
	filtered := filterBySeverity(issues, opts.Severity)
	switch opts.Format {
	case "html":
		return []byte(renderHTML(filtered)), nil
	case "text":
		return []byte(renderText(filtered)), nil
	default: // "json"
		return json.MarshalIndent(filtered, "", "  ")
	}
}

func filterBySeverity(issues []validate.ValidationResult, sev string) []validate.ValidationResult {
	if sev == "" {
		return issues
	}
	out := make([]validate.ValidationResult, 0, len(issues))
	for _, i := range issues {
		if i.Severity == sev {
			out = append(out, i)
		}
	}
	return out
}

func renderText(issues []validate.ValidationResult) string {
	var sb strings.Builder
	counts := map[string]int{}
	for _, i := range issues {
		counts[i.Severity]++
		fmt.Fprintf(&sb, "[%s] %s: %s\n", i.Severity, i.RuleID, i.Message)
		if i.Fix != "" {
			fmt.Fprintf(&sb, "  Fix: %s\n", i.Fix)
		}
	}
	fmt.Fprintf(&sb, "\nTotal: %d issues  ERROR:%d  WARNING:%d  INFO:%d\n",
		len(issues), counts["ERROR"], counts["WARNING"], counts["INFO"])
	return sb.String()
}

func renderHTML(issues []validate.ValidationResult) string {
	counts := map[string]int{}
	for _, i := range issues {
		counts[i.Severity]++
	}
	var rows strings.Builder
	severityColor := map[string]string{
		"ERROR": "#f85149", "WARNING": "#ffa657", "INFO": "#8b949e",
	}
	for _, i := range issues {
		color := severityColor[i.Severity]
		fix := ""
		if i.Fix != "" {
			fix = fmt.Sprintf(`<br><small style="color:#56d364">Fix: %s</small>`, htmlEsc(i.Fix))
		}
		fmt.Fprintf(&rows, `<tr>
<td style="color:%s;font-weight:600;white-space:nowrap">%s</td>
<td><code>%s</code></td>
<td>%s%s</td>
<td style="color:#8b949e;font-size:11px;word-break:break-all">%s</td>
<td style="color:#8b949e;font-size:11px">%s</td>
</tr>`, color, i.Severity, htmlEsc(i.RuleID),
			htmlEsc(i.Message), fix,
			htmlEsc(shortRaw(i.Raw, 60)),
			htmlEsc(i.UnitType))
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN"><head><meta charset="UTF-8">
<title>MEMV Validation Report</title>
<style>
body{background:#0d1117;color:#f0f6fc;font-family:-apple-system,'Segoe UI',sans-serif;padding:32px;font-size:13px}
h1{font-size:20px;margin-bottom:4px}
.summary{color:#8b949e;margin-bottom:24px}
table{border-collapse:collapse;width:100%%;font-size:12px}
th{background:#161b22;color:#8b949e;padding:8px 10px;text-align:left;border-bottom:1px solid #30363d;font-weight:600;font-size:11px;text-transform:uppercase;letter-spacing:.05em}
td{padding:8px 10px;border-bottom:1px solid #21262d;vertical-align:top}
code{background:#161b22;padding:1px 5px;border-radius:3px}
</style></head><body>
<h1>MEMV Validation Report</h1>
<p class="summary">%d issues &nbsp;·&nbsp;
<span style="color:#f85149">%d ERROR</span> &nbsp;·&nbsp;
<span style="color:#ffa657">%d WARNING</span> &nbsp;·&nbsp;
<span style="color:#8b949e">%d INFO</span></p>
<table>
<thead><tr><th>Severity</th><th>Rule</th><th>Message / Fix</th><th>Expression</th><th>Source Type</th></tr></thead>
<tbody>%s</tbody>
</table></body></html>`,
		len(issues), counts["ERROR"], counts["WARNING"], counts["INFO"], rows.String())
}

func htmlEsc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func shortRaw(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

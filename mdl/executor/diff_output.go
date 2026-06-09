// SPDX-License-Identifier: Apache-2.0

// Package executor - Diff output formatting functions
package executor

import (
	"fmt"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

// ============================================================================
// Output Formatters
// ============================================================================

// outputUnifiedDiff outputs diff in unified format
func outputUnifiedDiff(ctx *ExecContext, result DiffResult, useColor bool) {
	if result.IsNew {
		// New object - show all as additions
		header := fmt.Sprintf("--- /dev/null\n+++ %s.%s (new)\n",
			result.ObjectType, result.ObjectName)
		if useColor {
			header = colorCyan + header + colorReset
		}
		fmt.Fprint(ctx.Output, header)

		for line := range strings.SplitSeq(result.Proposed, "\n") {
			if useColor {
				fmt.Fprintf(ctx.Output, "%s+%s%s\n", colorGreen, line, colorReset)
			} else {
				fmt.Fprintf(ctx.Output, "+%s\n", line)
			}
		}
		fmt.Fprintln(ctx.Output)
		return
	}

	if result.Current == result.Proposed {
		return // No changes
	}

	// Use difflib for unified diff
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(result.Current),
		B:        difflib.SplitLines(result.Proposed),
		FromFile: fmt.Sprintf("%s.%s (current)", result.ObjectType, result.ObjectName),
		ToFile:   fmt.Sprintf("%s.%s (script)", result.ObjectType, result.ObjectName),
		Context:  3,
	}

	text, _ := difflib.GetUnifiedDiffString(diff)

	if useColor {
		lines := strings.SplitSeq(text, "\n")
		for line := range lines {
			if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "@@") {
				fmt.Fprintf(ctx.Output, "%s%s%s\n", colorCyan, line, colorReset)
			} else if strings.HasPrefix(line, "+") {
				fmt.Fprintf(ctx.Output, "%s%s%s\n", colorGreen, line, colorReset)
			} else if strings.HasPrefix(line, "-") {
				fmt.Fprintf(ctx.Output, "%s%s%s\n", colorRed, line, colorReset)
			} else {
				fmt.Fprintln(ctx.Output, line)
			}
		}
	} else {
		fmt.Fprint(ctx.Output, text)
	}
	fmt.Fprintln(ctx.Output)
}

// outputSideBySideDiff outputs diff in side-by-side format
func outputSideBySideDiff(ctx *ExecContext, result DiffResult, width int, useColor bool) {
	colWidth := (width - 3) / 2 // 3 for separator " | "

	// Header
	header := fmt.Sprintf("%s.%s", result.ObjectType, result.ObjectName)
	if useColor {
		header = colorCyan + header + colorReset
	}
	fmt.Fprintln(ctx.Output, header)
	fmt.Fprintln(ctx.Output, strings.Repeat("─", width))

	leftHeader := "Current"
	rightHeader := "Script"
	if result.IsNew {
		leftHeader = "(new)"
	}
	fmt.Fprintf(ctx.Output, "%-*s │ %s\n", colWidth, leftHeader, rightHeader)
	fmt.Fprintln(ctx.Output, strings.Repeat("─", width))

	currentLines := strings.Split(result.Current, "\n")
	proposedLines := strings.Split(result.Proposed, "\n")

	maxLines := max(len(proposedLines), len(currentLines))

	for i := range maxLines {
		left := ""
		right := ""
		marker := " "

		if i < len(currentLines) {
			left = truncateLine(currentLines[i], colWidth)
		}
		if i < len(proposedLines) {
			right = truncateLine(proposedLines[i], colWidth)
		}

		// Determine change marker
		if i >= len(currentLines) {
			marker = "+"
		} else if i >= len(proposedLines) {
			marker = "-"
		} else if currentLines[i] != proposedLines[i] {
			marker = "~"
		}

		if useColor {
			switch marker {
			case "+":
				right = colorGreen + right + colorReset
				marker = colorGreen + marker + colorReset
			case "-":
				left = colorRed + left + colorReset
				marker = colorRed + marker + colorReset
			case "~":
				marker = colorYellow + marker + colorReset
			}
		}

		fmt.Fprintf(ctx.Output, "%-*s │ %s %s\n", colWidth, left, right, marker)
	}
	fmt.Fprintln(ctx.Output)
}

// outputStructuralDiff outputs diff in structural format
func outputStructuralDiff(ctx *ExecContext, result DiffResult, useColor bool) {
	header := fmt.Sprintf("%s: %s", result.ObjectType, result.ObjectName)
	if useColor {
		header = colorCyan + header + colorReset
	}
	fmt.Fprintln(ctx.Output, header)

	if result.IsNew {
		if useColor {
			fmt.Fprintf(ctx.Output, "  %s+ New%s\n", colorGreen, colorReset)
		} else {
			fmt.Fprintln(ctx.Output, "  + New")
		}
	} else if result.Current == result.Proposed {
		fmt.Fprintln(ctx.Output, "  (no changes)")
	} else if len(result.Changes) == 0 {
		// Modified but no specific changes detected - show generic message
		if useColor {
			fmt.Fprintf(ctx.Output, "  %s~ Modified%s\n", colorYellow, colorReset)
		} else {
			fmt.Fprintln(ctx.Output, "  ~ Modified")
		}
	}

	for _, change := range result.Changes {
		marker := string(change.ChangeType)
		details := ""
		if change.Details != "" {
			details = ": " + change.Details
		}

		line := fmt.Sprintf("  %s %s %s%s", marker, change.ElementType, change.ElementName, details)

		if useColor {
			switch change.ChangeType {
			case ChangeAdded:
				line = fmt.Sprintf("  %s%s %s %s%s%s", colorGreen, marker, change.ElementType, change.ElementName, details, colorReset)
			case ChangeRemoved:
				line = fmt.Sprintf("  %s%s %s %s%s%s", colorRed, marker, change.ElementType, change.ElementName, details, colorReset)
			case ChangeModified:
				line = fmt.Sprintf("  %s%s %s %s%s%s", colorYellow, marker, change.ElementType, change.ElementName, details, colorReset)
			}
		}
		fmt.Fprintln(ctx.Output, line)
	}
	fmt.Fprintln(ctx.Output)
}

// truncateLine truncates a string to the given width for diff display
func truncateLine(s string, width int) string {
	if len(s) <= width {
		return s
	}
	if width <= 3 {
		return s[:width]
	}
	return s[:width-2] + ".."
}

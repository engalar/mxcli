// SPDX-License-Identifier: Apache-2.0
package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBold   = "\033[1m"
)

// renderTerminal writes a human-readable layered test summary to w.
// noColor disables ANSI escape sequences (for CI/redirect).
func renderTerminal(w io.Writer, layerMap map[string]*LayerSummary, benchDiff string, noColor bool) {
	color := func(c, s string) string {
		if noColor {
			return s
		}
		return c + s + colorReset
	}
	bold := func(s string) string { return color(colorBold, s) }

	fmt.Fprintln(w, bold("═══════════════════════════════════════════════════════════"))
	fmt.Fprintf(w, bold(" mxcli Test Report  %s\n"), time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintln(w, bold("═══════════════════════════════════════════════════════════"))
	fmt.Fprintln(w)

	// Sort layers by priority
	sorted := sortedLayers(layerMap)

	fmt.Fprintf(w, "%-24s %6s %6s %6s %7s\n", "Layer", "Tests", "Pass", "Fail", "Time")
	fmt.Fprintln(w, strings.Repeat("─", 57))

	totalPass, totalFail := 0, 0.0
	var totalElapsed float64
	for _, l := range sorted {
		total := l.Pass + l.Fail
		totalPass += l.Pass
		totalFail += float64(l.Fail)
		totalElapsed += l.Elapsed
		status := ""
		if l.Fail > 0 {
			status = color(colorRed, " ← FAIL")
		}
		fmt.Fprintf(w, "%-24s %6d %6d %6d %6.1fs%s\n",
			l.Name, total, l.Pass, l.Fail, l.Elapsed, status)
	}

	fmt.Fprintln(w, strings.Repeat("─", 57))
	totalTests := totalPass + int(totalFail)
	line := fmt.Sprintf("%-24s %6d %6d %6.0f %6.1fs", "TOTAL", totalTests, totalPass, totalFail, totalElapsed)
	if totalFail > 0 {
		fmt.Fprintln(w, color(colorRed, line))
	} else {
		fmt.Fprintln(w, color(colorGreen, line))
	}
	fmt.Fprintln(w)

	// Print failure details
	for _, l := range sorted {
		for _, f := range l.Failures {
			fmt.Fprintf(w, color(colorRed, "FAIL")+" %s/%s\n", f.Package, f.Test)
			if f.Output != "" {
				for _, line := range strings.Split(strings.TrimSpace(f.Output), "\n") {
					fmt.Fprintf(w, "     %s\n", line)
				}
			}
		}
	}

	// Benchmark diff
	if benchDiff != "" {
		fmt.Fprintln(w, bold("Benchmark Regressions (>20% slower vs baseline):"))
		fmt.Fprintln(w, benchDiff)
	}
}

func sortedLayers(m map[string]*LayerSummary) []*LayerSummary {
	out := make([]*LayerSummary, 0, len(m))
	for _, l := range m {
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Priority < out[j].Priority
	})
	return out
}

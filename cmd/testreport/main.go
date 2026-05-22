// SPDX-License-Identifier: Apache-2.0
// cmd/testreport — generate a layered test report from go test -json output.
//
// Usage (via Makefile):
//   go run ./cmd/testreport [--no-color] [--json-file path] [--bench-diff path] [--git-hash hash] [--out-html path]
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	noColor := flag.Bool("no-color", false, "disable ANSI colors")
	jsonFile := flag.String("json-file", "coverage/test-results.json", "path to go test -json output")
	benchDiffFile := flag.String("bench-diff", "coverage/bench-diff.txt", "path to benchstat diff output")
	gitHash := flag.String("git-hash", "", "git commit hash to embed in report")
	outHTML := flag.String("out-html", "coverage/report.html", "path for HTML report output")
	flag.Parse()

	// resolve git hash
	hash := *gitHash
	if hash == "" {
		if out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output(); err == nil {
			hash = strings.TrimSpace(string(out))
		}
	}

	// parse test results
	f, err := os.Open(*jsonFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "testreport: open %s: %v\n", *jsonFile, err)
		os.Exit(1)
	}
	defer f.Close()

	events, err := parseTestResults(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "testreport: parse: %v\n", err)
		os.Exit(1)
	}

	layerMap := aggregateLayers(events)

	// load bench diff
	benchDiff := ""
	if data, err := os.ReadFile(*benchDiffFile); err == nil {
		benchDiff = parseBenchRegressions(string(data))
	}

	// terminal output
	renderTerminal(os.Stdout, layerMap, benchDiff, *noColor)
	fmt.Printf("Coverage report: %s\n", *outHTML)

	// HTML output
	hf, err := os.Create(*outHTML)
	if err != nil {
		fmt.Fprintf(os.Stderr, "testreport: create %s: %v\n", *outHTML, err)
		os.Exit(1)
	}
	defer hf.Close()
	if err := renderHTML(hf, layerMap, benchDiff, hash); err != nil {
		fmt.Fprintf(os.Stderr, "testreport: render HTML: %v\n", err)
		os.Exit(1)
	}
}

// parseBenchRegressions extracts lines from benchstat output where
// the change is worse than -20% (shown as positive %).
// Returns a formatted string of regression lines, or "" if none.
func parseBenchRegressions(diff string) string {
	var lines []string
	for _, line := range strings.Split(diff, "\n") {
		// benchstat marks regressions with a + percentage, e.g. "+39.00%"
		if strings.Contains(line, "%") && (strings.Contains(line, " +2") ||
			strings.Contains(line, " +3") || strings.Contains(line, " +4") ||
			strings.Contains(line, " +5") || strings.Contains(line, " +6") ||
			strings.Contains(line, " +7") || strings.Contains(line, " +8") ||
			strings.Contains(line, " +9") || strings.Contains(line, " +1")) {
			lines = append(lines, "  ⚠  "+strings.TrimSpace(line))
		}
	}
	return strings.Join(lines, "\n")
}

package tui

import (
	"bufio"
	"os/exec"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
)

// CheckError represents a single mx check diagnostic.
type CheckError struct {
	Severity string // "ERROR" or "WARNING"
	Code     string // e.g. "CE0001"
	Message  string
	Location string // e.g. "Module.Microflow" (may be empty)
}

// MxCheckResultMsg carries the result of an async mx check run.
type MxCheckResultMsg struct {
	Errors []CheckError
	Err    error
}

// MxCheckStartMsg signals that a check run has started.
type MxCheckStartMsg struct{}

// checkOutputPattern matches mx check output lines like:
// [error] [CE1613] "The selected association no longer exists." at Combo box 'cmbPriority'
// [warning] [CW0001] "Some warning" at Page 'MyPage'
var checkOutputPattern = regexp.MustCompile(`^\[(error|warning)\]\s+\[(\w+)\]\s+"(.+?)"\s+at\s+(.+?)\s*$`)

// runMxCheck returns a tea.Cmd that runs mx check asynchronously.
func runMxCheck(projectPath string) tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return MxCheckStartMsg{} },
		func() tea.Msg {
			mxPath, err := docker.ResolveMx("")
			if err != nil {
				Trace("checker: mx not found: %v", err)
				return MxCheckResultMsg{Err: err}
			}

			Trace("checker: running %s check %s", mxPath, projectPath)
			cmd := exec.Command(mxPath, "check", projectPath)
			out, err := cmd.CombinedOutput()
			output := string(out)

			errors := parseCheckOutput(output)
			Trace("checker: done, %d diagnostics, err=%v", len(errors), err)

			// mx check returns non-zero exit code when there are errors,
			// but we still want to show the parsed errors — only propagate
			// err if we got no parseable output at all.
			if err != nil && len(errors) == 0 {
				return MxCheckResultMsg{Err: err}
			}
			return MxCheckResultMsg{Errors: errors}
		},
	)
}

// parseCheckOutput extracts CheckError entries from mx check stdout.
func parseCheckOutput(output string) []CheckError {
	var errors []CheckError
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		matches := checkOutputPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		errors = append(errors, CheckError{
			Severity: strings.ToUpper(matches[1]),
			Code:     matches[2],
			Message:  matches[3],
			Location: matches[4],
		})
	}
	return errors
}

// renderCheckResults formats check errors for display in an overlay.
func renderCheckResults(errors []CheckError) string {
	if len(errors) == 0 {
		return CheckPassStyle.Render("✓ Project check passed — no errors or warnings")
	}

	var sb strings.Builder
	var errorCount, warningCount int
	for _, e := range errors {
		if e.Severity == "ERROR" {
			errorCount++
		} else {
			warningCount++
		}
	}

	// Summary header
	sb.WriteString(CheckHeaderStyle.Render("mx check Results"))
	sb.WriteString("\n")
	var summaryParts []string
	if errorCount > 0 {
		summaryParts = append(summaryParts, CheckErrorStyle.Render("● "+itoa(errorCount)+" errors"))
	}
	if warningCount > 0 {
		summaryParts = append(summaryParts, CheckWarnStyle.Render("● "+itoa(warningCount)+" warnings"))
	}
	sb.WriteString(strings.Join(summaryParts, "  "))
	sb.WriteString("\n\n")

	// Detail lines — severity+code on first line, message and location on next
	for _, e := range errors {
		var label string
		if e.Severity == "ERROR" {
			label = CheckErrorStyle.Render(e.Severity + " " + e.Code)
		} else {
			label = CheckWarnStyle.Render(e.Severity + " " + e.Code)
		}
		sb.WriteString(label + "\n")
		sb.WriteString("  " + e.Message + "\n")
		if e.Location != "" {
			sb.WriteString("  " + CheckLocStyle.Render("at "+e.Location) + "\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// formatCheckBadge returns a compact badge string for the status bar.
func formatCheckBadge(errors []CheckError, running bool) string {
	if running {
		return CheckRunningStyle.Render("⟳ checking")
	}
	if errors == nil {
		return "" // no check has run yet
	}
	if len(errors) == 0 {
		return CheckPassStyle.Render("✓")
	}
	var errorCount, warningCount int
	for _, e := range errors {
		if e.Severity == "ERROR" {
			errorCount++
		} else {
			warningCount++
		}
	}
	var parts []string
	if errorCount > 0 {
		parts = append(parts, CheckErrorStyle.Render("✗ "+itoa(errorCount)+"E"))
	}
	if warningCount > 0 {
		parts = append(parts, CheckWarnStyle.Render(itoa(warningCount)+"W"))
	}
	return strings.Join(parts, " ")
}


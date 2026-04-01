// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	"github.com/mendixlabs/mxcli/sdk/mpr/version"
)

// scriptModuleDeps maps script filenames to marketplace module MPKs they require.
// These modules are imported via `mx module-import` before executing the script.
var scriptModuleDeps = map[string][]string{
	"05-database-connection-examples.mdl": {"ExternalDatabaseConnector-v6.2.3.mpk"},
	"13-business-events-examples.mdl":     {"BusinessEvents_3.12.0.mpk"},
}

// scriptKnownCEErrors lists CE error codes that are expected for specific scripts.
// These are syntax showcase scripts that intentionally omit entities, constants,
// headers etc. that full validation requires.
var scriptKnownCEErrors = map[string][]string{
	"06-rest-client-examples.mdl": {
		"CE0061", // No entity selected (JSON response/body mapping without entity)
		"CE6035", // RestOperationCallAction error handling not supported
		"CE7056", // Undefined parameter (dynamic header {1} placeholder)
		"CE7062", // Missing Accept header
		"CE7064", // POST/PUT must include body
		"CE7073", // Constant needs to be defined (auth with $ConstantName)
		"CE7247", // Name cannot be empty (body mapping without entity)
	},
	"17-custom-widget-examples.mdl": {
		"CE0463", // Widget definition changed (TEXTFILTER template property count mismatch)
		"CE1613", // ComboBox enum attribute written as association pointer
	},
}

// TestMxCheck_DoctypeScripts executes each doctype-tests/*.mdl example script
// in its own fresh Mendix project and validates the result with mx check.
//
// Each script runs in isolation so errors are cleanly attributed.
// Files matching *.test.mdl or *.tests.mdl are skipped (they require Docker).
func TestMxCheck_DoctypeScripts(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	// Locate doctype-tests directory
	doctypeDir, err := filepath.Abs("../../mdl-examples/doctype-tests")
	if err != nil {
		t.Fatalf("Failed to resolve doctype-tests path: %v", err)
	}
	if _, err := os.Stat(doctypeDir); err != nil {
		t.Skipf("doctype-tests directory not found at %s", doctypeDir)
	}

	// Locate mx-modules directory for marketplace dependencies
	modulesDir, err := filepath.Abs("../../mx-modules")
	if err != nil {
		t.Logf("Warning: could not resolve mx-modules path: %v", err)
	}

	// Collect eligible scripts (skip .test.mdl and .tests.mdl)
	entries, err := os.ReadDir(doctypeDir)
	if err != nil {
		t.Fatalf("Failed to read doctype-tests directory: %v", err)
	}

	var scripts []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".mdl") {
			continue
		}
		if strings.HasSuffix(name, ".test.mdl") || strings.HasSuffix(name, ".tests.mdl") {
			continue
		}
		scripts = append(scripts, name)
	}
	sort.Strings(scripts)

	if len(scripts) == 0 {
		t.Skip("no eligible MDL scripts found")
	}

	mxPath := findMxBinary()

	for _, name := range scripts {
		scriptPath := filepath.Join(doctypeDir, name)
		content, err := os.ReadFile(scriptPath)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", name, err)
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Fresh project for each script
			env := setupTestEnv(t)
			defer env.teardown()

			// Import required marketplace modules before executing script
			if deps, ok := scriptModuleDeps[name]; ok && modulesDir != "" && mxPath != "" {
				// Disconnect so mx can access the MPR file
				env.executor.Execute(&ast.DisconnectStmt{})

				for _, mpk := range deps {
					mpkPath := filepath.Join(modulesDir, mpk)
					if _, err := os.Stat(mpkPath); err != nil {
						t.Logf("Skipping module import (not found): %s", mpkPath)
						continue
					}
					cmd := exec.Command(mxPath, "module-import", mpkPath, env.projectPath)
					if out, err := cmd.CombinedOutput(); err != nil {
						t.Logf("Warning: module import failed for %s: %v\n%s", mpk, err, string(out))
					}
				}

				// Reconnect after module import
				if err := env.executor.Execute(&ast.ConnectStmt{Path: env.projectPath}); err != nil {
					t.Fatalf("Failed to reconnect after module import: %v", err)
				}
			}

			// Filter out version-gated sections that don't match this project's Mendix version
			pv := env.executor.reader.ProjectVersion()
			filtered, skippedLines := filterByVersion(string(content), pv)
			if skippedLines > 0 {
				t.Logf("Mendix %s: skipped %d version-gated lines", pv.ProductVersion, skippedLines)
			}

			// Execute the script
			prog, errs := visitor.Build(filtered)
			if len(errs) > 0 {
				t.Fatalf("Parse error: %v", errs[0])
			}

			if err := env.executor.ExecuteProgram(prog); err != nil {
				t.Errorf("Execution error: %v", err)
			}

			// Flush to disk
			env.executor.Execute(&ast.DisconnectStmt{})

			// Run mx check
			output, mxErr := runMxCheck(t, env.projectPath)
			if mxErr != nil {
				// Check for actual errors: [error] lines or ERROR: crash messages
				hasErrors := strings.Contains(output, "[error]") || strings.Contains(output, "ERROR:")
				if hasErrors {
					// Check if all errors are from known CE codes (limitations of syntax showcases)
					knownCodes := []string{"CE0161"} // XPath serializer limitation (global)
					if codes, ok := scriptKnownCEErrors[name]; ok {
						knownCodes = append(knownCodes, codes...)
					}
					if allErrorsKnown(output, knownCodes) {
						t.Logf("mx check has known limitations only (%d errors):\n%s",
							strings.Count(output, "[error]"), output)
					} else {
						t.Errorf("mx check found errors:\n%s", output)
					}
				} else {
					t.Logf("mx check output:\n%s", output)
				}
			} else {
				t.Logf("mx check passed: 0 errors")
			}
		})
	}
}

// allErrorsKnown returns true if every [error] line in the mx check output
// contains at least one of the known CE codes.
func allErrorsKnown(output string, knownCodes []string) bool {
	if strings.Contains(output, "ERROR:") {
		return false // Crash-level errors are never known
	}
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, "[error]") {
			continue
		}
		known := false
		for _, code := range knownCodes {
			if strings.Contains(line, code) {
				known = true
				break
			}
		}
		if !known {
			return false
		}
	}
	return true
}

// versionConstraint represents a min/max Mendix version range for -- @version: directives.
type versionConstraint struct {
	minMajor, minMinor int // -1 means no minimum
	maxMajor, maxMinor int // -1 means no maximum
}

// matches returns true if the project version satisfies this constraint.
func (vc *versionConstraint) matches(pv *version.ProjectVersion) bool {
	if vc.minMajor >= 0 {
		if !pv.IsAtLeast(vc.minMajor, vc.minMinor) {
			return false
		}
	}
	if vc.maxMajor >= 0 {
		// Check that version is at most maxMajor.maxMinor
		if pv.MajorVersion > vc.maxMajor || (pv.MajorVersion == vc.maxMajor && pv.MinorVersion > vc.maxMinor) {
			return false
		}
	}
	return true
}

func (vc *versionConstraint) String() string {
	if vc.minMajor >= 0 && vc.maxMajor >= 0 {
		return fmt.Sprintf("%d.%d..%d.%d", vc.minMajor, vc.minMinor, vc.maxMajor, vc.maxMinor)
	}
	if vc.minMajor >= 0 {
		return fmt.Sprintf("%d.%d+", vc.minMajor, vc.minMinor)
	}
	if vc.maxMajor >= 0 {
		return fmt.Sprintf("..%d.%d", vc.maxMajor, vc.maxMinor)
	}
	return "any"
}

// parseVersionDirective parses a "-- @version: <constraint>" line.
// Returns nil for "any" or unparseable directives.
// Formats: "11.0+", "10.6..10.24", "..10.24", "any"
func parseVersionDirective(line string) *versionConstraint {
	s := strings.TrimPrefix(line, "-- @version:")
	s = strings.TrimSpace(s)

	if s == "" || s == "any" {
		return nil
	}

	// Range: "10.6..10.24"
	if parts := strings.SplitN(s, "..", 2); len(parts) == 2 {
		vc := &versionConstraint{minMajor: -1, minMinor: -1, maxMajor: -1, maxMinor: -1}
		if parts[0] != "" {
			major, minor, ok := parseMajorMinor(parts[0])
			if !ok {
				return nil
			}
			vc.minMajor, vc.minMinor = major, minor
		}
		if parts[1] != "" {
			major, minor, ok := parseMajorMinor(parts[1])
			if !ok {
				return nil
			}
			vc.maxMajor, vc.maxMinor = major, minor
		}
		return vc
	}

	// Minimum: "11.0+"
	if strings.HasSuffix(s, "+") {
		s = strings.TrimSuffix(s, "+")
		major, minor, ok := parseMajorMinor(s)
		if !ok {
			return nil
		}
		return &versionConstraint{minMajor: major, minMinor: minor, maxMajor: -1, maxMinor: -1}
	}

	return nil
}

// parseMajorMinor parses "10.24" into (10, 24, true).
func parseMajorMinor(s string) (int, int, bool) {
	parts := strings.SplitN(s, ".", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, false
	}
	return major, minor, true
}

// filterByVersion removes MDL content sections that don't match the project's Mendix version.
// Sections are delimited by "-- @version: <constraint>" directives.
// A directive applies to all following lines until the next directive or end of file.
// "-- @version: any" resets to unconditional inclusion.
func filterByVersion(content string, pv *version.ProjectVersion) (string, int) {
	var result strings.Builder
	var currentConstraint *versionConstraint // nil = no constraint (always include)
	skippedLines := 0

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "-- @version:") {
			currentConstraint = parseVersionDirective(trimmed)
			// Keep the directive line as a comment (so line numbers stay close)
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}
		if currentConstraint == nil || currentConstraint.matches(pv) {
			result.WriteString(line)
			result.WriteString("\n")
		} else {
			// Replace with empty line to preserve line numbering for error messages
			result.WriteString("\n")
			if strings.TrimSpace(line) != "" && !strings.HasPrefix(trimmed, "--") {
				skippedLines++
			}
		}
	}
	return result.String(), skippedLines
}

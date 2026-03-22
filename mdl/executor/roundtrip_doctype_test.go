// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

// TestMxCheck_DoctypeScripts executes all doctype-tests/*.mdl example scripts
// against a fresh Mendix project and validates the result with mx check.
//
// Scripts are executed in alphabetical order on the same project copy so that
// later scripts (e.g., pages) can reference entities created by earlier ones.
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
	sort.Strings(scripts) // Execute in alphabetical order (01-, 02-, ...)

	if len(scripts) == 0 {
		t.Skip("no eligible MDL scripts found")
	}

	// Set up a single project for all scripts
	env := setupTestEnv(t)
	defer env.teardown()

	// Execute each script in order, recording any execution errors per-script
	var execErrors []string
	for _, name := range scripts {
		scriptPath := filepath.Join(doctypeDir, name)
		content, err := os.ReadFile(scriptPath)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", name, err)
		}

		t.Run("exec/"+name, func(t *testing.T) {
			prog, errs := visitor.Build(string(content))
			if len(errs) > 0 {
				t.Fatalf("Parse error: %v", errs[0])
			}

			if err := env.executor.ExecuteProgram(prog); err != nil {
				// Log but don't fatal — allow other scripts to continue
				t.Errorf("Execution error: %v", err)
				execErrors = append(execErrors, name+": "+err.Error())
			}
		})
	}

	// Disconnect to flush all changes to disk
	env.executor.Execute(&ast.DisconnectStmt{})

	// Run mx check on the final project state
	t.Run("mx-check", func(t *testing.T) {
		if len(execErrors) > 0 {
			t.Logf("Note: %d script(s) had execution errors", len(execErrors))
		}

		output, err := runMxCheck(t, env.projectPath)
		if err != nil {
			lowerOutput := strings.ToLower(output)
			if strings.Contains(lowerOutput, "error") {
				t.Errorf("mx check found errors after executing all doctype scripts:\n%s", output)
			} else {
				t.Logf("mx check output:\n%s", output)
			}
		} else {
			t.Logf("mx check passed:\n%s", output)
		}
	})
}

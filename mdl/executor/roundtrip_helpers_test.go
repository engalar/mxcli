// SPDX-License-Identifier: Apache-2.0

// Package executor provides roundtrip tests for MDL commands.
// These tests verify that creating a document and describing it back
// produces semantically equivalent results.
//
// Test categories:
// - Roundtrip tests: Create document → Describe → Verify semantic properties
// - MxCheck tests: Create document → Run mx check → Verify no errors
package executor

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	"github.com/pmezard/go-difflib/difflib"
)

// sourceProject is the pristine source project that gets copied for each test.
const sourceProject = "../../mx-test-projects/test-source-app"

// sourceProjectMPR is the MPR filename inside the source project.
const sourceProjectMPR = "test-source.mpr"

// testModule is the module name used for test entities.
const testModule = "RoundtripTest"

// testEnv holds the test environment for roundtrip tests.
type testEnv struct {
	t           *testing.T
	executor    *Executor
	output      *bytes.Buffer
	projectPath string // path to the copied MPR file
}

// copyTestProject copies the source project to a temp directory and returns the MPR path.
// If the source project doesn't exist, it falls back to creating a fresh project
// using `mx create-project` (requires the mx binary to be available).
// The temp directory is automatically cleaned up when the test finishes.
func copyTestProject(t *testing.T) string {
	t.Helper()

	srcDir, err := filepath.Abs(sourceProject)
	if err != nil {
		t.Fatalf("Failed to resolve source project path: %v", err)
	}

	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		// Source project not found — try mx create-project as fallback
		return createTestProject(t)
	}

	// Create temp directory (auto-cleaned by t.TempDir())
	destDir := t.TempDir()

	// Copy the MPR file
	srcMPR := filepath.Join(srcDir, sourceProjectMPR)
	destMPR := filepath.Join(destDir, sourceProjectMPR)
	if err := copyFile(srcMPR, destMPR); err != nil {
		t.Fatalf("Failed to copy MPR file: %v", err)
	}

	// Copy required directories
	for _, dir := range []string{"mprcontents", "widgets", "themesource", "theme", "javascriptsource"} {
		srcSub := filepath.Join(srcDir, dir)
		if _, err := os.Stat(srcSub); err == nil {
			if err := copyDir(srcSub, filepath.Join(destDir, dir)); err != nil {
				t.Fatalf("Failed to copy %s: %v", dir, err)
			}
		}
	}

	return destMPR
}

// createTestProject creates a fresh Mendix project using `mx create-project`.
// Returns the path to the App.mpr file in a temp directory.
func createTestProject(t *testing.T) string {
	t.Helper()

	mxPath := findMxBinary()
	if mxPath == "" {
		t.Skip("mx binary not available and source project not found")
	}

	destDir := t.TempDir()

	cmd := exec.Command(mxPath, "create-project")
	cmd.Dir = destDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("mx create-project failed: %v\n%s", err, output)
	}

	mprPath := filepath.Join(destDir, "App.mpr")
	if _, err := os.Stat(mprPath); os.IsNotExist(err) {
		t.Skipf("mx create-project did not produce App.mpr in %s", destDir)
	}

	return mprPath
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// copyDir recursively copies a directory tree.
func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// setupTestEnv creates a new test environment with a fresh copy of the source project.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	projectPath := copyTestProject(t)

	output := &bytes.Buffer{}
	exec := New(output)

	// Connect to project
	connectStmt := &ast.ConnectStmt{
		Path: projectPath,
	}
	if err := exec.Execute(connectStmt); err != nil {
		t.Fatalf("Failed to connect to project: %v", err)
	}

	// Ensure test module exists
	env := &testEnv{
		t:           t,
		executor:    exec,
		output:      output,
		projectPath: projectPath,
	}
	env.ensureTestModule()

	return env
}

// ensureTestModule creates the test module if it doesn't exist.
func (e *testEnv) ensureTestModule() {
	e.t.Helper()

	// Try to create module (ignore error if already exists)
	createModuleStmt := &ast.CreateModuleStmt{
		Name: testModule,
	}
	_ = e.executor.Execute(createModuleStmt)
}

// teardown disconnects from the project. No cleanup of created artifacts is needed
// since each test uses a fresh copy that is automatically deleted.
func (e *testEnv) teardown() {
	if e.executor != nil {
		e.executor.Execute(&ast.DisconnectStmt{})
	}
}

// registerCleanup is a no-op since each test uses a fresh project copy.
// Kept for API compatibility with existing test code.
func (e *testEnv) registerCleanup(docType, qualifiedName string) {
	// No-op: temp directory is automatically cleaned up
}

// executeMDL parses and executes MDL commands.
func (e *testEnv) executeMDL(mdl string) error {
	e.t.Helper()
	e.output.Reset()

	prog, errs := visitor.Build(mdl)
	if len(errs) > 0 {
		return errs[0]
	}

	for _, stmt := range prog.Statements {
		if err := e.executor.Execute(stmt); err != nil {
			return err
		}
	}
	return nil
}

// describeMDL executes a DESCRIBE command and returns the output.
func (e *testEnv) describeMDL(describeCmd string) (string, error) {
	e.t.Helper()
	e.output.Reset()

	prog, errs := visitor.Build(describeCmd)
	if len(errs) > 0 {
		return "", errs[0]
	}

	for _, stmt := range prog.Statements {
		if err := e.executor.Execute(stmt); err != nil {
			return "", err
		}
	}
	return e.output.String(), nil
}

// parseQualifiedName parses "Module.Name" into ast.QualifiedName.
func parseQualifiedName(name string) *ast.QualifiedName {
	parts := strings.SplitN(name, ".", 2)
	if len(parts) == 2 {
		return &ast.QualifiedName{Module: parts[0], Name: parts[1]}
	}
	return &ast.QualifiedName{Name: name}
}

// --- Diff-Based Roundtrip Helpers ---
//
// These helpers use the diff infrastructure for more precise comparison
// and better error messages when roundtrip tests fail.

// RoundtripOption configures roundtrip verification behavior.
type RoundtripOption func(*roundtripConfig)

type roundtripConfig struct {
	ignorePatterns   []string        // Lines matching these patterns are ignored
	ignoreAttributes map[string]bool // Attribute names to ignore (e.g., "changedDate")
	allowNewLines    bool            // Allow DESCRIBE output to have additional lines
	entityType       string          // "entity", "enumeration", "page", "microflow", etc.
}

// IgnorePattern returns an option to ignore lines containing the given pattern.
func IgnorePattern(pattern string) RoundtripOption {
	return func(c *roundtripConfig) {
		c.ignorePatterns = append(c.ignorePatterns, pattern)
	}
}

// IgnoreAttribute returns an option to ignore a specific attribute in comparison.
func IgnoreAttribute(attrName string) RoundtripOption {
	return func(c *roundtripConfig) {
		if c.ignoreAttributes == nil {
			c.ignoreAttributes = make(map[string]bool)
		}
		c.ignoreAttributes[attrName] = true
	}
}

// AllowNewLines returns an option to allow DESCRIBE output to have additional lines.
func AllowNewLines() RoundtripOption {
	return func(c *roundtripConfig) {
		c.allowNewLines = true
	}
}

// RoundtripResult contains the result of a roundtrip test.
type RoundtripResult struct {
	Expected string   // Normalized MDL from script
	Actual   string   // Normalized MDL from DESCRIBE
	Diff     string   // Unified diff if there are differences
	Changes  []string // List of structural changes
	Success  bool     // Whether roundtrip passed
}

// assertRoundtrip executes MDL, describes the result, and verifies they match.
// It uses the diff infrastructure for comparison and provides clear error output.
func (e *testEnv) assertRoundtrip(createMDL string, opts ...RoundtripOption) RoundtripResult {
	e.t.Helper()

	config := &roundtripConfig{
		ignorePatterns: []string{"@Position"}, // Default: ignore position annotations
	}
	for _, opt := range opts {
		opt(config)
	}

	result := RoundtripResult{}

	// Parse MDL to get statement type and qualified name
	prog, errs := visitor.Build(createMDL)
	if len(errs) > 0 {
		e.t.Fatalf("Failed to parse MDL: %v", errs[0])
		return result
	}
	if len(prog.Statements) == 0 {
		e.t.Fatal("No statements in MDL")
		return result
	}

	// Execute CREATE statement
	e.output.Reset()
	for _, stmt := range prog.Statements {
		if err := e.executor.Execute(stmt); err != nil {
			e.t.Fatalf("Failed to execute MDL: %v", err)
			return result
		}
	}

	// Determine object type and name for DESCRIBE
	var describeCmd string
	var qualifiedName string
	switch s := prog.Statements[0].(type) {
	case *ast.CreateEntityStmt:
		qualifiedName = s.Name.String()
		e.registerCleanup("entity", qualifiedName)
		describeCmd = "DESCRIBE ENTITY " + qualifiedName + ";"
	case *ast.CreateViewEntityStmt:
		qualifiedName = s.Name.String()
		e.registerCleanup("entity", qualifiedName)
		describeCmd = "DESCRIBE ENTITY " + qualifiedName + ";"
	case *ast.CreateEnumerationStmt:
		qualifiedName = s.Name.String()
		e.registerCleanup("enumeration", qualifiedName)
		describeCmd = "DESCRIBE ENUMERATION " + qualifiedName + ";"
	case *ast.CreatePageStmtV3:
		qualifiedName = s.Name.String()
		e.registerCleanup("page", qualifiedName)
		describeCmd = "DESCRIBE PAGE " + qualifiedName + ";"
	case *ast.CreateSnippetStmtV3:
		qualifiedName = s.Name.String()
		e.registerCleanup("snippet", qualifiedName)
		describeCmd = "DESCRIBE SNIPPET " + qualifiedName + ";"
	case *ast.CreateMicroflowStmt:
		qualifiedName = s.Name.String()
		e.registerCleanup("microflow", qualifiedName)
		describeCmd = "DESCRIBE MICROFLOW " + qualifiedName + ";"
	case *ast.CreateAssociationStmt:
		qualifiedName = s.Name.String()
		describeCmd = "DESCRIBE ASSOCIATION " + qualifiedName + ";"
	case *ast.CreateDatabaseConnectionStmt:
		qualifiedName = s.Name.String()
		describeCmd = "DESCRIBE DATABASE CONNECTION " + qualifiedName + ";"
	default:
		e.t.Fatalf("Unsupported statement type for roundtrip: %T", prog.Statements[0])
		return result
	}

	// Execute DESCRIBE
	describeOutput, err := e.describeMDL(describeCmd)
	if err != nil {
		e.t.Fatalf("Failed to describe %s: %v", qualifiedName, err)
		return result
	}

	// Normalize both MDL strings for comparison
	result.Expected = normalizeMDL(createMDL, config)
	result.Actual = normalizeMDL(describeOutput, config)

	// Compare and generate diff
	if result.Expected != result.Actual {
		diff := difflib.UnifiedDiff{
			A:        difflib.SplitLines(result.Expected),
			B:        difflib.SplitLines(result.Actual),
			FromFile: "expected (script)",
			ToFile:   "actual (describe)",
			Context:  3,
		}
		result.Diff, _ = difflib.GetUnifiedDiffString(diff)

		// Extract structural changes
		result.Changes = extractStructuralChanges(result.Expected, result.Actual)
		result.Success = false
	} else {
		result.Success = true
	}

	return result
}

// normalizeMDL normalizes MDL for comparison by removing ignored patterns and whitespace variations.
func normalizeMDL(mdl string, config *roundtripConfig) string {
	lines := strings.Split(mdl, "\n")
	var normalized []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// Skip ignored patterns
		skip := false
		for _, pattern := range config.ignorePatterns {
			if strings.Contains(trimmed, pattern) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Skip statement terminators on their own line
		if trimmed == "/" || trimmed == ";" {
			continue
		}

		normalized = append(normalized, trimmed)
	}

	return strings.Join(normalized, "\n")
}

// extractStructuralChanges extracts a list of high-level changes between two MDL strings.
func extractStructuralChanges(expected, actual string) []string {
	var changes []string

	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	// Build maps for comparison
	expectedSet := make(map[string]bool)
	actualSet := make(map[string]bool)

	for _, line := range expectedLines {
		expectedSet[strings.TrimSpace(line)] = true
	}
	for _, line := range actualLines {
		actualSet[strings.TrimSpace(line)] = true
	}

	// Find lines only in expected (removed/missing)
	for _, line := range expectedLines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !actualSet[trimmed] {
			changes = append(changes, "- "+trimmed)
		}
	}

	// Find lines only in actual (added/extra)
	for _, line := range actualLines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !expectedSet[trimmed] {
			changes = append(changes, "+ "+trimmed)
		}
	}

	return changes
}

// assertRoundtripSuccess is a convenience method that asserts roundtrip passes.
func (e *testEnv) assertRoundtripSuccess(createMDL string, opts ...RoundtripOption) {
	e.t.Helper()

	result := e.assertRoundtrip(createMDL, opts...)
	if !result.Success {
		e.t.Errorf("Roundtrip failed.\n\nDiff:\n%s\n\nStructural changes:\n%s",
			result.Diff, strings.Join(result.Changes, "\n"))
	} else {
		e.t.Logf("Roundtrip successful.\nActual output:\n%s", result.Actual)
	}
}

// assertContains verifies that the roundtrip output contains expected properties.
// This is useful when exact matching isn't possible but key properties must be present.
func (e *testEnv) assertContains(createMDL string, expectedProps []string, opts ...RoundtripOption) {
	e.t.Helper()

	result := e.assertRoundtrip(createMDL, opts...)

	var missing []string
	for _, prop := range expectedProps {
		if !strings.Contains(result.Actual, prop) {
			missing = append(missing, prop)
		}
	}

	if len(missing) > 0 {
		e.t.Errorf("Missing expected properties: %v\n\nActual output:\n%s",
			missing, result.Actual)
	} else {
		e.t.Logf("Roundtrip contains all expected properties.\nActual output:\n%s", result.Actual)
	}
}

// --- Legacy Semantic Comparison Helpers (kept for backward compatibility) ---

// containsProperty checks if the MDL output contains a property.
func containsProperty(mdl, property string) bool {
	return strings.Contains(mdl, property)
}

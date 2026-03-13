// SPDX-License-Identifier: Apache-2.0

// Tests for bug fixes discovered during BST Monitoring app session (2026-03-13).
package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

// TestDropCreateMicroflowReplacesContent verifies that DROP MICROFLOW followed by
// CREATE MICROFLOW produces a microflow with the new content, not stale content.
// Bug #2: DROP+CREATE reported success but DESCRIBE showed old content due to
// missing cache invalidation in execDropMicroflow.
func TestDropCreateMicroflowReplacesContent(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	name := testModule + ".MF_DropCreateTest"

	// Create original microflow with a LOG statement
	err := env.executeMDL(`CREATE MICROFLOW ` + name + ` ()
BEGIN
  LOG INFO 'original content';
END;
/`)
	if err != nil {
		t.Fatalf("Failed to create original microflow: %v", err)
	}

	// Verify original content
	output, err := env.describeMDL("DESCRIBE MICROFLOW " + name + ";")
	if err != nil {
		t.Fatalf("Failed to describe original: %v", err)
	}
	if !strings.Contains(output, "original content") {
		t.Fatalf("Original microflow missing expected content:\n%s", output)
	}

	// DROP and recreate with different content
	err = env.executeMDL("DROP MICROFLOW " + name + ";")
	if err != nil {
		t.Fatalf("Failed to drop microflow: %v", err)
	}

	err = env.executeMDL(`CREATE MICROFLOW ` + name + ` ()
BEGIN
  LOG WARNING 'replacement content';
END;
/`)
	if err != nil {
		t.Fatalf("Failed to create replacement microflow: %v", err)
	}

	// DESCRIBE should show the NEW content
	output, err = env.describeMDL("DESCRIBE MICROFLOW " + name + ";")
	if err != nil {
		t.Fatalf("Failed to describe replacement: %v", err)
	}
	if !strings.Contains(output, "replacement content") {
		t.Errorf("DROP+CREATE did not replace content. Got:\n%s", output)
	}
	if strings.Contains(output, "original content") {
		t.Errorf("DROP+CREATE still shows original content. Got:\n%s", output)
	}
}

// TestValidateDuplicateVariableDeclareRetrieve verifies that DECLARE followed by
// RETRIEVE for the same variable is caught as a duplicate (CE0111).
// Bug #3: mxcli check passed but mx check reported CE0111.
func TestValidateDuplicateVariableDeclareRetrieve(t *testing.T) {
	input := `CREATE MICROFLOW Test.MF_DuplicateVar ()
BEGIN
  DECLARE $Count Integer = 0;
  RETRIEVE $Count FROM Test.TestItem;
  RETURN $Count;
END;`

	errors := validateMicroflowFromMDL(t, input)

	found := false
	for _, e := range errors {
		if strings.Contains(e, "duplicate") && strings.Contains(e, "Count") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected duplicate variable error for $Count, got errors: %v", errors)
	}
}

// TestValidateDuplicateVariableDeclareOnly verifies that two DECLARE statements
// for the same variable are caught as duplicate.
func TestValidateDuplicateVariableDeclareOnly(t *testing.T) {
	input := `CREATE MICROFLOW Test.MF_DoubleDeclare ()
BEGIN
  DECLARE $X Integer = 0;
  DECLARE $X String = 'hello';
END;`

	errors := validateMicroflowFromMDL(t, input)

	found := false
	for _, e := range errors {
		if strings.Contains(e, "duplicate") && strings.Contains(e, "X") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected duplicate variable error for $X, got errors: %v", errors)
	}
}

// TestValidateNoDuplicateWhenRetrieveOnly verifies that a single RETRIEVE
// (without prior DECLARE) does not trigger a false positive.
func TestValidateNoDuplicateWhenRetrieveOnly(t *testing.T) {
	input := `CREATE MICROFLOW Test.MF_RetrieveOnly ()
BEGIN
  RETRIEVE $Items FROM Test.SomeEntity;
END;`

	errors := validateMicroflowFromMDL(t, input)

	for _, e := range errors {
		if strings.Contains(e, "duplicate") {
			t.Errorf("Unexpected duplicate variable error: %s", e)
		}
	}
}

// TestDescribeEnumerationInSubfolder verifies that DESCRIBE ENUMERATION works
// for enumerations that have been moved to subfolders.
// Bug #4: describeEnumeration used GetModuleName(containerID) which fails for
// subfoldered items; should use FindModuleID first.
func TestDescribeEnumerationInSubfolder(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	enumName := testModule + ".SubfolderTestStatus"

	// Create an enumeration
	err := env.executeMDL(`CREATE ENUMERATION ` + enumName + ` (
		Active 'Active',
		Inactive 'Inactive'
	);`)
	if err != nil {
		t.Fatalf("Failed to create enumeration: %v", err)
	}

	// Move it to a subfolder
	err = env.executeMDL(`MOVE ENUMERATION ` + enumName + ` TO FOLDER 'Enums';`)
	if err != nil {
		t.Fatalf("Failed to move enumeration to folder: %v", err)
	}

	// DESCRIBE should still find it
	output, err := env.describeMDL("DESCRIBE ENUMERATION " + enumName + ";")
	if err != nil {
		t.Errorf("DESCRIBE ENUMERATION failed for subfoldered enum: %v", err)
		return
	}
	if !strings.Contains(output, "Active") || !strings.Contains(output, "Inactive") {
		t.Errorf("DESCRIBE output missing enum values:\n%s", output)
	}
}

// validateMicroflowFromMDL parses a CREATE MICROFLOW statement and runs
// ValidateMicroflowBody, returning any validation errors.
func validateMicroflowFromMDL(t *testing.T, input string) []string {
	t.Helper()

	prog, errs := visitor.Build(input)
	if len(errs) > 0 {
		t.Fatalf("Parse error: %v", errs[0])
	}

	if len(prog.Statements) == 0 {
		t.Fatal("No statements parsed")
	}

	stmt, ok := prog.Statements[0].(*ast.CreateMicroflowStmt)
	if !ok {
		t.Fatalf("Expected CreateMicroflowStmt, got %T", prog.Statements[0])
	}

	return ValidateMicroflowBody(stmt)
}

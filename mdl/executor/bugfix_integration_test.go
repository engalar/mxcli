// SPDX-License-Identifier: Apache-2.0

//go:build integration

// Integration tests for bug fixes that require mx create-project.
package executor

import (
	"strings"
	"testing"
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

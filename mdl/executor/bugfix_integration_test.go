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
	err := env.executeMDL(`create microflow ` + name + ` ()
{
  log info 'original content';
}
/`)
	if err != nil {
		t.Fatalf("Failed to create original microflow: %v", err)
	}

	// Verify original content
	output, err := env.describeMDL("describe microflow " + name + ";")
	if err != nil {
		t.Fatalf("Failed to describe original: %v", err)
	}
	if !strings.Contains(output, "original content") {
		t.Fatalf("Original microflow missing expected content:\n%s", output)
	}

	// DROP and recreate with different content
	err = env.executeMDL("drop microflow " + name + ";")
	if err != nil {
		t.Fatalf("Failed to drop microflow: %v", err)
	}

	err = env.executeMDL(`create microflow ` + name + ` ()
{
  log warning 'replacement content';
}
/`)
	if err != nil {
		t.Fatalf("Failed to create replacement microflow: %v", err)
	}

	// DESCRIBE should show the NEW content
	output, err = env.describeMDL("describe microflow " + name + ";")
	if err != nil {
		t.Fatalf("Failed to describe replacement: %v", err)
	}
	if !strings.Contains(output, "replacement content") {
		t.Errorf("drop+create did not replace content. Got:\n%s", output)
	}
	if strings.Contains(output, "original content") {
		t.Errorf("drop+create still shows original content. Got:\n%s", output)
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
	err := env.executeMDL(`create enumeration ` + enumName + ` (
		Active 'Active',
		Inactive 'Inactive'
	);`)
	if err != nil {
		t.Fatalf("Failed to create enumeration: %v", err)
	}

	// Move it to a subfolder
	err = env.executeMDL(`move enumeration ` + enumName + ` to folder 'Enums';`)
	if err != nil {
		t.Fatalf("Failed to move enumeration to folder: %v", err)
	}

	// DESCRIBE should still find it
	output, err := env.describeMDL("describe enumeration " + enumName + ";")
	if err != nil {
		t.Errorf("describe enumeration failed for subfoldered enum: %v", err)
		return
	}
	if !strings.Contains(output, "Active") || !strings.Contains(output, "Inactive") {
		t.Errorf("describe output missing enum values:\n%s", output)
	}
}

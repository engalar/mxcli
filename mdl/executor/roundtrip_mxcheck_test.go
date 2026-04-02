// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

// --- MX Check Integration Tests ---
// These tests verify that created documents pass Mendix's validation.

// mxCheckAvailable checks if the mx command is available.
func mxCheckAvailable() bool {
	return findMxBinary() != ""
}

// runMxCheck runs mx check on the given project and returns any errors.
func runMxCheck(t *testing.T, projectPath string) (string, error) {
	t.Helper()

	mxPath := findMxBinary()
	if mxPath == "" {
		return "", fmt.Errorf("mx binary not found")
	}

	cmd := exec.Command(mxPath, "check", projectPath)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// TestMxCheck_Entity creates an entity and verifies mx check passes.
func TestMxCheck_Entity(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	env := setupTestEnv(t)
	defer env.teardown()

	entityName := testModule + ".MxCheckEntity"
	env.registerCleanup("entity", entityName)

	// Create entity
	createMDL := `CREATE OR MODIFY PERSISTENT ENTITY ` + entityName + ` (
		Code: String(50) NOT NULL,
		Description: String(500),
		Count: Integer DEFAULT 0
	);`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Disconnect to flush changes
	env.executor.Execute(&ast.DisconnectStmt{})

	// Run mx check
	output, err := runMxCheck(t, env.projectPath)
	if err != nil {
		// mx check returns non-zero exit code if there are errors
		if strings.Contains(output, "error") || strings.Contains(output, "Error") {
			t.Errorf("mx check found errors:\n%s", output)
		} else {
			t.Logf("mx check output:\n%s", output)
		}
	} else {
		t.Logf("mx check passed:\n%s", output)
	}
}

// TestMxCheck_Enumeration creates an enumeration and verifies mx check passes.
func TestMxCheck_Enumeration(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	env := setupTestEnv(t)
	defer env.teardown()

	enumName := testModule + ".MxCheckPriority"
	env.registerCleanup("enumeration", enumName)

	// Create enumeration
	createMDL := `CREATE ENUMERATION ` + enumName + ` (
		Low 'Low Priority',
		Medium 'Medium Priority',
		High 'High Priority',
		Critical 'Critical'
	);`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create enumeration: %v", err)
	}

	// Disconnect to flush changes
	env.executor.Execute(&ast.DisconnectStmt{})

	// Run mx check
	output, err := runMxCheck(t, env.projectPath)
	if err != nil {
		if strings.Contains(output, "error") || strings.Contains(output, "Error") {
			t.Errorf("mx check found errors:\n%s", output)
		} else {
			t.Logf("mx check output:\n%s", output)
		}
	} else {
		t.Logf("mx check passed:\n%s", output)
	}
}

// TestMxCheck_RetrieveWithLimit validates that RETRIEVE with LIMIT produces
// BSON that passes mx check (Studio Pro validation).
func TestMxCheck_RetrieveWithLimit(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	env := setupTestEnv(t)
	defer env.teardown()

	if err := env.executeMDL(`CREATE OR MODIFY PERSISTENT ENTITY RoundtripTest.MxCheckItem (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	mfName := testModule + ".MxCheck_RetrieveLimit"
	env.registerCleanup("microflow", mfName)

	createMDL := `CREATE MICROFLOW ` + mfName + ` () RETURNS Boolean
BEGIN
  RETRIEVE $Item FROM RoundtripTest.MxCheckItem
    LIMIT 1;
  RETURN true;
END;`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create microflow: %v", err)
	}

	env.executor.Execute(&ast.DisconnectStmt{})

	output, err := runMxCheck(t, env.projectPath)
	if err != nil {
		if strings.Contains(output, "error") || strings.Contains(output, "Error") {
			t.Errorf("mx check found errors:\n%s", output)
		} else {
			t.Logf("mx check output:\n%s", output)
		}
	} else {
		t.Logf("mx check passed:\n%s", output)
	}
}

// TestMxCheck_RetrieveWithLimitOffset validates that RETRIEVE with LIMIT and OFFSET
// produces BSON that passes mx check. Regression guard for LimitExpression/OffsetExpression
// being stored in the correct BSON fields within Microflows$ConstantRange.
func TestMxCheck_RetrieveWithLimitOffset(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	env := setupTestEnv(t)
	defer env.teardown()

	if err := env.executeMDL(`CREATE OR MODIFY PERSISTENT ENTITY RoundtripTest.MxCheckItem (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	mfName := testModule + ".MxCheck_RetrieveLimOff"
	env.registerCleanup("microflow", mfName)

	createMDL := `CREATE MICROFLOW ` + mfName + ` () RETURNS Boolean
BEGIN
  RETRIEVE $Items FROM RoundtripTest.MxCheckItem
    LIMIT 2
    OFFSET 3;
  RETURN true;
END;`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create microflow: %v", err)
	}

	env.executor.Execute(&ast.DisconnectStmt{})

	output, err := runMxCheck(t, env.projectPath)
	if err != nil {
		if strings.Contains(output, "error") || strings.Contains(output, "Error") {
			t.Errorf("mx check found errors:\n%s", output)
		} else {
			t.Logf("mx check output:\n%s", output)
		}
	} else {
		t.Logf("mx check passed:\n%s", output)
	}
}

// TestMxCheck_RetrieveWithSortBy validates that RETRIEVE with SORT BY produces
// BSON that passes mx check. Regression guard for sort items being stored under
// the correct BSON key (NewSortings vs sortItemList).
func TestMxCheck_RetrieveWithSortBy(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	env := setupTestEnv(t)
	defer env.teardown()

	if err := env.executeMDL(`CREATE OR MODIFY PERSISTENT ENTITY RoundtripTest.MxCheckItem (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	mfName := testModule + ".MxCheck_RetrieveSort"
	env.registerCleanup("microflow", mfName)

	createMDL := `CREATE MICROFLOW ` + mfName + ` () RETURNS Boolean
BEGIN
  RETRIEVE $Items FROM RoundtripTest.MxCheckItem
    SORT BY RoundtripTest.MxCheckItem.Name ASC;
  RETURN true;
END;`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create microflow: %v", err)
	}

	env.executor.Execute(&ast.DisconnectStmt{})

	output, err := runMxCheck(t, env.projectPath)
	if err != nil {
		if strings.Contains(output, "error") || strings.Contains(output, "Error") {
			t.Errorf("mx check found errors:\n%s", output)
		} else {
			t.Logf("mx check output:\n%s", output)
		}
	} else {
		t.Logf("mx check passed:\n%s", output)
	}
}

// TestMxCheck_RetrieveWithWhereSortLimitOffset validates the full RETRIEVE pattern
// (WHERE + SORT BY + LIMIT + OFFSET) passes mx check. This matches the
// M028_DataForm_Getter microflow pattern.
func TestMxCheck_RetrieveWithWhereSortLimitOffset(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	env := setupTestEnv(t)
	defer env.teardown()

	if err := env.executeMDL(`CREATE OR MODIFY PERSISTENT ENTITY RoundtripTest.MxCheckItem (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	mfName := testModule + ".MxCheck_RetrieveFull"
	env.registerCleanup("microflow", mfName)

	createMDL := `CREATE MICROFLOW ` + mfName + ` () RETURNS Boolean
BEGIN
  RETRIEVE $Items FROM RoundtripTest.MxCheckItem
    WHERE (starts-with(Name, 'a'))
    SORT BY RoundtripTest.MxCheckItem.Name ASC
    LIMIT 2
    OFFSET 3;
  RETURN true;
END;`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create microflow: %v", err)
	}

	env.executor.Execute(&ast.DisconnectStmt{})

	output, err := runMxCheck(t, env.projectPath)
	if err != nil {
		if strings.Contains(output, "error") || strings.Contains(output, "Error") {
			t.Errorf("mx check found errors:\n%s", output)
		} else {
			t.Logf("mx check output:\n%s", output)
		}
	} else {
		t.Logf("mx check passed:\n%s", output)
	}
}

// --- CE0066 "Entity access is out of date" scenario tests ---
//
// These tests enumerate mutation orderings that might trigger CE0066 when
// entity structure changes after access rules have been written.
//
// Scenarios to test:
//   S1: CREATE ENTITY (with attrs) → GRANT                      (baseline)
//   S2: CREATE ENTITY → GRANT → ALTER ADD ATTRIBUTE             (attribute added after security)
//   S3: CREATE ENTITY → GRANT → CREATE ASSOCIATION              (association added after security)
//   S4: CREATE ENTITY → GRANT → ALTER DROP ATTRIBUTE            (attribute removed after security)
//   S5: CREATE ENTITY → ALTER ADD ATTRIBUTE → GRANT             (security after mutation — should always work)
//   S6: CREATE ENTITY → GRANT(READ *) → ALTER → GRANT(READ *)  (re-grant after mutation)
//   S7: CREATE ENTITY + GRANT in single script                  (typical user script pattern)
//   S8: CREATE ENTITY → GRANT multiple roles → ALTER            (multiple rules + mutation)
//   S9: CREATE ENTITY → GRANT → ALTER + CREATE ASSOC (combined) (both attrs and assocs change)

// ce0066Scenario describes a single CE0066 test case.
type ce0066Scenario struct {
	name  string   // subtest name
	steps []string // MDL statements to execute in order
}

func TestMxCheck_CE0066_Scenarios(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	mod := testModule
	scenarios := []ce0066Scenario{
		{
			name: "S1_CreateEntity_Grant",
			steps: []string{
				`CREATE MODULE ROLE ` + mod + `.S1Admin;`,
				`CREATE OR MODIFY PERSISTENT ENTITY ` + mod + `.S1Entity (Name: String(100), Email: String(200));`,
				`GRANT ` + mod + `.S1Admin ON ` + mod + `.S1Entity (CREATE, DELETE, READ *, WRITE *);`,
			},
		},
		{
			name: "S2_Grant_ThenAddAttribute",
			steps: []string{
				`CREATE MODULE ROLE ` + mod + `.S2Admin;`,
				`CREATE OR MODIFY PERSISTENT ENTITY ` + mod + `.S2Entity (Name: String(100));`,
				`GRANT ` + mod + `.S2Admin ON ` + mod + `.S2Entity (CREATE, DELETE, READ *, WRITE *);`,
				`ALTER ENTITY ` + mod + `.S2Entity ADD ATTRIBUTE Email: String(200);`,
				`ALTER ENTITY ` + mod + `.S2Entity ADD ATTRIBUTE Active: Boolean DEFAULT false;`,
			},
		},
		{
			name: "S3_Grant_ThenAddAssociation",
			steps: []string{
				`CREATE MODULE ROLE ` + mod + `.S3Admin;`,
				`CREATE OR MODIFY PERSISTENT ENTITY ` + mod + `.S3Parent (Name: String(100));`,
				`CREATE OR MODIFY PERSISTENT ENTITY ` + mod + `.S3Child (Label: String(100));`,
				`GRANT ` + mod + `.S3Admin ON ` + mod + `.S3Parent (CREATE, DELETE, READ *, WRITE *);`,
				`GRANT ` + mod + `.S3Admin ON ` + mod + `.S3Child (CREATE, DELETE, READ *, WRITE *);`,
				`CREATE ASSOCIATION ` + mod + `.S3Child_S3Parent FROM ` + mod + `.S3Child TO ` + mod + `.S3Parent;`,
			},
		},
		{
			name: "S4_Grant_ThenDropAttribute",
			steps: []string{
				`CREATE MODULE ROLE ` + mod + `.S4Admin;`,
				`CREATE OR MODIFY PERSISTENT ENTITY ` + mod + `.S4Entity (Name: String(100), Temp: String(50));`,
				`GRANT ` + mod + `.S4Admin ON ` + mod + `.S4Entity (CREATE, DELETE, READ *, WRITE *);`,
				`ALTER ENTITY ` + mod + `.S4Entity DROP ATTRIBUTE Temp;`,
			},
		},
		{
			name: "S5_AddAttribute_ThenGrant",
			steps: []string{
				`CREATE MODULE ROLE ` + mod + `.S5Admin;`,
				`CREATE OR MODIFY PERSISTENT ENTITY ` + mod + `.S5Entity (Name: String(100));`,
				`ALTER ENTITY ` + mod + `.S5Entity ADD ATTRIBUTE Email: String(200);`,
				`GRANT ` + mod + `.S5Admin ON ` + mod + `.S5Entity (CREATE, DELETE, READ *, WRITE *);`,
			},
		},
		{
			name: "S6_Grant_Alter_Regrant",
			steps: []string{
				`CREATE MODULE ROLE ` + mod + `.S6Admin;`,
				`CREATE OR MODIFY PERSISTENT ENTITY ` + mod + `.S6Entity (Name: String(100));`,
				`GRANT ` + mod + `.S6Admin ON ` + mod + `.S6Entity (READ *);`,
				`ALTER ENTITY ` + mod + `.S6Entity ADD ATTRIBUTE Code: String(50);`,
				`GRANT ` + mod + `.S6Admin ON ` + mod + `.S6Entity (CREATE, DELETE, READ *, WRITE *);`,
			},
		},
		{
			name: "S7_SingleScript_CreateAndGrant",
			steps: []string{
				`CREATE MODULE ROLE ` + mod + `.S7Admin;`,
				`CREATE OR MODIFY PERSISTENT ENTITY ` + mod + `.S7Entity (
					Code: String(50) NOT NULL,
					Description: String(500),
					Count: Integer DEFAULT 0
				);` + "\n" +
					`GRANT ` + mod + `.S7Admin ON ` + mod + `.S7Entity (CREATE, DELETE, READ *, WRITE *);`,
			},
		},
		{
			name: "S8_MultipleRoles_ThenAlter",
			steps: []string{
				`CREATE MODULE ROLE ` + mod + `.S8Admin;`,
				`CREATE MODULE ROLE ` + mod + `.S8Viewer;`,
				`CREATE OR MODIFY PERSISTENT ENTITY ` + mod + `.S8Entity (Name: String(100));`,
				`GRANT ` + mod + `.S8Admin ON ` + mod + `.S8Entity (CREATE, DELETE, READ *, WRITE *);`,
				`GRANT ` + mod + `.S8Viewer ON ` + mod + `.S8Entity (READ *);`,
				`ALTER ENTITY ` + mod + `.S8Entity ADD ATTRIBUTE Status: String(50);`,
			},
		},
		{
			name: "S9_Grant_ThenAlterAndAssoc",
			steps: []string{
				`CREATE MODULE ROLE ` + mod + `.S9Admin;`,
				`CREATE OR MODIFY PERSISTENT ENTITY ` + mod + `.S9Main (Name: String(100));`,
				`CREATE OR MODIFY PERSISTENT ENTITY ` + mod + `.S9Related (Value: Integer);`,
				`GRANT ` + mod + `.S9Admin ON ` + mod + `.S9Main (CREATE, DELETE, READ *, WRITE *);`,
				`GRANT ` + mod + `.S9Admin ON ` + mod + `.S9Related (CREATE, DELETE, READ *, WRITE *);`,
				`ALTER ENTITY ` + mod + `.S9Main ADD ATTRIBUTE Extra: String(200);`,
				`CREATE ASSOCIATION ` + mod + `.S9Related_S9Main FROM ` + mod + `.S9Related TO ` + mod + `.S9Main;`,
			},
		},
		{
			name: "S10_DropIndexedAttribute",
			steps: []string{
				`CREATE MODULE ROLE ` + mod + `.S10Admin;`,
				`CREATE OR MODIFY PERSISTENT ENTITY ` + mod + `.S10Entity (Name: String(100), Code: String(50));`,
				`ALTER ENTITY ` + mod + `.S10Entity ADD INDEX (Code);`,
				`GRANT ` + mod + `.S10Admin ON ` + mod + `.S10Entity (CREATE, DELETE, READ *, WRITE *);`,
				`ALTER ENTITY ` + mod + `.S10Entity DROP ATTRIBUTE Code;`,
			},
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			env := setupTestEnv(t)
			defer env.teardown()

			// Combine all steps into a single script so ExecuteProgram
			// runs finalizeProgramExecution (ReconcileMemberAccesses).
			allMDL := strings.Join(sc.steps, "\n")
			prog, errs := visitor.Build(allMDL)
			if len(errs) > 0 {
				t.Fatalf("Parse failed: %v\nMDL:\n%s", errs[0], allMDL)
			}
			if err := env.executor.ExecuteProgram(prog); err != nil {
				if !strings.Contains(err.Error(), "already exists") {
					t.Fatalf("ExecuteProgram failed: %v\nMDL:\n%s", err, allMDL)
				}
			}

			env.executor.Execute(&ast.DisconnectStmt{})

			output, err := runMxCheck(t, env.projectPath)
			if err != nil {
				if strings.Contains(output, "CE0066") || strings.Contains(output, "out of date") {
					t.Errorf("CE0066 entity access out of date:\n%s", output)
				} else if strings.Contains(output, "error") || strings.Contains(output, "Error") {
					t.Errorf("mx check found errors:\n%s", output)
				} else {
					t.Logf("mx check output (non-zero exit but no errors):\n%s", output)
				}
			} else {
				t.Logf("mx check passed")
			}
		})
	}
}

// TestMxCheck_DropAddEnumAttribute validates that dropping and re-adding an attribute
// with an enumeration default value does not corrupt the MPR (GitHub issue #4).
// Before the fix, the StoredValue $ID was lost during parsing and a new GUID was
// generated on write, leaving dangling references that caused KeyNotFoundException.
func TestMxCheck_DropAddEnumAttribute(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	env := setupTestEnv(t)
	defer env.teardown()

	mod := testModule

	// Step 1: Create an enumeration and an entity with an enumeration attribute
	setupMDL := strings.Join([]string{
		`CREATE ENUMERATION ` + mod + `.SubmissionStatus (StatusNew 'New', StatusInProgress 'In Progress', StatusDone 'Done');`,
		`CREATE OR MODIFY PERSISTENT ENTITY ` + mod + `.Issue4Entity (
			Name: String(100),
			Status: Enumeration(` + mod + `.SubmissionStatus) DEFAULT ` + mod + `.SubmissionStatus.StatusNew
		);`,
	}, "\n")

	prog, errs := visitor.Build(setupMDL)
	if len(errs) > 0 {
		t.Fatalf("Parse failed: %v\nMDL:\n%s", errs[0], setupMDL)
	}
	if err := env.executor.ExecuteProgram(prog); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Step 2: Drop the attribute and re-add it with a different default
	alterMDL := strings.Join([]string{
		`ALTER ENTITY ` + mod + `.Issue4Entity DROP ATTRIBUTE Status;`,
		`ALTER ENTITY ` + mod + `.Issue4Entity ADD ATTRIBUTE Status: Enumeration(` + mod + `.SubmissionStatus) DEFAULT ` + mod + `.SubmissionStatus.StatusInProgress;`,
	}, "\n")

	prog, errs = visitor.Build(alterMDL)
	if len(errs) > 0 {
		t.Fatalf("Parse failed: %v\nMDL:\n%s", errs[0], alterMDL)
	}
	if err := env.executor.ExecuteProgram(prog); err != nil {
		t.Fatalf("Alter failed: %v", err)
	}

	// Step 3: Flush to disk and validate with mx check
	env.executor.Execute(&ast.DisconnectStmt{})

	output, err := runMxCheck(t, env.projectPath)
	if err != nil {
		if strings.Contains(output, "KeyNotFoundException") || strings.Contains(output, "not present in the dictionary") {
			t.Errorf("Issue #4 regression: dangling GUID reference after drop/add enum attribute:\n%s", output)
		} else if strings.Contains(output, "error") || strings.Contains(output, "Error") {
			t.Errorf("mx check found errors:\n%s", output)
		} else {
			t.Logf("mx check output (non-zero exit but no errors):\n%s", output)
		}
	} else {
		t.Logf("mx check passed")
	}
}

// TestMxCheck_RetrieveWithDateTimeToken validates that RETRIEVE with [%CurrentDateTime%]
// in a WHERE clause produces correctly quoted XPath (GitHub issue #1).
// The token must be quoted as '[%CurrentDateTime%]' in XPath constraints.
func TestMxCheck_RetrieveWithDateTimeToken(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	env := setupTestEnv(t)
	defer env.teardown()

	if err := env.executeMDL(`CREATE OR MODIFY PERSISTENT ENTITY RoundtripTest.MxCheckDated (
		Name: String(100),
		DueDate: DateTime
	);`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	mfName := testModule + ".MxCheck_DateTimeToken"
	env.registerCleanup("microflow", mfName)

	createMDL := `CREATE MICROFLOW ` + mfName + ` () RETURNS Boolean
BEGIN
  RETRIEVE $Items FROM RoundtripTest.MxCheckDated
    WHERE DueDate < [%CurrentDateTime%];
  RETURN true;
END;`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create microflow: %v", err)
	}

	env.executor.Execute(&ast.DisconnectStmt{})

	output, err := runMxCheck(t, env.projectPath)
	if err != nil {
		if strings.Contains(output, "error") || strings.Contains(output, "Error") {
			t.Errorf("mx check found errors:\n%s", output)
		} else {
			t.Logf("mx check output:\n%s", output)
		}
	} else {
		t.Logf("mx check passed:\n%s", output)
	}
}

// TestMxCheck_MicroflowWithCallParams tests microflow with CALL unified param syntax.
func TestMxCheck_MicroflowWithCallParams(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	env := setupTestEnv(t)
	defer env.teardown()

	// Create a helper microflow
	helperName := testModule + ".MxCheckCallHelper"
	env.registerCleanup("microflow", helperName)

	createHelperMDL := `CREATE MICROFLOW ` + helperName + ` ($InputValue: String) RETURNS String
	BEGIN
		RETURN $InputValue;
	END;`

	if err := env.executeMDL(createHelperMDL); err != nil {
		t.Fatalf("Failed to create helper microflow: %v", err)
	}

	// Create caller microflow with unified param syntax
	callerName := testModule + ".MxCheckCallCaller"
	env.registerCleanup("microflow", callerName)

	createCallerMDL := `CREATE MICROFLOW ` + callerName + ` () RETURNS String
	BEGIN
		$Result = CALL MICROFLOW ` + helperName + ` (InputValue = 'TestValue');
		RETURN $Result;
	END;`

	if err := env.executeMDL(createCallerMDL); err != nil {
		t.Fatalf("Failed to create caller microflow: %v", err)
	}

	// Disconnect to flush changes
	env.executor.Execute(&ast.DisconnectStmt{})

	// Run mx check
	output, err := runMxCheck(t, env.projectPath)
	if err != nil {
		if strings.Contains(output, "error") || strings.Contains(output, "Error") {
			t.Errorf("mx check found errors:\n%s", output)
		} else {
			t.Logf("mx check output:\n%s", output)
		}
	} else {
		t.Logf("mx check passed:\n%s", output)
	}
}

// TestMxCheck_ViewEntitySimple creates a simple VIEW entity (no aggregates)
// and verifies mx check passes.
func TestMxCheck_ViewEntitySimple(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	env := setupTestEnv(t)
	defer env.teardown()
	env.requireMinVersion(t, 11, 0) // VIEW ENTITY BSON format requires 11.0+

	mod := testModule

	entityName := mod + ".MxCheckProduct"
	env.registerCleanup("entity", entityName)

	if err := env.executeMDL(`CREATE OR MODIFY PERSISTENT ENTITY ` + entityName + ` (
		Name: String(100),
		Price: Decimal
	);`); err != nil {
		t.Fatalf("Failed to create source entity: %v", err)
	}

	viewName := mod + ".MxCheckProductView"
	env.registerCleanup("entity", viewName)

	viewMDL := `CREATE VIEW ENTITY ` + viewName + ` (
		Name: String(100),
		Price: Decimal
	) AS (
		SELECT p.Name AS Name, p.Price AS Price
		FROM ` + entityName + ` AS p
	);`

	if err := env.executeMDL(viewMDL); err != nil {
		t.Fatalf("Failed to create view entity: %v", err)
	}

	env.executor.Execute(&ast.DisconnectStmt{})

	output, err := runMxCheck(t, env.projectPath)
	if err != nil {
		if strings.Contains(output, "out of sync") {
			t.Errorf("mx check reports view entity out of sync with OQL:\n%s", output)
		} else if strings.Contains(output, "error") || strings.Contains(output, "Error") {
			t.Errorf("mx check found errors:\n%s", output)
		} else {
			t.Logf("mx check output:\n%s", output)
		}
	} else {
		t.Logf("mx check passed:\n%s", output)
	}
}

// TestMxCheck_ViewEntityWithAggregates creates a VIEW entity with aggregate OQL
// (COUNT, SUM, AVG, GROUP BY) and verifies mx check passes.
// Regression test for GitHub issue: COUNT must return Long, not Integer.
func TestMxCheck_ViewEntityWithAggregates(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	env := setupTestEnv(t)
	defer env.teardown()
	env.requireMinVersion(t, 11, 0) // VIEW ENTITY BSON format requires 11.0+

	mod := testModule

	// Create source entity with numeric fields for aggregation
	entityName := mod + ".MxCheckDeal"
	env.registerCleanup("entity", entityName)

	if err := env.executeMDL(`CREATE OR MODIFY PERSISTENT ENTITY ` + entityName + ` (
		Stage: String(50),
		Amount: Decimal
	);`); err != nil {
		t.Fatalf("Failed to create source entity: %v", err)
	}

	// Create VIEW entity with aggregate OQL
	// Note: COUNT returns Long in Mendix OQL, not Integer
	viewName := mod + ".MxCheckDealsByStage"
	env.registerCleanup("entity", viewName)

	viewMDL := `CREATE VIEW ENTITY ` + viewName + ` (
		Stage: String(50),
		DealCount: Integer,
		TotalAmount: Decimal,
		AvgAmount: Decimal
	) AS (
		SELECT
			d.Stage AS Stage,
			count(d.ID) AS DealCount,
			sum(d.Amount) AS TotalAmount,
			avg(d.Amount) AS AvgAmount
		FROM ` + entityName + ` AS d
		GROUP BY d.Stage
	);`

	if err := env.executeMDL(viewMDL); err != nil {
		t.Fatalf("Failed to create view entity: %v", err)
	}

	// Disconnect to flush changes
	env.executor.Execute(&ast.DisconnectStmt{})

	// Run mx check
	output, err := runMxCheck(t, env.projectPath)
	if err != nil {
		if strings.Contains(output, "out of sync") {
			t.Errorf("mx check reports view entity out of sync with OQL:\n%s", output)
		} else if strings.Contains(output, "error") || strings.Contains(output, "Error") {
			t.Errorf("mx check found errors:\n%s", output)
		} else {
			t.Logf("mx check output:\n%s", output)
		}
	} else {
		t.Logf("mx check passed:\n%s", output)
	}
}

// TestMxCheck_ComboBoxWithAssociation creates a page with a COMBOBOX widget that
// uses an association attribute and verifies mx check passes.
// Regression test for: COMBOBOX Attribute should resolve as association path (2-part),
// not regular attribute path (3-part).
func TestMxCheck_ComboBoxWithAssociation(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	env := setupTestEnv(t)
	defer env.teardown()
	env.requireMinVersion(t, 11, 0) // Widget template is 11.6, CE0463 on 10.x

	mod := testModule

	// Create target entity (for the association)
	companyEntity := mod + ".MxCheckCompany"
	env.registerCleanup("entity", companyEntity)

	if err := env.executeMDL(`CREATE OR MODIFY PERSISTENT ENTITY ` + companyEntity + ` (
		Name: String(100)
	);`); err != nil {
		t.Fatalf("Failed to create Company entity: %v", err)
	}

	// Create source entity (with association to Company)
	contactEntity := mod + ".MxCheckContact"
	env.registerCleanup("entity", contactEntity)

	if err := env.executeMDL(`CREATE OR MODIFY PERSISTENT ENTITY ` + contactEntity + ` (
		FullName: String(100),
		Email: String(200)
	);`); err != nil {
		t.Fatalf("Failed to create Contact entity: %v", err)
	}

	// Create association
	assocName := mod + ".MxCheckContact_MxCheckCompany"

	if err := env.executeMDL(`CREATE ASSOCIATION ` + assocName + ` FROM ` + contactEntity + ` TO ` + companyEntity + `;`); err != nil {
		t.Fatalf("Failed to create association: %v", err)
	}

	// Create a microflow that returns a Contact (for dataview source)
	mfName := mod + ".MxCheckGetContact"
	env.registerCleanup("microflow", mfName)

	mfMDL := `CREATE MICROFLOW ` + mfName + ` () RETURNS ` + contactEntity + `
BEGIN
  RETRIEVE $Contact FROM ` + contactEntity + ` LIMIT 1;
  RETURN $Contact;
END;`

	if err := env.executeMDL(mfMDL); err != nil {
		t.Fatalf("Failed to create microflow: %v", err)
	}

	// Create page with COMBOBOX using association attribute
	pageName := mod + ".MxCheckContactEdit"
	env.registerCleanup("page", pageName)

	pageMDL := `CREATE PAGE ` + pageName + ` (
		Title: 'Contact Edit',
		Layout: Atlas_Core.Atlas_Default
	) {
		DATAVIEW dvContact (DataSource: MICROFLOW ` + mfName + `) {
			LAYOUTGRID lgMain {
				ROW r1 {
					COLUMN c1 (DesktopWidth: 12) {
						TEXTBOX txtName (Attribute: FullName, Label: 'Full Name')
						COMBOBOX cmbCompany (
							Label: 'Company',
							Attribute: MxCheckContact_MxCheckCompany,
							DataSource: DATABASE ` + companyEntity + `,
							CaptionAttribute: Name
						)
					}
				}
			}
		}
	};`

	if err := env.executeMDL(pageMDL); err != nil {
		t.Fatalf("Failed to create page with ComboBox: %v", err)
	}

	// Disconnect to flush changes
	env.executor.Execute(&ast.DisconnectStmt{})

	// Run mx check
	output, err := runMxCheck(t, env.projectPath)
	if err != nil {
		if strings.Contains(output, "no longer exists") {
			t.Errorf("mx check reports attribute no longer exists (association not resolved correctly):\n%s", output)
		} else if strings.Contains(output, "CE0642") {
			t.Errorf("mx check reports Entity property missing on ComboBox (association EntityRef not set):\n%s", output)
		} else if strings.Contains(output, "CE8812") {
			t.Errorf("mx check reports association path missing on ComboBox:\n%s", output)
		} else if strings.Contains(output, "error") || strings.Contains(output, "Error") {
			t.Errorf("mx check found errors:\n%s", output)
		} else {
			t.Logf("mx check output:\n%s", output)
		}
	} else {
		t.Logf("mx check passed:\n%s", output)
	}
}

// TestMxCheck_WhileLoop creates a microflow with a WHILE loop and verifies
// mx check passes. Regression test for WHILE loop support.
func TestMxCheck_WhileLoop(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	env := setupTestEnv(t)
	defer env.teardown()

	mod := testModule

	mfName := mod + ".MxCheck_WhileLoop"
	env.registerCleanup("microflow", mfName)

	createMDL := `CREATE MICROFLOW ` + mfName + ` ($N: Integer) RETURNS Integer
BEGIN
  DECLARE $Counter Integer = 0;
  DECLARE $Sum Integer = 0;
  WHILE $Counter < $N
  BEGIN
    SET $Counter = $Counter + 1;
    SET $Sum = $Sum + $Counter;
  END WHILE;
  RETURN $Sum;
END;`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create microflow with WHILE loop: %v", err)
	}

	env.executor.Execute(&ast.DisconnectStmt{})

	output, err := runMxCheck(t, env.projectPath)
	if err != nil {
		if strings.Contains(output, "error") || strings.Contains(output, "Error") {
			t.Errorf("mx check found errors:\n%s", output)
		} else {
			t.Logf("mx check output:\n%s", output)
		}
	} else {
		t.Logf("mx check passed:\n%s", output)
	}
}

// TestMxCheck_DropModuleCleansUserRoles verifies that dropping a module removes
// its module roles from user roles in ProjectSecurity (prevents CE1613).
func TestMxCheck_DropModuleCleansUserRoles(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	env := setupTestEnv(t)
	defer env.teardown()

	mod := testModule

	// Create module role and user role referencing it
	setupMDL := strings.Join([]string{
		`CREATE MODULE ROLE ` + mod + `.TestAdmin;`,
		`CREATE USER ROLE DropTestUser (System.User, ` + mod + `.TestAdmin);`,
	}, "\n")

	prog, errs := visitor.Build(setupMDL)
	if len(errs) > 0 {
		t.Fatalf("Parse failed: %v", errs[0])
	}
	if err := env.executor.ExecuteProgram(prog); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Drop the module — should cascade-remove module roles from user roles
	dropMDL := `DROP MODULE ` + mod + `;`
	if err := env.executeMDL(dropMDL); err != nil {
		t.Fatalf("Drop module failed: %v", err)
	}

	// Flush and validate
	env.executor.Execute(&ast.DisconnectStmt{})

	output, err := runMxCheck(t, env.projectPath)
	if err != nil {
		if strings.Contains(output, "CE1613") {
			t.Errorf("CE1613: dangling module role reference after DROP MODULE:\n%s", output)
		} else if strings.Contains(output, "[error]") {
			t.Errorf("mx check found errors:\n%s", output)
		} else {
			t.Logf("mx check output:\n%s", output)
		}
	} else {
		t.Logf("mx check passed:\n%s", output)
	}
}

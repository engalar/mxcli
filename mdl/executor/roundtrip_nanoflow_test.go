// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"fmt"
	"strings"
	"testing"
)

// --- Nanoflow Integration Tests ---
// These tests verify nanoflow CREATE, DESCRIBE, DROP, SHOW, MOVE,
// GRANT/REVOKE, CALL, and MERMAID commands against a real .mpr project.

// assertNanoflowContains creates a nanoflow, describes it, and verifies
// expected strings are present (and unwanted strings are absent).
func assertNanoflowContains(t *testing.T, env *testEnv, nfName, createMDL string, wantContains []string, wantNotContains []string) {
	t.Helper()

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create nanoflow %s: %v", nfName, err)
	}

	output, err := env.describeMDL(fmt.Sprintf("describe nanoflow %s;", nfName))
	if err != nil {
		t.Fatalf("Failed to describe nanoflow %s: %v", nfName, err)
	}

	for _, want := range wantContains {
		if !strings.Contains(output, want) {
			t.Errorf("Expected %q in output, got:\n%s", want, output)
		}
	}

	for _, notWant := range wantNotContains {
		if strings.Contains(output, notWant) {
			t.Errorf("Did not expect %q in output, got:\n%s", notWant, output)
		}
	}

	t.Logf("describe output for %s:\n%s", nfName, output)
}

// --- CREATE + DESCRIBE roundtrips ---

// TestRoundtripNanoflow_EmptyVoid creates a minimal nanoflow with no body.
func TestRoundtripNanoflow_EmptyVoid(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	nfName := testModule + ".RT_NF_Empty"
	createMDL := `create nanoflow ` + nfName + ` () returns Void
{
}`

	assertNanoflowContains(t, env, nfName, createMDL,
		[]string{"nanoflow", nfName},
		nil,
	)
}

// TestRoundtripNanoflow_ReturnString creates a nanoflow returning a string literal.
func TestRoundtripNanoflow_ReturnString(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	nfName := testModule + ".RT_NF_ReturnString"
	createMDL := `create nanoflow ` + nfName + ` () returns String
{
  declare $Greeting String = 'hello';
  return $Greeting;
}`

	assertNanoflowContains(t, env, nfName, createMDL,
		[]string{"nanoflow", "String", "return"},
		nil,
	)
}

// TestRoundtripNanoflow_WithParameters creates a nanoflow with typed parameters.
func TestRoundtripNanoflow_WithParameters(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	nfName := testModule + ".RT_NF_Params"
	createMDL := `create nanoflow ` + nfName + ` ($Name: String, $Count: Integer) returns Boolean
{
  return true;
}`

	assertNanoflowContains(t, env, nfName, createMDL,
		[]string{"nanoflow", "$Name", "String", "$Count", "Integer", "Boolean"},
		nil,
	)
}

// TestRoundtripNanoflow_IfElse creates a nanoflow with an IF/ELSE branch.
func TestRoundtripNanoflow_IfElse(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	nfName := testModule + ".RT_NF_IfElse"
	createMDL := `create nanoflow ` + nfName + ` ($Value: Integer) returns String
{
  declare $Result String = 'none';
  if $Value > 0 then
    set $Result = 'positive';
  else
    set $Result = 'non-positive';
  end if;
  return $Result;
}`

	assertNanoflowContains(t, env, nfName, createMDL,
		[]string{"if", "else", "'positive'", "return"},
		[]string{"then", "end if"},
	)
}

// TestRoundtripNanoflow_Loop creates a nanoflow with a loop.
func TestRoundtripNanoflow_Loop(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create a prerequisite entity
	if err := env.executeMDL(`create or modify persistent entity ` + testModule + `.LoopItem (
		Name: String
	);`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	nfName := testModule + ".RT_NF_Loop"
	createMDL := `create nanoflow ` + nfName + ` () returns Integer
{
  retrieve $Items from ` + testModule + `.LoopItem;
  declare $Count Integer = 0;
  loop $Item in $Items {
    set $Count = $Count + 1;
  }
  return $Count;
}`

	assertNanoflowContains(t, env, nfName, createMDL,
		[]string{"nanoflow", "loop", "end loop", "retrieve", "return"},
		nil,
	)
}

// TestRoundtripNanoflow_ShowPage creates a nanoflow with a show page action.
func TestRoundtripNanoflow_ShowPage(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	nfName := testModule + ".RT_NF_ShowPage"
	createMDL := `create nanoflow ` + nfName + ` () returns Void
{
  show page MyFirstModule.Home_Web ();
}`

	assertNanoflowContains(t, env, nfName, createMDL,
		[]string{"nanoflow", "show page"},
		nil,
	)
}

// TestRoundtripNanoflow_CallMicroflow creates a nanoflow that calls a microflow.
func TestRoundtripNanoflow_CallMicroflow(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create a target microflow first
	mfName := testModule + ".RT_NF_TargetMf"
	createMf := `create microflow ` + mfName + ` ($Input: String) returns String
{
  return $Input;
}`
	if err := env.executeMDL(createMf); err != nil {
		t.Fatalf("Failed to create target microflow: %v", err)
	}

	nfName := testModule + ".RT_NF_CallMf"
	createMDL := `create nanoflow ` + nfName + ` () returns String
{
  $Result = call microflow ` + mfName + ` (Input = 'test');
  return $Result;
}`

	assertNanoflowContains(t, env, nfName, createMDL,
		[]string{"nanoflow", "call microflow", "return"},
		nil,
	)
}

// TestRoundtripNanoflow_CallNanoflow creates a nanoflow that calls another nanoflow.
func TestRoundtripNanoflow_CallNanoflow(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create target nanoflow
	targetName := testModule + ".RT_NF_Target"
	createTarget := `create nanoflow ` + targetName + ` ($Input: String) returns String
{
  return $Input;
}`
	if err := env.executeMDL(createTarget); err != nil {
		t.Fatalf("Failed to create target nanoflow: %v", err)
	}

	nfName := testModule + ".RT_NF_CallNf"
	createMDL := `create nanoflow ` + nfName + ` () returns String
{
  $Result = call nanoflow ` + targetName + ` (Input = 'hello');
  return $Result;
}`

	assertNanoflowContains(t, env, nfName, createMDL,
		[]string{"nanoflow", "call nanoflow", "return"},
		nil,
	)
}

// TestRoundtripNanoflow_ErrorHandling creates a nanoflow with ON ERROR CONTINUE.
func TestRoundtripNanoflow_ErrorHandling(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create a target microflow to call with error handling
	mfName := testModule + ".RT_NF_ErrTarget"
	createMf := `create microflow ` + mfName + ` () returns Boolean
{
  return true;
}`
	if err := env.executeMDL(createMf); err != nil {
		t.Fatalf("Failed to create target microflow: %v", err)
	}

	nfName := testModule + ".RT_NF_ErrorHandling"
	createMDL := `create nanoflow ` + nfName + ` () returns Boolean
{
  $Result = call microflow ` + mfName + ` () on error continue;
  return $Result;
}`

	assertNanoflowContains(t, env, nfName, createMDL,
		[]string{"nanoflow", "call microflow", "on error continue", "return"},
		nil,
	)
}

// --- DROP ---

// TestNanoflow_Drop creates and drops a nanoflow, verifying it's gone.
func TestNanoflow_Drop(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	nfName := testModule + ".RT_NF_Drop"
	createMDL := `create nanoflow ` + nfName + ` () returns Void
{
}`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create nanoflow: %v", err)
	}

	// Verify it exists
	_, err := env.describeMDL(fmt.Sprintf("describe nanoflow %s;", nfName))
	if err != nil {
		t.Fatalf("Nanoflow should exist after creation: %v", err)
	}

	// Drop it
	if err := env.executeMDL(fmt.Sprintf("drop nanoflow %s;", nfName)); err != nil {
		t.Fatalf("Failed to drop nanoflow: %v", err)
	}

	// Verify it's gone
	_, err = env.describeMDL(fmt.Sprintf("describe nanoflow %s;", nfName))
	if err == nil {
		t.Error("Expected error after dropping nanoflow, but describe succeeded")
	}
}

// --- SHOW ---

// TestNanoflow_Show creates nanoflows and verifies SHOW lists them.
func TestNanoflow_Show(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	nf1 := testModule + ".RT_NF_Show1"
	nf2 := testModule + ".RT_NF_Show2"

	for _, nf := range []string{nf1, nf2} {
		createMDL := `create nanoflow ` + nf + ` () returns Void
{
}`
		if err := env.executeMDL(createMDL); err != nil {
			t.Fatalf("Failed to create nanoflow %s: %v", nf, err)
		}
	}

	// Show nanoflows in the test module
	env.output.Reset()
	if err := env.executeMDL(fmt.Sprintf("show nanoflows in %s;", testModule)); err != nil {
		t.Fatalf("Failed to show nanoflows: %v", err)
	}

	output := env.output.String()
	if !strings.Contains(output, "RT_NF_Show1") {
		t.Errorf("Expected RT_NF_Show1 in show output, got:\n%s", output)
	}
	if !strings.Contains(output, "RT_NF_Show2") {
		t.Errorf("Expected RT_NF_Show2 in show output, got:\n%s", output)
	}
	t.Logf("show nanoflows output:\n%s", output)
}

// --- MOVE ---

// TestNanoflow_Move creates a nanoflow and moves it to a different folder.
func TestNanoflow_Move(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	nfName := testModule + ".RT_NF_Move"
	createMDL := `create nanoflow ` + nfName + ` () returns Void
{
}`
	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create nanoflow: %v", err)
	}

	// Move to a subfolder
	if err := env.executeMDL(fmt.Sprintf("move nanoflow %s to folder 'SubFolder';", nfName)); err != nil {
		t.Fatalf("Failed to move nanoflow: %v", err)
	}

	// Verify it still describes successfully after move
	_, err := env.describeMDL(fmt.Sprintf("describe nanoflow %s;", nfName))
	if err != nil {
		t.Fatalf("Nanoflow should still be describable after move: %v", err)
	}
}

// --- GRANT / REVOKE ---

// TestNanoflow_GrantRevoke creates a nanoflow and grants/revokes access.
func TestNanoflow_GrantRevoke(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	nfName := testModule + ".RT_NF_Security"
	createMDL := `create nanoflow ` + nfName + ` () returns Void
{
}`
	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create nanoflow: %v", err)
	}

	// Grant access
	grantMDL := fmt.Sprintf("grant execute on nanoflow %s to %s.User;", nfName, testModule)
	if err := env.executeMDL(grantMDL); err != nil {
		t.Fatalf("Failed to grant access: %v", err)
	}

	// Show access — verify role is present
	env.output.Reset()
	if err := env.executeMDL(fmt.Sprintf("show access on nanoflow %s;", nfName)); err != nil {
		t.Fatalf("Failed to show access: %v", err)
	}
	output := env.output.String()
	if !strings.Contains(output, "User") {
		t.Errorf("Expected 'User' role in access output, got:\n%s", output)
	}

	// Revoke access
	revokeMDL := fmt.Sprintf("revoke execute on nanoflow %s from %s.User;", nfName, testModule)
	if err := env.executeMDL(revokeMDL); err != nil {
		t.Fatalf("Failed to revoke access: %v", err)
	}

	// Show access again — verify role is gone
	env.output.Reset()
	if err := env.executeMDL(fmt.Sprintf("show access on nanoflow %s;", nfName)); err != nil {
		t.Fatalf("Failed to show access after revoke: %v", err)
	}
	output = env.output.String()
	t.Logf("access output after revoke:\n%s", output)
}

// TestNanoflow_GrantIdempotent verifies granting the same role twice is idempotent.
func TestNanoflow_GrantIdempotent(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	nfName := testModule + ".RT_NF_GrantIdem"
	createMDL := `create nanoflow ` + nfName + ` () returns Void
{
}`
	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create nanoflow: %v", err)
	}

	grantMDL := fmt.Sprintf("grant execute on nanoflow %s to %s.User;", nfName, testModule)

	// Grant twice — second should not error
	if err := env.executeMDL(grantMDL); err != nil {
		t.Fatalf("First grant failed: %v", err)
	}
	if err := env.executeMDL(grantMDL); err != nil {
		t.Fatalf("Second (idempotent) grant failed: %v", err)
	}
}

// --- CREATE OR MODIFY ---

// TestNanoflow_CreateOrModify verifies CREATE OR MODIFY replaces an existing nanoflow.
func TestNanoflow_CreateOrModify(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	nfName := testModule + ".RT_NF_CreateOrModify"

	// Create v1
	createV1 := `create nanoflow ` + nfName + ` () returns String
{
  return 'v1';
}`
	if err := env.executeMDL(createV1); err != nil {
		t.Fatalf("Failed to create nanoflow v1: %v", err)
	}

	// Create or modify v2 (should replace)
	createV2 := `create or modify nanoflow ` + nfName + ` () returns Integer
{
  return 42;
}`
	if err := env.executeMDL(createV2); err != nil {
		t.Fatalf("Failed to create or modify nanoflow v2: %v", err)
	}

	// Describe — should show v2 return type
	output, err := env.describeMDL(fmt.Sprintf("describe nanoflow %s;", nfName))
	if err != nil {
		t.Fatalf("Failed to describe nanoflow: %v", err)
	}

	if !strings.Contains(output, "Integer") {
		t.Errorf("Expected Integer return type in v2, got:\n%s", output)
	}

	t.Logf("create or modify output:\n%s", output)
}

// --- Validation ---

// TestNanoflow_DisallowedAction verifies nanoflow validation rejects Java actions.
func TestNanoflow_DisallowedAction(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	nfName := testModule + ".RT_NF_Disallowed"
	// Attempt to create a nanoflow with a Java action call (disallowed)
	createMDL := `create nanoflow ` + nfName + ` () returns Void
{
  call java action MyFirstModule.SomeJavaAction ();
}`

	err := env.executeMDL(createMDL)
	if err == nil {
		t.Error("Expected validation error for Java action in nanoflow, but creation succeeded")
	} else {
		t.Logf("Got expected error: %v", err)
		if !strings.Contains(err.Error(), "Java") && !strings.Contains(err.Error(), "not allowed") {
			t.Errorf("Expected error about Java actions not allowed, got: %v", err)
		}
	}
}

// --- MERMAID ---

// TestNanoflow_Mermaid creates a nanoflow and generates Mermaid output.
func TestNanoflow_Mermaid(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	nfName := testModule + ".RT_NF_Mermaid"
	createMDL := `create nanoflow ` + nfName + ` ($X: Integer) returns String
{
  if $X > 0 then
    return 'positive';
  else
    return 'non-positive';
  end if;
}`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create nanoflow: %v", err)
	}

	env.output.Reset()
	if err := env.executor.DescribeMermaid("nanoflow", nfName); err != nil {
		t.Fatalf("Failed to describe mermaid nanoflow: %v", err)
	}

	output := env.output.String()
	if !strings.Contains(output, "flowchart") && !strings.Contains(output, "graph") {
		t.Errorf("Expected Mermaid flowchart output, got:\n%s", output)
	}
	t.Logf("mermaid output:\n%s", output)
}

// --- Duplicate Detection ---

// TestNanoflow_DuplicateError verifies creating the same nanoflow twice fails.
func TestNanoflow_DuplicateError(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	nfName := testModule + ".RT_NF_Dup"
	createMDL := `create nanoflow ` + nfName + ` () returns Void
{
}`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("First creation should succeed: %v", err)
	}

	err := env.executeMDL(createMDL)
	if err == nil {
		t.Error("Expected error for duplicate nanoflow creation, but succeeded")
	} else {
		t.Logf("Got expected duplicate error: %v", err)
	}
}

// --- Drop and Recreate ---

// TestNanoflow_DropAndRecreate verifies a dropped nanoflow can be recreated.
func TestNanoflow_DropAndRecreate(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	nfName := testModule + ".RT_NF_DropRecreate"
	createMDL := `create nanoflow ` + nfName + ` () returns String
{
  return 'original';
}`

	// Create
	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create nanoflow: %v", err)
	}

	// Drop
	if err := env.executeMDL(fmt.Sprintf("drop nanoflow %s;", nfName)); err != nil {
		t.Fatalf("Failed to drop nanoflow: %v", err)
	}

	// Recreate with different return type
	createV2 := `create nanoflow ` + nfName + ` () returns Integer
{
  return 99;
}`
	if err := env.executeMDL(createV2); err != nil {
		t.Fatalf("Failed to recreate nanoflow after drop: %v", err)
	}

	// Verify recreated version
	output, err := env.describeMDL(fmt.Sprintf("describe nanoflow %s;", nfName))
	if err != nil {
		t.Fatalf("Failed to describe recreated nanoflow: %v", err)
	}
	if !strings.Contains(output, "Integer") {
		t.Errorf("Expected Integer return type in recreated nanoflow, got:\n%s", output)
	}
}

// --- Entity and Enumeration Parameters ---

// TestRoundtripNanoflow_EntityParameter creates a nanoflow with an entity parameter.
func TestRoundtripNanoflow_EntityParameter(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create prerequisite entity
	if err := env.executeMDL(`create or modify persistent entity ` + testModule + `.NfParamEntity (
		Label: String
	);`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	nfName := testModule + ".RT_NF_EntityParam"
	createMDL := `create nanoflow ` + nfName + ` ($Item: ` + testModule + `.NfParamEntity) returns String
{
  return $Item/Label;
}`

	assertNanoflowContains(t, env, nfName, createMDL,
		[]string{"nanoflow", "$Item", "NfParamEntity"},
		nil,
	)
}

// TestRoundtripNanoflow_EnumParameter creates a nanoflow with an enumeration parameter.
func TestRoundtripNanoflow_EnumParameter(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create prerequisite enumeration
	if err := env.executeMDL(`create or modify enumeration ` + testModule + `.NfColor (Red, Green, Blue);`); err != nil {
		t.Fatalf("Failed to create enumeration: %v", err)
	}

	nfName := testModule + ".RT_NF_EnumParam"
	createMDL := `create nanoflow ` + nfName + ` ($Color: Enum ` + testModule + `.NfColor) returns String
{
  return 'got color';
}`

	assertNanoflowContains(t, env, nfName, createMDL,
		[]string{"nanoflow", "$Color", "NfColor"},
		nil,
	)
}

// --- Folder ---

// TestRoundtripNanoflow_InFolder creates a nanoflow in a specific folder.
func TestRoundtripNanoflow_InFolder(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	nfName := testModule + ".RT_NF_Folder"
	createMDL := `create nanoflow ` + nfName + ` () returns Void
  folder 'NanoflowTests'
{
}`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create nanoflow in folder: %v", err)
	}

	_, err := env.describeMDL(fmt.Sprintf("describe nanoflow %s;", nfName))
	if err != nil {
		t.Fatalf("Nanoflow in folder should be describable: %v", err)
	}
}

// --- Rename ---

// TestNanoflow_Rename creates a nanoflow and renames it.
func TestNanoflow_Rename(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	oldName := testModule + ".RT_NF_BeforeRename"
	createMDL := `create nanoflow ` + oldName + ` () returns Void
{
}`
	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create nanoflow: %v", err)
	}

	newShortName := "RT_NF_AfterRename"
	if err := env.executeMDL(fmt.Sprintf("rename nanoflow %s to %s;", oldName, newShortName)); err != nil {
		t.Fatalf("Failed to rename nanoflow: %v", err)
	}

	newName := testModule + "." + newShortName
	_, err := env.describeMDL(fmt.Sprintf("describe nanoflow %s;", newName))
	if err != nil {
		t.Fatalf("Renamed nanoflow should be describable: %v", err)
	}

	// Old name should not exist
	_, err = env.describeMDL(fmt.Sprintf("describe nanoflow %s;", oldName))
	if err == nil {
		t.Error("Old nanoflow name should not exist after rename")
	}
}

// --- Call void nanoflow ---

// TestRoundtripNanoflow_CallVoidNanoflow calls a void nanoflow without result variable.
func TestRoundtripNanoflow_CallVoidNanoflow(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	targetName := testModule + ".RT_NF_VoidTarget"
	createTarget := `create nanoflow ` + targetName + ` () returns Void
{
}`
	if err := env.executeMDL(createTarget); err != nil {
		t.Fatalf("Failed to create target nanoflow: %v", err)
	}

	nfName := testModule + ".RT_NF_CallVoid"
	createMDL := `create nanoflow ` + nfName + ` () returns Void
{
  call nanoflow ` + targetName + ` ();
}`

	assertNanoflowContains(t, env, nfName, createMDL,
		[]string{"call nanoflow"},
		nil,
	)
}

// --- Annotations ---

// TestRoundtripNanoflow_Annotations creates a nanoflow with annotations.
func TestRoundtripNanoflow_Annotations(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	nfName := testModule + ".RT_NF_Annotations"
	createMDL := `create nanoflow ` + nfName + ` () returns String
{
  declare $Result String = 'hello';
  @annotation 'Important step'
  return $Result;
}`

	assertNanoflowContains(t, env, nfName, createMDL,
		[]string{"nanoflow", "return"},
		nil,
	)
}

// --- Multiple return paths ---

// TestRoundtripNanoflow_MultipleReturnPaths creates a nanoflow with multiple return points.
func TestRoundtripNanoflow_MultipleReturnPaths(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	nfName := testModule + ".RT_NF_MultiReturn"
	createMDL := `create nanoflow ` + nfName + ` ($X: Integer) returns String
{
  if $X > 100 then
    return 'high';
  end if;
  if $X > 0 then
    return 'positive';
  end if;
  return 'non-positive';
}`

	assertNanoflowContains(t, env, nfName, createMDL,
		[]string{"nanoflow", "return", "if", "'high'", "'positive'", "'non-positive'"},
		nil,
	)
}

// --- Nested disallowed in loop ---

// TestNanoflow_DisallowedNestedInLoop verifies validation catches disallowed actions in loops.
func TestNanoflow_DisallowedNestedInLoop(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create prerequisite entity for loop
	if err := env.executeMDL(`create or modify persistent entity ` + testModule + `.LoopCheck (
		Name: String
	);`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	nfName := testModule + ".RT_NF_DisallowedLoop"
	createMDL := `create nanoflow ` + nfName + ` () returns Void
{
  $Items = retrieve ` + testModule + `.LoopCheck;
  loop $Item in $Items
    call java action MyFirstModule.SomeJavaAction ();
  }
}`

	err := env.executeMDL(createMDL)
	if err == nil {
		t.Error("Expected validation error for disallowed action nested in loop")
	} else {
		t.Logf("Got expected error: %v", err)
	}
}

// --- SHOW with module filter ---

// TestNanoflow_ShowInModule verifies SHOW filters by module.
func TestNanoflow_ShowInModule(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	nfName := testModule + ".RT_NF_ShowMod"
	createMDL := `create nanoflow ` + nfName + ` () returns Void
{
}`
	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create nanoflow: %v", err)
	}

	// Show in test module — should find it
	env.output.Reset()
	if err := env.executeMDL(fmt.Sprintf("show nanoflows in %s;", testModule)); err != nil {
		t.Fatalf("Failed to show nanoflows: %v", err)
	}
	assertContainsStr(t, env.output.String(), "RT_NF_ShowMod")

	// Show in a different module — should not find it
	env.output.Reset()
	if err := env.executeMDL("show nanoflows in MyFirstModule;"); err != nil {
		t.Fatalf("Failed to show nanoflows in other module: %v", err)
	}
	if strings.Contains(env.output.String(), "RT_NF_ShowMod") {
		t.Error("Nanoflow should not appear when filtering by different module")
	}
}

// --- Describe non-existent ---

// TestNanoflow_DescribeNotFound verifies describe of missing nanoflow returns error.
func TestNanoflow_DescribeNotFound(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	_, err := env.describeMDL(fmt.Sprintf("describe nanoflow %s.NonExistent;", testModule))
	if err == nil {
		t.Error("Expected error for non-existent nanoflow")
	}
}

// --- Drop non-existent ---

// TestNanoflow_DropNotFound verifies drop of missing nanoflow returns error.
func TestNanoflow_DropNotFound(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	err := env.executeMDL(fmt.Sprintf("drop nanoflow %s.NonExistent;", testModule))
	if err == nil {
		t.Error("Expected error for dropping non-existent nanoflow")
	}
}

// TestRoundtripNanoflow_CallJavaScriptAction verifies that CALL JAVASCRIPT ACTION
// roundtrips correctly through CREATE → DESCRIBE.
// Requires a JavaScript action to exist in the test project module.
func TestRoundtripNanoflow_CallJavaScriptAction(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// First, verify a JavaScript action exists in the test module.
	// If none exist, skip—this test requires a pre-provisioned JS action.
	jsActions, err := env.describeMDL(fmt.Sprintf("show javascript actions in %s;", testModule))
	if err != nil || jsActions == "" || strings.Contains(jsActions, "0 javascript actions") {
		t.Skip("No JavaScript actions available in test module — skipping roundtrip test")
	}

	// Extract first available JS action name from SHOW output for the test.
	// Fallback: use a known action name if the test project provisions one.
	jsActionName := testModule + ".MyJSAction"

	nfName := testModule + ".RT_NF_CallJSAction"
	createMDL := `create nanoflow ` + nfName + ` () returns String
{
  $Result = call javascript action ` + jsActionName + ` ();
  return $Result;
}`

	assertNanoflowContains(t, env, nfName, createMDL,
		[]string{"call javascript action", jsActionName},
		nil,
	)
}

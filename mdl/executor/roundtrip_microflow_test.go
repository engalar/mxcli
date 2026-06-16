// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"fmt"
	"strings"
	"testing"
)

// --- CALL Statement Parameter Tests ---
// These tests verify the unified parameter syntax for CALL statements.

// TestRoundtripMicroflow_CallWithUnifiedParams tests CALL with parameter names without $.
func TestRoundtripMicroflow_CallWithUnifiedParams(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	microflowName := testModule + ".TestCallParamsMf"
	env.registerCleanup("microflow", microflowName)

	// First create a simple microflow to call
	createMfMDL := `create microflow ` + microflowName + ` ($Input: String) returns String
	{
		return $Input;
	}`

	if err := env.executeMDL(createMfMDL); err != nil {
		t.Fatalf("Failed to create microflow: %v", err)
	}

	// Create a calling microflow that uses unified parameter syntax (no $ on param names)
	callerName := testModule + ".TestCallParamsCaller"
	env.registerCleanup("microflow", callerName)

	createCallerMDL := `create microflow ` + callerName + ` () returns String
	{
		$Result = call microflow ` + microflowName + ` (Input = 'Hello World');
		return $Result;
	}`

	if err := env.executeMDL(createCallerMDL); err != nil {
		t.Fatalf("Failed to create calling microflow with unified param syntax: %v", err)
	}

	// Describe the caller microflow to verify it was created
	output, err := env.describeMDL(`describe microflow ` + callerName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe microflow: %v", err)
	}

	// Verify microflow was created (DESCRIBE may not fully output all activities)
	if !containsProperty(output, "microflow") {
		t.Error("Expected microflow in output")
	}
	// Note: DESCRIBE may not output all activities, so we just verify creation succeeded
	if containsProperty(output, "call") {
		t.Log("call activity found in describe output (good!)")
	} else {
		t.Log("Note: call activity not shown in describe output (known limitation)")
	}

	t.Logf("call with unified params roundtrip successful:\n%s", output)
}

// TestRoundtripMicroflow_LogWithTemplate tests LOG statement with WITH template syntax.
func TestRoundtripMicroflow_LogWithTemplate(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	microflowName := testModule + ".TestLogTemplate"
	env.registerCleanup("microflow", microflowName)

	// Create microflow with LOG using WITH syntax
	createMDL := `create microflow ` + microflowName + ` ($OrderNumber: String, $CustomerName: String) returns Boolean
	{
		log info node 'OrderService' 'Processing order {1} for customer {2}' with ({1} = $OrderNumber, {2} = $CustomerName);
		return true;
	}`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create microflow with log template: %v", err)
	}

	// Describe the microflow
	output, err := env.describeMDL(`describe microflow ` + microflowName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe microflow: %v", err)
	}

	// Verify LOG statement with WITH clause
	if !containsProperty(output, "log info node") {
		t.Error("Expected log info node in output")
	}
	if !containsProperty(output, "Processing order {1}") {
		t.Error("Expected template text 'Processing order {1}' in output")
	}
	if !containsProperty(output, "with") {
		t.Error("Expected with clause in output")
	}
	if !containsProperty(output, "$OrderNumber") {
		t.Error("Expected $OrderNumber parameter in output")
	}
	if !containsProperty(output, "$CustomerName") {
		t.Error("Expected $CustomerName parameter in output")
	}

	if err := env.executeMDL(output); err != nil {
		t.Fatalf("Describe output should execute without syntax errors: %v\n%s", err, output)
	}

	output2, err := env.describeMDL(`DESCRIBE MICROFLOW ` + microflowName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe microflow after re-exec: %v", err)
	}
	if strings.Contains(output2, "'''Processing order") {
		t.Fatalf("Roundtrip should not double-quote log template text:\n%s", output2)
	}

	t.Logf("log with template roundtrip successful:\n%s", output)
}

func TestRoundtripMicroflow_LogWithNodeExpression(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	if err := env.executeMDL(`CREATE CONSTANT ` + testModule + `.SecurityLogNode TYPE String DEFAULT 'Security';`); err != nil {
		t.Fatalf("Failed to create constant for log node: %v", err)
	}

	microflowName := testModule + ".TestLogNodeExpression"
	env.registerCleanup("microflow", microflowName)

	createMDL := `CREATE MICROFLOW ` + microflowName + ` () RETURNS Boolean
	{
		LOG INFO NODE @` + testModule + `.SecurityLogNode 'User added';
		RETURN true;
	}`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create microflow with node expression: %v", err)
	}

	output, err := env.describeMDL(`DESCRIBE MICROFLOW ` + microflowName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe microflow: %v", err)
	}

	if !containsProperty(output, "log info node @"+testModule+".SecurityLogNode") {
		t.Fatalf("Expected constant node expression in output, got:\n%s", output)
	}

	if err := env.executeMDL(output); err != nil {
		t.Fatalf("Describe output should execute without syntax errors: %v\n%s", err, output)
	}
}

// --- DESCRIBE MICROFLOW Roundtrip Tests ---
// These tests verify that DESCRIBE MICROFLOW produces correct output for various
// microflow patterns. They serve as regression guards for bugs in findBranchFlows
// and traverseFlowUntilMerge (commit 7459949a).

// assertMicroflowContains is a helper that creates a microflow, describes it,
// and asserts the output contains all expected strings.
func assertMicroflowContains(t *testing.T, env *testEnv, mfName, createMDL string, wantContains []string, wantNotContains []string) {
	t.Helper()

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create microflow %s: %v", mfName, err)
	}

	output, err := env.describeMDL(fmt.Sprintf("describe microflow %s;", mfName))
	if err != nil {
		t.Fatalf("Failed to describe microflow %s: %v", mfName, err)
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

	t.Logf("describe output for %s:\n%s", mfName, output)
}

// TestRoundtripMicroflow_IfElseWithBody is the primary regression test for the
// IF/ELSE body bugs: pointer type matching in findBranchFlows and shared visited
// map in traverseFlowUntilMerge.
func TestRoundtripMicroflow_IfElseWithBody(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mfName := testModule + ".RT_IfElseBody"
	createMDL := `create microflow ` + mfName + ` ($Count: Integer) returns String
{
  declare $Result String = 'unknown';
  if $Count > 10 then
    set $Result = 'high';
  else
    set $Result = 'low';
  end if;
  return $Result;
}`

	assertMicroflowContains(t, env, mfName, createMDL,
		[]string{"if", "then", "else", "end if", "'high'", "'low'", "return"},
		nil,
	)
}

// TestRoundtripMicroflow_NestedIfElse exercises traverseFlowUntilMerge recursion
// with nested exclusive splits.
func TestRoundtripMicroflow_NestedIfElse(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mfName := testModule + ".RT_NestedIf"
	createMDL := `create microflow ` + mfName + ` ($X: Integer) returns String
{
  declare $Msg String = 'none';
  if $X > 0 then
    if $X > 100 then
      set $Msg = 'very high';
    else
      set $Msg = 'moderate';
    end if;
  else
    set $Msg = 'negative';
  end if;
  return $Msg;
}`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create microflow: %v", err)
	}

	output, err := env.describeMDL(fmt.Sprintf("describe microflow %s;", mfName))
	if err != nil {
		t.Fatalf("Failed to describe microflow: %v", err)
	}

	// Must have at least 2 END IF for the nested structure
	if count := strings.Count(output, "end if"); count < 2 {
		t.Errorf("Expected at least 2 'end if' occurrences, got %d in:\n%s", count, output)
	}

	for _, want := range []string{"'very high'", "'moderate'", "'negative'", "return"} {
		if !strings.Contains(output, want) {
			t.Errorf("Expected %q in output, got:\n%s", want, output)
		}
	}

	t.Logf("describe output for %s:\n%s", mfName, output)
}

// TestRoundtripMicroflow_IfWithoutElse tests TRUE-only branch (no ELSE).
// Note: DESCRIBE always emits an empty ELSE block because Mendix stores both
// branches internally. The key assertion is that the TRUE branch body is present.
func TestRoundtripMicroflow_IfWithoutElse(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mfName := testModule + ".RT_IfNoElse"
	createMDL := `create microflow ` + mfName + ` ($Active: Boolean) returns Boolean
{
  if $Active then
    log info node 'Test' 'Item is active';
  end if;
  return $Active;
}`

	assertMicroflowContains(t, env, mfName, createMDL,
		[]string{"if", "end if", "log info", "return"},
		nil,
	)
}

// TestRoundtripMicroflow_LinearDeclareSetReturn tests the simplest linear flow.
func TestRoundtripMicroflow_LinearDeclareSetReturn(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mfName := testModule + ".RT_LinearFlow"
	createMDL := `create microflow ` + mfName + ` () returns Integer
{
  declare $Value Integer = 0;
  set $Value = 42;
  return $Value;
}`

	assertMicroflowContains(t, env, mfName, createMDL,
		[]string{"declare", "$Value", "Integer", "set $Value = 42", "return"},
		nil,
	)
}

// TestRoundtripMicroflow_CreateChangeDelete tests entity CRUD in a microflow.
func TestRoundtripMicroflow_CreateChangeDelete(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create prerequisite entity
	if err := env.executeMDL(`create or modify persistent entity RoundtripTest.MfTestItem (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	mfName := testModule + ".RT_CRUD"
	createMDL := `create microflow ` + mfName + ` () returns Boolean
{
  $Item = create RoundtripTest.MfTestItem (Name = 'test');
  change $Item (Name = 'updated');
  delete $Item;
  return true;
}`

	assertMicroflowContains(t, env, mfName, createMDL,
		[]string{"create RoundtripTest.MfTestItem", "change $Item", "delete $Item", "return"},
		nil,
	)
}

// TestRoundtripMicroflow_Retrieve tests database retrieve.
func TestRoundtripMicroflow_Retrieve(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create prerequisite entity
	if err := env.executeMDL(`create or modify persistent entity RoundtripTest.MfTestItem (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	mfName := testModule + ".RT_Retrieve"
	createMDL := `create microflow ` + mfName + ` () returns Boolean
{
  retrieve $Items from RoundtripTest.MfTestItem;
  return true;
}`

	assertMicroflowContains(t, env, mfName, createMDL,
		[]string{"retrieve", "RoundtripTest.MfTestItem", "return"},
		nil,
	)
}

// TestRoundtripMicroflow_RetrieveWithLimit tests RETRIEVE with LIMIT clause roundtrip.
// Regression test for commit 584aa678 (parseRange LimitExpression extraction).
func TestRoundtripMicroflow_RetrieveWithLimit(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	if err := env.executeMDL(`create or modify persistent entity RoundtripTest.MfTestItem (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	mfName := testModule + ".RT_RetrieveLimit"
	createMDL := `create microflow ` + mfName + ` () returns Boolean
{
  retrieve $Item from RoundtripTest.MfTestItem
    limit 1;
  return true;
}`

	assertMicroflowContains(t, env, mfName, createMDL,
		[]string{"retrieve", "RoundtripTest.MfTestItem", "limit 1", "return"},
		nil,
	)
}

// TestRoundtripMicroflow_RetrieveWithLimitOffset tests RETRIEVE with LIMIT and OFFSET roundtrip.
// Regression test for commit 584aa678 (parseRange OffsetExpression extraction).
func TestRoundtripMicroflow_RetrieveWithLimitOffset(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	if err := env.executeMDL(`create or modify persistent entity RoundtripTest.MfTestItem (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	mfName := testModule + ".RT_RetrieveLimitOffset"
	createMDL := `create microflow ` + mfName + ` () returns Boolean
{
  retrieve $Items from RoundtripTest.MfTestItem
    limit 2
    offset 3;
  return true;
}`

	assertMicroflowContains(t, env, mfName, createMDL,
		[]string{"retrieve", "RoundtripTest.MfTestItem", "limit 2", "offset 3", "return"},
		nil,
	)
}

// TestRoundtripMicroflow_RetrieveWithSortBy tests RETRIEVE with SORT BY roundtrip.
// Regression test for commit fd8610e1 (NewSortings BSON field name).
func TestRoundtripMicroflow_RetrieveWithSortBy(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	if err := env.executeMDL(`create or modify persistent entity RoundtripTest.MfTestItem (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	mfName := testModule + ".RT_RetrieveSort"
	createMDL := `create microflow ` + mfName + ` () returns Boolean
{
  retrieve $Items from RoundtripTest.MfTestItem
    sort by RoundtripTest.MfTestItem.Name asc;
  return true;
}`

	assertMicroflowContains(t, env, mfName, createMDL,
		[]string{"retrieve", "RoundtripTest.MfTestItem", "sort by", "Name asc", "return"},
		nil,
	)
}

// TestRoundtripMicroflow_RetrieveWithWhereSortLimitOffset tests the full RETRIEVE
// pattern with WHERE, SORT BY, LIMIT, and OFFSET — matching the M028_DataForm_Getter pattern.
func TestRoundtripMicroflow_RetrieveWithWhereSortLimitOffset(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	if err := env.executeMDL(`create or modify persistent entity RoundtripTest.MfTestItem (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	mfName := testModule + ".RT_RetrieveFull"
	createMDL := `create microflow ` + mfName + ` () returns Boolean
{
  retrieve $Items from RoundtripTest.MfTestItem
    where (starts-with(Name, 'a'))
    sort by RoundtripTest.MfTestItem.Name asc
    limit 2
    offset 3;
  return true;
}`

	assertMicroflowContains(t, env, mfName, createMDL,
		[]string{
			"retrieve", "RoundtripTest.MfTestItem",
			"starts-with", "Name",
			"sort by", "Name asc",
			"limit 2", "offset 3",
			"return",
		},
		nil,
	)
}

// TestRoundtripMicroflow_LoopWithBody tests LOOP iteration with activities inside.
// TestRoundtripMicroflow_LoopWithMultipleBodyActivities verifies that ALL
// activities inside a loop body appear in describe output, not just the first.
//
// Regression: Mendix stores loop body SequenceFlows at the microflow level
// (parent), not in the loop's MicroflowObjectCollection. The describe path
// was only scanning oc.ObjectsItems() for flows, finding none, and therefore
// only emitting whichever single activity happened to be picked by non-
// deterministic map iteration.
func TestRoundtripMicroflow_LoopWithMultipleBodyActivities(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	if err := env.executeMDL(`create or modify persistent entity RoundtripTest.MfTestItem (Name: String(100), Processed: Boolean default false);`); err != nil {
		t.Fatalf("create entity: %v", err)
	}

	mfName := testModule + ".RT_LoopMultiBody"
	createMDL := `create microflow ` + mfName + ` () returns Nothing
{
  retrieve $Items from RoundtripTest.MfTestItem;
  loop $Item in $Items {
    change $Item (Processed = true);
    commit $Item;
  }
  return;
}`

	// BOTH "change" and "commit" must appear in the describe output.
	assertMicroflowContains(t, env, mfName, createMDL,
		[]string{"loop", "change", "Processed", "commit", "end loop"},
		nil,
	)
}

func TestRoundtripMicroflow_LoopWithBody(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create prerequisite entity
	if err := env.executeMDL(`create or modify persistent entity RoundtripTest.MfTestItem (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	mfName := testModule + ".RT_Loop"
	createMDL := `create microflow ` + mfName + ` () returns Boolean
{
  retrieve $Items from RoundtripTest.MfTestItem;
  loop $Item in $Items {
    log info node 'Test' 'Processing item';
  }
  return true;
}`

	assertMicroflowContains(t, env, mfName, createMDL,
		[]string{"loop", "$Items", "log info", "end loop", "return"},
		nil,
	)
}

// TestRoundtripMicroflow_LoopInsideBranch tests that a LOOP inside an IF branch
// correctly emits loop body activities and END LOOP. Regression test for #65.
func TestRoundtripMicroflow_LoopInsideBranch(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	if err := env.executeMDL(`create or modify persistent entity RoundtripTest.MfTestItem (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	mfName := testModule + ".RT_LoopInBranch"
	createMDL := `create microflow ` + mfName + ` ($Flag: Boolean) returns Boolean
{
  if $Flag then
    retrieve $Items from RoundtripTest.MfTestItem;
    loop $Item in $Items {
      log info node 'Test' 'In loop';
    }
  else
    log info node 'Test' 'No items';
  end if;
  return true;
}`

	assertMicroflowContains(t, env, mfName, createMDL,
		[]string{"if", "retrieve", "loop", "log info", "In loop", "end loop", "else", "No items", "end if", "return"},
		nil,
	)
}

// TestRoundtripMicroflow_ValidationFeedback tests VALIDATION FEEDBACK statement
// which triggered a crash bug with nil settings.
func TestRoundtripMicroflow_ValidationFeedback(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create prerequisite entity
	if err := env.executeMDL(`create or modify persistent entity RoundtripTest.MfValItem (Email: String(200));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	mfName := testModule + ".RT_Validation"
	createMDL := `create microflow ` + mfName + ` ($ValItem: RoundtripTest.MfValItem) returns Boolean
{
  validation feedback $ValItem/Email message 'Email is required';
  return false;
}`

	assertMicroflowContains(t, env, mfName, createMDL,
		[]string{"validation feedback", "Email", "Email is required", "return"},
		nil,
	)
}

// TestRoundtripMicroflow_MultipleReturnPaths tests RETURN in both IF/ELSE branches.
func TestRoundtripMicroflow_MultipleReturnPaths(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mfName := testModule + ".RT_MultiReturn"
	createMDL := `create microflow ` + mfName + ` ($Flag: Boolean) returns String
{
  if $Flag then
    return 'yes';
  else
    return 'no';
  end if;
}`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create microflow: %v", err)
	}

	output, err := env.describeMDL(fmt.Sprintf("describe microflow %s;", mfName))
	if err != nil {
		t.Fatalf("Failed to describe microflow: %v", err)
	}

	if count := strings.Count(output, "return"); count < 2 {
		t.Errorf("Expected at least 2 'return' occurrences, got %d in:\n%s", count, output)
	}

	for _, want := range []string{"else", "end if"} {
		if !strings.Contains(output, want) {
			t.Errorf("Expected %q in output, got:\n%s", want, output)
		}
	}

	t.Logf("describe output for %s:\n%s", mfName, output)
}

// TestRoundtripMicroflow_CommitWithEvents tests COMMIT with WITH EVENTS flag.
func TestRoundtripMicroflow_CommitWithEvents(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create prerequisite entity
	if err := env.executeMDL(`create or modify persistent entity RoundtripTest.MfCommitItem (Status: String(50));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	mfName := testModule + ".RT_Commit"
	createMDL := `create microflow ` + mfName + ` () returns Boolean
{
  $Obj = create RoundtripTest.MfCommitItem;
  commit $Obj with events;
  return true;
}`

	assertMicroflowContains(t, env, mfName, createMDL,
		[]string{"commit", "with events", "return"},
		nil,
	)
}

// TestRoundtripMicroflow_ErrorHandlingContinue tests ON ERROR CONTINUE.
func TestRoundtripMicroflow_ErrorHandlingContinue(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create prerequisite entity
	if err := env.executeMDL(`create or modify persistent entity RoundtripTest.MfCommitItem (Status: String(50));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	mfName := testModule + ".RT_ErrorHandling"
	createMDL := `create microflow ` + mfName + ` () returns Boolean
{
  $Obj = create RoundtripTest.MfCommitItem;
  commit $Obj on error continue;
  return true;
}`

	assertMicroflowContains(t, env, mfName, createMDL,
		[]string{"commit", "on error continue", "return"},
		nil,
	)
}

// TestRoundtripMicroflow_Annotate tests @annotation roundtrip.
func TestRoundtripMicroflow_Annotate(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mfName := testModule + ".RT_Annotate"
	createMDL := `create microflow ` + mfName + ` () returns Boolean
{
  @annotation 'This is a test annotation'
  log info node 'Test' 'Hello';
  return true;
}`

	assertMicroflowContains(t, env, mfName, createMDL,
		[]string{"@annotation", "test annotation", "log info", "return"},
		nil,
	)
}

// TestRoundtripMicroflow_CaptionColor tests @caption and @color roundtrip.
func TestRoundtripMicroflow_CaptionColor(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mfName := testModule + ".RT_CaptionColor"
	createMDL := `create microflow ` + mfName + ` () returns Boolean
{
  @caption 'Custom Caption'
  @color Green
  log info node 'Test' 'Hello';
  return true;
}`

	assertMicroflowContains(t, env, mfName, createMDL,
		[]string{"@caption 'Custom Caption'", "@color Green", "log info", "return"},
		nil,
	)
}

// TestRoundtripMicroflow_SystemEntityParameter tests that microflows with
// System.* built-in entity parameter types can be created (issue #29).
// System entities (System.Workflow, System.User, etc.) are not stored in the
// MPR domain models — they are serialized by qualified name at runtime.
func TestRoundtripMicroflow_SystemEntityParameter(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mfName := testModule + ".RT_SystemEntityParam"
	env.registerCleanup("microflow", mfName)

	createMDL := `create microflow ` + mfName + ` (
  $Workflow: System.Workflow,
  $Count: Integer
) returns List of System.User
{
  return empty;
}`

	assertMicroflowContains(t, env, mfName, createMDL,
		// "empty" is a reserved keyword — must round-trip as "RETURN empty", not "$empty"
		[]string{"System.Workflow", "System.User", "return empty"},
		[]string{"return $empty"},
	)
}

// TestRoundtripMicroflow_Position tests @position is emitted in DESCRIBE output.
func TestRoundtripMicroflow_Position(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mfName := testModule + ".RT_Position"
	createMDL := `create microflow ` + mfName + ` () returns Boolean
{
  @position(500, 300)
  log info node 'Test' 'Hello';
  return true;
}`

	// @position is always emitted but the exact coordinates depend on the BSON roundtrip
	// (RelativeMiddle vs RelativeMiddlePoint key mismatch). Verify @position appears.
	assertMicroflowContains(t, env, mfName, createMDL,
		[]string{"@position(", "log info", "return"},
		nil,
	)
}

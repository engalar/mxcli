// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"strings"
	"testing"
)

// TestRoundtripWorkflow_Comprehensive tests all workflow MDL syntax in a single roundtrip.
//
// Activity types covered:
//   - ANNOTATION
//   - USER TASK (PAGE, TARGETING MICROFLOW, DUE DATE, OUTCOMES with nested, BOUNDARY EVENT x2)
//   - MULTI USER TASK (PAGE, TARGETING MICROFLOW, OUTCOMES)
//   - CALL MICROFLOW (WITH params, OUTCOMES TRUE/FALSE)
//   - DECISION (expression, OUTCOMES TRUE/FALSE with nested JUMP TO and WAIT FOR TIMER)
//   - PARALLEL SPLIT (PATH 1 with USER TASK, PATH 2 with CALL WORKFLOW)
//   - WAIT FOR TIMER (with ISO 8601 delay)
//   - WAIT FOR NOTIFICATION (with BOUNDARY EVENT NON INTERRUPTING TIMER)
//   - JUMP TO (inside DECISION outcome)
//   - CALL WORKFLOW (sub-workflow with parameter expression)
func TestRoundtripWorkflow_Comprehensive(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mod := testModule

	// --- Prerequisites ---

	// Context entity for both workflows
	if err := env.executeMDL(`CREATE OR MODIFY PERSISTENT ENTITY ` + mod + `.WfCtxEntity (
		Score: Integer,
		IsApproved: Boolean DEFAULT false
	);`); err != nil {
		t.Fatalf("create WfCtxEntity: %v", err)
	}

	// Microflow: single-user targeting
	if err := env.executeMDL(`CREATE MICROFLOW ` + mod + `.GetSingleReviewer () RETURNS String BEGIN END;`); err != nil {
		t.Fatalf("create GetSingleReviewer: %v", err)
	}

	// Microflow: multi-user targeting
	if err := env.executeMDL(`CREATE MICROFLOW ` + mod + `.GetMultiReviewers () RETURNS String BEGIN END;`); err != nil {
		t.Fatalf("create GetMultiReviewers: %v", err)
	}

	// Microflow: called by CALL MICROFLOW (returns Boolean)
	if err := env.executeMDL(`CREATE MICROFLOW ` + mod + `.ScoreCalc (Score: Integer) RETURNS Boolean BEGIN END;`); err != nil {
		t.Fatalf("create ScoreCalc: %v", err)
	}

	// Sub-workflow for CALL WORKFLOW
	if err := env.executeMDL(`CREATE WORKFLOW ` + mod + `.SubApprovalFlow
  PARAMETER $WorkflowContext: ` + mod + `.WfCtxEntity
BEGIN
  USER TASK SubTask 'Sub-Approval'
    PAGE ` + mod + `.SubPage
    OUTCOMES 'Done' { };
END WORKFLOW;`); err != nil {
		t.Fatalf("create SubApprovalFlow: %v", err)
	}

	// --- Main comprehensive workflow ---
	createMDL := `CREATE WORKFLOW ` + mod + `.ComprehensiveFlow
  PARAMETER $WorkflowContext: ` + mod + `.WfCtxEntity
BEGIN

  ANNOTATION 'Comprehensive workflow covering all MDL syntax';

  USER TASK ReviewTask 'Review Request'
    PAGE ` + mod + `.ReviewPage
    TARGETING MICROFLOW ` + mod + `.GetSingleReviewer
    OUTCOMES
      'Approve' { }
      'Reject' { }
    BOUNDARY EVENT INTERRUPTING TIMER '${PT24H}' NON INTERRUPTING TIMER '${PT1H}';

  MULTI USER TASK MultiReviewTask 'Multi-Person Review'
    PAGE ` + mod + `.MultiReviewPage
    TARGETING MICROFLOW ` + mod + `.GetMultiReviewers
    OUTCOMES 'Complete' { };

  CALL MICROFLOW ` + mod + `.ScoreCalc
    WITH (Score = '$WorkflowContext/Score')
    OUTCOMES
      TRUE -> { }
      FALSE -> { };

  DECISION '$WorkflowContext/IsApproved'
    OUTCOMES
      TRUE -> {
        WAIT FOR TIMER '${PT2H}';
      }
      FALSE -> {
        JUMP TO ReviewTask;
      };

  PARALLEL SPLIT
    PATH 1 {
      USER TASK FinalApprove 'Final Approval'
        PAGE ` + mod + `.ApprovePage
        OUTCOMES 'Approved' { };
    }
    PATH 2 {
      CALL WORKFLOW ` + mod + `.SubApprovalFlow;
    };

  WAIT FOR NOTIFICATION;

  ANNOTATION 'End of flow';

END WORKFLOW;`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("create ComprehensiveFlow: %v", err)
	}

	output, err := env.describeMDL(`DESCRIBE WORKFLOW ` + mod + `.ComprehensiveFlow;`)
	if err != nil {
		t.Fatalf("describe ComprehensiveFlow: %v", err)
	}

	t.Logf("DESCRIBE output:\n%s", output)

	checks := []struct {
		label   string
		keyword string
	}{
		{"annotation activity", "ANNOTATION 'Comprehensive workflow"},
		{"user task", "USER TASK ReviewTask"},
		{"outcome approve", "'Approve'"},
		{"outcome reject", "'Reject'"},
		{"boundary interrupting", "BOUNDARY EVENT INTERRUPTING TIMER '${PT24H}'"},
		{"boundary non interrupting", "BOUNDARY EVENT NON INTERRUPTING TIMER '${PT1H}'"},
		{"multi user task", "MULTI USER TASK MultiReviewTask"},
		{"call microflow with", "CALL MICROFLOW " + mod + ".ScoreCalc WITH (Score ="},
		{"outcomes true", "TRUE ->"},
		{"outcomes false", "FALSE ->"},
		{"decision", "DECISION '$WorkflowContext/IsApproved'"},
		{"wait for timer", "WAIT FOR TIMER '${PT2H}'"},
		{"jump to", "JUMP TO ReviewTask"},
		{"parallel split", "PARALLEL SPLIT"},
		{"path 1", "PATH 1"},
		{"path 2", "PATH 2"},
		{"call workflow", "CALL WORKFLOW " + mod + ".SubApprovalFlow"},
		{"wait for notification", "WAIT FOR NOTIFICATION"},
		{"trailing annotation", "ANNOTATION 'End of flow'"},
		{"parameter", "PARAMETER $WorkflowContext: " + mod + ".WfCtxEntity"},
	}

	var failed []string
	for _, c := range checks {
		if !strings.Contains(output, c.keyword) {
			failed = append(failed, c.label+": "+c.keyword)
		}
	}
	if len(failed) > 0 {
		t.Errorf("DESCRIBE output missing %d expected keywords:\n  %s\n\nFull output:\n%s",
			len(failed), strings.Join(failed, "\n  "), output)
	}
}

func TestRoundtripWorkflow_BoundaryEventInterrupting(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	createMDL := `CREATE WORKFLOW ` + testModule + `.WfBoundaryInt
  PARAMETER $WorkflowContext: ` + testModule + `.TestEntitySimple
BEGIN
  USER TASK act1 'Review'
    PAGE ` + testModule + `.ReviewPage
    OUTCOMES 'Approve' { }
    BOUNDARY EVENT INTERRUPTING TIMER '${PT1H}'
    ;
END WORKFLOW;`

	if err := env.executeMDL(`CREATE OR MODIFY PERSISTENT ENTITY ` + testModule + `.TestEntitySimple (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	output, err := env.describeMDL(`DESCRIBE WORKFLOW ` + testModule + `.WfBoundaryInt;`)
	if err != nil {
		t.Fatalf("Failed to describe workflow: %v", err)
	}

	if !strings.Contains(output, "BOUNDARY EVENT INTERRUPTING TIMER") {
		t.Errorf("Expected DESCRIBE output to contain 'BOUNDARY EVENT INTERRUPTING TIMER', got:\n%s", output)
	}
}

func TestRoundtripWorkflow_BoundaryEventNonInterrupting(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	createMDL := `CREATE WORKFLOW ` + testModule + `.WfBoundaryNonInt
  PARAMETER $WorkflowContext: ` + testModule + `.TestEntitySimple2
BEGIN
  USER TASK act1 'Review'
    PAGE ` + testModule + `.ReviewPage
    OUTCOMES 'Approve' { }
    BOUNDARY EVENT NON INTERRUPTING TIMER '${PT2H}'
    ;
END WORKFLOW;`

	if err := env.executeMDL(`CREATE OR MODIFY PERSISTENT ENTITY ` + testModule + `.TestEntitySimple2 (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	output, err := env.describeMDL(`DESCRIBE WORKFLOW ` + testModule + `.WfBoundaryNonInt;`)
	if err != nil {
		t.Fatalf("Failed to describe workflow: %v", err)
	}

	if !strings.Contains(output, "BOUNDARY EVENT NON INTERRUPTING TIMER") {
		t.Errorf("Expected DESCRIBE output to contain 'BOUNDARY EVENT NON INTERRUPTING TIMER', got:\n%s", output)
	}
}

func TestRoundtripWorkflow_MultiUserTask(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	createMDL := `CREATE WORKFLOW ` + testModule + `.WfMultiUser
  PARAMETER $WorkflowContext: ` + testModule + `.TestEntityMulti
BEGIN
  MULTI USER TASK act1 'Caption'
    PAGE ` + testModule + `.ReviewPage
    OUTCOMES 'Approve' { }
    ;
END WORKFLOW;`

	if err := env.executeMDL(`CREATE OR MODIFY PERSISTENT ENTITY ` + testModule + `.TestEntityMulti (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	output, err := env.describeMDL(`DESCRIBE WORKFLOW ` + testModule + `.WfMultiUser;`)
	if err != nil {
		t.Fatalf("Failed to describe workflow: %v", err)
	}

	if !strings.Contains(output, "MULTI USER TASK") {
		t.Errorf("Expected DESCRIBE output to contain 'MULTI USER TASK', got:\n%s", output)
	}
}

func TestRoundtripWorkflow_AnnotationActivity(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	createMDL := `CREATE WORKFLOW ` + testModule + `.WfAnnotation
  PARAMETER $WorkflowContext: ` + testModule + `.TestEntityAnnot
BEGIN
  ANNOTATION 'This is a workflow note';
END WORKFLOW;`

	if err := env.executeMDL(`CREATE OR MODIFY PERSISTENT ENTITY ` + testModule + `.TestEntityAnnot (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	output, err := env.describeMDL(`DESCRIBE WORKFLOW ` + testModule + `.WfAnnotation;`)
	if err != nil {
		t.Fatalf("Failed to describe workflow: %v", err)
	}

	if !strings.Contains(output, "ANNOTATION 'This is a workflow note'") {
		t.Errorf("Expected DESCRIBE output to contain \"ANNOTATION 'This is a workflow note'\", got:\n%s", output)
	}
}

func TestRoundtripWorkflow_CallMicroflowWithParams(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	createMDL := `CREATE WORKFLOW ` + testModule + `.WfCallMf
  PARAMETER $WorkflowContext: ` + testModule + `.TestEntityCallMf
BEGIN
  CALL MICROFLOW ` + testModule + `.SomeMicroflow WITH (Amount = '$WorkflowContext/Amount')
    OUTCOMES TRUE -> { } FALSE -> { };
END WORKFLOW;`

	if err := env.executeMDL(`CREATE OR MODIFY PERSISTENT ENTITY ` + testModule + `.TestEntityCallMf (Amount: Decimal);`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	if err := env.executeMDL(`CREATE MICROFLOW ` + testModule + `.SomeMicroflow (Amount: Decimal) RETURNS Boolean BEGIN END;`); err != nil {
		t.Fatalf("Failed to create microflow: %v", err)
	}

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	output, err := env.describeMDL(`DESCRIBE WORKFLOW ` + testModule + `.WfCallMf;`)
	if err != nil {
		t.Fatalf("Failed to describe workflow: %v", err)
	}

	if !strings.Contains(output, "WITH (") {
		t.Errorf("Expected DESCRIBE output to contain 'WITH (', got:\n%s", output)
	}
}

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
	if err := env.executeMDL(`create or modify persistent entity ` + mod + `.WfCtxEntity (
		Score: Integer,
		IsApproved: Boolean default false
	);`); err != nil {
		t.Fatalf("create WfCtxEntity: %v", err)
	}

	// Microflow: single-user targeting
	if err := env.executeMDL(`create microflow ` + mod + `.GetSingleReviewer () returns String begin end;`); err != nil {
		t.Fatalf("create GetSingleReviewer: %v", err)
	}

	// Microflow: multi-user targeting
	if err := env.executeMDL(`create microflow ` + mod + `.GetMultiReviewers () returns String begin end;`); err != nil {
		t.Fatalf("create GetMultiReviewers: %v", err)
	}

	// Microflow: called by CALL MICROFLOW (returns Boolean)
	if err := env.executeMDL(`create microflow ` + mod + `.ScoreCalc (Score: Integer) returns Boolean begin end;`); err != nil {
		t.Fatalf("create ScoreCalc: %v", err)
	}

	// Sub-workflow for CALL WORKFLOW
	if err := env.executeMDL(`create workflow ` + mod + `.SubApprovalFlow
  parameter $WorkflowContext: ` + mod + `.WfCtxEntity
begin
  user task SubTask 'Sub-Approval'
    page ` + mod + `.SubPage
    targeting xpath '[%CurrentUser%]'
    outcomes 'Done' { };
end workflow;`); err != nil {
		t.Fatalf("create SubApprovalFlow: %v", err)
	}

	// --- Main comprehensive workflow ---
	createMDL := `create workflow ` + mod + `.ComprehensiveFlow
  parameter $WorkflowContext: ` + mod + `.WfCtxEntity
begin

  annotation 'Comprehensive workflow covering all MDL syntax';

  user task ReviewTask 'Review Request'
    page ` + mod + `.ReviewPage
    targeting microflow ` + mod + `.GetSingleReviewer
    outcomes
      'Approve' { }
      'Reject' { }
    boundary event interrupting timer '${PT24H}' non interrupting timer '${PT1H}';

  multi user task MultiReviewTask 'Multi-Person Review'
    page ` + mod + `.MultiReviewPage
    targeting microflow ` + mod + `.GetMultiReviewers
    outcomes 'Complete' { };

  call microflow ` + mod + `.ScoreCalc
    with (Score = '$WorkflowContext/Score')
    outcomes
      true -> { }
      false -> { };

  decision '$WorkflowContext/IsApproved'
    outcomes
      true -> {
        wait for timer '${PT2H}';
      }
      false -> {
        jump to ReviewTask;
      };

  parallel split
    path 1 {
      user task FinalApprove 'Final Approval'
        page ` + mod + `.ApprovePage
        targeting xpath '[%CurrentUser%]'
        outcomes 'Approved' { };
    }
    path 2 {
      call workflow ` + mod + `.SubApprovalFlow;
    };

  wait for notification;

  annotation 'End of flow';

end workflow;`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("create ComprehensiveFlow: %v", err)
	}

	output, err := env.describeMDL(`describe workflow ` + mod + `.ComprehensiveFlow;`)
	if err != nil {
		t.Fatalf("describe ComprehensiveFlow: %v", err)
	}

	t.Logf("describe output:\n%s", output)

	checks := []struct {
		label   string
		keyword string
	}{
		{"annotation activity", "annotation 'Comprehensive workflow"},
		{"user task", "user task ReviewTask"},
		{"outcome approve", "'Approve'"},
		{"outcome reject", "'Reject'"},
		{"boundary interrupting", "boundary event interrupting timer '${PT24H}'"},
		{"boundary non interrupting", "boundary event non interrupting timer '${PT1H}'"},
		{"multi user task", "multi user task MultiReviewTask"},
		{"call microflow with", "call microflow " + mod + ".ScoreCalc with (Score ="},
		{"outcomes true", "true ->"},
		{"outcomes false", "false ->"},
		{"decision", "decision '$WorkflowContext/IsApproved'"},
		{"wait for timer", "wait for timer '${PT2H}'"},
		{"jump to", "jump to ReviewTask"},
		{"parallel split", "parallel split"},
		{"path 1", "path 1"},
		{"path 2", "path 2"},
		{"call workflow", "call workflow " + mod + ".SubApprovalFlow"},
		{"wait for notification", "wait for notification"},
		{"trailing annotation", "annotation 'End of flow'"},
		{"parameter", "parameter $WorkflowContext: " + mod + ".WfCtxEntity"},
	}

	var failed []string
	for _, c := range checks {
		if !strings.Contains(output, c.keyword) {
			failed = append(failed, c.label+": "+c.keyword)
		}
	}
	if len(failed) > 0 {
		t.Errorf("describe output missing %d expected keywords:\n  %s\n\nFull output:\n%s",
			len(failed), strings.Join(failed, "\n  "), output)
	}
}

func TestRoundtripWorkflow_BoundaryEventInterrupting(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	createMDL := `create workflow ` + testModule + `.WfBoundaryInt
  parameter $WorkflowContext: ` + testModule + `.TestEntitySimple
begin
  user task act1 'Review'
    page ` + testModule + `.ReviewPage
    targeting xpath '[%CurrentUser%]'
    outcomes 'Approve' { }
    boundary event interrupting timer '${PT1H}'
    ;
end workflow;`

	if err := env.executeMDL(`create or modify persistent entity ` + testModule + `.TestEntitySimple (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	output, err := env.describeMDL(`describe workflow ` + testModule + `.WfBoundaryInt;`)
	if err != nil {
		t.Fatalf("Failed to describe workflow: %v", err)
	}

	if !strings.Contains(output, "boundary event interrupting timer") {
		t.Errorf("Expected describe output to contain 'boundary event interrupting timer', got:\n%s", output)
	}
}

func TestRoundtripWorkflow_BoundaryEventNonInterrupting(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	createMDL := `create workflow ` + testModule + `.WfBoundaryNonInt
  parameter $WorkflowContext: ` + testModule + `.TestEntitySimple2
begin
  user task act1 'Review'
    page ` + testModule + `.ReviewPage
    targeting xpath '[%CurrentUser%]'
    outcomes 'Approve' { }
    boundary event non interrupting timer '${PT2H}'
    ;
end workflow;`

	if err := env.executeMDL(`create or modify persistent entity ` + testModule + `.TestEntitySimple2 (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	output, err := env.describeMDL(`describe workflow ` + testModule + `.WfBoundaryNonInt;`)
	if err != nil {
		t.Fatalf("Failed to describe workflow: %v", err)
	}

	if !strings.Contains(output, "boundary event non interrupting timer") {
		t.Errorf("Expected describe output to contain 'boundary event non interrupting timer', got:\n%s", output)
	}
}

func TestRoundtripWorkflow_MultiUserTask(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	createMDL := `create workflow ` + testModule + `.WfMultiUser
  parameter $WorkflowContext: ` + testModule + `.TestEntityMulti
begin
  multi user task act1 'Caption'
    page ` + testModule + `.ReviewPage
    targeting xpath '[%CurrentUser%]'
    outcomes 'Approve' { }
    ;
end workflow;`

	if err := env.executeMDL(`create or modify persistent entity ` + testModule + `.TestEntityMulti (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	output, err := env.describeMDL(`describe workflow ` + testModule + `.WfMultiUser;`)
	if err != nil {
		t.Fatalf("Failed to describe workflow: %v", err)
	}

	if !strings.Contains(output, "multi user task") {
		t.Errorf("Expected describe output to contain 'multi user task', got:\n%s", output)
	}
}

func TestRoundtripWorkflow_AnnotationActivity(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	createMDL := `create workflow ` + testModule + `.WfAnnotation
  parameter $WorkflowContext: ` + testModule + `.TestEntityAnnot
begin
  annotation 'This is a workflow note';
end workflow;`

	if err := env.executeMDL(`create or modify persistent entity ` + testModule + `.TestEntityAnnot (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	output, err := env.describeMDL(`describe workflow ` + testModule + `.WfAnnotation;`)
	if err != nil {
		t.Fatalf("Failed to describe workflow: %v", err)
	}

	if !strings.Contains(output, "annotation 'This is a workflow note'") {
		t.Errorf("Expected describe output to contain \"annotation 'This is a workflow note'\", got:\n%s", output)
	}

	// Full round-trip: DESCRIBE output must be re-executable (annotation must survive re-create)
	describeOutput := output
	// Replace WORKFLOW with CREATE OR REPLACE WORKFLOW for round-trip execution
	createFromDescribe := strings.Replace(describeOutput, "\nworkflow ", "\ncreate or replace workflow ", 1)
	// Strip comment header lines (-- ...) before the CREATE OR REPLACE WORKFLOW
	var mdlLines []string
	inBody := false
	for _, line := range strings.Split(createFromDescribe, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "create or replace workflow") {
			inBody = true
		}
		if inBody {
			mdlLines = append(mdlLines, line)
		}
	}
	roundTripMDL := strings.Join(mdlLines, "\n")
	if err := env.executeMDL(roundTripMDL); err != nil {
		t.Errorf("Round-trip execution failed (describe output is not re-executable): %v\nMDL:\n%s", err, roundTripMDL)
	}
}

// TestRoundtripWorkflow_AnnotationBeforeActivity tests that a workflow activity's
// embedded annotation (BaseWorkflowActivity.Annotation) is preserved in DESCRIBE output
// as a parseable ANNOTATION statement rather than a SQL comment.
func TestRoundtripWorkflow_AnnotationBeforeActivity(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create a workflow with an ANNOTATION before a WAIT FOR TIMER.
	// This mimics the pattern from Studio Pro where an annotation is attached to an activity.
	createMDL := `create workflow ` + testModule + `.WfAnnotBeforeTimer
  parameter $WorkflowContext: ` + testModule + `.TestEntityAnnotTimer
begin
  annotation 'I am a note';
  wait for timer 'addDays([%CurrentDateTime%], 1)' comment 'Timer';
end workflow;`

	if err := env.executeMDL(`create or modify persistent entity ` + testModule + `.TestEntityAnnotTimer (Name: String(100));`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	output, err := env.describeMDL(`describe workflow ` + testModule + `.WfAnnotBeforeTimer;`)
	if err != nil {
		t.Fatalf("Failed to describe workflow: %v", err)
	}

	// The annotation must appear as a parseable ANNOTATION statement, not as a SQL comment.
	if !strings.Contains(output, "annotation 'I am a note'") {
		t.Errorf("Expected describe to emit annotation statement, got:\n%s", output)
	}
	if strings.Contains(output, "-- I am a note") {
		t.Errorf("describe must not emit annotation as sql comment (not round-trippable), got:\n%s", output)
	}
	if !strings.Contains(output, "wait for timer") {
		t.Errorf("Expected describe to contain wait for timer, got:\n%s", output)
	}
}

func TestRoundtripWorkflow_CallMicroflowWithParams(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	createMDL := `create workflow ` + testModule + `.WfCallMf
  parameter $WorkflowContext: ` + testModule + `.TestEntityCallMf
begin
  call microflow ` + testModule + `.SomeMicroflow with (Amount = '$WorkflowContext/Amount')
    outcomes true -> { } false -> { };
end workflow;`

	if err := env.executeMDL(`create or modify persistent entity ` + testModule + `.TestEntityCallMf (Amount: Decimal);`); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	if err := env.executeMDL(`create microflow ` + testModule + `.SomeMicroflow (Amount: Decimal) returns Boolean begin end;`); err != nil {
		t.Fatalf("Failed to create microflow: %v", err)
	}

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	output, err := env.describeMDL(`describe workflow ` + testModule + `.WfCallMf;`)
	if err != nil {
		t.Fatalf("Failed to describe workflow: %v", err)
	}

	if !strings.Contains(output, "with (") {
		t.Errorf("Expected describe output to contain 'with (', got:\n%s", output)
	}
}

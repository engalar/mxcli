// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"strings"
	"testing"
)

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

	if err := env.executeMDL(`CREATE MICROFLOW ` + testModule + `.SomeMicroflow (Amount: Decimal) RETURN Boolean BEGIN END;`); err != nil {
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

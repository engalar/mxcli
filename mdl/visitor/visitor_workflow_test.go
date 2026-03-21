// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestWorkflowVisitor_BoundaryEventInterrupting(t *testing.T) {
	input := `CREATE WORKFLOW M.TestWF
BEGIN
  USER TASK act1 'Caption'
    OUTCOMES 'Done' { }
    BOUNDARY EVENT INTERRUPTING TIMER '${PT1H}';
END WORKFLOW;`

	prog, errs := Build(input)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("Parse error: %v", err)
		}
		t.FailNow()
	}

	if len(prog.Statements) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(prog.Statements))
	}

	stmt, ok := prog.Statements[0].(*ast.CreateWorkflowStmt)
	if !ok {
		t.Fatalf("Expected CreateWorkflowStmt, got %T", prog.Statements[0])
	}

	if len(stmt.Activities) == 0 {
		t.Fatal("Expected at least 1 activity")
	}

	userTask, ok := stmt.Activities[0].(*ast.WorkflowUserTaskNode)
	if !ok {
		t.Fatalf("Expected WorkflowUserTaskNode, got %T", stmt.Activities[0])
	}

	if len(userTask.BoundaryEvents) == 0 {
		t.Fatal("Expected at least 1 boundary event")
	}

	be := userTask.BoundaryEvents[0]
	if be.EventType != "InterruptingTimer" {
		t.Errorf("Expected EventType 'InterruptingTimer', got %q", be.EventType)
	}
	if be.Delay != "${PT1H}" {
		t.Errorf("Expected Delay '${PT1H}', got %q", be.Delay)
	}
}

func TestWorkflowVisitor_BoundaryEventNonInterrupting(t *testing.T) {
	input := `CREATE WORKFLOW M.TestWF
BEGIN
  USER TASK act1 'Caption'
    OUTCOMES 'Done' { }
    BOUNDARY EVENT NON INTERRUPTING TIMER '${PT2H}';
END WORKFLOW;`

	prog, errs := Build(input)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("Parse error: %v", err)
		}
		t.FailNow()
	}

	stmt := prog.Statements[0].(*ast.CreateWorkflowStmt)
	userTask := stmt.Activities[0].(*ast.WorkflowUserTaskNode)

	if len(userTask.BoundaryEvents) == 0 {
		t.Fatal("Expected at least 1 boundary event")
	}

	be := userTask.BoundaryEvents[0]
	if be.EventType != "NonInterruptingTimer" {
		t.Errorf("Expected EventType 'NonInterruptingTimer', got %q", be.EventType)
	}
	if be.Delay != "${PT2H}" {
		t.Errorf("Expected Delay '${PT2H}', got %q", be.Delay)
	}
}

func TestWorkflowVisitor_BoundaryEventTimerBare(t *testing.T) {
	input := `CREATE WORKFLOW M.TestWF
BEGIN
  USER TASK act1 'Caption'
    OUTCOMES 'Done' { }
    BOUNDARY EVENT TIMER;
END WORKFLOW;`

	prog, errs := Build(input)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("Parse error: %v", err)
		}
		t.FailNow()
	}

	stmt := prog.Statements[0].(*ast.CreateWorkflowStmt)
	userTask := stmt.Activities[0].(*ast.WorkflowUserTaskNode)

	if len(userTask.BoundaryEvents) == 0 {
		t.Fatal("Expected at least 1 boundary event")
	}

	be := userTask.BoundaryEvents[0]
	if be.EventType != "Timer" {
		t.Errorf("Expected EventType 'Timer', got %q", be.EventType)
	}
	if be.Delay != "" {
		t.Errorf("Expected empty Delay, got %q", be.Delay)
	}
}

func TestWorkflowVisitor_MultiUserTask(t *testing.T) {
	input := `CREATE WORKFLOW M.TestWF
BEGIN
  MULTI USER TASK act1 'Caption'
    PAGE M.ReviewPage
    OUTCOMES 'Approve' { };
END WORKFLOW;`

	prog, errs := Build(input)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("Parse error: %v", err)
		}
		t.FailNow()
	}

	stmt := prog.Statements[0].(*ast.CreateWorkflowStmt)

	if len(stmt.Activities) == 0 {
		t.Fatal("Expected at least 1 activity")
	}

	userTask, ok := stmt.Activities[0].(*ast.WorkflowUserTaskNode)
	if !ok {
		t.Fatalf("Expected WorkflowUserTaskNode, got %T", stmt.Activities[0])
	}

	if !userTask.IsMultiUser {
		t.Error("Expected IsMultiUser to be true")
	}
	if userTask.Page.Module != "M" || userTask.Page.Name != "ReviewPage" {
		t.Errorf("Expected Page M.ReviewPage, got %s.%s", userTask.Page.Module, userTask.Page.Name)
	}
}

func TestWorkflowVisitor_ParameterMappingWith(t *testing.T) {
	input := `CREATE WORKFLOW M.TestWF
BEGIN
  CALL MICROFLOW M.CalcDiscount
    WITH (Amount = '$WorkflowContext/Amount')
    OUTCOMES
      TRUE -> { }
      FALSE -> { };
END WORKFLOW;`

	prog, errs := Build(input)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("Parse error: %v", err)
		}
		t.FailNow()
	}

	stmt := prog.Statements[0].(*ast.CreateWorkflowStmt)

	if len(stmt.Activities) == 0 {
		t.Fatal("Expected at least 1 activity")
	}

	callMf, ok := stmt.Activities[0].(*ast.WorkflowCallMicroflowNode)
	if !ok {
		t.Fatalf("Expected WorkflowCallMicroflowNode, got %T", stmt.Activities[0])
	}

	if len(callMf.ParameterMappings) == 0 {
		t.Fatal("Expected at least 1 parameter mapping")
	}

	pm := callMf.ParameterMappings[0]
	if pm.Parameter != "Amount" {
		t.Errorf("Expected Parameter 'Amount', got %q", pm.Parameter)
	}
	if pm.Expression != "$WorkflowContext/Amount" {
		t.Errorf("Expected Expression '$WorkflowContext/Amount', got %q", pm.Expression)
	}
}

func TestWorkflowVisitor_UserTaskDueDate(t *testing.T) {
	input := `CREATE WORKFLOW M.TestWF
BEGIN
  USER TASK task1 'My Task'
    ENTITY M.TaskContext
    DUE DATE 'PT24H'
    OUTCOMES 'Done' { };
END WORKFLOW;`

	prog, errs := Build(input)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("Parse error: %v", err)
		}
		t.FailNow()
	}

	stmt := prog.Statements[0].(*ast.CreateWorkflowStmt)
	userTask, ok := stmt.Activities[0].(*ast.WorkflowUserTaskNode)
	if !ok {
		t.Fatalf("Expected WorkflowUserTaskNode, got %T", stmt.Activities[0])
	}

	if userTask.DueDate != "PT24H" {
		t.Errorf("Expected DueDate 'PT24H', got %q", userTask.DueDate)
	}
}

func TestWorkflowVisitor_UserTaskDueDateWithXPath(t *testing.T) {
	input := `CREATE WORKFLOW M.TestWF
BEGIN
  USER TASK task1 'My Task'
    TARGETING XPATH '[Assignee = $currentUser]'
    ENTITY M.TaskContext
    DUE DATE 'PT48H'
    OUTCOMES 'Done' { };
END WORKFLOW;`

	prog, errs := Build(input)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("Parse error: %v", err)
		}
		t.FailNow()
	}

	stmt := prog.Statements[0].(*ast.CreateWorkflowStmt)
	userTask, ok := stmt.Activities[0].(*ast.WorkflowUserTaskNode)
	if !ok {
		t.Fatalf("Expected WorkflowUserTaskNode, got %T", stmt.Activities[0])
	}

	if userTask.Targeting.Kind != "xpath" {
		t.Errorf("Expected Targeting.Kind 'xpath', got %q", userTask.Targeting.Kind)
	}
	if userTask.DueDate != "PT48H" {
		t.Errorf("Expected DueDate 'PT48H', got %q", userTask.DueDate)
	}
}

func TestWorkflowVisitor_Annotation(t *testing.T) {
	input := `CREATE WORKFLOW M.TestWF
BEGIN
  ANNOTATION 'This is a workflow note';
END WORKFLOW;`

	prog, errs := Build(input)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("Parse error: %v", err)
		}
		t.FailNow()
	}

	stmt := prog.Statements[0].(*ast.CreateWorkflowStmt)

	if len(stmt.Activities) == 0 {
		t.Fatal("Expected at least 1 activity")
	}

	ann, ok := stmt.Activities[0].(*ast.WorkflowAnnotationActivityNode)
	if !ok {
		t.Fatalf("Expected WorkflowAnnotationActivityNode, got %T", stmt.Activities[0])
	}

	if ann.Text != "This is a workflow note" {
		t.Errorf("Expected Text 'This is a workflow note', got %q", ann.Text)
	}
}

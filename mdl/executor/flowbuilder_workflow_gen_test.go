// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.e — workflow-block adder tests.
//
// Verifies wrapping into ActionActivity, the WorkflowOperation
// dispatch, and the gen-specific quirks (StringTemplate Reason,
// WorkflowDefinitionObjectSelection for Lock/Unlock).

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
	"github.com/mendixlabs/mxcli/modelsdk/version"
)

func newTestFlowBuilderGen() *flowBuilderGen {
	return &flowBuilderGen{
		posX: 200, posY: 200, baseY: 200, spacing: HorizontalSpacing,
	}
}

func newTestFlowBuilderGenWithVersion(v string) *flowBuilderGen {
	fb := newTestFlowBuilderGen()
	fb.version = version.Parse(v)
	return fb
}

func TestWrapActionGenAdvancesPosAndAssignsActivity(t *testing.T) {
	fb := newTestFlowBuilderGen()
	startX := fb.posX
	op := genMf.NewWorkflowCallAction()
	assignFreshID(op)

	id := fb.wrapActionGen(op, nil)

	if id == "" {
		t.Fatal("wrapActionGen should return a non-empty ID")
	}
	if fb.posX != startX+HorizontalSpacing {
		t.Fatalf("posX = %d, want %d", fb.posX, startX+HorizontalSpacing)
	}
	if len(fb.objects) != 1 {
		t.Fatalf("objects = %d, want 1", len(fb.objects))
	}
	act, ok := fb.objects[0].(*genMf.ActionActivity)
	if !ok {
		t.Fatalf("want *ActionActivity, got %T", fb.objects[0])
	}
	if act.Action() == nil {
		t.Fatal("inner action must be set")
	}
	if !act.AutoGenerateCaption() {
		t.Fatal("AutoGenerateCaption should default to true")
	}
}

func TestWrapActionGenRegistersEmptyCustomHandler(t *testing.T) {
	fb := newTestFlowBuilderGen()
	op := genMf.NewWorkflowCallAction()
	assignFreshID(op)

	id := fb.wrapActionGen(op, &ast.ErrorHandlingClause{Type: ast.ErrorHandlingCustom})

	if fb.emptyErrorHandlerFrom != id {
		t.Fatalf("emptyErrorHandlerFrom = %s, want %s", fb.emptyErrorHandlerFrom, id)
	}
}

func TestAddCallWorkflowActionGenSetsAllFields(t *testing.T) {
	fb := newTestFlowBuilderGen()
	stmt := &ast.CallWorkflowStmt{
		Workflow: ast.QualifiedName{Module: "Sales", Name: "ApprovalWF"},
		Arguments: []ast.CallArgument{
			{Value: &ast.VariableExpr{Name: "Order"}},
		},
		OutputVariable: "WfResult",
	}
	id := fb.addCallWorkflowActionGen(stmt)

	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	act := fb.objects[0].(*genMf.ActionActivity).Action().(*genMf.WorkflowCallAction)
	if act.WorkflowQualifiedName() != "Sales.ApprovalWF" {
		t.Fatalf("workflow QN = %q", act.WorkflowQualifiedName())
	}
	if act.WorkflowContextVariable() != "Order" {
		t.Fatalf("context var = %q, want Order (no $ prefix)", act.WorkflowContextVariable())
	}
	if act.OutputVariableName() != "WfResult" {
		t.Fatalf("output var = %q", act.OutputVariableName())
	}
	if !act.UseReturnVariable() {
		t.Fatal("UseReturnVariable must be true when output var set")
	}
	if act.ErrorHandlingType() != "Rollback" {
		t.Fatalf("default eh = %q, want Rollback", act.ErrorHandlingType())
	}
}

func TestAddWorkflowOperationActionGenAbortAttachesStringTemplateReason(t *testing.T) {
	fb := newTestFlowBuilderGen()
	stmt := &ast.WorkflowOperationStmt{
		OperationType:    "abort",
		WorkflowVariable: "WF",
		Reason: &ast.LiteralExpr{
			Kind:  ast.LiteralString,
			Value: "manual stop",
		},
	}
	id := fb.addWorkflowOperationActionGen(stmt)
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	act := fb.objects[0].(*genMf.ActionActivity).Action().(*genMf.WorkflowOperationAction)
	abort, ok := act.Operation().(*genMf.AbortOperation)
	if !ok {
		t.Fatalf("operation = %T, want *AbortOperation", act.Operation())
	}
	if abort.WorkflowVariable() != "WF" {
		t.Fatalf("workflow var = %q", abort.WorkflowVariable())
	}
	tmpl, ok := abort.Reason().(*genMf.StringTemplate)
	if !ok || tmpl == nil {
		t.Fatalf("Reason = %T, want *StringTemplate", abort.Reason())
	}
	// expressionToString quotes string literals; doubled-quote escape rule
	// for the inner text isn't triggered here so the result is `'manual stop'`.
	if tmpl.Text() != "'manual stop'" {
		t.Fatalf("reason text = %q, want %q", tmpl.Text(), "'manual stop'")
	}
}

func TestAddWorkflowOperationActionGenContinueProducesContinueOperation(t *testing.T) {
	fb := newTestFlowBuilderGen()
	stmt := &ast.WorkflowOperationStmt{OperationType: "continue", WorkflowVariable: "WF"}
	fb.addWorkflowOperationActionGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity).Action().(*genMf.WorkflowOperationAction)
	if _, ok := act.Operation().(*genMf.ContinueOperation); !ok {
		t.Fatalf("operation = %T, want *ContinueOperation", act.Operation())
	}
}

func TestAddWorkflowOperationActionGenRetryProducesRetryOperation(t *testing.T) {
	fb := newTestFlowBuilderGen()
	stmt := &ast.WorkflowOperationStmt{OperationType: "retry", WorkflowVariable: "WF"}
	fb.addWorkflowOperationActionGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity).Action().(*genMf.WorkflowOperationAction)
	if _, ok := act.Operation().(*genMf.RetryOperation); !ok {
		t.Fatalf("operation = %T, want *RetryOperation", act.Operation())
	}
}

func TestAddWorkflowOperationActionGenUnknownLeavesOperationNil(t *testing.T) {
	fb := newTestFlowBuilderGen()
	stmt := &ast.WorkflowOperationStmt{OperationType: "bogus", WorkflowVariable: "WF"}
	fb.addWorkflowOperationActionGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity).Action().(*genMf.WorkflowOperationAction)
	if act.Operation() != nil {
		t.Fatalf("unknown operation should leave Operation nil, got %T", act.Operation())
	}
}

func TestAddSetTaskOutcomeActionGenSetsFields(t *testing.T) {
	fb := newTestFlowBuilderGen()
	stmt := &ast.SetTaskOutcomeStmt{
		WorkflowTaskVariable: "Task",
		OutcomeValue:         "Approved",
	}
	fb.addSetTaskOutcomeActionGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity).Action().(*genMf.SetTaskOutcomeAction)
	if act.WorkflowTaskVariable() != "Task" {
		t.Fatalf("task var = %q", act.WorkflowTaskVariable())
	}
	if act.OutcomeValue() != "Approved" {
		t.Fatalf("outcome = %q", act.OutcomeValue())
	}
}

func TestAddLockWorkflowActionGenAllSetsPauseAllOnly(t *testing.T) {
	fb := newTestFlowBuilderGen()
	stmt := &ast.LockWorkflowStmt{PauseAllWorkflows: true}
	fb.addLockWorkflowActionGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity).Action().(*genMf.LockWorkflowAction)
	if !act.PauseAllWorkflows() {
		t.Fatal("PauseAllWorkflows should be true")
	}
	if act.WorkflowSelection() != nil {
		t.Fatalf("WorkflowSelection should be nil for `lock all`, got %T", act.WorkflowSelection())
	}
}

func TestAddLockWorkflowActionGenVariableWrapsObjectSelection(t *testing.T) {
	fb := newTestFlowBuilderGen()
	stmt := &ast.LockWorkflowStmt{WorkflowVariable: "Wf"}
	fb.addLockWorkflowActionGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity).Action().(*genMf.LockWorkflowAction)
	sel, ok := act.WorkflowSelection().(*genWf.WorkflowDefinitionObjectSelection)
	if !ok {
		t.Fatalf("WorkflowSelection = %T, want *WorkflowDefinitionObjectSelection", act.WorkflowSelection())
	}
	if sel.WorkflowDefinitionVariable() != "Wf" {
		t.Fatalf("variable = %q, want Wf", sel.WorkflowDefinitionVariable())
	}
}

func TestAddUnlockWorkflowActionGenResumeAllOnly(t *testing.T) {
	fb := newTestFlowBuilderGen()
	stmt := &ast.UnlockWorkflowStmt{ResumeAllPausedWorkflows: true}
	fb.addUnlockWorkflowActionGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity).Action().(*genMf.UnlockWorkflowAction)
	if !act.ResumeAllPausedWorkflows() {
		t.Fatal("ResumeAllPausedWorkflows should be true")
	}
}

func TestAddOpenUserTaskActionGenSetsFields(t *testing.T) {
	fb := newTestFlowBuilderGen()
	stmt := &ast.OpenUserTaskStmt{UserTaskVariable: "Task"}
	fb.addOpenUserTaskActionGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity).Action().(*genMf.OpenUserTaskAction)
	if act.UserTaskVariable() != "Task" {
		t.Fatalf("task var = %q", act.UserTaskVariable())
	}
}

func TestAddNotifyWorkflowActionGenSetsFields(t *testing.T) {
	fb := newTestFlowBuilderGen()
	stmt := &ast.NotifyWorkflowStmt{
		WorkflowVariable: "WF",
		OutputVariable:   "Result",
	}
	fb.addNotifyWorkflowActionGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity).Action().(*genMf.NotifyWorkflowAction)
	if act.WorkflowVariable() != "WF" {
		t.Fatalf("wf var = %q", act.WorkflowVariable())
	}
	if act.OutputVariableName() != "Result" {
		t.Fatalf("output = %q", act.OutputVariableName())
	}
}

func TestAddNotifyWorkflowActionGenSetsActivityQN(t *testing.T) {
	fb := newTestFlowBuilderGen()
	stmt := &ast.NotifyWorkflowStmt{
		WorkflowVariable:      "WF",
		OutputVariable:        "IsReceived",
		ActivityQualifiedName: "HD.WF_TicketEscalation.WaitForManagerAvailable",
	}
	fb.addNotifyWorkflowActionGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity).Action().(*genMf.NotifyWorkflowAction)
	if act.ActivityQualifiedName() != "HD.WF_TicketEscalation.WaitForManagerAvailable" {
		t.Fatalf("activity QN = %q", act.ActivityQualifiedName())
	}
}

// TestAddNotifyWorkflowActionGenUsesNotifyTargetOnV11_7 verifies that on
// Mendix 11.7.0+ the executor sets the notifyTarget Part instead of the
// deprecated activity ByNameRef (which is version-gated out in 11.7+).
func TestAddNotifyWorkflowActionGenUsesNotifyTargetOnV11_7(t *testing.T) {
	fb := newTestFlowBuilderGenWithVersion("11.7.0")
	stmt := &ast.NotifyWorkflowStmt{
		WorkflowVariable:      "WF",
		OutputVariable:        "IsReceived",
		ActivityQualifiedName: "HD.WF_TicketEscalation.WaitForManagerAvailable",
	}
	fb.addNotifyWorkflowActionGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity).Action().(*genMf.NotifyWorkflowAction)

	target := act.NotifyTarget()
	if target == nil {
		t.Fatal("notifyTarget must be set for Mendix 11.7.0+")
	}
	nt, ok := target.(*genWf.NotifyWaitForNotificationActivityTarget)
	if !ok {
		t.Fatalf("notifyTarget type = %T, want *NotifyWaitForNotificationActivityTarget", target)
	}
	if nt.ActivityQualifiedName() != "HD.WF_TicketEscalation.WaitForManagerAvailable" {
		t.Fatalf("notifyTarget activity QN = %q", nt.ActivityQualifiedName())
	}
}

func TestAddOpenWorkflowActionGenSetsField(t *testing.T) {
	fb := newTestFlowBuilderGen()
	stmt := &ast.OpenWorkflowStmt{WorkflowVariable: "WF"}
	fb.addOpenWorkflowActionGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity).Action().(*genMf.OpenWorkflowAction)
	if act.WorkflowVariable() != "WF" {
		t.Fatalf("wf var = %q", act.WorkflowVariable())
	}
}

func TestAddGetWorkflowDataActionGenSetsAllFields(t *testing.T) {
	fb := newTestFlowBuilderGen()
	stmt := &ast.GetWorkflowDataStmt{
		Workflow:         ast.QualifiedName{Module: "M", Name: "WF"},
		WorkflowVariable: "WfVar",
		OutputVariable:   "Data",
	}
	fb.addGetWorkflowDataActionGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity).Action().(*genMf.GetWorkflowDataAction)
	if act.WorkflowQualifiedName() != "M.WF" {
		t.Fatalf("wf qn = %q", act.WorkflowQualifiedName())
	}
	if act.WorkflowVariable() != "WfVar" {
		t.Fatalf("wf var = %q", act.WorkflowVariable())
	}
	if act.OutputVariableName() != "Data" {
		t.Fatalf("output = %q", act.OutputVariableName())
	}
}

func TestAddGetWorkflowsActionGenSetsContextVarName(t *testing.T) {
	fb := newTestFlowBuilderGen()
	stmt := &ast.GetWorkflowsStmt{
		WorkflowContextVariableName: "CtxVar",
		OutputVariable:              "List",
	}
	fb.addGetWorkflowsActionGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity).Action().(*genMf.GetWorkflowsAction)
	if act.WorkflowContextVariableName() != "CtxVar" {
		t.Fatalf("ctx var = %q", act.WorkflowContextVariableName())
	}
	if act.OutputVariableName() != "List" {
		t.Fatalf("output = %q", act.OutputVariableName())
	}
}

func TestAddGetWorkflowActivityRecordsActionGenSetsFields(t *testing.T) {
	fb := newTestFlowBuilderGen()
	stmt := &ast.GetWorkflowActivityRecordsStmt{
		WorkflowVariable: "WF",
		OutputVariable:   "Records",
	}
	fb.addGetWorkflowActivityRecordsActionGen(stmt)
	act := fb.objects[0].(*genMf.ActionActivity).Action().(*genMf.GetWorkflowActivityRecordsAction)
	if act.WorkflowVariable() != "WF" {
		t.Fatalf("wf var = %q", act.WorkflowVariable())
	}
	if act.OutputVariableName() != "Records" {
		t.Fatalf("output = %q", act.OutputVariableName())
	}
}

// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.e — gen-typed workflow-block adders.
//
// This file is the gen-typed counterpart of
// `cmd_microflows_builder_workflow.go`. It owns the per-statement
// adders for workflow-related microflow activities:
//
//   - addCallWorkflowActionGen — `call workflow Mod.WF(ctx)`
//   - addGetWorkflowDataActionGen — `$x = get workflow data ...`
//   - addGetWorkflowsActionGen — `$x = get workflows ...`
//   - addGetWorkflowActivityRecordsActionGen — `$x = get workflow activity records ...`
//   - addWorkflowOperationActionGen — abort/continue/pause/restart/retry/unpause
//   - addSetTaskOutcomeActionGen — `set task outcome ...`
//   - addOpenUserTaskActionGen / addNotifyWorkflowActionGen /
//     addOpenWorkflowActionGen / addLockWorkflowActionGen /
//     addUnlockWorkflowActionGen
//
// Plus `wrapActionGen`, the shared builder helper that wraps an
// arbitrary gen action element into an *ActionActivity at the current
// cursor and queues any custom error handler for processing.
//
// Custom error-handler body emission depends on `addStatement`
// (commit (h)/i) and `addErrorHandlerFlow` (commit (h)). Until those
// land, wrapActionGen routes the EMPTY-clause case through the
// already-implemented register helper but defers non-empty bodies to
// a TODO comment + no-op so the file compiles and tests can exercise
// the basic shape. Stage 3.2.3.h backfills the body emission.

package executor

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// wrapActionGen wraps a gen action element in an ActionActivity at the
// current builder cursor, advances posX by spacing, and registers any
// empty `on error custom` handler against the new activity. Returns
// the activity ID.
//
// Non-empty error-handler bodies are intentionally deferred to commit
// (h) — the body emission needs addStatement, which doesn't land
// until the actions/calls/control commits.
func (fb *flowBuilderGen) wrapActionGen(action element.Element, errorHandling *ast.ErrorHandlingClause) element.ID {
	activity := genMf.NewActionActivity()
	id := assignFreshID(activity)
	activity.SetRelativeMiddlePoint(layoutPos(fb.posX, fb.posY))
	activity.SetSize(layoutSize(ActivityWidth, ActivityHeight))
	activity.SetAutoGenerateCaption(true)
	activity.SetAction(action)

	fb.objects = append(fb.objects, activity)
	fb.posX += fb.spacing

	// The empty-handler register helper is safe at any stage.
	// Non-empty bodies need addStatement — left as a TODO until
	// commit (h) wires it in.
	if isEmptyCustomErrorHandlerGen(errorHandling) {
		fb.registerEmptyCustomErrorHandlerWithSkipGen(id, errorHandling, "")
	}
	// TODO Stage 3.2.3.h: emit non-empty error-handler bodies here
	// via finishCustomErrorHandlerGen → addErrorHandlerFlowGen.

	return id
}

// addCallWorkflowActionGen emits a `call workflow` activity.
func (fb *flowBuilderGen) addCallWorkflowActionGen(s *ast.CallWorkflowStmt) element.ID {
	wfQN := s.Workflow.Module + "." + s.Workflow.Name

	ctxVar := ""
	if len(s.Arguments) > 0 {
		ctxVar = fb.exprToString(s.Arguments[0].Value)
		ctxVar = strings.TrimPrefix(ctxVar, "$")
	}

	action := genMf.NewWorkflowCallAction()
	assignFreshID(action)
	action.SetErrorHandlingType(convertErrorHandlingTypeGen(s.ErrorHandling))
	action.SetWorkflowQualifiedName(wfQN)
	action.SetWorkflowContextVariable(ctxVar)
	action.SetOutputVariableName(s.OutputVariable)
	action.SetUseReturnVariable(s.OutputVariable != "")

	return fb.wrapActionGen(action, s.ErrorHandling)
}

// addGetWorkflowDataActionGen emits `$x = get workflow data ...`.
func (fb *flowBuilderGen) addGetWorkflowDataActionGen(s *ast.GetWorkflowDataStmt) element.ID {
	action := genMf.NewGetWorkflowDataAction()
	assignFreshID(action)
	action.SetErrorHandlingType(convertErrorHandlingTypeGen(s.ErrorHandling))
	action.SetOutputVariableName(s.OutputVariable)
	action.SetWorkflowQualifiedName(s.Workflow.Module + "." + s.Workflow.Name)
	action.SetWorkflowVariable(s.WorkflowVariable)
	return fb.wrapActionGen(action, s.ErrorHandling)
}

// addGetWorkflowsActionGen emits `$x = get workflows ...`.
func (fb *flowBuilderGen) addGetWorkflowsActionGen(s *ast.GetWorkflowsStmt) element.ID {
	action := genMf.NewGetWorkflowsAction()
	assignFreshID(action)
	action.SetErrorHandlingType(convertErrorHandlingTypeGen(s.ErrorHandling))
	action.SetOutputVariableName(s.OutputVariable)
	action.SetWorkflowContextVariableName(s.WorkflowContextVariableName)
	return fb.wrapActionGen(action, s.ErrorHandling)
}

// addGetWorkflowActivityRecordsActionGen emits
// `$x = get workflow activity records ...`.
func (fb *flowBuilderGen) addGetWorkflowActivityRecordsActionGen(s *ast.GetWorkflowActivityRecordsStmt) element.ID {
	action := genMf.NewGetWorkflowActivityRecordsAction()
	assignFreshID(action)
	action.SetErrorHandlingType(convertErrorHandlingTypeGen(s.ErrorHandling))
	action.SetOutputVariableName(s.OutputVariable)
	action.SetWorkflowVariable(s.WorkflowVariable)
	return fb.wrapActionGen(action, s.ErrorHandling)
}

// addWorkflowOperationActionGen emits a workflow operation
// (abort/continue/pause/restart/retry/unpause). Unknown operation
// strings produce an action with a nil Operation — the caller's
// validate pass should have rejected them earlier.
func (fb *flowBuilderGen) addWorkflowOperationActionGen(s *ast.WorkflowOperationStmt) element.ID {
	var op element.Element
	switch s.OperationType {
	case "abort":
		abort := genMf.NewAbortOperation()
		assignFreshID(abort)
		if s.Reason != nil {
			tmpl := genMf.NewStringTemplate()
			assignFreshID(tmpl)
			tmpl.SetText(fb.exprToString(s.Reason))
			abort.SetReason(tmpl)
		}
		abort.SetWorkflowVariable(s.WorkflowVariable)
		op = abort
	case "continue":
		cont := genMf.NewContinueOperation()
		assignFreshID(cont)
		cont.SetWorkflowVariable(s.WorkflowVariable)
		op = cont
	case "pause":
		pause := genMf.NewPauseOperation()
		assignFreshID(pause)
		pause.SetWorkflowVariable(s.WorkflowVariable)
		op = pause
	case "restart":
		restart := genMf.NewRestartOperation()
		assignFreshID(restart)
		restart.SetWorkflowVariable(s.WorkflowVariable)
		op = restart
	case "retry":
		retry := genMf.NewRetryOperation()
		assignFreshID(retry)
		retry.SetWorkflowVariable(s.WorkflowVariable)
		op = retry
	case "unpause":
		unpause := genMf.NewUnpauseOperation()
		assignFreshID(unpause)
		unpause.SetWorkflowVariable(s.WorkflowVariable)
		op = unpause
	}

	action := genMf.NewWorkflowOperationAction()
	assignFreshID(action)
	action.SetErrorHandlingType(convertErrorHandlingTypeGen(s.ErrorHandling))
	if op != nil {
		action.SetOperation(op)
	}
	return fb.wrapActionGen(action, s.ErrorHandling)
}

// addSetTaskOutcomeActionGen emits `set task outcome ...`.
func (fb *flowBuilderGen) addSetTaskOutcomeActionGen(s *ast.SetTaskOutcomeStmt) element.ID {
	action := genMf.NewSetTaskOutcomeAction()
	assignFreshID(action)
	action.SetErrorHandlingType(convertErrorHandlingTypeGen(s.ErrorHandling))
	action.SetOutcomeValue(s.OutcomeValue)
	action.SetWorkflowTaskVariable(s.WorkflowTaskVariable)
	return fb.wrapActionGen(action, s.ErrorHandling)
}

// addOpenUserTaskActionGen emits `open user task ...`.
func (fb *flowBuilderGen) addOpenUserTaskActionGen(s *ast.OpenUserTaskStmt) element.ID {
	action := genMf.NewOpenUserTaskAction()
	assignFreshID(action)
	action.SetErrorHandlingType(convertErrorHandlingTypeGen(s.ErrorHandling))
	action.SetUserTaskVariable(s.UserTaskVariable)
	return fb.wrapActionGen(action, s.ErrorHandling)
}

// addNotifyWorkflowActionGen emits `notify workflow ...`.
func (fb *flowBuilderGen) addNotifyWorkflowActionGen(s *ast.NotifyWorkflowStmt) element.ID {
	action := genMf.NewNotifyWorkflowAction()
	assignFreshID(action)
	action.SetErrorHandlingType(convertErrorHandlingTypeGen(s.ErrorHandling))
	action.SetOutputVariableName(s.OutputVariable)
	action.SetWorkflowVariable(s.WorkflowVariable)
	if s.ActivityQualifiedName != "" {
		action.SetActivityQualifiedName(s.ActivityQualifiedName)
	}
	return fb.wrapActionGen(action, s.ErrorHandling)
}

// addOpenWorkflowActionGen emits `open workflow ...`.
func (fb *flowBuilderGen) addOpenWorkflowActionGen(s *ast.OpenWorkflowStmt) element.ID {
	action := genMf.NewOpenWorkflowAction()
	assignFreshID(action)
	action.SetErrorHandlingType(convertErrorHandlingTypeGen(s.ErrorHandling))
	action.SetWorkflowVariable(s.WorkflowVariable)
	return fb.wrapActionGen(action, s.ErrorHandling)
}

// addLockWorkflowActionGen emits `lock workflow ... [pause all]`. The
// gen LockWorkflowAction stores its target via a `WorkflowSelection`
// element rather than the legacy SDK's flat `WorkflowVariable` field —
// for variable-form locks we wrap the variable in a
// WorkflowDefinitionObjectSelection (matches what the gen describer
// reads back via workflowSelectionGen in
// cmd_microflows_format_workflow_gen.go).
func (fb *flowBuilderGen) addLockWorkflowActionGen(s *ast.LockWorkflowStmt) element.ID {
	action := genMf.NewLockWorkflowAction()
	assignFreshID(action)
	action.SetErrorHandlingType(convertErrorHandlingTypeGen(s.ErrorHandling))
	action.SetPauseAllWorkflows(s.PauseAllWorkflows)
	if s.WorkflowVariable != "" {
		sel := genWf.NewWorkflowDefinitionObjectSelection()
		assignFreshID(sel)
		sel.SetWorkflowDefinitionVariable(s.WorkflowVariable)
		action.SetWorkflowSelection(sel)
	}
	return fb.wrapActionGen(action, s.ErrorHandling)
}

// addUnlockWorkflowActionGen emits `unlock workflow ... [resume all paused]`.
// Same WorkflowSelection wrapping as addLockWorkflowActionGen.
func (fb *flowBuilderGen) addUnlockWorkflowActionGen(s *ast.UnlockWorkflowStmt) element.ID {
	action := genMf.NewUnlockWorkflowAction()
	assignFreshID(action)
	action.SetErrorHandlingType(convertErrorHandlingTypeGen(s.ErrorHandling))
	action.SetResumeAllPausedWorkflows(s.ResumeAllPausedWorkflows)
	if s.WorkflowVariable != "" {
		sel := genWf.NewWorkflowDefinitionObjectSelection()
		assignFreshID(sel)
		sel.SetWorkflowDefinitionVariable(s.WorkflowVariable)
		action.SetWorkflowSelection(sel)
	}
	return fb.wrapActionGen(action, s.ErrorHandling)
}

// addGenerateJumpToActionGen emits
// `[$Options =] generate jump to options for $wf as Module.WF_Name`.
func (fb *flowBuilderGen) addGenerateJumpToActionGen(s *ast.GenerateJumpToStmt) element.ID {
	action := genMf.NewGenerateJumpToOptionsAction()
	assignFreshID(action)
	action.SetErrorHandlingType(convertErrorHandlingTypeGen(s.ErrorHandling))
	action.SetWorkflowVariable(s.WorkflowVariable)
	if s.WorkflowQN.Module != "" && s.WorkflowQN.Name != "" {
		action.SetWorkflowQualifiedName(s.WorkflowQN.Module + "." + s.WorkflowQN.Name)
	}
	action.SetOutputVariableName(s.OutputVariable)
	return fb.wrapActionGen(action, s.ErrorHandling)
}

// addApplyJumpToActionGen emits
// `[$Result =] apply jump to option $options`.
func (fb *flowBuilderGen) addApplyJumpToActionGen(s *ast.ApplyJumpToStmt) element.ID {
	action := genMf.NewApplyJumpToOptionAction()
	assignFreshID(action)
	action.SetErrorHandlingType(convertErrorHandlingTypeGen(s.ErrorHandling))
	action.SetWorkflowJumpToDetailsVariable(s.JumpOptionsVariable)
	action.SetOutputVariableName(s.OutputVariable)
	return fb.wrapActionGen(action, s.ErrorHandling)
}

// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.h1 — addStatementGen dispatcher + Break/Continue events.
//
// Mirror of the legacy `addStatement` type-switch (cmd_microflows_
// builder_graph.go:509) over the gen-typed adders. Every action /
// page / call / workflow / data adder shipped in a-g is wired here.
//
// Pre-dispatch hooks (mirror legacy):
//
//   1. mergeStatementAnnotations — pulls @position / @caption /
//      @color / @anchor / etc onto fb.pendingAnnotations.
//   2. @position application — moves fb.posX/Y so the activity lands
//      where the annotation requested.
//   3. @annotation free-text attach — emits free Annotation objects
//      (no flow attachment) before the activity is created.
//
// Type-switch routing — 41 already-implemented adders + 5 deferred
// to h2/h3/h4:
//
//   Already wired (a-g):
//     DeclareStmt, MfSetStmt (path → ChangeObject; bare → ChangeVariable),
//     ReturnStmt, RaiseErrorStmt, BreakStmt, ContinueStmt,
//     CastObjectStmt, CreateObjectStmt, ChangeObjectStmt,
//     RetrieveStmt, MfCommitStmt, DeleteObjectStmt, RollbackStmt,
//     LogStmt, DownloadFileStmt, ValidationFeedbackStmt,
//     ListOperationStmt, AggregateListStmt, CreateListStmt,
//     AddToListStmt, RemoveFromListStmt,
//     CallMicroflowStmt, CallNanoflowStmt, CallJavaActionStmt,
//     CallJavaScriptActionStmt, CallWebServiceStmt, CallExternalActionStmt,
//     ExecuteDatabaseQueryStmt, ImportFromMappingStmt,
//     ExportToMappingStmt, TransformJsonStmt,
//     RestCallStmt, SendRestRequestStmt,
//     ShowPageStmt, ClosePageStmt, ShowHomePageStmt, ShowMessageStmt,
//     CallWorkflowStmt, GetWorkflowDataStmt, GetWorkflowsStmt,
//     GetWorkflowActivityRecordsStmt, WorkflowOperationStmt,
//     SetTaskOutcomeStmt, OpenUserTaskStmt, NotifyWorkflowStmt,
//     OpenWorkflowStmt, LockWorkflowStmt, UnlockWorkflowStmt
//
//   Returns "" until h2/h3/h4 land:
//     IfStmt, LoopStmt, WhileStmt, EnumSplitStmt, InheritanceSplitStmt
//
// The deferred cases use the same legacy fallback (return ""); they
// don't panic so the dispatcher stays buildable while the control
// adders are being filled in. Each commit (h2/h3/h4) replaces the
// empty-return placeholders with the real implementation.

package executor

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// addStatementGen is the dispatcher: maps any ast.MicroflowStatement
// to its gen-typed add*Gen counterpart and emits the corresponding
// activity / event / split into fb.objects.
//
// Returns the new element's ID, or "" when the statement type is
// not recognised or its adder hasn't landed yet (h2-h4).
func (fb *flowBuilderGen) addStatementGen(stmt ast.MicroflowStatement) element.ID {
	// Pre-dispatch: pull annotations onto pendingAnnotations.
	fb.mergeStatementAnnotations(stmt)

	// @position applies BEFORE the activity is created so the
	// activity lands at the requested coordinates.
	if fb.pendingAnnotations != nil && fb.pendingAnnotations.Position != nil {
		fb.posX = fb.pendingAnnotations.Position.X
		fb.posY = fb.pendingAnnotations.Position.Y
	}

	// Free @annotation entries become standalone Annotation objects
	// (no flow attachment). Drained here so they don't accidentally
	// re-emit on the next dispatch.
	if fb.pendingAnnotations != nil {
		for _, text := range fb.pendingAnnotations.FreeAnnotations {
			fb.attachFreeAnnotation(text)
		}
		fb.pendingAnnotations.FreeAnnotations = nil
	}

	switch s := stmt.(type) {
	case *ast.DeclareStmt:
		return fb.addCreateVariableActionGen(s)

	// ─── Splits — h4 ───
	case *ast.EnumSplitStmt:
		return "" // TODO Stage 3.2.3.h4: addEnumSplitGen
	case *ast.InheritanceSplitStmt:
		return "" // TODO Stage 3.2.3.h4: addInheritanceSplitGen

	case *ast.CastObjectStmt:
		return fb.addCastActionGen(s)

	case *ast.MfSetStmt:
		// Path target ($Var/Module.Assoc) routes to ChangeObject;
		// bare target routes to ChangeVariable. Mirrors legacy
		// addStatement dispatch.
		if idx := strings.IndexByte(s.Target, '/'); idx >= 0 {
			varName := strings.TrimPrefix(s.Target[:idx], "$")
			assoc := s.Target[idx+1:]
			return fb.addChangeObjectActionGen(&ast.ChangeObjectStmt{
				Variable:    varName,
				Changes:     []ast.ChangeItem{{Attribute: assoc, Value: s.Value}},
				Annotations: s.Annotations,
			})
		}
		return fb.addChangeVariableActionGen(s)

	case *ast.ReturnStmt:
		return fb.addEndEventWithReturn(s)
	case *ast.RaiseErrorStmt:
		return fb.addErrorEvent()

	case *ast.BreakStmt:
		return fb.addBreakEventGen()
	case *ast.ContinueStmt:
		return fb.addContinueEventGen()

	case *ast.LogStmt:
		return fb.addLogMessageActionGen(s)
	case *ast.CreateObjectStmt:
		return fb.addCreateObjectActionGen(s)
	case *ast.ChangeObjectStmt:
		return fb.addChangeObjectActionGen(s)
	case *ast.RetrieveStmt:
		return fb.addRetrieveActionGen(s)
	case *ast.MfCommitStmt:
		return fb.addCommitActionGen(s)
	case *ast.DeleteObjectStmt:
		return fb.addDeleteActionGen(s)
	case *ast.RollbackStmt:
		return fb.addRollbackActionGen(s)

	// ─── Control flow — h2/h3 ───
	case *ast.IfStmt:
		return fb.addIfStatementGen(s)
	case *ast.LoopStmt:
		return "" // TODO Stage 3.2.3.h3: addLoopStatementGen
	case *ast.WhileStmt:
		return "" // TODO Stage 3.2.3.h3: addWhileStatementGen

	case *ast.ListOperationStmt:
		return fb.addListOperationActionGen(s)
	case *ast.AggregateListStmt:
		return fb.addAggregateListActionGen(s)
	case *ast.CreateListStmt:
		return fb.addCreateListActionGen(s)
	case *ast.AddToListStmt:
		return fb.addAddToListActionGen(s)
	case *ast.RemoveFromListStmt:
		return fb.addRemoveFromListActionGen(s)

	case *ast.CallMicroflowStmt:
		return fb.addCallMicroflowActionGen(s)
	case *ast.CallNanoflowStmt:
		return fb.addCallNanoflowActionGen(s)
	case *ast.CallJavaActionStmt:
		return fb.addCallJavaActionActionGen(s)
	case *ast.CallJavaScriptActionStmt:
		return fb.addCallJavaScriptActionActionGen(s)
	case *ast.CallWebServiceStmt:
		return fb.addCallWebServiceActionGen(s)
	case *ast.ExecuteDatabaseQueryStmt:
		return fb.addExecuteDatabaseQueryActionGen(s)
	case *ast.CallExternalActionStmt:
		return fb.addCallExternalActionGen(s)

	case *ast.ShowPageStmt:
		return fb.addShowPageActionGen(s)
	case *ast.ClosePageStmt:
		return fb.addClosePageActionGen(s)
	case *ast.ShowHomePageStmt:
		return fb.addShowHomePageActionGen(s)
	case *ast.ShowMessageStmt:
		return fb.addShowMessageActionGen(s)
	case *ast.DownloadFileStmt:
		return fb.addDownloadFileActionGen(s)
	case *ast.ValidationFeedbackStmt:
		return fb.addValidationFeedbackActionGen(s)

	case *ast.RestCallStmt:
		return fb.addRestCallActionGen(s)
	case *ast.SendRestRequestStmt:
		return fb.addSendRestRequestActionGen(s)
	case *ast.ImportFromMappingStmt:
		return fb.addImportFromMappingActionGen(s)
	case *ast.ExportToMappingStmt:
		return fb.addExportToMappingActionGen(s)
	case *ast.TransformJsonStmt:
		return fb.addTransformJsonActionGen(s)

	// ─── Workflow microflow actions ───
	case *ast.CallWorkflowStmt:
		return fb.addCallWorkflowActionGen(s)
	case *ast.GetWorkflowDataStmt:
		return fb.addGetWorkflowDataActionGen(s)
	case *ast.GetWorkflowsStmt:
		return fb.addGetWorkflowsActionGen(s)
	case *ast.GetWorkflowActivityRecordsStmt:
		return fb.addGetWorkflowActivityRecordsActionGen(s)
	case *ast.WorkflowOperationStmt:
		return fb.addWorkflowOperationActionGen(s)
	case *ast.SetTaskOutcomeStmt:
		return fb.addSetTaskOutcomeActionGen(s)
	case *ast.OpenUserTaskStmt:
		return fb.addOpenUserTaskActionGen(s)
	case *ast.NotifyWorkflowStmt:
		return fb.addNotifyWorkflowActionGen(s)
	case *ast.OpenWorkflowStmt:
		return fb.addOpenWorkflowActionGen(s)
	case *ast.LockWorkflowStmt:
		return fb.addLockWorkflowActionGen(s)
	case *ast.UnlockWorkflowStmt:
		return fb.addUnlockWorkflowActionGen(s)
	}

	// Unknown statement type — legacy returns "" silently.
	return ""
}

// addBreakEventGen emits a `break;` BreakEvent terminator. Mirrors
// flowBuilder.addBreakEvent — advances posX by spacing/2 to leave
// room for downstream geometry.
func (fb *flowBuilderGen) addBreakEventGen() element.ID {
	event := genMf.NewBreakEvent()
	id := assignFreshID(event)
	event.SetRelativeMiddlePoint(layoutPos(fb.posX, fb.posY))
	event.SetSize(layoutSize(EventSize, EventSize))
	fb.objects = append(fb.objects, event)
	fb.posX += fb.spacing / 2
	return id
}

// addContinueEventGen emits a `continue;` ContinueEvent terminator
// for ordinary loop bodies. When manualLoopBackTarget is set
// (manual while-true pattern), returns that target ID instead and
// emits no new event. Mirrors flowBuilder.addContinueEvent.
func (fb *flowBuilderGen) addContinueEventGen() element.ID {
	if fb.manualLoopBackTarget != "" {
		return fb.manualLoopBackTarget
	}
	event := genMf.NewContinueEvent()
	id := assignFreshID(event)
	event.SetRelativeMiddlePoint(layoutPos(fb.posX, fb.posY))
	event.SetSize(layoutSize(EventSize, EventSize))
	fb.objects = append(fb.objects, event)
	fb.posX += fb.spacing / 2
	return id
}

// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.2.g — Workflow / Misc family formatters (gen-typed).
//
// This file implements the gen-typed counterpart to legacy
// `cmd_microflows_format_action.go` for the eleven workflow-action
// kinds below plus the WorkflowOperationAction sub-operation dispatch.
//
//   | gen Go type                            | BSON $Type                              | MDL keyword                              |
//   |----------------------------------------|-----------------------------------------|------------------------------------------|
//   | *genMf.GetWorkflowDataAction           | Microflows$GetWorkflowDataAction        | `[$X =] get workflow data $W as Mod.Wf;` |
//   | *genMf.WorkflowCallAction              | Microflows$WorkflowCallAction           | `[$X =] call workflow Mod.Wf ($Ctx);`    |
//   | *genMf.GetWorkflowsAction              | Microflows$GetWorkflowsAction           | `[$X =] get workflows for $Ctx;`         |
//   | *genMf.GetWorkflowActivityRecordsAction| Microflows$GetWorkflowActivityRecordsAction | `[$X =] get workflow activity records $W;` |
//   | *genMf.WorkflowOperationAction         | Microflows$WorkflowOperationAction      | `workflow operation <op> $W [reason …];` |
//   | *genMf.SetTaskOutcomeAction            | Microflows$SetTaskOutcomeAction         | `set task outcome $T '<value>';`         |
//   | *genMf.OpenUserTaskAction              | Microflows$OpenUserTaskAction           | `open user task $T;`                     |
//   | *genMf.NotifyWorkflowAction            | Microflows$NotifyWorkflowAction         | `[$X =] notify workflow $W;`             |
//   | *genMf.OpenWorkflowAction              | Microflows$OpenWorkflowAction           | `open workflow $W;`                      |
//   | *genMf.LockWorkflowAction              | Microflows$LockWorkflowAction           | `lock workflow {all|Mod.Wf|$W};`         |
//   | *genMf.UnlockWorkflowAction            | Microflows$UnlockWorkflowAction         | `unlock workflow {all|Mod.Wf|$W};`       |
//
// WorkflowOperationAction sub-ops (dispatched by formatWorkflowOperationGen):
//
//   | gen Go type              | BSON $Type                  | MDL keyword                                |
//   |--------------------------|-----------------------------|--------------------------------------------|
//   | *genMf.AbortOperation    | Microflows$AbortOperation   | `workflow operation abort $W [reason '…'];`|
//   | *genMf.ContinueOperation | Microflows$ContinueOperation| `workflow operation continue $W;`          |
//   | *genMf.PauseOperation    | Microflows$PauseOperation   | `workflow operation pause $W;`             |
//   | *genMf.RestartOperation  | Microflows$RestartOperation | `workflow operation restart $W;`           |
//   | *genMf.RetryOperation    | Microflows$RetryOperation   | `workflow operation retry $W;`             |
//   | *genMf.UnpauseOperation  | Microflows$UnpauseOperation | `workflow operation unpause $W;`           |
//   | *genMf.ResumeOperation   | Microflows$ResumeOperation  | `workflow operation unpause $W;` (modern alias) |
//
// Notable gen / legacy alignment notes preserved verbatim:
//
//  1. `LockWorkflowAction` / `UnlockWorkflowAction` legacy reads
//     `WorkflowSelection` and dispatches on its `$Type` to populate
//     either `Workflow` (qualified name) or `WorkflowVariable`. Gen
//     exposes both `WorkflowQualifiedName()` (bound to the BSON `Workflow`
//     key when the Selection is a NameSelection — the gen `init` reads
//     this via a top-level `Workflow` key) and a `WorkflowSelection()`
//     element. We dispatch on the selection element when present and
//     fall back to the top-level qualified name otherwise so both the
//     modern (Selection-typed) and older (top-level Workflow string)
//     BSON shapes render correctly.
//
//  2. `AbortOperation.Reason` in legacy is a plain string extracted from
//     the BSON sub-document `Reason.Text` (StringTemplate). Gen exposes
//     `Reason() element.Element` which decodes to a `*genMf.StringTemplate`.
//     We read `.Text()` from the StringTemplate and skip rendering when
//     the template is empty — matching legacy's empty-string suppression.
//
//  3. Gen has a `ResumeOperation` type that legacy does not handle.
//     Modern Mendix versions use `ResumeOperation` as the new name for
//     `UnpauseOperation`. We render both with the same `unpause` keyword
//     so legacy text parity is preserved while still supporting future
//     fixtures that use the modern type name.
//
//  4. `WorkflowCallAction` legacy reads `Workflow` (qualified name) and
//     `WorkflowContextVariable` directly. Gen renames the qualified-name
//     field to `WorkflowQualifiedName()` (bound to the same `Workflow`
//     BSON key — verified by reading `init` in
//     `modelsdk/gen/microflows/types.go`).

package executor

import (
	"fmt"
	"strings"

	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"

	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// ────────────────────────────────────────────────────────
// GetWorkflowDataAction
// ────────────────────────────────────────────────────────

// formatGetWorkflowDataActionGen emits
// `[$Out =] get workflow data $WfVar as Module.Wf;`. Mirrors legacy
// GetWorkflowDataAction handling: the `as <qualified>` clause carries
// the workflow type even when no output variable is set.
func formatGetWorkflowDataActionGen(a *genMf.GetWorkflowDataAction) string {
	wfVar := a.WorkflowVariable()
	wfQN := a.WorkflowQualifiedName()
	if outVar := a.OutputVariableName(); outVar != "" {
		return fmt.Sprintf("$%s = get workflow data $%s as %s;", outVar, wfVar, wfQN)
	}
	return fmt.Sprintf("get workflow data $%s as %s;", wfVar, wfQN)
}

// ────────────────────────────────────────────────────────
// WorkflowCallAction
// ────────────────────────────────────────────────────────

// formatWorkflowCallActionGen emits
// `[$Out =] call workflow Module.Wf ($Ctx);`. Mirrors legacy
// WorkflowCallAction: the output variable is rendered only when both
// `UseReturnVariable` is true and `OutputVariableName` is set.
func formatWorkflowCallActionGen(a *genMf.WorkflowCallAction) string {
	wfQN := a.WorkflowQualifiedName()
	ctxVar := a.WorkflowContextVariable()

	// Grammar: callArgument : (VARIABLE | parameterName) EQUALS expression
	// The write path stores only the variable name, not the formal parameter name.
	// Use ctxVar as both the parameter name and the variable to ensure roundtrip
	// stability: re-executing the output restores the same ctxVar value.
	var argStr string
	if ctxVar != "" {
		argStr = fmt.Sprintf("(%s = $%s)", ctxVar, ctxVar)
	} else {
		argStr = "()"
	}

	if a.UseReturnVariable() {
		if outVar := a.OutputVariableName(); outVar != "" {
			return fmt.Sprintf("$%s = call workflow %s %s;", outVar, wfQN, argStr)
		}
	}
	return fmt.Sprintf("call workflow %s %s;", wfQN, argStr)
}

// ────────────────────────────────────────────────────────
// GetWorkflowsAction
// ────────────────────────────────────────────────────────

// formatGetWorkflowsActionGen emits
// `[$Out =] get workflows for $Ctx;`. Mirrors legacy
// GetWorkflowsAction handling.
func formatGetWorkflowsActionGen(a *genMf.GetWorkflowsAction) string {
	ctx := a.WorkflowContextVariableName()
	if outVar := a.OutputVariableName(); outVar != "" {
		return fmt.Sprintf("$%s = get workflows for $%s;", outVar, ctx)
	}
	return fmt.Sprintf("get workflows for $%s;", ctx)
}

// ────────────────────────────────────────────────────────
// GetWorkflowActivityRecordsAction
// ────────────────────────────────────────────────────────

// formatGetWorkflowActivityRecordsActionGen emits
// `[$Out =] get workflow activity records $WfVar;`.
func formatGetWorkflowActivityRecordsActionGen(a *genMf.GetWorkflowActivityRecordsAction) string {
	wfVar := a.WorkflowVariable()
	if outVar := a.OutputVariableName(); outVar != "" {
		return fmt.Sprintf("$%s = get workflow activity records $%s;", outVar, wfVar)
	}
	return fmt.Sprintf("get workflow activity records $%s;", wfVar)
}

// ────────────────────────────────────────────────────────
// WorkflowOperationAction (dispatcher)
// ────────────────────────────────────────────────────────

// formatWorkflowOperationActionGen wraps the inner Operation element with
// the action-level surface. Mirrors legacy WorkflowOperationAction: nil
// Operation gives the stable `workflow operation ...;` placeholder.
func formatWorkflowOperationActionGen(a *genMf.WorkflowOperationAction) string {
	op := a.Operation()
	if op == nil {
		return "workflow operation ...;"
	}
	return formatWorkflowOperationGen(op)
}

// formatWorkflowOperationGen dispatches a workflow-operation primitive
// to its MDL form. Output is 1:1 with legacy `formatWorkflowOperationAction`
// so the migrated body diffs cleanly against the SDK path. The modern
// `ResumeOperation` (new gen-only type) renders with the same `unpause`
// keyword as the legacy `UnpauseOperation` for parity.
func formatWorkflowOperationGen(op element.Element) string {
	switch o := op.(type) {
	case *genMf.AbortOperation:
		reason := abortReasonStringGen(o)
		if reason != "" {
			// reason is already a valid MDL expression string (e.g. '''text'''),
			// so emit it as-is without further quoting.
			return fmt.Sprintf("workflow operation abort $%s reason %s;", o.WorkflowVariable(), reason)
		}
		return fmt.Sprintf("workflow operation abort $%s;", o.WorkflowVariable())
	case *genMf.ContinueOperation:
		return fmt.Sprintf("workflow operation continue $%s;", o.WorkflowVariable())
	case *genMf.PauseOperation:
		return fmt.Sprintf("workflow operation pause $%s;", o.WorkflowVariable())
	case *genMf.RestartOperation:
		return fmt.Sprintf("workflow operation restart $%s;", o.WorkflowVariable())
	case *genMf.RetryOperation:
		return fmt.Sprintf("workflow operation retry $%s;", o.WorkflowVariable())
	case *genMf.UnpauseOperation:
		return fmt.Sprintf("workflow operation unpause $%s;", o.WorkflowVariable())
	case *genMf.ResumeOperation:
		return fmt.Sprintf("workflow operation unpause $%s;", o.WorkflowVariable())
	default:
		return fmt.Sprintf("-- Unknown workflow operation: %T", op)
	}
}

// abortReasonStringGen returns the text body of an AbortOperation's
// Reason element, which decodes to a `*genMf.StringTemplate`. Returns
// "" when the reason is nil, not a StringTemplate, or has empty text —
// matching legacy `extractString(reason["Text"])` behaviour.
func abortReasonStringGen(o *genMf.AbortOperation) string {
	tmpl, ok := o.Reason().(*genMf.StringTemplate)
	if !ok || tmpl == nil {
		return ""
	}
	return strings.TrimSpace(tmpl.Text())
}

// ────────────────────────────────────────────────────────
// SetTaskOutcomeAction
// ────────────────────────────────────────────────────────

// formatSetTaskOutcomeActionGen emits
// `set task outcome $TaskVar '<value>';`. Mirrors legacy
// SetTaskOutcomeAction: the outcome value is mdlQuote'd (escaping
// internal single quotes via doubling).
func formatSetTaskOutcomeActionGen(a *genMf.SetTaskOutcomeAction) string {
	return fmt.Sprintf("set task outcome $%s %s;", a.WorkflowTaskVariable(), mdlQuote(a.OutcomeValue()))
}

// ────────────────────────────────────────────────────────
// OpenUserTaskAction
// ────────────────────────────────────────────────────────

// formatOpenUserTaskActionGen emits `open user task $TaskVar;`.
// Mirrors legacy OpenUserTaskAction (only the variable surface is
// rendered; AssignOnOpen / OpenWhenAssigned are not part of the legacy
// MDL shape).
func formatOpenUserTaskActionGen(a *genMf.OpenUserTaskAction) string {
	return fmt.Sprintf("open user task $%s;", a.UserTaskVariable())
}

// ────────────────────────────────────────────────────────
// NotifyWorkflowAction
// ────────────────────────────────────────────────────────

// formatNotifyWorkflowActionGen emits
// `[$Out =] notify workflow $WfVar [activity Module.WF.ActivityName];`.
func formatNotifyWorkflowActionGen(a *genMf.NotifyWorkflowAction) string {
	wfVar := a.WorkflowVariable()
	actPart := ""
	if aqn := a.ActivityQualifiedName(); aqn != "" {
		actPart = " activity " + aqn
	}
	if outVar := a.OutputVariableName(); outVar != "" {
		return fmt.Sprintf("$%s = notify workflow $%s%s;", outVar, wfVar, actPart)
	}
	return fmt.Sprintf("notify workflow $%s%s;", wfVar, actPart)
}

// ────────────────────────────────────────────────────────
// OpenWorkflowAction
// ────────────────────────────────────────────────────────

// formatOpenWorkflowActionGen emits `open workflow $WfVar;`.
func formatOpenWorkflowActionGen(a *genMf.OpenWorkflowAction) string {
	return fmt.Sprintf("open workflow $%s;", a.WorkflowVariable())
}

// ────────────────────────────────────────────────────────
// LockWorkflowAction / UnlockWorkflowAction
// ────────────────────────────────────────────────────────

// formatLockWorkflowActionGen emits one of three forms:
//
//   - `lock workflow all;` when PauseAllWorkflows is true
//   - `lock workflow Module.Wf;` when a Name selection is present
//   - `lock workflow $WfVar;`     when an Object selection is present
//
// Mirrors legacy LockWorkflowAction handling. Falls back to the
// top-level WorkflowQualifiedName when no selection element is set
// (older BSON shape), then to the empty string `lock workflow ;` when
// neither source resolves a workflow.
func formatLockWorkflowActionGen(a *genMf.LockWorkflowAction) string {
	if a.PauseAllWorkflows() {
		return "lock workflow all;"
	}
	wfQN, wfVar := workflowSelectionGen(a.WorkflowSelection(), a.WorkflowQualifiedName())
	if wfQN != "" {
		return fmt.Sprintf("lock workflow %s;", wfQN)
	}
	return fmt.Sprintf("lock workflow $%s;", wfVar)
}

// formatUnlockWorkflowActionGen mirrors formatLockWorkflowActionGen
// for the `unlock` variant. ResumeAllPausedWorkflows replaces
// PauseAllWorkflows as the "all" sentinel.
func formatUnlockWorkflowActionGen(a *genMf.UnlockWorkflowAction) string {
	if a.ResumeAllPausedWorkflows() {
		return "unlock workflow all;"
	}
	wfQN, wfVar := workflowSelectionGen(a.WorkflowSelection(), a.WorkflowQualifiedName())
	if wfQN != "" {
		return fmt.Sprintf("unlock workflow %s;", wfQN)
	}
	return fmt.Sprintf("unlock workflow $%s;", wfVar)
}

// formatGenerateJumpToOptionsActionGen emits
// `[$Out =] generate jump to options for $wf as Module.WF_Name;`
func formatGenerateJumpToOptionsActionGen(a *genMf.GenerateJumpToOptionsAction) string {
	wfVar := a.WorkflowVariable()
	wfQN := a.WorkflowQualifiedName()
	if outVar := a.OutputVariableName(); outVar != "" {
		return fmt.Sprintf("$%s = generate jump to options for $%s as %s;", outVar, wfVar, wfQN)
	}
	return fmt.Sprintf("generate jump to options for $%s as %s;", wfVar, wfQN)
}

// formatApplyJumpToOptionActionGen emits
// `[$Out =] apply jump to option $options;`
func formatApplyJumpToOptionActionGen(a *genMf.ApplyJumpToOptionAction) string {
	optVar := a.WorkflowJumpToDetailsVariable()
	if outVar := a.OutputVariableName(); outVar != "" {
		return fmt.Sprintf("$%s = apply jump to option $%s;", outVar, optVar)
	}
	return fmt.Sprintf("apply jump to option $%s;", optVar)
}

// workflowSelectionGen normalises a Lock/Unlock action's WorkflowSelection
// element + top-level qualified-name fallback into a (qualifiedName,
// workflowVariable) pair. Exactly one of the two return values is
// expected to be non-empty for a well-formed action; the caller picks
// the rendering form based on which is set.
//
// Dispatch order matches legacy `parseLockWorkflowAction` /
// `parseUnlockWorkflowAction`: prefer the typed Selection element, then
// fall back to the top-level qualified name when no Selection is
// present (older BSON shape that gen still decodes via the top-level
// `Workflow` BSON key).
func workflowSelectionGen(sel element.Element, topLevelQN string) (string, string) {
	switch s := sel.(type) {
	case *genWf.WorkflowDefinitionNameSelection:
		if s != nil {
			if qn := s.WorkflowQualifiedName(); qn != "" {
				return qn, ""
			}
		}
	case *genWf.WorkflowDefinitionObjectSelection:
		if s != nil {
			if v := s.WorkflowDefinitionVariable(); v != "" {
				return "", v
			}
		}
	}
	// Fallback: gen reads the top-level `Workflow` BSON key into
	// LockWorkflowAction.WorkflowQualifiedName / Unlock equivalent
	// even when no Selection sub-document is present.
	return topLevelQN, ""
}

// formatSynchronizeActionGen renders SYNCHRONIZE (All mode) or
// SYNCHRONIZE $var (SelectedObjects mode).
func formatSynchronizeActionGen(a *genMf.SynchronizeAction) string {
	if a == nil {
		return ""
	}
	stmt := "synchronize"
	if a.Type() == "SelectedObjects" {
		vars := strings.TrimSpace(a.VariableNames())
		if vars != "" {
			parts := strings.Split(vars, ",")
			for i, v := range parts {
				parts[i] = "$" + strings.TrimSpace(v)
			}
			stmt += " " + strings.Join(parts, ", ")
		}
	}
	return stmt + ";"
}

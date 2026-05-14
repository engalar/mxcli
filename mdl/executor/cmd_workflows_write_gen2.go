// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.3 Phase D — gen-typed CREATE WORKFLOW write path.
//
// This file is the gen-typed twin of cmd_workflows_write.go. The legacy
// build* functions stay in tree until Phase E2 deletes the whole file.
// Each gen-typed builder lands in its own D1 sub-commit so reviewers
// can tell them apart in git log.
//
// Notes:
//
//   - Activity dispatch is by AST Go type (matches legacy
//     buildWorkflowActivity).
//   - Storage type assignment (SetTypeName) is automatic — every
//     gen *NewX() factory calls SetTypeName with the canonical
//     Workflows$XYZ string.
//   - Boundary event types: gen splits into 3 concrete subtypes
//     (InterruptingTimerBoundaryEvent / NonInterruptingTimerBoundaryEvent /
//     TimerBoundaryEvent). buildBoundaryEventGen dispatches by AST EventType.
//
// D1.a covers the leaf builders (no nested flow): JumpTo, WaitForTimer,
// WaitForNotification, EndWorkflow, FloatingAnnotation, plus the
// boundary-event helper. The composite formatters (UserTask, CallMicroflow,
// CallWorkflow, ExclusiveSplit, ParallelSplit) land in D1.b/c/d.

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	"github.com/mendixlabs/mxcli/modelsdk/gen/texts"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// buildWorkflowActivitiesGen mirrors buildWorkflowActivities
// (cmd_workflows_write.go:170): walks AST nodes and builds gen-typed
// activity elements. Returns []element.Element so AddActivities on the
// gen Flow accepts them directly.
func buildWorkflowActivitiesGen(nodes []ast.WorkflowActivityNode) []element.Element {
	var out []element.Element
	for _, node := range nodes {
		if elem := buildWorkflowActivityGen(node); elem != nil {
			out = append(out, elem)
		}
	}
	return out
}

// buildWorkflowActivityGen dispatches a single AST node to its concrete
// gen-typed builder. Composite builders (UserTask, CallMicroflow, …)
// land in D1.b/c/d sub-commits; leaf cases are wired here in D1.a.
func buildWorkflowActivityGen(node ast.WorkflowActivityNode) element.Element {
	switch n := node.(type) {
	case *ast.WorkflowJumpToNode:
		return buildJumpToGenActivity(n)
	case *ast.WorkflowWaitForTimerNode:
		return buildWaitForTimerGenActivity(n)
	case *ast.WorkflowWaitForNotificationNode:
		return buildWaitForNotificationGenActivity(n)
	case *ast.WorkflowEndNode:
		return buildEndWorkflowGenActivity(n)
	case *ast.WorkflowAnnotationActivityNode:
		return buildAnnotationActivityGen(n)
	case *ast.WorkflowUserTaskNode:
		return buildUserTaskGenActivity(n)
	case *ast.WorkflowCallMicroflowNode:
		return buildCallMicroflowGenActivity(n)
	case *ast.WorkflowCallWorkflowNode:
		return buildCallWorkflowGenActivity(n)
	case *ast.WorkflowDecisionNode:
		return buildExclusiveSplitGenActivity(n)
	case *ast.WorkflowParallelSplitNode:
		return buildParallelSplitGenActivity(n)
	}
	return nil
}

// buildJumpToGenActivity mirrors buildJumpTo (cmd_workflows_write.go:444).
func buildJumpToGenActivity(n *ast.WorkflowJumpToNode) *genWf.JumpToActivity {
	act := genWf.NewJumpToActivity()
	act.SetID(element.ID(types.GenerateID()))
	act.SetName(n.Target)
	caption := n.Caption
	if caption == "" {
		caption = n.Target
	}
	act.SetCaption(caption)
	act.SetTargetActivityQualifiedName(n.Target)
	return act
}

// buildWaitForTimerGenActivity mirrors buildWaitForTimer
// (cmd_workflows_write.go:458). Note the gen field rename:
// WaitForTimerActivity.Delay() corresponds to sdk DelayExpression.
func buildWaitForTimerGenActivity(n *ast.WorkflowWaitForTimerNode) *genWf.WaitForTimerActivity {
	act := genWf.NewWaitForTimerActivity()
	act.SetID(element.ID(types.GenerateID()))
	caption := n.Caption
	if caption == "" {
		caption = "Wait for timer"
	}
	act.SetCaption(caption)
	act.SetName(caption)
	act.SetDelay(n.DelayExpression)
	return act
}

// buildWaitForNotificationGenActivity mirrors buildWaitForNotification
// (cmd_workflows_write.go:472).
func buildWaitForNotificationGenActivity(n *ast.WorkflowWaitForNotificationNode) *genWf.WaitForNotificationActivity {
	act := genWf.NewWaitForNotificationActivity()
	act.SetID(element.ID(types.GenerateID()))
	caption := n.Caption
	if caption == "" {
		caption = "Wait for notification"
	}
	act.SetCaption(caption)
	act.SetName(caption)
	for _, ev := range buildBoundaryEventsGen(n.BoundaryEvents) {
		act.AddBoundaryEvents(ev)
	}
	return act
}

// buildEndWorkflowGenActivity mirrors buildEndWorkflow
// (cmd_workflows_write.go:488).
func buildEndWorkflowGenActivity(n *ast.WorkflowEndNode) *genWf.EndWorkflowActivity {
	act := genWf.NewEndWorkflowActivity()
	act.SetID(element.ID(types.GenerateID()))
	caption := n.Caption
	if caption == "" {
		caption = "End"
	}
	act.SetCaption(caption)
	act.SetName(caption)
	return act
}

// buildAnnotationActivityGen mirrors buildAnnotationActivity
// (cmd_workflows_write.go:586). gen exposes 3 annotation-shaped types
// (Annotation / AnnotationActivity / FloatingAnnotation); the legacy
// `WorkflowAnnotationActivity` storage maps to FloatingAnnotation —
// the inflow positioned annotation node.
func buildAnnotationActivityGen(n *ast.WorkflowAnnotationActivityNode) *genWf.FloatingAnnotation {
	a := genWf.NewFloatingAnnotation()
	a.SetID(element.ID(types.GenerateID()))
	a.SetDescription(n.Text)
	return a
}

// buildBoundaryEventsGen mirrors buildBoundaryEvents
// (cmd_workflows_write.go:182). gen splits BoundaryEvent into three
// concrete subtypes — dispatch by AST EventType string.
func buildBoundaryEventsGen(nodes []ast.WorkflowBoundaryEventNode) []element.Element {
	var out []element.Element
	for _, be := range nodes {
		ev := buildBoundaryEventGen(be)
		if ev != nil {
			out = append(out, ev)
		}
	}
	return out
}

// buildBoundaryEventGen builds a single gen boundary-event element.
// gen field rename: Delay() (was sdk TimerDelay).
func buildBoundaryEventGen(be ast.WorkflowBoundaryEventNode) element.Element {
	id := element.ID(types.GenerateID())
	subActivities := buildWorkflowActivitiesGen(be.Activities)
	flow := newGenFlowWithActivities(subActivities)

	switch be.EventType {
	case "InterruptingTimer":
		ev := genWf.NewInterruptingTimerBoundaryEvent()
		ev.SetID(id)
		ev.SetIsInterrupting(true)
		ev.SetEventType(be.EventType)
		ev.SetDelay(be.Delay)
		if flow != nil {
			ev.SetFlow(flow)
		}
		return ev
	case "NonInterruptingTimer":
		ev := genWf.NewNonInterruptingTimerBoundaryEvent()
		ev.SetID(id)
		ev.SetIsInterrupting(false)
		ev.SetEventType(be.EventType)
		ev.SetDelay(be.Delay)
		if flow != nil {
			ev.SetFlow(flow)
		}
		return ev
	default: // "Timer" or anything else falls back to the abstract base
		ev := genWf.NewTimerBoundaryEvent()
		ev.SetID(id)
		ev.SetEventType(be.EventType)
		ev.SetDelay(be.Delay)
		if flow != nil {
			ev.SetFlow(flow)
		}
		return ev
	}
}

// newGenFlowWithActivities wraps a slice of gen activity elements in a
// gen *Flow Part. Returns nil when the slice is empty so callers don't
// emit empty Flow nodes.
func newGenFlowWithActivities(activities []element.Element) *genWf.Flow {
	if len(activities) == 0 {
		return nil
	}
	flow := genWf.NewFlow()
	flow.SetID(element.ID(types.GenerateID()))
	for _, a := range activities {
		flow.AddActivities(a)
	}
	return flow
}

// ---------------------------------------------------------------------------
// D1.b — UserTask family + UserSource + UserTaskOutcome
// ---------------------------------------------------------------------------

// buildUserTaskGenActivity mirrors buildUserTask (cmd_workflows_write.go:229).
// Dispatches by AST IsMultiUser to either MultiUserTaskActivity (true)
// or SingleUserTaskActivity (false). The legacy unified UserTask gen
// type still exists for back-compat decode; new writes pick a concrete
// subtype.
func buildUserTaskGenActivity(n *ast.WorkflowUserTaskNode) element.Element {
	if n.IsMultiUser {
		return buildMultiUserTaskGenActivity(n)
	}
	return buildSingleUserTaskGenActivity(n)
}

func buildSingleUserTaskGenActivity(n *ast.WorkflowUserTaskNode) *genWf.SingleUserTaskActivity {
	task := genWf.NewSingleUserTaskActivity()
	task.SetID(element.ID(types.GenerateID()))
	task.SetName(n.Name)
	task.SetCaption(n.Caption)
	task.SetDueDate(n.DueDate)
	if n.TaskDescription != "" {
		task.SetTaskDescription(newTextWrapperGen(n.TaskDescription))
	}
	if src := buildUserSourceGen(n.Targeting); src != nil {
		task.SetUserSource(src)
	}
	for _, oc := range buildUserTaskOutcomesGen(n.Outcomes) {
		task.AddOutcomes(oc)
	}
	for _, ev := range buildBoundaryEventsGen(n.BoundaryEvents) {
		task.AddBoundaryEvents(ev)
	}
	return task
}

func buildMultiUserTaskGenActivity(n *ast.WorkflowUserTaskNode) *genWf.MultiUserTaskActivity {
	task := genWf.NewMultiUserTaskActivity()
	task.SetID(element.ID(types.GenerateID()))
	task.SetName(n.Name)
	task.SetCaption(n.Caption)
	task.SetDueDate(n.DueDate)
	if n.TaskDescription != "" {
		task.SetTaskDescription(newTextWrapperGen(n.TaskDescription))
	}
	if src := buildUserSourceGen(n.Targeting); src != nil {
		task.SetUserSource(src)
	}
	for _, oc := range buildUserTaskOutcomesGen(n.Outcomes) {
		task.AddOutcomes(oc)
	}
	for _, ev := range buildBoundaryEventsGen(n.BoundaryEvents) {
		task.AddBoundaryEvents(ev)
	}
	return task
}

// buildUserSourceGen builds a gen UserSource element from the AST
// targeting node. gen rename: MicroflowGroupSource → MicroflowGroupTargeting,
// XPathGroupSource → XPathGroupTargeting (per Phase A R5 finding).
func buildUserSourceGen(t ast.WorkflowTargetingNode) element.Element {
	switch t.Kind {
	case "microflow":
		src := genWf.NewMicroflowBasedUserSource()
		src.SetMicroflowQualifiedName(t.Microflow.Module + "." + t.Microflow.Name)
		return src
	case "xpath":
		src := genWf.NewXPathBasedUserSource()
		src.SetXPathConstraint(t.XPath)
		return src
	case "group_microflow":
		src := genWf.NewMicroflowGroupTargeting()
		src.SetMicroflowQualifiedName(t.Microflow.Module + "." + t.Microflow.Name)
		return src
	case "group_xpath":
		src := genWf.NewXPathGroupTargeting()
		src.SetXPathConstraint(t.XPath)
		return src
	}
	return nil
}

// buildUserTaskOutcomesGen mirrors the outcomes loop in buildUserTask
// (cmd_workflows_write.go:267). Each outcome stores the same string in
// Name/Caption/Value (legacy semantics).
func buildUserTaskOutcomesGen(nodes []ast.WorkflowUserTaskOutcomeNode) []*genWf.UserTaskOutcome {
	out := make([]*genWf.UserTaskOutcome, 0, len(nodes))
	for _, n := range nodes {
		oc := genWf.NewUserTaskOutcome()
		oc.SetID(element.ID(types.GenerateID()))
		oc.SetName(n.Caption)
		oc.SetCaption(n.Caption)
		oc.SetValue(n.Caption)
		if flow := newGenFlowWithActivities(buildWorkflowActivitiesGen(n.Activities)); flow != nil {
			oc.SetFlow(flow)
		}
		out = append(out, oc)
	}
	return out
}

// ---------------------------------------------------------------------------
// D1.c — CallMicroflow + CallWorkflow + ParameterMapping
// ---------------------------------------------------------------------------

// buildCallMicroflowGenActivity mirrors buildCallMicroflowTask
// (cmd_workflows_write.go:291). Picks Workflows$CallMicroflowActivity
// (the gen-canonical storage) over the legacy CallMicroflowTask so
// fresh writes round-trip through the new gen schema.
func buildCallMicroflowGenActivity(n *ast.WorkflowCallMicroflowNode) *genWf.CallMicroflowActivity {
	act := genWf.NewCallMicroflowActivity()
	act.SetID(element.ID(types.GenerateID()))
	act.SetName(n.Microflow.Name)
	caption := n.Caption
	if caption == "" {
		caption = n.Microflow.Name
	}
	act.SetCaption(caption)
	mfQN := n.Microflow.Module + "." + n.Microflow.Name
	act.SetMicroflowQualifiedName(mfQN)

	// Outcomes
	for _, oc := range buildConditionOutcomesGen(n.Outcomes) {
		act.AddOutcomes(oc)
	}

	// Parameter mappings — fully qualified per BSON requirement
	// (legacy buildCallMicroflowTask:312).
	for _, pm := range n.ParameterMappings {
		mapping := genWf.NewMicroflowCallParameterMapping()
		mapping.SetID(element.ID(types.GenerateID()))
		mapping.SetParameterQualifiedName(mfQN + "." + pm.Parameter)
		mapping.SetExpression(pm.Expression)
		act.AddParameterMappings(mapping)
	}

	// Boundary events
	for _, ev := range buildBoundaryEventsGen(n.BoundaryEvents) {
		act.AddBoundaryEvents(ev)
	}
	return act
}

// buildCallWorkflowGenActivity mirrors buildCallWorkflowActivity
// (cmd_workflows_write.go:325).
func buildCallWorkflowGenActivity(n *ast.WorkflowCallWorkflowNode) *genWf.CallWorkflowActivity {
	act := genWf.NewCallWorkflowActivity()
	act.SetID(element.ID(types.GenerateID()))
	act.SetName(n.Workflow.Name)
	caption := n.Caption
	if caption == "" {
		caption = n.Workflow.Name
	}
	act.SetCaption(caption)
	wfQN := n.Workflow.Module + "." + n.Workflow.Name
	act.SetWorkflowQualifiedName(wfQN)
	// Auto-bind $WorkflowContext expression — same default the legacy
	// builder applied. autoBindCallWorkflowGen (D5) will refine when
	// the target workflow has no Parameter.
	act.SetParameterExpression("$WorkflowContext")

	for _, pm := range n.ParameterMappings {
		mapping := genWf.NewWorkflowCallParameterMapping()
		mapping.SetID(element.ID(types.GenerateID()))
		mapping.SetParameterQualifiedName(wfQN + "." + pm.Parameter)
		mapping.SetExpression(pm.Expression)
		act.AddParameterMappings(mapping)
	}
	return act
}

// buildConditionOutcomesGen mirrors the outcomes loop in
// buildCallMicroflowTask (cmd_workflows_write.go:302). Dispatches on
// AST Value to BooleanConditionOutcome / VoidConditionOutcome /
// EnumerationValueConditionOutcome.
func buildConditionOutcomesGen(nodes []ast.WorkflowConditionOutcomeNode) []element.Element {
	out := make([]element.Element, 0, len(nodes))
	for _, n := range nodes {
		oc := buildConditionOutcomeGen(n)
		if oc != nil {
			out = append(out, oc)
		}
	}
	return out
}

// buildConditionOutcomeGen mirrors buildConditionOutcome
// (cmd_workflows_write.go:390).
func buildConditionOutcomeGen(n ast.WorkflowConditionOutcomeNode) element.Element {
	subFlow := newGenFlowWithActivities(buildWorkflowActivitiesGen(n.Activities))
	switch n.Value {
	case "True":
		o := genWf.NewBooleanConditionOutcome()
		o.SetID(element.ID(types.GenerateID()))
		o.SetValue(true)
		if subFlow != nil {
			o.SetFlow(subFlow)
		}
		return o
	case "False":
		o := genWf.NewBooleanConditionOutcome()
		o.SetID(element.ID(types.GenerateID()))
		o.SetValue(false)
		if subFlow != nil {
			o.SetFlow(subFlow)
		}
		return o
	case "Default":
		o := genWf.NewVoidConditionOutcome()
		o.SetID(element.ID(types.GenerateID()))
		if subFlow != nil {
			o.SetFlow(subFlow)
		}
		return o
	default:
		// Enumeration value — sdk EnumerationValueConditionOutcome.Value
		// is just the bare value; gen exposes it as ValueQualifiedName.
		o := genWf.NewEnumerationValueConditionOutcome()
		o.SetID(element.ID(types.GenerateID()))
		o.SetValueQualifiedName(n.Value)
		if subFlow != nil {
			o.SetFlow(subFlow)
		}
		return o
	}
}

// ---------------------------------------------------------------------------
// D1.d — ExclusiveSplit + ParallelSplit
// ---------------------------------------------------------------------------

// buildExclusiveSplitGenActivity mirrors buildExclusiveSplit
// (cmd_workflows_write.go:354). Implements the same boolean-decision
// detection rule: if any outcome is True/False, we drop any "Default"
// outcome (Mendix 11 runtime rejects VoidConditionOutcome on a
// boolean decision).
func buildExclusiveSplitGenActivity(n *ast.WorkflowDecisionNode) *genWf.ExclusiveSplitActivity {
	act := genWf.NewExclusiveSplitActivity()
	act.SetID(element.ID(types.GenerateID()))
	act.SetExpression(n.Expression)
	caption := n.Caption
	if caption == "" {
		caption = "Decision"
	}
	act.SetCaption(caption)
	act.SetName(caption)

	isBooleanDecision := false
	for _, o := range n.Outcomes {
		if o.Value == "True" || o.Value == "False" {
			isBooleanDecision = true
			break
		}
	}

	for _, oc := range n.Outcomes {
		if isBooleanDecision && oc.Value == "Default" {
			continue
		}
		built := buildConditionOutcomeGen(oc)
		if built != nil {
			act.AddOutcomes(built)
		}
	}
	return act
}

// buildParallelSplitGenActivity mirrors buildParallelSplit
// (cmd_workflows_write.go:420). Each AST path becomes a
// ParallelSplitOutcome carrying its own gen Flow.
func buildParallelSplitGenActivity(n *ast.WorkflowParallelSplitNode) *genWf.ParallelSplitActivity {
	act := genWf.NewParallelSplitActivity()
	act.SetID(element.ID(types.GenerateID()))
	caption := n.Caption
	if caption == "" {
		caption = "Parallel split"
	}
	act.SetCaption(caption)
	act.SetName(caption)

	for _, p := range n.Paths {
		oc := genWf.NewParallelSplitOutcome()
		oc.SetID(element.ID(types.GenerateID()))
		if flow := newGenFlowWithActivities(buildWorkflowActivitiesGen(p.Activities)); flow != nil {
			oc.SetFlow(flow)
		}
		act.AddOutcomes(oc)
	}
	return act
}

// newTextWrapperGen wraps a plain string in a Texts$Text element with
// a single English Translation (LanguageCode="en_US"). Mirrors the
// shape Studio Pro emits for default-language workflow text fields
// (TaskName, TaskDescription, WorkflowName, WorkflowDescription).
//
// The describe-side reader (readTextElementGen) accepts Text /
// Translation / Value field names with raw-BSON fallback so the
// round-trip is symmetrical even with this minimal payload.
func newTextWrapperGen(s string) element.Element {
	tx := texts.NewText()
	tx.SetID(element.ID(types.GenerateID()))
	tr := texts.NewTranslation()
	tr.SetID(element.ID(types.GenerateID()))
	tr.SetLanguageCode("en_US")
	tr.SetText(s)
	tx.AddTranslations(tr)
	return tx
}

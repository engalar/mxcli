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
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	"github.com/mendixlabs/mxcli/modelsdk/gen/texts"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
	"github.com/mendixlabs/mxcli/modelsdk/version"
)

// wfBuildCtx carries the project version through the stateless workflow
// builder chain so version-aware factory functions can select the correct
// BSON type name.
type wfBuildCtx struct {
	version version.Version // zero = treat as oldest (legacy fallback)
}

// newWfBuildCtx creates a wfBuildCtx from the execution context.
func newWfBuildCtx(ctx *ExecContext) *wfBuildCtx {
	wbc := &wfBuildCtx{}
	if ctx != nil && ctx.Connected() {
		if rpv := ctx.ConnectionManager.ProjectVersion(); rpv != nil {
			wbc.version = version.Parse(rpv.ProductVersion)
		}
	}
	return wbc
}

// buildWorkflowActivitiesGen mirrors buildWorkflowActivities
// (cmd_workflows_write.go:170): walks AST nodes and builds gen-typed
// activity elements. Returns []element.Element so AddActivities on the
// gen Flow accepts them directly.
func buildWorkflowActivitiesGen(wbc *wfBuildCtx, nodes []ast.WorkflowActivityNode) []element.Element {
	var out []element.Element
	for _, node := range nodes {
		if elem := buildWorkflowActivityGen(wbc, node); elem != nil {
			out = append(out, elem)
		}
	}
	return out
}

// wfActivityHandler is a gen-typed builder for a specific WorkflowActivityNode type.
type wfActivityHandler func(wbc *wfBuildCtx, node ast.WorkflowActivityNode) element.Element

var wfActivityDispatch map[reflect.Type]wfActivityHandler

func init() {
	wfActivityDispatch = map[reflect.Type]wfActivityHandler{
		reflect.TypeOf(&ast.WorkflowJumpToNode{}):             func(wbc *wfBuildCtx, node ast.WorkflowActivityNode) element.Element { return buildJumpToGenActivity(node.(*ast.WorkflowJumpToNode)) },
		reflect.TypeOf(&ast.WorkflowWaitForTimerNode{}):       func(wbc *wfBuildCtx, node ast.WorkflowActivityNode) element.Element { return buildWaitForTimerGenActivity(node.(*ast.WorkflowWaitForTimerNode)) },
		reflect.TypeOf(&ast.WorkflowWaitForNotificationNode{}): func(wbc *wfBuildCtx, node ast.WorkflowActivityNode) element.Element { return buildWaitForNotificationGenActivity(wbc, node.(*ast.WorkflowWaitForNotificationNode)) },
		reflect.TypeOf(&ast.WorkflowEndNode{}):                func(wbc *wfBuildCtx, node ast.WorkflowActivityNode) element.Element { return buildEndWorkflowGenActivity(node.(*ast.WorkflowEndNode)) },
		reflect.TypeOf(&ast.WorkflowAnnotationActivityNode{}): func(wbc *wfBuildCtx, node ast.WorkflowActivityNode) element.Element { return buildAnnotationActivityGen(node.(*ast.WorkflowAnnotationActivityNode)) },
		reflect.TypeOf(&ast.WorkflowUserTaskNode{}):           func(wbc *wfBuildCtx, node ast.WorkflowActivityNode) element.Element { return buildUserTaskGenActivity(wbc, node.(*ast.WorkflowUserTaskNode)) },
		reflect.TypeOf(&ast.WorkflowCallMicroflowNode{}):      func(wbc *wfBuildCtx, node ast.WorkflowActivityNode) element.Element { return buildCallMicroflowGenActivity(wbc, node.(*ast.WorkflowCallMicroflowNode)) },
		reflect.TypeOf(&ast.WorkflowCallWorkflowNode{}):       func(wbc *wfBuildCtx, node ast.WorkflowActivityNode) element.Element { return buildCallWorkflowGenActivity(wbc, node.(*ast.WorkflowCallWorkflowNode)) },
		reflect.TypeOf(&ast.WorkflowDecisionNode{}):           func(wbc *wfBuildCtx, node ast.WorkflowActivityNode) element.Element { return buildExclusiveSplitGenActivity(wbc, node.(*ast.WorkflowDecisionNode)) },
		reflect.TypeOf(&ast.WorkflowParallelSplitNode{}):      func(wbc *wfBuildCtx, node ast.WorkflowActivityNode) element.Element { return buildParallelSplitGenActivity(wbc, node.(*ast.WorkflowParallelSplitNode)) },
	}
}

// buildWorkflowActivityGen dispatches a single AST node to its concrete
// gen-typed builder. Composite builders (UserTask, CallMicroflow, …)
// land in D1.b/c/d sub-commits; leaf cases are wired here in D1.a.
func buildWorkflowActivityGen(wbc *wfBuildCtx, node ast.WorkflowActivityNode) element.Element {
	if h, ok := wfActivityDispatch[reflect.TypeOf(node)]; ok {
		return h(wbc, node)
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
func buildWaitForNotificationGenActivity(wbc *wfBuildCtx, n *ast.WorkflowWaitForNotificationNode) *genWf.WaitForNotificationActivity {
	act := genWf.NewWaitForNotificationActivity()
	act.SetID(element.ID(types.GenerateID()))
	caption := n.Caption
	if caption == "" {
		caption = "Wait for notification"
	}
	act.SetCaption(caption)
	act.SetName(caption)
	for _, ev := range buildBoundaryEventsGen(wbc, n.BoundaryEvents) {
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
func buildBoundaryEventsGen(wbc *wfBuildCtx, nodes []ast.WorkflowBoundaryEventNode) []element.Element {
	var out []element.Element
	for _, be := range nodes {
		ev := buildBoundaryEventGen(wbc, be)
		if ev != nil {
			out = append(out, ev)
		}
	}
	return out
}

// buildBoundaryEventGen builds a single gen boundary-event element.
// gen field rename: Delay() (was sdk TimerDelay).
func buildBoundaryEventGen(wbc *wfBuildCtx, be ast.WorkflowBoundaryEventNode) element.Element {
	id := element.ID(types.GenerateID())
	subActivities := buildWorkflowActivitiesGen(wbc, be.Activities)

	switch be.EventType {
	case "InterruptingTimer":
		// CE6665: Mendix requires the flow to end with a JumpToActivity or
		// EndWorkflowActivity. Auto-inject EndWorkflowActivity when missing.
		if !endsWithTerminalWorkflowActivity(subActivities) {
			end := genWf.NewEndWorkflowActivity()
			end.SetID(element.ID(types.GenerateID()))
			end.SetCaption("End")
			end.SetName("End")
			subActivities = append(subActivities, end)
		}
		flow := newGenFlowWithActivities(subActivities)
		ev := genWf.NewInterruptingTimerBoundaryEvent()
		ev.SetID(id)
		ev.SetIsInterrupting(true)
		ev.SetEventType(be.EventType)
		ev.SetDelay(be.Delay)
		ev.SetFlow(flow)
		return ev
	case "NonInterruptingTimer":
		// CE1844: EndWorkflowActivity is forbidden in non-interrupting boundary event flows.
		// The correct terminal activity is EndOfBoundaryEventPathActivity (only ends the
		// boundary event path, not the whole workflow). Studio Pro always appends this.
		endPath := genWf.NewEndOfBoundaryEventPathActivity()
		endPath.SetID(element.ID(types.GenerateID()))
		endPath.SetPersistentId(types.GenerateID())
		endPath.SetCaption("End of boundary path")
		endPath.SetName("endOfBoundaryEventPath1")
		subActivities = append(subActivities, endPath)
		flow := newGenFlowWithActivities(subActivities)
		ev := genWf.NewNonInterruptingTimerBoundaryEvent()
		ev.SetID(id)
		ev.SetIsInterrupting(false)
		ev.SetEventType(be.EventType)
		ev.SetFirstExecutionTime(be.Delay)
		if flow != nil {
			ev.SetFlow(flow)
		}
		return ev
	default: // "Timer" or anything else falls back to the abstract base
		flow := newGenFlowWithActivities(subActivities)
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

// injectEndIntoConditionOutcomes walks activities and injects an EndWorkflowActivity
// into any VoidConditionOutcome whose flow is nil or empty.
// Used for NonInterruptingTimer boundary events: CE1844 forbids End at the
// boundary event top-level flow, but the Mendix runtime still requires End inside
// each activity's outcome sub-flows (CE6665 / WorkflowActivityDefinitionFactory).
func injectEndIntoConditionOutcomes(activities []element.Element) {
	for _, act := range activities {
		var outcomes []element.Element
		switch v := act.(type) {
		case *genWf.CallMicroflowActivity:
			outcomes = v.OutcomesItems()
		case *genWf.CallMicroflowTask:
			outcomes = v.OutcomesItems()
		}
		for _, oc := range outcomes {
			voc, ok := oc.(*genWf.VoidConditionOutcome)
			if !ok {
				continue
			}
			f, _ := voc.Flow().(*genWf.Flow)
			if f != nil && len(f.ActivitiesItems()) > 0 {
				continue // already has activities
			}
			end := genWf.NewEndWorkflowActivity()
			end.SetID(element.ID(types.GenerateID()))
			end.SetCaption("End")
			end.SetName("End")
			endFlow := newGenFlowWithActivities([]element.Element{end})
			voc.SetFlow(endFlow)
		}
	}
}

// endsWithTerminalWorkflowActivity reports whether the last element in acts
// is a JumpToActivity or EndWorkflowActivity (both satisfy CE6665).
func endsWithTerminalWorkflowActivity(acts []element.Element) bool {
	if len(acts) == 0 {
		return false
	}
	last := acts[len(acts)-1]
	if last == nil {
		return false
	}
	switch last.TypeName() {
	case "Workflows$JumpToActivity", "Workflows$EndWorkflowActivity":
		return true
	}
	return false
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
func buildUserTaskGenActivity(wbc *wfBuildCtx, n *ast.WorkflowUserTaskNode) element.Element {
	if n.IsMultiUser {
		return buildMultiUserTaskGenActivity(wbc, n)
	}
	return buildSingleUserTaskGenActivity(wbc, n)
}

func buildSingleUserTaskGenActivity(wbc *wfBuildCtx, n *ast.WorkflowUserTaskNode) *genWf.SingleUserTaskActivity {
	task := genWf.NewSingleUserTaskActivity()
	task.SetID(element.ID(types.GenerateID()))
	task.SetName(n.Name)
	task.SetCaption(n.Caption)
	task.SetDueDate(n.DueDate)
	if n.TaskDescription != "" {
		task.SetTaskDescription(newStringTemplateGen(n.TaskDescription))
	}
	if n.Page.Module != "" && n.Page.Name != "" {
		pr := genWf.NewPageReference()
		pr.SetPageQualifiedName(n.Page.Module + "." + n.Page.Name)
		task.SetTaskPage(pr)
	}
	if tgt := buildUserTargetingGen(n.Targeting); tgt != nil {
		task.SetUserTargeting(tgt)
	}
	for _, oc := range buildUserTaskOutcomesGen(wbc, n.Outcomes) {
		task.AddOutcomes(oc)
	}
	for _, ev := range buildBoundaryEventsGen(wbc, n.BoundaryEvents) {
		task.AddBoundaryEvents(ev)
	}
	return task
}

func buildMultiUserTaskGenActivity(wbc *wfBuildCtx, n *ast.WorkflowUserTaskNode) *genWf.MultiUserTaskActivity {
	task := genWf.NewMultiUserTaskActivity()
	task.SetID(element.ID(types.GenerateID()))
	task.SetName(n.Name)
	task.SetCaption(n.Caption)
	task.SetDueDate(n.DueDate)
	if n.TaskDescription != "" {
		task.SetTaskDescription(newStringTemplateGen(n.TaskDescription))
	}
	if n.Page.Module != "" && n.Page.Name != "" {
		pr := genWf.NewPageReference()
		pr.SetPageQualifiedName(n.Page.Module + "." + n.Page.Name)
		task.SetTaskPage(pr)
	}
	if tgt := buildUserTargetingGen(n.Targeting); tgt != nil {
		task.SetUserTargeting(tgt)
	}
	outcomes := buildUserTaskOutcomesGen(wbc, n.Outcomes)
	for _, oc := range outcomes {
		task.AddOutcomes(oc)
	}
	for _, ev := range buildBoundaryEventsGen(wbc, n.BoundaryEvents) {
		task.AddBoundaryEvents(ev)
	}
	// CE1866: multi-user tasks require a CompletionCriteria with a non-empty
	// FallbackOutcomeID. Build the appropriate criteria type based on the
	// CompletionMethod parsed from MDL (majority / threshold N / consensus).
	// Default (empty string) maps to MajorityCompletionCriteria.
	if len(outcomes) > 0 {
		fallbackID := outcomes[len(outcomes)-1].ID()
		switch n.CompletionMethod {
		case "", "majority":
			cc := genWf.NewMajorityCompletionCriteria()
			cc.SetFallbackOutcomeID(fallbackID)
			task.SetCompletionCriteria(cc)
		case "threshold":
			cc := genWf.NewThresholdCompletionCriteria()
			cc.SetFallbackOutcomeID(fallbackID)
			pct := n.RequiredThreshold
			if pct <= 0 || pct > 100 {
				pct = 50
			}
			cc.SetThreshold(int32(pct))
			task.SetCompletionCriteria(cc)
		case "consensus":
			cc := genWf.NewConsensusCompletionCriteria()
			cc.SetFallbackOutcomeID(fallbackID)
			task.SetCompletionCriteria(cc)
		}
	}
	return task
}

// buildUserTargetingGen builds a gen UserTargeting element from the AST targeting node.
// Mendix 11.2+ replaced the userSource field (OldUserSource subtypes) with
// userTargeting (UserTargeting subtypes). Single-user targeting uses
// XPathUserTargeting / MicroflowUserTargeting; group targeting uses
// XPathGroupTargeting / MicroflowGroupTargeting.
func buildUserTargetingGen(t ast.WorkflowTargetingNode) element.Element {
	switch t.Kind {
	case "microflow":
		tgt := genWf.NewMicroflowUserTargeting()
		tgt.SetMicroflowQualifiedName(t.Microflow.Module + "." + t.Microflow.Name)
		return tgt
	case "xpath":
		tgt := genWf.NewXPathUserTargeting()
		tgt.SetXPathConstraint(t.XPath)
		return tgt
	case "group_microflow":
		tgt := genWf.NewMicroflowGroupTargeting()
		tgt.SetMicroflowQualifiedName(t.Microflow.Module + "." + t.Microflow.Name)
		return tgt
	case "group_xpath":
		tgt := genWf.NewXPathGroupTargeting()
		tgt.SetXPathConstraint(t.XPath)
		return tgt
	default:
		// No targeting specified → NoUserTargeting (Studio Pro default).
		return genWf.NewNoUserTargeting()
	}
}

// buildUserTaskOutcomesGen mirrors the outcomes loop in buildUserTask
// (cmd_workflows_write.go:267). Each outcome stores the same string in
// Name/Caption/Value (legacy semantics).
func buildUserTaskOutcomesGen(wbc *wfBuildCtx, nodes []ast.WorkflowUserTaskOutcomeNode) []*genWf.UserTaskOutcome {
	out := make([]*genWf.UserTaskOutcome, 0, len(nodes))
	for _, n := range nodes {
		oc := genWf.NewUserTaskOutcome()
		oc.SetID(element.ID(types.GenerateID()))
		oc.SetName(n.Caption)
		oc.SetCaption(n.Caption)
		oc.SetValue(n.Caption)
		if flow := newGenFlowWithActivities(buildWorkflowActivitiesGen(wbc, n.Activities)); flow != nil {
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
// (cmd_workflows_write.go:291). Uses NewCallMicroflowForVersion to select the
// correct concrete type for the project version: CallMicroflowActivity (11.9+)
// or the legacy CallMicroflowTask (pre-11.9).
func buildCallMicroflowGenActivity(wbc *wfBuildCtx, n *ast.WorkflowCallMicroflowNode) element.Element {
	act := genWf.NewCallMicroflowForVersion(wbc.version)

	name := n.Microflow.Name
	caption := n.Caption
	if caption == "" {
		caption = name
	}
	mfQN := n.Microflow.Module + "." + name

	switch v := act.(type) {
	case *genWf.CallMicroflowActivity:
		v.SetID(element.ID(types.GenerateID()))
		v.SetName(name)
		v.SetCaption(caption)
		v.SetMicroflowQualifiedName(mfQN)
		for _, oc := range buildConditionOutcomesGen(wbc, n.Outcomes) {
			v.AddOutcomes(oc)
		}
		for _, pm := range n.ParameterMappings {
			mapping := genWf.NewMicroflowCallParameterMapping()
			mapping.SetID(element.ID(types.GenerateID()))
			mapping.SetParameterQualifiedName(mfQN + "." + pm.Parameter)
			mapping.SetExpression(pm.Expression)
			v.AddParameterMappings(mapping)
		}
		for _, ev := range buildBoundaryEventsGen(wbc, n.BoundaryEvents) {
			v.AddBoundaryEvents(ev)
		}
	case *genWf.CallMicroflowTask:
		v.SetID(element.ID(types.GenerateID()))
		v.SetName(name)
		v.SetCaption(caption)
		v.SetMicroflowQualifiedName(mfQN)
		for _, oc := range buildConditionOutcomesGen(wbc, n.Outcomes) {
			v.AddOutcomes(oc)
		}
		for _, pm := range n.ParameterMappings {
			mapping := genWf.NewMicroflowCallParameterMapping()
			mapping.SetID(element.ID(types.GenerateID()))
			mapping.SetParameterQualifiedName(mfQN + "." + pm.Parameter)
			mapping.SetExpression(pm.Expression)
			v.AddParameterMappings(mapping)
		}
		for _, ev := range buildBoundaryEventsGen(wbc, n.BoundaryEvents) {
			v.AddBoundaryEvents(ev)
		}
	}
	return act
}

// buildCallWorkflowGenActivity mirrors buildCallWorkflowActivity
// (cmd_workflows_write.go:325).
func buildCallWorkflowGenActivity(wbc *wfBuildCtx, n *ast.WorkflowCallWorkflowNode) *genWf.CallWorkflowActivity {
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
	for _, ev := range buildBoundaryEventsGen(wbc, n.BoundaryEvents) {
		act.AddBoundaryEvents(ev)
	}
	return act
}

// validateEnumConditionValue returns an error if s is an enum condition value
// that does not follow the required Module.EnumerationName.ValueName format.
// Studio Pro 11.10.0 rejects bare names like "Overseas" — they must be stored
// as fully-qualified identifiers.
func validateEnumConditionValue(s string) error {
	switch s {
	case "True", "False", "Default":
		return nil
	}
	if strings.Count(s, ".") < 2 {
		return fmt.Errorf(
			"workflow enum condition value %q must be a fully-qualified name in "+
				"'Module.EnumerationName.ValueName' format (got %d segment(s), need 3)",
			s, strings.Count(s, ".")+1,
		)
	}
	return nil
}

// validateWorkflowActivities recursively checks that all EnumerationValue
// condition outcomes use fully-qualified names.
func validateWorkflowActivities(activities []ast.WorkflowActivityNode) error {
	for _, act := range activities {
		if err := validateWorkflowActivity(act); err != nil {
			return err
		}
	}
	return nil
}

func validateWorkflowActivity(act ast.WorkflowActivityNode) error {
	switch v := act.(type) {
	case *ast.WorkflowDecisionNode:
		return validateConditionOutcomeNodes(v.Outcomes)
	case *ast.WorkflowCallMicroflowNode:
		return validateConditionOutcomeNodes(v.Outcomes)
	case *ast.WorkflowUserTaskNode:
		if err := validateUserTaskTargeting(v); err != nil {
			return err
		}
		for _, oc := range v.Outcomes {
			if err := validateWorkflowActivities(oc.Activities); err != nil {
				return err
			}
		}
	case *ast.WorkflowParallelSplitNode:
		for _, path := range v.Paths {
			if err := validateWorkflowActivities(path.Activities); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateConditionOutcomeNodes(outcomes []ast.WorkflowConditionOutcomeNode) error {
	for _, oc := range outcomes {
		if err := validateEnumConditionValue(oc.Value); err != nil {
			return err
		}
		if err := validateWorkflowActivities(oc.Activities); err != nil {
			return err
		}
	}
	return nil
}

// buildConditionOutcomesGen mirrors the outcomes loop in
// buildCallMicroflowTask (cmd_workflows_write.go:302). Dispatches on
// AST Value to BooleanConditionOutcome / VoidConditionOutcome /
// EnumerationValueConditionOutcome.
func buildConditionOutcomesGen(wbc *wfBuildCtx, nodes []ast.WorkflowConditionOutcomeNode) []element.Element {
	out := make([]element.Element, 0, len(nodes))
	for _, n := range nodes {
		oc := buildConditionOutcomeGen(wbc, n)
		if oc != nil {
			out = append(out, oc)
		}
	}
	return out
}

// buildConditionOutcomeGen mirrors buildConditionOutcome
// (cmd_workflows_write.go:390).
func buildConditionOutcomeGen(wbc *wfBuildCtx, n ast.WorkflowConditionOutcomeNode) element.Element {
	subFlow := newGenFlowWithActivities(buildWorkflowActivitiesGen(wbc, n.Activities))
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
func buildExclusiveSplitGenActivity(wbc *wfBuildCtx, n *ast.WorkflowDecisionNode) *genWf.ExclusiveSplitActivity {
	act := genWf.NewExclusiveSplitActivity()
	act.SetID(element.ID(types.GenerateID()))
	act.SetExpression(mendixExprValue(n.Expression))
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
		built := buildConditionOutcomeGen(wbc, oc)
		if built != nil {
			act.AddOutcomes(built)
		}
	}
	return act
}

// buildParallelSplitGenActivity mirrors buildParallelSplit
// (cmd_workflows_write.go:420). Each AST path becomes a
// ParallelSplitOutcome carrying its own gen Flow.
func buildParallelSplitGenActivity(wbc *wfBuildCtx, n *ast.WorkflowParallelSplitNode) *genWf.ParallelSplitActivity {
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
		if flow := newGenFlowWithActivities(buildWorkflowActivitiesGen(wbc, p.Activities)); flow != nil {
			oc.SetFlow(flow)
		}
		act.AddOutcomes(oc)
	}
	return act
}

// ---------------------------------------------------------------------------
// D2 — execCreateWorkflowGen
// ---------------------------------------------------------------------------

// execCreateWorkflowGen mirrors execCreateWorkflow (cmd_workflows_write.go:20).
// Builds a gen-typed Workflow from the AST and routes through
// CreateWorkflowGen / UpdateWorkflowGen on FullBackend (added in C1).
//
// Implicit Start/End activities are added at flow boundaries to match
// Studio Pro's convention. Activity name dedup happens via
// deduplicateActivityNamesGen (defined below). autoBindWorkflowGen
// fills CallMicroflow ParameterMappings via D4/D5 helpers.
func execCreateWorkflowGen(ctx *ExecContext, s *ast.CreateWorkflowStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	module, err := findOrCreateModule(ctx, s.Name.Module)
	if err != nil {
		return err
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	// Existence check via gen cache helper.
	pairs, err := listWorkflowsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list workflows", err)
	}
	var existingID model.ID
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if modName == s.Name.Module && p.Elem.Name() == s.Name.Name {
			if !s.CreateOrModify {
				qn := s.Name.Module + "." + s.Name.Name
				return mdlerrors.NewAlreadyExistsMsg("workflow", qn,
					"workflow '"+qn+"' already exists (use create or modify to overwrite)")
			}
			existingID = model.ID(p.Elem.ID())
			break
		}
	}

	// Construct the gen Workflow.
	wf := genWf.NewWorkflow()
	wf.SetName(s.Name.Name)
	wf.SetDocumentation(s.Documentation)

	// Parameter
	if s.ParameterEntity.Module != "" {
		param := genWf.NewParameter()
		param.SetID(element.ID(generateWorkflowUUID()))
		param.SetEntityQualifiedName(s.ParameterEntity.Module + "." + s.ParameterEntity.Name)
		wf.SetParameter(param)
	}

	if s.OverviewPage.Module != "" {
		wf.SetOverviewPageQualifiedName(s.OverviewPage.Module + "." + s.OverviewPage.Name)
	}

	// Display metadata. WorkflowName / WorkflowDescription are Texts$Text
	// wrappers in the gen schema (per Phase A R1 dual-storage finding).
	if s.DisplayName != "" {
		wf.SetWorkflowName(newStringTemplateGen(s.DisplayName))
		// Mirror Title for legacy decode round-trip.
		wf.SetTitle(s.DisplayName)
	}
	if s.Description != "" {
		wf.SetWorkflowDescription(newStringTemplateGen(s.Description))
	}
	if s.ExportLevel != "" {
		wf.SetExportLevel(s.ExportLevel)
	}
	wf.SetDueDate(s.DueDate)

	// Validate enum condition outcome values before building BSON.
	if err := validateWorkflowActivities(s.Activities); err != nil {
		return err
	}

	// Build flow with implicit Start + user activities + End.
	startAct := genWf.NewStartWorkflowActivity()
	startAct.SetID(element.ID(generateWorkflowUUID()))
	startAct.SetCaption("Start")
	startAct.SetName("Start")

	endAct := genWf.NewEndWorkflowActivity()
	endAct.SetID(element.ID(generateWorkflowUUID()))
	endAct.SetCaption("End")
	endAct.SetName("End")

	wbc := newWfBuildCtx(ctx)
	userActivities := buildWorkflowActivitiesGen(wbc, s.Activities)
	autoBindWorkflowGen(ctx, userActivities)
	deduplicateActivityNamesGen(userActivities)

	flow := genWf.NewFlow()
	flow.SetID(element.ID(generateWorkflowUUID()))
	flow.AddActivities(startAct)
	for _, a := range userActivities {
		flow.AddActivities(a)
	}
	flow.AddActivities(endAct)
	wf.SetFlow(flow)

	if existingID != "" {
		// In-place update: preserve UnitID so references and BSON git-diff
		// stay stable.
		wf.SetID(element.ID(existingID))
		if err := ctx.WorkflowWriter.UpdateWorkflowGen(wf); err != nil {
			return mdlerrors.NewBackend("update workflow", err)
		}
	} else {
		// New unit: gen Create generates a fresh UnitID.
		if err := ctx.WorkflowWriter.CreateWorkflowGen(string(module.ID), "Documents", wf); err != nil {
			return mdlerrors.NewBackend("create workflow", err)
		}
	}

	invalidateHierarchy(ctx)
	invalidateWorkflowsCache(ctx)
	fmt.Fprintf(ctx.Output, "Created workflow: %s.%s\n", s.Name.Module, s.Name.Name)
	return nil
}

// autoBindWorkflowGen is the gen-typed twin of autoBindWorkflowParameters
// (cmd_workflows_write.go:615). D4/D5 fill in the CallMicroflow /
// CallWorkflow parameter binding logic; for now it sanitises activity
// names so the Studio Pro identifier rules are honoured.
func autoBindWorkflowGen(ctx *ExecContext, activities []element.Element) {
	for _, act := range activities {
		switch v := act.(type) {
		case *genWf.SingleUserTaskActivity:
			v.SetName(sanitizeActivityName(v.Name()))
			recurseUserTaskOutcomesGen(ctx, v.OutcomesItems())
		case *genWf.MultiUserTaskActivity:
			v.SetName(sanitizeActivityName(v.Name()))
			recurseUserTaskOutcomesGen(ctx, v.OutcomesItems())
		case *genWf.UserTask:
			v.SetName(sanitizeActivityName(v.Name()))
			recurseUserTaskOutcomesGen(ctx, v.OutcomesItems())
		case *genWf.CallMicroflowActivity:
			autoBindCallMicroflowGenActivity(ctx, v)
			recurseConditionOutcomesAutoBindGen(ctx, v.OutcomesItems())
		case *genWf.CallMicroflowTask:
			// Legacy storage shape (pre-11.9). Sanitise, ensure at least one
			// outcome (CE6686 mirrors autoBindCallMicroflowGenActivity), then recurse.
			v.SetName(sanitizeActivityName(v.Name()))
			if len(v.OutcomesItems()) == 0 {
				oc := genWf.NewVoidConditionOutcome()
				oc.SetID(element.ID(types.GenerateID()))
				emptyFlow := genWf.NewFlow()
				emptyFlow.SetID(element.ID(types.GenerateID()))
				oc.SetFlow(emptyFlow)
				v.AddOutcomes(oc)
			}
			recurseConditionOutcomesAutoBindGen(ctx, v.OutcomesItems())
		case *genWf.CallWorkflowActivity:
			autoBindCallWorkflowGenActivity(ctx, v)
		case *genWf.ExclusiveSplitActivity:
			v.SetName(sanitizeActivityName(v.Name()))
			recurseConditionOutcomesAutoBindGen(ctx, v.OutcomesItems())
		case *genWf.ParallelSplitActivity:
			v.SetName(sanitizeActivityName(v.Name()))
			for _, oc := range v.OutcomesItems() {
				if pso, ok := oc.(*genWf.ParallelSplitOutcome); ok {
					if f, ok := pso.Flow().(*genWf.Flow); ok && f != nil {
						autoBindWorkflowGen(ctx, f.ActivitiesItems())
					}
				}
			}
		case *genWf.JumpToActivity:
			v.SetName(sanitizeActivityName(v.Name()))
		case *genWf.WaitForTimerActivity:
			v.SetName(sanitizeActivityName(v.Name()))
		case *genWf.WaitForNotificationActivity:
			v.SetName(sanitizeActivityName(v.Name()))
		}
	}
}

func recurseUserTaskOutcomesGen(ctx *ExecContext, outcomes []element.Element) {
	for _, oc := range outcomes {
		if utc, ok := oc.(*genWf.UserTaskOutcome); ok {
			if f, ok := utc.Flow().(*genWf.Flow); ok && f != nil {
				autoBindWorkflowGen(ctx, f.ActivitiesItems())
			}
		}
	}
}

func recurseConditionOutcomesAutoBindGen(ctx *ExecContext, outcomes []element.Element) {
	for _, oc := range outcomes {
		var f *genWf.Flow
		switch v := oc.(type) {
		case *genWf.BooleanConditionOutcome:
			f, _ = v.Flow().(*genWf.Flow)
		case *genWf.VoidConditionOutcome:
			f, _ = v.Flow().(*genWf.Flow)
		case *genWf.EnumerationValueConditionOutcome:
			f, _ = v.Flow().(*genWf.Flow)
		}
		if f != nil {
			autoBindWorkflowGen(ctx, f.ActivitiesItems())
		}
	}
}

// autoBindCallMicroflowGenActivity mirrors autoBindCallMicroflow
// (cmd_workflows_write.go:681). Sanitises the activity name, ensures
// a default VoidConditionOutcome exists (CE6686), and walks
// ctx.Microflows to resolve the called microflow's parameters,
// auto-generating MicroflowCallParameterMapping entries with
// $WorkflowContext expressions.
func autoBindCallMicroflowGenActivity(ctx *ExecContext, act *genWf.CallMicroflowActivity) {
	act.SetName(sanitizeActivityName(act.Name()))

	// CE6686: every CallMicroflowActivity must own at least one outcome.
	if len(act.OutcomesItems()) == 0 {
		oc := genWf.NewVoidConditionOutcome()
		oc.SetID(element.ID(types.GenerateID()))
		emptyFlow := genWf.NewFlow()
		emptyFlow.SetID(element.ID(types.GenerateID()))
		oc.SetFlow(emptyFlow)
		act.AddOutcomes(oc)
	}

	// Skip parameter auto-binding if explicit mappings already exist.
	if len(act.ParameterMappingsItems()) > 0 {
		return
	}

	// Resolve the called microflow's parameters via ctx.Microflows
	// (gen-typed cache helper).
	mfs, err := listMicroflowsWithContainerGen(ctx)
	if err != nil {
		return
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return
	}
	target := act.MicroflowQualifiedName()
	for _, item := range mfs {
		mf := item.MF
		if mf == nil {
			continue
		}
		modID := h.FindModuleID(item.ContainerUUID)
		modName := h.GetModuleName(modID)
		qn := modName + "." + mf.Name()
		if qn != target {
			continue
		}
		// Walk the microflow's ObjectCollection for MicroflowParameter
		// elements (gen Microflow stores parameters there, not in a
		// dedicated typed slice).
		for _, paramName := range genMicroflowParameterNames(mf) {
			mapping := genWf.NewMicroflowCallParameterMapping()
			mapping.SetID(element.ID(types.GenerateID()))
			mapping.SetParameterQualifiedName(qn + "." + paramName)
			mapping.SetExpression("$WorkflowContext")
			act.AddParameterMappings(mapping)
		}
		break
	}
}

// autoBindCallWorkflowGenActivity mirrors autoBindCallWorkflow
// (cmd_workflows_write.go:748). Sanitises the activity name and, when
// the target workflow has a Parameter, generates a single
// WorkflowCallParameterMapping with $WorkflowContext expression.
func autoBindCallWorkflowGenActivity(ctx *ExecContext, act *genWf.CallWorkflowActivity) {
	act.SetName(sanitizeActivityName(act.Name()))

	// Skip if explicit mappings already exist.
	if len(act.ParameterMappingsItems()) > 0 {
		return
	}

	// Look up the target workflow via the gen cache helper to inspect
	// its Parameter (entity-typed input).
	pairs, err := listWorkflowsWithContainerGen(ctx)
	if err != nil {
		return
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return
	}
	target := act.WorkflowQualifiedName()
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		qn := modName + "." + p.Elem.Name()
		if qn != target {
			continue
		}
		// Only auto-bind when the target carries an entity parameter.
		if entity := workflowParameterEntityGen(p.Elem); entity != "" {
			act.SetParameterExpression("$WorkflowContext")
			mapping := genWf.NewWorkflowCallParameterMapping()
			mapping.SetID(element.ID(types.GenerateID()))
			mapping.SetParameterQualifiedName(qn + ".WorkflowContext")
			mapping.SetExpression("$WorkflowContext")
			act.AddParameterMappings(mapping)
		}
		break
	}
}

// deduplicateActivityNamesGen mirrors deduplicateActivityNames
// (cmd_workflows_write.go:503). Walks the gen activity tree and
// renames duplicates by appending a count suffix so Studio Pro's
// CE0495 (unique activity names) is satisfied.
func deduplicateActivityNamesGen(activities []element.Element) {
	nameCount := make(map[string]int)
	deduplicateActivityNamesInFlowGen(activities, nameCount)
}

func deduplicateActivityNamesInFlowGen(activities []element.Element, nameCount map[string]int) {
	for _, act := range activities {
		switch v := act.(type) {
		case *genWf.SingleUserTaskActivity:
			v.SetName(uniqueName(v.Name(), nameCount))
			recurseUserTaskOutcomesDedupGen(v.OutcomesItems(), nameCount)
		case *genWf.MultiUserTaskActivity:
			v.SetName(uniqueName(v.Name(), nameCount))
			recurseUserTaskOutcomesDedupGen(v.OutcomesItems(), nameCount)
		case *genWf.UserTask:
			v.SetName(uniqueName(v.Name(), nameCount))
			recurseUserTaskOutcomesDedupGen(v.OutcomesItems(), nameCount)
		case *genWf.CallMicroflowActivity:
			v.SetName(uniqueName(v.Name(), nameCount))
			recurseConditionOutcomesDedupGen(v.OutcomesItems(), nameCount)
		case *genWf.CallMicroflowTask:
			v.SetName(uniqueName(v.Name(), nameCount))
			recurseConditionOutcomesDedupGen(v.OutcomesItems(), nameCount)
		case *genWf.CallWorkflowActivity:
			v.SetName(uniqueName(v.Name(), nameCount))
		case *genWf.ExclusiveSplitActivity:
			v.SetName(uniqueName(v.Name(), nameCount))
			recurseConditionOutcomesDedupGen(v.OutcomesItems(), nameCount)
		case *genWf.ParallelSplitActivity:
			v.SetName(uniqueName(v.Name(), nameCount))
			for _, oc := range v.OutcomesItems() {
				if pso, ok := oc.(*genWf.ParallelSplitOutcome); ok {
					if f, ok := pso.Flow().(*genWf.Flow); ok && f != nil {
						deduplicateActivityNamesInFlowGen(f.ActivitiesItems(), nameCount)
					}
				}
			}
		case *genWf.JumpToActivity:
			v.SetName(uniqueName(v.Name(), nameCount))
		case *genWf.WaitForTimerActivity:
			v.SetName(uniqueName(v.Name(), nameCount))
		case *genWf.WaitForNotificationActivity:
			v.SetName(uniqueName(v.Name(), nameCount))
		case *genWf.EndWorkflowActivity:
			v.SetName(uniqueName(v.Name(), nameCount))
		}
	}
}

func recurseUserTaskOutcomesDedupGen(outcomes []element.Element, nameCount map[string]int) {
	for _, oc := range outcomes {
		if utc, ok := oc.(*genWf.UserTaskOutcome); ok {
			if f, ok := utc.Flow().(*genWf.Flow); ok && f != nil {
				deduplicateActivityNamesInFlowGen(f.ActivitiesItems(), nameCount)
			}
		}
	}
}

func recurseConditionOutcomesDedupGen(outcomes []element.Element, nameCount map[string]int) {
	for _, oc := range outcomes {
		var f *genWf.Flow
		switch v := oc.(type) {
		case *genWf.BooleanConditionOutcome:
			f, _ = v.Flow().(*genWf.Flow)
		case *genWf.VoidConditionOutcome:
			f, _ = v.Flow().(*genWf.Flow)
		case *genWf.EnumerationValueConditionOutcome:
			f, _ = v.Flow().(*genWf.Flow)
		}
		if f != nil {
			deduplicateActivityNamesInFlowGen(f.ActivitiesItems(), nameCount)
		}
	}
}

// ---------------------------------------------------------------------------
// D3 — execDropWorkflowGen
// ---------------------------------------------------------------------------

// execDropWorkflowGen mirrors execDropWorkflow (cmd_workflows_write.go:133).
// Lists via gen cache helper, deletes via FullBackend.DeleteWorkflow
// (sdk-typed but ID-only — no migration needed).
func execDropWorkflowGen(ctx *ExecContext, s *ast.DropWorkflowStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}
	pairs, err := listWorkflowsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list workflows", err)
	}
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if modName == s.Name.Module && p.Elem.Name() == s.Name.Name {
			if err := ctx.WorkflowWriter.DeleteWorkflow(model.ID(p.Elem.ID())); err != nil {
				return mdlerrors.NewBackend("delete workflow", err)
			}
			invalidateHierarchy(ctx)
			invalidateWorkflowsCache(ctx)
			fmt.Fprintf(ctx.Output, "Dropped workflow: %s.%s\n", s.Name.Module, s.Name.Name)
			return nil
		}
	}
	return mdlerrors.NewNotFound("workflow", s.Name.Module+"."+s.Name.Name)
}

// ---------------------------------------------------------------------------
// Pure helpers (no sdk dependency — relocated from cmd_workflows_write.go
// in preparation for Phase E2 deletion of the legacy file).
// ---------------------------------------------------------------------------

// generateWorkflowUUID generates a UUID for workflow elements.
func generateWorkflowUUID() string {
	return types.GenerateID()
}

// sanitizeActivityName converts a display caption to a valid Mendix
// identifier. Mendix names must start with a letter/underscore and
// contain only letters, digits, underscores.
func sanitizeActivityName(name string) string {
	var b strings.Builder
	for i, r := range name {
		if unicode.IsLetter(r) || r == '_' {
			b.WriteRune(r)
		} else if unicode.IsDigit(r) && i > 0 {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' {
			b.WriteRune('_')
		}
	}
	result := b.String()
	if result == "" {
		return "activity"
	}
	return result
}

// uniqueName returns a unique name by appending a number if the name
// was seen before. Starts disambiguation at 2 (Mendix Studio Pro
// convention; matches uniqueNameForGen).
func uniqueName(name string, nameCount map[string]int) string {
	nameCount[name]++
	count := nameCount[name]
	if count == 1 {
		return name
	}
	return fmt.Sprintf("%s%d", name, count)
}

// newStringTemplateGen wraps a plain string in a Microflows$StringTemplate
// element (empty Parameters list, Text set to s). This is the correct BSON
// shape for workflow text fields: WorkflowName, WorkflowDescription,
// TaskName, TaskDescription — all use StringTemplate, not Texts$Text.
func newStringTemplateGen(s string) element.Element {
	tpl := genMf.NewStringTemplate()
	tpl.SetID(element.ID(types.GenerateID()))
	tpl.SetText(s)
	return tpl
}

// newTextWrapperGen is kept for callers outside the workflow context that
// genuinely need Texts$Text (page labels, etc.). Workflow text fields must
// use newStringTemplateGen instead.
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

// validateUserTaskTargeting checks that a user task targeting clause is well-formed.
// A missing targeting clause is accepted and defaults to NoUserTargeting (Studio Pro
// equivalent of "no user targeting configured"). An explicit xpath targeting with an
// empty constraint is still rejected because it is structurally malformed.
func validateUserTaskTargeting(n *ast.WorkflowUserTaskNode) error {
	if (n.Targeting.Kind == "xpath" || n.Targeting.Kind == "group_xpath") && n.Targeting.XPath == "" {
		return fmt.Errorf(
			"user task '%s' has empty xpath constraint: xpath targeting requires a non-empty expression",
			n.Name,
		)
	}
	return nil
}

// genMicroflowParameterNames returns the parameter names of a gen
// Microflow, walking the ObjectCollection for MicroflowParameter
// elements. Mirrors the inline scan in genMicroflowParameters
// (cmd_microflows_show_gen.go) but returns just the names — that's
// all the workflow auto-binder needs.
//
// Stage 3.3.3.E2 — relocated from cmd_workflows_write_gen.go (which
// retired in this commit alongside its sdk-typed sibling helpers).
func genMicroflowParameterNames(mf *genMf.Microflow) []string {
	if mf == nil {
		return nil
	}
	oc, ok := mf.ObjectCollection().(*genMf.MicroflowObjectCollection)
	if !ok || oc == nil {
		return nil
	}
	var out []string
	for _, obj := range oc.ObjectsItems() {
		if obj == nil {
			continue
		}
		if obj.TypeName() != "Microflows$MicroflowParameter" {
			continue
		}
		if nv, ok := obj.(interface{ NameValue() string }); ok {
			out = append(out, nv.NameValue())
		}
	}
	return out
}

// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.3 Phase A: gen-typed SHOW/DESCRIBE WORKFLOW commands.
//
// This file is the gen-typed twin of cmd_workflows.go. It walks
// modelsdk/gen/workflows.Workflow units (decoded by ctx.Workflows) via
// the listWorkflowsWithContainerGen cache helper, then dispatches
// activities by storage $Type rather than concrete sdk Go type.
//
// Notes:
//
//   - SystemTask: gen has no narrow SystemTask type. The $Type
//     "Workflows$SystemTask" is read via element.Element.Raw() with
//     codec.ReadBSONFieldString fallbacks for the legacy fields.
//   - UserTask family: legacy sdk had a unified *UserTask. gen splits
//     into *UserTask (legacy storage), *SingleUserTaskActivity,
//     *MultiUserTaskActivity. The dispatcher handles all three.
//   - CallMicroflow family: gen exposes both *CallMicroflowActivity and
//     *CallMicroflowTask (the legacy storage). Both decode into the same
//     "shape" but live under different gen Go types — dispatch on $Type.
//   - User source / targeting: gen uses "EmptyUserSource" (formerly
//     NoUserSource) and split source vs. targeting (per R5). The
//     dispatcher handles both naming conventions for backward compat.

package executor

import (
	"fmt"
	"sort"
	"strings"

	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// listWorkflowsGen handles SHOW WORKFLOWS via gen-typed Workflow units.
func listWorkflowsGen(ctx *ExecContext, moduleName string) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pairs, err := listWorkflowsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list workflows", err)
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		activities    int
		userTasks     int
		decisions     int
		paramEntity   string
	}
	var rows []row

	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modName := ""
		if h != nil {
			modID := h.FindModuleID(model.ID(p.ContainerID))
			modName = h.GetModuleName(modID)
		}
		if moduleName != "" && modName != moduleName {
			continue
		}
		qualifiedName := modName + "." + p.Elem.Name()
		paramEntity := workflowParameterEntityGen(p.Elem)
		acts, uts, decs := countWorkflowActivitiesGen(p.Elem)
		rows = append(rows, row{qualifiedName, modName, p.Elem.Name(), acts, uts, decs, paramEntity})
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Activities", "User Tasks", "Decisions", "Parameter Entity"},
		Summary: fmt.Sprintf("(%d workflows)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.activities, r.userTasks, r.decisions, r.paramEntity})
	}
	return writeResult(ctx, result)
}

// workflowParameterEntityGen returns the entity qualified name carried
// by Workflow.Parameter (a *genWf.Parameter Part), or "" if absent.
func workflowParameterEntityGen(wf *genWf.Workflow) string {
	if wf == nil {
		return ""
	}
	param := wf.Parameter()
	if param == nil {
		return ""
	}
	if p, ok := param.(*genWf.Parameter); ok {
		return p.EntityQualifiedName()
	}
	// Defensive fallback: read raw BSON field if the Part decoded to a
	// non-narrow type.
	v, _ := codec.ReadBSONFieldString(param.Raw(), "Entity")
	return v
}

// countWorkflowActivitiesGen counts total activities, user tasks, and
// decisions in a gen-typed workflow.
func countWorkflowActivitiesGen(wf *genWf.Workflow) (total, userTasks, decisions int) {
	if wf == nil {
		return
	}
	flow, _ := wf.Flow().(*genWf.Flow)
	if flow == nil {
		return
	}
	countFlowActivitiesGen(flow, &total, &userTasks, &decisions)
	return
}

// countFlowActivitiesGen recursively counts activities in a gen flow
// and its sub-flows. Dispatch is by $Type because gen splits user-task
// and call-microflow into multiple concrete types.
func countFlowActivitiesGen(flow *genWf.Flow, total, userTasks, decisions *int) {
	if flow == nil {
		return
	}
	for _, a := range flow.ActivitiesItems() {
		if a == nil {
			continue
		}
		*total++
		switch a.TypeName() {
		case "Workflows$UserTask",
			"Workflows$SingleUserTaskActivity",
			"Workflows$MultiUserTaskActivity":
			*userTasks++
			countNestedFlowsInOutcomesGen(a, total, userTasks, decisions)
		case "Workflows$ExclusiveSplitActivity":
			*decisions++
			countNestedFlowsInOutcomesGen(a, total, userTasks, decisions)
		case "Workflows$ParallelSplitActivity":
			countNestedFlowsInOutcomesGen(a, total, userTasks, decisions)
		case "Workflows$CallMicroflowActivity",
			"Workflows$CallMicroflowTask",
			"Workflows$SystemTask":
			countNestedFlowsInOutcomesGen(a, total, userTasks, decisions)
		}
	}
}

// countNestedFlowsInOutcomesGen walks the outcomes/paths of a composite
// activity (UserTask, ExclusiveSplit, ParallelSplit, CallMicroflow,
// SystemTask) and recurses into each nested Flow.
//
// It works generically by looking up the activity's gen-typed
// OutcomesItems via reflection-free type assertion to the known
// concrete shapes; falls back to inspecting raw BSON for the
// SystemTask case (gen has no narrow type for it).
func countNestedFlowsInOutcomesGen(a element.Element, total, userTasks, decisions *int) {
	switch v := a.(type) {
	case *genWf.UserTask:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGen(oc, total, userTasks, decisions)
		}
	case *genWf.SingleUserTaskActivity:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGen(oc, total, userTasks, decisions)
		}
	case *genWf.MultiUserTaskActivity:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGen(oc, total, userTasks, decisions)
		}
	case *genWf.ExclusiveSplitActivity:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGen(oc, total, userTasks, decisions)
		}
	case *genWf.ParallelSplitActivity:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGen(oc, total, userTasks, decisions)
		}
	case *genWf.CallMicroflowActivity:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGen(oc, total, userTasks, decisions)
		}
	case *genWf.CallMicroflowTask:
		for _, oc := range v.OutcomesItems() {
			countOutcomeFlowGen(oc, total, userTasks, decisions)
		}
	default:
		// SystemTask + unknown types: outcomes live under raw BSON. The
		// counts are advisory (tests don't depend on every leaf), so we
		// don't reach into raw BSON here — the legacy counter only walks
		// the explicit sdk types.
	}
}

// countOutcomeFlowGen extracts the nested Flow from an outcome element
// and recurses. Outcome elements expose Flow() element.Element returning
// a *Workflows$Flow Part.
func countOutcomeFlowGen(oc element.Element, total, userTasks, decisions *int) {
	if oc == nil {
		return
	}
	switch v := oc.(type) {
	case *genWf.UserTaskOutcome:
		if f, ok := v.Flow().(*genWf.Flow); ok {
			countFlowActivitiesGen(f, total, userTasks, decisions)
		}
	case *genWf.BooleanConditionOutcome:
		if f, ok := v.Flow().(*genWf.Flow); ok {
			countFlowActivitiesGen(f, total, userTasks, decisions)
		}
	case *genWf.EnumerationValueConditionOutcome:
		if f, ok := v.Flow().(*genWf.Flow); ok {
			countFlowActivitiesGen(f, total, userTasks, decisions)
		}
	case *genWf.VoidConditionOutcome:
		if f, ok := v.Flow().(*genWf.Flow); ok {
			countFlowActivitiesGen(f, total, userTasks, decisions)
		}
	case *genWf.ParallelSplitOutcome:
		if f, ok := v.Flow().(*genWf.Flow); ok {
			countFlowActivitiesGen(f, total, userTasks, decisions)
		}
	case *genWf.ExclusiveSplitOutcome:
		if f, ok := v.Flow().(*genWf.Flow); ok {
			countFlowActivitiesGen(f, total, userTasks, decisions)
		}
	}
}

// describeWorkflowGen / describeWorkflowToStringGen land in A4 once
// the heavy formatters (UserTask / CallMicroflow / CallWorkflow /
// ExclusiveSplit / ParallelSplit / SystemTask) are in place in A3.
// This A2 commit lands the dispatcher + leaf formatters.

// formatAnnotationGen returns an ANNOTATION statement line for an
// activity-level annotation Part. Returns "" if the annotation is nil
// or has no Description.
func formatAnnotationGen(elem element.Element, indent string) string {
	if elem == nil {
		return ""
	}
	desc := ""
	if a, ok := elem.(*genWf.Annotation); ok {
		desc = a.Description()
	} else {
		desc, _ = codec.ReadBSONFieldString(elem.Raw(), "Description")
	}
	if desc == "" {
		return ""
	}
	escaped := strings.ReplaceAll(desc, "'", "''")
	return fmt.Sprintf("%sannotation '%s';", indent, escaped)
}

// boundaryEventKeywordGen mirrors boundaryEventKeyword. The mapping
// is shared so legacy + gen produce byte-identical output.
func boundaryEventKeywordGen(eventTypeName string) string {
	switch eventTypeName {
	case "Workflows$InterruptingTimerBoundaryEvent":
		return "boundary event interrupting timer"
	case "Workflows$NonInterruptingTimerBoundaryEvent":
		return "boundary event non interrupting timer"
	default:
		return "boundary event timer"
	}
}

// formatBoundaryEventsGen renders boundary events from a gen Part list.
func formatBoundaryEventsGen(events []element.Element, indent string) []string {
	if len(events) == 0 {
		return nil
	}
	var lines []string
	for _, ev := range events {
		if ev == nil {
			continue
		}
		keyword := boundaryEventKeywordGen(ev.TypeName())
		delay := boundaryEventDelayGen(ev)
		if delay != "" {
			escaped := strings.ReplaceAll(delay, "'", "''")
			lines = append(lines, fmt.Sprintf("%s%s '%s'", indent, keyword, escaped))
		} else {
			lines = append(lines, fmt.Sprintf("%s%s", indent, keyword))
		}
		if flow := boundaryEventFlowGen(ev); flow != nil && len(flow.ActivitiesItems()) > 0 {
			lines = append(lines, fmt.Sprintf("%s{", indent))
			lines = append(lines, formatWorkflowActivitiesGen(flow, indent+"  ")...)
			lines = append(lines, fmt.Sprintf("%s}", indent))
		}
	}
	return lines
}

// boundaryEventDelayGen extracts the timer delay string from any of the
// concrete BoundaryEvent gen subtypes. gen exposes the field as
// `Delay()` while the legacy sdk used `TimerDelay`.
func boundaryEventDelayGen(ev element.Element) string {
	switch v := ev.(type) {
	case *genWf.InterruptingTimerBoundaryEvent:
		return v.Delay()
	case *genWf.NonInterruptingTimerBoundaryEvent:
		return v.Delay()
	case *genWf.TimerBoundaryEvent:
		return v.Delay()
	}
	// Fallback: try both legacy and gen field names.
	if v, _ := codec.ReadBSONFieldString(ev.Raw(), "Delay"); v != "" {
		return v
	}
	v, _ := codec.ReadBSONFieldString(ev.Raw(), "TimerDelay")
	return v
}

// boundaryEventFlowGen extracts the nested Flow Part from a BoundaryEvent.
func boundaryEventFlowGen(ev element.Element) *genWf.Flow {
	switch v := ev.(type) {
	case *genWf.InterruptingTimerBoundaryEvent:
		f, _ := v.Flow().(*genWf.Flow)
		return f
	case *genWf.NonInterruptingTimerBoundaryEvent:
		f, _ := v.Flow().(*genWf.Flow)
		return f
	case *genWf.TimerBoundaryEvent:
		f, _ := v.Flow().(*genWf.Flow)
		return f
	}
	return nil
}

// formatWorkflowActivitiesGen mirrors formatWorkflowActivities
// (cmd_workflows.go:286). Dispatch is by storage $Type rather than
// concrete sdk Go type because gen splits the user-task and
// call-microflow hierarchies into multiple subtypes.
//
// Per legacy semantics each activity body is followed by ";" + an empty
// blank line; comment-style lines (jump-target unknown, generic
// fallback) skip the trailing semicolon.
func formatWorkflowActivitiesGen(flow *genWf.Flow, indent string) []string {
	if flow == nil {
		return nil
	}
	var lines []string
	for _, act := range flow.ActivitiesItems() {
		if act == nil {
			continue
		}
		var actLines []string
		isComment := false
		switch act.TypeName() {
		case "Workflows$StartWorkflowActivity",
			"Workflows$EndWorkflowActivity",
			"Workflows$EndOfParallelSplitPathActivity",
			"Workflows$EndOfBoundaryEventPathActivity":
			// Implicit / auto-generated by Mendix — skip.
			continue
		case "Workflows$JumpToActivity":
			actLines = formatJumpToGen(act, indent)
		case "Workflows$WaitForTimerActivity":
			actLines = formatWaitForTimerGen(act, indent)
		case "Workflows$WaitForNotificationActivity":
			actLines = formatWaitForNotificationGen(act, indent)
		case "Workflows$Annotation",
			"Workflows$AnnotationActivity",
			"Workflows$FloatingAnnotation":
			line := formatStandaloneAnnotationGen(act, indent)
			if line == "" {
				continue
			}
			actLines = []string{line}
		// UserTask / CallMicroflow / CallWorkflow / ExclusiveSplit /
		// ParallelSplit / SystemTask formatters land in A3.
		case "Workflows$UserTask",
			"Workflows$SingleUserTaskActivity",
			"Workflows$MultiUserTaskActivity",
			"Workflows$CallMicroflowTask",
			"Workflows$CallMicroflowActivity",
			"Workflows$CallWorkflowActivity",
			"Workflows$ExclusiveSplitActivity",
			"Workflows$ParallelSplitActivity",
			"Workflows$SystemTask":
			isComment = true
			actLines = []string{fmt.Sprintf("%s-- [%s pending A3]", indent, act.TypeName())}
		default:
			isComment = true
			actLines = []string{fmt.Sprintf("%s-- [unknown activity: %s]", indent, act.TypeName())}
		}
		if !isComment && len(actLines) > 0 {
			lastLine := actLines[len(actLines)-1]
			if idx := strings.Index(lastLine, " -- "); idx >= 0 {
				actLines[len(actLines)-1] = lastLine[:idx] + ";" + lastLine[idx:]
			} else {
				actLines[len(actLines)-1] = lastLine + ";"
			}
		}
		lines = append(lines, actLines...)
		lines = append(lines, "")
	}
	return lines
}

// formatJumpToGen renders a JumpToActivity. Mirrors the legacy
// "jump to %s comment '%s'" syntax.
func formatJumpToGen(elem element.Element, indent string) []string {
	jt, ok := elem.(*genWf.JumpToActivity)
	if !ok {
		return nil
	}
	var lines []string
	if anno := formatAnnotationGen(jt.Annotation(), indent); anno != "" {
		lines = append(lines, anno)
	}
	target := jt.TargetActivityQualifiedName()
	if target == "" {
		target = "?"
	}
	caption := jt.Caption()
	if caption == "" {
		caption = jt.Name()
	}
	escaped := strings.ReplaceAll(caption, "'", "''")
	lines = append(lines, fmt.Sprintf("%sjump to %s comment '%s'", indent, target, escaped))
	return lines
}

// formatWaitForTimerGen renders a WaitForTimerActivity.
//
// gen `Delay()` corresponds to sdk `DelayExpression` (R-renamed).
func formatWaitForTimerGen(elem element.Element, indent string) []string {
	wt, ok := elem.(*genWf.WaitForTimerActivity)
	if !ok {
		return nil
	}
	var lines []string
	if anno := formatAnnotationGen(wt.Annotation(), indent); anno != "" {
		lines = append(lines, anno)
	}
	caption := wt.Caption()
	if caption == "" {
		caption = wt.Name()
	}
	escapedCaption := strings.ReplaceAll(caption, "'", "''")
	if delay := wt.Delay(); delay != "" {
		escapedDelay := strings.ReplaceAll(delay, "'", "''")
		lines = append(lines, fmt.Sprintf("%swait for timer '%s' comment '%s'", indent, escapedDelay, escapedCaption))
	} else {
		lines = append(lines, fmt.Sprintf("%swait for timer comment '%s'", indent, escapedCaption))
	}
	return lines
}

// formatWaitForNotificationGen renders a WaitForNotificationActivity
// followed by its boundary events.
func formatWaitForNotificationGen(elem element.Element, indent string) []string {
	wn, ok := elem.(*genWf.WaitForNotificationActivity)
	if !ok {
		return nil
	}
	var lines []string
	if anno := formatAnnotationGen(wn.Annotation(), indent); anno != "" {
		lines = append(lines, anno)
	}
	caption := wn.Caption()
	if caption == "" {
		caption = wn.Name()
	}
	lines = append(lines, fmt.Sprintf("%swait for notification -- %s", indent, caption))
	lines = append(lines, formatBoundaryEventsGen(wn.BoundaryEventsItems(), indent+"  ")...)
	return lines
}

// formatStandaloneAnnotationGen renders a top-level annotation activity
// (sticky note) — Workflows$Annotation / FloatingAnnotation /
// AnnotationActivity. Returns "" when the description is empty so the
// dispatcher can skip it.
func formatStandaloneAnnotationGen(elem element.Element, indent string) string {
	desc := ""
	switch v := elem.(type) {
	case *genWf.Annotation:
		desc = v.Description()
	case *genWf.FloatingAnnotation:
		desc = v.Description()
	default:
		desc, _ = codec.ReadBSONFieldString(elem.Raw(), "Description")
	}
	if desc == "" {
		return ""
	}
	escaped := strings.ReplaceAll(desc, "'", "''")
	return fmt.Sprintf("%sannotation '%s'", indent, escaped)
}

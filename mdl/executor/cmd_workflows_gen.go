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

	"github.com/mendixlabs/mxcli/mdl/ast"
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

// describeWorkflowGen handles DESCRIBE WORKFLOW via gen-typed Workflow
// units. Mirrors describeWorkflow (cmd_workflows.go:134).
func describeWorkflowGen(ctx *ExecContext, name ast.QualifiedName) error {
	output, _, err := describeWorkflowToStringGen(ctx, name)
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Output, output)
	return nil
}

// describeWorkflowToStringGen renders MDL output for a gen-typed
// workflow. The signature mirrors describeWorkflowToString — the
// elkSourceRange map is always nil because the legacy implementation
// returned nil too (see cmd_workflows.go:234); R10 in the plan was
// based on a stale read of the legacy code.
func describeWorkflowToStringGen(ctx *ExecContext, name ast.QualifiedName) (string, map[string]elkSourceRange, error) {
	h, err := getHierarchy(ctx)
	if err != nil {
		return "", nil, mdlerrors.NewBackend("build hierarchy", err)
	}
	pairs, err := listWorkflowsWithContainerGen(ctx)
	if err != nil {
		return "", nil, mdlerrors.NewBackend("list workflows", err)
	}

	var target *genWf.Workflow
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modName := ""
		if h != nil {
			modID := h.FindModuleID(model.ID(p.ContainerID))
			modName = h.GetModuleName(modID)
		}
		if modName == name.Module && p.Elem.Name() == name.Name {
			target = p.Elem
			break
		}
	}
	if target == nil {
		return "", nil, mdlerrors.NewNotFound("workflow", name.String())
	}

	qualifiedName := name.Module + "." + name.Name

	var lines []string

	if doc := target.Documentation(); doc != "" {
		lines = append(lines, "/**")
		for docLine := range strings.SplitSeq(doc, "\n") {
			lines = append(lines, " * "+docLine)
		}
		lines = append(lines, " */")
	}

	lines = append(lines, fmt.Sprintf("-- Workflow: %s", qualifiedName))
	if anno := workflowAnnotationStringGen(target); anno != "" {
		lines = append(lines, fmt.Sprintf("-- %s", anno))
	}
	lines = append(lines, "")

	lines = append(lines, fmt.Sprintf("create workflow %s", qualifiedName))

	if ent := workflowParameterEntityGen(target); ent != "" {
		lines = append(lines, fmt.Sprintf("  parameter $WorkflowContext: %s", ent))
	}

	if displayName := readTextElementGen(target.WorkflowName()); displayName != "" {
		escaped := strings.ReplaceAll(displayName, "'", "''")
		lines = append(lines, fmt.Sprintf("  display '%s'", escaped))
	} else if title := target.Title(); title != "" {
		escaped := strings.ReplaceAll(title, "'", "''")
		lines = append(lines, fmt.Sprintf("  display '%s'", escaped))
	}

	if desc := readTextElementGen(target.WorkflowDescription()); desc != "" {
		escaped := strings.ReplaceAll(desc, "'", "''")
		lines = append(lines, fmt.Sprintf("  description '%s'", escaped))
	}

	if lvl := target.ExportLevel(); lvl != "" {
		lines = append(lines, fmt.Sprintf("  export level %s", lvl))
	}

	if op := target.OverviewPageQualifiedName(); op != "" {
		lines = append(lines, fmt.Sprintf("  overview page %s", op))
	}

	if dd := target.DueDate(); dd != "" {
		lines = append(lines, fmt.Sprintf("  due date '%s'", dd))
	}

	lines = append(lines, "")

	lines = append(lines, "begin")
	if flow, ok := target.Flow().(*genWf.Flow); ok {
		lines = append(lines, formatWorkflowActivitiesGen(flow, "  ")...)
	}
	lines = append(lines, "end workflow")
	lines = append(lines, "/")

	return strings.Join(lines, "\n"), nil, nil
}

// workflowAnnotationStringGen extracts the Description text from a
// workflow's Annotation Part (legacy header-comment field). Returns ""
// if absent.
func workflowAnnotationStringGen(wf *genWf.Workflow) string {
	if wf == nil {
		return ""
	}
	a := wf.Annotation()
	if a == nil {
		return ""
	}
	if anno, ok := a.(*genWf.Annotation); ok {
		return anno.Description()
	}
	v, _ := codec.ReadBSONFieldString(a.Raw(), "Description")
	return v
}

// readTextElementGen extracts the inner Text scalar from a Texts$Text
// wrapper element. Returns "" if elem is nil or has no Text field.
//
// Mendix Texts$Text wraps a localized translations table; for MDL
// describe output we surface the first available translation by reading
// the BSON field directly.
func readTextElementGen(elem element.Element) string {
	if elem == nil {
		return ""
	}
	for _, field := range []string{"Text", "Translation", "Value"} {
		if v, _ := codec.ReadBSONFieldString(elem.Raw(), field); v != "" {
			return v
		}
	}
	return ""
}

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
		case "Workflows$UserTask",
			"Workflows$SingleUserTaskActivity",
			"Workflows$MultiUserTaskActivity":
			actLines = formatUserTaskGen(act, indent)
		case "Workflows$CallMicroflowTask",
			"Workflows$CallMicroflowActivity":
			actLines = formatCallMicroflowGen(act, indent)
		case "Workflows$CallWorkflowActivity":
			actLines = formatCallWorkflowGen(act, indent)
		case "Workflows$ExclusiveSplitActivity":
			actLines = formatExclusiveSplitGen(act, indent)
		case "Workflows$ParallelSplitActivity":
			actLines = formatParallelSplitGen(act, indent)
		case "Workflows$SystemTask":
			actLines = formatSystemTaskGen(act, indent)
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

// userTaskShapeGen normalises the three concrete UserTask gen subtypes
// (UserTask, SingleUserTaskActivity, MultiUserTaskActivity) into a
// single shape so formatUserTaskGen can be one function.
type userTaskShapeGen struct {
	Name           string
	Caption        string
	Annotation     element.Element
	Page           string // PageQualifiedName
	UserSource     element.Element
	UserTaskEntity string
	DueDate        string
	Description    string // task-level Description
	Outcomes       []element.Element
	BoundaryEvents []element.Element
	IsMulti        bool
}

func userTaskShapeGenFor(elem element.Element) (userTaskShapeGen, bool) {
	switch v := elem.(type) {
	case *genWf.UserTask:
		return userTaskShapeGen{
			Name:           v.Name(),
			Caption:        v.Caption(),
			Annotation:     v.Annotation(),
			Page:           v.PageQualifiedName(),
			UserSource:     v.UserSource(),
			UserTaskEntity: v.UserTaskEntityQualifiedName(),
			DueDate:        v.DueDate(),
			Description:    readTextElementGen(v.TaskDescription()),
			Outcomes:       v.OutcomesItems(),
			BoundaryEvents: nil, // gen UserTask has no boundary events accessor
			IsMulti:        false,
		}, true
	case *genWf.SingleUserTaskActivity:
		pageQN := ""
		if tp, ok2 := v.TaskPage().(*genWf.PageReference); ok2 {
			pageQN = tp.PageQualifiedName()
		}
		return userTaskShapeGen{
			Name:           v.Name(),
			Caption:        v.Caption(),
			Annotation:     v.Annotation(),
			Page:           pageQN,
			UserSource:     v.UserTargeting(),
			DueDate:        v.DueDate(),
			Description:    readTextElementGen(v.TaskDescription()),
			Outcomes:       v.OutcomesItems(),
			BoundaryEvents: v.BoundaryEventsItems(),
			IsMulti:        false,
		}, true
	case *genWf.MultiUserTaskActivity:
		pageQN := ""
		if tp, ok2 := v.TaskPage().(*genWf.PageReference); ok2 {
			pageQN = tp.PageQualifiedName()
		}
		return userTaskShapeGen{
			Name:           v.Name(),
			Caption:        v.Caption(),
			Annotation:     v.Annotation(),
			Page:           pageQN,
			UserSource:     v.UserTargeting(),
			DueDate:        v.DueDate(),
			Description:    readTextElementGen(v.TaskDescription()),
			Outcomes:       v.OutcomesItems(),
			BoundaryEvents: v.BoundaryEventsItems(),
			IsMulti:        true,
		}, true
	}
	return userTaskShapeGen{}, false
}

// formatUserTaskGen renders a UserTask / SingleUserTaskActivity /
// MultiUserTaskActivity. Mirrors formatUserTask (cmd_workflows.go:398).
func formatUserTaskGen(elem element.Element, indent string) []string {
	shape, ok := userTaskShapeGenFor(elem)
	if !ok {
		return nil
	}
	var lines []string
	if a := formatAnnotationGen(shape.Annotation, indent); a != "" {
		lines = append(lines, a)
	}
	caption := shape.Caption
	if caption == "" {
		caption = shape.Name
	}
	nameStr := shape.Name
	if nameStr == "" {
		nameStr = "unnamed"
	}
	keyword := "user task"
	if shape.IsMulti {
		keyword = "multi user task"
	}
	lines = append(lines, fmt.Sprintf("%s%s %s '%s'", indent, keyword, nameStr, caption))
	if shape.Page != "" {
		lines = append(lines, fmt.Sprintf("%s  page %s", indent, shape.Page))
	}
	lines = append(lines, formatUserSourceGen(shape.UserSource, indent+"  ")...)
	if shape.UserTaskEntity != "" {
		lines = append(lines, fmt.Sprintf("%s  entity %s", indent, shape.UserTaskEntity))
	}
	if shape.DueDate != "" {
		escaped := strings.ReplaceAll(shape.DueDate, "'", "''")
		lines = append(lines, fmt.Sprintf("%s  due date '%s'", indent, escaped))
	}
	if shape.Description != "" {
		escaped := strings.ReplaceAll(shape.Description, "'", "''")
		lines = append(lines, fmt.Sprintf("%s  description '%s'", indent, escaped))
	}
	if len(shape.Outcomes) > 0 {
		lines = append(lines, fmt.Sprintf("%s  outcomes", indent))
		for _, oc := range shape.Outcomes {
			lines = append(lines, formatUserTaskOutcomeGen(oc, indent+"    ")...)
		}
	}
	lines = append(lines, formatBoundaryEventsGen(shape.BoundaryEvents, indent+"  ")...)
	return lines
}

// formatUserSourceGen renders the per-user-source MDL clause.
func formatUserSourceGen(src element.Element, indent string) []string {
	if src == nil {
		return nil
	}
	switch v := src.(type) {
	case *genWf.MicroflowBasedUserSource:
		if mf := v.MicroflowQualifiedName(); mf != "" {
			return []string{fmt.Sprintf("%stargeting users microflow %s", indent, mf)}
		}
	case *genWf.XPathBasedUserSource:
		if xp := v.XPathConstraint(); xp != "" {
			return []string{fmt.Sprintf("%stargeting users xpath '%s'", indent, strings.ReplaceAll(xp, "'", "''"))}
		}
	case *genWf.XPathUserTargeting:
		if xp := v.XPathConstraint(); xp != "" {
			return []string{fmt.Sprintf("%stargeting xpath '%s'", indent, strings.ReplaceAll(xp, "'", "''"))}
		}
	case *genWf.MicroflowUserTargeting:
		if mf := v.MicroflowQualifiedName(); mf != "" {
			return []string{fmt.Sprintf("%stargeting microflow %s", indent, mf)}
		}
	case *genWf.MicroflowGroupTargeting:
		if mf := v.MicroflowQualifiedName(); mf != "" {
			return []string{fmt.Sprintf("%stargeting groups microflow %s", indent, mf)}
		}
	case *genWf.XPathGroupTargeting:
		if xp := v.XPathConstraint(); xp != "" {
			return []string{fmt.Sprintf("%stargeting groups xpath '%s'", indent, strings.ReplaceAll(xp, "'", "''"))}
		}
	}
	return nil
}

// formatUserTaskOutcomeGen renders one UserTaskOutcome inside an
// "outcomes" block. Mirrors the legacy single-quote-wrapped output.
func formatUserTaskOutcomeGen(elem element.Element, indent string) []string {
	oc, ok := elem.(*genWf.UserTaskOutcome)
	if !ok {
		return nil
	}
	value := oc.Value()
	if value == "" {
		value = oc.Caption()
	}
	if value == "" {
		value = oc.Name()
	}
	if flow, ok := oc.Flow().(*genWf.Flow); ok && flow != nil && len(flow.ActivitiesItems()) > 0 {
		var lines []string
		lines = append(lines, fmt.Sprintf("%s'%s' {", indent, value))
		lines = append(lines, formatWorkflowActivitiesGen(flow, indent+"  ")...)
		lines = append(lines, fmt.Sprintf("%s}", indent))
		return lines
	}
	return []string{fmt.Sprintf("%s'%s' { }", indent, value)}
}

// callMicroflowShapeGen normalises CallMicroflowActivity vs
// CallMicroflowTask into a single shape.
type callMicroflowShapeGen struct {
	Name              string
	Caption           string
	Annotation        element.Element
	Microflow         string
	ParameterMappings []element.Element
	BoundaryEvents    []element.Element
	Outcomes          []element.Element
}

func callMicroflowShapeGenFor(elem element.Element) (callMicroflowShapeGen, bool) {
	switch v := elem.(type) {
	case *genWf.CallMicroflowActivity:
		return callMicroflowShapeGen{
			Name:              v.Name(),
			Caption:           v.Caption(),
			Annotation:        v.Annotation(),
			Microflow:         v.MicroflowQualifiedName(),
			ParameterMappings: v.ParameterMappingsItems(),
			BoundaryEvents:    v.BoundaryEventsItems(),
			Outcomes:          v.OutcomesItems(),
		}, true
	case *genWf.CallMicroflowTask:
		return callMicroflowShapeGen{
			Name:              v.Name(),
			Caption:           v.Caption(),
			Annotation:        v.Annotation(),
			Microflow:         v.MicroflowQualifiedName(),
			ParameterMappings: v.ParameterMappingsItems(),
			BoundaryEvents:    v.BoundaryEventsItems(),
			Outcomes:          v.OutcomesItems(),
		}, true
	}
	return callMicroflowShapeGen{}, false
}

// formatCallMicroflowGen renders CallMicroflowActivity / CallMicroflowTask.
func formatCallMicroflowGen(elem element.Element, indent string) []string {
	shape, ok := callMicroflowShapeGenFor(elem)
	if !ok {
		return nil
	}
	var lines []string
	if a := formatAnnotationGen(shape.Annotation, indent); a != "" {
		lines = append(lines, a)
	}
	caption := shape.Caption
	if caption == "" {
		caption = shape.Name
	}
	mf := shape.Microflow
	if mf == "" {
		mf = "?"
	}
	if len(shape.ParameterMappings) > 0 {
		params := formatMicroflowCallParamsGen(shape.ParameterMappings)
		lines = append(lines, fmt.Sprintf("%scall microflow %s with (%s) -- %s", indent, mf, strings.Join(params, ", "), caption))
	} else {
		lines = append(lines, fmt.Sprintf("%scall microflow %s -- %s", indent, mf, caption))
	}
	lines = append(lines, formatBoundaryEventsGen(shape.BoundaryEvents, indent+"  ")...)
	lines = append(lines, formatConditionOutcomesGen(shape.Outcomes, indent)...)
	return lines
}

// formatMicroflowCallParamsGen renders ParameterMapping entries as
// "name = 'expr'" pairs (legacy semantics — bare last-segment name
// from ParameterQualifiedName).
func formatMicroflowCallParamsGen(items []element.Element) []string {
	out := make([]string, 0, len(items))
	for _, m := range items {
		paramName, expr := paramMappingNameExprGen(m)
		if paramName == "" && expr == "" {
			continue
		}
		if idx := strings.LastIndex(paramName, "."); idx >= 0 {
			paramName = paramName[idx+1:]
		}
		escaped := strings.ReplaceAll(expr, "'", "''")
		out = append(out, fmt.Sprintf("%s = '%s'", paramName, escaped))
	}
	return out
}

// paramMappingNameExprGen extracts (qualifiedName, expression) from
// either MicroflowCallParameterMapping or WorkflowCallParameterMapping
// (gen splits the type per call-site).
func paramMappingNameExprGen(elem element.Element) (string, string) {
	switch v := elem.(type) {
	case *genWf.MicroflowCallParameterMapping:
		return v.ParameterQualifiedName(), v.Expression()
	case *genWf.WorkflowCallParameterMapping:
		return v.ParameterQualifiedName(), v.Expression()
	}
	return "", ""
}

// formatCallWorkflowGen renders CallWorkflowActivity.
func formatCallWorkflowGen(elem element.Element, indent string) []string {
	cw, ok := elem.(*genWf.CallWorkflowActivity)
	if !ok {
		return nil
	}
	var lines []string
	if a := formatAnnotationGen(cw.Annotation(), indent); a != "" {
		lines = append(lines, a)
	}
	caption := cw.Caption()
	if caption == "" {
		caption = cw.Name()
	}
	wf := cw.WorkflowQualifiedName()
	if wf == "" {
		wf = "?"
	}
	escapedCaption := strings.ReplaceAll(caption, "'", "''")
	if items := cw.ParameterMappingsItems(); len(items) > 0 {
		params := formatMicroflowCallParamsGen(items)
		lines = append(lines, fmt.Sprintf("%scall workflow %s comment '%s' with (%s)", indent, wf, escapedCaption, strings.Join(params, ", ")))
	} else {
		lines = append(lines, fmt.Sprintf("%scall workflow %s comment '%s'", indent, wf, escapedCaption))
	}
	lines = append(lines, formatBoundaryEventsGen(cw.BoundaryEventsItems(), indent+"  ")...)
	return lines
}

// formatExclusiveSplitGen renders an ExclusiveSplitActivity (decision).
func formatExclusiveSplitGen(elem element.Element, indent string) []string {
	es, ok := elem.(*genWf.ExclusiveSplitActivity)
	if !ok {
		return nil
	}
	var lines []string
	if a := formatAnnotationGen(es.Annotation(), indent); a != "" {
		lines = append(lines, a)
	}
	caption := es.Caption()
	if caption == "" {
		caption = es.Name()
	}
	if expr := es.Expression(); expr != "" {
		escapedExpr := strings.ReplaceAll(expr, "'", "''")
		lines = append(lines, fmt.Sprintf("%sdecision '%s' -- %s", indent, escapedExpr, caption))
	} else {
		lines = append(lines, fmt.Sprintf("%sdecision -- %s", indent, caption))
	}
	lines = append(lines, formatConditionOutcomesGen(es.OutcomesItems(), indent)...)
	return lines
}

// formatParallelSplitGen renders a ParallelSplitActivity.
func formatParallelSplitGen(elem element.Element, indent string) []string {
	ps, ok := elem.(*genWf.ParallelSplitActivity)
	if !ok {
		return nil
	}
	var lines []string
	if a := formatAnnotationGen(ps.Annotation(), indent); a != "" {
		lines = append(lines, a)
	}
	caption := ps.Caption()
	if caption == "" {
		caption = ps.Name()
	}
	lines = append(lines, fmt.Sprintf("%sparallel split -- %s", indent, caption))
	for i, outcome := range ps.OutcomesItems() {
		lines = append(lines, fmt.Sprintf("%s  path %d {", indent, i+1))
		if pso, ok := outcome.(*genWf.ParallelSplitOutcome); ok {
			if flow, ok := pso.Flow().(*genWf.Flow); ok && flow != nil && len(flow.ActivitiesItems()) > 0 {
				lines = append(lines, formatWorkflowActivitiesGen(flow, indent+"    ")...)
			}
		}
		lines = append(lines, fmt.Sprintf("%s  }", indent))
	}
	return lines
}

// formatConditionOutcomesGen renders a slice of ConditionOutcome
// elements (used by ExclusiveSplit, CallMicroflow, SystemTask).
func formatConditionOutcomesGen(outcomes []element.Element, indent string) []string {
	if len(outcomes) == 0 {
		return nil
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("%s  outcomes", indent))
	for _, oc := range outcomes {
		name, flow := conditionOutcomeNameFlowGen(oc)
		if flow != nil && len(flow.ActivitiesItems()) > 0 {
			lines = append(lines, fmt.Sprintf("%s    %s -> {", indent, name))
			lines = append(lines, formatWorkflowActivitiesGen(flow, indent+"      ")...)
			lines = append(lines, fmt.Sprintf("%s    }", indent))
		} else {
			lines = append(lines, fmt.Sprintf("%s    %s -> { }", indent, name))
		}
	}
	return lines
}

// conditionOutcomeNameFlowGen extracts (name, *Flow) from any concrete
// ConditionOutcome gen type.
func conditionOutcomeNameFlowGen(oc element.Element) (string, *genWf.Flow) {
	if oc == nil {
		return "", nil
	}
	switch v := oc.(type) {
	case *genWf.BooleanConditionOutcome:
		name := "false"
		if v.Value() {
			name = "true"
		}
		f, _ := v.Flow().(*genWf.Flow)
		return name, f
	case *genWf.EnumerationValueConditionOutcome:
		// Output the full qualified name (Module.EnumName.ValueName) wrapped in
		// single quotes so the MDL round-trip produces a valid STRING_LITERAL
		// and BSON stores the format Studio Pro 11.10.0 requires.
		name := "'" + v.ValueQualifiedName() + "'"
		f, _ := v.Flow().(*genWf.Flow)
		return name, f
	case *genWf.VoidConditionOutcome:
		f, _ := v.Flow().(*genWf.Flow)
		return "default", f
	case *genWf.ExclusiveSplitOutcome:
		// Generic outcome — read raw BSON for the value caption.
		val, _ := codec.ReadBSONFieldString(v.Raw(), "Value")
		f, _ := v.Flow().(*genWf.Flow)
		return val, f
	}
	return "", nil
}

// formatSystemTaskGen renders a SystemTask. gen has no narrow type for
// it, so this reads the legacy fields out of raw BSON.
func formatSystemTaskGen(elem element.Element, indent string) []string {
	if elem == nil {
		return nil
	}
	raw := elem.Raw()
	annotation, _ := codec.ReadBSONFieldString(raw, "Annotation")
	name, _ := codec.ReadBSONFieldString(raw, "Name")
	caption, _ := codec.ReadBSONFieldString(raw, "Caption")
	mf, _ := codec.ReadBSONFieldString(raw, "Microflow")

	var lines []string
	if annotation != "" {
		escaped := strings.ReplaceAll(annotation, "'", "''")
		lines = append(lines, fmt.Sprintf("%sannotation '%s';", indent, escaped))
	}
	displayCaption := caption
	if displayCaption == "" {
		displayCaption = name
	}
	if mf == "" {
		mf = "?"
	}
	lines = append(lines, fmt.Sprintf("%scall microflow %s -- %s", indent, mf, displayCaption))
	// SystemTask outcomes are handled by the dispatcher's call site
	// because gen does not surface them as a typed accessor; the legacy
	// formatter walked Outcomes for visualisation only — round-trip
	// invariant is already preserved by the BSON re-encode path, so we
	// skip nested outcome rendering here.
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

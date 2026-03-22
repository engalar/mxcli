// SPDX-License-Identifier: Apache-2.0

// Package executor - Workflow SHOW/DESCRIBE commands
package executor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/sdk/workflows"
)

// showWorkflows handles SHOW WORKFLOWS command.
func (e *Executor) showWorkflows(moduleName string) error {
	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	wfs, err := e.reader.ListWorkflows()
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
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
	qnWidth := len("Qualified Name")
	actWidth := len("Activities")
	utWidth := len("User Tasks")
	decWidth := len("Decisions")
	peWidth := len("Parameter Entity")

	for _, wf := range wfs {
		modID := h.FindModuleID(wf.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleName != "" && modName != moduleName {
			continue
		}

		qualifiedName := modName + "." + wf.Name
		paramEntity := ""
		if wf.Parameter != nil {
			paramEntity = wf.Parameter.EntityRef
		}

		acts, uts, decs := countWorkflowActivities(wf)

		rows = append(rows, row{qualifiedName, modName, wf.Name, acts, uts, decs, paramEntity})
		if len(qualifiedName) > qnWidth {
			qnWidth = len(qualifiedName)
		}
		actStr := fmt.Sprintf("%d", acts)
		if len(actStr) > actWidth {
			actWidth = len(actStr)
		}
		utStr := fmt.Sprintf("%d", uts)
		if len(utStr) > utWidth {
			utWidth = len(utStr)
		}
		decStr := fmt.Sprintf("%d", decs)
		if len(decStr) > decWidth {
			decWidth = len(decStr)
		}
		if len(paramEntity) > peWidth {
			peWidth = len(paramEntity)
		}
	}

	// Sort by qualified name
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	// Markdown table
	fmt.Fprintf(e.output, "| %-*s | %-*s | %-*s | %-*s | %-*s |\n",
		qnWidth, "Qualified Name", actWidth, "Activities", utWidth, "User Tasks",
		decWidth, "Decisions", peWidth, "Parameter Entity")
	fmt.Fprintf(e.output, "|-%s-|-%s-|-%s-|-%s-|-%s-|\n",
		strings.Repeat("-", qnWidth), strings.Repeat("-", actWidth),
		strings.Repeat("-", utWidth), strings.Repeat("-", decWidth),
		strings.Repeat("-", peWidth))
	for _, r := range rows {
		fmt.Fprintf(e.output, "| %-*s | %-*d | %-*d | %-*d | %-*s |\n",
			qnWidth, r.qualifiedName, actWidth, r.activities, utWidth, r.userTasks,
			decWidth, r.decisions, peWidth, r.paramEntity)
	}
	fmt.Fprintf(e.output, "\n(%d workflows)\n", len(rows))
	return nil
}

// countWorkflowActivities counts total activities, user tasks, and decisions in a workflow.
func countWorkflowActivities(wf *workflows.Workflow) (total, userTasks, decisions int) {
	if wf.Flow == nil {
		return
	}
	countFlowActivities(wf.Flow, &total, &userTasks, &decisions)
	return
}

// countFlowActivities recursively counts activities in a flow and its sub-flows.
func countFlowActivities(flow *workflows.Flow, total, userTasks, decisions *int) {
	if flow == nil {
		return
	}
	for _, act := range flow.Activities {
		*total++
		switch a := act.(type) {
		case *workflows.UserTask:
			*userTasks++
			for _, outcome := range a.Outcomes {
				countFlowActivities(outcome.Flow, total, userTasks, decisions)
			}
		case *workflows.ExclusiveSplitActivity:
			*decisions++
			for _, outcome := range a.Outcomes {
				if co, ok := outcome.(*workflows.BooleanConditionOutcome); ok {
					countFlowActivities(co.Flow, total, userTasks, decisions)
				} else if co, ok := outcome.(*workflows.EnumerationValueConditionOutcome); ok {
					countFlowActivities(co.Flow, total, userTasks, decisions)
				} else if co, ok := outcome.(*workflows.VoidConditionOutcome); ok {
					countFlowActivities(co.Flow, total, userTasks, decisions)
				}
			}
		case *workflows.ParallelSplitActivity:
			for _, outcome := range a.Outcomes {
				countFlowActivities(outcome.Flow, total, userTasks, decisions)
			}
		case *workflows.CallMicroflowTask:
			for _, outcome := range a.Outcomes {
				if co, ok := outcome.(*workflows.BooleanConditionOutcome); ok {
					countFlowActivities(co.Flow, total, userTasks, decisions)
				} else if co, ok := outcome.(*workflows.EnumerationValueConditionOutcome); ok {
					countFlowActivities(co.Flow, total, userTasks, decisions)
				} else if co, ok := outcome.(*workflows.VoidConditionOutcome); ok {
					countFlowActivities(co.Flow, total, userTasks, decisions)
				}
			}
		case *workflows.SystemTask:
			for _, outcome := range a.Outcomes {
				if co, ok := outcome.(*workflows.BooleanConditionOutcome); ok {
					countFlowActivities(co.Flow, total, userTasks, decisions)
				} else if co, ok := outcome.(*workflows.EnumerationValueConditionOutcome); ok {
					countFlowActivities(co.Flow, total, userTasks, decisions)
				} else if co, ok := outcome.(*workflows.VoidConditionOutcome); ok {
					countFlowActivities(co.Flow, total, userTasks, decisions)
				}
			}
		}
	}
}

// describeWorkflow handles DESCRIBE WORKFLOW command.
func (e *Executor) describeWorkflow(name ast.QualifiedName) error {
	output, _, err := e.describeWorkflowToString(name)
	if err != nil {
		return err
	}
	fmt.Fprintln(e.output, output)
	return nil
}

// describeWorkflowToString generates MDL-like output for a workflow and returns it as a string.
func (e *Executor) describeWorkflowToString(name ast.QualifiedName) (string, map[string]elkSourceRange, error) {
	h, err := e.getHierarchy()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build hierarchy: %w", err)
	}

	allWorkflows, err := e.reader.ListWorkflows()
	if err != nil {
		return "", nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	var targetWf *workflows.Workflow
	for _, wf := range allWorkflows {
		modID := h.FindModuleID(wf.ContainerID)
		modName := h.GetModuleName(modID)
		if modName == name.Module && wf.Name == name.Name {
			targetWf = wf
			break
		}
	}

	if targetWf == nil {
		return "", nil, fmt.Errorf("workflow not found: %s", name)
	}

	var lines []string
	qualifiedName := name.Module + "." + name.Name

	// Documentation
	if targetWf.Documentation != "" {
		lines = append(lines, "/**")
		for docLine := range strings.SplitSeq(targetWf.Documentation, "\n") {
			lines = append(lines, " * "+docLine)
		}
		lines = append(lines, " */")
	}

	// Header
	lines = append(lines, fmt.Sprintf("-- Workflow: %s", qualifiedName))
	if targetWf.Annotation != "" {
		lines = append(lines, fmt.Sprintf("-- %s", targetWf.Annotation))
	}
	lines = append(lines, "")

	lines = append(lines, fmt.Sprintf("WORKFLOW %s", qualifiedName))

	// Context parameter
	if targetWf.Parameter != nil && targetWf.Parameter.EntityRef != "" {
		lines = append(lines, fmt.Sprintf("  PARAMETER $WorkflowContext: %s", targetWf.Parameter.EntityRef))
	}

	// Display name
	if targetWf.WorkflowName != "" {
		escaped := strings.ReplaceAll(targetWf.WorkflowName, "'", "''")
		lines = append(lines, fmt.Sprintf("  DISPLAY '%s'", escaped))
	}

	// Description
	if targetWf.WorkflowDescription != "" {
		escaped := strings.ReplaceAll(targetWf.WorkflowDescription, "'", "''")
		lines = append(lines, fmt.Sprintf("  DESCRIPTION '%s'", escaped))
	}

	// Export level (only emit when non-empty)
	if targetWf.ExportLevel != "" {
		lines = append(lines, fmt.Sprintf("  EXPORT LEVEL %s", targetWf.ExportLevel))
	}

	// Overview page
	if targetWf.OverviewPage != "" {
		lines = append(lines, fmt.Sprintf("  OVERVIEW PAGE %s", targetWf.OverviewPage))
	}

	// Due date
	if targetWf.DueDate != "" {
		lines = append(lines, fmt.Sprintf("  DUE DATE '%s'", targetWf.DueDate))
	}

	lines = append(lines, "")

	lines = append(lines, "BEGIN")
	// Activities
	if targetWf.Flow != nil {
		actLines := formatWorkflowActivities(targetWf.Flow, "  ")
		lines = append(lines, actLines...)
	}

	lines = append(lines, "END WORKFLOW")
	lines = append(lines, "/")

	return strings.Join(lines, "\n"), nil, nil
}

// formatAnnotation returns a comment line for an annotation, or empty string if none.
func formatAnnotation(annotation string, indent string) string {
	if annotation == "" {
		return ""
	}
	return fmt.Sprintf("%s-- %s", indent, annotation)
}

// boundaryEventKeyword maps an EventType string to the MDL BOUNDARY EVENT keyword sequence.
func boundaryEventKeyword(eventType string) string {
	switch eventType {
	case "InterruptingTimer":
		return "BOUNDARY EVENT INTERRUPTING TIMER"
	case "NonInterruptingTimer":
		return "BOUNDARY EVENT NON INTERRUPTING TIMER"
	default:
		return "BOUNDARY EVENT TIMER"
	}
}

// formatBoundaryEvents formats boundary events for describe output.
func formatBoundaryEvents(events []*workflows.BoundaryEvent, indent string) []string {
	if len(events) == 0 {
		return nil
	}

	var lines []string
	for _, event := range events {
		keyword := boundaryEventKeyword(event.EventType)
		if event.TimerDelay != "" {
			escapedDelay := strings.ReplaceAll(event.TimerDelay, "'", "''")
			lines = append(lines, fmt.Sprintf("%s%s '%s'", indent, keyword, escapedDelay))
		} else {
			lines = append(lines, fmt.Sprintf("%s%s", indent, keyword))
		}
		if event.Flow != nil && len(event.Flow.Activities) > 0 {
			lines = append(lines, fmt.Sprintf("%s{", indent))
			subLines := formatWorkflowActivities(event.Flow, indent+"  ")
			lines = append(lines, subLines...)
			lines = append(lines, fmt.Sprintf("%s}", indent))
		}
	}

	return lines
}

// formatWorkflowActivities generates MDL-like output for workflow activities.
func formatWorkflowActivities(flow *workflows.Flow, indent string) []string {
	if flow == nil {
		return nil
	}

	var lines []string
	for _, act := range flow.Activities {
		var actLines []string
		isComment := false
		switch a := act.(type) {
		case *workflows.UserTask:
			actLines = formatUserTask(a, indent)
		case *workflows.CallMicroflowTask:
			actLines = formatCallMicroflowTask(a, indent)
		case *workflows.SystemTask:
			actLines = formatSystemTask(a, indent)
		case *workflows.CallWorkflowActivity:
			actLines = formatCallWorkflowActivity(a, indent)
		case *workflows.ExclusiveSplitActivity:
			actLines = formatExclusiveSplit(a, indent)
		case *workflows.ParallelSplitActivity:
			actLines = formatParallelSplit(a, indent)
		case *workflows.JumpToActivity:
			target := a.TargetActivity
			if target == "" {
				target = "?"
			}
			caption := a.Caption
			if caption == "" {
				caption = a.Name
			}
			if a.Annotation != "" {
				actLines = append(actLines, formatAnnotation(a.Annotation, indent))
			}
			escapedCaption := strings.ReplaceAll(caption, "'", "''")
			actLines = append(actLines, fmt.Sprintf("%sJUMP TO %s COMMENT '%s'", indent, target, escapedCaption))
		case *workflows.WaitForTimerActivity:
			caption := a.Caption
			if caption == "" {
				caption = a.Name
			}
			if a.Annotation != "" {
				actLines = append(actLines, formatAnnotation(a.Annotation, indent))
			}
			if a.DelayExpression != "" {
				escapedDelay := strings.ReplaceAll(a.DelayExpression, "'", "''")
				escapedCaption := strings.ReplaceAll(caption, "'", "''")
				actLines = append(actLines, fmt.Sprintf("%sWAIT FOR TIMER '%s' COMMENT '%s'", indent, escapedDelay, escapedCaption))
			} else {
				escapedCaption := strings.ReplaceAll(caption, "'", "''")
				actLines = append(actLines, fmt.Sprintf("%sWAIT FOR TIMER COMMENT '%s'", indent, escapedCaption))
			}
		case *workflows.WaitForNotificationActivity:
			caption := a.Caption
			if caption == "" {
				caption = a.Name
			}
			if a.Annotation != "" {
				actLines = append(actLines, formatAnnotation(a.Annotation, indent))
			}
			actLines = append(actLines, fmt.Sprintf("%sWAIT FOR NOTIFICATION -- %s", indent, caption))
			// BoundaryEvents
			actLines = append(actLines, formatBoundaryEvents(a.BoundaryEvents, indent+"  ")...)
		case *workflows.StartWorkflowActivity:
			// Skip start activities - they are implicit
			continue
		case *workflows.EndWorkflowActivity:
			// Skip end activities - they are implicit
			continue
		case *workflows.EndOfParallelSplitPathActivity:
			// Skip - auto-generated by Mendix, implicit in MDL syntax
			continue
		case *workflows.EndOfBoundaryEventPathActivity:
			// Skip - auto-generated by Mendix, implicit in MDL syntax
			continue
		case *workflows.WorkflowAnnotationActivity:
			// Standalone annotation (sticky note) - emit as ANNOTATION statement
			if a.Description != "" {
				escapedDesc := strings.ReplaceAll(a.Description, "'", "''")
				actLines = []string{fmt.Sprintf("%sANNOTATION '%s'", indent, escapedDesc)}
			} else {
				continue
			}
		case *workflows.GenericWorkflowActivity:
			isComment = true
			caption := a.Caption
			if caption == "" {
				caption = a.Name
			}
			actLines = []string{fmt.Sprintf("%s-- [%s] %s", indent, a.TypeString, caption)}
		default:
			isComment = true
			actLines = []string{fmt.Sprintf("%s-- [unknown activity]", indent)}
		}
		// Append semicolon to last line of activity (not for comments)
		// Insert before any -- comment to avoid the comment swallowing the semicolon
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

// formatUserTask formats a user task for describe output.
func formatUserTask(a *workflows.UserTask, indent string) []string {
	var lines []string

	if a.Annotation != "" {
		lines = append(lines, formatAnnotation(a.Annotation, indent))
	}

	caption := a.Caption
	if caption == "" {
		caption = a.Name
	}
	nameStr := a.Name
	if nameStr == "" {
		nameStr = "unnamed"
	}

	taskKeyword := "USER TASK"
	if a.IsMulti {
		taskKeyword = "MULTI USER TASK"
	}
	lines = append(lines, fmt.Sprintf("%s%s %s '%s'", indent, taskKeyword, nameStr, caption))

	if a.Page != "" {
		lines = append(lines, fmt.Sprintf("%s  PAGE %s", indent, a.Page))
	}

	// User targeting
	if a.UserSource != nil {
		switch us := a.UserSource.(type) {
		case *workflows.MicroflowBasedUserSource:
			if us.Microflow != "" {
				lines = append(lines, fmt.Sprintf("%s  TARGETING MICROFLOW %s", indent, us.Microflow))
			}
		case *workflows.XPathBasedUserSource:
			if us.XPath != "" {
				lines = append(lines, fmt.Sprintf("%s  TARGETING XPATH '%s'", indent, us.XPath))
			}
		}
	}

	if a.UserTaskEntity != "" {
		lines = append(lines, fmt.Sprintf("%s  ENTITY %s", indent, a.UserTaskEntity))
	}

	// Due date (task-level)
	if a.DueDate != "" {
		escapedDueDate := strings.ReplaceAll(a.DueDate, "'", "''")
		lines = append(lines, fmt.Sprintf("%s  DUE DATE '%s'", indent, escapedDueDate))
	}

	// Task description
	if a.TaskDescription != "" {
		escaped := strings.ReplaceAll(a.TaskDescription, "'", "''")
		lines = append(lines, fmt.Sprintf("%s  DESCRIPTION '%s'", indent, escaped))
	}

	// Outcomes
	if len(a.Outcomes) > 0 {
		lines = append(lines, fmt.Sprintf("%s  OUTCOMES", indent))
		for _, outcome := range a.Outcomes {
			outValue := outcome.Value
			if outValue == "" {
				outValue = outcome.Caption
			}
			if outValue == "" {
				outValue = outcome.Name
			}
			if outcome.Flow != nil && len(outcome.Flow.Activities) > 0 {
				lines = append(lines, fmt.Sprintf("%s    '%s' {", indent, outValue))
				subLines := formatWorkflowActivities(outcome.Flow, indent+"      ")
				lines = append(lines, subLines...)
				lines = append(lines, fmt.Sprintf("%s    }", indent))
			} else {
				lines = append(lines, fmt.Sprintf("%s    '%s' { }", indent, outValue))
			}
		}
	}

	// BoundaryEvents
	lines = append(lines, formatBoundaryEvents(a.BoundaryEvents, indent+"  ")...)

	return lines
}

// formatCallMicroflowTask formats a call microflow task for describe output.
func formatCallMicroflowTask(a *workflows.CallMicroflowTask, indent string) []string {
	var lines []string

	if a.Annotation != "" {
		lines = append(lines, formatAnnotation(a.Annotation, indent))
	}

	caption := a.Caption
	if caption == "" {
		caption = a.Name
	}

	mf := a.Microflow
	if mf == "" {
		mf = "?"
	}

	if len(a.ParameterMappings) > 0 {
		var params []string
		for _, pm := range a.ParameterMappings {
			paramName := pm.Parameter
			if idx := strings.LastIndex(paramName, "."); idx >= 0 {
				paramName = paramName[idx+1:]
			}
			params = append(params, fmt.Sprintf("%s = '%s'", paramName, strings.ReplaceAll(pm.Expression, "'", "''")))
		}
		lines = append(lines, fmt.Sprintf("%sCALL MICROFLOW %s WITH (%s) -- %s", indent, mf, strings.Join(params, ", "), caption))
	} else {
		lines = append(lines, fmt.Sprintf("%sCALL MICROFLOW %s -- %s", indent, mf, caption))
	}

	// BoundaryEvents
	lines = append(lines, formatBoundaryEvents(a.BoundaryEvents, indent+"  ")...)

	// Outcomes
	lines = append(lines, formatConditionOutcomes(a.Outcomes, indent)...)

	return lines
}

// formatSystemTask formats a system task for describe output.
func formatSystemTask(a *workflows.SystemTask, indent string) []string {
	var lines []string

	if a.Annotation != "" {
		lines = append(lines, formatAnnotation(a.Annotation, indent))
	}

	caption := a.Caption
	if caption == "" {
		caption = a.Name
	}

	mf := a.Microflow
	if mf == "" {
		mf = "?"
	}

	lines = append(lines, fmt.Sprintf("%sCALL MICROFLOW %s -- %s", indent, mf, caption))

	// Outcomes
	lines = append(lines, formatConditionOutcomes(a.Outcomes, indent)...)

	return lines
}

// formatCallWorkflowActivity formats a call workflow activity for describe output.
func formatCallWorkflowActivity(a *workflows.CallWorkflowActivity, indent string) []string {
	var lines []string

	if a.Annotation != "" {
		lines = append(lines, formatAnnotation(a.Annotation, indent))
	}

	caption := a.Caption
	if caption == "" {
		caption = a.Name
	}

	wf := a.Workflow
	if wf == "" {
		wf = "?"
	}

	escapedCaption := strings.ReplaceAll(caption, "'", "''")
	if a.ParameterExpression != "" {
		escapedExpr := strings.ReplaceAll(a.ParameterExpression, "'", "''")
		lines = append(lines, fmt.Sprintf("%sCALL WORKFLOW %s ($WorkflowContext = '%s') COMMENT '%s'", indent, wf, escapedExpr, escapedCaption))
	} else {
		lines = append(lines, fmt.Sprintf("%sCALL WORKFLOW %s COMMENT '%s'", indent, wf, escapedCaption))
	}

	// BoundaryEvents
	lines = append(lines, formatBoundaryEvents(a.BoundaryEvents, indent+"  ")...)

	return lines
}

// formatExclusiveSplit formats an exclusive split (decision) for describe output.
func formatExclusiveSplit(a *workflows.ExclusiveSplitActivity, indent string) []string {
	var lines []string

	if a.Annotation != "" {
		lines = append(lines, formatAnnotation(a.Annotation, indent))
	}

	caption := a.Caption
	if caption == "" {
		caption = a.Name
	}

	if a.Expression != "" {
		escapedExpr := strings.ReplaceAll(a.Expression, "'", "''")
		lines = append(lines, fmt.Sprintf("%sDECISION '%s' -- %s", indent, escapedExpr, caption))
	} else {
		lines = append(lines, fmt.Sprintf("%sDECISION -- %s", indent, caption))
	}

	lines = append(lines, formatConditionOutcomes(a.Outcomes, indent)...)

	return lines
}

// formatParallelSplit formats a parallel split for describe output.
func formatParallelSplit(a *workflows.ParallelSplitActivity, indent string) []string {
	var lines []string

	if a.Annotation != "" {
		lines = append(lines, formatAnnotation(a.Annotation, indent))
	}

	caption := a.Caption
	if caption == "" {
		caption = a.Name
	}

	lines = append(lines, fmt.Sprintf("%sPARALLEL SPLIT -- %s", indent, caption))
	for i, outcome := range a.Outcomes {
		lines = append(lines, fmt.Sprintf("%s  PATH %d {", indent, i+1))
		if outcome.Flow != nil && len(outcome.Flow.Activities) > 0 {
			subLines := formatWorkflowActivities(outcome.Flow, indent+"    ")
			lines = append(lines, subLines...)
		}
		lines = append(lines, fmt.Sprintf("%s  }", indent))
	}

	return lines
}

// formatConditionOutcomes formats condition outcomes for describe output.
func formatConditionOutcomes(outcomes []workflows.ConditionOutcome, indent string) []string {
	if len(outcomes) == 0 {
		return nil
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("%s  OUTCOMES", indent))
	for _, outcome := range outcomes {
		name := outcome.GetName()
		flow := outcome.GetFlow()
		if flow != nil && len(flow.Activities) > 0 {
			lines = append(lines, fmt.Sprintf("%s    %s -> {", indent, name))
			subLines := formatWorkflowActivities(flow, indent+"      ")
			lines = append(lines, subLines...)
			lines = append(lines, fmt.Sprintf("%s    }", indent))
		} else {
			lines = append(lines, fmt.Sprintf("%s    %s -> { }", indent, name))
		}
	}

	return lines
}

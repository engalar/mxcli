// SPDX-License-Identifier: Apache-2.0

// Package executor - CREATE/DROP WORKFLOW commands
package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/mpr"
	"github.com/mendixlabs/mxcli/sdk/workflows"
)

// execCreateWorkflow handles CREATE WORKFLOW statements.
func (e *Executor) execCreateWorkflow(s *ast.CreateWorkflowStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project")
	}

	module, err := e.findModule(s.Name.Module)
	if err != nil {
		return err
	}

	// Check if workflow already exists
	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	existingWorkflows, err := e.reader.ListWorkflows()
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	var existingID model.ID
	for _, existing := range existingWorkflows {
		modID := h.FindModuleID(existing.ContainerID)
		modName := h.GetModuleName(modID)
		if modName == s.Name.Module && existing.Name == s.Name.Name {
			if !s.CreateOrModify {
				return fmt.Errorf("workflow '%s.%s' already exists (use CREATE OR REPLACE to overwrite)", s.Name.Module, s.Name.Name)
			}
			existingID = existing.ID
			break
		}
	}

	wf := &workflows.Workflow{}
	wf.ContainerID = module.ID
	wf.Name = s.Name.Name
	wf.Documentation = s.Documentation

	// Parameter
	if s.ParameterEntity.Module != "" {
		wf.Parameter = &workflows.WorkflowParameter{
			EntityRef: s.ParameterEntity.Module + "." + s.ParameterEntity.Name,
		}
		wf.Parameter.ID = model.ID(generateWorkflowUUID())
	}

	// Overview page
	if s.OverviewPage.Module != "" {
		wf.OverviewPage = s.OverviewPage.Module + "." + s.OverviewPage.Name
	}

	// Due date
	wf.DueDate = s.DueDate

	// Build activities with implicit start/end
	flow := &workflows.Flow{}
	flow.ID = model.ID(generateWorkflowUUID())

	// Add implicit start activity
	startAct := &workflows.StartWorkflowActivity{}
	startAct.ID = model.ID(generateWorkflowUUID())
	startAct.Caption = "Start"
	startAct.Name = "Start"

	// Add implicit end activity
	endAct := &workflows.EndWorkflowActivity{}
	endAct.ID = model.ID(generateWorkflowUUID())
	endAct.Caption = "End"
	endAct.Name = "End"

	// Build user-defined activities
	userActivities := buildWorkflowActivities(s.Activities)

	// Deduplicate activity names to avoid CE0495
	deduplicateActivityNames(userActivities)

	// Compose: start + user activities + end
	flow.Activities = make([]workflows.WorkflowActivity, 0, len(userActivities)+2)
	flow.Activities = append(flow.Activities, startAct)
	flow.Activities = append(flow.Activities, userActivities...)
	flow.Activities = append(flow.Activities, endAct)

	wf.Flow = flow

	if existingID != "" {
		// Delete existing and recreate
		if err := e.writer.DeleteWorkflow(existingID); err != nil {
			return fmt.Errorf("failed to delete existing workflow: %w", err)
		}
	}

	if err := e.writer.CreateWorkflow(wf); err != nil {
		return fmt.Errorf("failed to create workflow: %w", err)
	}

	e.invalidateHierarchy()
	fmt.Fprintf(e.output, "Created workflow: %s.%s\n", s.Name.Module, s.Name.Name)
	return nil
}

// execDropWorkflow handles DROP WORKFLOW statements.
func (e *Executor) execDropWorkflow(s *ast.DropWorkflowStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project")
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	wfs, err := e.reader.ListWorkflows()
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	for _, wf := range wfs {
		modID := h.FindModuleID(wf.ContainerID)
		modName := h.GetModuleName(modID)
		if modName == s.Name.Module && wf.Name == s.Name.Name {
			if err := e.writer.DeleteWorkflow(wf.ID); err != nil {
				return fmt.Errorf("failed to delete workflow: %w", err)
			}
			e.invalidateHierarchy()
			fmt.Fprintf(e.output, "Dropped workflow: %s.%s\n", s.Name.Module, s.Name.Name)
			return nil
		}
	}

	return fmt.Errorf("workflow not found: %s.%s", s.Name.Module, s.Name.Name)
}

// generateWorkflowUUID generates a UUID for workflow elements.
func generateWorkflowUUID() string {
	return mpr.GenerateID()
}

// buildWorkflowActivities converts AST activity nodes to SDK workflow activities.
func buildWorkflowActivities(nodes []ast.WorkflowActivityNode) []workflows.WorkflowActivity {
	var activities []workflows.WorkflowActivity
	for _, node := range nodes {
		act := buildWorkflowActivity(node)
		if act != nil {
			activities = append(activities, act)
		}
	}
	return activities
}

// buildWorkflowActivity converts a single AST activity node to an SDK workflow activity.
func buildWorkflowActivity(node ast.WorkflowActivityNode) workflows.WorkflowActivity {
	switch n := node.(type) {
	case *ast.WorkflowUserTaskNode:
		return buildUserTask(n)
	case *ast.WorkflowCallMicroflowNode:
		return buildCallMicroflowTask(n)
	case *ast.WorkflowCallWorkflowNode:
		return buildCallWorkflowActivity(n)
	case *ast.WorkflowDecisionNode:
		return buildExclusiveSplit(n)
	case *ast.WorkflowParallelSplitNode:
		return buildParallelSplit(n)
	case *ast.WorkflowJumpToNode:
		return buildJumpTo(n)
	case *ast.WorkflowWaitForTimerNode:
		return buildWaitForTimer(n)
	case *ast.WorkflowWaitForNotificationNode:
		return buildWaitForNotification(n)
	case *ast.WorkflowEndNode:
		return buildEndWorkflow(n)
	default:
		return nil
	}
}

func buildUserTask(n *ast.WorkflowUserTaskNode) *workflows.UserTask {
	task := &workflows.UserTask{}
	task.ID = model.ID(generateWorkflowUUID())
	task.Name = n.Name
	task.Caption = n.Caption

	if n.Page.Module != "" {
		task.Page = n.Page.Module + "." + n.Page.Name
	}

	if n.Entity.Module != "" {
		task.UserTaskEntity = n.Entity.Module + "." + n.Entity.Name
	}

	// Targeting
	switch n.Targeting.Kind {
	case "microflow":
		task.UserSource = &workflows.MicroflowBasedUserSource{
			Microflow: n.Targeting.Microflow.Module + "." + n.Targeting.Microflow.Name,
		}
	case "xpath":
		task.UserSource = &workflows.XPathBasedUserSource{
			XPath: n.Targeting.XPath,
		}
	}

	// Outcomes
	for _, outcomeNode := range n.Outcomes {
		outcome := &workflows.UserTaskOutcome{
			Name:    outcomeNode.Caption,
			Caption: outcomeNode.Caption,
		}
		outcome.ID = model.ID(generateWorkflowUUID())

		if len(outcomeNode.Activities) > 0 {
			outcome.Flow = &workflows.Flow{
				Activities: buildWorkflowActivities(outcomeNode.Activities),
			}
			outcome.Flow.ID = model.ID(generateWorkflowUUID())
		}

		task.Outcomes = append(task.Outcomes, outcome)
	}

	return task
}

func buildCallMicroflowTask(n *ast.WorkflowCallMicroflowNode) *workflows.CallMicroflowTask {
	task := &workflows.CallMicroflowTask{}
	task.ID = model.ID(generateWorkflowUUID())
	task.Name = n.Microflow.Name
	task.Caption = n.Caption
	task.Microflow = n.Microflow.Module + "." + n.Microflow.Name

	if task.Caption == "" {
		task.Caption = task.Name
	}

	for _, outcomeNode := range n.Outcomes {
		outcome := buildConditionOutcome(outcomeNode)
		if outcome != nil {
			task.Outcomes = append(task.Outcomes, outcome)
		}
	}

	return task
}

func buildCallWorkflowActivity(n *ast.WorkflowCallWorkflowNode) *workflows.CallWorkflowActivity {
	act := &workflows.CallWorkflowActivity{}
	act.ID = model.ID(generateWorkflowUUID())
	act.Name = n.Workflow.Name
	act.Caption = n.Caption
	act.Workflow = n.Workflow.Module + "." + n.Workflow.Name

	if act.Caption == "" {
		act.Caption = act.Name
	}

	// Auto-bind $WorkflowContext parameter expression
	act.ParameterExpression = "$WorkflowContext"

	return act
}

func buildExclusiveSplit(n *ast.WorkflowDecisionNode) *workflows.ExclusiveSplitActivity {
	act := &workflows.ExclusiveSplitActivity{}
	act.ID = model.ID(generateWorkflowUUID())
	act.Expression = n.Expression
	act.Caption = n.Caption

	if act.Caption == "" {
		act.Caption = "Decision"
	}
	act.Name = act.Caption

	for _, outcomeNode := range n.Outcomes {
		outcome := buildConditionOutcome(outcomeNode)
		if outcome != nil {
			act.Outcomes = append(act.Outcomes, outcome)
		}
	}

	return act
}

func buildConditionOutcome(n ast.WorkflowConditionOutcomeNode) workflows.ConditionOutcome {
	var subFlow *workflows.Flow
	if len(n.Activities) > 0 {
		subFlow = &workflows.Flow{
			Activities: buildWorkflowActivities(n.Activities),
		}
		subFlow.ID = model.ID(generateWorkflowUUID())
	}

	switch n.Value {
	case "True":
		o := &workflows.BooleanConditionOutcome{Value: true, Flow: subFlow}
		o.ID = model.ID(generateWorkflowUUID())
		return o
	case "False":
		o := &workflows.BooleanConditionOutcome{Value: false, Flow: subFlow}
		o.ID = model.ID(generateWorkflowUUID())
		return o
	case "Default":
		o := &workflows.VoidConditionOutcome{Flow: subFlow}
		o.ID = model.ID(generateWorkflowUUID())
		return o
	default:
		// Enumeration value
		o := &workflows.EnumerationValueConditionOutcome{Value: n.Value, Flow: subFlow}
		o.ID = model.ID(generateWorkflowUUID())
		return o
	}
}

func buildParallelSplit(n *ast.WorkflowParallelSplitNode) *workflows.ParallelSplitActivity {
	act := &workflows.ParallelSplitActivity{}
	act.ID = model.ID(generateWorkflowUUID())
	act.Caption = n.Caption
	if act.Caption == "" {
		act.Caption = "Parallel split"
	}
	act.Name = act.Caption

	for _, pathNode := range n.Paths {
		outcome := &workflows.ParallelSplitOutcome{}
		outcome.ID = model.ID(generateWorkflowUUID())
		if len(pathNode.Activities) > 0 {
			outcome.Flow = &workflows.Flow{
				Activities: buildWorkflowActivities(pathNode.Activities),
			}
			outcome.Flow.ID = model.ID(generateWorkflowUUID())
		}
		act.Outcomes = append(act.Outcomes, outcome)
	}

	return act
}

func buildJumpTo(n *ast.WorkflowJumpToNode) *workflows.JumpToActivity {
	act := &workflows.JumpToActivity{}
	act.ID = model.ID(generateWorkflowUUID())
	act.Name = n.Target
	act.Caption = n.Caption
	act.TargetActivity = n.Target

	if act.Caption == "" {
		act.Caption = act.Name
	}

	return act
}

func buildWaitForTimer(n *ast.WorkflowWaitForTimerNode) *workflows.WaitForTimerActivity {
	act := &workflows.WaitForTimerActivity{}
	act.ID = model.ID(generateWorkflowUUID())
	act.DelayExpression = n.DelayExpression
	act.Caption = n.Caption

	if act.Caption == "" {
		act.Caption = "Wait for timer"
	}
	act.Name = act.Caption

	return act
}

func buildWaitForNotification(n *ast.WorkflowWaitForNotificationNode) *workflows.WaitForNotificationActivity {
	act := &workflows.WaitForNotificationActivity{}
	act.ID = model.ID(generateWorkflowUUID())
	act.Caption = n.Caption

	if act.Caption == "" {
		act.Caption = "Wait for notification"
	}
	act.Name = act.Caption

	return act
}

func buildEndWorkflow(n *ast.WorkflowEndNode) *workflows.EndWorkflowActivity {
	act := &workflows.EndWorkflowActivity{}
	act.ID = model.ID(generateWorkflowUUID())
	act.Caption = n.Caption

	if act.Caption == "" {
		act.Caption = "End"
	}
	act.Name = act.Caption

	return act
}

// deduplicateActivityNames ensures all activity names within a workflow are unique.
// Mendix Studio Pro requires unique activity names (CE0495).
func deduplicateActivityNames(activities []workflows.WorkflowActivity) {
	nameCount := make(map[string]int)
	deduplicateActivityNamesInFlow(activities, nameCount)
}

// deduplicateActivityNamesInFlow recursively deduplicates activity names.
func deduplicateActivityNamesInFlow(activities []workflows.WorkflowActivity, nameCount map[string]int) {
	for _, act := range activities {
		switch a := act.(type) {
		case *workflows.UserTask:
			a.Name = uniqueName(a.Name, nameCount)
			for _, outcome := range a.Outcomes {
				if outcome.Flow != nil {
					deduplicateActivityNamesInFlow(outcome.Flow.Activities, nameCount)
				}
			}
		case *workflows.CallMicroflowTask:
			a.Name = uniqueName(a.Name, nameCount)
			for _, outcome := range a.Outcomes {
				switch o := outcome.(type) {
				case *workflows.BooleanConditionOutcome:
					if o.Flow != nil {
						deduplicateActivityNamesInFlow(o.Flow.Activities, nameCount)
					}
				case *workflows.EnumerationValueConditionOutcome:
					if o.Flow != nil {
						deduplicateActivityNamesInFlow(o.Flow.Activities, nameCount)
					}
				case *workflows.VoidConditionOutcome:
					if o.Flow != nil {
						deduplicateActivityNamesInFlow(o.Flow.Activities, nameCount)
					}
				}
			}
		case *workflows.CallWorkflowActivity:
			a.Name = uniqueName(a.Name, nameCount)
		case *workflows.ExclusiveSplitActivity:
			a.Name = uniqueName(a.Name, nameCount)
			for _, outcome := range a.Outcomes {
				switch o := outcome.(type) {
				case *workflows.BooleanConditionOutcome:
					if o.Flow != nil {
						deduplicateActivityNamesInFlow(o.Flow.Activities, nameCount)
					}
				case *workflows.EnumerationValueConditionOutcome:
					if o.Flow != nil {
						deduplicateActivityNamesInFlow(o.Flow.Activities, nameCount)
					}
				case *workflows.VoidConditionOutcome:
					if o.Flow != nil {
						deduplicateActivityNamesInFlow(o.Flow.Activities, nameCount)
					}
				}
			}
		case *workflows.ParallelSplitActivity:
			a.Name = uniqueName(a.Name, nameCount)
			for _, outcome := range a.Outcomes {
				if outcome.Flow != nil {
					deduplicateActivityNamesInFlow(outcome.Flow.Activities, nameCount)
				}
			}
		case *workflows.JumpToActivity:
			a.Name = uniqueName(a.Name, nameCount)
		case *workflows.WaitForTimerActivity:
			a.Name = uniqueName(a.Name, nameCount)
		case *workflows.WaitForNotificationActivity:
			a.Name = uniqueName(a.Name, nameCount)
		case *workflows.EndWorkflowActivity:
			a.Name = uniqueName(a.Name, nameCount)
		}
	}
}

// uniqueName returns a unique name by appending a number if the name was seen before.
func uniqueName(name string, nameCount map[string]int) string {
	nameCount[name]++
	count := nameCount[name]
	if count == 1 {
		return name
	}
	return fmt.Sprintf("%s%d", name, count)
}

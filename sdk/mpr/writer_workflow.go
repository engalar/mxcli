// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/workflows"

	"go.mongodb.org/mongo-driver/bson"
)

// CreateWorkflow creates a new workflow in the MPR.
func (w *Writer) CreateWorkflow(wf *workflows.Workflow) error {
	if wf.ID == "" {
		wf.ID = model.ID(generateUUID())
	}
	wf.TypeName = "Workflows$Workflow"

	contents, err := w.serializeWorkflow(wf)
	if err != nil {
		return fmt.Errorf("failed to serialize workflow: %w", err)
	}

	return w.insertUnit(string(wf.ID), string(wf.ContainerID), "Documents", "Workflows$Workflow", contents)
}

// DeleteWorkflow deletes a workflow from the MPR.
func (w *Writer) DeleteWorkflow(id model.ID) error {
	return w.deleteUnit(string(id))
}

func (w *Writer) serializeWorkflow(wf *workflows.Workflow) ([]byte, error) {
	doc := bson.D{
		{Key: "$ID", Value: idToBsonBinary(string(wf.ID))},
		{Key: "$Type", Value: "Workflows$Workflow"},
		{Key: "AllowedModuleRoles", Value: allowedModuleRolesArray(wf.AllowedModuleRoles)},
		{Key: "Documentation", Value: wf.Documentation},
		{Key: "DueDate", Value: wf.DueDate},
		{Key: "Excluded", Value: wf.Excluded},
		{Key: "ExportLevel", Value: "Hidden"},
	}

	// Flow
	if wf.Flow != nil {
		doc = append(doc, bson.E{Key: "Flow", Value: serializeWorkflowFlow(wf.Flow)})
	} else {
		// Empty flow
		emptyFlow := &workflows.Flow{}
		emptyFlow.ID = model.ID(generateUUID())
		doc = append(doc, bson.E{Key: "Flow", Value: serializeWorkflowFlow(emptyFlow)})
	}

	doc = append(doc, bson.E{Key: "Name", Value: wf.Name})

	// OverviewPage (BY_NAME reference)
	doc = append(doc, bson.E{Key: "OverviewPage", Value: wf.OverviewPage})

	// Parameter
	if wf.Parameter != nil {
		doc = append(doc, bson.E{Key: "Parameter", Value: serializeWorkflowParameter(wf.Parameter)})
	}

	// WorkflowDescription (StringTemplate)
	doc = append(doc, bson.E{Key: "WorkflowDescription", Value: serializeWorkflowStringTemplate(wf.WorkflowDescription)})

	// WorkflowName (StringTemplate)
	doc = append(doc, bson.E{Key: "WorkflowName", Value: serializeWorkflowStringTemplate(wf.WorkflowName)})

	return bson.Marshal(doc)
}

// serializeWorkflowStringTemplate creates a minimal Mendix StringTemplate BSON structure for workflows.
func serializeWorkflowStringTemplate(text string) bson.D {
	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(generateUUID())},
		{Key: "$Type", Value: "Texts$StringTemplate"},
		{Key: "Parameters", Value: bson.A{int32(3)}},
		{Key: "Text", Value: text},
	}
}

// serializeWorkflowParameter serializes a workflow parameter.
func serializeWorkflowParameter(param *workflows.WorkflowParameter) bson.D {
	paramID := string(param.ID)
	if paramID == "" {
		paramID = generateUUID()
	}
	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(paramID)},
		{Key: "$Type", Value: "Workflows$Parameter"},
		{Key: "EntityRef", Value: bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DomainModels$IndirectEntityRef"},
			{Key: "EntityQualifiedName", Value: param.EntityRef},
		}},
	}
}

// serializeWorkflowFlow serializes a workflow flow with its activities.
func serializeWorkflowFlow(flow *workflows.Flow) bson.D {
	flowID := string(flow.ID)
	if flowID == "" {
		flowID = generateUUID()
	}

	activities := bson.A{int32(3)} // array type marker
	for _, act := range flow.Activities {
		actDoc := serializeWorkflowActivity(act)
		if actDoc != nil {
			activities = append(activities, actDoc)
		}
	}

	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(flowID)},
		{Key: "$Type", Value: "Workflows$Flow"},
		{Key: "Activities", Value: activities},
	}
}

// serializeWorkflowActivity dispatches to the correct serializer.
func serializeWorkflowActivity(act workflows.WorkflowActivity) bson.D {
	switch a := act.(type) {
	case *workflows.UserTask:
		return serializeUserTask(a)
	case *workflows.CallMicroflowTask:
		return serializeCallMicroflowTask(a)
	case *workflows.CallWorkflowActivity:
		return serializeCallWorkflowActivity(a)
	case *workflows.ExclusiveSplitActivity:
		return serializeExclusiveSplit(a)
	case *workflows.ParallelSplitActivity:
		return serializeParallelSplit(a)
	case *workflows.JumpToActivity:
		return serializeJumpTo(a)
	case *workflows.WaitForTimerActivity:
		return serializeWaitForTimer(a)
	case *workflows.WaitForNotificationActivity:
		return serializeWaitForNotification(a)
	case *workflows.StartWorkflowActivity:
		return serializeStartWorkflow(a)
	case *workflows.EndWorkflowActivity:
		return serializeEndWorkflow(a)
	default:
		return nil
	}
}

func activityID(a *workflows.BaseWorkflowActivity) string {
	if string(a.ID) != "" {
		return string(a.ID)
	}
	return generateUUID()
}

func serializeUserTask(a *workflows.UserTask) bson.D {
	doc := bson.D{
		{Key: "$ID", Value: idToBsonBinary(activityID(&a.BaseWorkflowActivity))},
		{Key: "$Type", Value: "Workflows$UserTask"},
		{Key: "Caption", Value: a.Caption},
		{Key: "Name", Value: a.Name},
	}

	// Outcomes
	outcomes := bson.A{int32(3)}
	for _, outcome := range a.Outcomes {
		outcomes = append(outcomes, serializeUserTaskOutcome(outcome))
	}
	doc = append(doc, bson.E{Key: "Outcomes", Value: outcomes})

	// Page
	doc = append(doc, bson.E{Key: "Page", Value: a.Page})

	// TaskDescription
	doc = append(doc, bson.E{Key: "TaskDescription", Value: serializeWorkflowStringTemplate(a.TaskDescription)})

	// TaskName
	doc = append(doc, bson.E{Key: "TaskName", Value: serializeWorkflowStringTemplate(a.TaskName)})

	// UserSource
	if a.UserSource != nil {
		doc = append(doc, bson.E{Key: "UserSource", Value: serializeUserSource(a.UserSource)})
	}

	// UserTaskEntity
	if a.UserTaskEntity != "" {
		doc = append(doc, bson.E{Key: "UserTaskEntity", Value: a.UserTaskEntity})
	}

	return doc
}

func serializeUserTaskOutcome(outcome *workflows.UserTaskOutcome) bson.D {
	outcomeID := string(outcome.ID)
	if outcomeID == "" {
		outcomeID = generateUUID()
	}

	doc := bson.D{
		{Key: "$ID", Value: idToBsonBinary(outcomeID)},
		{Key: "$Type", Value: "Workflows$UserTaskOutcome"},
		{Key: "Caption", Value: outcome.Caption},
		{Key: "Name", Value: outcome.Name},
	}

	if outcome.Flow != nil {
		doc = append(doc, bson.E{Key: "Flow", Value: serializeWorkflowFlow(outcome.Flow)})
	}

	return doc
}

func serializeUserSource(source workflows.UserSource) bson.D {
	switch s := source.(type) {
	case *workflows.MicroflowBasedUserSource:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "Workflows$MicroflowBasedUserSource"},
			{Key: "Microflow", Value: s.Microflow},
		}
	case *workflows.XPathBasedUserSource:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "Workflows$XPathBasedUserSource"},
			{Key: "XPath", Value: s.XPath},
		}
	default:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "Workflows$EmptyUserSource"},
		}
	}
}

func serializeCallMicroflowTask(a *workflows.CallMicroflowTask) bson.D {
	doc := bson.D{
		{Key: "$ID", Value: idToBsonBinary(activityID(&a.BaseWorkflowActivity))},
		{Key: "$Type", Value: "Workflows$CallMicroflowTask"},
		{Key: "Caption", Value: a.Caption},
		{Key: "Microflow", Value: a.Microflow},
		{Key: "Name", Value: a.Name},
	}

	outcomes := bson.A{int32(3)}
	for _, outcome := range a.Outcomes {
		outcomes = append(outcomes, serializeConditionOutcome(outcome))
	}
	doc = append(doc, bson.E{Key: "Outcomes", Value: outcomes})

	return doc
}

func serializeCallWorkflowActivity(a *workflows.CallWorkflowActivity) bson.D {
	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(activityID(&a.BaseWorkflowActivity))},
		{Key: "$Type", Value: "Workflows$CallWorkflowActivity"},
		{Key: "Caption", Value: a.Caption},
		{Key: "Name", Value: a.Name},
		{Key: "Workflow", Value: a.Workflow},
	}
}

func serializeExclusiveSplit(a *workflows.ExclusiveSplitActivity) bson.D {
	doc := bson.D{
		{Key: "$ID", Value: idToBsonBinary(activityID(&a.BaseWorkflowActivity))},
		{Key: "$Type", Value: "Workflows$ExclusiveSplitActivity"},
		{Key: "Caption", Value: a.Caption},
		{Key: "Expression", Value: a.Expression},
		{Key: "Name", Value: a.Name},
	}

	outcomes := bson.A{int32(3)}
	for _, outcome := range a.Outcomes {
		outcomes = append(outcomes, serializeConditionOutcome(outcome))
	}
	doc = append(doc, bson.E{Key: "Outcomes", Value: outcomes})

	return doc
}

func serializeConditionOutcome(outcome workflows.ConditionOutcome) bson.D {
	switch o := outcome.(type) {
	case *workflows.BooleanConditionOutcome:
		outcomeID := string(o.ID)
		if outcomeID == "" {
			outcomeID = generateUUID()
		}
		doc := bson.D{
			{Key: "$ID", Value: idToBsonBinary(outcomeID)},
			{Key: "$Type", Value: "Workflows$BooleanConditionOutcome"},
			{Key: "Value", Value: o.Value},
		}
		if o.Flow != nil {
			doc = append(doc, bson.E{Key: "Flow", Value: serializeWorkflowFlow(o.Flow)})
		}
		return doc
	case *workflows.EnumerationValueConditionOutcome:
		outcomeID := string(o.ID)
		if outcomeID == "" {
			outcomeID = generateUUID()
		}
		doc := bson.D{
			{Key: "$ID", Value: idToBsonBinary(outcomeID)},
			{Key: "$Type", Value: "Workflows$EnumerationValueConditionOutcome"},
			{Key: "Value", Value: o.Value},
		}
		if o.Flow != nil {
			doc = append(doc, bson.E{Key: "Flow", Value: serializeWorkflowFlow(o.Flow)})
		}
		return doc
	case *workflows.VoidConditionOutcome:
		outcomeID := string(o.ID)
		if outcomeID == "" {
			outcomeID = generateUUID()
		}
		doc := bson.D{
			{Key: "$ID", Value: idToBsonBinary(outcomeID)},
			{Key: "$Type", Value: "Workflows$VoidConditionOutcome"},
		}
		if o.Flow != nil {
			doc = append(doc, bson.E{Key: "Flow", Value: serializeWorkflowFlow(o.Flow)})
		}
		return doc
	default:
		return nil
	}
}

func serializeParallelSplit(a *workflows.ParallelSplitActivity) bson.D {
	doc := bson.D{
		{Key: "$ID", Value: idToBsonBinary(activityID(&a.BaseWorkflowActivity))},
		{Key: "$Type", Value: "Workflows$ParallelSplitActivity"},
		{Key: "Caption", Value: a.Caption},
		{Key: "Name", Value: a.Name},
	}

	outcomes := bson.A{int32(3)}
	for _, outcome := range a.Outcomes {
		outcomeID := string(outcome.ID)
		if outcomeID == "" {
			outcomeID = generateUUID()
		}
		outDoc := bson.D{
			{Key: "$ID", Value: idToBsonBinary(outcomeID)},
			{Key: "$Type", Value: "Workflows$ParallelSplitOutcome"},
		}
		if outcome.Flow != nil {
			outDoc = append(outDoc, bson.E{Key: "Flow", Value: serializeWorkflowFlow(outcome.Flow)})
		}
		outcomes = append(outcomes, outDoc)
	}
	doc = append(doc, bson.E{Key: "Outcomes", Value: outcomes})

	return doc
}

func serializeJumpTo(a *workflows.JumpToActivity) bson.D {
	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(activityID(&a.BaseWorkflowActivity))},
		{Key: "$Type", Value: "Workflows$JumpToActivity"},
		{Key: "Caption", Value: a.Caption},
		{Key: "Name", Value: a.Name},
		{Key: "TargetActivity", Value: a.TargetActivity},
	}
}

func serializeWaitForTimer(a *workflows.WaitForTimerActivity) bson.D {
	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(activityID(&a.BaseWorkflowActivity))},
		{Key: "$Type", Value: "Workflows$WaitForTimerActivity"},
		{Key: "Caption", Value: a.Caption},
		{Key: "DelayExpression", Value: a.DelayExpression},
		{Key: "Name", Value: a.Name},
	}
}

func serializeWaitForNotification(a *workflows.WaitForNotificationActivity) bson.D {
	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(activityID(&a.BaseWorkflowActivity))},
		{Key: "$Type", Value: "Workflows$WaitForNotificationActivity"},
		{Key: "Caption", Value: a.Caption},
		{Key: "Name", Value: a.Name},
	}
}

func serializeStartWorkflow(a *workflows.StartWorkflowActivity) bson.D {
	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(activityID(&a.BaseWorkflowActivity))},
		{Key: "$Type", Value: "Workflows$StartWorkflowActivity"},
		{Key: "Caption", Value: a.Caption},
		{Key: "Name", Value: a.Name},
	}
}

func serializeEndWorkflow(a *workflows.EndWorkflowActivity) bson.D {
	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(activityID(&a.BaseWorkflowActivity))},
		{Key: "$Type", Value: "Workflows$EndWorkflowActivity"},
		{Key: "Caption", Value: a.Caption},
		{Key: "Name", Value: a.Name},
	}
}

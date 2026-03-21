// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/sdk/workflows"
	"go.mongodb.org/mongo-driver/bson"
)

// loadWorkflowBSON loads a workflow BSON fixture from testdata/workflows/<name>.bson.
func loadWorkflowBSON(t *testing.T, name string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "workflows", name+".bson"))
	if err != nil {
		t.Fatalf("load fixture %s: %v", name, err)
	}
	var raw map[string]any
	if err := bson.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal fixture %s: %v", name, err)
	}
	return raw
}

// workflowActivities returns the activity slice (skipping the array marker) from a flow map.
func workflowActivities(t *testing.T, flowRaw map[string]any) []map[string]any {
	t.Helper()
	arr, ok := flowRaw["Activities"].(bson.A)
	if !ok {
		t.Fatalf("Activities is not bson.A, got %T", flowRaw["Activities"])
	}
	var acts []map[string]any
	for _, item := range arr[1:] { // skip marker at index 0
		m := toMap(item)
		if m != nil {
			acts = append(acts, m)
		}
	}
	return acts
}

func TestParseWorkflowParameter_FromFixture(t *testing.T) {
	raw := loadWorkflowBSON(t, "WorkflowBaseline.Workflow")
	paramRaw := toMap(raw["Parameter"])
	if paramRaw == nil {
		t.Fatal("fixture has no Parameter")
	}

	param := parseWorkflowParameter(paramRaw)
	if param == nil {
		t.Fatal("parseWorkflowParameter returned nil")
	}
	if param.EntityRef != "WorkflowBaseline.Entity" {
		t.Errorf("EntityRef = %q, want %q", param.EntityRef, "WorkflowBaseline.Entity")
	}
}

func TestParseWorkflowFlow_FromFixture_ActivityCount(t *testing.T) {
	raw := loadWorkflowBSON(t, "WorkflowBaseline.Workflow")
	flowRaw := toMap(raw["Flow"])
	if flowRaw == nil {
		t.Fatal("fixture has no Flow")
	}

	flow := parseWorkflowFlow(flowRaw)
	if flow == nil {
		t.Fatal("parseWorkflowFlow returned nil")
	}
	// Fixture has: Start, SingleUserTask, MultiUserTask, CallMicroflow, ParallelSplit, ExclusiveSplit, End
	if len(flow.Activities) != 7 {
		t.Errorf("len(Activities) = %d, want 7", len(flow.Activities))
	}
}

func TestParseWorkflowActivity_FromFixture_StartIsFirst(t *testing.T) {
	raw := loadWorkflowBSON(t, "WorkflowBaseline.Workflow")
	flowRaw := toMap(raw["Flow"])
	acts := workflowActivities(t, flowRaw)

	activity := parseWorkflowActivity(acts[0])
	if _, ok := activity.(*workflows.StartWorkflowActivity); !ok {
		t.Errorf("activities[0] = %T, want *workflows.StartWorkflowActivity", activity)
	}
}

func TestParseWorkflowActivity_FromFixture_UserTask(t *testing.T) {
	raw := loadWorkflowBSON(t, "WorkflowBaseline.Workflow")
	flowRaw := toMap(raw["Flow"])
	acts := workflowActivities(t, flowRaw)

	// activities[1] is SingleUserTaskActivity in fixture
	activity := parseWorkflowActivity(acts[1])
	userTask, ok := activity.(*workflows.UserTask)
	if !ok {
		t.Fatalf("activities[1] = %T, want *workflows.UserTask", activity)
	}
	if userTask.Name != "userTask1" {
		t.Errorf("Name = %q, want %q", userTask.Name, "userTask1")
	}
}

func TestParseWorkflowActivity_FromFixture_CallMicroflow(t *testing.T) {
	raw := loadWorkflowBSON(t, "WorkflowBaseline.Workflow")
	flowRaw := toMap(raw["Flow"])
	acts := workflowActivities(t, flowRaw)

	// activities[3] is CallMicroflowTask in fixture
	activity := parseWorkflowActivity(acts[3])
	callMf, ok := activity.(*workflows.CallMicroflowTask)
	if !ok {
		t.Fatalf("activities[3] = %T, want *workflows.CallMicroflowTask", activity)
	}
	if callMf.Microflow != "WorkflowBaseline.Microflow" {
		t.Errorf("Microflow = %q, want %q", callMf.Microflow, "WorkflowBaseline.Microflow")
	}
}

func TestParseWorkflowActivity_FromFixture_EndIsLast(t *testing.T) {
	raw := loadWorkflowBSON(t, "WorkflowBaseline.Workflow")
	flowRaw := toMap(raw["Flow"])
	acts := workflowActivities(t, flowRaw)

	last := parseWorkflowActivity(acts[len(acts)-1])
	if _, ok := last.(*workflows.EndWorkflowActivity); !ok {
		t.Errorf("last activity = %T, want *workflows.EndWorkflowActivity", last)
	}
}

func TestParseUserTaskOutcome_FromFixture(t *testing.T) {
	raw := loadWorkflowBSON(t, "WorkflowBaseline.Workflow")
	flowRaw := toMap(raw["Flow"])
	acts := workflowActivities(t, flowRaw)

	// activities[1] is SingleUserTaskActivity with one outcome
	outcomesRaw := acts[1]["Outcomes"]
	arr, ok := outcomesRaw.(bson.A)
	if !ok || len(arr) < 2 {
		t.Fatalf("expected Outcomes array with marker+1 element, got %T len=%d", outcomesRaw, len(arr))
	}
	outcomeMap := toMap(arr[1]) // skip marker
	if outcomeMap == nil {
		t.Fatal("outcome element is nil")
	}

	outcome := parseUserTaskOutcome(outcomeMap)
	if outcome == nil {
		t.Fatal("parseUserTaskOutcome returned nil")
	}
	if outcome.Value != "Outcome" {
		t.Errorf("Value = %q, want %q", outcome.Value, "Outcome")
	}
}

func TestParseParameterMappings_FromFixture(t *testing.T) {
	raw := loadWorkflowBSON(t, "WorkflowBaseline.Workflow")
	flowRaw := toMap(raw["Flow"])
	acts := workflowActivities(t, flowRaw)

	// activities[3] is CallMicroflowTask with 1 parameter mapping
	mappingsRaw := acts[3]["ParameterMappings"]
	mappings := parseParameterMappings(mappingsRaw)
	if len(mappings) != 1 {
		t.Fatalf("len(mappings) = %d, want 1", len(mappings))
	}
}

func TestParseWorkflowFlow_FromFixture_SubWorkflow(t *testing.T) {
	raw := loadWorkflowBSON(t, "WorkflowBaseline.Sub_Workflow")
	flowRaw := toMap(raw["Flow"])
	if flowRaw == nil {
		t.Fatal("Sub_Workflow fixture has no Flow")
	}

	flow := parseWorkflowFlow(flowRaw)
	if flow == nil {
		t.Fatal("parseWorkflowFlow returned nil")
	}
	if len(flow.Activities) != 2 {
		t.Errorf("Sub_Workflow has %d activities, want exactly 2 (Start+End)", len(flow.Activities))
	}
}

func TestParseWorkflowParameter_Nil(t *testing.T) {
	param := parseWorkflowParameter(nil)
	if param != nil {
		t.Errorf("parseWorkflowParameter(nil) = %v, want nil", param)
	}
}

func TestParseWorkflowActivity_UnknownType(t *testing.T) {
	raw := map[string]any{
		"$Type": "Workflows$SomeUnknownFutureActivity",
		"$ID":   "abc123",
		"Name":  "mystery",
	}
	activity := parseWorkflowActivity(raw)
	generic, ok := activity.(*workflows.GenericWorkflowActivity)
	if !ok {
		t.Fatalf("unknown type = %T, want *workflows.GenericWorkflowActivity", activity)
	}
	if generic.Name != "mystery" {
		t.Errorf("Name = %q, want %q", generic.Name, "mystery")
	}
}

func TestParseBoundaryEvents_EmptyArray(t *testing.T) {
	// nil input
	events := parseBoundaryEvents(nil)
	if len(events) != 0 {
		t.Errorf("parseBoundaryEvents(nil) len = %d, want 0", len(events))
	}
	// marker-only array (bson.A with just the int32 marker)
	events = parseBoundaryEvents(bson.A{int32(2)})
	if len(events) != 0 {
		t.Errorf("parseBoundaryEvents(marker-only) len = %d, want 0", len(events))
	}
}

func TestParseBoundaryEvents_TimerEvent(t *testing.T) {
	eventMap := map[string]any{
		"$Type":              "Workflows$InterruptingTimerBoundaryEvent",
		"$ID":                "be-001",
		"Caption":            "Timeout",
		"FirstExecutionTime": "PT1H",
	}
	events := parseBoundaryEvents(bson.A{int32(2), eventMap})
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	ev := events[0]
	if ev.EventType != "InterruptingTimer" {
		t.Errorf("EventType = %q, want %q", ev.EventType, "InterruptingTimer")
	}
	if ev.TimerDelay != "PT1H" {
		t.Errorf("TimerDelay = %q, want %q", ev.TimerDelay, "PT1H")
	}
	if ev.Caption != "Timeout" {
		t.Errorf("Caption = %q, want %q", ev.Caption, "Timeout")
	}
}

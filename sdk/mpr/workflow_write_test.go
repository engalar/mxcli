// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/workflows"
	"go.mongodb.org/mongo-driver/bson"
)

func getBSONField(doc bson.D, key string) any {
	for _, e := range doc {
		if e.Key == key {
			return e.Value
		}
	}
	return nil
}

func TestSerializeWorkflowFlow_ActivitiesMarker(t *testing.T) {
	flow := &workflows.Flow{
		BaseElement: model.BaseElement{ID: "flow-1"},
		Activities: []workflows.WorkflowActivity{
			&workflows.StartWorkflowActivity{
				BaseWorkflowActivity: workflows.BaseWorkflowActivity{
					BaseElement: model.BaseElement{ID: "start-1"},
					Name:        "Start",
				},
			},
		},
	}

	doc := serializeWorkflowFlow(flow)
	activities := getBSONField(doc, "Activities")
	arr, ok := activities.(bson.A)
	if !ok {
		t.Fatalf("Activities is not bson.A, got %T", activities)
	}
	if len(arr) < 1 {
		t.Fatal("Activities array is empty")
	}
	marker, ok := arr[0].(int32)
	if !ok {
		t.Fatalf("Activities[0] is not int32, got %T", arr[0])
	}
	if marker != int32(3) {
		t.Errorf("Activities[0] = %d, want 3", marker)
	}
	if len(arr) != 2 {
		t.Errorf("Activities length = %d, want 2 (marker + 1 activity)", len(arr))
	}
}

func TestSerializeWorkflowFlow_EmptyActivities(t *testing.T) {
	flow := &workflows.Flow{
		BaseElement: model.BaseElement{ID: "flow-empty"},
	}

	doc := serializeWorkflowFlow(flow)
	activities := getBSONField(doc, "Activities")
	arr, ok := activities.(bson.A)
	if !ok {
		t.Fatalf("Activities is not bson.A, got %T", activities)
	}
	if len(arr) != 1 {
		t.Fatalf("Activities length = %d, want 1 (marker only)", len(arr))
	}
	marker, ok := arr[0].(int32)
	if !ok {
		t.Fatalf("Activities[0] is not int32, got %T", arr[0])
	}
	if marker != int32(3) {
		t.Errorf("Activities[0] = %d, want 3", marker)
	}
}

func TestSerializeUserTask_OutcomesMarker(t *testing.T) {
	task := &workflows.UserTask{
		BaseWorkflowActivity: workflows.BaseWorkflowActivity{
			BaseElement: model.BaseElement{ID: "ut-1"},
			Name:        "ReviewTask",
			Caption:     "Review",
		},
		Outcomes: []*workflows.UserTaskOutcome{
			{
				BaseElement: model.BaseElement{ID: "outcome-1"},
				Value:       "Approve",
			},
		},
	}

	doc := serializeUserTask(task)
	outcomes := getBSONField(doc, "Outcomes")
	arr, ok := outcomes.(bson.A)
	if !ok {
		t.Fatalf("Outcomes is not bson.A, got %T", outcomes)
	}
	if len(arr) < 1 {
		t.Fatal("Outcomes array is empty")
	}
	marker, ok := arr[0].(int32)
	if !ok {
		t.Fatalf("Outcomes[0] is not int32, got %T", arr[0])
	}
	if marker != int32(3) {
		t.Errorf("Outcomes[0] = %d, want 3", marker)
	}
}

func TestSerializeUserTask_BoundaryEventsMarker(t *testing.T) {
	task := &workflows.UserTask{
		BaseWorkflowActivity: workflows.BaseWorkflowActivity{
			BaseElement: model.BaseElement{ID: "ut-2"},
			Name:        "Task",
			Caption:     "Task",
		},
	}

	doc := serializeUserTask(task)
	boundaryEvents := getBSONField(doc, "BoundaryEvents")
	arr, ok := boundaryEvents.(bson.A)
	if !ok {
		t.Fatalf("BoundaryEvents is not bson.A, got %T", boundaryEvents)
	}
	if len(arr) < 1 {
		t.Fatal("BoundaryEvents array is empty")
	}
	marker, ok := arr[0].(int32)
	if !ok {
		t.Fatalf("BoundaryEvents[0] is not int32, got %T", arr[0])
	}
	if marker != int32(2) {
		t.Errorf("BoundaryEvents[0] = %d, want 2", marker)
	}
}

func TestSerializeCallMicroflowTask_ParameterMappingsMarker(t *testing.T) {
	task := &workflows.CallMicroflowTask{
		BaseWorkflowActivity: workflows.BaseWorkflowActivity{
			BaseElement: model.BaseElement{ID: "cmt-1"},
			Name:        "CallMF",
			Caption:     "Call Microflow",
		},
		Microflow: "MyModule.DoSomething",
		ParameterMappings: []*workflows.ParameterMapping{
			{
				BaseElement: model.BaseElement{ID: "pm-1"},
				Parameter:   "InputParam",
				Expression:  "$WorkflowContext",
			},
		},
	}

	doc := serializeCallMicroflowTask(task)
	mappings := getBSONField(doc, "ParameterMappings")
	arr, ok := mappings.(bson.A)
	if !ok {
		t.Fatalf("ParameterMappings is not bson.A, got %T", mappings)
	}
	if len(arr) < 1 {
		t.Fatal("ParameterMappings array is empty")
	}
	marker, ok := arr[0].(int32)
	if !ok {
		t.Fatalf("ParameterMappings[0] is not int32, got %T", arr[0])
	}
	if marker != int32(2) {
		t.Errorf("ParameterMappings[0] = %d, want 2", marker)
	}
	if len(arr) != 2 {
		t.Errorf("ParameterMappings length = %d, want 2 (marker + 1 mapping)", len(arr))
	}
}

func TestSerializeUserTaskOutcome_ValueField(t *testing.T) {
	outcome := &workflows.UserTaskOutcome{
		BaseElement: model.BaseElement{ID: "uto-1"},
		Value:       "Approve",
	}

	doc := serializeUserTaskOutcome(outcome)

	// Must have "Value" key
	val := getBSONField(doc, "Value")
	if val == nil {
		t.Fatal("missing 'Value' field in UserTaskOutcome BSON")
	}
	if val != "Approve" {
		t.Errorf("Value = %v, want %q", val, "Approve")
	}

	// Must NOT have "Caption" or "Name" keys
	if getBSONField(doc, "Caption") != nil {
		t.Error("UserTaskOutcome should not have 'Caption' key")
	}
	if getBSONField(doc, "Name") != nil {
		t.Error("UserTaskOutcome should not have 'Name' key")
	}
}

func TestSerializeWorkflowParameter_EntityAsString(t *testing.T) {
	param := &workflows.WorkflowParameter{
		BaseElement: model.BaseElement{ID: "param-1"},
		EntityRef:   "MyModule.Customer",
	}

	doc := serializeWorkflowParameter(param)

	entity := getBSONField(doc, "Entity")
	if entity == nil {
		t.Fatal("missing 'Entity' field in WorkflowParameter BSON")
	}
	entityStr, ok := entity.(string)
	if !ok {
		t.Fatalf("Entity is %T, want string", entity)
	}
	if entityStr != "MyModule.Customer" {
		t.Errorf("Entity = %q, want %q", entityStr, "MyModule.Customer")
	}
}

func TestSerializeWorkflowFlow_Roundtrip(t *testing.T) {
	flow := &workflows.Flow{
		BaseElement: model.BaseElement{ID: "flow-rt"},
		Activities: []workflows.WorkflowActivity{
			&workflows.StartWorkflowActivity{
				BaseWorkflowActivity: workflows.BaseWorkflowActivity{
					BaseElement: model.BaseElement{ID: "start-rt"},
					Name:        "Start",
					Caption:     "Start",
				},
			},
			&workflows.UserTask{
				BaseWorkflowActivity: workflows.BaseWorkflowActivity{
					BaseElement: model.BaseElement{ID: "ut-rt"},
					Name:        "ReviewTask",
					Caption:     "Review",
				},
				Page: "MyModule.TaskPage",
				Outcomes: []*workflows.UserTaskOutcome{
					{
						BaseElement: model.BaseElement{ID: "out-rt"},
						Value:       "Approve",
					},
				},
			},
			&workflows.EndWorkflowActivity{
				BaseWorkflowActivity: workflows.BaseWorkflowActivity{
					BaseElement: model.BaseElement{ID: "end-rt"},
					Name:        "End",
					Caption:     "End",
				},
			},
		},
	}

	// Serialize
	doc := serializeWorkflowFlow(flow)

	// Marshal to BSON bytes
	data, err := bson.Marshal(doc)
	if err != nil {
		t.Fatalf("failed to marshal BSON: %v", err)
	}

	// Unmarshal to map
	var raw map[string]any
	if err := bson.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal BSON: %v", err)
	}

	// Parse back
	parsed := parseWorkflowFlow(raw)
	if parsed == nil {
		t.Fatal("parseWorkflowFlow returned nil")
	}

	if len(parsed.Activities) != 3 {
		t.Errorf("parsed Activities count = %d, want 3", len(parsed.Activities))
	}
}

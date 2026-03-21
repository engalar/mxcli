// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"testing"

	"github.com/mendixlabs/mxcli/sdk/workflows"
	"go.mongodb.org/mongo-driver/bson"
)

func TestParseWorkflowParameter_EntityRef(t *testing.T) {
	raw := map[string]any{
		"Entity": "MyModule.MyEntity",
	}
	param := parseWorkflowParameter(raw)
	if param == nil {
		t.Fatal("expected non-nil parameter")
	}
	if param.EntityRef != "MyModule.MyEntity" {
		t.Errorf("EntityRef = %q, want %q", param.EntityRef, "MyModule.MyEntity")
	}
}

func TestParseWorkflowParameter_EntityRefNested(t *testing.T) {
	raw := map[string]any{
		"EntityRef": map[string]any{
			"EntityQualifiedName": "Sales.Order",
		},
	}
	param := parseWorkflowParameter(raw)
	if param == nil {
		t.Fatal("expected non-nil parameter")
	}
	if param.EntityRef != "Sales.Order" {
		t.Errorf("EntityRef = %q, want %q", param.EntityRef, "Sales.Order")
	}
}

func TestParseWorkflowParameter_Nil(t *testing.T) {
	param := parseWorkflowParameter(nil)
	if param != nil {
		t.Error("expected nil for nil input")
	}
}

func TestParseWorkflowFlow_ActivityCount(t *testing.T) {
	raw := map[string]any{
		"Activities": bson.A{
			int32(3), // array marker
			map[string]any{
				"$Type": "Workflows$StartWorkflowActivity",
				"Name":  "Start",
			},
			map[string]any{
				"$Type": "Workflows$EndWorkflowActivity",
				"Name":  "End",
			},
		},
	}
	flow := parseWorkflowFlow(raw)
	if flow == nil {
		t.Fatal("expected non-nil flow")
	}
	if len(flow.Activities) != 2 {
		t.Fatalf("len(Activities) = %d, want 2", len(flow.Activities))
	}
}

func TestParseWorkflowActivity_UserTask(t *testing.T) {
	raw := map[string]any{
		"$Type": "Workflows$SingleUserTaskActivity",
		"Name":  "ReviewOrder",
		"Page":  "Sales.ReviewOrder_Page",
	}
	activity := parseWorkflowActivity(raw)
	if activity == nil {
		t.Fatal("expected non-nil activity")
	}
	userTask, ok := activity.(*workflows.UserTask)
	if !ok {
		t.Fatalf("expected *workflows.UserTask, got %T", activity)
	}
	if userTask.Name != "ReviewOrder" {
		t.Errorf("Name = %q, want %q", userTask.Name, "ReviewOrder")
	}
	if userTask.Page != "Sales.ReviewOrder_Page" {
		t.Errorf("Page = %q, want %q", userTask.Page, "Sales.ReviewOrder_Page")
	}
}

func TestParseWorkflowActivity_CallMicroflow(t *testing.T) {
	raw := map[string]any{
		"$Type":      "Workflows$CallMicroflowTask",
		"Name":       "ValidateOrder",
		"Microflow":  "Sales.ValidateOrder",
	}
	activity := parseWorkflowActivity(raw)
	if activity == nil {
		t.Fatal("expected non-nil activity")
	}
	callMf, ok := activity.(*workflows.CallMicroflowTask)
	if !ok {
		t.Fatalf("expected *workflows.CallMicroflowTask, got %T", activity)
	}
	if callMf.Name != "ValidateOrder" {
		t.Errorf("Name = %q, want %q", callMf.Name, "ValidateOrder")
	}
	if callMf.Microflow != "Sales.ValidateOrder" {
		t.Errorf("Microflow = %q, want %q", callMf.Microflow, "Sales.ValidateOrder")
	}
}

func TestParseWorkflowActivity_UnknownType(t *testing.T) {
	raw := map[string]any{
		"$Type": "Workflows$FutureActivity",
		"Name":  "Something",
	}
	activity := parseWorkflowActivity(raw)
	if activity == nil {
		t.Fatal("expected non-nil generic activity")
	}
	generic, ok := activity.(*workflows.GenericWorkflowActivity)
	if !ok {
		t.Fatalf("expected *workflows.GenericWorkflowActivity, got %T", activity)
	}
	if generic.TypeString != "Workflows$FutureActivity" {
		t.Errorf("TypeString = %q, want %q", generic.TypeString, "Workflows$FutureActivity")
	}
}

func TestParseUserTaskOutcome_ValueField(t *testing.T) {
	raw := map[string]any{
		"Value":   "Approve",
		"Name":    "ApproveOutcome",
		"Caption": "Approve it",
	}
	outcome := parseUserTaskOutcome(raw)
	if outcome == nil {
		t.Fatal("expected non-nil outcome")
	}
	if outcome.Value != "Approve" {
		t.Errorf("Value = %q, want %q", outcome.Value, "Approve")
	}
	if outcome.Name != "ApproveOutcome" {
		t.Errorf("Name = %q, want %q", outcome.Name, "ApproveOutcome")
	}
	if outcome.Caption != "Approve it" {
		t.Errorf("Caption = %q, want %q", outcome.Caption, "Approve it")
	}
}

func TestParseParameterMappings_ArrayMarker(t *testing.T) {
	input := bson.A{
		int32(2),
		map[string]any{
			"Parameter":  "A.B.C",
			"Expression": "$ctx",
		},
	}
	mappings := parseParameterMappings(input)
	if len(mappings) != 1 {
		t.Fatalf("len(mappings) = %d, want 1", len(mappings))
	}
	if mappings[0].Parameter != "A.B.C" {
		t.Errorf("Parameter = %q, want %q", mappings[0].Parameter, "A.B.C")
	}
	if mappings[0].Expression != "$ctx" {
		t.Errorf("Expression = %q, want %q", mappings[0].Expression, "$ctx")
	}
}

func TestParseParameterMappings_MultipleEntries(t *testing.T) {
	input := bson.A{
		int32(2),
		map[string]any{"Parameter": "P1", "Expression": "$a"},
		map[string]any{"Parameter": "P2", "Expression": "$b"},
	}
	mappings := parseParameterMappings(input)
	if len(mappings) != 2 {
		t.Fatalf("len(mappings) = %d, want 2", len(mappings))
	}
}

func TestParseBoundaryEvents_EmptyArray(t *testing.T) {
	input := bson.A{int32(2)}
	events := parseBoundaryEvents(input)
	if events != nil {
		t.Errorf("expected nil for empty array with only marker, got len=%d", len(events))
	}
}

func TestParseBoundaryEvents_TimerEvent(t *testing.T) {
	input := bson.A{
		int32(2),
		map[string]any{
			"$Type":              "Workflows$InterruptingTimerBoundaryEvent",
			"Caption":            "Timeout",
			"FirstExecutionTime": "addDays([%CurrentDateTime%], 3)",
		},
	}
	events := parseBoundaryEvents(input)
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	if events[0].EventType != "InterruptingTimer" {
		t.Errorf("EventType = %q, want %q", events[0].EventType, "InterruptingTimer")
	}
	if events[0].Caption != "Timeout" {
		t.Errorf("Caption = %q, want %q", events[0].Caption, "Timeout")
	}
	if events[0].TimerDelay != "addDays([%CurrentDateTime%], 3)" {
		t.Errorf("TimerDelay = %q, want %q", events[0].TimerDelay, "addDays([%CurrentDateTime%], 3)")
	}
}

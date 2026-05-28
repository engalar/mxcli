// SPDX-License-Identifier: Apache-2.0

package workflows

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func mustMarshal(t *testing.T, d bson.D) []byte {
	t.Helper()
	raw, err := bson.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return raw
}

func TestRawFieldString(t *testing.T) {
	raw := mustMarshal(t, bson.D{{Key: "Hello", Value: "world"}})
	if got := RawFieldString(raw, "Hello"); got != "world" {
		t.Errorf("RawFieldString: got %q, want %q", got, "world")
	}
	if got := RawFieldString(raw, "Missing"); got != "" {
		t.Errorf("RawFieldString(missing): got %q, want empty", got)
	}
	if got := RawFieldString(nil, "Hello"); got != "" {
		t.Errorf("RawFieldString(nil): got %q, want empty", got)
	}
}

func TestAnnotationDescription(t *testing.T) {
	if got := AnnotationDescription(nil); got != "" {
		t.Errorf("AnnotationDescription(nil): got %q, want empty", got)
	}
	a := &Annotation{}
	a.SetRaw(mustMarshal(t, bson.D{{Key: "Description", Value: "hello"}}))
	if got := AnnotationDescription(a); got != "hello" {
		t.Errorf("AnnotationDescription: got %q, want %q", got, "hello")
	}
}

func TestAnnotationText(t *testing.T) {
	if got := AnnotationText(nil); got != "" {
		t.Errorf("AnnotationText(nil): got %q, want empty", got)
	}
	a := &Annotation{}
	a.SetRaw(mustMarshal(t, bson.D{{Key: "Text", Value: "legacy"}}))
	if got := AnnotationText(a); got != "legacy" {
		t.Errorf("AnnotationText: got %q, want %q", got, "legacy")
	}
}

func TestAnnotationTextWrappers(t *testing.T) {
	raw := mustMarshal(t, bson.D{
		{Key: "Translation", Value: "你好"},
		{Key: "Value", Value: "literal"},
	})
	if got := AnnotationTextTranslation(raw); got != "你好" {
		t.Errorf("AnnotationTextTranslation: got %q", got)
	}
	if got := AnnotationTextValue(raw); got != "literal" {
		t.Errorf("AnnotationTextValue: got %q", got)
	}
}

func TestWorkflowParameterEntity(t *testing.T) {
	if got := WorkflowParameterEntity(nil); got != "" {
		t.Errorf("WorkflowParameterEntity(nil): got %q, want empty", got)
	}
	p := &Parameter{}
	p.SetRaw(mustMarshal(t, bson.D{{Key: "Entity", Value: "MyModule.MyEntity"}}))
	if got := WorkflowParameterEntity(p); got != "MyModule.MyEntity" {
		t.Errorf("WorkflowParameterEntity: got %q", got)
	}
}

func TestExclusiveSplitOutcomeValue(t *testing.T) {
	if got := ExclusiveSplitOutcomeValue(nil); got != "" {
		t.Errorf("ExclusiveSplitOutcomeValue(nil): got %q, want empty", got)
	}
	o := &ExclusiveSplitOutcome{}
	o.SetRaw(mustMarshal(t, bson.D{{Key: "Value", Value: "caption"}}))
	if got := ExclusiveSplitOutcomeValue(o); got != "caption" {
		t.Errorf("ExclusiveSplitOutcomeValue: got %q", got)
	}
}

func TestSystemTaskFields(t *testing.T) {
	raw := mustMarshal(t, bson.D{
		{Key: "Annotation", Value: "note"},
		{Key: "Name", Value: "Approve"},
		{Key: "Caption", Value: "Approve Request"},
		{Key: "Microflow", Value: "M.Approve"},
	})
	if got := SystemTaskAnnotation(raw); got != "note" {
		t.Errorf("SystemTaskAnnotation: got %q", got)
	}
	if got := SystemTaskName(raw); got != "Approve" {
		t.Errorf("SystemTaskName: got %q", got)
	}
	if got := SystemTaskCaption(raw); got != "Approve Request" {
		t.Errorf("SystemTaskCaption: got %q", got)
	}
	if got := SystemTaskMicroflow(raw); got != "M.Approve" {
		t.Errorf("SystemTaskMicroflow: got %q", got)
	}
}

func TestTimerActivityFields(t *testing.T) {
	raw := mustMarshal(t, bson.D{
		{Key: "Delay", Value: "PT1H"},
		{Key: "FirstExecutionTime", Value: "2026-01-01"},
	})
	if got := TimerActivityDelay(raw); got != "PT1H" {
		t.Errorf("TimerActivityDelay: got %q", got)
	}
	if got := TimerActivityFirstExecutionTime(raw); got != "2026-01-01" {
		t.Errorf("TimerActivityFirstExecutionTime: got %q", got)
	}
}

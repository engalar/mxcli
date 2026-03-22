//go:build debug

package bson

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestRenderScalarFields(t *testing.T) {
	doc := bson.D{
		{Key: "$Type", Value: "Workflows$Workflow"},
		{Key: "Name", Value: "TestWf"},
		{Key: "Excluded", Value: false},
		{Key: "AdminPage", Value: nil},
	}
	got := Render(doc, 0)
	want := `Workflows$Workflow
  AdminPage: null
  Excluded: false
  Name: "TestWf"`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderUUIDNormalized(t *testing.T) {
	doc := bson.D{
		{Key: "$Type", Value: "Workflows$Flow"},
		{Key: "$ID", Value: primitive.Binary{Subtype: 3, Data: []byte("anything")}},
		{Key: "PersistentId", Value: primitive.Binary{Subtype: 3, Data: []byte("anything")}},
	}
	got := Render(doc, 0)
	want := `Workflows$Flow
  PersistentId: <uuid>`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderArrayWithMarker(t *testing.T) {
	doc := bson.D{
		{Key: "$Type", Value: "Workflows$Flow"},
		{Key: "Activities", Value: bson.A{int32(3), bson.D{
			{Key: "$Type", Value: "Workflows$EndWorkflowActivity"},
			{Key: "Name", Value: "end1"},
		}}},
	}
	got := Render(doc, 0)
	want := `Workflows$Flow
  Activities [marker=3]:
    - Workflows$EndWorkflowActivity
        Name: "end1"`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderEmptyArray(t *testing.T) {
	doc := bson.D{
		{Key: "$Type", Value: "Workflows$StartWorkflowActivity"},
		{Key: "BoundaryEvents", Value: bson.A{int32(2)}},
	}
	got := Render(doc, 0)
	want := `Workflows$StartWorkflowActivity
  BoundaryEvents [marker=2]: []`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

package bson

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestRenderScalarFields(t *testing.T) {
	doc := bson.D{
		{Key: "$Type", Value: "Workflows$Workflow"},
		{Key: "Name", Value: "TestWf"},
		{Key: "Excluded", Value: false},
		{Key: "AdminPage", Value: nil},
		{Key: "$ID", Value: bson.Binary{Subtype: 0, Data: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}}},
	}
	got := Render(doc, 0)
	want := `$ID: 00000000-0000-0000-0000-000000000000
$Type: "Workflows$Workflow"
AdminPage: null
Excluded: false
Name: "TestWf"`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderUUIDOutput(t *testing.T) {
	data := []byte{0x78, 0x56, 0x34, 0x12, 0xab, 0xcd, 0xef, 0x01, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}
	doc := bson.D{
		{Key: "$Type", Value: "Workflows$Flow"},
		{Key: "$ID", Value: bson.Binary{Subtype: 3, Data: data}},
		{Key: "PersistentId", Value: bson.Binary{Subtype: 3, Data: data}},
	}
	got := Render(doc, 0)
	want := `$ID: 12345678-cdab-01ef-1234-56789abcdef0
$Type: "Workflows$Flow"
PersistentId: 12345678-cdab-01ef-1234-56789abcdef0`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderArrayWithMarker(t *testing.T) {
	doc := bson.D{
		{Key: "$Type", Value: "Workflows$Flow"},
		{Key: "$ID", Value: bson.Binary{Subtype: 0, Data: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}}},
		{Key: "Activities", Value: bson.A{int32(3), bson.D{
			{Key: "$Type", Value: "Workflows$EndWorkflowActivity"},
			{Key: "Name", Value: "end1"},
			{Key: "$ID", Value: bson.Binary{Subtype: 0, Data: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}}},
		}}},
	}
	got := Render(doc, 0)
	want := `$ID: 00000000-0000-0000-0000-000000000000
$Type: "Workflows$Flow"
Activities [marker=3]:
  $ID: 00000000-0000-0000-0000-000000000000
  $Type: "Workflows$EndWorkflowActivity"
  Name: "end1"`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderEmptyArray(t *testing.T) {
	doc := bson.D{
		{Key: "$Type", Value: "Workflows$StartWorkflowActivity"},
		{Key: "BoundaryEvents", Value: bson.A{int32(2)}},
		{Key: "$ID", Value: bson.Binary{Subtype: 0, Data: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}}},
	}
	got := Render(doc, 0)
	want := `$ID: 00000000-0000-0000-0000-000000000000
$Type: "Workflows$StartWorkflowActivity"
BoundaryEvents [marker=2]: []`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

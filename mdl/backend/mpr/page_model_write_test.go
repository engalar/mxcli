// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/types"
)

// TestWidgetToBSON_Container verifies a Container WidgetNode encodes to a
// Forms$DivContainer BSON doc with Name, Appearance, and Widgets fields.
func TestWidgetToBSON_Container(t *testing.T) {
	node := &types.WidgetNode{
		Kind: types.WidgetContainer,
		Name: "mainBox",
	}
	doc := widgetToBSON(node)
	if doc == nil {
		t.Fatal("widgetToBSON returned nil")
	}
	if got := dGetString(doc, "$Type"); got != "Forms$DivContainer" {
		t.Errorf("$Type = %q, want Forms$DivContainer", got)
	}
	if got := dGetString(doc, "Name"); got != "mainBox" {
		t.Errorf("Name = %q, want mainBox", got)
	}
	// Container must include a Widgets field (even if empty array).
	if dGet(doc, "Widgets") == nil {
		t.Error("missing Widgets field for container")
	}
}

// TestWidgetToBSON_RoundTrip verifies widgetToBSON → widgetNodeFromBSON
// returns an equivalent WidgetNode for a button with caption.
func TestWidgetToBSON_RoundTrip(t *testing.T) {
	orig := &types.WidgetNode{
		Kind:    types.WidgetButton,
		Name:    "btn",
		Caption: "Click Me",
	}
	encoded := widgetToBSON(orig)
	if encoded == nil {
		t.Fatal("widgetToBSON returned nil")
	}
	decoded := widgetNodeFromBSON(encoded)
	if decoded == nil {
		t.Fatal("widgetNodeFromBSON returned nil")
	}
	if decoded.Kind != types.WidgetButton {
		t.Errorf("Kind round-trip lost: got %q, want %q", decoded.Kind, types.WidgetButton)
	}
	if decoded.Name != "btn" {
		t.Errorf("Name round-trip lost: got %q, want %q", decoded.Name, "btn")
	}
	if decoded.Caption != "Click Me" {
		t.Errorf("Caption round-trip lost: got %q, want %q", decoded.Caption, "Click Me")
	}
}

// TestBsonVersionedArray verifies the int32(1) prefix is prepended to a
// slice of bson.D documents.
func TestBsonVersionedArray(t *testing.T) {
	docs := []bson.D{
		{{Key: "Name", Value: "a"}},
		{{Key: "Name", Value: "b"}},
	}
	arr := bsonVersionedArray(docs)
	if len(arr) != 3 {
		t.Fatalf("expected len 3 (marker + 2 docs), got %d", len(arr))
	}
	marker, ok := arr[0].(int32)
	if !ok || marker != 1 {
		t.Errorf("expected int32(1) marker at index 0, got %v (%T)", arr[0], arr[0])
	}
}

// TestKindToBSONType_AllKinds covers every WidgetKind→storage type mapping.
// The Mendix storage namespace is "Forms$" (NOT "Pages$" — that's the SDK
// display namespace). Writing a Pages$ type into the unit BSON triggers
// TypeCacheUnknownTypeException when Studio Pro / mx check loads the page.
func TestKindToBSONType_AllKinds(t *testing.T) {
	cases := map[types.WidgetKind]string{
		types.WidgetContainer:    "Forms$DivContainer",
		types.WidgetButton:       "Forms$ActionButton",
		types.WidgetLayoutGrid:   "Forms$LayoutGrid",
		types.WidgetLayoutRow:    "Forms$LayoutGridRow",
		types.WidgetLayoutCol:    "Forms$LayoutGridColumn",
		types.WidgetTabContainer: "Forms$TabControl",
		types.WidgetTabPage:      "Forms$TabPage",
		types.WidgetDataView:     "Forms$DataView",
		types.WidgetTextBox:      "Forms$TextBox",
		types.WidgetSnippet:      "Forms$SnippetCallWidget",
		types.WidgetDataGrid:     "CustomWidgets$CustomWidget",
		types.WidgetImage:        "CustomWidgets$CustomWidget",
		types.WidgetUnknown:      "CustomWidgets$CustomWidget",
	}
	for k, want := range cases {
		if got := kindToBSONType(k); got != want {
			t.Errorf("kindToBSONType(%q) = %q, want %q", k, got, want)
		}
	}
}

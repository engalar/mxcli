// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/types"
)

// TestLayoutDocToModel_WebLayout verifies that layoutDocToModel correctly
// extracts a ScrollContainer with a Placeholder from a Forms$WebLayoutContent
// document — the structure used by Responsive/Phone/Tablet layouts.
func TestLayoutDocToModel_WebLayout(t *testing.T) {
	t.Parallel()
	placeholderDoc := bson.D{
		{Key: "$Type", Value: "Forms$Placeholder"},
		{Key: "Name", Value: "Main"},
	}
	regionDoc := bson.D{
		{Key: "$Type", Value: "Forms$ScrollContainerRegion"},
		{Key: "Widgets", Value: bson.A{int32(2), placeholderDoc}},
	}
	scrollDoc := bson.D{
		{Key: "$Type", Value: "Forms$ScrollContainer"},
		{Key: "Name", Value: "scrollContainer1"},
		{Key: "CenterRegion", Value: regionDoc},
		{Key: "Widgets", Value: bson.A{int32(2)}}, // empty in layout context
	}
	contentDoc := bson.D{
		{Key: "$Type", Value: "Forms$WebLayoutContent"},
		{Key: "LayoutType", Value: "Responsive"},
		{Key: "Widgets", Value: bson.A{int32(2), scrollDoc}},
	}
	doc := bson.D{
		{Key: "$Type", Value: "Forms$Layout"},
		{Key: "Content", Value: contentDoc},
	}

	pm := layoutDocToModel(doc)
	if pm == nil {
		t.Fatal("layoutDocToModel returned nil")
	}
	if len(pm.Widgets) == 0 {
		t.Fatal("expected at least 1 top-level widget from Content.Widgets, got 0")
	}
	scrollNode := pm.Widgets[0]
	if scrollNode.Kind != types.WidgetScrollView {
		t.Errorf("top widget Kind = %q, want %q", scrollNode.Kind, types.WidgetScrollView)
	}
	// Placeholder should appear as a child via CenterRegion
	if len(scrollNode.Children) == 0 {
		t.Fatal("ScrollContainer should have at least 1 child (Placeholder from CenterRegion)")
	}
	phNode := scrollNode.Children[0]
	if phNode.Kind != types.WidgetPlaceholder {
		t.Errorf("child Kind = %q, want %q", phNode.Kind, types.WidgetPlaceholder)
	}
	if phNode.Name != "Main" {
		t.Errorf("placeholder Name = %q, want %q", phNode.Name, "Main")
	}
}

// TestLayoutDocToModel_NativeLayout verifies that layoutDocToModel handles the
// Forms$NativeLayoutContent BSON structure used by native mobile layouts.
func TestLayoutDocToModel_NativeLayout(t *testing.T) {
	t.Parallel()
	contentDoc := bson.D{
		{Key: "$Type", Value: "Forms$NativeLayoutContent"},
		{Key: "LayoutType", Value: "Default"},
		{Key: "Widgets", Value: bson.A{
			int32(2),
			bson.D{
				{Key: "$Type", Value: "Forms$Placeholder"},
				{Key: "Name", Value: "Content"},
			},
		}},
	}
	doc := bson.D{
		{Key: "$Type", Value: "Forms$Layout"},
		{Key: "Content", Value: contentDoc},
	}

	pm := layoutDocToModel(doc)
	if pm == nil {
		t.Fatal("layoutDocToModel returned nil")
	}
	if len(pm.Widgets) == 0 {
		t.Fatal("expected at least 1 widget from native layout Content.Widgets")
	}
	phNode := pm.Widgets[0]
	if phNode.Kind != types.WidgetPlaceholder {
		t.Errorf("widget Kind = %q, want %q", phNode.Kind, types.WidgetPlaceholder)
	}
	if phNode.Name != "Content" {
		t.Errorf("placeholder Name = %q, want %q", phNode.Name, "Content")
	}
}

// TestPageDocToModel_Container builds a minimal BSON page document that wraps
// a single Forms$DivContainer named "mainBox" and verifies pageDocToModel
// extracts the container as a WidgetNode with Kind=container.
func TestPageDocToModel_Container(t *testing.T) {
	t.Parallel()
	doc := bson.D{
		{Key: "FormCall", Value: bson.D{
			{Key: "Arguments", Value: bson.A{
				int32(2),
				bson.D{
					{Key: "Widgets", Value: bson.A{
						int32(2),
						bson.D{
							{Key: "$Type", Value: "Forms$DivContainer"},
							{Key: "Name", Value: "mainBox"},
						},
					}},
				},
			}},
		}},
	}

	pm := pageDocToModel(doc)
	if pm == nil {
		t.Fatal("pageDocToModel returned nil")
	}
	if len(pm.Widgets) != 1 {
		t.Fatalf("expected 1 top-level widget, got %d", len(pm.Widgets))
	}
	w := pm.Widgets[0]
	if w.Kind != types.WidgetContainer {
		t.Errorf("widget Kind = %q, want %q", w.Kind, types.WidgetContainer)
	}
	if w.Name != "mainBox" {
		t.Errorf("widget Name = %q, want %q", w.Name, "mainBox")
	}
}

// TestPageDocToModel_ButtonCaption verifies extractTextFromTemplate picks up
// the en_US translation of a button caption stored as a translatable template.
func TestPageDocToModel_ButtonCaption(t *testing.T) {
	t.Parallel()
	doc := bson.D{
		{Key: "FormCall", Value: bson.D{
			{Key: "Arguments", Value: bson.A{
				int32(2),
				bson.D{
					{Key: "Widgets", Value: bson.A{
						int32(2),
						bson.D{
							{Key: "$Type", Value: "Forms$ActionButton"},
							{Key: "Name", Value: "btn"},
							{Key: "Caption", Value: bson.D{
								{Key: "Translations", Value: bson.A{
									int32(2),
									bson.D{
										{Key: "LanguageCode", Value: "en_US"},
										{Key: "Text", Value: "Click Me"},
									},
								}},
							}},
						},
					}},
				},
			}},
		}},
	}

	pm := pageDocToModel(doc)
	if pm == nil || len(pm.Widgets) != 1 {
		t.Fatalf("expected 1 widget, got %v", pm)
	}
	w := pm.Widgets[0]
	if w.Kind != types.WidgetButton {
		t.Errorf("widget Kind = %q, want %q", w.Kind, types.WidgetButton)
	}
	if w.Caption != "Click Me" {
		t.Errorf("widget Caption = %q, want %q", w.Caption, "Click Me")
	}
}

func makeFooterBSON(footerName string) bson.D {
	return bson.D{
		{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
		{Key: "$Type", Value: "Forms$DivContainer"},
		{Key: "Name", Value: footerName},
		{Key: "Appearance", Value: bson.D{
			{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
			{Key: "$Type", Value: "Forms$Appearance"},
		}},
	}
}

func TestWidgetNodeFromBSON_DataView_ExtractsFooter(t *testing.T) {
	t.Parallel()
	footerBSON := makeFooterBSON("footer1")
	dvBSON := bson.D{
		{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
		{Key: "$Type", Value: "Forms$DataView"},
		{Key: "Name", Value: "dvMain"},
		{Key: "Appearance", Value: bson.D{
			{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
			{Key: "$Type", Value: "Forms$Appearance"},
		}},
		{Key: "ShowFooter", Value: true},
		{Key: "FooterWidgets", Value: bsonVersionedArray([]bson.D{footerBSON})},
	}

	node := widgetNodeFromBSON(dvBSON)
	if node == nil {
		t.Fatal("widgetNodeFromBSON returned nil for DataView")
	}
	if len(node.Footer) != 1 {
		t.Errorf("want 1 footer widget, got %d", len(node.Footer))
	}
	if node.Footer[0].Name != "footer1" {
		t.Errorf("footer widget name: want footer1, got %q", node.Footer[0].Name)
	}
}

func TestWidgetToBSON_DataView_WritesFooter(t *testing.T) {
	t.Parallel()
	footerNode := &types.WidgetNode{
		Kind: types.WidgetContainer,
		Name: "footer1",
	}
	dvNode := &types.WidgetNode{
		Kind:   types.WidgetDataView,
		Name:   "dvMain",
		Footer: []*types.WidgetNode{footerNode},
	}

	doc := widgetToBSON(dvNode)
	if doc == nil {
		t.Fatal("widgetToBSON returned nil")
	}

	showFooter, ok := dGet(doc, "ShowFooter").(bool)
	if !ok || !showFooter {
		t.Errorf("ShowFooter should be true in BSON, got %v", dGet(doc, "ShowFooter"))
	}

	footerArr := dGetArrayElements(dGet(doc, "FooterWidgets"))
	if len(footerArr) == 0 {
		t.Error("FooterWidgets array should not be empty")
	}
}

func TestWidgetToBSON_DataView_NoFooterWhenEmpty(t *testing.T) {
	t.Parallel()
	dvNode := &types.WidgetNode{
		Kind:   types.WidgetDataView,
		Name:   "dvMain",
		Footer: nil,
	}
	doc := widgetToBSON(dvNode)
	if dGet(doc, "ShowFooter") != nil {
		t.Error("ShowFooter should not be written when Footer is empty")
	}
}

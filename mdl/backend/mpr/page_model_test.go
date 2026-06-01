// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/types"
)

// TestPageDocToModel_Container builds a minimal BSON page document that wraps
// a single Forms$DivContainer named "mainBox" and verifies pageDocToModel
// extracts the container as a WidgetNode with Kind=container.
func TestPageDocToModel_Container(t *testing.T) {
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

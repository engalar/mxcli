// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// mkTextsDoc builds a Texts$Text with a leading int32 version marker, matching
// the real Mendix BSON layout for widget captions.
func mkTextsDoc(translations ...bson.D) bson.D {
	items := bson.A{int32(3)}
	for _, tr := range translations {
		items = append(items, tr)
	}
	return bson.D{
		{Key: "$Type", Value: "Texts$Text"},
		{Key: "Items", Value: items},
	}
}

func TestSetVersionedTranslation_UpdateExisting(t *testing.T) {
	doc := mkTextsDoc(bson.D{
		{Key: "$Type", Value: "Texts$Translation"},
		{Key: "LanguageCode", Value: "en_US"},
		{Key: "Text", Value: "Submit"},
	})
	setVersionedTranslation(doc, "en_US", "Submit Updated")

	items := toBsonA(dGet(doc, "Items"))
	if len(items) != 2 {
		t.Fatalf("expected marker + 1 item, got %d", len(items))
	}
	if _, ok := items[0].(int32); !ok {
		t.Errorf("expected int32 marker preserved at index 0, got %T", items[0])
	}
	tr := items[1].(bson.D)
	if dGetString(tr, "Text") != "Submit Updated" {
		t.Errorf("expected 'Submit Updated', got %q", dGetString(tr, "Text"))
	}
}

func TestSetVersionedTranslation_AddNew(t *testing.T) {
	doc := mkTextsDoc(bson.D{
		{Key: "$Type", Value: "Texts$Translation"},
		{Key: "LanguageCode", Value: "en_US"},
		{Key: "Text", Value: "Submit"},
	})
	setVersionedTranslation(doc, "zh_CN", "提交")

	items := toBsonA(dGet(doc, "Items"))
	if len(items) != 3 {
		t.Fatalf("expected marker + 2 items, got %d", len(items))
	}
	if _, ok := items[0].(int32); !ok {
		t.Errorf("expected int32 marker at index 0, got %T", items[0])
	}
	found := false
	for _, it := range items[1:] {
		tr := it.(bson.D)
		if dGetString(tr, "LanguageCode") == "zh_CN" && dGetString(tr, "Text") == "提交" {
			found = true
		}
	}
	if !found {
		t.Error("zh_CN translation not appended")
	}
}

func TestLocateWidgetTextsDoc_Caption(t *testing.T) {
	widget := bson.D{
		{Key: "Caption", Value: mkTextsDoc()},
	}
	if locateWidgetTextsDoc(widget, "caption") == nil {
		t.Error("expected to locate Caption Texts$Text")
	}
	if locateWidgetTextsDoc(widget, "placeholder") != nil {
		t.Error("expected nil for absent placeholder")
	}
}

func TestLocateWidgetTextsDoc_LabelCaption(t *testing.T) {
	widget := bson.D{
		{Key: "Label", Value: bson.D{
			{Key: "Caption", Value: mkTextsDoc()},
		}},
	}
	if locateWidgetTextsDoc(widget, "label") == nil {
		t.Error("expected to locate Label.Caption Texts$Text")
	}
}

// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func mkShowMessageActivity(text string) bson.D {
	return bson.D{
		{Key: "$Type", Value: "Microflows$ActionActivity"},
		{Key: "Action", Value: bson.D{
			{Key: "$Type", Value: "Microflows$ShowMessageAction"},
			{Key: "Template", Value: bson.D{
				{Key: "$Type", Value: "Microflows$TextTemplate"},
				{Key: "Text", Value: bson.D{
					{Key: "$Type", Value: "Texts$Text"},
					{Key: "Items", Value: bson.A{
						int32(3),
						bson.D{
							{Key: "$Type", Value: "Texts$Translation"},
							{Key: "LanguageCode", Value: "en_US"},
							{Key: "Text", Value: text},
						},
					}},
				}},
			}},
		}},
	}
}

func mkMicroflowDoc(activities ...bson.D) bson.D {
	objs := bson.A{int32(3)}
	for _, a := range activities {
		objs = append(objs, a)
	}
	return bson.D{
		{Key: "$Type", Value: "Microflows$Microflow"},
		{Key: "ObjectCollection", Value: bson.D{
			{Key: "$Type", Value: "Microflows$MicroflowObjectCollection"},
			{Key: "Objects", Value: objs},
		}},
	}
}

func TestSetMicroflowActionTranslation_FirstShowMessage(t *testing.T) {
	t.Parallel()
	doc := mkMicroflowDoc(
		mkShowMessageActivity("First"),
		mkShowMessageActivity("Second"),
	)
	if !setMicroflowActionTranslationBSON(doc, "Microflows$ShowMessageAction", 0, "message", "zh_CN", "第一个") {
		t.Fatal("expected translation applied to index 0")
	}
	// Verify index 0 got the zh_CN translation and index 1 was untouched.
	objs := dGetArrayElements(dGet(dGetDoc(doc, "ObjectCollection"), "Objects"))
	act0 := dGetDoc(objs[0].(bson.D), "Action")
	tmpl0 := dGetDoc(act0, "Template")
	text0 := dGetDoc(tmpl0, "Text")
	items0 := dGetArrayElements(dGet(text0, "Items"))
	foundZh := false
	for _, it := range items0 {
		tr := it.(bson.D)
		if dGetString(tr, "LanguageCode") == "zh_CN" && dGetString(tr, "Text") == "第一个" {
			foundZh = true
		}
	}
	if !foundZh {
		t.Error("zh_CN translation not added to first ShowMessage")
	}

	act1 := dGetDoc(objs[1].(bson.D), "Action")
	items1 := dGetArrayElements(dGet(dGetDoc(dGetDoc(act1, "Template"), "Text"), "Items"))
	for _, it := range items1 {
		if dGetString(it.(bson.D), "LanguageCode") == "zh_CN" {
			t.Error("second ShowMessage should not have been translated")
		}
	}
}

func TestSetMicroflowActionTranslation_SecondShowMessage(t *testing.T) {
	t.Parallel()
	doc := mkMicroflowDoc(
		mkShowMessageActivity("First"),
		mkShowMessageActivity("Second"),
	)
	if !setMicroflowActionTranslationBSON(doc, "Microflows$ShowMessageAction", 1, "message", "zh_CN", "第二个") {
		t.Fatal("expected translation applied to index 1")
	}
	objs := dGetArrayElements(dGet(dGetDoc(doc, "ObjectCollection"), "Objects"))
	act1 := dGetDoc(objs[1].(bson.D), "Action")
	items1 := dGetArrayElements(dGet(dGetDoc(dGetDoc(act1, "Template"), "Text"), "Items"))
	found := false
	for _, it := range items1 {
		tr := it.(bson.D)
		if dGetString(tr, "LanguageCode") == "zh_CN" && dGetString(tr, "Text") == "第二个" {
			found = true
		}
	}
	if !found {
		t.Error("zh_CN translation not added to second ShowMessage")
	}
}

func TestSetMicroflowActionTranslation_IndexOutOfRange(t *testing.T) {
	t.Parallel()
	doc := mkMicroflowDoc(mkShowMessageActivity("Only"))
	if setMicroflowActionTranslationBSON(doc, "Microflows$ShowMessageAction", 5, "message", "zh_CN", "x") {
		t.Error("expected false for out-of-range index")
	}
}

func TestSetMicroflowActionTranslation_UnknownType(t *testing.T) {
	t.Parallel()
	doc := mkMicroflowDoc(mkShowMessageActivity("Only"))
	if setMicroflowActionTranslationBSON(doc, "Microflows$LogMessageAction", 0, "message", "zh_CN", "x") {
		t.Error("expected false when no action of that type exists")
	}
}

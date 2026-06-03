// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestSetTranslationForLang_UpdateExisting(t *testing.T) {
	textDoc := bson.D{
		{Key: "$Type", Value: "Texts$Text"},
		{Key: "Items", Value: bson.A{
			bson.D{
				{Key: "$Type", Value: "Texts$Translation"},
				{Key: "LanguageCode", Value: "en_US"},
				{Key: "Text", Value: "Submit"},
			},
		}},
	}
	setTranslationForLang(textDoc, "en_US", "Submit Updated")
	items := extractBsonArray(dGet(textDoc, "Items"))
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	doc := extractBsonMap(items[0])
	if extractString(doc["Text"]) != "Submit Updated" {
		t.Errorf("expected 'Submit Updated', got %q", extractString(doc["Text"]))
	}
}

func TestSetTranslationForLang_AddNew(t *testing.T) {
	textDoc := bson.D{
		{Key: "$Type", Value: "Texts$Text"},
		{Key: "Items", Value: bson.A{
			bson.D{
				{Key: "$Type", Value: "Texts$Translation"},
				{Key: "LanguageCode", Value: "en_US"},
				{Key: "Text", Value: "Submit"},
			},
		}},
	}
	setTranslationForLang(textDoc, "zh_CN", "提交")
	items := extractBsonArray(dGet(textDoc, "Items"))
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	found := false
	for _, item := range items {
		doc := extractBsonMap(item)
		if extractString(doc["LanguageCode"]) == "zh_CN" && extractString(doc["Text"]) == "提交" {
			found = true
		}
	}
	if !found {
		t.Error("zh_CN translation not found")
	}
}

func TestSetTranslationForLang_EmptyItems(t *testing.T) {
	textDoc := bson.D{
		{Key: "$Type", Value: "Texts$Text"},
		{Key: "Items", Value: bson.A{}},
	}
	setTranslationForLang(textDoc, "zh_CN", "提交")
	items := extractBsonArray(dGet(textDoc, "Items"))
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

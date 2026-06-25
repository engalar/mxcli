// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestSetTranslationForLang_UpdateExisting(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
			// New translations must carry a $ID; Studio Pro's storage loader
			// dereferences it and crashes (NullReferenceException) otherwise.
			if doc["$ID"] == nil {
				t.Error("new Texts$Translation is missing $ID")
			}
		}
	}
	if !found {
		t.Error("zh_CN translation not found")
	}
}

// TestSetTranslationForLang_PreservesVersionMarker guards against dropping the
// Mendix versioned-array prefix (int32 at index 0) when appending a new
// translation. Real Texts$Text.Items arrays carry this marker; losing it
// corrupts the BSON and triggers StorageLoadException in Studio Pro.
func TestSetTranslationForLang_PreservesVersionMarker(t *testing.T) {
	t.Parallel()
	textDoc := bson.D{
		{Key: "$Type", Value: "Texts$Text"},
		{Key: "Items", Value: bson.A{
			int32(3), // versioned-array prefix, as written by Studio Pro
			bson.D{
				{Key: "$Type", Value: "Texts$Translation"},
				{Key: "LanguageCode", Value: "en_US"},
				{Key: "Text", Value: "Submit"},
			},
		}},
	}
	setTranslationForLang(textDoc, "zh_CN", "提交")

	rawItems, ok := dGet(textDoc, "Items").(bson.A)
	if !ok {
		t.Fatalf("Items is not bson.A, got %T", dGet(textDoc, "Items"))
	}
	if len(rawItems) == 0 {
		t.Fatal("Items is empty")
	}
	marker, ok := rawItems[0].(int32)
	if !ok || marker != 3 {
		t.Fatalf("version marker int32(3) lost: first element is %#v", rawItems[0])
	}
	// Marker + 2 translations.
	if len(rawItems) != 3 {
		t.Fatalf("expected marker + 2 translations = 3 elements, got %d", len(rawItems))
	}
}

func TestSetEnumValueTranslation(t *testing.T) {
	t.Parallel()
	enumDoc := bson.D{
		{Key: "$Type", Value: "Enumerations$Enumeration"},
		{Key: "Name", Value: "Status"},
		{Key: "Values", Value: bson.A{
			int32(3),
			bson.D{
				{Key: "$Type", Value: "Enumerations$EnumerationValue"},
				{Key: "Name", Value: "ACTIVE"},
				{Key: "Caption", Value: bson.D{
					{Key: "$Type", Value: "Texts$Text"},
					{Key: "Items", Value: bson.A{
						int32(3),
						bson.D{
							{Key: "$Type", Value: "Texts$Translation"},
							{Key: "LanguageCode", Value: "en_US"},
							{Key: "Text", Value: "Active"},
						},
					}},
				}},
			},
		}},
	}

	if ok := setEnumValueTranslation(enumDoc, "ACTIVE", "zh_CN", "活跃"); !ok {
		t.Fatal("expected setEnumValueTranslation to succeed for ACTIVE")
	}

	// Verify the zh_CN translation was added while en_US is preserved.
	val := extractBsonArray(dGet(enumDoc, "Values"))[0].(bson.D)
	caption := dGetDoc(val, "Caption")
	items := extractBsonArray(dGet(caption, "Items"))
	if len(items) != 2 {
		t.Fatalf("expected 2 translations, got %d", len(items))
	}
	got := map[string]string{}
	for _, it := range items {
		d := it.(bson.D)
		got[dGetString(d, "LanguageCode")] = dGetString(d, "Text")
	}
	if got["en_US"] != "Active" {
		t.Errorf("en_US translation lost: %q", got["en_US"])
	}
	if got["zh_CN"] != "活跃" {
		t.Errorf("zh_CN translation wrong: %q", got["zh_CN"])
	}

	// Version marker preserved.
	rawItems := dGet(caption, "Items").(bson.A)
	if m, _ := rawItems[0].(int32); m != 3 {
		t.Errorf("version marker lost on Caption.Items: %#v", rawItems[0])
	}
}

func TestSetEnumValueTranslation_NoSuchValue(t *testing.T) {
	t.Parallel()
	enumDoc := bson.D{
		{Key: "$Type", Value: "Enumerations$Enumeration"},
		{Key: "Values", Value: bson.A{int32(3)}},
	}
	if ok := setEnumValueTranslation(enumDoc, "MISSING", "zh_CN", "x"); ok {
		t.Error("expected false for non-existent value name")
	}
}

func TestSetTranslationForLang_EmptyItems(t *testing.T) {
	t.Parallel()
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

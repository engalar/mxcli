// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/bsonutil"
)

// setTranslationForLang updates or inserts a Texts$Translation entry inside a
// Texts$Text BSON document. If a translation for langCode already exists, its
// Text field is updated in place. Otherwise a new Texts$Translation is appended.
//
// New Texts$Translation entries are given a fresh $ID: although translations are
// nested value objects, Studio Pro's storage loader dereferences $ID during
// unit construction and throws a NullReferenceException when it is absent.
func setTranslationForLang(textsBsonD bson.D, langCode, text string) {
	raw := dGet(textsBsonD, "Items")
	marker, items := splitVersionMarker(raw)
	for _, item := range items {
		doc, ok := item.(bson.D)
		if !ok {
			continue
		}
		if extractString(dGet(doc, "LanguageCode")) == langCode {
			dSet(doc, "Text", text)
			return
		}
	}
	newTr := bson.D{
		{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
		{Key: "$Type", Value: "Texts$Translation"},
		{Key: "LanguageCode", Value: langCode},
		{Key: "Text", Value: text},
	}
	items = append(items, newTr)

	out := make(bson.A, 0, len(items)+1)
	if marker != 0 {
		out = append(out, marker)
	}
	out = append(out, items...)
	dSet(textsBsonD, "Items", out)
}

// splitVersionMarker separates the Mendix versioned-array prefix (int32 marker
// 2 or 3) from the actual items. Returns marker=0 when the array carries no
// recognized prefix, so callers can decide whether to add one.
func splitVersionMarker(v any) (int32, []any) {
	var slice []any
	switch val := v.(type) {
	case bson.A:
		slice = []any(val)
	case []any:
		slice = val
	default:
		return 0, nil
	}
	if len(slice) > 0 {
		if marker, ok := slice[0].(int32); ok && (marker == 2 || marker == 3) {
			return marker, slice[1:]
		}
	}
	return 0, slice
}

// setTranslationInField finds a Texts$Text sub-document at fieldKey within
// parentDoc and sets the translation for langCode. Returns false if the field
// is absent or is not a Texts$Text document.
func setTranslationInField(parentDoc bson.D, fieldKey, langCode, text string) bool {
	textDoc := dGetDoc(parentDoc, fieldKey)
	if textDoc == nil {
		return false
	}
	if extractString(dGet(textDoc, "$Type")) != "Texts$Text" {
		return false
	}
	setTranslationForLang(textDoc, langCode, text)
	return true
}

// setEnumValueTranslation sets the langCode translation on the Caption of the
// enumeration value named valueName. enumDoc is the root Enumerations$Enumeration
// unit document. Returns false if no value with that name exists.
func setEnumValueTranslation(enumDoc bson.D, valueName, langCode, text string) bool {
	for _, item := range extractBsonArray(dGet(enumDoc, "Values")) {
		valDoc, ok := item.(bson.D)
		if !ok {
			continue
		}
		if dGetString(valDoc, "Name") != valueName {
			continue
		}
		return setTranslationInField(valDoc, "Caption", langCode, text)
	}
	return false
}

// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

// setTranslationForLang updates or inserts a Texts$Translation entry inside a
// Texts$Text BSON document. If a translation for langCode already exists, its
// Text field is updated in place. Otherwise a new Texts$Translation is appended.
//
// Texts$Translation is a value object, not a Mendix unit, so it carries no $ID.
func setTranslationForLang(textsBsonD bson.D, langCode, text string) {
	items := extractBsonArray(dGet(textsBsonD, "Items"))
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
		{Key: "$Type", Value: "Texts$Translation"},
		{Key: "LanguageCode", Value: langCode},
		{Key: "Text", Value: text},
	}
	newItems := append(items, newTr)
	dSet(textsBsonD, "Items", bson.A(newItems))
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

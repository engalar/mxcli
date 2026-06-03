// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/bsonutil"
)

// SetWidgetTranslation sets the translation for langCode on a translatable text
// property of the named widget. The property is one of caption, placeholder,
// tooltip, label (label caption) or content. The Texts$Text document for that
// property is located and its Items array (a versioned Texts$Translation list)
// is updated in place: an existing entry for langCode is overwritten, otherwise
// a new Texts$Translation is appended.
func (m *mprPageMutator) SetWidgetTranslation(widgetRef, prop, langCode, text string) error {
	result := m.widgetFinder(m.rawData, widgetRef)
	if result == nil {
		return fmt.Errorf("widget %q not found", widgetRef)
	}
	textsDoc := locateWidgetTextsDoc(result.widget, prop)
	if textsDoc == nil {
		return fmt.Errorf("widget %q has no translatable %q property", widgetRef, prop)
	}
	setVersionedTranslation(textsDoc, langCode, text)
	return nil
}

// SetPageTitleTranslation sets the translation for langCode on the page/snippet
// title (the root Title Texts$Text document).
func (m *mprPageMutator) SetPageTitleTranslation(langCode, text string) error {
	title := dGetDoc(m.rawData, "Title")
	if title == nil {
		return fmt.Errorf("%s has no Title property", m.containerType)
	}
	setVersionedTranslation(title, langCode, text)
	return nil
}

// locateWidgetTextsDoc returns the Texts$Text sub-document for the given
// property name on a widget, or nil if the widget does not carry it.
func locateWidgetTextsDoc(widget bson.D, prop string) bson.D {
	switch strings.ToLower(prop) {
	case "caption":
		return dGetDoc(widget, "Caption")
	case "tooltip":
		return dGetDoc(widget, "Tooltip")
	case "placeholder":
		return dGetDoc(widget, "Placeholder")
	case "label":
		if label := dGetDoc(widget, "Label"); label != nil {
			return dGetDoc(label, "Caption")
		}
		return nil
	case "content":
		if content := dGetDoc(widget, "Content"); content != nil {
			return dGetDoc(content, "Template")
		}
		return nil
	default:
		return nil
	}
}

// setVersionedTranslation updates or inserts a Texts$Translation entry inside a
// Texts$Text document whose Items array carries a leading int32 version marker.
// The marker is preserved via the dGetArrayElements / dSetArray pair.
func setVersionedTranslation(textsDoc bson.D, langCode, text string) {
	items := dGetArrayElements(dGet(textsDoc, "Items"))
	for _, item := range items {
		tr, ok := item.(bson.D)
		if !ok {
			continue
		}
		if dGetString(tr, "LanguageCode") == langCode {
			dSet(tr, "Text", text)
			return
		}
	}
	newTr := bson.D{
		{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
		{Key: "$Type", Value: "Texts$Translation"},
		{Key: "LanguageCode", Value: langCode},
		{Key: "Text", Value: text},
	}
	dSetArray(textsDoc, "Items", append(items, newTr))
}

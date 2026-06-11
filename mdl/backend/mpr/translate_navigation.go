// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/model"
)

// SetNavigationCaptionTranslation finds the navigation document, locates the
// menu item matching the caption hierarchy (menuPath), and sets the langCode
// translation on its Caption Texts$Text node.
func (b *MprBackend) SetNavigationCaptionTranslation(profileName string, menuPath []string, langCode, text string) error {
	rawUnits, err := b.msdkReader.ListRawUnitsByType(navigationDocumentBsonType)
	if err != nil {
		return fmt.Errorf("list navigation units: %w", err)
	}
	if len(rawUnits) == 0 {
		return fmt.Errorf("no navigation document found")
	}
	navUnit := rawUnits[0]

	var navDoc bson.D
	if err := bson.Unmarshal(navUnit.Contents, &navDoc); err != nil {
		return fmt.Errorf("unmarshal navigation BSON: %w", err)
	}

	if !navpSetCaptionTranslation(navDoc, profileName, menuPath, langCode, text) {
		return fmt.Errorf("menu item not found in navigation profile %q: %s", profileName, strings.Join(menuPath, "."))
	}

	contents, err := bson.Marshal(navDoc)
	if err != nil {
		return fmt.Errorf("marshal navigation BSON: %w", err)
	}
	return b.writeUnitContents(model.ID(navUnit.ID), contents)
}

// navpSetCaptionTranslation traverses the navigation BSON document to find the
// matching menu item and sets the langCode translation on its Caption.
// Returns false if the profile or menu item is not found.
func navpSetCaptionTranslation(navDoc bson.D, profileName string, menuPath []string, langCode, text string) bool {
	profiles := dGet(navDoc, "Profiles")
	if profiles == nil {
		return false
	}
	_, profileEntries := splitVersionMarker(profiles)
	for _, item := range profileEntries {
		profDoc, ok := item.(bson.D)
		if !ok {
			continue
		}
		name := dGetString(profDoc, "Name")
		if !strings.EqualFold(name, profileName) {
			continue
		}
		// Found the profile. Find its Menu.
		menuDoc := dGetDoc(profDoc, "Menu")
		if menuDoc == nil {
			return false
		}
		items := dGet(menuDoc, "Items")
		if items == nil {
			return false
		}
		return navpWalkMenuItems(items, menuPath, 0, langCode, text)
	}
	return false
}

// navpWalkMenuItems walks the menu item hierarchy, matching each segment of
// menuPath against the Caption text of items. When the last segment is matched,
// it sets the translation. Returns false if no match is found.
func navpWalkMenuItems(itemsRaw any, menuPath []string, depth int, langCode, text string) bool {
	_, entries := splitVersionMarker(itemsRaw)
	expectedCaption := menuPath[depth]
	for _, item := range entries {
		mi, ok := item.(bson.D)
		if !ok {
			continue
		}
		captionDoc := dGetDoc(mi, "Caption")
		if captionDoc == nil {
			continue
		}
		captionText := navpExtractTextFromCaptionD(captionDoc)
		if !strings.EqualFold(captionText, expectedCaption) {
			continue
		}
		if depth == len(menuPath)-1 {
			// Last segment — set the translation.
			setTranslationForLang(captionDoc, langCode, text)
			return true
		}
		// Deeper — look in sub-items.
		subItems := dGet(mi, "Items")
		if subItems == nil {
			continue
		}
		if navpWalkMenuItems(subItems, menuPath, depth+1, langCode, text) {
			return true
		}
	}
	return false
}

// navpExtractTextFromCaptionD returns the first non-empty text from a
// Texts$Text bson.D, preferring en_US.
func navpExtractTextFromCaptionD(captionDoc bson.D) string {
	raw := dGet(captionDoc, "Items")
	_, entries := splitVersionMarker(raw)
	// First pass: look for en_US
	for _, item := range entries {
		tr, ok := item.(bson.D)
		if !ok {
			continue
		}
		lang := dGetString(tr, "LanguageCode")
		text := dGetString(tr, "Text")
		if strings.EqualFold(lang, "en_US") && text != "" {
			return text
		}
	}
	// Fallback: first non-empty text in any language
	for _, item := range entries {
		tr, ok := item.(bson.D)
		if !ok {
			continue
		}
		text := dGetString(tr, "Text")
		if text != "" {
			return text
		}
	}
	return ""
}

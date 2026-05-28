// SPDX-License-Identifier: Apache-2.0

// Supplement for raw-BSON access to Texts$Text.Items, the versioned
// array of (LanguageCode, Text) translation pairs. Keeps bson decoding
// out of the executor: callers read translations via the typed
// ReadTranslationPairs helper instead of unmarshalling raw bytes.

package texts

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

// TranslationPair holds a language code and its translated text as
// extracted from a Texts$Text raw document.
type TranslationPair struct {
	LanguageCode string
	Text         string
}

// ReadTranslationPairs decodes the "Items" array of a Texts$Text raw
// BSON document and returns each translation as a TranslationPair.
// The Items array is a versioned BSON array
// `[<int32 version>, <doc>, <doc>, …]`; each `<doc>` is a
// Texts$Translation with LanguageCode + Text string fields.
// Returns nil when raw is empty, malformed, or has no Items array.
func ReadTranslationPairs(raw []byte) []TranslationPair {
	if len(raw) == 0 {
		return nil
	}
	var doc bson.M
	if err := bson.Unmarshal(raw, &doc); err != nil {
		return nil
	}
	itemsRaw, ok := doc["Items"]
	if !ok {
		return nil
	}
	arr, ok := itemsRaw.(bson.A)
	if !ok {
		return nil
	}
	var pairs []TranslationPair
	for _, item := range arr {
		m, ok := item.(bson.M)
		if !ok {
			continue
		}
		lang, _ := m["LanguageCode"].(string)
		text, _ := m["Text"].(string)
		if text == "" {
			text, _ = m["Value"].(string)
		}
		pairs = append(pairs, TranslationPair{LanguageCode: lang, Text: text})
	}
	return pairs
}

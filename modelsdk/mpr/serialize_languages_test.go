// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// TestSerializeProjectSettings_Languages verifies that the Languages array of a
// LanguageSettings part is serialized back into BSON. Regression guard for the
// ALTER SETTINGS LANGUAGE ADD/DROP write path, which previously lost all
// languages because serPSLanguageSettings only wrote DefaultLanguageCode.
func TestSerializeProjectSettings_Languages(t *testing.T) {
	t.Parallel()
	ps := &model.ProjectSettings{
		Language: &model.LanguageSettings{
			DefaultLanguageCode: "en_US",
			Languages: []model.Language{
				{Code: "en_US"},
				{Code: "nl_NL", CheckCompleteness: true, CustomDateFormat: "dd-MM-yyyy"},
			},
		},
		RawParts: []map[string]any{
			{"$Type": "Settings$LanguageSettings", "DefaultLanguageCode": "en_US"},
		},
	}
	ps.ID = "settings-id"

	raw, err := SerializeProjectSettings(ps)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	var doc bson.M
	if err := bson.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	langPart := findLanguagePart(t, doc)

	langs := extractBsonArray(langPart["Languages"])
	if len(langs) != 2 {
		t.Fatalf("expected 2 languages serialized, got %d (part=%#v)", len(langs), langPart)
	}

	got := map[string]map[string]any{}
	for _, l := range langs {
		lm := extractBsonMap(l)
		code, _ := lm["Code"].(string)
		got[code] = lm
		if tn, _ := lm["$Type"].(string); tn != "Texts$Language" {
			t.Errorf("language %s: expected $Type Texts$Language, got %v", code, lm["$Type"])
		}
		if lm["$ID"] == nil {
			t.Errorf("language %s: missing $ID", code)
		}
	}

	if _, ok := got["en_US"]; !ok {
		t.Error("en_US not serialized")
	}
	nl, ok := got["nl_NL"]
	if !ok {
		t.Fatal("nl_NL not serialized")
	}
	if cc, _ := nl["CheckCompleteness"].(bool); !cc {
		t.Errorf("nl_NL CheckCompleteness: expected true, got %v", nl["CheckCompleteness"])
	}
	if df, _ := nl["CustomDateFormat"].(string); df != "dd-MM-yyyy" {
		t.Errorf("nl_NL CustomDateFormat: expected dd-MM-yyyy, got %v", nl["CustomDateFormat"])
	}
}

func findLanguagePart(t *testing.T, doc bson.M) map[string]any {
	t.Helper()
	settings := extractBsonArray(doc["Settings"])
	for _, s := range settings {
		sm := extractBsonMap(s)
		if sm == nil {
			continue
		}
		if tn, _ := sm["$Type"].(string); tn == "Settings$LanguageSettings" {
			return sm
		}
	}
	t.Fatal("Settings$LanguageSettings part not found in serialized output")
	return nil
}

// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/model"
)

func textsNode(lang, text string) bson.D {
	return bson.D{
		{Key: "$Type", Value: "Texts$Text"},
		{Key: "Items", Value: bson.A{
			int32(3),
			bson.D{
				{Key: "$Type", Value: "Texts$Translation"},
				{Key: "LanguageCode", Value: lang},
				{Key: "Text", Value: text},
			},
		}},
	}
}

func TestCollectTranslationNodes_EnumValues(t *testing.T) {
	enumDoc := bson.D{
		{Key: "$Type", Value: "Enumerations$Enumeration"},
		{Key: "Name", Value: "Status"},
		{Key: "Values", Value: bson.A{
			int32(3),
			bson.D{
				{Key: "Name", Value: "ACTIVE"},
				{Key: "Caption", Value: textsNode("en_US", "Active")},
			},
			bson.D{
				{Key: "Name", Value: "CLOSED"},
				{Key: "Caption", Value: textsNode("en_US", "Closed")},
			},
		}},
	}

	var nodes []model.TranslationNode
	collectTranslationNodes(enumDoc, "", "ENUMERATION", &nodes)

	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d: %+v", len(nodes), nodes)
	}
	byPath := map[string]model.TranslationNode{}
	for _, n := range nodes {
		byPath[n.Path] = n
	}
	active, ok := byPath["ACTIVE.caption"]
	if !ok {
		t.Fatalf("missing ACTIVE.caption node; got paths %v", keysOf(byPath))
	}
	if active.Property != "caption" || active.DocType != "ENUMERATION" {
		t.Errorf("unexpected node fields: %+v", active)
	}
	if active.Texts["en_US"] != "Active" {
		t.Errorf("expected en_US=Active, got %q", active.Texts["en_US"])
	}
}

func TestCollectTranslationNodes_PageTitleNoOwner(t *testing.T) {
	// A Texts$Text directly under the root (no named ancestor between) yields a
	// bare property path such as a page-level title.
	pageDoc := bson.D{
		{Key: "$Type", Value: "Forms$Form"},
		{Key: "Title", Value: textsNode("en_US", "Home")},
	}

	var nodes []model.TranslationNode
	collectTranslationNodes(pageDoc, "", "PAGE", &nodes)

	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Path != "title" {
		t.Errorf("expected bare path 'title', got %q", nodes[0].Path)
	}
}

func TestDocTypeLabelFromBsonType(t *testing.T) {
	cases := map[string]string{
		"Enumerations$Enumeration": "ENUMERATION",
		"Forms$Form":               "PAGE",
		"Forms$Snippet":            "SNIPPET",
		"Workflows$Workflow":       "WORKFLOW",
		"Microflows$Microflow":     "MICROFLOW",
		"Something$Else":           "",
	}
	for in, want := range cases {
		if got := docTypeLabelFromBsonType(in); got != want {
			t.Errorf("docTypeLabelFromBsonType(%q) = %q, want %q", in, got, want)
		}
	}
}

func keysOf(m map[string]model.TranslationNode) []string {
	var k []string
	for key := range m {
		k = append(k, key)
	}
	return k
}

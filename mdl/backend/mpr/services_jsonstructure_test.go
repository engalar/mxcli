// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// TestCreateJsonStructureGen_WritesElementTree verifies that createJsonStructureGen
// serializes the parsed JsonElement tree (not just the raw JsonSnippet). Without the
// element tree, an import/export mapping referencing the structure fails mx check
// with CE0271 "The selected source is not valid."
func TestCreateJsonStructureGen_WritesElementTree(t *testing.T) {
	// Seed an existing unit so the test MPR has a valid container blob (all-zeros).
	const seedID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	seed := makeBSONUnit(t, seedID, "JsonStructures$JsonStructure", bson.D{
		{Key: "Name", Value: "Seed"},
		{Key: "JsonSnippet", Value: "{}"},
	})
	mprPath, _ := makeServiceTestMPR(t, seedID, seed)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	elements, err := types.BuildJsonElementsFromSnippet(
		`{"ticketRef": "HD-001", "techName": "Jane"}`, nil)
	if err != nil {
		t.Fatalf("BuildJsonElementsFromSnippet: %v", err)
	}

	newID := model.ID("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	js := &types.JsonStructure{
		ContainerID: model.ID("00000000-0000-0000-0000-000000000000"),
		Name:        "WorkOrderPayload",
		JsonSnippet: `{"ticketRef": "HD-001", "techName": "Jane"}`,
		Elements:    elements,
	}
	js.ID = newID

	if err := b.createJsonStructureGen(js); err != nil {
		t.Fatalf("createJsonStructureGen: %v", err)
	}

	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(string(newID))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}

	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Find the Elements array; it must contain a root JsonElement.
	var elemsArr bson.A
	for _, e := range doc {
		if e.Key == "Elements" {
			elemsArr, _ = e.Value.(bson.A)
		}
	}
	if len(elemsArr) < 2 {
		t.Fatalf("Elements array = %v, want version marker + root element", elemsArr)
	}

	root, ok := elemsArr[1].(bson.D)
	if !ok {
		t.Fatalf("Elements[1] = %T, want bson.D root element", elemsArr[1])
	}

	gotType, gotExposed, gotElemType := "", "", ""
	var childCount int
	for _, f := range root {
		switch f.Key {
		case "$Type":
			gotType, _ = f.Value.(string)
		case "ExposedName":
			gotExposed, _ = f.Value.(string)
		case "ElementType":
			gotElemType, _ = f.Value.(string)
		case "Children":
			if arr, ok := f.Value.(bson.A); ok {
				childCount = len(arr) - 1 // minus version marker
			}
		}
	}

	if gotType != "JsonStructures$JsonElement" {
		t.Errorf("root $Type = %q, want JsonStructures$JsonElement", gotType)
	}
	if gotExposed != "Root" {
		t.Errorf("root ExposedName = %q, want Root", gotExposed)
	}
	if gotElemType != "Object" {
		t.Errorf("root ElementType = %q, want Object", gotElemType)
	}
	if childCount != 2 {
		t.Errorf("root Children count = %d, want 2 (ticketRef, techName)", childCount)
	}
}

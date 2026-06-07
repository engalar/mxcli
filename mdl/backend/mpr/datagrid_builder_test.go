// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"bytes"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/bsonutil"
)

// TestFilterWidgets_EachInstanceHasUniqueTypeID verifies that each filter widget
// instance gets its own CustomWidgets$CustomWidgetType with unique $IDs.
//
// Background: sharing type-schema $IDs across instances (even of the same widgetID)
// causes CE0463 "widget definition changed" in Mendix because duplicate $IDs appear
// in the same page BSON document. The correct deduplication approach requires a
// page-level type registry — a separate architectural change to the column-filter format.
func TestFilterWidgets_EachInstanceHasUniqueTypeID(t *testing.T) {
	b := &MprBackend{}

	w1 := b.buildFilterWidgetBSON(backend.FilterWidgetSpec{
		WidgetID:   widgetIDDataGridTextFilter,
		FilterName: "fSubject",
	}, "")
	w2 := b.buildFilterWidgetBSON(backend.FilterWidgetSpec{
		WidgetID:   widgetIDDataGridTextFilter,
		FilterName: "fStatus",
	}, "")

	type1 := dGetDoc(w1, "Type")
	type2 := dGetDoc(w2, "Type")
	if type1 == nil || type2 == nil {
		t.Fatal("Type field missing from filter widget BSON")
	}

	// Every instance must have a UNIQUE type $ID to avoid CE0463 duplicate-$ID errors.
	if binaryEqual(type1[0].Value, type2[0].Value) {
		t.Error("filter widget instances must NOT share the same CustomWidgetType $ID; " +
			"shared $IDs in one page BSON cause CE0463 in Mendix")
	}
}

// TestFilterWidgets_DifferentTypesHaveDifferentTypeSchemas verifies that filter widgets
// of different widgetIDs each retain their own CustomWidgetType schema.
func TestFilterWidgets_DifferentTypesHaveDifferentTypeSchemas(t *testing.T) {
	b := &MprBackend{}

	wText := b.buildFilterWidgetBSON(backend.FilterWidgetSpec{
		WidgetID: widgetIDDataGridTextFilter, FilterName: "fText",
	}, "")
	wDrop := b.buildFilterWidgetBSON(backend.FilterWidgetSpec{
		WidgetID: widgetIDDataGridDropdownFilter, FilterName: "fDrop",
	}, "")

	typeText := dGetDoc(wText, "Type")
	typeDrop := dGetDoc(wDrop, "Type")
	if typeText == nil || typeDrop == nil {
		t.Fatal("Type field missing")
	}

	if binaryEqual(typeText[0].Value, typeDrop[0].Value) {
		t.Error("textfilter and dropdownfilter must have different CustomWidgetType $IDs")
	}
}

// TestFilterWidgets_BeginEndPageBuildIsNoOp verifies that BeginPageBuild/EndPageBuild
// exist and do not break widget construction (reserved for future type-registry work).
func TestFilterWidgets_BeginEndPageBuildIsNoOp(t *testing.T) {
	b := &MprBackend{}
	b.BeginPageBuild()
	defer b.EndPageBuild()

	w := b.buildFilterWidgetBSON(backend.FilterWidgetSpec{
		WidgetID: widgetIDDataGridTextFilter, FilterName: "f1",
	}, "")

	if dGetDoc(w, "Type") == nil {
		t.Error("BeginPageBuild must not break filter widget BSON construction")
	}
}


func TestDeepCloneWithNewIDs_RegeneratesAllIDs(t *testing.T) {
	origID1 := bsonutil.NewIDBsonBinary()
	origID2 := bsonutil.NewIDBsonBinary()
	origID3 := bsonutil.NewIDBsonBinary()

	doc := bson.D{
		{Key: "$ID", Value: origID1},
		{Key: "$Type", Value: "Forms$TextBox"},
		{Key: "AttributeRef", Value: bson.D{
			{Key: "$ID", Value: origID2},
			{Key: "$Type", Value: "DomainModels$AttributeRef"},
			{Key: "Attribute", Value: "Module.Entity.Name"},
			{Key: "EntityRef", Value: bson.D{
				{Key: "$ID", Value: origID3},
				{Key: "$Type", Value: "DomainModels$DirectEntityRef"},
				{Key: "Entity", Value: "Module.Entity"},
			}},
		}},
		{Key: "Name", Value: "txtName"},
	}

	cloned := deepCloneWithNewIDs(doc)

	if dGetString(cloned, "$Type") != "Forms$TextBox" {
		t.Error("$Type not preserved")
	}
	if dGetString(cloned, "Name") != "txtName" {
		t.Error("Name not preserved")
	}

	clonedID1 := cloned[0].Value
	if binaryEqual(clonedID1, origID1) {
		t.Error("top-level $ID was not regenerated")
	}

	attrRef := dGetDoc(cloned, "AttributeRef")
	if attrRef == nil {
		t.Fatal("AttributeRef missing")
	}
	clonedID2 := attrRef[0].Value
	if binaryEqual(clonedID2, origID2) {
		t.Error("AttributeRef $ID was not regenerated — stale GUID would cause CE1613")
	}
	if dGetString(attrRef, "Attribute") != "Module.Entity.Name" {
		t.Error("Attribute value not preserved")
	}

	entityRef := dGetDoc(attrRef, "EntityRef")
	if entityRef == nil {
		t.Fatal("EntityRef missing")
	}
	clonedID3 := entityRef[0].Value
	if binaryEqual(clonedID3, origID3) {
		t.Error("EntityRef $ID was not regenerated")
	}
	if dGetString(entityRef, "Entity") != "Module.Entity" {
		t.Error("Entity value not preserved")
	}
}

func TestDeepCloneWithNewIDs_HandlesArrays(t *testing.T) {
	origID := bsonutil.NewIDBsonBinary()
	innerID := bsonutil.NewIDBsonBinary()

	doc := bson.D{
		{Key: "$ID", Value: origID},
		{Key: "Items", Value: bson.A{
			int32(2),
			bson.D{
				{Key: "$ID", Value: innerID},
				{Key: "$Type", Value: "SomeType"},
				{Key: "Value", Value: "test"},
			},
		}},
	}

	cloned := deepCloneWithNewIDs(doc)

	items := cloned[1].Value.(bson.A)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	itemDoc := items[1].(bson.D)
	if binaryEqual(itemDoc[0].Value, innerID) {
		t.Error("nested array item $ID was not regenerated")
	}
	if dGetString(itemDoc, "Value") != "test" {
		t.Error("nested value not preserved")
	}
}

func TestDeepCloneWithNewIDs_PreservesNil(t *testing.T) {
	origID := bsonutil.NewIDBsonBinary()

	doc := bson.D{
		{Key: "$ID", Value: origID},
		{Key: "EntityRef", Value: nil},
		{Key: "Name", Value: "test"},
	}

	cloned := deepCloneWithNewIDs(doc)
	if cloned[1].Value != nil {
		t.Error("nil value not preserved")
	}
	if dGetString(cloned, "Name") != "test" {
		t.Error("string value not preserved")
	}
}

// Test helpers

func binaryEqual(a, b any) bool {
	ab, aOk := a.(bson.Binary)
	bb, bOk := b.(bson.Binary)
	if !aOk || !bOk {
		return false
	}
	return bytes.Equal(ab.Data, bb.Data)
}

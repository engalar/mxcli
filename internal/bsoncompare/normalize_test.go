package bsoncompare_test

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

func makeID(b byte) primitive.Binary {
	data := make([]byte, 16)
	for i := range data {
		data[i] = b
	}
	return primitive.Binary{Data: data}
}

func TestNormalize_SelfIDOmitted(t *testing.T) {
	m := bsoncompare.IDMap{}
	doc := bson.D{{Key: "$ID", Value: makeID(0xAA)}, {Key: "Name", Value: "Foo"}}
	n := bsoncompare.Normalize(doc, m, bsoncompare.DefaultOptions())
	if _, ok := n["$ID"]; ok {
		t.Error("$ID must be omitted from normalized output")
	}
	if n["Name"] != "Foo" {
		t.Errorf("Name must be preserved, got %v", n["Name"])
	}
}

func TestNormalize_PointerResolved(t *testing.T) {
	id := makeID(0xBB)
	m := bsoncompare.IDMap{bsoncompare.HexOf(id.Data): "Microflow:ACT_Save"}
	doc := bson.D{{Key: "TargetPointer", Value: id}}
	n := bsoncompare.Normalize(doc, m, bsoncompare.DefaultOptions())
	if n["TargetPointer"] != "<ref:Microflow:ACT_Save>" {
		t.Errorf("got %v, want <ref:Microflow:ACT_Save>", n["TargetPointer"])
	}
}

func TestNormalize_UnknownPointer(t *testing.T) {
	m := bsoncompare.IDMap{}
	doc := bson.D{{Key: "TargetPointer", Value: makeID(0xCC)}}
	n := bsoncompare.Normalize(doc, m, bsoncompare.DefaultOptions())
	if n["TargetPointer"] != "<ref:?>" {
		t.Errorf("got %v, want <ref:?>", n["TargetPointer"])
	}
}

func TestNormalize_StableIdOmitted(t *testing.T) {
	m := bsoncompare.IDMap{}
	doc := bson.D{{Key: "StableId", Value: makeID(0xDD)}, {Key: "Name", Value: "X"}}
	n := bsoncompare.Normalize(doc, m, bsoncompare.DefaultOptions())
	if _, ok := n["StableId"]; ok {
		t.Error("StableId must be omitted when IgnoreStableId=true")
	}
}

func TestNormalize_LayoutFieldsOmitted(t *testing.T) {
	m := bsoncompare.IDMap{}
	doc := bson.D{
		{Key: "CanvasHeight", Value: int64(600)},
		{Key: "CanvasWidth", Value: int64(1200)},
		{Key: "DestinationControlVector", Value: "-15;0"},
		{Key: "ExportLevel", Value: "Hidden"},
	}
	n := bsoncompare.Normalize(doc, m, bsoncompare.DefaultOptions())
	for _, k := range []string{"CanvasHeight", "CanvasWidth", "DestinationControlVector"} {
		if _, ok := n[k]; ok {
			t.Errorf("%s must be omitted by layout ignore", k)
		}
	}
	if n["ExportLevel"] != "Hidden" {
		t.Errorf("ExportLevel must be preserved, got %v", n["ExportLevel"])
	}
}

func TestNormalize_DocumentationOmitted(t *testing.T) {
	m := bsoncompare.IDMap{}
	doc := bson.D{{Key: "Documentation", Value: "some docs"}, {Key: "Name", Value: "Y"}}
	n := bsoncompare.Normalize(doc, m, bsoncompare.DefaultOptions())
	if _, ok := n["Documentation"]; ok {
		t.Error("Documentation must be omitted when IgnoreDocumentation=true")
	}
}

func TestNormalize_VersionedArrayPrefixSkipped(t *testing.T) {
	m := bsoncompare.IDMap{}
	doc := bson.D{{Key: "Items", Value: bson.A{int32(2), "hello", "world"}}}
	n := bsoncompare.Normalize(doc, m, bsoncompare.DefaultOptions())
	items, ok := n["Items"].([]any)
	if !ok {
		t.Fatalf("Items must be a slice, got %T", n["Items"])
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items (prefix stripped), got %d", len(items))
	}
	if items[0] != "hello" {
		t.Errorf("first item must be hello, got %v", items[0])
	}
}

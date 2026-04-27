package codec

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	"github.com/mendixlabs/mxcli/modelsdk/property"
	"go.mongodb.org/mongo-driver/bson"
)

func TestEncoderNewElementHasBinaryID(t *testing.T) {
	elem := &element.Base{}
	elem.SetTypeName("Test$New")
	elem.SetID("aaaabbbb-cccc-dddd-eeee-ffffffffffff")
	elem.MarkDirty(63) // new element

	enc := &Encoder{}
	out, err := enc.Encode(elem)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	raw := bson.Raw(out)
	idVal, _ := raw.LookupErr("$ID")
	if idVal.Type.String() != "binary" {
		t.Errorf("$ID type = %s, want binary", idVal.Type)
	}
}

func TestEncoderPreservesUnknownFields(t *testing.T) {
	original := mustMarshal(bson.D{
		{Key: "$ID", Value: "id-1"},
		{Key: "$Type", Value: "Test$X"},
		{Key: "known", Value: "v1"},
		{Key: "unknown_field", Value: int32(42)},
		{Key: "another_unknown", Value: true},
	})

	knownProp := property.NewPrimitive[string]("known", property.DecodeString)
	knownProp.Init(original)

	elem := &element.Base{}
	elem.SetID("id-1")
	elem.SetTypeName("Test$X")
	elem.SetRaw(original)
	elem.SetProperties([]element.Property{knownProp})

	// Modify the known property
	knownProp.Bind(elem, 0)
	knownProp.Set("v2")

	enc := &Encoder{}
	out, err := enc.Encode(elem)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	var doc bson.D
	bson.Unmarshal(out, &doc)
	fields := map[string]any{}
	for _, e := range doc {
		fields[e.Key] = e.Value
	}

	if fields["known"] != "v2" {
		t.Errorf("known = %v, want v2", fields["known"])
	}
	if fields["unknown_field"] != int32(42) {
		t.Errorf("unknown_field = %v, want 42", fields["unknown_field"])
	}
	if fields["another_unknown"] != true {
		t.Errorf("another_unknown = %v, want true", fields["another_unknown"])
	}
}

func TestEncoderPartDirty(t *testing.T) {
	parentRaw := mustMarshal(bson.D{
		{Key: "$ID", Value: "p-1"},
		{Key: "$Type", Value: "Test$P"},
		{Key: "child", Value: bson.D{
			{Key: "$ID", Value: "c-1"},
			{Key: "$Type", Value: "Test$C"},
			{Key: "val", Value: "old"},
		}},
	})

	child := &element.Base{}
	child.SetID("c-1")
	child.SetTypeName("Test$C")
	child.SetRaw(mustMarshal(bson.D{
		{Key: "$ID", Value: "c-1"},
		{Key: "$Type", Value: "Test$C"},
		{Key: "val", Value: "old"},
	}))
	valProp := property.NewPrimitive[string]("val", property.DecodeString)
	valProp.Init(child.Raw())
	valProp.Bind(child, 0)
	valProp.Set("new")
	child.SetProperties([]element.Property{valProp})

	parent := &element.Base{}
	parent.SetID("p-1")
	parent.SetTypeName("Test$P")
	parent.SetRaw(parentRaw)

	childPart := property.NewPart[element.Element]("child")
	childPart.Bind(parent, 0)
	parent.SetProperties([]element.Property{childPart})

	// Mark the Part as dirty by setting the child
	childPart.Set(child)

	enc := &Encoder{}
	out, err := enc.Encode(parent)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	var doc bson.D
	bson.Unmarshal(out, &doc)
	for _, e := range doc {
		if e.Key == "child" {
			childDoc := e.Value.(bson.D)
			for _, f := range childDoc {
				if f.Key == "val" && f.Value != "new" {
					t.Errorf("child.val = %v, want new", f.Value)
				}
			}
		}
	}
}

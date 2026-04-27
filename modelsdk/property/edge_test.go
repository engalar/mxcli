package property

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	"go.mongodb.org/mongo-driver/bson"
)

// --- Primitive edge cases ---

func TestPrimitiveGetBeforeInit(t *testing.T) {
	p := NewPrimitive[int32]("count", DecodeInt32)
	// No Init, no raw — should return zero
	if got := p.Get(); got != 0 {
		t.Errorf("Get() = %d, want 0", got)
	}
}

func TestPrimitiveSetOverridesLazy(t *testing.T) {
	raw := mustMarshal(bson.D{{Key: "name", Value: "lazy"}})
	p := NewPrimitive[string]("name", DecodeString)
	p.Init(raw)

	p.Set("eager")
	if p.Get() != "eager" {
		t.Errorf("Set should override lazy, got %q", p.Get())
	}
}

func TestPrimitiveFloat64(t *testing.T) {
	raw := mustMarshal(bson.D{{Key: "val", Value: 3.14}})
	p := NewPrimitive[float64]("val", DecodeFloat64)
	p.Init(raw)
	if got := p.Get(); got != 3.14 {
		t.Errorf("Get() = %f, want 3.14", got)
	}
}

func TestPrimitiveBSONValue(t *testing.T) {
	p := NewPrimitive[string]("name", DecodeString)
	p.Set("test")
	if p.BSONValue() != "test" {
		t.Errorf("BSONValue() = %v", p.BSONValue())
	}
}

// --- Part edge cases ---

func TestPartSetNil(t *testing.T) {
	p := NewPart[element.Element]("gen")
	p.Set(nil)
	if p.Get() != nil {
		t.Error("Set(nil) then Get should return nil")
	}
	if !p.Dirty() {
		t.Error("Set(nil) should still mark dirty")
	}
}

func TestPartChildElement(t *testing.T) {
	p := NewPart[element.Element]("gen")
	if p.ChildElement() != nil {
		t.Error("unset Part.ChildElement should be nil")
	}
	child := &element.Base{}
	p.Set(child)
	if p.ChildElement() != child {
		t.Error("ChildElement should return the set child")
	}
}

// --- PartList edge cases ---

func TestPartListRemoveOutOfBoundsEdge(t *testing.T) {
	pl := NewPartList[element.Element]("items")
	pl.Append(&element.Base{})

	pl.Remove(-1)
	pl.Remove(999)
	if pl.Len() != 1 {
		t.Errorf("out-of-bounds Remove should not change Len, got %d", pl.Len())
	}
}

func TestPartListChildElements(t *testing.T) {
	pl := NewPartList[element.Element]("items")
	a := &element.Base{}
	b := &element.Base{}
	pl.AppendFromDecode(a)
	pl.AppendFromDecode(b)

	children := pl.ChildElements()
	if len(children) != 2 {
		t.Errorf("ChildElements len = %d, want 2", len(children))
	}
	if children[0] != a || children[1] != b {
		t.Error("ChildElements order wrong")
	}
}

func TestPartListAppendFromDecodeNotDirty(t *testing.T) {
	pl := NewPartList[element.Element]("items")
	pl.AppendFromDecode(&element.Base{})
	if pl.Dirty() {
		t.Error("AppendFromDecode should not mark dirty")
	}
}

// --- ByNameRef edge cases ---

func TestByNameRefSetFromDecodeNotDirty(t *testing.T) {
	r := NewByNameRef[element.Element]("img", "Images$Image")
	r.SetFromDecode("Mod.Img")
	if r.Dirty() {
		t.Error("SetFromDecode should not mark dirty")
	}
	if r.QualifiedName() != "Mod.Img" {
		t.Errorf("QualifiedName = %q", r.QualifiedName())
	}
}

func TestByNameRefBSONValue(t *testing.T) {
	r := NewByNameRef[element.Element]("img", "Images$Image")
	r.SetQualifiedName("Mod.Img")
	if r.BSONValue() != "Mod.Img" {
		t.Errorf("BSONValue = %v", r.BSONValue())
	}
}

// --- ByNameRefList edge cases ---

func TestByNameRefListAppend(t *testing.T) {
	r := NewByNameRefList[element.Element]("roles", "Security$ModuleRole")
	r.Append("Admin")
	r.Append("User")
	if len(r.QualifiedNames()) != 2 {
		t.Errorf("len = %d", len(r.QualifiedNames()))
	}
	if !r.Dirty() {
		t.Error("should be dirty after Append")
	}
}

// --- ByIdRef edge cases ---

func TestByIdRefSetFromDecodeNotDirty(t *testing.T) {
	r := NewByIdRef[element.Element]("child")
	r.SetFromDecode("id-123")
	if r.Dirty() {
		t.Error("SetFromDecode should not mark dirty")
	}
}

// --- Enum edge cases ---

func TestEnumSetFromDecodeNotDirty(t *testing.T) {
	e := NewEnum[string]("level")
	e.SetFromDecode("Hidden")
	if e.Dirty() {
		t.Error("SetFromDecode should not mark dirty")
	}
	if e.Get() != "Hidden" {
		t.Errorf("Get = %q", e.Get())
	}
}

func TestEnumBSONValue(t *testing.T) {
	e := NewEnum[string]("level")
	e.Set("API")
	if e.BSONValue() != "API" {
		t.Errorf("BSONValue = %v", e.BSONValue())
	}
}

// --- EnumList ---

func TestEnumListAppend(t *testing.T) {
	el := NewEnumList[string]("tags")
	el.Append("A")
	el.Append("B")
	if len(el.Items()) != 2 {
		t.Errorf("Items len = %d", len(el.Items()))
	}
	if !el.Dirty() {
		t.Error("should be dirty")
	}
}

func TestEnumListBSONValue(t *testing.T) {
	el := NewEnumList[string]("tags")
	el.Append("X")
	bv := el.BSONValue().([]string)
	if len(bv) != 1 || bv[0] != "X" {
		t.Errorf("BSONValue = %v", bv)
	}
}

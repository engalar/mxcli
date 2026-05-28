package property

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func mustMarshal(d bson.D) bson.Raw {
	b, err := bson.Marshal(d)
	if err != nil {
		panic(err)
	}
	return bson.Raw(b)
}

func TestPrimitiveString(t *testing.T) {
	raw := mustMarshal(bson.D{{Key: "name", Value: "Customer"}})

	p := NewPrimitive[string]("name", DecodeString)
	p.Init(raw)

	if got := p.Get(); got != "Customer" {
		t.Errorf("expected 'Customer', got %q", got)
	}
	if p.Dirty() {
		t.Error("expected Dirty() == false before Set()")
	}

	p.Set("Order")

	if got := p.Get(); got != "Order" {
		t.Errorf("expected 'Order' after Set(), got %q", got)
	}
	if !p.Dirty() {
		t.Error("expected Dirty() == true after Set()")
	}
}

func TestPrimitiveBool(t *testing.T) {
	raw := mustMarshal(bson.D{{Key: "excluded", Value: true}})

	p := NewPrimitive[bool]("excluded", DecodeBool)
	p.Init(raw)

	if got := p.Get(); !got {
		t.Errorf("expected true, got %v", got)
	}
}

func TestPrimitiveInt32(t *testing.T) {
	raw := mustMarshal(bson.D{{Key: "length", Value: int32(200)}})

	p := NewPrimitive[int32]("length", DecodeInt32)
	p.Init(raw)

	if got := p.Get(); got != 200 {
		t.Errorf("expected 200, got %d", got)
	}
}

func TestPrimitiveDefault(t *testing.T) {
	p := NewPrimitive[string]("name", DecodeString)
	// no Init call — raw remains nil

	if got := p.Get(); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
	if p.Dirty() {
		t.Error("expected Dirty() == false when no raw and no Set()")
	}
}

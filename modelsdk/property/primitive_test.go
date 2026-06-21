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

func TestBinaryPrimitive(t *testing.T) {
	blob := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic bytes
	raw := mustMarshal(bson.D{{Key: "image", Value: bson.Binary{Subtype: 0x00, Data: blob}}})

	p := NewBinaryPrimitive("image")
	p.Init(raw)

	got := p.Get()
	if len(got) != 4 {
		t.Fatalf("expected 4 bytes, got %d", len(got))
	}
	if got[0] != 0x89 || got[1] != 0x50 || got[2] != 0x4E || got[3] != 0x47 {
		t.Errorf("expected PNG magic bytes, got %v", got)
	}
	if p.Dirty() {
		t.Error("expected Dirty() == false before Set()")
	}

	p.Set([]byte{0x01, 0x02})
	if !p.Dirty() {
		t.Error("expected Dirty() == true after Set()")
	}
	if got2 := p.Get(); len(got2) != 2 || got2[0] != 0x01 || got2[1] != 0x02 {
		t.Errorf("expected [1, 2] after Set(), got %v", got2)
	}
}

func TestBinaryPrimitiveMissing(t *testing.T) {
	raw := mustMarshal(bson.D{{Key: "other", Value: "value"}})

	p := NewBinaryPrimitive("image")
	p.Init(raw)

	if got := p.Get(); got != nil {
		t.Errorf("expected nil for missing key, got %v", got)
	}
}

func TestBinaryPrimitiveNotBinary(t *testing.T) {
	raw := mustMarshal(bson.D{{Key: "image", Value: "not-binary"}})

	p := NewBinaryPrimitive("image")
	p.Init(raw)

	if got := p.Get(); got != nil {
		t.Errorf("expected nil for non-binary field, got %v", got)
	}
}

func TestBinaryPrimitiveNoInit(t *testing.T) {
	p := NewBinaryPrimitive("image")

	if got := p.Get(); got != nil {
		t.Errorf("expected nil when no Init, got %v", got)
	}
	if p.Dirty() {
		t.Error("expected Dirty() == false when no Init and no Set()")
	}
}

func TestBinaryPrimitiveBSONValue(t *testing.T) {
	blob := []byte{0x01, 0x02, 0x03}
	p := NewBinaryPrimitive("image")
	p.Set(blob)

	val := p.BSONValue()
	bv, ok := val.(bson.Binary)
	if !ok {
		t.Fatalf("expected bson.Binary, got %T", val)
	}
	if bv.Subtype != 0x00 {
		t.Errorf("expected Subtype 0x00, got 0x%02x", bv.Subtype)
	}
	if len(bv.Data) != 3 || bv.Data[0] != 0x01 || bv.Data[1] != 0x02 || bv.Data[2] != 0x03 {
		t.Errorf("expected [1,2,3], got %v", bv.Data)
	}
}

func TestBinaryPrimitiveRoundTrip(t *testing.T) {
	blob := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	doc := bson.D{{Key: "image", Value: bson.Binary{Subtype: 0x00, Data: blob}}}
	raw := mustMarshal(doc)

	p := NewBinaryPrimitive("image")
	p.Init(raw)

	got := p.Get()
	if len(got) != 4 || got[0] != 0xDE || got[3] != 0xEF {
		t.Errorf("round-trip mismatch: got %v, want %v", got, blob)
	}

	// Marshal back via BSONValue
	out, err := bson.Marshal(bson.D{{Key: "image", Value: p.BSONValue()}})
	if err != nil {
		t.Fatal(err)
	}
	var decoded bson.D
	if err := bson.Unmarshal(out, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, elem := range decoded {
		if elem.Key == "image" {
			bv, ok := elem.Value.(bson.Binary)
			if !ok {
				t.Fatalf("expected bson.Binary, got %T", elem.Value)
			}
			if len(bv.Data) != 4 || bv.Data[0] != 0xDE || bv.Data[3] != 0xEF {
				t.Errorf("round-trip value mismatch: got %v, want %v", bv.Data, blob)
			}
		}
	}
}

func TestDecodeBinary(t *testing.T) {
	blob := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG magic bytes
	raw := mustMarshal(bson.D{{Key: "img", Value: bson.Binary{Subtype: 0x00, Data: blob}}})

	got := DecodeBinary(raw, "img")
	if len(got) != 4 {
		t.Fatalf("expected 4 bytes, got %d", len(got))
	}
	if got[0] != 0xFF || got[1] != 0xD8 {
		t.Errorf("expected JPEG magic bytes, got %v", got)
	}
}

func TestDecodeBinaryMissingKey(t *testing.T) {
	raw := mustMarshal(bson.D{{Key: "other", Value: "value"}})
	if got := DecodeBinary(raw, "missing"); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestDecodeBinaryStringField(t *testing.T) {
	raw := mustMarshal(bson.D{{Key: "img", Value: "not-binary"}})
	if got := DecodeBinary(raw, "img"); got != nil {
		t.Errorf("expected nil for string field, got %v", got)
	}
}

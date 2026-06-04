package property

import (
	"encoding/hex"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// DecodeFunc extracts a value of type T from bson.Raw by key name.
type DecodeFunc[T any] func(raw bson.Raw, key string) T

// Primitive[T] is a lazy-decoded scalar property.
type Primitive[T any] struct {
	propertyBase
	decode DecodeFunc[T]
	raw    bson.Raw
	val    T
	loaded bool
}

func NewPrimitive[T any](name string, decode DecodeFunc[T]) *Primitive[T] {
	return &Primitive[T]{propertyBase: propertyBase{name: name}, decode: decode}
}

func (p *Primitive[T]) Init(raw bson.Raw) { p.raw = raw }

func (p *Primitive[T]) Get() T {
	if !p.loaded {
		if p.raw != nil {
			p.val = p.decode(p.raw, p.name)
		}
		p.loaded = true
	}
	return p.val
}

func (p *Primitive[T]) Set(v T) {
	p.val = v
	p.loaded = true
	p.markDirty()
}

// BSONValue returns the current value for BSON serialization.
func (p *Primitive[T]) BSONValue() any { return p.Get() }

// --- Decode functions for common types ---

func DecodeString(raw bson.Raw, key string) string {
	val, err := raw.LookupErr(key)
	if err != nil {
		return ""
	}
	s, _ := val.StringValueOK()
	return s
}

func DecodeBool(raw bson.Raw, key string) bool {
	val, err := raw.LookupErr(key)
	if err != nil {
		return false
	}
	b, _ := val.BooleanOK()
	return b
}

func DecodeInt32(raw bson.Raw, key string) int32 {
	val, err := raw.LookupErr(key)
	if err != nil {
		return 0
	}
	i, _ := val.Int32OK()
	return i
}

func DecodeFloat64(raw bson.Raw, key string) float64 {
	val, err := raw.LookupErr(key)
	if err != nil {
		return 0
	}
	f, _ := val.DoubleOK()
	return f
}

// DecodeBinaryUUID reads a BSON Binary field and returns it as a UUID string
// (e.g. "241468d1-387d-4585-a7a4-1880a96998b8"). Mendix stores identifiers like
// PersistentId as BSON Binary subtype 0 in Microsoft GUID byte-swapped format.
// Returns "" when the field is absent or not a Binary value.
func DecodeBinaryUUID(raw bson.Raw, key string) string {
	val, err := raw.LookupErr(key)
	if err != nil {
		return ""
	}
	_, data, ok := val.BinaryOK()
	if !ok || len(data) != 16 {
		return ""
	}
	// Reverse the Microsoft GUID byte-swap to recover the original UUID bytes.
	var u [16]byte
	u[0] = data[3]; u[1] = data[2]; u[2] = data[1]; u[3] = data[0]
	u[4] = data[5]; u[5] = data[4]
	u[6] = data[7]; u[7] = data[6]
	copy(u[8:], data[8:])
	h := hex.EncodeToString(u[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

// EncodeBinaryUUID converts a UUID string to a BSON Binary value in Mendix
// Microsoft GUID byte-swapped format. Returns nil when id is "".
func EncodeBinaryUUID(id string) any {
	if id == "" {
		return nil
	}
	clean := strings.ReplaceAll(id, "-", "")
	decoded, err := hex.DecodeString(clean)
	if err != nil || len(decoded) != 16 {
		return nil
	}
	blob := make([]byte, 16)
	blob[0] = decoded[3]; blob[1] = decoded[2]; blob[2] = decoded[1]; blob[3] = decoded[0]
	blob[4] = decoded[5]; blob[5] = decoded[4]
	blob[6] = decoded[7]; blob[7] = decoded[6]
	copy(blob[8:], decoded[8:])
	return bson.Binary{Subtype: 0x00, Data: blob}
}

// BinaryUUIDPrimitive stores a UUID string internally but reads/writes as
// a BSON Binary (Mendix GUID byte-swapped format). Used for PersistentId fields
// that Studio Pro serializes as Binary rather than as a plain string.
type BinaryUUIDPrimitive struct {
	propertyBase
	raw  bson.Raw
	val  string
	loaded bool
}

func NewBinaryUUIDPrimitive(name string) *BinaryUUIDPrimitive {
	return &BinaryUUIDPrimitive{propertyBase: propertyBase{name: name}}
}

func (p *BinaryUUIDPrimitive) Init(raw bson.Raw) { p.raw = raw }

func (p *BinaryUUIDPrimitive) Get() string {
	if !p.loaded {
		if p.raw != nil {
			p.val = DecodeBinaryUUID(p.raw, p.name)
		}
		p.loaded = true
	}
	return p.val
}

func (p *BinaryUUIDPrimitive) Set(v string) {
	p.val = v
	p.loaded = true
	p.markDirty()
}

// BSONValue returns the UUID as a BSON Binary for serialization.
func (p *BinaryUUIDPrimitive) BSONValue() any {
	return EncodeBinaryUUID(p.Get())
}

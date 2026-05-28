package property

import (
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

package property

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func BenchmarkPrimitiveLazyDecode(b *testing.B) {
	raw := mustMarshal(bson.D{
		{Key: "name", Value: "BenchmarkEntity"},
		{Key: "doc", Value: "some documentation text"},
		{Key: "count", Value: int32(42)},
		{Key: "flag", Value: true},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := NewPrimitive[string]("name", DecodeString)
		p.Init(raw)
		_ = p.Get() // lazy decode
	}
}

func BenchmarkPrimitiveSetGet(b *testing.B) {
	p := NewPrimitive[string]("name", DecodeString)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Set("value")
		_ = p.Get()
	}
}

func BenchmarkPrimitiveCachedGet(b *testing.B) {
	raw := mustMarshal(bson.D{{Key: "name", Value: "cached"}})
	p := NewPrimitive[string]("name", DecodeString)
	p.Init(raw)
	_ = p.Get() // warm cache

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.Get() // cached read
	}
}

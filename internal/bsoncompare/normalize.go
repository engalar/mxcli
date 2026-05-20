// SPDX-License-Identifier: Apache-2.0
package bsoncompare

import (
	"encoding/hex"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func HexOf(data []byte) string { return hex.EncodeToString(data) }

func Normalize(doc bson.D, m IDMap, opts Options) map[string]any {
	return normalizeDoc(doc, m, opts)
}

func normalizeDoc(doc bson.D, m IDMap, opts Options) map[string]any {
	out := make(map[string]any, len(doc))
	for _, e := range doc {
		if shouldIgnore(e.Key, opts) {
			continue
		}
		out[e.Key] = normalizeVal(e.Value, m, opts)
	}
	return out
}

func normalizeVal(v any, m IDMap, opts Options) any {
	switch val := v.(type) {
	case primitive.Binary:
		if len(val.Data) == 16 {
			return m.Lookup(val.Data)
		}
		return fmt.Sprintf("<binary:%d>", len(val.Data))
	case bson.D:
		return normalizeDoc(val, m, opts)
	case bson.A:
		return normalizeArray(val, m, opts)
	default:
		return val
	}
}

func normalizeArray(arr bson.A, m IDMap, opts Options) []any {
	start := 0
	if len(arr) > 0 {
		if _, ok := arr[0].(int32); ok {
			start = 1
		}
	}
	cap := len(arr) - start
	if cap < 0 {
		cap = 0
	}
	out := make([]any, 0, cap)
	for i := start; i < len(arr); i++ {
		out = append(out, normalizeVal(arr[i], m, opts))
	}
	return out
}

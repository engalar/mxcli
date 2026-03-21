// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"encoding/base64"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// extractBsonID extracts an ID string from various BSON ID representations.
// Mendix stores IDs as Binary with Subtype/Data or as primitive.Binary.
func extractBsonID(v any) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return blobToUUID(val)
	case primitive.Binary:
		return blobToUUID(val.Data)
	case map[string]any:
		// Binary UUID stored as {Subtype: 0, Data: "base64..."}
		if data, ok := val["Data"].(string); ok {
			decoded, err := base64.StdEncoding.DecodeString(data)
			if err == nil {
				return blobToUUID(decoded)
			}
		}
		// Also try $ID field
		if id, ok := val["$ID"]; ok {
			return extractBsonID(id)
		}
	}

	return ""
}

// extractInt extracts an integer from various BSON number types.
func extractInt(v any) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int32:
		return int(val)
	case int64:
		return int(val)
	case int:
		return val
	case float64:
		return int(val)
	}
	return 0
}

// extractString extracts a string from various BSON representations.
func extractString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// extractBool extracts a boolean from BSON, with default value.
func extractBool(v any, defaultVal bool) bool {
	if v == nil {
		return defaultVal
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return defaultVal
}

// extractBsonArray extracts items from a Mendix BSON array.
// Mendix arrays start with a type indicator (2 or 3 for storageListType), followed by items.
func extractBsonArray(v any) []any {
	if v == nil {
		return nil
	}

	arr, ok := v.(primitive.A)
	if !ok {
		// Try regular slice
		if slice, ok := v.([]any); ok {
			// Check if first element is the array type indicator
			if len(slice) > 0 {
				if typeIndicator, ok := slice[0].(int32); ok && (typeIndicator == 2 || typeIndicator == 3) {
					// Skip the type indicator
					return slice[1:]
				}
			}
			return slice
		}
		return nil
	}

	// primitive.A is []interface{} underneath
	slice := []any(arr)

	// Check if first element is the array type indicator (2 or 3)
	if len(slice) > 0 {
		if typeIndicator, ok := slice[0].(int32); ok && (typeIndicator == 2 || typeIndicator == 3) {
			// Skip the type indicator
			return slice[1:]
		}
	}

	return slice
}

// extractBsonMap coerces a BSON value to map[string]interface{}.
// Handles map[string]interface{}, primitive.D, and primitive.M.
func extractBsonMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case map[string]any:
		return val
	case primitive.D:
		return val.Map()
	case primitive.M:
		return map[string]any(val)
	}
	return nil
}

// extractBsonSlice coerces a BSON value to []interface{}.
// Handles []interface{} and primitive.A. Unlike extractBsonArray,
// this does NOT strip Mendix type-indicator prefixes.
func extractBsonSlice(v any) []any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []any:
		return val
	case primitive.A:
		return []any(val)
	}
	return nil
}

// BsonArrayInfo holds the extracted items and the marker from a Mendix BSON array.
type BsonArrayInfo struct {
	Marker int32
	Items  []any
}

// extractBsonArrayWithMarker extracts items from a Mendix BSON array, preserving the marker.
// Returns the marker (2 or 3) and the items after the marker.
func extractBsonArrayWithMarker(v any) BsonArrayInfo {
	if v == nil {
		return BsonArrayInfo{}
	}

	var slice []any
	switch val := v.(type) {
	case primitive.A:
		slice = []any(val)
	case []any:
		slice = val
	default:
		return BsonArrayInfo{}
	}

	if len(slice) > 0 {
		if marker, ok := slice[0].(int32); ok && (marker == 2 || marker == 3) {
			return BsonArrayInfo{Marker: marker, Items: slice[1:]}
		}
	}
	return BsonArrayInfo{Items: slice}
}

// inferPropertyKind determines whether a BSON field value represents a primitive,
// a by-name reference, or a part (embedded object). This is used by the generic
// unknown-element parser to preserve raw fields with semantic context.
func inferPropertyKind(key string, v any) string {
	if v == nil {
		return "primitive"
	}

	switch val := v.(type) {
	case map[string]any:
		if _, hasType := val["$Type"]; hasType {
			return "part"
		}
		if _, hasID := val["$ID"]; hasID {
			return "part"
		}
		return "primitive"

	case primitive.D:
		m := val.Map()
		if _, hasType := m["$Type"]; hasType {
			return "part"
		}
		if _, hasID := m["$ID"]; hasID {
			return "part"
		}
		return "primitive"

	case primitive.M:
		if _, hasType := val["$Type"]; hasType {
			return "part"
		}
		if _, hasID := val["$ID"]; hasID {
			return "part"
		}
		return "primitive"

	case primitive.A, []any:
		return "collection"

	case string:
		// Heuristic: qualified names like "Module.Entity" are likely by-name references
		if strings.Contains(val, ".") && !strings.Contains(val, " ") && !strings.Contains(val, "/") {
			return "by-name-reference"
		}
		return "primitive"

	default:
		return "primitive"
	}
}

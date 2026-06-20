// SPDX-License-Identifier: Apache-2.0

// TDD test: protect against Duplicate Guid when multiple widget instances
// of the same type are on the same page. Mendix raises InvalidOperationException
// "Duplicate Guid in unit page" when two CustomWidget instances share the same
// $ID values in their embedded CustomWidgetType documents.
//
// Root cause: using identity mapping for Type $IDs (stableIds) means every
// instance of the same widget type carries identical $IDs, causing conflicts.
// Fresh $IDs must be generated for every call to GetTemplateFullBSON.

package widgets

import (
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// minimalStableTemplate returns a minimal WidgetTemplate with StableIds=true
// (simulating a template extracted from Studio Pro). The hex IDs are fixed
// so we can verify they get remapped to different values on each call.
func minimalStableTemplate() *WidgetTemplate {
	return &WidgetTemplate{
		WidgetID:  "test.widget.Minimal",
		StableIds: true,
		Type: map[string]any{
			"$ID":   "aabbccdd00112233445566778899aabb",
			"$Type": "CustomWidgets$CustomWidgetType",
			"Name":  "test.widget.Minimal",
			"Properties": []any{
				map[string]any{
					"$ID":         "11223344556677889900aabbccddeeff",
					"$Type":       "CustomWidgets$WidgetPropertyType",
					"PropertyKey": "caption",
					"ValueType": map[string]any{
						"$ID":   "ffeeddccbbaa99887766554433221100",
						"$Type": "CustomWidgets$WidgetStringType",
					},
				},
			},
		},
		Object: map[string]any{
			"$ID":         "00112233445566778899aabbccddeeff",
			"$Type":       "CustomWidgets$WidgetObject",
			"TypePointer": "aabbccdd00112233445566778899aabb",
		},
	}
}

// TestGetTemplateFullBSON_NoDuplicateGuid verifies that two successive calls
// to GetTemplateFullBSON for the same widget ID produce DIFFERENT Type $IDs.
// If both calls return the same Type (with shared $IDs), Mendix would raise
// InvalidOperationException: Duplicate Guid when both instances appear on the
// same page.
func TestGetTemplateFullBSON_NoDuplicateGuid(t *testing.T) {
	// Register a synthetic stable-ID template in the session cache so this
	// test runs without a real project on disk.
	tmpl := minimalStableTemplate()
	const widgetID = "test.widget.Minimal"
	generatedCache.Store(widgetID, tmpl)
	defer generatedCache.Delete(widgetID)

	counter := 0
	idGen := func() string {
		counter++
		return idForCounter(counter)
	}

	bsonType1, _, _, _, _, err1 := GetTemplateFullBSON(widgetID, idGen, "")
	if err1 != nil {
		t.Fatalf("first call failed: %v", err1)
	}
	bsonType2, _, _, _, _, err2 := GetTemplateFullBSON(widgetID, idGen, "")
	if err2 != nil {
		t.Fatalf("second call failed: %v", err2)
	}

	// Extract the $ID from the root CustomWidgetType element.
	rootID1 := bsonDocID(bsonType1)
	rootID2 := bsonDocID(bsonType2)

	if rootID1 == "" {
		t.Fatal("first call returned a type with no $ID")
	}
	// The root CustomWidgetType $IDs must be different — same ID would cause Duplicate Guid.
	if rootID1 == rootID2 {
		t.Fatalf("Duplicate Guid detected: both widget instances have CustomWidgetType.$ID = %q\n"+
			"This would cause InvalidOperationException in Mendix Studio Pro when both\n"+
			"widgets are on the same page. Do NOT use identity mapping for Type $IDs.", rootID1)
	}
}

// bsonDocID returns the $ID of a BSON document as a hex string, or "".
// Handles both []byte (raw from hexToIDBlob) and bson.Binary forms.
func bsonDocID(doc bson.D) string {
	for _, elem := range doc {
		if elem.Key == "$ID" {
			switch v := elem.Value.(type) {
			case bson.Binary:
				return fmt.Sprintf("%x", v.Data)
			case []byte:
				return fmt.Sprintf("%x", v)
			}
		}
	}
	return ""
}

func idForCounter(n int) string {
	return formatHex32(n)
}

// formatHex32 produces a deterministic 32-char hex string from n.
func formatHex32(n int) string {
	const hex = "0123456789abcdef"
	out := make([]byte, 32)
	for i := 31; i >= 0; i-- {
		out[i] = hex[n&0xf]
		n >>= 4
	}
	return string(out)
}

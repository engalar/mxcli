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
	"testing"
)

// TestGetTemplateFullBSON_NoDuplicateGuid verifies that two successive calls
// to GetTemplateFullBSON for the same widget ID produce DIFFERENT Type $IDs.
// If both calls return the same Type (with shared $IDs), Mendix would raise
// InvalidOperationException: Duplicate Guid when both instances appear on the
// same page.
func TestGetTemplateFullBSON_NoDuplicateGuid(t *testing.T) {
	widgetID := "com.mendix.widget.web.combobox.Combobox"
	counter := 0
	idGen := func() string {
		counter++
		return idForCounter(counter)
	}

	bsonType1, _, _, objectTypeID1, _, err1 := GetTemplateFullBSON(widgetID, idGen, "")
	if err1 != nil {
		t.Skipf("template not found for %s: %v", widgetID, err1)
	}
	if bsonType1 == nil {
		t.Skipf("no template for %s", widgetID)
	}

	_, _, _, objectTypeID2, _, err2 := GetTemplateFullBSON(widgetID, idGen, "")
	if err2 != nil {
		t.Fatalf("second call failed: %v", err2)
	}

	// The ObjectType $IDs must be different — same ID would cause Duplicate Guid.
	if objectTypeID1 == objectTypeID2 {
		t.Fatalf("Duplicate Guid detected: both widget instances have ObjectType.$ID = %q\n"+
			"This would cause InvalidOperationException in Mendix Studio Pro when both\n"+
			"widgets are on the same page. Do NOT use identity mapping for Type $IDs.", objectTypeID1)
	}
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

package rules

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/widgets"
)

// TestMergeValue_TextTemplateOverridden verifies that TextTemplate state
// always comes from the canonical value, not the current — this is the
// CE0463-critical invariant.
func TestMergeValue_TextTemplateOverridden(t *testing.T) {
	canonVal := map[string]any{
		"$ID":         "aa000000000000000000000000000001",
		"TextTemplate": nil,
		"PrimitiveValue": "true",
	}
	curVal := map[string]any{
		"$ID":         []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		"TextTemplate": map[string]any{"$ID": "stale-tt-id", "$Type": "Forms$ClientTemplate"},
		"PrimitiveValue": "true",
	}
	got := mergeValue(canonVal, curVal)

	// TextTemplate must be nil (from canonical), not the stale ClientTemplate
	if got["TextTemplate"] != nil {
		t.Errorf("mergeValue: TextTemplate = %v, want nil", got["TextTemplate"])
	}
	// PrimitiveValue should be preserved from current
	if s := extractStr(got["PrimitiveValue"]); s != "true" {
		t.Errorf("mergeValue: PrimitiveValue = %q, want %q", s, "true")
	}
}

// TestMergeValue_DataSourcePreserved verifies that a user-configured
// DataSource in the current value is preserved after merge.
func TestMergeValue_DataSourcePreserved(t *testing.T) {
	canonVal := map[string]any{
		"$ID":         "aa000000000000000000000000000002",
		"DataSource":  nil,
		"TextTemplate": nil,
	}
	curVal := map[string]any{
		"$ID":        []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		"DataSource": map[string]any{"$Type": "SomeDataSource", "$ID": "ds-id"},
	}
	got := mergeValue(canonVal, curVal)

	if got["DataSource"] == nil {
		t.Error("mergeValue: DataSource = nil, want non-nil (preserved from current)")
	}
}

// TestMergeValue_ExpressionPreserved verifies that a user expression is
// preserved while TextTemplate comes from canonical.
func TestMergeValue_ExpressionPreserved(t *testing.T) {
	canonVal := map[string]any{
		"$ID":         "aa000000000000000000000000000003",
		"Expression":   "",
		"TextTemplate": nil,
	}
	curVal := map[string]any{
		"$ID":        []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3},
		"Expression": "$currentObject/SomeAttribute",
		"TextTemplate": map[string]any{"$Type": "Forms$ClientTemplate"},
	}
	got := mergeValue(canonVal, curVal)

	if s := extractStr(got["Expression"]); s != "$currentObject/SomeAttribute" {
		t.Errorf("mergeValue: Expression = %q, want %q", s, "$currentObject/SomeAttribute")
	}
	if got["TextTemplate"] != nil {
		t.Error("mergeValue: TextTemplate should be nil (from canonical)")
	}
}

// TestBuildCurrentKeyToValueMap verifies the mapping from PropertyKey → Value
// using sample current Type and Object data.
func TestBuildCurrentKeyToValueMap(t *testing.T) {
	currentType := map[string]any{
		"ObjectType": map[string]any{
			"PropertyTypes": []any{
				int32(2),
				map[string]any{
					"$ID":         []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x10},
					"PropertyKey": "datasource",
				},
				map[string]any{
					"$ID":         []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x20},
					"PropertyKey": "label",
				},
			},
		},
	}
	currentObj := map[string]any{
		"Properties": []any{
			int32(2),
			map[string]any{
				"TypePointer": []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x10},
				"Value":       map[string]any{"Expression": "$currentObject"},
			},
			map[string]any{
				"TypePointer": []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x20},
				"Value":       map[string]any{"TextTemplate": map[string]any{"$Type": "Forms$ClientTemplate"}},
			},
		},
	}

	m := buildCurrentKeyToValueMap(currentObj, currentType)
	if m == nil {
		t.Fatal("buildCurrentKeyToValueMap returned nil")
	}

	dsVal, ok := m["datasource"]
	if !ok {
		t.Fatal("buildCurrentKeyToValueMap: 'datasource' key missing")
	}
	if s := extractStr(dsVal["Expression"]); s != "$currentObject" {
		t.Errorf("datasource.Expression = %q, want %q", s, "$currentObject")
	}

	labelVal, ok := m["label"]
	if !ok {
		t.Fatal("buildCurrentKeyToValueMap: 'label' key missing")
	}
	if labelVal["TextTemplate"] == nil {
		t.Error("label.TextTemplate should be non-nil (ClientTemplate)")
	}
}

// TestBuildCanonicalKeyToTypePointer verifies PropertyKey extraction
// from a canonical Type JSON (hex string IDs).
func TestBuildCanonicalKeyToTypePointer(t *testing.T) {
	canonType := map[string]any{
		"ObjectType": map[string]any{
			"PropertyTypes": []any{
				int32(2),
				map[string]any{
					"$ID":         "aa000000000000000000000000000010",
					"PropertyKey": "datasource",
				},
				map[string]any{
					"$ID":         "aa000000000000000000000000000020",
					"PropertyKey": "label",
				},
			},
		},
	}

	m := buildCanonicalKeyToTypePointer(canonType)
	if len(m) != 2 {
		t.Fatalf("buildCanonicalKeyToTypePointer returned %d entries, want 2", len(m))
	}
	if m["datasource"] != "aa000000000000000000000000000010" {
		t.Errorf("datasource TP = %q", m["datasource"])
	}
	if m["label"] != "aa000000000000000000000000000020" {
		t.Errorf("label TP = %q", m["label"])
	}
}

// TestRebuildWidgetObject_MissingProperty verifies that a canonical
// property missing from the current Object is included with canonical defaults.
func TestRebuildWidgetObject_MissingProperty(t *testing.T) {
	currentType := map[string]any{
		"ObjectType": map[string]any{
			"PropertyTypes": []any{
				int32(2),
				map[string]any{
					"$ID":         []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x10},
					"PropertyKey": "datasource",
				},
			},
		},
	}
	currentObj := map[string]any{
		"Properties": []any{
			int32(2),
			map[string]any{
				"TypePointer": []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x10},
				"Value":       map[string]any{"Expression": "$currentObject"},
			},
		},
	}

	// Canonical template has TWO properties: datasource and label.
	// Current only has datasource — label should be added by RebuildWidgetObject.
	canonTemplate := &widgets.WidgetTemplate{
		WidgetID: "test.widget",
		Type: map[string]any{
			"ObjectType": map[string]any{
				"PropertyTypes": []any{
					int32(2),
					map[string]any{
						"$ID":         "aa100000000000000000000000000010",
						"PropertyKey": "datasource",
						"ValueType":   map[string]any{"$ID": "aa100000000000000000000000000011", "Type": "Expression"},
					},
					map[string]any{
						"$ID":         "aa100000000000000000000000000020",
						"PropertyKey": "label",
						"ValueType":   map[string]any{"$ID": "aa100000000000000000000000000021", "Type": "TextTemplate"},
					},
				},
			},
		},
		Object: map[string]any{
			"$ID":   "aa200000000000000000000000000001",
			"$Type": "CustomWidgets$WidgetObject",
			"Properties": []any{
				int32(2),
				map[string]any{
					"$ID":         "aa200000000000000000000000000010",
					"$Type":       "CustomWidgets$WidgetProperty",
					"TypePointer": "aa100000000000000000000000000010",
					"Value": map[string]any{
						"$ID":         "aa200000000000000000000000000011",
						"$Type":       "CustomWidgets$WidgetValue",
						"Expression":  "",
						"TypePointer": "aa100000000000000000000000000011",
					},
				},
				map[string]any{
					"$ID":         "aa200000000000000000000000000020",
					"$Type":       "CustomWidgets$WidgetProperty",
					"TypePointer": "aa100000000000000000000000000020",
					"Value": map[string]any{
						"$ID":         "aa200000000000000000000000000021",
						"$Type":       "CustomWidgets$WidgetValue",
						"TextTemplate": nil,
						"TypePointer": "aa100000000000000000000000000021",
					},
				},
			},
		},
	}

	mergedType, mergedObj := RebuildWidgetObject(currentObj, currentType, canonTemplate)
	if mergedObj == nil {
		t.Fatal("RebuildWidgetObject returned nil Object")
	}

	props := getBsonArray(mergedObj["Properties"])
	if len(props) != 2 {
		t.Fatalf("merged Object has %d properties, want 2", len(props))
	}

	// Check both type and object are preserved
	if mergedType == nil {
		t.Fatal("RebuildWidgetObject returned nil Type")
	}
}

// TestRebuildWidgetObject_ExtraPropertyRemoved verifies that a property
// present in current but not in the canonical Type is excluded.
func TestRebuildWidgetObject_ExtraPropertyRemoved(t *testing.T) {
	currentType := map[string]any{
		"ObjectType": map[string]any{
			"PropertyTypes": []any{
				int32(2),
				map[string]any{
					"$ID":         []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x10},
					"PropertyKey": "datasource",
				},
				map[string]any{
					"$ID":         []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x20},
					"PropertyKey": "obsoleteProp", // Not in canonical
				},
			},
		},
	}
	currentObj := map[string]any{
		"Properties": []any{
			int32(2),
			map[string]any{
				"TypePointer": []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x10},
				"Value":       map[string]any{"Expression": "$currentObject"},
			},
			map[string]any{
				"TypePointer": []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x20},
				"Value":       map[string]any{"PrimitiveValue": "stale"},
			},
		},
	}

	// Canonical only has datasource (no obsoleteProp)
	canonTemplate := &widgets.WidgetTemplate{
		WidgetID: "test.widget",
		Type: map[string]any{
			"ObjectType": map[string]any{
				"PropertyTypes": []any{
					int32(2),
					map[string]any{
						"$ID":         "aa100000000000000000000000000010",
						"PropertyKey": "datasource",
						"ValueType":   map[string]any{"$ID": "aa100000000000000000000000000011", "Type": "Expression"},
					},
				},
			},
		},
		Object: map[string]any{
			"$ID":   "aa200000000000000000000000000001",
			"$Type": "CustomWidgets$WidgetObject",
			"Properties": []any{
				int32(2),
				map[string]any{
					"$ID":         "aa200000000000000000000000000010",
					"$Type":       "CustomWidgets$WidgetProperty",
					"TypePointer": "aa100000000000000000000000000010",
					"Value": map[string]any{
						"$ID":         "aa200000000000000000000000000011",
						"Expression":  "",
						"TypePointer": "aa100000000000000000000000000011",
					},
				},
			},
		},
	}

	_, mergedObj := RebuildWidgetObject(currentObj, currentType, canonTemplate)
	props := getBsonArray(mergedObj["Properties"])
	if len(props) != 1 {
		t.Fatalf("merged Object has %d properties, want 1 (obsoleteProp should be removed)", len(props))
	}
}

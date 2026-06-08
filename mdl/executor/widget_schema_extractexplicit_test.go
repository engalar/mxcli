// mdl/executor/widget_schema_extractexplicit_test.go
package executor

import (
	"testing"
)

// TestExtractExplicitProperties_UsesSchemaDefault verifies that a non-boolean
// property matching its schema default is suppressed.
func TestExtractExplicitProperties_UsesSchemaDefault(t *testing.T) {
	raw := makeTestRaw(
		[]map[string]any{
			{"$ID": "ptr1", "PropertyKey": "size", "ValueType": map[string]any{"Type": "Integer", "DefaultValue": "10"}},
		},
		[]map[string]any{
			{"TypePointer": "ptr1", "Value": map[string]any{"PrimitiveValue": "10"}}, // equals default
		},
	)
	// extractExplicitProperties should return nothing (default value)
	result := extractExplicitPropertiesFromRaw(raw)
	if len(result) != 0 {
		t.Errorf("expected 0 explicit props, got %d: %v", len(result), result)
	}
}

// TestExtractExplicitProperties_OutputsNonDefault verifies a value differing from
// the schema default is emitted.
func TestExtractExplicitProperties_OutputsNonDefault(t *testing.T) {
	raw := makeTestRaw(
		[]map[string]any{
			{"$ID": "ptr1", "PropertyKey": "size", "ValueType": map[string]any{"Type": "Integer", "DefaultValue": "10"}},
		},
		[]map[string]any{
			{"TypePointer": "ptr1", "Value": map[string]any{"PrimitiveValue": "25"}},
		},
	)
	result := extractExplicitPropertiesFromRaw(raw)
	if len(result) != 1 {
		t.Fatalf("expected 1 explicit prop, got %d", len(result))
	}
	if result[0].Key != "size" || result[0].Value != "25" {
		t.Errorf("got %+v, want {size 25}", result[0])
	}
}

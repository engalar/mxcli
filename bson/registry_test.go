//go:build debug

package bson

import (
	"testing"
)

func TestTypeRegistryHasEntries(t *testing.T) {
	if len(TypeRegistry) == 0 {
		t.Fatal("TypeRegistry is empty, expected Workflows$ entries")
	}
	t.Logf("TypeRegistry contains %d entries", len(TypeRegistry))
}

func TestGetFieldMetaWorkflowsWorkflow(t *testing.T) {
	fields := GetFieldMeta("Workflows$Workflow")
	if fields == nil {
		t.Fatal("GetFieldMeta returned nil for Workflows$Workflow")
	}
	if len(fields) == 0 {
		t.Fatal("GetFieldMeta returned empty fields for Workflows$Workflow")
	}

	fieldByStorage := make(map[string]PropertyMeta)
	for _, f := range fields {
		fieldByStorage[f.StorageName] = f
	}

	// Verify expected semantic fields exist (json tags use lowercase)
	expectedFields := []string{"name", "flow", "parameter"}
	for _, name := range expectedFields {
		if _, ok := fieldByStorage[name]; !ok {
			t.Errorf("expected field %q not found in Workflows$Workflow metadata", name)
		}
	}

	// Verify name field properties
	nameField, ok := fieldByStorage["name"]
	if ok {
		if nameField.GoType != "string" {
			t.Errorf("name field GoType = %q, want %q", nameField.GoType, "string")
		}
		if nameField.Category != Semantic {
			t.Errorf("name field Category = %v, want Semantic", nameField.Category)
		}
		if nameField.GoFieldName != "Name" {
			t.Errorf("name field GoFieldName = %q, want %q", nameField.GoFieldName, "Name")
		}
	}
}

func TestGetFieldMetaUnknownType(t *testing.T) {
	fields := GetFieldMeta("Nonexistent$Type")
	if fields != nil {
		t.Errorf("expected nil for unknown type, got %d fields", len(fields))
	}
}

func TestClassifyField(t *testing.T) {
	tests := []struct {
		storageName string
		want        FieldCategory
	}{
		{"$ID", Structural},
		{"$Type", Structural},
		{"PersistentId", Structural},
		{"RelativeMiddlePoint", Layout},
		{"Size", Layout},
		{"Name", Semantic},
		{"Flow", Semantic},
		{"Expression", Semantic},
	}

	for _, tc := range tests {
		got := classifyField(tc.storageName)
		if got != tc.want {
			t.Errorf("classifyField(%q) = %v, want %v", tc.storageName, got, tc.want)
		}
	}
}

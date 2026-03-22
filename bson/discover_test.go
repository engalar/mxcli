package bson

import (
	"strings"
	"testing"
)

func TestCheckFieldCoverage_StringValues(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		mdlText  string
		expected CoverageStatus
	}{
		{
			name:     "empty string is default",
			value:    "",
			mdlText:  "anything",
			expected: DefaultValue,
		},
		{
			name:     "string found in MDL",
			value:    "MyWorkflow",
			mdlText:  "CREATE WORKFLOW MyWorkflow BEGIN END",
			expected: Covered,
		},
		{
			name:     "string not found in MDL",
			value:    "HiddenTitle",
			mdlText:  "CREATE WORKFLOW MyWorkflow BEGIN END",
			expected: Uncovered,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkFieldCoverage("TestField", tt.value, tt.mdlText)
			if got != tt.expected {
				t.Errorf("checkFieldCoverage(%q, %v, ...) = %v, want %v",
					"TestField", tt.value, got, tt.expected)
			}
		})
	}
}

func TestCheckFieldCoverage_BoolValues(t *testing.T) {
	tests := []struct {
		name     string
		value    bool
		mdlText  string
		expected CoverageStatus
	}{
		{
			name:     "false is default",
			value:    false,
			mdlText:  "anything",
			expected: DefaultValue,
		},
		{
			name:     "true found in MDL",
			value:    true,
			mdlText:  "SET flag = true",
			expected: Covered,
		},
		{
			name:     "true not found in MDL",
			value:    true,
			mdlText:  "CREATE WORKFLOW BEGIN END",
			expected: Uncovered,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkFieldCoverage("TestField", tt.value, tt.mdlText)
			if got != tt.expected {
				t.Errorf("checkFieldCoverage(%q, %v, ...) = %v, want %v",
					"TestField", tt.value, got, tt.expected)
			}
		})
	}
}

func TestCheckFieldCoverage_NilValue(t *testing.T) {
	got := checkFieldCoverage("NullField", nil, "anything")
	if got != DefaultValue {
		t.Errorf("checkFieldCoverage with nil = %v, want DefaultValue", got)
	}
}

func TestCheckFieldCoverage_UnknownTypes(t *testing.T) {
	// int32 should return Unknown since we don't handle it in the main switch
	got := checkFieldCoverage("IntField", int32(42), "42")
	if got != Unknown {
		t.Errorf("checkFieldCoverage with int32 = %v, want Unknown", got)
	}
}

func TestCheckFieldCoverage_NestedObject(t *testing.T) {
	tests := []struct {
		name     string
		value    map[string]any
		mdlText  string
		expected CoverageStatus
	}{
		{
			name: "leaf string found in MDL",
			value: map[string]any{
				"$Type": "Workflows$XPathUserTargeting",
				"XPath": "[Module.MyEntity]",
			},
			mdlText:  "TARGETING [Module.MyEntity]",
			expected: Covered,
		},
		{
			name: "no leaf strings found",
			value: map[string]any{
				"$Type": "Workflows$NoEvent",
				"$ID":   "abc-123",
			},
			mdlText:  "CREATE WORKFLOW BEGIN END",
			expected: Uncovered,
		},
		{
			name: "deeply nested string found",
			value: map[string]any{
				"Outer": map[string]any{
					"Inner": "DeepValue",
				},
			},
			mdlText:  "DeepValue is here",
			expected: Covered,
		},
		{
			name: "all empty strings",
			value: map[string]any{
				"Field1": "",
				"Field2": "",
			},
			mdlText:  "anything",
			expected: Uncovered,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkFieldCoverage("NestedField", tt.value, tt.mdlText)
			if got != tt.expected {
				t.Errorf("checkFieldCoverage(%q, nested, ...) = %v, want %v",
					"NestedField", got, tt.expected)
			}
		})
	}
}

func TestRunDiscover_BasicCoverage(t *testing.T) {
	units := []RawUnit{
		{
			QualifiedName: "Module.Wf1",
			BsonType:      "TestType$Foo",
			Fields: map[string]any{
				"Name":    "Wf1",
				"Title":   "My Workflow",
				"Enabled": true,
				"Notes":   "",
				"Extra":   nil,
			},
		},
	}

	mdlText := "CREATE WORKFLOW Wf1 BEGIN END"

	dr := RunDiscover(units, mdlText)
	if dr == nil {
		t.Fatal("RunDiscover returned nil")
	}
	if len(dr.Types) != 1 {
		t.Fatalf("expected 1 type, got %d", len(dr.Types))
	}

	tc := dr.Types[0]
	if tc.BsonType != "TestType$Foo" {
		t.Errorf("BsonType = %q, want TestType$Foo", tc.BsonType)
	}
	if tc.InstanceCount != 1 {
		t.Errorf("InstanceCount = %d, want 1", tc.InstanceCount)
	}

	// Since TestType$Foo is not in TypeRegistry, all fields are reported directly.
	statusByField := make(map[string]CoverageStatus)
	for _, f := range tc.Fields {
		statusByField[f.StorageName] = f.Status
	}

	if statusByField["Name"] != Covered {
		t.Errorf("Name should be Covered, got %v", statusByField["Name"])
	}
	if statusByField["Title"] != Uncovered {
		t.Errorf("Title should be Uncovered, got %v", statusByField["Title"])
	}
	if statusByField["Notes"] != DefaultValue {
		t.Errorf("Notes (empty string) should be DefaultValue, got %v", statusByField["Notes"])
	}
	if statusByField["Extra"] != DefaultValue {
		t.Errorf("Extra (nil) should be DefaultValue, got %v", statusByField["Extra"])
	}
}

func TestRunDiscover_MultipleInstances(t *testing.T) {
	units := []RawUnit{
		{
			QualifiedName: "M.A",
			BsonType:      "TestType$Bar",
			Fields: map[string]any{
				"Name":  "A",
				"Color": nil,
			},
		},
		{
			QualifiedName: "M.B",
			BsonType:      "TestType$Bar",
			Fields: map[string]any{
				"Name":  "B",
				"Color": "red",
			},
		},
	}

	dr := RunDiscover(units, "A B")
	if len(dr.Types) != 1 {
		t.Fatalf("expected 1 type, got %d", len(dr.Types))
	}

	tc := dr.Types[0]
	if tc.InstanceCount != 2 {
		t.Errorf("InstanceCount = %d, want 2", tc.InstanceCount)
	}

	// Color should prefer the non-nil sample ("red").
	for _, f := range tc.Fields {
		if f.StorageName == "Color" {
			if f.SampleValue != `"red"` {
				t.Errorf("Color SampleValue = %q, want %q", f.SampleValue, `"red"`)
			}
		}
	}
}

func TestRunDiscover_MultipleTypes(t *testing.T) {
	units := []RawUnit{
		{BsonType: "TypeA$X", Fields: map[string]any{"A": "val"}},
		{BsonType: "TypeB$Y", Fields: map[string]any{"B": "val"}},
	}

	dr := RunDiscover(units, "val")
	if len(dr.Types) != 2 {
		t.Fatalf("expected 2 types, got %d", len(dr.Types))
	}
	if dr.Types[0].BsonType != "TypeA$X" {
		t.Errorf("first type = %q, want TypeA$X", dr.Types[0].BsonType)
	}
	if dr.Types[1].BsonType != "TypeB$Y" {
		t.Errorf("second type = %q, want TypeB$Y", dr.Types[1].BsonType)
	}
}

func TestRunDiscover_EmptyMDLText(t *testing.T) {
	units := []RawUnit{
		{
			BsonType: "TestType$Empty",
			Fields: map[string]any{
				"Name":  "Something",
				"Empty": "",
			},
		},
	}

	dr := RunDiscover(units, "")
	tc := dr.Types[0]

	statusByField := make(map[string]CoverageStatus)
	for _, f := range tc.Fields {
		statusByField[f.StorageName] = f.Status
	}

	if statusByField["Name"] != Uncovered {
		t.Errorf("Name with empty MDL should be Uncovered, got %v", statusByField["Name"])
	}
	if statusByField["Empty"] != DefaultValue {
		t.Errorf("Empty string should still be DefaultValue, got %v", statusByField["Empty"])
	}
}

func TestFormatResult(t *testing.T) {
	dr := &DiscoverResult{
		Types: []TypeCoverage{
			{
				BsonType:      "Test$Widget",
				InstanceCount: 3,
				Fields: []FieldCoverage{
					{StorageName: "Name", Category: Semantic, Status: Covered},
					{StorageName: "Title", GoType: "string", Category: Semantic, Status: Uncovered, SampleValue: `"hello"`},
					{StorageName: "$ID", Category: Structural, Status: DefaultValue},
					{StorageName: "$Type", Category: Structural, Status: DefaultValue},
				},
			},
		},
	}

	output := FormatResult(dr)

	if !strings.Contains(output, "Test$Widget (3 objects scanned)") {
		t.Errorf("output missing type header: %s", output)
	}
	if !strings.Contains(output, "Coverage: 1/2 semantic fields (50%)") {
		t.Errorf("output missing coverage summary: %s", output)
	}
	if !strings.Contains(output, "UNCOVERED") {
		t.Errorf("output missing UNCOVERED marker: %s", output)
	}
}

func TestSampleValueString(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{"nil", nil, "null"},
		{"empty string", "", `""`},
		{"short string", "hello", `"hello"`},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"int32", int32(42), "42"},
		{"slice", []any{1, 2, 3}, "[3 elements]"},
		{"map with $Type", map[string]any{"$Type": "Foo"}, "{$Type: Foo}"},
		{"map without $Type", map[string]any{"a": 1, "b": 2}, "{2 fields}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sampleValueString(tt.value)
			if got != tt.expected {
				t.Errorf("sampleValueString(%v) = %q, want %q", tt.value, got, tt.expected)
			}
		})
	}
}

func TestCoverageStatusString(t *testing.T) {
	if Covered.String() != "covered" {
		t.Errorf("Covered.String() = %q", Covered.String())
	}
	if Uncovered.String() != "UNCOVERED" {
		t.Errorf("Uncovered.String() = %q", Uncovered.String())
	}
	if DefaultValue.String() != "default" {
		t.Errorf("DefaultValue.String() = %q", DefaultValue.String())
	}
	if Unknown.String() != "unknown" {
		t.Errorf("Unknown.String() = %q", Unknown.String())
	}
}

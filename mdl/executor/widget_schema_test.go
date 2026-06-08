// mdl/executor/widget_schema_test.go
package executor

import (
	"testing"
)

func makeTestRaw(propTypes []map[string]any, props []map[string]any) map[string]any {
	return map[string]any{
		"$Type": "CustomWidgets$CustomWidget",
		"Name":  "testWidget",
		"Type": map[string]any{
			"ObjectType": map[string]any{
				"PropertyTypes": propTypes,
			},
		},
		"Object": map[string]any{
			"Properties": props,
		},
	}
}

func TestBuildSchemaMap_ExtractsKeyAndDefault(t *testing.T) {
	raw := makeTestRaw(
		[]map[string]any{
			{"$ID": "ptr1", "PropertyKey": "source", "ValueType": map[string]any{"Type": "Enumeration", "DefaultValue": "context"}},
		},
		nil,
	)
	schema := buildSchemaMap(raw)
	entry, ok := schema["ptr1"]
	if !ok {
		t.Fatal("ptr1 not found in schema")
	}
	if entry.Key != "source" {
		t.Errorf("Key = %q, want %q", entry.Key, "source")
	}
	if entry.DefaultValue != "context" {
		t.Errorf("DefaultValue = %q, want %q", entry.DefaultValue, "context")
	}
	if entry.ValueType != "Enumeration" {
		t.Errorf("ValueType = %q, want %q", entry.ValueType, "Enumeration")
	}
}

func TestFilterDefaults_SkipsMatchingPrimitive(t *testing.T) {
	schema := SchemaMap{"ptr1": SchemaEntry{Key: "source", DefaultValue: "context", ValueType: "Enumeration"}}
	props := []PropertyValue{{TypePointerID: "ptr1", PrimitiveValue: "context"}}
	result := filterDefaults(props, schema)
	if len(result) != 0 {
		t.Errorf("expected 0 non-default props, got %d", len(result))
	}
}

func TestFilterDefaults_OutputsNonDefaultPrimitive(t *testing.T) {
	schema := SchemaMap{"ptr1": SchemaEntry{Key: "source", DefaultValue: "context", ValueType: "Enumeration"}}
	props := []PropertyValue{{TypePointerID: "ptr1", PrimitiveValue: "datasource"}}
	result := filterDefaults(props, schema)
	if len(result) != 1 {
		t.Fatalf("expected 1 non-default prop, got %d", len(result))
	}
	if result[0].Key != "source" {
		t.Errorf("Key = %q, want %q", result[0].Key, "source")
	}
	if result[0].PrimitiveValue != "datasource" {
		t.Errorf("Value = %q", result[0].PrimitiveValue)
	}
}

func TestFilterDefaults_SkipsBoolDefaultFalse(t *testing.T) {
	schema := SchemaMap{"ptr1": SchemaEntry{Key: "enabled", DefaultValue: "false", ValueType: "Boolean"}}
	props := []PropertyValue{{TypePointerID: "ptr1", PrimitiveValue: "false"}}
	if len(filterDefaults(props, schema)) != 0 {
		t.Error("false should be skipped when default is false")
	}
}

func TestFilterDefaults_OutputsNonDefaultBool(t *testing.T) {
	schema := SchemaMap{"ptr1": SchemaEntry{Key: "enabled", DefaultValue: "false", ValueType: "Boolean"}}
	props := []PropertyValue{{TypePointerID: "ptr1", PrimitiveValue: "true"}}
	result := filterDefaults(props, schema)
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
}

func TestFilterDefaults_SkipsEmptyStringDefault(t *testing.T) {
	schema := SchemaMap{"ptr1": SchemaEntry{Key: "label", DefaultValue: "", ValueType: "String"}}
	props := []PropertyValue{{TypePointerID: "ptr1", PrimitiveValue: ""}}
	if len(filterDefaults(props, schema)) != 0 {
		t.Error("empty string matching default should be skipped")
	}
}

func TestExtractWidgetTypeID_ReadsFromTypeField(t *testing.T) {
	raw := map[string]any{
		"Type": map[string]any{"WidgetId": "com.mendix.widget.web.datagrid2.DataGrid"},
	}
	got := extractWidgetTypeID(raw)
	want := "com.mendix.widget.web.datagrid2.DataGrid"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestReadProperties_ExtractsPrimitiveValues(t *testing.T) {
	raw := makeTestRaw(nil, []map[string]any{
		{"TypePointer": "ptr1", "Value": map[string]any{"PrimitiveValue": "hello"}},
	})
	props := readProperties(raw)
	if len(props) != 1 {
		t.Fatalf("expected 1, got %d", len(props))
	}
	if props[0].TypePointerID != "ptr1" {
		t.Errorf("TypePointerID = %q", props[0].TypePointerID)
	}
	if props[0].PrimitiveValue != "hello" {
		t.Errorf("PrimitiveValue = %q", props[0].PrimitiveValue)
	}
}

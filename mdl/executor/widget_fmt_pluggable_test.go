// mdl/executor/widget_fmt_pluggable_test.go
package executor

import (
	"bytes"
	"strings"
	"testing"
)

func TestGenericPluggableFormatter_NoNonDefaultProps(t *testing.T) {
	raw := map[string]any{
		"$Type": "CustomWidgets$CustomWidget",
		"Name":  "myWidget",
		"Type": map[string]any{
			"WidgetId": "com.co.widget.Foo",
			"ObjectType": map[string]any{
				"PropertyTypes": []map[string]any{
					{"$ID": "ptr1", "PropertyKey": "enabled", "ValueType": map[string]any{"Type": "Boolean", "DefaultValue": "false"}},
				},
			},
		},
		"Object": map[string]any{
			"Properties": []map[string]any{
				{"TypePointer": "ptr1", "Value": map[string]any{"PrimitiveValue": "false"}},
			},
		},
	}
	var buf bytes.Buffer
	d := newDefaultDispatcher()
	ctx := &FormatContext{Output: &buf, Indent: 1, Dispatcher: d}
	GenericPluggableFactory(raw).FormatMDL(ctx)

	got := buf.String()
	if !strings.Contains(got, "pluggablewidget 'com.co.widget.Foo' myWidget") {
		t.Errorf("expected pluggablewidget line, got: %q", got)
	}
	if strings.Contains(got, "enabled") {
		t.Error("default-value property should not appear in output")
	}
}

func TestGenericPluggableFormatter_OutputsNonDefaultProp(t *testing.T) {
	raw := map[string]any{
		"$Type": "CustomWidgets$CustomWidget",
		"Name":  "myWidget",
		"Type": map[string]any{
			"WidgetId": "com.co.widget.Foo",
			"ObjectType": map[string]any{
				"PropertyTypes": []map[string]any{
					{"$ID": "ptr1", "PropertyKey": "source", "ValueType": map[string]any{"Type": "Enumeration", "DefaultValue": "context"}},
				},
			},
		},
		"Object": map[string]any{
			"Properties": []map[string]any{
				{"TypePointer": "ptr1", "Value": map[string]any{"PrimitiveValue": "xpath"}},
			},
		},
	}
	var buf bytes.Buffer
	d := newDefaultDispatcher()
	ctx := &FormatContext{Output: &buf, Indent: 0, Dispatcher: d}
	GenericPluggableFactory(raw).FormatMDL(ctx)

	got := buf.String()
	if !strings.Contains(got, "source: 'xpath'") {
		t.Errorf("expected non-default prop in output, got: %q", got)
	}
}

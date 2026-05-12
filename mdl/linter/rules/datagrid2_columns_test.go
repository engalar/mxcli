// SPDX-License-Identifier: Apache-2.0

package rules

import "testing"

// makeDG2Widget builds a minimal CustomWidgets$CustomWidget BSON map for a
// DataGrid2, given a list of pre-built column property maps.
func makeDG2Widget(name string, columnProps []any) map[string]any {
	var colObjects any
	if columnProps == nil {
		// Empty columns list (stub datagrid)
		colObjects = []any{}
	} else {
		cols := make([]any, len(columnProps))
		for i, cp := range columnProps {
			cols[i] = map[string]any{
				"$Type":      "CustomWidgets$WidgetObject",
				"Properties": cp,
			}
		}
		colObjects = cols
	}

	return map[string]any{
		"$Type": "CustomWidgets$CustomWidget",
		"Name":  name,
		"Object": map[string]any{
			"$Type": "CustomWidgets$WidgetObject",
			"Properties": []any{
				map[string]any{
					"$Type": "CustomWidgets$WidgetProperty",
					"Key":   "columns",
					"Value": map[string]any{
						"$Type":   "CustomWidgets$WidgetListValue",
						"Objects": colObjects,
					},
				},
			},
		},
	}
}

// makeColumn builds a column property list. attributeRef may be nil (unbound).
func makeColumn(caption string, sortable bool, attributeRef map[string]any) []any {
	captionProp := map[string]any{
		"$Type": "CustomWidgets$WidgetProperty",
		"Key":   "caption",
		"Value": map[string]any{
			"$Type": "CustomWidgets$WidgetTranslatableStringValue",
			"Value": caption,
		},
	}
	sortableProp := map[string]any{
		"$Type": "CustomWidgets$WidgetProperty",
		"Key":   "sortable",
		"Value": map[string]any{
			"$Type": "CustomWidgets$WidgetBooleanValue",
			"Value": sortable,
		},
	}
	attrValue := map[string]any{
		"$Type":        "CustomWidgets$WidgetAttributeValue",
		"AttributeRef": attributeRef,
	}
	attrProp := map[string]any{
		"$Type": "CustomWidgets$WidgetProperty",
		"Key":   "attribute",
		"Value": attrValue,
	}
	return []any{captionProp, sortableProp, attrProp}
}

// makeActionColumn builds a customContent column (action buttons, no attribute needed).
func makeActionColumn(caption string) []any {
	captionProp := map[string]any{
		"$Type": "CustomWidgets$WidgetProperty",
		"Key":   "caption",
		"Value": map[string]any{
			"$Type": "CustomWidgets$WidgetTranslatableStringValue",
			"Value": caption,
		},
	}
	showAsProp := map[string]any{
		"$Type": "CustomWidgets$WidgetProperty",
		"Key":   "showContentAs",
		"Value": map[string]any{
			"$Type": "CustomWidgets$WidgetEnumerationValue",
			"Value": "customContent",
		},
	}
	return []any{captionProp, showAsProp}
}

func wrapInPage(widgets ...map[string]any) map[string]any {
	ws := make([]any, len(widgets))
	for i, w := range widgets {
		ws[i] = w
	}
	return map[string]any{
		"FormCall": map[string]any{
			"Arguments": []any{
				map[string]any{"Widgets": ws},
			},
		},
	}
}

func wrapInSnippet(widgets ...map[string]any) map[string]any {
	ws := make([]any, len(widgets))
	for i, w := range widgets {
		ws[i] = w
	}
	return map[string]any{"Widgets": ws}
}

// TestDataGrid2_NoColumns: empty datagrid stub reports NoColumns.
func TestDataGrid2_NoColumns(t *testing.T) {
	dg := makeDG2Widget("dgContractors", nil)
	rawData := wrapInPage(dg)

	issues := findDataGrid2Issues(rawData)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if !issues[0].NoColumns {
		t.Errorf("expected NoColumns=true")
	}
	if issues[0].WidgetName != "dgContractors" {
		t.Errorf("expected WidgetName 'dgContractors', got %q", issues[0].WidgetName)
	}
}

// TestDataGrid2_SortableWithoutAttribute: column with sortable=true and no attribute.
func TestDataGrid2_SortableWithoutAttribute(t *testing.T) {
	brokenCol := makeColumn("Title", true, nil) // sortable=true, attribute unbound
	dg := makeDG2Widget("dgContractors", []any{brokenCol})
	rawData := wrapInPage(dg)

	issues := findDataGrid2Issues(rawData)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].NoColumns {
		t.Errorf("expected NoColumns=false")
	}
	if issues[0].ColumnCaption != "Title" {
		t.Errorf("expected ColumnCaption 'Title', got %q", issues[0].ColumnCaption)
	}
}

// TestDataGrid2_SortableWithAttribute: correct — sortable=true with attribute bound.
func TestDataGrid2_SortableWithAttribute(t *testing.T) {
	goodCol := makeColumn("Name", true, map[string]any{"$ID": "attr-uuid-1"})
	dg := makeDG2Widget("dgCustomers", []any{goodCol})
	rawData := wrapInPage(dg)

	issues := findDataGrid2Issues(rawData)
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d: %+v", len(issues), issues)
	}
}

// TestDataGrid2_SortableDisabledNoAttribute: sortable=false without attribute is OK.
func TestDataGrid2_SortableDisabledNoAttribute(t *testing.T) {
	col := makeColumn("Actions", false, nil) // sortable=false, no attribute
	dg := makeDG2Widget("dgList", []any{col})
	rawData := wrapInPage(dg)

	issues := findDataGrid2Issues(rawData)
	if len(issues) != 0 {
		t.Errorf("expected 0 issues (sortable=false), got %d", len(issues))
	}
}

// TestDataGrid2_CustomContentColumn: action-only column has no attribute, never an error.
func TestDataGrid2_CustomContentColumn(t *testing.T) {
	actionCol := makeActionColumn(" ")
	dg := makeDG2Widget("dgList", []any{actionCol})
	rawData := wrapInPage(dg)

	issues := findDataGrid2Issues(rawData)
	if len(issues) != 0 {
		t.Errorf("expected 0 issues (customContent column), got %d", len(issues))
	}
}

// TestDataGrid2_MixedColumns: one good + one broken column.
func TestDataGrid2_MixedColumns(t *testing.T) {
	goodCol := makeColumn("Name", true, map[string]any{"$ID": "attr-uuid-1"})
	badCol := makeColumn("Title", true, nil)
	dg := makeDG2Widget("dgContractors", []any{goodCol, badCol})
	rawData := wrapInPage(dg)

	issues := findDataGrid2Issues(rawData)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].ColumnCaption != "Title" {
		t.Errorf("expected 'Title', got %q", issues[0].ColumnCaption)
	}
}

// TestDataGrid2_SnippetStructure: also detected in snippets.
func TestDataGrid2_SnippetStructure(t *testing.T) {
	badCol := makeColumn("Organization", true, nil)
	dg := makeDG2Widget("dgTargetOrg", []any{badCol})
	rawData := wrapInSnippet(dg)

	issues := findDataGrid2Issues(rawData)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue in snippet, got %d", len(issues))
	}
	if issues[0].WidgetName != "dgTargetOrg" {
		t.Errorf("got WidgetName %q", issues[0].WidgetName)
	}
}

// SPDX-License-Identifier: Apache-2.0

// Tests for data container context hints (upstream issue #123).
package executor

import (
	"bytes"
	"strings"
	"testing"
)

func TestOutputDataContainerContext_DataView(t *testing.T) {
	var buf bytes.Buffer
	outputDataContainerContext(&buf, "  ", "dvOrder", "OrderManagement.PurchaseOrder", false)
	got := buf.String()
	expected := "  -- Context: $currentObject (OrderManagement.PurchaseOrder)\n"
	if got != expected {
		t.Errorf("DataView context:\ngot:  %q\nwant: %q", got, expected)
	}
}

func TestOutputDataContainerContext_ListContainer(t *testing.T) {
	var buf bytes.Buffer
	outputDataContainerContext(&buf, "  ", "dgOrders", "OrderManagement.PurchaseOrder", true)
	got := buf.String()
	expected := "  -- Context: $currentObject (OrderManagement.PurchaseOrder), $dgOrders (selection)\n"
	if got != expected {
		t.Errorf("List container context:\ngot:  %q\nwant: %q", got, expected)
	}
}

func TestOutputDataContainerContext_EmptyEntity(t *testing.T) {
	var buf bytes.Buffer
	outputDataContainerContext(&buf, "  ", "dv1", "", false)
	got := buf.String()
	if got != "" {
		t.Errorf("Expected no output for empty entity, got: %q", got)
	}
}

func TestOutputDataContainerContext_ListNoName(t *testing.T) {
	var buf bytes.Buffer
	outputDataContainerContext(&buf, "  ", "", "Module.Entity", true)
	got := buf.String()
	// Should only show $currentObject, no selection variable when widget has no name
	expected := "  -- Context: $currentObject (Module.Entity)\n"
	if got != expected {
		t.Errorf("List container without name:\ngot:  %q\nwant: %q", got, expected)
	}
}

func TestOutputWidgetMDLV3_DataViewWithContext(t *testing.T) {
	buf := &bytes.Buffer{}
	e := New(buf)
	w := rawWidget{
		Type:          "Forms$DataView",
		Name:          "dvOrder",
		EntityContext: "OrderManagement.PurchaseOrder",
		DataSource:    &rawDataSource{Type: "parameter", Reference: "Order"},
		Children: []rawWidget{
			{Type: "Forms$TextBox", Name: "txtName", Content: "Name"},
		},
	}
	e.outputWidgetMDLV3(w, 0)
	got := buf.String()
	if !strings.Contains(got, "-- Context: $currentObject (OrderManagement.PurchaseOrder)") {
		t.Errorf("DataView output should contain context comment, got:\n%s", got)
	}
}

func TestOutputWidgetMDLV3_ListViewWithContext(t *testing.T) {
	buf := &bytes.Buffer{}
	e := New(buf)
	w := rawWidget{
		Type:          "Forms$ListView",
		Name:          "lvItems",
		EntityContext: "Module.Item",
		DataSource:    &rawDataSource{Type: "database", Reference: "Module.Item"},
		Children: []rawWidget{
			{Type: "Forms$TextBox", Name: "txtDesc", Content: "Description"},
		},
	}
	e.outputWidgetMDLV3(w, 0)
	got := buf.String()
	if !strings.Contains(got, "-- Context: $currentObject (Module.Item), $lvItems (selection)") {
		t.Errorf("ListView output should contain context comment with selection, got:\n%s", got)
	}
}

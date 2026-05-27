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

func TestOutputWidgetMDLV3_DataViewInheritsParentContext(t *testing.T) {
	// A DataView without its own DataSource should inherit parent context
	buf := &bytes.Buffer{}
	e := New(buf)
	w := rawWidget{
		Type:          "Forms$DataView",
		Name:          "dvNested",
		EntityContext: "Sales.OrderLine", // inherited from parent during parse
		Children: []rawWidget{
			{Type: "Forms$TextBox", Name: "txtQty", Content: "Quantity"},
		},
	}
	e.outputWidgetMDLV3(w, 0)
	got := buf.String()
	if !strings.Contains(got, "-- Context: $currentObject (Sales.OrderLine)") {
		t.Errorf("Nested DataView should show inherited context, got:\n%s", got)
	}
}

func TestResolveSelectionEntityContexts(t *testing.T) {
	// Gallery artGallery knows its entity (KB.Article).
	// DataView dvDetail listens to artGallery — its EntityContext should be resolved.
	widgets := []rawWidget{
		{
			Type:          "Forms$Gallery",
			Name:          "artGallery",
			EntityContext: "KB.Article",
		},
		{
			Type:          "Forms$DataView",
			Name:          "dvDetail",
			DataSource:    &rawDataSource{Type: "selection", Reference: "artGallery"},
			EntityContext: "artGallery", // set from DataSource.Reference before resolve
		},
	}
	resolveSelectionEntityContexts(widgets)
	if got := widgets[1].EntityContext; got != "KB.Article" {
		t.Errorf("dvDetail EntityContext = %q, want KB.Article", got)
	}
}

func TestExtractDataViewDataSource_ListenTarget(t *testing.T) {
	w := map[string]any{
		"DataSource": map[string]any{
			"$Type":        "Forms$ListenTargetSource",
			"ListenTarget": "artGallery",
		},
	}
	ctx := &ExecContext{}
	ds := extractDataViewDataSource(ctx, w)
	if ds == nil {
		t.Fatal("expected non-nil datasource for Forms$ListenTargetSource")
	}
	if ds.Type != "selection" {
		t.Errorf("ds.Type = %q, want selection", ds.Type)
	}
	if ds.Reference != "artGallery" {
		t.Errorf("ds.Reference = %q, want artGallery", ds.Reference)
	}
}

func TestOutputWidgetMDLV3_DataViewSelectionContext(t *testing.T) {
	buf := &bytes.Buffer{}
	e := New(buf)
	w := rawWidget{
		Type:          "Forms$DataView",
		Name:          "dvDetail",
		EntityContext: "KB.Article",
		DataSource:    &rawDataSource{Type: "selection", Reference: "artGallery"},
		Children: []rawWidget{
			{Type: "Forms$TextBox", Name: "txtTitle", Content: "Title"},
		},
	}
	e.outputWidgetMDLV3(w, 0)
	got := buf.String()
	if !strings.Contains(got, "datasource: selection artGallery") {
		t.Errorf("DataView should show selection datasource, got:\n%s", got)
	}
	// DataView listening to a gallery selection sees $currentObject (the selected
	// item) AND $artGallery (selection) — both are in scope inside the DataView.
	if !strings.Contains(got, "-- Context: $currentObject (KB.Article), $artGallery (selection)") {
		t.Errorf("DataView listening to selection should show both context vars, got:\n%s", got)
	}
}

func TestOutputWidgetMDLV3_GalleryTemplateOnlyCurrentObject(t *testing.T) {
	// Real gallery widgets are CustomWidgets$CustomWidget with RenderMode="gallery".
	// Context comment must appear INSIDE template1 { }, not at the gallery level.
	buf := &bytes.Buffer{}
	e := New(buf)
	w := rawWidget{
		Type:          "CustomWidgets$CustomWidget",
		RenderMode:    "gallery",
		Name:          "artGallery",
		EntityContext: "KB.Article",
		Selection:     "Single",
		DataSource:    &rawDataSource{Type: "database", Reference: "KB.Article"},
		FilterWidgets: []rawWidget{
			{Type: "CustomWidgets$CustomWidget", RenderMode: "dropdownfilter", Name: "fStatus"},
		},
		Children: []rawWidget{
			{Type: "Forms$DynamicText", Name: "txtTitle", Content: "Title"},
		},
	}
	e.outputWidgetMDLV3(w, 0)
	got := buf.String()
	if strings.Contains(got, "$artGallery (selection)") {
		t.Errorf("gallery template should NOT show $artGallery (selection), got:\n%s", got)
	}
	// Context must be inside "template template1 {", not before it.
	templateIdx := strings.Index(got, "template template1 {")
	if templateIdx == -1 {
		t.Fatalf("expected template template1 block in output, got:\n%s", got)
	}
	contextIdx := strings.Index(got, "-- Context: $currentObject (KB.Article)")
	if contextIdx == -1 {
		t.Fatalf("gallery template should show $currentObject context, got:\n%s", got)
	}
	if contextIdx < templateIdx {
		t.Errorf("context comment must appear inside template1 block, not before it, got:\n%s", got)
	}
}

func TestOutputWidgetMDLV3_DataGridColumnInheritsContext(t *testing.T) {
	// DataGrid2 column content widgets should inherit DataGrid2's context
	buf := &bytes.Buffer{}
	e := New(buf)
	w := rawWidget{
		Type:          "CustomWidgets$CustomWidget",
		Name:          "dgProducts",
		RenderMode:    "datagrid2",
		EntityContext: "Shop.Product",
		DataSource:    &rawDataSource{Type: "database", Reference: "Shop.Product"},
		DataGridColumns: []rawDataGridColumn{
			{
				Attribute: "Name",
				Caption:   "Product Name",
			},
		},
	}
	e.outputWidgetMDLV3(w, 0)
	got := buf.String()
	if !strings.Contains(got, "-- Context: $currentObject (Shop.Product), $dgProducts (selection)") {
		t.Errorf("DataGrid2 should show context comment, got:\n%s", got)
	}
}

// TestExtractListViewDataSource_MicroflowSource verifies that extractListViewDataSource
// reads the microflow reference from Forms$MicroflowSource.MicroflowSettings.Microflow.
// Regression guard: the direct ds["Microflow"] access must not be the only lookup path.
func TestExtractListViewDataSource_MicroflowSource(t *testing.T) {
	const mf = "MyModule.DS_ListItems"
	w := map[string]any{
		"DataSource": map[string]any{
			"$Type": "Forms$MicroflowSource",
			"MicroflowSettings": map[string]any{
				"$Type":     "Forms$MicroflowSettings",
				"Microflow": mf,
			},
		},
	}
	ctx := newPageDecodeCtx()

	got := extractListViewDataSource(ctx, w)

	if got == nil {
		t.Fatal("extractListViewDataSource returned nil; want *rawDataSource with microflow reference")
	}
	if got.Type != "microflow" {
		t.Errorf("got.Type = %q, want %q", got.Type, "microflow")
	}
	if got.Reference != mf {
		t.Errorf("got.Reference = %q, want %q", got.Reference, mf)
	}
}

// TestExtractDataViewDataSource_MicroflowSource verifies that extractDataViewDataSource
// reads the microflow reference from Forms$MicroflowSource.MicroflowSettings.Microflow.
// Regression guard: direct ds["Microflow"] access (without MicroflowSettings) returns nil.
func TestExtractDataViewDataSource_MicroflowSource(t *testing.T) {
	const mf = "MyModule.DS_DataViewItems"
	w := map[string]any{
		"DataSource": map[string]any{
			"$Type": "Forms$MicroflowSource",
			"MicroflowSettings": map[string]any{
				"$Type":     "Forms$MicroflowSettings",
				"Microflow": mf,
			},
		},
	}
	ctx := newPageDecodeCtx()

	got := extractDataViewDataSource(ctx, w)

	if got == nil {
		t.Fatal("extractDataViewDataSource returned nil; want *rawDataSource with microflow reference")
	}
	if got.Type != "microflow" {
		t.Errorf("got.Type = %q, want %q", got.Type, "microflow")
	}
	if got.Reference != mf {
		t.Errorf("got.Reference = %q, want %q", got.Reference, mf)
	}
}

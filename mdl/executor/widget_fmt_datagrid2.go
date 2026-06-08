// mdl/executor/widget_fmt_datagrid2.go
package executor

// DataGrid2 is a pluggable widget (CustomWidgets$CustomWidget) identified by the
// widget ID com.mendix.widget.web.datagrid.Datagrid. This file registers the
// CustomWidget meta-type (with a sub-key extractor on WidgetId) plus the DataGrid2
// widget ID specifically. Gallery/ComboBox/Image (Tasks 12–14) only register their
// own widget IDs; the meta-type registration lives here.
//
// During the Phase 2 bridge the formatter delegates to the legacy
// outputWidgetMDLV3 CustomWidget case so describe output stays byte-identical to
// the pre-refactor pipeline (verified by the helpdesk golden snapshot). Phase 3
// rewrites FormatMDL to emit MDL directly.

// widgetIDDataGrid2 is the BSON Type.WidgetId of the pluggable Data Grid 2
// widget. The marketplace package is "Data Grid 2" but its widget ID is
// "com.mendix.widget.web.datagrid.Datagrid" (no "2", lowercase "grid") — the
// exact value the legacy extractCustomWidgetType switch matches.
const widgetIDDataGrid2 = "com.mendix.widget.web.datagrid.Datagrid"

func init() {
	d := DefaultDispatcher()
	// Meta-type: any CustomWidget dispatches on its WidgetId sub-key. Unknown
	// widget IDs fall through to the dispatcher fallback (GenericPluggableFormatter
	// once installed in Phase 3).
	d.registerBSONType("CustomWidgets$CustomWidget", FactoryEntry{ // nolint:describe-raw-bson
		Factory:         GenericPluggableFactory,
		SubKeyExtractor: extractWidgetTypeID,
	})
	d.registerBSONType(widgetIDDataGrid2, FactoryEntry{Factory: dataGrid2Factory})
}

func dataGrid2Factory(raw map[string]any) WidgetFormatter { return &legacyDelegate{raw: raw} }

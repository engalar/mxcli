// mdl/executor/widget_fmt_datagrid.go
package executor

// Built-in DataGrid formatter: the legacy Forms$DataGrid widget (NOT the
// DataGrid2 pluggable widget, which is a CustomWidgets$CustomWidget handled by
// widget_fmt_datagrid2.go).
//
// During the Phase 1/2 bridge this formatter delegates to the legacy
// outputWidgetMDLV3 path (via legacyDelegate, defined in widget_fmt_basic.go) so
// describe output stays byte-identical to the pre-refactor pipeline. The legacy
// pipeline has no dedicated Forms$DataGrid case, so it routes through the
// default branch (an "-- Forms$DataGrid" comment). Registering the type here
// only changes the dispatch indirection, not the output. Phase 3 rewrites
// FormatMDL to emit real `datagrid` MDL with datasource, selection, and columns.

func init() {
	d := DefaultDispatcher()
	for _, t := range []string{"Forms$DataGrid", "Pages$DataGrid"} {
		d.registerBSONType(t, FactoryEntry{Factory: builtinDataGridFactory})
	}
}

func builtinDataGridFactory(raw map[string]any) WidgetFormatter {
	return &legacyDelegate{raw: raw}
}

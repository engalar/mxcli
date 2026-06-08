// mdl/executor/widget_fmt_data.go
package executor

// Data-container widget formatters: DataView, ListView, NavigationList.
//
// During the Phase 1/2 bridge each formatter delegates to the legacy
// outputWidgetMDLV3 case for its type (via legacyDelegate, defined in
// widget_fmt_basic.go) so describe output is byte-identical to the pre-refactor
// pipeline. outputWidgetMDLV3 recurses into children itself, so the whole
// subtree is reproduced exactly. In Phase 3 each FormatMDL is rewritten to emit
// MDL directly and dispatch children through the dispatcher.

func init() {
	d := DefaultDispatcher()
	register := func(factory FormatterFactory, types ...string) {
		for _, t := range types {
			d.registerBSONType(t, FactoryEntry{Factory: factory})
		}
	}
	register(dataViewFactory, "Forms$DataView", "Pages$DataView")
	register(listViewFactory, "Forms$ListView", "Pages$ListView")
	register(navigationListFactory, "Forms$NavigationList", "Pages$NavigationList")
}

func dataViewFactory(raw map[string]any) WidgetFormatter       { return &legacyDelegate{raw: raw} }
func listViewFactory(raw map[string]any) WidgetFormatter       { return &legacyDelegate{raw: raw} }
func navigationListFactory(raw map[string]any) WidgetFormatter { return &legacyDelegate{raw: raw} }

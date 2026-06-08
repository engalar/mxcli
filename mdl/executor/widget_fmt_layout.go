// mdl/executor/widget_fmt_layout.go
package executor

// LayoutGrid formatter.
//
// Like the container formatters, this delegates to the legacy outputWidgetMDLV3
// case during the Phase 1/2 bridge so describe output is byte-identical to the
// pre-refactor pipeline. outputWidgetMDLV3 emits the rows/columns block and
// recurses into column children itself, so registering LayoutGrid here moves the
// whole subtree off the legacyWidgetFallback path with no output change
// (verified by the helpdesk golden snapshot). In Phase 3 FormatMDL is rewritten
// to emit rows/columns directly and recurse via ctx.Dispatcher.

func init() {
	d := DefaultDispatcher()
	for _, t := range []string{"Forms$LayoutGrid", "Pages$LayoutGrid"} {
		d.registerBSONType(t, FactoryEntry{Factory: layoutGridFactory})
	}
}

func layoutGridFactory(raw map[string]any) WidgetFormatter { return &legacyDelegate{raw: raw} }

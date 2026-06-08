// mdl/executor/widget_fmt_container.go
package executor

// Container-widget formatters: DivContainer, GroupBox, ScrollContainer,
// TabControl, TabPage.
//
// Like the basic leaf formatters, these delegate to the legacy
// outputWidgetMDLV3 case during the Phase 1/2 bridge so describe output is
// byte-identical to the pre-refactor pipeline. outputWidgetMDLV3 recurses into
// children itself, so registering a container type here moves the whole subtree
// off the legacyWidgetFallback path with no output change (verified by the
// helpdesk golden snapshot). In Phase 3 each FormatMDL is rewritten to emit MDL
// directly and recurse via ctx.Dispatcher.

func init() {
	d := DefaultDispatcher()
	register := func(factory FormatterFactory, types ...string) {
		for _, t := range types {
			d.registerBSONType(t, FactoryEntry{Factory: factory})
		}
	}
	register(divContainerFactory, "Forms$DivContainer", "Pages$DivContainer")
	register(groupBoxFactory, "Forms$GroupBox", "Pages$GroupBox")
	register(scrollContainerFactory, "Forms$ScrollContainer", "Pages$ScrollContainer")
	register(tabControlFactory, "Forms$TabControl", "Pages$TabControl")
	register(tabPageFactory, "Pages$TabPage")
	// "Footer" is a synthetic container type emitted by the page parser (not a
	// real Mendix BSON $Type). Registering it keeps footers off the dispatcher
	// fallback once the fallback becomes GenericPluggableFormatter (Task 15).
	register(footerFactory, "Footer")
}

func divContainerFactory(raw map[string]any) WidgetFormatter    { return &legacyDelegate{raw: raw} }
func groupBoxFactory(raw map[string]any) WidgetFormatter        { return &legacyDelegate{raw: raw} }
func scrollContainerFactory(raw map[string]any) WidgetFormatter { return &legacyDelegate{raw: raw} }
func tabControlFactory(raw map[string]any) WidgetFormatter      { return &legacyDelegate{raw: raw} }
func tabPageFactory(raw map[string]any) WidgetFormatter         { return &legacyDelegate{raw: raw} }
func footerFactory(raw map[string]any) WidgetFormatter          { return &legacyDelegate{raw: raw} }

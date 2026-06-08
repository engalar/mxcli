// mdl/executor/widget_fmt_image.go
package executor

// Image is a pluggable widget identified by widget ID
// com.mendix.widget.web.image.Image. The CustomWidgets$CustomWidget meta-type is
// registered in widget_fmt_datagrid2.go (Task 11); this file only registers the
// Image widget ID.
//
// Phase 2 bridge: delegates to the legacy outputWidgetMDLV3 CustomWidget case so
// describe output is byte-identical to the pre-refactor pipeline. Phase 3 emits
// MDL directly.

const widgetIDImage = "com.mendix.widget.web.image.Image"

func init() {
	DefaultDispatcher().registerBSONType(widgetIDImage, FactoryEntry{Factory: imageFactory})
}

func imageFactory(raw map[string]any) WidgetFormatter { return &legacyDelegate{raw: raw} }

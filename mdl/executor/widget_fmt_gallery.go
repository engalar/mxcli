// mdl/executor/widget_fmt_gallery.go
package executor

// Gallery is a pluggable widget identified by widget ID
// com.mendix.widget.web.gallery.Gallery. The CustomWidgets$CustomWidget
// meta-type is registered in widget_fmt_datagrid2.go (Task 11); this file only
// registers the Gallery widget ID.
//
// Phase 2 bridge: delegates to the legacy outputWidgetMDLV3 CustomWidget case so
// describe output is byte-identical to the pre-refactor pipeline. Phase 3 emits
// MDL directly.

const widgetIDGallery = "com.mendix.widget.web.gallery.Gallery"

func init() {
	DefaultDispatcher().registerBSONType(widgetIDGallery, FactoryEntry{Factory: galleryFactory})
}

func galleryFactory(raw map[string]any) WidgetFormatter { return &legacyDelegate{raw: raw} }

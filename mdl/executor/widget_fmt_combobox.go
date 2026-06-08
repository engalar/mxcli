// mdl/executor/widget_fmt_combobox.go
package executor

// ComboBox is a pluggable widget identified by widget ID
// com.mendix.widget.web.combobox.ComboBox. The CustomWidgets$CustomWidget
// meta-type is registered in widget_fmt_datagrid2.go (Task 11); this file only
// registers the ComboBox widget ID.
//
// Phase 2 bridge: delegates to the legacy outputWidgetMDLV3 CustomWidget case so
// describe output is byte-identical to the pre-refactor pipeline. Phase 3 emits
// MDL directly.

const widgetIDComboBox = "com.mendix.widget.web.combobox.ComboBox"

func init() {
	DefaultDispatcher().registerBSONType(widgetIDComboBox, FactoryEntry{Factory: comboBoxFactory})
}

func comboBoxFactory(raw map[string]any) WidgetFormatter { return &legacyDelegate{raw: raw} }

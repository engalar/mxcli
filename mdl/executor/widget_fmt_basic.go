// mdl/executor/widget_fmt_basic.go
package executor

// Basic leaf-widget formatters: ActionButton, DynamicText, TextBox, TextArea,
// DatePicker, RadioButtons, CheckBox, Label, StaticText (Text), Title,
// SnippetCallWidget.
//
// During the Phase 1/2 bridge these formatters delegate to the legacy
// outputWidgetMDLV3 case for their type so describe output is byte-identical to
// the pre-refactor pipeline (verified by the helpdesk golden snapshot). The
// dispatch indirection is what changes — registering a type here removes it from
// the legacyWidgetFallback path. In Phase 3 each FormatMDL is rewritten to emit
// MDL directly and outputWidgetMDLV3 is deleted.

func init() {
	d := DefaultDispatcher()
	register := func(factory FormatterFactory, types ...string) {
		for _, t := range types {
			d.registerBSONType(t, FactoryEntry{Factory: factory})
		}
	}
	register(actionButtonFactory, "Forms$ActionButton", "Pages$ActionButton")
	register(dynamicTextFactory, "Forms$DynamicText", "Pages$DynamicText")
	register(textBoxFactory, "Forms$TextBox", "Pages$TextBox")
	register(textAreaFactory, "Forms$TextArea", "Pages$TextArea")
	register(datePickerFactory, "Forms$DatePicker", "Pages$DatePicker")
	register(radioButtonsFactory, "Forms$RadioButtons", "Pages$RadioButtons",
		"Forms$RadioButtonGroup", "Pages$RadioButtonGroup")
	register(checkBoxFactory, "Forms$CheckBox", "Pages$CheckBox")
	register(labelFactory, "Forms$Label", "Pages$Label")
	register(staticTextFactory, "Forms$Text", "Pages$Text")
	register(titleFactory, "Forms$Title", "Pages$Title")
	register(snippetCallFactory, "Forms$SnippetCallWidget", "Pages$SnippetCallWidget")
}

// basicFormatters exposes factory ordering for tests. Index 0 is ActionButton.
var basicFormatters = []struct{ factory FormatterFactory }{
	{factory: actionButtonFactory},
}

// legacyDelegate emits a widget via the legacy outputWidgetMDLV3 case for its
// type, guaranteeing identical output during the bridge phase. The leaf widget
// cases only use ExecContext.Output, so a minimal ExecContext wired to the
// FormatContext's writer reproduces the legacy output exactly.
type legacyDelegate struct{ raw map[string]any }

func (f *legacyDelegate) FormatMDL(ctx *FormatContext) {
	outputWidgetMDLV3(&ExecContext{Output: ctx.Output}, rawWidgetFromMap(f.raw), ctx.Indent)
}

func actionButtonFactory(raw map[string]any) WidgetFormatter { return &legacyDelegate{raw: raw} }
func dynamicTextFactory(raw map[string]any) WidgetFormatter  { return &legacyDelegate{raw: raw} }
func textBoxFactory(raw map[string]any) WidgetFormatter      { return &legacyDelegate{raw: raw} }
func textAreaFactory(raw map[string]any) WidgetFormatter     { return &legacyDelegate{raw: raw} }
func datePickerFactory(raw map[string]any) WidgetFormatter   { return &legacyDelegate{raw: raw} }
func radioButtonsFactory(raw map[string]any) WidgetFormatter { return &legacyDelegate{raw: raw} }
func checkBoxFactory(raw map[string]any) WidgetFormatter     { return &legacyDelegate{raw: raw} }
func labelFactory(raw map[string]any) WidgetFormatter        { return &legacyDelegate{raw: raw} }
func staticTextFactory(raw map[string]any) WidgetFormatter   { return &legacyDelegate{raw: raw} }
func titleFactory(raw map[string]any) WidgetFormatter        { return &legacyDelegate{raw: raw} }
func snippetCallFactory(raw map[string]any) WidgetFormatter  { return &legacyDelegate{raw: raw} }

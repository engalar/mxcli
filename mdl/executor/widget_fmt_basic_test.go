// mdl/executor/widget_fmt_basic_test.go
package executor

import "testing"

// The basic formatters delegate to the legacy outputWidgetMDLV3 case via the
// rawWidget carried in the dispatch map (the same contract describePage uses).
// Tests therefore build the carried rawWidget directly rather than relying on
// parseRawWidget, which reads nested BSON structures not present in these maps.

func TestActionButtonFormatter_PrimaryStyle(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := rawWidgetToMap(rawWidget{
		Type:        "Forms$ActionButton",
		Name:        "btnSubmit",
		ButtonStyle: "primary",
	})
	basicFormatters[0].factory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "actionbutton btnSubmit")
	assertOutput(t, buf, "buttonstyle: primary")
}

func TestActionButtonFormatter_DefaultStyleSuppressed(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := rawWidgetToMap(rawWidget{
		Type:        "Forms$ActionButton",
		Name:        "btnCancel",
		ButtonStyle: "Default",
	})
	basicFormatters[0].factory(raw).FormatMDL(ctx)
	assertNotOutput(t, buf, "buttonstyle")
}

func TestTextBoxFormatter_NameOnly(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := rawWidgetToMap(rawWidget{Type: "Forms$TextBox", Name: "tbName"})
	textBoxFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "textbox tbName")
}

func TestCheckBoxFormatter_NonDefaultEditable(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := rawWidgetToMap(rawWidget{
		Type:      "Forms$CheckBox",
		Name:      "cbActive",
		Editable:  "Never",
		ShowLabel: true,
	})
	checkBoxFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "checkbox cbActive")
	assertOutput(t, buf, "editable: Never")
}

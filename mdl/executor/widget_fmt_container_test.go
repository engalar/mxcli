// mdl/executor/widget_fmt_container_test.go
package executor

import "testing"

// Container formatters delegate to the legacy outputWidgetMDLV3 case via the
// rawWidget carried in the dispatch map. Tests build the carried rawWidget
// directly (including Children) rather than relying on parseRawWidget, which
// reads nested BSON structures not present in these maps.

func TestDivContainerFormatter_RendersChildrenIndented(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := rawWidgetToMap(rawWidget{
		Type: "Forms$DivContainer",
		Name: "ctnMain",
		Children: []rawWidget{
			{Type: "Forms$Text", Name: "lbl1", Content: "hello"},
		},
	})
	divContainerFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "container ctnMain")
	assertOutput(t, buf, "statictext")
}

func TestGroupBoxFormatter_WithCaption(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := rawWidgetToMap(rawWidget{
		Type:    "Forms$GroupBox",
		Name:    "gbSection",
		Caption: "My Section",
	})
	groupBoxFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "groupbox gbSection")
	assertOutput(t, buf, "caption: 'My Section'")
}

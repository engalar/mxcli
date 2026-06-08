// mdl/executor/widget_fmt_layout_test.go
package executor

import "testing"

// LayoutGrid delegates to the legacy outputWidgetMDLV3 case via the rawWidget
// carried in the dispatch map. The test builds the carried rawWidget directly
// (including Rows/Columns) rather than relying on parseRawWidget.

func TestLayoutGridFormatter_EmitsLayoutgridKeyword(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := rawWidgetToMap(rawWidget{
		Type: "Forms$LayoutGrid",
		Name: "lgMain",
	})
	layoutGridFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "layoutgrid lgMain")
}

func TestLayoutGridFormatter_EmitsRowsAndColumns(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := rawWidgetToMap(rawWidget{
		Type: "Forms$LayoutGrid",
		Name: "lgMain",
		Rows: []rawWidgetRow{
			{Columns: []rawWidgetColumn{
				{Width: 6, Widgets: []rawWidget{{Type: "Forms$Text", Name: "t1", Content: "hi"}}},
			}},
		},
	})
	layoutGridFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "layoutgrid lgMain")
	assertOutput(t, buf, "row row1")
	assertOutput(t, buf, "column col1")
	assertOutput(t, buf, "DesktopWidth: 6")
	assertOutput(t, buf, "statictext")
}

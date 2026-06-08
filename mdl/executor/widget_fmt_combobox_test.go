// mdl/executor/widget_fmt_combobox_test.go
package executor

import "testing"

func TestComboBoxFormatter_EmitsComboBoxKeyword(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	// The legacy CustomWidget output emits the `combobox` keyword only when
	// widgetType (RenderMode) is "combobox" AND WidgetID is unset — otherwise the
	// generic `pluggablewidget 'id'` branch intercepts. This mirrors how the
	// legacy parser records ComboBox widgets (RenderMode set, WidgetID empty).
	raw := rawWidgetToMap(rawWidget{
		Type:             "CustomWidgets$CustomWidget",
		Name:             "cbStatus",
		RenderMode:       "combobox",
		DataSource:       &rawDataSource{Type: "database", Reference: "MyModule.Status"},
		CaptionAttribute: "Name",
	})
	comboBoxFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "combobox cbStatus")
}

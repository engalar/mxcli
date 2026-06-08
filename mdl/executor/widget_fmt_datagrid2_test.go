// mdl/executor/widget_fmt_datagrid2_test.go
package executor

import "testing"

func TestDataGrid2Formatter_EmitsDatagridKeyword(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := rawWidgetToMap(rawWidget{
		Type:       "CustomWidgets$CustomWidget",
		Name:       "dgTickets",
		WidgetID:   widgetIDDataGrid2,
		RenderMode: "datagrid2",
		DataSource: &rawDataSource{Type: "database", Reference: "MyModule.Ticket"},
	})
	dataGrid2Factory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "datagrid dgTickets")
}

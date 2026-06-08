// mdl/executor/widget_fmt_data_test.go
package executor

import "testing"

func TestDataViewFormatter_EmitsDataviewKeyword(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := map[string]any{
		"$Type": "Forms$DataView",
		"Name":  "dvTicket",
	}
	dataViewFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "dataview dvTicket")
}

func TestListViewFormatter_EmitsListviewKeyword(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := map[string]any{"$Type": "Forms$ListView", "Name": "lvItems"}
	listViewFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "listview lvItems")
}

func TestNavigationListFormatter_EmitsNavigationlistKeyword(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := map[string]any{"$Type": "Forms$NavigationList", "Name": "nlMenu"}
	navigationListFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "navigationlist nlMenu")
}

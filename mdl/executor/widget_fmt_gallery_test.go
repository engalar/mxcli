// mdl/executor/widget_fmt_gallery_test.go
package executor

import "testing"

func TestGalleryFormatter_EmitsGalleryKeyword(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := rawWidgetToMap(rawWidget{
		Type:       "CustomWidgets$CustomWidget",
		Name:       "galTickets",
		WidgetID:   widgetIDGallery,
		RenderMode: "gallery",
		DataSource: &rawDataSource{Type: "database", Reference: "MyModule.Ticket"},
	})
	galleryFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "gallery galTickets")
}

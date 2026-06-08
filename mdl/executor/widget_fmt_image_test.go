// mdl/executor/widget_fmt_image_test.go
package executor

import "testing"

func TestImageFormatter_EmitsImageKeyword(t *testing.T) {
	ctx, buf := newTestFormatCtx(t)
	raw := rawWidgetToMap(rawWidget{
		Type:       "CustomWidgets$CustomWidget",
		Name:       "imgLogo",
		WidgetID:   widgetIDImage,
		RenderMode: "image",
		ImageUrl:   "https://example.com/logo.png",
	})
	imageFactory(raw).FormatMDL(ctx)
	assertOutput(t, buf, "image imgLogo")
}

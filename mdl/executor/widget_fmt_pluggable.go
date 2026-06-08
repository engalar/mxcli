// mdl/executor/widget_fmt_pluggable.go
package executor

import "fmt"

// GenericPluggableFormatter handles any CustomWidgets$CustomWidget whose widget
// ID is not registered in the dispatcher. It reads the embedded schema from the
// Type BSON, compares each property value against its declared default, and
// outputs only the non-default values. This ensures the output is the minimal
// sufficient MDL needed to recreate the widget.
type GenericPluggableFormatter struct{ raw map[string]any }

// GenericPluggableFactory is the FormatterFactory for GenericPluggableFormatter.
func GenericPluggableFactory(raw map[string]any) WidgetFormatter {
	return &GenericPluggableFormatter{raw: raw}
}

func (f *GenericPluggableFormatter) FormatMDL(ctx *FormatContext) {
	name := safeStr(f.raw, "Name")
	widgetID := extractWidgetTypeID(f.raw)
	if widgetID == "" {
		widgetID = "unknown"
	}

	schema := buildSchemaMap(f.raw)
	props := readProperties(f.raw)
	nonDef := filterDefaults(props, schema)

	if len(nonDef) == 0 {
		ctx.Write(fmt.Sprintf("pluggablewidget '%s' %s", widgetID, name))
		return
	}

	ctx.Write(fmt.Sprintf("pluggablewidget '%s' %s (", widgetID, name))
	for _, p := range nonDef {
		ctx.Child().Write(fmt.Sprintf("%s: %s,", p.Key, formatPropertyValue(p)))
	}
	ctx.Write(")")
}

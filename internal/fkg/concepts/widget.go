// internal/fkg/concepts/widget.go
package concepts

import (
	"context"
	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func init() { Register(&WidgetAdapter{}) }

type WidgetAdapter struct{}

func (a *WidgetAdapter) Name() string { return "fkg:widget" }
func (a *WidgetAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{NodeLabels: []mxgraph.Label{LabelConcept, LabelSyntaxFeature, LabelSkill}}
}
func (a *WidgetAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
func (a *WidgetAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	return sink.Emit([]mxgraph.Event{
		conceptNode("widget", "Widget", "Built-in and custom pluggable widgets"),
		conceptNode("pluggable-widget", "Pluggable Widget", "Custom React widget packaged as .mpk"),

		syntaxNode("widget.pluggable", "PLUGGABLEWIDGET 'id' name (prop: binding) — inline widget use"),
		syntaxNode("widget.new", "mxcli widget new — scaffold a pluggable widget project"),
		syntaxNode("widget.build", "mxcli widget build — compile and package widget to .mpk"),

		skillNode("mendix/custom-widgets", "Widget discovery, def.json extraction, PLUGGABLEWIDGET MDL syntax"),

		edge("pluggable-widget", "widget", Specializes),

		edge("widget", "syntax:widget.pluggable", HasSyntax),
		edge("pluggable-widget", "syntax:widget.new", HasSyntax),
		edge("pluggable-widget", "syntax:widget.build", HasSyntax),

		edge("widget", "skill:mendix/custom-widgets", HasSkill),

		edge("widget", "page", Requires),
	})
}

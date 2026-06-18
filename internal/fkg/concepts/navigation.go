// internal/fkg/concepts/navigation.go
package concepts

import (
	"context"
	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func init() { Register(&NavigationAdapter{}) }

type NavigationAdapter struct{}

func (a *NavigationAdapter) Name() string { return "fkg:navigation" }
func (a *NavigationAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{NodeLabels: []mxgraph.Label{LabelConcept, LabelSyntaxFeature, LabelSkill}}
}
func (a *NavigationAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
func (a *NavigationAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	return sink.Emit([]mxgraph.Event{
		conceptNode("navigation", "Navigation", "Navigation profiles, home pages, and menu items"),

		syntaxNode("navigation.profile", "SET NAVIGATION PROFILE — home page and role mapping"),
		syntaxNode("navigation.menu", "Navigation menu item definitions"),

		skillNode("manage-navigation", "Navigation profiles, home pages, menus, login pages"),

		edge("navigation", "syntax:navigation.profile", HasSyntax),
		edge("navigation", "syntax:navigation.menu", HasSyntax),

		edge("navigation", "skill:manage-navigation", HasSkill),

		edge("navigation", "page", Requires),
		edge("navigation", "security", RelatedTo),
	})
}

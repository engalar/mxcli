// internal/fkg/concepts/page.go
package concepts

import (
	"context"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func init() { Register(&PageAdapter{}) }

// PageAdapter emits concept nodes for Page, DataGrid, Form and all related
// syntax, skill, and cross-concept edges.
type PageAdapter struct{}

func (a *PageAdapter) Name() string { return "fkg:page" }

func (a *PageAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{LabelConcept, LabelSyntaxFeature, LabelSkill},
		EdgeTypes: []struct {
			Type mxgraph.RelType
			From mxgraph.Label
			To   mxgraph.Label
		}{
			{Specializes, LabelConcept, LabelConcept},
			{HasSyntax, LabelConcept, LabelSyntaxFeature},
			{HasSkill, LabelConcept, LabelSkill},
			{RelatedTo, LabelConcept, LabelConcept},
		},
	}
}

func (a *PageAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}

func (a *PageAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	events := []mxgraph.Event{
		// Concept nodes
		conceptNode("page", "Page", "UI pages, layouts, and widgets"),
		conceptNode("datagrid", "DataGrid", "List view with columns and filter bar"),
		conceptNode("form", "Form (DataView)", "Detail form backed by a single object"),
		conceptNode("layout", "Layout", "Page skeleton that frames content pages"),

		// SyntaxFeature nodes
		syntaxNode("page.create", "CREATE OR REPLACE PAGE — full page definition"),
		syntaxNode("page.alter", "ALTER PAGE — in-place structural modifications"),
		syntaxNode("page.widget.datagrid", "DataGrid widget with columns, filters, controlbar"),
		syntaxNode("page.widget.dataview", "DataView (Form) widget with input fields"),

		// Skill nodes
		skillNode("create-page", "Patterns for creating pages and widget compositions"),
		skillNode("alter-page", "ALTER PAGE anchor targeting and SET/INSERT/DROP/REPLACE"),
		skillNode("overview-pages", "CRUD overview page patterns with DataGrid"),
		skillNode("master-detail-pages", "Master-detail page patterns"),

		// Sub-concept specialisation
		edge("datagrid", "page", Specializes),
		edge("form", "page", Specializes),
		edge("layout", "page", Specializes),

		// Concept → SyntaxFeature
		edge("page", "syntax:page.create", HasSyntax),
		edge("page", "syntax:page.alter", HasSyntax),
		edge("datagrid", "syntax:page.widget.datagrid", HasSyntax),
		edge("form", "syntax:page.widget.dataview", HasSyntax),

		// Concept → Skill
		edge("page", "skill:create-page", HasSkill),
		edge("page", "skill:alter-page", HasSkill),
		edge("page", "skill:overview-pages", HasSkill),
		edge("page", "skill:master-detail-pages", HasSkill),

		// Cross-concept edges (target nodes emitted by other adapters)
		edge("page", "microflow", RelatedTo),
		edge("page", "navigation", RelatedTo),
		edge("page", "security", RelatedTo),
		edge("page", "widget", RelatedTo),
	}
	return sink.Emit(events)
}

// internal/fkg/concepts/cross_patterns.go
package concepts

import (
	"context"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func init() { Register(&CrossPatternsAdapter{}) }

type CrossPatternsAdapter struct{}

func (a *CrossPatternsAdapter) Name() string { return "fkg:cross-patterns" }
func (a *CrossPatternsAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{NodeLabels: []mxgraph.Label{LabelPattern, LabelImplDetail}}
}
func (a *CrossPatternsAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
func (a *CrossPatternsAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	return sink.Emit([]mxgraph.Event{
		// ── Cross-cutting Patterns ──────────────────────────────────────────
		patternNode("seed-demo-data", "Seed Demo Data Pattern",
			"Loop over sample data and create entities with associations for development/testing"),
		patternNode("self-ref-association", "Self-Referencing Association Pattern",
			"Hierarchical tree: from KB.Category to KB.Category type reference owner default"),
		patternNode("many-to-many", "Many-to-Many Pattern",
			"Intermediate entity with two associations connecting to parent entities"),

		// ── CodeExtension nodes (shared, not in dedicated adapters) ───────────
		extNode("css-theme", "CSS Theme", "Atlas UI CSS variable override: --brand-primary, --border-radius-button"),
		skillNode("theme-atlas-ui", "Atlas UI theme customization with CSS variable overrides in theme/web/main.css"),

		// ── ImplDetail nodes ──────────────────────────────────────────────────
		implDetailNode("self-ref-assoc", "Self-Referencing Association", "from KB.Category to KB.Category type reference owner default"),
		implDetailNode("intermediate-entity", "Intermediate Entity", "Many-to-Many via intermediate entity with two associations"),
		implDetailNode("string-unique", "Unique String", "Name: string(100) not null unique — database-level uniqueness"),
		implDetailNode("integer-default", "Integer Default", "ViewCount: integer default 0"),
		implDetailNode("system-members", "System Members", "system members (owner, createdDate, changedDate, changedBy)"),
		implDetailNode("non-persistent", "Non-Persistent Entity", "create or modify non-persistent entity HD.TicketSearch ( ... )"),

		// ── Edges: pattern → concept ──────────────────────────────────────────
		edge("microflow", "pattern:seed-demo-data", HasPattern),
		edge("entity", "pattern:seed-demo-data", HasPattern),
		edge("entity", "pattern:self-ref-association", HasPattern),
		edge("entity", "pattern:many-to-many", HasPattern),
		edge("page", "ext:css-theme", HasExt),

		// ── Edges: pattern → detail ──────────────────────────────────────────
		edge("pattern:self-ref-association", "detail:self-ref-assoc", HasSyntax),
		edge("pattern:many-to-many", "detail:intermediate-entity", HasSyntax),
		edge("pattern:seed-demo-data", "detail:loop", HasSyntax),
		edge("pattern:seed-demo-data", "detail:change-object", HasSyntax),
		edge("pattern:seed-demo-data", "detail:commit-object", HasSyntax),
	})
}

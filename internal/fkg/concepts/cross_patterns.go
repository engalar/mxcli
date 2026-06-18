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

		// ── CSS Theme step nodes ───────────────────────────────────────────────
		patternNode("theme-branding", "Branding Theme Pattern",
			"Override Atlas UI CSS variables for brand colors and button styles in theme/web/main.css"),

		stepNode("theme-locate", "Locate theme file",
			"Find theme/web/custom-variables.css (Mendix 9) or theme/web/main.css (Mendix 10/11)",
			1, "configure", "ThemeFile", "main.css",
			"Edit theme/web/main.css in the project directory"),
		stepNode("theme-override", "Override CSS variables",
			"Set --brand-primary, --brand-primary-hover, --border-radius-button etc.",
			2, "configure", "CSSVariables", "brand colors",
			":root { --brand-primary: #1565C0; --brand-primary-hover: #0D47A1; --border-radius-button: 8px; }"),
		stepNode("theme-verify", "Verify theme",
			"CSS changes do not affect mxcli check — purely visual",
			3, "verify", "Theme", "visual check",
			"Run the app (mxcli docker check does not validate CSS)"),

		// ── ImplDetail nodes ──────────────────────────────────────────────────
		implDetailNode("self-ref-assoc", "Self-Referencing Association", "from KB.Category to KB.Category type reference owner default"),
		implDetailNode("intermediate-entity", "Intermediate Entity", "Many-to-Many via intermediate entity with two associations"),
		implDetailNode("string-unique", "Unique String", "Name: string(100) not null unique — database-level uniqueness"),
		implDetailNode("integer-default", "Integer Default", "ViewCount: integer default 0"),
		implDetailNode("system-members", "System Members", "system members (owner, createdDate, changedDate, changedBy)"),
		implDetailNode("non-persistent", "Non-Persistent Entity", "create or modify non-persistent entity HD.TicketSearch ( ... )"),

		// ── Entity attribute type ImplDetail nodes (P3) ───────────────────────
		implDetailNode("attr-string", "string attribute", "string(200), string(500), string not null, string(100) not null unique"),
		implDetailNode("attr-boolean", "boolean attribute", "boolean default true, boolean default false"),
		implDetailNode("attr-datetime", "datetime attribute", "datetime — date and time value"),
		implDetailNode("attr-integer", "integer attribute", "integer default 0 — numeric value"),
		implDetailNode("attr-enum", "enumeration attribute", "Status: HD.TicketStatus default Draft"),
		implDetailNode("attr-unique", "unique constraint", "Name: string(100) not null unique — database-level uniqueness"),
		implDetailNode("attr-system-members", "system members", "system members (owner, createdDate, changedDate, changedBy)"),

		// ── Edges: pattern → concept ──────────────────────────────────────────
		edge("microflow", "pattern:seed-demo-data", HasPattern),
		edge("entity", "pattern:seed-demo-data", HasPattern),
		edge("entity", "pattern:self-ref-association", HasPattern),
		edge("entity", "pattern:many-to-many", HasPattern),
		edge("page", "ext:css-theme", HasExt),
		edge("page", "pattern:theme-branding", HasPattern),

		// ── Edges: entity → attribute type details (P3) ─────────────────────
		edge("entity", "detail:attr-string", HasSyntax),
		edge("entity", "detail:attr-boolean", HasSyntax),
		edge("entity", "detail:attr-datetime", HasSyntax),
		edge("entity", "detail:attr-integer", HasSyntax),
		edge("entity", "detail:attr-enum", HasSyntax),
		edge("entity", "detail:attr-unique", HasSyntax),
		edge("entity", "detail:attr-system-members", HasSyntax),

		// ── Edges: pattern → detail ──────────────────────────────────────────
		edge("pattern:self-ref-association", "detail:self-ref-assoc", HasSyntax),
		edge("pattern:many-to-many", "detail:intermediate-entity", HasSyntax),
		edge("pattern:seed-demo-data", "detail:loop", HasSyntax),
		edge("pattern:seed-demo-data", "detail:change-object", HasSyntax),
		edge("pattern:seed-demo-data", "detail:commit-object", HasSyntax),
		edge("pattern:theme-branding", "step:theme-locate", HasSyntax),
		edge("pattern:theme-branding", "step:theme-override", HasSyntax),
		edge("pattern:theme-branding", "step:theme-verify", HasSyntax),
	})
}

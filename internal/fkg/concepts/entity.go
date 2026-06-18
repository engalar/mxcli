// internal/fkg/concepts/entity.go
package concepts

import (
	"context"
	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func init() { Register(&EntityAdapter{}) }

type EntityAdapter struct{}

func (a *EntityAdapter) Name() string { return "fkg:entity" }
func (a *EntityAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{NodeLabels: []mxgraph.Label{LabelConcept, LabelSyntaxFeature, LabelSkill}}
}
func (a *EntityAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
func (a *EntityAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	return sink.Emit([]mxgraph.Event{
		conceptNode("entity", "Entity", "Persistent data object with attributes and associations"),
		conceptNode("enumeration", "Enumeration", "Named set of discrete values"),
		conceptNode("association", "Association", "Relationship between two entities"),

		syntaxNode("entity.create", "CREATE OR MODIFY ENTITY — attributes, system members, validation"),
		syntaxNode("enumeration.create", "CREATE OR MODIFY ENUMERATION — values, captions"),
		syntaxNode("association.create", "CREATE ASSOCIATION — ownership, multiplicity, delete behavior"),

		skillNode("generate-domain-model", "Entity/attribute/association MDL syntax and patterns"),
		skillNode("mendix/associations", "Ownership, direction, Parent/Child naming, 4 multiplicity patterns"),

		edge("enumeration", "entity", RelatedTo),
		edge("association", "entity", RelatedTo),

		edge("entity", "syntax:entity.create", HasSyntax),
		edge("enumeration", "syntax:enumeration.create", HasSyntax),
		edge("association", "syntax:association.create", HasSyntax),

		edge("entity", "skill:generate-domain-model", HasSkill),
		edge("association", "skill:mendix/associations", HasSkill),

		edge("entity", "security", Requires),
		edge("entity", "microflow", RelatedTo),
	})
}

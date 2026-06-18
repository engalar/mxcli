// internal/fkg/concepts/microflow.go
package concepts

import (
	"context"
	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func init() { Register(&MicroflowAdapter{}) }

type MicroflowAdapter struct{}

func (a *MicroflowAdapter) Name() string { return "fkg:microflow" }
func (a *MicroflowAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{LabelConcept, LabelSyntaxFeature, LabelSkill},
		EdgeTypes:  nil,
	}
}
func (a *MicroflowAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
func (a *MicroflowAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	return sink.Emit([]mxgraph.Event{
		conceptNode("microflow", "Microflow", "Server-side programmatic logic"),
		conceptNode("nanoflow", "Nanoflow", "Client-side logic without server round-trip"),

		syntaxNode("microflow.create", "CREATE OR MODIFY MICROFLOW — full microflow definition"),
		syntaxNode("microflow.variables", "DECLARE / SET — variable declarations and assignments"),
		syntaxNode("microflow.retrieve", "RETRIEVE FROM database with WHERE/SORT/LIMIT"),
		syntaxNode("microflow.control-flow", "IF/ELSIF/ELSE, LOOP, WHILE, BREAK, RETURN"),
		syntaxNode("microflow.object-ops", "CREATE / CHANGE / DELETE / COMMIT / ROLLBACK objects"),
		syntaxNode("microflow.list-ops", "List operations: union, intersect, subtract, sort, filter"),
		syntaxNode("microflow.call", "CALL MICROFLOW / CALL NANOFLOW — sub-flow invocation"),

		skillNode("write-microflows", "Microflow syntax, idioms, and validation checklist"),
		skillNode("write-nanoflows", "Nanoflow restrictions, disallowed activities, checklist"),
		skillNode("patterns-data-processing", "Delta merge, batch processing, list operation patterns"),

		edge("nanoflow", "microflow", Specializes),

		edge("microflow", "syntax:microflow.create", HasSyntax),
		edge("microflow", "syntax:microflow.variables", HasSyntax),
		edge("microflow", "syntax:microflow.retrieve", HasSyntax),
		edge("microflow", "syntax:microflow.control-flow", HasSyntax),
		edge("microflow", "syntax:microflow.object-ops", HasSyntax),
		edge("microflow", "syntax:microflow.list-ops", HasSyntax),
		edge("microflow", "syntax:microflow.call", HasSyntax),

		edge("microflow", "skill:write-microflows", HasSkill),
		edge("nanoflow", "skill:write-nanoflows", HasSkill),
		edge("microflow", "skill:patterns-data-processing", HasSkill),

		edge("microflow", "entity", RelatedTo),
		edge("microflow", "page", RelatedTo),
		edge("microflow", "integration", RelatedTo),
		edge("microflow", "security", RelatedTo),
	})
}

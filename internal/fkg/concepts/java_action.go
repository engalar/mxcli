// internal/fkg/concepts/java_action.go
package concepts

import (
	"context"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func init() { Register(&JavaActionAdapter{}) }

type JavaActionAdapter struct{}

func (a *JavaActionAdapter) Name() string { return "fkg:java-action" }
func (a *JavaActionAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{NodeLabels: []mxgraph.Label{LabelConcept, LabelCodeExtension, LabelSyntaxFeature, LabelSkill}}
}
func (a *JavaActionAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
func (a *JavaActionAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	return sink.Emit([]mxgraph.Event{
		conceptNode("java-action", "Java Action", "Server-side Java extension called from microflows"),

		extNode("java-action", "Java Action", "create or modify java action with imports and code blocks"),
		extNode("java-action.bcrypt", "BCrypt Integration", "Password hashing with external bcrypt.jar in userlib/"),
		extNode("java-action.sha256", "SHA-256 Hashing", "Pure MDL SHA-256 implementation using java.security"),

		syntaxNode("java-action.create", "CREATE OR MODIFY JAVA ACTION name (params) returns type imports $$ ... $$ code $$ ... $$"),
		syntaxNode("java-action.call", "CALL JAVA ACTION Module.ActionName(params) from microflow"),

		skillNode("extend-with-java", "Java Action creation, parameters, return types, external JAR dependencies"),

		edge("java-action", "ext:java-action", HasExt),
		edge("java-action", "ext:java-action.bcrypt", HasExt),
		edge("java-action", "ext:java-action.sha256", HasExt),
		edge("java-action", "syntax:java-action.create", HasSyntax),
		edge("java-action", "syntax:java-action.call", HasSyntax),
		edge("java-action", "skill:extend-with-java", HasSkill),
		edge("java-action", "microflow", RelatedTo),
		edge("java-action", "entity", RelatedTo),
	})
}

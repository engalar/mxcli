// internal/fkg/concepts/security.go
package concepts

import (
	"context"
	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func init() { Register(&SecurityAdapter{}) }

type SecurityAdapter struct{}

func (a *SecurityAdapter) Name() string { return "fkg:security" }
func (a *SecurityAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{NodeLabels: []mxgraph.Label{LabelConcept, LabelSyntaxFeature, LabelSkill}}
}
func (a *SecurityAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
func (a *SecurityAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	return sink.Emit([]mxgraph.Event{
		conceptNode("security", "Security", "Roles, module roles, entity/page/microflow access rules"),

		syntaxNode("security.grant", "GRANT VIEW/WRITE/EXECUTE ON entity/page/microflow TO role"),
		syntaxNode("security.revoke", "REVOKE access rules"),
		syntaxNode("security.role", "CREATE MODULE ROLE — module-level role definition"),
		syntaxNode("security.entity-access", "Entity access rules with member-level read/write control"),

		skillNode("manage-security", "Security roles, access control, GRANT/REVOKE patterns"),

		edge("security", "syntax:security.grant", HasSyntax),
		edge("security", "syntax:security.revoke", HasSyntax),
		edge("security", "syntax:security.role", HasSyntax),
		edge("security", "syntax:security.entity-access", HasSyntax),

		edge("security", "skill:manage-security", HasSkill),

		edge("security", "entity", RelatedTo),
		edge("security", "page", RelatedTo),
		edge("security", "microflow", RelatedTo),
		edge("security", "navigation", RelatedTo),
	})
}

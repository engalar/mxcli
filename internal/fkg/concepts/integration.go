// internal/fkg/concepts/integration.go
package concepts

import (
	"context"
	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func init() { Register(&IntegrationAdapter{}) }

type IntegrationAdapter struct{}

func (a *IntegrationAdapter) Name() string { return "fkg:integration" }
func (a *IntegrationAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{NodeLabels: []mxgraph.Label{LabelConcept, LabelSyntaxFeature, LabelSkill}}
}
func (a *IntegrationAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
func (a *IntegrationAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	return sink.Emit([]mxgraph.Event{
		conceptNode("integration", "Integration", "External connectivity: REST, database, OQL"),
		conceptNode("rest", "REST Service", "Consumed or published REST API"),
		conceptNode("external-db", "External Database", "Direct SQL query against PostgreSQL/Oracle/SQL Server"),

		syntaxNode("integration.rest-call", "CALL REST SERVICE — HTTP method, headers, body, response mapping"),
		syntaxNode("integration.database-connect", "sql connect / sql <alias> select — external DB queries"),
		syntaxNode("integration.oql", "OQL query via mxcli oql against running runtime"),

		skillNode("database-connections", "External database connections from microflows"),

		edge("rest", "integration", Specializes),
		edge("external-db", "integration", Specializes),

		edge("integration", "syntax:integration.rest-call", HasSyntax),
		edge("external-db", "syntax:integration.database-connect", HasSyntax),
		edge("integration", "syntax:integration.oql", HasSyntax),

		edge("integration", "skill:database-connections", HasSkill),

		edge("integration", "microflow", RelatedTo),
	})
}

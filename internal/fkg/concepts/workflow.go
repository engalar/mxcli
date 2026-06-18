// internal/fkg/concepts/workflow.go
package concepts

import (
	"context"
	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func init() { Register(&WorkflowAdapter{}) }

type WorkflowAdapter struct{}

func (a *WorkflowAdapter) Name() string { return "fkg:workflow" }
func (a *WorkflowAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{NodeLabels: []mxgraph.Label{LabelConcept, LabelSyntaxFeature}}
}
func (a *WorkflowAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
func (a *WorkflowAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	return sink.Emit([]mxgraph.Event{
		conceptNode("workflow", "Workflow", "Long-running native Mendix workflow with human tasks"),

		syntaxNode("workflow.create", "CREATE OR MODIFY WORKFLOW — workflow definition and activities"),
		syntaxNode("workflow.activity", "User task, decision, parallel split, and system activities"),

		edge("workflow", "syntax:workflow.create", HasSyntax),
		edge("workflow", "syntax:workflow.activity", HasSyntax),

		edge("workflow", "microflow", RelatedTo),
		edge("workflow", "entity", RelatedTo),
		edge("workflow", "security", RelatedTo),
	})
}

// internal/fkg/concepts/constant.go
package concepts

import (
	"context"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func init() { Register(&ConstantAdapter{}) }

type ConstantAdapter struct{}

func (a *ConstantAdapter) Name() string { return "fkg:constant" }
func (a *ConstantAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{NodeLabels: []mxgraph.Label{LabelConcept, LabelImplDetail, LabelSyntaxFeature, LabelSkill}}
}
func (a *ConstantAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
func (a *ConstantAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	return sink.Emit([]mxgraph.Event{
		conceptNode("constant", "Constant", "Named constant for configurable values referenced in microflows"),

		implDetailNode("constant.create", "CREATE OR MODIFY CONSTANT", "create or modify constant HD.SLA_HIGH_HOURS type integer default 8"),
		implDetailNode("constant.type", "Constant types", "integer, string, boolean, datetime, decimal, float"),
		implDetailNode("constant.reference", "Reference syntax", "@Module.ConstantName in microflow expressions"),

		syntaxNode("constant.create", "CREATE OR MODIFY CONSTANT name type ... default ... comment '...'"),
		syntaxNode("constant.reference", "@Module.ConstantName in microflow expressions"),

		skillNode("manage-constants", "Creating and referencing constants in microflows; SLA config pattern"),

		edge("constant", "detail:constant.create", HasSyntax),
		edge("constant", "detail:constant.type", HasSyntax),
		edge("constant", "detail:constant.reference", HasSyntax),
		edge("constant", "syntax:constant.create", HasSyntax),
		edge("constant", "syntax:constant.reference", HasSyntax),
		edge("constant", "skill:manage-constants", HasSkill),
		edge("constant", "microflow", RelatedTo),
		edge("constant", "entity", RelatedTo),
	})
}

package mpr

import (
	"context"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
)

// WorkflowAdapter emits Workflow nodes from the project.
//
// BSON type: Workflows$Workflow
type WorkflowAdapter struct {
	Model *modelsdk.Model
}

func (a *WorkflowAdapter) Name() string { return "workflow" }

func (a *WorkflowAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"Workflow"},
		EdgeTypes:  nil,
	}
}

func (a *WorkflowAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}

func (a *WorkflowAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
	var events []mxgraph.Event

	for _, unit := range a.Model.Units() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		elem, err := a.Model.LoadUnit(unit.ID)
		if err != nil {
			continue
		}
		if elem.TypeName() != "Workflows$Workflow" {
			continue
		}

		// Workflow gen type is missing SetProperties (codegen gap), so
		// elem.Properties() returns empty. Build props manually here.
		props := map[string]any{"$Type": elem.TypeName()}
		if n, ok := elem.(interface{ Name() string }); ok {
			props["Name"] = n.Name()
		}
		if d, ok := elem.(interface{ Documentation() string }); ok {
			if d.Documentation() != "" {
				props["Documentation"] = d.Documentation()
			}
		}
		node := &mxgraph.Node{ID: mxgraph.NodeID(elem.ID()), Label: "Workflow", Props: props}
		setDerived(node, a.Model.ResolveModuleName(unit.ID))

		events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: node})
	}

	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

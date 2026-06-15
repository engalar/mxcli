package mpr

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
)

// PageAdapter emits Page / Layout / Snippet nodes and HAS_LAYOUT edges.
//
// Verified BSON type names (modelsdk/gen/pages/descriptors.go):
//   - Forms$Page    — pages
//   - Forms$Layout  — layouts
//   - Forms$Snippet — snippets
//
// A page references its layout through a nested Part chain (not a direct property):
//
//	Forms$Page → FormCall (Part, Forms$LayoutCall) → Form (ByNameRef → Layout QN)
type PageAdapter struct {
	Model *modelsdk.Model
}

func (a *PageAdapter) Name() string { return "page" }

func (a *PageAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"Page", "Layout", "Snippet"},
		EdgeTypes: []struct {
			Type mxgraph.RelType
			From mxgraph.Label
			To   mxgraph.Label
		}{
			{"HAS_LAYOUT", "Page", "Layout"},
		},
	}
}

func (a *PageAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
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

		var label mxgraph.Label
		switch elem.TypeName() {
		case "Forms$Page":
			label = "Page"
		case "Forms$Layout":
			label = "Layout"
		case "Forms$Snippet":
			label = "Snippet"
		default:
			continue
		}

		node := nodeForElement(elem, label)
		events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: node})

		if label == "Page" {
			if call := childElement(elem, "FormCall"); call != nil {
				if layoutQN := refValue(call, "Form"); layoutQN != "" {
					events = append(events, mxgraph.Event{
						Type: mxgraph.EdgeCreated,
						Edge: &mxgraph.Edge{
							ID:   mxgraph.NodeID(fmt.Sprintf("%s--HAS_LAYOUT-->%s", node.ID, layoutQN)),
							From: node.ID,
							To:   mxgraph.NodeID(layoutQN),
							Type: "HAS_LAYOUT",
						},
					})
				}
			}
		}
	}

	if len(events) > 0 {
		if err := sink.Emit(events); err != nil {
			return fmt.Errorf("emit events: %w", err)
		}
	}
	return nil
}

func (a *PageAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}

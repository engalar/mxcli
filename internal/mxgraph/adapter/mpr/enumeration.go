package mpr

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
)

// EnumerationAdapter emits Enumeration / EnumValue nodes and HAS_VALUE edges.
//
// Verified BSON structure (modelsdk/gen/enumerations/descriptors.go):
//
//	Enumerations$Enumeration (the unit) → Values (PartList of Enumerations$EnumerationValue)
type EnumerationAdapter struct {
	Model *modelsdk.Model
}

func (a *EnumerationAdapter) Name() string { return "enumeration" }

func (a *EnumerationAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"Enumeration", "EnumValue"},
		EdgeTypes: []struct {
			Type mxgraph.RelType
			From mxgraph.Label
			To   mxgraph.Label
		}{
			{"HAS_VALUE", "Enumeration", "EnumValue"},
		},
	}
}

func (a *EnumerationAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
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
		if elem.TypeName() != "Enumerations$Enumeration" {
			continue
		}

		enumNode := nodeForElement(elem, "Enumeration")
		events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: enumNode})

		for _, val := range childList(elem, "Values") {
			if val == nil {
				continue
			}
			valNode := nodeForElement(val, "EnumValue")
			events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: valNode})
			events = append(events, mxgraph.Event{
				Type: mxgraph.EdgeCreated,
				Edge: &mxgraph.Edge{
					ID:   mxgraph.NodeID(fmt.Sprintf("%s--HAS_VALUE-->%s", enumNode.ID, valNode.ID)),
					From: enumNode.ID,
					To:   valNode.ID,
					Type: "HAS_VALUE",
				},
			})
		}
	}

	if len(events) > 0 {
		if err := sink.Emit(events); err != nil {
			return fmt.Errorf("emit events: %w", err)
		}
	}
	return nil
}

func (a *EnumerationAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}

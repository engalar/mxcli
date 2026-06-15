package mpr

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

type Adapter struct {
	Model *modelsdk.Model
}

func (a *Adapter) Name() string { return "mpr" }

func (a *Adapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{
			"Module", "DomainModel", "Entity", "Attribute",
			"Microflow", "Page", "Layout", "Association",
		},
		EdgeTypes: []struct {
			Type mxgraph.RelType
			From mxgraph.Label
			To   mxgraph.Label
		}{
			{"HAS_ENTITY", "DomainModel", "Entity"},
			{"HAS_ATTRIBUTE", "Entity", "Attribute"},
			{"HAS_ASSOCIATION", "DomainModel", "Association"},
		},
	}
}

func (a *Adapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
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

		typeName := elem.TypeName()
		switch {
		case typeName == "DomainModels$DomainModel":
			n := nodeForUnit(unit.ID, "DomainModel", elem)
			events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: n})

			for _, prop := range elem.Properties() {
				if prop.Name() != "Entities" {
					continue
				}
				cl, ok := prop.(element.ChildListProperty)
				if !ok {
					continue
				}
				for _, child := range cl.ChildElements() {
					if child == nil {
						continue
					}
					ct := child.TypeName()
					var label mxgraph.Label
					switch {
					case ct == "DomainModels$Entity" || ct == "DomainModels$EntityImpl":
						label = "Entity"
					default:
						continue
					}
					cn := nodeForElement(child, label)
					events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: cn})
					events = append(events, mxgraph.Event{
						Type: mxgraph.EdgeCreated,
						Edge: &mxgraph.Edge{
							ID:   mxgraph.NodeID(fmt.Sprintf("%s->%s", unit.ID, child.ID())),
							From: mxgraph.NodeID(unit.ID),
							To:   mxgraph.NodeID(child.ID()),
							Type: "HAS_ENTITY",
						},
					})

					for _, ap := range child.Properties() {
						if ap.Name() != "Attributes" {
							continue
						}
						cl2, ok := ap.(element.ChildListProperty)
						if !ok {
							continue
						}
						for _, attr := range cl2.ChildElements() {
							if attr == nil {
								continue
							}
							an := nodeForElement(attr, "Attribute")
							events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: an})
							events = append(events, mxgraph.Event{
								Type: mxgraph.EdgeCreated,
								Edge: &mxgraph.Edge{
									ID:   mxgraph.NodeID(fmt.Sprintf("%s->%s", child.ID(), attr.ID())),
									From: mxgraph.NodeID(child.ID()),
									To:   mxgraph.NodeID(attr.ID()),
									Type: "HAS_ATTRIBUTE",
								},
							})
						}
					}
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

func (a *Adapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}

func nodeForUnit(id element.ID, label mxgraph.Label, elem element.Element) *mxgraph.Node {
	props := map[string]any{}
	props["$Type"] = elem.TypeName()
	for _, p := range elem.Properties() {
		if wp, ok := p.(element.WritableProperty); ok {
			if v := wp.BSONValue(); v != nil {
				props[p.Name()] = v
			}
		}
	}
	return &mxgraph.Node{ID: mxgraph.NodeID(id), Label: label, Props: props}
}

func nodeForElement(elem element.Element, label mxgraph.Label) *mxgraph.Node {
	props := map[string]any{}
	props["$Type"] = elem.TypeName()
	for _, p := range elem.Properties() {
		if wp, ok := p.(element.WritableProperty); ok {
			if v := wp.BSONValue(); v != nil {
				props[p.Name()] = v
			}
		}
	}
	return &mxgraph.Node{ID: mxgraph.NodeID(elem.ID()), Label: label, Props: props}
}

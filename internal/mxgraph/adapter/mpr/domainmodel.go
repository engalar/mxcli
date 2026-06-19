package mpr

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// DomainModelAdapter emits DomainModel / Entity / Attribute / Association nodes
// and their containment edges.
//
// Verified BSON type names (modelsdk/gen/domainmodels/descriptors.go):
//   - DomainModels$DomainModel — the unit
//   - DomainModels$Entity      — entities (Entities child list)
//   - DomainModels$Attribute   — attributes (Attributes child list on Entity)
//   - DomainModels$Association — associations (Associations child list)
type DomainModelAdapter struct {
	Model *modelsdk.Model
}

func (a *DomainModelAdapter) Name() string { return "domainmodel" }

func (a *DomainModelAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"DomainModel", "Entity", "Attribute", "Association"},
		EdgeTypes: []struct {
			Type mxgraph.RelType
			From mxgraph.Label
			To   mxgraph.Label
		}{
			{"HAS_ENTITY", "DomainModel", "Entity"},
			{"HAS_ATTRIBUTE", "Entity", "Attribute"},
			{"HAS_ASSOCIATION", "DomainModel", "Association"},
			{"HAS_ASSOCIATION", "Entity", "Association"},
			{"ASSOCIATES_TO", "Association", "Entity"},
		},
	}
}

func (a *DomainModelAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
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

		if elem.TypeName() != "DomainModels$DomainModel" {
			continue
		}

		module := a.Model.ResolveModuleName(unit.ID)

		dmNode := nodeForElement(elem, "DomainModel")
		setDerived(dmNode, module)
		events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: dmNode})

		for _, prop := range elem.Properties() {
			switch prop.Name() {
			case "Entities":
				cl, ok := prop.(element.ChildListProperty)
				if !ok {
					continue
				}
				for _, child := range cl.ChildElements() {
					if child == nil {
						continue
					}
					ct := child.TypeName()
					if ct != "DomainModels$Entity" && ct != "DomainModels$EntityImpl" {
						continue
					}
					entityNode := nodeForElement(child, "Entity")
					setDerived(entityNode, module)
					events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: entityNode})
					events = append(events, mxgraph.Event{
						Type: mxgraph.EdgeCreated,
						Edge: &mxgraph.Edge{
							ID:   mxgraph.NodeID(fmt.Sprintf("%s->%s", dmNode.ID, entityNode.ID)),
							From: dmNode.ID,
							To:   entityNode.ID,
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
							attrNode := nodeForElement(attr, "Attribute")
							setDerived(attrNode, module)
							events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: attrNode})
							events = append(events, mxgraph.Event{
								Type: mxgraph.EdgeCreated,
								Edge: &mxgraph.Edge{
									ID:   mxgraph.NodeID(fmt.Sprintf("%s->%s", entityNode.ID, attrNode.ID)),
									From: entityNode.ID,
									To:   attrNode.ID,
									Type: "HAS_ATTRIBUTE",
								},
							})
						}
					}
				}
			case "Associations":
				cl, ok := prop.(element.ChildListProperty)
				if !ok {
					continue
				}
				for _, child := range cl.ChildElements() {
					if child == nil {
						continue
					}

					assocNode := nodeForElement(child, "Association")
					setDerived(assocNode, module)
					events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: assocNode})
					events = append(events, mxgraph.Event{
						Type: mxgraph.EdgeCreated,
						Edge: &mxgraph.Edge{
							ID:   mxgraph.NodeID(fmt.Sprintf("%s->%s", dmNode.ID, assocNode.ID)),
							From: dmNode.ID,
							To:   assocNode.ID,
							Type: "HAS_ASSOCIATION",
						},
					})

					// 提取 parent/child 实体引用，创建 ASSOCIATES_TO 边
					if assoc, ok := child.(*genDm.Association); ok {
						parentID := assoc.ParentRefID()
						childID := assoc.ChildRefID()
						if parentID != "" {
							events = append(events, mxgraph.Event{
								Type: mxgraph.EdgeCreated,
								Edge: &mxgraph.Edge{
									ID:   mxgraph.NodeID(fmt.Sprintf("%s--HAS_ASSOCIATION-->%s", parentID, assocNode.ID)),
									From: mxgraph.NodeID(parentID),
									To:   assocNode.ID,
									Type: "HAS_ASSOCIATION",
								},
							})
							events = append(events, mxgraph.Event{
								Type: mxgraph.EdgeCreated,
								Edge: &mxgraph.Edge{
									ID:   mxgraph.NodeID(fmt.Sprintf("%s--ASSOCIATES_TO-->%s", assocNode.ID, parentID)),
									From: assocNode.ID,
									To:   mxgraph.NodeID(parentID),
									Type: "ASSOCIATES_TO",
								},
							})
						}
						if childID != "" {
							events = append(events, mxgraph.Event{
								Type: mxgraph.EdgeCreated,
								Edge: &mxgraph.Edge{
									ID:   mxgraph.NodeID(fmt.Sprintf("%s--HAS_ASSOCIATION-->%s", childID, assocNode.ID)),
									From: mxgraph.NodeID(childID),
									To:   assocNode.ID,
									Type: "HAS_ASSOCIATION",
								},
							})
							events = append(events, mxgraph.Event{
								Type: mxgraph.EdgeCreated,
								Edge: &mxgraph.Edge{
									ID:   mxgraph.NodeID(fmt.Sprintf("%s--ASSOCIATES_TO-->%s", assocNode.ID, childID)),
									From: assocNode.ID,
									To:   mxgraph.NodeID(childID),
									Type: "ASSOCIATES_TO",
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

func (a *DomainModelAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}

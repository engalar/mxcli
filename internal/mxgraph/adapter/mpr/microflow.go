package mpr

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// MicroflowAdapter emits Microflow / Nanoflow nodes and the reference edges they
// carry: CALLS, CREATES, RETRIEVES, SHOWS_PAGE.
//
// Verified BSON structure (modelsdk/gen/microflows/descriptors.go,
// modelsdk/gen/pages/descriptors.go) — note both microflows and nanoflows live
// in the "Microflows$" type namespace:
//
//	Microflows$Microflow / Microflows$Nanoflow  (the unit)
//	  └─ ObjectCollection: Microflows$MicroflowObjectCollection (Part)
//	       └─ Objects: []Microflows$ActionActivity (PartList)
//	            └─ Action: <concrete action> (Part)
//
// Reference targets are ByNameRef values, i.e. *qualified-name strings*
// (e.g. "MyModule.SomeMicroflow"), not element IDs. Edge.To therefore holds the
// qualified name; ProjectGraph resolves edges against the QualifiedName prop.
//
//	MicroflowCallAction → MicroflowCall (Part) → Microflow (ByNameRef → Microflow QN)
//	CreateChangeAction  → Entity (ByNameRef → Entity QN)
//	RetrieveAction      → RetrieveSource (Part) → Entity / AssociationId (ByNameRef)
//	ShowFormAction      → FormSettings (Part, Forms$FormSettings) → Form (ByNameRef → Page QN)
type MicroflowAdapter struct {
	Model *modelsdk.Model
}

func (a *MicroflowAdapter) Name() string { return "microflow" }

func (a *MicroflowAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"Microflow", "Nanoflow"},
		EdgeTypes: []struct {
			Type mxgraph.RelType
			From mxgraph.Label
			To   mxgraph.Label
		}{
			{"CALLS", "Microflow", "Microflow"},
			{"CALLS", "Microflow", "Nanoflow"},
			{"CALLS", "Nanoflow", "Nanoflow"},
			{"CREATES", "Microflow", "Entity"},
			{"RETRIEVES", "Microflow", "Entity"},
			{"SHOWS_PAGE", "Microflow", "Page"},
		},
	}
}

func (a *MicroflowAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
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
		case "Microflows$Microflow":
			label = "Microflow"
		case "Microflows$Nanoflow":
			label = "Nanoflow"
		default:
			continue
		}

		mfNode := nodeForElement(elem, label)
		setDerived(mfNode, a.Model.ResolveModuleName(unit.ID))
		events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: mfNode})

		for _, action := range microflowActions(elem) {
			events = append(events, extractActivityEdges(mfNode.ID, action)...)
		}
	}

	if len(events) > 0 {
		if err := sink.Emit(events); err != nil {
			return fmt.Errorf("emit events: %w", err)
		}
	}
	return nil
}

// microflowActions walks ObjectCollection → Objects → ActionActivity.Action and
// returns each concrete action element. Activities without an Action child (loops,
// annotations, etc.) are skipped.
func microflowActions(mf element.Element) []element.Element {
	var actions []element.Element
	collection := childElement(mf, "ObjectCollection")
	if collection == nil {
		return nil
	}
	for _, obj := range childList(collection, "Objects") {
		if action := childElement(obj, "Action"); action != nil {
			actions = append(actions, action)
		}
	}
	return actions
}

// extractActivityEdges turns one concrete microflow action into reference edges.
func extractActivityEdges(fromID mxgraph.NodeID, action element.Element) []mxgraph.Event {
	typeName := action.TypeName()

	emit := func(rel mxgraph.RelType, target string) mxgraph.Event {
		return mxgraph.Event{
			Type: mxgraph.EdgeCreated,
			Edge: &mxgraph.Edge{
				ID:   mxgraph.NodeID(fmt.Sprintf("%s--%s-->%s", fromID, rel, target)),
				From: fromID,
				To:   mxgraph.NodeID(target),
				Type: rel,
			},
		}
	}

	var events []mxgraph.Event
	switch {
	case strings.HasSuffix(typeName, "$MicroflowCallAction"):
		if call := childElement(action, "MicroflowCall"); call != nil {
			if qn := refValue(call, "Microflow"); qn != "" {
				events = append(events, emit("CALLS", qn))
			}
		}
	case strings.HasSuffix(typeName, "$NanoflowCallAction"):
		if call := childElement(action, "NanoflowCall"); call != nil {
			if qn := refValue(call, "Nanoflow"); qn != "" {
				events = append(events, emit("CALLS", qn))
			}
		}
	case strings.HasSuffix(typeName, "$CreateChangeAction"),
		strings.HasSuffix(typeName, "$CreateObjectAction"):
		if qn := refValue(action, "Entity"); qn != "" {
			events = append(events, emit("CREATES", qn))
		}
	case strings.HasSuffix(typeName, "$RetrieveAction"):
		if src := childElement(action, "RetrieveSource"); src != nil {
			if qn := refValue(src, "Entity"); qn != "" {
				events = append(events, emit("RETRIEVES", qn))
			}
		}
	case strings.HasSuffix(typeName, "$ShowFormAction"):
		if settings := childElement(action, "FormSettings"); settings != nil {
			if qn := refValue(settings, "Form"); qn != "" {
				events = append(events, emit("SHOWS_PAGE", qn))
			}
		}
	}
	return events
}

func (a *MicroflowAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}

// childElement returns the single child Element held by the named Part property,
// or nil if the property is missing, not a Part, or empty.
func childElement(elem element.Element, propName string) element.Element {
	for _, p := range elem.Properties() {
		if p.Name() != propName {
			continue
		}
		if cp, ok := p.(element.ChildProperty); ok {
			return cp.ChildElement()
		}
		return nil
	}
	return nil
}

// childList returns the child Elements held by the named PartList property.
func childList(elem element.Element, propName string) []element.Element {
	for _, p := range elem.Properties() {
		if p.Name() != propName {
			continue
		}
		if cl, ok := p.(element.ChildListProperty); ok {
			return cl.ChildElements()
		}
		return nil
	}
	return nil
}

// refValue returns the qualified-name string of the named ByNameRef property,
// or "" if missing/empty. ByNameRef.BSONValue() returns the qualified name.
func refValue(elem element.Element, propName string) string {
	for _, p := range elem.Properties() {
		if p.Name() != propName {
			continue
		}
		if wp, ok := p.(element.WritableProperty); ok {
			if v := wp.BSONValue(); v != nil {
				if s, ok := v.(string); ok {
					return s
				}
			}
		}
		return ""
	}
	return ""
}

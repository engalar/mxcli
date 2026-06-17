package mpr

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// WorkflowAdapter emits Workflow nodes and edges from the project.
//
// BSON type: Workflows$Workflow
// Edges:
//   CALLS  → Microflow  (CallMicroflowTask / CallMicroflowActivity)
//   CALLS  → Workflow   (CallWorkflowActivity)
type WorkflowAdapter struct {
	Model *modelsdk.Model
}

func (a *WorkflowAdapter) Name() string { return "workflow" }

func (a *WorkflowAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"Workflow"},
		EdgeTypes: []struct {
			Type mxgraph.RelType
			From mxgraph.Label
			To   mxgraph.Label
		}{
			{"CALLS", "Workflow", "Microflow"},
			{"CALLS", "Workflow", "Workflow"},
		},
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
		wfNode := &mxgraph.Node{ID: mxgraph.NodeID(elem.ID()), Label: "Workflow", Props: props}
		setDerived(wfNode, a.Model.ResolveModuleName(unit.ID))
		events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: wfNode})

		// Walk the workflow's flow tree to find call activities.
		// Use typed getter Flow() on Workflow (Properties() is empty due
		// to codegen gap), then generic helpers on sub-elements.
		if wf, ok := elem.(interface{ Flow() element.Element }); ok {
			flow := wf.Flow()
			if flow != nil {
				events = append(events, walkWorkflowFlow(wfNode.ID, flow)...)
			}
		}
	}

	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

// walkWorkflowFlow recursively walks a Workflows$Flow's Activities list
// and emits CALLS edges for call-microflow / call-workflow activities.
func walkWorkflowFlow(wfNodeID mxgraph.NodeID, flow element.Element) []mxgraph.Event {
	var events []mxgraph.Event

	for _, act := range childList(flow, "Activities") {

		// Flatten versioned-array entries (index 0 is an int32 marker)
		if list, ok := act.(interface{ Items() []element.Element }); ok {
			for _, item := range list.Items() {
				events = append(events, inspectActivity(wfNodeID, item)...)
			}
			continue
		}

		events = append(events, inspectActivity(wfNodeID, act)...)
	}
	return events
}

// inspectActivity checks a single activity element and emits CALLS edges
// for call-microflow and call-workflow activities, plus recursively walks
// nested flows (decision outcomes, user task outcomes, boundary events).
func inspectActivity(wfNodeID mxgraph.NodeID, act element.Element) []mxgraph.Event {
	var events []mxgraph.Event

	tn := act.TypeName()

	switch {
	case strings.HasSuffix(tn, "$CallMicroflowTask"),
		strings.HasSuffix(tn, "$CallMicroflowActivity"):
		if qn := refValue(act, "Microflow"); qn != "" {
			events = append(events, callEdge(wfNodeID, "CALLS", qn))
		}

	case strings.HasSuffix(tn, "$CallWorkflowActivity"):
		if qn := refValue(act, "Workflow"); qn != "" {
			events = append(events, callEdge(wfNodeID, "CALLS", qn))
		}
	}

	// Recurse into outcome flows (user task outcomes, decision outcomes)
	for _, oc := range childList(act, "Outcomes") {
		if flow := childElement(oc, "Flow"); flow != nil {
			events = append(events, walkWorkflowFlow(wfNodeID, flow)...)
		}
	}

	// Recurse into boundary events (each has its own Flow)
	for _, be := range childList(act, "BoundaryEvents") {
		if flow := childElement(be, "Flow"); flow != nil {
			events = append(events, walkWorkflowFlow(wfNodeID, flow)...)
		}
	}

	return events
}

func callEdge(from mxgraph.NodeID, rel mxgraph.RelType, target string) mxgraph.Event {
	return mxgraph.Event{
		Type: mxgraph.EdgeCreated,
		Edge: &mxgraph.Edge{
			ID:   mxgraph.NodeID(fmt.Sprintf("%s--%s-->%s", from, rel, target)),
			From: from,
			To:   mxgraph.NodeID(target),
			Type: rel,
		},
	}
}

// ── BSON walk helpers (delegated to microflow.go helpers in the same package) ─

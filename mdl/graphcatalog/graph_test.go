package graphcatalog_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
)

// buildTestProjectGraph constructs an in-memory graph that mirrors the adapter
// conventions: node IDs are element IDs, but reference edges (CALLS/CREATES/...)
// store the *target QualifiedName* as Edge.To, not the target node's ID.
func buildTestProjectGraph() *graphcatalog.ProjectGraph {
	mgr := mxgraph.NewIndexManager()
	g := mgr.Query()

	g.AddNode("e1", "Entity", map[string]any{
		"Name":          "Ticket",
		"Module":        "Helpdesk",
		"QualifiedName": "Helpdesk.Ticket",
	})
	g.AddNode("e2", "Entity", map[string]any{
		"Name":          "Agent",
		"Module":        "Helpdesk",
		"QualifiedName": "Helpdesk.Agent",
	})
	g.AddNode("mf1", "Microflow", map[string]any{
		"Name":          "ACT_CreateTicket",
		"Module":        "Helpdesk",
		"QualifiedName": "Helpdesk.ACT_CreateTicket",
	})
	g.AddNode("mf2", "Microflow", map[string]any{
		"Name":          "ACT_AssignAgent",
		"Module":        "Helpdesk",
		"QualifiedName": "Helpdesk.ACT_AssignAgent",
	})

	// mf2 CALLS mf1 — Edge.To is mf1's QualifiedName string, not its NodeID.
	g.AddEdge("call1", "mf2", mxgraph.NodeID("Helpdesk.ACT_CreateTicket"), "CALLS", nil)
	// mf1 CREATES Ticket — Edge.To is the entity QualifiedName.
	g.AddEdge("creates1", "mf1", mxgraph.NodeID("Helpdesk.Ticket"), "CREATES", nil)

	return graphcatalog.NewProjectGraph(mgr)
}

func TestProjectGraph_Entities(t *testing.T) {
	pg := buildTestProjectGraph()
	entities := pg.Entities("Helpdesk")
	if len(entities) != 2 {
		t.Fatalf("Entities(Helpdesk) = %d, want 2", len(entities))
	}
}

func TestProjectGraph_Entity(t *testing.T) {
	pg := buildTestProjectGraph()
	e := pg.Entity("Helpdesk.Ticket")
	if e == nil {
		t.Fatal("Entity(Helpdesk.Ticket) returned nil")
	}
	if e.Name != "Ticket" {
		t.Errorf("Name = %q, want Ticket", e.Name)
	}
}

func TestProjectGraph_Microflows(t *testing.T) {
	pg := buildTestProjectGraph()
	mfs := pg.Microflows("Helpdesk")
	if len(mfs) != 2 {
		t.Fatalf("Microflows(Helpdesk) = %d, want 2", len(mfs))
	}
}

func TestProjectGraph_Callers_direct(t *testing.T) {
	pg := buildTestProjectGraph()
	callers := pg.Callers("Helpdesk.ACT_CreateTicket", false)
	if len(callers) != 1 {
		t.Fatalf("Callers(ACT_CreateTicket) = %d, want 1", len(callers))
	}
	if callers[0].Caller != "Helpdesk.ACT_AssignAgent" {
		t.Errorf("Caller = %q, want Helpdesk.ACT_AssignAgent", callers[0].Caller)
	}
	if callers[0].Callee != "Helpdesk.ACT_CreateTicket" {
		t.Errorf("Callee = %q, want Helpdesk.ACT_CreateTicket", callers[0].Callee)
	}
}

func TestProjectGraph_Callees_direct(t *testing.T) {
	pg := buildTestProjectGraph()
	callees := pg.Callees("Helpdesk.ACT_AssignAgent", false)
	if len(callees) != 1 {
		t.Fatalf("Callees(ACT_AssignAgent) = %d, want 1", len(callees))
	}
	if callees[0].Callee != "Helpdesk.ACT_CreateTicket" {
		t.Errorf("Callee = %q, want Helpdesk.ACT_CreateTicket", callees[0].Callee)
	}
}

func TestProjectGraph_Callers_transitive(t *testing.T) {
	mgr := mxgraph.NewIndexManager()
	g := mgr.Query()
	g.AddNode("a", "Microflow", map[string]any{"QualifiedName": "M.A"})
	g.AddNode("b", "Microflow", map[string]any{"QualifiedName": "M.B"})
	g.AddNode("c", "Microflow", map[string]any{"QualifiedName": "M.C"})
	// A CALLS B, B CALLS C
	g.AddEdge("ab", "a", mxgraph.NodeID("M.B"), "CALLS", nil)
	g.AddEdge("bc", "b", mxgraph.NodeID("M.C"), "CALLS", nil)
	pg := graphcatalog.NewProjectGraph(mgr)

	callers := pg.Callers("M.C", true)
	got := map[string]bool{}
	for _, c := range callers {
		got[c.Caller] = true
	}
	if !got["M.B"] || !got["M.A"] {
		t.Fatalf("transitive Callers(M.C) = %+v, want both M.A and M.B", callers)
	}
}

func TestProjectGraph_References(t *testing.T) {
	pg := buildTestProjectGraph()
	refs := pg.References("Helpdesk.ACT_CreateTicket")
	if len(refs) == 0 {
		t.Fatal("References(ACT_CreateTicket) returned empty")
	}
	found := false
	for _, r := range refs {
		if r.RefKind == "CREATES" && r.Target == "Helpdesk.Ticket" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected CREATES ref to Helpdesk.Ticket, got %+v", refs)
	}
}

func TestProjectGraph_Impact(t *testing.T) {
	pg := buildTestProjectGraph()
	// Ticket is CREATEd by mf1 — inbound CREATES edge.
	impact := pg.Impact("Helpdesk.Ticket")
	found := false
	for _, r := range impact {
		if r.RefKind == "CREATES" && r.Source == "Helpdesk.ACT_CreateTicket" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected CREATES impact from ACT_CreateTicket, got %+v", impact)
	}
}

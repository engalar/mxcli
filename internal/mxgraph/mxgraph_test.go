package mxgraph

import "testing"

func TestDirectionValues(t *testing.T) {
	if Outbound != 0 || Inbound != 1 || Both != 2 {
		t.Error("unexpected Direction iota values")
	}
}

func TestEventTypeValues(t *testing.T) {
	if NodeCreated != 0 || NodeUpdated != 1 || NodeDeleted != 2 || EdgeCreated != 3 || EdgeDeleted != 4 {
		t.Error("unexpected EventType iota values")
	}
}

func TestGraphAddNode(t *testing.T) {
	g := New()
	g.AddNode("n1", "Entity", map[string]any{"Name": "Payer"})
	n := g.GetNode("n1")
	if n == nil {
		t.Fatal("GetNode returned nil")
	}
	if n.Label != "Entity" {
		t.Errorf("label = %q, want Entity", n.Label)
	}
	if n.Props["Name"] != "Payer" {
		t.Errorf("Name = %v, want Payer", n.Props["Name"])
	}
}

func TestGraphAddEdge(t *testing.T) {
	g := New()
	g.AddNode("m1", "Module", nil)
	g.AddNode("e1", "Entity", nil)
	g.AddEdge("edge1", "m1", "e1", "HAS_ENTITY", nil)

	neighbors := g.Neighbors("m1", "HAS_ENTITY")
	if len(neighbors) != 1 {
		t.Fatalf("Neighbors returned %d, want 1", len(neighbors))
	}
	if neighbors[0].ID != "e1" {
		t.Errorf("neighbor ID = %q, want e1", neighbors[0].ID)
	}
}

func TestGraphRemoveNode(t *testing.T) {
	g := New()
	g.AddNode("e1", "Entity", nil)
	g.AddNode("a1", "Attribute", nil)
	g.AddEdge("edge1", "e1", "a1", "HAS_ATTRIBUTE", nil)

	g.RemoveNode("e1")
	if g.GetNode("e1") != nil {
		t.Error("e1 should be nil after RemoveNode")
	}
	edges := g.Edges("e1", Outbound)
	if len(edges) != 0 {
		t.Errorf("expected 0 edges after node removal, got %d", len(edges))
	}
}

func TestGraphApplyEvents(t *testing.T) {
	g := New()
	g.Apply([]Event{
		{Type: NodeCreated, Node: &Node{ID: "e1", Label: "Entity"}},
		{Type: NodeCreated, Node: &Node{ID: "a1", Label: "Attribute"}},
		{Type: EdgeCreated, Edge: &Edge{ID: "edge1", From: "e1", To: "a1", Type: "HAS_ATTRIBUTE"}},
	})
	if g.GetNode("e1") == nil {
		t.Fatal("node e1 not found after Apply")
	}
	if g.GetNode("a1") == nil {
		t.Fatal("node a1 not found after Apply")
	}
}

func TestGraphFindNodesByLabel(t *testing.T) {
	g := New()
	g.AddNode("e1", "Entity", nil)
	g.AddNode("e2", "Entity", nil)
	g.AddNode("m1", "Microflow", nil)
	entities := g.FindNodes("Entity", nil)
	if len(entities) != 2 {
		t.Errorf("FindNodes(Entity) returned %d, want 2", len(entities))
	}
}

func TestGraphFindNodesByProps(t *testing.T) {
	g := New()
	g.AddNode("e1", "Entity", map[string]any{"Name": "Payer", "Module": "MyModule"})
	g.AddNode("e2", "Entity", map[string]any{"Name": "Claim", "Module": "MyModule"})
	g.AddNode("e3", "Entity", map[string]any{"Name": "Payer", "Module": "Other"})
	results := g.FindNodes("Entity", map[string]any{"Name": "Payer"})
	if len(results) != 2 {
		t.Errorf("FindNodes with Name=Payer returned %d, want 2", len(results))
	}
	results2 := g.FindNodes("Entity", map[string]any{"Name": "Payer", "Module": "MyModule"})
	if len(results2) != 1 {
		t.Errorf("FindNodes with Name=Payer+Module=MyModule returned %d, want 1", len(results2))
	}
}

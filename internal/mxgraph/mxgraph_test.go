package mxgraph

import (
	"bytes"
	"context"
	"io"
	"testing"
)

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

func TestGraphRemoveNodeCleansAdjacency(t *testing.T) {
	g := New()
	g.AddNode("e1", "Entity", nil)
	g.AddNode("a1", "Attribute", nil)
	g.AddNode("a2", "Attribute", nil)
	g.AddEdge("edge1", "e1", "a1", "HAS_ATTRIBUTE", nil)
	g.AddEdge("edge2", "e1", "a2", "HAS_ATTRIBUTE", nil)

	g.RemoveNode("e1")

	// a1's inEdges should not contain e1 after removal
	edges := g.Edges("a1", Inbound)
	if len(edges) != 0 {
		t.Errorf("a1 inbound edges after e1 removal: got %d, want 0", len(edges))
	}
	// a2's inEdges should not contain e1 after removal
	edges = g.Edges("a2", Inbound)
	if len(edges) != 0 {
		t.Errorf("a2 inbound edges after e1 removal: got %d, want 0", len(edges))
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

func buildTestGraph() *Graph {
	g := New()
	g.AddNode("m1", "Module", map[string]any{"Name": "MyModule"})
	g.AddNode("e1", "Entity", map[string]any{"Name": "Payer"})
	g.AddNode("a1", "Attribute", map[string]any{"Name": "Status"})
	g.AddNode("mf1", "Microflow", map[string]any{"Name": "ACT_Process"})
	g.AddEdge("edge1", "m1", "e1", "HAS_ENTITY", nil)
	g.AddEdge("edge2", "e1", "a1", "HAS_ATTRIBUTE", nil)
	g.AddEdge("edge3", "a1", "mf1", "USED_IN_MICROFLOW", nil)
	return g
}

func TestFindPathSchemas(t *testing.T) {
	g := buildTestGraph()
	schemas := g.FindPathSchemas("e1", "mf1", 5)
	if len(schemas) == 0 {
		t.Fatal("FindPathSchemas returned 0 schemas")
	}
	found := false
	for _, s := range schemas {
		if s.Label == "Entity→HAS_ATTRIBUTE→Attribute→USED_IN_MICROFLOW→Microflow" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected path Entity→Attribute→Microflow, got schemas: %v", schemas)
	}
}

func TestExplorePath(t *testing.T) {
	g := buildTestGraph()
	schemas := g.FindPathSchemas("e1", "mf1", 5)
	if len(schemas) == 0 {
		t.Fatal("no schemas found")
	}
	path := g.ExplorePath("e1", schemas[0])
	if len(path) < 2 {
		t.Fatalf("ExplorePath returned %d nodes, want >= 2", len(path))
	}
	if path[0].Node.ID != "e1" {
		t.Errorf("first node = %q, want e1", path[0].Node.ID)
	}
	if path[len(path)-1].Node.ID != "mf1" {
		t.Errorf("last node = %q, want mf1", path[len(path)-1].Node.ID)
	}
}

func TestTraverse(t *testing.T) {
	g := buildTestGraph()
	results := g.Traverse("m1", "HAS_ENTITY", 1)
	if len(results) == 0 {
		t.Fatal("Traverse returned 0 results")
	}
	if results[0].ID != "e1" {
		t.Errorf("first result = %q, want e1", results[0].ID)
	}
}

func TestTraverseDepth(t *testing.T) {
	g := buildTestGraph()
	results := g.Traverse("m1", "HAS_ENTITY", 2)
	if len(results) != 1 {
		t.Errorf("expected 1 result (e1), got %d", len(results))
	}
}

func TestSnapshotRoundtrip(t *testing.T) {
	g := buildTestGraph()
	data, err := MarshalSnapshot(g)
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}
	g2, err := UnmarshalSnapshot(data)
	if err != nil {
		t.Fatalf("UnmarshalSnapshot: %v", err)
	}
	if g2.GetNode("e1") == nil {
		t.Error("e1 missing after roundtrip")
	}
	if len(g2.Neighbors("e1")) != 1 {
		t.Errorf("expected 1 neighbor, got %d", len(g2.Neighbors("e1")))
	}
}

func TestDeltaAppendReplay(t *testing.T) {
	var buf bytes.Buffer
	w := NewDeltaWriter(&buf)
	if err := w.WriteEvent(Event{Type: NodeCreated, Node: &Node{ID: "n1", Label: "Entity"}}); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteEvent(Event{Type: NodeCreated, Node: &Node{ID: "n2", Label: "Attribute"}}); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteEvent(Event{Type: EdgeCreated, Edge: &Edge{ID: "e1", From: "n1", To: "n2", Type: "HAS_ATTRIBUTE"}}); err != nil {
		t.Fatal(err)
	}
	w.Close()

	g := New()
	r := NewDeltaReader(&buf)
	for {
		ev, err := r.ReadEvent()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("ReadEvent: %v", err)
		}
		g.Apply([]Event{ev})
	}
	if g.GetNode("n1") == nil || g.GetNode("n2") == nil {
		t.Error("nodes not restored after delta replay")
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

type testAdapter struct {
	name   string
	schema *GraphSchema
	events []Event
}

func (a *testAdapter) Name() string                     { return a.name }
func (a *testAdapter) Schema() *GraphSchema              { return a.schema }
func (a *testAdapter) Build(ctx context.Context, sink EventSink) error {
	return sink.Emit(a.events)
}
func (a *testAdapter) Watch(ctx context.Context, sink EventSink) (func(), error) {
	return func() {}, nil
}

func TestIndexManagerBuild(t *testing.T) {
	m := NewIndexManager()
	m.RegisterAdapter(&testAdapter{
		name:   "test",
		schema: &GraphSchema{NodeLabels: []Label{"Entity"}},
		events: []Event{
			{Type: NodeCreated, Node: &Node{ID: "e1", Label: "Entity"}},
		},
	})

	ctx := context.Background()
	if err := m.BuildAll(ctx); err != nil {
		t.Fatalf("BuildAll: %v", err)
	}

	n := m.Query().GetNode("e1")
	if n == nil {
		t.Fatal("e1 not found after BuildAll")
	}
}

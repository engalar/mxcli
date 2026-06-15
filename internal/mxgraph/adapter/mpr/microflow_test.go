package mpr

import (
	"context"
	"testing"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
)

func TestMicroflowAdapter_Schema(t *testing.T) {
	a := &MicroflowAdapter{}
	s := a.Schema()
	if s == nil {
		t.Fatal("Schema() returned nil")
	}
	labels := map[mxgraph.Label]bool{}
	for _, l := range s.NodeLabels {
		labels[l] = true
	}
	for _, want := range []mxgraph.Label{"Microflow", "Nanoflow"} {
		if !labels[want] {
			t.Errorf("Schema missing label %q", want)
		}
	}
	relTypes := map[mxgraph.RelType]bool{}
	for _, et := range s.EdgeTypes {
		relTypes[et.Type] = true
	}
	for _, want := range []mxgraph.RelType{"CALLS", "CREATES", "RETRIEVES", "SHOWS_PAGE"} {
		if !relTypes[want] {
			t.Errorf("Schema missing edge type %q", want)
		}
	}
}

func TestMicroflowAdapter_Name(t *testing.T) {
	a := &MicroflowAdapter{}
	if a.Name() != "microflow" {
		t.Errorf("Name() = %q, want microflow", a.Name())
	}
}

// TestMicroflowAdapter_BuildRealMPR verifies the nested BSON traversal against a
// real project: it must produce Microflow nodes and at least one reference edge.
func TestMicroflowAdapter_BuildRealMPR(t *testing.T) {
	mprPath := findTestMPR(t)
	if mprPath == "" {
		t.Skip("no test MPR found")
	}
	m, err := modelsdk.Open(mprPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer m.Close()

	a := &MicroflowAdapter{Model: m}
	sink := &recordingSink{}
	if err := a.Build(context.Background(), sink); err != nil {
		t.Fatalf("Build: %v", err)
	}

	var mfNodes int
	edgeCounts := map[mxgraph.RelType]int{}
	for _, ev := range sink.events {
		switch {
		case ev.Type == mxgraph.NodeCreated && (ev.Node.Label == "Microflow" || ev.Node.Label == "Nanoflow"):
			mfNodes++
		case ev.Type == mxgraph.EdgeCreated:
			edgeCounts[ev.Edge.Type]++
		}
	}
	t.Logf("microflow/nanoflow nodes=%d edges=%v", mfNodes, edgeCounts)

	if mfNodes == 0 {
		t.Fatal("expected at least one Microflow/Nanoflow node")
	}
	totalEdges := 0
	for _, c := range edgeCounts {
		totalEdges += c
	}
	if totalEdges == 0 {
		t.Error("expected at least one reference edge (CALLS/CREATES/RETRIEVES/SHOWS_PAGE) from real MPR")
	}
	// Every emitted edge must carry a non-empty qualified-name target.
	for _, ev := range sink.events {
		if ev.Type == mxgraph.EdgeCreated && ev.Edge.To == "" {
			t.Errorf("edge %s has empty To target", ev.Edge.Type)
		}
	}
}

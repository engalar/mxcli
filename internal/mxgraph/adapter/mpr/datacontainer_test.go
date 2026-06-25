package mpr

import (
	"context"
	"testing"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
)

func TestDataContainerAdapter_Schema(t *testing.T) {
	t.Parallel()
	a := &DataContainerAdapter{}
	s := a.Schema()
	if s == nil {
		t.Fatal("Schema() returned nil")
	}
	labels := map[mxgraph.Label]bool{}
	for _, l := range s.NodeLabels {
		labels[l] = true
	}
	if !labels["DataContainer"] {
		t.Errorf("Schema missing label DataContainer")
	}
	relTypes := map[mxgraph.RelType]bool{}
	for _, et := range s.EdgeTypes {
		relTypes[et.Type] = true
	}
	for _, want := range []mxgraph.RelType{"HAS_DATA_CONTAINER", "HAS_DATASOURCE_ENTITY", "HAS_DATASOURCE_MICROFLOW", "HAS_SELECTION_CONTEXT"} {
		if !relTypes[want] {
			t.Errorf("Schema missing edge type %q", want)
		}
	}
}

func TestDataContainerAdapter_Name(t *testing.T) {
	t.Parallel()
	if (&DataContainerAdapter{}).Name() != "datacontainer" {
		t.Errorf("Name() = %q, want datacontainer", (&DataContainerAdapter{}).Name())
	}
}

func TestDataContainerAdapter_BuildRealMPR(t *testing.T) {
	t.Parallel()
	mprPath := findTestMPR(t)
	if mprPath == "" {
		t.Skip("no test MPR found")
	}

	m, err := modelsdk.Open(mprPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer m.Close()

	a := &DataContainerAdapter{
		Source: &ModelsdkUnitSource{Model: m},
		Model:  m,
	}
	sink := &recordingSink{}
	if err := a.Build(context.Background(), sink); err != nil {
		t.Fatalf("Build: %v", err)
	}

	var dcNodes int
	edgeCounts := map[mxgraph.RelType]int{}
	for _, ev := range sink.events {
		switch {
		case ev.Type == mxgraph.NodeCreated && ev.Node.Label == "DataContainer":
			dcNodes++
		case ev.Type == mxgraph.EdgeCreated:
			edgeCounts[ev.Edge.Type]++
		}
	}
	t.Logf("DataContainer nodes=%d edges=%v", dcNodes, edgeCounts)

	if dcNodes > 0 {
		t.Logf("First DataContainer props: %v", sink.events[0].Node.Props)
	}
}

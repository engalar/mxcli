package mpr

import (
	"context"
	"testing"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
)

func TestNavigationAdapter_Schema(t *testing.T) {
	a := &NavigationAdapter{}
	s := a.Schema()
	if s == nil {
		t.Fatal("Schema() returned nil")
	}
	labels := map[mxgraph.Label]bool{}
	for _, l := range s.NodeLabels {
		labels[l] = true
	}
	for _, want := range []mxgraph.Label{"NavigationProfile", "NavigationMenuItem"} {
		if !labels[want] {
			t.Errorf("Schema missing label %q", want)
		}
	}
	relTypes := map[mxgraph.RelType]bool{}
	for _, et := range s.EdgeTypes {
		relTypes[et.Type] = true
	}
	for _, want := range []mxgraph.RelType{"HAS_PROFILE", "HAS_MENU_ITEM", "HAS_CHILD_ITEM", "TARGETS_PAGE", "TARGETS_MICROFLOW", "HAS_LOGIN_PAGE", "HAS_NOT_FOUND_PAGE", "HAS_OFFLINE_ENTITY"} {
		if !relTypes[want] {
			t.Errorf("Schema missing edge type %q", want)
		}
	}
}

func TestNavigationAdapter_Name(t *testing.T) {
	if (&NavigationAdapter{}).Name() != "navigation" {
		t.Errorf("Name() = %q, want navigation", (&NavigationAdapter{}).Name())
	}
}

func TestNavigationAdapter_BuildRealMPR(t *testing.T) {
	mprPath := findTestMPR(t)
	if mprPath == "" {
		t.Skip("no test MPR found")
	}

	m, err := modelsdk.Open(mprPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer m.Close()

	a := &NavigationAdapter{
		Source: &ModelsdkUnitSource{Model: m},
	}
	sink := &recordingSink{}
	if err := a.Build(context.Background(), sink); err != nil {
		t.Fatalf("Build: %v", err)
	}

	var navProfiles int
	var navMenuItems int
	edgeCounts := map[mxgraph.RelType]int{}
	for _, ev := range sink.events {
		switch {
		case ev.Type == mxgraph.NodeCreated && ev.Node.Label == "NavigationProfile":
			navProfiles++
		case ev.Type == mxgraph.NodeCreated && ev.Node.Label == "NavigationMenuItem":
			navMenuItems++
		case ev.Type == mxgraph.EdgeCreated:
			edgeCounts[ev.Edge.Type]++
		}
	}
	t.Logf("NavigationProfile nodes=%d NavigationMenuItem nodes=%d edges=%v", navProfiles, navMenuItems, edgeCounts)

	if navProfiles == 0 {
		t.Fatal("expected at least one NavigationProfile node")
	}
}

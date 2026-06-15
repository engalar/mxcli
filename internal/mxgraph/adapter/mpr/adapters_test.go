package mpr

import (
	"context"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
)

func TestPageAdapter_Schema(t *testing.T) {
	a := &PageAdapter{}
	s := a.Schema()
	labels := map[mxgraph.Label]bool{}
	for _, l := range s.NodeLabels {
		labels[l] = true
	}
	for _, want := range []mxgraph.Label{"Page", "Layout", "Snippet"} {
		if !labels[want] {
			t.Errorf("PageAdapter.Schema missing label %q", want)
		}
	}
}

func TestPageAdapter_Name(t *testing.T) {
	if (&PageAdapter{}).Name() != "page" {
		t.Errorf("PageAdapter.Name() = %q, want page", (&PageAdapter{}).Name())
	}
}

func TestSecurityAdapter_Schema(t *testing.T) {
	a := &SecurityAdapter{}
	s := a.Schema()
	labels := map[mxgraph.Label]bool{}
	for _, l := range s.NodeLabels {
		labels[l] = true
	}
	for _, want := range []mxgraph.Label{"UserRole", "ModuleRole"} {
		if !labels[want] {
			t.Errorf("SecurityAdapter.Schema missing label %q", want)
		}
	}
}

func TestSecurityAdapter_Name(t *testing.T) {
	if (&SecurityAdapter{}).Name() != "security" {
		t.Errorf("SecurityAdapter.Name() = %q, want security", (&SecurityAdapter{}).Name())
	}
}

func TestEnumerationAdapter_Schema(t *testing.T) {
	a := &EnumerationAdapter{}
	s := a.Schema()
	labels := map[mxgraph.Label]bool{}
	for _, l := range s.NodeLabels {
		labels[l] = true
	}
	for _, want := range []mxgraph.Label{"Enumeration", "EnumValue"} {
		if !labels[want] {
			t.Errorf("EnumerationAdapter.Schema missing label %q", want)
		}
	}
}

func TestEnumerationAdapter_Name(t *testing.T) {
	if (&EnumerationAdapter{}).Name() != "enumeration" {
		t.Errorf("EnumerationAdapter.Name() = %q, want enumeration", (&EnumerationAdapter{}).Name())
	}
}

// TestPageEnumAdapters_BuildRealMPR verifies the page and enumeration adapters
// produce nodes (and page layout edges) against a real project.
func TestPageEnumAdapters_BuildRealMPR(t *testing.T) {
	mprPath := findTestMPR(t)
	if mprPath == "" {
		t.Skip("no test MPR found")
	}
	m, err := modelsdk.Open(mprPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer m.Close()

	countNodes := func(a mxgraph.IndexAdapter) (map[mxgraph.Label]int, map[mxgraph.RelType]int) {
		sink := &recordingSink{}
		if err := a.Build(context.Background(), sink); err != nil {
			t.Fatalf("%s Build: %v", a.Name(), err)
		}
		nodes := map[mxgraph.Label]int{}
		edges := map[mxgraph.RelType]int{}
		for _, ev := range sink.events {
			if ev.Type == mxgraph.NodeCreated {
				nodes[ev.Node.Label]++
			}
			if ev.Type == mxgraph.EdgeCreated {
				edges[ev.Edge.Type]++
			}
		}
		return nodes, edges
	}

	pNodes, pEdges := countNodes(&PageAdapter{Model: m})
	t.Logf("page adapter nodes=%v edges=%v", pNodes, pEdges)
	if pNodes["Page"] == 0 {
		t.Error("expected at least one Page node")
	}

	eNodes, eEdges := countNodes(&EnumerationAdapter{Model: m})
	t.Logf("enum adapter nodes=%v edges=%v", eNodes, eEdges)
	if eNodes["Enumeration"] == 0 {
		t.Error("expected at least one Enumeration node")
	}

	sNodes, _ := countNodes(&SecurityAdapter{Model: m})
	t.Logf("security adapter nodes=%v", sNodes)
}

// TestDomainModelAdapter_EntityHasModuleAndQN verifies that Entity nodes built
// from a real MPR carry the derived Module and QualifiedName props that
// graphcatalog's FindNodes filters depend on.
func TestDomainModelAdapter_EntityHasModuleAndQN(t *testing.T) {
	mprPath := findTestMPR(t)
	if mprPath == "" {
		t.Skip("no test MPR found")
	}
	m, err := modelsdk.Open(mprPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer m.Close()

	mg := mxgraph.NewIndexManager()
	mg.RegisterAdapter(&DomainModelAdapter{Model: m})
	if err := mg.BuildAll(context.Background()); err != nil {
		t.Fatalf("BuildAll: %v", err)
	}

	entities := mg.Query().FindNodes("Entity", nil)
	if len(entities) == 0 {
		t.Fatal("no Entity nodes in graph")
	}

	e := entities[0]
	module, _ := e.Props["Module"].(string)
	qn, _ := e.Props["QualifiedName"].(string)
	t.Logf("sample Entity: Module=%q QualifiedName=%q", module, qn)

	if module == "" {
		t.Errorf("Entity node %s missing Module prop", e.ID)
	}
	if qn == "" || !strings.Contains(qn, ".") {
		t.Errorf("Entity node %s QualifiedName=%q, want a dotted Module.Entity name", e.ID, qn)
	}
	if module != "" && qn != "" && !strings.HasPrefix(qn, module+".") {
		t.Errorf("QualifiedName %q does not start with Module %q", qn, module)
	}

	// The derived Module prop must actually drive FindNodes filtering.
	filtered := mg.Query().FindNodes("Entity", map[string]any{"Module": module})
	if len(filtered) == 0 {
		t.Errorf("FindNodes filtered by Module=%q returned 0, derived prop not indexed", module)
	}
}

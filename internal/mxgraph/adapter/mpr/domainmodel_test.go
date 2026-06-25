package mpr

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func TestDomainModelAdapter_Schema(t *testing.T) {
	t.Parallel()
	a := &DomainModelAdapter{}
	s := a.Schema()
	if s == nil {
		t.Fatal("Schema() returned nil")
	}
	labels := map[mxgraph.Label]bool{}
	for _, l := range s.NodeLabels {
		labels[l] = true
	}
	for _, want := range []mxgraph.Label{"DomainModel", "Entity", "Attribute", "Association"} {
		if !labels[want] {
			t.Errorf("Schema missing label %q", want)
		}
	}
}

func TestDomainModelAdapter_Name(t *testing.T) {
	t.Parallel()
	a := &DomainModelAdapter{}
	if a.Name() != "domainmodel" {
		t.Errorf("Name() = %q, want domainmodel", a.Name())
	}
}

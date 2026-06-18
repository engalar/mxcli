// internal/fkg/concepts/page_test.go
package concepts_test

import (
	"context"
	"testing"

	"github.com/mendixlabs/mxcli/internal/fkg/concepts"
	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

// collectSink captures all events emitted by an adapter.
type collectSink struct{ events []mxgraph.Event }

func (s *collectSink) Emit(events []mxgraph.Event) error {
	s.events = append(s.events, events...)
	return nil
}

func nodeIDs(events []mxgraph.Event) map[string]mxgraph.Label {
	out := map[string]mxgraph.Label{}
	for _, e := range events {
		if e.Type == mxgraph.NodeCreated && e.Node != nil {
			out[string(e.Node.ID)] = e.Node.Label
		}
	}
	return out
}

func edgeRels(events []mxgraph.Event) map[string]bool {
	out := map[string]bool{}
	for _, e := range events {
		if e.Type == mxgraph.EdgeCreated && e.Edge != nil {
			out[string(e.Edge.Type)] = true
		}
	}
	return out
}

func TestPageAdapter_EmitsPageConceptNode(t *testing.T) {
	a := &concepts.PageAdapter{}
	sink := &collectSink{}
	if err := a.Build(context.Background(), sink); err != nil {
		t.Fatalf("Build: %v", err)
	}
	ids := nodeIDs(sink.events)
	if ids["page"] != concepts.LabelConcept {
		t.Errorf("expected node 'page' with label Concept, got %q", ids["page"])
	}
}

func TestPageAdapter_EmitsSyntaxAndSkillNodes(t *testing.T) {
	a := &concepts.PageAdapter{}
	sink := &collectSink{}
	_ = a.Build(context.Background(), sink)

	ids := nodeIDs(sink.events)
	if ids["syntax:page.create"] != concepts.LabelSyntaxFeature {
		t.Error("expected syntax:page.create node")
	}
	if ids["skill:create-page"] != concepts.LabelSkill {
		t.Error("expected skill:create-page node")
	}
}

func TestPageAdapter_EmitsExpectedEdgeTypes(t *testing.T) {
	a := &concepts.PageAdapter{}
	sink := &collectSink{}
	_ = a.Build(context.Background(), sink)

	rels := edgeRels(sink.events)
	for _, want := range []mxgraph.RelType{
		concepts.Specializes, concepts.HasSyntax, concepts.HasSkill, concepts.RelatedTo,
	} {
		if !rels[string(want)] {
			t.Errorf("expected edge type %q", want)
		}
	}
}

func TestPageAdapter_NameAndWatch(t *testing.T) {
	a := &concepts.PageAdapter{}
	if a.Name() == "" {
		t.Error("Name() must not be empty")
	}
	stop, err := a.Watch(context.Background(), &collectSink{})
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}
	stop() // must not panic
}

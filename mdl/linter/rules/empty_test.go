// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog/mock"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// mfWithActivities builds a gen Microflow whose object collection holds n
// objects (an empty collection means the microflow has no activities).
func mfWithActivities(id, name string, n int) *genMf.Microflow {
	mf := genMf.NewMicroflow()
	mf.SetID(element.ID(id))
	mf.SetName(name)
	oc := genMf.NewMicroflowObjectCollection()
	for i := 0; i < n; i++ {
		oc.AddObjects(genMf.NewActionActivity())
	}
	mf.SetObjectCollection(oc)
	return mf
}

// buildMicroflowContext wires a mock graph (microflow node listing) plus a deep
// reader returning the gen-typed bodies, keyed by ID.
func buildMicroflowContext(nodes []graphcatalog.MicroflowNode, bodies map[model.ID]*genMf.Microflow) (*mock.MockProjectGraph, *mockReader) {
	g := &mock.MockProjectGraph{
		MicroflowsFunc: func(module string) []graphcatalog.MicroflowNode {
			if module == "" {
				return nodes
			}
			var out []graphcatalog.MicroflowNode
			for _, n := range nodes {
				if n.Module == module {
					out = append(out, n)
				}
			}
			return out
		},
	}
	return g, &mockReader{microflows: bodies}
}

func TestEmptyMicroflowRule_NoViolations(t *testing.T) {
	nodes := []graphcatalog.MicroflowNode{microflowNode("id1", "ACT_Process", "MyModule")}
	bodies := map[model.ID]*genMf.Microflow{
		model.ID("id1"): mfWithActivities("id1", "ACT_Process", 3),
	}
	g, reader := buildMicroflowContext(nodes, bodies)
	violations := NewEmptyMicroflowRule().Check(newGraphContext(g, reader))

	if len(violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(violations))
	}
}

func TestEmptyMicroflowRule_DetectsEmpty(t *testing.T) {
	nodes := []graphcatalog.MicroflowNode{
		microflowNode("id1", "ACT_Process", "MyModule"),
		microflowNode("id2", "ACT_Other", "MyModule"),
	}
	bodies := map[model.ID]*genMf.Microflow{
		model.ID("id1"): mfWithActivities("id1", "ACT_Process", 0),
		model.ID("id2"): mfWithActivities("id2", "ACT_Other", 5),
	}
	g, reader := buildMicroflowContext(nodes, bodies)
	violations := NewEmptyMicroflowRule().Check(newGraphContext(g, reader))

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].RuleID != "MPR002" {
		t.Errorf("expected rule ID MPR002, got %s", violations[0].RuleID)
	}
	if violations[0].Location.DocumentName != "ACT_Process" {
		t.Errorf("expected document ACT_Process, got %s", violations[0].Location.DocumentName)
	}
}

func TestEmptyMicroflowRule_Metadata(t *testing.T) {
	r := NewEmptyMicroflowRule()
	if r.ID() != "MPR002" {
		t.Errorf("ID = %q, want MPR002", r.ID())
	}
	if r.Category() != "quality" {
		t.Errorf("Category = %q, want quality", r.Category())
	}
}

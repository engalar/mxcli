// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"fmt"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog/mock"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// entitySpec describes a synthetic entity for domain/security rule tests.
type entitySpec struct {
	id          string
	name        string
	module      string
	persistent  bool
	external    bool
	accessRules int
}

// buildEntityContext wires a mock graph (entity node listing) and a deep reader
// (gen-typed domain model carrying persistability + access-rule facts) from the
// given entity specs.
func buildEntityContext(specs []entitySpec) *mock.MockProjectGraph {
	nodes := make([]graphcatalog.EntityNode, 0, len(specs))
	for _, s := range specs {
		nodes = append(nodes, entityNode(s.id, s.name, s.module, s.external))
	}
	return &mock.MockProjectGraph{
		EntitiesFunc: func(module string) []graphcatalog.EntityNode {
			if module == "" {
				return nodes
			}
			var out []graphcatalog.EntityNode
			for _, n := range nodes {
				if n.Module == module {
					out = append(out, n)
				}
			}
			return out
		},
	}
}

func buildEntityReader(specs []entitySpec) *mockReader {
	var gens []*genDm.Entity
	for _, s := range specs {
		gens = append(gens, genEntity(s.id, s.name, s.persistent, s.accessRules))
	}
	return &mockReader{domainModels: []*genDm.DomainModel{domainModelWith(gens...)}}
}

func persistentSpecs(module string, n int) []entitySpec {
	specs := make([]entitySpec, 0, n)
	for i := 0; i < n; i++ {
		specs = append(specs, entitySpec{
			id: fmt.Sprintf("id%d", i), name: fmt.Sprintf("Entity%d", i),
			module: module, persistent: true, accessRules: 1,
		})
	}
	return specs
}

func TestDomainModelSizeRule_NoViolation(t *testing.T) {
	specs := persistentSpecs("MyModule", 10)
	ctx := newGraphContext(buildEntityContext(specs), buildEntityReader(specs))
	violations := NewDomainModelSizeRule().Check(ctx)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for 10 entities, got %d", len(violations))
	}
}

func TestDomainModelSizeRule_ExceedsThreshold(t *testing.T) {
	specs := persistentSpecs("BigModule", 20)
	ctx := newGraphContext(buildEntityContext(specs), buildEntityReader(specs))
	violations := NewDomainModelSizeRule().Check(ctx)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].RuleID != "MPR003" {
		t.Errorf("expected rule ID MPR003, got %s", violations[0].RuleID)
	}
}

func TestDomainModelSizeRule_NonPersistentIgnored(t *testing.T) {
	var specs []entitySpec
	for i := 0; i < 20; i++ {
		specs = append(specs, entitySpec{
			id: fmt.Sprintf("id%d", i), name: fmt.Sprintf("Entity%d", i),
			module: "MyModule", persistent: false,
		})
	}
	ctx := newGraphContext(buildEntityContext(specs), buildEntityReader(specs))
	violations := NewDomainModelSizeRule().Check(ctx)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for non-persistent entities, got %d", len(violations))
	}
}

func TestDomainModelSizeRule_ExactlyAtThreshold(t *testing.T) {
	specs := persistentSpecs("MyModule", DefaultMaxPersistentEntities)
	ctx := newGraphContext(buildEntityContext(specs), buildEntityReader(specs))
	violations := NewDomainModelSizeRule().Check(ctx)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations at threshold (%d entities), got %d", DefaultMaxPersistentEntities, len(violations))
	}
}

func TestDomainModelSizeRule_OneOverThreshold(t *testing.T) {
	count := DefaultMaxPersistentEntities + 1
	specs := persistentSpecs("MyModule", count)
	ctx := newGraphContext(buildEntityContext(specs), buildEntityReader(specs))
	violations := NewDomainModelSizeRule().Check(ctx)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation at %d entities, got %d", count, len(violations))
	}
}

func TestDomainModelSizeRule_Metadata(t *testing.T) {
	r := NewDomainModelSizeRule()
	if r.ID() != "MPR003" {
		t.Errorf("ID = %q, want MPR003", r.ID())
	}
	if r.Category() != "design" {
		t.Errorf("Category = %q, want design", r.Category())
	}
	if r.MaxPersistentEntities != DefaultMaxPersistentEntities {
		t.Errorf("MaxPersistentEntities = %d, want %d", r.MaxPersistentEntities, DefaultMaxPersistentEntities)
	}
}

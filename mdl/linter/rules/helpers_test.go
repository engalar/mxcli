// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog/mock"
	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// testMicroflow returns a synthetic MicroflowNode for use in walker unit tests.
func testMicroflow() graphcatalog.MicroflowNode {
	return graphcatalog.MicroflowNode{
		ID:            "mf1",
		Name:          "ACT_Process",
		QualifiedName: "MyModule.ACT_Process",
		Module:        "MyModule",
	}
}

// mockReader is a configurable linter.LintReader for rule unit tests. Embedding
// the interface makes unconfigured methods panic instead of returning zero
// values, surfacing accidental dependencies.
type mockReader struct {
	linter.LintReader
	microflows   map[model.ID]*genMf.Microflow
	domainModels []*genDm.DomainModel
}

func (r *mockReader) GetMicroflowGen(id model.ID) (*genMf.Microflow, error) {
	if r.microflows == nil {
		return nil, nil
	}
	return r.microflows[id], nil
}

func (r *mockReader) ListDomainModelsGen() ([]*genDm.DomainModel, error) {
	return r.domainModels, nil
}

// newGraphContext builds a LintContext backed by a MockProjectGraph configured
// with the given node listings, and an optional deep reader.
func newGraphContext(g *mock.MockProjectGraph, reader linter.LintReader) *linter.LintContext {
	return linter.NewLintContext(g, reader)
}

// entityNode is a convenience constructor for graphcatalog.EntityNode.
func entityNode(id, name, module string, external bool) graphcatalog.EntityNode {
	return graphcatalog.EntityNode{
		ID:            id,
		Name:          name,
		QualifiedName: module + "." + name,
		Module:        module,
		IsExternal:    external,
	}
}

// microflowNode is a convenience constructor for graphcatalog.MicroflowNode.
func microflowNode(id, name, module string) graphcatalog.MicroflowNode {
	return graphcatalog.MicroflowNode{
		ID:            id,
		Name:          name,
		QualifiedName: module + "." + name,
		Module:        module,
	}
}

// domainModelWith builds a gen DomainModel containing the given entities, for
// use as deep-reader fixture data (persistability / access-rule facts).
func domainModelWith(entities ...*genDm.Entity) *genDm.DomainModel {
	dm := genDm.NewDomainModel()
	for _, e := range entities {
		dm.AddEntities(e)
	}
	return dm
}

// genEntity builds a gen Entity with the given persistability and access-rule
// count, for use as deep-reader fixture data.
func genEntity(id, name string, persistable bool, accessRules int) *genDm.Entity {
	e := genDm.NewEntity()
	e.SetID(element.ID(id))
	e.SetName(name)

	g := genDm.NewNoGeneralization()
	g.SetPersistable(persistable)
	e.SetGeneralization(g)

	for i := 0; i < accessRules; i++ {
		e.AddAccessRules(genDm.NewAccessRule())
	}
	return e
}

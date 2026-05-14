// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
	"github.com/mendixlabs/mxcli/sdk/pages"
)

func TestDescribeMermaid_DomainModel_Mock(t *testing.T) {
	mod := mkModule("MyModule")

	// Build domain model first to get its ID, then create entities with dm as container.
	dm := &domainmodel.DomainModel{
		BaseElement: model.BaseElement{ID: nextID("dm")},
		ContainerID: mod.ID,
	}
	ent1 := mkEntity(dm.ID, "Customer")
	ent2 := mkEntity(dm.ID, "Order")
	dm.Entities = []*domainmodel.Entity{ent1, ent2}
	dm.Associations = []*domainmodel.Association{
		mkAssociation(mod.ID, "Order_Customer", ent2.ID, ent1.ID),
	}

	// Hierarchy: entity.ContainerID (dm.ID) -> mod.ID (module)
	// Entities are contained by the domain model; the domain model is contained by the module.
	h := mkHierarchy(mod)
	withContainer(h, dm.ID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelFunc: func(moduleID model.ID) (*domainmodel.DomainModel, error) {
			return dm, nil
		},
		ListDomainModelsFunc: func() ([]*domainmodel.DomainModel, error) {
			return []*domainmodel.DomainModel{dm}, nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, describeMermaid(ctx, "DOMAINMODEL", "MyModule"))

	out := buf.String()
	assertContainsStr(t, out, "erDiagram")
	assertContainsStr(t, out, "Customer")
	assertContainsStr(t, out, "Order")
	assertContainsStr(t, out, "Order_Customer")
}

// Stage 3.2.6.1: Microflow / Nanoflow mock tests removed. The dispatch
// in `describeMermaid` now routes to `microflowToMermaidGen` /
// `nanoflowToMermaidGen` (modelsdk/gen-typed), which read from
// `ctx.Microflows` / `ctx.Nanoflows` repositories instead of the legacy
// sdk/microflows-typed `ctx.Backend.List*` cache. Equivalent coverage
// lives in cmd_mermaid_gen_test.go (TestMicroflowToMermaidGen_*,
// TestNanoflowToMermaidGen_*).

func TestDescribeMermaid_Page_NotFound(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListPagesFunc:   func() ([]*pages.Page, error) { return nil, nil },
	}

	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertError(t, describeMermaid(ctx, "page", "MyModule.NoSuch"))
}

func TestDescribeMermaid_UnsupportedType(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}

	ctx, _ := newMockCtx(t, withBackend(mb))
	err := describeMermaid(ctx, "workflow", "MyModule.Something")
	assertError(t, err)
	assertContainsStr(t, fmt.Sprint(err), "not supported")
}

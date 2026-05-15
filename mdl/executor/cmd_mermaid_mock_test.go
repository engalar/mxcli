// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

func TestDescribeMermaid_DomainModel_Mock(t *testing.T) {
	mod := mkModule("MyModule")

	ent1 := mkEntityGen("Customer")
	ent2 := mkEntityGen("Order")
	dm := mkDomainModelGen(mod.ID, ent1, ent2)
	dm.AddAssociations(mkAssociationGen("Order_Customer", model.ID(ent2.ID()), model.ID(ent1.ID())))
	dmRepo := makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{mod.ID: {dm}})

	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h), withDomainModelsRepo(dmRepo))
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

	// Stage 3.3.5.B1 routes pageToMermaid through ctx.Pages — leaving
	// it nil makes listPagesWithContainerGen return empty, which is
	// enough for the not-found assertion.
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
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

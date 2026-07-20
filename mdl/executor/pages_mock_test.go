// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.5.C6 — page mock tests migrated to gen.
//
// SHOW PAGES / SNIPPETS / LAYOUTS now exercise listPagesGen /
// listSnippetsGen / listLayoutsGen via Recording*Repository fixtures
// wired into ctx.Pages / ctx.Layouts / ctx.Snippets (Stage 3.3.5.A0).
//
// describePage / describeSnippet / describeLayout still go through
// the legacy sdk-typed handlers — the gen-typed describe path is
// part of the deferred Phase A2-A5 widget formatter rebuild and
// retains its sdk-typed mock fixtures until then.

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	repostesting "github.com/mendixlabs/mxcli/mdl/repos/testing"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// makePagesRepo wires a RecordingPageRepository whose ListAll returns
// pgs and whose GetContainerUUID always returns containerID. Mirrors
// makeWorkflowsRepo (Stage 3.3.3.E0).
func makePagesRepo(pgs []*genPg.Page, containerID model.ID) *repostesting.RecordingPageRepository {
	return &repostesting.RecordingPageRepository{
		ListAllFunc:          func() ([]*genPg.Page, error) { return pgs, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) { return containerID, nil },
	}
}

func makeLayoutsRepo(lays []*genPg.Layout, containerID model.ID) *repostesting.RecordingLayoutRepository {
	return &repostesting.RecordingLayoutRepository{
		ListAllFunc:          func() ([]*genPg.Layout, error) { return lays, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) { return containerID, nil },
	}
}

func makeSnippetsRepo(snps []*genPg.Snippet, containerID model.ID) *repostesting.RecordingSnippetRepository {
	return &repostesting.RecordingSnippetRepository{
		ListAllFunc:          func() ([]*genPg.Snippet, error) { return snps, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) { return containerID, nil },
	}
}

// makePagesRepoMulti wires per-page container resolution by ID. Used
// for tests that mix pages from multiple modules.
func makePagesRepoMulti(pgs []*genPg.Page, containerByID map[element.ID]model.ID) *repostesting.RecordingPageRepository {
	return &repostesting.RecordingPageRepository{
		ListAllFunc: func() ([]*genPg.Page, error) { return pgs, nil },
		GetContainerUUIDFunc: func(id model.ID) (model.ID, error) {
			return containerByID[element.ID(id)], nil
		},
	}
}

func makeLayoutsRepoMulti(lays []*genPg.Layout, containerByID map[element.ID]model.ID) *repostesting.RecordingLayoutRepository {
	return &repostesting.RecordingLayoutRepository{
		ListAllFunc: func() ([]*genPg.Layout, error) { return lays, nil },
		GetContainerUUIDFunc: func(id model.ID) (model.ID, error) {
			return containerByID[element.ID(id)], nil
		},
	}
}

func makeSnippetsRepoMulti(snps []*genPg.Snippet, containerByID map[element.ID]model.ID) *repostesting.RecordingSnippetRepository {
	return &repostesting.RecordingSnippetRepository{
		ListAllFunc: func() ([]*genPg.Snippet, error) { return snps, nil },
		GetContainerUUIDFunc: func(id model.ID) (model.ID, error) {
			return containerByID[element.ID(id)], nil
		},
	}
}

func TestShowPages_Mock(t *testing.T) {
	mod := mkModule("MyModule")
	pg := mkPageGen(string(nextID("pg")), "Home")

	h := mkHierarchy(mod)

	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Pages = makePagesRepo([]*genPg.Page{pg}, mod.ID)
	rebuildDeps(ctx)
	assertNoError(t, listPagesGen(ctx, ""))

	out := buf.String()
	assertContainsStr(t, out, "MyModule.Home")
	assertContainsStr(t, out, "(1 pages)")
}

func TestShowPages_Mock_FilterByModule(t *testing.T) {
	mod1 := mkModule("Sales")
	mod2 := mkModule("HR")
	pg1 := mkPageGen(string(nextID("pg")), "OrderList")
	pg2 := mkPageGen(string(nextID("pg")), "EmployeeList")

	h := mkHierarchy(mod1, mod2)

	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Pages = makePagesRepoMulti(
		[]*genPg.Page{pg1, pg2},
		map[element.ID]model.ID{
			pg1.ID(): mod1.ID,
			pg2.ID(): mod2.ID,
		},
	)
	rebuildDeps(ctx)
	assertNoError(t, listPagesGen(ctx, "HR"))

	out := buf.String()
	assertNotContainsStr(t, out, "Sales.OrderList")
	assertContainsStr(t, out, "HR.EmployeeList")
}

func TestShowSnippets_Mock_FilterByModule(t *testing.T) {
	mod1 := mkModule("Sales")
	mod2 := mkModule("HR")
	snp1 := mkSnippetGen(string(nextID("snp")), "OrderHeader")
	snp2 := mkSnippetGen(string(nextID("snp")), "EmployeeCard")

	h := mkHierarchy(mod1, mod2)

	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Snippets = makeSnippetsRepoMulti(
		[]*genPg.Snippet{snp1, snp2},
		map[element.ID]model.ID{
			snp1.ID(): mod1.ID,
			snp2.ID(): mod2.ID,
		},
	)
	rebuildDeps(ctx)
	assertNoError(t, listSnippetsGen(ctx, "HR"))

	out := buf.String()
	assertNotContainsStr(t, out, "Sales.OrderHeader")
	assertContainsStr(t, out, "HR.EmployeeCard")
}

func TestShowLayouts_Mock_FilterByModule(t *testing.T) {
	mod1 := mkModule("Sales")
	mod2 := mkModule("HR")
	lay1 := mkLayoutGen(string(nextID("lay")), "SalesLayout")
	lay2 := mkLayoutGen(string(nextID("lay")), "HRLayout")

	h := mkHierarchy(mod1, mod2)

	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Layouts = makeLayoutsRepoMulti(
		[]*genPg.Layout{lay1, lay2},
		map[element.ID]model.ID{
			lay1.ID(): mod1.ID,
			lay2.ID(): mod2.ID,
		},
	)
	rebuildDeps(ctx)
	assertNoError(t, listLayoutsGen(ctx, "HR"))

	out := buf.String()
	assertNotContainsStr(t, out, "Sales.SalesLayout")
	assertContainsStr(t, out, "HR.HRLayout")
}

// describePage / describeSnippet / describeLayout were migrated to the
// gen-typed listPagesWithContainerGen family in Stage 3.3.5.C7b.
// Leaving the gen repos unset (ctx.Pages == nil etc.) makes the lookup
// return zero pairs, exercising the not-found path without exposing
// sdk-typed Page/Snippet/Layout fixtures here.

func TestDescribePage_Mock_NotFound(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}

	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	_ = mod
	assertError(t, describePage(ctx, ast.QualifiedName{Module: "MyModule", Name: "NonExistent"}))
}

func TestDescribeSnippet_Mock_NotFound(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}

	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	_ = mod
	assertError(t, describeSnippet(ctx, ast.QualifiedName{Module: "MyModule", Name: "NonExistent"}))
}

func TestDescribeLayout_Mock_NotFound(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}

	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	_ = mod
	assertError(t, describeLayout(ctx, ast.QualifiedName{Module: "MyModule", Name: "NonExistent"}))
}

func TestShowSnippets_Mock(t *testing.T) {
	mod := mkModule("MyModule")
	snp := mkSnippetGen(string(nextID("snp")), "Header")

	h := mkHierarchy(mod)

	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Snippets = makeSnippetsRepo([]*genPg.Snippet{snp}, mod.ID)
	rebuildDeps(ctx)
	assertNoError(t, listSnippetsGen(ctx, ""))

	out := buf.String()
	assertContainsStr(t, out, "MyModule.Header")
	assertContainsStr(t, out, "(1 snippets)")
}

func TestShowLayouts_Mock(t *testing.T) {
	mod := mkModule("MyModule")
	lay := mkLayoutGen(string(nextID("lay")), "Atlas_Default")

	h := mkHierarchy(mod)

	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Layouts = makeLayoutsRepo([]*genPg.Layout{lay}, mod.ID)
	rebuildDeps(ctx)
	assertNoError(t, listLayoutsGen(ctx, ""))

	out := buf.String()
	assertContainsStr(t, out, "MyModule.Atlas_Default")
	assertContainsStr(t, out, "(1 layouts)")
}

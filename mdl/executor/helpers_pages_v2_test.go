// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.5.A0 unit tests for the pages cache helpers. Mirrors the
// workflows helper tests; uses Recording*Repository fixtures from
// mdl/repos/testing rather than a real MPR (those round-trips live in
// mdl/backend/mpr/repos/*_test.go).

package executor

import (
	"errors"
	"testing"

	repostesting "github.com/mendixlabs/mxcli/mdl/repos/testing"
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// --- pages ---

func newPagesCacheTestContext(t *testing.T, pages []*genPg.Page, containerID string) *ExecContext {
	t.Helper()
	repo := &repostesting.RecordingPageRepository{
		ListAllFunc: func() ([]*genPg.Page, error) { return pages, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) {
			return model.ID(containerID), nil
		},
	}
	ctx := &ExecContext{
		Pages: repo,
	}
	ctx.ensureCache()
	return ctx
}

func TestListPagesWithContainerGen_NilCtxReturnsNil(t *testing.T) {
	got, err := listPagesWithContainerGen(nil)
	if err != nil {
		t.Fatalf("listPagesWithContainerGen(nil): %v", err)
	}
	if got != nil {
		t.Errorf("expected nil result for nil ctx, got %v", got)
	}
}

func TestListPagesWithContainerGen_NoRepoReturnsEmpty(t *testing.T) {
	ctx := &ExecContext{}
	ctx.ensureCache()
	got, err := listPagesWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listPagesWithContainerGen(): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 entries with no repo, got %d", len(got))
	}
}

func TestListPagesWithContainerGen_CachesAcrossCalls(t *testing.T) {
	p1 := genPg.NewPage()
	p1.SetID("ID1")
	p1.SetName("P1")

	ctx := newPagesCacheTestContext(t, []*genPg.Page{p1}, "MOD1")

	first, err := listPagesWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if len(first) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(first))
	}
	if first[0].Elem.Name() != "P1" {
		t.Errorf("Name() = %q, want P1", first[0].Elem.Name())
	}
	if first[0].ContainerID != "MOD1" {
		t.Errorf("ContainerID = %q, want MOD1", first[0].ContainerID)
	}

	second, err := listPagesWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if len(first) != len(second) || &first[0] != &second[0] {
		t.Errorf("cache must return identical slice on repeat call")
	}
}

func TestListPagesWithContainerGen_PropagatesListError(t *testing.T) {
	wantErr := errors.New("boom")
	repo := &repostesting.RecordingPageRepository{
		ListAllFunc: func() ([]*genPg.Page, error) { return nil, wantErr },
	}
	ctx := &ExecContext{
		Pages: repo,
	}
	ctx.ensureCache()
	_, err := listPagesWithContainerGen(ctx)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected error to wrap %v, got %v", wantErr, err)
	}
}

// --- layouts ---

func TestListLayoutsWithContainerGen_CachesAcrossCalls(t *testing.T) {
	l1 := genPg.NewLayout()
	l1.SetID("LID1")
	l1.SetName("L1")
	repo := &repostesting.RecordingLayoutRepository{
		ListAllFunc: func() ([]*genPg.Layout, error) { return []*genPg.Layout{l1}, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) {
			return model.ID("MOD1"), nil
		},
	}
	ctx := &ExecContext{
		Layouts: repo,
	}
	ctx.ensureCache()

	first, err := listLayoutsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if len(first) != 1 || first[0].Elem.Name() != "L1" {
		t.Fatalf("got %v, want one entry named L1", first)
	}

	second, err := listLayoutsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if &first[0] != &second[0] {
		t.Errorf("layout cache did not return identical slice on repeat call")
	}
}

// --- snippets ---

func TestListSnippetsWithContainerGen_CachesAcrossCalls(t *testing.T) {
	s1 := genPg.NewSnippet()
	s1.SetID("SID1")
	s1.SetName("S1")
	repo := &repostesting.RecordingSnippetRepository{
		ListAllFunc: func() ([]*genPg.Snippet, error) { return []*genPg.Snippet{s1}, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) {
			return model.ID("MOD1"), nil
		},
	}
	ctx := &ExecContext{
		Snippets: repo,
	}
	ctx.ensureCache()

	first, err := listSnippetsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if len(first) != 1 || first[0].Elem.Name() != "S1" {
		t.Fatalf("got %v, want one entry named S1", first)
	}

	second, err := listSnippetsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if &first[0] != &second[0] {
		t.Errorf("snippet cache did not return identical slice on repeat call")
	}
}

func TestInvalidatePagesGenCache_ClearsAllThree(t *testing.T) {
	page := genPg.NewPage()
	page.SetID("PA")
	page.SetName("PA")
	ctx := newPagesCacheTestContext(t, []*genPg.Page{page}, "MOD")
	if _, err := listPagesWithContainerGen(ctx); err != nil {
		t.Fatalf("warm pages: %v", err)
	}
	ctx.Cache.layoutsWithContainerGen = []ContainerWithGen[*genPg.Layout]{{}}
	ctx.Cache.snippetsWithContainerGen = []ContainerWithGen[*genPg.Snippet]{{}}

	invalidatePagesGenCache(ctx)
	if ctx.Cache.pagesWithContainerGen != nil ||
		ctx.Cache.layoutsWithContainerGen != nil ||
		ctx.Cache.snippetsWithContainerGen != nil {
		t.Errorf("invalidate must clear all three caches")
	}
}

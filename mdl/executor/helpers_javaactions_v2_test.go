// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"io"
	"testing"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
)

// newJavaActionsTestContext builds an ExecContext with JavaActions and
// JavaScriptActions wired from a fixture MPR. Mirrors
// newSecurityTestContext / newGenDescribeContext.
func newJavaActionsTestContext(t *testing.T) *ExecContext {
	t.Helper()
	w := openMprWriterForTest(t)

	path := w.ConcreteReader().Path()
	be, err := mprbackend.NewFromPath(path)
	if err != nil {
		t.Fatalf("mprbackend.NewFromPath(%s): %v", path, err)
	}
	t.Cleanup(func() { _ = be.Disconnect() })

	ctx := &ExecContext{
		Backend:   be,
		JavaActions: mprrepos.NewJavaActionRepository(w), JavaScriptActions: mprrepos.NewJavaScriptActionRepository(w),
		Output: io.Discard,
	}
	ctx.initRoles()
	ctx.ensureCache()
	return ctx
}

func TestListJavaActionsWithContainerGen_CachesAcrossCalls(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	list1, err := listJavaActionsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listJavaActionsWithContainerGen: %v", err)
	}
	list2, _ := listJavaActionsWithContainerGen(ctx)
	if len(list1) != len(list2) {
		t.Fatalf("cache produced different lengths: %d vs %d", len(list1), len(list2))
	}
	if len(list1) > 0 && &list1[0] != &list2[0] {
		t.Fatalf("cache must return same slice header on second call")
	}
}

func TestListJavaActionsWithContainerGen_ResolvesContainerUUID(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	list, err := listJavaActionsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listJavaActionsWithContainerGen: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("fixture returned no java actions")
	}
	for _, item := range list {
		if item.Elem.Name() == "" {
			t.Errorf("entry has empty Name: %+v", item)
		}
		if item.ContainerID == "" {
			t.Errorf("entry %s has empty ContainerID", item.Elem.Name())
		}
	}
}

func TestListJavaActionsWithContainerGen_NilCtxReturnsNil(t *testing.T) {
	list, err := listJavaActionsWithContainerGen(nil)
	if err != nil {
		t.Errorf("expected no error for nil ctx, got %v", err)
	}
	if list != nil {
		t.Errorf("expected nil list for nil ctx, got %v", list)
	}
}

func TestInvalidateJavaActionsCache_ClearsCache(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	if _, err := listJavaActionsWithContainerGen(ctx); err != nil {
		t.Fatalf("warm-up: %v", err)
	}
	if ctx.Cache.javaActionsWithContainerGen == nil {
		t.Fatal("warm-up did not populate cache")
	}
	invalidateJavaActionsCache(ctx)
	if ctx.Cache.javaActionsWithContainerGen != nil {
		t.Errorf("invalidate should clear cache, still got %d entries", len(ctx.Cache.javaActionsWithContainerGen))
	}
}

func TestListJavaScriptActionsWithContainerGen_CachesAcrossCalls(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	list1, err := listJavaScriptActionsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listJavaScriptActionsWithContainerGen: %v", err)
	}
	list2, _ := listJavaScriptActionsWithContainerGen(ctx)
	if len(list1) != len(list2) {
		t.Fatalf("cache produced different lengths: %d vs %d", len(list1), len(list2))
	}
}

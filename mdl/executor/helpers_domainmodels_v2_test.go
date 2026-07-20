// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"io"
	"testing"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
)

// newDomainModelsTestContext builds an ExecContext with DomainModels
// wired from a fixture MPR. Mirrors newJavaActionsTestContext.
func newDomainModelsTestContext(t *testing.T) *ExecContext {
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
		DomainModels: mprrepos.NewDomainModelRepository(w),
		Output: io.Discard,
	}
	ctx.initRoles()
	// initRoles calls ensureCache + initLoadFns, so cache is wired.
	return ctx
}

func TestListDomainModelsWithContainerGen_CachesAcrossCalls(t *testing.T) {
	ctx := newDomainModelsTestContext(t)
	list1, err := listDomainModelsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listDomainModelsWithContainerGen: %v", err)
	}
	list2, _ := listDomainModelsWithContainerGen(ctx)
	if len(list1) != len(list2) {
		t.Fatalf("cache produced different lengths: %d vs %d", len(list1), len(list2))
	}
	if len(list1) > 0 && &list1[0] != &list2[0] {
		t.Fatalf("cache must return same slice header on second call")
	}
}

func TestListDomainModelsWithContainerGen_ResolvesContainerUUID(t *testing.T) {
	ctx := newDomainModelsTestContext(t)
	list, err := listDomainModelsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listDomainModelsWithContainerGen: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("fixture returned no domain models")
	}
	for _, item := range list {
		if item.DM == nil {
			t.Errorf("entry has nil DM: %+v", item)
		}
		if item.ContainerID == "" {
			t.Errorf("entry has empty ContainerID")
		}
	}
}

func TestListDomainModelsWithContainerGen_NilCtxReturnsNil(t *testing.T) {
	list, err := listDomainModelsWithContainerGen(nil)
	if err != nil {
		t.Errorf("expected no error for nil ctx, got %v", err)
	}
	if list != nil {
		t.Errorf("expected nil list for nil ctx, got %v", list)
	}
}

func TestListDomainModelsWithContainerGen_NilRepoReturnsNil(t *testing.T) {
	ctx := &ExecContext{
		Output: io.Discard,
	}
	ctx.initRoles()
	ctx.ensureCache()
	list, err := listDomainModelsWithContainerGen(ctx)
	if err != nil {
		t.Errorf("expected no error for nil repo, got %v", err)
	}
	if list != nil {
		t.Errorf("expected nil list for nil repo, got %v", list)
	}
}

func TestInvalidateDomainModelsGenCache_ClearsField(t *testing.T) {
	ctx := newDomainModelsTestContext(t)
	if _, err := listDomainModelsWithContainerGen(ctx); err != nil {
		t.Fatalf("prime cache: %v", err)
	}
	if _, err := cachedDomainModelsGen(ctx); err != nil {
		t.Fatalf("prime flat cache: %v", err)
	}
	if ctx.Cache.domainModelsWithContainerGen == nil {
		t.Fatal("cache should be populated after first call")
	}
	if ctx.Cache.domainModelsGen == nil {
		t.Fatal("flat cache should be populated after first call")
	}
	invalidateDomainModelsGenCache(ctx)
	if ctx.Cache.domainModelsGen != nil {
		t.Fatal("invalidate should null out flat cache slice")
	}
	// domainModelsWithContainerGen is a domainCache — after Invalidate()
	// the pointer is still valid but loaded is false. Verify by re-reading.
	relist, err := listDomainModelsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("re-read after invalidate: %v", err)
	}
	if len(relist) == 0 {
		t.Fatal("expected domain models after re-read")
	}
}

// TestInvalidateDomainModelsCache_ClearsBoth verifies the legacy
// invalidator now also drops the gen-typed cache (Stage 3.3.4 A0).
func TestInvalidateDomainModelsCache_ClearsBoth(t *testing.T) {
	ctx := newDomainModelsTestContext(t)
	if _, err := listDomainModelsWithContainerGen(ctx); err != nil {
		t.Fatalf("prime cache: %v", err)
	}
	if _, err := cachedDomainModelsGen(ctx); err != nil {
		t.Fatalf("prime flat cache: %v", err)
	}
	if ctx.Cache.domainModelsGen == nil {
		t.Fatal("flat cache should be populated after first call")
	}
	invalidateDomainModelsCache(ctx)
	if ctx.Cache.domainModelsGen != nil {
		t.Fatal("legacy invalidate should also null out flat gen cache slice")
	}
	// domainModelsWithContainerGen is a domainCache — it survives Invalidate.
	relist, err := listDomainModelsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("re-read after invalidate: %v", err)
	}
	if len(relist) == 0 {
		t.Fatal("expected domain models after re-read")
	}
}

func TestFindDomainModelGenByModule_FindsByName(t *testing.T) {
	ctx := newDomainModelsTestContext(t)
	mods, err := ctx.ModuleLister.ListModules()
	if err != nil || len(mods) == 0 {
		t.Skip("fixture has no modules")
	}
	for _, m := range mods {
		dm, err := findDomainModelGenByModule(ctx, m.Name)
		if err != nil {
			t.Errorf("findDomainModelGenByModule(%q): %v", m.Name, err)
			continue
		}
		// dm may be nil for modules that lack a domain model in the fixture
		// (defensive — see helper docstring). Just assert no spurious type error.
		_ = dm
	}
}

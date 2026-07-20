// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	repostesting "github.com/mendixlabs/mxcli/mdl/repos/testing"
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// TestListMicroflowsWithContainerGen_CachesAcrossCalls verifies the
// session-cache contract: the first call resolves all container UUIDs,
// the second call returns the cached slice without re-invoking
// ListAll / GetContainerUUID. This is the foundation that prevents
// O(N²) regressions when 28 production callers fan out to the helper.
func TestListMicroflowsWithContainerGen_CachesAcrossCalls(t *testing.T) {
	mfA := mkMicroflowGen("A")
	mfB := mkMicroflowGen("B")
	repo := &repostesting.RecordingMicroflowRepository{
		ListAllFunc: func() ([]*genMf.Microflow, error) {
			return []*genMf.Microflow{mfA, mfB}, nil
		},
		GetContainerUUIDFunc: func(id model.ID) (model.ID, error) {
			return model.ID(string(id) + "-container"), nil
		},
	}
	ctx, _ := newMockCtx(t, withMicroflowsRepo(repo))
	// initRoles already wired the cache with stale deps (before repo was set).
	// Reset and re-init so load functions capture the repo-backed deps.
	ctx.Cache = newExecutorCache()
	ctx.Cache.initLoadFns(ctx.Deps)

	first, err := listMicroflowsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listMicroflowsWithContainerGen: %v", err)
	}
	if got, want := len(first), 2; got != want {
		t.Fatalf("len(first) = %d, want %d", got, want)
	}
	if first[0].ContainerUUID == "" || first[1].ContainerUUID == "" {
		t.Errorf("ContainerUUID not resolved: %+v", first)
	}

	listAllCalls1 := repo.ListedAll
	containerCalls1 := len(repo.GetContainerIDs)

	second, err := listMicroflowsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listMicroflowsWithContainerGen second call: %v", err)
	}
	if &second[0] != &first[0] {
		t.Errorf("cache miss on second call: returned a fresh slice, want the cached one")
	}

	if repo.ListedAll != listAllCalls1 {
		t.Errorf("ListAll re-invoked on cache hit: was %d, now %d", listAllCalls1, repo.ListedAll)
	}
	if len(repo.GetContainerIDs) != containerCalls1 {
		t.Errorf("GetContainerUUID re-invoked on cache hit: was %d, now %d", containerCalls1, len(repo.GetContainerIDs))
	}
}

// TestListMicroflowsWithContainerGen_InvalidationClearsCache verifies
// that invalidateMicroflowsCache (called from microflow/nanoflow
// create/drop paths) drops the cached slice so subsequent reads see
// fresh container linkage.
func TestListMicroflowsWithContainerGen_InvalidationClearsCache(t *testing.T) {
	mf := mkMicroflowGen("X")
	repo := &repostesting.RecordingMicroflowRepository{
		ListAllFunc: func() ([]*genMf.Microflow, error) {
			return []*genMf.Microflow{mf}, nil
		},
	}
	ctx, _ := newMockCtx(t, withMicroflowsRepo(repo))
	ctx.Cache = newExecutorCache()
	ctx.Cache.initLoadFns(ctx.Deps)

	if _, err := listMicroflowsWithContainerGen(ctx); err != nil {
		t.Fatalf("first call: %v", err)
	}
	preCalls := repo.ListedAll

	invalidateMicroflowsCache(ctx)

	if _, err := listMicroflowsWithContainerGen(ctx); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if repo.ListedAll != preCalls+1 {
		t.Errorf("invalidation did not force a re-read: ListAll calls went %d -> %d, want +1", preCalls, repo.ListedAll)
	}
}

// TestListNanoflowsWithContainerGen_CachesAcrossCalls mirrors the
// microflow test for the nanoflow helper.
func TestListNanoflowsWithContainerGen_CachesAcrossCalls(t *testing.T) {
	nfA := mkNanoflowGen("A")
	nfB := mkNanoflowGen("B")
	mfRepo := &repostesting.RecordingMicroflowRepository{
		GetContainerUUIDFunc: func(id model.ID) (model.ID, error) {
			return model.ID(string(id) + "-container"), nil
		},
	}
	nfRepo := &repostesting.RecordingNanoflowRepository{
		ListFunc: func(_ model.ID) ([]*genMf.Nanoflow, error) {
			return []*genMf.Nanoflow{nfA, nfB}, nil
		},
	}
	ctx, _ := newMockCtx(t, withMicroflowsRepo(mfRepo), withNanoflowsRepo(nfRepo))
	ctx.Cache = newExecutorCache()
	ctx.Cache.initLoadFns(ctx.Deps)

	first, err := listNanoflowsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listNanoflowsWithContainerGen: %v", err)
	}
	if got, want := len(first), 2; got != want {
		t.Fatalf("len(first) = %d, want %d", got, want)
	}

	listCalls1 := len(nfRepo.ListedModule)
	containerCalls1 := len(mfRepo.GetContainerIDs)

	if _, err := listNanoflowsWithContainerGen(ctx); err != nil {
		t.Fatalf("second call: %v", err)
	}

	if len(nfRepo.ListedModule) != listCalls1 {
		t.Errorf("Nanoflows.List re-invoked on cache hit: was %d, now %d", listCalls1, len(nfRepo.ListedModule))
	}
	if len(mfRepo.GetContainerIDs) != containerCalls1 {
		t.Errorf("GetContainerUUID re-invoked on cache hit: was %d, now %d", containerCalls1, len(mfRepo.GetContainerIDs))
	}
}

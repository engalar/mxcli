// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.4 A0: cache helpers for the domainmodel domain.
// Each Mendix module owns exactly one DomainModel unit
// ("DomainModels$DomainModel"); the helper iterates modules from the
// backend and pairs each module's DomainModel with the module ID as
// the container UUID. This mirrors the security helper shape rather
// than the javaactions ContainerWithGen pattern because the
// DomainModelRepository interface is intentionally minimal (no
// ListAll / GetContainerUUID — see mdl/repos/domainmodels.go).
//
// Per memory `feedback_executor_cache_pattern`: list calls in
// mdl/executor/ MUST go through this helper to avoid O(N^2) walks.

package executor

import (
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// DomainModelGenWithContainer pairs a gen-typed *DomainModel with its
// container UUID (the parent module ID).
type DomainModelGenWithContainer struct {
	DM          *genDm.DomainModel
	ContainerID model.ID
}

// listDomainModelsWithContainerGen returns every DomainModel unit in the
// project paired with its owning module ID. Caches on
// ctx.Cache.domainModelsWithContainerGen for the session.
//
// Uses ListAllWithContainerID for a single O(#domain_models) scan instead
// of one List(moduleID) call per module (which was O(N × all_units)).
func listDomainModelsWithContainerGen(ctx *ExecContext) ([]DomainModelGenWithContainer, error) {
	if ctx == nil || ctx.DomainModels == nil {
		return nil, nil
	}
	if ctx.Cache != nil && ctx.Cache.domainModelsWithContainerGen != nil {
		return ctx.Cache.domainModelsWithContainerGen, nil
	}

	// Build module ID set for filtering (ContainerID of a DomainModel IS its module ID).
	mods, err := ctx.ModuleLister.ListModules()
	if err != nil {
		return nil, err
	}
	moduleIDs := make(map[model.ID]bool, len(mods))
	for _, m := range mods {
		moduleIDs[m.ID] = true
	}

	pairs, err := ctx.DomainModels.ListAllWithContainerID()
	if err != nil {
		return nil, err
	}
	out := make([]DomainModelGenWithContainer, 0, len(pairs))
	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		// Only include DomainModels whose direct parent is a known module.
		if !moduleIDs[p.ContainerID] {
			continue
		}
		out = append(out, DomainModelGenWithContainer{DM: p.DM, ContainerID: p.ContainerID})
	}
	if ctx.Cache != nil {
		ctx.Cache.domainModelsWithContainerGen = out
	}
	return out, nil
}

// findDomainModelGenByModule returns the DomainModel for a given module
// name. Returns (nil, nil) when the module exists but has no domain
// model (defensive — Mendix invariant says each module has exactly
// one).
func findDomainModelGenByModule(ctx *ExecContext, moduleName string) (*genDm.DomainModel, error) {
	pairs, err := listDomainModelsWithContainerGen(ctx)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		return nil, nil
	}
	mods, err := ctx.ModuleLister.ListModules()
	if err != nil {
		return nil, err
	}
	wantContainer := model.ID("")
	for _, m := range mods {
		if m.Name == moduleName {
			wantContainer = m.ID
			break
		}
	}
	if wantContainer == "" {
		return nil, nil
	}
	for _, p := range pairs {
		if p.ContainerID == wantContainer {
			return p.DM, nil
		}
	}
	return nil, nil
}

// cachedDomainModelsGen returns the flat gen-typed DomainModel list, populated
// lazily from ctx.Backend.ListDomainModelsGen on first call. Required for cache
// discipline per memory feedback_executor_cache_pattern — direct
// ctx.Backend.ListDomainModelsGen calls in batch contexts are O(N²).
func cachedDomainModelsGen(ctx *ExecContext) ([]*genDm.DomainModel, error) {
	if ctx.Cache != nil && ctx.Cache.domainModelsGen != nil {
		return ctx.Cache.domainModelsGen, nil
	}
	list, err := ctx.DomainModelReader.ListDomainModelsGen()
	if err != nil {
		return nil, err
	}
	if ctx.Cache != nil {
		ctx.Cache.domainModelsGen = list
	}
	return list, nil
}

// cachedDomainModelsGenDeps is the HandlerDeps version of cachedDomainModelsGen.
func cachedDomainModelsGenDeps(deps *HandlerDeps) ([]*genDm.DomainModel, error) {
	if deps.Cache != nil && deps.Cache.domainModelsGen != nil {
		return deps.Cache.domainModelsGen, nil
	}
	list, err := deps.DomainModelReader.ListDomainModelsGen()
	if err != nil {
		return nil, err
	}
	if deps.Cache != nil {
		deps.Cache.domainModelsGen = list
	}
	return list, nil
}

// invalidateDomainModelsGenCache clears the cached gen-typed DomainModel
// listings. The legacy invalidateDomainModelsCache (in hierarchy.go,
// which clears the sdk-typed slice) is also extended to call this so
// older callers automatically refresh both caches; new gen-only call
// sites should prefer invalidateDomainModelsGenCache directly.
func invalidateDomainModelsGenCache(ctx *ExecContext) {
	if ctx == nil || ctx.Cache == nil {
		return
	}
	ctx.Cache.domainModelsWithContainerGen = nil
	ctx.Cache.domainModels = nil
	ctx.Cache.domainModelsGen = nil
	// Invalidate sub-backend cache so the next ListDomainModelsGen call
	// re-fetches from the backend. MprBackend.InvalidateCache() clears
	// per-domain caches on microflowBackend, pageBackend, domainModelBackend, etc.
	if ctx.CacheInvalidator != nil {
		ctx.CacheInvalidator.InvalidateCache()
	}
	// Per-module cache (Fix 1) is NOT cleared here. Mutator paths must
	// either:
	//   - call setDomainModelGenCached(ctx, moduleID, dm) for write-through
	//     when they hold the updated *DomainModel pointer; OR
	//   - call invalidateDomainModelGenForModule(ctx, moduleID) when they
	//     mutate via the backend without an updated pointer (CreateEntityGen,
	//     UpdateEntityGen, etc.).
}

// invalidateDomainModelGenForModule drops the cached DomainModel for a
// single module. Use this after backend mutations that update the SQLite
// state without producing a new in-memory *DomainModel pointer (e.g.
// CreateEntityGen, UpdateEntityGen, CreateAssociationGen).
func invalidateDomainModelGenForModule(ctx *ExecContext, moduleID model.ID) {
	if ctx == nil || ctx.Cache == nil || ctx.Cache.domainModelByModule == nil {
		return
	}
	delete(ctx.Cache.domainModelByModule, moduleID)
}

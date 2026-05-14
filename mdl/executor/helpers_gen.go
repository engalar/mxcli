// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5a — gen-typed shared helpers.
//
// This file provides modelsdk/gen-native equivalents of the
// microflow / nanoflow qualified-name builders in helpers.go that
// today consume sdk/microflows via Backend.ListMicroflows /
// Backend.ListNanoflows. Other reference builders (entity, page,
// snippet, javaaction, javascriptaction, ...) do not touch
// sdk/microflows so they stay in helpers.go.
//
// The dispatch layer (Stage 3.2.6) will route validation, autocomplete
// and other consumers to these *Gen variants and then delete the
// sdk-typed originals.

package executor

import (
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// listMicroflowsGen returns every microflow in the project as gen
// objects. Resolution order:
//
//  1. ctx.Microflows.ListAll() — the modelsdk-native repo, populated by
//     MprBackend in production and by tests via withMicroflowsRepo.
//  2. ctx.Backend.ListMicroflowsGen() — the gen-typed surface added in
//     Followup C; mock-only test contexts that have not wired a
//     ctx.Microflows repo can still seed data here.
//  3. (nil, nil) when neither path is available — callers treat this as
//     "no microflows in project".
func listMicroflowsGen(ctx *ExecContext) ([]*genMf.Microflow, error) {
	if ctx == nil {
		return nil, nil
	}
	if ctx.Microflows != nil {
		return ctx.Microflows.ListAll()
	}
	if ctx.Backend != nil {
		return ctx.Backend.ListMicroflowsGen()
	}
	return nil, nil
}

// listNanoflowsGen returns every nanoflow in the project as gen objects.
// Resolution order mirrors listMicroflowsGen:
//
//  1. ctx.Nanoflows.List("") — repo path.
//  2. ctx.Backend.ListNanoflowsGen() — backend surface.
//  3. (nil, nil) — no nanoflows.
func listNanoflowsGen(ctx *ExecContext) ([]*genMf.Nanoflow, error) {
	if ctx == nil {
		return nil, nil
	}
	if ctx.Nanoflows != nil {
		return ctx.Nanoflows.List("")
	}
	if ctx.Backend != nil {
		return ctx.Backend.ListNanoflowsGen()
	}
	return nil, nil
}

// genFlowContainerModule resolves the owning module name for a gen
// flow's element ID, going through the SQL-backed unit container chain.
// Mirrors lookupGenContainerModule (used by viz files) but accepts the
// hierarchy explicitly so callers that already built it avoid a second
// lookup. Returns "" if the container chain doesn't resolve to a module.
func genFlowContainerModule(ctx *ExecContext, h *ContainerHierarchy, id model.ID) string {
	if id == "" || ctx == nil || ctx.Microflows == nil || h == nil {
		return ""
	}
	containerID, err := ctx.Microflows.GetContainerUUID(id)
	if err != nil || containerID == "" {
		return ""
	}
	modID := h.FindModuleID(containerID)
	return h.GetModuleName(modID)
}

// buildMicroflowQualifiedNamesGen returns a set of all microflow
// qualified names in the project. Consumes gen types via
// ctx.Microflows (or ctx.Backend.ListMicroflowsGen as a fallback in
// mock-only test contexts).
func buildMicroflowQualifiedNamesGen(ctx *ExecContext) map[string]bool {
	result := make(map[string]bool)
	h, err := getHierarchy(ctx)
	if err != nil {
		return result
	}
	mfs, err := listMicroflowsGen(ctx)
	if err != nil {
		return result
	}
	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		modName := genFlowContainerModule(ctx, h, model.ID(mf.ID()))
		if modName == "" {
			continue
		}
		result[modName+"."+mf.Name()] = true
	}
	return result
}

// MicroflowGenWithContainer pairs a gen-typed *Microflow with its
// container UUID (the parent folder or module ID, resolved from the
// MPR Unit table). Returned by listMicroflowsWithContainerGen so call
// sites can do "for _, item := range list { ... item.ContainerUUID
// ... item.MF.Name() ... }" without doing a per-element
// GetContainerUUID lookup.
type MicroflowGenWithContainer struct {
	MF            *genMf.Microflow
	ContainerUUID model.ID
}

// NanoflowGenWithContainer mirrors MicroflowGenWithContainer for nanoflows.
type NanoflowGenWithContainer struct {
	NF            *genMf.Nanoflow
	ContainerUUID model.ID
}

// listMicroflowsWithContainerGen returns every microflow in the
// project paired with its container UUID, caching the result on
// ctx.Cache.microflowsWithContainerGen for the duration of the
// session.
//
// Why this exists (Followup E1, [[Executor cache pattern]]):
// Migrating production callers from the legacy sdk-typed
// ctx.Backend.ListMicroflows() to the gen-typed
// ctx.Microflows.ListAll() lost the inline ContainerID field
// (codec roundtrip drops Container linkage by design). The naive
// fix — calling ctx.Microflows.GetContainerUUID(mf.ID()) inside the
// loop — turns every list-and-filter pass into O(N²) SQL on large
// projects. This helper resolves all containers once and caches the
// pairing so the second caller in the same session pays O(1).
//
// Cache invalidation: invalidateMicroflowsCache (called by
// microflow/nanoflow create/drop paths) clears this slice along with
// microflowNames so subsequent reads see fresh container linkage.
func listMicroflowsWithContainerGen(ctx *ExecContext) ([]MicroflowGenWithContainer, error) {
	if ctx == nil {
		return nil, nil
	}
	if ctx.Cache != nil && ctx.Cache.microflowsWithContainerGen != nil {
		return ctx.Cache.microflowsWithContainerGen, nil
	}
	mfs, err := listMicroflowsGen(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]MicroflowGenWithContainer, 0, len(mfs))
	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		var containerUUID model.ID
		if ctx.Microflows != nil {
			if cid, err := ctx.Microflows.GetContainerUUID(model.ID(mf.ID())); err == nil {
				containerUUID = cid
			}
		}
		out = append(out, MicroflowGenWithContainer{MF: mf, ContainerUUID: containerUUID})
	}
	if ctx.Cache != nil {
		ctx.Cache.microflowsWithContainerGen = out
	}
	return out, nil
}

// listNanoflowsWithContainerGen mirrors listMicroflowsWithContainerGen
// for nanoflows. Container UUIDs are resolved through ctx.Microflows
// because nanoflows live alongside microflows in the Unit table and
// share the same lookup path.
func listNanoflowsWithContainerGen(ctx *ExecContext) ([]NanoflowGenWithContainer, error) {
	if ctx == nil {
		return nil, nil
	}
	if ctx.Cache != nil && ctx.Cache.nanoflowsWithContainerGen != nil {
		return ctx.Cache.nanoflowsWithContainerGen, nil
	}
	nfs, err := listNanoflowsGen(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]NanoflowGenWithContainer, 0, len(nfs))
	for _, nf := range nfs {
		if nf == nil {
			continue
		}
		var containerUUID model.ID
		if ctx.Microflows != nil {
			if cid, err := ctx.Microflows.GetContainerUUID(model.ID(nf.ID())); err == nil {
				containerUUID = cid
			}
		}
		out = append(out, NanoflowGenWithContainer{NF: nf, ContainerUUID: containerUUID})
	}
	if ctx.Cache != nil {
		ctx.Cache.nanoflowsWithContainerGen = out
	}
	return out, nil
}

// buildNanoflowQualifiedNamesGen returns a set of all nanoflow
// qualified names in the project, using gen types via ctx.Nanoflows.
func buildNanoflowQualifiedNamesGen(ctx *ExecContext) map[string]bool {
	result := make(map[string]bool)
	h, err := getHierarchy(ctx)
	if err != nil {
		return result
	}
	nfs, err := listNanoflowsGen(ctx)
	if err != nil {
		return result
	}
	for _, nf := range nfs {
		if nf == nil {
			continue
		}
		// Nanoflows live alongside microflows in the unit table, so
		// the same GetContainerUUID lookup resolves their owning module.
		modName := genFlowContainerModule(ctx, h, model.ID(nf.ID()))
		if modName == "" {
			continue
		}
		result[modName+"."+nf.Name()] = true
	}
	return result
}

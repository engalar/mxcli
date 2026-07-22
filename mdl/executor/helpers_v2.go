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
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// paramEntityRef returns the qualified-name reference for a microflow/nanoflow
// parameter that carries an entity type, or nil for primitive types.
//
// This helper handles the TypeEnumeration vs TypeEntity ambiguity documented in
// CLAUDE.md: buildDataType stores bare qualified names (e.g. "HD.Ticket") as
// TypeEnumeration+EnumRef rather than TypeEntity+EntityRef because the MDL
// grammar cannot distinguish the two at parse time. Both EntityRef and EnumRef
// must therefore be accepted as entity references when initialising varTypes
// for parameter resolution (e.g. classifyValidationTarget for CE0639).
//
// Callers use the returned *ast.QualifiedName to build the entity QN string
// ("Module.Name") for varTypes population. A nil return means the parameter
// is a primitive or list type with no entity component.
func paramEntityRef(dt ast.DataType) *ast.QualifiedName {
	if dt.EntityRef != nil {
		return dt.EntityRef
	}
	// EnumRef is set for bare qualified names parsed by buildDataType —
	// these are entity params in practice (user writes "Module.Entity").
	if dt.EnumRef != nil && dt.Kind == ast.TypeEnumeration {
		return dt.EnumRef
	}
	return nil
}

// listMicroflowsGen returns every microflow in the project as gen objects.
func listMicroflowsGen(ctx *ExecContext) ([]*genMf.Microflow, error) {
	if ctx == nil || ctx.Microflows == nil {
		return nil, nil
	}
	return ctx.Microflows.ListAll()
}

// listNanoflowsGen returns every nanoflow in the project as gen objects.
func listNanoflowsGen(ctx *ExecContext) ([]*genMf.Nanoflow, error) {
	if ctx == nil || ctx.Nanoflows == nil {
		return nil, nil
	}
	return ctx.Nanoflows.List("")
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
	return containerModuleName(h, containerID)
}

// genFlowContainerModuleDeps is the HandlerDeps version of genFlowContainerModule.
func genFlowContainerModuleDeps(deps *HandlerDeps, h *ContainerHierarchy, id model.ID) string {
	if id == "" || deps == nil || deps.MicroflowRepo == nil || h == nil {
		return ""
	}
	containerID, err := deps.MicroflowRepo.GetContainerUUID(id)
	if err != nil || containerID == "" {
		return ""
	}
	return containerModuleName(h, containerID)
}

// containerModuleName resolves the module name from a known container
// UUID using the in-memory container hierarchy.  This is the fast-path
// half of genFlowContainerModule — callers that already hold the
// container UUID (e.g. from listMicroflowsWithContainerGen) avoid the
// SQL GetContainerUUID round-trip.
func containerModuleName(h *ContainerHierarchy, containerID model.ID) string {
	if h == nil || containerID == "" {
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

// listMicroflowsWithContainerGen returns every microflow paired with its
// container UUID, using the domain-cached listing.
func listMicroflowsWithContainerGen(ctx *ExecContext) ([]MicroflowGenWithContainer, error) {
	if ctx == nil {
		return nil, nil
	}
	if ctx.Cache.microflowsWithContainerGen == nil {
		ctx.Cache.microflowsWithContainerGen = newDomainCache(func() ([]MicroflowGenWithContainer, error) {
			return loadMicroflowsWithContainerGen(ctx.Deps)
		})
	}
	return ctx.Cache.microflowsWithContainerGen.Get()
}

// loadMicroflowsWithContainerGen loads microflows without caching.
func loadMicroflowsWithContainerGen(deps *HandlerDeps) ([]MicroflowGenWithContainer, error) {
	if deps == nil || deps.MicroflowRepo == nil {
		return nil, nil
	}
	mfs, err := deps.MicroflowRepo.ListAll()
	if err != nil {
		return nil, err
	}
	out := make([]MicroflowGenWithContainer, 0, len(mfs))
	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		var containerUUID model.ID
		if cid, err := deps.MicroflowRepo.GetContainerUUID(model.ID(mf.ID())); err == nil {
			containerUUID = cid
		}
		out = append(out, MicroflowGenWithContainer{MF: mf, ContainerUUID: containerUUID})
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
	if ctx.Cache.nanoflowsWithContainerGen == nil {
		ctx.Cache.nanoflowsWithContainerGen = newDomainCache(func() ([]NanoflowGenWithContainer, error) {
			return loadNanoflowsWithContainerGen(ctx.Deps)
		})
	}
	return ctx.Cache.nanoflowsWithContainerGen.Get()
}

// invalidateMicroflowsCache clears the cached microflow+nanoflow listings.
func invalidateMicroflowsCache(ctx *ExecContext) {
	if ctx == nil || ctx.Cache == nil {
		return
	}
	ctx.Cache.Invalidate(CacheDomainMicroflows, CacheDomainNanoflows)
}

// loadNanoflowsWithContainerGen loads nanoflows without caching.
func loadNanoflowsWithContainerGen(deps *HandlerDeps) ([]NanoflowGenWithContainer, error) {
	if deps == nil || deps.NanoflowRepo == nil {
		return nil, nil
	}
	nfs, err := deps.NanoflowRepo.List("")
	if err != nil {
		return nil, err
	}
	out := make([]NanoflowGenWithContainer, 0, len(nfs))
	for _, nf := range nfs {
		if nf == nil {
			continue
		}
		var containerUUID model.ID
		if deps.MicroflowRepo != nil {
			if cid, err := deps.MicroflowRepo.GetContainerUUID(model.ID(nf.ID())); err == nil {
				containerUUID = cid
			}
		}
		out = append(out, NanoflowGenWithContainer{NF: nf, ContainerUUID: containerUUID})
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

// resolveBareEntityQN looks up a bare entity name (no module prefix) across
// all domain models and returns the fully qualified name Module.Entity, or ""
// if the entity cannot be uniquely identified.
func resolveBareEntityQN(dmRepo repos.DomainModelRepository, modLister backend.ModuleLister, bareName string) string {
	if bareName == "" {
		return ""
	}

	modules, err := modLister.ListModules()
	if err != nil {
		return ""
	}
	modNameByID := make(map[model.ID]string, len(modules))
	for _, m := range modules {
		modNameByID[m.ID] = m.Name
	}

	pairs, err := dmRepo.ListAllWithContainerID()
	if err != nil {
		return ""
	}

	for _, pair := range pairs {
		if pair.DM == nil {
			continue
		}
		for _, elem := range pair.DM.EntitiesItems() {
			ent, ok := elem.(*genDm.Entity)
			if !ok || ent == nil {
				continue
			}
			if ent.Name() == bareName {
				modName := modNameByID[pair.ContainerID]
				if modName != "" {
					return modName + "." + bareName
				}
			}
		}
	}
	return ""
}

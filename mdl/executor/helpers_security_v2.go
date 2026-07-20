// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"github.com/mendixlabs/mxcli/model"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// ModuleSecurityGenWithContainer pairs a gen-typed *ModuleSecurity with its
// container UUID (the parent module ID).
type ModuleSecurityGenWithContainer struct {
	MS          *genSec.ModuleSecurity
	ContainerID model.ID
}

// getProjectSecurityGen returns the singleton gen-typed ProjectSecurity,
// caching the result on ctx.Cache.projectSecurityGen for the session.
//
// Cache invalidation: any project-security mutation (SetSecurityLevel,
// AddUserRole, etc.) must call invalidateProjectSecurityCache to drop
// the cached pointer.
func getProjectSecurityGen(ctx *ExecContext) (*genSec.ProjectSecurity, error) {
	if ctx == nil || ctx.Security == nil {
		return nil, nil
	}
	if ctx.Cache != nil && ctx.Cache.projectSecurityGen != nil {
		return ctx.Cache.projectSecurityGen, nil
	}
	ps, err := ctx.Security.Ensure()
	if err != nil {
		return nil, err
	}
	if ctx.Cache != nil {
		ctx.Cache.projectSecurityGen = ps
	}
	return ps, nil
}

// listModuleSecurityWithContainerGen returns every ModuleSecurity unit
// in the project paired with its container module ID. Caches on
// ctx.Cache.moduleSecurityWithContainerGen for the session.
func listModuleSecurityWithContainerGen(ctx *ExecContext) ([]ModuleSecurityGenWithContainer, error) {
	if ctx == nil || ctx.Security == nil {
		return nil, nil
	}
	if ctx.Cache != nil && ctx.Cache.moduleSecurityWithContainerGen != nil {
		return ctx.Cache.moduleSecurityWithContainerGen, nil
	}
	modules, err := ctx.ModuleLister.ListModules()
	if err != nil {
		return nil, err
	}
	out := make([]ModuleSecurityGenWithContainer, 0, len(modules))
	for _, m := range modules {
		ms, err := ctx.Security.GetModuleSecurity(m.ID)
		if err != nil || ms == nil {
			continue
		}
		out = append(out, ModuleSecurityGenWithContainer{MS: ms, ContainerID: m.ID})
	}
	if ctx.Cache != nil {
		ctx.Cache.moduleSecurityWithContainerGen = out
	}
	return out, nil
}

// invalidateProjectSecurityCache clears the cached ProjectSecurity pointer.
// Called by any write path that mutates ProjectSecurity.
// getProjectSecurityGenDeps is the HandlerDeps version of getProjectSecurityGen.
func getProjectSecurityGenDeps(deps *HandlerDeps) (*genSec.ProjectSecurity, error) {
	if deps == nil || deps.Security == nil {
		return nil, nil
	}
	if deps.Cache != nil && deps.Cache.projectSecurityGen != nil {
		return deps.Cache.projectSecurityGen, nil
	}
	ps, err := deps.Security.Ensure()
	if err != nil {
		return nil, err
	}
	if deps.Cache != nil {
		deps.Cache.projectSecurityGen = ps
	}
	return ps, nil
}

func invalidateProjectSecurityCache(ctx *ExecContext) {
	if ctx == nil || ctx.Cache == nil {
		return
	}
	ctx.Cache.projectSecurityGen = nil
}

// invalidateModuleSecurityCache clears the cached per-module security list.
// Called by any write path that mutates ModuleSecurity (Add/Remove ModuleRole).
func invalidateModuleSecurityCache(ctx *ExecContext) {
	if ctx == nil || ctx.Cache == nil {
		return
	}
	ctx.Cache.moduleSecurityWithContainerGen = nil
}

// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.3.A0: cache helpers for workflows.
// Mirrors helpers_javaactions_gen.go shape, using the generic
// ContainerWithGen[T] factory introduced for the Stage 3.3 marathon
// (see helpers_gen_container.go).

package executor

import (
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// listWorkflowsWithContainerGen returns every workflow paired with its
// container UUID, using the domain-cached listing.
func listWorkflowsWithContainerGen(ctx *ExecContext) ([]ContainerWithGen[*genWf.Workflow], error) {
	if ctx == nil {
		return nil, nil
	}
	if ctx.Cache.workflowsWithContainerGen == nil {
		ctx.Cache.workflowsWithContainerGen = newDomainCache(func() ([]ContainerWithGen[*genWf.Workflow], error) {
			return loadWorkflowsWithContainerGen(ctx.Deps)
		})
	}
	return ctx.Cache.workflowsWithContainerGen.Get()
}

// loadWorkflowsWithContainerGen loads workflows without caching.
func loadWorkflowsWithContainerGen(deps *HandlerDeps) ([]ContainerWithGen[*genWf.Workflow], error) {
	if deps == nil || deps.WorkflowRepo == nil {
		return nil, nil
	}
	return listUnitsWithContainerGen(
		func() ([]*genWf.Workflow, error) {
			all, err := deps.WorkflowRepo.ListAll()
			if err != nil {
				return nil, err
			}
			filtered := all[:0]
			for _, w := range all {
				if w != nil {
					filtered = append(filtered, w)
				}
			}
			return filtered, nil
		},
		func(id element.ID) (element.ID, error) {
			c, err := deps.WorkflowRepo.GetContainerUUID(model.ID(id))
			return element.ID(c), err
		},
		func() ([]ContainerWithGen[*genWf.Workflow], bool) { return nil, false },
		func([]ContainerWithGen[*genWf.Workflow]) {},
	)
}

// listWorkflowsWithContainerGenDeps is the HandlerDeps version of listWorkflowsWithContainerGen.
func listWorkflowsWithContainerGenDeps(deps *HandlerDeps) ([]ContainerWithGen[*genWf.Workflow], error) {
	if deps == nil || deps.Cache == nil {
		return nil, nil
	}
	if deps.Cache.workflowsWithContainerGen == nil {
		deps.Cache.workflowsWithContainerGen = newDomainCache(func() ([]ContainerWithGen[*genWf.Workflow], error) {
			return loadWorkflowsWithContainerGen(deps)
		})
	}
	return deps.Cache.workflowsWithContainerGen.Get()
}

// invalidateWorkflowsCache clears the cached workflow listing.
func invalidateWorkflowsCache(ctx *ExecContext) {
	if ctx == nil || ctx.Cache == nil {
		return
	}
	ctx.Cache.Invalidate(CacheDomainWorkflows)
}

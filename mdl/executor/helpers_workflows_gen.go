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
// container UUID, caching the result on
// ctx.Cache.workflowsWithContainerGen for the session.
//
// Per memory `feedback_executor_cache_pattern`: list calls in
// mdl/executor/ MUST go through this helper to avoid O(N²) container
// lookups when iterating across the project.
func listWorkflowsWithContainerGen(ctx *ExecContext) ([]ContainerWithGen[*genWf.Workflow], error) {
	if ctx == nil {
		return nil, nil
	}
	listFn := func() ([]*genWf.Workflow, error) {
		if ctx.Workflows != nil {
			return ctx.Workflows.ListAll()
		}
		return nil, nil
	}
	resolveFn := func(id element.ID) (element.ID, error) {
		if ctx.Workflows != nil {
			c, err := ctx.Workflows.GetContainerUUID(model.ID(id))
			return element.ID(c), err
		}
		return "", nil
	}
	return listUnitsWithContainerGen(
		listFn,
		resolveFn,
		func() ([]ContainerWithGen[*genWf.Workflow], bool) {
			if ctx.Cache != nil && ctx.Cache.workflowsWithContainerGen != nil {
				return ctx.Cache.workflowsWithContainerGen, true
			}
			return nil, false
		},
		func(s []ContainerWithGen[*genWf.Workflow]) {
			if ctx.Cache != nil {
				ctx.Cache.workflowsWithContainerGen = s
			}
		},
	)
}

// invalidateWorkflowsCache clears the cached gen-typed workflow listing.
// Call from any write path that creates, drops, or otherwise mutates
// workflow units.
func invalidateWorkflowsCache(ctx *ExecContext) {
	if ctx == nil || ctx.Cache == nil {
		return
	}
	ctx.Cache.workflowsWithContainerGen = nil
}

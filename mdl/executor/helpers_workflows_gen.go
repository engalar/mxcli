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
		// Prefer ctx.Workflows (gen-native repo wired via the duck-type
		// provider on MprBackend); fall back to ctx.Backend.ListWorkflowsGen
		// which goes through the regular FullBackend interface (used by
		// MockBackend tests that wire ListWorkflowsGenFunc).
		var all []*genWf.Workflow
		var err error
		switch {
		case ctx.Workflows != nil:
			all, err = ctx.Workflows.ListAll()
		case ctx.Backend != nil:
			all, err = ctx.Backend.ListWorkflowsGen()
		default:
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		// Drop nil entries before handing off to the generic helper —
		// listUnitsWithContainerGen does NOT skip nils (its godoc names
		// per-domain filtering as the caller's contract).
		filtered := all[:0]
		for _, w := range all {
			if w != nil {
				filtered = append(filtered, w)
			}
		}
		return filtered, nil
	}
	resolveFn := func(id element.ID) (element.ID, error) {
		if ctx.Workflows != nil {
			c, err := ctx.Workflows.GetContainerUUID(model.ID(id))
			return element.ID(c), err
		}
		// MockBackend fallback: gen Workflow drops ContainerID during
		// codec roundtrip, but the legacy sdk Workflow carries it.
		// Mock tests typically wire ListWorkflowsFunc/GetWorkflowFunc,
		// so we fish ContainerID out of the legacy GetWorkflow path.
		// Returning "" is safe: the hierarchy walker tolerates it (the
		// caller's eventual h.FindModuleID just yields the empty module).
		//
		// E1 deletion: this fallback survives until Phase E1 retires
		// FullBackend.GetWorkflow. At that point mock tests must wire
		// RecordingWorkflowRepository.GetContainerUUIDFunc directly via
		// ctx.Workflows.
		if ctx.Backend != nil {
			if wf, err := ctx.Backend.GetWorkflow(model.ID(id)); err == nil && wf != nil {
				return element.ID(wf.ContainerID), nil
			}
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

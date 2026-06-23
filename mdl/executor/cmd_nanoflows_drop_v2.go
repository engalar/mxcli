// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5c — gen-typed DROP NANOFLOW.
//
// Parallel of cmd_nanoflows_drop.go that resolves the target through
// ctx.Nanoflows + the SQL-backed container hierarchy, then deletes
// via ctx.Nanoflows.Delete (a thin wrapper over the modelsdk-native
// writer's UnitDelete). No sdk/microflows types are touched.
//
// Drop is intentionally a thin handler — most of the work was already
// done by Stage 2 wiring ctx.Nanoflows. This file exists so the
// dispatch layer (Stage 3.2.6) can route execDropNanoflow to the gen
// path without leaving any sdk/microflows reference behind.

package executor

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// execDropNanoflowGen handles DROP NANOFLOW statements via the
// gen-typed read + delete path. Behaviour parity with the legacy
// execDropNanoflow:
//
//   - Refuses when the project is not opened for writing.
//   - Looks up the target nanoflow by qualified name.
//   - Records the drop in the executor cache (rememberDroppedNanoflow)
//     so a subsequent CREATE OR MODIFY can recover the original ID
//     and allowed-roles set.
//   - Invalidates hierarchy + microflows caches so subsequent reads
//     don't observe the deleted unit.
//
// One difference from the sdk path: dropped allowed-roles are recorded
// as nil because the gen Nanoflow stores qualified-name strings, not
// model.IDs. Recovering them on subsequent CREATE OR MODIFY would
// require a lossless QN→ID round-trip that the dropped-tracker schema
// does not yet model. The legacy path's role-preservation behaviour
// is exercised on the same flow via the sdk track until Stage 3.2.3
// reworks the dropped-tracker.
func execDropNanoflowGen(ctx *ExecContext, s *ast.DropNanoflowStmt) error {
	return execDropNanoflowGenFn(ctx, s, execContextToDeps(ctx))
}

// execDropNanoflowGenFn is the HandlerDeps version of execDropNanoflowGen.
func execDropNanoflowGenFn(ctx context.Context, s *ast.DropNanoflowStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if deps.NanoflowRepo == nil {
		return mdlerrors.NewBackend("nanoflows repo unavailable", nil)
	}

	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	all, err := deps.NanoflowRepo.List("")
	if err != nil {
		return mdlerrors.NewBackend("list nanoflows", err)
	}

	for _, nf := range all {
		if nf == nil {
			continue
		}
		ectx := phase3d2bNewExecContext(ctx, deps)
		modName := genFlowContainerModule(ectx, h, model.ID(nf.ID()))
		if modName != s.Name.Module || nf.Name() != s.Name.Name {
			continue
		}

		qualifiedName := s.Name.Module + "." + s.Name.Name
		containerID := model.ID("")
		if deps.MicroflowRepo != nil {
			if cid, err := deps.MicroflowRepo.GetContainerUUID(model.ID(nf.ID())); err == nil {
				containerID = cid
			}
		}
		rememberDroppedMicroflowFn(deps, qualifiedName, model.ID(nf.ID()), containerID, nil)

		if err := deps.NanoflowRepo.Delete(model.ID(nf.ID())); err != nil {
			return mdlerrors.NewBackend("delete nanoflow", err)
		}

		if deps.Cache != nil && deps.Cache.createdNanoflows != nil {
			delete(deps.Cache.createdNanoflows, qualifiedName)
		}
		invalidateHierarchyFn(deps)
		invalidateMicroflowsCacheFn(deps)
		fmt.Fprintf(deps.Output, "Dropped nanoflow: %s\n", qualifiedName)
		return nil
	}

	return mdlerrors.NewNotFound("nanoflow", s.Name.Module+"."+s.Name.Name)
}

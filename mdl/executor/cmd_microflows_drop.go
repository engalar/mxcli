// SPDX-License-Identifier: Apache-2.0

// Package executor - DROP MICROFLOW command
package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// execDropMicroflow handles DROP MICROFLOW statements.
//
// Stage 3.2.6.5: rewired to consume gen Microflow objects from
// ctx.Microflows; the legacy `[]*microflows.Microflow` walk and the
// sdk-typed `mf.AllowedModuleRoles` slice were replaced with
// `mf.AllowedModuleRolesQualifiedNames()` (gen accessor). The dropped
// info is recorded with the same qualified-name shape so a subsequent
// CREATE OR MODIFY can reuse the UnitID via consumeDroppedMicroflow.
func execDropMicroflow(ctx *ExecContext, s *ast.DropMicroflowStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	mfs, err := listMicroflowsGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list microflows", err)
	}

	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		mfID := model.ID(mf.ID())
		containerID, _ := ctx.Microflows.GetContainerUUID(mfID)
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if modName == s.Name.Module && mf.Name() == s.Name.Name {
			qualifiedName := s.Name.Module + "." + s.Name.Name
			roleQNs := mf.AllowedModuleRolesQualifiedNames()
			roles := make([]model.ID, 0, len(roleQNs))
			for _, qn := range roleQNs {
				roles = append(roles, model.ID(qn))
			}
			// Remember the UnitID and ContainerID *before* deletion so a
			// subsequent CREATE OR REPLACE/MODIFY for the same qualified
			// name can reuse them. This keeps Studio Pro compatible by
			// turning delete+insert into an in-place update — same
			// UnitID, same folder, just new bytes.
			rememberDroppedMicroflow(ctx, qualifiedName, mfID, containerID, roles)
			if err := ctx.deleteMicroflowViaRepoOrBackend(mfID); err != nil {
				return mdlerrors.NewBackend("delete microflow", err)
			}
			if ctx.Cache != nil && ctx.Cache.createdMicroflows != nil {
				delete(ctx.Cache.createdMicroflows, qualifiedName)
			}
			invalidateHierarchy(ctx)
			invalidateMicroflowsCache(ctx)
			fmt.Fprintf(ctx.Output, "Dropped microflow: %s.%s\n", s.Name.Module, s.Name.Name)
			return nil
		}
	}

	return mdlerrors.NewNotFound("microflow", s.Name.Module+"."+s.Name.Name)
}

// SPDX-License-Identifier: Apache-2.0

// Package executor - DROP MICROFLOW command
package executor

import (
	"context"
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
	return execDropMicroflowFn(ctx, s, execContextToDeps(ctx))
}

// execDropMicroflowFn is the HandlerDeps version of execDropMicroflow.
func execDropMicroflowFn(ctx context.Context, s *ast.DropMicroflowStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	if deps.MicroflowRepo == nil {
		return mdlerrors.NewBackend("microflows repo unavailable", nil)
	}
	mfs, err := deps.MicroflowRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list microflows", err)
	}

	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		mfID := model.ID(mf.ID())
		containerID, _ := deps.MicroflowRepo.GetContainerUUID(mfID)
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if modName == s.Name.Module && mf.Name() == s.Name.Name {
			qualifiedName := s.Name.Module + "." + s.Name.Name
			roleQNs := mf.AllowedModuleRolesQualifiedNames()
			roles := make([]model.ID, 0, len(roleQNs))
			for _, qn := range roleQNs {
				roles = append(roles, model.ID(qn))
			}
			rememberDroppedMicroflowFn(deps, qualifiedName, mfID, containerID, roles)
			if err := deleteMicroflowViaRepoOrBackendFn(deps, mfID); err != nil {
				return mdlerrors.NewBackend("delete microflow", err)
			}
			if deps.Cache != nil && deps.Cache.createdMicroflows != nil {
				delete(deps.Cache.createdMicroflows, qualifiedName)
			}
			invalidateHierarchyFn(deps)
			invalidateMicroflowsCacheFn(deps)
			fmt.Fprintf(deps.Output, "Dropped microflow: %s.%s\n", s.Name.Module, s.Name.Name)
			return nil
		}
	}

	return mdlerrors.NewNotFound("microflow", s.Name.Module+"."+s.Name.Name)
}

// rememberDroppedMicroflowFn is the HandlerDeps version of rememberDroppedMicroflow.
func rememberDroppedMicroflowFn(deps *HandlerDeps, qualifiedName string, id, containerID model.ID, allowedRoles []model.ID) {
	if deps == nil || qualifiedName == "" || id == "" {
		return
	}
	if deps.Cache == nil {
		deps.Cache = &executorCache{}
	}
	if deps.Cache.droppedMicroflows == nil {
		deps.Cache.droppedMicroflows = make(map[string]*droppedUnitInfo)
	}
	rolesCopy := make([]model.ID, len(allowedRoles))
	copy(rolesCopy, allowedRoles)
	deps.Cache.droppedMicroflows[qualifiedName] = &droppedUnitInfo{
		ID:           id,
		ContainerID:  containerID,
		AllowedRoles: rolesCopy,
	}
}

// deleteMicroflowViaRepoOrBackendFn is the HandlerDeps version of deleteMicroflowViaRepoOrBackend.
func deleteMicroflowViaRepoOrBackendFn(deps *HandlerDeps, id model.ID) error {
	if deps.MicroflowRepo != nil {
		return deps.MicroflowRepo.Delete(id)
	}
	return deps.MicroflowWriter.DeleteMicroflow(id)
}

// invalidateHierarchyFn is the HandlerDeps version of invalidateHierarchy.
func invalidateHierarchyFn(deps *HandlerDeps) {
	if deps.Cache != nil {
		deps.Cache.hierarchy = nil
	}
}

// invalidateMicroflowsCacheFn is the HandlerDeps version of invalidateMicroflowsCache.
func invalidateMicroflowsCacheFn(deps *HandlerDeps) {
	if deps.Cache != nil {
		deps.Cache.microflowNames = nil
		deps.Cache.microflowsWithContainerGen = nil
		deps.Cache.nanoflowsWithContainerGen = nil
	}
}

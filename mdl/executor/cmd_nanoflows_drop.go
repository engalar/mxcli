// SPDX-License-Identifier: Apache-2.0

// Package executor - DROP NANOFLOW command
package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

// execDropNanoflow handles DROP NANOFLOW statements.
func execDropNanoflow(ctx *ExecContext, s *ast.DropNanoflowStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	// Get hierarchy for module/folder resolution
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	// Find and delete the nanoflow
	nfs, err := ctx.Backend.ListNanoflows()
	if err != nil {
		return mdlerrors.NewBackend("list nanoflows", err)
	}

	for _, nf := range nfs {
		modID := h.FindModuleID(nf.ContainerID)
		modName := h.GetModuleName(modID)
		if modName == s.Name.Module && nf.Name == s.Name.Name {
			qualifiedName := s.Name.Module + "." + s.Name.Name
			rememberDroppedNanoflow(ctx, qualifiedName, nf.ID, nf.ContainerID, nf.AllowedModuleRoles)
			if err := ctx.deleteNanoflowViaRepoOrBackend(nf.ID); err != nil {
				return mdlerrors.NewBackend("delete nanoflow", err)
			}
			// Clear executor-level caches
			if ctx.Cache != nil && ctx.Cache.createdNanoflows != nil {
				delete(ctx.Cache.createdNanoflows, qualifiedName)
			}
			invalidateHierarchy(ctx)
			invalidateMicroflowsCache(ctx)
			fmt.Fprintf(ctx.Output, "Dropped nanoflow: %s.%s\n", s.Name.Module, s.Name.Name)
			return nil
		}
	}

	return mdlerrors.NewNotFound("nanoflow", s.Name.Module+"."+s.Name.Name)
}

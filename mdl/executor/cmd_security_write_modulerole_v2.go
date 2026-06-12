// SPDX-License-Identifier: Apache-2.0

// Stage 3.3 D1 + D2 — gen-typed module role write paths.
//
// execCreateModuleRoleGen mirrors execCreateModuleRole (cmd_security_write.go:20-72)
// but reads module security through ctx.Backend.GetModuleSecurityGen and
// iterates using ms.ModuleRolesItems() + (*genSec.ModuleRole) type assertions.
//
// execDropModuleRoleGen mirrors execDropModuleRole (cmd_security_write.go:77-173)
// with the same gen read path. Cascade helpers (microflows, nanoflows) reuse
// the existing cascadeRemoveRoleFromMicroflowsGen / cascadeRemoveRoleFromNanoflowsGen;
// entity, page, and OData cascades go through existing backend methods.

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// execCreateModuleRoleGen handles CREATE MODULE ROLE Module.RoleName [DESCRIPTION '...'].
// Gen-typed read path: ctx.Backend.GetModuleSecurityGen.
func execCreateModuleRoleGen(ctx *ExecContext, s *ast.CreateModuleRoleStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	module, err := findModule(ctx, s.Name.Module)
	if err != nil {
		return err
	}

	ms, err := ctx.SecurityModuleManager.GetModuleSecurityGen(module.ID)
	if err != nil {
		return mdlerrors.NewBackend(fmt.Sprintf("read module security for %s", s.Name.Module), err)
	}

	// Check for a name collision (case-insensitive, matching Mendix CE0123
	// behaviour). An auto-provisioned role with the sentinel description is
	// treated as an overwrite target so the caller's casing is adopted and
	// downstream references are renamed via UpdateQualifiedNameInAllUnits.
	for _, mr := range ms.ModuleRolesItems() {
		typed, ok := mr.(*genSec.ModuleRole)
		if !ok || !strings.EqualFold(typed.Name(), s.Name.Name) {
			continue
		}

		if typed.Description() == autoDocumentRoleDescription {
			oldQualified := s.Name.Module + "." + typed.Name()
			newQualified := s.Name.Module + "." + s.Name.Name
			// Remove first: AddModuleRole is a plain append with no dedup.
			// Without this, two roles with the same name cause Mendix CE1613.
			if err := ctx.SecurityModuleManager.RemoveModuleRole(model.ID(ms.ID()), typed.Name()); err != nil {
				return mdlerrors.NewBackend("remove auto-provisioned role", err)
			}
			if err := ctx.SecurityModuleManager.AddModuleRole(model.ID(ms.ID()), s.Name.Name, s.Description); err != nil {
				return mdlerrors.NewBackend("create module role", err)
			}
			// Propagate a casing rename across every unit that referenced the
			// old name (AllowedModuleRoles on microflows, pages, REST services,
			// etc.). Without this, mx check fails with CE1613.
			if oldQualified != newQualified {
				if _, err := ctx.RenameManager.UpdateQualifiedNameInAllUnits(oldQualified, newQualified); err != nil {
					return mdlerrors.NewBackend(
						fmt.Sprintf("rename references %s -> %s", oldQualified, newQualified), err)
				}
			}
			invalidateModuleSecurityCache(ctx)
			if !ctx.Quiet {
				fmt.Fprintf(ctx.Output, "Module role %s.%s already exists (auto-provisioned)\n",
					s.Name.Module, s.Name.Name)
			}
			return nil
		}
		// Custom role already exists — idempotent: skip with a helpful hint.
		if !ctx.Quiet {
			fmt.Fprintf(ctx.Output,
				"Module role %s.%s already exists.\nTo link it to a user role: ALTER USER ROLE \"User\" ADD MODULE ROLES (%s.\"%s\");\n",
				s.Name.Module, s.Name.Name, s.Name.Module, s.Name.Name)
		}
		return nil
	}

	if err := ctx.SecurityModuleManager.AddModuleRole(model.ID(ms.ID()), s.Name.Name, s.Description); err != nil {
		return mdlerrors.NewBackend("create module role", err)
	}

	invalidateModuleSecurityCache(ctx)
	fmt.Fprintf(ctx.Output, "Created module role: %s.%s\n", s.Name.Module, s.Name.Name)
	return nil
}

// execDropModuleRoleGen handles DROP MODULE ROLE Module.RoleName.
// Cascade-removes the role from all entity access rules, microflow/nanoflow/page
// allowed roles, and OData service allowed roles before deleting the role itself.
// Gen-typed read path: ctx.Backend.GetModuleSecurityGen.
func execDropModuleRoleGen(ctx *ExecContext, s *ast.DropModuleRoleStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	module, err := findModule(ctx, s.Name.Module)
	if err != nil {
		return err
	}

	ms, err := ctx.SecurityModuleManager.GetModuleSecurityGen(module.ID)
	if err != nil {
		return mdlerrors.NewBackend(fmt.Sprintf("read module security for %s", s.Name.Module), err)
	}

	// Verify the role exists.
	found := false
	for _, mr := range ms.ModuleRolesItems() {
		typed, ok := mr.(*genSec.ModuleRole)
		if !ok {
			continue
		}
		if typed.Name() == s.Name.Name {
			found = true
			break
		}
	}
	if !found {
		return mdlerrors.NewNotFound("module role", s.Name.Module+"."+s.Name.Name)
	}

	qualifiedRole := s.Name.Module + "." + s.Name.Name

	// Cascade: remove role from entity access rules.
	dm, err := getDomainModelGenCached(ctx, module.ID)
	if err == nil && dm != nil {
		if n, err := ctx.SecurityEntityAccessManager.RemoveRoleFromAllEntities(model.ID(dm.ID()), qualifiedRole); err != nil {
			return mdlerrors.NewBackend("cascade-remove entity access rules", err)
		} else if n > 0 {
			fmt.Fprintf(ctx.Output, "Removed %s from %d entity access rule(s)\n", qualifiedRole, n)
			invalidateDomainModelGenForModule(ctx, module.ID)
		}
	}

	// Cascade: remove role from microflow and nanoflow allowed roles via gen.
	if err := cascadeRemoveRoleFromMicroflowsGen(ctx, module.ID, qualifiedRole); err != nil {
		return err
	}
	if err := cascadeRemoveRoleFromNanoflowsGen(ctx, module.ID, qualifiedRole); err != nil {
		return err
	}

	h, err := getHierarchy(ctx)
	if err == nil {
		// Cascade: pages
		if pgPairs, err := listPagesWithContainerGen(ctx); err == nil {
			for _, pair := range pgPairs {
				pg := pair.Elem
				modID := h.FindModuleID(model.ID(pair.ContainerID))
				if modID != module.ID {
					continue
				}
				if removed, err := ctx.SecurityEntityAccessManager.RemoveFromAllowedRoles(model.ID(pg.ID()), qualifiedRole); err == nil && removed {
					fmt.Fprintf(ctx.Output, "Removed %s from page %s allowed roles\n", qualifiedRole, pg.Name())
				}
			}
		}

		// Cascade: OData services
		if svcs, err := ctx.ServiceLister.ListPublishedODataServices(); err == nil {
			for _, svc := range svcs {
				modID := h.FindModuleID(svc.ContainerID)
				if modID != module.ID {
					continue
				}
				if removed, err := ctx.SecurityEntityAccessManager.RemoveFromAllowedRoles(svc.ID, qualifiedRole); err == nil && removed {
					fmt.Fprintf(ctx.Output, "Removed %s from OData service %s allowed roles\n", qualifiedRole, svc.Name)
				}
			}
		}
	}

	// Cascade: remove role from user roles in ProjectSecurity.
	if ps, err := ctx.SecurityProjectManager.GetProjectSecurityGen(); err == nil && ps != nil {
		if n, err := ctx.SecurityModuleManager.RemoveModuleRoleFromAllUserRoles(model.ID(ps.ID()), qualifiedRole); err == nil && n > 0 {
			fmt.Fprintf(ctx.Output, "Removed %s from %d user role(s)\n", qualifiedRole, n)
		}
		if err := pruneInvalidUserRoles(ctx, nil); err != nil {
			return mdlerrors.NewBackend("cleanup invalid user roles", err)
		}
	}

	// Finally, remove the role itself.
	if err := ctx.SecurityModuleManager.RemoveModuleRole(model.ID(ms.ID()), s.Name.Name); err != nil {
		return mdlerrors.NewBackend("drop module role", err)
	}

	invalidateModuleSecurityCache(ctx)
	fmt.Fprintf(ctx.Output, "Dropped module role: %s.%s\n", s.Name.Module, s.Name.Name)
	return nil
}

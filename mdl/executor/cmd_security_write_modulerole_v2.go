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
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// ExecCreateModuleRoleGenFn is the HandlerDeps version of execCreateModuleRoleGen.
func ExecCreateModuleRoleGenFn(ctx context.Context, s *ast.CreateModuleRoleStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if deps.SecurityModuleManager == nil || deps.ModuleLister == nil || deps.RenameManager == nil {
		return mdlerrors.NewBackend("backend not fully initialized", nil)
	}

	module, err := findModuleDeps(ctx, deps, s.Name.Module)
	if err != nil {
		return err
	}

	ms, err := deps.SecurityModuleManager.GetModuleSecurityGen(module.ID)
	if err != nil {
		return mdlerrors.NewBackend(fmt.Sprintf("read module security for %s", s.Name.Module), err)
	}

	for _, mr := range ms.ModuleRolesItems() {
		typed, ok := mr.(*genSec.ModuleRole)
		if !ok || !strings.EqualFold(typed.Name(), s.Name.Name) {
			continue
		}

		if typed.Description() == autoDocumentRoleDescription {
			oldQualified := s.Name.Module + "." + typed.Name()
			newQualified := s.Name.Module + "." + s.Name.Name
			if err := deps.SecurityModuleManager.RemoveModuleRole(model.ID(ms.ID()), typed.Name()); err != nil {
				return mdlerrors.NewBackend("remove auto-provisioned role", err)
			}
			if err := deps.SecurityModuleManager.AddModuleRole(model.ID(ms.ID()), s.Name.Name, s.Description); err != nil {
				return mdlerrors.NewBackend("create module role", err)
			}
			if oldQualified != newQualified {
				if _, err := deps.RenameManager.UpdateQualifiedNameInAllUnits(oldQualified, newQualified); err != nil {
					return mdlerrors.NewBackend(
						fmt.Sprintf("rename references %s -> %s", oldQualified, newQualified), err)
				}
			}
			if deps.Cache != nil {
				deps.Cache.moduleSecurityWithContainerGen = nil
			}
			if !deps.Quiet {
				fmt.Fprintf(deps.Output, "Module role %s.%s already exists (auto-provisioned)\n",
					s.Name.Module, s.Name.Name)
			}
			return nil
		}
		if !deps.Quiet {
			fmt.Fprintf(deps.Output,
				"Module role %s.%s already exists.\nTo link it to a user role: ALTER USER ROLE \"User\" ADD MODULE ROLES (%s.\"%s\");\n",
				s.Name.Module, s.Name.Name, s.Name.Module, s.Name.Name)
		}
		return nil
	}

	if err := deps.SecurityModuleManager.AddModuleRole(model.ID(ms.ID()), s.Name.Name, s.Description); err != nil {
		return mdlerrors.NewBackend("create module role", err)
	}

	if deps.Cache != nil {
		deps.Cache.moduleSecurityWithContainerGen = nil
	}
	fmt.Fprintf(deps.Output, "Created module role: %s.%s\n", s.Name.Module, s.Name.Name)
	return nil
}

// ExecDropModuleRoleGenFn is the HandlerDeps version of execDropModuleRoleGen.
func ExecDropModuleRoleGenFn(ctx context.Context, s *ast.DropModuleRoleStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	module, err := findModuleDeps(ctx, deps, s.Name.Module)
	if err != nil {
		return err
	}

	ms, err := deps.SecurityModuleManager.GetModuleSecurityGen(module.ID)
	if err != nil {
		return mdlerrors.NewBackend(fmt.Sprintf("read module security for %s", s.Name.Module), err)
	}

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

	if dm, err := getDomainModelGenCachedDeps(ctx, deps, module.ID); err == nil && dm != nil {
		if n, err := deps.SecurityEntityAccessManager.RemoveRoleFromAllEntities(model.ID(dm.ID()), qualifiedRole); err != nil {
			return mdlerrors.NewBackend("cascade-remove entity access rules", err)
		} else if n > 0 {
			fmt.Fprintf(deps.Output, "Removed %s from %d entity access rule(s)\n", qualifiedRole, n)
			invalidateDomainModelGenForModuleDeps(deps, module.ID)
		}
	}

	if err := cascadeRemoveRoleFromMicroflowsGenFn(ctx, deps, module.ID, qualifiedRole); err != nil {
		return err
	}
	if err := cascadeRemoveRoleFromNanoflowsGenFn(ctx, deps, module.ID, qualifiedRole); err != nil {
		return err
	}

	h, err := getHierarchyDeps(deps)
	if err == nil {
		bg := context.Background()
		if pgPairs, err := listPagesWithContainerGenDeps(bg, deps); err == nil {
			for _, pair := range pgPairs {
				pg := pair.Elem
				modID := h.FindModuleID(model.ID(pair.ContainerID))
				if modID != module.ID {
					continue
				}
				if removed, err := deps.SecurityEntityAccessManager.RemoveFromAllowedRoles(model.ID(pg.ID()), qualifiedRole); err == nil && removed {
					fmt.Fprintf(deps.Output, "Removed %s from page %s allowed roles\n", qualifiedRole, pg.Name())
				}
			}
		}

		if svcs, err := deps.ServiceLister.ListPublishedODataServices(); err == nil {
			for _, svc := range svcs {
				modID := h.FindModuleID(svc.ContainerID)
				if modID != module.ID {
					continue
				}
				if removed, err := deps.SecurityEntityAccessManager.RemoveFromAllowedRoles(svc.ID, qualifiedRole); err == nil && removed {
					fmt.Fprintf(deps.Output, "Removed %s from OData service %s allowed roles\n", qualifiedRole, svc.Name)
				}
			}
		}
	}

	if ps, err := deps.SecurityProjectManager.GetProjectSecurityGen(); err == nil && ps != nil {
		if n, err := deps.SecurityModuleManager.RemoveModuleRoleFromAllUserRoles(model.ID(ps.ID()), qualifiedRole); err == nil && n > 0 {
			fmt.Fprintf(deps.Output, "Removed %s from %d user role(s)\n", qualifiedRole, n)
		}
		if err := pruneInvalidUserRolesDeps(deps, nil); err != nil {
			return mdlerrors.NewBackend("cleanup invalid user roles", err)
		}
	}

	if err := deps.SecurityModuleManager.RemoveModuleRole(model.ID(ms.ID()), s.Name.Name); err != nil {
		return mdlerrors.NewBackend("drop module role", err)
	}

	if deps.Cache != nil {
		deps.Cache.moduleSecurityWithContainerGen = nil
	}
	fmt.Fprintf(deps.Output, "Dropped module role: %s.%s\n", s.Name.Module, s.Name.Name)
	return nil
}

// execCreateModuleRoleGen handles CREATE MODULE ROLE. Delegates to Fn version.

// execDropModuleRoleGen handles DROP MODULE ROLE. Delegates to Fn version.

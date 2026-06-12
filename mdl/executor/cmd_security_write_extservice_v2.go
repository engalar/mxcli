// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.1.D8 — gen-typed OData / Published REST service access grant/revoke.
//
// Mirrors cmd_security_write.go:908–1149 (execGrantODataServiceAccess,
// execRevokeODataServiceAccess, execGrantPublishedRestServiceAccess,
// execRevokePublishedRestServiceAccess) using the gen write path.
//
// Role validation goes through ctx.Backend.GetModuleSecurityGen (same as
// validateModuleRole, which is called from the shared helper).
//
// Service reads stay on legacy backend methods (GetODataService,
// GetPublishedRestService) — those domains are not in Stage 3.3 scope.
//
// Mutations use ctx.Backend.UpdateAllowedRoles (OData) and
// ctx.Backend.UpdatePublishedRestServiceRoles (REST), which are already
// wired to the gen write path (updateAllowedRolesViaModelsdk).
//
// Role-list helpers (mergeAllowedRoles, filterAllowedRoles) are reused
// from cmd_security_write_gen.go — do NOT duplicate them here.

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

// ─────────────────────────────────────────────────────────────────
// GRANT / REVOKE ACCESS ON ODATA SERVICE
// ─────────────────────────────────────────────────────────────────

// execGrantODataServiceAccessGen handles GRANT ACCESS ON ODATA SERVICE Module.Svc TO roles.
// Mirrors execGrantODataServiceAccess but uses mergeAllowedRoles helper.
func execGrantODataServiceAccessGen(ctx *ExecContext, s *ast.GrantODataServiceAccessStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	services, err := ctx.Backend.ListPublishedODataServices()
	if err != nil {
		return mdlerrors.NewBackend("list published OData services", err)
	}

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if modName != s.Service.Module || svc.Name != s.Service.Name {
			continue
		}

		var validRoles []ast.QualifiedName
		for _, role := range s.Roles {
			found, err := validateModuleRole(ctx, role)
			if err != nil {
				return err
			}
			if found {
				validRoles = append(validRoles, role)
			}
		}
		if len(validRoles) == 0 {
			return nil
		}

		merged, added := mergeAllowedRoles(svc.AllowedModuleRoles, validRoles)

		if err := ctx.SecurityEntityAccessManager.UpdateAllowedRoles(svc.ID, merged); err != nil {
			return mdlerrors.NewBackend("update OData service access", err)
		}

		if len(added) == 0 {
			fmt.Fprintf(ctx.Output, "All specified roles already have access on OData service %s.%s\n", modName, svc.Name)
		} else {
			fmt.Fprintf(ctx.Output, "Granted access on OData service %s.%s to %s\n", modName, svc.Name, strings.Join(added, ", "))
		}
		return nil
	}

	return mdlerrors.NewNotFound("published OData service", s.Service.Module+"."+s.Service.Name)
}

// execRevokeODataServiceAccessGen handles REVOKE ACCESS ON ODATA SERVICE Module.Svc FROM roles.
// Mirrors execRevokeODataServiceAccess but uses filterAllowedRoles helper.
func execRevokeODataServiceAccessGen(ctx *ExecContext, s *ast.RevokeODataServiceAccessStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	services, err := ctx.Backend.ListPublishedODataServices()
	if err != nil {
		return mdlerrors.NewBackend("list published OData services", err)
	}

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if modName != s.Service.Module || svc.Name != s.Service.Name {
			continue
		}

		remaining, removed := filterAllowedRoles(svc.AllowedModuleRoles, s.Roles)

		if err := ctx.SecurityEntityAccessManager.UpdateAllowedRoles(svc.ID, remaining); err != nil {
			return mdlerrors.NewBackend("update OData service access", err)
		}

		if len(removed) == 0 {
			fmt.Fprintf(ctx.Output, "None of the specified roles had access on OData service %s.%s\n", modName, svc.Name)
		} else {
			fmt.Fprintf(ctx.Output, "Revoked access on OData service %s.%s from %s\n", modName, svc.Name, strings.Join(removed, ", "))
		}
		return nil
	}

	return mdlerrors.NewNotFound("published OData service", s.Service.Module+"."+s.Service.Name)
}

// ─────────────────────────────────────────────────────────────────
// GRANT / REVOKE ACCESS ON PUBLISHED REST SERVICE
// ─────────────────────────────────────────────────────────────────

// execGrantPublishedRestServiceAccessGen handles GRANT ACCESS ON PUBLISHED REST SERVICE Module.Svc TO roles.
// Mirrors execGrantPublishedRestServiceAccess but uses mergeAllowedRoles helper.
func execGrantPublishedRestServiceAccessGen(ctx *ExecContext, s *ast.GrantPublishedRestServiceAccessStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	if err := checkFeature(ctx, "integration", "published_rest_grant_revoke",
		"grant access on published rest service",
		"upgrade your project to 10.0+"); err != nil {
		return err
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	services, err := ctx.Backend.ListPublishedRestServices()
	if err != nil {
		return mdlerrors.NewBackend("list published rest services", err)
	}

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if modName != s.Service.Module || svc.Name != s.Service.Name {
			continue
		}

		var validRoles []ast.QualifiedName
		for _, role := range s.Roles {
			found, err := validateModuleRole(ctx, role)
			if err != nil {
				return err
			}
			if found {
				validRoles = append(validRoles, role)
			}
		}
		if len(validRoles) == 0 {
			return nil
		}

		merged, added := mergeAllowedRoles(svc.AllowedRoles, validRoles)

		if err := ctx.SecurityEntityAccessManager.UpdatePublishedRestServiceRoles(svc.ID, merged); err != nil {
			return mdlerrors.NewBackend("update published rest service access", err)
		}

		if len(added) == 0 {
			fmt.Fprintf(ctx.Output, "All specified roles already have access on published rest service %s.%s\n", modName, svc.Name)
		} else {
			fmt.Fprintf(ctx.Output, "Granted access on published rest service %s.%s to %s\n", modName, svc.Name, strings.Join(added, ", "))
		}
		return nil
	}

	return mdlerrors.NewNotFound("published rest service", s.Service.Module+"."+s.Service.Name)
}

// execRevokePublishedRestServiceAccessGen handles REVOKE ACCESS ON PUBLISHED REST SERVICE Module.Svc FROM roles.
// Mirrors execRevokePublishedRestServiceAccess but uses filterAllowedRoles helper.
func execRevokePublishedRestServiceAccessGen(ctx *ExecContext, s *ast.RevokePublishedRestServiceAccessStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	services, err := ctx.Backend.ListPublishedRestServices()
	if err != nil {
		return mdlerrors.NewBackend("list published rest services", err)
	}

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if modName != s.Service.Module || svc.Name != s.Service.Name {
			continue
		}

		remaining, removed := filterAllowedRoles(svc.AllowedRoles, s.Roles)

		if err := ctx.SecurityEntityAccessManager.UpdatePublishedRestServiceRoles(svc.ID, remaining); err != nil {
			return mdlerrors.NewBackend("update published rest service access", err)
		}

		if len(removed) == 0 {
			fmt.Fprintf(ctx.Output, "None of the specified roles had access on published rest service %s.%s\n", modName, svc.Name)
		} else {
			fmt.Fprintf(ctx.Output, "Revoked access on published rest service %s.%s from %s\n", modName, svc.Name, strings.Join(removed, ", "))
		}
		return nil
	}

	return mdlerrors.NewNotFound("published rest service", s.Service.Module+"."+s.Service.Name)
}

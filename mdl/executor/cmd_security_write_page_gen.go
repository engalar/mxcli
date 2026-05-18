// SPDX-License-Identifier: Apache-2.0

// Stage 3.3 D7 — gen-typed page security write paths.
//
// execGrantPageAccessGen mirrors execGrantPageAccess (cmd_security_write.go:562-621)
// execRevokePageAccessGen mirrors execRevokePageAccess (cmd_security_write.go:623-677)
//
// Page reads remain on ctx.Backend.ListPages (page domain migration deferred
// to Stage 3.3 priority #5 territory). Mutations go through
// ctx.Backend.UpdateAllowedRoles which is now correctly versioned post-D6a.
// Role validation reuses validateModuleRole which internally calls
// ctx.Backend.GetModuleSecurityGen (C1).
//
// mergeAllowedRoles / filterAllowedRoles helpers are defined in
// cmd_security_write_gen.go — do NOT duplicate them here.

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// execGrantPageAccessGen handles GRANT VIEW ON PAGE Module.Page TO roles.
// Mirrors execGrantPageAccess but uses mergeAllowedRoles helper and
// validates roles through validateModuleRole (gen read path).
func execGrantPageAccessGen(ctx *ExecContext, s *ast.GrantPageAccessStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pgPairs, err := listPagesWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}

	for _, pair := range pgPairs {
		pg := pair.Elem
		modID := h.FindModuleID(model.ID(pair.ContainerID))
		modName := h.GetModuleName(modID)
		if modName != s.Page.Module || pg.Name() != s.Page.Name {
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

		existing := pg.AllowedRolesQualifiedNames()
		merged, added := mergeAllowedRoles(existing, validRoles)

		if err := ctx.Backend.UpdateAllowedRoles(model.ID(pg.ID()), merged); err != nil {
			return mdlerrors.NewBackend("update page access", err)
		}

		if len(added) == 0 {
			fmt.Fprintf(ctx.Output, "All specified roles already have view access on %s.%s\n", modName, pg.Name())
		} else {
			fmt.Fprintf(ctx.Output, "Granted view access on %s.%s to %s\n", modName, pg.Name(), strings.Join(added, ", "))
		}
		return nil
	}

	return mdlerrors.NewNotFound("page", s.Page.Module+"."+s.Page.Name)
}

// execRevokePageAccessGen handles REVOKE VIEW ON PAGE Module.Page FROM roles.
// Mirrors execRevokePageAccess but uses filterAllowedRoles helper.
func execRevokePageAccessGen(ctx *ExecContext, s *ast.RevokePageAccessStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pgPairs2, err := listPagesWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}

	for _, pair := range pgPairs2 {
		pg := pair.Elem
		modID := h.FindModuleID(model.ID(pair.ContainerID))
		modName := h.GetModuleName(modID)
		if modName != s.Page.Module || pg.Name() != s.Page.Name {
			continue
		}

		existing := pg.AllowedRolesQualifiedNames()
		remaining, removed := filterAllowedRoles(existing, s.Roles)

		if err := ctx.Backend.UpdateAllowedRoles(model.ID(pg.ID()), remaining); err != nil {
			return mdlerrors.NewBackend("update page access", err)
		}

		if len(removed) == 0 {
			fmt.Fprintf(ctx.Output, "None of the specified roles had view access on %s.%s\n", modName, pg.Name())
		} else {
			fmt.Fprintf(ctx.Output, "Revoked view access on %s.%s from %s\n", modName, pg.Name(), strings.Join(removed, ", "))
		}
		return nil
	}

	return mdlerrors.NewNotFound("page", s.Page.Module+"."+s.Page.Name)
}

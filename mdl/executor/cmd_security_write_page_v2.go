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

	// First, check session-created pages (avoids re-listing all pages
	// when the page was created earlier in the same ExecuteProgram call).
	if ctx.Cache != nil && ctx.Cache.createdPages != nil {
		qualifiedName := s.Page.Module + "." + s.Page.Name
		if info, ok := ctx.Cache.createdPages[qualifiedName]; ok {
			return execGrantExistingPage(ctx, s, info.ID, info.ModuleName)
		}
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

		return execGrantExistingPage(ctx, s, model.ID(pg.ID()), modName)
	}

	return mdlerrors.NewNotFound("page", s.Page.Module+"."+s.Page.Name)
}

// execGrantExistingPage applies role grants to a page identified by ID, deduplicating
// the common grant-role-validation logic between session-created and backend pages.
func execGrantExistingPage(ctx *ExecContext, s *ast.GrantPageAccessStmt, pageID model.ID, modName string) error {
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

	// Read the page to get current allowed roles for merging.
	page, err := ctx.Pages.Get(pageID)
	if err != nil {
		return mdlerrors.NewBackend("get page", err)
	}
	if page == nil {
		return mdlerrors.NewNotFound("page", s.Page.Module+"."+s.Page.Name)
	}
	existing := filterAutoDocumentRoles(page.AllowedRolesQualifiedNames())
	merged, added := mergeAllowedRoles(existing, validRoles)

	if err := ctx.SecurityEntityAccessManager.UpdateAllowedRoles(pageID, merged); err != nil {
		return mdlerrors.NewBackend("update page access", err)
	}

	if len(added) == 0 {
		fmt.Fprintf(ctx.Output, "All specified roles already have view access on %s.%s\n", modName, s.Page.Name)
	} else {
		fmt.Fprintf(ctx.Output, "Granted view access on %s.%s to %s\n", modName, s.Page.Name, strings.Join(added, ", "))
	}
	return nil
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

	// Check session-created pages first (same rationale as execGrantPageAccessGen).
	pageID, err := lookupCreatedPageID(ctx, s.Page.Module+"."+s.Page.Name)
	if err != nil {
		return err
	}
	if pageID != "" {
		return execRevokeExistingPage(ctx, s, pageID)
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

		if err := ctx.SecurityEntityAccessManager.UpdateAllowedRoles(model.ID(pg.ID()), remaining); err != nil {
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

// lookupCreatedPageID finds a session-created page by qualified name and
// returns its ID. Returns ("", nil) if the page is not found in the session cache.
func lookupCreatedPageID(ctx *ExecContext, qualifiedName string) (model.ID, error) {
	if ctx == nil || ctx.Cache == nil || ctx.Cache.createdPages == nil {
		return "", nil
	}
	if info, ok := ctx.Cache.createdPages[qualifiedName]; ok {
		return info.ID, nil
	}
	return "", nil
}

// execRevokeExistingPage applies role revocation to a page by ID.
func execRevokeExistingPage(ctx *ExecContext, s *ast.RevokePageAccessStmt, pageID model.ID) error {
	page, err := ctx.Pages.Get(pageID)
	if err != nil {
		return mdlerrors.NewBackend("get page", err)
	}
	if page == nil {
		return mdlerrors.NewNotFound("page", s.Page.Module+"."+s.Page.Name)
	}
	existing := page.AllowedRolesQualifiedNames()
	remaining, removed := filterAllowedRoles(existing, s.Roles)

	if err := ctx.SecurityEntityAccessManager.UpdateAllowedRoles(pageID, remaining); err != nil {
		return mdlerrors.NewBackend("update page access", err)
	}

	if len(removed) == 0 {
		fmt.Fprintf(ctx.Output, "None of the specified roles had view access on %s.%s\n", s.Page.Module, s.Page.Name)
	} else {
		fmt.Fprintf(ctx.Output, "Revoked view access on %s.%s from %s\n", s.Page.Module, s.Page.Name, strings.Join(removed, ", "))
	}
	return nil
}

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
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// ExecGrantPageAccessGenFn is the HandlerDeps version of execGrantPageAccessGen.
func ExecGrantPageAccessGenFn(ctx context.Context, s *ast.GrantPageAccessStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	ectx := NewExecContext(ctx, deps)
	h, err := getHierarchy(ectx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}
	if ectx.Cache != nil && ectx.Cache.createdPages != nil {
		qualifiedName := s.Page.Module + "." + s.Page.Name
		if info, ok := ectx.Cache.createdPages[qualifiedName]; ok {
			return execGrantExistingPageFn(ctx, s, info.ID, info.ModuleName, deps)
		}
	}
	pgPairs, err := listPagesWithContainerGen(ectx)
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
		return execGrantExistingPageFn(ctx, s, model.ID(pg.ID()), modName, deps)
	}
	return mdlerrors.NewNotFound("page", s.Page.Module+"."+s.Page.Name)
}

func execGrantExistingPageFn(ctx context.Context, s *ast.GrantPageAccessStmt, pageID model.ID, modName string, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	var validRoles []ast.QualifiedName
	for _, role := range s.Roles {
		found, err := validateModuleRole(ectx, role)
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
	page, err := deps.PageRepo.Get(pageID)
	if err != nil {
		return mdlerrors.NewBackend("get page", err)
	}
	if page == nil {
		return mdlerrors.NewNotFound("page", s.Page.Module+"."+s.Page.Name)
	}
	existing := filterAutoDocumentRoles(page.AllowedRolesQualifiedNames())
	merged, added := mergeAllowedRoles(existing, validRoles)
	if err := deps.SecurityEntityAccessManager.UpdateAllowedRoles(pageID, merged); err != nil {
		return mdlerrors.NewBackend("update page access", err)
	}
	if len(added) == 0 {
		fmt.Fprintf(deps.Output, "All specified roles already have view access on %s.%s\n", modName, s.Page.Name)
	} else {
		fmt.Fprintf(deps.Output, "Granted view access on %s.%s to %s\n", modName, s.Page.Name, strings.Join(added, ", "))
	}
	return nil
}

// ExecRevokePageAccessGenFn is the HandlerDeps version of execRevokePageAccessGen.
func ExecRevokePageAccessGenFn(ctx context.Context, s *ast.RevokePageAccessStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	ectx := NewExecContext(ctx, deps)
	h, err := getHierarchy(ectx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}
	pageID, err := lookupCreatedPageIDFn(ctx, deps, s.Page.Module+"."+s.Page.Name)
	if err != nil {
		return err
	}
	if pageID != "" {
		return execRevokeExistingPageFn(ctx, s, pageID, deps)
	}
	pgPairs, err := listPagesWithContainerGen(ectx)
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
		existing := pg.AllowedRolesQualifiedNames()
		remaining, removed := filterAllowedRoles(existing, s.Roles)
		if err := deps.SecurityEntityAccessManager.UpdateAllowedRoles(model.ID(pg.ID()), remaining); err != nil {
			return mdlerrors.NewBackend("update page access", err)
		}
		if len(removed) == 0 {
			fmt.Fprintf(deps.Output, "None of the specified roles had view access on %s.%s\n", modName, pg.Name())
		} else {
			fmt.Fprintf(deps.Output, "Revoked view access on %s.%s from %s\n", modName, pg.Name(), strings.Join(removed, ", "))
		}
		return nil
	}
	return mdlerrors.NewNotFound("page", s.Page.Module+"."+s.Page.Name)
}

func execRevokeExistingPageFn(ctx context.Context, s *ast.RevokePageAccessStmt, pageID model.ID, deps *HandlerDeps) error {
	page, err := deps.PageRepo.Get(pageID)
	if err != nil {
		return mdlerrors.NewBackend("get page", err)
	}
	if page == nil {
		return mdlerrors.NewNotFound("page", s.Page.Module+"."+s.Page.Name)
	}
	existing := page.AllowedRolesQualifiedNames()
	remaining, removed := filterAllowedRoles(existing, s.Roles)
	if err := deps.SecurityEntityAccessManager.UpdateAllowedRoles(pageID, remaining); err != nil {
		return mdlerrors.NewBackend("update page access", err)
	}
	if len(removed) == 0 {
		fmt.Fprintf(deps.Output, "None of the specified roles had view access on %s.%s\n", s.Page.Module, s.Page.Name)
	} else {
		fmt.Fprintf(deps.Output, "Revoked view access on %s.%s from %s\n", s.Page.Module, s.Page.Name, strings.Join(removed, ", "))
	}
	return nil
}

func lookupCreatedPageIDFn(ctx context.Context, deps *HandlerDeps, qualifiedName string) (model.ID, error) {
	if deps.Cache == nil || deps.Cache.createdPages == nil {
		return "", nil
	}
	if info, ok := deps.Cache.createdPages[qualifiedName]; ok {
		return info.ID, nil
	}
	return "", nil
}

// execGrantPageAccessGen handles GRANT VIEW ON PAGE. Delegates to Fn version.

// execRevokePageAccessGen handles REVOKE VIEW ON PAGE. Delegates to Fn version.

// lookupCreatedPageID finds a session-created page by qualified name.
func lookupCreatedPageID(ctx *ExecContext, qualifiedName string) (model.ID, error) {
	return lookupCreatedPageIDFn(ctx, execContextToDeps(ctx), qualifiedName)
}

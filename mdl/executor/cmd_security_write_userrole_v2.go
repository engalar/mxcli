// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.1.D3 — gen-typed user role write paths.
//
// Mirrors execCreateUserRole / execAlterUserRole / execDropUserRole from
// cmd_security_write.go:177-288 but reads project security through the
// gen path (getProjectSecurityGen / invalidateProjectSecurityCache) and
// iterates user roles via ps.UserRolesItems() + (*genSec.UserRole) cast.
//
// Backend mutations (AddUserRole, AlterUserRoleModuleRoles, RemoveUserRole)
// are unchanged — they are already wired to gen via *ViaModelsdk.

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// execCreateUserRoleGen handles CREATE USER ROLE via the gen read path.
// Mirrors execCreateUserRole but iterates ps.UserRolesItems() and casts
// each entry to *genSec.UserRole to access typed accessors.
func execCreateUserRoleGen(ctx *ExecContext, s *ast.CreateUserRoleStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	ps, err := getProjectSecurityGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", nil)
	}

	// Build qualified module role names from the statement.
	var moduleRoleNames []string
	for _, mr := range s.ModuleRoles {
		qn := mr.Module + "." + mr.Name
		moduleRoleNames = append(moduleRoleNames, qn)
	}

	// Check if role already exists.
	for _, item := range ps.UserRolesItems() {
		ur, ok := item.(*genSec.UserRole)
		if !ok || ur == nil {
			continue
		}
		if ur.Name() == s.Name {
			if !s.CreateOrModify {
				return mdlerrors.NewAlreadyExists("user role", s.Name)
			}
			// Replace: remove existing module roles not in the new list, then add new ones.
			// This makes "create or modify user role" idempotent (replace semantics, not additive).
			existing := ur.ModuleRolesQualifiedNames()
			if err := ctx.Backend.AlterUserRoleModuleRoles(model.ID(ps.ID()), s.Name, false, existing); err != nil {
				return mdlerrors.NewBackend("clear user role module roles", err)
			}
			if err := ctx.Backend.AlterUserRoleModuleRoles(model.ID(ps.ID()), s.Name, true, moduleRoleNames); err != nil {
				return mdlerrors.NewBackend("update user role", err)
			}
			invalidateProjectSecurityCache(ctx)
			fmt.Fprintf(ctx.Output, "Modified user role: %s\n", s.Name)
			return nil
		}
	}

	if err := ctx.Backend.AddUserRole(model.ID(ps.ID()), s.Name, moduleRoleNames, s.ManageAllRoles); err != nil {
		return mdlerrors.NewBackend("create user role", err)
	}
	invalidateProjectSecurityCache(ctx)

	fmt.Fprintf(ctx.Output, "Created user role: %s\n", s.Name)
	return nil
}

// execAlterUserRoleGen handles ALTER USER ROLE Name ADD/REMOVE MODULE ROLES
// via the gen read path.
// Mirrors execAlterUserRole but iterates ps.UserRolesItems() + (*genSec.UserRole) cast.
func execAlterUserRoleGen(ctx *ExecContext, s *ast.AlterUserRoleStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	ps, err := getProjectSecurityGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", nil)
	}

	// Check user role exists.
	found := false
	for _, item := range ps.UserRolesItems() {
		ur, ok := item.(*genSec.UserRole)
		if !ok || ur == nil {
			continue
		}
		if ur.Name() == s.Name {
			found = true
			break
		}
	}
	if !found {
		return mdlerrors.NewNotFound("user role", s.Name)
	}

	// Build qualified module role names.
	var moduleRoleNames []string
	for _, mr := range s.ModuleRoles {
		moduleRoleNames = append(moduleRoleNames, mr.Module+"."+mr.Name)
	}

	if err := ctx.Backend.AlterUserRoleModuleRoles(model.ID(ps.ID()), s.Name, s.Add, moduleRoleNames); err != nil {
		return mdlerrors.NewBackend("alter user role", err)
	}
	invalidateProjectSecurityCache(ctx)

	action := "Added"
	prep := "to"
	if !s.Add {
		action = "Removed"
		prep = "from"
	}
	fmt.Fprintf(ctx.Output, "%s module roles %s %s user role %s\n", action, strings.Join(moduleRoleNames, ", "), prep, s.Name)
	return nil
}

// execDropUserRoleGen handles DROP USER ROLE Name via the gen read path.
// Mirrors execDropUserRole but iterates ps.UserRolesItems() + (*genSec.UserRole) cast.
func execDropUserRoleGen(ctx *ExecContext, s *ast.DropUserRoleStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	ps, err := getProjectSecurityGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", nil)
	}

	// Check user role exists.
	found := false
	for _, item := range ps.UserRolesItems() {
		ur, ok := item.(*genSec.UserRole)
		if !ok || ur == nil {
			continue
		}
		if ur.Name() == s.Name {
			found = true
			break
		}
	}
	if !found {
		return mdlerrors.NewNotFound("user role", s.Name)
	}

	if err := ctx.Backend.RemoveUserRole(model.ID(ps.ID()), s.Name); err != nil {
		return mdlerrors.NewBackend("drop user role", err)
	}
	invalidateProjectSecurityCache(ctx)

	fmt.Fprintf(ctx.Output, "Dropped user role: %s\n", s.Name)
	return nil
}

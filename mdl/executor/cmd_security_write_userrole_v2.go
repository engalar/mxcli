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
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// ExecCreateUserRoleGenFn is the HandlerDeps version of execCreateUserRoleGen.
func ExecCreateUserRoleGenFn(ctx context.Context, s *ast.CreateUserRoleStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	ps, err := getProjectSecurityGenDeps(deps)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", nil)
	}
	var moduleRoleNames []string
	for _, mr := range s.ModuleRoles {
		qn := mr.Module + "." + mr.Name
		moduleRoleNames = append(moduleRoleNames, qn)
	}
	for _, item := range ps.UserRolesItems() {
		ur, ok := item.(*genSec.UserRole)
		if !ok || ur == nil {
			continue
		}
		if ur.Name() == s.Name {
			if !s.CreateOrModify {
				return mdlerrors.NewAlreadyExists("user role", s.Name)
			}
			existing := ur.ModuleRolesQualifiedNames()
			if err := deps.SecurityProjectManager.AlterUserRoleModuleRoles(model.ID(ps.ID()), s.Name, false, existing); err != nil {
				return mdlerrors.NewBackend("clear user role module roles", err)
			}
			if err := deps.SecurityProjectManager.AlterUserRoleModuleRoles(model.ID(ps.ID()), s.Name, true, moduleRoleNames); err != nil {
				return mdlerrors.NewBackend("update user role", err)
			}
			if deps.Cache != nil {
				deps.Cache.projectSecurityGen = nil
			}
			fmt.Fprintf(deps.Output, "Modified user role: %s\n", s.Name)
			return nil
		}
	}
	if err := deps.SecurityProjectManager.AddUserRole(model.ID(ps.ID()), s.Name, moduleRoleNames, s.ManageAllRoles); err != nil {
		return mdlerrors.NewBackend("create user role", err)
	}
	if deps.Cache != nil {
		deps.Cache.projectSecurityGen = nil
	}
	fmt.Fprintf(deps.Output, "Created user role: %s\n", s.Name)
	return nil
}

// ExecAlterUserRoleGenFn is the HandlerDeps version of execAlterUserRoleGen.
func ExecAlterUserRoleGenFn(ctx context.Context, s *ast.AlterUserRoleStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	ps, err := getProjectSecurityGenDeps(deps)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", nil)
	}
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
	var moduleRoleNames []string
	for _, mr := range s.ModuleRoles {
		moduleRoleNames = append(moduleRoleNames, mr.Module+"."+mr.Name)
	}
	if err := deps.SecurityProjectManager.AlterUserRoleModuleRoles(model.ID(ps.ID()), s.Name, s.Add, moduleRoleNames); err != nil {
		return mdlerrors.NewBackend("alter user role", err)
	}
	if deps.Cache != nil {
		deps.Cache.projectSecurityGen = nil
	}
	action := "Added"
	prep := "to"
	if !s.Add {
		action = "Removed"
		prep = "from"
	}
	fmt.Fprintf(deps.Output, "%s module roles %s %s user role %s\n", action, strings.Join(moduleRoleNames, ", "), prep, s.Name)
	return nil
}

// ExecDropUserRoleGenFn is the HandlerDeps version of execDropUserRoleGen.
func ExecDropUserRoleGenFn(ctx context.Context, s *ast.DropUserRoleStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	ps, err := getProjectSecurityGenDeps(deps)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", nil)
	}
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
	if err := deps.SecurityProjectManager.RemoveUserRole(model.ID(ps.ID()), s.Name); err != nil {
		return mdlerrors.NewBackend("drop user role", err)
	}
	if deps.Cache != nil {
		deps.Cache.projectSecurityGen = nil
	}
	fmt.Fprintf(deps.Output, "Dropped user role: %s\n", s.Name)
	return nil
}

// execCreateUserRoleGen handles CREATE USER ROLE. Delegates to Fn version.

// execAlterUserRoleGen handles ALTER USER ROLE. Delegates to Fn version.

// execDropUserRoleGen handles DROP USER ROLE. Delegates to Fn version.

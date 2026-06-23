// SPDX-License-Identifier: Apache-2.0

package security

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// ─────────────────────────────────────────────────────────────
// Module Role
// ─────────────────────────────────────────────────────────────

const autoDocumentRoleDescription = "(Auto-document role for access rules)"

func ExecCreateModuleRoleFn(ctx context.Context, s *ast.CreateModuleRoleStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if d.SecurityModuleManager == nil || d.ModuleLister == nil || d.RenameManager == nil {
		return mdlerrors.NewBackend("backend not fully initialized", nil)
	}

	module, err := d.FindModule(s.Name.Module)
	if err != nil {
		return err
	}

	ms, err := d.SecurityModuleManager.GetModuleSecurityGen(module.ID)
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
			if err := d.SecurityModuleManager.RemoveModuleRole(model.ID(ms.ID()), typed.Name()); err != nil {
				return mdlerrors.NewBackend("remove auto-provisioned role", err)
			}
			if err := d.SecurityModuleManager.AddModuleRole(model.ID(ms.ID()), s.Name.Name, s.Description); err != nil {
				return mdlerrors.NewBackend("create module role", err)
			}
			if oldQualified != newQualified {
				if _, err := d.RenameManager.UpdateQualifiedNameInAllUnits(oldQualified, newQualified); err != nil {
					return mdlerrors.NewBackend(
						fmt.Sprintf("rename references %s -> %s", oldQualified, newQualified), err)
				}
			}
			d.InvalidateModuleSecurityCache()
			if !d.Quiet {
				fmt.Fprintf(d.Output, "Module role %s.%s already exists (auto-provisioned)\n",
					s.Name.Module, s.Name.Name)
			}
			return nil
		}
		if !d.Quiet {
			fmt.Fprintf(d.Output,
				"Module role %s.%s already exists.\nTo link it to a user role: ALTER USER ROLE \"User\" ADD MODULE ROLES (%s.\"%s\");\n",
				s.Name.Module, s.Name.Name, s.Name.Module, s.Name.Name)
		}
		return nil
	}

	if err := d.SecurityModuleManager.AddModuleRole(model.ID(ms.ID()), s.Name.Name, s.Description); err != nil {
		return mdlerrors.NewBackend("create module role", err)
	}

	d.InvalidateModuleSecurityCache()
	fmt.Fprintf(d.Output, "Created module role: %s.%s\n", s.Name.Module, s.Name.Name)
	return nil
}

func ExecDropModuleRoleFn(ctx context.Context, s *ast.DropModuleRoleStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	module, err := d.FindModule(s.Name.Module)
	if err != nil {
		return err
	}

	ms, err := d.SecurityModuleManager.GetModuleSecurityGen(module.ID)
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

	if dm, err := d.GetDomainModelGenCached(module.ID); err == nil && dm != nil {
		if n, err := d.SecurityEntityAccessManager.RemoveRoleFromAllEntities(model.ID(dm.ID()), qualifiedRole); err != nil {
			return mdlerrors.NewBackend("cascade-remove entity access rules", err)
		} else if n > 0 {
			fmt.Fprintf(d.Output, "Removed %s from %d entity access rule(s)\n", qualifiedRole, n)
			d.InvalidateDomainModelGenForModule(module.ID)
		}
	}

	if err := d.CascadeRemoveRoleFromMicroflows(module.ID, qualifiedRole); err != nil {
		return err
	}
	if err := d.CascadeRemoveRoleFromNanoflows(module.ID, qualifiedRole); err != nil {
		return err
	}

	pgPairs, err := d.PagesRepo.ListAll()
	if err == nil {
		for _, pg := range pgPairs {
			containerID, cerr := d.PagesRepo.GetContainerUUID(model.ID(pg.ID()))
			if cerr != nil {
				continue
			}
			modID := d.FindModuleID(containerID)
			if modID != module.ID {
				continue
			}
			if removed, err := d.SecurityEntityAccessManager.RemoveFromAllowedRoles(model.ID(pg.ID()), qualifiedRole); err == nil && removed {
				fmt.Fprintf(d.Output, "Removed %s from page %s allowed roles\n", qualifiedRole, pg.Name())
			}
		}
	}

	if svcs, err := d.ServiceLister.ListPublishedODataServices(); err == nil {
		for _, svc := range svcs {
			modID := d.FindModuleID(svc.ContainerID)
			if modID != module.ID {
				continue
			}
			if removed, err := d.SecurityEntityAccessManager.RemoveFromAllowedRoles(svc.ID, qualifiedRole); err == nil && removed {
				fmt.Fprintf(d.Output, "Removed %s from OData service %s allowed roles\n", qualifiedRole, svc.Name)
			}
		}
	}

	if ps, err := d.GetProjectSecurityGen(); err == nil && ps != nil {
		if n, err := d.SecurityModuleManager.RemoveModuleRoleFromAllUserRoles(model.ID(ps.ID()), qualifiedRole); err == nil && n > 0 {
			fmt.Fprintf(d.Output, "Removed %s from %d user role(s)\n", qualifiedRole, n)
		}
		if err := d.PruneInvalidUserRoles(nil); err != nil {
			return mdlerrors.NewBackend("cleanup invalid user roles", err)
		}
	}

	if err := d.SecurityModuleManager.RemoveModuleRole(model.ID(ms.ID()), s.Name.Name); err != nil {
		return mdlerrors.NewBackend("drop module role", err)
	}

	d.InvalidateModuleSecurityCache()
	fmt.Fprintf(d.Output, "Dropped module role: %s.%s\n", s.Name.Module, s.Name.Name)
	return nil
}

// ─────────────────────────────────────────────────────────────
// User Role
// ─────────────────────────────────────────────────────────────

func ExecCreateUserRoleFn(ctx context.Context, s *ast.CreateUserRoleStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	ps, err := d.GetProjectSecurityGen()
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
			if err := d.SecurityProjectManager.AlterUserRoleModuleRoles(model.ID(ps.ID()), s.Name, false, existing); err != nil {
				return mdlerrors.NewBackend("clear user role module roles", err)
			}
			if err := d.SecurityProjectManager.AlterUserRoleModuleRoles(model.ID(ps.ID()), s.Name, true, moduleRoleNames); err != nil {
				return mdlerrors.NewBackend("update user role", err)
			}
			d.InvalidateProjectSecurityCache()
			fmt.Fprintf(d.Output, "Modified user role: %s\n", s.Name)
			return nil
		}
	}
	if err := d.SecurityProjectManager.AddUserRole(model.ID(ps.ID()), s.Name, moduleRoleNames, s.ManageAllRoles); err != nil {
		return mdlerrors.NewBackend("create user role", err)
	}
	d.InvalidateProjectSecurityCache()
	fmt.Fprintf(d.Output, "Created user role: %s\n", s.Name)
	return nil
}

func ExecAlterUserRoleFn(ctx context.Context, s *ast.AlterUserRoleStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	ps, err := d.GetProjectSecurityGen()
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
	if err := d.SecurityProjectManager.AlterUserRoleModuleRoles(model.ID(ps.ID()), s.Name, s.Add, moduleRoleNames); err != nil {
		return mdlerrors.NewBackend("alter user role", err)
	}
	d.InvalidateProjectSecurityCache()
	action := "Added"
	prep := "to"
	if !s.Add {
		action = "Removed"
		prep = "from"
	}
	fmt.Fprintf(d.Output, "%s module roles %s %s user role %s\n", action, strings.Join(moduleRoleNames, ", "), prep, s.Name)
	return nil
}

func ExecDropUserRoleFn(ctx context.Context, s *ast.DropUserRoleStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	ps, err := d.GetProjectSecurityGen()
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
	if err := d.SecurityProjectManager.RemoveUserRole(model.ID(ps.ID()), s.Name); err != nil {
		return mdlerrors.NewBackend("drop user role", err)
	}
	d.InvalidateProjectSecurityCache()
	fmt.Fprintf(d.Output, "Dropped user role: %s\n", s.Name)
	return nil
}

// ─────────────────────────────────────────────────────────────
// Entity Access (GRANT / REVOKE)
// ─────────────────────────────────────────────────────────────

func ExecGrantEntityAccessFn(ctx context.Context, s *ast.GrantEntityAccessStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	module, err := d.FindModule(s.Entity.Module)
	if err != nil {
		return err
	}

	dm, err := d.GetDomainModelGenCached(module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewNotFound("entity", s.Entity.Module+"."+s.Entity.Name)
	}

	entity, _, err := d.FindEntityGen(s.Entity)
	if err != nil {
		return mdlerrors.NewBackend("find entity", err)
	}
	if entity == nil {
		return mdlerrors.NewNotFound("entity", s.Entity.Module+"."+s.Entity.Name)
	}

	var roleNames []string
	for _, role := range s.Roles {
		found, err := d.ValidateModuleRole(role)
		if err != nil {
			return err
		}
		if found {
			roleNames = append(roleNames, role.Module+"."+role.Name)
		}
	}
	if len(roleNames) == 0 {
		return nil
	}

	allowCreate, allowDelete := false, false
	defaultMemberAccess := "None"
	var readMembers, writeMembers []string
	for _, right := range s.Rights {
		switch right.Type {
		case ast.EntityAccessCreate:
			allowCreate = true
		case ast.EntityAccessDelete:
			allowDelete = true
		case ast.EntityAccessReadAll:
			if defaultMemberAccess == "None" {
				defaultMemberAccess = "ReadOnly"
			}
		case ast.EntityAccessReadMembers:
			readMembers = right.Members
		case ast.EntityAccessWriteAll:
			defaultMemberAccess = "ReadWrite"
		case ast.EntityAccessWriteMembers:
			writeMembers = right.Members
		}
	}

	var memberAccesses []types.EntityMemberAccess

	memberSetFrom := func(members []string) map[string]bool {
		s := make(map[string]bool, len(members)*2)
		for _, m := range members {
			s[m] = true
			if idx := strings.LastIndex(m, "."); idx >= 0 {
				s[m[idx+1:]] = true
			}
		}
		return s
	}
	writeMemberSet := memberSetFrom(writeMembers)
	readMemberSet := memberSetFrom(readMembers)

	for _, attrElem := range entity.AttributesItems() {
		attr, ok := attrElem.(*genDm.Attribute)
		if !ok {
			continue
		}
		rights := defaultMemberAccess
		if writeMemberSet[attr.Name()] {
			rights = "ReadWrite"
		} else if readMemberSet[attr.Name()] {
			rights = "ReadOnly"
		}
		isCalculated := false
		if val := attr.Value(); val != nil && val.TypeName() == "DomainModels$CalculatedValue" {
			isCalculated = true
		}
		if isCalculated && (rights == "ReadWrite" || rights == "WriteOnly") {
			rights = "ReadOnly"
		}
		memberAccesses = append(memberAccesses, types.EntityMemberAccess{
			AttributeRef: module.Name + "." + s.Entity.Name + "." + attr.Name(),
			AccessRights: rights,
		})
	}

	for _, assocElem := range dm.AssociationsItems() {
		assoc, ok := assocElem.(*genDm.Association)
		if !ok || model.ID(assoc.ParentRefID()) != model.ID(entity.ID()) {
			continue
		}
		rights := defaultMemberAccess
		if writeMemberSet[assoc.Name()] {
			rights = "ReadWrite"
		} else if readMemberSet[assoc.Name()] {
			rights = "ReadOnly"
		}
		memberAccesses = append(memberAccesses, types.EntityMemberAccess{
			AssociationRef: module.Name + "." + assoc.Name(),
			AccessRights:   rights,
		})
	}
	for _, crossElem := range dm.CrossAssociationsItems() {
		ca, ok := crossElem.(*genDm.CrossAssociation)
		if !ok || model.ID(ca.ParentRefID()) != model.ID(entity.ID()) {
			continue
		}
		rights := defaultMemberAccess
		if writeMemberSet[ca.Name()] {
			rights = "ReadWrite"
		} else if readMemberSet[ca.Name()] {
			rights = "ReadOnly"
		}
		memberAccesses = append(memberAccesses, types.EntityMemberAccess{
			AssociationRef: module.Name + "." + ca.Name(),
			AccessRights:   rights,
		})
	}

	if ng, ok := entity.Generalization().(*genDm.NoGeneralization); ok && ng.HasOwner() {
		memberAccesses = append(memberAccesses, types.EntityMemberAccess{
			AssociationRef: "System.owner",
			AccessRights:   defaultMemberAccess,
		})
	}
	if ng, ok := entity.Generalization().(*genDm.NoGeneralization); ok && ng.HasChangedBy() {
		memberAccesses = append(memberAccesses, types.EntityMemberAccess{
			AssociationRef: "System.changedBy",
			AccessRights:   defaultMemberAccess,
		})
	}

	if err := d.SecurityEntityAccessManager.AddEntityAccessRule(backend.EntityAccessRuleParams{
		UnitID:              model.ID(dm.ID()),
		EntityName:          s.Entity.Name,
		RoleNames:           roleNames,
		AllowCreate:         allowCreate,
		AllowDelete:         allowDelete,
		DefaultMemberAccess: defaultMemberAccess,
		XPathConstraint:     s.XPathConstraint,
		MemberAccesses:      memberAccesses,
	}); err != nil {
		return mdlerrors.NewBackend("grant entity access", err)
	}

	d.InvalidateDomainModelGenForModule(module.ID)
	d.InvalidateDomainModelsCache()

	if msgs, err := d.SecurityEntityAccessManager.ReconcileMemberAccesses(model.ID(dm.ID()), module.Name); err != nil {
		return mdlerrors.NewBackend("reconcile member accesses", err)
	} else if len(msgs) > 0 && !d.Quiet {
		for _, msg := range msgs {
			fmt.Fprintf(d.Output, "  reconciled: %s\n", msg)
		}
	}

	d.TrackModifiedDomainModel(module.ID, module.Name)
	fmt.Fprintf(d.Output, "Granted access on %s.%s to %s\n", s.Entity.Module, s.Entity.Name, strings.Join(roleNames, ", "))
	if !d.Quiet {
		fmt.Fprint(d.Output, d.FormatAccessRuleResult(s.Entity.Module, s.Entity.Name, roleNames))
	}
	return nil
}

func ExecRevokeEntityAccessFn(ctx context.Context, s *ast.RevokeEntityAccessStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	module, err := d.FindModule(s.Entity.Module)
	if err != nil {
		return err
	}

	dm, err := d.GetDomainModelGenCached(module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewNotFound("entity", s.Entity.Module+"."+s.Entity.Name)
	}

	entity, _, err := d.FindEntityGen(s.Entity)
	if err != nil {
		return mdlerrors.NewBackend("find entity", err)
	}
	if entity == nil {
		return mdlerrors.NewNotFound("entity", s.Entity.Module+"."+s.Entity.Name)
	}

	var roleNames []string
	for _, role := range s.Roles {
		found, err := d.ValidateModuleRole(role)
		if err != nil {
			return err
		}
		if found {
			roleNames = append(roleNames, role.Module+"."+role.Name)
		}
	}
	if len(roleNames) == 0 {
		return nil
	}

	if len(s.Rights) > 0 {
		revocation := types.EntityAccessRevocation{}
		for _, right := range s.Rights {
			switch right.Type {
			case ast.EntityAccessCreate:
				revocation.RevokeCreate = true
			case ast.EntityAccessDelete:
				revocation.RevokeDelete = true
			case ast.EntityAccessReadAll:
				revocation.RevokeReadAll = true
			case ast.EntityAccessWriteAll:
				revocation.RevokeWriteAll = true
			case ast.EntityAccessReadMembers:
				for _, m := range right.Members {
					revocation.RevokeReadMembers = append(revocation.RevokeReadMembers,
						module.Name+"."+s.Entity.Name+"."+m)
				}
			case ast.EntityAccessWriteMembers:
				for _, m := range right.Members {
					revocation.RevokeWriteMembers = append(revocation.RevokeWriteMembers,
						module.Name+"."+s.Entity.Name+"."+m)
				}
			}
		}

		modified, err := d.SecurityEntityAccessManager.RevokeEntityMemberAccess(model.ID(dm.ID()), s.Entity.Name, roleNames, revocation)
		if err != nil {
			return mdlerrors.NewBackend("revoke entity access", err)
		}

		if modified == 0 {
			fmt.Fprintf(d.Output, "No access rules found matching %s on %s.%s\n",
				strings.Join(roleNames, ", "), s.Entity.Module, s.Entity.Name)
		} else {
			fmt.Fprintf(d.Output, "Revoked partial access on %s.%s from %s\n",
				s.Entity.Module, s.Entity.Name, strings.Join(roleNames, ", "))
			if !d.Quiet {
				fmt.Fprint(d.Output, d.FormatAccessRuleResult(s.Entity.Module, s.Entity.Name, roleNames))
			}
		}
	} else {
		modified, err := d.SecurityEntityAccessManager.RemoveEntityAccessRule(model.ID(dm.ID()), s.Entity.Name, roleNames)
		if err != nil {
			return mdlerrors.NewBackend("revoke entity access", err)
		}

		if modified == 0 {
			fmt.Fprintf(d.Output, "No access rules found matching %s on %s.%s\n",
				strings.Join(roleNames, ", "), s.Entity.Module, s.Entity.Name)
		} else {
			fmt.Fprintf(d.Output, "Revoked access on %s.%s from %s\n",
				s.Entity.Module, s.Entity.Name, strings.Join(roleNames, ", "))
			if !d.Quiet {
				fmt.Fprint(d.Output, "  Result: (no access)\n")
			}
		}
	}

	d.TrackModifiedDomainModel(module.ID, module.Name)
	return nil
}

// ─────────────────────────────────────────────────────────────
// Page Access (GRANT / REVOKE)
// ─────────────────────────────────────────────────────────────

func ExecGrantPageAccessFn(ctx context.Context, s *ast.GrantPageAccessStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	pageID, err := d.LookupCreatedPageID(s.Page.Module + "." + s.Page.Name)
	if err == nil && pageID != "" {
		return execGrantExistingPageFn(s, pageID, d)
	}

	pgPairs, err := d.PagesRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}
	for _, pg := range pgPairs {
		containerID, cerr := d.PagesRepo.GetContainerUUID(model.ID(pg.ID()))
		if cerr != nil {
			continue
		}
		modID := d.FindModuleID(containerID)
		modName := d.FindModuleName(modID)
		if modName != s.Page.Module || pg.Name() != s.Page.Name {
			continue
		}
		return execGrantExistingPageFn(s, model.ID(pg.ID()), d)
	}
	return mdlerrors.NewNotFound("page", s.Page.Module+"."+s.Page.Name)
}

func execGrantExistingPageFn(s *ast.GrantPageAccessStmt, pageID model.ID, d SecurityDeps) error {
	var validRoles []ast.QualifiedName
	for _, role := range s.Roles {
		found, err := d.ValidateModuleRole(role)
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
	page, err := d.PagesRepo.Get(pageID)
	if err != nil {
		return mdlerrors.NewBackend("get page", err)
	}
	if page == nil {
		return mdlerrors.NewNotFound("page", s.Page.Module+"."+s.Page.Name)
	}
	existing := d.FilterAutoDocumentRoles(page.AllowedRolesQualifiedNames())
	merged, added := d.MergeAllowedRoles(existing, validRoles)
	if err := d.SecurityEntityAccessManager.UpdateAllowedRoles(pageID, merged); err != nil {
		return mdlerrors.NewBackend("update page access", err)
	}
	if len(added) == 0 {
		fmt.Fprintf(d.Output, "All specified roles already have view access on %s.%s\n", s.Page.Module, s.Page.Name)
	} else {
		fmt.Fprintf(d.Output, "Granted view access on %s.%s to %s\n", s.Page.Module, s.Page.Name, strings.Join(added, ", "))
	}
	return nil
}

func ExecRevokePageAccessFn(ctx context.Context, s *ast.RevokePageAccessStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	pageID, err := d.LookupCreatedPageID(s.Page.Module + "." + s.Page.Name)
	if err == nil && pageID != "" {
		return execRevokeExistingPageFn(s, pageID, d)
	}

	pgPairs, err := d.PagesRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}
	for _, pg := range pgPairs {
		containerID, cerr := d.PagesRepo.GetContainerUUID(model.ID(pg.ID()))
		if cerr != nil {
			continue
		}
		modID := d.FindModuleID(containerID)
		modName := d.FindModuleName(modID)
		if modName != s.Page.Module || pg.Name() != s.Page.Name {
			continue
		}
		existing := pg.AllowedRolesQualifiedNames()
		remaining, removed := d.FilterAllowedRoles(existing, s.Roles)
		if err := d.SecurityEntityAccessManager.UpdateAllowedRoles(model.ID(pg.ID()), remaining); err != nil {
			return mdlerrors.NewBackend("update page access", err)
		}
		if len(removed) == 0 {
			fmt.Fprintf(d.Output, "None of the specified roles had view access on %s.%s\n", modName, pg.Name())
		} else {
			fmt.Fprintf(d.Output, "Revoked view access on %s.%s from %s\n", modName, pg.Name(), strings.Join(removed, ", "))
		}
		return nil
	}
	return mdlerrors.NewNotFound("page", s.Page.Module+"."+s.Page.Name)
}

func execRevokeExistingPageFn(s *ast.RevokePageAccessStmt, pageID model.ID, d SecurityDeps) error {
	page, err := d.PagesRepo.Get(pageID)
	if err != nil {
		return mdlerrors.NewBackend("get page", err)
	}
	if page == nil {
		return mdlerrors.NewNotFound("page", s.Page.Module+"."+s.Page.Name)
	}
	existing := page.AllowedRolesQualifiedNames()
	remaining, removed := d.FilterAllowedRoles(existing, s.Roles)
	if err := d.SecurityEntityAccessManager.UpdateAllowedRoles(pageID, remaining); err != nil {
		return mdlerrors.NewBackend("update page access", err)
	}
	if len(removed) == 0 {
		fmt.Fprintf(d.Output, "None of the specified roles had view access on %s.%s\n", s.Page.Module, s.Page.Name)
	} else {
		fmt.Fprintf(d.Output, "Revoked view access on %s.%s from %s\n", s.Page.Module, s.Page.Name, strings.Join(removed, ", "))
	}
	return nil
}

// ─────────────────────────────────────────────────────────────
// Microflow / Nanoflow Access (GRANT / REVOKE)
// ─────────────────────────────────────────────────────────────

func ExecGrantMicroflowAccessFn(ctx context.Context, s *ast.GrantMicroflowAccessStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if d.MicroflowsRepo == nil {
		return mdlerrors.NewBackend("microflows repo unavailable", nil)
	}

	mfs, err := d.MicroflowsRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list microflows", err)
	}

	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		modName := genFlowContainerModuleFn(d.MicroflowsRepo, d.FindModuleID, d.FindModuleName, model.ID(mf.ID()))
		if modName != s.Microflow.Module || mf.Name() != s.Microflow.Name {
			continue
		}

		var validRoles []ast.QualifiedName
		for _, role := range s.Roles {
			resolved := expandRoleModule(role, modName)
			found, err := d.ValidateModuleRole(resolved)
			if err != nil {
				return err
			}
			if found {
				validRoles = append(validRoles, resolved)
			}
		}
		if len(validRoles) == 0 {
			return nil
		}

		merged, added := d.MergeAllowedRoles(d.FilterAutoDocumentRoles(mf.AllowedModuleRolesQualifiedNames()), validRoles)
		mf.SetAllowedModuleRolesQualifiedNames(merged)

		if err := d.MicroflowsRepo.Update(mf); err != nil {
			return mdlerrors.NewBackend("update microflow access", err)
		}

		if len(added) == 0 {
			fmt.Fprintf(d.Output, "All specified roles already have execute access on %s.%s\n", modName, mf.Name())
		} else {
			fmt.Fprintf(d.Output, "Granted execute access on %s.%s to %s\n", modName, mf.Name(), strings.Join(added, ", "))
		}
		return nil
	}

	return mdlerrors.NewNotFound("microflow", s.Microflow.Module+"."+s.Microflow.Name)
}

func ExecRevokeMicroflowAccessFn(ctx context.Context, s *ast.RevokeMicroflowAccessStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if d.MicroflowsRepo == nil {
		return mdlerrors.NewBackend("microflows repo unavailable", nil)
	}

	mfs, err := d.MicroflowsRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list microflows", err)
	}

	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		modName := genFlowContainerModuleFn(d.MicroflowsRepo, d.FindModuleID, d.FindModuleName, model.ID(mf.ID()))
		if modName != s.Microflow.Module || mf.Name() != s.Microflow.Name {
			continue
		}

		expandedRoles := make([]ast.QualifiedName, len(s.Roles))
		for i, r := range s.Roles {
			expandedRoles[i] = expandRoleModule(r, modName)
		}
		remaining, removed := d.FilterAllowedRoles(mf.AllowedModuleRolesQualifiedNames(), expandedRoles)
		mf.SetAllowedModuleRolesQualifiedNames(remaining)

		if err := d.MicroflowsRepo.Update(mf); err != nil {
			return mdlerrors.NewBackend("update microflow access", err)
		}

		if len(removed) == 0 {
			fmt.Fprintf(d.Output, "None of the specified roles had execute access on %s.%s\n", modName, mf.Name())
		} else {
			fmt.Fprintf(d.Output, "Revoked execute access on %s.%s from %s\n", modName, mf.Name(), strings.Join(removed, ", "))
		}
		return nil
	}

	return mdlerrors.NewNotFound("microflow", s.Microflow.Module+"."+s.Microflow.Name)
}

func ExecGrantNanoflowAccessFn(ctx context.Context, s *ast.GrantNanoflowAccessStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if d.NanoflowsRepo == nil {
		return mdlerrors.NewBackend("nanoflows repo unavailable", nil)
	}

	nfs, err := d.NanoflowsRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list nanoflows", err)
	}

	for _, nf := range nfs {
		if nf == nil {
			continue
		}
		modName := genFlowContainerModuleFn(d.NanoflowsRepo, d.FindModuleID, d.FindModuleName, model.ID(nf.ID()))
		if modName != s.Nanoflow.Module || nf.Name() != s.Nanoflow.Name {
			continue
		}

		var validRoles []ast.QualifiedName
		for _, role := range s.Roles {
			resolved := expandRoleModule(role, modName)
			found, err := d.ValidateModuleRole(resolved)
			if err != nil {
				return err
			}
			if found {
				validRoles = append(validRoles, resolved)
			}
		}
		if len(validRoles) == 0 {
			return nil
		}

		merged, added := d.MergeAllowedRoles(d.FilterAutoDocumentRoles(nf.AllowedModuleRolesQualifiedNames()), validRoles)
		nf.SetAllowedModuleRolesQualifiedNames(merged)

		if err := d.NanoflowsRepo.Update(nf); err != nil {
			return mdlerrors.NewBackend("update nanoflow access", err)
		}

		if len(added) == 0 {
			fmt.Fprintf(d.Output, "All specified roles already have execute access on %s.%s\n", modName, nf.Name())
		} else {
			fmt.Fprintf(d.Output, "Granted execute access on %s.%s to %s\n", modName, nf.Name(), strings.Join(added, ", "))
		}
		return nil
	}

	return mdlerrors.NewNotFound("nanoflow", s.Nanoflow.Module+"."+s.Nanoflow.Name)
}

func ExecRevokeNanoflowAccessFn(ctx context.Context, s *ast.RevokeNanoflowAccessStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if d.NanoflowsRepo == nil {
		return mdlerrors.NewBackend("nanoflows repo unavailable", nil)
	}

	nfs, err := d.NanoflowsRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list nanoflows", err)
	}

	for _, nf := range nfs {
		if nf == nil {
			continue
		}
		modName := genFlowContainerModuleFn(d.NanoflowsRepo, d.FindModuleID, d.FindModuleName, model.ID(nf.ID()))
		if modName != s.Nanoflow.Module || nf.Name() != s.Nanoflow.Name {
			continue
		}

		expandedRoles := make([]ast.QualifiedName, len(s.Roles))
		for i, r := range s.Roles {
			expandedRoles[i] = expandRoleModule(r, modName)
		}
		remaining, removed := d.FilterAllowedRoles(nf.AllowedModuleRolesQualifiedNames(), expandedRoles)
		nf.SetAllowedModuleRolesQualifiedNames(remaining)

		if err := d.NanoflowsRepo.Update(nf); err != nil {
			return mdlerrors.NewBackend("update nanoflow access", err)
		}

		if len(removed) == 0 {
			fmt.Fprintf(d.Output, "None of the specified roles had execute access on %s.%s\n", modName, nf.Name())
		} else {
			fmt.Fprintf(d.Output, "Revoked execute access on %s.%s from %s\n", modName, nf.Name(), strings.Join(removed, ", "))
		}
		return nil
	}

	return mdlerrors.NewNotFound("nanoflow", s.Nanoflow.Module+"."+s.Nanoflow.Name)
}

// ─────────────────────────────────────────────────────────────
// OData / Published REST Service Access
// ─────────────────────────────────────────────────────────────

func ExecGrantODataServiceAccessFn(ctx context.Context, s *ast.GrantODataServiceAccessStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	services, err := d.ServiceLister.ListPublishedODataServices()
	if err != nil {
		return mdlerrors.NewBackend("list published OData services", err)
	}
	for _, svc := range services {
		modID := d.FindModuleID(svc.ContainerID)
		modName := d.FindModuleName(modID)
		if modName != s.Service.Module || svc.Name != s.Service.Name {
			continue
		}
		var validRoles []ast.QualifiedName
		for _, role := range s.Roles {
			found, err := d.ValidateModuleRole(role)
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
		merged, added := d.MergeAllowedRoles(svc.AllowedModuleRoles, validRoles)
		if err := d.SecurityEntityAccessManager.UpdateAllowedRoles(svc.ID, merged); err != nil {
			return mdlerrors.NewBackend("update OData service access", err)
		}
		if len(added) == 0 {
			fmt.Fprintf(d.Output, "All specified roles already have access on OData service %s.%s\n", modName, svc.Name)
		} else {
			fmt.Fprintf(d.Output, "Granted access on OData service %s.%s to %s\n", modName, svc.Name, strings.Join(added, ", "))
		}
		return nil
	}
	return mdlerrors.NewNotFound("published OData service", s.Service.Module+"."+s.Service.Name)
}

func ExecRevokeODataServiceAccessFn(ctx context.Context, s *ast.RevokeODataServiceAccessStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	services, err := d.ServiceLister.ListPublishedODataServices()
	if err != nil {
		return mdlerrors.NewBackend("list published OData services", err)
	}
	for _, svc := range services {
		modID := d.FindModuleID(svc.ContainerID)
		modName := d.FindModuleName(modID)
		if modName != s.Service.Module || svc.Name != s.Service.Name {
			continue
		}
		remaining, removed := d.FilterAllowedRoles(svc.AllowedModuleRoles, s.Roles)
		if err := d.SecurityEntityAccessManager.UpdateAllowedRoles(svc.ID, remaining); err != nil {
			return mdlerrors.NewBackend("update OData service access", err)
		}
		if len(removed) == 0 {
			fmt.Fprintf(d.Output, "None of the specified roles had access on OData service %s.%s\n", modName, svc.Name)
		} else {
			fmt.Fprintf(d.Output, "Revoked access on OData service %s.%s from %s\n", modName, svc.Name, strings.Join(removed, ", "))
		}
		return nil
	}
	return mdlerrors.NewNotFound("published OData service", s.Service.Module+"."+s.Service.Name)
}

func ExecGrantPublishedRestServiceAccessFn(ctx context.Context, s *ast.GrantPublishedRestServiceAccessStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if err := d.CheckFeature("integration", "published_rest_grant_revoke",
		"grant access on published rest service",
		"upgrade your project to 10.0+"); err != nil {
		return err
	}
	services, err := d.ServiceLister.ListPublishedRestServices()
	if err != nil {
		return mdlerrors.NewBackend("list published rest services", err)
	}
	for _, svc := range services {
		modID := d.FindModuleID(svc.ContainerID)
		modName := d.FindModuleName(modID)
		if modName != s.Service.Module || svc.Name != s.Service.Name {
			continue
		}
		var validRoles []ast.QualifiedName
		for _, role := range s.Roles {
			found, err := d.ValidateModuleRole(role)
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
		merged, added := d.MergeAllowedRoles(svc.AllowedRoles, validRoles)
		if err := d.SecurityEntityAccessManager.UpdatePublishedRestServiceRoles(svc.ID, merged); err != nil {
			return mdlerrors.NewBackend("update published rest service access", err)
		}
		if len(added) == 0 {
			fmt.Fprintf(d.Output, "All specified roles already have access on published rest service %s.%s\n", modName, svc.Name)
		} else {
			fmt.Fprintf(d.Output, "Granted access on published rest service %s.%s to %s\n", modName, svc.Name, strings.Join(added, ", "))
		}
		return nil
	}
	return mdlerrors.NewNotFound("published rest service", s.Service.Module+"."+s.Service.Name)
}

func ExecRevokePublishedRestServiceAccessFn(ctx context.Context, s *ast.RevokePublishedRestServiceAccessStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	services, err := d.ServiceLister.ListPublishedRestServices()
	if err != nil {
		return mdlerrors.NewBackend("list published rest services", err)
	}
	for _, svc := range services {
		modID := d.FindModuleID(svc.ContainerID)
		modName := d.FindModuleName(modID)
		if modName != s.Service.Module || svc.Name != s.Service.Name {
			continue
		}
		remaining, removed := d.FilterAllowedRoles(svc.AllowedRoles, s.Roles)
		if err := d.SecurityEntityAccessManager.UpdatePublishedRestServiceRoles(svc.ID, remaining); err != nil {
			return mdlerrors.NewBackend("update published rest service access", err)
		}
		if len(removed) == 0 {
			fmt.Fprintf(d.Output, "None of the specified roles had access on published rest service %s.%s\n", modName, svc.Name)
		} else {
			fmt.Fprintf(d.Output, "Revoked access on published rest service %s.%s from %s\n", modName, svc.Name, strings.Join(removed, ", "))
		}
		return nil
	}
	return mdlerrors.NewNotFound("published rest service", s.Service.Module+"."+s.Service.Name)
}

// ─────────────────────────────────────────────────────────────
// Project Security
// ─────────────────────────────────────────────────────────────

func ExecAlterProjectSecurityFn(ctx context.Context, s *ast.AlterProjectSecurityStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	ps, err := d.GetProjectSecurityGen()
	if err != nil || ps == nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if s.SecurityLevel != "" {
		var bsonLevel string
		switch s.SecurityLevel {
		case "Production":
			bsonLevel = "CheckEverything"
		case "Prototype":
			bsonLevel = "CheckFormsAndMicroflows"
		case "Off":
			bsonLevel = "CheckNothing"
		default:
			return mdlerrors.NewUnsupported(fmt.Sprintf("unknown security level: %s", s.SecurityLevel))
		}
		if err := d.SecurityProjectManager.SetProjectSecurityLevel(model.ID(ps.ID()), bsonLevel); err != nil {
			return mdlerrors.NewBackend("set security level", err)
		}
		d.InvalidateProjectSecurityCache()
		fmt.Fprintf(d.Output, "Set project security level to %s\n", s.SecurityLevel)
	}
	if s.DemoUsersEnabled != nil {
		if err := d.SecurityProjectManager.SetProjectDemoUsersEnabled(model.ID(ps.ID()), *s.DemoUsersEnabled); err != nil {
			return mdlerrors.NewBackend("set demo users", err)
		}
		d.InvalidateProjectSecurityCache()
		state := "disabled"
		if *s.DemoUsersEnabled {
			state = "enabled"
		}
		fmt.Fprintf(d.Output, "Demo users %s\n", state)
	}
	if s.PasswordPolicy != nil {
		return execAlterPasswordPolicyFn(ctx, s, d)
	}
	return nil
}

func execAlterPasswordPolicyFn(ctx context.Context, s *ast.AlterProjectSecurityStmt, d SecurityDeps) error {
	opts := s.PasswordPolicy
	if opts == nil {
		return nil
	}
	ps, err := d.GetProjectSecurityGen()
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", fmt.Errorf("ProjectSecurity not found"))
	}
	if err := d.SecurityProjectManager.SetPasswordPolicy(
		model.ID(ps.ID()),
		opts.MinLength,
		opts.RequireDigit,
		opts.RequireMixedCase,
		opts.RequireSymbol,
	); err != nil {
		return mdlerrors.NewBackend("set password policy", err)
	}
	d.InvalidateProjectSecurityCache()
	fmt.Fprintf(d.Output, "Updated password policy")
	if opts.MinLength != nil {
		fmt.Fprintf(d.Output, ": min_length=%d", *opts.MinLength)
	}
	if opts.RequireDigit != nil {
		fmt.Fprintf(d.Output, ", require_digit=%v", *opts.RequireDigit)
	}
	if opts.RequireMixedCase != nil {
		fmt.Fprintf(d.Output, ", require_mixed_case=%v", *opts.RequireMixedCase)
	}
	if opts.RequireSymbol != nil {
		fmt.Fprintf(d.Output, ", require_symbol=%v", *opts.RequireSymbol)
	}
	fmt.Fprintln(d.Output)
	return nil
}

// ─────────────────────────────────────────────────────────────
// Update Security (Reconcile)
// ─────────────────────────────────────────────────────────────

func ExecUpdateSecurityFn(ctx context.Context, s *ast.UpdateSecurityStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	modules, err := d.GetModulesFromCache()
	if err != nil {
		return err
	}
	totalModified := 0
	for _, mod := range modules {
		if s.Module != "" && mod.Name != s.Module {
			continue
		}
		dm, err := d.GetDomainModelGenCached(mod.ID)
		if err != nil || dm == nil {
			continue
		}
		msgs, err := d.SecurityEntityAccessManager.ReconcileMemberAccesses(model.ID(dm.ID()), mod.Name)
		if err != nil {
			return mdlerrors.NewBackend(fmt.Sprintf("reconcile security for module %s", mod.Name), err)
		}
		if len(msgs) > 0 {
			d.InvalidateDomainModelGenForModule(mod.ID)
		}
		for _, msg := range msgs {
			fmt.Fprintf(d.Output, "  [%s] %s\n", mod.Name, msg)
			totalModified++
		}
	}
	if totalModified == 0 {
		fmt.Fprintf(d.Output, "All entity access rules are up to date\n")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────
// Demo Users
// ─────────────────────────────────────────────────────────────

func ExecCreateDemoUserFn(ctx context.Context, s *ast.CreateDemoUserStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	ps, err := d.GetProjectSecurityGen()
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", fmt.Errorf("ProjectSecurity not found"))
	}
	if raw := ps.PasswordPolicySettings(); raw != nil {
		if pp, ok := raw.(*genSec.PasswordPolicySettings); ok && pp != nil {
			if err := validatePasswordPolicy(s.UserName, s.Password, pp); err != nil {
				return err
			}
		}
	}
	for _, du := range ps.DemoUsersItems() {
		typed, ok := du.(*genSec.DemoUser)
		if !ok {
			continue
		}
		if typed.UserName() != s.UserName {
			continue
		}
		if !s.CreateOrModify {
			return mdlerrors.NewAlreadyExists("demo user", s.UserName)
		}
		mergedRoles := typed.UserRolesQualifiedNames()
		existingSet := make(map[string]bool)
		for _, r := range mergedRoles {
			existingSet[r] = true
		}
		for _, r := range s.UserRoles {
			if !existingSet[r] {
				mergedRoles = append(mergedRoles, r)
			}
		}
		entity := typed.EntityQualifiedName()
		if s.Entity != "" {
			entity = s.Entity
		}
		if err := d.SecurityProjectManager.RemoveDemoUser(model.ID(ps.ID()), s.UserName); err != nil {
			return mdlerrors.NewBackend("update demo user", err)
		}
		if err := d.SecurityProjectManager.AddDemoUser(model.ID(ps.ID()), s.UserName, s.Password, entity, mergedRoles); err != nil {
			return mdlerrors.NewBackend("update demo user", err)
		}
		d.InvalidateProjectSecurityCache()
		fmt.Fprintf(d.Output, "Modified demo user: %s\n", s.UserName)
		return nil
	}
	entity := s.Entity
	if entity == "" {
		detected, err := d.DetectUserEntity()
		if err != nil {
			return err
		}
		entity = detected
	}
	if err := d.SecurityProjectManager.AddDemoUser(model.ID(ps.ID()), s.UserName, s.Password, entity, s.UserRoles); err != nil {
		return mdlerrors.NewBackend("create demo user", err)
	}
	d.InvalidateProjectSecurityCache()
	fmt.Fprintf(d.Output, "Created demo user: %s (entity: %s)\n", s.UserName, entity)
	return nil
}

func ExecDropDemoUserFn(ctx context.Context, s *ast.DropDemoUserStmt, d SecurityDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	ps, err := d.GetProjectSecurityGen()
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", fmt.Errorf("ProjectSecurity not found"))
	}
	found := false
	for _, du := range ps.DemoUsersItems() {
		typed, ok := du.(*genSec.DemoUser)
		if !ok {
			continue
		}
		if typed.UserName() == s.UserName {
			found = true
			break
		}
	}
	if !found {
		return mdlerrors.NewNotFound("demo user", s.UserName)
	}
	if err := d.SecurityProjectManager.RemoveDemoUser(model.ID(ps.ID()), s.UserName); err != nil {
		return mdlerrors.NewBackend("drop demo user", err)
	}
	d.InvalidateProjectSecurityCache()
	fmt.Fprintf(d.Output, "Dropped demo user: %s\n", s.UserName)
	return nil
}

// ─────────────────────────────────────────────────────────────
// Stateless Helpers
// ─────────────────────────────────────────────────────────────

func genFlowContainerModuleFn(repo interface{ GetContainerUUID(id model.ID) (model.ID, error) }, findModuleID func(model.ID) model.ID, findModuleName func(model.ID) string, flowID model.ID) string {
	if repo == nil {
		return ""
	}
	containerID, err := repo.GetContainerUUID(flowID)
	if err != nil || containerID == "" {
		return ""
	}
	return findModuleName(findModuleID(containerID))
}

func expandRoleModule(role ast.QualifiedName, defaultModule string) ast.QualifiedName {
	if role.Module == "" {
		return ast.QualifiedName{Module: defaultModule, Name: role.Name}
	}
	return role
}

func validatePasswordPolicy(userName, password string, pp *genSec.PasswordPolicySettings) error {
	if pp == nil {
		return nil
	}
	if min := int(pp.MinimumLength()); min > 0 && len(password) < min {
		return mdlerrors.NewValidationf(
			"demo user '%s': password is %d characters, policy requires at least %d\n"+
				"hint: either use a longer password, or relax the policy first:\n"+
				"  alter project security password policy (min_length: %d, require_digit: %v, require_mixed_case: %v, require_symbol: %v);\n"+
				"  create or modify demo user '%s' password '<new-password>' ...;",
			userName, len(password), min,
			len(password), pp.RequireDigit(), pp.RequireMixedCase(), pp.RequireSymbol(),
			userName,
		)
	}
	if pp.RequireDigit() && !passwordContainsDigit(password) {
		return mdlerrors.NewValidationf(
			"demo user '%s': password must contain at least one digit (policy: require_digit: true)\n"+
				"hint: add a digit to the password, or disable the requirement:\n"+
				"  alter project security password policy (require_digit: false);",
			userName,
		)
	}
	if pp.RequireMixedCase() && (!passwordContainsUpper(password) || !passwordContainsLower(password)) {
		return mdlerrors.NewValidationf(
			"demo user '%s': password must contain both uppercase and lowercase letters (policy: require_mixed_case: true)\n"+
				"hint: add mixed-case letters, or disable the requirement:\n"+
				"  alter project security password policy (require_mixed_case: false);",
			userName,
		)
	}
	if pp.RequireSymbol() && !passwordContainsSymbol(password) {
		return mdlerrors.NewValidationf(
			"demo user '%s': password must contain at least one symbol (policy: require_symbol: true)\n"+
				"hint: add a symbol such as '!' or '@', or disable the requirement:\n"+
				"  alter project security password policy (require_symbol: false);",
			userName,
		)
	}
	return nil
}

func passwordContainsDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func passwordContainsUpper(s string) bool {
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

func passwordContainsLower(s string) bool {
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			return true
		}
	}
	return false
}

func passwordContainsSymbol(s string) bool {
	for _, r := range s {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			return true
		}
	}
	return false
}

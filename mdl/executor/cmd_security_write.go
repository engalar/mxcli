// SPDX-License-Identifier: Apache-2.0

// Package executor - Security write commands (CREATE/DROP/ALTER/GRANT/REVOKE)
package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/mpr"
	"github.com/mendixlabs/mxcli/sdk/security"
)

// execCreateModuleRole handles CREATE MODULE ROLE Module.RoleName [DESCRIPTION '...'].
func (e *Executor) execCreateModuleRole(s *ast.CreateModuleRoleStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	module, err := e.findModule(s.Name.Module)
	if err != nil {
		return err
	}

	ms, err := e.reader.GetModuleSecurity(module.ID)
	if err != nil {
		return fmt.Errorf("failed to read module security for %s: %w", s.Name.Module, err)
	}

	// Check if role already exists
	for _, mr := range ms.ModuleRoles {
		if mr.Name == s.Name.Name {
			return fmt.Errorf("module role already exists: %s.%s", s.Name.Module, s.Name.Name)
		}
	}

	if err := e.writer.AddModuleRole(ms.ID, s.Name.Name, s.Description); err != nil {
		return fmt.Errorf("failed to create module role: %w", err)
	}

	fmt.Fprintf(e.output, "Created module role: %s.%s\n", s.Name.Module, s.Name.Name)
	return nil
}

// execDropModuleRole handles DROP MODULE ROLE Module.RoleName.
// Cascade-removes the role from all entity access rules, microflow/nanoflow/page
// allowed roles, and OData service allowed roles before deleting the role itself.
func (e *Executor) execDropModuleRole(s *ast.DropModuleRoleStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	module, err := e.findModule(s.Name.Module)
	if err != nil {
		return err
	}

	ms, err := e.reader.GetModuleSecurity(module.ID)
	if err != nil {
		return fmt.Errorf("failed to read module security for %s: %w", s.Name.Module, err)
	}

	// Check role exists
	found := false
	for _, mr := range ms.ModuleRoles {
		if mr.Name == s.Name.Name {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("module role not found: %s.%s", s.Name.Module, s.Name.Name)
	}

	qualifiedRole := s.Name.Module + "." + s.Name.Name

	// Cascade: remove role from entity access rules
	dm, err := e.reader.GetDomainModel(module.ID)
	if err == nil {
		if n, err := e.writer.RemoveRoleFromAllEntities(dm.ID, qualifiedRole); err != nil {
			return fmt.Errorf("failed to cascade-remove entity access rules: %w", err)
		} else if n > 0 {
			fmt.Fprintf(e.output, "Removed %s from %d entity access rule(s)\n", qualifiedRole, n)
		}
	}

	// Cascade: remove role from microflow/nanoflow/page allowed roles
	h, err := e.getHierarchy()
	if err == nil {
		// Microflows
		if mfs, err := e.reader.ListMicroflows(); err == nil {
			for _, mf := range mfs {
				modID := h.FindModuleID(mf.ContainerID)
				if modID != module.ID {
					continue
				}
				if removed, err := e.writer.RemoveFromAllowedRoles(mf.ID, qualifiedRole); err == nil && removed {
					fmt.Fprintf(e.output, "Removed %s from microflow %s allowed roles\n", qualifiedRole, mf.Name)
				}
			}
		}

		// Nanoflows
		if nfs, err := e.reader.ListNanoflows(); err == nil {
			for _, nf := range nfs {
				modID := h.FindModuleID(nf.ContainerID)
				if modID != module.ID {
					continue
				}
				if removed, err := e.writer.RemoveFromAllowedRoles(nf.ID, qualifiedRole); err == nil && removed {
					fmt.Fprintf(e.output, "Removed %s from nanoflow %s allowed roles\n", qualifiedRole, nf.Name)
				}
			}
		}

		// Pages
		if pgs, err := e.reader.ListPages(); err == nil {
			for _, pg := range pgs {
				modID := h.FindModuleID(pg.ContainerID)
				if modID != module.ID {
					continue
				}
				if removed, err := e.writer.RemoveFromAllowedRoles(pg.ID, qualifiedRole); err == nil && removed {
					fmt.Fprintf(e.output, "Removed %s from page %s allowed roles\n", qualifiedRole, pg.Name)
				}
			}
		}

		// OData services
		if svcs, err := e.reader.ListPublishedODataServices(); err == nil {
			for _, svc := range svcs {
				modID := h.FindModuleID(svc.ContainerID)
				if modID != module.ID {
					continue
				}
				if removed, err := e.writer.RemoveFromAllowedRoles(svc.ID, qualifiedRole); err == nil && removed {
					fmt.Fprintf(e.output, "Removed %s from OData service %s allowed roles\n", qualifiedRole, svc.Name)
				}
			}
		}
	}

	// Finally, remove the role itself
	if err := e.writer.RemoveModuleRole(ms.ID, s.Name.Name); err != nil {
		return fmt.Errorf("failed to drop module role: %w", err)
	}

	fmt.Fprintf(e.output, "Dropped module role: %s.%s\n", s.Name.Module, s.Name.Name)
	return nil
}

// execCreateUserRole handles CREATE USER ROLE Name (ModuleRoles) [MANAGE ALL ROLES].
func (e *Executor) execCreateUserRole(s *ast.CreateUserRoleStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	ps, err := e.reader.GetProjectSecurity()
	if err != nil {
		return fmt.Errorf("failed to read project security: %w", err)
	}

	// Check if role already exists
	for _, ur := range ps.UserRoles {
		if ur.Name == s.Name {
			return fmt.Errorf("user role already exists: %s", s.Name)
		}
	}

	// Build qualified module role names
	var moduleRoleNames []string
	for _, mr := range s.ModuleRoles {
		qn := mr.Module + "." + mr.Name
		moduleRoleNames = append(moduleRoleNames, qn)
	}

	if err := e.writer.AddUserRole(ps.ID, s.Name, moduleRoleNames, s.ManageAllRoles); err != nil {
		return fmt.Errorf("failed to create user role: %w", err)
	}

	fmt.Fprintf(e.output, "Created user role: %s\n", s.Name)
	return nil
}

// execAlterUserRole handles ALTER USER ROLE Name ADD/REMOVE MODULE ROLES (...).
func (e *Executor) execAlterUserRole(s *ast.AlterUserRoleStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	ps, err := e.reader.GetProjectSecurity()
	if err != nil {
		return fmt.Errorf("failed to read project security: %w", err)
	}

	// Check user role exists
	found := false
	for _, ur := range ps.UserRoles {
		if ur.Name == s.Name {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("user role not found: %s", s.Name)
	}

	// Build qualified module role names
	var moduleRoleNames []string
	for _, mr := range s.ModuleRoles {
		moduleRoleNames = append(moduleRoleNames, mr.Module+"."+mr.Name)
	}

	if err := e.writer.AlterUserRoleModuleRoles(ps.ID, s.Name, s.Add, moduleRoleNames); err != nil {
		return fmt.Errorf("failed to alter user role: %w", err)
	}

	action := "Added"
	prep := "to"
	if !s.Add {
		action = "Removed"
		prep = "from"
	}
	fmt.Fprintf(e.output, "%s module roles %s %s user role %s\n", action, strings.Join(moduleRoleNames, ", "), prep, s.Name)
	return nil
}

// execDropUserRole handles DROP USER ROLE Name.
func (e *Executor) execDropUserRole(s *ast.DropUserRoleStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	ps, err := e.reader.GetProjectSecurity()
	if err != nil {
		return fmt.Errorf("failed to read project security: %w", err)
	}

	// Check user role exists
	found := false
	for _, ur := range ps.UserRoles {
		if ur.Name == s.Name {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("user role not found: %s", s.Name)
	}

	if err := e.writer.RemoveUserRole(ps.ID, s.Name); err != nil {
		return fmt.Errorf("failed to drop user role: %w", err)
	}

	fmt.Fprintf(e.output, "Dropped user role: %s\n", s.Name)
	return nil
}

// execGrantEntityAccess handles GRANT roles ON Module.Entity (rights) [WHERE '...'].
func (e *Executor) execGrantEntityAccess(s *ast.GrantEntityAccessStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	module, err := e.findModule(s.Entity.Module)
	if err != nil {
		return err
	}

	dm, err := e.reader.GetDomainModel(module.ID)
	if err != nil {
		return fmt.Errorf("failed to get domain model: %w", err)
	}

	// Verify entity exists
	entity := dm.FindEntityByName(s.Entity.Name)
	if entity == nil {
		return fmt.Errorf("entity not found: %s.%s", s.Entity.Module, s.Entity.Name)
	}

	// Build role name list
	var roleNames []string
	for _, role := range s.Roles {
		roleNames = append(roleNames, role.Module+"."+role.Name)
	}

	// Parse access rights from the statement.
	// Note: Mendix has no AllowRead/AllowWrite properties on AccessRule.
	// Read/write access is determined by DefaultMemberAccessRights and MemberAccesses.
	allowCreate, allowDelete := false, false
	defaultMemberAccess := "None"
	var readMembers, writeMembers []string // nil = all (wildcard)
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

	// Build MemberAccess entries for all entity attributes and associations.
	// Mendix requires explicit MemberAccess entries for every member — an empty
	// MemberAccesses array triggers CE0066 "Entity access is out of date".
	var memberAccesses []mpr.EntityMemberAccess

	// Build sets for specific member overrides (when READ (Name, Email) syntax is used)
	writeMemberSet := make(map[string]bool)
	for _, m := range writeMembers {
		writeMemberSet[m] = true
	}
	readMemberSet := make(map[string]bool)
	for _, m := range readMembers {
		readMemberSet[m] = true
	}

	// Create entries for all entity attributes
	for _, attr := range entity.Attributes {
		rights := defaultMemberAccess
		if writeMemberSet[attr.Name] {
			rights = "ReadWrite"
		} else if readMemberSet[attr.Name] {
			rights = "ReadOnly"
		}
		// Calculated attributes cannot have write rights (CE6592)
		isCalculated := attr.Value != nil && attr.Value.Type == "CalculatedValue"
		if isCalculated && (rights == "ReadWrite" || rights == "WriteOnly") {
			rights = "ReadOnly"
		}
		memberAccesses = append(memberAccesses, mpr.EntityMemberAccess{
			AttributeRef: module.Name + "." + s.Entity.Name + "." + attr.Name,
			AccessRights: rights,
		})
	}

	// Create entries for associations owned by this entity
	for _, assoc := range dm.Associations {
		if assoc.ParentID == entity.ID {
			rights := defaultMemberAccess
			if writeMemberSet[assoc.Name] {
				rights = "ReadWrite"
			} else if readMemberSet[assoc.Name] {
				rights = "ReadOnly"
			}
			memberAccesses = append(memberAccesses, mpr.EntityMemberAccess{
				AssociationRef: module.Name + "." + assoc.Name,
				AccessRights:   rights,
			})
		}
	}
	for _, ca := range dm.CrossAssociations {
		if ca.ParentID == entity.ID {
			rights := defaultMemberAccess
			if writeMemberSet[ca.Name] {
				rights = "ReadWrite"
			} else if readMemberSet[ca.Name] {
				rights = "ReadOnly"
			}
			memberAccesses = append(memberAccesses, mpr.EntityMemberAccess{
				AssociationRef: module.Name + "." + ca.Name,
				AccessRights:   rights,
			})
		}
	}

	if err := e.writer.AddEntityAccessRule(dm.ID, s.Entity.Name, roleNames,
		allowCreate, allowDelete,
		defaultMemberAccess, s.XPathConstraint, memberAccesses); err != nil {
		return fmt.Errorf("failed to grant entity access: %w", err)
	}

	// Reconcile MemberAccesses on pre-existing rules for this entity's domain model
	if count, err := e.writer.ReconcileMemberAccesses(dm.ID, module.Name); err != nil {
		return fmt.Errorf("failed to reconcile member accesses: %w", err)
	} else if count > 0 && !e.quiet {
		fmt.Fprintf(e.output, "Reconciled %d access rule(s) in module %s\n", count, module.Name)
	}

	e.trackModifiedDomainModel(module.ID, module.Name)
	fmt.Fprintf(e.output, "Granted access on %s.%s to %s\n", s.Entity.Module, s.Entity.Name, strings.Join(roleNames, ", "))
	return nil
}

// execRevokeEntityAccess handles REVOKE roles ON Module.Entity.
func (e *Executor) execRevokeEntityAccess(s *ast.RevokeEntityAccessStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	module, err := e.findModule(s.Entity.Module)
	if err != nil {
		return err
	}

	dm, err := e.reader.GetDomainModel(module.ID)
	if err != nil {
		return fmt.Errorf("failed to get domain model: %w", err)
	}

	// Verify entity exists
	entity := dm.FindEntityByName(s.Entity.Name)
	if entity == nil {
		return fmt.Errorf("entity not found: %s.%s", s.Entity.Module, s.Entity.Name)
	}

	// Build role name list
	var roleNames []string
	for _, role := range s.Roles {
		roleNames = append(roleNames, role.Module+"."+role.Name)
	}

	modified, err := e.writer.RemoveEntityAccessRule(dm.ID, s.Entity.Name, roleNames)
	if err != nil {
		return fmt.Errorf("failed to revoke entity access: %w", err)
	}

	if modified == 0 {
		fmt.Fprintf(e.output, "No access rules found matching %s on %s.%s\n", strings.Join(roleNames, ", "), s.Entity.Module, s.Entity.Name)
	} else {
		fmt.Fprintf(e.output, "Revoked access on %s.%s from %s\n", s.Entity.Module, s.Entity.Name, strings.Join(roleNames, ", "))
	}
	e.trackModifiedDomainModel(module.ID, module.Name)
	return nil
}

// execGrantMicroflowAccess handles GRANT EXECUTE ON MICROFLOW Module.MF TO roles.
func (e *Executor) execGrantMicroflowAccess(s *ast.GrantMicroflowAccessStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	// Find the microflow
	mfs, err := e.reader.ListMicroflows()
	if err != nil {
		return fmt.Errorf("failed to list microflows: %w", err)
	}

	for _, mf := range mfs {
		modID := h.FindModuleID(mf.ContainerID)
		modName := h.GetModuleName(modID)
		if modName != s.Microflow.Module || mf.Name != s.Microflow.Name {
			continue
		}

		// Validate all roles exist
		for _, role := range s.Roles {
			if err := e.validateModuleRole(role); err != nil {
				return err
			}
		}

		// Merge new roles with existing (skip duplicates)
		existing := make(map[string]bool)
		var merged []string
		for _, r := range mf.AllowedModuleRoles {
			existing[string(r)] = true
			merged = append(merged, string(r))
		}
		var added []string
		for _, role := range s.Roles {
			qn := role.Module + "." + role.Name
			if !existing[qn] {
				merged = append(merged, qn)
				added = append(added, qn)
			}
		}

		if err := e.writer.UpdateAllowedRoles(mf.ID, merged); err != nil {
			return fmt.Errorf("failed to update microflow access: %w", err)
		}

		if len(added) == 0 {
			fmt.Fprintf(e.output, "All specified roles already have execute access on %s.%s\n", modName, mf.Name)
		} else {
			fmt.Fprintf(e.output, "Granted execute access on %s.%s to %s\n", modName, mf.Name, strings.Join(added, ", "))
		}
		return nil
	}

	return fmt.Errorf("microflow not found: %s.%s", s.Microflow.Module, s.Microflow.Name)
}

// execRevokeMicroflowAccess handles REVOKE EXECUTE ON MICROFLOW Module.MF FROM roles.
func (e *Executor) execRevokeMicroflowAccess(s *ast.RevokeMicroflowAccessStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	// Find the microflow
	mfs, err := e.reader.ListMicroflows()
	if err != nil {
		return fmt.Errorf("failed to list microflows: %w", err)
	}

	for _, mf := range mfs {
		modID := h.FindModuleID(mf.ContainerID)
		modName := h.GetModuleName(modID)
		if modName != s.Microflow.Module || mf.Name != s.Microflow.Name {
			continue
		}

		// Build set of roles to remove
		toRemove := make(map[string]bool)
		for _, role := range s.Roles {
			toRemove[role.Module+"."+role.Name] = true
		}

		// Filter out removed roles
		var remaining []string
		var removed []string
		for _, r := range mf.AllowedModuleRoles {
			if toRemove[string(r)] {
				removed = append(removed, string(r))
			} else {
				remaining = append(remaining, string(r))
			}
		}

		if err := e.writer.UpdateAllowedRoles(mf.ID, remaining); err != nil {
			return fmt.Errorf("failed to update microflow access: %w", err)
		}

		if len(removed) == 0 {
			fmt.Fprintf(e.output, "None of the specified roles had execute access on %s.%s\n", modName, mf.Name)
		} else {
			fmt.Fprintf(e.output, "Revoked execute access on %s.%s from %s\n", modName, mf.Name, strings.Join(removed, ", "))
		}
		return nil
	}

	return fmt.Errorf("microflow not found: %s.%s", s.Microflow.Module, s.Microflow.Name)
}

// execGrantPageAccess handles GRANT VIEW ON PAGE Module.Page TO roles.
func (e *Executor) execGrantPageAccess(s *ast.GrantPageAccessStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	// Find the page
	pages, err := e.reader.ListPages()
	if err != nil {
		return fmt.Errorf("failed to list pages: %w", err)
	}

	for _, pg := range pages {
		modID := h.FindModuleID(pg.ContainerID)
		modName := h.GetModuleName(modID)
		if modName != s.Page.Module || pg.Name != s.Page.Name {
			continue
		}

		// Validate all roles exist
		for _, role := range s.Roles {
			if err := e.validateModuleRole(role); err != nil {
				return err
			}
		}

		// Merge new roles with existing (skip duplicates)
		existing := make(map[string]bool)
		var merged []string
		for _, r := range pg.AllowedRoles {
			existing[string(r)] = true
			merged = append(merged, string(r))
		}
		var added []string
		for _, role := range s.Roles {
			qn := role.Module + "." + role.Name
			if !existing[qn] {
				merged = append(merged, qn)
				added = append(added, qn)
			}
		}

		if err := e.writer.UpdateAllowedRoles(pg.ID, merged); err != nil {
			return fmt.Errorf("failed to update page access: %w", err)
		}

		if len(added) == 0 {
			fmt.Fprintf(e.output, "All specified roles already have view access on %s.%s\n", modName, pg.Name)
		} else {
			fmt.Fprintf(e.output, "Granted view access on %s.%s to %s\n", modName, pg.Name, strings.Join(added, ", "))
		}
		return nil
	}

	return fmt.Errorf("page not found: %s.%s", s.Page.Module, s.Page.Name)
}

// execRevokePageAccess handles REVOKE VIEW ON PAGE Module.Page FROM roles.
func (e *Executor) execRevokePageAccess(s *ast.RevokePageAccessStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	// Find the page
	pages, err := e.reader.ListPages()
	if err != nil {
		return fmt.Errorf("failed to list pages: %w", err)
	}

	for _, pg := range pages {
		modID := h.FindModuleID(pg.ContainerID)
		modName := h.GetModuleName(modID)
		if modName != s.Page.Module || pg.Name != s.Page.Name {
			continue
		}

		// Build set of roles to remove
		toRemove := make(map[string]bool)
		for _, role := range s.Roles {
			toRemove[role.Module+"."+role.Name] = true
		}

		// Filter out removed roles
		var remaining []string
		var removed []string
		for _, r := range pg.AllowedRoles {
			if toRemove[string(r)] {
				removed = append(removed, string(r))
			} else {
				remaining = append(remaining, string(r))
			}
		}

		if err := e.writer.UpdateAllowedRoles(pg.ID, remaining); err != nil {
			return fmt.Errorf("failed to update page access: %w", err)
		}

		if len(removed) == 0 {
			fmt.Fprintf(e.output, "None of the specified roles had view access on %s.%s\n", modName, pg.Name)
		} else {
			fmt.Fprintf(e.output, "Revoked view access on %s.%s from %s\n", modName, pg.Name, strings.Join(removed, ", "))
		}
		return nil
	}

	return fmt.Errorf("page not found: %s.%s", s.Page.Module, s.Page.Name)
}

// execGrantWorkflowAccess handles GRANT EXECUTE ON WORKFLOW Module.WF TO roles.
func (e *Executor) execGrantWorkflowAccess(s *ast.GrantWorkflowAccessStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	// Find the workflow
	wfs, err := e.reader.ListWorkflows()
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	for _, wf := range wfs {
		modID := h.FindModuleID(wf.ContainerID)
		modName := h.GetModuleName(modID)
		if modName != s.Workflow.Module || wf.Name != s.Workflow.Name {
			continue
		}

		// Validate all roles exist
		for _, role := range s.Roles {
			if err := e.validateModuleRole(role); err != nil {
				return err
			}
		}

		// Merge new roles with existing (skip duplicates)
		existing := make(map[string]bool)
		var merged []string
		for _, r := range wf.AllowedModuleRoles {
			existing[string(r)] = true
			merged = append(merged, string(r))
		}
		var added []string
		for _, role := range s.Roles {
			qn := role.Module + "." + role.Name
			if !existing[qn] {
				merged = append(merged, qn)
				added = append(added, qn)
			}
		}

		if err := e.writer.UpdateAllowedRoles(wf.ID, merged); err != nil {
			return fmt.Errorf("failed to update workflow access: %w", err)
		}

		if len(added) == 0 {
			fmt.Fprintf(e.output, "All specified roles already have execute access on %s.%s\n", modName, wf.Name)
		} else {
			fmt.Fprintf(e.output, "Granted execute access on %s.%s to %s\n", modName, wf.Name, strings.Join(added, ", "))
		}
		return nil
	}

	return fmt.Errorf("workflow not found: %s.%s", s.Workflow.Module, s.Workflow.Name)
}

// execRevokeWorkflowAccess handles REVOKE EXECUTE ON WORKFLOW Module.WF FROM roles.
func (e *Executor) execRevokeWorkflowAccess(s *ast.RevokeWorkflowAccessStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	// Find the workflow
	wfs, err := e.reader.ListWorkflows()
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	for _, wf := range wfs {
		modID := h.FindModuleID(wf.ContainerID)
		modName := h.GetModuleName(modID)
		if modName != s.Workflow.Module || wf.Name != s.Workflow.Name {
			continue
		}

		// Build set of roles to remove
		toRemove := make(map[string]bool)
		for _, role := range s.Roles {
			toRemove[role.Module+"."+role.Name] = true
		}

		// Filter out removed roles
		var remaining []string
		var removed []string
		for _, r := range wf.AllowedModuleRoles {
			if toRemove[string(r)] {
				removed = append(removed, string(r))
			} else {
				remaining = append(remaining, string(r))
			}
		}

		if err := e.writer.UpdateAllowedRoles(wf.ID, remaining); err != nil {
			return fmt.Errorf("failed to update workflow access: %w", err)
		}

		if len(removed) == 0 {
			fmt.Fprintf(e.output, "None of the specified roles had execute access on %s.%s\n", modName, wf.Name)
		} else {
			fmt.Fprintf(e.output, "Revoked execute access on %s.%s from %s\n", modName, wf.Name, strings.Join(removed, ", "))
		}
		return nil
	}

	return fmt.Errorf("workflow not found: %s.%s", s.Workflow.Module, s.Workflow.Name)
}

// validateModuleRole checks that a module role exists in the project.
func (e *Executor) validateModuleRole(role ast.QualifiedName) error {
	module, err := e.findModule(role.Module)
	if err != nil {
		return fmt.Errorf("module not found for role %s.%s: %w", role.Module, role.Name, err)
	}

	ms, err := e.reader.GetModuleSecurity(module.ID)
	if err != nil {
		return fmt.Errorf("failed to read module security for %s: %w", role.Module, err)
	}

	for _, mr := range ms.ModuleRoles {
		if mr.Name == role.Name {
			return nil
		}
	}

	return fmt.Errorf("module role not found: %s.%s", role.Module, role.Name)
}

// execAlterProjectSecurity handles ALTER PROJECT SECURITY LEVEL/DEMO USERS.
func (e *Executor) execAlterProjectSecurity(s *ast.AlterProjectSecurityStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	ps, err := e.reader.GetProjectSecurity()
	if err != nil {
		return fmt.Errorf("failed to read project security: %w", err)
	}

	if s.SecurityLevel != "" {
		// Map from display name to BSON value
		var bsonLevel string
		switch s.SecurityLevel {
		case "Production":
			bsonLevel = security.SecurityLevelProduction
		case "Prototype":
			bsonLevel = security.SecurityLevelPrototype
		case "Off":
			bsonLevel = security.SecurityLevelOff
		default:
			return fmt.Errorf("unknown security level: %s", s.SecurityLevel)
		}

		if err := e.writer.SetProjectSecurityLevel(ps.ID, bsonLevel); err != nil {
			return fmt.Errorf("failed to set security level: %w", err)
		}
		fmt.Fprintf(e.output, "Set project security level to %s\n", s.SecurityLevel)
	}

	if s.DemoUsersEnabled != nil {
		if err := e.writer.SetProjectDemoUsersEnabled(ps.ID, *s.DemoUsersEnabled); err != nil {
			return fmt.Errorf("failed to set demo users: %w", err)
		}
		state := "disabled"
		if *s.DemoUsersEnabled {
			state = "enabled"
		}
		fmt.Fprintf(e.output, "Demo users %s\n", state)
	}

	return nil
}

// execCreateDemoUser handles CREATE DEMO USER 'name' PASSWORD 'pw' [ENTITY Module.Entity] (Roles).
func (e *Executor) execCreateDemoUser(s *ast.CreateDemoUserStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	ps, err := e.reader.GetProjectSecurity()
	if err != nil {
		return fmt.Errorf("failed to read project security: %w", err)
	}

	// Check if user already exists
	for _, du := range ps.DemoUsers {
		if du.UserName == s.UserName {
			return fmt.Errorf("demo user already exists: %s", s.UserName)
		}
	}

	// Resolve entity: use explicit value or auto-detect from domain models
	entity := s.Entity
	if entity == "" {
		detected, err := e.detectUserEntity()
		if err != nil {
			return err
		}
		entity = detected
	}

	if err := e.writer.AddDemoUser(ps.ID, s.UserName, s.Password, entity, s.UserRoles); err != nil {
		return fmt.Errorf("failed to create demo user: %w", err)
	}

	fmt.Fprintf(e.output, "Created demo user: %s (entity: %s)\n", s.UserName, entity)
	return nil
}

// detectUserEntity finds the entity that generalizes System.User.
func (e *Executor) detectUserEntity() (string, error) {
	modules, err := e.reader.ListModules()
	if err != nil {
		return "", fmt.Errorf("failed to list modules: %w", err)
	}
	moduleNameByID := make(map[model.ID]string, len(modules))
	for _, m := range modules {
		moduleNameByID[m.ID] = m.Name
	}

	dms, err := e.reader.ListDomainModels()
	if err != nil {
		return "", fmt.Errorf("failed to list domain models: %w", err)
	}

	var candidates []string
	for _, dm := range dms {
		moduleName := moduleNameByID[dm.ContainerID]
		for _, ent := range dm.Entities {
			if ent.GeneralizationRef == "System.User" {
				candidates = append(candidates, moduleName+"."+ent.Name)
			}
		}
	}

	switch len(candidates) {
	case 0:
		return "", fmt.Errorf("no entity found that generalizes System.User; use ENTITY clause to specify one")
	case 1:
		return candidates[0], nil
	default:
		return "", fmt.Errorf("multiple entities generalize System.User: %s; use ENTITY clause to specify one", joinCandidates(candidates))
	}
}

func joinCandidates(candidates []string) string {
	result := candidates[0]
	for i := 1; i < len(candidates); i++ {
		result += ", " + candidates[i]
	}
	return result
}

// execDropDemoUser handles DROP DEMO USER 'name'.
func (e *Executor) execDropDemoUser(s *ast.DropDemoUserStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	ps, err := e.reader.GetProjectSecurity()
	if err != nil {
		return fmt.Errorf("failed to read project security: %w", err)
	}

	// Check if user exists
	found := false
	for _, du := range ps.DemoUsers {
		if du.UserName == s.UserName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("demo user not found: %s", s.UserName)
	}

	if err := e.writer.RemoveDemoUser(ps.ID, s.UserName); err != nil {
		return fmt.Errorf("failed to drop demo user: %w", err)
	}

	fmt.Fprintf(e.output, "Dropped demo user: %s\n", s.UserName)
	return nil
}

// ============================================================================
// GRANT/REVOKE ACCESS ON ODATA SERVICE
// ============================================================================

// execGrantODataServiceAccess handles GRANT ACCESS ON ODATA SERVICE Module.Svc TO roles.
func (e *Executor) execGrantODataServiceAccess(s *ast.GrantODataServiceAccessStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	// Find the published OData service
	services, err := e.reader.ListPublishedODataServices()
	if err != nil {
		return fmt.Errorf("failed to list published OData services: %w", err)
	}

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if modName != s.Service.Module || svc.Name != s.Service.Name {
			continue
		}

		// Validate all roles exist
		for _, role := range s.Roles {
			if err := e.validateModuleRole(role); err != nil {
				return err
			}
		}

		// Merge new roles with existing (skip duplicates)
		existing := make(map[string]bool)
		var merged []string
		for _, r := range svc.AllowedModuleRoles {
			existing[r] = true
			merged = append(merged, r)
		}
		var added []string
		for _, role := range s.Roles {
			qn := role.Module + "." + role.Name
			if !existing[qn] {
				merged = append(merged, qn)
				added = append(added, qn)
			}
		}

		if err := e.writer.UpdateAllowedRoles(svc.ID, merged); err != nil {
			return fmt.Errorf("failed to update OData service access: %w", err)
		}

		if len(added) == 0 {
			fmt.Fprintf(e.output, "All specified roles already have access on OData service %s.%s\n", modName, svc.Name)
		} else {
			fmt.Fprintf(e.output, "Granted access on OData service %s.%s to %s\n", modName, svc.Name, strings.Join(added, ", "))
		}
		return nil
	}

	return fmt.Errorf("published OData service not found: %s.%s", s.Service.Module, s.Service.Name)
}

// execRevokeODataServiceAccess handles REVOKE ACCESS ON ODATA SERVICE Module.Svc FROM roles.
func (e *Executor) execRevokeODataServiceAccess(s *ast.RevokeODataServiceAccessStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	// Find the published OData service
	services, err := e.reader.ListPublishedODataServices()
	if err != nil {
		return fmt.Errorf("failed to list published OData services: %w", err)
	}

	for _, svc := range services {
		modID := h.FindModuleID(svc.ContainerID)
		modName := h.GetModuleName(modID)
		if modName != s.Service.Module || svc.Name != s.Service.Name {
			continue
		}

		// Build set of roles to remove
		toRemove := make(map[string]bool)
		for _, role := range s.Roles {
			toRemove[role.Module+"."+role.Name] = true
		}

		// Filter out removed roles
		var remaining []string
		var removed []string
		for _, r := range svc.AllowedModuleRoles {
			if toRemove[r] {
				removed = append(removed, r)
			} else {
				remaining = append(remaining, r)
			}
		}

		if err := e.writer.UpdateAllowedRoles(svc.ID, remaining); err != nil {
			return fmt.Errorf("failed to update OData service access: %w", err)
		}

		if len(removed) == 0 {
			fmt.Fprintf(e.output, "None of the specified roles had access on OData service %s.%s\n", modName, svc.Name)
		} else {
			fmt.Fprintf(e.output, "Revoked access on OData service %s.%s from %s\n", modName, svc.Name, strings.Join(removed, ", "))
		}
		return nil
	}

	return fmt.Errorf("published OData service not found: %s.%s", s.Service.Module, s.Service.Name)
}

// execUpdateSecurity handles UPDATE SECURITY [IN Module].
func (e *Executor) execUpdateSecurity(s *ast.UpdateSecurityStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project in write mode")
	}

	modules, err := e.getModulesFromCache()
	if err != nil {
		return err
	}

	totalModified := 0
	for _, mod := range modules {
		if s.Module != "" && mod.Name != s.Module {
			continue
		}

		dm, err := e.reader.GetDomainModel(mod.ID)
		if err != nil {
			continue // module may not have a domain model
		}

		count, err := e.writer.ReconcileMemberAccesses(dm.ID, mod.Name)
		if err != nil {
			return fmt.Errorf("failed to reconcile security for module %s: %w", mod.Name, err)
		}
		if count > 0 {
			fmt.Fprintf(e.output, "Reconciled %d access rule(s) in module %s\n", count, mod.Name)
			totalModified += count
		}
	}

	if totalModified == 0 {
		fmt.Fprintf(e.output, "All entity access rules are up to date\n")
	}

	return nil
}

// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5a — gen-typed security read paths.
//
// SHOW ACCESS ON {MICROFLOW,NANOFLOW} and SHOW SECURITY MATRIX both
// pull microflow / nanoflow lists from Backend.ListMicroflows /
// ListNanoflows today. This file provides the modelsdk/gen-native
// equivalents that consume ctx.Microflows / ctx.Nanoflows.
//
// The entity-side reads (listAccessOnEntity, the entities section of
// the matrix, listProjectSecurity, listModuleRoles, listUserRoles,
// listDemoUsers, describeModuleRole, describeDemoUser, describeUserRole)
// do not touch sdk/microflows so they stay in cmd_security.go.
//
// The write paths (alter project security, GRANT/REVOKE, CREATE module
// role, etc.) live in cmd_security_write.go and belong to Stage 3.2.5b
// — not 3.2.5a.

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// listAccessOnMicroflowGen handles SHOW ACCESS ON MICROFLOW Module.MF
// using gen-typed microflows from ctx.Microflows. Returns a not-found
// error if the named microflow does not exist.
func listAccessOnMicroflowGen(ctx *ExecContext, name *ast.QualifiedName) error {
	if name == nil {
		return mdlerrors.NewValidation("microflow name required")
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	mfs, err := listMicroflowsGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list microflows", err)
	}

	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		modName := genFlowContainerModule(ctx, h, model.ID(mf.ID()))
		if modName != name.Module || mf.Name() != name.Name {
			continue
		}
		roles := mf.AllowedModuleRolesQualifiedNames()
		if ctx.Format == FormatJSON {
			result := &TableResult{Columns: []string{"Module", "Role"}}
			for _, role := range roles {
				mod, r := splitRoleQualifiedName(role)
				result.Rows = append(result.Rows, []any{mod, r})
			}
			return writeResult(ctx, result)
		}
		if len(roles) == 0 {
			fmt.Fprintf(ctx.Output, "No module roles granted execute access on %s.%s\n", modName, mf.Name())
			return nil
		}
		fmt.Fprintf(ctx.Output, "Allowed module roles for %s.%s:\n", modName, mf.Name())
		for _, role := range roles {
			fmt.Fprintf(ctx.Output, "  %s\n", role)
		}
		return nil
	}

	return mdlerrors.NewNotFound("microflow", name.String())
}

// listAccessOnNanoflowGen mirrors listAccessOnMicroflowGen for nanoflows.
func listAccessOnNanoflowGen(ctx *ExecContext, name *ast.QualifiedName) error {
	if name == nil {
		return mdlerrors.NewValidation("nanoflow name required")
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	nfs, err := listNanoflowsGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list nanoflows", err)
	}

	for _, nf := range nfs {
		if nf == nil {
			continue
		}
		modName := genFlowContainerModule(ctx, h, model.ID(nf.ID()))
		if modName != name.Module || nf.Name() != name.Name {
			continue
		}
		roles := nf.AllowedModuleRolesQualifiedNames()
		if ctx.Format == FormatJSON {
			result := &TableResult{Columns: []string{"Module", "Role"}}
			for _, role := range roles {
				mod, r := splitRoleQualifiedName(role)
				result.Rows = append(result.Rows, []any{mod, r})
			}
			return writeResult(ctx, result)
		}
		if len(roles) == 0 {
			fmt.Fprintf(ctx.Output, "No module roles granted execute access on %s.%s\n", modName, nf.Name())
			return nil
		}
		fmt.Fprintf(ctx.Output, "Allowed module roles for %s.%s:\n", modName, nf.Name())
		for _, role := range roles {
			fmt.Fprintf(ctx.Output, "  %s\n", role)
		}
		return nil
	}

	return mdlerrors.NewNotFound("nanoflow", name.String())
}

// listSecurityMatrixGen handles SHOW SECURITY MATRIX [IN module] using
// gen-typed microflows for the microflow access section. Entity and
// page sections still flow through Backend (entities in cmd_security.go,
// pages have their own backend list — pages migration is out of
// 3.2.5a's scope).
func listSecurityMatrixGen(ctx *ExecContext, moduleName string) error {
	if ctx.Format == FormatJSON {
		return listSecurityMatrixJSONGen(ctx, moduleName)
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	modules, err := ctx.Backend.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("list modules", err)
	}

	// Build role list for the target module(s).
	type moduleRoleInfo struct {
		moduleName string
		roleName   string
	}
	var roles []moduleRoleInfo
	for _, mod := range modules {
		if moduleName != "" && mod.Name != moduleName {
			continue
		}
		ms, err := ctx.Backend.GetModuleSecurityGen(mod.ID)
		if err != nil || ms == nil {
			continue
		}
		for _, mrItem := range ms.ModuleRolesItems() {
			mr, ok := mrItem.(*genSec.ModuleRole)
			if !ok {
				continue
			}
			roles = append(roles, moduleRoleInfo{mod.Name, mr.Name()})
		}
	}

	if len(roles) == 0 {
		if moduleName != "" {
			fmt.Fprintf(ctx.Output, "No module roles found in %s\n", moduleName)
		} else {
			fmt.Fprintln(ctx.Output, "No module roles found")
		}
		return nil
	}

	pairs, err := listDomainModelsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list domain models", err)
	}

	fmt.Fprintf(ctx.Output, "Security Matrix")
	if moduleName != "" {
		fmt.Fprintf(ctx.Output, " for %s", moduleName)
	}
	fmt.Fprintln(ctx.Output, ":")
	fmt.Fprintln(ctx.Output)

	// Entities section — gen-typed walk (Stage 3.3.4 C3).
	fmt.Fprintln(ctx.Output, "## Entity Access")
	fmt.Fprintln(ctx.Output)

	entityFound := false
	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		modID := h.FindModuleID(p.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleName != "" && modName != moduleName {
			continue
		}
		for _, e := range p.DM.EntitiesItems() {
			entity, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			rules := entity.AccessRulesItems()
			if len(rules) == 0 {
				continue
			}
			entityFound = true
			fmt.Fprintf(ctx.Output, "### %s.%s\n", modName, entity.Name())
			for _, r := range rules {
				rule, ok := r.(*genDm.AccessRule)
				if !ok {
					continue
				}
				roleStrs := entityRuleRoleStringsGen(rule)
				rights := entityRuleRightStringsGen(rule)
				fmt.Fprintf(ctx.Output, "  %s: %s\n", strings.Join(roleStrs, ", "), strings.Join(rights, ""))
			}
			fmt.Fprintln(ctx.Output)
		}
	}
	if !entityFound {
		fmt.Fprintln(ctx.Output, "(no entity access rules configured)")
		fmt.Fprintln(ctx.Output)
	}

	// Microflow section — gen-typed.
	fmt.Fprintln(ctx.Output, "## Microflow Access")
	fmt.Fprintln(ctx.Output)

	mfs, err := listMicroflowsGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list microflows", err)
	}

	mfFound := false
	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		roleStrs := mf.AllowedModuleRolesQualifiedNames()
		if len(roleStrs) == 0 {
			continue
		}
		modName := genFlowContainerModule(ctx, h, model.ID(mf.ID()))
		if moduleName != "" && modName != moduleName {
			continue
		}
		mfFound = true
		fmt.Fprintf(ctx.Output, "  %s.%s: %s\n", modName, mf.Name(), strings.Join(roleStrs, ", "))
	}
	if !mfFound {
		fmt.Fprintln(ctx.Output, "(no microflow access rules configured)")
	}
	fmt.Fprintln(ctx.Output)

	// Page section — gen-typed listing via listPagesWithContainerGen.
	fmt.Fprintln(ctx.Output, "## Page Access")
	fmt.Fprintln(ctx.Output)

	pgPairs, err := listPagesWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}

	pgFound := false
	for _, pair := range pgPairs {
		pg := pair.Elem
		roles := pg.AllowedRolesQualifiedNames()
		if len(roles) == 0 {
			continue
		}
		modID := h.FindModuleID(model.ID(pair.ContainerID))
		modName := h.GetModuleName(modID)
		if moduleName != "" && modName != moduleName {
			continue
		}
		pgFound = true
		fmt.Fprintf(ctx.Output, "  %s.%s: %s\n", modName, pg.Name(), strings.Join(roles, ", "))
	}
	if !pgFound {
		fmt.Fprintln(ctx.Output, "(no page access rules configured)")
	}
	fmt.Fprintln(ctx.Output)

	// Workflow section — workflows do not have document-level
	// AllowedModuleRoles; preserve the legacy explanatory note.
	fmt.Fprintln(ctx.Output, "## Workflow Access")
	fmt.Fprintln(ctx.Output)
	fmt.Fprintln(ctx.Output, "(workflow access is controlled through triggering microflows and UserTask targeting, not document-level roles)")
	fmt.Fprintln(ctx.Output)

	return nil
}

// listSecurityMatrixJSONGen emits the security matrix as a JSON table,
// using gen-typed microflows for the microflow rows.
func listSecurityMatrixJSONGen(ctx *ExecContext, moduleName string) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	tr := &TableResult{
		Columns: []string{"ObjectType", "QualifiedName", "Roles", "Rights"},
	}

	// Entities — gen-typed walk (Stage 3.3.4 C3).
	pairs, _ := listDomainModelsWithContainerGen(ctx)
	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		modID := h.FindModuleID(p.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleName != "" && modName != moduleName {
			continue
		}
		for _, e := range p.DM.EntitiesItems() {
			entity, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			for _, r := range entity.AccessRulesItems() {
				rule, ok := r.(*genDm.AccessRule)
				if !ok {
					continue
				}
				roleStrs := entityRuleRoleStringsGen(rule)
				rights := entityRuleRightStringsGen(rule)
				tr.Rows = append(tr.Rows, []any{
					"Entity",
					modName + "." + entity.Name(),
					strings.Join(roleStrs, ", "),
					strings.Join(rights, ""),
				})
			}
		}
	}

	// Microflows — gen-typed.
	mfs, _ := listMicroflowsGen(ctx)
	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		roleStrs := mf.AllowedModuleRolesQualifiedNames()
		if len(roleStrs) == 0 {
			continue
		}
		modName := genFlowContainerModule(ctx, h, model.ID(mf.ID()))
		if moduleName != "" && modName != moduleName {
			continue
		}
		tr.Rows = append(tr.Rows, []any{
			"Microflow",
			modName + "." + mf.Name(),
			strings.Join(roleStrs, ", "),
			"X",
		})
	}

	// Pages — gen-typed listing via listPagesWithContainerGen.
	pgPairs2, _ := listPagesWithContainerGen(ctx)
	for _, pair := range pgPairs2 {
		pg := pair.Elem
		roles := pg.AllowedRolesQualifiedNames()
		if len(roles) == 0 {
			continue
		}
		modID := h.FindModuleID(model.ID(pair.ContainerID))
		modName := h.GetModuleName(modID)
		if moduleName != "" && modName != moduleName {
			continue
		}
		tr.Rows = append(tr.Rows, []any{
			"Page",
			modName + "." + pg.Name(),
			strings.Join(roles, ", "),
			"X",
		})
	}

	return writeResult(ctx, tr)
}

// ────────────────────────────────────────────────────────
// Small helpers shared between the gen access readers.
// ────────────────────────────────────────────────────────

// splitRoleQualifiedName splits "Module.Role" into ("Module","Role").
// Mirrors the inline split in listAccessOnMicroflow / listAccessOnPage,
// returning ("", role) when no dot is present.
func splitRoleQualifiedName(qn string) (string, string) {
	parts := strings.SplitN(qn, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", qn
}

// entityRuleRoleStringsGen mirrors entityRuleRoleStrings for the
// gen-typed AccessRule. Gen exposes role qualified names via the
// combined accessor ModuleRolesQualifiedNames(); the historical
// ModuleRoleNames vs ModuleRoles split from sdk has been collapsed.
func entityRuleRoleStringsGen(rule *genDm.AccessRule) []string {
	if rule == nil {
		return nil
	}
	qns := rule.ModuleRolesQualifiedNames()
	out := make([]string, len(qns))
	copy(out, qns)
	return out
}

// listProjectSecurityGen handles SHOW PROJECT SECURITY using the gen-typed
// ProjectSecurity from ctx.Security. Mirrors listProjectSecurity exactly
// in output shape; only the type source changes.
func listProjectSecurityGen(ctx *ExecContext) error {
	ps, err := getProjectSecurityGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", fmt.Errorf("ProjectSecurity unit not found"))
	}

	// schema gap: PasswordPolicySettings() returns element.Element; type-assert
	// to *genSec.PasswordPolicySettings to access typed fields.
	var pp *genSec.PasswordPolicySettings
	if raw := ps.PasswordPolicySettings(); raw != nil {
		pp, _ = raw.(*genSec.PasswordPolicySettings)
	}

	if ctx.Format == FormatJSON {
		result := &TableResult{Columns: []string{"Property", "Value"}}
		result.Rows = append(result.Rows,
			[]any{"SecurityLevel", securityLevelDisplay(ps.SecurityLevel())},
			[]any{"CheckSecurity", fmt.Sprintf("%v", ps.CheckSecurity())},
			[]any{"StrictMode", fmt.Sprintf("%v", ps.StrictMode())},
			[]any{"DemoUsersEnabled", fmt.Sprintf("%v", ps.EnableDemoUsers())},
			[]any{"GuestAccess", fmt.Sprintf("%v", ps.EnableGuestAccess())},
			[]any{"UserRoles", fmt.Sprintf("%d", len(ps.UserRolesItems()))},
			[]any{"DemoUsers", fmt.Sprintf("%d", len(ps.DemoUsersItems()))},
		)
		if ps.AdminUserName() != "" {
			result.Rows = append(result.Rows, []any{"AdminUser", ps.AdminUserName()})
		}
		// schema gap: GuestUserRoleQualifiedName() does not exist; gen uses GuestUserRoleName()
		if ps.GuestUserRoleName() != "" {
			result.Rows = append(result.Rows, []any{"GuestUserRole", ps.GuestUserRoleName()})
		}
		if pp != nil {
			result.Rows = append(result.Rows,
				[]any{"PasswordPolicy.MinimumLength", fmt.Sprintf("%d", pp.MinimumLength())},
				[]any{"PasswordPolicy.RequireDigit", fmt.Sprintf("%v", pp.RequireDigit())},
				[]any{"PasswordPolicy.RequireMixedCase", fmt.Sprintf("%v", pp.RequireMixedCase())},
				[]any{"PasswordPolicy.RequireSymbol", fmt.Sprintf("%v", pp.RequireSymbol())},
			)
		}
		return writeResult(ctx, result)
	}

	fmt.Fprintf(ctx.Output, "Security Level: %s\n", securityLevelDisplay(ps.SecurityLevel()))
	fmt.Fprintf(ctx.Output, "Check Security: %v\n", ps.CheckSecurity())
	fmt.Fprintf(ctx.Output, "Strict Mode: %v\n", ps.StrictMode())
	fmt.Fprintf(ctx.Output, "Demo Users Enabled: %v\n", ps.EnableDemoUsers())
	fmt.Fprintf(ctx.Output, "Guest Access: %v\n", ps.EnableGuestAccess())
	if ps.AdminUserName() != "" {
		fmt.Fprintf(ctx.Output, "Admin User: %s\n", ps.AdminUserName())
	}
	// schema gap: GuestUserRoleQualifiedName() does not exist; gen uses GuestUserRoleName()
	if ps.GuestUserRoleName() != "" {
		fmt.Fprintf(ctx.Output, "Guest User Role: %s\n", ps.GuestUserRoleName())
	}
	fmt.Fprintf(ctx.Output, "User Roles: %d\n", len(ps.UserRolesItems()))
	fmt.Fprintf(ctx.Output, "Demo Users: %d\n", len(ps.DemoUsersItems()))

	if pp != nil {
		fmt.Fprintf(ctx.Output, "\nPassword Policy:\n")
		fmt.Fprintf(ctx.Output, "  Minimum Length: %d\n", pp.MinimumLength())
		fmt.Fprintf(ctx.Output, "  Require Digit: %v\n", pp.RequireDigit())
		fmt.Fprintf(ctx.Output, "  Require Mixed Case: %v\n", pp.RequireMixedCase())
		fmt.Fprintf(ctx.Output, "  Require Symbol: %v\n", pp.RequireSymbol())
	}
	return nil
}

// listModuleRolesGen handles SHOW MODULE ROLES [IN module] using
// gen-typed ModuleSecurity units from listModuleSecurityWithContainerGen.
func listModuleRolesGen(ctx *ExecContext, moduleName string) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}
	pairs, err := listModuleSecurityWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("read module security", err)
	}
	result := &TableResult{Columns: []string{"Qualified Name", "Module", "Role", "Description"}}
	for _, p := range pairs {
		modName := h.GetModuleName(p.ContainerID)
		if modName == "" {
			continue
		}
		if moduleName != "" && modName != moduleName {
			continue
		}
		// schema gap: ModuleRolesItems() returns []element.Element; type-assert
		// each entry to *genSec.ModuleRole. Remove when gen narrows return to []*ModuleRole.
		for _, mr := range p.MS.ModuleRolesItems() {
			typed, ok := mr.(*genSec.ModuleRole)
			if !ok {
				continue
			}
			qn := modName + "." + typed.Name()
			result.Rows = append(result.Rows, []any{qn, modName, typed.Name(), typed.Description()})
		}
	}
	result.Summary = fmt.Sprintf("(%d module roles)", len(result.Rows))
	return writeResult(ctx, result)
}

// listUserRolesGen handles SHOW USER ROLES using the gen-typed ProjectSecurity
// from ctx.Security. Mirrors listUserRoles output shape; only the type source changes.
func listUserRolesGen(ctx *ExecContext) error {
	ps, err := getProjectSecurityGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", fmt.Errorf("ProjectSecurity not found"))
	}
	result := &TableResult{Columns: []string{"Name", "Module Roles", "Manage All", "Check Security"}}
	for _, ur := range ps.UserRolesItems() {
		typed, ok := ur.(*genSec.UserRole)
		if !ok {
			continue
		}
		ma := "No"
		if typed.ManageAllRoles() {
			ma = "Yes"
		}
		cs := "No"
		if typed.CheckSecurity() {
			cs = "Yes"
		}
		result.Rows = append(result.Rows, []any{typed.Name(), len(typed.ModuleRolesQualifiedNames()), ma, cs})
	}
	result.Summary = fmt.Sprintf("(%d user roles)", len(result.Rows))
	return writeResult(ctx, result)
}

// securityLevelDisplay maps gen-typed BSON SecurityLevel constants to the
// human-friendly labels used by `show project security`. Mirrors
// security.SecurityLevelDisplay (sdk/security/security.go) without
// importing the sdk package.
func securityLevelDisplay(level string) string {
	switch level {
	case "CheckNothing":
		return "Off"
	case "CheckFormsAndMicroflows":
		return "Prototype / demo"
	case "CheckEverything":
		return "Production"
	default:
		return level
	}
}

// entityRuleRightStringsGen mirrors entityRuleRightStrings for the
// gen-typed AccessRule. Gen exposes DefaultMemberAccessRights() as a
// plain string ("ReadOnly", "ReadWrite", "None") and per-member
// AccessRights via MemberAccessesItems()[i].(*MemberAccess).AccessRights().
func entityRuleRightStringsGen(rule *genDm.AccessRule) []string {
	if rule == nil {
		return nil
	}
	var rights []string
	if rule.AllowCreate() {
		rights = append(rights, "C")
	}
	def := rule.DefaultMemberAccessRights()
	rr := def == "ReadOnly" || def == "ReadWrite"
	rw := def == "ReadWrite"
	for _, m := range rule.MemberAccessesItems() {
		ma, ok := m.(*genDm.MemberAccess)
		if !ok {
			continue
		}
		switch ma.AccessRights() {
		case "ReadWrite":
			rr = true
			rw = true
		case "ReadOnly":
			rr = true
		}
	}
	if rr {
		rights = append(rights, "R")
	}
	if rw {
		rights = append(rights, "W")
	}
	if rule.AllowDelete() {
		rights = append(rights, "D")
	}
	return rights
}

// describeModuleRoleGen handles DESCRIBE MODULE ROLE Module.Role using
// gen-typed ModuleSecurity units. Emits a re-executable CREATE MODULE ROLE
// statement followed by inclusion in user roles (if any).
// Not yet wired into executor_query.go dispatch — DESCRIBE MODULE ROLE still
// routes to legacy describeModuleRole. Dispatcher cutover in task A10.
func describeModuleRoleGen(ctx *ExecContext, name ast.QualifiedName) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}
	pairs, err := listModuleSecurityWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("read module security", err)
	}
	for _, p := range pairs {
		modName := h.GetModuleName(p.ContainerID)
		if name.Module != "" && modName != name.Module {
			continue
		}
		// schema gap: ModuleRolesItems() returns []element.Element; type-assert
		// each entry to *genSec.ModuleRole. Remove when gen narrows return to []*ModuleRole.
		for _, mr := range p.MS.ModuleRolesItems() {
			typed, ok := mr.(*genSec.ModuleRole)
			if !ok || typed.Name() != name.Name {
				continue
			}
			fmt.Fprintf(ctx.Output, "create module role %s.%s", modName, typed.Name())
			if typed.Description() != "" {
				fmt.Fprintf(ctx.Output, " description '%s'", typed.Description())
			}
			fmt.Fprintln(ctx.Output, ";")
			fmt.Fprintln(ctx.Output, "/")
			qualifiedRole := modName + "." + typed.Name()
			if ps, psErr := getProjectSecurityGen(ctx); psErr == nil && ps != nil {
				var includedBy []string
				for _, ur := range ps.UserRolesItems() {
					urTyped, ok := ur.(*genSec.UserRole)
					if !ok {
						continue
					}
					for _, mref := range urTyped.ModuleRolesQualifiedNames() {
						if mref == qualifiedRole {
							includedBy = append(includedBy, urTyped.Name())
						}
					}
				}
				if len(includedBy) > 0 {
					fmt.Fprintf(ctx.Output, "\n-- Included in user roles: %s\n", strings.Join(includedBy, ", "))
				}
			}
			return nil
		}
	}
	return mdlerrors.NewNotFound("module role", name.String())
}

// describeUserRoleGen handles DESCRIBE USER ROLE '<name>' using the gen-typed
// ProjectSecurity from ctx.Security. Mirrors describeUserRole output exactly —
// same quoting style, module role parentheses, manage-all-roles suffix,
// semicolon + slash terminators, and trailing comment lines for description
// and check-security flag.
//
// schema gap: UserRolesItems() returns []element.Element; type-assert each
// entry to *genSec.UserRole. Remove cast when gen narrows return to []*UserRole.
func describeUserRoleGen(ctx *ExecContext, name ast.QualifiedName) error {
	ps, err := getProjectSecurityGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewNotFound("user role", name.Name)
	}
	for _, ur := range ps.UserRolesItems() {
		typed, ok := ur.(*genSec.UserRole)
		if !ok || typed.Name() != name.Name {
			continue
		}
		fmt.Fprintf(ctx.Output, "create user role %s", typed.Name())

		// Module roles
		moduleRoles := typed.ModuleRolesQualifiedNames()
		if len(moduleRoles) > 0 {
			fmt.Fprintf(ctx.Output, " (%s)", strings.Join(moduleRoles, ", "))
		}

		if typed.ManageAllRoles() {
			fmt.Fprint(ctx.Output, " manage all roles")
		}

		fmt.Fprintln(ctx.Output, ";")
		fmt.Fprintln(ctx.Output, "/")

		// Show description if present
		if typed.Description() != "" {
			fmt.Fprintf(ctx.Output, "\n-- Description: %s\n", typed.Description())
		}

		// Show check security flag
		if typed.CheckSecurity() {
			fmt.Fprintln(ctx.Output, "-- Check security: enabled")
		}

		return nil
	}
	return mdlerrors.NewNotFound("user role", name.Name)
}

// describeDemoUserGen handles DESCRIBE DEMO USER 'name' using the gen-typed
// ProjectSecurity. Mirrors legacy describeDemoUser output exactly.
// schema gap: DemoUsersItems() returns []element.Element; type-assert each
// entry to *genSec.DemoUser. Remove cast when gen narrows return to []*DemoUser.
func describeDemoUserGen(ctx *ExecContext, userName string) error {
	ps, err := getProjectSecurityGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewNotFound("demo user", userName)
	}
	for _, du := range ps.DemoUsersItems() {
		typed, ok := du.(*genSec.DemoUser)
		if !ok || typed.UserName() != userName {
			continue
		}
		fmt.Fprintf(ctx.Output, "create demo user '%s' password '***'", typed.UserName())
		if typed.EntityQualifiedName() != "" {
			fmt.Fprintf(ctx.Output, " entity %s", typed.EntityQualifiedName())
		}
		roles := typed.UserRolesQualifiedNames()
		if len(roles) > 0 {
			fmt.Fprintf(ctx.Output, " (%s)", strings.Join(roles, ", "))
		}
		fmt.Fprintln(ctx.Output, ";")
		fmt.Fprintln(ctx.Output, "/")
		return nil
	}
	return mdlerrors.NewNotFound("demo user", userName)
}

// listDemoUsersGen handles SHOW DEMO USERS using the gen-typed ProjectSecurity
// from ctx.Security. When demo users are disabled it emits a human-readable
// hint (table format) or an empty TableResult (JSON format).
// schema gap: DemoUsersItems() returns []element.Element; type-assert each
// entry to *genSec.DemoUser. Remove cast when gen narrows return to []*DemoUser.
func listDemoUsersGen(ctx *ExecContext) error {
	ps, err := getProjectSecurityGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", fmt.Errorf("ProjectSecurity not found"))
	}
	if !ps.EnableDemoUsers() {
		if ctx.Format != FormatJSON {
			fmt.Fprintln(ctx.Output, "Demo users are disabled.")
			fmt.Fprintln(ctx.Output, "Enable with: alter project security demo users on;")
			return nil
		}
		return writeResult(ctx, &TableResult{Columns: []string{"User Name", "User Roles"}})
	}
	result := &TableResult{Columns: []string{"User Name", "User Roles"}}
	for _, du := range ps.DemoUsersItems() {
		typed, ok := du.(*genSec.DemoUser)
		if !ok {
			continue
		}
		rolesStr := strings.Join(typed.UserRolesQualifiedNames(), ", ")
		result.Rows = append(result.Rows, []any{typed.UserName(), rolesStr})
	}
	result.Summary = fmt.Sprintf("(%d demo users)", len(result.Rows))
	return writeResult(ctx, result)
}

// listAccessOnEntityGen handles SHOW ACCESS ON ENTITY Module.Entity using
// the gen-typed domainmodel walk.
func listAccessOnEntityGen(ctx *ExecContext, name *ast.QualifiedName) error {
	if name == nil {
		return mdlerrors.NewValidation("entity name required")
	}

	entity, _, err := findEntityGen(ctx, *name)
	if err != nil {
		return mdlerrors.NewBackend("get entity", err)
	}
	if entity == nil {
		return mdlerrors.NewNotFound("entity", name.String())
	}

	attrNames := make(map[string]string)
	for _, a := range entity.AttributesItems() {
		attr, ok := a.(*genDm.Attribute)
		if !ok {
			continue
		}
		attrNames[string(attr.ID())] = attr.Name()
		attrNames[attr.Name()] = attr.Name()
	}

	if ctx.Format == FormatJSON {
		result := &TableResult{
			Columns: []string{"Rule", "Roles", "Rights", "DefaultMemberAccess", "MemberAccess", "XPath"},
		}
		ruleNum := 0
		for _, r := range entity.AccessRulesItems() {
			rule, ok := r.(*genDm.AccessRule)
			if !ok {
				continue
			}
			ruleNum++
			var memberParts []string
			for _, m := range rule.MemberAccessesItems() {
				ma, ok := m.(*genDm.MemberAccess)
				if !ok {
					continue
				}
				memberName := ma.AttributeQualifiedName()
				if memberName == "" {
					memberName = ma.AssociationQualifiedName()
				}
				if mapped, ok := attrNames[memberName]; ok {
					memberName = mapped
				} else if attrName := extractAttrNameFromQualified(memberName); attrName != "" {
					memberName = attrName
				}
				memberParts = append(memberParts, memberName+":"+ma.AccessRights())
			}
			result.Rows = append(result.Rows, []any{
				ruleNum,
				strings.Join(entityRuleRoleStringsGen(rule), ", "),
				strings.ToLower(strings.Join(entityRuleRightStringsGen(rule), ", ")),
				rule.DefaultMemberAccessRights(),
				strings.Join(memberParts, ", "),
				rule.XPathConstraint(),
			})
		}
		return writeResult(ctx, result)
	}

	if len(entity.AccessRulesItems()) == 0 {
		fmt.Fprintf(ctx.Output, "No access rules on %s\n", name)
		return nil
	}

	fmt.Fprintf(ctx.Output, "Access rules for %s.%s:\n\n", name.Module, name.Name)

	ruleNum := 0
	for _, r := range entity.AccessRulesItems() {
		rule, ok := r.(*genDm.AccessRule)
		if !ok {
			continue
		}
		ruleNum++
		var rights []string
		for _, right := range entityRuleRightStringsGen(rule) {
			rights = append(rights, strings.ToLower(right))
		}
		fmt.Fprintf(ctx.Output, "Rule %d: %s\n", ruleNum, strings.Join(entityRuleRoleStringsGen(rule), ", "))
		fmt.Fprintf(ctx.Output, "  Rights: %s\n", strings.Join(rights, ", "))

		if rule.DefaultMemberAccessRights() != "" {
			fmt.Fprintf(ctx.Output, "  Default member access: %s\n", rule.DefaultMemberAccessRights())
		}

		for _, m := range rule.MemberAccessesItems() {
			ma, ok := m.(*genDm.MemberAccess)
			if !ok {
				continue
			}
			memberName := ma.AttributeQualifiedName()
			if memberName == "" {
				memberName = ma.AssociationQualifiedName()
			}
			if mapped, ok := attrNames[memberName]; ok {
				memberName = mapped
			} else if attrName := extractAttrNameFromQualified(memberName); attrName != "" {
				memberName = attrName
			}
			fmt.Fprintf(ctx.Output, "  %s: %s\n", memberName, ma.AccessRights())
		}

		if rule.XPathConstraint() != "" {
			fmt.Fprintf(ctx.Output, "  where '%s'\n", rule.XPathConstraint())
		}
		fmt.Fprintln(ctx.Output)
	}

	return nil
}

// listAccessOnPageGen handles SHOW ACCESS ON PAGE Module.Page using
// the same backend.ListPages() path as legacy listAccessOnPage. Pages
// domain has no gen migration yet (Stage 3.3 priority #5), so this is
// a passthrough rename — no behavior change. Once pages migrates, the
// AllowedRoles read can be replaced by gen-typed Page.AllowedRoles.
//
// Wiring: not yet routed from executor_query.go; the dispatcher cutover
// happens in task A10 once all A1-A9 gen twins exist.
func listAccessOnPageGen(ctx *ExecContext, name *ast.QualifiedName) error {
	if name == nil {
		return mdlerrors.NewValidation("page name required")
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pairs, err := listPagesWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}

	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modName := h.GetModuleName(h.FindModuleID(model.ID(p.ContainerID)))
		if modName == name.Module && p.Elem.Name() == name.Name {
			allowed := p.Elem.AllowedRolesQualifiedNames()
			if ctx.Format == FormatJSON {
				result := &TableResult{Columns: []string{"Module", "Role"}}
				for _, role := range allowed {
					parts := strings.SplitN(role, ".", 2)
					mod, r := "", role
					if len(parts) == 2 {
						mod, r = parts[0], parts[1]
					}
					result.Rows = append(result.Rows, []any{mod, r})
				}
				return writeResult(ctx, result)
			}
			if len(allowed) == 0 {
				fmt.Fprintf(ctx.Output, "No module roles granted view access on %s.%s\n", modName, p.Elem.Name())
				return nil
			}
			fmt.Fprintf(ctx.Output, "Allowed module roles for %s.%s:\n", modName, p.Elem.Name())
			for _, role := range allowed {
				fmt.Fprintf(ctx.Output, "  %s\n", role)
			}
			return nil
		}
	}

	return mdlerrors.NewNotFound("page", name.String())
}

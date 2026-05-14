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
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
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

	allMS, err := ctx.Backend.ListModuleSecurity()
	if err != nil {
		return mdlerrors.NewBackend("read module security", err)
	}

	// Build role list for the target module(s).
	type moduleRoleInfo struct {
		moduleName string
		roleName   string
	}
	var roles []moduleRoleInfo
	for _, ms := range allMS {
		modName := h.GetModuleName(ms.ContainerID)
		if modName == "" {
			continue
		}
		if moduleName != "" && modName != moduleName {
			continue
		}
		for _, mr := range ms.ModuleRoles {
			roles = append(roles, moduleRoleInfo{modName, mr.Name})
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

	dms, err := ctx.Backend.ListDomainModels()
	if err != nil {
		return mdlerrors.NewBackend("list domain models", err)
	}

	fmt.Fprintf(ctx.Output, "Security Matrix")
	if moduleName != "" {
		fmt.Fprintf(ctx.Output, " for %s", moduleName)
	}
	fmt.Fprintln(ctx.Output, ":")
	fmt.Fprintln(ctx.Output)

	// Entities section — domainmodel reads stay in cmd_security.go land.
	fmt.Fprintln(ctx.Output, "## Entity Access")
	fmt.Fprintln(ctx.Output)

	entityFound := false
	for _, dm := range dms {
		modID := h.FindModuleID(dm.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleName != "" && modName != moduleName {
			continue
		}

		for _, entity := range dm.Entities {
			if len(entity.AccessRules) == 0 {
				continue
			}
			entityFound = true
			fmt.Fprintf(ctx.Output, "### %s.%s\n", modName, entity.Name)

			for _, rule := range entity.AccessRules {
				roleStrs := entityRuleRoleStrings(rule)
				rights := entityRuleRightStrings(rule)
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

	// Page section — no microflow involvement; reuse Backend list as in
	// the legacy path. (Page-side migration is outside 3.2.5a.)
	fmt.Fprintln(ctx.Output, "## Page Access")
	fmt.Fprintln(ctx.Output)

	pages, err := ctx.Backend.ListPages()
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}

	pgFound := false
	for _, pg := range pages {
		if len(pg.AllowedRoles) == 0 {
			continue
		}
		modID := h.FindModuleID(pg.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleName != "" && modName != moduleName {
			continue
		}
		pgFound = true
		var roleStrs []string
		for _, r := range pg.AllowedRoles {
			roleStrs = append(roleStrs, string(r))
		}
		fmt.Fprintf(ctx.Output, "  %s.%s: %s\n", modName, pg.Name, strings.Join(roleStrs, ", "))
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

	// Entities — same path as legacy.
	dms, _ := ctx.Backend.ListDomainModels()
	for _, dm := range dms {
		modID := h.FindModuleID(dm.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleName != "" && modName != moduleName {
			continue
		}
		for _, entity := range dm.Entities {
			for _, rule := range entity.AccessRules {
				roleStrs := entityRuleRoleStrings(rule)
				rights := entityRuleRightStrings(rule)
				tr.Rows = append(tr.Rows, []any{
					"Entity",
					modName + "." + entity.Name,
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

	// Pages — no microflow involvement; reuse Backend as in legacy path.
	pages, _ := ctx.Backend.ListPages()
	for _, pg := range pages {
		if len(pg.AllowedRoles) == 0 {
			continue
		}
		modID := h.FindModuleID(pg.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleName != "" && modName != moduleName {
			continue
		}
		var roleStrs []string
		for _, r := range pg.AllowedRoles {
			roleStrs = append(roleStrs, string(r))
		}
		tr.Rows = append(tr.Rows, []any{
			"Page",
			modName + "." + pg.Name,
			strings.Join(roleStrs, ", "),
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

// entityRuleRoleStrings returns the role names for an entity access
// rule, preferring ModuleRoleNames over the raw ModuleRoles ID list.
func entityRuleRoleStrings(rule *domainmodel.AccessRule) []string {
	if len(rule.ModuleRoleNames) > 0 {
		out := make([]string, len(rule.ModuleRoleNames))
		copy(out, rule.ModuleRoleNames)
		return out
	}
	out := make([]string, 0, len(rule.ModuleRoles))
	for _, rid := range rule.ModuleRoles {
		out = append(out, string(rid))
	}
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

// entityRuleRightStrings computes CRUD right tokens for an entity
// access rule. Mirrors the rights computation inline in
// listSecurityMatrix / listSecurityMatrixJSON.
func entityRuleRightStrings(rule *domainmodel.AccessRule) []string {
	var rights []string
	if rule.AllowCreate {
		rights = append(rights, "C")
	}
	rr := rule.DefaultMemberAccessRights == domainmodel.MemberAccessRightsReadOnly ||
		rule.DefaultMemberAccessRights == domainmodel.MemberAccessRightsReadWrite
	rw := rule.DefaultMemberAccessRights == domainmodel.MemberAccessRightsReadWrite
	for _, ma := range rule.MemberAccesses {
		if ma.AccessRights == domainmodel.MemberAccessRightsReadOnly || ma.AccessRights == domainmodel.MemberAccessRightsReadWrite {
			rr = true
		}
		if ma.AccessRights == domainmodel.MemberAccessRightsReadWrite {
			rw = true
		}
	}
	if rr {
		rights = append(rights, "R")
	}
	if rw {
		rights = append(rights, "W")
	}
	if rule.AllowDelete {
		rights = append(rights, "D")
	}
	return rights
}

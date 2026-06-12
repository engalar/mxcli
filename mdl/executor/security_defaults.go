// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/model"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

const (
	autoDocumentRoleName        = "User"
	autoDocumentRoleDescription = "Auto-created default role for mxcli document access"
)

// defaultDocumentAccessRoles returns a conservative fallback role set for newly
// created pages/microflows when the target module has no module roles at all.
//
// Mendix accepts document access only when it references a role from the same
// module; using an existing role from another module causes CE0148 on freshly
// created documents. To keep mx-check green, auto-create a local `User` module
// role only for modules that currently have zero roles. Modules that already
// manage their own roles keep the existing "no access by default" behavior.
func defaultDocumentAccessRoles(ctx *ExecContext, module *model.Module) []model.ID {
	if module == nil {
		return nil
	}

	ms, err := ctx.SecurityModuleManager.GetModuleSecurityGen(module.ID)
	if err != nil || ms == nil {
		return nil
	}
	if moduleUsesAutoDocumentRoleGen(ms) {
		return []model.ID{model.ID(module.Name + "." + autoDocumentRoleName)}
	}
	if len(ms.ModuleRolesItems()) > 0 {
		return nil
	}

	if err := ctx.SecurityModuleManager.AddModuleRole(model.ID(ms.ID()), autoDocumentRoleName, autoDocumentRoleDescription); err != nil {
		return nil
	}
	return []model.ID{model.ID(module.Name + "." + autoDocumentRoleName)}
}

func moduleUsesAutoDocumentRoleGen(ms *genSec.ModuleSecurity) bool {
	if ms == nil {
		return false
	}
	items := ms.ModuleRolesItems()
	if len(items) != 1 {
		return false
	}
	mr, ok := items[0].(*genSec.ModuleRole)
	if !ok {
		return false
	}
	return mr.Name() == autoDocumentRoleName && mr.Description() == autoDocumentRoleDescription
}

func remapDocumentAccessRoles(ctx *ExecContext, targetModule *model.Module, currentRoles []model.ID) []model.ID {
	if targetModule == nil {
		return nil
	}

	ms, err := ctx.SecurityModuleManager.GetModuleSecurityGen(targetModule.ID)
	if err != nil || ms == nil {
		return nil
	}
	items := ms.ModuleRolesItems()
	if len(items) == 0 || moduleUsesAutoDocumentRoleGen(ms) {
		return defaultDocumentAccessRoles(ctx, targetModule)
	}

	targetRoleNames := make(map[string]bool, len(items))
	for _, item := range items {
		if mr, ok := item.(*genSec.ModuleRole); ok {
			targetRoleNames[mr.Name()] = true
		}
	}

	var remapped []model.ID
	seen := make(map[string]bool)
	for _, qualifiedRole := range currentRoles {
		roleName := string(qualifiedRole)
		if idx := strings.LastIndex(roleName, "."); idx >= 0 {
			roleName = roleName[idx+1:]
		}
		if !targetRoleNames[roleName] {
			continue
		}
		targetQualifiedRole := targetModule.Name + "." + roleName
		if seen[targetQualifiedRole] {
			continue
		}
		seen[targetQualifiedRole] = true
		remapped = append(remapped, model.ID(targetQualifiedRole))
	}

	return remapped
}

func documentRoleStrings(roles []model.ID) []string {
	values := make([]string, 0, len(roles))
	for _, role := range roles {
		values = append(values, string(role))
	}
	return values
}

// filterAutoDocumentRoles removes the auto-created "User" placeholder role
// from a list of qualified role names. The placeholder (Module.User) is added
// by mxcli for mx-check compliance and should not appear in describe output.
func filterAutoDocumentRoles(roles []string) []string {
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		local := r
		if idx := strings.LastIndex(r, "."); idx >= 0 {
			local = r[idx+1:]
		}
		if local == autoDocumentRoleName {
			continue
		}
		out = append(out, r)
	}
	return out
}

func cloneRoleIDs(roles []model.ID) []model.ID {
	if len(roles) == 0 {
		return nil
	}
	cloned := make([]model.ID, len(roles))
	copy(cloned, roles)
	return cloned
}

// pruneInvalidUserRoles removes user roles that no longer have any non-System
// module role assignments. Mendix rejects those roles with CE0157.
func pruneInvalidUserRoles(ctx *ExecContext, _ *genSec.ProjectSecurity) error {
	ps, err := ctx.SecurityProjectManager.GetProjectSecurityGen()
	if err != nil || ps == nil {
		return err
	}

	for _, ur := range ps.UserRolesItems() {
		typed, ok := ur.(*genSec.UserRole)
		if !ok {
			continue
		}
		hasNonSystemRole := false
		for _, moduleRole := range typed.ModuleRolesQualifiedNames() {
			if !strings.HasPrefix(moduleRole, "System.") {
				hasNonSystemRole = true
				break
			}
		}
		if hasNonSystemRole {
			continue
		}
		if err := ctx.SecurityProjectManager.RemoveUserRole(model.ID(ps.ID()), typed.Name()); err != nil {
			return err
		}
		if !ctx.Quiet {
			fmt.Fprintf(ctx.Output, "Dropped invalid user role: %s\n", typed.Name())
		}
	}

	return nil
}

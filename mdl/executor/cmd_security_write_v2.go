// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5b — gen-typed security write paths.
//
// Mirrors the microflow / nanoflow halves of:
//   - execGrantMicroflowAccess        / execRevokeMicroflowAccess
//   - execGrantNanoflowAccess         / execRevokeNanoflowAccess
//   - execDropModuleRole's cascade    (microflow + nanoflow allowed-roles cleanup)
//
// Lookup goes through ctx.Microflows / ctx.Nanoflows (gen path); the
// allowed-roles list lives on the gen Microflow / Nanoflow as a
// `[]string` of qualified names exposed via
// `AllowedModuleRolesQualifiedNames()` and persisted by
// `SetAllowedModuleRolesQualifiedNames` + `Update`.
//
// Page / OData / Workflow / Published-REST grants stay in
// cmd_security_write.go — none of them touch sdk/microflows so they
// were never part of this migration. The entity grant family
// (execGrantEntityAccess / execRevokeEntityAccess) and module-role
// CRUD itself (execCreateModuleRole / the non-microflow tail of
// execDropModuleRole) likewise stay legacy.
//
// The dispatch layer (Stage 3.2.6) will route each statement to its
// `*Gen` variant and delete the sdk-typed originals.

package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/mdl/repos"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// ─────────────────────────────────────────────────────────────
// Fn (HandlerDeps) versions
// ─────────────────────────────────────────────────────────────

// execGrantMicroflowAccessGenFn handles GRANT EXECUTE ON MICROFLOW.
func execGrantMicroflowAccessGenFn(ctx context.Context, s *ast.GrantMicroflowAccessStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if deps.MicroflowRepo == nil {
		return mdlerrors.NewBackend("microflows repo unavailable", nil)
	}

	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	mfs, err := listMicroflowsGenFn(deps.MicroflowRepo)
	if err != nil {
		return mdlerrors.NewBackend("list microflows", err)
	}

	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		modName := genFlowContainerModuleFn(deps.MicroflowRepo, h, model.ID(mf.ID()))
		if modName != s.Microflow.Module || mf.Name() != s.Microflow.Name {
			continue
		}

		var validRoles []ast.QualifiedName
		for _, role := range s.Roles {
			resolved := expandRoleModule(role, modName)
			found, err := validateModuleRoleFn(deps, resolved)
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

		merged, added := mergeAllowedRoles(filterAutoDocumentRoles(mf.AllowedModuleRolesQualifiedNames()), validRoles)
		mf.SetAllowedModuleRolesQualifiedNames(merged)

		if err := deps.MicroflowRepo.Update(mf); err != nil {
			return mdlerrors.NewBackend("update microflow access", err)
		}

		if len(added) == 0 {
			fmt.Fprintf(deps.Output, "All specified roles already have execute access on %s.%s\n", modName, mf.Name())
		} else {
			fmt.Fprintf(deps.Output, "Granted execute access on %s.%s to %s\n", modName, mf.Name(), strings.Join(added, ", "))
		}
		return nil
	}

	return mdlerrors.NewNotFound("microflow", s.Microflow.Module+"."+s.Microflow.Name)
}

// execRevokeMicroflowAccessGenFn handles REVOKE EXECUTE ON MICROFLOW.
func execRevokeMicroflowAccessGenFn(ctx context.Context, s *ast.RevokeMicroflowAccessStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if deps.MicroflowRepo == nil {
		return mdlerrors.NewBackend("microflows repo unavailable", nil)
	}

	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	mfs, err := listMicroflowsGenFn(deps.MicroflowRepo)
	if err != nil {
		return mdlerrors.NewBackend("list microflows", err)
	}

	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		modName := genFlowContainerModuleFn(deps.MicroflowRepo, h, model.ID(mf.ID()))
		if modName != s.Microflow.Module || mf.Name() != s.Microflow.Name {
			continue
		}

		expandedRoles := make([]ast.QualifiedName, len(s.Roles))
		for i, r := range s.Roles {
			expandedRoles[i] = expandRoleModule(r, modName)
		}
		remaining, removed := filterAllowedRoles(mf.AllowedModuleRolesQualifiedNames(), expandedRoles)
		mf.SetAllowedModuleRolesQualifiedNames(remaining)

		if err := deps.MicroflowRepo.Update(mf); err != nil {
			return mdlerrors.NewBackend("update microflow access", err)
		}

		if len(removed) == 0 {
			fmt.Fprintf(deps.Output, "None of the specified roles had execute access on %s.%s\n", modName, mf.Name())
		} else {
			fmt.Fprintf(deps.Output, "Revoked execute access on %s.%s from %s\n", modName, mf.Name(), strings.Join(removed, ", "))
		}
		return nil
	}

	return mdlerrors.NewNotFound("microflow", s.Microflow.Module+"."+s.Microflow.Name)
}

// execGrantNanoflowAccessGenFn handles GRANT EXECUTE ON NANOFLOW.
func execGrantNanoflowAccessGenFn(ctx context.Context, s *ast.GrantNanoflowAccessStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if deps.NanoflowRepo == nil {
		return mdlerrors.NewBackend("nanoflows repo unavailable", nil)
	}

	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	nfs, err := listNanoflowsGenFn(deps.NanoflowRepo)
	if err != nil {
		return mdlerrors.NewBackend("list nanoflows", err)
	}

	for _, nf := range nfs {
		if nf == nil {
			continue
		}
		modName := genFlowContainerModuleFn(deps.NanoflowRepo, h, model.ID(nf.ID()))
		if modName != s.Nanoflow.Module || nf.Name() != s.Nanoflow.Name {
			continue
		}

		var validRoles []ast.QualifiedName
		for _, role := range s.Roles {
			resolved := expandRoleModule(role, modName)
			found, err := validateModuleRoleFn(deps, resolved)
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

		merged, added := mergeAllowedRoles(filterAutoDocumentRoles(nf.AllowedModuleRolesQualifiedNames()), validRoles)
		nf.SetAllowedModuleRolesQualifiedNames(merged)

		if err := deps.NanoflowRepo.Update(nf); err != nil {
			return mdlerrors.NewBackend("update nanoflow access", err)
		}

		if len(added) == 0 {
			fmt.Fprintf(deps.Output, "All specified roles already have execute access on %s.%s\n", modName, nf.Name())
		} else {
			fmt.Fprintf(deps.Output, "Granted execute access on %s.%s to %s\n", modName, nf.Name(), strings.Join(added, ", "))
		}
		return nil
	}

	return mdlerrors.NewNotFound("nanoflow", s.Nanoflow.Module+"."+s.Nanoflow.Name)
}

// execRevokeNanoflowAccessGenFn handles REVOKE EXECUTE ON NANOFLOW.
func execRevokeNanoflowAccessGenFn(ctx context.Context, s *ast.RevokeNanoflowAccessStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if deps.NanoflowRepo == nil {
		return mdlerrors.NewBackend("nanoflows repo unavailable", nil)
	}

	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	nfs, err := listNanoflowsGenFn(deps.NanoflowRepo)
	if err != nil {
		return mdlerrors.NewBackend("list nanoflows", err)
	}

	for _, nf := range nfs {
		if nf == nil {
			continue
		}
		modName := genFlowContainerModuleFn(deps.NanoflowRepo, h, model.ID(nf.ID()))
		if modName != s.Nanoflow.Module || nf.Name() != s.Nanoflow.Name {
			continue
		}

		expandedRoles := make([]ast.QualifiedName, len(s.Roles))
		for i, r := range s.Roles {
			expandedRoles[i] = expandRoleModule(r, modName)
		}
		remaining, removed := filterAllowedRoles(nf.AllowedModuleRolesQualifiedNames(), expandedRoles)
		nf.SetAllowedModuleRolesQualifiedNames(remaining)

		if err := deps.NanoflowRepo.Update(nf); err != nil {
			return mdlerrors.NewBackend("update nanoflow access", err)
		}

		if len(removed) == 0 {
			fmt.Fprintf(deps.Output, "None of the specified roles had execute access on %s.%s\n", modName, nf.Name())
		} else {
			fmt.Fprintf(deps.Output, "Revoked execute access on %s.%s from %s\n", modName, nf.Name(), strings.Join(removed, ", "))
		}
		return nil
	}

	return mdlerrors.NewNotFound("nanoflow", s.Nanoflow.Module+"."+s.Nanoflow.Name)
}

// cascadeRemoveRoleFromMicroflowsGenFn removes a qualified role from every
// microflow in the module.
func cascadeRemoveRoleFromMicroflowsGenFn(ctx context.Context, deps *HandlerDeps, moduleID model.ID, qualifiedRole string) error {
	if deps.MicroflowRepo == nil {
		return nil
	}
	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return nil
	}
	mfs, err := listMicroflowsGenFn(deps.MicroflowRepo)
	if err != nil {
		return nil
	}
	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		containerID, err := deps.MicroflowRepo.GetContainerUUID(model.ID(mf.ID()))
		if err != nil || containerID == "" {
			continue
		}
		modID := h.FindModuleID(containerID)
		if modID != moduleID {
			continue
		}
		remaining, removed := removeRoleFromList(mf.AllowedModuleRolesQualifiedNames(), qualifiedRole)
		if !removed {
			continue
		}
		mf.SetAllowedModuleRolesQualifiedNames(remaining)
		if err := deps.MicroflowRepo.Update(mf); err != nil {
			return mdlerrors.NewBackend(fmt.Sprintf("update microflow %s allowed roles", mf.Name()), err)
		}
		fmt.Fprintf(deps.Output, "Removed %s from microflow %s allowed roles\n", qualifiedRole, mf.Name())
	}
	return nil
}

// cascadeRemoveRoleFromNanoflowsGenFn mirrors the microflow cascade for nanoflows.
func cascadeRemoveRoleFromNanoflowsGenFn(ctx context.Context, deps *HandlerDeps, moduleID model.ID, qualifiedRole string) error {
	if deps.NanoflowRepo == nil {
		return nil
	}
	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return nil
	}
	nfs, err := listNanoflowsGenFn(deps.NanoflowRepo)
	if err != nil {
		return nil
	}
	for _, nf := range nfs {
		if nf == nil {
			continue
		}
		containerID, err := deps.NanoflowRepo.GetContainerUUID(model.ID(nf.ID()))
		if err != nil || containerID == "" {
			continue
		}
		modID := h.FindModuleID(containerID)
		if modID != moduleID {
			continue
		}
		remaining, removed := removeRoleFromList(nf.AllowedModuleRolesQualifiedNames(), qualifiedRole)
		if !removed {
			continue
		}
		nf.SetAllowedModuleRolesQualifiedNames(remaining)
		if err := deps.NanoflowRepo.Update(nf); err != nil {
			return mdlerrors.NewBackend(fmt.Sprintf("update nanoflow %s allowed roles", nf.Name()), err)
		}
		fmt.Fprintf(deps.Output, "Removed %s from nanoflow %s allowed roles\n", qualifiedRole, nf.Name())
	}
	return nil
}

// listMicroflowsGenFn lists all microflows via the gen repository.
func listMicroflowsGenFn(mfRepo repos.MicroflowRepository) ([]*genMf.Microflow, error) {
	if mfRepo == nil {
		return nil, nil
	}
	return mfRepo.ListAll()
}

// listNanoflowsGenFn lists all nanoflows via the gen repository.
func listNanoflowsGenFn(nfRepo repos.NanoflowRepository) ([]*genMf.Nanoflow, error) {
	if nfRepo == nil {
		return nil, nil
	}
	// NanoflowRepository uses List("") to list all.
	return nfRepo.List("")
}

// containerIDResolver is satisfied by both repos.MicroflowRepository and
// repos.NanoflowRepository.
type containerIDResolver interface {
	GetContainerUUID(id model.ID) (model.ID, error)
}

// genFlowContainerModuleFn resolves the module name for a gen-typed flow.
func genFlowContainerModuleFn(repo containerIDResolver, h *ContainerHierarchy, flowID model.ID) string {
	if repo == nil {
		return ""
	}
	containerID, err := repo.GetContainerUUID(flowID)
	if err != nil || containerID == "" {
		return ""
	}
	return h.GetModuleName(h.FindModuleID(containerID))
}

// validateModuleRoleFn checks that a qualified role name exists.
func validateModuleRoleFn(deps *HandlerDeps, role ast.QualifiedName) (bool, error) {
	module, err := findModuleFn(deps.ModuleLister, role.Module)
	if err != nil {
		var nfe *mdlerrors.NotFoundError
		if errors.As(err, &nfe) {
			fmt.Fprintf(deps.Output, "WARNING: module '%s' not found — grant skipped\n", role.Module)
			return false, nil
		}
		return false, mdlerrors.NewBackend(fmt.Sprintf("read module for role %s.%s", role.Module, role.Name), err)
	}

	// Fallback: use Backend directly when Security repo is not available
	// (e.g., mock backends that don't implement repos.SecurityRepository).
	if deps.Security == nil {
		ms, err := deps.Backend.GetModuleSecurityGen(module.ID)
		if err != nil {
			return false, mdlerrors.NewBackend(fmt.Sprintf("read module security for %s", role.Module), err)
		}
		if ms != nil {
			for _, item := range ms.ModuleRolesItems() {
				if item != nil {
					if mi, ok := item.(*genSec.ModuleRole); ok && mi.Name() == role.Name {
						return true, nil
					}
				}
			}
		}
		fmt.Fprintf(deps.Output, "WARNING: module role '%s.%s' not found — grant skipped\n",
			role.Module, role.Name)
		return false, nil
	}

	ms, err := deps.Security.GetModuleSecurity(module.ID)
	if err != nil {
		return false, mdlerrors.NewBackend(fmt.Sprintf("read module security for %s", role.Module), err)
	}

	if ms != nil {
		for _, item := range ms.ModuleRolesItems() {
			if mr, ok := item.(*genSec.ModuleRole); ok && mr.Name() == role.Name {
				return true, nil
			}
		}
	}

	fmt.Fprintf(deps.Output, "WARNING: module role '%s.%s' not found — grant skipped\n",
		role.Module, role.Name)
	return false, nil
}

// ─────────────────────────────────────────────────────────────
// Old ExecContext wrappers (delegate to Fn versions)
// ─────────────────────────────────────────────────────────────





func cascadeRemoveRoleFromMicroflowsGen(ctx *ExecContext, moduleID model.ID, qualifiedRole string) error {
	return cascadeRemoveRoleFromMicroflowsGenFn(ctx, execContextToDeps(ctx), moduleID, qualifiedRole)
}

func cascadeRemoveRoleFromNanoflowsGen(ctx *ExecContext, moduleID model.ID, qualifiedRole string) error {
	return cascadeRemoveRoleFromNanoflowsGenFn(ctx, execContextToDeps(ctx), moduleID, qualifiedRole)
}

// ─────────────────────────────────────────────────────────────
// Stateless helpers (no ctx/deps needed — unchanged from original)
// ─────────────────────────────────────────────────────────────

// mergeAllowedRoles computes the union of an existing qualified-role
// list and a set of newly-granted roles. Returns the merged list and
// the slice of qualified names actually added (skip duplicates).
func mergeAllowedRoles(existing []string, toAdd []ast.QualifiedName) (merged []string, added []string) {
	seen := make(map[string]bool, len(existing))
	merged = make([]string, 0, len(existing)+len(toAdd))
	for _, r := range existing {
		if seen[r] {
			continue
		}
		seen[r] = true
		merged = append(merged, r)
	}
	for _, role := range toAdd {
		qn := role.Module + "." + role.Name
		if seen[qn] {
			continue
		}
		seen[qn] = true
		merged = append(merged, qn)
		added = append(added, qn)
	}
	return merged, added
}

// filterAllowedRoles removes a set of roles from an existing role
// list. Returns the remaining list and the actually-removed slice.
func filterAllowedRoles(existing []string, toRemove []ast.QualifiedName) (remaining []string, removed []string) {
	dropSet := make(map[string]bool, len(toRemove))
	for _, role := range toRemove {
		dropSet[role.Module+"."+role.Name] = true
	}
	remaining = make([]string, 0, len(existing))
	for _, r := range existing {
		if dropSet[r] {
			removed = append(removed, r)
			continue
		}
		remaining = append(remaining, r)
	}
	return remaining, removed
}

// removeRoleFromList removes a single qualified role from a list.
func removeRoleFromList(existing []string, qualifiedRole string) (remaining []string, removed bool) {
	remaining = make([]string, 0, len(existing))
	for _, r := range existing {
		if r == qualifiedRole {
			removed = true
			continue
		}
		remaining = append(remaining, r)
	}
	return remaining, removed
}

// expandRoleModule returns role with Module set to defaultModule when
// the parsed role has no module prefix.
func expandRoleModule(role ast.QualifiedName, defaultModule string) ast.QualifiedName {
	if role.Module == "" {
		return ast.QualifiedName{Module: defaultModule, Name: role.Name}
	}
	return role
}

// genMicroflowAllowedRoles is a thin re-export of the gen accessor so
// the test file can read the persisted state without importing
// modelsdk/gen/microflows directly.
func genMicroflowAllowedRoles(mf *genMf.Microflow) []string {
	if mf == nil {
		return nil
	}
	return mf.AllowedModuleRolesQualifiedNames()
}

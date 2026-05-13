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
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// ─────────────────────────────────────────────────────────────────
// GRANT / REVOKE EXECUTE ON MICROFLOW
// ─────────────────────────────────────────────────────────────────

// execGrantMicroflowAccessGen handles GRANT EXECUTE ON MICROFLOW.
// Mirrors execGrantMicroflowAccess but resolves the microflow via
// ctx.Microflows and persists the merged role list via
// ctx.Microflows.Update (no Backend.UpdateAllowedRoles).
func execGrantMicroflowAccessGen(ctx *ExecContext, s *ast.GrantMicroflowAccessStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if ctx.Microflows == nil {
		return mdlerrors.NewBackend("microflows repo unavailable", nil)
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
		if modName != s.Microflow.Module || mf.Name() != s.Microflow.Name {
			continue
		}

		for _, role := range s.Roles {
			if err := validateModuleRole(ctx, role); err != nil {
				return err
			}
		}

		merged, added := mergeAllowedRoles(mf.AllowedModuleRolesQualifiedNames(), s.Roles)
		mf.SetAllowedModuleRolesQualifiedNames(merged)

		if err := ctx.Microflows.Update(mf); err != nil {
			return mdlerrors.NewBackend("update microflow access", err)
		}

		if len(added) == 0 {
			fmt.Fprintf(ctx.Output, "All specified roles already have execute access on %s.%s\n", modName, mf.Name())
		} else {
			fmt.Fprintf(ctx.Output, "Granted execute access on %s.%s to %s\n", modName, mf.Name(), strings.Join(added, ", "))
		}
		return nil
	}

	return mdlerrors.NewNotFound("microflow", s.Microflow.Module+"."+s.Microflow.Name)
}

// execRevokeMicroflowAccessGen handles REVOKE EXECUTE ON MICROFLOW.
func execRevokeMicroflowAccessGen(ctx *ExecContext, s *ast.RevokeMicroflowAccessStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if ctx.Microflows == nil {
		return mdlerrors.NewBackend("microflows repo unavailable", nil)
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
		if modName != s.Microflow.Module || mf.Name() != s.Microflow.Name {
			continue
		}

		remaining, removed := filterAllowedRoles(mf.AllowedModuleRolesQualifiedNames(), s.Roles)
		mf.SetAllowedModuleRolesQualifiedNames(remaining)

		if err := ctx.Microflows.Update(mf); err != nil {
			return mdlerrors.NewBackend("update microflow access", err)
		}

		if len(removed) == 0 {
			fmt.Fprintf(ctx.Output, "None of the specified roles had execute access on %s.%s\n", modName, mf.Name())
		} else {
			fmt.Fprintf(ctx.Output, "Revoked execute access on %s.%s from %s\n", modName, mf.Name(), strings.Join(removed, ", "))
		}
		return nil
	}

	return mdlerrors.NewNotFound("microflow", s.Microflow.Module+"."+s.Microflow.Name)
}

// ─────────────────────────────────────────────────────────────────
// GRANT / REVOKE EXECUTE ON NANOFLOW
// ─────────────────────────────────────────────────────────────────

// execGrantNanoflowAccessGen handles GRANT EXECUTE ON NANOFLOW.
func execGrantNanoflowAccessGen(ctx *ExecContext, s *ast.GrantNanoflowAccessStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if ctx.Nanoflows == nil {
		return mdlerrors.NewBackend("nanoflows repo unavailable", nil)
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
		if modName != s.Nanoflow.Module || nf.Name() != s.Nanoflow.Name {
			continue
		}

		for _, role := range s.Roles {
			if err := validateModuleRole(ctx, role); err != nil {
				return err
			}
		}

		merged, added := mergeAllowedRoles(nf.AllowedModuleRolesQualifiedNames(), s.Roles)
		nf.SetAllowedModuleRolesQualifiedNames(merged)

		if err := ctx.Nanoflows.Update(nf); err != nil {
			return mdlerrors.NewBackend("update nanoflow access", err)
		}

		if len(added) == 0 {
			fmt.Fprintf(ctx.Output, "All specified roles already have execute access on %s.%s\n", modName, nf.Name())
		} else {
			fmt.Fprintf(ctx.Output, "Granted execute access on %s.%s to %s\n", modName, nf.Name(), strings.Join(added, ", "))
		}
		return nil
	}

	return mdlerrors.NewNotFound("nanoflow", s.Nanoflow.Module+"."+s.Nanoflow.Name)
}

// execRevokeNanoflowAccessGen handles REVOKE EXECUTE ON NANOFLOW.
func execRevokeNanoflowAccessGen(ctx *ExecContext, s *ast.RevokeNanoflowAccessStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if ctx.Nanoflows == nil {
		return mdlerrors.NewBackend("nanoflows repo unavailable", nil)
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
		if modName != s.Nanoflow.Module || nf.Name() != s.Nanoflow.Name {
			continue
		}

		remaining, removed := filterAllowedRoles(nf.AllowedModuleRolesQualifiedNames(), s.Roles)
		nf.SetAllowedModuleRolesQualifiedNames(remaining)

		if err := ctx.Nanoflows.Update(nf); err != nil {
			return mdlerrors.NewBackend("update nanoflow access", err)
		}

		if len(removed) == 0 {
			fmt.Fprintf(ctx.Output, "None of the specified roles had execute access on %s.%s\n", modName, nf.Name())
		} else {
			fmt.Fprintf(ctx.Output, "Revoked execute access on %s.%s from %s\n", modName, nf.Name(), strings.Join(removed, ", "))
		}
		return nil
	}

	return mdlerrors.NewNotFound("nanoflow", s.Nanoflow.Module+"."+s.Nanoflow.Name)
}

// ─────────────────────────────────────────────────────────────────
// DROP MODULE ROLE — cascade halves for microflows / nanoflows
// ─────────────────────────────────────────────────────────────────

// cascadeRemoveRoleFromMicroflowsGen removes a qualified role from the
// allowed-roles list of every microflow living in the named module.
// Reports successful removals on ctx.Output. Mirrors the sdk-typed
// inline cascade in execDropModuleRole.
//
// Returns nil if the gen repo is unavailable so the caller can fall
// back to legacy without aborting (parallel-track contract).
func cascadeRemoveRoleFromMicroflowsGen(ctx *ExecContext, moduleID model.ID, qualifiedRole string) error {
	if ctx.Microflows == nil {
		return nil
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil
	}
	mfs, err := listMicroflowsGen(ctx)
	if err != nil {
		return nil
	}
	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		containerID, err := ctx.Microflows.GetContainerUUID(model.ID(mf.ID()))
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
		if err := ctx.Microflows.Update(mf); err != nil {
			return mdlerrors.NewBackend(fmt.Sprintf("update microflow %s allowed roles", mf.Name()), err)
		}
		fmt.Fprintf(ctx.Output, "Removed %s from microflow %s allowed roles\n", qualifiedRole, mf.Name())
	}
	return nil
}

// cascadeRemoveRoleFromNanoflowsGen mirrors the microflow cascade for
// nanoflows.
func cascadeRemoveRoleFromNanoflowsGen(ctx *ExecContext, moduleID model.ID, qualifiedRole string) error {
	if ctx.Nanoflows == nil {
		return nil
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil
	}
	nfs, err := listNanoflowsGen(ctx)
	if err != nil {
		return nil
	}
	for _, nf := range nfs {
		if nf == nil {
			continue
		}
		// Nanoflows live alongside microflows in the unit table — same
		// container-resolution path as genFlowContainerModule.
		containerID, err := ctx.Microflows.GetContainerUUID(model.ID(nf.ID()))
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
		if err := ctx.Nanoflows.Update(nf); err != nil {
			return mdlerrors.NewBackend(fmt.Sprintf("update nanoflow %s allowed roles", nf.Name()), err)
		}
		fmt.Fprintf(ctx.Output, "Removed %s from nanoflow %s allowed roles\n", qualifiedRole, nf.Name())
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────
// Small role-list helpers (string-typed, gen path uses these)
// ─────────────────────────────────────────────────────────────────

// mergeAllowedRoles computes the union of an existing qualified-role
// list and a set of newly-granted roles. Returns the merged list and
// the slice of qualified names actually added (skip duplicates).
// Mirrors the inline merge inside execGrantMicroflowAccess.
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
// list. Returns the remaining list and the actually-removed slice
// (useful for the user-facing diff message). Mirrors the inline
// filter inside execRevokeMicroflowAccess.
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
// Returns the filtered list and whether anything changed. Mirrors
// the per-microflow Backend.RemoveFromAllowedRoles call inside
// execDropModuleRole's cascade.
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

// genMicroflowAllowedRoles is a thin re-export of the gen accessor so
// the test file can read the persisted state without importing
// modelsdk/gen/microflows directly. Returns nil when mf is nil.
func genMicroflowAllowedRoles(mf *genMf.Microflow) []string {
	if mf == nil {
		return nil
	}
	return mf.AllowedModuleRolesQualifiedNames()
}

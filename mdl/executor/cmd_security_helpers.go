// SPDX-License-Identifier: Apache-2.0

// Package executor — security helpers shared across cmd_security_*_gen.go.
//
// Stage 3.3.1.E2 extracted these from the deleted cmd_security.go /
// cmd_security_write.go because the gen-typed write twins (D1-D9) all
// depend on validateModuleRole, and dispatch still references the
// workflow-access stubs (workflows have no document-level AllowedModuleRoles).
package executor

import (
	"errors"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// validateModuleRole checks that a module role exists in the project using
// the gen-typed ModuleSecurity reader.
//
// Returns (true, nil) when the role exists, (false, nil) when it is missing
// (a WARNING is printed so the user is informed), and (false, err) for backend
// failures (module not found, read error) which remain fatal.
//
// Callers must skip phantom roles (found == false) rather than forwarding them
// to the BSON merge — otherwise ghost role names end up in the MPR.
func validateModuleRole(ctx *ExecContext, role ast.QualifiedName) (bool, error) {
	module, err := findModule(ctx, role.Module)
	if err != nil {
		// Module not found is a non-fatal warning — skip the grant for this role.
		// Only real backend failures (I/O errors, DB errors) remain fatal.
		var nfe *mdlerrors.NotFoundError
		if errors.As(err, &nfe) {
			fmt.Fprintf(ctx.Output, "WARNING: module '%s' not found — grant skipped\n", role.Module)
			return false, nil
		}
		return false, mdlerrors.NewBackend(fmt.Sprintf("read module for role %s.%s", role.Module, role.Name), err)
	}

	ms, err := ctx.SecurityModuleManager.GetModuleSecurityGen(module.ID)
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

	fmt.Fprintf(ctx.Output, "WARNING: module role '%s.%s' not found — grant skipped\n",
		role.Module, role.Name)
	return false, nil
}

// listAccessOnWorkflow handles SHOW ACCESS ON WORKFLOW. Workflows do not
// have document-level AllowedModuleRoles (unlike microflows and pages),
// so the operation is not supported.
func listAccessOnWorkflow(ctx *ExecContext, name *ast.QualifiedName) error {
	return mdlerrors.NewUnsupported("show access on workflow is not supported: Mendix workflows do not have document-level AllowedModuleRoles (unlike microflows and pages). Workflow access is controlled through the microflow that triggers the workflow and UserTask targeting")
}

// execGrantWorkflowAccess handles GRANT EXECUTE ON WORKFLOW. Same reason
// as listAccessOnWorkflow: not supported.
func execGrantWorkflowAccess(ctx *ExecContext, s *ast.GrantWorkflowAccessStmt) error {
	return mdlerrors.NewUnsupported("grant execute on workflow is not supported: Mendix workflows do not have document-level AllowedModuleRoles (unlike microflows and pages). Workflow access is controlled through the microflow that triggers the workflow and UserTask targeting")
}

// execRevokeWorkflowAccess handles REVOKE EXECUTE ON WORKFLOW. Not supported.
func execRevokeWorkflowAccess(ctx *ExecContext, s *ast.RevokeWorkflowAccessStmt) error {
	return mdlerrors.NewUnsupported("revoke execute on workflow is not supported: Mendix workflows do not have document-level AllowedModuleRoles (unlike microflows and pages). Workflow access is controlled through the microflow that triggers the workflow and UserTask targeting")
}

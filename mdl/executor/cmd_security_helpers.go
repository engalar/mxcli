// SPDX-License-Identifier: Apache-2.0

// Package executor — security helpers shared across cmd_security_*_gen.go.
//
// Stage 3.3.1.E2 extracted these from the deleted cmd_security.go /
// cmd_security_write.go because the gen-typed write twins (D1-D9) all
// depend on validateModuleRole, and dispatch still references the
// workflow-access stubs (workflows have no document-level AllowedModuleRoles).
package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// validateModuleRole checks that a module role exists in the project using
// the gen-typed ModuleSecurity reader. Returns NotFound if absent.
func validateModuleRole(ctx *ExecContext, role ast.QualifiedName) error {
	module, err := findModule(ctx, role.Module)
	if err != nil {
		return mdlerrors.NewBackend(fmt.Sprintf("module not found for role %s.%s", role.Module, role.Name), err)
	}

	ms, err := ctx.Backend.GetModuleSecurityGen(module.ID)
	if err != nil {
		return mdlerrors.NewBackend(fmt.Sprintf("read module security for %s", role.Module), err)
	}

	if ms != nil {
		for _, item := range ms.ModuleRolesItems() {
			if mr, ok := item.(*genSec.ModuleRole); ok && mr.Name() == role.Name {
				return nil
			}
		}
	}

	return mdlerrors.NewNotFound("module role", role.Module+"."+role.Name)
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

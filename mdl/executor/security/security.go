// SPDX-License-Identifier: Apache-2.0

package security

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func ExecCreateModuleRoleGenFn(ctx context.Context, s *ast.CreateModuleRoleStmt, deps *executor.HandlerDeps) error {
	return executor.ExecCreateModuleRoleGenFn(ctx, s, deps)
}

func ExecDropModuleRoleGenFn(ctx context.Context, s *ast.DropModuleRoleStmt, deps *executor.HandlerDeps) error {
	return executor.ExecDropModuleRoleGenFn(ctx, s, deps)
}

func ExecCreateUserRoleGenFn(ctx context.Context, s *ast.CreateUserRoleStmt, deps *executor.HandlerDeps) error {
	return executor.ExecCreateUserRoleGenFn(ctx, s, deps)
}

func ExecAlterUserRoleGenFn(ctx context.Context, s *ast.AlterUserRoleStmt, deps *executor.HandlerDeps) error {
	return executor.ExecAlterUserRoleGenFn(ctx, s, deps)
}

func ExecDropUserRoleGenFn(ctx context.Context, s *ast.DropUserRoleStmt, deps *executor.HandlerDeps) error {
	return executor.ExecDropUserRoleGenFn(ctx, s, deps)
}

func ExecGrantEntityAccessGenFn(ctx context.Context, s *ast.GrantEntityAccessStmt, deps *executor.HandlerDeps) error {
	return executor.ExecGrantEntityAccessGenFn(ctx, s, deps)
}

func ExecRevokeEntityAccessGenFn(ctx context.Context, s *ast.RevokeEntityAccessStmt, deps *executor.HandlerDeps) error {
	return executor.ExecRevokeEntityAccessGenFn(ctx, s, deps)
}

func ExecGrantPageAccessGenFn(ctx context.Context, s *ast.GrantPageAccessStmt, deps *executor.HandlerDeps) error {
	return executor.ExecGrantPageAccessGenFn(ctx, s, deps)
}

func ExecRevokePageAccessGenFn(ctx context.Context, s *ast.RevokePageAccessStmt, deps *executor.HandlerDeps) error {
	return executor.ExecRevokePageAccessGenFn(ctx, s, deps)
}

func ExecGrantMicroflowAccessGenFn(ctx context.Context, s *ast.GrantMicroflowAccessStmt, deps *executor.HandlerDeps) error {
	return executor.ExecGrantMicroflowAccessGenFn(ctx, s, deps)
}

func ExecRevokeMicroflowAccessGenFn(ctx context.Context, s *ast.RevokeMicroflowAccessStmt, deps *executor.HandlerDeps) error {
	return executor.ExecRevokeMicroflowAccessGenFn(ctx, s, deps)
}

func ExecGrantNanoflowAccessGenFn(ctx context.Context, s *ast.GrantNanoflowAccessStmt, deps *executor.HandlerDeps) error {
	return executor.ExecGrantNanoflowAccessGenFn(ctx, s, deps)
}

func ExecRevokeNanoflowAccessGenFn(ctx context.Context, s *ast.RevokeNanoflowAccessStmt, deps *executor.HandlerDeps) error {
	return executor.ExecRevokeNanoflowAccessGenFn(ctx, s, deps)
}

func ExecGrantODataServiceAccessGenFn(ctx context.Context, s *ast.GrantODataServiceAccessStmt, deps *executor.HandlerDeps) error {
	return executor.ExecGrantODataServiceAccessGenFn(ctx, s, deps)
}

func ExecRevokeODataServiceAccessGenFn(ctx context.Context, s *ast.RevokeODataServiceAccessStmt, deps *executor.HandlerDeps) error {
	return executor.ExecRevokeODataServiceAccessGenFn(ctx, s, deps)
}

func ExecGrantPublishedRestServiceAccessGenFn(ctx context.Context, s *ast.GrantPublishedRestServiceAccessStmt, deps *executor.HandlerDeps) error {
	return executor.ExecGrantPublishedRestServiceAccessGenFn(ctx, s, deps)
}

func ExecRevokePublishedRestServiceAccessGenFn(ctx context.Context, s *ast.RevokePublishedRestServiceAccessStmt, deps *executor.HandlerDeps) error {
	return executor.ExecRevokePublishedRestServiceAccessGenFn(ctx, s, deps)
}

func ExecAlterProjectSecurityGenFn(ctx context.Context, s *ast.AlterProjectSecurityStmt, deps *executor.HandlerDeps) error {
	return executor.ExecAlterProjectSecurityGenFn(ctx, s, deps)
}

func ExecUpdateSecurityGenFn(ctx context.Context, s *ast.UpdateSecurityStmt, deps *executor.HandlerDeps) error {
	return executor.ExecUpdateSecurityGenFn(ctx, s, deps)
}

func ExecCreateDemoUserGenFn(ctx context.Context, s *ast.CreateDemoUserStmt, deps *executor.HandlerDeps) error {
	return executor.ExecCreateDemoUserGenFn(ctx, s, deps)
}

func ExecDropDemoUserGenFn(ctx context.Context, s *ast.DropDemoUserStmt, deps *executor.HandlerDeps) error {
	return executor.ExecDropDemoUserGenFn(ctx, s, deps)
}

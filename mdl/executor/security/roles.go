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

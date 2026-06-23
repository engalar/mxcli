// SPDX-License-Identifier: Apache-2.0

package security

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

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

func ExecAlterNavigationFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecAlterNavigationFuture(ctx, stmt, deps)
}

func AlterLanguageFn(ctx context.Context, s *ast.AlterLanguageStmt, deps *executor.HandlerDeps) error {
	ectx := executor.NewExecContext(ctx, deps)
	return executor.AlterLanguage(ectx, s)
}

func ExecCreateImageCollectionFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecCreateImageCollectionFuture(ctx, stmt, deps)
}

func ExecDropImageCollectionFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecDropImageCollectionFuture(ctx, stmt, deps)
}

func ExecAlterImageCollectionFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecAlterImageCollectionFuture(ctx, stmt, deps)
}

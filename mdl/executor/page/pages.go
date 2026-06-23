// SPDX-License-Identifier: Apache-2.0

package page

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func ExecCreatePageV3Fn(ctx context.Context, s *ast.CreatePageStmtV3, deps *executor.HandlerDeps) error {
	return executor.ExecCreatePageV3Fn(ctx, s, deps)
}

func ExecDropPageFn(ctx context.Context, s *ast.DropPageStmt, deps *executor.HandlerDeps) error {
	ectx := executor.NewExecContext(ctx, deps)
	return executor.ExecDropPage(ectx, s)
}

func ExecCreateSnippetV3Fn(ctx context.Context, s *ast.CreateSnippetStmtV3, deps *executor.HandlerDeps) error {
	return executor.ExecCreateSnippetV3Fn(ctx, s, deps)
}

func ExecDropSnippetFn(ctx context.Context, s *ast.DropSnippetStmt, deps *executor.HandlerDeps) error {
	ectx := executor.NewExecContext(ctx, deps)
	return executor.ExecDropSnippet(ectx, s)
}

func ExecCreateOrModifyLayoutFn(ctx context.Context, s *ast.CreateLayoutStmt, deps *executor.HandlerDeps) error {
	return executor.ExecCreateOrModifyLayoutFn(ctx, s, deps)
}

func ExecAlterPageFn(ctx context.Context, s *ast.AlterPageStmt, deps *executor.HandlerDeps) error {
	return executor.ExecAlterPageFn(ctx, s, deps)
}

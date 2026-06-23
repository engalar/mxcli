// SPDX-License-Identifier: Apache-2.0

package page

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func RegisterHandlers(r *executor.Registry, deps *executor.HandlerDeps) {
	r.RegisterFuture("CreatePageStmtV3", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreatePageV3Fn(ctx, stmt.(*ast.CreatePageStmtV3), deps)
	})
	r.RegisterFuture("DropPage", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropPageFn(ctx, stmt.(*ast.DropPageStmt), deps)
	})
	r.RegisterFuture("CreateSnippetStmtV3", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateSnippetV3Fn(ctx, stmt.(*ast.CreateSnippetStmtV3), deps)
	})
	r.RegisterFuture("DropSnippet", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropSnippetFn(ctx, stmt.(*ast.DropSnippetStmt), deps)
	})
	r.RegisterFuture("CreateLayout", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateOrModifyLayoutFn(ctx, stmt.(*ast.CreateLayoutStmt), deps)
	})
	r.RegisterFuture("AlterPage", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterPageFn(ctx, stmt.(*ast.AlterPageStmt), deps)
	})
}

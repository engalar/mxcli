package page

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func RegisterHandlers(r *executor.Registry, deps *executor.HandlerDeps) {
	r.RegisterFuture("CreatePageStmtV3", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreatePageV3Fn(ctx, stmt.(*ast.CreatePageStmtV3), deps)
	})
	r.RegisterFuture("DropPage", func(ctx context.Context, stmt ast.Statement) error {
		ectx := executor.NewExecContext(ctx, deps)
		return executor.ExecDropPage(ectx, stmt.(*ast.DropPageStmt))
	})
	r.RegisterFuture("CreateSnippetStmtV3", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateSnippetV3Fn(ctx, stmt.(*ast.CreateSnippetStmtV3), deps)
	})
	r.RegisterFuture("DropSnippet", func(ctx context.Context, stmt ast.Statement) error {
		ectx := executor.NewExecContext(ctx, deps)
		return executor.ExecDropSnippet(ectx, stmt.(*ast.DropSnippetStmt))
	})
	r.RegisterFuture("CreateLayout", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateOrModifyLayoutFn(ctx, stmt.(*ast.CreateLayoutStmt), deps)
	})
	r.RegisterFuture("AlterPage", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecAlterPageFn(ctx, stmt.(*ast.AlterPageStmt), deps)
	})
}

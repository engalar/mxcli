package domainmodel

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func ExecCreateConstantFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	ectx := executor.NewExecContext(ctx, deps)
	return executor.ExecCreateConstant(ectx, stmt.(*ast.CreateConstantStmt))
}

func ExecDropConstantFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	ectx := executor.NewExecContext(ctx, deps)
	return executor.ExecDropConstant(ectx, stmt.(*ast.DropConstantStmt))
}

func ExecCreateDatabaseConnectionFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	ectx := executor.NewExecContext(ctx, deps)
	return executor.ExecCreateDatabaseConnection(ectx, stmt.(*ast.CreateDatabaseConnectionStmt))
}

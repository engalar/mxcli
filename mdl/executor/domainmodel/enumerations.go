package domainmodel

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

func ExecCreateEnumerationFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	ectx := executor.NewExecContext(ctx, deps)
	return executor.ExecCreateEnumeration(ectx, stmt.(*ast.CreateEnumerationStmt))
}

func ExecAlterEnumerationFuture(ctx context.Context, deps *executor.HandlerDeps) error {
	return mdlerrors.NewUnsupported("alter enumeration not yet implemented")
}

func ExecDropEnumerationFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	ectx := executor.NewExecContext(ctx, deps)
	return executor.ExecDropEnumeration(ectx, stmt.(*ast.DropEnumerationStmt))
}

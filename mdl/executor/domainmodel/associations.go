package domainmodel

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func ExecCreateAssociationFn(ctx context.Context, s *ast.CreateAssociationStmt, deps *executor.HandlerDeps) error {
	ectx := executor.NewExecContext(ctx, deps)
	return executor.ExecCreateAssociation(ectx, s)
}

func ExecAlterAssociationFn(ctx context.Context, s *ast.AlterAssociationStmt, deps *executor.HandlerDeps) error {
	ectx := executor.NewExecContext(ctx, deps)
	return executor.ExecAlterAssociationGen(ectx, s)
}

func ExecDropAssociationFn(ctx context.Context, s *ast.DropAssociationStmt, deps *executor.HandlerDeps) error {
	ectx := executor.NewExecContext(ctx, deps)
	return executor.ExecDropAssociationGen(ectx, s)
}

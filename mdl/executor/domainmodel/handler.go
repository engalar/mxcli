package domainmodel

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func RegisterHandlers(r *executor.Registry, deps *executor.HandlerDeps) {
	r.RegisterFuture("CreateEntity", func(ctx context.Context, stmt ast.Statement) error {
		ectx := executor.NewExecContext(ctx, deps)
		return executor.ExecCreateEntity(ectx, stmt.(*ast.CreateEntityStmt))
	})
	r.RegisterFuture("AlterEntity", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecAlterEntityGenFn(ctx, stmt.(*ast.AlterEntityStmt), deps)
	})
	r.RegisterFuture("DropEntity", func(ctx context.Context, stmt ast.Statement) error {
		ectx := executor.NewExecContext(ctx, deps)
		return executor.ExecDropEntity(ectx, stmt.(*ast.DropEntityStmt))
	})
	r.RegisterFuture("CreateViewEntity", func(ctx context.Context, stmt ast.Statement) error {
		ectx := executor.NewExecContext(ctx, deps)
		return executor.ExecCreateViewEntity(ectx, stmt.(*ast.CreateViewEntityStmt))
	})
	r.RegisterFuture("CreateAssociation", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateAssociationFn(ctx, stmt.(*ast.CreateAssociationStmt), deps)
	})
	r.RegisterFuture("AlterAssociation", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecAlterAssociationFn(ctx, stmt.(*ast.AlterAssociationStmt), deps)
	})
	r.RegisterFuture("DropAssociation", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropAssociationFn(ctx, stmt.(*ast.DropAssociationStmt), deps)
	})
	r.RegisterFuture("CreateEnumeration", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateEnumerationFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("AlterEnumeration", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecAlterEnumerationFuture(ctx, deps)
	})
	r.RegisterFuture("DropEnumeration", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropEnumerationFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("CreateConstant", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateConstantFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropConstant", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropConstantFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("CreateDatabaseConnection", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateDatabaseConnectionFuture(ctx, stmt, deps)
	})
}

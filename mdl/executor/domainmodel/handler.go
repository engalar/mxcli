package domainmodel

import (
	"context"
	"fmt"

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
		ectx := executor.NewExecContext(ctx, deps)
		return executor.ExecCreateAssociation(ectx, stmt.(*ast.CreateAssociationStmt))
	})
	r.RegisterFuture("AlterAssociation", func(ctx context.Context, stmt ast.Statement) error {
		ectx := executor.NewExecContext(ctx, deps)
		return executor.ExecAlterAssociation(ectx, stmt.(*ast.AlterAssociationStmt))
	})
	r.RegisterFuture("DropAssociation", func(ctx context.Context, stmt ast.Statement) error {
		ectx := executor.NewExecContext(ctx, deps)
		return executor.ExecDropAssociation(ectx, stmt.(*ast.DropAssociationStmt))
	})
	r.RegisterFuture("CreateEnumeration", func(ctx context.Context, stmt ast.Statement) error {
		ectx := executor.NewExecContext(ctx, deps)
		return executor.ExecCreateEnumeration(ectx, stmt.(*ast.CreateEnumerationStmt))
	})
	r.RegisterFuture("AlterEnumeration", func(ctx context.Context, stmt ast.Statement) error {
		return fmt.Errorf("alter enumeration not yet implemented")
	})
	r.RegisterFuture("DropEnumeration", func(ctx context.Context, stmt ast.Statement) error {
		ectx := executor.NewExecContext(ctx, deps)
		return executor.ExecDropEnumeration(ectx, stmt.(*ast.DropEnumerationStmt))
	})
	r.RegisterFuture("CreateConstant", func(ctx context.Context, stmt ast.Statement) error {
		ectx := executor.NewExecContext(ctx, deps)
		return executor.ExecCreateConstant(ectx, stmt.(*ast.CreateConstantStmt))
	})
	r.RegisterFuture("DropConstant", func(ctx context.Context, stmt ast.Statement) error {
		ectx := executor.NewExecContext(ctx, deps)
		return executor.ExecDropConstant(ectx, stmt.(*ast.DropConstantStmt))
	})
	r.RegisterFuture("CreateDatabaseConnection", func(ctx context.Context, stmt ast.Statement) error {
		ectx := executor.NewExecContext(ctx, deps)
		return executor.ExecCreateDatabaseConnection(ectx, stmt.(*ast.CreateDatabaseConnectionStmt))
	})
}

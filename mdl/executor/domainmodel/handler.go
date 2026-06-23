package domainmodel

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func RegisterHandlers(r *executor.Registry, deps *executor.HandlerDeps) {
	r.RegisterFuture("CreateEntity", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateEntityFn(ctx, stmt.(*ast.CreateEntityStmt), deps)
	})
	r.RegisterFuture("AlterEntity", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterEntityFn(ctx, stmt.(*ast.AlterEntityStmt), deps)
	})
	r.RegisterFuture("DropEntity", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropEntityFn(ctx, stmt.(*ast.DropEntityStmt), deps)
	})
	r.RegisterFuture("CreateViewEntity", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateViewEntityFn(ctx, stmt.(*ast.CreateViewEntityStmt), deps)
	})
	r.RegisterFuture("CreateAssociation", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateAssociationFn(ctx, stmt.(*ast.CreateAssociationStmt), deps)
	})
	r.RegisterFuture("AlterAssociation", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterAssociationFn(ctx, stmt.(*ast.AlterAssociationStmt), deps)
	})
	r.RegisterFuture("DropAssociation", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropAssociationFn(ctx, stmt.(*ast.DropAssociationStmt), deps)
	})
	r.RegisterFuture("CreateEnumeration", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateEnumerationFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("AlterEnumeration", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterEnumerationFuture(ctx, deps)
	})
	r.RegisterFuture("DropEnumeration", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropEnumerationFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("CreateConstant", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateConstantFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropConstant", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropConstantFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("CreateDatabaseConnection", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateDatabaseConnectionFuture(ctx, stmt, deps)
	})
}

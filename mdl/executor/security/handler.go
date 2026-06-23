package security

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func RegisterHandlers(r *executor.Registry, deps *executor.HandlerDeps) {
	r.RegisterFuture("CreateModuleRole", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateModuleRoleGenFn(ctx, stmt.(*ast.CreateModuleRoleStmt), deps)
	})
	r.RegisterFuture("DropModuleRole", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropModuleRoleGenFn(ctx, stmt.(*ast.DropModuleRoleStmt), deps)
	})
	r.RegisterFuture("CreateUserRole", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateUserRoleGenFn(ctx, stmt.(*ast.CreateUserRoleStmt), deps)
	})
	r.RegisterFuture("AlterUserRole", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecAlterUserRoleGenFn(ctx, stmt.(*ast.AlterUserRoleStmt), deps)
	})
	r.RegisterFuture("DropUserRole", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropUserRoleGenFn(ctx, stmt.(*ast.DropUserRoleStmt), deps)
	})
	r.RegisterFuture("GrantEntityAccess", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecGrantEntityAccessGenFn(ctx, stmt.(*ast.GrantEntityAccessStmt), deps)
	})
	r.RegisterFuture("RevokeEntityAccess", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecRevokeEntityAccessGenFn(ctx, stmt.(*ast.RevokeEntityAccessStmt), deps)
	})
	r.RegisterFuture("GrantPageAccess", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecGrantPageAccessGenFn(ctx, stmt.(*ast.GrantPageAccessStmt), deps)
	})
	r.RegisterFuture("RevokePageAccess", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecRevokePageAccessGenFn(ctx, stmt.(*ast.RevokePageAccessStmt), deps)
	})
	r.RegisterFuture("GrantMicroflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecGrantMicroflowAccessGenFn(ctx, stmt.(*ast.GrantMicroflowAccessStmt), deps)
	})
	r.RegisterFuture("RevokeMicroflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecRevokeMicroflowAccessGenFn(ctx, stmt.(*ast.RevokeMicroflowAccessStmt), deps)
	})
	r.RegisterFuture("GrantNanoflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecGrantNanoflowAccessGenFn(ctx, stmt.(*ast.GrantNanoflowAccessStmt), deps)
	})
	r.RegisterFuture("RevokeNanoflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecRevokeNanoflowAccessGenFn(ctx, stmt.(*ast.RevokeNanoflowAccessStmt), deps)
	})
	r.RegisterFuture("GrantWorkflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		ectx := executor.NewExecContext(ctx, deps)
		return executor.ExecGrantWorkflowAccess(ectx, stmt.(*ast.GrantWorkflowAccessStmt))
	})
	r.RegisterFuture("RevokeWorkflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		ectx := executor.NewExecContext(ctx, deps)
		return executor.ExecRevokeWorkflowAccess(ectx, stmt.(*ast.RevokeWorkflowAccessStmt))
	})
	r.RegisterFuture("GrantODataServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecGrantODataServiceAccessGenFn(ctx, stmt.(*ast.GrantODataServiceAccessStmt), deps)
	})
	r.RegisterFuture("RevokeODataServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecRevokeODataServiceAccessGenFn(ctx, stmt.(*ast.RevokeODataServiceAccessStmt), deps)
	})
	r.RegisterFuture("GrantPublishedRestServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecGrantPublishedRestServiceAccessGenFn(ctx, stmt.(*ast.GrantPublishedRestServiceAccessStmt), deps)
	})
	r.RegisterFuture("RevokePublishedRestServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecRevokePublishedRestServiceAccessGenFn(ctx, stmt.(*ast.RevokePublishedRestServiceAccessStmt), deps)
	})
	r.RegisterFuture("AlterProjectSecurity", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecAlterProjectSecurityGenFn(ctx, stmt.(*ast.AlterProjectSecurityStmt), deps)
	})
	r.RegisterFuture("UpdateSecurity", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecUpdateSecurityGenFn(ctx, stmt.(*ast.UpdateSecurityStmt), deps)
	})
	r.RegisterFuture("CreateDemoUser", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateDemoUserGenFn(ctx, stmt.(*ast.CreateDemoUserStmt), deps)
	})
	r.RegisterFuture("DropDemoUser", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropDemoUserGenFn(ctx, stmt.(*ast.DropDemoUserStmt), deps)
	})
	r.RegisterFuture("AlterNavigation", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecAlterNavigationFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("AlterLanguage", func(ctx context.Context, stmt ast.Statement) error {
		ectx := executor.NewExecContext(ctx, deps)
		return executor.AlterLanguage(ectx, stmt.(*ast.AlterLanguageStmt))
	})
	r.RegisterFuture("CreateImageCollection", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateImageCollectionFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropImageCollection", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropImageCollectionFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("AlterImageCollection", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecAlterImageCollectionFuture(ctx, stmt, deps)
	})
}

// SPDX-License-Identifier: Apache-2.0

package security

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func RegisterHandlers(r *executor.Registry, deps *executor.HandlerDeps) {
	r.RegisterFuture("CreateModuleRole", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateModuleRoleGenFn(ctx, stmt.(*ast.CreateModuleRoleStmt), deps)
	})
	r.RegisterFuture("DropModuleRole", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropModuleRoleGenFn(ctx, stmt.(*ast.DropModuleRoleStmt), deps)
	})
	r.RegisterFuture("CreateUserRole", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateUserRoleGenFn(ctx, stmt.(*ast.CreateUserRoleStmt), deps)
	})
	r.RegisterFuture("AlterUserRole", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterUserRoleGenFn(ctx, stmt.(*ast.AlterUserRoleStmt), deps)
	})
	r.RegisterFuture("DropUserRole", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropUserRoleGenFn(ctx, stmt.(*ast.DropUserRoleStmt), deps)
	})
	r.RegisterFuture("GrantEntityAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecGrantEntityAccessGenFn(ctx, stmt.(*ast.GrantEntityAccessStmt), deps)
	})
	r.RegisterFuture("RevokeEntityAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRevokeEntityAccessGenFn(ctx, stmt.(*ast.RevokeEntityAccessStmt), deps)
	})
	r.RegisterFuture("GrantPageAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecGrantPageAccessGenFn(ctx, stmt.(*ast.GrantPageAccessStmt), deps)
	})
	r.RegisterFuture("RevokePageAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRevokePageAccessGenFn(ctx, stmt.(*ast.RevokePageAccessStmt), deps)
	})
	r.RegisterFuture("GrantMicroflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecGrantMicroflowAccessGenFn(ctx, stmt.(*ast.GrantMicroflowAccessStmt), deps)
	})
	r.RegisterFuture("RevokeMicroflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRevokeMicroflowAccessGenFn(ctx, stmt.(*ast.RevokeMicroflowAccessStmt), deps)
	})
	r.RegisterFuture("GrantNanoflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecGrantNanoflowAccessGenFn(ctx, stmt.(*ast.GrantNanoflowAccessStmt), deps)
	})
	r.RegisterFuture("RevokeNanoflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRevokeNanoflowAccessGenFn(ctx, stmt.(*ast.RevokeNanoflowAccessStmt), deps)
	})
	r.RegisterFuture("GrantWorkflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecGrantWorkflowAccessFn(ctx, stmt.(*ast.GrantWorkflowAccessStmt), deps)
	})
	r.RegisterFuture("RevokeWorkflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRevokeWorkflowAccessFn(ctx, stmt.(*ast.RevokeWorkflowAccessStmt), deps)
	})
	r.RegisterFuture("GrantODataServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecGrantODataServiceAccessGenFn(ctx, stmt.(*ast.GrantODataServiceAccessStmt), deps)
	})
	r.RegisterFuture("RevokeODataServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRevokeODataServiceAccessGenFn(ctx, stmt.(*ast.RevokeODataServiceAccessStmt), deps)
	})
	r.RegisterFuture("GrantPublishedRestServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecGrantPublishedRestServiceAccessGenFn(ctx, stmt.(*ast.GrantPublishedRestServiceAccessStmt), deps)
	})
	r.RegisterFuture("RevokePublishedRestServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRevokePublishedRestServiceAccessGenFn(ctx, stmt.(*ast.RevokePublishedRestServiceAccessStmt), deps)
	})
	r.RegisterFuture("AlterProjectSecurity", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterProjectSecurityGenFn(ctx, stmt.(*ast.AlterProjectSecurityStmt), deps)
	})
	r.RegisterFuture("UpdateSecurity", func(ctx context.Context, stmt ast.Statement) error {
		return ExecUpdateSecurityGenFn(ctx, stmt.(*ast.UpdateSecurityStmt), deps)
	})
	r.RegisterFuture("CreateDemoUser", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateDemoUserGenFn(ctx, stmt.(*ast.CreateDemoUserStmt), deps)
	})
	r.RegisterFuture("DropDemoUser", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropDemoUserGenFn(ctx, stmt.(*ast.DropDemoUserStmt), deps)
	})
	r.RegisterFuture("AlterNavigation", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterNavigationFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("AlterLanguage", func(ctx context.Context, stmt ast.Statement) error {
		return AlterLanguageFn(ctx, stmt.(*ast.AlterLanguageStmt), deps)
	})
	r.RegisterFuture("CreateImageCollection", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateImageCollectionFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropImageCollection", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropImageCollectionFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("AlterImageCollection", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterImageCollectionFuture(ctx, stmt, deps)
	})
}

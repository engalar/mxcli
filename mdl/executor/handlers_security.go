// SPDX-License-Identifier: Apache-2.0

package executor

import "github.com/mendixlabs/mxcli/mdl/ast"

func registerSecurityHandlers(r *Registry) {
	r.Register(&ast.CreateModuleRoleStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateModuleRoleGenFn(ctx, stmt.(*ast.CreateModuleRoleStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.DropModuleRoleStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropModuleRoleGenFn(ctx, stmt.(*ast.DropModuleRoleStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.CreateUserRoleStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateUserRoleGenFn(ctx, stmt.(*ast.CreateUserRoleStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.AlterUserRoleStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execAlterUserRoleGenFn(ctx, stmt.(*ast.AlterUserRoleStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.DropUserRoleStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropUserRoleGenFn(ctx, stmt.(*ast.DropUserRoleStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.GrantEntityAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execGrantEntityAccessGenFn(ctx, stmt.(*ast.GrantEntityAccessStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.RevokeEntityAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execRevokeEntityAccessGenFn(ctx, stmt.(*ast.RevokeEntityAccessStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.GrantMicroflowAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execGrantMicroflowAccessGenFn(ctx, stmt.(*ast.GrantMicroflowAccessStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.RevokeMicroflowAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execRevokeMicroflowAccessGenFn(ctx, stmt.(*ast.RevokeMicroflowAccessStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.GrantNanoflowAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execGrantNanoflowAccessGenFn(ctx, stmt.(*ast.GrantNanoflowAccessStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.RevokeNanoflowAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execRevokeNanoflowAccessGenFn(ctx, stmt.(*ast.RevokeNanoflowAccessStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.GrantPageAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execGrantPageAccessGenFn(ctx, stmt.(*ast.GrantPageAccessStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.RevokePageAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execRevokePageAccessGenFn(ctx, stmt.(*ast.RevokePageAccessStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.GrantWorkflowAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execGrantWorkflowAccess(ctx, stmt.(*ast.GrantWorkflowAccessStmt))
	})
	r.Register(&ast.RevokeWorkflowAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execRevokeWorkflowAccess(ctx, stmt.(*ast.RevokeWorkflowAccessStmt))
	})
	r.Register(&ast.AlterProjectSecurityStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execAlterProjectSecurityGenFn(ctx, stmt.(*ast.AlterProjectSecurityStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.CreateDemoUserStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateDemoUserGenFn(ctx, stmt.(*ast.CreateDemoUserStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.DropDemoUserStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropDemoUserGenFn(ctx, stmt.(*ast.DropDemoUserStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.UpdateSecurityStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execUpdateSecurityGenFn(ctx, stmt.(*ast.UpdateSecurityStmt), execContextToDeps(ctx))
	})
}

func registerNavigationHandlers(r *Registry) {
	r.Register(&ast.AlterNavigationStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execAlterNavigationFn(ctx, stmt.(*ast.AlterNavigationStmt), execContextToDeps(ctx))
	})
}

func registerImageHandlers(r *Registry) {
	r.Register(&ast.CreateImageCollectionStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateImageCollectionFn(ctx, stmt.(*ast.CreateImageCollectionStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.DropImageCollectionStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropImageCollectionFn(ctx, stmt.(*ast.DropImageCollectionStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.AlterImageCollectionStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execAlterImageCollectionFn(ctx, stmt.(*ast.AlterImageCollectionStmt), execContextToDeps(ctx))
	})
}

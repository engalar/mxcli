// SPDX-License-Identifier: Apache-2.0

package executor

import "github.com/mendixlabs/mxcli/mdl/ast"

func registerSecurityHandlers(r *Registry) {
	r.Register(&ast.CreateModuleRoleStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateModuleRoleGen(ctx, stmt.(*ast.CreateModuleRoleStmt))
	})
	r.Register(&ast.DropModuleRoleStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropModuleRoleGen(ctx, stmt.(*ast.DropModuleRoleStmt))
	})
	r.Register(&ast.CreateUserRoleStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateUserRoleGen(ctx, stmt.(*ast.CreateUserRoleStmt))
	})
	r.Register(&ast.AlterUserRoleStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execAlterUserRoleGen(ctx, stmt.(*ast.AlterUserRoleStmt))
	})
	r.Register(&ast.DropUserRoleStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropUserRoleGen(ctx, stmt.(*ast.DropUserRoleStmt))
	})
	r.Register(&ast.GrantEntityAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execGrantEntityAccessGen(ctx, stmt.(*ast.GrantEntityAccessStmt))
	})
	r.Register(&ast.RevokeEntityAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execRevokeEntityAccessGen(ctx, stmt.(*ast.RevokeEntityAccessStmt))
	})
	r.Register(&ast.GrantMicroflowAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execGrantMicroflowAccessGen(ctx, stmt.(*ast.GrantMicroflowAccessStmt))
	})
	r.Register(&ast.RevokeMicroflowAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execRevokeMicroflowAccessGen(ctx, stmt.(*ast.RevokeMicroflowAccessStmt))
	})
	r.Register(&ast.GrantNanoflowAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execGrantNanoflowAccessGen(ctx, stmt.(*ast.GrantNanoflowAccessStmt))
	})
	r.Register(&ast.RevokeNanoflowAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execRevokeNanoflowAccessGen(ctx, stmt.(*ast.RevokeNanoflowAccessStmt))
	})
	r.Register(&ast.GrantPageAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execGrantPageAccessGen(ctx, stmt.(*ast.GrantPageAccessStmt))
	})
	r.Register(&ast.RevokePageAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execRevokePageAccessGen(ctx, stmt.(*ast.RevokePageAccessStmt))
	})
	r.Register(&ast.GrantWorkflowAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execGrantWorkflowAccess(ctx, stmt.(*ast.GrantWorkflowAccessStmt))
	})
	r.Register(&ast.RevokeWorkflowAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execRevokeWorkflowAccess(ctx, stmt.(*ast.RevokeWorkflowAccessStmt))
	})
	r.Register(&ast.AlterProjectSecurityStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execAlterProjectSecurityGen(ctx, stmt.(*ast.AlterProjectSecurityStmt))
	})
	r.Register(&ast.CreateDemoUserStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateDemoUserGen(ctx, stmt.(*ast.CreateDemoUserStmt))
	})
	r.Register(&ast.DropDemoUserStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropDemoUserGen(ctx, stmt.(*ast.DropDemoUserStmt))
	})
	r.Register(&ast.UpdateSecurityStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execUpdateSecurityGen(ctx, stmt.(*ast.UpdateSecurityStmt))
	})
}

func registerNavigationHandlers(r *Registry) {
	r.Register(&ast.AlterNavigationStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execAlterNavigation(ctx, stmt.(*ast.AlterNavigationStmt))
	})
}

func registerImageHandlers(r *Registry) {
	r.Register(&ast.CreateImageCollectionStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateImageCollection(ctx, stmt.(*ast.CreateImageCollectionStmt))
	})
	r.Register(&ast.DropImageCollectionStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropImageCollection(ctx, stmt.(*ast.DropImageCollectionStmt))
	})
	r.Register(&ast.AlterImageCollectionStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execAlterImageCollection(ctx, stmt.(*ast.AlterImageCollectionStmt))
	})
}

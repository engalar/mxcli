// SPDX-License-Identifier: Apache-2.0

package executor

import "github.com/mendixlabs/mxcli/mdl/ast"

func registerModuleHandlers(r *Registry) {
	r.Register(&ast.CreateModuleStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateModule(ctx, stmt.(*ast.CreateModuleStmt))
	})
	r.Register(&ast.DropModuleStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropModule(ctx, stmt.(*ast.DropModuleStmt))
	})
	r.Register(&ast.AlterModuleJarDepStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execAlterModuleJarDep(ctx, stmt.(*ast.AlterModuleJarDepStmt))
	})
}

func registerEnumerationHandlers(r *Registry) {
	r.Register(&ast.CreateEnumerationStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateEnumeration(ctx, stmt.(*ast.CreateEnumerationStmt))
	})
	r.Register(&ast.AlterEnumerationStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execAlterEnumeration(ctx, stmt.(*ast.AlterEnumerationStmt))
	})
	r.Register(&ast.DropEnumerationStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropEnumeration(ctx, stmt.(*ast.DropEnumerationStmt))
	})
}

func registerConstantHandlers(r *Registry) {
	r.Register(&ast.CreateConstantStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return createConstant(ctx, stmt.(*ast.CreateConstantStmt))
	})
	r.Register(&ast.DropConstantStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return dropConstant(ctx, stmt.(*ast.DropConstantStmt))
	})
}

// SPDX-License-Identifier: Apache-2.0

package executor

import "github.com/mendixlabs/mxcli/mdl/ast"

func registerEntityHandlers(r *Registry) {
	r.Register(&ast.CreateEntityStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		if canExecCreateEntityGen(ctx, stmt.(*ast.CreateEntityStmt)) {
			return execCreateEntityGen(ctx, stmt.(*ast.CreateEntityStmt))
		}
		return execCreateEntity(ctx, stmt.(*ast.CreateEntityStmt))
	})
	r.Register(&ast.CreateViewEntityStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateViewEntity(ctx, stmt.(*ast.CreateViewEntityStmt))
	})
	r.Register(&ast.AlterEntityStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execAlterEntity(ctx, stmt.(*ast.AlterEntityStmt))
	})
	r.Register(&ast.DropEntityStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		if ctx.DomainModels != nil {
			return execDropEntityGen(ctx, stmt.(*ast.DropEntityStmt))
		}
		return execDropEntity(ctx, stmt.(*ast.DropEntityStmt))
	})
}

func registerAssociationHandlers(r *Registry) {
	r.Register(&ast.CreateAssociationStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		if ctx.DomainModels != nil {
			return execCreateAssociationGen(ctx, stmt.(*ast.CreateAssociationStmt))
		}
		return execCreateAssociation(ctx, stmt.(*ast.CreateAssociationStmt))
	})
	r.Register(&ast.AlterAssociationStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		if ctx.DomainModels != nil {
			return execAlterAssociationGen(ctx, stmt.(*ast.AlterAssociationStmt))
		}
		return execAlterAssociation(ctx, stmt.(*ast.AlterAssociationStmt))
	})
	r.Register(&ast.DropAssociationStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		if ctx.DomainModels != nil {
			return execDropAssociationGen(ctx, stmt.(*ast.DropAssociationStmt))
		}
		return execDropAssociation(ctx, stmt.(*ast.DropAssociationStmt))
	})
}

func registerDatabaseConnectionHandlers(r *Registry) {
	r.Register(&ast.CreateDatabaseConnectionStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return createDatabaseConnection(ctx, stmt.(*ast.CreateDatabaseConnectionStmt))
	})
}

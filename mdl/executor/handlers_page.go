// SPDX-License-Identifier: Apache-2.0

package executor

import "github.com/mendixlabs/mxcli/mdl/ast"

func registerPageHandlers(r *Registry) {
	r.Register(&ast.CreatePageStmtV3{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreatePageV3(ctx, stmt.(*ast.CreatePageStmtV3))
	})
	r.Register(&ast.DropPageStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropPage(ctx, stmt.(*ast.DropPageStmt))
	})
	r.Register(&ast.CreateSnippetStmtV3{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateSnippetV3(ctx, stmt.(*ast.CreateSnippetStmtV3))
	})
	r.Register(&ast.CreateLayoutStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateOrModifyLayoutFn(ctx, stmt.(*ast.CreateLayoutStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.DropSnippetStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropSnippet(ctx, stmt.(*ast.DropSnippetStmt))
	})
	r.Register(&ast.DropJavaActionStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropJavaActionGenFn(ctx, stmt.(*ast.DropJavaActionStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.CreateJavaActionStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateJavaActionGenFn(ctx, stmt.(*ast.CreateJavaActionStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.CreateJavaScriptActionStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateJavaScriptActionFn(ctx, stmt.(*ast.CreateJavaScriptActionStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.DropFolderStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropFolderFn(ctx, stmt.(*ast.DropFolderStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.MoveFolderStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execMoveFolderFn(ctx, stmt.(*ast.MoveFolderStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.MoveStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execMoveFn(ctx, stmt.(*ast.MoveStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.RenameStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execRenameFn(ctx, stmt.(*ast.RenameStmt), execContextToDeps(ctx))
	})
}

func registerAlterPageHandlers(r *Registry) {
	r.Register(&ast.AlterPageStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execAlterPage(ctx, stmt.(*ast.AlterPageStmt))
	})
}

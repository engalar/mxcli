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
		return execCreateOrModifyLayout(ctx, stmt.(*ast.CreateLayoutStmt))
	})
	r.Register(&ast.DropSnippetStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropSnippet(ctx, stmt.(*ast.DropSnippetStmt))
	})
	r.Register(&ast.DropJavaActionStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropJavaActionGen(ctx, stmt.(*ast.DropJavaActionStmt))
	})
	r.Register(&ast.CreateJavaActionStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateJavaActionGen(ctx, stmt.(*ast.CreateJavaActionStmt))
	})
	r.Register(&ast.CreateJavaScriptActionStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateJavaScriptAction(ctx, stmt.(*ast.CreateJavaScriptActionStmt))
	})
	r.Register(&ast.DropFolderStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropFolder(ctx, stmt.(*ast.DropFolderStmt))
	})
	r.Register(&ast.MoveFolderStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execMoveFolder(ctx, stmt.(*ast.MoveFolderStmt))
	})
	r.Register(&ast.MoveStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execMove(ctx, stmt.(*ast.MoveStmt))
	})
	r.Register(&ast.RenameStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execRename(ctx, stmt.(*ast.RenameStmt))
	})
}

func registerAlterPageHandlers(r *Registry) {
	r.Register(&ast.AlterPageStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execAlterPage(ctx, stmt.(*ast.AlterPageStmt))
	})
}

// SPDX-License-Identifier: Apache-2.0

package executor

import "github.com/mendixlabs/mxcli/mdl/ast"

func registerMicroflowAndNanoflowHandlers(r *Registry) {
	r.Register(&ast.CreateMicroflowStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateMicroflowGenFn(ctx, stmt.(*ast.CreateMicroflowStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.DropMicroflowStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropMicroflowFn(ctx, stmt.(*ast.DropMicroflowStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.CreateNanoflowStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateNanoflowGenFn(ctx, stmt.(*ast.CreateNanoflowStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.DropNanoflowStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropNanoflowGenFn(ctx, stmt.(*ast.DropNanoflowStmt), execContextToDeps(ctx))
	})
}

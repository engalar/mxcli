// SPDX-License-Identifier: Apache-2.0

package executor

import "github.com/mendixlabs/mxcli/mdl/ast"

func registerMicroflowAndNanoflowHandlers(r *Registry) {
	r.Register(&ast.CreateMicroflowStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateMicroflowGen(ctx, stmt.(*ast.CreateMicroflowStmt))
	})
	r.Register(&ast.DropMicroflowStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropMicroflow(ctx, stmt.(*ast.DropMicroflowStmt))
	})
	r.Register(&ast.CreateNanoflowStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateNanoflowGen(ctx, stmt.(*ast.CreateNanoflowStmt))
	})
	r.Register(&ast.DropNanoflowStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropNanoflowGen(ctx, stmt.(*ast.DropNanoflowStmt))
	})
}

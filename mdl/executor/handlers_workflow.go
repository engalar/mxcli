// SPDX-License-Identifier: Apache-2.0

package executor

import "github.com/mendixlabs/mxcli/mdl/ast"

func registerWorkflowHandlers(r *Registry) {
	r.Register(&ast.CreateWorkflowStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateWorkflowGenFn(ctx, stmt.(*ast.CreateWorkflowStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.DropWorkflowStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropWorkflowGenFn(ctx, stmt.(*ast.DropWorkflowStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.AlterWorkflowStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execAlterWorkflow(ctx, stmt.(*ast.AlterWorkflowStmt))
	})
}

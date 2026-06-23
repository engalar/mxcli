// SPDX-License-Identifier: Apache-2.0

package executor

import "github.com/mendixlabs/mxcli/mdl/ast"

func registerAgentEditorHandlers(r *Registry) {
	r.Register(&ast.CreateModelStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateAgentEditorModel(ctx, stmt.(*ast.CreateModelStmt))
	})
	r.Register(&ast.DropModelStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropAgentEditorModel(ctx, stmt.(*ast.DropModelStmt))
	})
	r.Register(&ast.CreateConsumedMCPServiceStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateConsumedMCPServiceFn(ctx, stmt.(*ast.CreateConsumedMCPServiceStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.DropConsumedMCPServiceStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropConsumedMCPServiceFn(ctx, stmt.(*ast.DropConsumedMCPServiceStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.CreateKnowledgeBaseStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateKnowledgeBaseFn(ctx, stmt.(*ast.CreateKnowledgeBaseStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.DropKnowledgeBaseStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropKnowledgeBaseFn(ctx, stmt.(*ast.DropKnowledgeBaseStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.CreateAgentStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateAgentFn(ctx, stmt.(*ast.CreateAgentStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.DropAgentStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropAgentFn(ctx, stmt.(*ast.DropAgentStmt), execContextToDeps(ctx))
	})
}

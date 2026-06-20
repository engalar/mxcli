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
		return execCreateConsumedMCPService(ctx, stmt.(*ast.CreateConsumedMCPServiceStmt))
	})
	r.Register(&ast.DropConsumedMCPServiceStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropConsumedMCPService(ctx, stmt.(*ast.DropConsumedMCPServiceStmt))
	})
	r.Register(&ast.CreateKnowledgeBaseStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateKnowledgeBase(ctx, stmt.(*ast.CreateKnowledgeBaseStmt))
	})
	r.Register(&ast.DropKnowledgeBaseStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropKnowledgeBase(ctx, stmt.(*ast.DropKnowledgeBaseStmt))
	})
	r.Register(&ast.CreateAgentStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateAgent(ctx, stmt.(*ast.CreateAgentStmt))
	})
	r.Register(&ast.DropAgentStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropAgent(ctx, stmt.(*ast.DropAgentStmt))
	})
}

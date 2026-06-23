// SPDX-License-Identifier: Apache-2.0

package workflow

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func RegisterHandlers(r *executor.Registry, deps *executor.HandlerDeps) {
	r.RegisterFuture("CreateWorkflow", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateWorkflowGenFn(ctx, stmt.(*ast.CreateWorkflowStmt), deps)
	})
	r.RegisterFuture("DropWorkflow", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropWorkflowGenFn(ctx, stmt.(*ast.DropWorkflowStmt), deps)
	})
	r.RegisterFuture("AlterWorkflow", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterWorkflowFn(ctx, stmt.(*ast.AlterWorkflowStmt), deps)
	})
}

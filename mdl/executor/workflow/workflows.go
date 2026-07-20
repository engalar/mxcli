// SPDX-License-Identifier: Apache-2.0

package workflow

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func ExecCreateWorkflowGenFn(ctx context.Context, s *ast.CreateWorkflowStmt, deps *executor.HandlerDeps) error {
	return executor.ExecCreateWorkflowGenFn(ctx, s, deps)
}

func ExecDropWorkflowGenFn(ctx context.Context, s *ast.DropWorkflowStmt, deps *executor.HandlerDeps) error {
	return executor.ExecDropWorkflowGenFn(ctx, s, deps)
}

func ExecAlterWorkflowFn(ctx context.Context, s *ast.AlterWorkflowStmt, deps *executor.HandlerDeps) error {
	ectx := executor.NewMinimalExecCtx(ctx, deps)
	return executor.ExecAlterWorkflow(ectx, s)
}

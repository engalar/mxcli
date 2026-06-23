// SPDX-License-Identifier: Apache-2.0

package microflow

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func ExecCreateMicroflowGenFn(ctx context.Context, s *ast.CreateMicroflowStmt, deps *executor.HandlerDeps) error {
	return executor.ExecCreateMicroflowGenFn(ctx, s, deps)
}

func ExecDropMicroflowFn(ctx context.Context, s *ast.DropMicroflowStmt, deps *executor.HandlerDeps) error {
	return executor.ExecDropMicroflowFn(ctx, s, deps)
}

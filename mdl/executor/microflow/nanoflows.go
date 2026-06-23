// SPDX-License-Identifier: Apache-2.0

package microflow

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func ExecCreateNanoflowGenFn(ctx context.Context, s *ast.CreateNanoflowStmt, deps *executor.HandlerDeps) error {
	return executor.ExecCreateNanoflowGenFn(ctx, s, deps)
}

func ExecDropNanoflowGenFn(ctx context.Context, s *ast.DropNanoflowStmt, deps *executor.HandlerDeps) error {
	return executor.ExecDropNanoflowGenFn(ctx, s, deps)
}

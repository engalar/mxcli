// SPDX-License-Identifier: Apache-2.0

package microflow

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func RegisterHandlers(r *executor.Registry, deps *executor.HandlerDeps) {
	r.RegisterFuture("CreateMicroflow", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateMicroflowGenFn(ctx, stmt.(*ast.CreateMicroflowStmt), deps)
	})
	r.RegisterFuture("DropMicroflow", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropMicroflowFn(ctx, stmt.(*ast.DropMicroflowStmt), deps)
	})
	r.RegisterFuture("CreateNanoflow", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateNanoflowGenFn(ctx, stmt.(*ast.CreateNanoflowStmt), deps)
	})
	r.RegisterFuture("DropNanoflow", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropNanoflowGenFn(ctx, stmt.(*ast.DropNanoflowStmt), deps)
	})
}

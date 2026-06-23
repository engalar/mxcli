// SPDX-License-Identifier: Apache-2.0

package microflow

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/modelsdk/version"
)

func RegisterHandlers(r *executor.Registry, deps *executor.HandlerDeps) {
	v := version.Version{}
	if deps.ConnectionManager != nil && deps.ConnectionManager.IsConnected() {
		if rpv := deps.ConnectionManager.ProjectVersion(); rpv != nil {
			v = version.Parse(rpv.ProductVersion)
		}
	}
	_ = v

	r.RegisterFuture("CreateMicroflow", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateMicroflowGenFn(ctx, stmt.(*ast.CreateMicroflowStmt), deps)
	})
	r.RegisterFuture("DropMicroflow", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropMicroflowFn(ctx, stmt.(*ast.DropMicroflowStmt), deps)
	})
	r.RegisterFuture("CreateNanoflow", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateNanoflowGenFn(ctx, stmt.(*ast.CreateNanoflowStmt), deps)
	})
	r.RegisterFuture("DropNanoflow", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropNanoflowGenFn(ctx, stmt.(*ast.DropNanoflowStmt), deps)
	})
}

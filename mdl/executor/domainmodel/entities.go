// SPDX-License-Identifier: Apache-2.0

package domainmodel

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func ExecCreateEntityFn(ctx context.Context, s *ast.CreateEntityStmt, deps *executor.HandlerDeps) error {
	ectx := executor.NewExecContext(ctx, deps)
	return executor.ExecCreateEntity(ectx, s)
}

func ExecAlterEntityFn(ctx context.Context, s *ast.AlterEntityStmt, deps *executor.HandlerDeps) error {
	return executor.ExecAlterEntityGenFn(ctx, s, deps)
}

func ExecDropEntityFn(ctx context.Context, s *ast.DropEntityStmt, deps *executor.HandlerDeps) error {
	ectx := executor.NewExecContext(ctx, deps)
	return executor.ExecDropEntity(ectx, s)
}

func ExecCreateViewEntityFn(ctx context.Context, s *ast.CreateViewEntityStmt, deps *executor.HandlerDeps) error {
	ectx := executor.NewExecContext(ctx, deps)
	return executor.ExecCreateViewEntity(ectx, s)
}

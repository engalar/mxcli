// SPDX-License-Identifier: Apache-2.0

package query

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func ExecShowFn(ctx context.Context, s *ast.ShowStmt, deps *executor.HandlerDeps) error {
	return executor.ExecShow(ctx, s, deps)
}

func ExecDescribeFn(ctx context.Context, s *ast.DescribeStmt, deps *executor.HandlerDeps) error {
	return executor.ExecDescribe(ctx, s, deps)
}

func ExecSelectFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecSelectFuture(ctx, stmt, deps)
}

func ExecDescribeTranslationsFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecDescribeTranslationsFuture(ctx, stmt, deps)
}

func ExecDescribeCatalogTableFuture(ctx context.Context, deps *executor.HandlerDeps) error {
	return executor.ExecDescribeCatalogTableFuture(ctx, deps)
}

func ExecSearchFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecSearchFuture(ctx, stmt, deps)
}

func ExecLintFn(ctx context.Context, s *ast.LintStmt, deps *executor.HandlerDeps) error {
	return executor.ExecLintFn(ctx, s, deps)
}

func ExecShowFeaturesFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecShowFeaturesFuture(ctx, stmt, deps)
}

func ExecShowDesignPropertiesFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecShowDesignPropertiesFuture(ctx, stmt, deps)
}

func ExecDescribeStylingFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecDescribeStylingFuture(ctx, stmt, deps)
}

func ExecAlterStylingFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecAlterStylingFuture(ctx, stmt, deps)
}

func ExecShowThemeVariablesFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecShowThemeVariablesFuture(ctx, stmt, deps)
}

func ExecRefreshCatalogFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecRefreshCatalogFuture(ctx, stmt, deps)
}

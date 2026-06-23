// SPDX-License-Identifier: Apache-2.0

package query

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func RegisterHandlers(r *executor.Registry, deps *executor.HandlerDeps) {
	r.RegisterFuture("Show", func(ctx context.Context, stmt ast.Statement) error {
		return ExecShowFn(ctx, stmt.(*ast.ShowStmt), deps)
	})
	r.RegisterFuture("Describe", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDescribeFn(ctx, stmt.(*ast.DescribeStmt), deps)
	})
	r.RegisterFuture("Select", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSelectFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DescribeTranslations", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDescribeTranslationsFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DescribeCatalogTable", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDescribeCatalogTableFuture(ctx, deps)
	})
	r.RegisterFuture("Search", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSearchFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("Lint", func(ctx context.Context, stmt ast.Statement) error {
		return ExecLintFn(ctx, stmt.(*ast.LintStmt), deps)
	})
	r.RegisterFuture("ShowFeatures", func(ctx context.Context, stmt ast.Statement) error {
		return ExecShowFeaturesFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("ShowDesignProperties", func(ctx context.Context, stmt ast.Statement) error {
		return ExecShowDesignPropertiesFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DescribeStyling", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDescribeStylingFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("AlterStyling", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterStylingFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("ShowThemeVariables", func(ctx context.Context, stmt ast.Statement) error {
		return ExecShowThemeVariablesFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("RefreshCatalog", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRefreshCatalogFuture(ctx, stmt, deps)
	})
}

package query

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func RegisterHandlers(r *executor.Registry, deps *executor.HandlerDeps) {
	r.RegisterFuture("Show", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecShow(ctx, stmt.(*ast.ShowStmt), deps)
	})
	r.RegisterFuture("Describe", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDescribe(ctx, stmt.(*ast.DescribeStmt), deps)
	})
	r.RegisterFuture("Select", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecSelectFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DescribeTranslations", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDescribeTranslationsFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DescribeCatalogTable", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDescribeCatalogTableFuture(ctx, deps)
	})
	r.RegisterFuture("Search", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecSearchFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("Lint", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecLintFn(ctx, stmt.(*ast.LintStmt), deps)
	})
	r.RegisterFuture("ShowFeatures", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecShowFeaturesFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("ShowDesignProperties", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecShowDesignPropertiesFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DescribeStyling", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDescribeStylingFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("AlterStyling", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecAlterStylingFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("ShowThemeVariables", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecShowThemeVariablesFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("RefreshCatalog", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecRefreshCatalogFuture(ctx, stmt, deps)
	})
}

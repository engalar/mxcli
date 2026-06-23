// SPDX-License-Identifier: Apache-2.0

package executor

import "github.com/mendixlabs/mxcli/mdl/ast"

func registerQueryHandlers(r *Registry) {
	r.Register(&ast.ShowStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execShow(ctx, stmt.(*ast.ShowStmt))
	})
	r.Register(&ast.ShowWidgetsStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execShowWidgetsFn(ctx, stmt.(*ast.ShowWidgetsStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.ShowInstalledWidgetsStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execShowInstalledWidgetsFn(ctx, stmt.(*ast.ShowInstalledWidgetsStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.UpdateWidgetsStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execUpdateWidgetsFn(ctx, stmt.(*ast.UpdateWidgetsStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.SelectStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCatalogQuery(ctx, stmt.(*ast.SelectStmt).Query)
	})
	r.Register(&ast.DescribeStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDescribe(ctx, stmt.(*ast.DescribeStmt))
	})
	r.Register(&ast.DescribeTranslationsStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return describeTranslations(ctx, stmt.(*ast.DescribeTranslationsStmt))
	})
	r.Register(&ast.DescribeCatalogTableStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDescribeCatalogTable(ctx, stmt.(*ast.DescribeCatalogTableStmt))
	})
	r.Register(&ast.ShowFeaturesStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execShowFeatures(ctx, stmt.(*ast.ShowFeaturesStmt))
	})
}

func registerStylingHandlers(r *Registry) {
	r.Register(&ast.ShowDesignPropertiesStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execShowDesignPropertiesFn(ctx, stmt.(*ast.ShowDesignPropertiesStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.DescribeStylingStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDescribeStylingFn(ctx, stmt.(*ast.DescribeStylingStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.AlterStylingStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execAlterStylingFn(ctx, stmt.(*ast.AlterStylingStmt), execContextToDeps(ctx))
	})
}

func registerThemeCommandHandlers(r *Registry) {
	r.Register(&ast.ShowThemeVariablesStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execShowThemeVariablesFn(ctx, stmt.(*ast.ShowThemeVariablesStmt), execContextToDeps(ctx))
	})
}

func registerRepositoryHandlers(r *Registry) {
	r.Register(&ast.UpdateStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execUpdate(ctx)
	})
	r.Register(&ast.RefreshStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execRefresh(ctx)
	})
	r.Register(&ast.RefreshCatalogStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		if err := execRefreshCatalogStmt(ctx, stmt.(*ast.RefreshCatalogStmt)); err != nil {
			return err
		}
		return buildGraph(ctx)
	})
	r.Register(&ast.SearchStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSearch(ctx, stmt.(*ast.SearchStmt))
	})
}

func registerSessionHandlers(r *Registry) {
	r.Register(&ast.SetStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSet(ctx, stmt.(*ast.SetStmt))
	})
	r.Register(&ast.HelpStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execHelp(ctx, stmt.(*ast.HelpStmt))
	})
	r.Register(&ast.ExitStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execExit(ctx)
	})
	r.Register(&ast.ExecuteScriptStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execExecuteScript(ctx, stmt.(*ast.ExecuteScriptStmt))
	})
}

func registerLintHandlers(r *Registry) {
	r.Register(&ast.LintStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execLintFn(ctx, stmt.(*ast.LintStmt), execContextToDeps(ctx))
	})
}

func registerFragmentHandlers(r *Registry) {
	r.Register(&ast.DefineFragmentStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDefineFragmentFn(ctx, stmt.(*ast.DefineFragmentStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.DescribeFragmentFromStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return describeFragmentFrom(ctx, stmt.(*ast.DescribeFragmentFromStmt))
	})
}

func registerSQLHandlers(r *Registry) {
	r.Register(&ast.SQLConnectStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSQLConnectFn(ctx, stmt.(*ast.SQLConnectStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.SQLDisconnectStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSQLDisconnectFn(ctx, stmt.(*ast.SQLDisconnectStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.SQLConnectionsStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSQLConnectionsFn(ctx, execContextToDeps(ctx))
	})
	r.Register(&ast.SQLQueryStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSQLQueryFn(ctx, stmt.(*ast.SQLQueryStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.SQLShowTablesStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSQLShowTablesFn(ctx, stmt.(*ast.SQLShowTablesStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.SQLShowViewsStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSQLShowViewsFn(ctx, stmt.(*ast.SQLShowViewsStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.SQLShowFunctionsStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSQLShowFunctionsFn(ctx, stmt.(*ast.SQLShowFunctionsStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.SQLDescribeTableStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSQLDescribeTableFn(ctx, stmt.(*ast.SQLDescribeTableStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.SQLGenerateConnectorStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSQLGenerateConnectorFn(ctx, stmt.(*ast.SQLGenerateConnectorStmt), execContextToDeps(ctx))
	})
}

func registerImportHandlers(r *Registry) {
	r.Register(&ast.ImportStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execImportFn(ctx, stmt.(*ast.ImportStmt), execContextToDeps(ctx))
	})
}

// SPDX-License-Identifier: Apache-2.0

package executor

import "github.com/mendixlabs/mxcli/mdl/ast"

func registerQueryHandlers(r *Registry) {
	r.Register(&ast.ShowStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execShow(ctx, stmt.(*ast.ShowStmt))
	})
	r.Register(&ast.ShowWidgetsStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execShowWidgets(ctx, stmt.(*ast.ShowWidgetsStmt))
	})
	r.Register(&ast.ShowInstalledWidgetsStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execShowInstalledWidgets(ctx, stmt.(*ast.ShowInstalledWidgetsStmt))
	})
	r.Register(&ast.UpdateWidgetsStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execUpdateWidgets(ctx, stmt.(*ast.UpdateWidgetsStmt))
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
		return execShowDesignProperties(ctx, stmt.(*ast.ShowDesignPropertiesStmt))
	})
	r.Register(&ast.DescribeStylingStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDescribeStyling(ctx, stmt.(*ast.DescribeStylingStmt))
	})
	r.Register(&ast.AlterStylingStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execAlterStyling(ctx, stmt.(*ast.AlterStylingStmt))
	})
}

func registerThemeCommandHandlers(r *Registry) {
	r.Register(&ast.ShowThemeVariablesStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execShowThemeVariables(ctx, stmt.(*ast.ShowThemeVariablesStmt))
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
		return execLint(ctx, stmt.(*ast.LintStmt))
	})
}

func registerFragmentHandlers(r *Registry) {
	r.Register(&ast.DefineFragmentStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDefineFragment(ctx, stmt.(*ast.DefineFragmentStmt))
	})
	r.Register(&ast.DescribeFragmentFromStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return describeFragmentFrom(ctx, stmt.(*ast.DescribeFragmentFromStmt))
	})
}

func registerSQLHandlers(r *Registry) {
	r.Register(&ast.SQLConnectStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSQLConnect(ctx, stmt.(*ast.SQLConnectStmt))
	})
	r.Register(&ast.SQLDisconnectStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSQLDisconnect(ctx, stmt.(*ast.SQLDisconnectStmt))
	})
	r.Register(&ast.SQLConnectionsStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSQLConnections(ctx)
	})
	r.Register(&ast.SQLQueryStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSQLQuery(ctx, stmt.(*ast.SQLQueryStmt))
	})
	r.Register(&ast.SQLShowTablesStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSQLShowTables(ctx, stmt.(*ast.SQLShowTablesStmt))
	})
	r.Register(&ast.SQLShowViewsStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSQLShowViews(ctx, stmt.(*ast.SQLShowViewsStmt))
	})
	r.Register(&ast.SQLShowFunctionsStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSQLShowFunctions(ctx, stmt.(*ast.SQLShowFunctionsStmt))
	})
	r.Register(&ast.SQLDescribeTableStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSQLDescribeTable(ctx, stmt.(*ast.SQLDescribeTableStmt))
	})
	r.Register(&ast.SQLGenerateConnectorStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execSQLGenerateConnector(ctx, stmt.(*ast.SQLGenerateConnectorStmt))
	})
}

func registerImportHandlers(r *Registry) {
	r.Register(&ast.ImportStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execImport(ctx, stmt.(*ast.ImportStmt))
	})
}

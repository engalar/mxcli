// SPDX-License-Identifier: Apache-2.0

package misc

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func RegisterHandlers(r *executor.Registry, deps *executor.HandlerDeps) {
	// Module CRUD
	r.RegisterFuture("CreateModule", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateModuleFn(ctx, stmt.(*ast.CreateModuleStmt), deps)
	})
	r.RegisterFuture("DropModule", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropModuleFn(ctx, stmt.(*ast.DropModuleStmt), deps)
	})

	// Module settings
	r.RegisterFuture("AlterModuleJarDep", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterModuleJarDepFuture(ctx, stmt, deps)
	})

	// Settings
	r.RegisterFuture("AlterSettings", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterSettingsFuture(ctx, stmt, deps)
	})

	// Configuration CRUD
	r.RegisterFuture("CreateConfiguration", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateConfigurationFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropConfiguration", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropConfigurationFuture(ctx, stmt, deps)
	})

	// Translate
	r.RegisterFuture("Translate", func(ctx context.Context, stmt ast.Statement) error {
		return ExecTranslateFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("TranslateMicroflow", func(ctx context.Context, stmt ast.Statement) error {
		return ExecTranslateMicroflowFuture(ctx, deps)
	})

	// Java/JavaScript action CRUD
	r.RegisterFuture("CreateJavaAction", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateJavaActionFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropJavaAction", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropJavaActionFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("CreateJavaScriptAction", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateJavaScriptActionFuture(ctx, stmt, deps)
	})

	// Folder/rename/move
	r.RegisterFuture("DropFolder", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropFolderFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("MoveFolder", func(ctx context.Context, stmt ast.Statement) error {
		return ExecMoveFolderFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("Move", func(ctx context.Context, stmt ast.Statement) error {
		return ExecMoveFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("Rename", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRenameFuture(ctx, stmt, deps)
	})

	// Fragment commands
	r.RegisterFuture("DefineFragment", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDefineFragmentFn(ctx, stmt.(*ast.DefineFragmentStmt), deps)
	})
	r.RegisterFuture("DescribeFragmentFrom", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDescribeFragmentFromFn(ctx, stmt.(*ast.DescribeFragmentFromStmt), deps)
	})

	// SQL commands
	r.RegisterFuture("SQLConnect", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSQLConnectFn(ctx, stmt.(*ast.SQLConnectStmt), deps)
	})
	r.RegisterFuture("SQLDisconnect", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSQLDisconnectFn(ctx, stmt.(*ast.SQLDisconnectStmt), deps)
	})
	r.RegisterFuture("SQLConnections", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSQLConnectionsFn(ctx, deps)
	})
	r.RegisterFuture("SQLQuery", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSQLQueryFn(ctx, stmt.(*ast.SQLQueryStmt), deps)
	})
	r.RegisterFuture("SQLShowTables", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSQLShowTablesFn(ctx, stmt.(*ast.SQLShowTablesStmt), deps)
	})
	r.RegisterFuture("SQLShowViews", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSQLShowViewsFn(ctx, stmt.(*ast.SQLShowViewsStmt), deps)
	})
	r.RegisterFuture("SQLShowFunctions", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSQLShowFunctionsFn(ctx, stmt.(*ast.SQLShowFunctionsStmt), deps)
	})
	r.RegisterFuture("SQLDescribeTable", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSQLDescribeTableFn(ctx, stmt.(*ast.SQLDescribeTableStmt), deps)
	})
	r.RegisterFuture("SQLGenerateConnector", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSQLGenerateConnectorFn(ctx, stmt.(*ast.SQLGenerateConnectorStmt), deps)
	})

	// Import
	r.RegisterFuture("Import", func(ctx context.Context, stmt ast.Statement) error {
		return ExecImportFn(ctx, stmt.(*ast.ImportStmt), deps)
	})

	// Business event service CRUD
	r.RegisterFuture("CreateBusinessEventService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateBusinessEventServiceFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropBusinessEventService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropBusinessEventServiceFuture(ctx, stmt, deps)
	})

	// OData client CRUD
	r.RegisterFuture("CreateODataClient", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateODataClientFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("AlterODataClient", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterODataClientFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropODataClient", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropODataClientFuture(ctx, stmt, deps)
	})

	// OData service CRUD
	r.RegisterFuture("CreateODataService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateODataServiceFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("AlterODataService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterODataServiceFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropODataService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropODataServiceFuture(ctx, stmt, deps)
	})

	// JSON structure CRUD
	r.RegisterFuture("CreateJsonStructure", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateJsonStructureFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropJsonStructure", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropJsonStructureFuture(ctx, stmt, deps)
	})

	// Import/Export mapping CRUD
	r.RegisterFuture("CreateImportMapping", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateImportMappingFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropImportMapping", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropImportMappingFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("CreateExportMapping", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateExportMappingFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropExportMapping", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropExportMappingFuture(ctx, stmt, deps)
	})

	// REST client CRUD
	r.RegisterFuture("CreateRestClient", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateRestClientFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropRestClient", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropRestClientFuture(ctx, stmt, deps)
	})

	// Contract from OpenAPI
	r.RegisterFuture("DescribeContractFromOpenAPI", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDescribeContractFromOpenAPIFuture(ctx, stmt, deps)
	})

	// Published REST service CRUD
	r.RegisterFuture("CreatePublishedRestService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreatePublishedRestServiceFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropPublishedRestService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropPublishedRestServiceFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("AlterPublishedRestService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterPublishedRestServiceFuture(ctx, stmt, deps)
	})

	// External entities
	r.RegisterFuture("CreateExternalEntity", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateExternalEntityFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("CreateExternalEntities", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateExternalEntitiesFuture(ctx, stmt, deps)
	})

	// Data transformer CRUD
	r.RegisterFuture("CreateDataTransformer", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateDataTransformerFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropDataTransformer", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropDataTransformerFuture(ctx, stmt, deps)
	})

	// Widget commands
	r.RegisterFuture("ShowWidgets", func(ctx context.Context, stmt ast.Statement) error {
		return ExecShowWidgetsFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("ShowInstalledWidgets", func(ctx context.Context, stmt ast.Statement) error {
		return ExecShowInstalledWidgetsFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("UpdateWidgets", func(ctx context.Context, stmt ast.Statement) error {
		return ExecUpdateWidgetsFuture(ctx, stmt, deps)
	})

	// Agent editor CRUD
	r.RegisterFuture("CreateModel", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateModelFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropModel", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropModelFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("CreateConsumedMCPService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateConsumedMCPServiceFn(ctx, stmt.(*ast.CreateConsumedMCPServiceStmt), deps)
	})
	r.RegisterFuture("DropConsumedMCPService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropConsumedMCPServiceFn(ctx, stmt.(*ast.DropConsumedMCPServiceStmt), deps)
	})
	r.RegisterFuture("CreateKnowledgeBase", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateKnowledgeBaseFn(ctx, stmt.(*ast.CreateKnowledgeBaseStmt), deps)
	})
	r.RegisterFuture("DropKnowledgeBase", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropKnowledgeBaseFn(ctx, stmt.(*ast.DropKnowledgeBaseStmt), deps)
	})
	r.RegisterFuture("CreateAgent", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateAgentFn(ctx, stmt.(*ast.CreateAgentStmt), deps)
	})
	r.RegisterFuture("DropAgent", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropAgentFn(ctx, stmt.(*ast.DropAgentStmt), deps)
	})
}

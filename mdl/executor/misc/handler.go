package misc

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func RegisterHandlers(r *executor.Registry, deps *executor.HandlerDeps) {
	// Module CRUD
	r.RegisterFuture("CreateModule", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateModuleFn(ctx, stmt.(*ast.CreateModuleStmt), deps)
	})
	r.RegisterFuture("DropModule", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropModuleFn(ctx, stmt.(*ast.DropModuleStmt), deps)
	})

	// Module settings
	r.RegisterFuture("AlterModuleJarDep", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecAlterModuleJarDepFuture(ctx, stmt, deps)
	})

	// Settings
	r.RegisterFuture("AlterSettings", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecAlterSettingsFuture(ctx, stmt, deps)
	})

	// Configuration CRUD
	r.RegisterFuture("CreateConfiguration", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateConfigurationFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropConfiguration", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropConfigurationFuture(ctx, stmt, deps)
	})

	// Translate
	r.RegisterFuture("Translate", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecTranslateFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("TranslateMicroflow", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecTranslateMicroflowFuture(ctx, deps)
	})

	// Java/JavaScript action CRUD
	r.RegisterFuture("CreateJavaAction", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateJavaActionFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropJavaAction", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropJavaActionFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("CreateJavaScriptAction", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateJavaScriptActionFuture(ctx, stmt, deps)
	})

	// Folder/rename/move
	r.RegisterFuture("DropFolder", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropFolderFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("MoveFolder", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecMoveFolderFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("Move", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecMoveFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("Rename", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecRenameFuture(ctx, stmt, deps)
	})

	// Fragment commands
	r.RegisterFuture("DefineFragment", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDefineFragmentFn(ctx, stmt.(*ast.DefineFragmentStmt), deps)
	})
	r.RegisterFuture("DescribeFragmentFrom", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDescribeFragmentFromFn(ctx, stmt.(*ast.DescribeFragmentFromStmt), deps)
	})

	// SQL commands
	r.RegisterFuture("SQLConnect", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecSQLConnectFn(ctx, stmt.(*ast.SQLConnectStmt), deps)
	})
	r.RegisterFuture("SQLDisconnect", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecSQLDisconnectFn(ctx, stmt.(*ast.SQLDisconnectStmt), deps)
	})
	r.RegisterFuture("SQLConnections", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecSQLConnectionsFn(ctx, deps)
	})
	r.RegisterFuture("SQLQuery", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecSQLQueryFn(ctx, stmt.(*ast.SQLQueryStmt), deps)
	})
	r.RegisterFuture("SQLShowTables", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecSQLShowTablesFn(ctx, stmt.(*ast.SQLShowTablesStmt), deps)
	})
	r.RegisterFuture("SQLShowViews", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecSQLShowViewsFn(ctx, stmt.(*ast.SQLShowViewsStmt), deps)
	})
	r.RegisterFuture("SQLShowFunctions", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecSQLShowFunctionsFn(ctx, stmt.(*ast.SQLShowFunctionsStmt), deps)
	})
	r.RegisterFuture("SQLDescribeTable", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecSQLDescribeTableFn(ctx, stmt.(*ast.SQLDescribeTableStmt), deps)
	})
	r.RegisterFuture("SQLGenerateConnector", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecSQLGenerateConnectorFn(ctx, stmt.(*ast.SQLGenerateConnectorStmt), deps)
	})

	// Import
	r.RegisterFuture("Import", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecImportFn(ctx, stmt.(*ast.ImportStmt), deps)
	})

	// Business event service CRUD
	r.RegisterFuture("CreateBusinessEventService", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateBusinessEventServiceFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropBusinessEventService", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropBusinessEventServiceFuture(ctx, stmt, deps)
	})

	// OData client CRUD
	r.RegisterFuture("CreateODataClient", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateODataClientFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("AlterODataClient", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecAlterODataClientFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropODataClient", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropODataClientFuture(ctx, stmt, deps)
	})

	// OData service CRUD
	r.RegisterFuture("CreateODataService", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateODataServiceFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("AlterODataService", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecAlterODataServiceFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropODataService", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropODataServiceFuture(ctx, stmt, deps)
	})

	// JSON structure CRUD
	r.RegisterFuture("CreateJsonStructure", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateJsonStructureFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropJsonStructure", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropJsonStructureFuture(ctx, stmt, deps)
	})

	// Import/Export mapping CRUD
	r.RegisterFuture("CreateImportMapping", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateImportMappingFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropImportMapping", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropImportMappingFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("CreateExportMapping", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateExportMappingFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropExportMapping", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropExportMappingFuture(ctx, stmt, deps)
	})

	// REST client CRUD
	r.RegisterFuture("CreateRestClient", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateRestClientFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropRestClient", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropRestClientFuture(ctx, stmt, deps)
	})

	// Contract from OpenAPI
	r.RegisterFuture("DescribeContractFromOpenAPI", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDescribeContractFromOpenAPIFuture(ctx, stmt, deps)
	})

	// Published REST service CRUD
	r.RegisterFuture("CreatePublishedRestService", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreatePublishedRestServiceFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropPublishedRestService", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropPublishedRestServiceFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("AlterPublishedRestService", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecAlterPublishedRestServiceFuture(ctx, stmt, deps)
	})

	// External entities
	r.RegisterFuture("CreateExternalEntity", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateExternalEntityFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("CreateExternalEntities", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateExternalEntitiesFuture(ctx, stmt, deps)
	})

	// Data transformer CRUD
	r.RegisterFuture("CreateDataTransformer", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateDataTransformerFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropDataTransformer", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropDataTransformerFuture(ctx, stmt, deps)
	})

	// Widget commands
	r.RegisterFuture("ShowWidgets", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecShowWidgetsFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("ShowInstalledWidgets", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecShowInstalledWidgetsFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("UpdateWidgets", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecUpdateWidgetsFuture(ctx, stmt, deps)
	})

	// Agent editor CRUD
	r.RegisterFuture("CreateModel", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateModelFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropModel", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropModelFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("CreateConsumedMCPService", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateConsumedMCPServiceFn(ctx, stmt.(*ast.CreateConsumedMCPServiceStmt), deps)
	})
	r.RegisterFuture("DropConsumedMCPService", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropConsumedMCPServiceFn(ctx, stmt.(*ast.DropConsumedMCPServiceStmt), deps)
	})
	r.RegisterFuture("CreateKnowledgeBase", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateKnowledgeBaseFn(ctx, stmt.(*ast.CreateKnowledgeBaseStmt), deps)
	})
	r.RegisterFuture("DropKnowledgeBase", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropKnowledgeBaseFn(ctx, stmt.(*ast.DropKnowledgeBaseStmt), deps)
	})
	r.RegisterFuture("CreateAgent", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateAgentFn(ctx, stmt.(*ast.CreateAgentStmt), deps)
	})
	r.RegisterFuture("DropAgent", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropAgentFn(ctx, stmt.(*ast.DropAgentStmt), deps)
	})
}

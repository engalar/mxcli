// SPDX-License-Identifier: Apache-2.0

package misc

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func ExecCreateModuleFn(ctx context.Context, s *ast.CreateModuleStmt, deps *executor.HandlerDeps) error {
	return executor.ExecCreateModuleFn(ctx, s, deps)
}

func ExecDropModuleFn(ctx context.Context, s *ast.DropModuleStmt, deps *executor.HandlerDeps) error {
	return executor.ExecDropModuleFn(ctx, s, deps)
}

func ExecAlterModuleJarDepFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecAlterModuleJarDepFuture(ctx, stmt, deps)
}

func ExecAlterSettingsFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecAlterSettingsFuture(ctx, stmt, deps)
}

func ExecCreateConfigurationFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecCreateConfigurationFuture(ctx, stmt, deps)
}

func ExecDropConfigurationFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecDropConfigurationFuture(ctx, stmt, deps)
}

func ExecTranslateFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecTranslateFuture(ctx, stmt, deps)
}

func ExecTranslateMicroflowFuture(ctx context.Context, deps *executor.HandlerDeps) error {
	return executor.ExecTranslateMicroflowFuture(ctx, deps)
}

func ExecCreateJavaActionFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecCreateJavaActionFuture(ctx, stmt, deps)
}

func ExecDropJavaActionFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecDropJavaActionFuture(ctx, stmt, deps)
}

func ExecCreateJavaScriptActionFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecCreateJavaScriptActionFuture(ctx, stmt, deps)
}

func ExecDropFolderFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecDropFolderFuture(ctx, stmt, deps)
}

func ExecMoveFolderFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecMoveFolderFuture(ctx, stmt, deps)
}

func ExecMoveFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecMoveFuture(ctx, stmt, deps)
}

func ExecRenameFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecRenameFuture(ctx, stmt, deps)
}

func ExecDefineFragmentFn(ctx context.Context, s *ast.DefineFragmentStmt, deps *executor.HandlerDeps) error {
	return executor.ExecDefineFragmentFn(ctx, s, deps)
}

func ExecDescribeFragmentFromFn(ctx context.Context, s *ast.DescribeFragmentFromStmt, deps *executor.HandlerDeps) error {
	return executor.ExecDescribeFragmentFromFn(ctx, s, deps)
}

func ExecSQLConnectFn(ctx context.Context, s *ast.SQLConnectStmt, deps *executor.HandlerDeps) error {
	return executor.ExecSQLConnectFn(ctx, s, deps)
}

func ExecSQLDisconnectFn(ctx context.Context, s *ast.SQLDisconnectStmt, deps *executor.HandlerDeps) error {
	return executor.ExecSQLDisconnectFn(ctx, s, deps)
}

func ExecSQLConnectionsFn(ctx context.Context, deps *executor.HandlerDeps) error {
	return executor.ExecSQLConnectionsFn(ctx, deps)
}

func ExecSQLQueryFn(ctx context.Context, s *ast.SQLQueryStmt, deps *executor.HandlerDeps) error {
	return executor.ExecSQLQueryFn(ctx, s, deps)
}

func ExecSQLShowTablesFn(ctx context.Context, s *ast.SQLShowTablesStmt, deps *executor.HandlerDeps) error {
	return executor.ExecSQLShowTablesFn(ctx, s, deps)
}

func ExecSQLShowViewsFn(ctx context.Context, s *ast.SQLShowViewsStmt, deps *executor.HandlerDeps) error {
	return executor.ExecSQLShowViewsFn(ctx, s, deps)
}

func ExecSQLShowFunctionsFn(ctx context.Context, s *ast.SQLShowFunctionsStmt, deps *executor.HandlerDeps) error {
	return executor.ExecSQLShowFunctionsFn(ctx, s, deps)
}

func ExecSQLDescribeTableFn(ctx context.Context, s *ast.SQLDescribeTableStmt, deps *executor.HandlerDeps) error {
	return executor.ExecSQLDescribeTableFn(ctx, s, deps)
}

func ExecSQLGenerateConnectorFn(ctx context.Context, s *ast.SQLGenerateConnectorStmt, deps *executor.HandlerDeps) error {
	return executor.ExecSQLGenerateConnectorFn(ctx, s, deps)
}

func ExecImportFn(ctx context.Context, s *ast.ImportStmt, deps *executor.HandlerDeps) error {
	return executor.ExecImportFn(ctx, s, deps)
}

func ExecCreateBusinessEventServiceFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecCreateBusinessEventServiceFuture(ctx, stmt, deps)
}

func ExecDropBusinessEventServiceFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecDropBusinessEventServiceFuture(ctx, stmt, deps)
}

func ExecCreateODataClientFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecCreateODataClientFuture(ctx, stmt, deps)
}

func ExecAlterODataClientFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecAlterODataClientFuture(ctx, stmt, deps)
}

func ExecDropODataClientFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecDropODataClientFuture(ctx, stmt, deps)
}

func ExecCreateODataServiceFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecCreateODataServiceFuture(ctx, stmt, deps)
}

func ExecAlterODataServiceFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecAlterODataServiceFuture(ctx, stmt, deps)
}

func ExecDropODataServiceFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecDropODataServiceFuture(ctx, stmt, deps)
}

func ExecCreateJsonStructureFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecCreateJsonStructureFuture(ctx, stmt, deps)
}

func ExecDropJsonStructureFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecDropJsonStructureFuture(ctx, stmt, deps)
}

func ExecCreateImportMappingFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecCreateImportMappingFuture(ctx, stmt, deps)
}

func ExecDropImportMappingFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecDropImportMappingFuture(ctx, stmt, deps)
}

func ExecCreateExportMappingFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecCreateExportMappingFuture(ctx, stmt, deps)
}

func ExecDropExportMappingFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecDropExportMappingFuture(ctx, stmt, deps)
}

func ExecCreateRestClientFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecCreateRestClientFuture(ctx, stmt, deps)
}

func ExecDropRestClientFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecDropRestClientFuture(ctx, stmt, deps)
}

func ExecDescribeContractFromOpenAPIFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecDescribeContractFromOpenAPIFuture(ctx, stmt, deps)
}

func ExecCreatePublishedRestServiceFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecCreatePublishedRestServiceFuture(ctx, stmt, deps)
}

func ExecDropPublishedRestServiceFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecDropPublishedRestServiceFuture(ctx, stmt, deps)
}

func ExecAlterPublishedRestServiceFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecAlterPublishedRestServiceFuture(ctx, stmt, deps)
}

func ExecCreateExternalEntityFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecCreateExternalEntityFuture(ctx, stmt, deps)
}

func ExecCreateExternalEntitiesFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecCreateExternalEntitiesFuture(ctx, stmt, deps)
}

func ExecCreateDataTransformerFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecCreateDataTransformerFuture(ctx, stmt, deps)
}

func ExecDropDataTransformerFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecDropDataTransformerFuture(ctx, stmt, deps)
}

func ExecShowWidgetsFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecShowWidgetsFuture(ctx, stmt, deps)
}

func ExecShowInstalledWidgetsFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecShowInstalledWidgetsFuture(ctx, stmt, deps)
}

func ExecUpdateWidgetsFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecUpdateWidgetsFuture(ctx, stmt, deps)
}

func ExecCreateModelFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecCreateModelFuture(ctx, stmt, deps)
}

func ExecDropModelFuture(ctx context.Context, stmt ast.Statement, deps *executor.HandlerDeps) error {
	return executor.ExecDropModelFuture(ctx, stmt, deps)
}

func ExecCreateConsumedMCPServiceFn(ctx context.Context, s *ast.CreateConsumedMCPServiceStmt, deps *executor.HandlerDeps) error {
	return executor.ExecCreateConsumedMCPServiceFn(ctx, s, deps)
}

func ExecDropConsumedMCPServiceFn(ctx context.Context, s *ast.DropConsumedMCPServiceStmt, deps *executor.HandlerDeps) error {
	return executor.ExecDropConsumedMCPServiceFn(ctx, s, deps)
}

func ExecCreateKnowledgeBaseFn(ctx context.Context, s *ast.CreateKnowledgeBaseStmt, deps *executor.HandlerDeps) error {
	return executor.ExecCreateKnowledgeBaseFn(ctx, s, deps)
}

func ExecDropKnowledgeBaseFn(ctx context.Context, s *ast.DropKnowledgeBaseStmt, deps *executor.HandlerDeps) error {
	return executor.ExecDropKnowledgeBaseFn(ctx, s, deps)
}

func ExecCreateAgentFn(ctx context.Context, s *ast.CreateAgentStmt, deps *executor.HandlerDeps) error {
	return executor.ExecCreateAgentFn(ctx, s, deps)
}

func ExecDropAgentFn(ctx context.Context, s *ast.DropAgentStmt, deps *executor.HandlerDeps) error {
	return executor.ExecDropAgentFn(ctx, s, deps)
}

// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// executeInner dispatches a statement to its handler.
func (e *Executor) executeInner(stmt ast.Statement) error {
	switch s := stmt.(type) {
	// Connection statements
	case *ast.ConnectStmt:
		return e.execConnect(s)
	case *ast.DisconnectStmt:
		return e.execDisconnect()
	case *ast.StatusStmt:
		return e.execStatus()

	// Module statements
	case *ast.CreateModuleStmt:
		return e.execCreateModule(s)
	case *ast.DropModuleStmt:
		return e.execDropModule(s)

	// Enumeration statements
	case *ast.CreateEnumerationStmt:
		return e.execCreateEnumeration(s)
	case *ast.AlterEnumerationStmt:
		return e.execAlterEnumeration(s)
	case *ast.DropEnumerationStmt:
		return e.execDropEnumeration(s)

	// Constant statements
	case *ast.CreateConstantStmt:
		return e.createConstant(s)
	case *ast.DropConstantStmt:
		return e.dropConstant(s)

	// Database Connection statements
	case *ast.CreateDatabaseConnectionStmt:
		return e.createDatabaseConnection(s)

	// Entity statements
	case *ast.CreateEntityStmt:
		return e.execCreateEntity(s)
	case *ast.CreateViewEntityStmt:
		return e.execCreateViewEntity(s)
	case *ast.AlterEntityStmt:
		return e.execAlterEntity(s)
	case *ast.DropEntityStmt:
		return e.execDropEntity(s)

	// Association statements
	case *ast.CreateAssociationStmt:
		return e.execCreateAssociation(s)
	case *ast.AlterAssociationStmt:
		return e.execAlterAssociation(s)
	case *ast.DropAssociationStmt:
		return e.execDropAssociation(s)

	// Microflow statements
	case *ast.CreateMicroflowStmt:
		return e.execCreateMicroflow(s)
	case *ast.DropMicroflowStmt:
		return e.execDropMicroflow(s)

	// Page statements
	case *ast.CreatePageStmtV3:
		return e.execCreatePageV3(s)
	case *ast.DropPageStmt:
		return e.execDropPage(s)
	case *ast.CreateSnippetStmtV3:
		return e.execCreateSnippetV3(s)
	case *ast.DropSnippetStmt:
		return e.execDropSnippet(s)
	case *ast.DropJavaActionStmt:
		return e.execDropJavaAction(s)
	case *ast.CreateJavaActionStmt:
		return e.execCreateJavaAction(s)
	case *ast.DropFolderStmt:
		return e.execDropFolder(s)
	case *ast.MoveFolderStmt:
		return e.execMoveFolder(s)
	case *ast.MoveStmt:
		return e.execMove(s)
	case *ast.RenameStmt:
		return e.execRename(s)

	// Security statements
	case *ast.CreateModuleRoleStmt:
		return e.execCreateModuleRole(s)
	case *ast.DropModuleRoleStmt:
		return e.execDropModuleRole(s)
	case *ast.CreateUserRoleStmt:
		return e.execCreateUserRole(s)
	case *ast.AlterUserRoleStmt:
		return e.execAlterUserRole(s)
	case *ast.DropUserRoleStmt:
		return e.execDropUserRole(s)
	case *ast.GrantEntityAccessStmt:
		return e.execGrantEntityAccess(s)
	case *ast.RevokeEntityAccessStmt:
		return e.execRevokeEntityAccess(s)
	case *ast.GrantMicroflowAccessStmt:
		return e.execGrantMicroflowAccess(s)
	case *ast.RevokeMicroflowAccessStmt:
		return e.execRevokeMicroflowAccess(s)
	case *ast.GrantPageAccessStmt:
		return e.execGrantPageAccess(s)
	case *ast.RevokePageAccessStmt:
		return e.execRevokePageAccess(s)
	case *ast.GrantWorkflowAccessStmt:
		return e.execGrantWorkflowAccess(s)
	case *ast.RevokeWorkflowAccessStmt:
		return e.execRevokeWorkflowAccess(s)
	case *ast.AlterProjectSecurityStmt:
		return e.execAlterProjectSecurity(s)
	case *ast.CreateDemoUserStmt:
		return e.execCreateDemoUser(s)
	case *ast.DropDemoUserStmt:
		return e.execDropDemoUser(s)
	case *ast.UpdateSecurityStmt:
		return e.execUpdateSecurity(s)

	// Navigation statements
	case *ast.AlterNavigationStmt:
		return e.execAlterNavigation(s)

	// Image collection statements
	case *ast.CreateImageCollectionStmt:
		return e.execCreateImageCollection(s)
	case *ast.DropImageCollectionStmt:
		return e.execDropImageCollection(s)

	// Workflow statements
	case *ast.CreateWorkflowStmt:
		return e.execCreateWorkflow(s)
	case *ast.DropWorkflowStmt:
		return e.execDropWorkflow(s)
	case *ast.AlterWorkflowStmt:
		return e.execAlterWorkflow(s)

	// Business Event statements
	case *ast.CreateBusinessEventServiceStmt:
		return e.createBusinessEventService(s)
	case *ast.DropBusinessEventServiceStmt:
		return e.dropBusinessEventService(s)

	// Settings statements
	case *ast.AlterSettingsStmt:
		return e.alterSettings(s)
	case *ast.CreateConfigurationStmt:
		return e.createConfiguration(s)
	case *ast.DropConfigurationStmt:
		return e.dropConfiguration(s)

	// OData statements
	case *ast.CreateODataClientStmt:
		return e.createODataClient(s)
	case *ast.AlterODataClientStmt:
		return e.alterODataClient(s)
	case *ast.DropODataClientStmt:
		return e.dropODataClient(s)
	case *ast.CreateODataServiceStmt:
		return e.createODataService(s)
	case *ast.AlterODataServiceStmt:
		return e.alterODataService(s)
	case *ast.DropODataServiceStmt:
		return e.dropODataService(s)

	// JSON Structure statements
	case *ast.CreateJsonStructureStmt:
		return e.execCreateJsonStructure(s)
	case *ast.DropJsonStructureStmt:
		return e.execDropJsonStructure(s)

	// Import Mapping statements
	case *ast.CreateImportMappingStmt:
		return e.execCreateImportMapping(s)
	case *ast.DropImportMappingStmt:
		return e.execDropImportMapping(s)

	// Export Mapping statements
	case *ast.CreateExportMappingStmt:
		return e.execCreateExportMapping(s)
	case *ast.DropExportMappingStmt:
		return e.execDropExportMapping(s)

	// REST client statements
	case *ast.CreateRestClientStmt:
		return e.createRestClient(s)
	case *ast.DropRestClientStmt:
		return e.dropRestClient(s)

	// Published REST service statements
	case *ast.CreatePublishedRestServiceStmt:
		return e.execCreatePublishedRestService(s)
	case *ast.DropPublishedRestServiceStmt:
		return e.execDropPublishedRestService(s)
	case *ast.AlterPublishedRestServiceStmt:
		return e.execAlterPublishedRestService(s)
	case *ast.CreateExternalEntityStmt:
		return e.execCreateExternalEntity(s)
	case *ast.CreateExternalEntitiesStmt:
		return e.createExternalEntities(s)
	case *ast.GrantODataServiceAccessStmt:
		return e.execGrantODataServiceAccess(s)
	case *ast.RevokeODataServiceAccessStmt:
		return e.execRevokeODataServiceAccess(s)
	case *ast.GrantPublishedRestServiceAccessStmt:
		return e.execGrantPublishedRestServiceAccess(s)
	case *ast.RevokePublishedRestServiceAccessStmt:
		return e.execRevokePublishedRestServiceAccess(s)

	// Query statements
	case *ast.ShowStmt:
		return e.execShow(s)
	case *ast.ShowWidgetsStmt:
		return e.execShowWidgets(s)
	case *ast.UpdateWidgetsStmt:
		return e.execUpdateWidgets(s)
	case *ast.SelectStmt:
		return e.execCatalogQuery(s.Query)
	case *ast.DescribeStmt:
		return e.execDescribe(s)
	case *ast.DescribeCatalogTableStmt:
		return e.execDescribeCatalogTable(s)

	// Styling statements
	case *ast.ShowDesignPropertiesStmt:
		return e.execShowDesignProperties(s)
	case *ast.DescribeStylingStmt:
		return e.execDescribeStyling(s)
	case *ast.AlterStylingStmt:
		return e.execAlterStyling(s)

	// Repository statements
	case *ast.UpdateStmt:
		return e.execUpdate()
	case *ast.RefreshStmt:
		return e.execRefresh()
	case *ast.RefreshCatalogStmt:
		return e.execRefreshCatalogStmt(s)
	case *ast.SearchStmt:
		return e.execSearch(s)

	// Session statements
	case *ast.SetStmt:
		return e.execSet(s)
	case *ast.HelpStmt:
		return e.execHelp(s)
	case *ast.ExitStmt:
		return e.execExit()
	case *ast.ExecuteScriptStmt:
		return e.execExecuteScript(s)

	// Lint statements
	case *ast.LintStmt:
		return e.execLint(s)

	// ALTER PAGE statements
	case *ast.AlterPageStmt:
		return e.execAlterPage(s)

	// Fragment statements
	case *ast.DefineFragmentStmt:
		return e.execDefineFragment(s)
	case *ast.DescribeFragmentFromStmt:
		return e.describeFragmentFrom(s)

	// SQL statements (external database connectivity)
	case *ast.SQLConnectStmt:
		return e.execSQLConnect(s)
	case *ast.SQLDisconnectStmt:
		return e.execSQLDisconnect(s)
	case *ast.SQLConnectionsStmt:
		return e.execSQLConnections()
	case *ast.SQLQueryStmt:
		return e.execSQLQuery(s)
	case *ast.SQLShowTablesStmt:
		return e.execSQLShowTables(s)
	case *ast.SQLShowViewsStmt:
		return e.execSQLShowViews(s)
	case *ast.SQLShowFunctionsStmt:
		return e.execSQLShowFunctions(s)
	case *ast.SQLDescribeTableStmt:
		return e.execSQLDescribeTable(s)
	case *ast.SQLGenerateConnectorStmt:
		return e.execSQLGenerateConnector(s)

	// Import statements
	case *ast.ImportStmt:
		return e.execImport(s)

	default:
		return fmt.Errorf("unknown statement type: %T", stmt)
	}
}

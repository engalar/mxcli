// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// ────────────────────────────────────────────────────────────────────
// Bridge functions: *HandlerDeps → *ExecContext adapters.
//
// These bridge functions create a temporary *ExecContext via NewExecContext
// and delegate to the old *ExecContext-based implementation. They exist so
// that registry files (handler_deps.go, show_registry.go, describe_registry.go)
// no longer call NewExecContext directly, concentrating the bridge in a single
// package. Once every underlying function is fully migrated to *HandlerDeps,
// this entire file can be deleted.
// ────────────────────────────────────────────────────────────────────

// ── Entities ──

func ExecCreateEntityDeps(ctx context.Context, s *ast.CreateEntityStmt, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return ExecCreateEntity(ectx, s)
}

func ExecDropEntityDeps(ctx context.Context, s *ast.DropEntityStmt, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return ExecDropEntity(ectx, s)
}

func ExecCreateViewEntityDeps(ctx context.Context, s *ast.CreateViewEntityStmt, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return ExecCreateViewEntity(ectx, s)
}

func listEntitiesGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listEntitiesGen(ectx, moduleName)
}

func listEntityDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return listEntity(ectx, name)
}

func describeEntityGenDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeEntityGen(ectx, name)
}

// ── Associations ──

func ExecCreateAssociationDeps(ctx context.Context, s *ast.CreateAssociationStmt, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return ExecCreateAssociation(ectx, s)
}

func ExecAlterAssociationGenDeps(ctx context.Context, s *ast.AlterAssociationStmt, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return ExecAlterAssociationGen(ectx, s)
}

func ExecDropAssociationGenDeps(ctx context.Context, s *ast.DropAssociationStmt, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return ExecDropAssociationGen(ectx, s)
}

func listAssociationsDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listAssociations(ectx, moduleName)
}

func listAssociationDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return listAssociation(ectx, name)
}

func describeAssociationDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeAssociation(ectx, name)
}

// ── Microflows / Nanoflows ──

func listMicroflowsDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listMicroflows(ectx, moduleName)
}

func listNanoflowsDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listNanoflows(ectx, moduleName)
}

func describeMicroflowGenDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeMicroflowGen(ectx, name)
}

func describeNanoflowGenDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeNanoflowGen(ectx, name)
}

// ── Pages / Snippets / Layouts ──

func ExecDropPageDeps(ctx context.Context, s *ast.DropPageStmt, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return ExecDropPage(ectx, s)
}

func ExecDropSnippetDeps(ctx context.Context, s *ast.DropSnippetStmt, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return ExecDropSnippet(ectx, s)
}

func listPagesGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listPagesGen(ectx, moduleName)
}

func listSnippetsGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listSnippetsGen(ectx, moduleName)
}

func listLayoutsGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listLayoutsGen(ectx, moduleName)
}

func describePageDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describePage(ectx, name)
}

func describeSnippetDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeSnippet(ectx, name)
}

func describeLayoutDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeLayout(ectx, name)
}

// ── Workflows ──

func ExecAlterWorkflowDeps(ctx context.Context, s *ast.AlterWorkflowStmt, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return ExecAlterWorkflow(ectx, s)
}

func listWorkflowsGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listWorkflowsGen(ectx, moduleName)
}

func describeWorkflowGenDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeWorkflowGen(ectx, name)
}

// ── Java / JavaScript Actions ──

func listJavaActionsGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listJavaActionsGen(ectx, moduleName)
}

func listJavaScriptActionsGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listJavaScriptActionsGen(ectx, moduleName)
}

func describeJavaActionGenDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeJavaActionGen(ectx, name)
}

func describeJavaScriptActionGenDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeJavaScriptActionGen(ectx, name)
}

// ── Modules ──

func listModulesDeps(ctx context.Context, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return listModules(ectx)
}

func describeModuleDeps(ctx context.Context, deps *HandlerDeps, moduleName string, withAll bool) error {
	ectx := NewExecContext(ctx, deps)
	return describeModule(ectx, moduleName, withAll)
}

func execListJarDependenciesDeps(ctx context.Context, deps *HandlerDeps, inModule string) error {
	ectx := NewExecContext(ctx, deps)
	return execListJarDependencies(ectx, inModule)
}

func execDescribeJarDependencyDeps(ctx context.Context, deps *HandlerDeps, moduleName, coordinate string) error {
	ectx := NewExecContext(ctx, deps)
	return execDescribeJarDependency(ectx, moduleName, coordinate)
}

// ── Catalog ──

func execShowCatalogTablesDeps(ctx context.Context, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return execShowCatalogTables(ectx)
}

func execShowCatalogStatusDeps(ctx context.Context, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return execShowCatalogStatus(ectx)
}

// ── Version ──

func listVersionDeps(ctx context.Context, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return listVersion(ectx)
}

// ── Context / Structure ──

func execShowContextDeps(ctx context.Context, deps *HandlerDeps, s *ast.ShowStmt) error {
	ectx := NewExecContext(ctx, deps)
	return execShowContext(ectx, s)
}

// ── Security ──

func listProjectSecurityGenDeps(ctx context.Context, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return listProjectSecurityGen(ectx)
}

func listModuleRolesGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listModuleRolesGen(ectx, moduleName)
}

func listUserRolesGenDeps(ctx context.Context, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return listUserRolesGen(ectx)
}

func listDemoUsersGenDeps(ctx context.Context, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return listDemoUsersGen(ectx)
}

func listAccessOnEntityGenDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return listAccessOnEntityGen(ectx, name)
}

func listAccessOnMicroflowGenDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return listAccessOnMicroflowGen(ectx, name)
}

func listAccessOnPageGenDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return listAccessOnPageGen(ectx, name)
}

func listAccessOnNanoflowGenDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return listAccessOnNanoflowGen(ectx, name)
}

func listSecurityMatrixGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listSecurityMatrixGen(ectx, moduleName)
}

func describeModuleRoleGenDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeModuleRoleGen(ectx, name)
}

func describeUserRoleGenDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeUserRoleGen(ectx, name)
}

func describeDemoUserGenDeps(ctx context.Context, deps *HandlerDeps, userName string) error {
	ectx := NewExecContext(ctx, deps)
	return describeDemoUserGen(ectx, userName)
}

// ── Enumerations ──

func ExecCreateEnumerationDeps(ctx context.Context, s *ast.CreateEnumerationStmt, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return ExecCreateEnumeration(ectx, s)
}

func ExecDropEnumerationDeps(ctx context.Context, s *ast.DropEnumerationStmt, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return ExecDropEnumeration(ectx, s)
}

func listEnumerationsDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listEnumerations(ectx, moduleName)
}

func describeEnumerationDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeEnumeration(ectx, name)
}

// ── Constants ──

func ExecCreateConstantDeps(ctx context.Context, stmt *ast.CreateConstantStmt, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return ExecCreateConstant(ectx, stmt)
}

func ExecDropConstantDeps(ctx context.Context, stmt *ast.DropConstantStmt, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return ExecDropConstant(ectx, stmt)
}

func listConstantsDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listConstants(ectx, moduleName)
}

func listConstantValuesDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listConstantValues(ectx, moduleName)
}

func describeConstantDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeConstant(ectx, name)
}

// ── Languages / Settings ──

func AlterLanguageDeps(ctx context.Context, stmt *ast.AlterLanguageStmt, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return AlterLanguage(ectx, stmt)
}

func listLanguagesDeps(ctx context.Context, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return listLanguages(ectx)
}

func listSupportedLanguagesDeps(ctx context.Context, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return listSupportedLanguages(ectx)
}

func listSettingsDeps(ctx context.Context, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return listSettings(ectx)
}

func describeSettingsDeps(ctx context.Context, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return describeSettings(ectx)
}

// ── Navigation ──

func listNavigationDeps(ctx context.Context, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return listNavigation(ectx)
}

func listNavigationMenuDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return listNavigationMenu(ectx, name)
}

func listNavigationHomesDeps(ctx context.Context, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return listNavigationHomes(ectx)
}

func describeNavigationDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeNavigation(ectx, name)
}

// ── OData ──

// Note: listODataClientsDeps, listODataServicesDeps, listExternalEntitiesDeps,
// listExternalActionsDeps, describeODataClientDeps, describeODataServiceDeps,
// and describeExternalEntityDeps are defined in cmd_odata.go with additional
// format parameter. Do NOT redeclare here.

// ── Business Events ──

func listBusinessEventServicesDeps(ctx context.Context, deps *HandlerDeps, inModule string) error {
	ectx := NewExecContext(ctx, deps)
	return listBusinessEventServices(ectx, inModule)
}

func listBusinessEventClientsDeps(ctx context.Context, deps *HandlerDeps, inModule string) error {
	ectx := NewExecContext(ctx, deps)
	return listBusinessEventClients(ectx, inModule)
}

func listBusinessEventsDeps(ctx context.Context, deps *HandlerDeps, inModule string) error {
	ectx := NewExecContext(ctx, deps)
	return listBusinessEvents(ectx, inModule)
}

func describeBusinessEventServiceDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeBusinessEventService(ectx, name)
}

// ── Fragments ──

func listFragmentsDeps(ctx context.Context, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return listFragments(ectx)
}

func describeFragmentDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeFragment(ectx, name)
}

// ── Database Connections ──

func ExecCreateDatabaseConnectionDeps(ctx context.Context, stmt *ast.CreateDatabaseConnectionStmt, deps *HandlerDeps) error {
	ectx := NewExecContext(ctx, deps)
	return ExecCreateDatabaseConnection(ectx, stmt)
}

func listDatabaseConnectionsDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listDatabaseConnections(ectx, moduleName)
}

func describeDatabaseConnectionDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeDatabaseConnection(ectx, name)
}

// ── Image Collections ──

func listImageCollectionsDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listImageCollections(ectx, moduleName)
}

func describeImageCollectionDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeImageCollection(ectx, name)
}

// ── Agent Editor ──

func listAgentEditorModelsDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listAgentEditorModels(ectx, moduleName)
}

func listAgentEditorAgentsDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listAgentEditorAgents(ectx, moduleName)
}

func listAgentEditorKnowledgeBasesDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listAgentEditorKnowledgeBases(ectx, moduleName)
}

func listAgentEditorConsumedMCPServicesDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listAgentEditorConsumedMCPServices(ectx, moduleName)
}

func describeAgentEditorModelDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeAgentEditorModel(ectx, name)
}

func describeAgentEditorAgentDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeAgentEditorAgent(ectx, name)
}

func describeAgentEditorKnowledgeBaseDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeAgentEditorKnowledgeBase(ectx, name)
}

func describeAgentEditorConsumedMCPServiceDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeAgentEditorConsumedMCPService(ectx, name)
}

// ── REST / External Services ──

func listRestClientsDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listRestClients(ectx, moduleName)
}

func listPublishedRestServicesDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listPublishedRestServices(ectx, moduleName)
}

func listDataTransformersDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listDataTransformers(ectx, moduleName)
}

func describeRestClientDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeRestClient(ectx, name)
}

func describePublishedRestServiceDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describePublishedRestService(ectx, name)
}

func describeDataTransformerDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeDataTransformer(ectx, name)
}

// ── Contracts ──

func listContractEntitiesDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return listContractEntities(ectx, name)
}

func listContractActionsDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return listContractActions(ectx, name)
}

func listContractChannelsDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return listContractChannels(ectx, name)
}

func listContractMessagesDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return listContractMessages(ectx, name)
}

func describeContractEntityDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName, format string) error {
	ectx := NewExecContext(ctx, deps)
	return describeContractEntity(ectx, name, format)
}

func describeContractActionDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName, format string) error {
	ectx := NewExecContext(ctx, deps)
	return describeContractAction(ectx, name, format)
}

func describeContractMessageDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeContractMessage(ectx, name)
}

// ── JSON Structures ──

func listJsonStructuresDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	ectx := NewExecContext(ctx, deps)
	return listJsonStructures(ectx, moduleName)
}

func describeJsonStructureDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeJsonStructure(ectx, name)
}

// ── Import / Export Mappings ──

func listImportMappingsDeps(ctx context.Context, deps *HandlerDeps, inModule string) error {
	ectx := NewExecContext(ctx, deps)
	return listImportMappings(ectx, inModule)
}

func listExportMappingsDeps(ctx context.Context, deps *HandlerDeps, inModule string) error {
	ectx := NewExecContext(ctx, deps)
	return listExportMappings(ectx, inModule)
}

func describeImportMappingDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeImportMapping(ectx, name)
}

func describeExportMappingDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeExportMapping(ectx, name)
}

// ── Widgets ──

func describeWidgetDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	ectx := NewExecContext(ctx, deps)
	return describeWidget(ectx, name)
}

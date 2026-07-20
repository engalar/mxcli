// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

// ────────────────────────────────────────────────────────────────────
// Bridge functions: *HandlerDeps → *ExecContext adapters.
//
// Wherever a Future implementation exists, the bridge bypasses
// NewExecContext entirely and delegates directly. Functions without
// Future counterparts still use the ExecContext bridge.
// Once every underlying function is fully migrated to *HandlerDeps,
// this entire file can be deleted.
// ────────────────────────────────────────────────────────────────────

// ── Entities ──

func ExecCreateEntityDeps(ctx context.Context, s *ast.CreateEntityStmt, deps *HandlerDeps) error {
	return execCreateEntityGenDeps(ctx, s, deps)
}

func ExecDropEntityDeps(ctx context.Context, s *ast.DropEntityStmt, deps *HandlerDeps) error {
	return execDropEntityGenDeps(ctx, s, deps)
}

func ExecCreateViewEntityDeps(ctx context.Context, s *ast.CreateViewEntityStmt, deps *HandlerDeps) error {
	return execCreateViewEntityGenDeps(ctx, s, deps)
}

func listEntitiesGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listEntitiesGenFuture(ctx, deps.Output, deps.Format, deps.ModuleLister, deps.DomainModels, moduleName)
}

func listEntityDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	return listEntityFuture(ctx, deps.Output, deps.ModuleLister, deps.DomainModels, name)
}

func describeEntityGenDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeEntityGenFuture(ctx, deps.Output, deps.ModuleLister, deps.DomainModels, deps.Security, name)
}

// ── Associations ──

func ExecCreateAssociationDeps(ctx context.Context, s *ast.CreateAssociationStmt, deps *HandlerDeps) error {
	return execCreateAssociationDepsImpl(ctx, s, deps)
}

func ExecAlterAssociationGenDeps(ctx context.Context, s *ast.AlterAssociationStmt, deps *HandlerDeps) error {
	return execAlterAssociationGenDeps(ctx, s, deps)
}

func ExecDropAssociationGenDeps(ctx context.Context, s *ast.DropAssociationStmt, deps *HandlerDeps) error {
	return execDropAssociationGenDeps(ctx, s, deps)
}

func listAssociationsDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listAssociationsFuture(ctx, deps.Output, deps.Format, deps.ModuleLister, deps.DomainModels, moduleName)
}

func listAssociationDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	return listAssociationFuture(ctx, deps.Output, deps.ModuleLister, deps.DomainModelReader, name)
}

func describeAssociationDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeAssociationFuture(ctx, deps.Output, deps.ModuleLister, deps.DomainModelReader, name)
}

// ── Microflows / Nanoflows ──

func listMicroflowsDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listMicroflowsFuture(ctx, deps.Output, deps.Format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.MicroflowRepo, moduleName)
}

func listNanoflowsDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listNanoflowsFuture(ctx, deps.Output, deps.Format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.NanoflowRepo, moduleName)
}

func describeMicroflowGenDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeMicroflowGenFuture(ctx, deps.Output, deps.MicroflowRepo, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, name)
}

func describeNanoflowGenDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeNanoflowGenFuture(ctx, deps.Output, deps.NanoflowRepo, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, name)
}

// ── Pages / Snippets / Layouts ──

func ExecDropPageDeps(ctx context.Context, s *ast.DropPageStmt, deps *HandlerDeps) error {
	return execDropPageDepsImpl(ctx, s, deps)
}

func ExecDropSnippetDeps(ctx context.Context, s *ast.DropSnippetStmt, deps *HandlerDeps) error {
	return execDropSnippetDepsImpl(ctx, s, deps)
}

func listPagesGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listPagesFuture(ctx, deps.Output, deps.Format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.PageRepo, moduleName)
}

func listSnippetsGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listSnippetsFuture(ctx, deps.Output, deps.Format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.SnippetRepo, moduleName)
}

func listLayoutsGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listLayoutsFuture(ctx, deps.Output, deps.Format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.LayoutRepo, moduleName)
}

func describePageDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describePageFuture(ctx, deps.Output, deps.PageRepo, deps.ImageBackend, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, name)
}

func describeSnippetDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeSnippetFuture(ctx, deps.Output, deps.SnippetRepo, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, name)
}

func describeLayoutDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeLayoutFuture(ctx, deps.Output, deps.LayoutRepo, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, name)
}

// ── Workflows ──

func ExecAlterWorkflowDeps(ctx context.Context, s *ast.AlterWorkflowStmt, deps *HandlerDeps) error {
	return execAlterWorkflowDepsImpl(ctx, s, deps)
}

func listWorkflowsGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listWorkflowsFuture(ctx, deps.Output, deps.Format, deps.ConnectionManager, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.WorkflowRepo, moduleName)
}

func describeWorkflowGenDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeWorkflowGenFuture(ctx, deps.Output, deps.WorkflowRepo, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, name)
}

// ── Java / JavaScript Actions ──

func listJavaActionsGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listJavaActionsFuture(ctx, deps.Output, deps.Format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.JavaActionRepo, moduleName)
}

func listJavaScriptActionsGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listJavaScriptActionsFuture(ctx, deps.Output, deps.Format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.JavaScriptActionRepo, moduleName)
}

func describeJavaActionGenDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeJavaActionGenFuture(ctx, deps.Output, deps.JavaActionRepo, name)
}

func describeJavaScriptActionGenDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeJavaScriptActionGenFn(ctx, name, deps)
}

// ── Modules ──

func listModulesDeps(ctx context.Context, deps *HandlerDeps) error {
	return listModulesFuture(ctx, deps.Output, deps.Format, deps.ConnectionManager, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.DomainModels)
}

func describeModuleDeps(ctx context.Context, deps *HandlerDeps, moduleName string, withAll bool) error {
	return describeModuleFuture(ctx, deps.Output,
		moduleName, withAll,
		deps.ModuleLister, deps.MetadataReader, deps.FolderManager,
		deps.EnumerationReader, deps.ConstantReader,
		deps.DomainModels, deps.Security,
		deps.MicroflowRepo, deps.NanoflowRepo,
		deps.PageRepo, deps.SnippetRepo, deps.LayoutRepo, deps.WorkflowRepo,
		deps.ImageBackend, deps.NavigationReader)
}

// execListJarDependenciesDeps and execDescribeJarDependencyDeps are
// defined in cmd_modules.go.

// ── Catalog ──

func execShowCatalogTablesDeps(ctx context.Context, deps *HandlerDeps) error {
	return execShowCatalogTablesFuture(ctx, deps.Output)
}

func execShowCatalogStatusDeps(ctx context.Context, deps *HandlerDeps) error {
	return execShowCatalogStatusFuture(ctx, deps.Output)
}

// ── Version ──

func listVersionDeps(ctx context.Context, deps *HandlerDeps) error {
	return listVersionFuture(ctx, deps.Output, deps.ConnectionManager)
}

// ── Context / Structure ──

func execShowContextDeps(ctx context.Context, deps *HandlerDeps, s *ast.ShowStmt) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	return mdlerrors.NewUnsupported(
		"SHOW CONTEXT OF has been replaced by MXGraph-based commands.\n" +
			"Use SHOW CALLERS OF / SHOW CALLEES OF / SHOW REFERENCES TO / SHOW IMPACT OF instead.")
}

// ── Security ──

func listProjectSecurityGenDeps(ctx context.Context, deps *HandlerDeps) error {
	return listProjectSecurityFuture(ctx, deps.Output, deps.Format, deps.Security)
}

func listModuleRolesGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listModuleRolesFuture(ctx, deps.Output, deps.Format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.Security, moduleName)
}

func listUserRolesGenDeps(ctx context.Context, deps *HandlerDeps) error {
	return listUserRolesFuture(ctx, deps.Output, deps.Format, deps.Security)
}

func listDemoUsersGenDeps(ctx context.Context, deps *HandlerDeps) error {
	return listDemoUsersFuture(ctx, deps.Output, deps.Format, deps.Security)
}

func listAccessOnEntityGenDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	return listAccessOnEntityFuture(ctx, deps.Output, deps.Format, deps.ModuleLister, deps.DomainModels, name)
}

func listAccessOnMicroflowGenDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	return listAccessOnMicroflowFuture(ctx, deps.Output, deps.Format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.MicroflowRepo, name)
}

func listAccessOnPageGenDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	return listAccessOnPageFuture(ctx, deps.Output, deps.Format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.PageRepo, name)
}

func listAccessOnNanoflowGenDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	return listAccessOnNanoflowFuture(ctx, deps.Output, deps.Format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.NanoflowRepo, name)
}

func listSecurityMatrixGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listSecurityMatrixFuture(ctx, deps.Output, deps.Format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.Security, deps.DomainModels, deps.MicroflowRepo, deps.PageRepo, moduleName)
}

func describeModuleRoleGenDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeModuleRoleGenFuture(ctx, deps.Output, deps.Security, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, name)
}

func describeUserRoleGenDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeUserRoleGenFuture(ctx, deps.Output, deps.Security, name)
}

func describeDemoUserGenDeps(ctx context.Context, deps *HandlerDeps, userName string) error {
	return describeDemoUserGenFuture(ctx, deps.Output, deps.Security, userName)
}

// ── Enumerations ──

func ExecCreateEnumerationDeps(ctx context.Context, s *ast.CreateEnumerationStmt, deps *HandlerDeps) error {
	return execCreateEnumerationDepsImpl(ctx, s, deps)
}

func ExecDropEnumerationDeps(ctx context.Context, s *ast.DropEnumerationStmt, deps *HandlerDeps) error {
	return execDropEnumerationDepsImpl(ctx, s, deps)
}

func listEnumerationsDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listEnumerationsFuture(ctx, deps.Output, deps.Format, deps.ConnectionManager, deps.EnumerationReader, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, moduleName)
}

func describeEnumerationDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeEnumerationFuture(ctx, deps.Output, deps.EnumerationReader, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, name)
}

// ── Constants ──

func ExecCreateConstantDeps(ctx context.Context, stmt *ast.CreateConstantStmt, deps *HandlerDeps) error {
	return execCreateConstantDepsImpl(ctx, stmt, deps)
}

func ExecDropConstantDeps(ctx context.Context, stmt *ast.DropConstantStmt, deps *HandlerDeps) error {
	return execDropConstantDepsImpl(ctx, stmt, deps)
}

func listConstantsDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listConstantsFuture(ctx, deps.Output, deps.Format, deps.ConnectionManager, deps.ConstantReader, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, moduleName)
}

func listConstantValuesDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listConstantValuesFuture(ctx, deps.Output, deps.Format, deps.ConnectionManager, deps.ConstantReader, deps.SettingsReader, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, moduleName)
}

func describeConstantDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeConstantFuture(ctx, deps.Output, deps.ConstantReader, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, name)
}

// ── Languages / Settings ──

func AlterLanguageDeps(ctx context.Context, stmt *ast.AlterLanguageStmt, deps *HandlerDeps) error {
	return alterLanguageDepsImpl(ctx, stmt, deps)
}

func listLanguagesDeps(ctx context.Context, deps *HandlerDeps) error {
	return listLanguagesFuture(ctx, deps.Output, deps.Format, deps.SettingsReader)
}

func listSupportedLanguagesDeps(ctx context.Context, deps *HandlerDeps) error {
	return listSupportedLanguagesFuture(ctx, deps.Output, deps.Format)
}

func listSettingsDeps(ctx context.Context, deps *HandlerDeps) error {
	return listSettingsFuture(ctx, deps.Output, deps.Format, deps.SettingsReader)
}

func describeSettingsDeps(ctx context.Context, deps *HandlerDeps) error {
	return describeSettingsFuture(ctx, deps.Output, deps.ConnectionManager, deps.SettingsReader)
}

// ── Navigation ──

func listNavigationDeps(ctx context.Context, deps *HandlerDeps) error {
	return listNavigationFuture(ctx, deps.Output, deps.NavigationReader)
}

func listNavigationMenuDeps(ctx context.Context, deps *HandlerDeps, name *ast.QualifiedName) error {
	return listNavigationMenuFuture(ctx, deps.Output, deps.NavigationReader, name)
}

func listNavigationHomesDeps(ctx context.Context, deps *HandlerDeps) error {
	return listNavigationHomesFuture(ctx, deps.Output, deps.NavigationReader)
}

func describeNavigationDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeNavigationFuture(ctx, deps.Output, deps.NavigationReader, name)
}

// ── OData ──

// Note: listODataClientsDeps, listODataServicesDeps, listExternalEntitiesDeps,
// listExternalActionsDeps, describeODataClientDeps, describeODataServiceDeps,
// and describeExternalEntityDeps are defined in cmd_odata.go with additional
// format parameter. Do NOT redeclare here.

// ── Business Events ──

func listBusinessEventServicesDeps(ctx context.Context, deps *HandlerDeps, inModule string) error {
	return listBusinessEventServicesFuture(ctx, deps.Output, deps.Format, deps.ConnectionManager, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.BusinessEventBackend, inModule)
}

func listBusinessEventClientsDeps(ctx context.Context, deps *HandlerDeps, inModule string) error {
	return listBusinessEventClientsFuture(ctx, deps.Output)
}

func listBusinessEventsDeps(ctx context.Context, deps *HandlerDeps, inModule string) error {
	return listBusinessEventsFuture(ctx, deps.Output, deps.Format, deps.ConnectionManager, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.BusinessEventBackend, inModule)
}

// ── Fragments ──

func listFragmentsDeps(ctx context.Context, deps *HandlerDeps) error {
	return listFragmentsFuture(ctx, deps.Output, deps.Fragments)
}

func describeFragmentDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeFragmentFuture(ctx, deps.Output, deps.Fragments, name)
}

// ── Database Connections ──

func ExecCreateDatabaseConnectionDeps(ctx context.Context, stmt *ast.CreateDatabaseConnectionStmt, deps *HandlerDeps) error {
	return execCreateDatabaseConnectionDepsImpl(ctx, stmt, deps)
}

func describeDatabaseConnectionDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeDatabaseConnectionFuture(ctx, deps.Output, deps.ServiceLister, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, name)
}

// ── Image Collections ──

func listImageCollectionsDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listImageCollectionsFn(ctx, deps.Output, deps.Format, deps, moduleName)
}

func describeImageCollectionDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeImageCollectionFuture(ctx, deps.Output, deps.ImageBackend, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, name)
}

// ── REST / External Services ──

func listPublishedRestServicesDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listPublishedRestServicesFn(ctx, deps.Output, deps.Format, deps, moduleName)
}

func listDataTransformersDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listDataTransformersFn(ctx, moduleName, deps)
}

func describePublishedRestServiceDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describePublishedRestServiceFn(ctx, deps.Output, deps, name)
}

func describeDataTransformerDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeDataTransformerFn(ctx, name, deps)
}

// ── JSON Structures ──

func listJsonStructuresDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listJsonStructuresFn(ctx, deps, moduleName)
}

func describeJsonStructureDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeJsonStructureFn(ctx, deps, name)
}

// ── Import / Export Mappings ──

func listImportMappingsDeps(ctx context.Context, deps *HandlerDeps, inModule string) error {
	return listImportMappingsFn(ctx, inModule, deps)
}

func listExportMappingsDeps(ctx context.Context, deps *HandlerDeps, inModule string) error {
	return listExportMappingsFn(ctx, inModule, deps)
}

func describeImportMappingDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeImportMappingFn(ctx, name, deps)
}

func describeExportMappingDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeExportMappingFn(ctx, name, deps)
}

// ── Widgets ──

func describeWidgetDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeWidgetFn(ctx, deps.Output, deps, name)
}

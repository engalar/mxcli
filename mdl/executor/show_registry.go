// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// ShowHandler is the function signature for all SHOW statement handlers.
type ShowHandler func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error

// showHandlers maps each ShowObjectType to its handler.
// Adding a new SHOW command only requires appending one entry here.
//
// ast.ShowWidgets is intentionally absent: it is dispatched via the dedicated
// ShowWidgetsStmt statement type, not through ExecShow.
var showHandlers = map[ast.ShowObjectType]ShowHandler{
	ast.ShowModules: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listModulesDeps(ctx, deps)
	},
	ast.ShowEnumerations: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listEnumerationsDeps(ctx, deps, s.InModule)
	},
	ast.ShowConstants: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listConstantsDeps(ctx, deps, s.InModule)
	},
	ast.ShowConstantValues: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listConstantValuesDeps(ctx, deps, s.InModule)
	},
	ast.ShowEntities: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listEntitiesGenDeps(ctx, deps, s.InModule)
	},
	ast.ShowEntity: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listEntityDeps(ctx, deps, s.Name)
	},
	ast.ShowAssociations: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAssociationsDeps(ctx, deps, s.InModule)
	},
	ast.ShowAssociation: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAssociationDeps(ctx, deps, s.Name)
	},
	ast.ShowMicroflows: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listMicroflowsDeps(ctx, deps, s.InModule)
	},
	ast.ShowNanoflows: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listNanoflowsDeps(ctx, deps, s.InModule)
	},
	ast.ShowPages: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listPagesGenDeps(ctx, deps, s.InModule)
	},
	ast.ShowSnippets: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listSnippetsGenDeps(ctx, deps, s.InModule)
	},
	ast.ShowLayouts: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listLayoutsGenDeps(ctx, deps, s.InModule)
	},
	ast.ShowJavaActions: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listJavaActionsGenDeps(ctx, deps, s.InModule)
	},
	ast.ShowJavaScriptActions: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listJavaScriptActionsGenDeps(ctx, deps, s.InModule)
	},
	ast.ShowVersion: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listVersionDeps(ctx, deps)
	},
	ast.ShowCatalogTables: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return execShowCatalogTablesDeps(ctx, deps)
	},
	ast.ShowCatalogStatus: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return execShowCatalogStatusDeps(ctx, deps)
	},
	ast.ShowCallers: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return ExecShowCallersFn(ctx, s, deps)
	},
	ast.ShowCallees: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return ExecShowCalleesFn(ctx, s, deps)
	},
	ast.ShowReferences: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return ExecShowReferencesFn(ctx, s, deps)
	},
	ast.ShowImpact: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return ExecShowImpactFn(ctx, s, deps)
	},
	ast.ShowContext: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return execShowContextDeps(ctx, deps, s)
	},
	ast.ShowProjectSecurity: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listProjectSecurityGenDeps(ctx, deps)
	},
	ast.ShowModuleRoles: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listModuleRolesGenDeps(ctx, deps, s.InModule)
	},
	ast.ShowUserRoles: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listUserRolesGenDeps(ctx, deps)
	},
	ast.ShowDemoUsers: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listDemoUsersGenDeps(ctx, deps)
	},
	ast.ShowAccessOn: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAccessOnEntityGenDeps(ctx, deps, s.Name)
	},
	ast.ShowAccessOnMicroflow: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAccessOnMicroflowGenDeps(ctx, deps, s.Name)
	},
	ast.ShowAccessOnPage: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAccessOnPageGenDeps(ctx, deps, s.Name)
	},
	ast.ShowAccessOnWorkflow: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAccessOnWorkflowDeps(ctx, deps, s.Name)
	},
	ast.ShowAccessOnNanoflow: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAccessOnNanoflowGenDeps(ctx, deps, s.Name)
	},
	ast.ShowSecurityMatrix: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listSecurityMatrixGenDeps(ctx, deps, s.InModule)
	},
	ast.ShowODataClients: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listODataClientsDeps(ctx, deps, deps.Format, s.InModule)
	},
	ast.ShowODataServices: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listODataServicesDeps(ctx, deps, deps.Format, s.InModule)
	},
	ast.ShowExternalEntities: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listExternalEntitiesDeps(ctx, deps, deps.Format, s.InModule)
	},
	ast.ShowExternalActions: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listExternalActionsDeps(ctx, deps, deps.Format, s.InModule)
	},
	ast.ShowNavigation: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listNavigationDeps(ctx, deps)
	},
	ast.ShowNavigationMenu: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listNavigationMenuDeps(ctx, deps, s.Name)
	},
	ast.ShowNavigationHomes: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listNavigationHomesDeps(ctx, deps)
	},
	ast.ShowStructure: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return execShowStructureGenFn(ctx, s, deps)
	},
	ast.ShowWorkflows: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listWorkflowsGenDeps(ctx, deps, s.InModule)
	},
	ast.ShowBusinessEventServices: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listBusinessEventServicesDeps(ctx, deps, s.InModule)
	},
	ast.ShowBusinessEventClients: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listBusinessEventClientsDeps(ctx, deps, s.InModule)
	},
	ast.ShowBusinessEvents: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listBusinessEventsDeps(ctx, deps, s.InModule)
	},
	ast.ShowSettings: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listSettingsDeps(ctx, deps)
	},
	ast.ShowLanguages: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listLanguagesDeps(ctx, deps)
	},
	ast.ShowSupportedLanguages: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listSupportedLanguagesDeps(ctx, deps)
	},
	ast.ShowFragments: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listFragmentsDeps(ctx, deps)
	},
	ast.ShowDatabaseConnections: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listDatabaseConnectionsDeps(ctx, deps, s.InModule)
	},
	ast.ShowImageCollections: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listImageCollectionsDeps(ctx, deps, s.InModule)
	},
	ast.ShowModels: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAgentEditorModelsDeps(ctx, deps, s.InModule)
	},
	ast.ShowAgents: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAgentEditorAgentsDeps(ctx, deps, s.InModule)
	},
	ast.ShowKnowledgeBases: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAgentEditorKnowledgeBasesDeps(ctx, deps, s.InModule)
	},
	ast.ShowConsumedMCPServices: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAgentEditorConsumedMCPServicesDeps(ctx, deps, s.InModule)
	},
	ast.ShowRestClients: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listRestClientsDeps(ctx, deps, s.InModule)
	},
	ast.ShowPublishedRestServices: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listPublishedRestServicesDeps(ctx, deps, s.InModule)
	},
	ast.ShowDataTransformers: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listDataTransformersDeps(ctx, deps, s.InModule)
	},
	ast.ShowContractEntities: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listContractEntitiesDeps(ctx, deps, s.Name)
	},
	ast.ShowContractActions: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listContractActionsDeps(ctx, deps, s.Name)
	},
	ast.ShowContractChannels: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listContractChannelsDeps(ctx, deps, s.Name)
	},
	ast.ShowContractMessages: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listContractMessagesDeps(ctx, deps, s.Name)
	},
	ast.ShowJsonStructures: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listJsonStructuresDeps(ctx, deps, s.InModule)
	},
	ast.ShowImportMappings: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listImportMappingsDeps(ctx, deps, s.InModule)
	},
	ast.ShowExportMappings: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listExportMappingsDeps(ctx, deps, s.InModule)
	},
	ast.ShowJarDependencies: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return execListJarDependenciesDeps(ctx, deps, s.InModule)
	},
}

// ShowHandlers returns the show handler map (exported for tests).
func ShowHandlers() map[ast.ShowObjectType]ShowHandler {
	return showHandlers
}

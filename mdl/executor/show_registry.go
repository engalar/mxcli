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
// ShowWidgetsStmt statement type, not through execShow.
var showHandlers = map[ast.ShowObjectType]ShowHandler{
	ast.ShowModules: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listModules(phase3d2bNewExecContext(ctx, deps))
	},
	ast.ShowEnumerations: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listEnumerations(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowConstants: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listConstants(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowConstantValues: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listConstantValues(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowEntities: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listEntitiesGen(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowEntity: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listEntity(phase3d2bNewExecContext(ctx, deps), s.Name)
	},
	ast.ShowAssociations: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAssociations(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowAssociation: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAssociation(phase3d2bNewExecContext(ctx, deps), s.Name)
	},
	ast.ShowMicroflows: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listMicroflows(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowNanoflows: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listNanoflows(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowPages: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listPagesGen(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowSnippets: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listSnippetsGen(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowLayouts: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listLayoutsGen(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowJavaActions: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listJavaActionsGen(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowJavaScriptActions: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listJavaScriptActionsGen(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowVersion: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listVersion(phase3d2bNewExecContext(ctx, deps))
	},
	ast.ShowCatalogTables: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return execShowCatalogTables(phase3d2bNewExecContext(ctx, deps))
	},
	ast.ShowCatalogStatus: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return execShowCatalogStatus(phase3d2bNewExecContext(ctx, deps))
	},
	ast.ShowCallers: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return execShowCallersFn(ctx, s, deps)
	},
	ast.ShowCallees: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return execShowCalleesFn(ctx, s, deps)
	},
	ast.ShowReferences: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return execShowReferencesFn(ctx, s, deps)
	},
	ast.ShowImpact: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return execShowImpactFn(ctx, s, deps)
	},
	ast.ShowContext: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return execShowContext(phase3d2bNewExecContext(ctx, deps), s)
	},
	ast.ShowProjectSecurity: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listProjectSecurityGen(phase3d2bNewExecContext(ctx, deps))
	},
	ast.ShowModuleRoles: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listModuleRolesGen(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowUserRoles: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listUserRolesGen(phase3d2bNewExecContext(ctx, deps))
	},
	ast.ShowDemoUsers: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listDemoUsersGen(phase3d2bNewExecContext(ctx, deps))
	},
	ast.ShowAccessOn: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAccessOnEntityGen(phase3d2bNewExecContext(ctx, deps), s.Name)
	},
	ast.ShowAccessOnMicroflow: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAccessOnMicroflowGen(phase3d2bNewExecContext(ctx, deps), s.Name)
	},
	ast.ShowAccessOnPage: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAccessOnPageGen(phase3d2bNewExecContext(ctx, deps), s.Name)
	},
	ast.ShowAccessOnWorkflow: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAccessOnWorkflow(phase3d2bNewExecContext(ctx, deps), s.Name)
	},
	ast.ShowAccessOnNanoflow: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAccessOnNanoflowGen(phase3d2bNewExecContext(ctx, deps), s.Name)
	},
	ast.ShowSecurityMatrix: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listSecurityMatrixGen(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowODataClients: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listODataClients(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowODataServices: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listODataServices(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowExternalEntities: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listExternalEntities(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowExternalActions: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listExternalActions(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowNavigation: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listNavigation(phase3d2bNewExecContext(ctx, deps))
	},
	ast.ShowNavigationMenu: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listNavigationMenu(phase3d2bNewExecContext(ctx, deps), s.Name)
	},
	ast.ShowNavigationHomes: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listNavigationHomes(phase3d2bNewExecContext(ctx, deps))
	},
	ast.ShowStructure: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return execShowStructureGenFn(ctx, s, deps)
	},
	ast.ShowWorkflows: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listWorkflowsGen(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowBusinessEventServices: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listBusinessEventServices(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowBusinessEventClients: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listBusinessEventClients(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowBusinessEvents: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listBusinessEvents(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowSettings: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listSettings(phase3d2bNewExecContext(ctx, deps))
	},
	ast.ShowLanguages: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listLanguages(phase3d2bNewExecContext(ctx, deps))
	},
	ast.ShowSupportedLanguages: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listSupportedLanguages(phase3d2bNewExecContext(ctx, deps))
	},
	ast.ShowFragments: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listFragments(phase3d2bNewExecContext(ctx, deps))
	},
	ast.ShowDatabaseConnections: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listDatabaseConnections(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowImageCollections: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listImageCollections(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowModels: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAgentEditorModels(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowAgents: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAgentEditorAgents(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowKnowledgeBases: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAgentEditorKnowledgeBases(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowConsumedMCPServices: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAgentEditorConsumedMCPServices(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowRestClients: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listRestClients(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowPublishedRestServices: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listPublishedRestServices(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowDataTransformers: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listDataTransformers(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowContractEntities: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listContractEntities(phase3d2bNewExecContext(ctx, deps), s.Name)
	},
	ast.ShowContractActions: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listContractActions(phase3d2bNewExecContext(ctx, deps), s.Name)
	},
	ast.ShowContractChannels: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listContractChannels(phase3d2bNewExecContext(ctx, deps), s.Name)
	},
	ast.ShowContractMessages: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listContractMessages(phase3d2bNewExecContext(ctx, deps), s.Name)
	},
	ast.ShowJsonStructures: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listJsonStructures(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowImportMappings: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listImportMappings(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowExportMappings: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listExportMappings(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowJarDependencies: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return execListJarDependencies(phase3d2bNewExecContext(ctx, deps), s.InModule)
	},
}

// ShowHandlers returns the show handler map (exported for tests).
func ShowHandlers() map[ast.ShowObjectType]ShowHandler {
	return showHandlers
}

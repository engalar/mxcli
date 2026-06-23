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
		return listModules(NewExecContext(ctx, deps))
	},
	ast.ShowEnumerations: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listEnumerations(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowConstants: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listConstants(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowConstantValues: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listConstantValues(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowEntities: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listEntitiesGen(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowEntity: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listEntity(NewExecContext(ctx, deps), s.Name)
	},
	ast.ShowAssociations: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAssociations(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowAssociation: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAssociation(NewExecContext(ctx, deps), s.Name)
	},
	ast.ShowMicroflows: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listMicroflows(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowNanoflows: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listNanoflows(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowPages: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listPagesGen(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowSnippets: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listSnippetsGen(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowLayouts: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listLayoutsGen(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowJavaActions: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listJavaActionsGen(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowJavaScriptActions: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listJavaScriptActionsGen(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowVersion: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listVersion(NewExecContext(ctx, deps))
	},
	ast.ShowCatalogTables: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return execShowCatalogTables(NewExecContext(ctx, deps))
	},
	ast.ShowCatalogStatus: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return execShowCatalogStatus(NewExecContext(ctx, deps))
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
		return execShowContext(NewExecContext(ctx, deps), s)
	},
	ast.ShowProjectSecurity: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listProjectSecurityGen(NewExecContext(ctx, deps))
	},
	ast.ShowModuleRoles: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listModuleRolesGen(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowUserRoles: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listUserRolesGen(NewExecContext(ctx, deps))
	},
	ast.ShowDemoUsers: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listDemoUsersGen(NewExecContext(ctx, deps))
	},
	ast.ShowAccessOn: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAccessOnEntityGen(NewExecContext(ctx, deps), s.Name)
	},
	ast.ShowAccessOnMicroflow: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAccessOnMicroflowGen(NewExecContext(ctx, deps), s.Name)
	},
	ast.ShowAccessOnPage: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAccessOnPageGen(NewExecContext(ctx, deps), s.Name)
	},
	ast.ShowAccessOnWorkflow: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAccessOnWorkflow(NewExecContext(ctx, deps), s.Name)
	},
	ast.ShowAccessOnNanoflow: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAccessOnNanoflowGen(NewExecContext(ctx, deps), s.Name)
	},
	ast.ShowSecurityMatrix: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listSecurityMatrixGen(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowODataClients: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listODataClients(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowODataServices: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listODataServices(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowExternalEntities: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listExternalEntities(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowExternalActions: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listExternalActions(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowNavigation: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listNavigation(NewExecContext(ctx, deps))
	},
	ast.ShowNavigationMenu: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listNavigationMenu(NewExecContext(ctx, deps), s.Name)
	},
	ast.ShowNavigationHomes: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listNavigationHomes(NewExecContext(ctx, deps))
	},
	ast.ShowStructure: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return execShowStructureGenFn(ctx, s, deps)
	},
	ast.ShowWorkflows: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listWorkflowsGen(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowBusinessEventServices: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listBusinessEventServices(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowBusinessEventClients: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listBusinessEventClients(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowBusinessEvents: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listBusinessEvents(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowSettings: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listSettings(NewExecContext(ctx, deps))
	},
	ast.ShowLanguages: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listLanguages(NewExecContext(ctx, deps))
	},
	ast.ShowSupportedLanguages: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listSupportedLanguages(NewExecContext(ctx, deps))
	},
	ast.ShowFragments: func(ctx context.Context, _ *ast.ShowStmt, deps *HandlerDeps) error {
		return listFragments(NewExecContext(ctx, deps))
	},
	ast.ShowDatabaseConnections: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listDatabaseConnections(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowImageCollections: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listImageCollections(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowModels: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAgentEditorModels(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowAgents: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAgentEditorAgents(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowKnowledgeBases: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAgentEditorKnowledgeBases(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowConsumedMCPServices: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listAgentEditorConsumedMCPServices(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowRestClients: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listRestClients(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowPublishedRestServices: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listPublishedRestServices(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowDataTransformers: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listDataTransformers(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowContractEntities: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listContractEntities(NewExecContext(ctx, deps), s.Name)
	},
	ast.ShowContractActions: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listContractActions(NewExecContext(ctx, deps), s.Name)
	},
	ast.ShowContractChannels: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listContractChannels(NewExecContext(ctx, deps), s.Name)
	},
	ast.ShowContractMessages: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listContractMessages(NewExecContext(ctx, deps), s.Name)
	},
	ast.ShowJsonStructures: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listJsonStructures(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowImportMappings: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listImportMappings(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowExportMappings: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return listExportMappings(NewExecContext(ctx, deps), s.InModule)
	},
	ast.ShowJarDependencies: func(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
		return execListJarDependencies(NewExecContext(ctx, deps), s.InModule)
	},
}

// ShowHandlers returns the show handler map (exported for tests).
func ShowHandlers() map[ast.ShowObjectType]ShowHandler {
	return showHandlers
}

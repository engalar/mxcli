// SPDX-License-Identifier: Apache-2.0

package executor

import "github.com/mendixlabs/mxcli/mdl/ast"

// ShowHandler is the function signature for all SHOW statement handlers
// dispatched through execShow.
type ShowHandler func(ctx *ExecContext, s *ast.ShowStmt) error

// showHandlers maps each ShowObjectType to its handler.
// Adding a new SHOW command only requires appending one entry here.
//
// ast.ShowWidgets is intentionally absent: it is dispatched via the dedicated
// ShowWidgetsStmt statement type (register_stubs.go), not through execShow.
var showHandlers = map[ast.ShowObjectType]ShowHandler{
	ast.ShowModules:           func(ctx *ExecContext, _ *ast.ShowStmt) error { return listModules(ctx) },
	ast.ShowEnumerations:      func(ctx *ExecContext, s *ast.ShowStmt) error { return listEnumerations(ctx, s.InModule) },
	ast.ShowConstants:         func(ctx *ExecContext, s *ast.ShowStmt) error { return listConstants(ctx, s.InModule) },
	ast.ShowConstantValues:    func(ctx *ExecContext, s *ast.ShowStmt) error { return listConstantValues(ctx, s.InModule) },
	ast.ShowEntities:          func(ctx *ExecContext, s *ast.ShowStmt) error { return listEntitiesGen(ctx, s.InModule) },
	ast.ShowEntity:            func(ctx *ExecContext, s *ast.ShowStmt) error { return listEntity(ctx, s.Name) },
	ast.ShowAssociations:      func(ctx *ExecContext, s *ast.ShowStmt) error { return listAssociations(ctx, s.InModule) },
	ast.ShowAssociation:       func(ctx *ExecContext, s *ast.ShowStmt) error { return listAssociation(ctx, s.Name) },
	ast.ShowMicroflows:        func(ctx *ExecContext, s *ast.ShowStmt) error { return listMicroflows(ctx, s.InModule) },
	ast.ShowNanoflows:         func(ctx *ExecContext, s *ast.ShowStmt) error { return listNanoflows(ctx, s.InModule) },
	ast.ShowPages:             func(ctx *ExecContext, s *ast.ShowStmt) error { return listPagesGen(ctx, s.InModule) },
	ast.ShowSnippets:          func(ctx *ExecContext, s *ast.ShowStmt) error { return listSnippetsGen(ctx, s.InModule) },
	ast.ShowLayouts:           func(ctx *ExecContext, s *ast.ShowStmt) error { return listLayoutsGen(ctx, s.InModule) },
	ast.ShowJavaActions:       func(ctx *ExecContext, s *ast.ShowStmt) error { return listJavaActionsGen(ctx, s.InModule) },
	ast.ShowJavaScriptActions: func(ctx *ExecContext, s *ast.ShowStmt) error { return listJavaScriptActionsGen(ctx, s.InModule) },
	ast.ShowVersion:           func(ctx *ExecContext, _ *ast.ShowStmt) error { return listVersion(ctx) },
	ast.ShowCatalogTables:     func(ctx *ExecContext, _ *ast.ShowStmt) error { return execShowCatalogTables(ctx) },
	ast.ShowCatalogStatus:     func(ctx *ExecContext, _ *ast.ShowStmt) error { return execShowCatalogStatus(ctx) },
	ast.ShowCallers:           func(ctx *ExecContext, s *ast.ShowStmt) error { return execShowCallers(ctx, s) },
	ast.ShowCallees:           func(ctx *ExecContext, s *ast.ShowStmt) error { return execShowCallees(ctx, s) },
	ast.ShowReferences:        func(ctx *ExecContext, s *ast.ShowStmt) error { return execShowReferences(ctx, s) },
	ast.ShowImpact:            func(ctx *ExecContext, s *ast.ShowStmt) error { return execShowImpact(ctx, s) },
	ast.ShowContext:           func(ctx *ExecContext, s *ast.ShowStmt) error { return execShowContext(ctx, s) },
	ast.ShowProjectSecurity:   func(ctx *ExecContext, _ *ast.ShowStmt) error { return listProjectSecurityGen(ctx) },
	ast.ShowModuleRoles:       func(ctx *ExecContext, s *ast.ShowStmt) error { return listModuleRolesGen(ctx, s.InModule) },
	ast.ShowUserRoles:         func(ctx *ExecContext, _ *ast.ShowStmt) error { return listUserRolesGen(ctx) },
	ast.ShowDemoUsers:         func(ctx *ExecContext, _ *ast.ShowStmt) error { return listDemoUsersGen(ctx) },
	ast.ShowAccessOn:          func(ctx *ExecContext, s *ast.ShowStmt) error { return listAccessOnEntityGen(ctx, s.Name) },
	ast.ShowAccessOnMicroflow: func(ctx *ExecContext, s *ast.ShowStmt) error { return listAccessOnMicroflowGen(ctx, s.Name) },
	ast.ShowAccessOnPage:      func(ctx *ExecContext, s *ast.ShowStmt) error { return listAccessOnPageGen(ctx, s.Name) },
	ast.ShowAccessOnWorkflow:  func(ctx *ExecContext, s *ast.ShowStmt) error { return listAccessOnWorkflow(ctx, s.Name) },
	ast.ShowAccessOnNanoflow:  func(ctx *ExecContext, s *ast.ShowStmt) error { return listAccessOnNanoflowGen(ctx, s.Name) },
	ast.ShowSecurityMatrix:    func(ctx *ExecContext, s *ast.ShowStmt) error { return listSecurityMatrixGen(ctx, s.InModule) },
	ast.ShowODataClients:      func(ctx *ExecContext, s *ast.ShowStmt) error { return listODataClients(ctx, s.InModule) },
	ast.ShowODataServices:     func(ctx *ExecContext, s *ast.ShowStmt) error { return listODataServices(ctx, s.InModule) },
	ast.ShowExternalEntities:  func(ctx *ExecContext, s *ast.ShowStmt) error { return listExternalEntities(ctx, s.InModule) },
	ast.ShowExternalActions:   func(ctx *ExecContext, s *ast.ShowStmt) error { return listExternalActions(ctx, s.InModule) },
	ast.ShowNavigation:        func(ctx *ExecContext, _ *ast.ShowStmt) error { return listNavigation(ctx) },
	ast.ShowNavigationMenu:    func(ctx *ExecContext, s *ast.ShowStmt) error { return listNavigationMenu(ctx, s.Name) },
	ast.ShowNavigationHomes:   func(ctx *ExecContext, _ *ast.ShowStmt) error { return listNavigationHomes(ctx) },
	ast.ShowStructure:         func(ctx *ExecContext, s *ast.ShowStmt) error { return execShowStructureGen(ctx, s) },
	ast.ShowWorkflows:         func(ctx *ExecContext, s *ast.ShowStmt) error { return listWorkflowsGen(ctx, s.InModule) },
	ast.ShowBusinessEventServices: func(ctx *ExecContext, s *ast.ShowStmt) error {
		return listBusinessEventServices(ctx, s.InModule)
	},
	ast.ShowBusinessEventClients: func(ctx *ExecContext, s *ast.ShowStmt) error {
		return listBusinessEventClients(ctx, s.InModule)
	},
	ast.ShowBusinessEvents:     func(ctx *ExecContext, s *ast.ShowStmt) error { return listBusinessEvents(ctx, s.InModule) },
	ast.ShowSettings:           func(ctx *ExecContext, _ *ast.ShowStmt) error { return listSettings(ctx) },
	ast.ShowLanguages:          func(ctx *ExecContext, _ *ast.ShowStmt) error { return listLanguages(ctx) },
	ast.ShowSupportedLanguages: func(ctx *ExecContext, _ *ast.ShowStmt) error { return listSupportedLanguages(ctx) },
	ast.ShowFragments:          func(ctx *ExecContext, _ *ast.ShowStmt) error { return listFragments(ctx) },
	ast.ShowDatabaseConnections: func(ctx *ExecContext, s *ast.ShowStmt) error {
		return listDatabaseConnections(ctx, s.InModule)
	},
	ast.ShowImageCollections: func(ctx *ExecContext, s *ast.ShowStmt) error { return listImageCollections(ctx, s.InModule) },
	ast.ShowModels:           func(ctx *ExecContext, s *ast.ShowStmt) error { return listAgentEditorModels(ctx, s.InModule) },
	ast.ShowAgents:           func(ctx *ExecContext, s *ast.ShowStmt) error { return listAgentEditorAgents(ctx, s.InModule) },
	ast.ShowKnowledgeBases: func(ctx *ExecContext, s *ast.ShowStmt) error {
		return listAgentEditorKnowledgeBases(ctx, s.InModule)
	},
	ast.ShowConsumedMCPServices: func(ctx *ExecContext, s *ast.ShowStmt) error {
		return listAgentEditorConsumedMCPServices(ctx, s.InModule)
	},
	ast.ShowRestClients: func(ctx *ExecContext, s *ast.ShowStmt) error { return listRestClients(ctx, s.InModule) },
	ast.ShowPublishedRestServices: func(ctx *ExecContext, s *ast.ShowStmt) error {
		return listPublishedRestServices(ctx, s.InModule)
	},
	ast.ShowDataTransformers: func(ctx *ExecContext, s *ast.ShowStmt) error { return listDataTransformers(ctx, s.InModule) },
	ast.ShowContractEntities: func(ctx *ExecContext, s *ast.ShowStmt) error { return listContractEntities(ctx, s.Name) },
	ast.ShowContractActions:  func(ctx *ExecContext, s *ast.ShowStmt) error { return listContractActions(ctx, s.Name) },
	ast.ShowContractChannels: func(ctx *ExecContext, s *ast.ShowStmt) error { return listContractChannels(ctx, s.Name) },
	ast.ShowContractMessages: func(ctx *ExecContext, s *ast.ShowStmt) error { return listContractMessages(ctx, s.Name) },
	ast.ShowJsonStructures:   func(ctx *ExecContext, s *ast.ShowStmt) error { return listJsonStructures(ctx, s.InModule) },
	ast.ShowImportMappings:   func(ctx *ExecContext, s *ast.ShowStmt) error { return listImportMappings(ctx, s.InModule) },
	ast.ShowExportMappings:   func(ctx *ExecContext, s *ast.ShowStmt) error { return listExportMappings(ctx, s.InModule) },
	ast.ShowJarDependencies:  func(ctx *ExecContext, s *ast.ShowStmt) error { return execListJarDependencies(ctx, s.InModule) },
}

// ShowHandlers returns the show handler map (exported for tests).
func ShowHandlers() map[ast.ShowObjectType]ShowHandler {
	return showHandlers
}

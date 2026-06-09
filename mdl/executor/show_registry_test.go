// SPDX-License-Identifier: Apache-2.0
package executor_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

// TestShowRegistryCoverage 确认每个经由 execShow 分发的 ShowObjectType 常量都有注册 handler。
// 这个测试在添加新 ShowObjectType 后立刻失败，提醒同步注册。
//
// 注意：ast.ShowWidgets 虽是 ShowObjectType 枚举值，但通过独立的 ShowWidgetsStmt
// 语句类型分发（见 register_stubs.go / cmd_widgets.go），不走 execShow，因此不在此覆盖范围内。
func TestShowRegistryCoverage(t *testing.T) {
	allTypes := []ast.ShowObjectType{
		ast.ShowModules, ast.ShowEnumerations, ast.ShowConstants,
		ast.ShowConstantValues, ast.ShowEntities, ast.ShowEntity,
		ast.ShowAssociations, ast.ShowAssociation,
		ast.ShowMicroflows, ast.ShowNanoflows,
		ast.ShowPages, ast.ShowSnippets, ast.ShowLayouts,
		ast.ShowJavaActions, ast.ShowJavaScriptActions,
		ast.ShowVersion, ast.ShowCatalogTables, ast.ShowCatalogStatus,
		ast.ShowCallers, ast.ShowCallees, ast.ShowReferences,
		ast.ShowImpact, ast.ShowContext,
		ast.ShowProjectSecurity, ast.ShowModuleRoles,
		ast.ShowUserRoles, ast.ShowDemoUsers,
		ast.ShowAccessOn, ast.ShowAccessOnMicroflow,
		ast.ShowAccessOnPage, ast.ShowAccessOnWorkflow,
		ast.ShowAccessOnNanoflow, ast.ShowSecurityMatrix,
		ast.ShowODataClients, ast.ShowODataServices,
		ast.ShowExternalEntities, ast.ShowExternalActions,
		ast.ShowNavigation, ast.ShowNavigationMenu, ast.ShowNavigationHomes,
		ast.ShowStructure, ast.ShowWorkflows,
		ast.ShowBusinessEventServices, ast.ShowBusinessEventClients,
		ast.ShowBusinessEvents, ast.ShowSettings, ast.ShowLanguages,
		ast.ShowSupportedLanguages, ast.ShowFragments,
		ast.ShowDatabaseConnections, ast.ShowImageCollections,
		ast.ShowModels, ast.ShowAgents, ast.ShowKnowledgeBases,
		ast.ShowConsumedMCPServices, ast.ShowRestClients,
		ast.ShowPublishedRestServices, ast.ShowDataTransformers,
		ast.ShowContractEntities, ast.ShowContractActions,
		ast.ShowContractChannels, ast.ShowContractMessages,
		ast.ShowJsonStructures, ast.ShowImportMappings,
		ast.ShowExportMappings, ast.ShowJarDependencies,
	}
	for _, ot := range allTypes {
		if _, ok := executor.ShowHandlers()[ot]; !ok {
			t.Errorf("ShowObjectType %v has no handler in showHandlers — add it to show_registry.go", ot)
		}
	}
}

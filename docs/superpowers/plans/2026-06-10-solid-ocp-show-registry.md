# OCP 修复：Show/Describe 语句注册表替代 130+ case switch

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `executor_query.go` 里两个大型 switch（`execShow` 66 case + `execDescribe` 77 case）替换为初始化时注册的 handler map，让新增 show/describe 类型只需在对应注册文件追加一行，不再修改 switch 函数本身。

**Architecture:** 在 `executor` 包内新建两个注册表文件：`show_registry.go` 和 `describe_registry.go`，各包含一个 `map[ast.ShowObjectType]func(...)` 和一个 `init` 函数（或包级 `var`）。`execShow`/`execDescribe` 函数改为查表调用。`describeObjectTypeLabel` 的 label switch 同样改为 map 查询。所有现有调用无需修改。

**Tech Stack:** Go 1.24，`mdl/executor` 包，`mdl/ast` 包。

---

## 影响文件概览

| 文件 | 操作 |
|------|------|
| `mdl/executor/show_registry.go` | 新建：`showHandlers` map 和 `ShowHandler` 类型 |
| `mdl/executor/describe_registry.go` | 新建：`describeHandlers` map 和 `DescribeHandler` 类型 |
| `mdl/executor/executor_query.go` | 修改：`execShow`/`execDescribe`/`describeObjectTypeLabel` 改为查表 |

---

## Task 1：建立 Show 注册表

- [ ] **Step 1.1：先写一个失败测试，确认注册表覆盖所有 ShowObjectType**

新建 `mdl/executor/show_registry_test.go`：

```go
// SPDX-License-Identifier: Apache-2.0
package executor_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// TestShowRegistryCoverage 确认每个 ShowObjectType 常量都有注册 handler。
// 这个测试在添加新 ShowObjectType 后立刻失败，提醒同步注册。
func TestShowRegistryCoverage(t *testing.T) {
	// 所有在 ast_query.go 中定义的 ShowObjectType 值
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
		ast.ShowWidgets,
	}
	for _, ot := range allTypes {
		if _, ok := ShowHandlers()[ot]; !ok {
			t.Errorf("ShowObjectType %v has no handler in showHandlers — add it to show_registry.go", ot)
		}
	}
}
```

- [ ] **Step 1.2：运行确认测试编译失败（`ShowHandlers` 不存在）**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/... -run TestShowRegistryCoverage 2>&1 | head -10
```

预期：`undefined: ShowHandlers`。

- [ ] **Step 1.3：新建 `mdl/executor/show_registry.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import "github.com/mendixlabs/mxcli/mdl/ast"

// ShowHandler is the function signature for all SHOW statement handlers.
type ShowHandler func(ctx *ExecContext, s *ast.ShowStmt) error

// showHandlers maps each ShowObjectType to its handler.
// Adding a new SHOW command only requires appending one entry here.
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
	ast.ShowBusinessEvents:    func(ctx *ExecContext, s *ast.ShowStmt) error { return listBusinessEvents(ctx, s.InModule) },
	ast.ShowSettings:          func(ctx *ExecContext, _ *ast.ShowStmt) error { return listSettings(ctx) },
	ast.ShowLanguages:         func(ctx *ExecContext, _ *ast.ShowStmt) error { return listLanguages(ctx) },
	ast.ShowSupportedLanguages: func(ctx *ExecContext, _ *ast.ShowStmt) error { return listSupportedLanguages(ctx) },
	ast.ShowFragments:         func(ctx *ExecContext, _ *ast.ShowStmt) error { return listFragments(ctx) },
	ast.ShowDatabaseConnections: func(ctx *ExecContext, s *ast.ShowStmt) error {
		return listDatabaseConnections(ctx, s.InModule)
	},
	ast.ShowImageCollections:   func(ctx *ExecContext, s *ast.ShowStmt) error { return listImageCollections(ctx, s.InModule) },
	ast.ShowModels:             func(ctx *ExecContext, s *ast.ShowStmt) error { return listAgentEditorModels(ctx, s.InModule) },
	ast.ShowAgents:             func(ctx *ExecContext, s *ast.ShowStmt) error { return listAgentEditorAgents(ctx, s.InModule) },
	ast.ShowKnowledgeBases:     func(ctx *ExecContext, s *ast.ShowStmt) error { return listAgentEditorKnowledgeBases(ctx, s.InModule) },
	ast.ShowConsumedMCPServices: func(ctx *ExecContext, s *ast.ShowStmt) error {
		return listAgentEditorConsumedMCPServices(ctx, s.InModule)
	},
	ast.ShowRestClients:          func(ctx *ExecContext, s *ast.ShowStmt) error { return listRestClients(ctx, s.InModule) },
	ast.ShowPublishedRestServices: func(ctx *ExecContext, s *ast.ShowStmt) error {
		return listPublishedRestServices(ctx, s.InModule)
	},
	ast.ShowDataTransformers:  func(ctx *ExecContext, s *ast.ShowStmt) error { return listDataTransformers(ctx, s.InModule) },
	ast.ShowContractEntities:  func(ctx *ExecContext, s *ast.ShowStmt) error { return listContractEntities(ctx, s.Name) },
	ast.ShowContractActions:   func(ctx *ExecContext, s *ast.ShowStmt) error { return listContractActions(ctx, s.Name) },
	ast.ShowContractChannels:  func(ctx *ExecContext, s *ast.ShowStmt) error { return listContractChannels(ctx, s.Name) },
	ast.ShowContractMessages:  func(ctx *ExecContext, s *ast.ShowStmt) error { return listContractMessages(ctx, s.Name) },
	ast.ShowJsonStructures:    func(ctx *ExecContext, s *ast.ShowStmt) error { return listJsonStructures(ctx, s.InModule) },
	ast.ShowImportMappings:    func(ctx *ExecContext, s *ast.ShowStmt) error { return listImportMappings(ctx, s.InModule) },
	ast.ShowExportMappings:    func(ctx *ExecContext, s *ast.ShowStmt) error { return listExportMappings(ctx, s.InModule) },
	ast.ShowJarDependencies:   func(ctx *ExecContext, s *ast.ShowStmt) error { return execListJarDependencies(ctx, s.InModule) },
	ast.ShowWidgets:           func(ctx *ExecContext, s *ast.ShowStmt) error { return listWidgets(ctx, s) },
}

// ShowHandlers returns the show handler map (exported for tests).
func ShowHandlers() map[ast.ShowObjectType]ShowHandler {
	return showHandlers
}
```

> **注意**：`listWidgets` 和具体函数名需与 `executor_query.go` 中 switch case 的实际调用一致。先从 switch 复制，后面编译会报错如果有遗漏。

- [ ] **Step 1.4：运行覆盖率测试确认通过**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/... -run TestShowRegistryCoverage -v
```

预期：`PASS`。如果有 `ShowObjectType X has no handler`，在 `show_registry.go` 补充对应条目。

---

## Task 2：建立 Describe 注册表

- [ ] **Step 2.1：写 Describe 注册表覆盖率测试**

新建 `mdl/executor/describe_registry_test.go`：

```go
// SPDX-License-Identifier: Apache-2.0
package executor_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestDescribeRegistryCoverage(t *testing.T) {
	allTypes := []ast.DescribeObjectType{
		ast.DescribeEnumeration, ast.DescribeEntity, ast.DescribeAssociation,
		ast.DescribeMicroflow, ast.DescribeNanoflow, ast.DescribeModule,
		ast.DescribePage, ast.DescribeSnippet, ast.DescribeLayout,
		ast.DescribeConstant, ast.DescribeJavaAction, ast.DescribeJavaScriptAction,
		ast.DescribeModuleRole, ast.DescribeUserRole, ast.DescribeDemoUser,
		ast.DescribeODataClient, ast.DescribeODataService, ast.DescribeExternalEntity,
		ast.DescribeNavigation, ast.DescribeWorkflow, ast.DescribeBusinessEventService,
		ast.DescribeDatabaseConnection, ast.DescribeSettings, ast.DescribeFragment,
		ast.DescribeImageCollection, ast.DescribeModel, ast.DescribeAgent,
		ast.DescribeKnowledgeBase, ast.DescribeConsumedMCPService,
		ast.DescribeRestClient, ast.DescribePublishedRestService,
		ast.DescribeDataTransformer, ast.DescribeContractEntity,
		ast.DescribeContractAction, ast.DescribeContractMessage,
		ast.DescribeJsonStructure, ast.DescribeImportMapping, ast.DescribeExportMapping,
		ast.DescribeJarDependency,
	}
	for _, ot := range allTypes {
		if _, ok := DescribeHandlers()[ot]; !ok {
			t.Errorf("DescribeObjectType %v has no handler — add it to describe_registry.go", ot)
		}
		if DescribeLabel(ot) == "unknown" {
			t.Errorf("DescribeObjectType %v has no label — add it to describe_registry.go", ot)
		}
	}
}
```

- [ ] **Step 2.2：新建 `mdl/executor/describe_registry.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
)

// DescribeHandler is the function signature for DESCRIBE statement handlers.
// The handler is called inside writeDescribeJSON; it writes JSON fields via ctx.Output.
type DescribeHandler func(ctx *ExecContext, s *ast.DescribeStmt) error

type describeEntry struct {
	handler DescribeHandler
	label   string
}

// describeHandlers maps each DescribeObjectType to its handler and human label.
var describeHandlers = map[ast.DescribeObjectType]describeEntry{
	ast.DescribeEnumeration: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeEnumeration(ctx, s.Name) },
		label:   "enumeration",
	},
	ast.DescribeEntity: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeEntityGen(ctx, s.Name) },
		label:   "entity",
	},
	ast.DescribeAssociation: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeAssociation(ctx, s.Name) },
		label:   "association",
	},
	ast.DescribeMicroflow: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeMicroflowGen(ctx, s.Name) },
		label:   "microflow",
	},
	ast.DescribeNanoflow: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeNanoflowGen(ctx, s.Name) },
		label:   "nanoflow",
	},
	ast.DescribeModule: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error {
			return describeModule(ctx, s.Name.Module, s.WithAll)
		},
		label: "module",
	},
	ast.DescribePage: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describePage(ctx, s.Name) },
		label:   "page",
	},
	ast.DescribeSnippet: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeSnippet(ctx, s.Name) },
		label:   "snippet",
	},
	ast.DescribeLayout: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeLayout(ctx, s.Name) },
		label:   "layout",
	},
	ast.DescribeConstant: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeConstant(ctx, s.Name) },
		label:   "constant",
	},
	ast.DescribeJavaAction: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeJavaActionGen(ctx, s.Name) },
		label:   "javaaction",
	},
	ast.DescribeJavaScriptAction: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeJavaScriptActionGen(ctx, s.Name) },
		label:   "javascriptaction",
	},
	ast.DescribeModuleRole: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeModuleRoleGen(ctx, s.Name) },
		label:   "modulerole",
	},
	ast.DescribeUserRole: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeUserRoleGen(ctx, s.Name) },
		label:   "userrole",
	},
	ast.DescribeDemoUser: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeDemoUserGen(ctx, s.Name.Name) },
		label:   "demouser",
	},
	ast.DescribeODataClient: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeODataClient(ctx, s.Name) },
		label:   "odataclient",
	},
	ast.DescribeODataService: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeODataService(ctx, s.Name) },
		label:   "odataservice",
	},
	ast.DescribeExternalEntity: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeExternalEntity(ctx, s.Name) },
		label:   "externalentity",
	},
	ast.DescribeNavigation: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeNavigation(ctx, s.Name) },
		label:   "navigation",
	},
	ast.DescribeWorkflow: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeWorkflowGen(ctx, s.Name) },
		label:   "workflow",
	},
	ast.DescribeBusinessEventService: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeBusinessEventService(ctx, s.Name) },
		label:   "businesseventservice",
	},
	ast.DescribeDatabaseConnection: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeDatabaseConnection(ctx, s.Name) },
		label:   "databaseconnection",
	},
	ast.DescribeSettings: {
		handler: func(ctx *ExecContext, _ *ast.DescribeStmt) error { return describeSettings(ctx) },
		label:   "settings",
	},
	ast.DescribeFragment: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeFragment(ctx, s.Name) },
		label:   "fragment",
	},
	ast.DescribeImageCollection: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeImageCollection(ctx, s.Name) },
		label:   "imagecollection",
	},
	ast.DescribeModel: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeAgentEditorModel(ctx, s.Name) },
		label:   "model",
	},
	ast.DescribeAgent: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeAgentEditorAgent(ctx, s.Name) },
		label:   "agent",
	},
	ast.DescribeKnowledgeBase: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeAgentEditorKnowledgeBase(ctx, s.Name) },
		label:   "knowledgebase",
	},
	ast.DescribeConsumedMCPService: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error {
			return describeAgentEditorConsumedMCPService(ctx, s.Name)
		},
		label: "consumedmcpservice",
	},
	ast.DescribeRestClient: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeRestClient(ctx, s.Name) },
		label:   "restclient",
	},
	ast.DescribePublishedRestService: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describePublishedRestService(ctx, s.Name) },
		label:   "publishedrestservice",
	},
	ast.DescribeDataTransformer: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeDataTransformer(ctx, s.Name) },
		label:   "datatransformer",
	},
	ast.DescribeContractEntity: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error {
			return describeContractEntity(ctx, s.Name, s.Format)
		},
		label: "contractentity",
	},
	ast.DescribeContractAction: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error {
			return describeContractAction(ctx, s.Name, s.Format)
		},
		label: "contractaction",
	},
	ast.DescribeContractMessage: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeContractMessage(ctx, s.Name) },
		label:   "contractmessage",
	},
	ast.DescribeJsonStructure: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeJsonStructure(ctx, s.Name) },
		label:   "jsonstructure",
	},
	ast.DescribeImportMapping: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeImportMapping(ctx, s.Name) },
		label:   "importmapping",
	},
	ast.DescribeExportMapping: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeExportMapping(ctx, s.Name) },
		label:   "exportmapping",
	},
	ast.DescribeJarDependency: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error {
			return execDescribeJarDependency(ctx, s.Name.String(), s.Qualifier)
		},
		label: "jardependency",
	},
}

// DescribeHandlers returns the describe handler map (exported for tests).
func DescribeHandlers() map[ast.DescribeObjectType]DescribeHandler {
	m := make(map[ast.DescribeObjectType]DescribeHandler, len(describeHandlers))
	for k, v := range describeHandlers {
		m[k] = v.handler
	}
	return m
}

// DescribeLabel returns the human-readable label for a describe object type.
func DescribeLabel(t ast.DescribeObjectType) string {
	if e, ok := describeHandlers[t]; ok {
		return e.label
	}
	return "unknown"
}
```

- [ ] **Step 2.3：运行覆盖率测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/... -run TestDescribeRegistryCoverage -v
```

预期：`PASS`。

---

## Task 3：把 executor_query.go 改为查表

- [ ] **Step 3.1：替换 `execShow` 函数**

找到 `executor_query.go:10-151`（从 `func execShow` 到末尾 `}`），替换为：

```go
func execShow(ctx *ExecContext, s *ast.ShowStmt) error {
	if !ctx.Connected() && s.ObjectType != ast.ShowModules && s.ObjectType != ast.ShowFragments {
		return mdlerrors.NewNotConnected()
	}
	handler, ok := showHandlers[s.ObjectType]
	if !ok {
		return mdlerrors.NewUnsupported("unknown show object type")
	}
	return handler(ctx, s)
}
```

- [ ] **Step 3.2：替换 `execDescribe` 函数**

找到 `executor_query.go:153-246`（从 `func execDescribe` 到末尾），替换为：

```go
func execDescribe(ctx *ExecContext, s *ast.DescribeStmt) error {
	if !ctx.Connected() && s.ObjectType != ast.DescribeFragment {
		return mdlerrors.NewNotConnected()
	}
	entry, ok := describeHandlers[s.ObjectType]
	if !ok {
		return mdlerrors.NewUnsupported("unknown describe object type")
	}
	name := s.Name.String()
	return writeDescribeJSON(ctx, name, entry.label, func() error {
		return entry.handler(ctx, s)
	})
}
```

- [ ] **Step 3.3：删除 `describeObjectTypeLabel` 函数**

找到 `executor_query.go:248-330`（从注释到末尾），完整删除。这个函数由 `describeHandlers` 中的 `label` 字段替代。

- [ ] **Step 3.4：检查 `describeObjectTypeLabel` 其他调用点**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
grep -rn "describeObjectTypeLabel" mdl/executor/ --include="*.go"
```

若有其他调用点，改为 `DescribeLabel(s.ObjectType)`。

- [ ] **Step 3.5：编译确认**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/executor/...
```

预期：无错误。

- [ ] **Step 3.6：运行全量 executor 测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/... -count=1 2>&1 | tail -15
```

预期：全部 `ok`。

- [ ] **Step 3.7：commit**

```bash
git add mdl/executor/show_registry.go mdl/executor/describe_registry.go \
        mdl/executor/show_registry_test.go mdl/executor/describe_registry_test.go \
        mdl/executor/executor_query.go
git commit -m "$(cat <<'EOF'
refactor(executor): replace 130+ case show/describe switch with handler maps

execShow (66 cases) and execDescribe (77 cases) in executor_query.go
violated OCP: adding any new SHOW/DESCRIBE command required modifying
the switch directly.

Introduce show_registry.go and describe_registry.go: each contains a
map[ObjectType]handler var. execShow/execDescribe are now 6-line table
lookups. describeObjectTypeLabel() deleted; label lives alongside handler
in the describeEntry struct. Coverage tests enforce that all ast.*ObjectType
values have a registered handler, catching gaps at test-time.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## 自检 Checklist

- [ ] `go build ./mdl/executor/...` 无错误
- [ ] `go test ./mdl/executor/... -run TestShowRegistryCoverage` PASS
- [ ] `go test ./mdl/executor/... -run TestDescribeRegistryCoverage` PASS
- [ ] `go test ./mdl/executor/...` 全部 `ok`，无 FAIL
- [ ] `grep -n "case ast.Show" mdl/executor/executor_query.go` 无输出（switch 已删除）
- [ ] `grep -n "describeObjectTypeLabel" mdl/executor/` 无输出（函数已删除）
- [ ] `wc -l mdl/executor/executor_query.go` 比之前小约 280 行

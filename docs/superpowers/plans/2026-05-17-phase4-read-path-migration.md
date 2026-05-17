# Phase 4: sdk/mpr 读路径迁移 — 完全退役 sdk/mpr 包

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `backend.go` 和 `project_tree.go` 的读路径从 `sdk/mpr.Reader` 完全迁移到 `modelsdk/mpr.Reader`，最终删除 `sdk/mpr/` 整目录。

**Tech Stack:** Go 1.26，`GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go`

**架构设计文档:** `docs/superpowers/mxcli-arch-viz.html`（5个Tab，含算法、进度、层次图）

---

## 路线图（总览）

```
阶段       目标文件                    关键工作                         预计提交数
─────────────────────────────────────────────────────────────────────────────
Phase 4A   cmd/mxcli/project_tree.go   Track A: mprread + join 模式       ~6
Phase 4B   modelsdk/mpr/reader_model.go  Track B: gen→model converter 按域   ~12
           mdl/backend/mpr/backend.go   合并双 reader，切换 b.reader 类型    ~2
Phase 5    sdk/mpr/                    删除整目录（前置: 4A+4B 全完成）    ~1
─────────────────────────────────────────────────────────────────────────────
```

**已完成前置工作：**
- ✅ `6f5efb6c` — 修复 `generic.go` 双重IO：`ListUnitsWithContainer[T]` 携带 ContainerID，1次SQL
- ✅ PR5 Phase 2 — 3个 bridge 文件已删
- ✅ PR5 Phase 1 — `modelsdk/mprread` 31个自由函数覆盖所有域

---

## 前提条件验证

```bash
# 确认 generic.go 修复已生效
grep -n "ListUnitsWithContainer" modelsdk/mprread/generic.go
# 期望：找到函数定义

# 确认 sdk/mpr 残余 import（应只有 2 处）
grep -r '"github.com/mendixlabs/mxcli/sdk/mpr"' . --include="*.go" | grep -v "_test"
# 期望：backend.go + project_tree.go 各 1 处

# 全量测试基线
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./...
# 期望：全 ok
```

---

## Phase 4A：project_tree.go 迁移（Track A）

**Files:**
- Modify: `cmd/mxcli/project_tree.go`
- Modify: `modelsdk/mprread/reader_documents.go`（新增 8 个 Gen 函数）

**策略：** 无需 converter。薄型方法（只用 ContainerID + Name）使用 `mprread.ListXxx(mreader)` + `mreader.ListUnitsByType` join。Gen 变体方法直接替换为新的 mprread 函数。富型方法改 helper 函数签名接受 gen 类型。

---

### Task A1：mprread 新增 9 个 Gen 列表函数

**Files:** `modelsdk/mprread/reader_documents.go`

每个函数 1 行，调用 `ListUnitsByType[T]`：

- [ ] **A1.1 确认各域的 BSON $Type 名称**

```bash
grep -n "listUnitsByType\|typePrefix" sdk/mpr/reader_documents.go | grep -i "page\|snippet\|layout\|domain\|workflow\|javaaction\|building\|pagetemplate\|javascript" | head -15
```

- [ ] **A1.2 在 reader_documents.go 末尾添加 9 个函数**

```go
// ListPages returns all pages (Forms$Page) in the project.
func ListPages(r *mmpr.Reader) ([]*genPg.Page, error) {
    return ListUnitsByType[*genPg.Page](r, "Forms$Page")
}

// ListSnippets returns all snippets (Forms$Snippet) in the project.
func ListSnippets(r *mmpr.Reader) ([]*genPg.Snippet, error) {
    return ListUnitsByType[*genPg.Snippet](r, "Forms$Snippet")
}

// ListLayouts returns all layouts (Forms$Layout) in the project.
func ListLayouts(r *mmpr.Reader) ([]*genPg.Layout, error) {
    return ListUnitsByType[*genPg.Layout](r, "Forms$Layout")
}

// ListDomainModels returns all domain models in the project.
func ListDomainModels(r *mmpr.Reader) ([]*genDM.DomainModel, error) {
    return ListUnitsByType[*genDM.DomainModel](r, "DomainModels$DomainModel")
}

// ListWorkflows returns all workflows in the project.
func ListWorkflows(r *mmpr.Reader) ([]*genWf.Workflow, error) {
    return ListUnitsByType[*genWf.Workflow](r, "Workflows$Workflow")
}

// ListJavaActions returns all Java actions in the project.
func ListJavaActions(r *mmpr.Reader) ([]*genJA.JavaAction, error) {
    return ListUnitsByType[*genJA.JavaAction](r, "JavaActions$JavaAction")
}

// ListBuildingBlocks returns all building blocks in the project.
func ListBuildingBlocks(r *mmpr.Reader) ([]*genPg.BuildingBlock, error) {
    return ListUnitsByType[*genPg.BuildingBlock](r, "Forms$BuildingBlock")
}

// ListPageTemplates returns all page templates in the project.
func ListPageTemplates(r *mmpr.Reader) ([]*genPg.PageTemplate, error) {
    return ListUnitsByType[*genPg.PageTemplate](r, "Forms$PageTemplate")
}

// ListJavaScriptActions returns all JavaScript actions in the project.
func ListJavaScriptActions(r *mmpr.Reader) ([]*genJSA.JavaScriptAction, error) {
    return ListUnitsByType[*genJSA.JavaScriptAction](r, "JavaScriptActions$JavaScriptAction")
}
```

注意：需在 import 块加对应 gen 包别名（genPg, genDM, genWf, genJA, genPg, genJSA）。

- [ ] **A1.3 编译验证**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./modelsdk/mprread/...
```

- [ ] **A1.4 提交**

```bash
git add modelsdk/mprread/reader_documents.go
git commit -m "feat(mprread): add Gen list functions for pages/snippets/layouts/DMs/workflows/JAs/BBs/PTs/JSAs"
```

---

### Task A2：薄型方法迁移（15 个，ContainerID+Name 够用）

**Files:** `cmd/mxcli/project_tree.go`

**迁移模式（以 ListDataTransformers 为例）：**

```go
// 旧：
dts, _ := reader.ListDataTransformers()
for _, dt := range dts {
    modID := h.FindModuleID(dt.ContainerID)
    md.documents = append(md.documents, treeElement{Name: dt.Name, ContainerID: dt.ContainerID, Type: "datatransformer"})
}

// 新（利用 ListUnitsWithContainer 消除 join 样板）：
dtUnits, _ := mprread.ListUnitsWithContainer[*genDT.DataTransformer](mreader, "DataTransformers$DataTransformer")
for _, u := range dtUnits {
    modID := h.FindModuleID(u.ContainerID)
    md.documents = append(md.documents, treeElement{Name: u.Element.Name(), ContainerID: u.ContainerID, Type: "datatransformer"})
}
```

等价地，对于已有专用 mprread 函数的域，使用 `ListUnitsWithContainer` 直接传 BSON 类型名，或保留 `mreader.ListUnitsByType` join 模式（两者均可）。

**15 个待迁移方法及其 BSON 类型名：**

| reader 方法 | BSON 类型名 | gen 类型 |
|------------|------------|---------|
| ListEnumerations | `Enumerations$Enumeration` | `*genEnum.Enumeration` |
| ListConstants | `Constants$Constant` | `*genConst.Constant` |
| ListScheduledEvents | `ScheduledEvents$ScheduledEvent` | `*genSched.ScheduledEvent` |
| ListImageCollections | `Images$ImageCollection` | `*genImg.ImageCollection` |
| ListJsonStructures | `JsonStructures$JsonStructure` | `*genJson.JsonStructure` |
| ListImportMappings | `ImportMappings$ImportMapping` | `*genImpMap.ImportMapping` |
| ListExportMappings | `ExportMappings$ExportMapping` | `*genExpMap.ExportMapping` |
| ListDataTransformers | `DataTransformers$DataTransformer` | `*genDT.DataTransformer` |
| ListDatabaseConnections | `DatabaseConnector$DatabaseConnection` | `*genDBC.DatabaseConnection` |
| ListConsumedODataServices | `Rest$ConsumedODataService` | `*genRest.ConsumedODataService` |
| ListPublishedODataServices | `ODataPublish$PublishedODataService2` | `*genODataPub.PublishedODataService2` |
| ListConsumedRestServices | `Rest$ConsumedRestService` | `*genRest.ConsumedRestService` |
| ListPublishedRestServices | `Rest$PublishedRestService` | `*genRest.PublishedRestService` |
| ListBusinessEventServices | `BusinessEvents$BusinessEventService` | `*genBE.BusinessEventService` |
| ListJavaScriptActions | `JavaScriptActions$JavaScriptAction` | `*genJSA.JavaScriptAction` |

**注意：** 对于 `ListPublishedODataServices`、`ListPublishedRestServices`、`ListBusinessEventServices`、`ListConsumedRestServices`、`ListDatabaseConnections`，project_tree.go 有 `buildXxxChildren` helper 函数，目前接受 `model.*` 类型参数。**Task A2 只改薄型主循环（ContainerID+Name），暂时保留 buildXxxChildren 的 model.* 参数**，等 Task A4 再改 helper 签名。

- [ ] **A2.1 迁移 AgentEditor 4 个（直接 passthrough，mprread 已返回 types.*）**

`ListAgentEditorModels/KnowledgeBases/ConsumedMCPServices/Agents` 在 mprread 中已返回 `*types.Model` 等，直接调 `mprread.ListAgentEditorXxx(mreader)`，无需 join（ContainerID 在返回的 types.* 里已有）。

```bash
grep -n "ListAgentEditor" modelsdk/mprread/reader_documents.go | head -8
# 确认返回类型（types.Model 已有 ContainerID 字段）
```

- [ ] **A2.2 迁移剩余 11 个薄型方法**

对每个方法，使用 `mprread.ListUnitsWithContainer[T]` 模式（利用 generic.go 的 P0 bugfix 成果）。

- [ ] **A2.3 编译验证**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./cmd/mxcli/...
```

- [ ] **A2.4 运行时验证（用 mx2026aiday）**

```bash
/tmp/mxcli-pr5 -p "/mnt/data_sdd/gh/Mx2026AIDay/Factory Management.mpr" -c "show structure" 2>&1
# 期望：数字与迁移前一致（158 enumerations, 82 constants 等）
```

- [ ] **A2.5 提交**

```bash
git add cmd/mxcli/project_tree.go
git commit -m "refactor(project-tree): migrate 15 thin-typed readers from sdk/mpr to mprread"
```

---

### Task A3：Gen 变体方法迁移（8 个）

**Files:** `cmd/mxcli/project_tree.go`

替换 `reader.ListXxxGen()` → 新的 mprread 函数 `mprread.ListXxx(mreader)`。

```go
// 旧：
pgs, _ := reader.ListPagesGen()
for _, pg := range pgs {
    containerID := pgContainerByID[string(pg.ID())]  // 仍需 join
    ...
}

// 新（join map 保留，Gen 变体改 mprread）：
pgs, _ := mprread.ListPages(mreader)  // A1 新增的函数
for _, pg := range pgs {
    containerID := pgContainerByID[string(pg.ID())]  // join map 不变
    ...
}
```

8 个方法：Pages / Snippets / Layouts / DomainModels / Workflows / JavaActions / BuildingBlocks / PageTemplates

- [ ] **A3.1 替换 8 处 reader.ListXxxGen() 调用**
- [ ] **A3.2 编译验证**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./cmd/mxcli/...
```

- [ ] **A3.3 运行时验证**

```bash
/tmp/mxcli-pr5 -p "/mnt/data_sdd/gh/Mx2026AIDay/Factory Management.mpr" project-tree 2>&1 | python3 -m json.tool | grep '"type"' | sort | uniq -c | sort -rn | head -20
# 期望：各类型数量与迁移前一致
```

- [ ] **A3.4 提交**

```bash
git add cmd/mxcli/project_tree.go
git commit -m "refactor(project-tree): replace ListXxxGen() with mprread.ListXxx() — use A1 new functions"
```

---

### Task A4：富型 helper 函数改 gen 类型参数（5 个）

**Files:** `cmd/mxcli/project_tree.go`

需要改 `buildXxxChildren` 函数签名，接受 gen/* 类型而非 model.* 类型：

| helper 函数 | 当前参数类型 | 目标参数类型 | 深度字段 |
|------------|------------|------------|---------|
| `buildPublishedODataChildren` | `*model.PublishedODataService` | `*genODataPub.PublishedODataService2` | EntitySets, EntityTypes, Members |
| `buildPublishedRestChildren` | `*model.PublishedRestService` | `*genRest.PublishedRestService` | Resources, Operations |
| `buildBusinessEventChildren` | `*model.BusinessEventService` | `*genBE.BusinessEventService` | Definition.Channels, Messages |
| `buildConsumedRestChildren`（内联） | `*model.ConsumedRestService` | `*genRest.ConsumedRestService` | Operations, HttpMethod, Path |
| `buildDatabaseConnectionChildren` | `*model.DatabaseConnection` | `*genDBC.DatabaseConnection` | Queries, Parameters, TableMappings |

**注意：** 需先确认 gen 类型的字段访问方式（用 `.XxxItems()` 列表方法，用 `.Xxx()` 取字符串等），对照 gen 包的 types.go 确认。

- [ ] **A4.1 确认 gen 类型字段访问方式**

```bash
grep -n "func.*EntitySets\|func.*Resources\|func.*Channels\|func.*Operations\|func.*Queries" \
  modelsdk/gen/odata_publish/types.go \
  modelsdk/gen/rest/types.go \
  modelsdk/gen/businessevents/types.go \
  modelsdk/gen/databaseconnector/types.go 2>/dev/null | head -30
```

- [ ] **A4.2 重写 5 个 helper 函数接受 gen 类型**
- [ ] **A4.3 更新调用点**
- [ ] **A4.4 编译验证**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./cmd/mxcli/...
```

- [ ] **A4.5 运行时验证（重点验证 buildXxx 输出的子节点）**

```bash
/tmp/mxcli-pr5 -p "/mnt/data_sdd/gh/Mx2026AIDay/Factory Management.mpr" project-tree 2>&1 | python3 -m json.tool | grep -A3 '"type": "publishedrestservice"' | head -30
```

- [ ] **A4.6 提交**

```bash
git add cmd/mxcli/project_tree.go
git commit -m "refactor(project-tree): migrate rich-typed buildXxxChildren helpers to gen types"
```

---

### Task A5：迁移 ListModules + GetProjectSecurity + GetProjectSettings + GetNavigation

**Files:** `cmd/mxcli/project_tree.go`

- `ListModules()` — `mreader.ListModules()` 已在 modelsdk/mpr.Reader，直接替换
- `GetProjectSecurity()` — `mprread.GetProjectSecurity(mreader)` 返回 `*genSec.ProjectSecurity`（project_tree.go 已在用 `ps.UserRolesItems()` 等 gen 方法，直接 passthrough）
- `GetProjectSettings()` — `mprread.GetProjectSettings(mreader)` 返回 `*genSet.ProjectSettings`，需改 Settings 节点构建逻辑
- `GetNavigation()` — `mprread.GetNavigation(mreader)` 返回 `*genNav.NavigationDocument`，需改 Navigation 节点构建逻辑

- [ ] **A5.1 确认 gen 类型字段访问方式**

```bash
grep -n "func.*Model\b\|func.*Language\|func.*AfterStartup\|func.*BeforeShutdown" \
  modelsdk/gen/settings/types.go 2>/dev/null | head -10
grep -n "func.*Profiles\|func.*HomePage\|func.*MenuItems\|func.*Kind" \
  modelsdk/gen/navigation/types.go 2>/dev/null | head -10
```

- [ ] **A5.2 替换 reader.GetProjectSecurity / GetProjectSettings / GetNavigation / ListModules**
- [ ] **A5.3 删除 `reader *sdkmpr.Reader` 字段和 `sdkmpr.Open(projectPath)` 调用**

```go
// 删除：
reader, err := sdkmpr.Open(projectPath)
defer reader.Close()

// mreader 已是唯一的 reader：
mreader, err := mmpr.Open(projectPath)
```

- [ ] **A5.4 删除 sdk/mpr import**

```go
// 删除：
sdkmpr "github.com/mendixlabs/mxcli/sdk/mpr"
```

- [ ] **A5.5 编译验证**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./cmd/mxcli/...
```

- [ ] **A5.6 全量测试**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./...
```

- [ ] **A5.7 运行时完整验证**

```bash
/tmp/mxcli-pr5 -p "/mnt/data_sdd/gh/Mx2026AIDay/Factory Management.mpr" project-tree 2>&1 | python3 -m json.tool | wc -l
# 期望：行数与迁移前一致（navigation/settings/security 节点存在）
```

- [ ] **A5.8 提交**

```bash
git add cmd/mxcli/project_tree.go
git commit -m "refactor(project-tree): remove sdk/mpr dependency — fully on modelsdk/mpr + mprread"
```

**Phase 4A 验收：**
```bash
grep '"github.com/mendixlabs/mxcli/sdk/mpr"' cmd/mxcli/project_tree.go
# 期望：空（sdk/mpr import 已删）
```

---

## Phase 4B：reader_model.go + backend.go 迁移（Track B）

**Files:**
- Create: `modelsdk/mpr/reader_model.go`（新文件，按域分批加方法）
- Modify: `mdl/backend/mpr/backend.go`（合并双 reader，切换类型）
- Delete: `mdl/backend/mpr/convert_reader.go`（3个旧 converter 迁移进 reader_model.go）

**通用模式（使用 P0 bugfix 的 ListUnitsWithContainer）：**

```go
func (r *Reader) ListEnumerations() ([]*model.Enumeration, error) {
    units, err := mprread.ListUnitsWithContainer[*genEnum.Enumeration](r, "Enumerations$Enumeration")
    if err != nil {
        return nil, err
    }
    out := make([]*model.Enumeration, 0, len(units))
    for _, u := range units {
        e := &model.Enumeration{
            BaseElement:   model.BaseElement{ID: model.ID(u.Element.ID())},
            ContainerID:   u.ContainerID,              // 直接从 Unit[T] 取，无需 join
            Name:          u.Element.Name(),
            Documentation: u.Element.Documentation(),
        }
        for _, v := range u.Element.ValuesItems() {
            ev, ok := v.(*genEnum.EnumerationValue)
            if !ok { continue }
            e.Values = append(e.Values, model.EnumerationValue{
                BaseElement: model.BaseElement{ID: model.ID(ev.ID())},
                Name:        ev.Name(),
            })
        }
        out = append(out, e)
    }
    return out, nil
}
```

---

### Task B1：α 类 — AgentEditor 4 个方法（passthrough）

`ListAgentEditorModels/KnowledgeBases/ConsumedMCPServices/Agents` 在 mprread 中已返回 `*types.Model` 等（`mdl/types` 包类型），直接 passthrough，零 converter。

- [ ] **B1.1 新建 modelsdk/mpr/reader_model.go**

```go
// SPDX-License-Identifier: Apache-2.0

// reader_model.go — model.*-typed reader methods for modelsdk/mpr.Reader.
// Each method delegates to modelsdk/mprread (gen-typed free functions) and
// converts gen/* types to model.* / mdl/types.* via thin converters.
// This file is the drop-in replacement for sdk/mpr.Reader's domain methods.
package mpr

import (
    "github.com/mendixlabs/mxcli/mdl/types"
    "github.com/mendixlabs/mxcli/modelsdk/mprread"
)

func (r *Reader) ListAgentEditorModels() ([]*types.Model, error) {
    return mprread.ListAgentEditorModels(r)
}

func (r *Reader) ListAgentEditorKnowledgeBases() ([]*types.KnowledgeBase, error) {
    return mprread.ListAgentEditorKnowledgeBases(r)
}

func (r *Reader) ListAgentEditorConsumedMCPServices() ([]*types.ConsumedMCPService, error) {
    return mprread.ListAgentEditorConsumedMCPServices(r)
}

func (r *Reader) ListAgentEditorAgents() ([]*types.Agent, error) {
    return mprread.ListAgentEditorAgents(r)
}
```

- [ ] **B1.2 编译验证**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./modelsdk/mpr/...
```

- [ ] **B1.3 提交**

```bash
git add modelsdk/mpr/reader_model.go
git commit -m "feat(modelsdk/mpr): add reader_model.go — AgentEditor passthrough methods (α)"
```

---

### Task B2：β 类 — 薄 converter 方法

**Enumeration / Constant / ScheduledEvent / ImageCollection / NavigationDocument / ModuleSettings（+ Get 变体）**

对每个域：参考 `model/types.go` 的字段定义，对照 gen 类型的 accessor，写薄 converter。

- [ ] **B2.1 ListEnumerations + GetEnumeration**（需映射 ContainerID, Name, Documentation, Values）
- [ ] **B2.2 ListConstants + GetConstant**（需映射 ContainerID, Name, Documentation, Type, DefaultValue, ExposedToClient）

**注意：** Constant.Type 是 `model.ConstantDataType`（含 Kind/EnumRef/EntityRef），对照 `genConst.Constant.DataType()` 和 `genConst.Constant.PropType()` 确认映射。

- [ ] **B2.3 ListScheduledEvents + GetScheduledEvent**（ContainerID, Name, Documentation）
- [ ] **B2.4 ListImageCollections**（ContainerID, Name）
- [ ] **B2.5 ListNavigationDocuments + GetNavigation**（ContainerID, Profiles[]，结构较深）
- [ ] **B2.6 ListModuleSettings + GetModuleSettings**（ContainerID，模块设置字段）

每个子步骤完成后：
```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./modelsdk/mpr/...
```

- [ ] **B2.7 迁移 convert_reader.go 中 3 个旧 converter 到 reader_model.go，删除 convert_reader.go**

```bash
git rm mdl/backend/mpr/convert_reader.go
```

- [ ] **B2.8 提交（每域独立 commit，或合并为一个）**

```bash
git commit -m "feat(modelsdk/mpr): add β-class reader methods — Enumeration/Constant/ScheduledEvent/ImageCollection/Navigation/ModuleSettings"
```

---

### Task B3：γ 类 — 中等字段 converter

**ImportMapping / ExportMapping / JsonStructure / DataTransformer / DatabaseConnection**

- [ ] **B3.1 ListDataTransformers**（Name, ContainerID, SourceType, Steps[].Technology）
- [ ] **B3.2 ListJsonStructures + GetJsonStructureByQualifiedName**
- [ ] **B3.3 ListImportMappings + GetImportMappingByQualifiedName**（含 Elements 树，按需映射深度）
- [ ] **B3.4 ListExportMappings + GetExportMappingByQualifiedName**
- [ ] **B3.5 ListDatabaseConnections**（含 Queries[].Parameters[]/TableMappings[]）

每个完成后编译验证，用 mx2026aiday 运行 `describe <entity>` 检查输出字段。

- [ ] **B3.6 提交**

```bash
git commit -m "feat(modelsdk/mpr): add γ-class reader methods — DataTransformer/JsonStructure/ImportMapping/ExportMapping/DatabaseConnection"
```

---

### Task B4：δ 类 — 富字段 converter

**PublishedODataService / PublishedRestService / BusinessEventService / ConsumedODataService / ConsumedRestService / ProjectSettings**

- [ ] **B4.1 ListConsumedODataServices / ListPublishedODataServices**
- [ ] **B4.2 ListConsumedRestServices / ListPublishedRestServices**
- [ ] **B4.3 ListBusinessEventServices**（Definition.Channels[].Messages[].Attributes[]）
- [ ] **B4.4 GetProjectSettings**（Model.AfterStartup/BeforeShutdown, Language）

- [ ] **B4.5 提交**

```bash
git commit -m "feat(modelsdk/mpr): add δ-class reader methods — OData/REST/BusinessEvents/ProjectSettings"
```

---

### Task B5：backend.go 合并双 reader

**Files:** `mdl/backend/mpr/backend.go`

- [ ] **B5.1 确认 modelsdk/mpr.Open 支持读写（非只读）**

```bash
grep -n "ReadOnly\|OpenOptions\|OpenWithOptions" modelsdk/mpr/reader.go | head -10
```

- [ ] **B5.2 合并 b.reader 和 b.msdkReader**

```go
// 删除：
reader     *sdkReader           // *sdk/mpr.Reader
msdkReader *modelsdkmpr.Reader

// 改为：
reader *modelsdkmpr.Reader      // 统一读写 reader
```

- [ ] **B5.3 简化 Connect() 方法**

```go
// 旧：
r, err := sdkOpenReader(path)            // sdk/mpr
mr, err := modelsdkmpr.OpenWithDB(...)   // 第二次 open
b.reader = r
b.msdkReader = mr

// 新：
r, err := modelsdkmpr.Open(path)  // 单次 open
b.reader = r
```

- [ ] **B5.4 删除 sdkReader 类型别名和 sdkOpenReader 函数（已无用）**
- [ ] **B5.5 删除 sdk/mpr import**

```go
// 删除：
sdkmpr "github.com/mendixlabs/mxcli/sdk/mpr"
```

- [ ] **B5.6 全量编译**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./...
```

- [ ] **B5.7 全量测试**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./... 2>&1 | grep -E "FAIL|ok"
```

- [ ] **B5.8 运行时完整验证**

```bash
# 构建新 binary
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build -o /tmp/mxcli-phase4 ./cmd/mxcli/
# 验证各命令
/tmp/mxcli-phase4 -p "/mnt/data_sdd/gh/Mx2026AIDay/Factory Management.mpr" -c "show structure"
/tmp/mxcli-phase4 -p "/mnt/data_sdd/gh/Mx2026AIDay/Factory Management.mpr" -c "show enumerations"
/tmp/mxcli-phase4 -p "/mnt/data_sdd/gh/Mx2026AIDay/Factory Management.mpr" -c "describe constant FactoryManagement.CONST_FactoryLocation_IsUS"
```

- [ ] **B5.9 提交**

```bash
git add mdl/backend/mpr/backend.go
git commit -m "refactor(backend): merge dual reader — b.reader is now *modelsdkmpr.Reader, remove sdk/mpr dependency"
```

---

## Phase 5：删除 sdk/mpr 整目录

**前置条件：** Phase 4A + 4B 全部完成，0 个 sdk/mpr import。

- [ ] **5.1 确认零残留 import**

```bash
grep -r '"github.com/mendixlabs/mxcli/sdk/mpr"' . --include="*.go"
# 期望：空输出
```

- [ ] **5.2 删除整目录**

```bash
git rm -r sdk/mpr/
```

- [ ] **5.3 全量编译**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./...
```

- [ ] **5.4 全量测试**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./...
```

- [ ] **5.5 提交**

```bash
git commit -m "feat(sdk): retire sdk/mpr — delete entire directory

All sdk/mpr functionality has been migrated:
- Read path: modelsdk/mpr.Reader (reader_model.go) via mprread free functions
- Write path: modelsdk/mpr.UnitWriter (completed in Stage 3.x)
- project_tree.go: fully on mprread + mmpr.Reader
- backend.go: single *modelsdkmpr.Reader for read and write
- parser_*.go: ~2500 lines of hand-written BSON parsers eliminated
- sdk/ now contains only: versions/ (version registry)"
```

---

## 进度表

| 任务 | 描述 | 文件 | 方法数 | 状态 | Commit |
|------|------|------|--------|------|--------|
| P0 | 修复 generic.go double IO | generic.go | — | ✅ 完成 | 6f5efb6c |
| A1 | mprread 新增 9 个 Gen 函数 | reader_documents.go | 9 | ⬜ 待做 | — |
| A2 | 薄型方法迁移（15 个） | project_tree.go | 15 | ⬜ 待做 | — |
| A3 | Gen 变体方法迁移（8 个） | project_tree.go | 8 | ⬜ 待做 | — |
| A4 | 富型 helper 改 gen 类型（5 个） | project_tree.go | 5 | ⬜ 待做 | — |
| A5 | 删除 project_tree.go 的 sdk/mpr import | project_tree.go | — | ⬜ 待做 | — |
| B1 | α 类：AgentEditor passthrough（4 个） | reader_model.go | 4 | ⬜ 待做 | — |
| B2 | β 类：薄 converter（Enum/Const/Sched/Image/Nav/ModSet） | reader_model.go | 12 | ⬜ 待做 | — |
| B3 | γ 类：中等 converter（DT/Json/ImpMap/ExpMap/DBC） | reader_model.go | 10 | ⬜ 待做 | — |
| B4 | δ 类：富字段 converter（OData/REST/BE/Settings） | reader_model.go | 9 | ⬜ 待做 | — |
| B5 | backend.go 合并双 reader | backend.go | — | ⬜ 待做 | — |
| C1 | 删除 sdk/mpr 整目录 | sdk/mpr/ | — | ⬜ 待做 | — |

**状态说明：** ✅ 完成 / 🔄 进行中 / ⬜ 待做 / ❌ 阻塞

**总 sdk/mpr 方法迁移：**
- modelsdk/mpr.Reader 已有：24 个（无需迁移）
- reader_model.go 待加：35 个（4A+4B+4C+4D）
- **Phase 4 完成后 sdk/mpr import 为 0**

---

## 验收清单（Phase 4 + Phase 5 完成）

```bash
# 1. sdk/mpr 目录不存在
ls sdk/  # 期望：只有 versions/

# 2. 全局零 sdk/mpr import
grep -r '"github.com/mendixlabs/mxcli/sdk/mpr"' . --include="*.go"
# 期望：空

# 3. 全量编译
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./...
# 期望：无错误

# 4. 全量测试
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./...
# 期望：全 ok，无 FAIL

# 5. 运行时验证（mx2026aiday）
mxcli -p "Factory Management.mpr" -c "show structure"  # 数字一致
mxcli -p "Factory Management.mpr" project-tree | wc -l  # 行数一致
mxcli -p "Factory Management.mpr" -c "describe constant FactoryManagement.CONST_FactoryLocation_IsUS"  # 深度字段正确
```

# GraphCatalog 设计文档

**日期:** 2026-06-15  
**状态:** 待实现  
**替换目标:** `mdl/catalog/`（SQLite，46 张表）

---

## 背景与目标

当前 `mdl/catalog/` 使用 SQLite 存储项目元数据，`show callers/callees/references/impact` 通过 `WITH RECURSIVE` CTE 做图遍历。主要问题：

- FULL 构建慢（大型项目 ~30s）
- 递归 CTE 是 SQLite 最弱的操作，单跳约 10ms
- `SELECT FROM CATALOG.xxx` ad-hoc 查询将被丢弃（用户确认）
- 46 张表维护负担重

目标：用 `internal/mxgraph` 内存图引擎彻底替换 SQLite catalog，消费方依赖接口而非具体类型。

---

## 性能基准（红线，CI 守护）

| 操作 | 当前 SQLite 基准 | mxgraph 目标 |
|------|----------------|-------------|
| `Callers` 单跳 | ~10ms（CTE） | <1ms（BFS 邻接索引） |
| `Edges(nodeID)` | 不适用 | O(degree)，不随图规模增大 |
| 图构建（中型项目） | FULL ~30s | <3s（纯内存，并行适配器） |
| gob snapshot 写入 | 不适用 | <500ms |

所有基准以 `BenchmarkXxx` 形式落入 `_bench_test.go`，在 CI 中用 `-benchtime=1s` 守护。

---

## 架构

### 层次结构

```
internal/mxgraph/               ← 通用图引擎（不含 Mendix 语义）
    engine.go                   ← Graph struct，AddNode/AddEdge/RemoveNode/Apply
    query.go                    ← FindPathSchemas / Traverse / ExplorePath
    adapter.go                  ← IndexAdapter 接口 / IndexManager / EventSink
    persist.go                  ← MarshalSnapshot / UnmarshalSnapshot / DeltaWriter
    types.go                    ← Node / Edge / Event / Direction 等基础类型
    adapter/mpr/
        domainmodel.go          ← DomainModelAdapter
        microflow.go            ← MicroflowAdapter
        page.go                 ← PageAdapter
        security.go             ← SecurityAdapter
        enumeration.go          ← EnumerationAdapter

mdl/graphcatalog/               ← Mendix 语义层（替换 mdl/catalog/）
    reader.go                   ← 消费者接口（ISP）
    nodes.go                    ← 类型化节点 struct
    graph.go                    ← ProjectGraph（实现所有接口）
    persist.go                  ← gob snapshot 读写（.mxcli/graph.gob）
    mock/
        mock.go                 ← MockProjectGraph（Func 字段 stub）

mdl/catalog/                    ← 迁移完成后整包删除（Phase 7）
```

---

## mxgraph Bug 修复（实现前置条件）

在任何新功能开发前，必须先修复以下三个 bug，并以 benchmark 守护：

### Bug 1：`Edges()` O(E) 全扫描
**文件:** `internal/mxgraph/engine.go:209`  
**问题:** 遍历全部 `g.edges` map 匹配 From/To，不使用 outEdges/inEdges 索引  
**修复:** 按方向查 outEdges/inEdges，O(degree)

```go
// 修复后
func (g *Graph) Edges(id NodeID, dir Direction, relTypes ...RelType) []*Edge {
    g.mu.RLock()
    defer g.mu.RUnlock()
    // Outbound: 查 outEdges[id]，Inbound: 查 inEdges[id]，Both: 合并
}
```

### Bug 2：`RemoveNode()` O(E) 边扫描
**文件:** `internal/mxgraph/engine.go:107`  
**问题:** `for eid, e := range g.edges { if e.From == id || e.To == id { delete } }`  
**修复:** 用 outEdges/inEdges 收集需删除的 edge ID，不全扫描

### Bug 3：`FindPathSchemas()` DFS O(N²) map 复制
**文件:** `internal/mxgraph/query.go:50`  
**问题:** 每步递归 `make(map[NodeID]bool)` + 全量复制 visitedNodes  
**修复:** 用路径局部 `[]NodeID` 记录访问，递归后回退（backtrack）

---

## 图的节点和边设计

### 节点 Label（替代 46 张表）

| Label | 对应 Mendix 类型 | 关键 Props |
|-------|----------------|-----------|
| `Module` | 模块 | `Name`, `AppStoreGuid` |
| `Entity` | 实体 | `Name`, `QualifiedName`, `IsExternal` |
| `Attribute` | 属性 | `Name`, `DataType`, `IsRequired` |
| `Association` | 关联 | `Name`, `QualifiedName`, `Owner` |
| `Microflow` | 微流 | `Name`, `QualifiedName`, `ReturnType` |
| `Nanoflow` | 纳流 | `Name`, `QualifiedName` |
| `Page` | 页面 | `Name`, `QualifiedName`, `LayoutRef` |
| `Layout` | 布局 | `Name`, `LayoutType` |
| `Snippet` | 代码片段 | `Name`, `QualifiedName` |
| `Enumeration` | 枚举 | `Name`, `QualifiedName` |
| `EnumValue` | 枚举值 | `Name`, `Caption` |
| `Constant` | 常量 | `Name`, `DataType` |
| `Widget` | 小部件 | `WidgetType`, `Name` |
| `JavaAction` | Java 动作 | `Name`, `QualifiedName` |
| `Permission` | 权限规则 | `EntityName`, `ModuleRole`, `AccessRights` |
| `RoleMapping` | 角色映射 | `UserRole`, `ModuleRole` |
| `DatabaseConnection` | 数据库连接 | `Name`, `DatabaseType` |
| `Workflow` | 工作流 | `Name`, `QualifiedName` |

### 边 RelType（替代 refs 表）

| RelType | From | To | 含义 |
|---------|------|----|----|
| `HAS_ENTITY` | Module | Entity | 模块包含实体 |
| `HAS_ATTRIBUTE` | Entity | Attribute | 实体含属性 |
| `HAS_ASSOCIATION` | Module | Association | 模块含关联 |
| `CALLS` | Microflow/Nanoflow | Microflow/Nanoflow | 调用关系 |
| `CREATES` | Microflow | Entity | 创建对象 |
| `RETRIEVES` | Microflow | Entity | 检索对象 |
| `SHOWS_PAGE` | Microflow | Page | 打开页面 |
| `HAS_ACTION` | Widget/Page | Microflow/Nanoflow | 按钮/动作调用 |
| `HAS_DATASOURCE` | Page/Widget | Microflow/Entity | 数据源 |
| `HAS_LAYOUT` | Page | Layout | 页面使用布局 |
| `HAS_PERMISSION` | Entity | Permission | 实体的访问规则 |
| `GENERALIZES` | Entity | Entity | 实体继承 |

---

## 接口设计（ISP）

### 消费者接口（`mdl/graphcatalog/reader.go`）

```go
// DomainReader — linter 域模型规则使用
type DomainReader interface {
    Entities(module string) []EntityNode
    Entity(qualifiedName string) *EntityNode
    Attributes(entityQualifiedName string) []AttributeNode
    Associations(module string) []AssociationNode
}

// BehaviorReader — linter 行为规则使用
type BehaviorReader interface {
    Microflows(module string) []MicroflowNode
    Pages(module string) []PageNode
    Snippets(module string) []SnippetNode
    Enumerations(module string) []EnumerationNode
}

// SecurityReader — linter 安全规则使用
type SecurityReader interface {
    Permissions() []PermissionNode
    RoleMappings() []RoleMappingNode
}

// ExtensionReader — linter 扩展规则使用
type ExtensionReader interface {
    Widgets(page string) []WidgetNode
    DatabaseConnections() []DatabaseConnectionNode
}

// TraversalReader — executor search 命令使用
type TraversalReader interface {
    Callers(qualifiedName string, transitive bool) []CallEdge
    Callees(qualifiedName string, transitive bool) []CallEdge
    Impact(qualifiedName string) []RefEdge
    References(qualifiedName string) []RefEdge
}

// LintReader — linter 全量接口（聚合）
type LintReader interface {
    DomainReader
    BehaviorReader
    SecurityReader
    ExtensionReader
}
```

### 类型化节点（`mdl/graphcatalog/nodes.go`）

```go
type EntityNode struct {
    ID            string
    Name          string
    QualifiedName string
    Module        string
    IsExternal    bool
}

type CallEdge struct {
    Caller string // QualifiedName
    Callee string
    Depth  int    // transitive 时的层级
}

type RefEdge struct {
    Source  string
    Target  string
    RefKind string // "CALLS" / "CREATES" / ...
}
// ... 其余节点 struct 类似
```

### ProjectGraph（`mdl/graphcatalog/graph.go`）

```go
type ProjectGraph struct {
    mgr *mxgraph.IndexManager
}

// 编译期接口检查
var _ LintReader = (*ProjectGraph)(nil)
var _ TraversalReader = (*ProjectGraph)(nil)

func (pg *ProjectGraph) Entities(module string) []EntityNode {
    nodes := pg.mgr.Query().FindNodes("Entity", map[string]any{"Module": module})
    // 转换为 EntityNode
}

func (pg *ProjectGraph) Callers(name string, transitive bool) []CallEdge {
    // 用 graph.Traverse(id, "CALLS", depth) 或 BFS
}
```

---

## Mock（`mdl/graphcatalog/mock/mock.go`）

```go
type MockProjectGraph struct {
    EntitiesFunc     func(module string) []graphcatalog.EntityNode
    CallersFunc      func(name string, transitive bool) []graphcatalog.CallEdge
    // ... 每个接口方法一个 Func 字段
}

func (m *MockProjectGraph) Entities(module string) []graphcatalog.EntityNode {
    if m.EntitiesFunc != nil {
        return m.EntitiesFunc(module)
    }
    panic("MockProjectGraph.Entities not configured")
}
```

---

## 持久化（替换 catalog.db）

- 路径：`{project_dir}/.mxcli/graph.gob`
- 格式：`mxgraph.MarshalSnapshot()` 输出的 gob 二进制
- 缓存失效：与当前一致，检查 `.mpr` 修改时间 + Mendix 版本
- 增量更新：`DeltaWriter` 追加 delta log（可选，Phase 2）

---

## 迁移计划（高层）

1. **Phase 1：修 mxgraph bug + benchmark 守护**（前置，不触碰 catalog）
2. **Phase 2：拆分 mpr 适配器**（5 个单域适配器 + 单元测试）
3. **Phase 3：实现 `mdl/graphcatalog/`**（接口 + ProjectGraph + Mock）
4. **Phase 4：迁移 linter**（`LintContext` 改依赖 `LintReader` 接口）
5. **Phase 5：迁移 executor**（exec_context + cmd_catalog + cmd_search）
6. **Phase 6：迁移 serve.go**（treemap 改用 `BehaviorReader`）
7. **Phase 7：删除 `mdl/catalog/`**

每个 Phase 独立可测，不允许跨 Phase 提交。

---

## 丢弃的功能

- `SELECT FROM CATALOG.xxx` — 不迁移，直接删除
- `REFRESH CATALOG FULL/SOURCE` — 改为 `REFRESH GRAPH`
- SQLite catalog.db 文件 — 迁移后删除

---

## 不在本次范围内

- Delta 增量更新（DeltaWriter 集成）— Phase 2 后续
- LSP hover 依赖 catalog — 单独评估
- Watch 模式（文件变更自动更新图）— 单独评估

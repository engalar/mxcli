# mxgraph Security/Authorization Index — Design & Implementation Plan

## 一、真实使用场景推演

### 场景 1：开发者运行 `mxcli lint` （安全合规检查）

```
$ mxcli lint -p app.mpr

SEC001: Entity "HD.Ticket" has no access rules
SEC004: Guest access enabled — review anonymous entity access
SEC007: Unconstrained anonymous READ on "HD.Ticket"
```

**当前行为**（每次全量 MPR 扫描）：
1. `ListDomainModelsGen()` → 10 DMs，遍历每个 Entity → 每个 AccessRule → 11 ms
2. `GetProjectSecurityGen()` → 1 unit → <1 ms
3. `ListPagesGen()` → 27 页面 → <1 ms
4. `ListMicroflowsGen()` → 36 MFs → <1 ms
5. `getPageModel()` × 27 → 解析 widget tree XML → ~50 ms
6. **合计：~65 ms / 次**

**mxgraph 加速后**：
1. 图已建好（一次建图成本）
2. `g.FindNodes("AccessRule", ...)` → µs 级
3. `g.FindNodes("Entity", {"HasAccessRules": false})` → µs 级
4. **每次 lint：~50 µs**（4-5 个图查询）
5. **加速比：~1300x**（65 ms → 50 µs）

### 场景 2：`mxcli check --references`（访问间隙分析）

```
$ mxcli check -p app.mpr --references

ACCESS-001: AgentRole can see HD.Ticket_Detail (reads "HD.Ticket") but has no READ grant
ACCESS-003: CustomerRole can reach HD.ACT_Escalate but has no execute grant
```

**当前行为**（每次全量 MPR 扫描 + widget tree XML 解析）：
1. `buildEntityGrants()` → 遍历所有 Entity AccessRules → 11 ms
2. `loadPages()` + `buildPageModels()` → 27 页面 XML 解析 → ~50 ms
3. `loadMicroflows()` + `buildMFMeta()` → 36 MFs → <1 ms
4. `buildDocumentGrants()` → 交叉引用 → <1 ms
5. `detectGaps()` → 所有 (userRole, page) 对 × widget refs → 5 ms
6. **合计：~70 ms / 次**

**mxgraph 加速后**：
1. 图查询 `EntityAccessRules` → 1 µs
2. 图查询 `PageAllowedRoles` → 1 µs
3. 图查询 `PageEntityRefs` + `PageMFRefs` → 2 µs
4. 图遍历 `detectGaps` → 10 µs
5. **每次 check：~15 µs**
6. **加速比：~4600x**

### 场景 3：AI 助手实时安全诊断（对话中）

```
用户：AgentRole 能访问哪些实体？哪些能写？
AI：查询 mxgraph → 3 次图查询 → ~5 µs → 直接回答
```

**当前**：AI 需要写 MDL 命令 `SHOW ACCESS ON ENTITY HD.Ticket`，触发一遍 MPR 扫描。
**mxgraph**：AI 直接查询内存图，零 MPR 开销。

### 场景 4：CI 流水线批量检测 50 个项目

```
for proj in $(ls projects/); do
  mxcli lint -p "$proj/app.mpr"
done
```

**当前**：50 × (建图 20ms + 安全扫描 65ms) = **4.25 秒 / 次**
**mxgraph**（一次建图后增量）：50 × (增量更新 2ms + 图查询 50µs) = **~100 ms / 次**
**加速比：~42x**

---

## 二、需要索引的数据（从使用场景反推）

### 2.1 Entity Access Rules（场景 1 SEC001、场景 2 ACCESS-001）

来源：`DomainModels$DomainModel` → `Entities[]` → `AccessRules[]`

| 字段 | gen type 方法 | 用途 |
|------|-------------|------|
| Entity QualifiedName | derived | 识别所属实体 |
| ModuleRole QualifiedName | `ModuleRolesQualifiedNames()` | 授权给谁 |
| DefaultMemberAccessRights | `DefaultMemberAccessRights()` | "ReadOnly" / "ReadWrite" / "" |
| AllowCreate | `AllowCreate()` | 创建权限 |
| AllowDelete | `AllowDelete()` | 删除权限 |
| XPathConstraint | derived from `XPathConstraint()` | 行级约束 |

### 2.2 Document-Level Grants（场景 2 ACCESS-003）

来源：`Forms$Page.AllowedRolesQualifiedNames()`
      `Microflows$Microflow.AllowedModuleRolesQualifiedNames()`

| 字段 | gen type 方法 | 用途 |
|------|-------------|------|
| Document QualifiedName | derived | 被授权的页面/MF |
| ModuleRole QualifiedName | `AllowedRolesQualifiedNames()` | 授权给谁 |

### 2.3 Page Widget → Entity/MF References（场景 2 ACCESS-001/003）

来源：Widget tree 中 datasource entity refs + OnClick microflow refs

当前 `collectWidgetRefs` 解析 page model XML 来提取。未来可以通过扩展 WidgetInstanceAdapter 直接从 raw BSON 提取 dataSource/action 引用。

### 2.4 Microflow ApplyEntityAccess（场景 2）

来源：`Microflows$Microflow.ApplyEntityAccess()`

---

## 三、SOLID 代码架构

### 3.1 接口层（I — 接口隔离）

```go
// mdl/graphcatalog/reader.go 追加

// EntityAccessReader 读取实体访问规则。
type EntityAccessReader interface {
    EntityAccessRules(entityQN string) []AccessRuleNode
    EntityAccessRulesForRole(moduleRoleQN string) []AccessRuleNode
    EntitiesWithMissingAccessRules() []EntityNode  // 无任何 AccessRule 的实体
}

// DocumentGrantReader 读取页面/微流的授权信息。
type DocumentGrantReader interface {
    PageAllowedRoles(pageQN string) []string  // 返回 ModuleRole QN 列表
    MFAllowedRoles(mfQN string) []string
    ApplyEntityAccess(mfQN string) bool
}

// PageRefReader 读取页面 widget 树中的实体/微流引用。
type PageRefReader interface {
    PageEntityRefs(pageQN string) []string  // 页面读取了哪些实体
    PageMFRefs(pageQN string) []string      // 页面调用了哪些微流
}
```

### 3.2 节点定义（graphcatalog/nodes.go 追加）

```go
type AccessRuleNode struct {
    ID               string
    EntityQN         string
    ModuleRoleQN     string
    CanRead          bool
    CanWrite         bool
    CanCreate        bool
    CanDelete        bool
    XPathConstraint  string
}

type PageGrantNode struct {
    PageQN       string
    ModuleRoleQN string
}

type MFGrantNode struct {
    MFQN           string
    ModuleRoleQN   string
    ApplyEntityAccess bool
}
```

### 3.3 适配器层（S — 单一职责）

```
internal/mxgraph/adapter/mpr/
  ├── security.go                ← 已有：UserRole + ModuleRole 节点
  ├── access_rule.go             ← NEW：AccessRule 节点
  ├── document_grant.go          ← NEW：Page/MF AllowedRoles 索引
  └── page_ref.go                ← NEW：Page widget tree entity/MF refs
```

### 3.4 数据流（D — 依赖倒置）

```
AccessRuleAdapter → EventSink → Graph
     │                              │
     │ (依赖 RoleManager 接口)       │ (graphcatalog)
     ▼                              ▼
entityAccessReader ──────────→ Analyzer.detectGaps()
                                     │
DocumentGrantAdapter → Graph ────────┤
PageRefAdapter      → Graph ────────┘
(MicroflowAdapter)  → Graph ────────┘ (ApplyEntityAccess prop)
```

---

## 四、实现计划

### Phase 1: AccessRuleAdapter

| 操作 | 文件 |
|------|------|
| Create | `internal/mxgraph/adapter/mpr/access_rule.go` |
| Create | `internal/mxgraph/adapter/mpr/access_rule_test.go` |
| Create | `mdl/graphcatalog/security_nodes.go`（AccessRuleNode 等） |
| Modify | `mdl/graphcatalog/reader.go`（EntityAccessReader 接口） |
| Modify | `mdl/graphcatalog/graph.go`（实现 + 编译期检查） |
| Modify | `mdl/graphcatalog/mock/mock.go`（追加 mock 方法） |
| Modify | `mdl/executor/cmd_graph.go`（注册新适配器） |

### Phase 2: DocumentGrantAdapter

| 操作 | 文件 |
|------|------|
| Create | `internal/mxgraph/adapter/mpr/document_grant.go` |
| Modify | `mdl/graphcatalog/reader.go`（DocumentGrantReader 接口） |
| Modify | `mdl/graphcatalog/graph.go`（实现）|
| Modify | `mdl/graphcatalog/mock/mock.go`（追加）|
| Modify | `mdl/executor/cmd_graph.go`（注册）|

### Phase 3: PageRefAdapter

| 操作 | 文件 |
|------|------|
| Create | `internal/mxgraph/adapter/mpr/page_ref.go` |
| Modify | `mdl/graphcatalog/reader.go`（PageRefReader 接口） |
| Modify | `mdl/graphcatalog/graph.go`（实现）|
| Modify | `mdl/graphcatalog/mock/mock.go`（追加）|

### Phase 4: Migrate Analyzer to graph

| 操作 | 文件 |
|------|------|
| Modify | `mdl/executor/security_access_check.go`（改用 graph 查询） |
| Delete/Deprecate | MPR 全扫描路径 |

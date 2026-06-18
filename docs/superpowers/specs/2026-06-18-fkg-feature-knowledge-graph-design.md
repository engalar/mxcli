# Feature Knowledge Graph (FKG) — 设计规范

**日期**：2026-06-18  
**状态**：待实现  
**主要消费者**：AI（Claude）  

---

## 1. 问题陈述

mxcli 的功能和 MDL 语法分散在三处：
- `cmd/mxcli/syntax/`：118 个 `SyntaxFeature`，通过 `mxcli syntax <topic>` 访问
- `.claude/skills/`：72 个 skill 文件，由 AI 按需加载
- `docs/`：379 个 Markdown 文件

三者之间没有显式关系，AI 在处理任务时需要猜测"应该看哪里"，发现成本高。

**目标**：构建一张手工策划的**能力本体图**，让 AI 能从任务先验节点（如"页面"、"安全"）出发，通过图遍历系统性地探索相关能力，再选择性深入。

---

## 2. 方案决策

| 决策项 | 结论 | 理由 |
|--------|------|------|
| 主要消费者 | AI（Claude） | 人工浏览不是目标 |
| 独立系统 | 是，不扩展现有命令 | 系统性核心功能，不污染现有命令树 |
| 底层引擎 | 复用 `internal/mxgraph` | 已有完整图引擎，无需重复造轮子 |
| 本体定义 | 手工策划，Go 代码 | 语义精准；Go adapter 模式与现有 mxgraph 一致 |
| 访问方式 | CLI + daemon 两路 | 前置规划（CLI）+ 即时查询（daemon 扩展，二期） |
| 设计原则 | 严格遵守 SOLID | 独立演进，易测试，可扩展 |

---

## 3. 架构

### 3.1 分层（依赖方向只向下）

```
cmd/mxcli/cmd_onto.go          ← CLI 入口，依赖 fkg.Querier 接口
        ↓
internal/fkg/querier.go        ← Querier 接口（I 原则）
        ↓ implemented by
internal/fkg/fkg.go            ← Graph 组装 + Querier 实现（S 原则）
        ↓
internal/fkg/concepts/         ← 手工策划的概念 adapter（O 原则，每文件一概念）
        ↓ all implement
internal/mxgraph.Adapter       ← 已有接口（L 原则）
        ↓ feeds into
internal/mxgraph.Graph         ← 已有图引擎
```

### 3.2 包结构

```
internal/fkg/
├── types.go          ← Label 常量、RelType 常量（FKG 专属命名空间）
├── querier.go        ← Querier 接口 + 输出类型定义
├── fkg.go            ← New()工厂、Graph 组装、Querier 实现
└── concepts/
    ├── registry.go   ← Register() + All()，扩展点（O 原则）
    ├── page.go
    ├── microflow.go
    ├── entity.go
    ├── security.go
    ├── navigation.go
    ├── widget.go
    ├── workflow.go
    └── integration.go

cmd/mxcli/
└── cmd_onto.go       ← cobra 命令定义，只依赖 fkg.Querier
```

---

## 4. 图本体

### 4.1 节点类型（`mxgraph.Label`）

| Label | 含义 | 创建者 |
|-------|------|--------|
| `Concept` | 顶层领域概念 | 手工策划，`concepts/*.go` |
| `SyntaxFeature` | MDL 语法条目 | 手工引用 syntax registry path |
| `Skill` | `.claude/skills/` 文件 | 手工引用 skill 名称 |
| `Doc` | 文档路径 | 手工引用 doc 路径 |

### 4.2 边类型（`mxgraph.RelType`）

| RelType | 方向 | 语义 |
|---------|------|------|
| `SPECIALIZES` | SubConcept → Concept | 子概念（DataGrid → Page） |
| `REQUIRES` | Concept → Concept | 强依赖（Widget → Page） |
| `RELATED_TO` | Concept ↔ Concept | 软关联（Security ↔ Navigation） |
| `HAS_SYNTAX` | Concept → SyntaxFeature | 对应的 MDL 语法条目 |
| `HAS_SKILL` | Concept → Skill | 对应的操作 skill |
| `HAS_DOC` | Concept → Doc | 对应的文档 |

### 4.3 初始概念节点（一期）

```
Page
├── SPECIALIZES ← DataGrid
├── SPECIALIZES ← Form (DataView)
├── SPECIALIZES ← Layout
├── HAS_SYNTAX → page.create, page.alter, page.widget.*
├── HAS_SKILL  → mendix/create-page, mendix/alter-page, mendix/overview-pages
└── RELATED_TO → Microflow, Navigation, Security

Microflow
├── SPECIALIZES ← Nanoflow
├── HAS_SYNTAX → microflow.create, microflow.variables, microflow.retrieve, ...
├── HAS_SKILL  → write-microflows, write-nanoflows, patterns-data-processing
└── RELATED_TO → Entity, Page, Integration

Entity
├── HAS_SYNTAX → entity.create, association.create, enumeration.create
├── HAS_SKILL  → generate-domain-model, mendix/associations
└── REQUIRES   → Security

Security
├── HAS_SYNTAX → security.grant, security.revoke, security.role
├── HAS_SKILL  → manage-security
└── RELATED_TO → Entity, Page, Microflow, Navigation

Navigation
├── HAS_SYNTAX → navigation.profile, navigation.menu
├── HAS_SKILL  → manage-navigation
└── REQUIRES   → Page, Security

Widget
├── SPECIALIZES ← PluggableWidget
├── HAS_SYNTAX → widget.pluggable, widget.builtin.*
├── HAS_SKILL  → mendix/custom-widgets
└── REQUIRES   → Page

Workflow
├── HAS_SYNTAX → workflow.create, workflow.activity
└── RELATED_TO → Microflow, Entity, Security

Integration
├── SPECIALIZES ← REST
├── SPECIALIZES ← Database
├── HAS_SYNTAX → integration.rest, integration.database
└── RELATED_TO → Microflow
```

---

## 5. 接口与类型

### 5.1 Querier 接口（I 原则）

```go
// internal/fkg/querier.go

// Querier is the only interface the CLI layer depends on.
type Querier interface {
    Explore(id string, depth int) (*ExploreResult, error)
    Path(from, to string) ([]PathSchema, error)
    Schema() *SchemaResult
}
```

### 5.2 输出类型（S 原则，每类型单一职责）

```go
type NodeSummary struct {
    ID      string
    Label   string  // "Concept" | "SyntaxFeature" | "Skill" | "Doc"
    Name    string
    Summary string
}

type EdgeSummary struct {
    RelType string
    From    string
    To      string
}

type ExploreResult struct {
    Seed  NodeSummary
    Nodes []NodeSummary
    Edges []EdgeSummary
}

type PathSchema struct {
    Steps []PathStep
    Label string  // e.g. "Concept→HAS_SYNTAX→SyntaxFeature"
}

type PathStep struct {
    NodeLabel string
    RelType   string
}

type NodeTypeInfo struct {
    Label string
    Count int
}

type EdgeTypeInfo struct {
    RelType string
    Count   int
}

type SchemaResult struct {
    NodeTypes []NodeTypeInfo
    EdgeTypes []EdgeTypeInfo
    Roots     []NodeSummary  // top-level Concept nodes only
}
```

### 5.3 Concept Adapter（O 原则）

```go
// internal/fkg/concepts/registry.go

type ConceptAdapter interface {
    mxgraph.Adapter  // Apply(chan<- mxgraph.Event) error
}

var adapters []ConceptAdapter

func Register(a ConceptAdapter) { adapters = append(adapters, a) }
func All() []ConceptAdapter     { return adapters }
```

```go
// internal/fkg/concepts/page.go — 典型实现

func init() { Register(&PageAdapter{}) }

type PageAdapter struct{}

func (a *PageAdapter) Apply(ch chan<- mxgraph.Event) error {
    emit(ch, node("page", Concept, "Page", "UI pages, layouts, and widgets"))
    emit(ch, node("datagrid", Concept, "DataGrid", "List view with columns and filters"))
    emit(ch, edge("datagrid", "page", Specializes))
    emit(ch, edge("page", "syntax:page.create",  HasSyntax))
    emit(ch, edge("page", "skill:create-page",   HasSkill))
    emit(ch, edge("page", "skill:alter-page",    HasSkill))
    emit(ch, edge("page", "concept:microflow",   RelatedTo))
    return nil
}
```

### 5.4 FKG 工厂（D 原则）

```go
// internal/fkg/fkg.go

// New builds the feature knowledge graph from all registered concept adapters.
// The returned Querier is the only dependency the CLI needs.
func New() (Querier, error) {
    mgr := mxgraph.NewIndexManager()
    for _, a := range concepts.All() {
        mgr.RegisterAdapter(a)
    }
    g, err := mgr.Build(context.Background())
    if err != nil {
        return nil, err
    }
    return &fkgQuerier{graph: g}, nil
}
```

---

## 6. CLI 命令

```
mxcli onto schema                         # 完整本体骨架
mxcli onto explore <id> [--depth 2]       # 邻域展开
mxcli onto path <from> <to>               # 通路结构
```

所有子命令支持 `--format json`，默认人类可读文本。

**示例**：

```bash
$ mxcli onto explore page
Concept: Page
  Specializations: DataGrid, Form, Layout
  Related: Microflow, Navigation, Security
  Syntax (3): page.create, page.alter, page.widget.*
  Skills (3): create-page, alter-page, overview-pages

$ mxcli onto path page security
Path schemas (2 found):
  Page →[RELATED_TO]→ Security
  Page →[REQUIRES]→ Navigation →[RELATED_TO]→ Security

$ mxcli onto schema
Node types: Concept(9)  SyntaxFeature(118)  Skill(72)  Doc(12)
Edge types: HAS_SYNTAX(74)  HAS_SKILL(48)  HAS_DOC(12)  RELATED_TO(22)  REQUIRES(8)  SPECIALIZES(14)
Roots: Page  Microflow  Entity  Security  Navigation  Widget  Workflow  Integration
```

---

## 7. 测试策略

| 层 | 测试方式 |
|----|--------|
| `concepts/*.go` | 每个 adapter 的 unit test：验证 emit 的节点和边数量及类型 |
| `fkg.go` | integration test：`New()` 后图中节点/边总数符合预期 |
| `Querier` | table-driven tests：Explore/Path/Schema 的输出结构正确 |
| `cmd_onto.go` | `cobra.Command` smoke test：子命令注册、`--format json` 输出可解析 |

---

## 8. 非目标（一期）

- daemon 集成（项目内容图 + FKG 合并查询）——二期
- 自动从 skill/doc 文件派生节点——手工策划优先，自动化是优化
- 语义/向量搜索——图遍历已足够 AI 使用
- Web UI 可视化

---

## 9. 文件影响范围

**新增**（不修改任何现有文件）：
- `internal/fkg/` （全新包）
- `cmd/mxcli/cmd_onto.go`

**不修改**：
- `internal/mxgraph/`（只复用接口和类型）
- `cmd/mxcli/syntax/`（SyntaxFeature 由 FKG 手工引用，不侵入）
- `.claude/skills/`（Skill 由 FKG 手工引用，不侵入）

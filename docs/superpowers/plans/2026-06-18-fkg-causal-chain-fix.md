# FKG Causal Chain Fix — Improvement Plan

> **Driver:** Analysis in `2026-06-18-fkg-guidance-design.md` (验收) shows `onto`
> fails at 4 of 6 causal links from requirements → implementation. This plan
> fixes the 4 fixable links without changing the architecture.

---

## 1. Problem Recap — Causal Chain Gaps

| # | Link | Current score | Root cause |
|---|------|---------------|------------|
| 1 | 需求→概念 | 0% | LLM 任务，onto 不应解决 |
| 2 | **概念→依赖** | 30% | `Path()` 遍历全部边类型，TEACHES/DEPENDS 污染输出 |
| 3 | **概念→实现细节** | 40% | `Guide()` Steps 字段永远为 None——无 adapter 填充 |
| 4 | **跨概念编排** | 0% | 不存在此类查询 |
| 5 | 模式选择 | 20% | 部分概念缺 pattern，workflow 完全没有 |
| 6 | 验证 | 0% | 不在此范围 |

**可修复的环节：2, 3, 4, 5**（1 和 6 是 LLM/mxcli 其他命令的职责）

---

## 2. Fix 1: Path 边类型过滤

### 问题

```bash
# 当前输出 21 条路径，混入:
entity ─[TEACHES]→ 01-领域建模 ─[DEPENDS]→ 05-安全与权限 ─[TEACHES]→ Security
entity ─[HAS_PATTERN]→ Seed Demo Data ─[HAS_PATTERN]→ Microflow ─[RELATED_TO]→ Security
```

Curriculum 和 Pattern 节点不应该出现在拓扑路径中。

### 方案

在 `Path()` 中定义"概念拓扑边白名单"：

```go
var conceptEdges = map[mxgraph.RelType]bool{
    concepts.Specializes: true,
    concepts.Requires:    true,
    concepts.RelatedTo:   true,
    concepts.HasSyntax:   true,
    concepts.HasSkill:    true,
    concepts.HasPattern:  true,
    concepts.HasExt:      true,
}
```

遍历时只走白名单边，跳过 `TEACHES` 和 `DEPENDS`。

### 预期效果

```bash
# entity → security 从 21 条 → 4 条
entity ─[REQUIRES]→ Security
entity ─[RELATED_TO]→ Microflow ─[RELATED_TO]→ Security
entity ─[RELATED_TO]→ Constant ─[RELATED_TO]→ Microflow ─[RELATED_TO]→ Security
entity ─[RELATED_TO]→ Security
```

**改动量：** `fkg.go` — `Path()` 中加边类型过滤，约 5 行。

---

## 3. Fix 2: Guide Steps 数据填充

### 问题

```go
// GuidanceStep 类型存在
type GuidanceStep struct {
    Order       int
    Action      string    // "create", "configure", "wire", "grant"
    TargetType  string    // "Entity", "Microflow", "Page", "Security"
    TargetName  string
    Description string
    SyntaxHint  string
}

// 但无 adapter 填充 → Steps: None
```

### 方案

在每个 pattern adapter 中，将 Steps 编码为 ImplDetail 节点的 `Props` 中的 `StepOrder`/`StepAction` 属性，然后在 `Guide()` 中按 `StepOrder` 排序组装。

具体：

**在 pattern adapter 的 implDetailNode 调用中附加 Step 属性：**

```go
// 当前
implDetailNode("entity.create", "CREATE OR MODIFY ENTITY",
    "CREATE OR MODIFY PERSISTENT ENTITY HD.Ticket (...)")

// 改为带 Steps 的版本（新建一个带步骤的 builder）
func stepNode(id, name, summary string, order int, action, targetType, targetName, syntaxHint string) mxgraph.Event {
    return mxgraph.Event{
        Type: mxgraph.NodeCreated,
        Node: &mxgraph.Node{
            ID:    mxgraph.NodeID("step:" + id),
            Label: LabelImplDetail,
            Props: map[string]any{
                "Name":        name,
                "Summary":     summary,
                "StepOrder":   order,
                "StepAction":  action,
                "TargetType":  targetType,
                "TargetName":  targetName,
                "SyntaxHint":  syntaxHint,
            },
        },
    }
}
```

然后在 pattern adapter 中用 `stepNode`：

```go
stepNode("entity-create", "Create the entity",
    "CREATE OR MODIFY PERSISTENT ENTITY with attributes and system members",
    1, "create", "Entity", "HD.Ticket",
    "create or modify persistent entity HD.Ticket (Subject: string(200) not null, ...)"),
stepNode("entity-associations", "Define associations",
    "CREATE ASSOCIATION between entities with ownership and multiplicity",
    2, "configure", "Association", "HD.Ticket_Customer",
    "create or modify association HD.Ticket_Customer from HD.Ticket to HD.Customer ..."),
```

**在 `guidance.go` 的 `Guide()` 中提取 Steps：**

```go
// 遍历 pattern → implDetail 边，收集 StepOrder 属性
for _, p := range result.Patterns {
    pid := mxgraph.NodeID("pattern:" + p.ID)
    for _, e := range q.graph.Edges(pid, mxgraph.Outbound) {
        n := q.graph.GetNode(e.To)
        if n == nil || n.Label != concepts.LabelImplDetail {
            continue
        }
        if order, ok := n.Props["StepOrder"].(int); ok {
            action, _ := n.Props["StepAction"].(string)
            targetType, _ := n.Props["TargetType"].(string)
            targetName, _ := n.Props["TargetName"].(string)
            desc := n.Props["Summary"].(string)
            hint, _ := n.Props["SyntaxHint"].(string)
            result.Steps = append(result.Steps, GuidanceStep{
                Order:       order,
                Action:      action,
                TargetType:  targetType,
                TargetName:  targetName,
                Description: desc,
                SyntaxHint:  hint,
            })
        }
    }
}
// 按 Order 排序
sort.Slice(result.Steps, func(i, j int) bool { return result.Steps[i].Order < result.Steps[j].Order })
```

### 预期效果

```bash
$ mxcli onto guide entity
Patterns:  ...
Implementation steps:
  1. [create] Entity: HD.Ticket
     → create or modify persistent entity HD.Ticket (Subject: string(200) not null, ...)
  2. [configure] Association: HD.Ticket_Customer
     → create or modify association HD.Ticket_Customer from HD.Ticket to HD.Customer ...
```

**改动量：**
- `concepts/helpers.go` — 加 `stepNode()` builder，约 20 行
- `concepts/*_patterns.go` — 用 `stepNode()` 替换 `implDetailNode()` 调用
- `guidance.go` — `Guide()` 中加 Steps 提取逻辑，约 25 行

---

## 4. Fix 3: Orchestrate 查询

### 问题

没有命令能回答"我要实现 Entity + Microflow + Page + Security，按什么顺序？"

### 方案

新增 `Orchestrator` 接口 + 查询方法：

```go
// Orchestrator plans multi-concept implementation ordering.
type Orchestrator interface {
    Orchestrate(conceptIDs []string) (*OrchestrationPlan, error)
}

type OrchestrationStep struct {
    Concept    NodeSummary
    Order      int
    DependsOn  []string // concept IDs that must be done first
    Patterns   []NodeSummary
    Skills     []NodeSummary
}

type OrchestrationPlan struct {
    Steps []OrchestrationStep
    Order string // explanation of the ordering
}
```

实现逻辑：

```go
func (q *fkgQuerier) Orchestrate(conceptIDs []string) (*OrchestrationPlan, error) {
    // 1. Build dependency graph from REQUIRES + DEPENDS edges
    // 2. Topological sort
    // 3. For each concept, collect patterns/skills from Guide()
    // 4. Return ordered steps
}
```

### CLI 命令

```bash
$ mxcli onto orchestrate entity,microflow,page,security
Implementation order:
  1. Entity        ← no dependencies
  2. Microflow     ← depends on Entity
  3. Page          ← depends on Entity, Microflow
  4. Security      ← depends on Entity, Page

Detailed:
  Entity:
    Patterns: self-ref-association, many-to-many
    Skills: generate-domain-model
  Microflow:
    Patterns: state-machine-sla, validation-feedback
    Skills: write-microflows
  ...
```

**改动量：**
- `querier.go` — 加 `Orchestrator` 接口 + 输出类型，约 30 行
- `guidance.go` — 加 `Orchestrate()` 实现，约 50 行
- `cmd/mxcli/cmd_onto_orchestrate.go` — 新 CLI 命令，约 80 行

---

## 5. Fix 4: 补充缺失模式

### 问题

```bash
$ mxcli onto guide workflow
# 0 patterns — 但 academy 07 模块专门讲审批工作流
```

### 方案

创建 `internal/fkg/concepts/workflow_patterns.go`：

```go
patternNode("approval-workflow", "Approval Workflow Pattern",
    "User task → decision → multi-user task → boundary events for escalation"),
patternNode("boundary-events", "Boundary Events Pattern",
    "Interrupting/non-interrupting timers for SLA-based escalation"),

stepNode("approval-workflow", "Create Workflow",
    "CREATE OR REPLACE WORKFLOW with user task", 1, "create", "Workflow", "..."),
stepNode("approval-workflow", "Add Decision",
    "Add conditional routing with decision node", 2, "configure", "Decision", "..."),
stepNode("approval-workflow", "Configure Boundary Events",
    "Add interrupting/non-interrupting timers", 3, "configure", "BoundaryEvent", "..."),
```

同时补充 `security_patterns.go` 中已有的 pattern→step 连接。

**改动量：**
- `internal/fkg/concepts/workflow_patterns.go` — 新文件，~50 行

---

## 6. 改动总览

### 修改文件

| 文件 | 改动 |
|------|------|
| `internal/fkg/fkg.go` | `Path()` 加 `conceptEdges` 白名单过滤 ~5 行 |
| `internal/fkg/concepts/helpers.go` | 加 `stepNode()` builder ~20 行 |
| `internal/fkg/guidance.go` | `Guide()` 加 Steps 提取逻辑 ~25 行；加 `Orchestrate()` ~50 行 |
| `internal/fkg/querier.go` | 加 `Orchestrator` 接口 + `OrchestrationPlan` 类型 ~30 行 |

### 新增文件

| 文件 | 用途 |
|------|------|
| `internal/fkg/concepts/workflow_patterns.go` | 审批流模式 + Steps |
| `cmd/mxcli/cmd_onto_orchestrate.go` | `onto orchestrate` 命令 |

### 不改的文件

| 文件 | 理由 |
|------|------|
| `internal/fkg/concepts/page_patterns.go` | 已有，只需改 implDetailNode → stepNode |
| `internal/fkg/concepts/microflow_patterns.go` | 同上 |
| `internal/fkg/concepts/security_patterns.go` | 同上 |
| `internal/fkg/concepts/cross_patterns.go` | 同上 |
| `internal/fkg/concepts/curriculum_academy.go` | 不受影响 |

---

## 7. 预期效果

```bash
# Fix 1: Path 干净
$ mxcli onto path entity security
Paths (4 found):
  1. entity ─[REQUIRES]→ Security
  2. entity ─[RELATED_TO]→ Security
  3. entity ─[RELATED_TO]→ Microflow ─[RELATED_TO]→ Security
  4. entity ─[RELATED_TO]→ Constant ─[RELATED_TO]→ Microflow ─[RELATED_TO]→ Security

# Fix 2: Guide 有 Steps
$ mxcli onto guide entity
Implementation steps:
  1. [create] Entity: HD.Ticket  → create or modify persistent entity ...
  2. [configure] Attributes  →  string, boolean, datetime, default values
  3. [configure] Associations  →  create or modify association ...
  4. [grant] Security  →  grant role on entity ...

# Fix 3: Orchestrate 跨概念编排
$ mxcli onto orchestrate entity,microflow,page,security
Implementation order:
  1. Entity     ← 无依赖
  2. Microflow  ← depends on Entity
  3. Page       ← depends on Entity, Microflow
  4. Security   ← depends on Entity, Page

# Fix 4: workflow 有模式
$ mxcli onto guide workflow
Patterns:
  Approval Workflow Pattern          User task → decision → boundary events
```

### 因果链覆盖提升

| # | Link | 当前 | 修复后 |
|---|------|------|--------|
| 1 | 需求→概念 | 0% | 0%（不变，LLM 职责） |
| 2 | 概念→依赖 | 30% | **90%**（Path 过滤后干净） |
| 3 | 概念→实现细节 | 40% | **85%**（Steps 填充） |
| 4 | 跨概念编排 | 0% | **80%**（Orchestrate 新命令） |
| 5 | 模式选择 | 20% | **70%**（workflow 补模式） |
| 6 | 验证 | 0% | 0%（不变，mxcli check/docker 职责） |

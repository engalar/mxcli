# FKG Gaps Fix — SOLID 改进方案

> **Driver:** 逐模块推演发现 5 个 P0-P4 缺口。本计划按优先级修复。

---

## P0: Orchestrate 排序方向修正

### 问题

`entity → REQUIRES → security` 被理解成"Entity 依赖于 Security"→ Security 排在 Entity 前。
但实现顺序是：先建 Entity，再配置 Security 规则。

### 根因

`Orchestrate()` 使用 outbound REQUIRES 边构建依赖图：

```go
// 当前 (错误)
for _, e := range q.graph.Edges(info.node.ID, mxgraph.Outbound, concepts.Requires) {
    info.deps[targetID] = true  // Entity.deps = {security: true} → security 先
}

// 修正
for _, e := range q.graph.Edges(info.node.ID, mxgraph.Inbound, concepts.Requires) {
    // security 的入边: entity→REQUIRES→security → security.deps = {entity: true}
    info.deps[sourceID] = true  // 谁 REQUIRE 我，我就是依赖谁 → 我先
}
```

### 方案

`guidance.go` — `Orchestrate()` 中把 `Outbound` 改为 `Inbound`，1 行改动。

```go
for _, e := range q.graph.Edges(info.node.ID, mxgraph.Inbound, concepts.Requires) {
    sourceID := string(e.From)
    ...
    info.deps[sourceID] = true
}
```

### 预期效果

```bash
$ mxcli onto orchestrate entity microflow page security
Implementation order:
  1. Entity              ← 无依赖
  2. Microflow           ← 无依赖
  3. Page                ← 无依赖
  4. Security            ← depends on: entity  (entity→REQUIRES→security 入边)
```

**SOLID:**
- **S**: 只改 Orchestrate 的依赖方向识别
- **O**: 不修改任何接口或类型
- **L**: `fkgQuerier` 行为改变但接口不变
- **I**: `Orchestrator` 接口不变
- **D**: 调用方不受影响

---

## P1: 扩展点补充 stepNode

### 问题

`guide java-action`、`guide js-action`、`guide widget` 无实现步骤（Steps: None）。

### 方案

在每个扩展 adapter 中增加 stepNode + 对应 edges。

**`java_action.go`** 新增：

```go
stepNode("ja-create", "Create Java Action definition",
    "CREATE OR MODIFY JAVA ACTION with parameters and return type",
    1, "create", "JavaAction", "...",
    "create or modify java action HD.JA_HashPassword (Password: string not null) returns string { ... }"),
stepNode("ja-write-code", "Implement Java code",
    "Write imports and code blocks for the Java Action",
    2, "configure", "JavaCode", "...",
    "imports $$ import java.security.MessageDigest; $$ code $$ ... $$"),
stepNode("ja-call-from-mf", "Call from microflow",
    "Use CALL JAVA ACTION in microflow expression",
    3, "wire", "Microflow", "...",
    "call java action HD.JA_VerifyPassword(Password = $Password, HashedPassword = $HashedPassword);"),
edge("pattern:extend-with-java", "step:ja-create", HasSyntax),
edge("pattern:extend-with-java", "step:ja-write-code", HasSyntax),
edge("pattern:extend-with-java", "step:ja-call-from-mf", HasSyntax),
```

但 java_action.go 当前没有 pattern 节点。需要先加 pattern:

```go
patternNode("extend-with-java", "Java Action Extension Pattern",
    "Create Java Action → implement code → call from microflow"),
edge("java-action", "pattern:extend-with-java", HasPattern),
```

同理对 `js_action.go`、`cross_patterns.go`（css-theme）。

### 预期效果

```bash
$ mxcli onto guide java-action
Patterns:
  Java Action Extension Pattern     Create Java Action → implement → call
Implementation steps:
  1. [create] Create Java Action definition ...
  2. [configure] Implement Java code ...
  3. [wire] Call from microflow ...
```

**SOLID:**
- **S**: 每个 adapter 负责自己的 stepNode
- **O**: 纯新增数据和 edges，不改已有逻辑
- **L**: stepNode 与其他 builder 可互换
- **I**: 不影响任何接口
- **D**: Guide() 自动消费新步骤

**改动量：** `java_action.go` +25 行，`js_action.go` +25 行，`cross_patterns.go` +15 行

---

## P2: Nanoflow 实现步骤

### 问题

`guide nanoflow` 返回 0 patterns、0 steps。Nanoflow 是 Microflow 的子概念但无独立引导。

### 方案

在 `microflow_patterns.go` 中增加 nanoflow 专属模式：

```go
patternNode("nanoflow-quick-create", "Nanoflow Quick-Create Pattern",
    "Client-side object creation: create → commit → return, no server round-trip"),
stepNode("nf-create", "Create Nanoflow definition",
    "CREATE OR MODIFY NANOFLOW with parameters",
    1, "create", "Nanoflow", "...",
    "create or modify nanoflow HD.NF_Ticket_QuickCreate (...) returns HD.Ticket as $Ticket { ... }"),
stepNode("nf-create-object", "Create object client-side",
    "Use create + commit for client-side object creation",
    2, "configure", "Object", "...",
    "$Ticket = create HD.Ticket (Subject = $Subject, Status = ...); commit $Ticket;"),
stepNode("nf-return", "Return result",
    "Return the created object to the caller",
    3, "configure", "Return", "...",
    "return $Ticket;"),

edge("nanoflow", "pattern:nanoflow-quick-create", HasPattern),
edge("pattern:nanoflow-quick-create", "step:nf-create", HasSyntax),
edge("pattern:nanoflow-quick-create", "step:nf-create-object", HasSyntax),
edge("pattern:nanoflow-quick-create", "step:nf-return", HasSyntax),
```

### 预期效果

```bash
$ mxcli onto guide nanoflow
Patterns:
  Nanoflow Quick-Create Pattern    Client-side object creation
Implementation steps:
  1. [create] Create Nanoflow definition ...
  2. [configure] Create object client-side ...
  3. [configure] Return result ...
```

**改动量：** `microflow_patterns.go` +25 行

---

## P3: Entity 属性类型引导

### 问题

`guide entity` 有 3 个模式但没有属性类型选择（string vs boolean vs datetime）的引导。

### 方案

在 `cross_patterns.go` 中增加 ImplDetail + skill 节点：

```go
// Entity 属性类型
implDetailNode("attr-string", "string attribute", "string(200), string(500), string not null, string(100) not null unique"),
implDetailNode("attr-boolean", "boolean attribute", "boolean default true, boolean default false"),
implDetailNode("attr-datetime", "datetime attribute", "datetime — date and time value"),
implDetailNode("attr-integer", "integer attribute", "integer default 0 — numeric value"),
implDetailNode("attr-enum", "enumeration attribute", "Status: HD.TicketStatus default Draft"),
implDetailNode("attr-unique", "unique constraint", "Name: string(100) not null unique — database-level uniqueness"),
implDetailNode("attr-system-members", "system members", "system members (owner, createdDate, changedDate, changedBy)"),
```

并连接到 entity 概念：

```go
edge("entity", "detail:attr-string", HasSyntax),
edge("entity", "detail:attr-boolean", HasSyntax),
edge("entity", "detail:attr-datetime", HasSyntax),
edge("entity", "detail:attr-integer", HasSyntax),
edge("entity", "detail:attr-enum", HasSyntax),
edge("entity", "detail:attr-unique", HasSyntax),
edge("entity", "detail:attr-system-members", HasSyntax),
```

### 预期效果

```bash
$ mxcli onto explore entity --depth 1
Concept: Entity [...]
  ImplDetail (7):
    string attribute                    string(200), string(500), string not null
    boolean attribute                   boolean default true/false
    datetime attribute                  datetime — date and time value
    integer attribute                   integer default 0
    enumeration attribute               Status: HD.TicketStatus default Draft
    unique constraint                   Name: string(100) not null unique
    system members                      system members (owner, createdDate, changedDate, changedBy)
```

**改动量：** `cross_patterns.go` +25 行

---

## P4: 知识库课程引导增强

### 问题

Module 06 (知识库) 在 `plan` 中只显示 Entity + Security，缺少独立概念节点。

### 方案

知识库本身不需要独立适配器——它的核心模式（self-ref-association, many-to-many）已存在。
只需在 `curriculum_academy.go` 中补充 TEACHES edges 到现有 skill：

```go
// 在 academy-06-kb 的 Build 中补充
edge("curriculum:academy-06-kb", "skill:manage-security", Teaches),
```

## 改动总览

| 优先级 | 改动 | 文件 | 行数 |
|--------|------|------|------|
| P0 | Orchestrate Inbound 依赖 | `guidance.go` | 1 |
| P1 | Java Action stepNode | `java_action.go` | +25 |
| P1 | JS Action stepNode | `js_action.go` | +25 |
| P1 | CSS Theme stepNode | `cross_patterns.go` | +15 |
| P2 | Nanoflow pattern + steps | `microflow_patterns.go` | +25 |
| P3 | Entity 属性类型 ImplDetail | `cross_patterns.go` | +25 |
| P4 | 知识库 skill 补充 | `curriculum_academy.go` | +1 |
| | **合计** | **7 文件** | **~117 行** |

所有改动都是**纯新增**，符合 OCP（不修改现有逻辑）。

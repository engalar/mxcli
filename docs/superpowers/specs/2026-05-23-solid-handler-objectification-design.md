# SOLID 重构设计：Handler 对象化

**日期:** 2026-05-23  
**范围:** `mdl/executor/` + `mdl/backend/`  
**原则:** SRP, OCP, ISP, DIP  

---

## 背景与动机

当前 `mdl/executor/` 包有 366 个文件、11 万行代码。所有 129 个命令 handler 都是自由函数，通过 `*ExecContext`（20+ 字段的上帝对象）获取全部依赖。`mdl/backend/mpr/MprBackend` 单一结构体实现 30 个接口、217 个方法、1481 行。

主要 SOLID 违反点：

| 原则 | 违反位置 | 表现 |
|------|---------|------|
| SRP | `cmd_pages_builder_v3.go`（2911 行）等大文件 | 单文件承担多个子职责 |
| OCP | `MprBackend`（217 个方法） | 新增功能必须在同一个 struct 上堆方法 |
| ISP | `ExecContext` 传给所有 handler | handler 只需 3-5 个字段，却拿到全部 |
| DIP | handler 通过 duck type 推断 backend 能力 | 依赖具体实现而非抽象 |

---

## 设计

### 1. 核心抽象：`StatementHandler` 接口

```go
// mdl/executor/handler.go（新文件）

// StatementHandler 是每个 MDL 领域命令的可测试执行单元。
// 实现者通过构造函数接收最小依赖，而非整个 ExecContext。
type StatementHandler interface {
    Execute(ctx context.Context, stmt ast.Statement) error
}

// HandlerFactory 从 ExecContext 提取最小依赖并构造 StatementHandler。
// 工厂在每次 Dispatch 时调用，handler 实例无状态，可安全丢弃。
type HandlerFactory func(ctx *ExecContext) StatementHandler

// StatementExecutor 让需要递归执行的 handler（如 EXECUTE SCRIPT）
// 依赖抽象而非 ExecContext.ExecuteFn 闭包。
type StatementExecutor interface {
    Execute(stmt ast.Statement) error
}
```

### 2. Registry 双轨并存

```go
// registry.go 改造：新增 HandlerFactory 注册路径，旧路径保持不变。

type Registry struct {
    handlers  map[reflect.Type]StmtHandler    // 旧路径（过渡期保留）
    factories map[reflect.Type]HandlerFactory  // 新路径（领域命令迁移目标）
}

// RegisterHandler 注册 StatementHandler 工厂（新路径）。
func (r *Registry) RegisterHandler(stmt ast.Statement, f HandlerFactory)

// Dispatch 优先走新路径，回退到旧路径。
func (r *Registry) Dispatch(ctx *ExecContext, stmt ast.Statement) error {
    t := reflect.TypeOf(stmt)
    if f, ok := r.factories[t]; ok {
        return f(ctx).Execute(ctx.Context, stmt)
    }
    if h, ok := r.handlers[t]; ok {
        return h(ctx, stmt)
    }
    return mdlerrors.NewUnsupported(...)
}
```

### 3. 领域 handler struct 模式

```go
// 示例：cmd_entity_handler.go

type EntityHandler struct {
    domain  backend.DomainModelBackend  // 窄接口，非 FullBackend
    modules backend.ModuleBackend
    sec     backend.SecurityBackend
    cache   *executorCache              // session 级共享状态，直接注入
    output  io.Writer
}

func NewEntityHandler(ctx *ExecContext) *EntityHandler {
    return &EntityHandler{
        domain:  ctx.Backend,
        modules: ctx.Backend,
        sec:     ctx.Backend,
        cache:   ctx.Cache,
        output:  ctx.Output,
    }
}

func (h *EntityHandler) Execute(ctx context.Context, stmt ast.Statement) error {
    switch s := stmt.(type) {
    case *ast.CreateEntityStmt:
        return h.create(ctx, s)
    case *ast.AlterEntityStmt:
        return h.alter(ctx, s)
    case *ast.DropEntityStmt:
        return h.drop(ctx, s)
    }
    return fmt.Errorf("EntityHandler: unhandled %T", stmt)
}
```

注册方式：

```go
func registerEntityHandlers(r *Registry) {
    factory := func(ctx *ExecContext) StatementHandler {
        return NewEntityHandler(ctx)
    }
    r.RegisterHandler(&ast.CreateEntityStmt{}, factory)
    r.RegisterHandler(&ast.AlterEntityStmt{}, factory)
    r.RegisterHandler(&ast.DropEntityStmt{}, factory)
}
```

### 4. 基础设施命令豁免

以下命令**不迁移**到 StatementHandler，保留旧 StmtHandler 函数形式：

| 命令 | 原因 |
|------|------|
| `ConnectStmt` / `DisconnectStmt` / `StatusStmt` | Execute 完成后需写回 `ctx.Backend`，接口签名无法表达此副作用 |
| `ExecuteScriptStmt` / `ExecuteSQLStmt` | 需要调用 `StatementExecutor`，过渡期保留以避免循环依赖 |

豁免类命令集中在 `executor_connect.go` 和 `cmd_misc.go`，不污染其他域。

### 5. 递归执行：StatementExecutor 接口提取

```go
// executor.go：Executor 显式实现 StatementExecutor
var _ StatementExecutor = (*Executor)(nil)

func (e *Executor) Execute(stmt ast.Statement) error { ... }
```

ScriptHandler（过渡期后）依赖 StatementExecutor 而非 ExecuteFn 闭包：

```go
type ScriptHandler struct {
    executor StatementExecutor
    output   io.Writer
}
```

### 6. MprBackend 内部拆分（独立阶段）

与 handler 对象化**解耦**，分开进行。目标是将 1481 行的 `backend.go` 按域拆分为嵌入子结构体：

```go
type MprBackend struct {
    *entityImpl      // domainmodel_impl.go
    *microflowImpl   // microflow_impl.go
    *pageImpl        // page_impl.go
    // ...
    reader *modelsdkmpr.Reader  // 共享，通过指针传递给各 impl
}
```

每个 `impl` 在独立文件中实现对应的 backend 子接口。新增域只需新建文件、嵌入到 MprBackend，不修改现有代码（OCP）。

**注意：** 多个 impl 共享同一 reader 指针，顺序执行下无冲突；如未来引入并发写，需在 reader 层加互斥锁。

### 7. 测试模式对比

**旧方式（全量 MockBackend）：**

```go
ctx := &ExecContext{
    Backend: newMockBackend(t),  // 需配置 217 个方法子集
    Cache:   newExecutorCache(),
    Output:  io.Discard,
    // + 15 个无关字段
}
err := execCreateEntity(ctx, &ast.CreateEntityStmt{...})
```

**新方式（窄接口 mock）：**

```go
h := &EntityHandler{
    domain:  &mockDomainModelBackend{CreateEntity: func(...) { ... }},
    modules: &mockModuleBackend{GetModuleByName: func(...) { ... }},
    sec:     &mockSecurityBackend{},
    cache:   &executorCache{},
    output:  io.Discard,
}
err := h.Execute(context.Background(), &ast.CreateEntityStmt{...})
```

测试只需实现 handler 真正调用的方法，**无关方法无需配置**。推荐引入 `mockery` 自动生成窄接口 mock。

---

## 迁移分期

### 阶段 0：基础设施（1-2 天）

- 新增 `handler.go`：定义 `StatementHandler`、`HandlerFactory`、`StatementExecutor`
- 改造 `registry.go`：新增 `factories` map 和 `RegisterHandler` 方法
- 提取 `StatementExecutor` 接口并让 `*Executor` 实现

**验收：** `make test` 全绿，旧注册路径不受影响。

### 阶段 1：试点域（3-5 天）

迁移 3 个低风险域验证模式：

1. **Enumeration**（3 个 handler，逻辑简单）
2. **Constant**（2 个 handler）
3. **Image**（3 个 handler）

每个域：新建 handler struct → 移动实现 → 更新注册 → 补充窄接口单测。

**验收：** 试点域有独立 handler_test.go，覆盖率 ≥ 已有水平，`make test` 全绿。

### 阶段 2：主力域（2-3 周）

按优先级分批：

| 批次 | 域 | handler 数量 |
|------|---|-------------|
| 1 | Entity, Association, Module | ~15 |
| 2 | Microflow, Nanoflow | ~10 |
| 3 | Security, Navigation | ~8 |
| 4 | Page, Snippet, Layout | ~12 |
| 5 | Workflow, BusinessEvent | ~10 |
| 6 | OData, REST, Mapping | ~12 |

每批完成后运行 `make test` + `make lint`，确认无回归。

### 阶段 3：MprBackend 内部拆分（独立 PR）

与阶段 1-2 并行或之后进行，不阻塞 handler 迁移。

### 豁免列表（永久）

- `executor_connect.go` 中的 Connect/Disconnect/Status
- `cmd_misc.go` 中的 ExecuteScript（过渡期）

---

## 成功指标

| 指标 | 目标 |
|------|------|
| handler 单测覆盖率 | 迁移后的域 ≥ 80% |
| 最大文件行数 | 无单文件超过 600 行（现有 2911 行） |
| `make test` | 全程绿色，无回归 |
| Mock 方法数 | 每个 handler 测试 mock ≤ 10 个方法 |
| 豁免命令数 | ≤ 5 个（Connect/Disconnect/Status/ExecuteScript/ExecuteSQL） |

---

## 关键约束

1. **ExecContext 不删除** — 过渡期内保持原样，factory 从中提取依赖
2. **旧 StmtHandler 路径不删除** — Dispatch 保留双轨，旧路径在豁免命令退出前不移除
3. **mockery 为前提** — 阶段 0 前引入，否则窄接口 mock 手写成本过高
4. **每个 PR 只迁移一个域** — 保证 review 粒度可控，失败时易回滚

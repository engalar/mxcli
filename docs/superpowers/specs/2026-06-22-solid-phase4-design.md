# SOLID Phase 4：基于 6 条 Go 原则的最终重构设计方案

> 目标：用 `docs/03-development/GO_SOLID_PRINCIPLES.md` 的 6 条原则，完成 mxcli 核心架构的最后清理。
> 策略：自下而上（先删死代码，后重构架构），逐域增量迁移。

---

## 原则映射

| 原则 | 在本设计中的应用 |
|------|-----------------|
| 1️⃣ 接口在消费侧 | 删除 `backend.FullBackend`，每个 handler 声明自己需要的窄接口 |
| 2️⃣ Option 模式 | `Executor` 用 Option 模式处理可选依赖（cache/graph/logger） |
| 3️⃣ 按域分包 | `handler_deps.go` 的 138 行注册拆回按域的 `handlers_*.go` 文件 |
| 4️⃣ 组合根组装 | 每个域的 `Register*Handlers` 声明精确依赖，`main()` 或 `Builder` 组装 |
| 5️⃣ 窄接口 + 多实现 | `*MprBackend` 实现 ~20 个窄接口，每个带 `var _ Interface = (*Impl)(nil)` |
| 6️⃣ 避免上帝结构体 | `Executor` → 只剩 5 字段；`HandlerDeps` → 按域拆，最终删除 |

---

## 目标架构

### Executor（原则 6）

```go
// 最终目标：只有 5 个字段
type Executor struct {
    registry  *Registry          // 语句分发
    output    io.Writer          // 输出
    guard     *outputGuard       // 行数限制
    logger    *diaglog.Logger    // 日志
    perfStats []perfStmt         // 性能统计
}

// 可选依赖通过 Option 模式（原则 2）
func WithCache(c *executorCache) Option
func WithGraph(g *graphcatalog.ProjectGraph) Option
func WithFragments(f map[string]*ast.DefineFragmentStmt) Option
```

### Handler 注册（原则 1+3+4）

```go
// mdl/executor/domainmodel/handler.go
package domainmodel

import "github.com/mendixlabs/mxcli/mdl/backend"

// 每域一个注册函数，声明精确依赖
func RegisterHandlers(
    r *executor.Registry,
    ml backend.ModuleLister,        // 消费者侧定义窄接口
    dmr backend.DomainModelReader,
    dmw backend.DomainModelWriter,
    output io.Writer,
) {
    r.RegisterFuture("CreateEntity", func(ctx context.Context, stmt ast.Statement) error {
        return execCreateEntityFn(ctx, stmt.(*ast.CreateEntityStmt), ml, dmr, dmw, output)
    })
}
```

### MprBackend（原则 5）

```go
// mdl/backend/mpr/backend.go
var _ backend.ConnectionBackend = (*MprBackend)(nil)
var _ backend.ModuleLister      = (*MprBackend)(nil)
var _ backend.EntityReader      = (*MprBackend)(nil)
var _ backend.EntityWriter      = (*MprBackend)(nil)
// ... ~20 行
```

---

## 执行阶段

### Phase A：删除死代码（原则 6 的"谁用谁拿"）

**目标**：删除所有 `*ExecContext` wrapper 和 bridge，为 ExecContext 结构体删除铺路。

**Step A1：替换 ~80 个旧 wrapper**

```go
// 从（在 cmd_xxx.go 末尾）：
func execCreateModule(ctx *ExecContext, s *ast.CreateModuleStmt) error {
    return execCreateModuleFn(ctx, s, execContextToDeps(ctx))
}

// 改为：删除 wrapper，直接调 Fn
// 测试中：execDropAgent(ctx, stmt) → execDropAgentFn(ctx, stmt, execContextToDeps(ctx))
```

**Step A2：删除 phase3d2bNewExecContext**

所有 bridge 函数直接调 Fn 版本：
```go
// 从：
r.RegisterFuture("CreateEntity", func(ctx context.Context, stmt ast.Statement) error {
    ectx := phase3d2bNewExecContext(ctx, deps)
    return execCreateEntity(ectx, stmt.(*ast.CreateEntityStmt))
})

// 改为：
r.RegisterFuture("CreateEntity", func(ctx context.Context, stmt ast.Statement) error {
    return execCreateEntityFn(ctx, stmt.(*ast.CreateEntityStmt), ml, dmr, dmw, output)
})
```

**Step A3：删除 ExecContext 结构体**

`newExecContext()` 返回 `*HandlerDeps`。所有 `*ExecContext` 类型替换为 `*HandlerDeps`。

**验证**：`go build` + `go test` 零回归，`grep ExecContext` 返回空。

---

### Phase B：注册表按域分包（原则 3+4）

**目标**：消除 `handler_deps.go` 的 138 行 God 注册。

**做法**：

```
mdl/executor/domainmodel/
  handler.go        ← RegisterHandlers(r, ml, dmr, dmw, output)

mdl/executor/microflow/
  handler.go        ← RegisterHandlers(r, mfReader, mfWriter, output)

mdl/executor/security/
  handler.go        ← RegisterHandlers(r, secReader, secWriter, output)
```

每个 `handler.go` 导出 `RegisterHandlers(r *Registry, ...narrow interfaces...)`：

```go
// security/handler.go
package security

func RegisterHandlers(
    r *executor.Registry,
    secReader backend.SecurityReader,
    secWriter backend.SecurityWriter,
    ml backend.ModuleLister,
    output io.Writer,
) {
    r.RegisterFuture("GrantEntityAccess", ...)
    r.RegisterFuture("RevokeEntityAccess", ...)
    r.RegisterFuture("GrantPageAccess", ...)
    r.RegisterFuture("RevokePageAccess", ...)
}
```

**组合根**（原则 4）：

```go
// cmd/mxcli/main.go
registry := executor.NewRegistry()
be := mprbackend.New()

domainmodel.RegisterHandlers(registry, be, be, be, output)
microflow.RegisterHandlers(registry, be, be, output)
security.RegisterHandlers(registry, be, be, be, output)
// 每个域一行，依赖精确
```

---

### Phase C：Executor 瘦身（原则 2+6）

**目标**：Executor 从 14 字段降到 5 字段。

```go
type Executor struct {
    registry   *Registry
    output     io.Writer
    guard      *outputGuard
    logger     *diaglog.Logger
    perfStats  []perfStmt
}
```

**移除的字段去向：**

| 字段 | 去向 | 理由 |
|------|------|------|
| `backend` `backendFactory` | 不需要 | handler 通过闭包捕获自己的窄接口 |
| `cache` | `finalizeProgramExecution()` 参数 | 只有 1 个方法用 |
| `graphCatalog` | `BuildGraph()` 返回值 | 只有 2 个方法用 |
| `fragments` | `ExecuteProgram()` 参数 | 只有 1 个方法用 |
| `mprPath` | `connect()` handler 的 deps | 只有 connect/disconnect 用 |
| `format` `quiet` | `output` 包装器 | 是输出格式，不是执行引擎属性 |

**Executor 工厂**（原则 2）：

```go
func New(registry *Registry, output io.Writer, opts ...Option) *Executor {
    guard := newOutputGuard(output, maxOutputLines)
    e := &Executor{
        registry: registry,
        output:   guard,
        guard:    guard,
    }
    for _, opt := range opts {
        opt(e)
    }
    return e
}
```

---

### Phase D：HandlerDeps 拆分 + 删除（原则 1+6）

**目标**：`HandlerDeps` 的 40+ 字段拆到各域注册函数的参数中，最终删除 `HandlerDeps`。

**中间步骤**：先拆为 3-4 个小组，再逐步消除：

```go
// Step 1: 按功能分组
type IODeps struct { Output, StatusOutput io.Writer; Format OutputFormat; Quiet bool }
type ReaderDeps struct { ModuleLister, DomainModelReader, ... }
type WriterDeps struct { ModuleWriter, DomainModelWriter, ... }

// Step 2: 每个 handler 只捕获自己需要的组
r.RegisterFuture("CreateEntity", func(ctx context.Context, stmt) error {
    return execCreateEntityFn(ctx, stmt, io, reader, writer)
})

// Step 3: 随着按域分包（Phase B）推进，窄接口直接变成参数
// ReaderDeps → ModuleLister, DomainModelReader 作为独立参数
```

**最终状态**：无 `HandlerDeps`，每个 handler 声明精确的窄接口参数。

---

### Phase E：check-solid 门控（回归防护）

**目标**：防止回归到旧模式。

```go
// cmd/check-solid/main.go
func main() {
    // 检查 1：无 ExecContext 导入
    // go list -f '{{.Imports}}' ./... | grep ExecContext → fail
    
    // 检查 2：无 backend.FullBackend 引用（backend.go 和 mock 除外）
    // grep -rn 'FullBackend' *.go | grep -v backend.go | grep -v mock_backend.go → fail
    
    // 检查 3：所有类型断言检查 ok
    // grep -rn '\.([^)]*)$' *.go | grep -v '_test.go' | grep -v '\.(ok)' → fail
}
```

集成到 CI：
```yaml
# .github/workflows/ci.yml
- run: go run ./cmd/check-solid
```

---

## 阶段依赖图

```
Phase A (删死代码)
    │
    ▼
Phase B (按域分包)
    │
    ▼
Phase C (Executor瘦身) ──► Phase D (HandlerDeps拆分)
    │                           │
    └───────────┬───────────────┘
                ▼
         Phase E (check-solid门控)
```

Phase A 是前置条件（删除旧 wrapper 是后续所有步骤的基础）。
Phase B 和 Phase C/D 可以并行（子包和 Executor 是相对独立的）。
Phase E 是最后一道防线。

---

## 成功标准

1. `go build ./...` 通过，零 `FullBackend` 外部引用
2. `go test ./...` 通过，零 `ExecContext` 引用
3. `grep -rn 'FullBackend\b' --include='*.go'` 只出现 `backend.go` 和 `mock_backend.go`
4. `grep -rn 'ExecContext\b' --include='*.go'` 输出为空
5. `handler_deps.go` 不再存在（按域分包后由各域 `handler.go` 替代）
6. `Executor` 字段 ≤ 6（执行引擎核心）

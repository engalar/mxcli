# SOLID Phase 4 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 完成 mxcli 核心架构的最终 SOLID 重构，删除 ExecContext，按域分包，瘦身 Executor。

**Architecture:** 自下而上：先删死代码（wrapper+bridge）→ 按域分包注册 → Executor瘦身 → HandlerDeps拆分 → check-solid门控。

**Tech Stack:** Go 1.24, `mdl/executor` 包

**Spec:** `docs/superpowers/specs/2026-06-22-solid-phase4-design.md`

## Global Constraints

- 每个 commit 必须编译通过：`go build ./mdl/executor/...`
- 每个 commit 必须通过 vet：`go vet ./mdl/executor/...`
- 每个 commit 必须通过测试：`go test ./mdl/executor/... -count=1`
- 优先用 `subagent-driven-development` 并行执行
- 每个域分包必须有独立的 `handler.go` 文件

---

### Task A1: 替换所有测试中对旧 wrapper 的直接调用

**Files:** 所有 `*_test.go` 文件，约 30-40 个文件

**Interfaces:**
- Consumes: 旧 wrapper 签名如 `execDropAgent(ctx *ExecContext, stmt *ast.DropAgentStmt)`
- Produces: 直接调 Fn 版本如 `execDropAgentFn(ctx, stmt, execContextToDeps(ctx))`

**Step 1: 找到所有测试中对旧 wrapper 的调用**

```bash
grep -rn '^func.*ExecContext' --include='*_test.go' mdl/executor/ | head -20
```

旧 wrapper 是 `cmd_*.go` 末尾的 ~80 个单行委托函数（形如 `return execXxxFn(...)`）。测试中直接调用 `execXxx(ctx, ...)`。

**Step 2: 为每个调用生成替换**

替换模式：
```go
// 从：
err := execDropAgent(ctx, &ast.DropAgentStmt{...})

// 改为：
err := execDropAgentFn(ctx, &ast.DropAgentStmt{...}, execContextToDeps(ctx))
```

**Step 3: 构建并测试**

```bash
go build ./mdl/executor/... && go vet ./mdl/executor/... && go test ./mdl/executor/... -count=1 2>&1 | tail -5
```

**Step 4: 提交**

```bash
git add -A && git commit -m "refactor: replace test calls to old ExecContext wrappers with Fn versions (Phase A1)"
```

---

### Task A2: 删除旧 wrapper 函数

**Files:** 所有 `cmd_*.go` 文件末尾的 wrapper 函数

**Step 1: 找到所有旧 wrapper**

```bash
grep -n '^func.*ExecContext' --include='*.go' -r mdl/executor/cmd_*.go | grep -v _test | grep -v '\.pb\.\|flowBuilder\|executorCache'
```

每个格式是：
```go
func execXxx(ctx *ExecContext, args...) error {
    return execXxxFn(ctx, args..., execContextToDeps(ctx))
}
```

**Step 2: 逐个删除**

对每个 wrapper：
1. 确认所有调用者已替换为 Fn 版本（Task A1）
2. 删除函数体 + 注释

**Step 3: 删除 execContextToDeps()**

确认无一调用后，从 `handler_deps.go` 删除 `execContextToDeps` 函数。

**Step 4: 构建并测试**

```bash
go build ./mdl/executor/... && go vet ./mdl/executor/... && go test ./mdl/executor/... -count=1
```

**Step 5: 提交**

```bash
git add -A && git commit -m "refactor: delete all ~80 old ExecContext wrapper functions (Phase A2)"
```

---

### Task A3: 删除 phase3d2bNewExecContext 和所有 bridge

**Files:** `mdl/executor/handlers_future.go`, `mdl/executor/handler_deps.go`

**Step 1: 找到所有 bridge 函数**

```bash
grep -n 'phase3d2bNewExecContext' --include='*.go' -r mdl/executor/
```

每个 bridge 是：
```go
func execXxxFuture(ctx context.Context, stmt, deps) error {
    ectx := phase3d2bNewExecContext(ctx, deps)
    return execXxx(ectx, stmt.(...))
}
```

**Step 2: 逐个替换**

对 bridge 函数中直接调用的旧 `execXxx`，确认 Fn 版本可用。替换为：
```go
// 从 bridge → 直接调 Fn
r.RegisterFuture("TypeName", func(ctx context.Context, stmt ast.Statement) error {
    return execXxxFn(ctx, stmt.(*ast.XxxStmt), ml, dmr, output)
})
```

**Step 3: 删除 phase3d2bNewExecContext**

确认无一调用后删除。

**Step 4: 构建并测试**

```bash
go build ./mdl/executor/... && go vet ./mdl/executor/... && go test ./mdl/executor/... -count=1
```

**Step 5: 提交**

```bash
git add -A && git commit -m "refactor: delete phase3d2bNewExecContext and all bridge functions (Phase A3)"
```

---

### Task A4: 删除 ExecContext 结构体

**Files:** `mdl/executor/executor.go`, `mdl/executor/executor_dispatch.go`

**Step 1: 确认 ExecContext 无引用**

```bash
grep -rn 'ExecContext' --include='*.go' mdl/executor/ | grep -v _test
```

预期输出为空（所有旧 wrapper 和 bridge 已删除）。

**Step 2: 删除 ExecContext 结构体**

从 `executor.go` 删除：
- `type ExecContext struct`
- `func (ctx *ExecContext) initRoles()`
- `func (ctx *ExecContext) Connected()`
- `func (ctx *ExecContext) statusWriter()`
- `execContextToDeps()`

**Step 3: 简化 newExecContext**

```go
// newExecContext creates a HandlerDeps from the current Executor state.
// Deprecated: only used by autocomplete.go. New code should use
// buildHandlerDeps() directly.
func (e *Executor) newExecContext(ctx context.Context) *HandlerDeps {
    return e.buildHandlerDeps()
}
```

**Step 4: 删除 ExecIO、ExecSession、ExecConnection、ExecCallbacks、ExecRepos 等辅助结构体**

这些是 ExecContext 的子结构体，不再需要。

**Step 5: 构建并测试**

```bash
go build ./mdl/executor/... && go vet ./mdl/executor/... && go test ./mdl/executor/... -count=1
```

**Step 6: 提交**

```bash
git add -A && git commit -m "refactor: delete ExecContext struct and all sub-types (Phase A4)"
```

---

### Task B1: 按域创建 handlers 子包

**Files:**
- Create: `mdl/executor/domainmodel/handler.go`
- Create: `mdl/executor/connection/handler.go`
- Create: `mdl/executor/microflow/handler.go`
- Create: `mdl/executor/page/handler.go`
- Create: `mdl/executor/security/handler.go`
- Create: `mdl/executor/workflow/handler.go`
- Create: `mdl/executor/query/handler.go`
- Create: `mdl/executor/misc/handler.go`
- Modify: `mdl/executor/handler_deps.go`（从中删除注册）
- Modify: `cmd/mxcli/main.go`（组合根）

**每个 handler.go 格式：**

```go
package domainmodel

import (
    "context"
    "github.com/mendixlabs/mxcli/mdl/backend"
    "github.com/mendixlabs/mxcli/mdl/executor"
)

func RegisterHandlers(
    r *executor.Registry,
    ml backend.ModuleLister,
    dmr backend.DomainModelReader,
    dmw backend.DomainModelWriter,
    output io.Writer,
) {
    r.RegisterFuture("CreateEntity", func(ctx context.Context, stmt ast.Statement) error {
        return executor.ExecCreateEntityFn(ctx, stmt.(*ast.CreateEntityStmt), ml, dmr, dmw, output)
    })
}
```

**注意：** 需要将 `execXxxFn` 函数导出（首字母大写），或通过 package-private 模式处理。

**验证：**
```bash
go build ./mdl/executor/... && go test ./mdl/executor/... -count=1
```

---

### Task B2: 更新组合根

**Files:** `cmd/mxcli/main.go` 或 `mdl/executor/builder.go`

```go
// 从 handler_deps.go 中移除 138 行注册
// 在 Builder.Create() 中调用各域 RegisterHandlers：
func (b *Builder) Create() *Executor {
    registry := NewRegistry()
    deps := b.buildHandlerDeps()
    
    domainmodel.RegisterHandlers(registry, deps.ModuleLister, deps.DomainModelReader, ...)
    microflow.RegisterHandlers(registry, deps.MicroflowReader, deps.MicroflowWriter, ...)
    // 每域一行
}
```

**验证：** `go build ./cmd/mxcli/...`

---

### Task C1: Executor 瘦身

**Files:** `mdl/executor/executor.go`

```go
// 从 14 字段减到 5 字段
type Executor struct {
    registry  *Registry
    output    io.Writer
    guard     *outputGuard
    logger    *diaglog.Logger
    perfStats []perfStmt
}
```

移除字段处理：
- `cache` → `finalizeProgramExecution` 内临时创建
- `graphCatalog` → `BuildGraph()` 返回本地变量，不存
- `fragments` → `ExecuteProgram()` 加参数 `fragments map[string]*DefineFragmentStmt`
- `mprPath` → `Connect` handler 中管理
- `format` `quiet` → 移到 `output` 包装器
- `backend` `backendFactory` → 不需要，handler 通过闭包捕获窄接口

**Option 模式**（原则 2）：

```go
type Option func(*Executor)

func WithLogger(l *diaglog.Logger) Option {
    return func(e *Executor) { e.logger = l }
}

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

**验证：** `go build ./mdl/executor/... && go test ./mdl/executor/... -count=1`

---

### Task D1: HandlerDeps 拆分 + 删除

**Files:** `mdl/executor/handler_deps.go`

**Step 1: 功能分组**

```go
type IODeps struct {
    Output       io.Writer
    StatusOutput io.Writer
    Format       OutputFormat
    Quiet        bool
}

type ConnDeps struct {
    ConnectionManager backend.ConnectionManager
    MprPath           string
    Cache             *executorCache
    Graph             *graphcatalog.ProjectGraph
}
```

**Step 2: 逐步替换** — 在 Phase B（按域分包）的推进中自然完成。每个域的 handler 不再需要 `*HandlerDeps`，而是声明精确的窄接口参数。

**Step 3: 删除 HandlerDeps**

确认所有注册函数不再引用 `HandlerDeps`。

---

### Task E1: check-solid 门控

**Files:** Create `cmd/check-solid/main.go`

```go
package main

import (
    "fmt"
    "os/exec"
    "strings"
)

func main() {
    // Check 1: No ExecContext imports
    out, _ := exec.Command("grep", "-rn", "ExecContext", "--include=*.go").Output()
    if len(out) > 0 {
        fmt.Fprintf(os.Stderr, "FAIL: ExecContext still referenced\n%s\n", out)
        os.Exit(1)
    }
    
    // Check 2: No FullBackend refs (except definitions)
    // ...
    
    fmt.Println("PASS: All SOLID checks passed")
}
```

集成到 CI：
```yml
- name: SOLID compliance
  run: go run ./cmd/check-solid
```

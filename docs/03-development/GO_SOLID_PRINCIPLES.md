# Go 语言 SOLID 实践指南

本文档定义了本项目在 Go 中应用 SOLID 原则的 6 条具体实践。每条原则都用 Go 代码示例说明，并在 mxcli 代码库中有对应的实际案例。

---

## 原则 1：接口定义在消费侧（ISP）

**核心思想**：接口属于调用者，不属于实现者。每个包定义自己需要的窄接口，而不是依赖一个全局宽接口。

**❌ 坏实践（mxcli 原始代码）：**

```go
// backend/backend.go — 定义 50+ 方法的巨接口
type FullBackend interface {
    Connect(), Disconnect(), ListModules(), ListDomainModels()
    CreateMicroflow(), DeleteEntity(), GetPageGen() // ... 50+
}
// executor 所有 handler 都依赖这个接口：
func listModules(ctx *ExecContext) error {
    modules, _ := ctx.Backend.ListModules() // 依赖整个 FullBackend
}
```

**✅ 好实践（重构后）：**

```go
// executor/xxx_handler.go — 消费侧定义自己需要的窄接口
type ModuleLister interface { ListModules() ([]Module, error) }

func listModulesFn(ctx context.Context, ml ModuleLister, output io.Writer) error {
    modules, _ := ml.ListModules() // 只依赖 1 个方法
}
```

**Go 谚语**："Accept interfaces, return structs." 接口定义在 `import` 的那一侧，不在 `package` 的那一侧。

**检验标准**：修改 `*MprBackend` 的 `ListMicroflows()` 实现时，不会影响只依赖 `ModuleLister` 的消费者。

---

## 原则 2：Option 模式处理可选依赖（SRP/ISP）

**核心思想**：struct 的核心字段在构造函数中声明，可选依赖通过 `Option` 模式注入。避免一个大 struct 承载所有可能需要的依赖。

**✅ 好实践：**

```go
type Executor struct {
    registry *Registry
    output   io.Writer

    // 可选依赖，注入后非 nil
    cache  *Cache
    tracer *Tracer
}

type Option func(*Executor)

func WithCache(c *Cache) Option {
    return func(e *Executor) { e.cache = c }
}

func New(registry *Registry, output io.Writer, opts ...Option) *Executor {
    e := &Executor{registry: registry, output: output}
    for _, opt := range opts {
        opt(e)
    }
    return e
}

// 使用者只注入需要的：
exec := New(reg, os.Stdout, WithCache(cache))
```

**适用场景**：Cache、Tracer、Logger 等横切关注点。核心依赖（Registry、Output）放在显式参数中。

---

## 原则 3：按域分包，不按层分包（SRP）

**核心思想**：相关功能放一个包。一个包负责一个完整域（读+写+注册）。不按 technical layer（handlers/readers/writers/models）分包。

**mxcli 实际案例：**

```
❌ 坏（当前结构，去掉了 handlers/ 目录）：
mdl/executor/
  handler_deps.go        // 138 行注册在一个文件
  cmd_entity.go          // 实体 handler
  helpers_entity.go      // 实体辅助

✅ 好（目标结构）：
mdl/executor/entity/
  handler.go             // RegisterEntityHandlers + 实现
  reader.go              // 实体读操作
  writer.go              // 实体写操作

mdl/executor/microflow/
  handler.go
  builder.go

mdl/executor/security/
  handler.go
```

**Go 约定**：`internal/` 目录下的包不允许外部导入，适合放内部实现细节。每个域的包可以有自己的 `handler_test.go`。

**检验标准**：修改域 A 的逻辑时，不需要打开域 B 的任何文件。

---

## 原则 4：组合根组装依赖（DIP）

**核心思想**："依赖注入在 main 里面，不在包里。" 每个包只声明自己的需求，组合根（`main()` 或 `Builder`）负责将所有依赖组装在一起。

**✅ 好实践：**

```go
// cmd/mxcli/main.go — 唯一知道所有依赖的地方
func main() {
    registry := executor.NewRegistry()
    backend := mprbackend.New()

    // 每个域自己注册，声明精确依赖
    executor.RegisterEntityHandlers(registry, backend, backend, os.Stdout)
    executor.RegisterMicroflowHandlers(registry, backend, backend, os.Stdout)
    executor.RegisterSecurityHandlers(registry, backend, backend, os.Stdout)

    exec := executor.New(registry, os.Stdout)
    exec.SetBackend(backend)
}
```

**注意**：Go 没有依赖注入框架 —— 手动注入是 Go 的方式。组合根只有一个文件，且不包含业务逻辑。

**检验标准**：如果你需要理解 `MprBackend` 的内部实现才能注册 handler，就是坏的。Handler 应该只声明"我需要 `ModuleLister`"。

---

## 原则 5：窄接口 + 多实现（ISP/DIP）

**核心思想**：一个 struct 可以实现任意多个窄接口。每个 handler 只看自己需要的接口，MprBackend 满足全部。

**✅ 好实践：**

```go
// 一个 struct 同时满足多个小接口
type MprBackend struct{}

// 编译时检查 —— 每个接口一行
var _ backend.ConnectionBackend = (*MprBackend)(nil)  // 3 methods
var _ backend.ModuleLister     = (*MprBackend)(nil)    // 1 method
var _ backend.EntityWriter     = (*MprBackend)(nil)    // 5 methods
var _ backend.MicroflowReader  = (*MprBackend)(nil)    // 4 methods

// 消费者只看到自己需要的：
func listModulesFn(ctx context.Context, ml backend.ModuleLister) error {
    modules, err := ml.ListModules()  // ✅ 只有 1 个方法可见
}
```

**注意**：`var _ Interface = (*Type)(nil)` 是 Go 的编译时类型检查惯用模式。忘记添加这一行时，如果 `*MprBackend` 没有实现某个接口，编译会报错。

**与原则 1 的关系**：原则 1 说接口定义在消费侧，原则 5 说实现侧提供所有接口。两者不矛盾 —— 接口在消费侧定义，实现侧提供编译时检查确保满足。

**检验标准**：每新增一个窄接口，只需在实现 struct 上加一行 `var _ MyInterface = (*Impl)(nil)` 和实现方法。不需要修改任何消费者。

---

## 原则 6：避免上帝结构体（SRP）

**核心思想**：不是把"这个 struct 太大了"拆开，而是让每个函数只持有它运行时真正需要的东西。如果某个字段只在 2-3 个方法里用到，它就不该是 struct 字段，而是方法参数。

**❌ 坏实践（当前 mxcli）：**

```go
type Executor struct {
    backend        backend.FullBackend   // Close() 只用 Disconnect()
    cache          *executorCache        // finalizeProgramExecution() 只用
    graphCatalog   *graphcatalog.ProjectGraph // BuildGraph() 只用
    fragments      map[string]*ast.DefineFragmentStmt // ExecuteProgram() 只用
    mprPath        string                // Close() 和 connect 用
    // 还有 9 个其他字段
}
```

**✅ 好实践：**

```go
type Executor struct {
    registry *Registry
    output   io.Writer
    guard    *outputGuard
    logger   *diaglog.Logger
    perfStats []perfStmt
}

// 谁用谁拿，用完即弃：
func finalizeProgramExecution(ctx context.Context, dmr DomainModelReader) error {
    cache := &executorCache{}  // 临时创建，不挂在 Executor 上
    // ...
}

// fragments 是 ExecuteProgram() 的参数，不是 Executor 的字段：
func (e *Executor) ExecuteProgram(prog *ast.Program, fragments map[string]*ast.DefineFragmentStmt) error {
    // ...
}
```

**Go 值传递特性**：Go 是值传递语言。struct 字段越多，每次复制开销越大，测试 setup 越复杂。字段越少，代码越容易推理。

**检验标准**：如果你需要在测试中 setup 5 个以上字段才能测一个方法，这个 struct 违反了 SRP。

---

## 检查清单

使用本指南时，逐项检查：

| # | 原则 | 检查项 |
|---|------|--------|
| 1 | 接口在消费侧 | 每个包定义自己的接口？没有依赖巨接口？ |
| 2 | Option 模式 | 可选依赖通过 Option 注入？核心依赖在构造函数参数中？ |
| 3 | 按域分包 | 一个包负责一个完整域？不按 layer 分包？ |
| 4 | 组合根组装 | 只有一个文件组装所有依赖？无包内依赖注入？ |
| 5 | 窄接口 + 多实现 | 每个 struct 有 `var _ Interface = (*Impl)(nil)`？接口 < 5 方法？ |
| 6 | 避免上帝结构体 | 没有 > 5 字段的 struct？（组合根除外） |

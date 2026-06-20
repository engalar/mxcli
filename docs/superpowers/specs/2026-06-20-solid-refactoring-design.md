# SOLID 重构设计方案

> 目标：将 `mxcli` 核心架构重构为严格遵循 SOLID 原则的形态。
> 策略：Big Bang 重写——冻结特性开发，一次性替换所有违规点，通过完整测试套件验证。

---

## 1. 当前违规全景

| # | 原则 | 违规点 | 位置 | 严重度 |
|---|------|--------|------|--------|
| 1 | ISP | `FullBackend` 50+ 方法单体接口 | `mdl/backend/backend.go:14-91` | 🔴 |
| 2 | ISP+SRP | `ExecContext` God 结构体（30+ 字段） | `mdl/executor/exec_context.go:81-147` | 🔴 |
| 3 | SRP | `MprBackend` 1659 行单文件实现所有后端 | `mdl/backend/mpr/backend.go` | 🔴 |
| 4 | SRP | `executorCache` 单体缓存 25+ 类型 | `mdl/executor/executor.go:33-112` | 🟡 |
| 5 | SRP | `Executor` 持有 10+ 不相关依赖 | `mdl/executor/executor.go:348-372` | 🟡 |
| 6 | SRP | `model/types.go` 981 行所有类型 | `model/types.go` | 🟡 |
| 7 | OCP | `NewRegistry()` 显式列出 30 个注册函数 | `mdl/executor/register_stubs.go:29-59` | 🟡 |
| 8 | LSP | 未检查的类型断言 | 跨 5 个文件（见 REFACTOR_PLAN.md） | 🔴 |
| 9 | DIP | 处理函数依赖 `*ExecContext` 具体类型 | `mdl/executor/registry.go:15` | 🔴 |
| 10 | DRY | `initSubBackends()` 70+ 次冗余调用 | `mdl/backend/mpr/backend.go:181-237` | 🟡 |
| 11 | DRY | `concreteWriter()` 11+ 次相同模式 | `mdl/backend/mpr/repos_provider.go:21-31` | 🟡 |
| 12 | DRY | `Create*/Update*` nil 检查模式重复 20+ 次 | `mdl/backend/mpr/backend.go` 各处 | 🟡 |
| 13 | 代码异味 | `reflect.TypeOf` 运行时分发 | `mdl/executor/registry.go:66,76` | 🟢 |
| 14 | 代码异味 | 每条语句 goroutine | `mdl/executor/executor.go:508` | 🟢 |
| 15 | 代码异味 | `os.Getenv()` 每次 Execute 调用 | `mdl/executor/executor.go:331` | 🟢 |
| 16 | 代码异味 | `panic()` 重复注册 | `mdl/executor/registry.go:68` | 🟢 |

---

## 2. 目标架构

### 2.1 消除 God Interface：`FullBackend` 拆解

**Before（ISP 违规）：**

```go
// FullBackend 包含 50+ 方法。任何消费者都必须依赖整个接口。
type FullBackend interface {
    ConnectionBackend  // 3 methods
    ModuleBackend      // 8 methods
    DomainModelBackend // 15 methods
    MicroflowBackend   // 10 methods
    // ... 还有 25+ 个子接口
}
```

**After（ISP 合规）：**

```go
// ── 构造时工厂：唯一的地方可以获取某个角色的实现 ──
type BackendFactory interface {
    Connect(path string) error
    Disconnect() error
    IsConnected() bool
    
    ModuleLister()    ModuleLister
    ModuleWriter()    ModuleWriter
    DomainModelReader() DomainModelReader
    DomainModelWriter() DomainModelWriter
    MicroflowReader()   MicroflowReader
    MicroflowWriter()   MicroflowWriter
    // ... 每个角色一个访问器
}

// ── 角色接口定义在 backend/role.go —— 它们已经存在 ──
// 注意：角色接口不再嵌入到任何父接口中。
// 每个消费者声明它需要的角色。
```

**原理：** ISP 要求"不应强迫客户端依赖它们不使用的方法"。`BackendFactory` 是唯一知道所有角色的类型，它只用于构造。业务逻辑处理函数只声明它们需要的角色：

```go
// execCreateModule 只需要 ModuleLister + ModuleWriter
func execCreateModule(
    ctx context.Context,
    stmt *ast.CreateModuleStmt,
    lister ModuleLister,
    writer ModuleWriter,
    output io.Writer,
) error { ... }
```

### 2.2 消除 God Struct：`ExecContext` 消除

**Before（SRP+ISP+DIP 三重违规）：**

```go
type ExecContext struct {
    context.Context
    Backend backend.FullBackend  // 已废弃但仍存在
    DomainModelReader backend.DomainModelReader  // 30+ 角色字段
    // ...
    ExecRepos       // 9 个仓库
    ExecIO          // Output, StatusOutput
    ExecSession     // Cache, Fragments, Settings
    ExecConnection  // MprPath, SqlMgr, Catalog, Graph
    ExecCallbacks   // 4 个回调函数
}
```

**After（SRP+ISP+DIP 合规）：**

`ExecContext` 被完全消除。处理函数通过闭包捕获它们需要的依赖：

```go
// ── 处理函数签名（DIP 合规：依赖抽象）──
type StmtHandler func(ctx context.Context, stmt ast.Statement) error

// ── 注册时通过闭包注入精确的依赖（ISP 合规）──
func RegisterCreateModule(
    reg *Registry,
    lister backend.ModuleLister,
    writer backend.ModuleWriter,
    output io.Writer,
) {
    reg.Register(&ast.CreateModuleStmt{}, func(ctx context.Context, stmt ast.Statement) error {
        s := stmt.(*ast.CreateModuleStmt)
        return execCreateModule(ctx, s, lister, writer, output)
    })
}

// ── 处理函数只接收它需要的参数（DIP 合规）──
func execCreateModule(
    ctx context.Context,
    s *ast.CreateModuleStmt,
    lister backend.ModuleLister,
    writer backend.ModuleWriter,
    output io.Writer,
) error {
    modules, err := lister.ListModules()
    // ...
    return writer.CreateModule(module)
}
```

**原理：**
- **DIP：** 处理函数依赖接口（`ModuleLister`、`ModuleWriter`），而非具体 `*ExecContext`
- **ISP：** 处理函数只接收它需要的参数，不多不少
- **SRP：** 没有 "上下文" 承担多个职责；每个处理函数声明自己的依赖
- 闭包捕获在注册时完成，运行时零开销

### 2.3 跨切关注点：共享服务的注入

某些关注点（缓存、日志、输出）需要被多个处理函数共享。我们用显式的服务接口处理它们：

```go
// ── 缓存 —— 按领域拆分（SRP 合规）──
type MicroflowCache interface {
    GetMicroflows(ctx context.Context) ([]*genMf.Microflow, error)
    Invalidate()
}

type PageCache interface {
    GetPages(ctx context.Context) ([]*genPg.Page, error)
    Invalidate()
}

// ── 日志 —— 窄接口（ISP 合规）──
type Logger interface {
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Debug(msg string, args ...any)
}

// ── 输出 —— 窄接口（ISP 合规）──
type OutputWriter interface {
    Write(p []byte) (n int, err error)
    WriteLine(format string, args ...any)
}
```

### 2.4 `MprBackend` 拆解为聚焦的子后端

**Before（SRP 违规）：**

`MprBackend` 是单文件 1659 行，实现所有 50+ 方法，带有 `initSubBackends()` 和 `concreteWriter()`。

**After（SRP 合规）：**

```go
// ── MprBackend 只是工厂 + 枚举 —— 没有业务方法 ──
type MprBackend struct {
    reader    *modelsdkmpr.Reader
    writer    *mmpr.Writer       // 具体类型，不再需要 concreteWriter()
    path      string
    scriptBuf *ScriptBuffer
    unitBuf   *unitstore.BufferedUnitStore
    
    moduleBkd    *moduleBackend
    microflowBkd *microflowBackend
    pageBkd      *pageBackend
    workflowBkd  *workflowBackend
    // ... 每个领域一个
}

func (b *MprBackend) Connect(path string) error {
    // 所有子后端在 Connect 时创建一次 —— 没有 initSubBackends()
    b.moduleBkd    = newModuleBackend(b.reader)
    b.microflowBkd = newMicroflowBackend(b.writer)
    b.pageBkd      = newPageBackend(b.writer)
    // ...
}

// ── BackendFactory 实现 ──
func (b *MprBackend) ModuleLister()    backend.ModuleLister    { return b.moduleBkd }
func (b *MprBackend) ModuleWriter()    backend.ModuleWriter    { return b.moduleBkd }
func (b *MprBackend) MicroflowReader() backend.MicroflowReader { return b.microflowBkd }
func (b *MprBackend) MicroflowWriter() backend.MicroflowWriter { return b.microflowBkd }
// ...
```

每个子后端只实现它被分配的角色，在其自己专用的文件中：

```
mdl/backend/mpr/
  backend.go         # MprBackend（仅工厂 + 连接管理，~150 行）
  mpr_module.go      # moduleBackend（实现 ModuleLister + ModuleWriter）
  mpr_microflow.go   # microflowBackend（实现 MicroflowReader + MicroflowWriter）
  mpr_page.go        # pageBackend（实现 PageReader + PageWriter）
  mpr_entity.go      # entityBackend（实现 DomainModelReader + DomainModelWriter）
  mpr_workflow.go    # workflowBackend
  mpr_security.go    # securityBackend
  repos_provider.go  # 已移除（concreteWriter() 消失）
```

**原理：** 每个子后端只有一个职责。构造函数直接接收具体 `*mmpr.Writer`，所以不需要 `concreteWriter()`。不需要 `initSubBackends()`，因为每个子后端都是在 `Connect()` 中创建一次的。

### 2.5 `executorCache` 消除

**Before（SRP 违规）：** 单个结构体缓存所有文档类型。

**After（SRP 合规）：**

缓存被移入其各自的领域包中。每个缓存独立管理其生命周期：

```go
// 在 mdl/backend/mpr/ 内 —— 靠近消费者
type microflowCache struct {
    mu       sync.RWMutex
    items    []*genMf.Microflow
    valid    bool
}

func (c *microflowCache) Get(ctx context.Context, reader backend.MicroflowReader) ([]*genMf.Microflow, error) {
    c.mu.RLock()
    if c.valid && c.items != nil {
        defer c.mu.RUnlock()
        return c.items, nil
    }
    c.mu.RUnlock()
    
    c.mu.Lock()
    defer c.mu.Unlock()
    if c.valid && c.items != nil {  // double-check
        return c.items, nil
    }
    items, err := reader.ListMicroflowsGen(ctx)
    if err != nil {
        return nil, err
    }
    c.items = items
    c.valid = true
    return c.items, nil
}

func (c *microflowCache) Invalidate() {
    c.mu.Lock()
    c.valid = false
    c.mu.Unlock()
}
```

缓存由子后端拥有并通过角色接口暴露：

```go
type MicroflowReader interface {
    ListMicroflowsGen(ctx context.Context) ([]*genMf.Microflow, error)
    GetMicroflowGen(ctx context.Context, id model.ID) (*genMf.Microflow, error)
    ListNanoflowsGen(ctx context.Context) ([]*genMf.Nanoflow, error)
    IsRule(ctx context.Context, qualifiedName string) (bool, error)
    InvalidateCache()
}
```

### 2.6 `Executor` 瘦身

**Before（SRP 违规）：** 持有 backend、factory、output、guard、mprPath、settings、cache、catalog、graphCatalog、quiet、format、logger、fragments、sqlMgr、themeRegistry、registry、catalogMu、catalogGen、perfStats。

**After（SRP 合规）：**

```go
type Executor struct {
    registry  *Registry
    guard     *outputGuard
    perfStats []perfStmt
    logger    Logger
}

func (e *Executor) Execute(ctx context.Context, stmt ast.Statement) error {
    e.guard.reset()
    h := e.registry.Lookup(stmt)
    if h == nil {
        return mdlerrors.NewUnsupported(...)
    }
    return h(ctx, stmt)
}
```

不再直接连接后端。`Executor` 只是 AST 的分发引擎。连接、缓存、目录——这些都注入到闭包中。

### 2.7 `model/types.go` 按领域拆分

**Before（SRP 违规）：** 981 行，包含 Module、Enumeration、Constant、ScheduledEvent、OData 服务、REST 服务、业务事件、数据库连接、数据转换器、ProjectSettings（10 个子类型）、ImportMapping、ExportMapping、UnknownElement。

**After（SRP 合规）：**

```
model/
  types.go        # 仅核心共享类型（ID, QualifiedName, Point, Size, Element, BaseElement, Unit, UnknownElement）
  module.go       # Module + Folder
  enumeration.go  # Enumeration + EnumerationValue
  constant.go     # Constant + ConstantDataType
  schedule.go     # ScheduledEvent
  odata.go        # ODataService 类型
  rest.go         # RestService 类型
  bizevent.go     # BusinessEventService
  dbconn.go       # DatabaseConnection
  transformer.go  # DataTransformer
  settings.go     # ProjectSettings + 所有 Part 类型
  mappings.go     # ImportMapping, ExportMapping
```

**原理：** 每个文件只有一个更改的原因。修改 ProjectSettings 不会影响 Module。

### 2.8 `Registry`：OCP 合规

**Before（OCP 违规）：** `NewRegistry()` 显式调用 30 个 `register*Handlers`。添加新语句类型必须修改此函数。

**After（OCP 合规）：** 每个领域包自行注册。`Registry` 只是映射。`NewRegistry()` 不再存在——构建时从声明者组合注册：

```go
// ── Registry：纯映射，无反射要求 ──
type Registry struct {
    handlers map[string]func(context.Context, ast.Statement) error
}

func (r *Registry) Register(stmtType string, fn func(context.Context, ast.Statement) error) {
    r.handlers[stmtType] = fn
}

func (r *Registry) Lookup(stmt ast.Statement) (func(context.Context, ast.Statement) error, bool) {
    fn, ok := r.handlers[stmt.TypeName()]
    return fn, ok
}
```

使用 `ast.Statement` 接口上的 `TypeName() string` 方法替代 `reflect.TypeOf`：

```go
// 在 ast/ast.go 中
type Statement interface {
    isStatement()
    TypeName() string  // 新增，返回 "CreateModule" / "DropMicroflow" 等
}
```

每个 AST 节点返回其规范名称：

```go
func (s *CreateModuleStmt) TypeName() string { return "CreateModule" }
```

组合性通过公开的 `Register` 函数强制执行——添加新语句类型不需要修改现有代码：

```go
// 在 executor/handlers/handler_microflow.go 中 ——
// 这是独立的，不触及 register_stubs.go
func init() {
    RegisterStmtType("CreateMicroflow", func(ctx context.Context, stmt ast.Statement) error {
        return execCreateMicroflow(ctx, stmt.(*ast.CreateMicroflowStmt))
    })
}
```

这解决了 OCP 违规：`Registry` 对扩展开放（任何包都可以 `init()` 注册），对修改关闭（不需要编辑 `Registry` 或中心注册文件）。

### 2.9 LSP：消除未检查的类型断言

**Before：** 5 个位置使用 `val, _ := expr.(*Type)` 忽略 `ok`。

**After：** 每个类型断言检查并优雅处理：

```go
// ── Before ──
db, _ := elem.(*genDm.Entity)

// ── After ──
db, ok := elem.(*genDm.Entity)
if !ok {
    return mdlerrors.NewTypeMismatch("Entity", fmt.Sprintf("%T", elem))
}
// 或者跳过：
oc, ok := mf.ObjectCollection().(*genMf.ObjectCollection)
if !ok {
    continue  // 静默跳过意外的类型
}
```

### 2.10 `concreteWriter()` 和 `initSubBackends()` 的消除

**`concreteWriter()` 修复：** 子后端直接在 `Connect()` 中接收 `*mmpr.Writer` 而不是从 `MprBackend` 请求它。转换到 `*mmpr.Writer` 只发生一次。

**`initSubBackends()` 修复：** 所有子后端在 `Connect()` 中急切地创建。没有延迟初始化。没有 nil 检查。`Disconnect()` 将它们全部设置为 nil。

### 2.11 语句注册的重组

**Before：** 30 个 `register*Handlers(r *Registry)` 函数都接受 `*ExecContext`。

**After：** 处理函数注册移动到其各自领域包附近的专用文件中。每个注册函数在闭包中捕获其精确的依赖关系：

```go
// mdl/executor/handlers/handler_microflow.go

func RegisterMicroflowHandlers(
    reg *Registry,
    mfReader repos.MicroflowReader,
    mfWriter repos.MicroflowWriter,
    output io.Writer,
    logger Logger,
) {
    reg.Register("CreateMicroflow", func(ctx context.Context, stmt ast.Statement) error {
        s := stmt.(*ast.CreateMicroflowStmt)
        return execCreateMicroflow(ctx, s, mfReader, mfWriter, output, logger)
    })
    reg.Register("DropMicroflow", func(ctx context.Context, stmt ast.Statement) error {
        s := stmt.(*ast.DropMicroflowStmt)
        return execDropMicroflow(ctx, s, mfReader, mfWriter, logger)
    })
}
```

---

## 3. 包结构（目标）

```
mdl/
  executor/
    executor.go           # 瘦 Executor：Execute + perf 报告
    registry.go           # Registry：映射 + 查找（可选泛型 Register[T]）
    builder.go            # Builder：fluent 构造
    handlers/             # 处理函数注册 + 实现
      handler_microflow.go
      handler_page.go
      handler_entity.go
      handler_security.go
      handler_module.go
      handler_workflow.go
      ...
  
  backend/
    role.go               # 窄角色接口（已存在，无变化）
    connection.go          # ConnectionBackend 接口
    factory.go            # BackendFactory 接口
    
    mock/                 # 每个角色的细分 mock（非单体 mock）
      mock_module.go
      mock_microflow.go
      ...
    
    mpr/
      backend.go          # MprBackend：仅 BackendFactory，~150 行
      mpr_module.go       # moduleBackend
      mpr_microflow.go    # microflowBackend（含 microflowCache）
      mpr_page.go         # pageBackend（含 pageCache）
      mpr_entity.go       # entityBackend
      mpr_workflow.go     # workflowBackend
      mpr_security.go     # securityBackend
      mpr_enum.go         # enumerationBackend
      mpr_constant.go     # constantBackend
      mpr_mapping.go      # mappingBackend
      mpr_settings.go     # settingsBackend
      mpr_navigation.go   # navigationBackend
      mpr_image.go        # imageBackend
      mpr_java.go         # javaBackend
      mpr_schedule.go     # scheduledEventBackend
      mpr_services.go     # serviceBackend
      repos/              # 不变的 gen-native 仓库
    
    unitstore/            # 不变的
  
  model/
    types.go              # 仅核心共享类型
    module.go
    enumeration.go
    constant.go
    schedule.go
    odata.go
    rest.go
    bizevent.go
    dbconn.go
    transformer.go
    settings.go
    mappings.go
```

---

## 4. 迁移计划（Big Bang）

### 阶段 0：基础设施

1. 编写 `cmd/check-solid` 工具，强制执行：
   - 无 `*ExecContext` 类型的导入
   - 无 `backend.FullBackend` 类型的导入（`mock.Backend` 除外）
   - 无类型断言忽略 `ok`
2. 设置 CI 门控，通过 `cmd/check-solid` 检查，并编译验证

### 阶段 1：重构模型包（无风险，独立）

- 将 `model/types.go` 拆分为每个领域一个文件
- 机械操作，零逻辑变化
- 验证：`go build ./model/...` 成功

### 阶段 2：重构 MprBackend

- 用 `BackendFactory` 方法替换 `FullBackend`：
  - `MprBackend` 移除所有领域特定方法，变为仅工厂
  - 每个子后端获得自己的文件并实现角色接口
  - `Connect()` 急切地创建子后端
  - 移除 `initSubBackends()` 和 `concreteWriter()`
- 每个子后端获得自己的缓存类型
- 移除 `repos_provider.go`（`concreteWriter()` 模式）
- 验证：`go build ./mdl/backend/...` 成功；所有测试通过

### 阶段 3：重构 Executor + Registry

- `Executor` 瘦身：移除 backend、cache、catalog、sqlMgr、themeRegistry 字段
- `executorCache` 分解
- `ExecContext` 消除：处理函数变为纯闭包
- 注册迁移到处理函数特定的文件，每个文件捕获其依赖关系
- `Registry` 获取泛型 `Register[T]` 方法
- `Builder` 更新以反映新架构
- 验证：`go build ./mdl/executor/...` 成功；所有测试通过

### 阶段 4：更新 CLI 层

- 更新 `cmd/mxcli/main.go` 以使用新的 `Builder` + 处理函数注册
- 移除所有 `cmd/mxcli/` 中对 `ExecContext.Backend` 的引用
- 验证：`go build ./cmd/...` 成功

### 阶段 5：修复 LSP 违规

- 在项目范围内的所有未检查断言中添加 `ok` 检查
- 验证：`cmd/check-solid` 通过，零未检查类型断言

### 阶段 6：最终验证

- 完整测试套件运行：`make test`
- 集成测试运行
- 性能基准测试：确保重构没有显著降级
- 黄金文件更新

---

## 5. 风险缓解

| 风险 | 可能性 | 影响 | 缓解措施 |
|------|--------|------|----------|
| `ExecContext` 消除暴露隐藏的耦合 | 中 | 高 | 每步都编译测试；如果耦合证明太复杂，改用参数对象模式——一个命名的 `HandlerDeps` 结构体，每个处理函数显式声明为最后一个参数，不具备"上下文"的语义重量 |
| 缓存失效行为改变 | 中 | 中 | 迁移后运行黄金文件/回归测试；编写显式的缓存竞争测试 |
| 注册语法错误 | 低 | 中 | 注册表具有编译时验证测试（`TestNewRegistry_Completeness` 和 `TestNewRegistry_HandlerCountSnapshot` 更新以匹配新模式） |
| 性能回归 | 低 | 中 | 基准测试 `make bench` 比较 before/after；缓存/密集代码路径的微基准测试 |
| 回滚计划 | — | — | Big Bang 提交被标记；如果失败，`git revert <commit>` 并恢复原状 |

---

## 6. 成功标准

1. 所有现有测试通过（零回归）
2. `go vet ./...` 清理
3. 无新导入 `backend.FullBackend` 或 `executor.ExecContext`
4. 所有类型断言检查 `ok`
5. 注册表注册是组合式的：新语句类型在不需要修改 `NewRegistry()` 的情况下添加它自己的 `Register` 调用
6. 每个处理函数声明它的依赖关系，作为函数参数（通过闭包捕获）
7. 包依赖图是无环的（已验证通过 `go mod graph`）

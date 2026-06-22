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

## 3. 当前完成状态（截至 2026-06-22）

| 阶段 | 状态 | 证据 |
|------|------|------|
| 1 (model/types 拆分) | ✅ 完成 | `model/types.go` 从 981→273 行 |
| 2 (MprBackend 重构) | ✅ 完成 | `mdl/backend/mpr/backend.go` 从 1659→67 行；`FullBackend` 已有 Deprecated 标注 |
| OCP pages 注册表 | ✅ 完成 | 3 个注册文件 (`page_widget_registry`, `page_datasource_registry`, `page_action_registry`); `pages_builder_v3.go` 从 3088→452 行 |
| Registry TypeName | ✅ 完成 | 无 `reflect.TypeOf` |
| goroutine-per-stmt | ✅ 完成 | 已移除 |
| **3a** Executor 瘦身 | ⬜ 未开始 | Executor 仍有 15 个字段 |
| **3b** executorCache 分解 | ⬜ 未开始 | 仍为单体 25+ 文档类型 |
| **3c** FullBackend→角色最后 55 处 | ⬜ 未开始 | 23 文件，55 处引用 |
| **3d** ExecContext 消除 | ⬜ 未开始 | 1229 处引用，200 文件 |
| **3e** Registry OCP | ⬜ 未开始 | `NewRegistry()` 仍显式列出 30+ 注册 |
| **4** CLI 层 | ⬜ 未开始 | 依赖阶段 3 |
| **5** LSP 断言 | ⬜ 部分 | 需全量审计 |
| **0** 门控工具 | ⬜ 未开始 | 最后写（避免中途重写） |

## 4. 迁移计划（增量可提交顺序）

### Phase 3a — Executor 瘦身 (SRP)

从 `Executor` 摘除非核心字段，降低耦合：

| 字段 | 移入目标 | 方案 |
|------|---------|------|
| `cache *executorCache` | 独立 `ExecutorCache` 类型，Executor 通过接口引用 | 提取 accessor 方法 |
| `graphCatalog *graphcatalog.ProjectGraph` | 懒加载，从 factory 获取 | `executor_connect.go` 中改为 `BackendFactory.GraphCatalog()` |
| `sqlMgr *sqllib.Manager` | `executor_connect.go` 中独立初始化 | Executor 不再直接持有 |
| `themeRegistry *ThemeRegistry` | `executor_connect.go` 中独立初始化 | 同 sqlMgr |
| `fragments map[string]*ast.DefineFragmentStmt` | 移到 session 局部作用域 | Execute() 参数或闭包 |
| `settings map[string]any` | 移到 Builder 或外部配置 | 不通过 Executor 传递 |

**目标：** `Executor` 只保留 `registry`, `guard`, `output`, `statusOutput`, `backendFactory`, `perfStats`, `quiet`, `format`, `logger`, `mprPath`（~10 字段，全是执行基础设施）。

**验证：** `go build ./mdl/executor/...` + 全部测试通过。

### Phase 3b — executorCache 分解 (SRP)

将单体缓存拆为按域的独立缓存类型，每个类靠近其消费者：

| 缓存 | 新位置 | 模式 |
|------|--------|------|
| `module`, `folder`, `unit` | `mdl/executor/module_cache.go` | `ModuleCache` struct + typed accessors |
| `microflowWithContainer`, `nanoflowWithContainer` | `mdl/executor/microflow_cache.go` | `MicroflowCache` struct + typed accessors |
| `page`, `layout`, `snippet` | `mdl/executor/page_cache.go` | `PageCache` struct + typed accessors |
| `domainModel`, `entity` | `mdl/executor/domainmodel_cache.go` | `DomainModelCache` struct |
| `projectSecurity`, `moduleSecurity` | `mdl/executor/security_cache.go` | `SecurityCache` struct |
| `workflow`, `javaAction`, `javaScriptAction` | `mdl/executor/flow_cache.go` | `FlowCache` struct |

`executorCache` 结构体删除，原来 `sessionTracker` 保留（它是会话状态，不是缓存）。

**验证：** 每个缓存独立测试可读性；`go test ./mdl/executor/...` 全部通过。

### Phase 3c — FullBackend→角色最后 55 处 (ISP)

23 个文件中 55 处 `FullBackend` 引用，逐个文件迁移：

| 文件 | FullBackend 用途 | 替换为 |
|------|-----------------|--------|
| `executor.go` (7 处) | Connect/Disconnect/Version | `ConnectionBackend` + `BackendFactory` |
| `flowbuilder_v2.go` (1) | microflow builder | `MicroflowReader` + `MicroflowWriter` |
| `cmd_pages_builder.go` (1) | page builder | `PageModelAccess` + `PageMutationOperator` |
| `cmd_workflows_write_v2.go` (2) | workflow write | `WorkflowReader` + `WorkflowWriter` + `WorkflowMutationOperator` |
| `hierarchy.go` (3) | container hierarchy | `ModuleLister` + `FolderManager` |
| `cmd_export_mappings.go` (1) | export | `MappingReader` |
| `cmd_import_mappings.go` (1) | import | `MappingReader` + `MappingWriter` |
| `backend/page.go` (1) | page interface | `PageReader` |
| `backend/role.go` (1) | role definitions | 保持（定义处） |
| `backend/workflow.go` (1) | workflow interface | `WorkflowReader` |
| 其余 internal/expr + mock + mpr | 测试/基础设施 | `BackendFactory` 或窄接口 |
| `executor.go` | Execute() | 通过 ExecContext 已有 role 字段 |

**验证：** 每改一个文件就编译。`go vet ./...`。

### Phase 3e — Registry OCP (OCP)

`NewRegistry()` 的 30+ 显式注册消除策略：

1. 在 `Registry` 上加 `RegisterHandlerFunc(typeName string, fn StmtHandlerFunc)` 方法
2. 创建 `handlers/` 子包，每个 handler 文件一个 `Register*()` 导出函数
3. 每个导出函数接受 `*Registry` + 它需要的依赖，闭包捕获
4. 用 `NewRegistryFromOpts(opts ...func(*Registry))` 替代 `NewRegistry()`
5. 每个 `register*Handlers` 函数改为接受窄接口而非 `*ExecContext`

**过渡策略：** `Registry` 同时支持 `StmtHandler`（旧）和 `StmtHandlerFunc`（新）。旧 handler 从 `ExecContext` 依赖中提取 role 字段。等全部迁移完后删除 `StmtHandler`。

**验证：** 覆盖率测试 `TestRegistry_Completeness` 确保所有 AST 类型都有 handler。

### Phase 3d — ExecContext 消除 (SRP+DIP+DIP)

**最危险，最后做。** 1229 处引用，用两阶段消除：

**阶段 3d-1 (机械迁移):** 在每个 handler 函数中，将 `ctx.Backend.SomeMethod()` 替换为 `ctx.SomeReader.SomeMethod()`。不改变函数签名，只改内部调用。可以批量 sed 完成大部分。

**阶段 3d-2 (签名变更):** 改 handler 签名从 `func(ctx *ExecContext, stmt ast.Statement) error` 到 `func(ctx context.Context, stmt ast.Statement) error`。用闭包在注册时捕获依赖。

```
// 迁移前
func execCreateModule(ctx *ExecContext, stmt *ast.CreateModuleStmt) error {
    modules, err := ctx.Backend.ListModules()
    ...
}

// 迁移后
func execCreateModule(
    ctx context.Context,
    stmt *ast.CreateModuleStmt,
    lister backend.ModuleLister,
    writer backend.ModuleWriter,
) error {
    modules, err := lister.ListModules()
    ...
}
```

**分 10 批迁移（每批 ~120 refs）：** 按文件列表排序，从最简单（纯查询）到最复杂（写+事务）。

### Phase 4 — CLI 层更新

- 更新 `cmd/mxcli/main.go` 使用新的 `Builder` + handler 注册组合
- 移除所有对 `ExecContext.Backend` 的直接引用
- `cmd/mxcli-daemon/` 同步更新

**验证：** `go build ./cmd/...` 成功。

### Phase 5 — LSP 修复

审计所有文件中的 `val, _ := expr.(*Type)` 模式：

```bash
rg '_, _ := .+\.\(' --type go | grep -v '_test.go' | grep -v 'ok :='
```

每个改为：
```go
val, ok := expr.(*Type)
if !ok {
    // handle gracefully
}
```

**验证：** `rg '_, _ := .+\.\(' --type go | grep -v '_test.go'` 输出为空（排除文件读取等非类型断言模式）。

### Phase 0 — 门控工具（最后写）

在所有重构完成后写 `cmd/check-solid`，确保新代码不会出现旧模式。这样工具逻辑基于最终代码状态，不需要中途修改。

### Phase 6 — 最终验证

- `make test` 全部通过
- `make bench` 无显著性能退化
- golden snapshot idempotency 验证
- `mx check` 在 baseline 项目上验证 BSON 正确性

---

## 5. 风险缓解

| 风险 | 可能性 | 影响 | 缓解措施 |
|------|--------|------|----------|
| `ExecContext` 消除暴露隐藏的耦合 | 中 | 高 | 每步都编译测试；使用分层策略（3a→3b→3c→3e→3d）确保 ExecContext 消除是最后一步，前面的解耦减少其耦合度。如果仍然复杂，改用 `HandlerDeps` 参数对象模式 |
| 缓存失效行为改变 | 中 | 中 | 每个缓存拆分后运行 golden test；编写显式缓存竞争测试 |
| 注册迁移中断 | 低 | 中 | Registry 同时支持旧 `StmtHandler` 和新 `StmtHandlerFunc` 过渡；覆盖率测试确保无遗漏 |
| 性能回归 | 低 | 中 | `make bench` + 黄金文件比较 |
| 回滚计划 | — | — | 每个 Phase 独立 commit，`git revert <phase-commit>` 精确回滚 |

---

## 6. 成功标准

1. 所有现有测试通过（零回归）
2. `go vet ./...` 清理
3. 无新导入 `backend.FullBackend` 或 `executor.ExecContext`（`FullBackend` 只在 `backend.go` 和 `mock_backend.go` 定义处保留）
4. 所有类型断言检查 `ok`
5. 注册表注册是组合式的：新语句类型在不需要修改 `NewRegistry()` 的情况下添加它自己的 `Register` 调用
6. 每个处理函数声明它的依赖关系，作为函数参数（通过闭包捕获）
7. `Executor` 字段 ≤ 10（仅执行基础设施）
8. `executorCache` 不再存在（拆分为 N 个域缓存）
